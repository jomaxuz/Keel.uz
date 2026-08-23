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

// ⚠️ **The bar sells one bottle in three measures.** Until the choices were
// counted, a hundred 100 ml pours took exactly as much off the shelf as a
// hundred 40 ml ones: the price already varied, only the stock did not, and the
// difference surfaced once a month as an unexplained shortfall nobody could
// attribute.
func TestPoursAreCountedByTheMeasureThatWasChosen(t *testing.T) {
	src := readSource(t, "stockreport.go")
	fn := between(t, src, "func (h *Handler) consumedInPeriod", "\n}\n")

	if !strings.Contains(fn, "chosenRecipes(d.Options, key.choices)") {
		t.Fatal("what was poured is no longer read off the choice that was ticked")
	}
	// ⚠️ The dish's own card is still taken as well: a gin and tonic is the
	// tonic and the lemon whichever measure of gin goes in.
	if !strings.Contains(fn, "take(d.Recipe, float64(sold[d.ID]))") {
		t.Fatal("the dish's own recipe stopped being consumed")
	}
}

// Two lines of the same pour have to add together, and the same two choices
// ticked in a different order are one pour.
//
// ⚠️ The group is part of the key, not only the choice: "50" under "Hajm" and
// "50" under "Muzli" are different answers that happen to share a word.
func TestOnePourIsOneKeyWhateverOrderItWasTickedIn(t *testing.T) {
	dish := primitive.NewObjectID()
	a := models.OrderItem{MenuItemID: dish, Options: []models.OrderItemOption{
		{Name: "Hajm", Choice: "50 ml"}, {Name: "Muzli", Choice: "Ha"},
	}}
	b := models.OrderItem{MenuItemID: dish, Options: []models.OrderItemOption{
		{Name: "Muzli", Choice: "Ha"}, {Name: "Hajm", Choice: "50 ml"},
	}}
	if optionKeyOf(a) != optionKeyOf(b) {
		t.Fatal("the same pour ticked in a different order counted as two")
	}

	c := models.OrderItem{MenuItemID: dish, Options: []models.OrderItemOption{
		{Name: "Hajm", Choice: "100 ml"}, {Name: "Muzli", Choice: "Ha"},
	}}
	if optionKeyOf(a) == optionKeyOf(c) {
		t.Fatal("two different measures counted as one pour")
	}

	// And the group is not interchangeable with the choice.
	d := models.OrderItem{MenuItemID: dish, Options: []models.OrderItemOption{
		{Name: "Muzli", Choice: "50 ml"}, {Name: "Hajm", Choice: "Ha"},
	}}
	if optionKeyOf(a) == optionKeyOf(d) {
		t.Fatal("the group and the choice are being run together")
	}
}

// Only the ticked choices come out of the store.
func TestOnlyTheTickedChoicePours(t *testing.T) {
	vodka := primitive.NewObjectID()
	groups := []models.MenuOption{{
		Name: "Hajm", Required: true,
		Choices: []models.OptionChoice{
			{Name: "40 ml", Recipe: []models.RecipeLine{{IngredientID: vodka, Qty: 40}}},
			{Name: "100 ml", Recipe: []models.RecipeLine{{IngredientID: vodka, Qty: 100}}},
		},
	}}
	got := chosenRecipes(groups, optionKeyOf(models.OrderItem{
		Options: []models.OrderItemOption{{Name: "Hajm", Choice: "100 ml"}},
	}).choices)

	if len(got) != 1 || len(got[0]) != 1 || got[0][0].Qty != 100 {
		t.Fatalf("the 100 ml pour was not the one taken: %+v", got)
	}
	// Nothing ticked takes nothing: a dish sold with no options must not pour.
	if len(chosenRecipes(groups, "")) != 0 {
		t.Error("an order with no choices poured something anyway")
	}
}
