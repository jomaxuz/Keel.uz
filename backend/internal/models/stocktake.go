package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Counting what is actually there ----
//
// ⚠️ **This is the piece that turns the flow report's difference from a
// question into an answer.** Deliveries say what came in, the cards say what
// the dishes should have used, write-offs say what was thrown away — and the
// remainder is still a guess until somebody walks into the store and counts.
//
// ⚠️ **The difference is the product**, exactly as it is for the cash drawer.
// A count that stores only what was found has recorded nothing: the shortfall
// it exists to surface has been overwritten by the person who might have caused
// it. So `Expected` is computed by the server, frozen at the moment of saving,
// and a count that disagrees cannot be saved without a sentence.
type StocktakeLine struct {
	IngredientID primitive.ObjectID `bson:"ingredientId" json:"ingredientId"`
	// What was found, in purchase units — the way it is counted on the shelf.
	Counted float64 `bson:"counted" json:"counted"`
	// What should have been there, from the previous count plus deliveries
	// less what the cards and the write-offs account for.
	//
	// ⚠️ Frozen: recomputing it when the page is opened again would quietly
	// change a variance somebody has already explained, the first time a late
	// invoice is entered.
	Expected float64 `bson:"expected" json:"expected"`
	Diff     float64 `bson:"diff" json:"diff"`
	// What the difference was worth, at that day's prices.
	Value int `bson:"value" json:"value"`
}

// Stocktake is one count of the store.
type Stocktake struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	// Which store was walked into and counted.
	//
	// ⚠️ **A count is one room, and this is what makes that true.** Counting
	// the bar and the kitchen as one list let a shortfall behind the bar cancel
	// against a surplus in the kitchen — arithmetically fine and useless to
	// act on, because the two are counted by different people on different
	// evenings. Empty means the one undivided store, which is every count taken
	// before warehouses existed and every restaurant that never splits them.
	WarehouseID primitive.ObjectID `bson:"warehouseId,omitempty" json:"warehouseId,omitempty"`
	At          time.Time          `bson:"at" json:"at"`
	Lines       []StocktakeLine    `bson:"lines" json:"lines"`
	// Why it disagreed — the same rule the cash drawer follows, and for the
	// same reason: a number nobody explained is a number nobody can use.
	//
	// ⚠️ **Asked for after the count is saved, never before it is accepted.**
	// It used to be a condition of saving, which was right while the counter
	// could see the expected figures: they knew what they were explaining. Once
	// the sheet went blind that same refusal became an oracle — enter numbers,
	// get "there is a variance", adjust, try again, and the rejection itself
	// tells you when you have matched the books. A count is now accepted
	// unconditionally and explained afterwards, when the numbers can no longer
	// be moved.
	Note string `bson:"note,omitempty" json:"note,omitempty"`
	// When the explanation was given, which is a different fact from what it
	// says.
	//
	// ⚠️ **Set once and never changed.** An explanation that can be rewritten
	// next week is not one — and the gap between the count and the note is
	// itself worth seeing: a variance explained three days later was explained
	// by somebody who had time to think about it.
	NotedAt *time.Time `bson:"notedAt,omitempty" json:"notedAt,omitempty"`
	// What the whole count was out by, in money. The line an owner reads.
	Value     int                `bson:"value" json:"value"`
	By        string             `bson:"by,omitempty" json:"by,omitempty"`
	ByID      primitive.ObjectID `bson:"byId,omitempty" json:"-"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}
