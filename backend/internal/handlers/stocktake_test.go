package handlers

import (
	"encoding/json"
	"strings"
	"testing"
)

// ⚠️ **The difference is the product**, exactly as it is for the cash drawer.
// A count that stores only what was found has recorded nothing: the shortfall
// it exists to surface has been overwritten by the person who might have
// caused it.
func TestACountKeepsWhatItWasOutBy(t *testing.T) {
	src := readSource(t, "stocktake.go")
	fn := between(t, src, "func (h *Handler) saveStocktake", "\n}\n")

	// ⚠️ Expected comes from the server, never from the browser: a count whose
	// own baseline came from the screen that recorded it can be made to agree
	// with anything.
	if !strings.Contains(fn, "h.expectedStockByWarehouse(r, scope, brand, branch, in.At)") {
		t.Fatal("the expected figure is being taken from the request")
	}
	// ⚠️ And it is **that store's** baseline. A count of the bar checked
	// against the branch's combined figures would reset the kitchen's starting
	// point to a number nobody walked in and looked at.
	if !strings.Contains(fn, "byWarehouse[in.WarehouseID]") {
		t.Fatal("a count is no longer checked against its own store")
	}
	// ⚠️ And against **this branch's** placement, not a field on the
	// ingredient: the catalogue is the brand's and the rooms are the branch's.
	if !strings.Contains(fn, "placed[l.IngredientID] != in.WarehouseID") {
		t.Fatal("a count of one store accepts lines belonging to another")
	}
	if !strings.Contains(fn, "Expected: exp") || !strings.Contains(fn, "Diff: diff") {
		t.Fatal("the count no longer freezes what it was out by")
	}
	// ⚠️ **A count is no longer refused for disagreeing, and the reversal is
	// deliberate.**
	//
	// It used to be: a variance without an explanation could not be saved — the
	// cash drawer's rule, and right while the counter could see the expected
	// figures, because then "explain the variance" named something in front of
	// them. The sheet is blind now, and against a blind sheet that same refusal
	// is an oracle: type numbers, be refused, adjust, be accepted — and the
	// acceptance has just told you that you match the books. Forty lines of
	// that is tedious and entirely possible, and the person who would bother is
	// exactly who the blind sheet is for.
	//
	// So the count is taken as given and explained afterwards, against numbers
	// that can no longer be moved. What this asserts is that nothing refuses on
	// the way in.
	if strings.Contains(fn, `in.Note == ""`) &&
		strings.Contains(fn, "http.StatusBadRequest") &&
		strings.Contains(fn, "farq bor") {
		t.Fatal("a blind count can still be brute-forced against the refusal")
	}
	if !strings.Contains(fn, "off && in.Note != \"\"") {
		t.Fatal("an explanation given with the count is no longer recorded as one")
	}
	// A derived prep item is not counted as itself: what it was made from is
	// already in the count of its ingredients, and counting both subtracts
	// twice. ⚠️ A batched one **is** counted — it is a tub in a fridge, and its
	// inputs were taken by the production document in another building.
	if !strings.Contains(fn, "ing.DerivedOnly()") {
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
		// ⚠️ Moved stock is the fifth fact: without it a transfer reads as a
		// theft from one store and a miscount in the other.
		"h.transferredInPeriod",
	} {
		if !strings.Contains(fn, part) {
			t.Fatalf("expected stock no longer accounts for %s", part)
		}
	}
	// The window starts at the previous count, not at the beginning of time.
	if !strings.Contains(fn, "from, &at") {
		t.Fatal("the movements are no longer counted from the last stocktake")
	}
	sheet := between(t, src, "func (h *Handler) stocktakeSheet", "\n}\n")
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

// ⚠️ **An empty phantom store made every caveat permanently pessimistic.** The
// undivided store used to be seeded whether or not anything lived in it, and
// nobody ever counts a store with nothing on its shelves — so `since`, which is
// the oldest count across the stores, was nil forever. Every screen reading it
// then told a restaurant that counts every Sunday that nothing had ever been
// counted, which is the fastest way to teach somebody a caveat is noise.
func TestAnEmptyStoreIsNotOneOfTheStores(t *testing.T) {
	src := readSource(t, "stocktake.go")
	fn := between(t, src, "func (h *Handler) expectedStockByWarehouse", "\n}\n")

	if strings.Contains(fn, "stores := map[primitive.ObjectID]bool{primitive.NilObjectID: true}") {
		t.Fatal("the undivided store is seeded again whether or not anything is in it")
	}
	// ⚠️ And it still appears the moment an ingredient lives there, which is
	// every install that has not split its stores.
	if !strings.Contains(fn, "stores[placed[in.ID]] = true") {
		t.Fatal("a store is no longer taken from the ingredients that live in it")
	}
}

// ⚠️ **The blind count, enforced on the side of the wire that can enforce it.**
//
// Both screens already refused to draw the expected figure until a number had
// been typed, and the rule was right. It lived in the browser, where it bought
// less than it looked like: the figure was in the page either way, and the
// reveal-after-typing was defeated by typing anything, reading the number, and
// correcting the entry — nothing stopped the second edit.
//
// This asserts the sheet carries no baseline at all, so the next person to add
// a helpful column has to delete a test to do it.
func TestTheCountSheetCarriesNoTarget(t *testing.T) {
	// The row type the handler marshals, by its JSON shape rather than by name:
	// what matters is what reaches a browser.
	type row struct {
		IngredientID string   `json:"ingredientId"`
		Name         string   `json:"name"`
		Unit         string   `json:"unit"`
		Expected     *float64 `json:"expected"`
		Diff         *float64 `json:"diff"`
		Balance      *float64 `json:"balance"`
	}
	raw := []byte(`{"ingredientId":"a","name":"Kartoshka","unit":"kg"}`)
	var r row
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	if r.Expected != nil || r.Diff != nil || r.Balance != nil {
		t.Fatal("the sheet handed the counter something to match")
	}
	if r.Name == "" || r.Unit == "" {
		t.Fatal("the sheet stopped saying what to count")
	}
}
