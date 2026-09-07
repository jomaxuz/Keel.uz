package handlers

import (
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

// A count with two lines: one short, one over.
func takeFixture(at time.Time, beef, parsley models.StocktakeLine) models.Stocktake {
	return models.Stocktake{
		ID: primitive.NewObjectID(), At: at, By: "Nodira",
		Lines: []models.StocktakeLine{beef, parsley},
		Value: beef.Value + parsley.Value,
	}
}

func namesFor(ids ...primitive.ObjectID) shortageNames {
	ing := map[primitive.ObjectID]models.Ingredient{}
	for i, id := range ids {
		ing[id] = models.Ingredient{ID: id, Name: string(rune('A' + i)), Unit: "kg"}
	}
	return shortageNames{
		ingredients: ing,
		stores:      map[primitive.ObjectID]string{},
		branches:    map[primitive.ObjectID]string{},
	}
}

// ⚠️ **A surplus is not a small shortage.** It is usually a delivery booked
// twice, it is fixed on a different screen by a different person, and a queue
// sorted by "worst" cannot hold both — the worst of one is the best of the
// other. A row that arrived here would also be answered with one of five
// verdicts, none of which is true of it.
func TestOnlyShortfallsBecomeCases(t *testing.T) {
	beef, parsley := primitive.NewObjectID(), primitive.NewObjectID()
	take := takeFixture(time.Now(),
		models.StocktakeLine{IngredientID: beef, Diff: -12, Value: -1_800_000},
		models.StocktakeLine{IngredientID: parsley, Diff: 3, Value: 40_000},
	)
	got := shortageRows([]models.Stocktake{take}, namesFor(beef, parsley), nil, nil)
	if len(got.Rows) != 1 {
		t.Fatalf("%d rows, want 1 — the surplus line became a case", len(got.Rows))
	}
	if got.Rows[0].IngredientID != beef.Hex() {
		t.Fatal("the wrong line became the case")
	}
}

// ⚠️ **The queue is in som, positive, worst first.** The count stores a
// shortfall negative because there it is a signed part of a total; a queue
// sorted by "most negative" reads backwards to everybody who is not its author,
// and one sorted by quantity puts a kilo of parsley above a kilo of beef.
func TestTheQueueIsSortedByMoneyNotByQuantity(t *testing.T) {
	beef, parsley := primitive.NewObjectID(), primitive.NewObjectID()
	take := takeFixture(time.Now(),
		// Twelve kilos gone, and the money that matters.
		models.StocktakeLine{IngredientID: beef, Diff: -12, Value: -1_800_000},
		// Forty kilos gone, and nobody's morning.
		models.StocktakeLine{IngredientID: parsley, Diff: -40, Value: -60_000},
	)
	got := shortageRows([]models.Stocktake{take}, namesFor(beef, parsley), nil, nil)
	if len(got.Rows) != 2 {
		t.Fatalf("%d rows, want 2", len(got.Rows))
	}
	if got.Rows[0].IngredientID != beef.Hex() {
		t.Error("the parsley leads the queue — it is sorted by quantity, not money")
	}
	if got.Rows[0].Value != 1_800_000 {
		t.Errorf("value %d — the queue shows the count's negative sign",
			got.Rows[0].Value)
	}
	if got.OpenValue != 1_860_000 || got.Open != 2 {
		t.Errorf("open %d worth %d, want 2 worth 1860000", got.Open, got.OpenValue)
	}
}

// ⚠️ **An answered case never sits above an unanswered one.** A month of
// verdicts is the sentence about the process and stays on the list; the queue
// exists to say what is still outstanding, and a two-million-som shortfall
// somebody already explained would otherwise hold the top of it forever.
func TestAnsweredCasesFallBelowOpenOnes(t *testing.T) {
	big, small := primitive.NewObjectID(), primitive.NewObjectID()
	take := takeFixture(time.Now(),
		models.StocktakeLine{IngredientID: big, Diff: -12, Value: -1_800_000},
		models.StocktakeLine{IngredientID: small, Diff: -1, Value: -50_000},
	)
	answered := map[caseKey]models.ShortageCase{
		{take.ID, big}: {Verdict: models.VerdictWaste, Note: "muzlatgich buzildi",
			By: "Aziz", At: time.Now()},
	}
	got := shortageRows([]models.Stocktake{take}, namesFor(big, small), nil, answered)
	if got.Rows[0].IngredientID != small.Hex() {
		t.Error("an explained shortfall still leads the queue")
	}
	if got.Open != 1 || got.OpenValue != 50_000 {
		t.Errorf("open %d worth %d — an answered case is still counted as work",
			got.Open, got.OpenValue)
	}
	if got.Total != 1_850_000 {
		t.Errorf("total %d — the answered one left the total too", got.Total)
	}
	if got.Rows[1].VerdictNote != "muzlatgich buzildi" {
		t.Error("the verdict did not reach the row it answers")
	}
}

// ⚠️ **A shortfall belongs to the stretch between two counts of *that store*,
// not to the day it was found.** A store counted three times in the window has
// three period starts; giving every row the oldest of them describes two of the
// three as a quarter's loss, which is the difference between "the freezer broke
// last week" and "somebody has been at this since March".
func TestEachCountIsMeasuredFromThePreviousCountOfItsOwnStore(t *testing.T) {
	src := readSource(t, "shortagecases.go")
	fn := between(t, src, "func (h *Handler) previousCounts", "\n}\n")
	if !strings.Contains(fn, "last[key]") || !strings.Contains(fn, "out[t.ID]") {
		t.Fatal("the period start is not kept per count")
	}
	// ⚠️ The database is asked only for the oldest count of each store; every
	// later one is preceded by a row already in the list. One read per count
	// would be a round trip per row on a screen an owner leaves open.
	if strings.Count(fn, "h.countBefore(") != 1 {
		t.Error("countBefore is called more than once per store")
	}
}

// ⚠️ **The undivided store matches counts with no warehouse field at all.** A
// Mongo filter on the zero id does not match a document that lacks the key, so
// every count taken before warehouses existed would stop being anybody's period
// start — and every one of those restaurants would read "never counted before".
// `expectedStockByWarehouse` learned this first; this is the same query.
func TestTheUndividedStoreMatchesCountsSavedBeforeWarehousesExisted(t *testing.T) {
	src := readSource(t, "shortagecases.go")
	fn := between(t, src, "func (h *Handler) countBefore", "\n}\n")
	if !strings.Contains(fn, `"warehouseId": bson.M{"$exists": false}`) {
		t.Fatal("a count saved before the warehouse field would not be found")
	}
}

// ⚠️ **The verdict is written with an insert, and the unique index refuses the
// second one.** Two managers read the same queue, both see an open case and
// both answer it; an upsert keeps whoever clicked second, silently, and the
// explanation on the screen then changes between refreshes with nobody having
// edited it. The count's own explanation makes Mongo decide for the same
// reason.
func TestAVerdictIsWrittenOnceAndTheDatabaseDecides(t *testing.T) {
	src := readSource(t, "shortagecases.go")
	fn := between(t, src, "func (h *Handler) AdminCloseShortageCase", "\n}\n")
	if !strings.Contains(fn, "ShortageCases.InsertOne") {
		t.Fatal("the verdict is not an insert — a second answer would overwrite the first")
	}
	for _, bad := range []string{"UpdateOne", "ReplaceOne", "SetUpsert"} {
		if strings.Contains(fn, bad) {
			t.Errorf("the verdict path uses %s — the later answer wins silently", bad)
		}
	}
	// ⚠️ And the count it answers is looked up **inside the scope filter**: an
	// id alone must never select a document, or a manager pinned to one branch
	// closes another branch's findings by pasting an id.
	if !strings.Contains(fn, "for k, v := range scope") {
		t.Error("the count is fetched by id alone — another branch's findings are reachable")
	}
	// A verdict needs a sentence beside its kind: the five kinds exist so a
	// month of these can be counted, not so a shortfall can be dismissed in
	// one click.
	if !strings.Contains(fn, `httpx.Error(w, http.StatusBadRequest, "sababini yozing")`) {
		t.Error("a shortfall can be closed with no explanation")
	}
}

// ⚠️ **The person who counted the shelf may not write the verdict on their own
// shortfall.** That is the failure the blind count sheet exists to prevent, one
// screen along. The storekeeper's allow-list is prefixes with no notion of
// methods (stocklogin.go), so the address itself is the gate: served from
// `/admin/stock/…` the queue would have been theirs to close.
func TestTheStorekeeperCannotReachTheShortfallQueue(t *testing.T) {
	for _, path := range []string{"/admin/shortages", "/admin/shortages/close"} {
		if stockAllowed(path) {
			t.Errorf("%s is on the storekeeper's list — they can answer for "+
				"a shelf they counted themselves", path)
		}
	}
}

// ⚠️ **A zero branch must not reach the browser as an id.** `omitempty` does
// not apply to an ObjectID — it is an array — so an unset branch arrives as
// "000000000000000000000000", which JavaScript reads as truthy. This codebase
// has shipped that bug twice (CLAUDE.md), once making every admin look pinned
// to a branch and once collapsing a menu to one dish per category.
func TestAnUnsetBranchIsSentAsNothingNotAsZeros(t *testing.T) {
	if got := hexOrEmpty(primitive.NilObjectID); got != "" {
		t.Fatalf("a zero id was sent as %q", got)
	}
	id := primitive.NewObjectID()
	if got := hexOrEmpty(id); got != id.Hex() {
		t.Fatalf("a real id came back as %q", got)
	}
}
