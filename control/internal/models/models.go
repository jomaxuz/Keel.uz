// Package models holds the control plane's own documents.
//
// These live in their own database, never in a tenant's. A tenant database
// contains one restaurant and nothing about any other — that separation is the
// reason the platform cannot leak one customer's orders into another's screen,
// and it survives only if nothing here is ever written next to it.
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Tenant statuses.
const (
	// Paying, running, reachable.
	StatusActive = "active"
	// Evaluating. Same capabilities; the difference is what happens when the
	// clock runs out.
	StatusTrial = "trial"
	// Behind on payment. The site stops answering and the container is stopped
	// to free memory — but **the database is kept**. A restaurant that pays on
	// Thursday gets Wednesday's menu back, not an empty panel.
	StatusSuspended = "suspended"
	// Gone: the customer left, or was never really one. Behaves like suspended
	// at the edge, and additionally drops out of the list so it stops being
	// something to read past every morning.
	//
	// **Deleting is not dropping the database.** The menu, the photographs, the
	// customer base and a year of orders survive, because the restaurant that
	// comes back in March should not be asked to type its menu in again — and
	// because a mistaken click here would otherwise be the one action on this
	// whole platform that cannot be undone. Reclaiming the disk is a separate,
	// later, deliberate decision.
	StatusDeleted = "deleted"
)

// Offline reports whether the tenant's own server should be stopped and its
// domains served the "switched off" page instead.
//
// One predicate rather than two comparisons scattered across provisioning, the
// edge and the aggregator: the day a fourth status appears, a missed comparison
// would leave a stopped customer's container running or a live customer's
// domain dark.
func (t Tenant) Offline() bool {
	return t.Status == StatusSuspended || t.Status == StatusDeleted
}

