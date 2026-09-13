package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TillDevice is one screen bound to a branch — a monoblock on the counter, a
// tablet on the floor.
//
// ⚠️ **It exists because "how many registers does this branch run" had no
// answer.** The plan a restaurant buys is sold by register count, and until
// there was a row per machine that number was a sentence in a price list and
// nothing else: every till carried an identical branch token, so they could not
// be counted, named, or revoked one at a time. The panel's rotate button had to
// kill every screen in the building for the same reason.
//
// ⚠️ **A row is created when the link is issued, not when the machine first
// calls.** Issuing the link *is* the act of adding a register — it is what
// somebody does when they put a new machine on the counter — and counting only
// machines that had phoned home would let a restaurant hold ten unused links
// and bind them all on a busy Friday, past a cap that had already been checked.
// The cost is that an abandoned link occupies a slot until somebody removes it,
// which is visible on the panel and takes one tap.
type TillDevice struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID primitive.ObjectID `bson:"branchId" json:"branchId"`

	// What to call it on the panel: "Kassa 1", "Zal planshet".
	//
	// ⚠️ Free text and never read as a permission — the same rule as
	// Staff.Position. Its only job is to let a manager pick the right row when
	// a machine is stolen or replaced, and a list of identical rows is a list
	// nobody can act on.
	Name string `bson:"name" json:"name"`

	// The branch's revocation counter at the moment this was issued. A token
	// whose version is behind the branch's is dead — that is how "rotate the
	// key" still works for everything at once.
	Version int `bson:"version" json:"version"`

	// ⚠️ **Whether it has ever actually been used.** A link issued and never
	// opened and a machine selling all evening look identical on the panel
	// otherwise, and the difference is the whole basis for deciding which row
	// to remove when the cap is reached.
	LastSeenAt *time.Time `bson:"lastSeenAt,omitempty" json:"lastSeenAt,omitempty"`

	// ⚠️ **Which machine this actually is.** "Kassa 1" and "Kassa 2" name rows
	// in a list; they do not name anything standing in a building. A manager at
	// their cap has to decide which row to unbind, and until these two fields
	// existed the only way to tell one row from another was the day it was last
	// used — which is the same day for every till in a working restaurant.
	//
	// Host is the Windows computer name, sent by the till application itself
	// (X-Till-Host). Empty for a screen paired from a browser, which is exactly
	// the distinction worth seeing: a row with no machine behind it is the
	// abandoned link.
	Host string `bson:"host,omitempty" json:"host,omitempty"`
	// IP is where that machine last called from, on the restaurant's own
	// network. ⚠️ Recorded rather than trusted: it identifies a monoblock for
	// somebody walking the floor and decides nothing.
	IP string `bson:"ip,omitempty" json:"ip,omitempty"`

	// Who added it, so a machine nobody recognises has somebody to ask.
	IssuedBy  string    `bson:"issuedBy,omitempty" json:"issuedBy,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}
