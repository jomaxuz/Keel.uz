package handlers

import (
	"strings"
	"testing"

	"restaurant-backend/internal/models"
)

// ⚠️ **A blank account may not count.** The usual "empty value keeps today's
// behaviour" rule is inverted here for the same reason it is on the pass
// screen: a permission everybody holds by default is not a permission. Unlike
// the pass, nothing needs grandfathering — the screen is new, so refusing takes
// nothing away from anybody who had it yesterday.
func TestCountingNeedsItsOwnPermission(t *testing.T) {
	if stockDenial(models.Staff{}) == "" {
		t.Fatal("a blank staff record could count the store")
	}

	waiter := models.Staff{IsActive: true, Perms: []string{models.PermWaiter}}
	if stockDenial(waiter) == "" {
		t.Fatal("a waiter could count the store")
	}

	counter := models.Staff{IsActive: true, Perms: []string{models.PermStock}}
	if why := stockDenial(counter); why != "" {
		t.Fatalf("a permitted, active employee was refused: %q", why)
	}
}

// ⚠️ A staff token outlives a shift by a long way, so somebody deactivated this
// morning still holds a working one this afternoon. The same hole the KDS had.
func TestADeactivatedEmployeeCannotCount(t *testing.T) {
	fired := models.Staff{IsActive: false, Perms: []string{models.PermStock}}
	if stockDenial(fired) == "" {
		t.Fatal("a deactivated account kept the counting screen")
	}
	// And the two refusals differ: one sends the person to their manager, the
	// other to whoever switched the account off.
	waiter := models.Staff{IsActive: true}
	if stockDenial(fired) == stockDenial(waiter) {
		t.Fatal("the two refusals read the same")
	}
}

// ⚠️ **The branch comes off the employee, and the posted one is discarded.** A
// phone that could name a branch could count somebody else's store — and a
// count is the one write that resets the baseline every later shortfall is
// measured from.
func TestAPhoneCountsItsOwnBranch(t *testing.T) {
	src := readSource(t, "staffstock.go")
	fn := between(t, src, "func (h *Handler) StaffSaveStocktake", "\n}\n")

	if !strings.Contains(fn, "in.BranchID = s.BranchID") {
		t.Fatal("the branch is no longer taken off the employee")
	}
	// And it saves through the same function the panel does, or the frozen
	// expected figure and the "explain the difference" rule drift apart.
	if !strings.Contains(fn, "h.saveStocktake(") {
		t.Fatal("the phone no longer saves through the shared implementation")
	}
}
