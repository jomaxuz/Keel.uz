package handlers

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

// ⚠️ **What was loaded and what arrived are two facts, and netting them erases
// the only number the paper's three signatures exist to produce.** Thirty kilos
// left the central store and twenty-eight were counted off the van: the sending
// shelf lost thirty, the receiving shelf gained twenty-eight, and the two are on
// neither — a finding about the journey rather than about either storekeeper.
func TestWhatIsLoadedAndWhatArrivesAreTwoFacts(t *testing.T) {
	got := 28.0
	line := models.DispatchLine{Qty: 30, Got: &got}
	if line.Arrived() != 28 {
		t.Errorf("arrived = %v, want 28", line.Arrived())
	}
	if line.Lost() != 2 {
		t.Errorf("lost = %v, want 2", line.Lost())
	}
	// ⚠️ **Not yet signed for is not "arrived empty".** Before anybody counts
	// it off the van the honest reading of the far end is "what was sent",
	// because that is the only figure anybody has written down.
	unsigned := models.DispatchLine{Qty: 30}
	if unsigned.Arrived() != 30 {
		t.Errorf("an unsigned line arrived as %v, want 30", unsigned.Arrived())
	}
	if unsigned.Lost() != 0 {
		t.Errorf("an unsigned line reported %v lost", unsigned.Lost())
	}
	// And a discrepancy is only reported once somebody has actually looked.
	d := models.Dispatch{Lines: []models.DispatchLine{line}}
	if d.LostLines() != nil {
		t.Error("an unaccepted dispatch reported a shortfall in transit")
	}
}

// ⚠️ **Nothing arrives on a shelf before somebody signs for it.** Counting a van
// onto the receiving branch's balance early shows stock it cannot cook with —
// and the stop list, which reads the same figures, would let the kitchen sell
// it. The sending shelf loses it immediately, because it physically left.
func TestAVanReachesTheShelfOnlyWhenItIsSignedFor(t *testing.T) {
	src := readSource(t, "dispatch.go")
	fn := between(t, src, "func (h *Handler) dispatchedInPeriod", "\n}\n")
	if !strings.Contains(fn, "out[l.IngredientID] += l.Qty") {
		t.Fatal("the sending shelf no longer loses what was loaded")
	}
	if !strings.Contains(fn, "if d.Accepted() {") {
		t.Fatal("an unsigned van is already on the receiving branch's shelf")
	}
	if !strings.Contains(fn, "in[l.IngredientID] += l.Arrived()") {
		t.Fatal("the receiving shelf gains what was sent rather than what arrived")
	}
}

// ⚠️ **Both halves reach the expected balance, or a central store reports a
// shortfall the size of everything it ever dispatched** — attributed, at the
// next count, to whoever counted it. The same lesson production's two halves
// taught, arriving again with a van.
func TestTheExpectedBalanceCountsBothEndsOfTheVan(t *testing.T) {
	fn := between(t, readSource(t, "stocktake.go"),
		"func (h *Handler) expectedStockByWarehouse", "\n}\n")
	if !strings.Contains(fn, "h.dispatchedInPeriod(") {
		t.Fatal("the expected balance does not know about vans at all")
	}
	if !strings.Contains(fn, "add(vanIn, 1)") || !strings.Contains(fn, "add(vanOut, -1)") {
		t.Fatal("a van moves the balance in only one direction")
	}
}

// ⚠️ **Only the branch a van was sent to may sign for it, and the branch lives
// inside the filter.** An id alone must never select a document (scope.go): a
// branch signing for somebody else's van would move goods onto its own shelves
// and off nobody's — and it is the one write in this file that changes two
// balances at once.
func TestOnlyTheReceivingBranchSignsForAVan(t *testing.T) {
	fn := between(t, readSource(t, "dispatch.go"),
		"func (h *Handler) AdminAcceptDispatch", "\n}\n")
	if !strings.Contains(fn, `filter["toBranchId"] = cond`) {
		t.Fatal("acceptance is not narrowed to the receiving branch")
	}
	// ⚠️ And once: the absence of a signature is part of the filter rather than
	// a check before the write, so two branch managers on two screens cannot
	// both sign and have the second silently replace the first.
	if !strings.Contains(fn, `"acceptedAt": bson.M{"$exists": false}`) {
		t.Fatal("a van can be signed for twice")
	}
}

