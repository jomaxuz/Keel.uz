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
