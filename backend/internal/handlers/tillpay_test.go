package handlers

import (
	"strings"
	"testing"

	"restaurant-backend/internal/instore"
	"restaurant-backend/internal/models"
)

// ⚠️ **A check paid through a provider must not close on the cashier's word.**
// The shortcut — mark it paid when the cashier picks Payme, let the callback
// catch up — passes every test anybody writes, because in a test the payment
// succeeds. In a queue it hands out food for a payment that was cancelled,
// expired, or made on somebody else's phone, and the restaurant finds out at
// the end of the month.
//
// It is the website's oldest payment rule applied at the counter: the browser
// proves nothing, only the server-to-server call does.
func TestAnOnlinePaymentClosesOnlyOnTheProvidersWord(t *testing.T) {
	src := readSource(t, "tillclose.go")
	fn := between(t, src, "func (h *Handler) StaffCloseCheck", "\n}\n")

	// ⚠️ **`bankConfirmed`, not `tillOnlineMethods`.** The list grew: a card
	// scanned at the counter is confirmed by a bank too, and the guard was
	// renamed rather than copied so a rail added to one list cannot be missing
	// from the other. This test names the function on purpose — pinning the
	// map instead would go green again the day somebody re-opened the gap.
	if !strings.Contains(fn, "bankConfirmed(method)") ||
		!strings.Contains(fn, "models.PayPaid") {
		t.Fatal("a check can be closed on a payment nobody confirmed")
	}
	// Before the sale is written, not after: a check that has already been
	// marked delivered cannot be un-delivered by a refusal further down.
	guard := strings.Index(fn, "bankConfirmed(method)")
	write := strings.Index(fn, `set["status"] = models.StatusDelivered`)
	if write >= 0 && guard > write {
		t.Fatal("the payment is checked after the sale has already been closed")
	}
	// ⚠️ And the bank's own minute survives. Overwriting paidAt with "when the
	// cashier pressed the button" moves money between shifts at exactly the
	// hour a shift changes.
	if !strings.Contains(fn, "if o.PaidAt == nil {") {
		t.Fatal("the provider's payment time is being overwritten at close")
	}
}

// ⚠️ **Starting a payment records that we asked, not that anybody answered.**
// Writing `paid` here would make the guard above unreachable — the two live in
// different files, so nothing else would notice.
func TestStartingAnOnlinePaymentDoesNotBookTheMoney(t *testing.T) {
	src := readSource(t, "tillpay.go")
	fn := between(t, src, "func (h *Handler) TillStartPayment", "\n}\n")

	if !strings.Contains(fn, "models.PayPending") {
		t.Fatal("a started payment is no longer marked pending")
	}
	if strings.Contains(fn, "models.PayPaid") {
		t.Fatal("asking for a QR code is booking the money as received")
	}
	// A provider that is not configured must refuse rather than draw a dead
	// QR: a code leading to the provider's error page is indistinguishable,
	// to everyone standing there, from a broken till.
	if !strings.Contains(fn, "Configured(req.Provider)") {
		t.Fatal("an unconfigured provider still produces a code")
	}
}

// The terminal stays outside all of this. It has its own receipt and its own
// settlement, and claiming to confirm it would put a green tick on the one
// payment this system cannot see.
func TestTheTerminalIsNotTreatedAsAnOnlineRail(t *testing.T) {
	if tillOnlineMethods["card"] || tillOnlineMethods["transfer"] {
		t.Fatal("the counter terminal is being waited on for a callback")
	}
	if !tillMethods[models.ProviderPayme] || !tillMethods[models.ProviderClick] ||
		!tillMethods[models.ProviderUzum] {
		t.Fatal("a confirmed online payment cannot close a check")
	}
	// ⚠️ And it is not waited on for a bank's word either. `card` means a
	// terminal was used; there is no request of ours that could confirm it, so
	// a gate that included it would leave the cashier unable to close a check
	// that was genuinely paid.
	if bankConfirmed("card") || bankConfirmed("transfer") ||
		bankConfirmed(models.ProviderCash) || bankConfirmed(models.MethodDebt) {
		t.Fatal("a payment nobody can confirm is being waited on")
	}
}

