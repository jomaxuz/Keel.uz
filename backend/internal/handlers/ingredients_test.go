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
	src := readSource(t, "costledger.go")
	fn := between(t, src, "func (l *costLedger) Cost", "\n}\n")

	if !strings.Contains(fn, "recipeComplete(d.recipe, rates)") {
		t.Fatal("an incomplete card is being used to cost a dish")
	}
	// ⚠️ Resolved for the **day of the sale**, not once for the report: a
	// price rise must not rewrite a month somebody has already read.
	if !strings.Contains(fn, "l.rates(at)") {
		t.Fatal("the rates are no longer resolved per day")
	}
}

// ⚠️ **A sauce is cooked, not bought**, and without prep items every dish
// containing it has to list its tomatoes again — which is how a kitchen ends up
// maintaining one recipe in seven places that stop agreeing within a month.
func TestAPrepItemCostsWhatItsBatchCosts(t *testing.T) {
	tomato := primitive.NewObjectID()
	sauce := primitive.NewObjectID()
	rates := map[primitive.ObjectID]float64{
		tomato: models.Ingredient{Unit: models.UnitKg, Price: 12000}.CostPerRecipeUnit(),
	}
	// 3 kg of tomatoes boil down to 2 kg of sauce: 36 000 for 2000 g.
	batch := []models.RecipeLine{{IngredientID: tomato, Qty: 3000}}
	cost := recipeCost(batch, rates)
	if cost != 36000 {
		t.Fatalf("batch=%d", cost)
	}
	rates[sauce] = float64(cost) / 2000

	// ⚠️ The yield is where a prep card is honest about evaporation: costing
	// the sauce at 3000 g would underprice every dish it appears in.
	dish := []models.RecipeLine{{IngredientID: sauce, Qty: 150}}
	if got := recipeCost(dish, rates); got != 2700 {
		t.Fatalf("dish=%d, want 2700", got)
	}
}

// ⚠️ Two cards can name each other — nothing stops somebody putting the sauce
// in the dough and the dough in the sauce. A resolver that recursed would hang
// the panel; one that "handled" it by costing the missing side at zero would
// quietly underprice both. The pass loop stops, and what is left is simply
// absent — which every screen downstream already says something honest about.
func TestTheResolverStopsOnACycle(t *testing.T) {
	src := readSource(t, "ingredients.go")
	fn := between(t, src, "func ratesAt", "\n}\n")

	if !strings.Contains(fn, "for range made") {
		t.Fatal("the resolver is no longer bounded by the number of prep items")
	}
	if !strings.Contains(fn, "if !progress") {
		t.Fatal("the resolver keeps going after a pass that resolved nothing")
	}
	if !strings.Contains(fn, "recipeComplete(in.Recipe, out)") {
		t.Fatal("a prep item is being costed before its own inputs are known")
	}
}

// ⚠️ A prep item's price is not typed — it is what its batch costs. Keeping an
// old typed figure beside a card leaves two answers on one document, and the
// stale one looks the more authoritative.
func TestSavingAPrepItemDropsItsTypedPrice(t *testing.T) {
	src := readSource(t, "ingredients.go")
	fn := between(t, src, "func (h *Handler) AdminSaveIngredient", "\n}\n")

	if !strings.Contains(fn, "in.Price = 0") {
		t.Fatal("a prep item can keep a typed price beside its card")
	}
}

// ⚠️ **Zero means "do not warn me", not "warn me at zero".** Most ingredients
// never need a minimum — nobody tracks one for cinnamon — and a list where
// every line eventually turns red is a list nobody reads.
func TestOnlyIngredientsWithAMinimumCanRunLow(t *testing.T) {
	src := readSource(t, "ingredients.go")
	fn := between(t, src, "func (h *Handler) AdminListIngredients", "\n}\n")

	if !strings.Contains(fn, "in.MinQty > 0 && v.Expected < in.MinQty") {
		t.Fatal("an ingredient with no minimum can be reported as running low")
	}
	// ⚠️ And the screen is told when the estimate was last anchored to a
	// count. Without it "should be there" reads as a balance the system has
	// been keeping, and somebody orders against it.
	if !strings.Contains(fn, `"countedAt": countedAt`) {
		t.Fatal("the list no longer says how much of an estimate it is showing")
	}
	// One walk of the movements for the whole list, not one per ingredient.
	if strings.Count(fn, "h.expectedStock(") != 1 {
		t.Fatal("expected stock is being computed per row")
	}
}
