package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Ingredients and tech cards ----
//
// ⚠️ **The point is that a cost stops being a number somebody remembers to
// update.** A dish's cost was typed by hand: correct on the day it was typed
// and quietly wrong from the next delivery onwards. A tech card moves the fact
// to where it actually changes — meat went up, and every dish with meat in it
// is dearer the same afternoon.
//
// ⚠️ **This is costing, not stock.** Nothing here tracks what is in the
// fridge, and pretending otherwise would be the expensive kind of half-feature:
// a restaurant that believes its stock figures and finds them wrong stops
// believing the panel entirely. Quantities are what leaves the store to make
// the dish (brutto) — which is exactly what it costs.

// UnitKg and friends are the three families a kitchen buys in.
//
// ⚠️ **Three, deliberately, and no conversion engine.** Real ingredient lists
// are weight, volume and count; everything else people write in ("bunch",
// "packet") is a purchase unit whose weight only the buyer knows. Offering a
// free-text unit would produce recipes that cannot be costed and a cost that
// looks computed.
const (
	// Bought by the kilo, put in the dish by the gram.
	UnitKg = "kg"
	// Bought by the litre, put in the dish by the millilitre.
	UnitL = "l"
	// Bought and used by the piece: eggs, lavash, a bottle of water.
	UnitPcs = "pcs"
)

// PerUnit is how many recipe units are in one purchase unit.
//
// ⚠️ The whole of the unit handling, in one place and three lines: a kilo is a
// thousand grams and a litre is a thousand millilitres. Owners type prices the
// way they buy ("one kilo, 42 000") and recipes the way they cook ("180 g"),
// and the arithmetic between the two is the only conversion this needs.
func PerUnit(unit string) int {
	switch unit {
	case UnitKg, UnitL:
		return 1000
	default:
		return 1
	}
}

// RecipeUnit is what a recipe line is measured in, for the screen's label.
func RecipeUnit(unit string) string {
	switch unit {
	case UnitKg:
		return "g"
	case UnitL:
		return "ml"
	default:
		return UnitPcs
	}
}

// PriceEntry is what an ingredient cost from a given day.
type PriceEntry struct {
	Price int       `bson:"price" json:"price"`
	At    time.Time `bson:"at" json:"at"`
	// Which delivery claimed this price, when one did.
	//
	// ⚠️ **This is what makes an invoice correctable.** Without it, fixing a
	// mistyped price means writing a second entry on the same day beside the
	// first, and the history — whose whole job is to answer "when did this go
	// up" — grows a contradiction that nothing can resolve afterwards. With it,
	// an edit withdraws exactly what that invoice claimed and nothing else.
	//
	// ⚠️ **Empty is a hand edit or an entry from before this existed**, and
	// those are never withdrawn by anything: nobody can say which invoice they
	// belonged to, and guessing would delete somebody's deliberate correction.
	PurchaseID primitive.ObjectID `bson:"purchaseId,omitempty" json:"-"`
}

