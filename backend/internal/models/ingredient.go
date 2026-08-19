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
	Note      string    `bson:"note,omitempty" json:"note,omitempty"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// CostPerRecipeUnit is what one gram, millilitre or piece costs.
//
// ⚠️ **Kept as a rate rather than rounded to a som.** A gram of anything is
// well under one som, and rounding here would turn a 42 000-som kilo into a dish
// cost of zero. The rounding happens once, at the finished dish.
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
