package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// The in-house till (POS): open checks on the floor.
//
// ⚠️ **An open check is an `order`, not a second kind of document.** Everything
// downstream of an order already exists and already works: the kitchen screen,
// the receipt, the statistics, ABC/XYZ, the financial report, the cash shift,
// discounts, loyalty, the POS bridge to somebody else's till. A second
// collection would have meant teaching every one of them a second kind of sale,
// and the first one anybody forgot would silently under-report the restaurant's
// own revenue — the number the owner trusts most.
//
// What genuinely differs is that a check is **built up over an hour** instead of
// arriving complete, and the codebase already has the timestamp that expresses
// it: `queuedAt` is "when this became the kitchen's problem". An open check has
// none, so the pass never sees it, the chime never rings for it and it counts
// as no revenue — exactly the behaviour the field was written for, reused
// rather than reinvented.
//
// Three states, and none of them is a new `status`:
//
//	Check != nil, Check.ClosedAt == nil   → open on the floor
//	Check.ClosedAt != nil                 → paid and closed
//	Check == nil                          → not a till sale at all
type OrderCheck struct {
	OpenedAt   time.Time          `bson:"openedAt" json:"openedAt"`
	OpenedByID primitive.ObjectID `bson:"openedById,omitempty" json:"openedById,omitempty"`
	// Denormalised so a closed check still reads correctly after the employee
	// leaves and the account is deleted — the same reason CourierName is kept
	// on the order.
	OpenedBy string `bson:"openedBy,omitempty" json:"openedBy,omitempty"`

	// The waiter this check belongs to. Separate from OpenedByID because a
	// cashier routinely opens a check for somebody else's section, and "whose
	// tables are these" is the only question the floor screen exists to answer.
	ServerID   primitive.ObjectID `bson:"serverId,omitempty" json:"serverId,omitempty"`
	ServerName string             `bson:"serverName,omitempty" json:"serverName,omitempty"`

	// ---- Who has this check open right now ----
	//
	// ⚠️ **Two screens editing one table is a bill that loses lines.** The
	// waiter's tablet and the cashier's monoblock both hold their own copy
	// while it is being worked on, and each writes it back whole: a dish added
	// on one and a quantity changed on the other end with whichever saved last,
	// and the other person's work simply gone. Nothing warns anybody, because
	// nothing failed.
	//
	// ⚠️ **It expires, and that is the important half.** A monoblock that loses
	// power mid-service would otherwise hold a table locked until somebody
	// found a database — during service, on the busiest table, with the guest
	// waiting. A lock nobody can lift is worse than the overwrite it prevents,
	// so this one lifts itself after a couple of minutes of nobody touching it.
	//
	// ⚠️ The name is kept beside the id for the same reason `ServerName` is: a
	// refusal has to say *who*, and "somebody else is editing this" sends the
	// person to look for a manager rather than to the colleague two metres away.
	HeldByID primitive.ObjectID `bson:"heldById,omitempty" json:"heldById,omitempty"`
	HeldBy   string             `bson:"heldBy,omitempty" json:"heldBy,omitempty"`
	HeldAt   *time.Time         `bson:"heldAt,omitempty" json:"heldAt,omitempty"`

	// How many people are sitting there. Not decoration: covers-per-table and
	// average-per-guest are the two numbers a dining room is actually run on,
	// and neither can be recovered later from anything else on the order.
	Guests int `bson:"guests,omitempty" json:"guests,omitempty"`

	// When the bill was printed for the table.
	//
	// ⚠️ **The one fact the floor screen cannot work out for itself**, and the
	// third state every till in the world draws: a table that has asked for the
	// bill is neither "eating" nor "gone". It is the table a waiter has to walk
	// back to with a card machine, and until this existed the only way to know
	// was to have been the person who printed it.
	//
	// A timestamp rather than a flag, for the reason `readyAt` is one: "asked
	// twenty minutes ago" and "asked just now" are different situations, and a
	// boolean says the same thing for both.
	PrecheckAt *time.Time `bson:"precheckAt,omitempty" json:"precheckAt,omitempty"`
	// When the guest's receipt was queued for a printer.
	//
	// ⚠️ **A sale prints once.** The receipts are queued from two places — the
	// close for a restaurant with no register, the filing for one with — and a
	// retry after a refused filing runs the second again. Two slips for one
	// meal is a guest asking which of them is real.
	ReceiptAt  *time.Time         `bson:"receiptAt,omitempty" json:"receiptAt,omitempty"`
	ClosedAt   *time.Time         `bson:"closedAt,omitempty" json:"closedAt,omitempty"`
	ClosedByID primitive.ObjectID `bson:"closedById,omitempty" json:"closedById,omitempty"`
	ClosedBy   string             `bson:"closedBy,omitempty" json:"closedBy,omitempty"`

	// Set on a check that was split off another one, pointing at the original.
	// A split is not a new sale and must never read as one: the report that
	// counts checks would otherwise show a busier night than the room had.
	SplitFromID primitive.ObjectID `bson:"splitFromId,omitempty" json:"splitFromId,omitempty"`
}

