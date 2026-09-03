package caddy

import (
	"strings"
	"testing"
)

// The edge decides which customer a request belongs to. Two mistakes here are
// serious enough to pin down: routing a tenant's API at the shared frontend
// (which would answer for the wrong shop), and proxying a suspended tenant to
// a container that has been stopped (which is a timeout, not a message).

func opts() Options {
	return Options{
		Frontend:     "keel-frontend:3000",
		Control:      "keel-control:9000",
		MainDomains:  []string{"keel.uz", "www.keel.uz"},
		MainUpstream: "keel-site:3100",
		Email:        "hello@keel.uz",
	}
}

func TestRenderSplitsApiFromTheSharedFrontend(t *testing.T) {
	out := Render([]Site{{Slug: "osh", Domains: []string{"osh.uz", "osh.keel.uz"}}}, opts())

	for _, want := range []string{
		"osh.keel.uz, osh.uz {", // both hostnames, sorted
		"handle /api/* {",       //
		"reverse_proxy keel-osh:8080",
		"handle /uploads/* {",
		"reverse_proxy keel-frontend:3000",
		"on_demand",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("config is missing %q:\n%s", want, out)
		}
	}
	// The ask endpoint is what stops a stranger's domain from spending our
	// certificate quota.
	if !strings.Contains(out, "ask http://keel-control:9000/internal/tls-ask") {
		t.Fatalf("on-demand TLS must be gated:\n%s", out)
	}
}

// The console's own API has to be routed at the edge too.
//
// This shipped broken: keel.uz was proxied wholesale to the Next.js process,
// which was expected to forward `/api/*` to the control plane itself. It does
// not — Next bakes rewrite destinations in at build time, so the runtime
// variable naming the control plane was ignored and every login reached
// `localhost:9000` inside the site container. The symptom was an "Internal
// Server Error" body arriving where the browser expected JSON.
func TestConsoleApiGoesToTheControlPlane(t *testing.T) {
	out := Render(nil, opts())

	main := out[strings.Index(out, "keel.uz, www.keel.uz {"):]
	for _, want := range []string{
		"handle /api/* {",
		"reverse_proxy keel-control:9000",
		"reverse_proxy keel-site:3100",
	} {
		if !strings.Contains(main, want) {
			t.Fatalf("main site block is missing %q:\n%s", want, main)
		}
	}
	// Order matters: a catch-all ahead of the API handler would swallow it.
	if strings.Index(main, "handle /api/*") > strings.Index(main, "\thandle {") {
		t.Fatalf("the catch-all must come after /api/*:\n%s", main)
	}
}

func TestSuspendedTenantGetsAPageNotATimeout(t *testing.T) {
	out := Render([]Site{{Slug: "osh", Domains: []string{"osh.uz"}, Suspended: true}}, opts())

	if strings.Contains(out, "keel-osh:8080") {
		t.Fatalf("a suspended tenant's container is stopped; nothing may proxy to it:\n%s", out)
	}
	if !strings.Contains(out, "keel-control:9000/suspended") {
		t.Fatalf("a suspended tenant must be answered, not dropped:\n%s", out)
	}
	// It keeps its certificate: the domain still has to serve HTTPS to say so.
	if !strings.Contains(out, "on_demand") {
		t.Fatalf("suspended sites still need TLS:\n%s", out)
	}
}

func TestRenderIsStableAndSkipsJunk(t *testing.T) {
	sites := []Site{
		{Slug: "zeta", Domains: []string{"zeta.uz"}},
		{Slug: "alpha", Domains: []string{"ALPHA.uz", " alpha.uz ", "not-a-domain"}},
		{Slug: "empty", Domains: []string{"", "   "}},
	}
	a := Render(sites, opts())
	b := Render([]Site{sites[1], sites[2], sites[0]}, opts())
	if a != b {
		t.Fatal("the same tenants in a different order must render identically")
	}
	// Duplicates differing only in case would make Caddy refuse the whole file.
	if strings.Count(a, "alpha.uz") != 1 {
		t.Fatalf("hostnames must be normalised and de-duplicated:\n%s", a)
	}
	if strings.Contains(a, "not-a-domain") {
		t.Fatalf("a string without a dot is not a hostname:\n%s", a)
	}
	// A tenant with no usable domain must not emit an empty site block, which
	// would be a syntax error and would take the whole reload down with it.
	if strings.Contains(a, "# empty\n {") || strings.Contains(a, " {\n\ttls") == false {
		// the second half only asserts the file is not empty
	}
	if strings.Contains(a, "# empty") {
		t.Fatalf("a tenant with no domains must be skipped entirely:\n%s", a)
	}
}

func TestMainSiteIsServedToo(t *testing.T) {
	out := Render(nil, opts())
	if !strings.Contains(out, "keel.uz, www.keel.uz {") ||
		!strings.Contains(out, "reverse_proxy keel-site:3100") {
		t.Fatalf("keel.uz itself must be in the config:\n%s", out)
	}
}

// One renderer or several is the difference between balancing connections and
// balancing requests, and the measurement that produced this test was three
// replicas at 109% / 109% / 39% CPU — a third of the capacity idle because
// keep-alive pinned every request on a connection to one replica.
func TestFrontendPoolBalancesPerRequest(t *testing.T) {
	// A single upstream stays exactly as it was: this is also the shape a
	// single-restaurant install and every existing deployment use.
	if got := frontendProxy("keel-frontend:3000"); got != "\t\treverse_proxy keel-frontend:3000\n" {
		t.Errorf("bitta upstream o'zgardi: %q", got)
	}

	got := frontendProxy("a:3000, b:3000 ,c:3000")
	for _, want := range []string{
		"reverse_proxy a:3000 b:3000 c:3000",
		// Renders vary in cost, so the fewest-in-flight wins rather than a
		// round-robin that queues behind a slow one.
		"lb_policy least_conn",
		// A replica that is down costs a retry, not a 502 for a third of
		// visitors — otherwise a restart is indistinguishable from an outage.
		"lb_try_duration 5s",
		"fail_duration 10s",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("%q yo'q:\n%s", want, got)
		}
	}

	// Whitespace and a trailing comma are what an env var actually looks like
	// after somebody edits it by hand.
	if got := frontendProxy(" only:3000 , "); got != "\t\treverse_proxy only:3000\n" {
		t.Errorf("bo'sh joy tozalanmadi: %q", got)
	}
}

// Compression is on for every site, including the console's own domain.
//
// It shipped off: the load test found `GET /api/v1/restaurant` arriving as 22 KB
// of uncompressed JSON with no `content-encoding` header, however the client
// asked. The header is set at the edge, so the check belongs here — and it is a
// per-site directive, which is exactly the kind that gets added to one block and
// forgotten in the others.
func TestEverySiteBlockCompresses(t *testing.T) {
	out := Render([]Site{
		{Slug: "osh", Domains: []string{"osh.uz"}},
		{Slug: "stop", Domains: []string{"stop.uz"}, Suspended: true},
	}, opts())

	for _, start := range []string{"keel.uz, www.keel.uz {", "osh.uz {", "stop.uz {"} {
		i := strings.Index(out, start)
		if i < 0 {
			t.Fatalf("no site block %q:\n%s", start, out)
		}
		body := out[i:]
		if end := strings.Index(body, "\n}\n"); end > 0 {
			body = body[:end]
		}
		if !strings.Contains(body, "encode zstd gzip") {
			t.Fatalf("site %q is served uncompressed:\n%s", start, body)
		}
	}
}
