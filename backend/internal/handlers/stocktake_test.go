package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **The difference is the product**, exactly as it is for the cash drawer.
// A count that stores only what was found has recorded nothing: the shortfall
// it exists to surface has been overwritten by the person who might have
// caused it.
func TestACountKeepsWhatItWasOutBy(t *testing.T) {
	src := readSource(t, "stocktake.go")
	fn := between(t, src, "func (h *Handler) AdminSaveStocktake", "\n}\n")

	// ⚠️ Expected comes from the server, never from the browser: a count whose
	// own baseline came from the screen that recorded it can be made to agree
	// with anything.
	if !strings.Contains(fn, "h.expectedStock(r, scope, in.At)") {
		t.Fatal("the expected figure is being taken from the request")
	}
	if !strings.Contains(fn, "Expected: exp") || !strings.Contains(fn, "Diff: diff") {
		t.Fatal("the count no longer freezes what it was out by")
	}
	// A discrepancy cannot be saved silently — the cash drawer's rule, and for
	// the same reason: the explanation is only available on the day.
	if !strings.Contains(fn, `off && in.Note == ""`) {
		t.Fatal("a count can disagree with the books and say nothing about it")
	}
	// A prep item is not counted as itself: what it was made from is already
	// in the count of its ingredients, and counting both subtracts twice.
	if !strings.Contains(fn, "ing.MadeInHouse()") {
		t.Fatal("prep items are being counted alongside their own ingredients")
	}
}

// ⚠️ Every part of "expected" is a recorded fact — the last count, the
// deliveries after it, what the cards say was used, what was written off. It
// is not a running balance the system has been keeping, which is why the sheet
// is told when the measurement starts.
func TestExpectedStockIsMeasuredFromTheLastCount(t *testing.T) {
	src := readSource(t, "stocktake.go")
	fn := between(t, src, "func (h *Handler) expectedStock", "\n}\n")

	for _, part := range []string{
		"h.Store.Stocktakes.FindOne", "h.deliveredInPeriod",
		"h.consumedInPeriod", "h.writtenOffInPeriod",
	} {
		if !strings.Contains(fn, part) {
			t.Fatalf("expected stock no longer accounts for %s", part)
		}
	}
	// The window starts at the previous count, not at the beginning of time.
	if !strings.Contains(fn, "since, &at") {
		t.Fatal("the movements are no longer counted from the last stocktake")
	}
	sheet := between(t, src, "func (h *Handler) AdminStocktakeSheet", "\n}\n")
	if !strings.Contains(sheet, `"since"`) {
		t.Fatal("the sheet no longer says what the expected figure is measured from")
	}
}
