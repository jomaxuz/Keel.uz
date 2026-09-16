package models

import "time"

// SubscriptionID is the single document's id: one subscription per install.
const SubscriptionID = "subscription"

// Module ids, mirrored from the platform's price list.
//
// ⚠️ **Stable strings, and they are stored.** Renaming one silently returns a
// paid module to every customer who bought it, or takes it away from every
// customer who has it — and neither shows up as an error anywhere. The same
// rule as the dashboard tile ids.
const (
	ModStock          = "stock"
	ModMultiBranch    = "multibranch"
	ModPOSIntegration = "posint"
	ModFranchise      = "franchise"
	// The televisions in the dining room: the content the panel sends them and
	// the order board. Priced per screen — see Subscription.Screens.
	ModTV = "tv"
	// The advertising section: the Meta ad account, the assistant that proposes
	// campaigns, and what they brought back.
	//
	// ⚠️ **Bought on its own and included in no plan** (billing/ads.go in the
	// console). The restaurant pays Meta for the advertising itself with its own
	// card; this module is the tool.
	ModAds = "ads"
)

// Subscription is what this restaurant bought, as the console resolved it.
//
// **Written by the console, read here, never written here.** The same shape as
// ExportGrant next door and for the same two reasons: one writer and one reader
// need nothing kept in sync, and no path is opened from this container back to
// the control plane. The difference is what it protects — the grant guards the
// customer's data, this one guards the price list, and leaving it in the
// restaurant's own settings would let an owner move themselves to the top rung.
//
// ⚠️ **Entitlements, not a plan to interpret.** `Modules` is the list this
// server checks; `Plan` is only a name for a heading. Deriving the first from
// the second would put a second copy of the ladder in here, and two copies of a
// price list part company on the first rung anybody renames.
//
// ⚠️ **An absent document means "no till", and that is the safe direction.**
// Every install that predates this has no monoblock, and reading a missing
// document as the cheapest plan would switch a counter on for customers who
// never bought one. Note this is the *opposite* default from most zero values
// in this codebase, for the same reason `canKitchen` is: the thing being
// defaulted is a permission, and the harm of granting one nobody bought is
// larger than the harm of withholding one — here it is also the harm of
// billing for it.
type Subscription struct {
	ID      string `bson:"_id" json:"-"`
	Enabled bool   `bson:"enabled" json:"enabled"`
	// The rung's name, for a heading. Never checked as a permission.
	Plan string `bson:"plan" json:"plan"`
	// What is actually switched on, resolved from the rung plus any add-on.
	Modules []string `bson:"modules" json:"modules"`
	// How many till screens one branch may bind. 0 means no cap.
	Registers int `bson:"registers" json:"registers"`
	// How many televisions one branch may pair. 0 means no cap.
	//
	// ⚠️ **Its own number rather than a module flag**, because the screens are
	// priced one at a time: "the TV module" is not what a restaurant buys, four
	// screens is. Resolved by the console like every other figure here — this
	// server never works a price or a limit out for itself.
	Screens int `bson:"screens" json:"screens"`
	// ⚠️ **A date, not a flag.** The screens in the restaurant count down to
	// it. A stored "subscription ok" boolean is stale the moment the clock
	// passes it, and nobody is watching a monoblock at midnight — the same
	// lesson as provisionStatus, lastEventAt and lastUpdateAt.
	PaidUntil *time.Time `bson:"paidUntil,omitempty" json:"paidUntil,omitempty"`
	// ---- What it costs ----
	//
	// ⚠️ **Resolved by the console, mirrored here, never worked out locally.**
	// Branch discounts, add-ons and a negotiated price all move this figure, so
	// a panel that multiplied a rung's list price by anything would show the
	// owner a number their invoice disagrees with — and this is the screen they
	// read before paying. Same argument as `Modules`: the entitlement arrives
	// resolved so this server never holds a second copy of the price list.
	//
	// ⚠️ 0 is not "free". An Enterprise rung is agreed per customer, and the
	// console deliberately mirrors nothing rather than invent a figure; the
	// panel shows the plan with no price instead of a price of nothing.
	Monthly int `bson:"monthly,omitempty" json:"monthly"`
	// What was bought on top of the rung, for the heading. Already inside
	// `Modules` — this is never checked as a permission.
	Addons []string `bson:"addons,omitempty" json:"addons"`
	// How many branches the price covers. Without it a chain's figure looks
	// like a mistake.
	Branches int `bson:"branches,omitempty" json:"branches"`
	// The ladder, so the panel can put a price on an upgrade button without a
	// second request to a service it cannot reach.
	Plans     []SubscriptionPlan `bson:"plans" json:"plans"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// SubscriptionPlan is one rung as the panel draws it on an upgrade card.
type SubscriptionPlan struct {
	ID         string   `bson:"id" json:"id"`
	Monthly    int      `bson:"monthly" json:"monthly"`
	Registers  int      `bson:"registers" json:"registers"`
	Modules    []string `bson:"modules" json:"modules"`
	Individual bool     `bson:"individual" json:"individual"`
}

// Has reports whether a module is switched on.
//
// A nil receiver answers false, so every caller can read a missing document
// without a nil check and get the safe answer.
func (s *Subscription) Has(mod string) bool {
	if s == nil || !s.Enabled {
		return false
	}
	for _, m := range s.Modules {
		if m == mod {
			return true
		}
	}
	return false
}

// DaysLeft is how many whole days remain on the paid period, and whether there
// is a date to count at all.
//
// ⚠️ **Counted in whole local days, not in hours.** The number goes on a screen
// as "3 kun qoldi", and a countdown that turns from 3 to 2 at four in the
// afternoon because that is when somebody paid is a countdown a restaurant
// stops believing. Negative once the date has passed — the caller decides what
// an expired subscription means, because that answer differs between the
// counter and the panel.
func (s *Subscription) DaysLeft(now time.Time) (int, bool) {
	if s == nil || s.PaidUntil == nil || s.PaidUntil.IsZero() {
		return 0, false
	}
	end := startOfLocalDay(s.PaidUntil.In(time.Local))
	today := startOfLocalDay(now)
	return int(end.Sub(today).Hours() / 24), true
}

// startOfLocalDay is written here rather than reused from a report helper so
// this file has no dependency on one: the till reads it on every wake.
//
// ⚠️ `.In(time.Local)` at the call site above is not decoration — the driver
// decodes every stored time as UTC, so a date written at local midnight comes
// back as 19:00 the previous day and the countdown would be a day out.
func startOfLocalDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}
