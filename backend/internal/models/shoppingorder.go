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

	// Where this half of the request is answered from — `market` or `store`.
	//
	// ⚠️ **One request becomes one document per place, not one document with two
	// kinds of line in it.** The barman writes a single list; cola comes off a
	// shelf in ten minutes and lemons come back from a market at nine, and the
	// two people never meet. A shared document would spend the morning in a
	// state neither of them can read — "half sent" is not a word either job has
	// — and every screen would have to say which half it meant. Two documents,
	// one `GroupID`, and each side sees a list that is entirely theirs.
	Source string `bson:"source" json:"source"`

	// The request both halves were written in, in one keystroke by one person.
	//
	// ⚠️ **Kept even when there is only one half**, so the panel never has to
	// ask whether a lone document is a whole request or the survivor of a split
	// one — the answer to "what did the barman ask for at six" is a group id,
	// always.
	GroupID primitive.ObjectID `bson:"groupId,omitempty" json:"groupId,omitempty"`

	// Which branch answers it. The same branch for a market run and for a
	// restaurant whose store room is its own; the central one for a chain that
	// has a branch holding everybody's stock (`branch.supplyBranchId`).
	//
	// ⚠️ **Stored rather than resolved at read time.** The setting can be
	// changed on a Wednesday, and a request written on Tuesday must not move to
	// a different storekeeper's phone half-finished.
	SupplyBranchID primitive.ObjectID `bson:"supplyBranchId,omitempty" json:"supplyBranchId,omitempty"`

	// ShoppingSent — waiting. ShoppingShipped — bought or picked, on its way.
	// ShoppingDone — counted and signed for at the far end.
	//
	// ⚠️ **Three states rather than two, and the third one is the whole point of
	// this feature.** "Bought" said the money had been spent; it said nothing
	// about whether the goods reached the person who asked. That gap is where
	// things disappear, and it is the one part of the morning nobody had a
	// record of.
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

	// The van the store half became, when the goods came from another branch.
	//
	// ⚠️ **Only then.** A restaurant whose store room is down the corridor moves
	// nothing between shelves when the barman is handed a case of cola: the
	// ingredient has exactly one home (models/warehouse.go), and the sale writes
	// it off when the drink is rung up. Recording an issue as stock leaving
	// would subtract the same bottle twice — once here and once at the till —
	// and the count that found the gap would blame the person holding the
	// clipboard. So this half is a record of an errand, not arithmetic, unless
	// two branches are genuinely involved.
	DispatchID primitive.ObjectID `bson:"dispatchId,omitempty" json:"dispatchId,omitempty"`

	// Who picked or bought it, and when they said it was on its way.
	ShippedAt *time.Time `bson:"shippedAt,omitempty" json:"shippedAt,omitempty"`
	ShippedBy string     `bson:"shippedBy,omitempty" json:"shippedBy,omitempty"`

	// Who counted it at the far end and signed for it.
	//
	// ⚠️ **A different person from `ShippedBy` is the point, not a rule the code
	// enforces.** A restaurant where one person does both on a quiet Sunday must
	// still be able to close the morning; what matters is that the two names are
	// written down separately, because "who said it arrived" is the question the
	// record exists to answer and a single name cannot answer it.
	AcceptedAt *time.Time `bson:"acceptedAt,omitempty" json:"acceptedAt,omitempty"`
	AcceptedBy string     `bson:"acceptedBy,omitempty" json:"acceptedBy,omitempty"`

	DoneAt    *time.Time `bson:"doneAt,omitempty" json:"doneAt,omitempty"`
	UpdatedAt time.Time  `bson:"updatedAt" json:"updatedAt"`
}

// Open reports whether this list is still waiting for the person answering it.
func (o ShoppingOrder) Open() bool { return o.Status == ShoppingSent }

// Waiting reports whether somebody has to go and count what arrived.
func (o ShoppingOrder) Waiting() bool { return o.Status == ShoppingShipped }

// FromStore reports which half of the morning this is.
func (o ShoppingOrder) FromStore() bool { return o.Source == SourceStore }

const (
	// ShoppingSent is written and waiting to be shopped or picked.
	ShoppingSent = "sent"
	// ShoppingShipped is bought or taken off the shelf, and on its way to the
	// person who asked. ⚠️ **Nothing has reached a shelf yet**: a market run is
	// only a delivery once somebody at the restaurant has counted it.
	ShoppingShipped = "shipped"
	// ShoppingDone is counted and signed for at the far end.
	//
	// ⚠️ **Still spelled `done`**, which is what the old finished-trip state was
	// called: every list written before this feature existed is terminal and
	// reads as terminal, and renaming it would have left a stored word nothing
	// answers to. The two meanings agree on the only thing any screen asks —
	// this list is finished.
	ShoppingDone = "done"
	// ShoppingBought is the old name for the terminal state, kept so existing
	// callers and stored documents mean the same thing.
	ShoppingBought = ShoppingDone
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
	// What the person who asked for it counted when it arrived.
	//
	// ⚠️ **Absent until somebody counts, and absent is not zero** — the same
	// distinction `DispatchLine.Got` is built on, for the same reason: a line
	// nobody has checked and a line that arrived empty are different facts, and
	// a screen that cannot tell them apart accuses somebody.
	//
	// ⚠️ **It is what reaches the shelf**, not `GotQty`. The buyer writes down
	// what he believes he handed over; the shelf gains what the restaurant
	// counted. Where the two differ, the difference is the record — it is
	// exactly the thing that had nowhere to be written before.
	TookQty *float64   `bson:"tookQty,omitempty" json:"tookQty,omitempty"`
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

// Took is what the person who asked for it counted: their own figure where they
// gave one, and otherwise what they were told was coming.
//
// ⚠️ **Defaulting to `GotQty` rather than to zero.** Somebody who accepts a
// list without touching a row is saying "this is right", which is the ordinary
// case and the one the screen must not punish; a default of zero would empty
// every line nobody retyped.
func (l ShoppingLine) Took() float64 {
	if l.TookQty != nil {
		return *l.TookQty
	}
	return l.GotQty
}

// Short is what was said to be on its way and did not arrive.
func (l ShoppingLine) Short() float64 {
	if l.TookQty == nil {
		return 0
	}
	return l.GotQty - *l.TookQty
}
