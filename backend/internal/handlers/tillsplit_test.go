package handlers

import (
	"strings"
	"testing"
)

// ⚠️ The rules a split has to keep, all of them learned from the way the
// nearby handlers already work. Source-level because the write path needs a
// database; each of these is a one-line change away from being lost.
func TestSplitKeepsTheRulesTheDiningRoomDependsOn(t *testing.T) {
	src := readSource(t, "tillsplit.go")
	fn := between(t, src, "func (h *Handler) StaffSplitCheck", "\n}\n")

	// A waiter's action. Asking for the cashier's code at the table is how the
	// cashier's PIN ends up known to every person in the building.
	if !strings.Contains(fn, "models.PermWaiter") {
		t.Fatal("splitting no longer runs on the waiter's permission")
	}
	// A voided line is the record of food written off *this* check.
	if !strings.Contains(fn, "it.Live()") {
		t.Fatal("voided lines can now be moved onto the new check")
	}
	// ⚠️ The whole point of the feature: without this the sales screen counts
	// a divided table twice.
	if !strings.Contains(fn, "SplitFromID: from.ID") {
		t.Fatal("the new check no longer records where it was split from")
	}
	// Covers belong to the table, once.
	if strings.Contains(fn, "Guests:") {
		t.Fatal("guests are being copied onto the split — every divided table would double its covers")
	}
	// Food already cooking makes the new check the kitchen's too.
	if !strings.Contains(fn, "it.FiredAt != nil") {
		t.Fatal("fired lines can land on a check the kitchen has never heard of")
	}
	// ⚠️ The source write can fail after the new check exists. Both halves
	// holding the same food charges the guest twice, which is worse than the
	// split simply not happening.
	if !strings.Contains(fn, "h.Store.Orders.DeleteOne") {
		t.Fatal("a failed split can leave the food on both checks")
	}
	// Splitting everything is refused: an empty check nobody can pay sits on
	// the floor screen looking like a table waiting to order.
	if !strings.Contains(fn, "anyLive(kept)") {
		t.Fatal("the whole check can be split off, leaving an empty one behind")
	}
}
