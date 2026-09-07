package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- One branch's store sending goods to another's ----
//
// ⚠️ **The movement a chain with a central store has all day and this system
// could not record.** Everything that existed moves stock *inside* one branch: a
// delivery arrives at it, a sale leaves it, a batch is made in it, and a
// transfer carries a case from its cellar to its bar (models/transfer.go — from
// one ingredient to another, both of them this branch's). The central kitchen
// that buys the meat, marinates it on Monday and sends it to four branches had
// no document at all, and the two ways of faking it both lie: a write-off puts
// food nobody wasted on the waste report, and a delivery at the far end writes
// a purchase price into the price history and costs every dish it goes into.
//
// ⚠️ **No warehouse on the document, and that is the model rather than an
// omission.** Where a thing is kept is a fact about the branch and the
// ingredient (`ingredient_placement`), so a dispatch routes itself line by line
// at both ends exactly as a delivery does — it leaves the shelf it lives on at
// the sender and arrives on the shelf it lives on at the receiver. Asking the
// storekeeper to name two rooms per line is a question that gets answered
// wrongly under pressure, and the wrong answer is invisible until a count.
//
// ⚠️ **What is sent and what arrives are two facts, because the van is between
// them.** The paper this replaces carries three signatures — the storekeeper,
// the driver, the branch foreman — for exactly that reason. So the sender's
// shelf loses what was loaded, the receiver's shelf gains what was counted off
// the van, and any difference belongs to neither: it is a finding about the
// journey, which is the only place it can honestly be filed. Nothing is
// "corrected" silently at either end.
type DispatchLine struct {
	IngredientID primitive.ObjectID `bson:"ingredientId" json:"ingredientId"`
	// ⚠️ **Frozen, like every other name this product prints.** The slip is
	// signed by three people and kept in a folder; a renamed ingredient must not
	// rewrite a piece of paper somebody signed in March.
	Name string `bson:"name" json:"name"`
	Unit string `bson:"unit" json:"unit"`
	// What was loaded, in purchase units — kilos, litres, pieces, the way the
	// storekeeper counts it onto the van.
	Qty float64 `bson:"qty" json:"qty"`
	// What the receiving branch counted off it.
	//
	// ⚠️ **Absent until somebody accepts, and absent is not zero**: an
	// unaccepted dispatch is one nobody has checked yet, not one that arrived
	// empty. `Arrived` answers for both.
	Got *float64 `bson:"got,omitempty" json:"got,omitempty"`
}

// Arrived is what the receiving shelf gains: what was counted off the van, or
// what was loaded while nobody has said otherwise.
func (l DispatchLine) Arrived() float64 {
	if l.Got != nil {
		return *l.Got
	}
	return l.Qty
}

// Lost is what left one shelf and reached no other.
func (l DispatchLine) Lost() float64 {
	if l.Got == nil {
		return 0
	}
	return l.Qty - *l.Got
}

// Dispatch is one van load: one store's goods, on their way to another's.
type Dispatch struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	// ⚠️ **Both ends share a catalogue, and the brand is what says so.** An
	// ingredient id means "the beef in this brand's catalogue"; sending it to a
	// branch of another brand would put a line on a shelf whose counts,
	// recipes and prices belong to a different list — and every screen at the
	// far end would show a name it has no row for.
	BrandID      primitive.ObjectID `bson:"brandId,omitempty" json:"brandId,omitempty"`
	FromBranchID primitive.ObjectID `bson:"fromBranchId" json:"fromBranchId"`
	ToBranchID   primitive.ObjectID `bson:"toBranchId" json:"toBranchId"`

	// What the paper calls itself: the day and that day's number, so two vans
	// to the same branch can be told apart by somebody holding both slips.
	Number string    `bson:"number" json:"number"`
	At     time.Time `bson:"at" json:"at"`

	Lines []DispatchLine `bson:"lines" json:"lines"`
	// What moved, at the sender's prices on the day. ⚠️ **Carried, not
	// created** — the same rule a transfer follows: nothing was bought and
	// nothing was lost, so this never reaches the financial report's expenses.
	Value int `bson:"value" json:"value"`

	// Who signs the slip. ⚠️ Free text rather than a staff id: the driver is
	// often somebody's cousin with a car, and a field that only accepts
	// employees is a field that gets left empty on the busiest morning.
	Driver string `bson:"driver,omitempty" json:"driver,omitempty"`
	Note   string `bson:"note,omitempty" json:"note,omitempty"`

	By   string             `bson:"by,omitempty" json:"by,omitempty"`
	ByID primitive.ObjectID `bson:"byId,omitempty" json:"-"`

	// When the far end counted it off the van, and who did.
	AcceptedAt *time.Time `bson:"acceptedAt,omitempty" json:"acceptedAt,omitempty"`
	AcceptedBy string     `bson:"acceptedBy,omitempty" json:"acceptedBy,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

// Accepted reports whether the receiving branch has signed for this.
func (d Dispatch) Accepted() bool { return d.AcceptedAt != nil }

// Lost is what the journey swallowed, in purchase units per ingredient.
//
// ⚠️ **Only for an accepted dispatch.** Before anybody counts it off the van
// there is no discrepancy to report — there is a slip nobody has checked, which
// is a different thing and reads differently on every screen that shows it.
func (d Dispatch) LostLines() []DispatchLine {
	if !d.Accepted() {
		return nil
	}
	var out []DispatchLine
	for _, l := range d.Lines {
		if l.Lost() != 0 {
			out = append(out, l)
		}
	}
	return out
}
