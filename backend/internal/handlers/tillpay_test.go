package handlers

import (
	"strings"
	"testing"

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

	if !strings.Contains(fn, "tillOnlineMethods[method]") ||
		!strings.Contains(fn, "models.PayPaid") {
		t.Fatal("a check can be closed on a payment nobody confirmed")
	}
	// Before the sale is written, not after: a check that has already been
	// marked delivered cannot be un-delivered by a refusal further down.
	guard := strings.Index(fn, "tillOnlineMethods[method]")
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
}
