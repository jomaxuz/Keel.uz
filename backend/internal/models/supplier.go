package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Who the food comes from ----
//
// ⚠️ **It was free text, and free text cannot be asked a question.** A delivery
// carried whoever typed it: "Makro", "makro", "Makro MCHJ" and "макро" are one
// company and four rows, so "how much did we buy from Makro this quarter" had
// no answer at all — not a wrong one, none. The original note said a restaurant
// with three regulars and a market run does not keep a supplier list, and that
// is still true, which is why **the text field stays** and naming a supplier is
// optional. This is for the three regulars.
//
// ⚠️ **Brand-level, like the ingredients it supplies.** A chain buys its meat
// from one butcher for every branch, and filing the butcher per branch would
// produce one row per kitchen and split every total the list exists to give.
// Deliveries stay branch-level — they are events in one kitchen — so "what did
// this branch buy from him" and "what did the company buy from him" are both
// still answerable.
//
// ⚠️ **Deactivated, never deleted.** Every delivery ever entered points here,
// and a supplier removed outright takes the meaning of a year of invoices with
// it — the same rule a warehouse follows.
type Supplier struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BrandID primitive.ObjectID `bson:"brandId,omitempty" json:"brandId,omitempty"`
	Name    string             `bson:"name" json:"name" validate:"required"`
	// Who to ring when the delivery is late. The single most useful field here
	// and the reason a phone in somebody's contacts becomes a fact the business
	// owns rather than one that leaves with them.
	Phone string `bson:"phone,omitempty" json:"phone,omitempty"`
	// Free text: what they bring, which day they come, who to ask for.
	Note      string    `bson:"note,omitempty" json:"note,omitempty"`
	Sort      int       `bson:"sort" json:"sort"`
	IsActive  bool      `bson:"isActive" json:"isActive"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}
