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
	if !strings.Contains(fn, "h.expectedStockByWarehouse(r, scope, sc.BrandID, in.At)") {
		t.Fatal("the expected figure is being taken from the request")
	}
	// ⚠️ And it is **that store's** baseline. A count of the bar checked
	// against the branch's combined figures would reset the kitchen's starting
	// point to a number nobody walked in and looked at.
	if !strings.Contains(fn, "byWarehouse[in.WarehouseID]") {
		t.Fatal("a count is no longer checked against its own store")
	}
	if !strings.Contains(fn, "ing.WarehouseID != in.WarehouseID") {
		t.Fatal("a count of one store accepts lines belonging to another")
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
	fn := between(t, src, "func (h *Handler) expectedStockByWarehouse", "\n}\n")

	for _, part := range []string{
		"h.Store.Stocktakes.FindOne", "h.deliveredInPeriod",
		"h.consumedInPeriod", "h.writtenOffInPeriod",
	} {
		if !strings.Contains(fn, part) {
			t.Fatalf("expected stock no longer accounts for %s", part)
		}
	}
	// The window starts at the previous count, not at the beginning of time.
	if !strings.Contains(fn, "from, &at") {
		t.Fatal("the movements are no longer counted from the last stocktake")
	}
	sheet := between(t, src, "func (h *Handler) AdminStocktakeSheet", "\n}\n")
	if !strings.Contains(sheet, `"since"`) {
		t.Fatal("the sheet no longer says what the expected figure is measured from")
	}
}

// ⚠️ **Each store is measured from its own last count.** The bar is counted on
// a Sunday and the kitchen on a Wednesday; one "since" for the whole branch
// measures half the ingredients from a date nobody counted them on, and the
// whole of that error lands on whichever store was counted less recently —
// which is the store the owner is least sure about already.
func TestEachWarehouseIsMeasuredFromItsOwnCount(t *testing.T) {
	src := readSource(t, "stocktake.go")
	fn := between(t, src, "func (h *Handler) expectedStockByWarehouse", "\n}\n")

	// The last count is looked up per store, not once for the branch.
	if !strings.Contains(fn, `filter["warehouseId"] = wh`) {
		t.Fatal("the previous count is no longer looked up per warehouse")
	}
	// ⚠️ And the undivided store has to match documents that predate the field.
	// A plain equality filter on the zero id matches nothing in Mongo when the
	// key is absent, which would silently discard every count ever taken.
	if !strings.Contains(fn, `{"warehouseId": bson.M{"$exists": false}}`) {
		t.Fatal("counts taken before warehouses existed no longer start anything")
	}
	// A movement only belongs to the store its ingredient is kept in.
	if !strings.Contains(fn, "if home[id] != wh {") {
		t.Fatal("deliveries and write-offs are no longer filed by store")
	}
}

// The oldest count is what the screen's caveat has to describe.
//
// ⚠️ A store nobody has ever counted reaches all the way back, and saying so is
// the point: the sentence beside the figure must describe the weakest half of
// the answer, not the strongest.
func TestSinceReportsTheWeakestStore(t *testing.T) {
	src := readSource(t, "stocktake.go")
	fn := between(t, src, "func (h *Handler) expectedStock(", "\n}\n")
	if !strings.Contains(fn, "return out, nil, nil") {
		t.Fatal("an uncounted store no longer makes the measurement open-ended")
	}
	if !strings.Contains(fn, "t.Before(*oldest)") {
		t.Fatal("since is no longer the oldest of the stores' counts")
	}
}
