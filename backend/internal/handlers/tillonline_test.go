package handlers

import (
	"strings"
	"testing"

	"restaurant-backend/internal/models"
)

// ⚠️ **Paid is checked before anything else.** A guest who paid with Payme is
// settled whether the order is delivered, collected, or changed halfway — and
// asking any further question about it can only produce a wrong answer, which
// on this screen means a cashier asking a courier for money nobody took.
func TestPaidOnlineOwesNothing(t *testing.T) {
	for _, typ := range []string{"delivery", "pickup"} {
		o := &models.Order{
			Type: typ, PaymentMethod: "payme", PaymentStatus: models.PayPaid,
		}
		if got := settlementOf(o); got != SettleNothing {
			t.Fatalf("%s paid online still owes %q", typ, got)
		}
	}
}

// ⚠️ **Cash and card are the same answer, and that is worth stating.** Both are
// money that has not moved yet, and both are collected by whoever is standing
// in front of the guest: the courier on a delivery — with notes or with the
// terminal they carry — and the cashier at a pickup.
func TestUnpaidIsCollectedByWhoeverFacesTheGuest(t *testing.T) {
	cases := []struct {
		typ, method string
		want        Settlement
	}{
		{"delivery", "cash", SettleFromCourier},
		{"delivery", "card", SettleFromCourier},
		{"pickup", "cash", SettleAtCounter},
		{"pickup", "card", SettleAtCounter},
	}
	for _, c := range cases {
		o := &models.Order{Type: c.typ, PaymentMethod: c.method}
		if got := settlementOf(o); got != c.want {
			t.Fatalf("%s/%s = %q, want %q", c.typ, c.method, got, c.want)
		}
	}
}

// ⚠️ **An order with no payment status is unpaid, not paid.** Every order that
// predates online payment has an empty field, and reading it as settled would
// hide exactly the cash orders this list exists to surface — silently, and in
// the direction that loses money.
func TestAnEmptyPaymentStatusIsNotSettled(t *testing.T) {
	o := &models.Order{Type: "delivery", PaymentMethod: "cash"}
	if got := settlementOf(o); got == SettleNothing {
		t.Fatal("an order that has never been paid was treated as settled")
	}
}

// A dine-in order has a check of its own and never reaches this list; if one
// somehow does, it must not tell the counter to chase a courier.
func TestDineInAsksTheCounterForNothing(t *testing.T) {
	o := &models.Order{Type: "dinein", PaymentMethod: "cash"}
	if got := settlementOf(o); got != SettleNothing {
		t.Fatalf("a table was put on the delivery list as %q", got)
	}
}

// ⚠️ **Only what is genuinely still owed goes into the total.** Including
// orders already paid online would give the counter a figure that disagrees
// with the drawer by the size of a good day's card takings — and the cashier
// would trust the screen over the money in front of them.
func TestTheTotalCountsOnlyWhatIsOwed(t *testing.T) {
	src := readLossSource(t, "tillonline.go")
	if !strings.Contains(src, "if row.Settle != SettleNothing {") {
		t.Fatal("settled orders are being added to what the counter is owed")
	}
}
