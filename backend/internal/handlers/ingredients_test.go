package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ⚠️ **Rounded once, at the finished dish.** A gram of anything costs well
// under one som, so rounding each line would cost most dishes at zero — the
// arithmetic is carried as a rate and turned into money exactly once.
func TestADishCostsWhatItsCardCosts(t *testing.T) {
	beef := primitive.NewObjectID()
	oil := primitive.NewObjectID()
	rates := map[primitive.ObjectID]float64{
		// 90 000 so'm a kilo, and 18 000 a litre.
		beef: models.Ingredient{Unit: models.UnitKg, Price: 90000}.CostPerRecipeUnit(),
		oil:  models.Ingredient{Unit: models.UnitL, Price: 18000}.CostPerRecipeUnit(),
	}
	card := []models.RecipeLine{
		{IngredientID: beef, Qty: 180}, // 16 200
		{IngredientID: oil, Qty: 15},   // 270
	}
	if got := recipeCost(card, rates); got != 16470 {
		t.Fatalf("cost=%d, want 16470", got)
	}
	// Each line on its own rounds to zero at this scale — the whole reason the
	// rate is carried rather than the som.
	if got := recipeCost([]models.RecipeLine{{IngredientID: oil, Qty: 1}}, rates); got != 18 {
		t.Fatalf("one millilitre=%d", got)
	}
}

// ⚠️ A card missing an ingredient somebody deleted must not quietly make the
// dish cheaper — which reads as good news on every screen that shows a margin.
func TestAnIncompleteCardDoesNotCostTheDish(t *testing.T) {
	present := primitive.NewObjectID()
	gone := primitive.NewObjectID()
	rates := map[primitive.ObjectID]float64{present: 12}
	card := []models.RecipeLine{
		{IngredientID: present, Qty: 100},
		{IngredientID: gone, Qty: 50},
	}
	if recipeComplete(card, rates) {
		t.Fatal("a card pointing at a deleted ingredient reads as complete")
	}
	// An empty card is not a complete one either: a dish with no lines has no
	// tech card, and treating it as "complete, costs nothing" would price every
	// unfinished dish at zero.
	if recipeComplete(nil, rates) {
		t.Fatal("an empty card reads as complete")
	}
}

// ⚠️ Half-finished rows are dropped rather than stored as zero: a zero-gram
// ingredient costs nothing and looks like a decision, and it makes "complete"
// impossible to see. The same ingredient twice becomes one line — a card
// listing beef twice is a card nobody can check against a plate.
func TestTheCardIsCleanedBeforeItIsStored(t *testing.T) {
	beef := primitive.NewObjectID()
	got := normalizeRecipe([]models.RecipeLine{
		{IngredientID: beef, Qty: 100},
		{IngredientID: beef, Qty: 50},
		{IngredientID: primitive.NewObjectID(), Qty: 0},
		{Qty: 30},
	})
	if len(got) != 1 || got[0].Qty != 150 {
		t.Fatalf("%+v", got)
	}
	if normalizeRecipe(nil) != nil {
		t.Fatal("an empty card is being stored as an empty array")
	}
}

// ⚠️ A recipe is a competitor's shopping list with the quantities filled in,
// and the dish document goes to every visitor — the same rule as the cost.
func TestTheCardNeverLeavesThroughADish(t *testing.T) {
	raw, err := json.Marshal(models.MenuItem{
		Name:   "Lag'mon",
		Recipe: []models.RecipeLine{{IngredientID: primitive.NewObjectID(), Qty: 180}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "recipe") || strings.Contains(string(raw), "180") {
		t.Fatalf("a dish carries its tech card to whoever asks: %s", raw)
	}
}

// ⚠️ The card wins over the typed cost, and is priced fresh on every read —
// that is the whole point: a price corrected this morning has to correct this
// afternoon's report and last month's.
func TestTheCardWinsOverTheTypedCost(t *testing.T) {
	src := readSource(t, "menucost.go")
	fn := between(t, src, "func (h *Handler) dishCosts", "\n}\n")

	if !strings.Contains(fn, "recipeComplete(r.Recipe, rates)") {
		t.Fatal("an incomplete card is being used to cost a dish")
	}
	if !strings.Contains(fn, "h.ingredientRates(ctx)") {
		t.Fatal("the rates are no longer read at report time — a stored copy drifts")
	}
}
