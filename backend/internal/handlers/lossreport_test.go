package handlers

import (
	"os"
	"strings"
	"testing"
)

// ⚠️ **A void is counted against whoever did it, never whoever allowed it.**
//
// The override exists so a waiter can act at nine on a Friday without learning
// a manager's PIN — the whole argument is written at the top of
// tilloverride.go, and it ends with "blaming the manager for every write-off in
// the building". A report that tallied voids under `AuthByID` would reintroduce
// exactly that, and would do it in the one place a restaurant goes looking for
// somebody to blame.
func TestVoidsAreCountedAgainstTheDoer(t *testing.T) {
	src := readLossSource(t, "lossreport.go")
	if !strings.Contains(src, "get(it.Void.ByID, it.Void.By)") {
		t.Fatal("voids are no longer tallied against the person who did them")
	}
	if strings.Contains(src, "get(it.Void.AuthByID") {
		t.Fatal("the authorising manager is being blamed for other people's voids")
	}
}

// ⚠️ **A discount nobody chose is measured against nobody.** A promotion that
// fired because a basket matched its rule, or points a guest spent, carries no
// name — and counting those would put the busiest cashier at the top of a list
// about judgement, which is the opposite of what it is for.
func TestAutomaticDiscountsBelongToNobody(t *testing.T) {
	src := readLossSource(t, "lossreport.go")
	if !strings.Contains(src, `d.ByID.IsZero() && d.By == ""`) {
		t.Fatal("rule-driven discounts are being attributed to a person")
	}
}

// ⚠️ **The report refuses to draw for one person.** With no colleagues every
// share is 100% of itself, and a table of confident-looking numbers about one
// named employee is worse than no table — it is an accusation with arithmetic
// on it.
func TestOnePersonIsNotComparable(t *testing.T) {
	src := readLossSource(t, "lossreport.go")
	if !strings.Contains(src, `"comparable": len(rows) > 1`) {
		t.Fatal("the report no longer says when it cannot be read")
	}
}

// ⚠️ **Per thousand, not per cent.** Two voids in four hundred checks is 0% at
// one decimal and 5‰ here, and the distance between 5‰ and 40‰ is the entire
// content of this report. Rounding it into zero leaves a table of zeros.
func TestSharesAreFineGrainedEnoughToShowAnything(t *testing.T) {
	src := readLossSource(t, "lossreport.go")
	if !strings.Contains(src, "* 1000 / int64(t.Checks)") ||
		!strings.Contains(src, "* 1000 / t.Sales") {
		t.Fatal("shares are computed at a resolution that hides the signal")
	}
}

// ⚠️ **Somebody with no recorded id still gets a row.** Those are the oldest
// checks, from before the till kept ids. Dropping them would quietly shrink the
// denominator every other row is divided by — so the people still working would
// each look worse, for a reason nothing on the screen explains.
func TestChecksWithNoIdStillCount(t *testing.T) {
	src := readLossSource(t, "lossreport.go")
	if !strings.Contains(src, `key = "name:" + name`) {
		t.Fatal("old checks are being dropped out of the denominator")
	}
}

// ⚠️ **Cash is tallied apart from everything else**, because it is the only
// tender that can leave in a pocket: a card refund goes back to a card.
func TestCashIsCountedSeparately(t *testing.T) {
	src := readLossSource(t, "lossreport.go")
	if !strings.Contains(src, `o.PaymentMethod == "cash"`) {
		t.Fatal("cash and card are being counted as one thing")
	}
}

func readLossSource(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