// Ingredient is one thing the kitchen buys.
type Ingredient struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BrandID primitive.ObjectID `bson:"brandId,omitempty" json:"brandId,omitempty"`
	Name    string             `bson:"name" json:"name" validate:"required"`
	// "kg" | "l" | "pcs" — how it is bought and priced.
	Unit string `bson:"unit" json:"unit"`
	// What one purchase unit costs, in whole so'm: one kilo, one litre, one
	// piece. ⚠️ Not per gram — nobody has a price per gram written anywhere,
	// and asking for one is asking to be given the wrong number.
	Price int `bson:"price" json:"price"`
	// Free text: "Makro, 3-sort". Kept because the price only means something
	// beside where it came from, and the next person to update it needs that.
	Note string `bson:"note,omitempty" json:"note,omitempty"`

	// ⚠️ **Legacy: read only by the migration.** Where an ingredient is kept
	// used to live here, which quietly assumed one branch — the catalogue is
	// the brand's and the rooms are the branch's, and one field cannot be both.
	// With two branches the second one could not count anything and had its
	// consumption filed against the first one's shelf, silently.
	//
	// `EnsureIngredientPlacements` moves each of these onto an
	// `ingredient_placement` row and nothing reads it afterwards. It is kept
	// rather than dropped so a rollback still has the data, and so a database
	// restored from an old backup migrates itself on the next boot.
	WarehouseID primitive.ObjectID `bson:"warehouseId,omitempty" json:"-"`

	// Order more when there is less than this, in purchase units.
	//
	// ⚠️ **Zero means "do not warn me", not "warn me at zero".** Most
	// ingredients never need this — nobody tracks a minimum for cinnamon — and
	// a list where every line eventually turns red is a list nobody reads. It
	// is opt-in, one ingredient at a time, for the ten that stop service when
	// they run out.
	MinQty float64 `bson:"minQty,omitempty" json:"minQty,omitempty"`

	// Every price this ingredient has had, oldest first.
	//
	// ⚠️ **Without it, raising a price rewrites history.** Beef going up today
	// would make March's dishes cost today's beef, and March's margin — a
	// number somebody has already looked at, quoted, maybe used to set a
	// price — would quietly become a different number. A report that changes
	// when nothing about that month changed is a report nobody can rely on.
	//
	// ⚠️ It is also the only way to tell the two edits apart: "the price went
	// up" belongs from today, and both look identical on the form. So an edit
	// is always **from now on**, and correcting a past figure is deliberately
	// not offered here — inventing a retroactive edit would put the rewriting
	// back, dressed as a feature.
	History []PriceEntry `bson:"history,omitempty" json:"history,omitempty"`

	// ---- Made in-house ----
	//
	// ⚠️ **A sauce is not bought, it is cooked**, and without this every dish
	// containing it has to list its tomatoes again. That is the reason tech
	// cards get abandoned: a kitchen with six sauces and forty dishes ends up
	// maintaining the same recipe in seven places, and they stop agreeing
	// within a month.
	//
	// An ingredient with a card is a **prep item**: its price is not typed, it
	// is what one batch costs divided by what the batch yields.
	Recipe []RecipeLine `bson:"recipe,omitempty" json:"recipe,omitempty"`
	// How many recipe units one batch produces — grams of sauce, millilitres of
	// stock, pieces of dough.
	//
	// ⚠️ **This is the yield, and it is where a prep card is honest about
	// evaporation.** Three kilos of tomatoes that boil down to two kilos of
	// sauce yield 2000, not 3000, and a card that says otherwise underprices
	// every dish the sauce is in — which is the whole failure this feature is
	// supposed to prevent, moved one level down.
	Output    float64   `bson:"output,omitempty" json:"output,omitempty"`
	// Batched marks a prep item that is **made in batches and kept on a
	// shelf**, rather than derived from what a dish sold.
	//
	// ⚠️ **This is the central-kitchen switch, and it changes what a number
	// means everywhere.** Ordinarily a prep item is not stock at all: nobody
	// counts "sauce", they count the tomatoes, and a dish that uses sauce is
	// read as having used tomatoes (`rawInputs`). That is right for one
	// kitchen making sauce as it goes, and wrong the moment a central kitchen
	// makes forty kilos on Monday and ships it to three branches — where it
	// *is* a physical thing in a fridge, and where the tomatoes never were.
	//
	// With this on, the item behaves like something bought: it is counted,
	// transferred, warned about — and a dish consumes **it** rather than what
	// it was made of. Its inputs are taken by the production document instead
	// (`models.Production`), which is what stops both from being subtracted.
	//
	// ⚠️ Empty is off, which is every prep item written before this existed and
	// every restaurant with one kitchen.
	Batched   bool      `bson:"batched,omitempty" json:"batched,omitempty"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// MadeInHouse reports whether this is cooked rather than bought.
func (i Ingredient) MadeInHouse() bool { return len(i.Recipe) > 0 && i.Output > 0 }

// DerivedOnly reports whether this item exists only as its inputs — cooked in
// house and **not** produced in batches.
//
// ⚠️ **This is the question every stock screen actually asks**, and it used to
// be spelled `MadeInHouse` because the two were the same thing. They stopped
// being the same when a batch could be made and shipped: a batched prep item is
// on a shelf, is counted, and moves between stores like anything else. Every
// place that skips prep items has to ask this one instead, or a central
// kitchen's output is invisible to the store that received it.
func (i Ingredient) DerivedOnly() bool { return i.MadeInHouse() && !i.Batched }

// CostPerRecipeUnit is what one gram, millilitre or piece of a **bought**
// ingredient costs.
//
// ⚠️ **Kept as a rate rather than rounded to a som.** A gram of anything is
// well under one som, and rounding here would turn a 42 000-som kilo into a dish
// cost of zero. The rounding happens once, at the finished dish.
//
// ⚠️ A prep item's rate is not this: it comes from its own card and its yield,
// which needs every other rate first — see ingredientRates.
func (i Ingredient) CostPerRecipeUnit() float64 {
	per := PerUnit(i.Unit)
	if per <= 0 {
		return 0
	}
	return float64(i.Price) / float64(per)
}

// RecipeLine is one ingredient in a dish, in recipe units (g, ml or pcs).
//
// ⚠️ **Brutto: what leaves the store**, not what ends up on the plate. A
// kilogram of potatoes costs a kilogram whether or not a third of it is peel,
// and costing the peeled weight is how a tech card quietly understates every
// dish it describes. Yield and portioning are a stock question and are not
// modelled here — see the package note.
type RecipeLine struct {
	IngredientID primitive.ObjectID `bson:"ingredientId" json:"ingredientId"`
	// Grams, millilitres or pieces, matching the ingredient's own unit.
	Qty float64 `bson:"qty" json:"qty"`
}

// PriceAt is what one purchase unit cost on a given day.
//
// ⚠️ **A price we only learned later is the best answer for earlier periods.**
// An ingredient added today with no history has to cost something in last
// month's report, and the alternative — costing it at nothing — would make
// every dish containing it look free. So the oldest entry reaches back.
func (i Ingredient) PriceAt(at time.Time) int {
	price := i.Price
	if len(i.History) == 0 {
		return price
	}
	price = i.History[0].Price
	for _, e := range i.History {
		if e.At.After(at) {
			break
		}
		price = e.Price
	}
	return price
}
