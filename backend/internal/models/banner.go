package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Banners: the strip the restaurant controls itself.
//
// ⚠️ **Separate from the hero, not a replacement for it.** The hero is the site's face —
// its name, its cover, the two buttons that never change. A banner is this week: a
// discount, a new dish, Navruz. Merging them would mean the owner editing their own
// identity every time they run a promotion, and forgetting to put it back.
//
// ⚠️ **A collection rather than a field on the restaurant.** Banners are added, reordered
// and switched off constantly, and each one is a row with its own life — a slice inside
// the singleton would be rewritten whole on every edit, which is how a second admin's
// change disappears.
// Where a banner is shown.
//
// ⚠️ **Two audiences, two shapes, one collection.** A site banner is wide, aimed
// at a guest, and usually a discount; a till banner is tall, aimed at the staff
// standing in front of a locked monoblock, and a discount there is advertising
// to the four people who already work here. They cannot share a slot. But they
// are the same *kind* of thing — added, reordered, switched off and scheduled
// constantly — so they share the collection and the CRUD rather than growing a
// second, half-finished copy of both.
//
// ⚠️ **The zero value is the site**, because every banner that exists today is a
// site banner and the field is absent on all of them. Reading a missing
// placement as anything else would empty the home page of every install on the
// deploy that shipped this.
const (
	BannerOnSite = "site"
	BannerOnTill = "till"
)

type Banner struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BrandID primitive.ObjectID `bson:"brandId,omitempty" json:"brandId,omitempty"`
	// "" / "site" — the home page strip. "till" — the lock screen of this
	// brand's monoblocks. See the constants above.
	Placement string `bson:"placement,omitempty" json:"placement,omitempty"`

	// The picture, and nothing else is required: a banner with no words is a normal
	// banner, and most of them are made in Canva with the words already on them.
	ImageURL string `bson:"imageUrl" json:"imageUrl"`
	// Where it leads. ⚠️ One of the site's own pages, or a dish — never an arbitrary
	// URL. A banner is the most-clicked thing on the page, and "any address" there is a
	// way to send a restaurant's own guests somewhere else.
	Link string `bson:"link,omitempty" json:"link,omitempty"`
	// Optional caption, in three languages like every other piece of site copy.
	Title LocalizedText `bson:"title,omitempty" json:"title,omitempty"`

	SortOrder int  `bson:"sortOrder" json:"sortOrder"`
	IsActive  bool `bson:"isActive" json:"isActive"`
	// A promotion that ends. ⚠️ Checked when the banners are read rather than by a job
	// that switches them off: a discount that expired at midnight must stop being
	// advertised at midnight, not at the next time anything happens to run.
	StartsAt *time.Time `bson:"startsAt,omitempty" json:"startsAt,omitempty"`
	EndsAt   *time.Time `bson:"endsAt,omitempty" json:"endsAt,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// Live reports whether this banner should be shown now.
func (b Banner) Live(now time.Time) bool {
	if !b.IsActive || b.ImageURL == "" {
		return false
	}
	if b.StartsAt != nil && now.Before(*b.StartsAt) {
		return false
	}
	if b.EndsAt != nil && now.After(*b.EndsAt) {
		return false
	}
	return true
}

// Vacancy is a job the restaurant is hiring for.
//
// ⚠️ On the site rather than only on Telegram, because the people who see it are the
// guests: a waiter is hired from the same street the customers walk down, and a restaurant
// with a "we are hiring" page fills a shift faster than one posting in a channel nobody
// in that street follows.
type Vacancy struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BrandID  primitive.ObjectID `bson:"brandId,omitempty" json:"brandId,omitempty"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`

	Title       LocalizedText `bson:"title" json:"title"`
	Description LocalizedText `bson:"description,omitempty" json:"description,omitempty"`
	// Free text on purpose: "3–5 mln", "kelishilgan holda", "kunlik to'lov". A number
	// field would force every restaurant into one way of paying, and most of them use
	// two.
	Salary string `bson:"salary,omitempty" json:"salary,omitempty"`
	// "full" | "part" | "shift" — a hint, not a contract.
	Employment string `bson:"employment,omitempty" json:"employment,omitempty"`

	SortOrder int  `bson:"sortOrder" json:"sortOrder"`
	IsActive  bool `bson:"isActive" json:"isActive"`

	// How many people applied. ⚠️ Kept as a counter as well as being countable from the
	// applications: the list of vacancies is the screen where an owner decides which one
	// to close, and a count per row is what makes that decision without opening six
	// screens.
	Applications int       `bson:"applications" json:"applications"`
	CreatedAt    time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time `bson:"updatedAt" json:"updatedAt"`
}

// JobApplication is one person asking about one job.
//
// Deliberately two fields and a comment. ⚠️ A CV upload, an email, a date of birth — each
// one is a reason not to apply from a phone on a bus, and the restaurant is going to ring
// them anyway. The name and the number are what makes that call possible.
type JobApplication struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	VacancyID    primitive.ObjectID `bson:"vacancyId" json:"vacancyId"`
	VacancyTitle string             `bson:"vacancyTitle" json:"vacancyTitle"`
	BranchID     primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`

	Name    string `bson:"name" json:"name"`
	Phone   string `bson:"phone" json:"phone"`
	Comment string `bson:"comment,omitempty" json:"comment,omitempty"`

	// new | called | hired | refused. The owner moves it; nothing else does.
	Status    string    `bson:"status" json:"status"`
	Note      string    `bson:"note,omitempty" json:"note,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

// Application statuses. Stored, so never renamed.
const (
	JobNew     = "new"
	JobCalled  = "called"
	JobHired   = "hired"
	JobRefused = "refused"
)

// OnTill reports whether this banner belongs to the till's lock screen.
//
// ⚠️ Written as "is it the till one" rather than "is it the site one", so the
// unknown value falls to the site — the same direction as the zero value, and
// the direction where a mistake shows a banner in the wrong place instead of
// blanking the home page.
func (b Banner) OnTill() bool { return b.Placement == BannerOnTill }
