package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Moving stock between stores ----
//
// ⚠️ **The one movement the chain was missing.** Food comes in (a delivery),
// leaves as sales (the cards), and leaves as waste (a write-off). It also
// simply *moves*: the barman takes a case of tonic out of the cellar. Until
// this existed the only way to record that was a write-off on one side and a
// delivery on the other, and both of those lie. The write-off adds a reason to
// the waste report for food nobody wasted — on the report whose entire value is
// that its reasons are actionable — and the delivery writes a purchase price
// into the price history, which then costs every dish the ingredient goes into.
//
// ⚠️ **From one ingredient to another, not one warehouse to another**, and that
// follows from the model rather than being a shortcut. A store belongs to the
// ingredient (see warehouse.go: "one ingredient, one warehouse"), so a
// restaurant keeping tonic in the cellar *and* behind the bar already has two
// ingredients — two shelves, two counts, two people responsible. A transfer is
// therefore the quantity leaving one of them and arriving at the other, which
// is also the only shape the balance arithmetic can express: each ingredient
// has exactly one home to be counted in.
//
// ⚠️ **Value is carried, not created.** A transfer costs the source's price on
// that day, and that figure is *not* a cost to the business — nothing was
// bought and nothing was lost. It is recorded so the movement report can say
// what moved in money as well as in kilos, and it is deliberately absent from
// the financial report's expenses.
type StockTransfer struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	At       time.Time          `bson:"at" json:"at"`
	// Which shelf it left and which it arrived at, named by the ingredient
	// that lives there.
	FromID primitive.ObjectID `bson:"fromId" json:"fromId"`
	ToID   primitive.ObjectID `bson:"toId" json:"toId"`
	// In purchase units, and the same unit on both sides — see transfers.go for
	// why a kilo cannot be moved into a litre.
	Qty float64 `bson:"qty" json:"qty"`
	// Free text. Not required, unlike a write-off's reason: nothing has gone
	// missing here, so there is nothing anybody has to account for.
	Note string `bson:"note,omitempty" json:"note,omitempty"`
	// What moved, at the source's price on that day. Frozen for the same reason
	// a write-off's value is: it must not shift when the next delivery arrives.
	Value     int                `bson:"value" json:"value"`
	By        string             `bson:"by,omitempty" json:"by,omitempty"`
	ByID      primitive.ObjectID `bson:"byId,omitempty" json:"-"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}
