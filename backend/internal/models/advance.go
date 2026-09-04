package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Cash handed to somebody to spend on the restaurant's behalf ----
//
// ⚠️ **The one fact in a market run that nothing recorded.** Two million so'm
// goes out of the safe at six in the morning, one and three quarters comes back
// as food, and a quarter comes back as notes — and until now the only part of
// that this system knew about was the food. Whether the buyer still holds
// anything, and how much, was a conversation.
//
// ⚠️ **An advance is not an expense, and this is the distinction the whole
// design rests on.** The money is not spent when it is handed over; it is spent
// when it buys something, and that is the `purchase` document, which the
// financial report already counts. Filing an advance as an outgoing too would
// count the same money twice — once as cash leaving and once as food arriving —
// and the month would read far worse than it was. This ledger answers a
// different question: **who is holding our cash right now.**
//
// ⚠️ **Its own collection rather than a cash entry.** A `cash_entry` belongs to
// an open till shift, and the shift that matters here does not exist: the money
// is handed over at six in the morning, from a safe, before anybody has opened a
// drawer. Where the cash *did* come out of the till, the cashier records that
// withdrawal as they always have — that is a fact about the drawer, and this is
// a fact about a person.
type StaffAdvance struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	// Who is holding it.
	StaffID primitive.ObjectID `bson:"staffId" json:"staffId"`
	// Frozen, for the reason a purchase freezes its supplier's name: a renamed
	// or dismissed employee must not rewrite last month's ledger.
	StaffName string `bson:"staffName,omitempty" json:"staffName,omitempty"`

	// AdvanceOut — handed over. AdvanceBack — returned.
	//
	// ⚠️ **Two kinds rather than a signed amount.** A negative number in a
	// money column is read as a correction by everybody who has ever used a
	// spreadsheet, and this ledger has real corrections in it too. The kind
	// says what happened; the amount stays a plain positive figure.
	Kind   string `bson:"kind" json:"kind"`
	Amount int    `bson:"amount" json:"amount"`

	At time.Time `bson:"at" json:"at"`
	// Who handed it over or took it back. ⚠️ Both halves of the movement have a
	// witness by construction — one person gives and another receives — and the
	// name recorded is the one who was *not* holding the money afterwards.
	ByID primitive.ObjectID `bson:"byId,omitempty" json:"-"`
	By   string             `bson:"by,omitempty" json:"by,omitempty"`
	Note string             `bson:"note,omitempty" json:"note,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

const (
	// AdvanceOut is money handed to somebody to spend.
	AdvanceOut = "out"
	// AdvanceBack is money handed back unspent.
	AdvanceBack = "back"
)

// AdvanceBalance is one person's account, as the panel and their phone read it.
//
// ⚠️ **Three measured figures and one subtraction**, rather than a running
// balance kept anywhere. A stored balance is a second copy of an answer the
// documents already contain, and it drifts the first time a delivery is deleted
// or an advance is corrected — silently, in a number about money.
type AdvanceBalance struct {
	StaffID   string `json:"staffId"`
	StaffName string `json:"staffName"`
	// Handed over, returned, and spent on deliveries this person recorded.
	Issued   int `json:"issued"`
	Returned int `json:"returned"`
	Spent    int `json:"spent"`
	// What should still be in their hands.
	//
	// ⚠️ **It can go below zero, and that is not an error to hide.** A buyer who
	// ran out and paid for the last crate themselves is owed money, and a ledger
	// that clamped at zero would be silent about exactly the debt somebody is
	// waiting to be paid.
	Balance int `json:"balance"`
	// When they were last given anything, so a screen can say how old the
	// number is rather than only what it is.
	LastAt *time.Time `json:"lastAt,omitempty"`
}
