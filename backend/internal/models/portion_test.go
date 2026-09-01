package models

import "testing"

// Money first: the part price is what the guest is charged and what the
// receipt, the fiscal document and every report are built from.
func TestPortionPriceRoundsRatherThanTruncates(t *testing.T) {
	cases := []struct {
		unit, percent, want int
	}{
		{23000, 50, 11500},
		// ⚠️ 11 499.5 — truncated this loses the restaurant a so'm on every
		// half sold, which is invisible on a receipt and real over a year.
		{22999, 50, 11500},
		{20000, 75, 15000},
		{10000, 33, 3300},
		// A whole one, said in both the ways a client can say it.
		{45000, 100, 45000},
		{45000, 0, 45000},
	}
	for _, c := range cases {
		if got := PortionPrice(c.unit, c.percent); got != c.want {
			t.Errorf("PortionPrice(%d, %d) = %d, want %d",
				c.unit, c.percent, got, c.want)
		}
	}
}

// What the store is told. ⚠️ Zero and 100 are the same answer: a line written
// before parts existed carries no portion at all, and it is a whole one.
func TestPortionFactorTreatsZeroAsWhole(t *testing.T) {
	if got := (OrderItem{}).PortionFactor(); got != 1 {
		t.Fatalf("a line with no portion counted as %g", got)
	}
	if got := (OrderItem{Portion: 100}).PortionFactor(); got != 1 {
		t.Fatalf("a whole portion counted as %g", got)
	}
	if got := (OrderItem{Portion: 50}).PortionFactor(); got != 0.5 {
		t.Fatalf("a half counted as %g", got)
	}
}

// ⚠️ **The dish decides, and the server asks it.** A till hides the control for
// a dish with no parts, but a request can still name one — and a half sold on a
// dish that does not divide is a half-price sale of a whole thing.
func TestAllowsPortionRefusesWhatTheDishDoesNotDivideInto(t *testing.T) {
	bread := MenuItem{Portions: []int{50}}
	water := MenuItem{}

	if !bread.AllowsPortion(50) {
		t.Fatal("a half was refused on a dish that sells halves")
	}
	if bread.AllowsPortion(25) {
		t.Fatal("a quarter was allowed on a dish that only sells halves")
	}
	// A whole one is always sellable, on every dish, however it is spelled.
	for _, p := range []int{0, 100} {
		if !water.AllowsPortion(p) {
			t.Fatalf("a whole portion (%d) was refused", p)
		}
	}
	if water.AllowsPortion(50) {
		t.Fatal("half a sealed bottle was allowed")
	}
}
