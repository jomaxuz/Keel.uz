package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Inkassatsiya: the day the cash leaves the building ----
//
// ⚠️ **The law makes this the one cash event with an outside deadline.** The
// rules for cash operations (Правила ведения кассовых операций, ст. 7) say
// every so'm above the limit agreed with the bank must be handed over for
// crediting to the account; only wages may stay, and only for three working
// days (ст. 8). So "how much is in the safe" is not merely an owner's
// curiosity — over a threshold it is a breach with a date on it, and the
// system that knows the number is the one that can say so.
//
// ⚠️ **A collection is a place, not a cost** — the same rule the safe follows.
// Money going to the bank has not been spent; it has moved from a box to an
// account. Counting it as an outgoing would subtract the restaurant's own
// takings from itself, which is exactly the mistake the courier settlement
// line was written to avoid.
//
// ⚠️ **The reconciliation is frozen onto the document, not recomputed.** What
// the shifts counted, what they were expected to hold, what the couriers handed
// in — all of it is written here at the moment of the handover. Recomputing it
// later would produce a different answer every time an old shift is corrected,
// and the one question this document exists to answer ("did the amount that
// left match the money that was there?") would quietly change its mind months
// after everybody signed.
type Collection struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`

	At     time.Time `bson:"at" json:"at"`
	Amount int       `bson:"amount" json:"amount"`

	// Where it went: CollectionToBank or CollectionToSafe.
	//
	// ⚠️ Both are recorded here even though only one of them is inkassatsiya in
	// the legal sense: a restaurant that carries its takings to the office box
	// on Friday and to the bank on Monday has two movements, and a ledger that
	// only knew about the second would show the safe filling itself.
	To string `bson:"to" json:"to"`
	// Who physically took it — the bank's collector, the owner, a manager. An
	// unsigned handover of a month's takings is the record that gets disputed.
	TakenBy string `bson:"takenBy,omitempty" json:"takenBy,omitempty"`
	// The bank's bag or receipt number, when there is one. This is what a
	// dispute with the bank is settled by.
	Bag  string `bson:"bag,omitempty" json:"bag,omitempty"`
	Note string `bson:"note,omitempty" json:"note,omitempty"`

	// ---- What the money should have been, frozen at the handover ----

	// The window this covers: from the previous collection to this one.
	FromAt time.Time `bson:"fromAt" json:"fromAt"`

	// The closed shifts inside the window — the Z reports, in this system's
	// terms. Kept by id so the document can be opened and read back.
	ShiftIDs []primitive.ObjectID `bson:"shiftIds,omitempty" json:"shiftIds,omitempty"`
	Shifts   int                  `bson:"shifts" json:"shifts"`

	// What those shifts counted in the drawer, and what they were expected to
	// hold. Their difference is the drawer variance for the window.
	Counted  int `bson:"counted" json:"counted"`
	Expected int `bson:"expected" json:"expected"`
	Variance int `bson:"variance" json:"variance"`

	// The safe's own balance at the moment of the handover, when the money came
	// out of it.
	SafeBefore int `bson:"safeBefore" json:"safeBefore"`

	// Amount − what the system believed was there.
	//
	// ⚠️ **Stored even when it is zero.** A document that only carried a
	// difference when there was one would leave "checked and correct"
	// indistinguishable from "nobody checked".
	Diff int `bson:"diff" json:"diff"`

	By        string    `bson:"by,omitempty" json:"by,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

const (
	// CollectionToBank is the legal inkassatsiya: cash credited to the account.
	CollectionToBank = "bank"
	// CollectionToSafe is the internal move — the drawer emptied into the
	// office box, on the way to the bank later.
	CollectionToSafe = "safe"
)
