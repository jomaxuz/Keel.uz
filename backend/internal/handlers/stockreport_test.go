package handlers

import (
	"strings"
	"testing"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ⚠️ **A dish using a sauce consumes tomatoes, not "sauce".** Leaving the prep
// item unexpanded would show tomatoes arriving every week and never being
// used — the shape of a theft report and the substance of an accounting bug.
func TestPrepItemsResolveToWhatTheyAreMadeOf(t *testing.T) {
	tomato := primitive.NewObjectID()
	oil := primitive.NewObjectID()
	sauce := primitive.NewObjectID()

	raw := rawInputs([]models.Ingredient{
		{ID: tomato, Unit: models.UnitKg, Price: 12000},
		{ID: oil, Unit: models.UnitL, Price: 18000},
		{ID: sauce, Unit: models.UnitKg, Output: 2000, Recipe: []models.RecipeLine{
			{IngredientID: tomato, Qty: 3000},
			{IngredientID: oil, Qty: 200},
		}},
	})

	// A bought thing is one unit of itself.
	if raw[tomato][tomato] != 1 {
		t.Fatalf("bought ingredient: %+v", raw[tomato])
	}
	// One gram of sauce is 1.5 g of tomato and 0.1 ml of oil — 3000 g and
	// 200 ml spread over a 2000 g batch.
	if got := raw[sauce][tomato]; got != 1.5 {
		t.Fatalf("tomato per gram of sauce=%v, want 1.5", got)
	}
	if got := raw[sauce][oil]; got != 0.1 {
		t.Fatalf("oil per gram of sauce=%v, want 0.1", got)
	}
}

// ⚠️ Bounded like the rate resolver, and for the same reason: two cards can
// name each other. What cannot be resolved is absent rather than invented —
// it shows up as an ingredient with deliveries and no usage, which is visible.
func TestACycleLeavesAnIngredientUnresolvedRatherThanLooping(t *testing.T) {
	a := primitive.NewObjectID()
	b := primitive.NewObjectID()
	raw := rawInputs([]models.Ingredient{
		{ID: a, Unit: models.UnitKg, Output: 100, Recipe: []models.RecipeLine{{IngredientID: b, Qty: 10}}},
		{ID: b, Unit: models.UnitKg, Output: 100, Recipe: []models.RecipeLine{{IngredientID: a, Qty: 10}}},
	})
	if len(raw) != 0 {
		t.Fatalf("a cycle resolved to something: %+v", raw)
	}
}

// ⚠️ The report must never be readable as a stock balance: there is no opening
// count, no write-offs and no stocktake, so a "remaining" figure would be a
// number nobody can check and everybody believes.
func TestTheFlowReportRefusesToBeAStockBalance(t *testing.T) {
	note := stockNote("uz")
	if !strings.Contains(note, "ombor qoldig'i EMAS") {
		t.Fatalf("the note stopped saying what this is not: %q", note)
	}
	if !strings.Contains(note, "texkarta") {
		t.Fatal("the note no longer says the usage figure comes from the cards")
	}

	src := readSource(t, "stockreport.go")
	fn := between(t, src, "func (h *Handler) AdminStockReport", "\n}\n")
	// A prep item is bought by nobody and used as itself by nothing: listing
	// it would be two empty columns with a name on them.
	if !strings.Contains(fn, "ing.MadeInHouse()") {
		t.Fatal("prep items are being listed as if they were delivered")
	}
	// Cancelled orders were not cooked — the same basis ABC counts on, so the
	// two reports cannot disagree about what sold.
	used := between(t, src, "func (h *Handler) consumedInPeriod", "\n}\n")
	if !strings.Contains(used, "models.StatusCancelled") {
		t.Fatal("cancelled orders are being counted as food that left the store")
	}
}
