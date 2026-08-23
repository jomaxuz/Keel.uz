package handlers

import (
	"strings"
	"testing"
)

// The rule that makes a till count worth doing.
//
// ⚠️ Expected cash includes what couriers have **handed over**, never what
// they have collected. Until the courier walks in, the money is in their
// pocket: counting it as if it were in the drawer would report a shortfall on
// every shift with a delivery still out, and a warning that fires every day is
// one nobody reads by Thursday.
func TestExpectedCashExcludesMoneyStillWithCouriers(t *testing.T) {
	f := cashFigures{
		OpeningFloat: 100_000,
		CounterCash:  250_000, // pickup and dine-in, paid at the till
		Settlements:  400_000, // couriers who have come back
		ManualIn:     0,
		ManualOut:    75_000,
		WithCouriers: 921_000, // still out on the road
	}
	f.Expected = f.OpeningFloat + f.CounterCash + f.Settlements + f.ManualIn - f.ManualOut

	if f.Expected != 675_000 {
		t.Fatalf("expected = %d, want 675 000", f.Expected)
	}
	// The number a shortfall investigation actually needs, kept beside the
	// total rather than folded into it.
	if f.WithCouriers == 0 {
		t.Error("cash still with couriers must be reported separately")
	}
}

// A negative variance is a shortfall, a positive one is a surplus, and both
// have to survive into the log with the right word.
func TestVarianceLabels(t *testing.T) {
	cases := map[int]string{0: "farqsiz", -5_000: "kamomad", 5_000: "ortiqcha"}
	for v, want := range cases {
		if got := varianceLabel(v); got != want {
			t.Errorf("%d → %q, want %q", v, got, want)
		}
	}
}

// Money cannot leave a drawer it is not in.
//
// ⚠️ **Nothing checked this, and the failure was silent.** A payout recorded
// against an empty till saved like any other entry, drove the expected balance
// negative, and surfaced only at the count hours later — as a difference nobody
// could explain, because the entry itself looked perfectly ordinary. A
// shortfall discovered at midnight is a shortfall blamed on whoever counted.
func TestCashOutCannotExceedTheDrawer(t *testing.T) {
	// The ordinary payout: less than what is there.
	if msg := cashOutRefusal(50_000, 200_000); msg != "" {
		t.Errorf("a payout within the drawer must be allowed, got %q", msg)
	}
	// ⚠️ Exactly emptying the drawer is allowed. It is what happens at the end
	// of every evening, and a rule that refused it would be worked around on
	// the first night — by splitting the payout in two, which is worse.
	if msg := cashOutRefusal(200_000, 200_000); msg != "" {
		t.Errorf("emptying the drawer must be allowed, got %q", msg)
	}
	// One so'm over is over.
	if msg := cashOutRefusal(200_001, 200_000); msg == "" {
		t.Error("a payout larger than the drawer must be refused")
	}
	// ⚠️ The refusal carries the number, because the cashier's next move
	// depends on it — retype a smaller amount, or go and find a manager.
	msg := cashOutRefusal(500_000, 0)
	if msg == "" {
		t.Fatal("an empty drawer must refuse every payout")
	}
	if !strings.Contains(msg, "0") {
		t.Errorf("the refusal must say how much is there, got %q", msg)
	}
}
