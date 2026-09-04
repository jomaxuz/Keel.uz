package handlers

// ---- The written rows, against a real database ----
//
// ⚠️ **The claim these exist to check is the one nobody can eyeball: that
// changing where consumption comes from changed no number.** The balance used
// to expand the orders on every read and now reads rows written when the food
// was rung up. If those two ever disagree, the disagreement is silent — a shelf
// figure that is wrong by a plausible amount, found at a stocktake, blamed on
// whoever counted. A source-inspecting test cannot see it; only running both
// and comparing can.
//
// Skipped where there is no Mongo, for the reason alertlive_test.go gives: a
// test that cannot run must not be a test that fails.

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

// stockFixture is one ingredient, one dish that uses it, and the branch they
// belong to.
type stockFixture struct {
	h      *Handler
	branch primitive.ObjectID
	beef   primitive.ObjectID
	dish   primitive.ObjectID
}

func liveStock(t *testing.T) stockFixture {
	t.Helper()
	h, branch := liveHandler(t)
	ctx := context.Background()

	beef := primitive.NewObjectID()
	if _, err := h.Store.Ingredients.InsertOne(ctx, models.Ingredient{
		ID: beef, Name: "Go'sht", Unit: models.UnitKg,
		// 90 000 so'm a kilo, so 90 so'm a gram.
		Price: 90_000,
	}); err != nil {
		t.Fatal(err)
	}
	dish := primitive.NewObjectID()
	if _, err := h.Store.Menu.InsertOne(ctx, bson.M{
		"_id": dish, "name": "Lag'mon", "price": 40_000,
		"recipe": []bson.M{{"ingredientId": beef, "qty": 150}},
	}); err != nil {
		t.Fatal(err)
	}
	return stockFixture{h: h, branch: branch, beef: beef, dish: dish}
}

