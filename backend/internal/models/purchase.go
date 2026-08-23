package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Deliveries ----
//
// ⚠️ **Where the price actually comes from.** Until now an ingredient's price
// was typed: somebody read the invoice and retyped a number, which is the step
// where prices go stale — nobody retypes forty of them after every delivery.
// Recording the delivery is work the restaurant is doing anyway, and the price
// falls out of it.
//
// ⚠️ **A delivery is dated by the invoice, not by when it was entered**, and
// that is the one thing that makes it different from editing a price. An edit
// cannot tell "we typed it wrong" from "beef went up" and so counts from today;
// a delivery is a measurement with its own date, and saying meat cost this much
// last Tuesday is a fact, not a rewriting of last Tuesday.
//
// ⚠️ **Still not stock.** Nothing here subtracts what the kitchen used, so no
// screen may claim a remaining quantity: a restaurant that believes a stock
// figure and finds it wrong stops believing the panel entirely. This records
// what came in and what it cost, which is exactly what an invoice is.

// PurchaseLine is one ingredient on one delivery.
type PurchaseLine struct {
	IngredientID primitive.ObjectID `bson:"ingredientId" json:"ingredientId"`
	// In purchase units — kilos, litres, pieces — because that is how the
	// invoice is written and how the delivery was counted at the door.
	Qty float64 `bson:"qty" json:"qty"`
	// What one purchase unit cost on this delivery.
	//
	// ⚠️ Per unit rather than a line total: an invoice usually shows both, and
	// the per-unit figure is the one that becomes the ingredient's price. A
	// line total would have to be divided by a quantity that may be rounded,
	// and the division would land in the price of everything the ingredient
	// goes into.
	Price int `bson:"price" json:"price"`
}

// Sum is what the line cost.
func (l PurchaseLine) Sum() int { return int(float64(l.Price) * l.Qty) }

// Purchase is one delivery, as the invoice reads.
type Purchase struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	// The invoice's own date — see the package note.
	At time.Time `bson:"at" json:"at"`
	// Free text: whoever it came from. Not a table of suppliers, because a
	// restaurant with three regular ones and a market run does not have a
	// supplier list, and asking it to keep one is asking it to stop recording
	// deliveries.
	// ⚠️ **Both the id and the name, and the name is a copy.** The id is what
	// totals and debts are grouped by; the text is what the invoice said the
	// day it was entered, kept so a renamed or deactivated supplier does not
	// rewrite last year's deliveries — the same rule `tableNumber` and an order
	// line's dish name follow.
	//
	// ⚠️ The id stays **optional**: a market run has no supplier, and requiring
	// one would stop deliveries being recorded at all, which costs far more
	// than an ungrouped row.
	SupplierID primitive.ObjectID `bson:"supplierId,omitempty" json:"supplierId,omitempty"`
	Supplier   string             `bson:"supplier,omitempty" json:"supplier,omitempty"`
	// ---- Whether it has been paid for ----
	//
	// ⚠️ **The mirror of a guest's slate**, and the reason it is here rather
	// than in a separate ledger: the invoice already carries the amount, the
	// date and who it is owed to. A second document would be a second place for
	// the same fact to be wrong in.
	//
	// ⚠️ **Zero means unpaid**, which is the opposite of this codebase's usual
	// "empty value keeps today's behaviour" rule and is deliberate: every
	// delivery entered before this existed was, as far as anything recorded
	// knows, settled — but reading them all as paid would hide a real debt on
	// the day this shipped, and reading them as unpaid at worst shows an owner
	// a list to tick through once. A wrong "you owe nothing" is the expensive
	// direction. The migration settles the old ones for exactly that reason.
	Paid   bool           `bson:"paid,omitempty" json:"paid"`
	PaidAt *time.Time     `bson:"paidAt,omitempty" json:"paidAt,omitempty"`
	Note   string         `bson:"note,omitempty" json:"note,omitempty"`
	Lines  []PurchaseLine `bson:"lines" json:"lines"`
	// What the delivery cost in total, frozen as entered.
	//
	// ⚠️ Stored rather than recomputed: the lines' prices are what fed the
	// ingredient history, and the invoice's own total is what the restaurant
	// paid. When they disagree — a rounding, a delivery charge, a discount at
	// the door — the invoice is right about the money and the lines are right
	// about the prices, and neither should be quietly overwritten by the other.
	Total       int                `bson:"total" json:"total"`
	CreatedBy   string             `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	CreatedByID primitive.ObjectID `bson:"createdById,omitempty" json:"-"`
}
