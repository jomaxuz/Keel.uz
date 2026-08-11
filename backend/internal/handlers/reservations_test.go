package handlers

import (
	"testing"

	"restaurant-backend/internal/models"
)

func tables(spec ...int) []models.FloorTable {
	out := make([]models.FloorTable, 0, len(spec))
	for i, seats := range spec {
		out = append(out, models.FloorTable{
			ID:       string(rune('a' + i)),
			Number:   string(rune('1' + i)),
			Seats:    seats,
			IsActive: true,
		})
	}
	return out
}

// ⚠️ **The smallest table that fits, not the first one that does.**
//
// A restaurant that hides its floor plan is not choosing to care less about
// which table a party gets — it is choosing not to make the *guest* decide. So
// the rule that replaces the guest's tap has to be the one a host would use:
// seating two people at the ten-seater because it came first in the list is how
// the party of ten an hour later gets turned away, and nothing on any screen
// shows it happening. Every individual booking looks correct.
func TestPickTableTakesTheSmallestThatFits(t *testing.T) {
	free := tables(10, 2, 6, 4)

	if got := pickTable(free, 2); got.Seats != 2 {
		t.Errorf("party of 2 got a %d-seater, want 2", got.Seats)
	}
	if got := pickTable(free, 3); got.Seats != 4 {
		t.Errorf("party of 3 got a %d-seater, want 4", got.Seats)
	}
	if got := pickTable(free, 10); got.Seats != 10 {
		t.Errorf("party of 10 got a %d-seater, want 10", got.Seats)
	}
}

// Nothing big enough is a real answer, and it has to be one: the alternative is
// accepting a booking the room cannot seat, which the restaurant only discovers
// when the guests are standing in the doorway.
func TestPickTableRefusesWhenNothingFits(t *testing.T) {
	if got := pickTable(tables(2, 4), 8); got != nil {
		t.Errorf("party of 8 was given a %d-seater", got.Seats)
	}
	if got := pickTable(nil, 2); got != nil {
		t.Error("a table was produced from an empty plan")
	}
}

// ⚠️ A table with no seat count means "unmeasured", not "seats nobody". It is
// usable — an unfilled field in the plan editor is ordinary — but it sorts last,
// so a table somebody actually measured is preferred while it still saves the
// booking when it is the only one left.
func TestPickTablePrefersMeasuredTables(t *testing.T) {
	free := []models.FloorTable{
		{ID: "x", Seats: 0, IsActive: true},
		{ID: "y", Seats: 4, IsActive: true},
	}
	if got := pickTable(free, 2); got.ID != "y" {
		t.Errorf("picked the unmeasured table over a 4-seater")
	}
	if got := pickTable(free[:1], 2); got == nil || got.ID != "x" {
		t.Error("an unmeasured table was refused when it was the only one free")
	}
}