func (f stockFixture) check(t *testing.T, items ...models.OrderItem) *models.Order {
	t.Helper()
	o := &models.Order{
		ID: primitive.NewObjectID(), BranchID: f.branch,
		Number: "T-1", Status: models.StatusPending, Items: items,
		Check:     &models.OrderCheck{OpenedAt: time.Now()},
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if _, err := f.h.Store.Orders.InsertOne(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	return o
}

func (f stockFixture) beefUsed(t *testing.T) float64 {
	t.Helper()
	used, err := f.h.consumedFromMoves(context.Background(),
		bson.M{"branchId": f.branch}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return used[f.beef]
}

// near compares kilos the way every screen that shows them does.
//
// ⚠️ **Exact equality is the wrong test for a shelf.** A portion is 0.15 kg and
// three of them are 0.44999999999999996 in binary floating point — the figure
// the panel prints is 0.45 because `round3` runs at the edge, which is where it
// belongs. A test demanding the raw sum be exact would be testing IEEE 754, and
// the first person to hit it would "fix" the arithmetic.
func near(t *testing.T, got, want float64, what string) {
	t.Helper()
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("%s: %v kg, wanted %v", what, got, want)
	}
}

func (f stockFixture) sync(t *testing.T, o *models.Order) {
	t.Helper()
	if err := f.h.reconcileOrderStock(context.Background(), o, nil); err != nil {
		t.Fatal(err)
	}
}

// Two portions of a dish carrying 150 g of beef take 0.3 kg — in purchase
// units, because that is what the balance arithmetic speaks.
func TestRingingUpADishTakesItOffTheShelf(t *testing.T) {
	f := liveStock(t)
	o := f.check(t, models.OrderItem{
		MenuItemID: f.dish, Name: "Lag'mon", Qty: 2, LineID: "a",
	})
	f.sync(t, o)

	near(t, f.beefUsed(t), 0.3, "used")
}

// ⚠️ **The cashier's mis-tap, end to end.** The row is written the moment the
// tile is pressed; the line is then removed before the kitchen is told, and the
// shelf has to be exactly where it started. Reversed rather than deleted, so
// somebody repeatedly ringing up and removing an expensive dish is still
// visible — but reversed rows count for nothing.
func TestRemovingALineBeforeItIsFiredPutsTheShelfBack(t *testing.T) {
	f := liveStock(t)
	o := f.check(t, models.OrderItem{
		MenuItemID: f.dish, Name: "Lag'mon", Qty: 2, LineID: "a",
	})
	f.sync(t, o)
	near(t, f.beefUsed(t), 0.3, "used before the void")

	// What StaffVoidCheckLine does to an unfired line: it is gone.
	o.Items = nil
	f.sync(t, o)

	near(t, f.beefUsed(t), 0, "used after an unfired line was removed")
	// The row is still there to be read, and says why.
	var row models.StockMovement
	if err := f.h.Store.StockMoves.FindOne(context.Background(),
		bson.M{"orderId": o.ID}).Decode(&row); err != nil {
		t.Fatal(err)
	}
	if row.ReversedAt == nil || row.ReversedReason == "" {
		t.Fatalf("the row was erased rather than reversed: %+v", row)
	}
}

// The other half: cooked, then taken off the bill. The beef is gone and stays
// gone, marked as waste.
func TestVoidingAFiredLineLeavesTheShelfShort(t *testing.T) {
	f := liveStock(t)
	fired := time.Now()
	o := f.check(t, models.OrderItem{
		MenuItemID: f.dish, Name: "Lag'mon", Qty: 1, LineID: "a", FiredAt: &fired,
	})
	f.sync(t, o)

	o.Items[0].Void = &models.CheckLineVoid{
		At: time.Now(), Reason: "mehmon qaytardi", Wasted: true,
	}
	f.sync(t, o)

	near(t, f.beefUsed(t), 0.15, "cooked food came back to the shelf")
	var row models.StockMovement
	if err := f.h.Store.StockMoves.FindOne(context.Background(),
		bson.M{"orderId": o.ID}).Decode(&row); err != nil {
		t.Fatal(err)
	}
	if !row.Wasted {
		t.Fatal("the row is not marked as waste — the waste report cannot see it")
	}
}

// ⚠️ **Idempotence is what makes the reconciler safe to call from ten places
// and from a sweep.** If a second call doubled the figure, every hook would
// have to be exactly once — and "exactly once" across a till, a sweep and a
// backfill is not a property anybody can hold.
func TestReconcilingTwiceChangesNothing(t *testing.T) {
	f := liveStock(t)
	o := f.check(t, models.OrderItem{
		MenuItemID: f.dish, Name: "Lag'mon", Qty: 3, LineID: "a",
	})
	f.sync(t, o)
	f.sync(t, o)
	f.sync(t, o)

	near(t, f.beefUsed(t), 0.45, "used after three reconciles")
	n, err := f.h.Store.StockMoves.CountDocuments(context.Background(),
		bson.M{"orderId": o.ID})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("%d rows for one line — the reconciler is inserting, not reconciling", n)
	}
}

// A quantity corrected before firing scales the row. ⚠️ Scaled, not re-read:
// the card is frozen per portion so a correction cannot reprice a sale against
// a recipe edited since.
func TestCorrectingAQuantityScalesTheRow(t *testing.T) {
	f := liveStock(t)
	o := f.check(t, models.OrderItem{
		MenuItemID: f.dish, Name: "Lag'mon", Qty: 3, LineID: "a",
	})
	f.sync(t, o)
	o.Items[0].Qty = 2
	f.sync(t, o)

	near(t, f.beefUsed(t), 0.3, "used after 3→2")
}

// ⚠️ **The claim the whole change rests on: the figure did not move.** The old
// path expanded the orders on every read; the new one reads what was written.
// Run both over the same sale and they have to agree to the gram — otherwise
// every restaurant's shelf shifted on the day this shipped, by an amount that
// looks exactly like ordinary drift.
func TestTheWrittenRowsAgreeWithTheOldArithmetic(t *testing.T) {
	f := liveStock(t)
	ctx := context.Background()
	fired := time.Now()
	o := f.check(t,
		models.OrderItem{MenuItemID: f.dish, Name: "Lag'mon", Qty: 2, LineID: "a"},
		// A half portion and a fired line, so the fractions and the firing
		// state are both in the comparison rather than a single plain line.
		models.OrderItem{
			MenuItemID: f.dish, Name: "Lag'mon", Qty: 1, LineID: "b",
			Portion: 50, FiredAt: &fired,
		},
	)
	f.sync(t, o)

	ingredients := f.h.scopedIngredients(ctx, primitive.NilObjectID)
	derived := f.h.consumedBy(ctx, []models.Order{*o}, ingredients)

	written := f.beefUsed(t)
	if diff := written - derived[f.beef]; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("written %v kg, derived %v kg — the two sources disagree",
			written, derived[f.beef])
	}
	// And it is the number a person would work out by hand: 2 × 150 g plus
	// half of 150 g.
	near(t, written, 0.375, "used")
}

// The backfill is the same reconciler over history, and it must be safe to have
// run before: a second boot must not double a year of cooking.
func TestTheBackfillIsSafeToRunTwice(t *testing.T) {
	f := liveStock(t)
	ctx := context.Background()
	f.check(t, models.OrderItem{
		MenuItemID: f.dish, Name: "Lag'mon", Qty: 4, LineID: "a",
	})

	if err := f.h.BackfillStockMovements(ctx); err != nil {
		t.Fatal(err)
	}
	near(t, f.beefUsed(t), 0.6, "used after the backfill")
	// ⚠️ The marker means the second call returns immediately — which is the
	// point of it, and also why it must only be written once the pass finished.
	if err := f.h.BackfillStockMovements(ctx); err != nil {
		t.Fatal(err)
	}
	near(t, f.beefUsed(t), 0.6, "used after a second boot")
}
