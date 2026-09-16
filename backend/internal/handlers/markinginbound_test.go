package handlers

import (
	"os"
	"strings"
	"testing"
)

// The goods-in check, read out of the source it is written in.
//
// ⚠️ **These are rules about *when* nothing happens**, and nothing is what makes
// them dangerous to get wrong. A check that fired on an install which does not
// scan deliveries would refuse every sale of every marked bottle in the product
// on the day it shipped — and it would do it at a counter, with a customer
// waiting. So the conditions are pinned here rather than trusted to a reading.

func markingSource(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile("markinginbound.go")
	if err != nil {
		t.Fatal(err)
	}
	return string(src)
}

// ⚠️ **Off is the default and the branch decides.** Every restaurant running
// today scans only at the till.
func TestTheInboundCheckIsSilentUntilABranchAsksForIt(t *testing.T) {
	fn := between(t, markingSource(t), "func (h *Handler) markInboundRefusal", "\n}\n")
	if !strings.Contains(fn, "!branch.MarkingInbound") {
		t.Error("the check does not ask whether this branch scans deliveries")
	}
	if !strings.Contains(fn, `return "", nil`) {
		t.Error("a branch that does not scan deliveries is not let through silently")
	}
}

// ⚠️ **A voided line is not sold and withdraws nothing.** Asking it for a code
// would hold a check shut over a bottle that went back in the fridge — the same
// rule the shape check already follows.
func TestAVoidedLineIsNotAskedForItsCode(t *testing.T) {
	fn := between(t, markingSource(t), "func (h *Handler) markInboundRefusal", "\n}\n")
	if !strings.Contains(fn, "it.Void != nil") {
		t.Error("a voided line is still checked")
	}
}

// ⚠️ **Marked rather than deleted.** "This bottle was sold on that receipt" is
// the only question anybody asks afterwards, and a row that disappeared would
// leave a re-scanned code looking exactly like one that never arrived.
func TestASoldBottleKeepsItsRowAndItsReceipt(t *testing.T) {
	fn := between(t, markingSource(t), "func (h *Handler) markUnitsSold", "\n}\n")
	if strings.Contains(fn, "DeleteOne") || strings.Contains(fn, "DeleteMany") {
		t.Error("a sold bottle is deleted instead of marked")
	}
	if !strings.Contains(fn, `"soldAt"`) || !strings.Contains(fn, `"orderId"`) {
		t.Error("the sale is not recorded against the bottle")
	}
	// ⚠️ Only a bottle that is still unsold is claimed: without the filter a
	// second close would move the sale onto the newer receipt and the first one
	// would stop being answerable.
	if !strings.Contains(fn, `"soldAt": nil`) {
		t.Error("an already-sold bottle can be claimed by a second receipt")
	}
}

// ⚠️ **The unique index is the check, not a lookup before it.** Two people
// unpacking two boxes at two tills both find nothing and both insert; the index
// is the only thing that is true at the moment of writing.
//
// ⚠️ **Read out of `saveMarks` rather than the handler**, because the rule moved
// house when the phone got a camera: the panel and the app file codes through
// one function (handlers/staffmarking.go), and the rule belongs wherever the
// insert is. The test that keeps them one implementation is below.
func TestADuplicateCodeIsCaughtByTheIndexRatherThanALookup(t *testing.T) {
	fn := between(t, markingSource(t), "func (h *Handler) saveMarks", "\n}\n")
	if !strings.Contains(fn, "mongo.IsDuplicateKeyError") {
		t.Error("a duplicate is not recognised as one")
	}
	if strings.Contains(fn, "CountDocuments") {
		t.Error("a lookup stands where the index should")
	}
}

// ⚠️ **A box of forty with one unreadable sticker is not a failed box.** The
// thirty-nine are in the store room either way, and an all-or-nothing save is
// one that gets abandoned halfway through unpacking.
func TestOneBadStickerDoesNotLoseTheBox(t *testing.T) {
	fn := between(t, markingSource(t), "func (h *Handler) saveMarks", "\n}\n")
	bad := strings.Index(fn, "bad = append(bad, raw)")
	if bad < 0 {
		t.Fatal("an unreadable code is not reported")
	}
	if !strings.Contains(fn[bad:bad+60], "continue") {
		t.Error("an unreadable code stops the save instead of being skipped")
	}
}

// ⚠️ **Two doors, one implementation.** The panel's receiving screen and the
// phone's camera both file marking codes, and a second insert written beside
// either one is how the same box comes to be recorded differently depending on
// what somebody was holding — a disagreement that surfaces weeks later as a
// sale the till refuses. The rules above are pinned to `saveMarks`; this is
// what keeps the handlers going through it.
func TestBothDoorsFileCodesThroughOneFunction(t *testing.T) {
	src := markingSource(t)
	fn := between(t, src, "func (h *Handler) AdminReceiveMarks", "\n}\n")
	if !strings.Contains(fn, "h.saveMarks(") {
		t.Error("the panel's screen does not file through saveMarks")
	}
	if strings.Contains(fn, "InsertOne") {
		t.Error("the handler writes its own rows beside the shared function")
	}

	phone, err := os.ReadFile("staffmarking.go")
	if err != nil {
		t.Fatal(err)
	}
	staff := between(t, string(phone), "func (h *Handler) StaffReceiveMarks", "\n}\n")
	if !strings.Contains(staff, "h.saveMarks(") {
		t.Error("the phone does not file through saveMarks")
	}
	if strings.Contains(staff, "InsertOne") {
		t.Error("the phone writes its own rows beside the shared function")
	}
	// ⚠️ **The branch comes off the employee, never the request.** A phone that
	// could name a branch could file somebody else's delivery — and a code is
	// unique across the platform, so filing it wrongly takes it away from the
	// branch that actually has the bottle.
	if !strings.Contains(staff, "s.BranchID") {
		t.Error("the phone does not take the branch from the employee")
	}
}
