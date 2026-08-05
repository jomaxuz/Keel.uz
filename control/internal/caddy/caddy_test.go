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
		"osh.keel.uz, osh.uz {",     // both hostnames, sorted
		"handle /api/* {",           //
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
