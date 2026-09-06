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

// encode is the one line that made every JSON response five to seven times
// smaller.
//
// Measured on the load test of 2026-09-03: `GET /api/v1/restaurant` was 22 KB
// with **no `content-encoding` at all**, even when the browser asked for gzip —
// nothing in the chain had ever been told to compress. The restaurant profile,
// the menu and the categories are the three biggest responses on the site and
// they are almost entirely repeated text, which is the shape compression is
// best at.
//
// ⚠️ **Per site block, not global.** Caddy's global options block takes no
// directives that touch a response, so `encode` has to be repeated in each
// site. Rendering it from one helper is what keeps "each site" from meaning
// "every site except the one somebody added last".
//
// zstd first, gzip second: Caddy picks the first the client accepts, and a
// client that takes zstd gets a smaller body for less CPU than gzip. Anything
// that speaks neither — an old TV set, a POS terminal — is served the plain
// body exactly as before, so this cannot break a client by being on.
const encodeDirective = "\tencode zstd gzip\n"

// securityHeaders is the response header block every served site carries.
//
// ⚠️ **At the edge, and per site for the same reason `encode` is** — Caddy's
// global options block sets nothing that touches a response, so one site added
// last and rendered without this is a site served without it. One helper, every
// block.
//
// ⚠️ **Deliberately not a full Content-Security-Policy.** A `script-src` policy
// is the thing that would actually stop a stolen-token XSS, and it is also the
// thing most likely to break a Next.js app silently: Next serves inline
// hydration scripts, and the map SDKs (2GIS, Yandex, Google) each load their own
// — a blind `script-src 'self'` blanks the site with an error only the console
// shows. That belongs in a tested, nonce-based rollout, not in a header string.
// What is here is the set that improves security without that risk.
//
//   - HSTS pins HTTPS for a year. No `includeSubDomains` and no `preload`:
//     tenants bring their own domains, and claiming their subdomains — or
//     locking a domain onto us after they leave — is not ours to do.
//   - `nosniff` stops a browser guessing an uploaded file is a script.
//   - `frame-ancestors`, not `X-Frame-Options: DENY`, because **the Telegram
//     mini app is this very site rendered inside Telegram** (channelreport.go),
//     which frames it on web.telegram.org. DENY would break every mini app; a
//     frame-ancestors CSP blocks clickjacking while letting Telegram embed us.
//     This directive touches only embedding — it does not restrict scripts or
//     styles, so it cannot blank the page the way a full CSP can.
//   - `Referrer-Policy` keeps a full URL (order numbers, ids) from leaking to
//     a third party a guest clicks through to.
//   - `-Server` drops the "Caddy" version banner; a smaller target, no cost.
const securityHeaders = "\theader {\n" +
	"\t\tStrict-Transport-Security \"max-age=31536000\"\n" +
	"\t\tX-Content-Type-Options nosniff\n" +
	"\t\tReferrer-Policy strict-origin-when-cross-origin\n" +
	"\t\tContent-Security-Policy \"frame-ancestors 'self' https://web.telegram.org https://*.telegram.org\"\n" +
	"\t\t-Server\n" +
	"\t}\n"

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
		b.WriteString(encodeDirective)
		b.WriteString(securityHeaders)
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
		b.WriteString(encodeDirective)
		b.WriteString(securityHeaders)
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
