package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- One dish leaving the store because it was sold ----
//
// ⚠️ **This is a change of source, not an extra record, and that distinction is
// the whole design.** Before it, the store's consumption was *derived*: every
// screen re-ran "orders in the period × the tech cards as they read today". The
// arithmetic was correct and had no memory — nothing could say when a kilo left,
// which check took it, or who removed the line afterwards.
//
// Writing documents *beside* that arithmetic would have been the worse of both:
// two answers to one question, drifting apart on the first edit, with nothing to
// say which is right. This codebase has paid for a duplicated rule twice
// already (§ Stop list, § soldDishes). So the documents are not beside it —
// they **replace** it. `consumedInPeriod` reads this collection, and there is no
// second path left to disagree with.
//
// ⚠️ **Therefore the backfill is not optional.** An install that starts with
// this collection empty and a year of orders behind it would read consumption
// as zero: every shelf full, every stop list released, every count reporting a
// surplus the size of a year's cooking. The migration runs at boot and the
// server does not start without it — the same rule the duplicate
// `pos_settings` index follows, for the same reason.
//
// ⚠️ **Written when the line is rung up, not when the food is cooked.** The
// waiter's tap is what this system can observe; it is also the moment a
// reservation should exist, so the last portion of osh cannot be promised to
// two tables. It matches what the derived arithmetic already counted (an open
// check was always in the figures), so switching source changed no number.
type StockMovement struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	// When the food was sold — the order's own time, never the moment the row
	// was written.
	//
	// ⚠️ **A till that was offline sends yesterday's checks today** (§ tillsync).
	// Stamped with the write, a week of reconnected checks would land on one
	// afternoon, and every stocktake either side of the gap would be wrong in
	// opposite directions.
	At time.Time `bson:"at" json:"at"`

	// What sold it.
	OrderID     primitive.ObjectID `bson:"orderId" json:"orderId"`
	OrderNumber string             `bson:"orderNumber,omitempty" json:"orderNumber,omitempty"`
	// Which line of it. ⚠️ Present on a till check and absent everywhere else:
	// a website order is written once and never edited, a check is edited all
	// evening and two guests ordering the same dish are deliberately two lines.
	// Where it is absent the line's position stands in — see lineKey.
	LineID     string             `bson:"lineId,omitempty" json:"lineId,omitempty"`
	LineKey    string             `bson:"lineKey" json:"lineKey"`
	MenuItemID primitive.ObjectID `bson:"menuItemId,omitempty" json:"menuItemId,omitempty"`
	// Frozen, so a renamed dish does not rewrite last month's stockroom.
	DishName string `bson:"dishName,omitempty" json:"dishName,omitempty"`

	// How many portions. Fractional because a half portion takes half the card
	// — the rule `PortionFactor` exists for.
	Qty float64 `bson:"qty" json:"qty"`

	// What **one portion** takes off the shelf, in purchase units.
	//
	// ⚠️ **Per portion rather than in total, so a quantity correction is exact.**
	// A waiter changing three to two must not re-read today's card — that would
	// reprice a sale against a recipe edited since — and scaling a frozen total
	// by 2/3 accumulates rounding on every edit. One portion, frozen once,
	// multiplied on read.
	//
	// ⚠️ **Frozen at all**, for the reason a batch's inputs are: the card is
	// corrected as recipes change, and a dish sold in March took what the March
	// card said. Re-deriving it would rewrite a month already counted and
	// reconciled — and the count is what somebody was asked to explain.
	Lines []StockMovementLine `bson:"lines" json:"lines"`
	// What one portion was worth in so'm, at that day's ingredient prices.
	ValuePer int `bson:"valuePer" json:"valuePer"`

	// ---- Taken back ----
	//
	// ⚠️ **Reversed, never deleted.** A cashier ringing up an expensive dish and
	// removing it again is both an ordinary mistake and a known way to test
	// whether anybody is watching; a row that disappears cannot tell them
	// apart. Reversed rows are excluded from every balance and stay readable.
	ReversedAt     *time.Time `bson:"reversedAt,omitempty" json:"reversedAt,omitempty"`
	ReversedReason string     `bson:"reversedReason,omitempty" json:"reversedReason,omitempty"`

	// The food was cooked and then thrown away.
	//
	// ⚠️ **This is the field that makes "write off when it is rung up" honest.**
	// A line removed before the kitchen was told took nothing off any shelf, and
	// its row is reversed. A line removed *after* — a dish sent back, a mistake
	// discovered at the pass — was made: the ingredients are gone, and putting
	// them back would be a shortfall filed as a correction. So the row stays and
	// says why. The till already asks the question (`CheckLineVoid.Wasted`);
	// this only records the answer.
	Wasted bool   `bson:"wasted,omitempty" json:"wasted,omitempty"`
	Reason string `bson:"reason,omitempty" json:"reason,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// StockMovementLine is one ingredient one portion of the dish takes.
type StockMovementLine struct {
	IngredientID primitive.ObjectID `bson:"ingredientId" json:"ingredientId"`
	// Frozen for the reason a purchase freezes its supplier's name.
	Name string `bson:"name,omitempty" json:"name,omitempty"`
	// In purchase units — kilos, litres, pieces — like everything else the
	// balance arithmetic speaks.
	Qty float64 `bson:"qty" json:"qty"`
}

// Counts reports whether this row is still on the shelf's books.
//
// ⚠️ A wasted row counts. The food left the store; that it was thrown away
// rather than eaten is a fact about the waste report, not about the shelf.
func (m StockMovement) Counts() bool { return m.ReversedAt == nil }
