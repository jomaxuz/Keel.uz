package handlers

import "testing"

// Which document the landing strip reads a customer's face from.
//
// The first version read only `restaurant` and shipped a strip that said
// "My Restaurant" with no logo, for a customer who had set both. The cause was
// invisible from the control plane: once a tenant has a brand — every one of
// them does — its settings page writes the name and logo onto the **brand**
// and deletes them from the company document. Nothing errored; the fields were
// simply empty forever.

func TestBrandWinsOverCompany(t *testing.T) {
	name, logo := resolveIdentity(
		siteIdentity{Name: "B5 Somsa", LogoURL: "/uploads/b5.png"},
		siteIdentity{Name: "My Restaurant", LogoURL: ""},
		"b5somsa",
	)
	if name != "B5 Somsa" || logo != "/uploads/b5.png" {
		t.Fatalf("got %q / %q — the brand is what the guest sees", name, logo)
	}
}

// The exact shape that shipped broken: brand set, company still seeded.
func TestSeededPlaceholderIsNeverPrinted(t *testing.T) {
	name, _ := resolveIdentity(
		siteIdentity{},
		siteIdentity{Name: seedRestaurantName},
		"b5somsa",
	)
	if name == seedRestaurantName {
		t.Fatal("the seeded placeholder reached the marketing page")
	}
	if name != "b5somsa" {
		t.Fatalf("name = %q, want the console's name as the last resort", name)
	}
}

func TestCompanyFillsWhatTheBrandLacks(t *testing.T) {
	// A brand named but never given a logo keeps the company's.
	name, logo := resolveIdentity(
		siteIdentity{Name: "B5 Somsa"},
		siteIdentity{Name: "Eski nom", LogoURL: "/uploads/old.png"},
		"b5somsa",
	)
	if name != "B5 Somsa" {
		t.Errorf("name = %q, want the brand's", name)
	}
	if logo != "/uploads/old.png" {
		t.Errorf("logo = %q, want the company's as the fallback", logo)
	}
}

// No logo anywhere is not a reason to drop the customer — the strip renders
// the name as a wordmark. Silently punishing whoever never uploaded one is not
// a rule anybody chose.
func TestNoLogoStillYieldsAName(t *testing.T) {
	name, logo := resolveIdentity(siteIdentity{}, siteIdentity{}, "b5somsa")
	if name != "b5somsa" || logo != "" {
		t.Fatalf("got %q / %q", name, logo)
	}
}

// The path the browser on keel.uz will actually load.
func TestAbsoluteBuildsTheCustomersOwnURL(t *testing.T) {
	const site = "https://b5somsa.keel.uz"
	if got := absolute(site, "/uploads/b5.png"); got != site+"/uploads/b5.png" {
		t.Errorf("got %q", got)
	}
	// A pasted CDN link is left alone.
	if got := absolute(site, "https://cdn.example/b5.png"); got != "https://cdn.example/b5.png" {
		t.Errorf("got %q", got)
	}
	if got := absolute(site, ""); got != "" {
		t.Errorf("empty logo must stay empty, got %q", got)
	}
}

// Which domain a human is sent to is a different question from which one the
// container knows, and conflating them put the free subdomain under the logo of
// a customer who had connected their own.
func TestPublicDomainPrefersTheCustomersOwn(t *testing.T) {
	cases := []struct {
		name    string
		domains []string
		want    string
	}{
		{
			// The case that shipped wrong: the free subdomain is always first,
			// because that is how a tenant is created and unlinking refuses to
			// remove it.
			name:    "o'z domeni ulangan",
			domains: []string{"kfc.keel.uz", "traderbot.uz"},
			want:    "traderbot.uz",
		},
		{
			name:    "faqat bepul subdomen",
			domains: []string{"kfc.keel.uz"},
			want:    "kfc.keel.uz",
		},
		{
			// www and the bare domain both belong to the customer; the first
			// one they added wins, and either is a real address.
			name:    "o'z domeni www bilan",
			domains: []string{"osh.keel.uz", "oshmarkazi.uz", "www.oshmarkazi.uz"},
			want:    "oshmarkazi.uz",
		},
		{
			// A second platform subdomain is still ours, not theirs.
			name:    "ikkita subdomen",
			domains: []string{"osh.keel.uz", "osh2.keel.uz"},
			want:    "osh.keel.uz",
		},
		{
			name:    "bo'sh",
			domains: nil,
			want:    "",
		},
	}
	for _, c := range cases {
		if got := publicDomain(c.domains, "keel.uz"); got != c.want {
			t.Errorf("%s: publicDomain = %q, kutilgan %q", c.name, got, c.want)
		}
	}
}

// The logo and the link have to name the same restaurant. They did not: uploads
// are stored as absolute URLs built from PUBLIC_BASE_URL — always the free
// subdomain — so a customer with their own domain got a link to traderbot.uz
// and an image from kfc.keel.uz. It works, which is exactly why it survived.
func TestRehostMovesTheLogoToTheDomainTheLinkUses(t *testing.T) {
	const site = "https://traderbot.uz"
	domains := []string{"kfc.keel.uz", "traderbot.uz"}

	if got := rehost(site, domains, "https://kfc.keel.uz/uploads/a.jpg"); got != site+"/uploads/a.jpg" {
		t.Errorf("o'z subdomeni ko'chirilmadi: %q", got)
	}
	// Already right — and must not be mangled by being "fixed" twice.
	if got := rehost(site, domains, "https://traderbot.uz/uploads/a.jpg"); got != site+"/uploads/a.jpg" {
		t.Errorf("to'g'ri URL buzildi: %q", got)
	}
	// A relative path behaves as before.
	if got := rehost(site, domains, "/uploads/a.jpg"); got != site+"/uploads/a.jpg" {
		t.Errorf("nisbiy yo'l: %q", got)
	}
	// ⚠️ Somebody else's host is left alone. We do not know that a CDN serves
	// the same path under our customer's name, and rewriting it would turn a
	// working image into a 404.
	const cdn = "https://cdn.example/x/a.jpg"
	if got := rehost(site, domains, cdn); got != cdn {
		t.Errorf("begona host o'zgartirildi: %q", got)
	}
	// Query strings survive: some uploads carry a cache-busting parameter.
	if got := rehost(site, domains, "https://kfc.keel.uz/uploads/a.jpg?v=2"); got != site+"/uploads/a.jpg?v=2" {
		t.Errorf("query yo'qoldi: %q", got)
	}
	if got := rehost(site, domains, ""); got != "" {
		t.Errorf("bo'sh logotip bo'sh qolishi kerak: %q", got)
	}
}