// CheckRefund is money handed back after a sale was closed.
//
// ⚠️ **A record, not a deletion.** The obvious implementation of "the guest
// wants their money back" is to cancel the sale, and it is wrong twice over:
// the food was cooked and eaten (the kitchen's night, the stock and the
// waiter's work all really happened), and a sale that disappears takes the
// reason with it. What changes is the money, so what is recorded is the money.
type CheckRefund struct {
	At   time.Time          `bson:"at" json:"at"`
	ByID primitive.ObjectID `bson:"byId,omitempty" json:"-"`
	By   string             `bson:"by,omitempty" json:"by,omitempty"`
	// ⚠️ Required. "Refunded 240 000" with no sentence beside it is the line
	// every argument about a shift starts from, and the person who could
	// answer has gone home.
	Reason string `bson:"reason" json:"reason"`
	Amount int    `bson:"amount" json:"amount"`
	// How the sale had been settled, copied because the setting it was read
	// from is about to stop being true of this order.
	Method string `bson:"method,omitempty" json:"method,omitempty"`
}

// IsOpen reports whether this check is still on the floor.
func (c *OrderCheck) IsOpen() bool { return c != nil && c.ClosedAt == nil }

// Till permissions.
//
// ⚠️ **Two permissions, not one, and the line is drawn where the money is.**
// A waiter opens checks, adds dishes and sends them to the kitchen; a cashier
// also takes payment, voids food that has already been cooked and gives
// discounts. Collapsing them would hand every person who can carry a plate the
// ability to close a check as "discount 100%" — which is not a hypothetical
// abuse, it is the one every till in the world is designed to make visible.
//
// ⚠️ The zero value is **false** for both, and unlike CanKitchen that needs no
// migration: nobody could do either of these before the feature existed, so
// there is no behaviour to grandfather. CanKitchen had to be backfilled
// precisely because it *removed* something people already had.
const (
	// May open checks, add and fire lines, and hand a check to the cashier.
	PermWaiter = "waiter"
	// Everything a waiter may do, plus payment, voids and discounts.
	PermCashier = "cashier"
)

// CheckLineVoid is why a cooked dish was taken off a check.
//
// ⚠️ **Only fired lines need one.** Removing a line nobody has cooked is a
// typo being corrected and demanding a reason for it teaches the staff to type
// ".". Removing a line the kitchen has already made is food that was paid for
// by the restaurant, and it is the single event a till exists to record.
type CheckLineVoid struct {
	At   time.Time          `bson:"at" json:"at"`
	ByID primitive.ObjectID `bson:"byId,omitempty" json:"byId,omitempty"`
	By   string             `bson:"by,omitempty" json:"by,omitempty"`
	// Who allowed it, when the person doing it needed permission. Empty when
	// they held it themselves.
	//
	// ⚠️ **Both names, not one.** Recording only the manager loses the fact
	// worth keeping — that somebody who could not do this asked for it — and
	// blames them for every write-off in the building; recording only the
	// waiter loses that it was authorised at all.
	AuthByID primitive.ObjectID `bson:"authById,omitempty" json:"-"`
	AuthBy   string             `bson:"authBy,omitempty" json:"authBy,omitempty"`
	Reason   string             `bson:"reason" json:"reason"`
	// Whether the food was actually made and thrown away, as opposed to the
	// kitchen catching it in time. The waste report is the reason to ask.
	Wasted bool `bson:"wasted,omitempty" json:"wasted,omitempty"`
}

// CheckHoldTTL is how long a hold survives without being renewed.
//
// ⚠️ **Short, because the cost of it being too long is a table nobody can
// serve.** Every edit renews it, so a person actually working on a check never
// meets it; what it measures is how long after somebody walks away their
// colleague has to wait. Two minutes is longer than a pause at the pass and
// shorter than a guest's patience.
const CheckHoldTTL = 2 * time.Minute

// HeldByOther reports whether somebody else has this check open.
//
// ⚠️ An unheld check, one held by this person, and one whose hold has gone
// stale are all the same answer: yes, you may edit it.
func (c *OrderCheck) HeldByOther(by primitive.ObjectID, now time.Time) bool {
	if c == nil || c.HeldByID.IsZero() || c.HeldByID == by || c.HeldAt == nil {
		return false
	}
	return now.Sub(*c.HeldAt) < CheckHoldTTL
}
