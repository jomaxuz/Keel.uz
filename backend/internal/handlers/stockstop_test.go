package handlers

import (
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

// ⚠️ **Zero has two meanings and only one of them may close a kitchen.**
// "We have none" is a reason to stop a dish; "nobody has ever counted this
// store" is not — and before the first stocktake every expected figure is the
// second one. Without this guard, switching the feature on would empty the menu
// of every restaurant that had not yet counted anything, during service, on the
// day they turned it on.
func TestAnUncountedStoreStopsNothing(t *testing.T) {
	bar := primitive.NewObjectID()
	kitchen := primitive.NewObjectID()
	vodka := primitive.NewObjectID()
	beef := primitive.NewObjectID()

	counted := time.Now().Add(-24 * time.Hour)
	ingredients := []models.Ingredient{
		{ID: vodka, Name: "Vodka"},
		{ID: beef, Name: "Mol go'shti"},
	}
	// Where this branch keeps them — a fact about the branch, not the
	// ingredient, since brands and branches own different halves of it.
	placed := map[primitive.ObjectID]primitive.ObjectID{vodka: bar, beef: kitchen}
	balances := map[primitive.ObjectID]map[primitive.ObjectID]float64{
		bar:     {vodka: 0},
		kitchen: {beef: 0},
	}
	// The bar has been counted; the kitchen never has.
	since := map[primitive.ObjectID]*time.Time{bar: &counted, kitchen: nil}

	empty := emptyIngredients(balances, since, ingredients, placed)
	if !empty[vodka] {
		t.Error("a counted store showing nothing left must stop its dishes")
	}
	if empty[beef] {
		t.Fatal("an uncounted store must not stop anything — zero there means 'nobody told us'")
	}
}

// ⚠️ A prep item is not on a shelf as itself. What it was made from is, and
// that is what gets checked — counting the pot of sauce as well would stop a
// dish twice for one shortage, and the second reason would name something
// nobody can go out and buy.
func TestPrepItemsAreNotShelves(t *testing.T) {
	kitchen := primitive.NewObjectID()
	tomato := primitive.NewObjectID()
	sauce := primitive.NewObjectID()
	counted := time.Now()

	ingredients := []models.Ingredient{
		{ID: tomato, Name: "Pomidor"},
		{
			ID: sauce, Name: "Sous", Output: 2000,
			Recipe: []models.RecipeLine{{IngredientID: tomato, Qty: 1500}},
		},
	}
	placed := map[primitive.ObjectID]primitive.ObjectID{tomato: kitchen, sauce: kitchen}
	balances := map[primitive.ObjectID]map[primitive.ObjectID]float64{
		kitchen: {tomato: 4, sauce: 0},
	}
	since := map[primitive.ObjectID]*time.Time{kitchen: &counted}

	empty := emptyIngredients(balances, since, ingredients, placed)
	if empty[sauce] {
		t.Error("a prep item must not be treated as an empty shelf")
	}
	if empty[tomato] {
		t.Error("an ingredient with stock left must not be called empty")
	}
}

// ⚠️ **A shortage is followed through the prep cards.** A dish made with a
// sauce made with tomatoes is short of tomatoes — and a check that only read
// the dish's own lines would sell it. That is the shape of bug that makes a
// stop list stop being trusted, because it is wrong only for the dishes with
// the most work in them.
func TestAShortageIsFollowedThroughPrepCards(t *testing.T) {
	tomato := primitive.NewObjectID()
	sauce := primitive.NewObjectID()
	salt := primitive.NewObjectID()

	ingredients := []models.Ingredient{
		{ID: tomato, Name: "Pomidor"},
		{ID: salt, Name: "Tuz"},
		{
			ID: sauce, Name: "Sous", Output: 2000,
			Recipe: []models.RecipeLine{{IngredientID: tomato, Qty: 1500}},
		},
	}
	pasta := primitive.NewObjectID()
	plain := primitive.NewObjectID()
	uncosted := primitive.NewObjectID()
	menu := []models.MenuItem{
		{ID: pasta, Recipe: []models.RecipeLine{{IngredientID: sauce, Qty: 120}}},
		{ID: plain, Recipe: []models.RecipeLine{{IngredientID: salt, Qty: 5}}},
		// No card at all — see below.
		{ID: uncosted},
	}

	stopped := dishesShortOf(menu, ingredients, map[primitive.ObjectID]bool{tomato: true}, nil)

	if !containsID(stopped, pasta) {
		t.Error("a dish is not short of what its own sauce is made from")
	}
	if containsID(stopped, plain) {
		t.Error("a dish whose ingredients are all in stock was stopped")
	}
	// ⚠️ An empty recipe is "nobody has written this one down", not "this needs
	// nothing". Stopping it would take out most of the menu at a restaurant
	// that has costed half of it — which is every restaurant, halfway through.
	if containsID(stopped, uncosted) {
		t.Fatal("a dish with no tech card must never be stopped by stock")
	}
}

// Two cards that reference each other must not hang the sync.
//
// ⚠️ Not a hypothetical: prep items may contain prep items, the editor does not
// forbid a loop, and this runs in a background goroutine where a hang is
// invisible until the stop list quietly stops updating.
func TestALoopInThePrepCardsTerminates(t *testing.T) {
	a := primitive.NewObjectID()
	b := primitive.NewObjectID()
	ingredients := []models.Ingredient{
		{ID: a, Output: 100, Recipe: []models.RecipeLine{{IngredientID: b, Qty: 1}}},
		{ID: b, Output: 100, Recipe: []models.RecipeLine{{IngredientID: a, Qty: 1}}},
	}
	dish := primitive.NewObjectID()
	menu := []models.MenuItem{
		{ID: dish, Recipe: []models.RecipeLine{{IngredientID: a, Qty: 1}}},
	}

	done := make(chan []primitive.ObjectID, 1)
	go func() {
		done <- dishesShortOf(menu, ingredients, map[primitive.ObjectID]bool{}, nil)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a cycle in the prep cards did not terminate")
	}
}

// ⚠️ The settings form must not write the stock stop list. It is the third time
// this trap has been worth a test: the form does not know what the shelves
// hold, so saving it would put back every dish the sync has stopped — until the
// next run took them off again, which reads as the toggle not working.
func TestTheBranchFormDoesNotWriteTheStockStopList(t *testing.T) {
	src := readSource(t, "brands.go")
	fn := between(t, src, "func (h *Handler) AdminUpdateBranch", "\n}\n")
	for _, field := range []string{"stockSoldOut", "stockSoldOutAt"} {
		if !strings.Contains(fn, `delete(set, "`+field+`")`) {
			t.Fatalf("the branch form writes %s over the stock sync's own list", field)
		}
	}
	// The switch itself is the owner's, and has to stay writable — a toggle
	// nothing can turn on is a feature nobody has.
	if strings.Contains(fn, `delete(set, "stockStop")`) {
		t.Fatal("the owner can no longer switch stock stopping on")
	}
}

// ---- The bar: one bottle, three measures ----

// ⚠️ **A required group stops the dish only when every answer is short.** A
// vodka offered at 40, 50 and 100 ml is unorderable when the vodka is gone —
// all three answers are the same empty bottle. It is still perfectly orderable
// when one size has run out and another has not, and stopping it then would
// take a sellable dish off the menu.
func TestAPourStopsTheDrinkOnlyWhenEveryMeasureIsShort(t *testing.T) {
	vodka := primitive.NewObjectID()
	gin := primitive.NewObjectID()
	tonic := primitive.NewObjectID()

	pours := func(id primitive.ObjectID) models.MenuOption {
		return models.MenuOption{
			Name: "Hajm", Required: true,
			Choices: []models.OptionChoice{
				{Name: "40 ml", Recipe: []models.RecipeLine{{IngredientID: id, Qty: 40}}},
				{Name: "50 ml", Recipe: []models.RecipeLine{{IngredientID: id, Qty: 50}}},
				{Name: "100 ml", Recipe: []models.RecipeLine{{IngredientID: id, Qty: 100}}},
			},
		}
	}

	vodkaDrink := primitive.NewObjectID()
	ginTonic := primitive.NewObjectID()
	menu := []models.MenuItem{
		// The whole card is on the choices — the bar's ordinary shape.
		{ID: vodkaDrink, Options: []models.MenuOption{pours(vodka)}},
		// A dish card plus a pour: the tonic and the lemon whichever measure
		// of gin goes in.
		{
			ID:      ginTonic,
			Recipe:  []models.RecipeLine{{IngredientID: tonic, Qty: 200}},
			Options: []models.MenuOption{pours(gin)},
		},
	}

	// The vodka is gone: every measure of it is the same empty bottle.
	stopped := dishesShortOf(menu, nil, map[primitive.ObjectID]bool{vodka: true}, nil)
	if !containsID(stopped, vodkaDrink) {
		t.Error("a drink whose only card is on its pours was not stopped")
	}
	if containsID(stopped, ginTonic) {
		t.Error("the gin and tonic has both its gin and its tonic")
	}

	// The tonic is gone: the dish's own card is short, whatever the pour.
	stopped = dishesShortOf(menu, nil, map[primitive.ObjectID]bool{tonic: true}, nil)
	if !containsID(stopped, ginTonic) {
		t.Error("a drink is not short of what its own card is made from")
	}
	if containsID(stopped, vodkaDrink) {
		t.Error("the vodka has nothing to do with the tonic")
	}
}

// ⚠️ An optional group never stops anything. "Extra olive" being out is a
// reason to refuse the olive, not the drink — refusing the drink would be the
// stop list making a decision the guest was never asked about.
func TestAnOptionalExtraNeverStopsTheDish(t *testing.T) {
	olive := primitive.NewObjectID()
	gin := primitive.NewObjectID()
	drink := primitive.NewObjectID()
	menu := []models.MenuItem{{
		ID:     drink,
		Recipe: []models.RecipeLine{{IngredientID: gin, Qty: 50}},
		Options: []models.MenuOption{{
			Name: "Qo'shimcha", Required: false,
			Choices: []models.OptionChoice{
				{Name: "Zaytun", Recipe: []models.RecipeLine{{IngredientID: olive, Qty: 2}}},
			},
		}},
	}}

	if containsID(dishesShortOf(menu, nil, map[primitive.ObjectID]bool{olive: true}, nil), drink) {
		t.Fatal("an optional extra running out took the whole drink off the menu")
	}
}

// ⚠️ A required group whose choices carry no card at all is "nobody has written
// this down", not "every answer is short" — the same rule the dish itself
// follows. "Katta"/"kichik" on a pizza changes the price and nothing the store
// can measure.
func TestASizeThatIsOnlyAPriceStopsNothing(t *testing.T) {
	cheese := primitive.NewObjectID()
	pizza := primitive.NewObjectID()
	menu := []models.MenuItem{{
		ID:     pizza,
		Recipe: []models.RecipeLine{{IngredientID: cheese, Qty: 100}},
		Options: []models.MenuOption{{
			Name: "O'lcham", Required: true,
			Choices: []models.OptionChoice{
				{Name: "Kichik", PriceDelta: 0},
				{Name: "Katta", PriceDelta: 15000},
			},
		}},
	}}

	if containsID(dishesShortOf(menu, nil, map[primitive.ObjectID]bool{}, nil), pizza) {
		t.Fatal("a size group that is only a price stopped the dish")
	}
	// …and the dish's own shortage still stops it.
	if !containsID(dishesShortOf(menu, nil, map[primitive.ObjectID]bool{cheese: true}, nil), pizza) {
		t.Error("the dish's own card no longer stops it")
	}
}

// ⚠️ **A set whose dishes are off has to be off too.** A combo has no card of
// its own, so both of the checks that stop a dish look at an empty recipe and
// let it through — and the kitchen gets an order for a family set on an evening
// the same screen has already stopped every dish inside it.
func TestASetIsStoppedWhenADishInItIs(t *testing.T) {
	tomato := primitive.NewObjectID()
	lagmon := models.MenuItem{
		ID:     primitive.NewObjectID(),
		Recipe: []models.RecipeLine{{IngredientID: tomato, Qty: 100}},
	}
	tea := models.MenuItem{
		ID:     primitive.NewObjectID(),
		Recipe: []models.RecipeLine{{IngredientID: primitive.NewObjectID(), Qty: 5}},
	}
	combo := models.MenuItem{
		ID: primitive.NewObjectID(),
		ComboItems: []models.ComboLine{
			{MenuItemID: lagmon.ID, Qty: 1}, {MenuItemID: tea.ID, Qty: 1},
		},
	}

	stopped := dishesShortOf(
		[]models.MenuItem{lagmon, tea, combo}, nil,
		map[primitive.ObjectID]bool{tomato: true}, nil,
	)

	got := map[primitive.ObjectID]bool{}
	for _, id := range stopped {
		got[id] = true
	}
	if !got[combo.ID] {
		t.Fatal("the set is still sellable with its lagmon stopped")
	}
	if got[tea.ID] {
		t.Fatal("the tea was stopped as well — nothing of it has run out")
	}
}

// ⚠️ And a set stays sellable while its dishes are: the arithmetic must not
// stop a combo merely for having no card of its own.
func TestASetIsNotStoppedJustForBeingASet(t *testing.T) {
	lagmon := models.MenuItem{
		ID:     primitive.NewObjectID(),
		Recipe: []models.RecipeLine{{IngredientID: primitive.NewObjectID(), Qty: 100}},
	}
	combo := models.MenuItem{
		ID:         primitive.NewObjectID(),
		ComboItems: []models.ComboLine{{MenuItemID: lagmon.ID, Qty: 1}},
	}

	if len(dishesShortOf([]models.MenuItem{lagmon, combo}, nil, nil, nil)) != 0 {
		t.Fatal("a set was stopped with a full store behind it")
	}
}

// ⚠️ **Two hundred grams against a five-hundred-gram card is a dish that
// cannot be cooked**, and the zero test let it be sold. Not enough for one
// portion is the same statement as empty, measured against the dish instead of
// against zero — and it is the door the whole feature's failure came through.
func TestADishIsStoppedWhenTheShelfCannotMakeOne(t *testing.T) {
	beef := models.Ingredient{
		ID: primitive.NewObjectID(), Unit: models.UnitKg,
	}
	// 500 g of beef per portion.
	steak := models.MenuItem{
		ID:     primitive.NewObjectID(),
		Recipe: []models.RecipeLine{{IngredientID: beef.ID, Qty: 500}},
	}
	menu := []models.MenuItem{steak}
	ings := []models.Ingredient{beef}

	// 0.2 kg on the shelf: nothing is empty, and the dish still cannot be made.
	short := dishesShortOf(menu, ings, nil,
		map[primitive.ObjectID]float64{beef.ID: 0.2})
	if !containsID(short, steak.ID) {
		t.Fatal("a dish was sellable with less than one portion of beef left")
	}

	// 0.6 kg is one portion and change: sellable.
	ok := dishesShortOf(menu, ings, nil,
		map[primitive.ObjectID]float64{beef.ID: 0.6})
	if containsID(ok, steak.ID) {
		t.Fatal("a dish with a portion left was stopped")
	}
}

// ⚠️ **One portion, never two.** The balance is an estimate; anything past
// "we cannot make this one" is a forecast about how many are about to be
// ordered, which no arithmetic over last month's invoices can know. Refusing a
// sale is the most expensive thing this system does.
func TestExactlyOnePortionIsStillSellable(t *testing.T) {
	flour := models.Ingredient{ID: primitive.NewObjectID(), Unit: models.UnitKg}
	bread := models.MenuItem{
		ID:     primitive.NewObjectID(),
		Recipe: []models.RecipeLine{{IngredientID: flour.ID, Qty: 1000}},
	}

	short := dishesShortOf([]models.MenuItem{bread}, []models.Ingredient{flour}, nil,
		map[primitive.ObjectID]float64{flour.ID: 1})
	if containsID(short, bread.ID) {
		t.Fatal("the last portion was refused")
	}
}

// ⚠️ **An uncounted shelf is unknown, not empty**, one level up from where that
// guard already lived: a missing key must never read as nothing, or switching
// the feature on empties the menu of every restaurant that has never counted.
func TestAnUncountedShelfCannotStopADish(t *testing.T) {
	beef := models.Ingredient{ID: primitive.NewObjectID(), Unit: models.UnitKg}
	steak := models.MenuItem{
		ID:     primitive.NewObjectID(),
		Recipe: []models.RecipeLine{{IngredientID: beef.ID, Qty: 500}},
	}
	// The shelf map holds a different ingredient entirely: beef is unknown.
	shelf := map[primitive.ObjectID]float64{primitive.NewObjectID(): 3}

	if len(dishesShortOf([]models.MenuItem{steak}, []models.Ingredient{beef}, nil, shelf)) != 0 {
		t.Fatal("an ingredient nobody has counted stopped a dish")
	}
}

// ⚠️ Followed through the prep cards, using the same resolution the consumption
// report uses: a dish made with a sauce made with tomatoes needs tomatoes, and
// "we cannot make this" must never disagree with "this is what making it used".
func TestAPortionIsMeasuredThroughPrepCards(t *testing.T) {
	tomato := models.Ingredient{ID: primitive.NewObjectID(), Unit: models.UnitKg}
	// 3 kg of tomatoes yields 2 kg of sauce.
	sauce := models.Ingredient{
		ID: primitive.NewObjectID(), Unit: models.UnitKg,
		Recipe: []models.RecipeLine{{IngredientID: tomato.ID, Qty: 3000}},
		Output: 2000,
	}
	// 200 g of sauce per plate → 300 g of tomatoes.
	pasta := models.MenuItem{
		ID:     primitive.NewObjectID(),
		Recipe: []models.RecipeLine{{IngredientID: sauce.ID, Qty: 200}},
	}
	menu := []models.MenuItem{pasta}
	ings := []models.Ingredient{tomato, sauce}

	if !containsID(dishesShortOf(menu, ings, nil,
		map[primitive.ObjectID]float64{tomato.ID: 0.2}), pasta.ID) {
		t.Fatal("200 g of tomatoes served a plate needing 300 g")
	}
	if containsID(dishesShortOf(menu, ings, nil,
		map[primitive.ObjectID]float64{tomato.ID: 0.4}), pasta.ID) {
		t.Fatal("400 g of tomatoes could not serve a plate needing 300 g")
	}
}
