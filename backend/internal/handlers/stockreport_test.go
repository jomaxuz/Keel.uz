package handlers

import (
	"strings"
	"testing"
	"time"

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
	// A derived prep item is bought by nobody and used as itself by nothing:
	// listing it would be two empty columns with a name on them.
	//
	// ⚠️ A **batched** one is neither — it is delivered by the central kitchen
	// and used as itself — so the guard asks `DerivedOnly`, and a chain's
	// central kitchen output stays visible to the branch that received it.
	if !strings.Contains(fn, "ing.DerivedOnly()") {
		t.Fatal("prep items are being listed as if they were delivered")
	}
}

// ⚠️ **A cancelled order was not cooked.** The same basis ABC counts on, so the
// two reports cannot disagree about what sold. Run rather than grepped for: the
// check moved once already, and the source test that guarded it passed right up
// until the line was somewhere else.
func TestACancelledOrderTakesNothingOffTheShelf(t *testing.T) {
	dish := primitive.NewObjectID()
	sold, _ := soldDishes([]models.Order{
		{Status: models.StatusCancelled, Items: []models.OrderItem{
			{MenuItemID: dish, Qty: 3},
		}},
		{Status: models.StatusDelivered, Items: []models.OrderItem{
			{MenuItemID: dish, Qty: 2},
		}},
	}, nil)

	if sold[dish] != 2 {
		t.Fatalf("sold %d portions, the cancelled order was counted", sold[dish])
	}
}

// ⚠️ **A void line is not food that left the kitchen either** — it is a line
// struck off an open check before anybody cooked it.
func TestAVoidedLineTakesNothingOffTheShelf(t *testing.T) {
	dish := primitive.NewObjectID()
	void := &models.CheckLineVoid{At: time.Now(), Reason: "adashib bosildi"}
	sold, _ := soldDishes([]models.Order{{
		Status: models.StatusDelivered,
		Items: []models.OrderItem{
			{MenuItemID: dish, Qty: 5, Void: void},
			{MenuItemID: dish, Qty: 1},
		},
	}}, nil)

	if sold[dish] != 1 {
		t.Fatalf("sold %d portions, the voided line was counted", sold[dish])
	}
}

// ⚠️ **A set took nothing off the shelf, and that was the quietest bug here.**
// A combo arrives as one order line whose own card is empty — the tomatoes are
// in its members. So every family set sold was, to the store, a sale of
// nothing, and the missing food surfaced weeks later at a count as a shortfall
// the person holding the clipboard was asked to explain.
func TestASetIsConsumedAsTheDishesItContains(t *testing.T) {
	combo, lagmon, salad := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()

	sold, _ := soldDishes([]models.Order{{
		Status: models.StatusDelivered,
		Items: []models.OrderItem{{
			MenuItemID: combo, Qty: 2,
			ComboItems: []models.OrderComboLine{
				{MenuItemID: lagmon, Qty: 1, Name: "Lag'mon"},
				{MenuItemID: salad, Qty: 2, Name: "Achichuk"},
			},
		}},
	}}, nil)

	// Two sets, each holding one lagmon and two salads.
	if sold[lagmon] != 2 || sold[salad] != 4 {
		t.Fatalf("the set opened into %d lagmon and %d salad, wanted 2 and 4",
			sold[lagmon], sold[salad])
	}
	// ⚠️ And the set itself is not also counted: its card is empty today, so
	// counting it changes nothing now and doubles everything the day somebody
	// gives a combo a card of its own.
	if sold[combo] != 0 {
		t.Fatalf("the set was counted as a dish as well (%d)", sold[combo])
	}
}

// ⚠️ **What the set contained the night it sold**, not what it contains now.
// Re-reading the live definition would restate last month's consumption from
// this month's menu — the same rule the frozen price follows.
func TestASetIsConsumedAsItWasSoldNotAsItIsNow(t *testing.T) {
	combo, then, now := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()

	sold, _ := soldDishes([]models.Order{{
		Status: models.StatusDelivered,
		Items: []models.OrderItem{{
			MenuItemID: combo, Qty: 1,
			ComboItems: []models.OrderComboLine{{MenuItemID: then, Qty: 1}},
		}},
	}}, map[primitive.ObjectID][]models.ComboLine{
		combo: {{MenuItemID: now, Qty: 1}},
	})

	if sold[then] != 1 || sold[now] != 0 {
		t.Fatal("the set was opened from today's menu instead of the order")
	}
}

// Orders taken before members carried ids have no frozen answer, and the live
// set is the only one there is — the same fallback comboMembersFor makes.
func TestAnOldSetFallsBackToTheLiveDefinition(t *testing.T) {
	combo, lagmon := primitive.NewObjectID(), primitive.NewObjectID()

	sold, _ := soldDishes([]models.Order{{
		Status: models.StatusDelivered,
		Items: []models.OrderItem{{
			MenuItemID: combo, Qty: 3,
			// No MenuItemID: this order predates the field.
			ComboItems: []models.OrderComboLine{{Name: "Lag'mon", Qty: 1}},
		}},
	}}, map[primitive.ObjectID][]models.ComboLine{
		combo: {{MenuItemID: lagmon, Qty: 1}},
	})

	if sold[lagmon] != 3 {
		t.Fatalf("the old set opened into %d, wanted 3", sold[lagmon])
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

// ⚠️ **Ranked on what was bought, not on what the cards say was used.** The
// spend is measured — an invoice with a date and a total — while the usage is
// an estimate only as good as the cards behind it, and half a menu is usually
// uncosted. A ranking built on the estimate puts the ten dishes somebody
// happened to write cards for at the top and calls that where the money goes.
func TestIngredientsAreClassedByWhatTheyCost(t *testing.T) {
	rows := []stockRow{
		{Name: "Go'sht", Spent: 800},
		{Name: "Guruch", Spent: 150},
		{Name: "Ziravor", Spent: 50},
	}
	classifySpend(rows)

	if rows[0].ABC != "A" {
		t.Fatalf("the biggest spend is class %q", rows[0].ABC)
	}
	if rows[1].ABC != "B" {
		t.Fatalf("the middle spend is class %q", rows[1].ABC)
	}
	if rows[2].ABC != "C" {
		t.Fatalf("the smallest spend is class %q", rows[2].ABC)
	}
}

// ⚠️ **The line that crosses 80% stays in A** — the dish report's rule, taken
// deliberately. Cutting after the row is added pushes it into B, and on a short
// list that ingredient is often a large part of the spend: precisely the one an
// owner must not be told to stop worrying about.
func TestTheIngredientCrossingTheLineStaysInA(t *testing.T) {
	rows := []stockRow{{Name: "Go'sht", Spent: 900}, {Name: "Guruch", Spent: 100}}
	classifySpend(rows)

	if rows[0].ABC != "A" {
		t.Fatalf("a single ingredient holding 90%% of the spend is class %q", rows[0].ABC)
	}
}

// Nothing bought is not everything in class C: an empty period must not label
// rows at all, or the letters describe a ranking of nothing.
func TestAnEmptyPeriodIsNotClassified(t *testing.T) {
	rows := []stockRow{{Name: "Go'sht"}, {Name: "Guruch"}}
	classifySpend(rows)

	for _, r := range rows {
		if r.ABC != "" {
			t.Fatalf("%s was classed %q with nothing bought", r.Name, r.ABC)
		}
	}
}
