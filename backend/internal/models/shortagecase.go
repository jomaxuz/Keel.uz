package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- A shortfall somebody still has to answer for ----
//
// ⚠️ **The case itself is not stored, and that is the whole design.** A count
// already records what was expected, what was found, what the difference was
// worth and who counted — four frozen facts. Storing a case beside them would
// be a fifth copy of the same event, and the copy is the one that goes stale:
// the shortfall list would keep showing a row after the count behind it was
// superseded, exactly as `provisionStatus` kept showing a customer as healthy
// while their site was down (CLAUDE.md). So the queue is **derived from the
// counts every time it is opened**, and the only thing written down here is the
// part no arithmetic can produce — a person's answer.
//
// ⚠️ **A verdict, never a suspicion.** Nothing in this document names who is
// thought to have taken anything. The restaurant's own alert rules already draw
// that line ("bu ayb emas, savol" — models/alert.go), and a shortage is the
// weakest possible evidence about a person: it covers every shift between two
// counts, which is usually a month and everybody who worked it. What is
// recorded is what the *shortfall* turned out to be, not who it was.
//
// ⚠️ **Once, like the count's own explanation.** A verdict that can be
// rewritten next week is not a verdict, and the one most likely to be rewritten
// is the one that turned out to matter. `(stocktakeId, ingredientId)` is unique
// and the write is an insert — Mongo decides, once, and the second manager
// looking at the same screen is told rather than silently overwriting the
// first.

// ShortageVerdict is what a shortfall turned out to be.
//
// ⚠️ **The list does not change with the business type, and the panel's offer
// does.** Which of these a chemist is shown is a question about words and
// relevance (a shop composes nothing, so `card` is meaningless there) — but a
// stored verdict has to mean the same thing on every install, and a kind that
// existed only for some of them would make a month's tally uncountable across
// a chain that runs a kitchen and a shop. The same line businesstype.go draws
// around the stock module: the arithmetic is one, the wording is not.
//
// ⚠️ **Six answers and no "other", because the list is the product.** A free
// sentence alone would make the queue a pile of prose nobody can count; the
// kind is what lets a month of them say "half of our shortfalls are deliveries
// nobody entered", which is a fixable sentence about a process. The sentence is
// still required — the kind says what class it was, and only the words say
// which delivery.
type ShortageVerdict string

const (
	// Counted wrong, and recounting agreed with the books.
	VerdictMiscount ShortageVerdict = "miscount"
	// Spoiled, spilled or thrown away without a write-off document.
	VerdictWaste ShortageVerdict = "waste"
	// The card takes more than the kitchen does, so the books expected too
	// little to be left. ⚠️ The one verdict that says the *arithmetic* is
	// wrong rather than the shelf — and the fix is on another screen.
	VerdictCard ShortageVerdict = "card"
	// A delivery, a transfer or a batch that was never entered. The shelf was
	// right all along and the books were short of a document.
	VerdictPaperwork ShortageVerdict = "paperwork"
	// Rung up as something else: two similar packets, one barcode, one tile —
	// so this row is short and its twin is over.
	//
	// ⚠️ **The verdict a shop cannot do without, and the reason the list is
	// six rather than five.** A grocery offered only miscount, waste,
	// paperwork and lost files every mis-scan under `lost` — and a month of
	// those reads as "half our shortfalls are unexplained" about a shop that
	// has a barcode problem and no theft at all. It is offered to a kitchen
	// too: a waiter taps the wrong tile for the same reason and with the same
	// result.
	VerdictSwap ShortageVerdict = "swap"
	// Gone, with no explanation found. ⚠️ Deliberately not called theft: the
	// honest content of this verdict is that somebody looked and could not
	// account for it.
	VerdictLost ShortageVerdict = "lost"
)

// ValidVerdict says whether this is one of the six.
func ValidVerdict(v ShortageVerdict) bool {
	switch v {
	case VerdictMiscount, VerdictWaste, VerdictCard, VerdictPaperwork,
		VerdictSwap, VerdictLost:
		return true
	}
	return false
}

// ShortageCase is one closed shortfall: the answer to one line of one count.
type ShortageCase struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	// Which line of which count this answers. ⚠️ The pair is the identity —
	// there is no case id anywhere else, because there is no case document
	// until somebody answers one.
	StocktakeID  primitive.ObjectID `bson:"stocktakeId" json:"stocktakeId"`
	IngredientID primitive.ObjectID `bson:"ingredientId" json:"ingredientId"`

	Verdict ShortageVerdict `bson:"verdict" json:"verdict"`
	Note    string          `bson:"note" json:"note"`

	// Who answered, and when. ⚠️ The panel account, not the counter: the
	// person who explains a shortfall is a manager reading the queue, and
	// recording the counter here would put somebody's name against a
	// conclusion they never drew.
	By   string             `bson:"by,omitempty" json:"by,omitempty"`
	ByID primitive.ObjectID `bson:"byId,omitempty" json:"-"`
	At   time.Time          `bson:"at" json:"at"`
}