// ⚠️ **A card scanned at the counter waits for the bank exactly as a QR does.**
// The two are easy to think of as opposites — one is asynchronous and one
// answers in the same request — and that is precisely the reasoning that would
// let a declined scan close a check: "it already answered, so it must be paid".
// It answered; the answer may have been no.
func TestAScannedCardIsAlsoTheBanksWord(t *testing.T) {
	for _, m := range []string{instore.ClickPass, instore.UzumFastPay} {
		if !bankConfirmed(m) {
			t.Fatalf("%s can close a check the bank never confirmed", m)
		}
		if !tillMethods[m] {
			t.Fatalf("%s cannot close a check even once the bank confirmed it", m)
		}
	}
	// And it may not be used to settle an old debt: the guest would be standing
	// over a closed sale while a fresh charge landed on it, and a repayment is
	// recorded against the drawer rather than against the original check.
	src := readSource(t, "tillpay.go")
	fn := between(t, src, "func (h *Handler) TillPayDebt", "\n}\n")
	if !strings.Contains(fn, "bankConfirmed(method)") {
		t.Fatal("a debt can be repaid through a rail that has no check to charge")
	}
}

// ⚠️ **A rail with no adapter can be configured and never enabled.** Payme GO
// and the terminal drivers are listed so an owner recognises the name and reads
// why — the fiscal providers' rule. Saving keys for one is fine; a till button
// that refuses every guest who presses it is not.
func TestARailWithNoAdapterCannotBeSwitchedOn(t *testing.T) {
	for _, p := range instore.Providers() {
		if p.Ready {
			continue
		}
		req := paymentSettingsRequest{InStore: &inStoreRequest{
			Rails: map[string]inStoreRailRequest{
				p.ID: {Enabled: true, ServiceID: "1", UserID: "1", SecretKey: "s"},
			},
		}}
		got := inStoreFrom(req, &models.PaymentSettings{})
		if got.Enabled[p.ID] {
			t.Fatalf("%s can be enabled with no adapter behind it", p.ID)
		}
		if got.Creds[p.ID].ServiceID != "1" {
			t.Fatalf("%s lost the credentials the owner typed in", p.ID)
		}
	}
}

// ⚠️ **A panel that says nothing about the counter rails must change nothing.**
// For the minutes after a deploy an open tab still holds the old settings page,
// and its next save posts no `inStore` at all. Decoded into a value rather than
// a pointer, that save writes an empty map over working credentials — silently,
// with the page reporting success. The same failure the fiscal drawers had, and
// the reason this is tested rather than remembered.
func TestAnOldSettingsPageDoesNotEraseTheCounterRails(t *testing.T) {
	current := &models.PaymentSettings{InStore: models.InStoreSettings{
		Enabled: map[string]bool{instore.ClickPass: true},
		Creds: map[string]models.InStoreCreds{
			instore.ClickPass: {ServiceID: "12345", UserID: "9", SecretKey: "live"},
		},
	}}
	got := inStoreFrom(paymentSettingsRequest{}, current)
	if !got.Enabled[instore.ClickPass] || got.Creds[instore.ClickPass].SecretKey != "live" {
		t.Fatal("an old panel wiped the counter rail credentials")
	}
}

// ⚠️ **An empty secret means "keep the stored one", never "erase it".** The
// panel cannot show the key it is editing, so an owner correcting a service id
// submits a blank secret — and the till stops taking cards with nothing on any
// screen to say why.
func TestCorrectingAServiceIdKeepsTheSecret(t *testing.T) {
	current := &models.PaymentSettings{InStore: models.InStoreSettings{
		Creds: map[string]models.InStoreCreds{
			instore.ClickPass: {ServiceID: "1", UserID: "9", SecretKey: "live"},
		},
	}}
	req := paymentSettingsRequest{InStore: &inStoreRequest{
		Rails: map[string]inStoreRailRequest{
			instore.ClickPass: {Enabled: true, ServiceID: "2", UserID: "9"},
		},
	}}
	got := inStoreFrom(req, current)
	if got.Creds[instore.ClickPass].SecretKey != "live" {
		t.Fatal("fixing a typo in the service id switched the rail off")
	}
	if got.Creds[instore.ClickPass].ServiceID != "2" {
		t.Fatal("the corrected service id was not saved")
	}
}