// Tenant is one customer: one restaurant, pharmacy, flower shop — whatever
// they sell — with its own database, its own container and its own domains.
type Tenant struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	// Short machine name. Decides the database (`t_<slug>`), the container
	// (`keel-<slug>`) and the default domain (`<slug>.keel.uz`), so it never
	// changes after creation.
	Slug string `bson:"slug" json:"slug"`
	// What the business calls itself, for the dashboard.
	Name string `bson:"name" json:"name"`
	// What kind of business it is — restoran, dorixona, gul do'koni… Free text
	// on purpose: the list of things people sell is longer than any enum we
	// would guess right.
	Kind string `bson:"kind" json:"kind"`

	// Every hostname that must reach this tenant, including the default
	// subdomain. Caddy's map and the TLS ask endpoint are built from this.
	Domains []string `bson:"domains" json:"domains"`

	Status string `bson:"status" json:"status"`
	// When a trial stops being a trial. Nil for paying customers.
	TrialEndsAt *time.Time `bson:"trialEndsAt,omitempty" json:"trialEndsAt,omitempty"`

	// The day this customer started paying — the anchor every invoice is
	// counted from.
	//
	// It is a date on the tenant rather than a global calendar rule because the
	// period is the customer's: subscribe on the 17th and the period runs to
	// the 16th, every month, for as long as they stay. Billing everyone on the
	// 1st would give whoever signed up on the 28th a three-day first month and
	// whoever signed up on the 2nd a full one, for the same money.
	//
	// Nil while a tenant is still evaluating: a trial has an end, not a cycle.
	SubscribedAt *time.Time `bson:"subscribedAt,omitempty" json:"subscribedAt,omitempty"`

	// When the trial sweep switched this tenant off by itself.
	//
	// Recorded rather than merely logged because of the question it answers:
	// weeks later somebody asks why a site went dark, and "it switched itself
	// off" is not an answer anyone accepts. A server log rotates away; this
	// stays on the row the operator is already looking at.
	//
	// Cleared as soon as the status moves off suspended, so it can never
	// describe a later, manual switch-off that a human did on purpose.
	AutoSuspendedAt *time.Time `bson:"autoSuspendedAt,omitempty" json:"autoSuspendedAt,omitempty"`

	// So'm per order. Kept per tenant rather than read from a global constant
	// so an early customer's price survives a later price rise.
	PricePerOrder int `bson:"pricePerOrder" json:"pricePerOrder"`

	// The paid removal of the "Powered by Keel" line in the site footer.
	//
	// It is stored **here**, on the control plane, and pushed into the tenant.
	// Left for the restaurant's own settings page to hold, an owner would
	// simply switch it off — the same trap as kioskSecret and soldOut in the
	// tenant app, except this one is the business model.
	HideWatermark bool `bson:"hideWatermark" json:"hideWatermark"`

	// Who to call.
	OwnerName  string `bson:"ownerName" json:"ownerName"`
	OwnerPhone string `bson:"ownerPhone" json:"ownerPhone"`

	// The first admin account of the tenant's own panel — what the customer
	// types at <their-domain>/admin.
	//
	// It is set here because it has to exist before the container does: the
	// tenant server seeds its first owner from ADMIN_USERNAME/ADMIN_PASSWORD on
	// first boot, and there is no other moment when a human is present to
	// choose one.
	AdminUsername string `bson:"adminUsername" json:"adminUsername"`
	// Held in the clear, and deliberately only until the container has been
	// started: the tenant app hashes it itself, so a hash here could not be
	// handed over. `json:"-"` keeps it out of every response — the operator
	// typed it, so nothing needs to read it back.
	AdminPassword string `bson:"adminPassword" json:"-"`
	// Computed for the panel, so a form can say "stored" without ever
	// receiving the value.
	HasAdminPassword bool `bson:"-" json:"hasAdminPassword"`

	// Its own secret, generated at creation. A token minted for one restaurant
	// is then not merely unauthorized at another — it is unreadable.
	JWTSecret string `bson:"jwtSecret" json:"-"`

	// What happened the last time we tried to bring this tenant up.
	// "" | "ready" | "failed" — kept on the tenant for the same reason the POS
	// result is kept on the order: the question is always about this one, and
	// the answer has to survive long enough for somebody to retry it.
	ProvisionStatus string     `bson:"provisionStatus" json:"provisionStatus"`
	ProvisionError  string     `bson:"provisionError" json:"provisionError"`
	ProvisionedAt   *time.Time `bson:"provisionedAt,omitempty" json:"provisionedAt,omitempty"`
	// Live container state, filled in for the panel. Not stored: it is Docker's
	// answer, and a cached one is worse than none.
	ContainerStatus string `bson:"-" json:"containerStatus,omitempty"`

	Note      string    `bson:"note" json:"note"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// DBName is where this tenant's data lives.
func (t Tenant) DBName() string { return "t_" + t.Slug }

// TenantDay is one tenant's numbers for one calendar day.
//
// Aggregated nightly rather than queried live: rendering the overview by
// dialling every tenant database takes as long as the customer list is, and
// the answer is the same one all day. The monthly invoice reads the same rows,
// so nothing is counted twice by two different pieces of code.
type TenantDay struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	TenantID primitive.ObjectID `bson:"tenantId" json:"tenantId"`
	// Local calendar day, "YYYY-MM-DD" — same convention as staff shifts in
	// the tenant app, and for the same reason: a day is what the owner sees on
	// a wall calendar, not a UTC boundary.
	Date string `bson:"date" json:"date"`

	// Orders that reached the kitchen. Cancelled ones are excluded on purpose:
	// they are not billed, and billing a restaurant for an order it cancelled
	// itself is the first argument you will have with a customer.
	Orders int `bson:"orders" json:"orders"`
	// So'm taken through the platform that day, for the tenant's own curve.
	Revenue int `bson:"revenue" json:"revenue"`
	// What we charge for that day: Orders × the tenant's price at the time.
	Billable int `bson:"billable" json:"billable"`

	CollectedAt time.Time `bson:"collectedAt" json:"collectedAt"`
}

// User is a Keel employee who can open the dashboard. Deliberately minimal:
// this is us, not the customers.
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username     string             `bson:"username" json:"username"`
	PasswordHash string             `bson:"passwordHash" json:"-"`
	Name         string             `bson:"name" json:"name"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
	LastLoginAt  *time.Time         `bson:"lastLoginAt,omitempty" json:"lastLoginAt,omitempty"`
}
