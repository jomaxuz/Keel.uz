package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- What the restaurant spends that nothing else records ----
//
// ⚠️ **The gap that made the financial report optimistic.** Deliveries are
// recorded, wages are recorded, and everything else a restaurant pays for was
// not: rent, electricity, gas, water, tax, repairs, the couriers' own pay. So
// "in − out" read better than the month had been — by whatever the building
// costs — and it read that way every month, consistently, which is exactly how
// a wrong number survives.
//
// ⚠️ **Only what has no document of its own.** A delivery is a `purchase` and a
// wage is a `staff_payment`; both already have their own line in the report.
// Writing either here as well would count it twice, and a double-counted cost
// looks precisely like a real one. The categories the screen offers are chosen
// to make that boundary obvious rather than to be exhaustive.
//
// ⚠️ **An expense and a cash movement are different facts.** Paying rent from
// the safe makes the restaurant poorer *and* moves money out of a box; paying it
// by transfer only does the first. So the safe's side is asked for, written as
// its own linked row, and the two never derive each other — see models/safe.go.
type Expense struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`

	// The day the money went, local, "YYYY-MM-DD".
	//
	// ⚠️ **Its own date, not the moment it was typed.** Rent paid on the first
	// and entered on the fourth belongs to the first — the same rule a delivery
	// note follows, and for the same reason: the report is about the month the
	// money left, not the month somebody had time.
	At time.Time `bson:"at" json:"at"`

	// What it was for. Free text with suggestions rather than a fixed list:
	// every restaurant pays for something the next one does not, and a closed
	// list sends all of it to "boshqa".
	Category string `bson:"category" json:"category"`
	Amount   int    `bson:"amount" json:"amount"`
	Note     string `bson:"note,omitempty" json:"note,omitempty"`

	// How it was paid: PaidCash, PaidTransfer, PaidCard.
	//
	// ⚠️ Recorded because it decides whether a box got lighter. It is not an
	// accounting nicety — it is the difference between a safe balance that
	// matches the notes and one that does not.
	Method string `bson:"method,omitempty" json:"method,omitempty"`

	CreatedBy   string             `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	CreatedByID primitive.ObjectID `bson:"createdById,omitempty" json:"-"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
}

const (
	// PaidCash — notes changed hands.
	PaidCash = "cash"
	// PaidTransfer — a bank transfer.
	PaidTransfer = "transfer"
	// PaidCard — a card payment.
	PaidCard = "card"
)
