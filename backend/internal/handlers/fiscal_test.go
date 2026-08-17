package handlers

import (
	"testing"

	"restaurant-backend/internal/fiscal"
	"restaurant-backend/internal/models"
)

// Saving a half-filled form has to keep working — an owner fills this in over
// several days, between signing a contract and a visit to the tax office. Only
// *enabling* is a claim, and a false claim here is found by an inspector.
func TestFiscalCanBeSavedIncompleteButNotEnabled(t *testing.T) {
	half := fiscalUpdateRequest{Provider: fiscal.Multikassa}
	if err := fiscalEnableRefusal(half); err != nil {
		t.Fatalf("saving a half-filled form was refused: %v", err)
	}
	half.Enabled = true
	if err := fiscalEnableRefusal(half); err == nil {
		t.Fatal("enabling with no TIN and no VAT rate was allowed")
	}
}

// ⚠️ The rate has to be *given*, because zero is a real answer. An unset rate
// silently defaulting to 0 would declare every restaurant VAT-exempt, and every
// receipt would look perfectly well-formed.
func TestEnablingRequiresAnExplicitVATRate(t *testing.T) {
	zero := 0
	req := fiscalUpdateRequest{
		Provider: fiscal.Multikassa,
		Enabled:  true,
		TIN:      "123456789",
	}
	if err := fiscalEnableRefusal(req); err == nil {
		t.Fatal("enabling without a VAT rate was allowed — 0 would be assumed")
	}
	req.VatPercent = &zero
	err := fiscalEnableRefusal(req)
	// Still refused today, but for the *other* reason: no adapter yet. The
	// point of this assertion is that the missing-rate complaint is gone.
	if err != nil && err.Error() ==
		"QQS stavkasi ko'rsatilmagan (QQS to'lovchisi bo'lmasangiz 0 kiriting)" {
		t.Fatal("an explicit zero rate was treated as unset")
	}
}

// A provider we cannot dial yet must refuse with *our* problem named, not with
// a credentials complaint — otherwise the owner spends the afternoon retyping a
// password that was never wrong.
func TestUnbuiltProviderIsRefusedAsOurGap(t *testing.T) {
	zero := 0
	err := fiscalEnableRefusal(fiscalUpdateRequest{
		Provider:   fiscal.FirstOFD,
		Enabled:    true,
		TIN:        "123456789",
		VatPercent: &zero,
	})
	if err == nil {
		t.Fatal("a provider with no adapter was allowed to be enabled")
	}
	if !contains(err.Error(), "hujjat") {
		t.Fatalf("refusal %q does not say the gap is ours", err)
	}
}

// ⚠️ A register on the restaurant's own network has nothing but its address.
// Enabling without one turns on the claim that sales are registered and then
// files nothing — and the claim is checked by an inspector, not by us.
func TestLocalProviderNeedsItsAddress(t *testing.T) {
	zero := 0
	base := fiscalUpdateRequest{
		Provider:   fiscal.Multikassa,
		Enabled:    true,
		TIN:        "123456789",
		VatPercent: &zero,
	}
	err := fiscalEnableRefusal(base)
	if err == nil {
		t.Fatal("a local register was enabled with no address to reach it at")
	}
	if !contains(err.Error(), "manzil") {
		t.Fatalf("refusal %q does not name the missing address", err)
	}

	base.Multikassa = models.FiscalCreds{BaseURL: "http://192.168.1.50:9090"}
	if err := fiscalEnableRefusal(base); err != nil {
		t.Fatalf("a fully configured local register was refused: %v", err)
	}

	// ⚠️ And the address is read from **that provider's own drawer**. Filling in
	// another provider's must not satisfy it — the separate drawers exist so one
	// provider is never handed another's settings, and a check that looks
	// anywhere else quietly undoes them.
	stray := fiscalUpdateRequest{
		Provider:   fiscal.Multikassa,
		Enabled:    true,
		TIN:        "123456789",
		VatPercent: &zero,
		EPOS:       models.FiscalCreds{BaseURL: "http://192.168.1.50:9090"},
	}
	if err := fiscalEnableRefusal(stray); err == nil {
		t.Fatal("another provider's address satisfied the check")
	}
}

func TestUnknownProviderIsNotEnableable(t *testing.T) {
	if fiscal.Known("nalog-ru") {
		t.Fatal("an invented provider passed Known")
	}
}

// Empty secrets mean "keep the stored one". The opposite reading unfiscalises a
// restaurant whose owner came in to fix a typo in the TIN.
func TestEmptySecretKeepsTheStoredOne(t *testing.T) {
	stored := models.FiscalCreds{Password: "real-password", Token: "real-token", Login: "acme"}
	merged := mergeCreds(models.FiscalCreds{Login: "acme-2"}, stored)
	if merged.Password != "real-password" || merged.Token != "real-token" {
		t.Fatalf("secrets were cleared by a save that did not retype them: %+v", merged)
	}
	// A visible field, though, is taken as sent: clearing it in the form means it.
	if merged.Login != "acme-2" {
		t.Fatalf("login = %q, want the submitted value", merged.Login)
	}
}

// The response the panel gets must never carry a secret. Written out by hand
// for exactly this reason — a field added to the model later would otherwise
// ride along.
func TestFiscalResponseCarriesNoSecrets(t *testing.T) {
	s := &models.FiscalSettings{
		Provider:   fiscal.Multikassa,
		Multikassa: models.FiscalCreds{Login: "acme", Password: "p", Token: "t"},
	}
	out := fiscalResponse(s)
	flags := out.Creds[fiscal.Multikassa]
	if !flags.HasSecret {
		t.Fatal("a stored secret was not flagged")
	}
	if flags.Login != "acme" {
		t.Fatalf("login = %q", flags.Login)
	}
	// VatPercent stays nil rather than becoming 0 — the panel has to be able to
	// tell "not declared" from "exempt".
	if out.VatPercent != nil {
		t.Fatal("an undeclared VAT rate came back as a number")
	}
}
