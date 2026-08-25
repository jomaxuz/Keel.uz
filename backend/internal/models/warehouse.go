package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Warehouses: which store a thing is kept in ----
//
// ⚠️ **A restaurant does not have one store, it has a bar and a kitchen.** They
// are counted by different people, on different evenings, and a bottle of vodka
// missing from the bar is not answered by "the kitchen has more stock than
// expected". Until this existed every count was one list covering both, so the
// bar's shortfall and the kitchen's surplus cancelled out — and the number the
// owner read was the one number that could not be acted on.
//
// ⚠️ **The warehouse belongs to the ingredient, not to the delivery or the
// dish.** "Where is the vodka kept" has one answer and it does not change per
// invoice; "which store did this dish come out of" has to be derived, and any
// scheme that asks a cashier or a buyer to answer it per line is a scheme that
// gets answered wrongly under pressure. So a delivery routes itself line by
// line, a write-off routes itself, and only a count — which is a person walking
// into one room — names a warehouse.
//
// ⚠️ **One ingredient, one warehouse.** A restaurant that genuinely keeps
// lemons behind the bar *and* in the kitchen makes two ingredients, which is
// what it already does on paper: two shelves, two counts, two people
// responsible. Per-warehouse balances for a shared ingredient would be a
// second, invisible dimension on every purchase, write-off and count — and the
// first thing that goes wrong with it is a count that cannot be reconciled
// because nobody knows which shelf the invoice landed on.
type Warehouse struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`
	Name     string             `bson:"name" json:"name" validate:"required"`
	// Free text: who counts it, where it is. Same judgement as the ingredient's
	// note — the fact only means something beside where it came from.
	Note string `bson:"note,omitempty" json:"note,omitempty"`
	// Kind marks a central kitchen ("production"); empty is an ordinary store.
	//
	// ⚠️ **It is not decoration — it decides where a batch may be made.** A
	// production document takes its inputs off one store's shelves and puts the
	// batch onto them, and that store has to be the room somebody actually
	// cooked in. Allowing it anywhere would let a branch "make" sauce out of
	// tomatoes standing in another branch's kitchen, which the arithmetic would
	// accept and no screen would question.
	//
	// ⚠️ Empty is an ordinary store, which is every store that existed before
	// this field — including the kitchens of restaurants that make their own
	// sauce as they go and never want a batch document.
	Kind string `bson:"kind,omitempty" json:"kind,omitempty"`
	Sort int    `bson:"sort" json:"sort"`
	// ⚠️ Deactivated rather than deleted, because every count, delivery and
	// write-off ever recorded points at it. A warehouse removed outright takes
	// the meaning of a year of stocktakes with it.
	IsActive  bool      `bson:"isActive" json:"isActive"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// IsProduction reports whether batches may be made in this store.
func (w Warehouse) IsProduction() bool { return w.Kind == "production" }
