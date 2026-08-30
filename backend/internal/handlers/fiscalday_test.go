package handlers

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ⚠️ The two filters must not be the same filter, and the difference is a real
// hole rather than a nicety.
//
// The alert waits five minutes before calling a pending filing a problem — a
// sale rung up thirty seconds ago is not news. The Z-report guard cannot wait
// at all: a receipt still in flight when the day is totalled either drops out
// of the day's figures or lands on tomorrow's, and neither can be corrected
// afterwards. The sales inside that five-minute window at closing time are
// exactly the last ones of the evening.
//
// Reusing the alert's filter for the guard read as obviously right and was
// wrong for precisely the receipts most likely to exist when somebody presses
// "close the day".
func TestZReportGuardHasNoGracePeriod(t *testing.T) {
	guard := anyUnfiledFilter()
	// ⚠️ The guard does branch — over the sale and its reversal, which are two
	// documents on one check. What it must never branch over is *time*: a
	// clause keyed on a timestamp is the grace period coming back.
	clauses, ok := guard["$or"].([]bson.M)
	if !ok {
		t.Fatalf("the guard no longer covers both filings: %v", guard)
	}
	for _, c := range clauses {
		for k := range c {
			if k != "fiscal.status" && k != "fiscalRefund.status" {
				t.Fatalf("the Z-report guard grew a %q branch — it must not wait", k)
			}
		}
	}
	for _, field := range []string{"fiscal.status", "fiscalRefund.status"} {
		var pending, failed bool
		for _, c := range clauses {
			m, is := c[field].(bson.M)
			if !is {
				continue
			}
			statuses, _ := m["$in"].([]string)
			for _, s := range statuses {
				pending = pending || s == "pending"
				failed = failed || s == "failed"
			}
		}
		if !pending || !failed {
			t.Fatalf("%s must catch both pending and failed: %v", field, clauses)
		}
	}

	// And the alert's own filter must keep its grace period, or every sale rings
	// a bell the moment it is paid for.
	alert := unfiledFiscalFilter(time.Now())
	if _, has := alert["$or"]; !has {
		t.Fatal("the alert lost its grace period — a fresh sale is not news")
	}
}

// ⚠️ `$in`, never `$nin`. A missing field *matches* `$nin` in Mongo, so "not
// filed" would sweep in every till sale rung up before a register was ever
// connected — and every sale at a restaurant that has none. Those never owed a
// receipt. The same trap pendingTillFilter documents, read from the other side.
func TestUnfiledFiltersNeverUseNotIn(t *testing.T) {
	for name, f := range map[string]bson.M{
		"guard": anyUnfiledFilter(),
		"alert": unfiledFiscalFilter(time.Now()),
	} {
		if hasNin(f) {
			t.Fatalf("%s uses $nin: a sale that never owed a receipt would match", name)
		}
	}
}

func hasNin(v any) bool {
	switch t := v.(type) {
	case bson.M:
		for k, sub := range t {
			if k == "$nin" || hasNin(sub) {
				return true
			}
		}
	case []bson.M:
		for _, sub := range t {
			if hasNin(sub) {
				return true
			}
		}
	}
	return false
}

// ⚠️ Scoping a shared filter must copy it. These filters are handed to the
// alert, the till list and this guard in the same process; one that quietly
// kept a branch from whoever called it last is how one restaurant ends up
// counting another's receipts — and it would only show under load.
func TestWithBranchDoesNotMutateTheSharedFilter(t *testing.T) {
	base := anyUnfiledFilter()
	branch := primitive.NewObjectID()

	scoped := withBranch(base, branch)
	if scoped["branchId"] != branch {
		t.Fatal("withBranch did not scope the filter")
	}
	if _, leaked := base["branchId"]; leaked {
		t.Fatal("withBranch mutated the shared filter")
	}

	// A second call with a different branch must not see the first one's.
	other := primitive.NewObjectID()
	if got := withBranch(base, other)["branchId"]; got != other {
		t.Fatalf("branchId = %v, want the branch just asked for", got)
	}
}
