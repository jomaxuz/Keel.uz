package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- What a courier was actually paid ----
//
// ⚠️ **The system knew what every courier had earned and never knew what any of
// them had been given.** `courierEarning` works out a delivery's payout from the
// rule on the courier, so the arithmetic existed — but the money changed hands
// in the corridor and nothing wrote it down. Two consequences, both quiet: the
// financial report left couriers out of its costs entirely (the note in
// `finreport.go` said so, honestly, which is not the same as being right), and
// a courier asking "have I been paid for last week?" had only somebody's memory
// to go on.
//
// ⚠️ **Its own collection rather than a `staff_payment` with a courier in it.**
// The obvious saving is to reuse the wage document, and it is a trap of the kind
// this codebase has been bitten by before: two id fields where one is always
// zero, and a zero ObjectID reaches the browser as "000…0", which is *truthy*.
// Every screen showing "paid to <someone>" would then have to remember which
// half of the document is real. A courier's pay is also computed differently —
// per delivery, from a rule — so the two documents answer to different
// arithmetic and only look alike.
//
// ⚠️ **And it gets its own line in the financial report**, not a share of
// "paid to staff": couriers are the cost that scales with delivery volume, and
// an owner deciding whether delivery pays for itself needs that number separate
// from the kitchen's wages.
type CourierPayment struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CourierID primitive.ObjectID `bson:"courierId" json:"courierId"`
	BranchID  primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	Amount    int                `bson:"amount" json:"amount"`

	// The window this settles, local "YYYY-MM-DD", so a payment always says
	// what it was for — the same rule a wage follows.
	From string `bson:"from" json:"from"`
	To   string `bson:"to" json:"to"`

	// Who handed it over. An unsigned cash payment is the record that gets
	// disputed later.
	PaidBy string `bson:"paidBy,omitempty" json:"paidBy,omitempty"`
	Note   string `bson:"note,omitempty" json:"note,omitempty"`

	At time.Time `bson:"at" json:"at"`
}
