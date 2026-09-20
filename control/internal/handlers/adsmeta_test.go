package handlers

import (
	"os"
	"strings"
	"testing"

	"keel-control/internal/models"
)

// between is one function's body, cut out of the file it lives in.
//
// ⚠️ Source-read tests, like the tenant's: every rule below fails quietly —
// a secret in an answer looks like nothing, and an ungated door looks like a
// working one.
func between(t *testing.T, src, from, to string) string {
	t.Helper()
	i := strings.Index(src, from)
	if i < 0 {
		t.Fatalf("%q is gone from the source", from)
	}
	rest := src[i:]
	j := strings.Index(rest, to)
	if j < 0 {
		t.Fatalf("%q never ends", from)
	}
	return rest[:j]
}

func adsMetaSource(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile("adsmeta.go")
	if err != nil {
		t.Fatal(err)
	}
	return string(src)
}

// ⚠️ **The app secret is used here and leaves in nothing.** A tenant container
// is the customer's own machine; the answer it gets carries the app id and the
// configuration id — both of which appear in Meta's own dialog URL — and never
// the secret that turns a code into a token.
func TestTheAppSecretIsNeverAnswered(t *testing.T) {
	src := adsMetaSource(t)
	app := between(t, src, "func (h *Handler) AdsApp", "\n}\n")
	if strings.Contains(app, "MetaAppSecret") {
		t.Fatal("the app secret is being written into an answer a tenant reads")
	}
	tok := between(t, src, "func (h *Handler) AdsToken", "\n}\n")
	if strings.Contains(tok, "MetaAppSecret") {
		t.Fatal("the app secret reaches the token answer")
	}
	exch := between(t, src, "func (h *Handler) exchangeMetaCode", "\n}\n")
	if !strings.Contains(exch, `"client_secret"`) {
		t.Fatal("the exchange stopped using the app secret — which means it " +
			"is being done somewhere else, by somebody who holds one")
	}
}

// ⚠️ **Both doors are gated on the add-on, read from our own record.** A tenant
// container can post anything it likes; a handler that believed a posted
// entitlement would give away a section sold at 1 250 000 so'm a month to
// anybody who can edit a JSON body.
func TestConnectingIsGatedOnTheAddonWeRecorded(t *testing.T) {
	src := adsMetaSource(t)
	for _, fn := range []string{"func (h *Handler) AdsApp", "func (h *Handler) AdsToken"} {
		body := between(t, src, fn, "\n}\n")
		if !strings.Contains(body, "h.grantOf(r.Context(), t)") ||
			!strings.Contains(body, "billing.AdsEntitled(addons)") {
			t.Fatalf("%s no longer checks the add-on against our own record", fn)
		}
	}
}

// ⚠️ **The Graph version must match the tenant's.** A code minted against one
// version and exchanged against another works today, and is exactly the
// mismatch that surfaces as an unexplained failure the week Meta retires one
// of them.
func TestTheGraphVersionMatchesTheTenantsOwn(t *testing.T) {
	src := adsMetaSource(t)
	if !strings.Contains(src, `const metaVersion = "v26.0"`) {
		t.Fatal("the console's Graph version moved; the tenant's meta.Version " +
			"has to move with it")
	}
}

// ⚠️ **The redirect target is never something a request asserts.** The caller
// is a customer's own container; a value it could name freely would decide
// where a browser holding an authorization code for *this* restaurant's ad
// account is sent next.
func TestTheReturnAddressIsCheckedAgainstTheTenantsOwnDomains(t *testing.T) {
	h := &Handler{}
	tenant := &models.Tenant{
		Slug:    "osh",
		Domains: []string{"osh.uz", "osh.keel.uz"},
	}
	if _, err := h.adsReturnTo(tenant, "https://osh.uz/admin/ads"); err != nil {
		t.Fatalf("a hostname this tenant actually serves was refused: %v", err)
	}
	for _, bad := range []string{
		"https://evil.example/admin/ads",
		"https://osh.uz.evil.example/admin/ads",
		"ftp://osh.uz/admin/ads",
		"/admin/ads",
		"",
	} {
		if _, err := h.adsReturnTo(tenant, bad); err == nil {
			t.Fatalf("%q was accepted as a place to send an authorization code", bad)
		}
	}
	// ⚠️ A query the caller attached is dropped: it would be glued onto the
	// address we are about to put a code on.
	got, err := h.adsReturnTo(tenant, "https://osh.uz/admin/ads?next=//evil.example")
	if err != nil || got != "https://osh.uz/admin/ads" {
		t.Fatalf("a caller's query survived onto the return address: %q (%v)", got, err)
	}
}

// ⚠️ **One address in Meta's settings, not one per restaurant.** Whitelisting
// each customer's domain means editing Meta for every sale, and the one nobody
// remembered is a connect button failing with a message about redirect URIs.
func TestTheDialogAndTheExchangeUseOnePlatformAddress(t *testing.T) {
	src := adsMetaSource(t)
	app := between(t, src, "func (h *Handler) AdsApp", "\n}\n")
	if !strings.Contains(app, "h.Cfg.MetaRedirectURI") {
		t.Fatal("the dialog is no longer opened against the platform's own " +
			"redirect address")
	}
	tok := between(t, src, "func (h *Handler) AdsToken", "\n}\n")
	if !strings.Contains(tok, "h.Cfg.MetaRedirectURI") {
		t.Fatal("the code exchange stopped using the same address the dialog " +
			"used; Meta compares the two byte for byte")
	}
	if strings.Contains(tok, "req.RedirectURI") {
		t.Fatal("a caller-supplied redirect address is back in the exchange")
	}
}

// ⚠️ **An unknown state is forwarded nowhere.** It is either an expired dialog
// or somebody trying the address by hand, and the one thing this endpoint must
// never do is hand a code to a destination it cannot account for.
func TestAnUnknownStateIsNotRedirectedAnywhere(t *testing.T) {
	fn := between(t, adsMetaSource(t), "func (h *Handler) AdsRedirect", "\n}\n")
	lookup := strings.Index(fn, "FindOneAndDelete")
	redirect := strings.Index(fn, "http.Redirect")
	if lookup < 0 || redirect < 0 {
		t.Fatal("the redirect no longer looks the state up, or no longer forwards")
	}
	if lookup > redirect {
		t.Fatal("the browser is forwarded before the state is resolved")
	}
	// Consumed on use: a code is single-use, and so is the row that says where
	// it may go.
	if !strings.Contains(fn, "FindOneAndDelete") {
		t.Fatal("the state survives its own use")
	}
}