// ⚠️ **The sending branch is the one in view, never one named in the body.**
// Taking it from the request would let one branch empty another's shelves by
// pasting an id — and the goods would arrive somewhere real, so no screen would
// look wrong.
func TestTheSenderIsTheBranchInViewNotOneInTheBody(t *testing.T) {
	fn := between(t, readSource(t, "dispatch.go"),
		"func (h *Handler) AdminCreateDispatch", "\n}\n")
	if !strings.Contains(fn, "scope, branch, brand, err := h.stockBranch(r)") {
		t.Fatal("the sending branch is not resolved from the request's scope")
	}
	if strings.Contains(fn, "FromBranchID: from") ||
		strings.Contains(fn, "req.FromBranchID") {
		t.Fatal("the sending branch comes from the request body")
	}
	// The far end has to be a branch of the same brand — an ingredient id means
	// "this brand's catalogue", and a shelf in another brand has no row for it.
	if !strings.Contains(fn, `filter["brandId"] = brand`) {
		t.Fatal("a van can be sent into another brand's branch")
	}
}

// The scope filter's branch condition has to survive being asked of two
// different fields — a dispatch names two branches, and a manager's clamp must
// travel with it exactly as it was computed.
func TestTheBranchConditionIsAskedOfBothEnds(t *testing.T) {
	one := primitive.NewObjectID()
	two := primitive.NewObjectID()
	other := primitive.NewObjectID()

	mine := matcher(one)
	if !mine(one) || mine(other) {
		t.Error("a single-branch clamp matched the wrong branch")
	}
	many := matcher(bson.M{"$in": []primitive.ObjectID{one, two}})
	if !many(one) || !many(two) || many(other) {
		t.Error("a multi-branch scope matched the wrong branches")
	}
	// ⚠️ An unknown shape matches nothing rather than everything: the failure
	// direction has to be "this branch sees no vans", never "every branch sees
	// every van".
	if matcher("nonsense")(one) {
		t.Fatal("an unrecognised scope let every van through")
	}
	if _, ok := branchCond(bson.M{}); ok {
		t.Fatal("an unscoped filter claimed to name a branch")
	}
}

// ⚠️ **A morning is one act, not five.** The central store stands at a shelf
// with five slips and writes the rows across all of them — which is why the
// paper form is four slips on one sheet. Five separate saves would be five times
// the typing and, worse, a failure halfway leaves two branches loaded and three
// not, with a driver already holding the paper for all five.
func TestAMorningOfVansIsWrittenInOneCall(t *testing.T) {
	fn := between(t, readSource(t, "dispatch.go"),
		"func (h *Handler) AdminCreateDispatch", "\n}\n")

	if !strings.Contains(fn, "Dispatches.InsertMany") {
		t.Fatal("the slips are written one at a time — a failure halfway leaves half a morning loaded")
	}
	// ⚠️ The numbering is worked out once and counted up: asking the database
	// per slip gives five slips written in the same second the same number, and
	// the number is what somebody holding two of them tells them apart by.
	if !strings.Contains(fn, "next := h.dispatchCount(") || !strings.Contains(fn, "next++") {
		t.Fatal("each slip asks the database for its own number")
	}
	// ⚠️ An empty column is a branch that ordered nothing today — skipped, not
	// refused, or the storekeeper hunts for which column it was.
	if !strings.Contains(fn, "if len(lines) == 0 {\n\t\t\tcontinue") {
		t.Fatal("one empty column refuses the whole morning")
	}
	// ⚠️ And the same branch twice is somebody having typed a column twice: the
	// second would take stock off the shelf again while looking, on paper, like
	// a legitimate second van.
	if !strings.Contains(fn, "if seen[to] {") {
		t.Fatal("one branch can appear in two columns of the same sheet")
	}
	// The old single-slip shape still works — the buyer's phone and any script
	// written against it must not break.
	if !strings.Contains(fn, "slips = []dispatchIn{{ToBranchID: req.ToBranchID") {
		t.Fatal("a body with one toBranchId no longer loads a van")
	}
}
