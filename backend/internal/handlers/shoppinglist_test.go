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
	fn := between(t, src, "func (h *Handler) AdminShoppingList", "\n}\n")

	if !strings.Contains(fn, "in.MinQty - math.Max(onHand, 0)") {
		t.Fatal("a shelf below zero is being ordered against its negative figure")
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
	fn := between(t, src, "func (h *Handler) AdminShoppingList", "\n}\n")

	if !strings.Contains(fn, "in.MinQty <= 0 || in.DerivedOnly()") {
		t.Fatal("the shopping list stopped being opt-in per ingredient")
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
