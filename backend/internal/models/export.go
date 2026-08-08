package models

import "time"

// ExportGrantID is the single document's id: one grant per install.
const ExportGrantID = "export"

// ExportGrant is the platform's permission for this restaurant to download
// everything it has, as one archive.
//
// **The data is the customer's and they must be able to leave with it.** A
// restaurant that cannot take its menu, its orders and its customer list to
// another system is a restaurant held by the exit cost rather than by the
// product, and that is both wrong and, in the end, bad business — nobody
// recommends a supplier they feel trapped by.
//
// So why is it not simply a button on their own settings page?
//
// Because the archive is the single most dangerous object this system can
// produce: one file holding every customer's name, phone number and address,
// every order they ever placed, and the restaurant's whole commercial history.
// A permanent download button turns any borrowed panel session — a manager who
// left, a laptop in a back office, a shared password — into a complete copy of
// the business, silently. The grant makes taking that copy a **deliberate,
// dated, attributable act**: somebody at Keel turned it on, for a written
// reason, and it turns itself off again.
//
// It lives in the tenant's own database rather than on the tenant row in the
// control plane, and the console writes it there directly. One document, one
// writer, one reader — nothing to keep in sync, and no new network path from a
// tenant container to the control plane (that path not existing is what makes
// one restaurant unable to reach another's data).
type ExportGrant struct {
	ID      string `bson:"_id" json:"-"`
	Enabled bool   `bson:"enabled" json:"enabled"`
	// Why it was opened, written by the person who opened it. Required: a
	// permission that exists for a reason nobody recorded is one nobody can
	// judge later, and this is the permission most likely to be asked about.
	Reason string `bson:"reason" json:"reason"`
	// Which console operator, and when.
	GrantedBy string    `bson:"grantedBy" json:"grantedBy"`
	GrantedAt time.Time `bson:"grantedAt" json:"grantedAt"`
	// When it closes again.
	//
	// **Required, and that is the point.** A switch left on is the leak: the
	// migration finishes, everyone moves on, and a year later the button is
	// still there on a panel whose password has been round three managers. An
	// expiring grant fails closed by doing nothing at all.
	ExpiresAt time.Time `bson:"expiresAt" json:"expiresAt"`

	// Every archive actually taken. Capped, newest last.
	//
	// Kept because "was it downloaded?" is the first question after a leak,
	// and the honest answer has to survive the grant expiring. The tenant's own
	// audit log records it too; this copy is the one the console can read
	// without opening the restaurant's panel.
	Downloads []ExportDownload `bson:"downloads" json:"downloads"`
}

// ExportDownload is one archive leaving the building.
type ExportDownload struct {
	At time.Time `bson:"at" json:"at"`
	// The panel account that pressed it — always an owner, never a manager.
	By string `bson:"by" json:"by"`
	// How big it was and how many files it held. Two numbers that make an
	// unusual download visible: an archive an order of magnitude off the last
	// one is either a mistake or somebody else's data.
	Bytes int64 `bson:"bytes" json:"bytes"`
	Files int   `bson:"files" json:"files"`
}

// Allowed reports whether an archive may be built right now.
//
// Time is checked here rather than at the moment the grant is written, because
// that is the only check that keeps being true: a grant written with an expiry
// is harmless the day after regardless of what any screen still shows.
func (g *ExportGrant) Allowed(now time.Time) bool {
	return g != nil && g.Enabled && now.Before(g.ExpiresAt)
}
