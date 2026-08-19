package handlers

import (
	"strings"
	"testing"
)

// ⚠️ Joining two checks is the mirror of splitting them, and it has one rule
// that splitting does not: the check that loses its food must **stop being a
// sale** without ceasing to exist. Deleting it would erase the voids written on
// it, the number that may already be on a printed bill, and who opened it.
func TestMergeCancelsTheAbsorbedCheckRatherThanDeletingIt(t *testing.T) {
	src := readSource(t, "tillmerge.go")
	fn := between(t, src, "func (h *Handler) StaffMergeChecks", "\n}\n")

	if strings.Contains(fn, "DeleteOne") {
		t.Fatal("the absorbed check is being deleted — its voids and number go with it")
	}
	if !strings.Contains(fn, "models.StatusCancelled") {
		t.Fatal("the absorbed check is left open — its table stays busy and its total is still countable")
	}
	// The reason has to name the bill the food ended up on: "cancelled" alone
	// is the answer nobody can act on a month later.
	if !strings.Contains(fn, "cancelReason") || !strings.Contains(fn, "to.Number") {
		t.Fatal("the cancelled check does not say where its food went")
	}
	// ⚠️ Covers add up. Two tables pushed together seat both parties, and
	// average-per-guest is one of the two numbers a dining room is run on.
	if !strings.Contains(fn, "to.Check.Guests += from.Check.Guests") {
		t.Fatal("the joined table lost half its covers")
	}
	// Voided lines stay where they were written off.
	if !strings.Contains(fn, "it.Live()") {
		t.Fatal("voided lines are moving onto a bill whose waiter never took that dish off")
	}
	// ⚠️ Destination written first: if the second write fails the food is on
	// two checks and a human can see both. The other order leaves it on none.
	if strings.Index(fn, "checkFilter(to.ID") > strings.Index(fn, "checkFilter(from.ID") {
		t.Fatal("the source is emptied before the destination is written")
	}
	// The branch lives inside both filters, as everywhere else in the till.
	if strings.Count(fn, "s.BranchID") < 3 {
		t.Fatal("a check from another branch can be named as the destination")
	}
}
