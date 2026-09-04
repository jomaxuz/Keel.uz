package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- The safe: where the restaurant's cash actually is ----
//
// ⚠️ **A place, not a profit and loss.** The financial report answers "did we
// make money"; this answers "where is it". They are different questions and
// mixing them is how a report starts subtracting the same money twice — cash
// moved from the drawer to the safe is not an expense, and money handed to a
// buyer is not spent until it buys something. Nothing here reaches the
// financial report, deliberately.
//
// ⚠️ **Every movement is one document, and nothing is derived.** The tempting
// design is to compute the safe from the records that already exist — the
// advances, the till collections, the salary payments — but each of those is
// only *sometimes* about the safe: a float can come from an owner's pocket, a
// wage can be transferred to a card, a drawer can be emptied into a bag that
// goes to the bank. A balance derived from documents that do not say where the
// money went would be a confident number about a place nobody checked.
//
// So the other screens *offer* to write one of these when the money plainly
// came from or went into the safe, and what they write is a single row with a
// link back. One movement, one document, and the link makes writing it twice
// impossible.
type SafeEntry struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`

	// SafeIn — money went in. SafeOut — money came out.
	//
	// ⚠️ **Two kinds rather than a signed amount**, the same choice the petty
	// cash ledger makes: a negative number in a money column is read as a
	// correction by everybody who has ever used a spreadsheet, and this ledger
	// has real corrections in it too.
	Kind string `bson:"kind" json:"kind"`
	// What it was for. Free text, like a till entry's: every restaurant spends
	// money on something the next one does not, and a fixed list sends all of
	// it to "boshqa".
	Category string `bson:"category,omitempty" json:"category,omitempty"`
	Amount   int    `bson:"amount" json:"amount"`

	At   time.Time          `bson:"at" json:"at"`
	ByID primitive.ObjectID `bson:"byId,omitempty" json:"-"`
	By   string             `bson:"by,omitempty" json:"by,omitempty"`
	Note string             `bson:"note,omitempty" json:"note,omitempty"`

	// What caused it, where something did.
	//
	// ⚠️ **This pair is what makes the automatic entries safe.** They are
	// unique together, so a screen that retries — or an owner who taps twice —
	// cannot put the same hand-over into the safe's ledger a second time. A
	// balance is exactly the kind of number where a duplicate is invisible:
	// it is plausible, it is wrong, and nothing downstream can tell.
	RefKind string             `bson:"refKind,omitempty" json:"refKind,omitempty"`
	RefID   primitive.ObjectID `bson:"refId,omitempty" json:"refId,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

const (
	// SafeIn is money going into the safe.
	SafeIn = "in"
	// SafeOut is money coming out of it.
	SafeOut = "out"

	// SafeRefAdvance links an entry to the petty cash it paid for.
	SafeRefAdvance = "advance"
	// SafeRefSalary links one to a wage that was paid in cash.
	SafeRefSalary = "salary"
	// SafeRefShift links one to a till collection.
	SafeRefShift = "shift"
)

// SafeBalance is what is in the safe and how it got there.
//
// ⚠️ **Summed from the ledger on every read, never stored.** A kept running
// total is a second copy of an answer the rows already contain, and it drifts
// the first time one is corrected — silently, in the one figure somebody counts
// the notes against.
type SafeBalance struct {
	In      int `json:"in"`
	Out     int `json:"out"`
	Balance int `json:"balance"`
	// When the last movement was, so a screen can say how old the figure is
	// rather than only what it is.
	LastAt *time.Time `json:"lastAt,omitempty"`
}
