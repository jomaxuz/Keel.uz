// Package caddy renders the whole edge configuration from the tenant list and
// hands it to a running Caddy.
//
// The whole file is rewritten every time rather than patched. A config built
// from one source — the tenant collection — cannot drift from it; a config
// edited in place drifts the first time a reload is missed, and the symptom is
// a customer's domain quietly serving somebody else's shop.
//
// Reloading goes through Caddy's admin API with `Content-Type: text/caddyfile`,
// so Caddy adapts and validates the text itself. A bad config is rejected whole
// and the running one keeps serving — which is the behaviour you want from the
// component that stands in front of every customer.
package caddy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Site is one tenant as the edge needs to see it.
type Site struct {
	Slug    string
	Domains []string
	// Suspended sites keep their certificate and their domain, and answer with
	// a page that says why. Returning nothing would look like an outage the
	// customer might blame on us rather than on their invoice.
	Suspended bool
}

// frontendProxy renders the reverse_proxy block for the shared renderers.
//
// With one address it is the plain one-liner it always was. With several it adds
// the two directives that make a pool behave:
//
//   - `least_conn` — the fewest in-flight requests wins. A page render takes
//     300 ms and varies, so round-robin would happily queue a third request
//     behind two slow ones while a replica sits idle.
//   - `lb_try_duration` — a replica that is down (a deploy, an OOM kill) costs a
//     retry rather than a 502 for a third of visitors. Without it, "one replica
//     restarting" and "the site is broken" look the same from outside.
func frontendProxy(frontend string) string {
	hosts := []string{}
	for _, h := range strings.Split(frontend, ",") {
		if h = strings.TrimSpace(h); h != "" {
			hosts = append(hosts, h)
		}
	}
	if len(hosts) == 0 {
		// Nothing configured at all. Emitted as-is so Caddy rejects the config
		// loudly rather than this silently writing a proxy to nowhere.
		return fmt.Sprintf("\t\treverse_proxy %s\n", strings.TrimSpace(frontend))
	}
	if len(hosts) == 1 {
		// The trimmed host, not the raw string: a test caught this returning
		// " only:3000 , " verbatim, which is what an env var looks like after
		// somebody edits it by hand.
		return fmt.Sprintf("\t\treverse_proxy %s\n", hosts[0])
	}
	return fmt.Sprintf("\t\treverse_proxy %s {\n\t\t\tlb_policy least_conn\n"+
		"\t\t\tlb_try_duration 5s\n\t\t\tfail_duration 10s\n\t\t}\n",
		strings.Join(hosts, " "))
}

// Options are the fixed parts of the edge.
type Options struct {
	// Where the shared Next.js renderers listen. One address, or several
	// separated by commas — "keel-frontend-1:3000,keel-frontend-2:3000,…".
	//
	// ⚠️ **A list, because a single DNS name balanced connections rather than
	// requests.** Three replicas behind one alias measured 43 renders/second with
	// the load on two of them (109%, 109%, 39%): Caddy sees one upstream, dials
	// it once per connection, and keep-alive then pins every request on that
	// connection to whichever replica answered first. Naming the replicas lets
	// Caddy pick per request, which is what `least_conn` needs to mean anything —
	// and page renders are exactly the uneven workload it is for.
	Frontend string
	// The control plane itself: it answers the TLS gate and serves the
	// suspended and unknown-domain pages.
	Control string
	// keel.uz itself — the marketing site and the console.
	MainDomains []string
	// Where the Keel site process listens.
	MainUpstream string
	// Contact for Let's Encrypt.
	Email string
}

// Render builds the Caddyfile.
func Render(sites []Site, o Options) string {
	var b strings.Builder

	b.WriteString("{\n")
	if o.Email != "" {
		fmt.Fprintf(&b, "\temail %s\n", o.Email)
	}
	b.WriteString("\tadmin 0.0.0.0:2019\n")
	b.WriteString("\ton_demand_tls {\n")
	// Without the ask, anybody who points a domain here makes us request a
	// certificate for it, and the rate limit is spent on strangers.
	fmt.Fprintf(&b, "\t\task http://%s/internal/tls-ask\n", o.Control)
	b.WriteString("\t}\n}\n\n")

	// keel.uz and the console.
	//
	// `/api/*` goes to the control plane here rather than through a Next.js
	// rewrite, for the same reason a tenant's does: Next bakes its rewrite
	// destinations into the build, so a runtime environment variable naming the
	// control plane is silently ignored and the console ends up calling
	// localhost. Routing at the edge keeps the browser single-origin without
	// asking the frontend to know where anything lives.
	if len(o.MainDomains) > 0 && o.MainUpstream != "" {
		fmt.Fprintf(&b, "%s {\n", strings.Join(o.MainDomains, ", "))
		fmt.Fprintf(&b, "\thandle /api/* {\n\t\treverse_proxy %s\n\t}\n", o.Control)
		fmt.Fprintf(&b, "\thandle {\n\t\treverse_proxy %s\n\t}\n", o.MainUpstream)
		b.WriteString("}\n\n")
	}

	// Sorted so an unchanged tenant list renders byte-identical: a diff in the
	// generated config should mean something actually changed.
	sorted := append([]Site(nil), sites...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Slug < sorted[j].Slug })

	for _, s := range sorted {
		domains := cleanDomains(s.Domains)
		if len(domains) == 0 {
			continue
		}
		fmt.Fprintf(&b, "# %s\n%s {\n", s.Slug, strings.Join(domains, ", "))
		b.WriteString("\ttls {\n\t\ton_demand\n\t}\n")
		if s.Suspended {
			// The container is stopped; there is nothing to proxy to.
			fmt.Fprintf(&b, "\treverse_proxy %s {\n\t\theader_up X-Keel-Tenant %s\n\t}\n",
				o.Control+"/suspended", s.Slug)
			b.WriteString("}\n\n")
			continue
		}
		// The browser only ever sees one origin: the API and the uploads are
		// this tenant's own container, everything else is the shared frontend.
		fmt.Fprintf(&b, "\thandle /api/* {\n\t\treverse_proxy %s:8080\n\t}\n", containerName(s.Slug))
		fmt.Fprintf(&b, "\thandle /uploads/* {\n\t\treverse_proxy %s:8080\n\t}\n", containerName(s.Slug))
		fmt.Fprintf(&b, "\thandle {\n%s\t}\n", frontendProxy(o.Frontend))
		b.WriteString("}\n\n")
	}

	return b.String()
}

func containerName(slug string) string { return "keel-" + slug }

func cleanDomains(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, d := range in {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" || seen[d] || !strings.Contains(d, ".") {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

// Apply hands the rendered config to Caddy.
//
// Caddy adapts and validates it; on any error the previously loaded config
// keeps running, so a mistake here degrades to "the change did not take"
// rather than "every customer is offline".
func Apply(ctx context.Context, adminURL, caddyfile string) error {
	url := strings.TrimRight(adminURL, "/") + "/load"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url,
		bytes.NewReader([]byte(caddyfile)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/caddyfile")
	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("caddy: ulanib bo'lmadi (%s): %w", adminURL, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 400 {
		return fmt.Errorf("caddy konfiguratsiyani qabul qilmadi: %s",
			strings.TrimSpace(string(raw)))
	}
	return nil
}
