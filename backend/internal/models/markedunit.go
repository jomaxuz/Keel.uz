package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MarkedUnit is one physical bottle this branch received, by its own code.
//
// ⚠️ **A row per object, not a count.** That is the whole point of the state's
// marking: the code identifies *this* bottle, and a quantity would be exactly
// the information it exists to replace. It is also what makes the two failures
// this catches visible — a code that was never received, and a code sold twice.
//
// ⚠️ **Scanning at goods-in is opt-in per branch** (`Branch.MarkingInbound`).
// Every restaurant running today scans only at the till, and a check that
// started refusing codes nobody had ever received would refuse every sale in the
// product on the day it shipped. Off means the till behaves exactly as it does
// now.
//
// ⚠️ **Where the refusal lands is the whole value.** Without this, a bottle
// bought outside the system, or a code already withdrawn, is discovered at the
// counter with a customer waiting — and the cashier's only remedy is to scan
// again, which cannot help. With it, the same fact is discovered in the store
// room by the person unpacking the box, who can put it aside and ring the
// supplier.
type MarkedUnit struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId" json:"branchId"`

	// The code as the scanner read it, normalized.
	//
	// ⚠️ **Unique across the whole collection, not per branch.** A code names a
	// physical object; the same one arriving twice is a duplicate scan or a
	// counterfeit, and both are things somebody has to look at rather than
	// facts to store twice. Two branches of one chain cannot legitimately hold
	// the same bottle either.
	Code string `bson:"code" json:"code"`

	// What was scanned into, as the person receiving said.
	//
	// ⚠️ **Their answer, not the code's.** The GTIN inside a DataMatrix names a
	// product in the national catalogue, which is not the same list as this
	// restaurant's menu — matching them would be a mapping nobody has filled in,
	// and a receiving screen that refused everything until they did is a screen
	// nobody uses.
	MenuItemID primitive.ObjectID `bson:"menuItemId,omitempty" json:"menuItemId,omitempty"`
	PurchaseID primitive.ObjectID `bson:"purchaseId,omitempty" json:"purchaseId,omitempty"`

	ReceivedAt time.Time `bson:"receivedAt" json:"receivedAt"`
	ReceivedBy string    `bson:"receivedBy,omitempty" json:"receivedBy,omitempty"`

	// When it left, and on which receipt.
	//
	// ⚠️ **Marked rather than deleted.** "This bottle was sold on that receipt"
	// is the answer to the only question anybody asks afterwards, and a row that
	// disappeared at the till would leave a re-scanned code looking exactly like
	// one that never arrived.
	SoldAt  *time.Time         `bson:"soldAt,omitempty" json:"soldAt,omitempty"`
	OrderID primitive.ObjectID `bson:"orderId,omitempty" json:"orderId,omitempty"`
}
