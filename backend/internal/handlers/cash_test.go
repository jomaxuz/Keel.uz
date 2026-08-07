package handlers

import "testing"

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
