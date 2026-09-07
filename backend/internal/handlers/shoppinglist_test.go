package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **A negative shelf is a measurement error, not a bigger order.** The
// balance can go below zero when a delivery has not been entered or a card
// overstates what a dish uses; ordering against that number buys the shortfall
// twice, at exactly the moment the figures are already least trustworthy.
func TestANegativeShelfIsOrderedAsEmptyNotAsNegative(t *testing.T) {
	src := readSource(t, "shoppinglist.go")
	// ⚠️ The list itself moved into `shoppingList` when the buyer's phone
	// needed the same answer as the panel. The guard follows the arithmetic:
	// pinned to the handler, which now only wraps it, this would go on passing
	// against a function that computes nothing.
	fn := between(t, src, "func (h *Handler) shoppingList", "\n}\n")

	// ⚠️ The expression moved when the forecast arrived — what is on hand is
	// now "the shelf, floored at zero, plus what is already on somebody's
	// list" — but the floor is the same fact and both rules read it.
	if !strings.Contains(fn, "have := math.Max(onHand, 0) + requested[in.ID]") {
		t.Fatal("a shelf below zero is being ordered against its negative figure")
	}
	if strings.Contains(fn, "in.MinQty - onHand") {
		t.Fatal("the minimum rule reads the raw balance again")
	}
}

// ⚠️ **Only where a minimum was set, and never a derived prep item.** Zero
// means "do not warn me" — nobody tracks a minimum for cinnamon — and a sauce
// cooked as it goes is not bought from anybody. Either on the list is how a
// list stops being read.
//
// ⚠️ **`DerivedOnly`, not `MadeInHouse`.** A batched prep item arrives from the
// central kitchen in a tub and does run out, so it belongs on the list — and it
// is the item somebody most needs warning about, because the answer takes a day
// rather than a phone call.
func TestTheListOnlyHoldsThingsSomebodyAskedToBeWarnedAbout(t *testing.T) {
	src := readSource(t, "shoppinglist.go")
	fn := between(t, src, "func (h *Handler) shoppingList", "\n}\n")

	// ⚠️ **The minimum rule is still opt-in**, and the forecast is not allowed
	// to smuggle every ingredient onto the list past it: it speaks only for a
	// row that has sold on enough separate days *and* has a delivery history to
	// measure a horizon from. Without the second guard a restaurant that never
	// enters purchases would get an invented week of cover for its whole
	// catalogue, every morning.
	if !strings.Contains(fn, "if in.MinQty > 0 && have < in.MinQty {") {
		t.Fatal("the minimum rule stopped being opt-in per ingredient")
	}
	if !strings.Contains(fn, "in.DerivedOnly()") {
		t.Fatal("a prep item can reach the shopping list — it is cooked, not bought")
	}
	if !strings.Contains(fn,
		"want.days >= forecastMinDays && fill.deliveries >= 2") {
		t.Fatal("the forecast speaks without a selling history or a delivery rhythm")
	}
}

// ⚠️ **Most recent supplier, not most frequent.** A restaurant that changed
// butcher last month should be told to ring the new one; a count over a year
// keeps naming the old one for exactly as long as the answer is most wrong.
func TestTheSupplierSuggestedIsTheLastOneWhoDelivered(t *testing.T) {
	src := readSource(t, "shoppinglist.go")
	fn := between(t, src, "func (h *Handler) lastSupplierOf", "\n}\n")

	if !strings.Contains(fn, `{Key: "at", Value: 1}`) {
		t.Fatal("deliveries are no longer read oldest-first, so the last write is not the latest")
	}
}

// ⚠️ **The buyer and the owner must be looking at the same list.** One is
// standing at a market with cash and the other is reading a panel; a second
// implementation would eventually have them disagree about whether the kitchen
// is out of beef, and that argument happens after the money is spent. Same
// reasoning as `soldOutHeldBy` and `saveStocktake`, arriving again.
func TestOneListForThePanelAndThePhone(t *testing.T) {
	src := readSource(t, "shoppinglist.go")
	admin := between(t, src, "func (h *Handler) AdminShoppingList", "\n}\n")
	if !strings.Contains(admin, "h.shoppingList(") {
		t.Fatal("the panel computes its own shopping list again")
	}
	phone := readSource(t, "staffbuy.go")
	fn := between(t, phone, "func (h *Handler) StaffBuyList", "\n}\n")
	if !strings.Contains(fn, "h.shoppingList(") {
		t.Fatal("the buyer's phone computes its own shopping list")
	}
	// ⚠️ And the branch comes off the employee, never the request — the rule
	// every staff endpoint follows, or a buyer shops for a kitchen they have
	// never stood in.
	if !strings.Contains(fn, "s.BranchID") {
		t.Fatal("the buyer's branch is not taken from the employee")
	}
}
