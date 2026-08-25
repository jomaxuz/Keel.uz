package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Making a batch ----
//
// ⚠️ **The document a central kitchen needs and a single kitchen does not.** A
// restaurant that makes its sauce as it goes needs no record of it: the sauce
// is never on a shelf, and a dish that used it is read as having used the
// tomatoes (`rawInputs`). That is exactly the arithmetic a chain breaks. The
// central kitchen makes forty kilos on Monday and sends it to three branches,
// where the sauce **is** a physical thing in a fridge and the tomatoes never
// were — so somewhere the tomatoes have to stop being the sauce and the sauce
// has to start being stock. This is that moment.
//
// ⚠️ **It is the counterpart of `Ingredient.Batched` and neither works alone.**
// The flag says the item is kept on a shelf and consumed as itself; the
// document is what puts it there and takes its inputs off. With the flag and no
// document the shelf never fills, and stock goes negative on a real thing. With
// the document and no flag the inputs are subtracted twice — once here, once
// through the card when a dish is sold — and the shortfall appears weeks later
// at a count, as an accusation against whoever counted.
//
// ⚠️ **Value is carried, not created**, exactly as a transfer's is: nothing was
// bought and nothing was lost, tomatoes became sauce. The figure is recorded so
// the movement report can say what a batch was worth, and it is deliberately
// absent from the financial report's expenses.
type Production struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	At       time.Time          `bson:"at" json:"at"`
	// The store the batch was cooked in and lands in. ⚠️ Must be a production
	// store: the inputs come off *these* shelves, and a batch made anywhere
	// would take tomatoes standing in another branch's kitchen.
	WarehouseID primitive.ObjectID `bson:"warehouseId" json:"warehouseId"`
	// What was made.
	IngredientID primitive.ObjectID `bson:"ingredientId" json:"ingredientId"`
	// How much, in the item's **purchase** unit — the unit it is counted and
	// transferred in, not the recipe's grams.
	Qty float64 `bson:"qty" json:"qty"`
	// What it took, frozen.
	//
	// ⚠️ **Frozen rather than re-derived from the card.** The card is corrected
	// as recipes change, and a batch made in March took what it took in March.
	// Re-reading it would rewrite a month that has already been counted and
	// reconciled — the same reason a write-off's value is frozen.
	Lines []ProductionLine `bson:"lines" json:"lines"`
	// What the inputs were worth on the day, in so'm.
	Value int    `bson:"value" json:"value"`
	Note  string `bson:"note,omitempty" json:"note,omitempty"`

	By        string             `bson:"by,omitempty" json:"by,omitempty"`
	ByID      primitive.ObjectID `bson:"byId,omitempty" json:"-"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

// ProductionLine is one input a batch consumed.
type ProductionLine struct {
	IngredientID primitive.ObjectID `bson:"ingredientId" json:"ingredientId"`
	// Frozen so a renamed ingredient does not rewrite last month's batches.
	Name string `bson:"name,omitempty" json:"name,omitempty"`
	// In purchase units, like everything the balance arithmetic speaks.
	Qty float64 `bson:"qty" json:"qty"`
}
