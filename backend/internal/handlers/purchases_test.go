package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **A delivery is dated by the invoice**, and that is what makes it
// different from editing a price by hand. An edit cannot tell "we typed it
// wrong" from "beef went up" and counts from today; a delivery is a measurement
// carrying its own date, so saying meat cost this much last Tuesday is a fact
// about last Tuesday rather than a rewriting of it.
func TestADeliveryWritesItsPriceAtItsOwnDate(t *testing.T) {
	src := readSource(t, "purchases.go")
	fn := between(t, src, "func (h *Handler) applyDeliveryPrices", "\n}\n")

	if !strings.Contains(fn, "At: p.At") {
		t.Fatal("the delivery's price is no longer stamped with the invoice date")
	}
	// ⚠️ Sorted, not appended: an invoice entered three days late describes
	// those three days, and a history ordered by when somebody typed it would
	// cost them at the wrong price.
	if !strings.Contains(fn, "sort.SliceStable") {
		t.Fatal("a late invoice lands out of order in the price history")
	}
	// ⚠️ Only the latest entry moves the headline price: an invoice from last
	// month says nothing about today, and letting it overwrite today's price
	// is how a back-dated row quietly makes every dish cheaper.
	if !strings.Contains(fn, "hist[len(hist)-1].At.Equal(p.At)") {
		t.Fatal("a back-dated invoice overwrites today's price")
	}
	// Unchanged prices write nothing: an entry per delivery buries the two or
	// three that matter under a hundred saying the same number.
	if !strings.Contains(fn, "ing.PriceAt(p.At) == l.Price") {
		t.Fatal("every delivery writes a price entry, changed or not")
	}
	// A prep item is cooked, not delivered.
	if !strings.Contains(fn, "ing.MadeInHouse()") {
		t.Fatal("a delivery line can overwrite a prep item's computed rate")
	}
}

// ⚠️ The invoice's own total wins when it was given: a delivery charge or a
// discount at the door is money the restaurant paid that no line explains.
// Only an empty total is computed, so the field is never a silent second
// opinion about what the lines say.
func TestTheInvoiceTotalIsNotOverwrittenByTheLines(t *testing.T) {
	src := readSource(t, "purchases.go")
	fn := between(t, src, "func (h *Handler) AdminCreatePurchase", "\n}\n")

	if !strings.Contains(fn, "if in.Total <= 0 {") {
		t.Fatal("the lines are overwriting the invoice's own total")
	}
	// ⚠️ A future-dated delivery would apply to nothing today and everything
	// from then on — a mistake with a delayed effect nobody would connect back
	// to this form.
	if !strings.Contains(fn, "in.At.After(now)") {
		t.Fatal("a delivery can be dated into the future")
	}
}

// ⚠️ Deleting a delivery leaves the prices it wrote alone. Unwinding them is
// not "put the old number back": later deliveries, hand edits and every dish
// costed in between sit on top, and a delete that quietly re-costed a month
// would be far worse than a wrong invoice row.
func TestDeletingADeliveryDoesNotUnwindPrices(t *testing.T) {
	src := readSource(t, "purchases.go")
	fn := between(t, src, "func (h *Handler) AdminDeletePurchase", "\n}\n")

	if strings.Contains(fn, "history") || strings.Contains(fn, "Ingredients") {
		t.Fatal("deleting a delivery is rewriting price history")
	}
}
