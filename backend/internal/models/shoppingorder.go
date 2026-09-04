package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- What the restaurant asked somebody to buy ----
//
// ⚠️ **The half the buying had no record of.** A delivery says what came back;
// nothing said what was *asked for*. So "we ran out of beef again" had no answer
// — nobody could tell whether it was never on the list, was on the list and not
// bought, or was bought and eaten faster than expected. Those are three
// different problems with three different fixes, and the restaurant could only
// see the symptom.
//
// ⚠️ **A request, not a delivery, and the two must not merge.** This document is
// somebody's intention; a `purchase` is money that left a hand. They meet at the
// end — the ticked lines become one delivery — and that is the only place they
// touch. Merged into one document, an order nobody shopped would sit on the
// shelf as food that was never bought.
type ShoppingOrder struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`

	// Which day the shopping is for, local, "YYYY-MM-DD".
	//
	// ⚠️ **A string rather than a time, and this codebase has paid for the
	// difference twice.** The driver hands every date back in UTC, so a day
	// written at nine in the evening comes back as the day before — and "which
	// day is this shopping for" is exactly the question that would then be
	// answered wrongly, by one day, silently. Same rule as `branch.LimitDate`
	// and the billing period's anchor.
	ForDate string `bson:"forDate" json:"forDate"`

	// StatusSent — waiting to be shopped. StatusBought — the trip is finished.
	Status string         `bson:"status" json:"status"`
	Lines  []ShoppingLine `bson:"lines" json:"lines"`
	Note   string         `bson:"note,omitempty" json:"note,omitempty"`

	// Who asked. ⚠️ Frozen, like every other name this product stores: a
	// manager who leaves must not rewrite last month's orders.
	CreatedBy   string             `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	CreatedByID primitive.ObjectID `bson:"createdById,omitempty" json:"-"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`

	// The delivery the finished trip became.
	//
	// ⚠️ **One direction only: the order points at the delivery.** The delivery
	// knows nothing about the order, so a corrected or deleted invoice cannot
	// reach back and rewrite what somebody was asked to buy — which is the
	// record this document exists to keep.
	PurchaseID primitive.ObjectID `bson:"purchaseId,omitempty" json:"purchaseId,omitempty"`
	DoneAt     *time.Time         `bson:"doneAt,omitempty" json:"doneAt,omitempty"`
	UpdatedAt  time.Time          `bson:"updatedAt" json:"updatedAt"`
}

const (
	// ShoppingSent is written and waiting to be shopped.
	ShoppingSent = "sent"
	// ShoppingBought is a finished trip: what came back is a delivery.
	ShoppingBought = "done"
)

// ShoppingLine is one thing to buy, and what came back of it.
type ShoppingLine struct {
	// ⚠️ **Its own id, the way a till check's line has one.** The list is
	// written on one device and ticked on another, minutes or hours apart, and
	// position cannot name a line across that gap: an item removed while the
	// buyer is at the market would shift every tick after it onto the wrong
	// row.
	ID string `bson:"id" json:"id"`

	// Empty when the person writing the list typed a name the catalogue does
	// not have. ⚠️ Allowed on purpose — a list somebody cannot finish writing
	// is a list they write on paper instead, and then nothing here sees it.
	IngredientID primitive.ObjectID `bson:"ingredientId,omitempty" json:"ingredientId,omitempty"`
	// Frozen at the moment it was asked for.
	Name string `bson:"name" json:"name"`
	// The catalogue's own unit — kilos, litres, pieces.
	//
	// ⚠️ **Never chosen by the buyer.** A market sells mint in bunches and
	// flour in sacks, and somebody typing "5" for five bunches into a field
	// measured in kilos puts five kilos on the shelf instead of a quarter of
	// one. The figure is then twenty times too high, the stop list never fires,
	// and the gap turns up a month later at a count as an unexplained
	// shortfall. So the unit is the catalogue's, shown and not editable, and
	// the screen questions a quantity far from what was asked for.
	Unit string  `bson:"unit,omitempty" json:"unit,omitempty"`
	Qty  float64 `bson:"qty" json:"qty"`
	Note string  `bson:"note,omitempty" json:"note,omitempty"`

	// ---- What came back ----
	//
	// ⚠️ **Kept beside what was asked rather than replacing it.** "Asked for
	// ten, brought six" is the sentence this whole document exists to make
	// possible; overwriting the request with the result would leave the same
	// silence the buying had before.
	GotQty float64    `bson:"gotQty,omitempty" json:"gotQty,omitempty"`
	Price  int        `bson:"price,omitempty" json:"price,omitempty"`
	GotAt  *time.Time `bson:"gotAt,omitempty" json:"gotAt,omitempty"`
	// The market did not have it. ⚠️ Its own answer, not a quantity of zero: a
	// line nobody touched and a line somebody looked for and could not find are
	// different facts, and only the second one is worth ringing a supplier
	// about.
	Missing bool `bson:"missing,omitempty" json:"missing,omitempty"`
}

// Got reports whether this line came back with something.
func (l ShoppingLine) Got() bool { return l.GotAt != nil && !l.Missing && l.GotQty > 0 }
