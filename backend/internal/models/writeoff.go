package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Writing food off ----
//
// ⚠️ **The half of the flow nobody records, and the reason the difference
// column cannot yet be an answer.** Food goes out of a restaurant in three
// ways: it is sold, it is eaten by the staff, and it is thrown away. The cards
// account for the first; without the other two the flow report shows a gap and
// leaves whoever reads it to guess whether it is waste, theft or a delivery
// still in the fridge.
//
// ⚠️ **A reason is required**, like every other place in this system where
// something disappears: a void, a refund, a cancelled order. "Twelve kilos of
// beef, written off" with no sentence beside it is the line every argument
// starts from, and the person who could answer has gone home.
type WriteOff struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID     primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	At           time.Time          `bson:"at" json:"at"`
	IngredientID primitive.ObjectID `bson:"ingredientId" json:"ingredientId"`
	// In purchase units — kilos, litres, pieces — because that is how it is
	// counted at the bin, and how the delivery it came from was counted.
	Qty float64 `bson:"qty" json:"qty"`
	// Why. Free text on purpose: every kitchen throws away something the next
	// one does not, and a fixed list sends all of it to "other" (the same
	// judgement the cash drawer's categories are built on).
	Reason string `bson:"reason" json:"reason"`
	// What it was worth, frozen at the prices of that day.
	//
	// ⚠️ **Stored rather than recomputed.** This is the number that changes
	// behaviour — an owner who sees "3 200 000 thrown away this month" moves
	// differently from one who sees "eleven write-offs" — and it must not shift
	// under them when the next delivery arrives at a different price.
	Value     int                `bson:"value" json:"value"`
	By        string             `bson:"by,omitempty" json:"by,omitempty"`
	ByID      primitive.ObjectID `bson:"byId,omitempty" json:"-"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}
