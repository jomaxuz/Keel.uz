package handlers

import (
	"strings"
	"testing"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ⚠️ **Putting up a price must not change what last month cost.** The first
// version of this applied today's prices across the whole report, so raising
// the price of beef this morning quietly moved March's margin — on a screen
// somebody had already read, quoted, and maybe priced a menu from.
func TestAPriceRiseDoesNotRewriteLastMonth(t *testing.T) {
	beef := primitive.NewObjectID()
	march := time.Date(2026, 3, 15, 12, 0, 0, 0, time.Local)
	august := time.Date(2026, 8, 15, 12, 0, 0, 0, time.Local)

	in := models.Ingredient{
		ID: beef, Unit: models.UnitKg, Price: 120000,
		History: []models.PriceEntry{
			{Price: 90000, At: time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)},
			{Price: 120000, At: time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)},
		},
	}
	if got := in.PriceAt(march); got != 90000 {
		t.Fatalf("march price=%d, want 90000", got)
	}
	if got := in.PriceAt(august); got != 120000 {
		t.Fatalf("august price=%d, want 120000", got)
	}

	rows := []models.Ingredient{in}
	if got := ratesAt(rows, march)[beef]; got != 90 {
		t.Fatalf("march rate=%v, want 90/g", got)
	}
	if got := ratesAt(rows, august)[beef]; got != 120 {
		t.Fatalf("august rate=%v, want 120/g", got)
	}
}

// ⚠️ A price we only learned later is the best answer for earlier periods: an
// ingredient added today has to cost something in last month's report, and the
// alternative — nothing — would make every dish containing it look free.
func TestAPriceReachesBackBeforeItWasKnown(t *testing.T) {
	in := models.Ingredient{
		Unit: models.UnitKg, Price: 90000,
		History: []models.PriceEntry{
			{Price: 90000, At: time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)},
		},
	}
	if got := in.PriceAt(time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)); got != 90000 {
		t.Fatalf("price before it was known=%d", got)
	}
}

// ⚠️ An edit is "from today", never a correction of the past: the form cannot
// tell "we typed it wrong" from "beef went up", and treating every edit as
// retroactive is what rewrites a month somebody has already read.
func TestAnEditIsRecordedFromTodayAndOnlyWhenTheMoneyChanged(t *testing.T) {
	src := readSource(t, "ingredients.go")
	fn := between(t, src, "func (h *Handler) priceHistoryFor", "\n}\n")

	if !strings.Contains(fn, "hist[len(hist)-1].Price == next.Price") {
		t.Fatal("renaming an ingredient leaves a price change nobody can explain")
	}
	if !strings.Contains(fn, "next.UpdatedAt") {
		t.Fatal("a price change is no longer stamped with when it happened")
	}
}

// A period spanning a price rise has portions at two costs; one figure
// multiplied out at the end would be wrong for both halves.
func TestEachPortionIsCostedAtItsOwnDay(t *testing.T) {
	src := readSource(t, "abcxyz.go")
	fn := between(t, src, "func classify", "\n}\n")

	if !strings.Contains(fn, "costs.Cost(it.MenuItemID, o.CreatedAt)") {
		t.Fatal("the dish is no longer costed at the price of the day it sold")
	}
	if strings.Contains(fn, "d.cost * d.qty") {
		t.Fatal("the cost is being multiplied out at the end again")
	}
}

// ⚠️ A dish's margin is computed against the revenue of the portions that could
// actually be costed. An order line old enough to have no dish id cannot be
// costed, and charging its revenue against the others' cost reports a margin
// nobody earned — the same rule the report-level coverage note follows, one
// level down where nothing else would say it.
func TestADishMarginOnlyCoversThePortionsItCosted(t *testing.T) {
	dish := primitive.NewObjectID()
	now := time.Now()
	orders := []models.Order{{
		CreatedAt: now,
		Items: []models.OrderItem{
			{MenuItemID: dish, Name: "Lag'mon", Qty: 1, Price: 40000},
			// The same dish, sold before order lines carried a dish id.
			{Name: "Lag'mon", Qty: 1, Price: 40000},
		},
	}}
	rows, _ := classify(orders, fixedCosts(map[primitive.ObjectID]int{dish: 15000}))

	if len(rows) != 1 {
		t.Fatalf("rows=%d", len(rows))
	}
	r := rows[0]
	if r.Revenue != 80000 || r.Qty != 2 {
		t.Fatalf("the dish lost a sale: %+v", r)
	}
	if r.Cost != 15000 {
		t.Fatalf("cost=%d — only one portion could be costed", r.Cost)
	}
	if r.Margin != 25000 {
		t.Fatalf("margin=%d, want 25000 (the costed portion only)", r.Margin)
	}
}
