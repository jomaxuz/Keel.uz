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

// PriceTier is one band of the volume ladder: taking this band prices **every**
// order in the period at Price. `UpTo` is the running order count the band ends
// at; **0 means no limit**, which the last band must use.
//
// A band's own `UpTo` also sets where the *next* band starts, and that entry
// price is what the next band costs at minimum — see PriceForOrders.
type PriceTier struct {
	UpTo  int `bson:"upTo" json:"upTo"`
	Price int `bson:"price" json:"price"`
}

// PriceForOrders is what a period's orders cost, before free terms and
// discounts.
//
// ⚠️ **Counted over the whole period, never per day.** Tiers applied daily
// would reset every midnight, so a restaurant doing 400 a day would never
// leave the first band and the ladder would do nothing at all. This is why
// TenantDay.Billable stays a flat daily estimate and the invoice recomputes
// from the period's order count — the two are allowed to differ, and the
// invoice is the one that is right.
//
// ⚠️ **"Whichever band is cheapest", not marginal bands.** This used to bill
// the first 3 000 orders at the top rate and only the overflow at the next one,
// which is arithmetically unable to ever reach the bottom rate: the average
// always sits above the last band it touched. Every competitor here prices the
// **whole** volume at the band's rate, so at the volumes where a chain actually
// negotiates, a marginal ladder loses to a headline number that looks higher —
// 6 000 orders averaged 850 against a competitor's flat 700.
//
// So a band may be entered early, by paying it in full: 2 000 orders may buy the
// 3 000-order band if 3 000 × its rate is less than 2 000 × the base rate. That
// keeps the two properties that matter at once — the average lands exactly on the
// published band rate, and the bill never *drops* as orders grow (each candidate
// is non-decreasing in `orders`, and so is their minimum).
func PriceForOrders(orders int, tiers []PriceTier, flat int) int {
	if orders <= 0 {
		return 0
	}
	if len(tiers) == 0 {
		return orders * flat
	}
	// Where the band being considered begins — the previous band's limit. The
	// first band starts at zero, so it is simply orders × rate.
	best, entry := 0, 0
	for i, t := range tiers {
		billed := max(orders, entry)
		if cost := billed * t.Price; i == 0 || cost < best {
			best = cost
		}
		// The open-ended band ends the ladder. Also stops a mis-ordered list
		// from inventing an entry price out of a limit that already passed.
		if t.UpTo <= 0 {
			break
		}
		entry = t.UpTo
	}
	return best
}

// FreeAt reports whether this customer pays nothing on the given day.
//
// One predicate rather than the same two comparisons in the invoice, the
// dashboard and the trial sweep — the day a third condition appears, a missed
// comparison means billing somebody who was promised otherwise, which is the
// one billing mistake that costs a customer rather than money.
func (t Tenant) FreeAt(now time.Time) bool {
	if !t.Free {
		return false
	}
	// Nil is forever, not "expired at the zero time".
	return t.FreeUntil == nil || now.Before(*t.FreeUntil)
}

// ChargeForOrders is the whole price of a period: the volume ladder, then free
// terms, then any standing discount — in that order, and in one place so the
// invoice, the customer list and the card cannot each work it out differently.
//
// `defaults` is the platform's tier table, used when this tenant has none of
// its own.
func (t Tenant) ChargeForOrders(orders int, defaults []PriceTier, minMonthly int, now time.Time) int {
	return t.ChargeFor(ApplyMinimum(
		PriceForOrders(orders, t.LadderOrDefault(defaults), t.PricePerOrder),
		orders, t.Minimum(minMonthly)), now)
}

// LadderOrDefault is the tier table this customer is actually priced by: their
// own if they negotiated one, otherwise the platform's.
//
// Extracted because a second caller appeared (the nightly estimate) and the
// precedence is the kind of two-line rule that gets retyped slightly
// differently the third time — at which point the dashboard and the invoice
// quote two prices for the same day.
func (t Tenant) LadderOrDefault(defaults []PriceTier) []PriceTier {
	if len(t.PriceTiers) > 0 {
		return t.PriceTiers
	}
	return defaults
}

// EntryRate is the so'm-per-order figure a single day is estimated at.
//
// ⚠️ **Read from the ladder, never from the stored `PricePerOrder`.** That
// field is documented as a mirror of the first band, and a mirror is a copy:
// every tenant created before the ladder moved to 800 still holds the old
// 1 000 in it, so the nightly rows — and therefore the console's "our takings"
// — were priced 25% above what those same customers were actually invoiced.
// Nothing failed and no figure looked odd; the dashboard simply reported a
// business a quarter larger than the one that existed, which is the kind of
// wrong number that gets planned against rather than questioned.
//
// A day is deliberately still priced at the **entry** band rather than the one
// the month will land on: the invoice recomputes from the period's total (see
// PriceForOrders), and a daily row that guessed the final band would read as a
// promise the month had not yet earned. The two are allowed to differ; the
// invoice is the one that is right.
func (t Tenant) EntryRate(defaults []PriceTier) int {
	if ladder := t.LadderOrDefault(defaults); len(ladder) > 0 {
		return ladder[0].Price
	}
	return t.PricePerOrder
}

// Minimum is the floor this customer's period is charged at, in so'm.
//
// Per tenant for the same reason the price is: a customer who agreed terms
// before a floor existed keeps them, and raising the platform default must not
// silently reprice everybody who already said yes to something else. A tenant
// value of 0 falls back to the platform's.
func (t Tenant) Minimum(platform int) int {
	if t.MinMonthly > 0 {
		return t.MinMonthly
	}
	return platform
}

// ApplyMinimum raises a period's charge to the floor.
//
// The ladder prices orders; this prices **being a customer**. A restaurant
// doing five orders a day bills about 150 000 so'm a month, and the support it
// needs — the calls, the menu fixes, the "why is the printer not printing" —
// costs the same as the restaurant doing four hundred. The ladder deliberately
// bends the top of the curve down; without a floor the bottom of it runs below
// what serving that customer costs at all.
//
// ⚠️ **A period with no orders is never floored.** Zero orders almost always
// means the site is not live yet, or the restaurant was closed — the customer
// has had nothing from us and knows it, and an invoice arriving for a month
// they did not use is the single most effective way to lose one. Charging for
// availability is a defensible model; it is just not the one anybody agreed to
// here, and it must not arrive as a side effect of a floor.
//
// Applied before the discount, not after: a negotiated percentage that could
// not move the floor would be a discount that quietly does nothing for exactly
// the customers small enough to have asked for one.
func ApplyMinimum(amount, orders, minimum int) int {
	if minimum <= 0 || orders <= 0 || amount >= minimum {
		return amount
	}
	return minimum
}

// ChargeFor turns a period's raw billable amount into what the customer is
// actually asked for.
//
// Free wins over the discount: an account marked free is free, whatever
// percentage was left behind from before.
func (t Tenant) ChargeFor(amount int, now time.Time) int {
	if t.FreeAt(now) {
		return 0
	}
	d := t.DiscountPercent
	if d <= 0 {
		return amount
	}
	if d >= 100 {
		return 0
	}
	// Integer so'm throughout, rounded down: the customer keeps the tiyin.
	return amount - amount*d/100
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
	// Which template the tenant's first brand was created from.
	//
	// ⚠️ **Not the same question as `Kind`, and they are deliberately both
	// here.** `Kind` is what a person reads in a list — free text, because the
	// list of things people sell is longer than any enum we would guess right,
	// and "gul do'koni va kofexona" is a real answer. This is what the software
	// *does*: which switches the first brand starts with. Folding them together
	// would either force the software to understand free text or force the
	// operator into six boxes.
	//
	// ⚠️ Sent to the tenant on its first boot only, like the admin credentials.
	// Afterwards the panel's own switches are the truth.
	BusinessType string `bson:"businessType,omitempty" json:"businessType,omitempty"`

	// This restaurant's own Firebase app id, for its Android app's
	// notifications — "1:889013622083:android:abc…".
	//
	// ⚠️ **One per restaurant, and it cannot be shared.** An FCM registration
	// token is bound to a Firebase app id, and the SDK sends the package name
	// with it: running `uz.keel.app.navvat` under the id issued for
	// `uz.keel.team` is not a supported configuration. `getToken()` usually
	// succeeds anyway and the sends quietly go nowhere — no error, on either
	// side, ever. So each application is registered in Keel's Firebase project
	// and its id is kept here.
	//
	// ⚠️ **Typed in by hand today, and that is the honest state.** Creating one
	// through the Firebase Management API needs a service account with rights
	// the messaging credentials do not carry; until that is set up it is thirty
	// seconds in a console, once per restaurant, and the field says so.
	//
	// ⚠️ Empty means the app is built without notifications — it works, and the
	// one thing it does not do is the one nobody has set up. Never a build
	// failure: refusing to build a restaurant's app over a feature they have not
	// asked for is the wrong failure.
	AndroidAppID string `bson:"androidAppId,omitempty" json:"androidAppId,omitempty"`
	// Who signed this customer up. ⚠️ The **id** as well as the name: an agent's
	// list is filtered on this, and a name is editable while an id is not — a
	// filter on a name is a filter somebody can walk out of by renaming
	// themselves.
	CreatedByID   primitive.ObjectID `bson:"createdById,omitempty" json:"createdById,omitempty"`
	CreatedBy     string             `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	CreatedByRole string             `bson:"createdByRole,omitempty" json:"createdByRole,omitempty"`

	// Who sent this customer to us from outside — see models/referral.go.
	//
	// ⚠️ **Beside `createdBy`, not instead of it.** They answer different
	// questions and both have money attached: the agent is who closed it, the
	// referrer is who found it. A register firm's lead closed by an agent has
	// to pay both, and one field cannot say that.
	//
	// ⚠️ The **id** filters and the **code** is frozen, the same split as the
	// agent above: a referrer may rename their code (they print new leaflets),
	// and last quarter's attribution must not move when they do.
	ReferrerID   primitive.ObjectID `bson:"referrerId,omitempty" json:"referrerId,omitempty"`
	ReferrerCode string             `bson:"referrerCode,omitempty" json:"referrerCode,omitempty"`

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

	// When this customer's data and infrastructure were erased for good, by whom
	// and why. See handlers/purge.go.
	//
	// ⚠️ The row itself stays, and that is the point of recording this on it: the
	// invoices point here, the slug must never be handed to somebody else, and
	// "there is no record at all" is the one answer that turns a closed account
	// into an argument. A purged tenant is `deleted` **and** carries this date;
	// `deleted` alone still means the reversible kind.
	PurgedAt    *time.Time `bson:"purgedAt,omitempty" json:"purgedAt,omitempty"`
	PurgedBy    string     `bson:"purgedBy,omitempty" json:"purgedBy,omitempty"`
	PurgeReason string     `bson:"purgeReason,omitempty" json:"purgeReason,omitempty"`

	// So'm per order. Kept per tenant rather than read from a global constant
	// so an early customer's price survives a later price rise.
	//
	// With tiers in play this is the **first tier's** rate; a tenant whose
	// tiers are empty is billed at it flat.
	PricePerOrder int `bson:"pricePerOrder" json:"pricePerOrder"`

	// Volume tiers: the more a restaurant sells, the less each order costs.
	//
	// **This exists because of what the flat rate does to the best customer.**
	// At 400 orders a day the bill is 12 million so'm a month — about a
	// mid-level developer's salary here — and that is the point where a chain's
	// finance person stops reading the invoice and starts doing arithmetic.
	// Not because it is poor value (it is 0.5–2% of their revenue, against
	// 15–20% for an aggregator) but because it is a large line item, and large
	// line items get negotiated.
	//
	// Tiers rather than a cap: a cap makes every order past it worth nothing to
	// us, which is the wrong incentive on both sides. Tiers keep the marginal
	// rate positive while the average falls, so growth still pays — and the
	// conversation turns from "you are getting expensive" into "the more you
	// grow, the cheaper it gets".
	//
	// It costs less than it looks: across a realistic mix of customers most
	// never leave the first tier, so the platform gives up single digits of
	// revenue to remove the churn cliff at the top.
	//
	// Empty falls back to the platform default (config), and only then to a
	// flat PricePerOrder — so switching tiers on reaches existing customers
	// without editing every row, and a tenant that negotiated its own keeps it.
	PriceTiers []PriceTier `bson:"priceTiers,omitempty" json:"priceTiers,omitempty"`

	// The least this customer is billed for a period they used, in so'm.
	//
	// The other end of the same curve the tiers bend. The ladder protects the
	// biggest customer from a bill that invites negotiation; this protects the
	// platform from the smallest one, whose 150 000 so'm a month buys support
	// that costs the same as the customer paying twenty times more.
	//
	// 0 means "use the platform default", which is itself 0 unless configured —
	// so a floor never appears on anybody's invoice as a side effect of a
	// deploy. It is a pricing decision and has to be made like one.
	MinMonthly int `bson:"minMonthly,omitempty" json:"minMonthly,omitempty"`

	// This customer pays nothing.
	//
	// Not the same as `pricePerOrder: 0`, and the difference is the point. A
	// zero price is indistinguishable from a mistake — somebody clearing a
	// field, an import that lost a number — and it silently produces invoices
	// for nothing with no record of why. This says so out loud, carries the
	// reason, and can end on a date.
	//
	// It exists because the early customers are worth more than their
	// invoices. A chain of twelve restaurants that agrees to be the first real
	// user is buying the product a reputation, and charging them 200 000 so'm
	// a month to do it is the worst trade available. So: free, deliberately,
	// with the terms written down where the next person can read them.
	Free bool `bson:"free" json:"free"`
	// Why. **Required when Free is set** — an account that pays nothing for a
	// reason nobody recorded becomes an argument the day somebody asks, and
	// the person who agreed it will have left.
	FreeReason string `bson:"freeReason,omitempty" json:"freeReason,omitempty"`
	// When the free period ends. **Nil means forever**, which is a real
	// answer here rather than an oversight: an anchor customer may well have
	// been promised exactly that.
	FreeUntil *time.Time `bson:"freeUntil,omitempty" json:"freeUntil,omitempty"`

	// A standing discount, 0–100. The middle ground between paying and free:
	// the chain that wants a number to take to its board, the customer kept
	// through a bad quarter.
	DiscountPercent int `bson:"discountPercent" json:"discountPercent"`

	// The paid removal of the "Powered by Keel" line in the site footer.
	//
	// It is stored **here**, on the control plane, and pushed into the tenant.
	// Left for the restaurant's own settings page to hold, an owner would
	// simply switch it off — the same trap as kioskSecret and soldOut in the
	// tenant app, except this one is the business model.
	HideWatermark bool `bson:"hideWatermark" json:"hideWatermark"`
	// ⚠️ **When it was switched on**, because the add-on is billed by the day.
	//
	// Without the date the first invoice charges a full month for a badge hidden yesterday,
	// and the argument that follows costs more than the fee. Absent on every tenant that had
	// it before this was billed — treated as "the whole period", because they have been
	// getting the thing.
	HideWatermarkSince *time.Time `bson:"hideWatermarkSince,omitempty" json:"hideWatermarkSince,omitempty"`

	// The till subscription: whether this customer's counter is switched on at
	// all, on which plan, and since when.
	//
	// ⚠️ **Here, exactly like HideWatermark, and for the same reason**: left in
	// the restaurant's own settings an owner would move themselves to Pro, and
	// this one is not a badge in a footer — it is the whole till business model.
	// The restaurant's server is *told* what it is entitled to (a mirror
	// document in its own database, written by the console); it never decides.
	Till TenantTill `bson:"till" json:"till"`

	// Show this customer's logo on keel.uz as a reference.
	//
	// **Opt-in, and off by default.** Putting a restaurant's brand on our
	// marketing page is their decision, not ours — a customer who finds their
	// logo there without being asked is a customer with a complaint, and the
	// one thing a reference page cannot survive is being resented. One tick in
	// the console, after somebody has actually asked them.
	Showcase bool `bson:"showcase" json:"showcase"`

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
// TenantTill is what a customer bought for their counter.
//
// ⚠️ **Absent means "no till", and that is the safe direction.** Every existing
// customer predates this field, and none of them has a monoblock — reading a
// missing plan as Start would put a price on every one of their invoices for
// something they never asked for. The same "zero value is today's behaviour"
// rule the tenant app applies to an empty mapProvider, pointed at the outcome
// that costs money rather than the one that merely looks wrong.
type TenantTill struct {
	// Whether the counter is sold to this customer at all.
	//
	// ⚠️ Separate from the plan rather than encoded as an empty plan string. A
	// customer switched off for a month keeps the rung they negotiated, so
	// switching them back on does not quietly restore them to the cheapest one
	// — and "was off in March" stays answerable.
	Enabled bool `bson:"enabled" json:"enabled"`

	// billing.PlanStart | PlanStandard | PlanPro | PlanEnterprise.
	Plan string `bson:"plan,omitempty" json:"plan,omitempty"`

	// Blocks of ten daily assistant requests bought on top of the plan.
	//
	// ⚠️ **A daily limit sold monthly, and the two units are deliberate.** The
	// cap is daily because that is what protects us from a runaway — a monthly
	// pool would be spent in an afternoon by a stuck browser tab and the
	// restaurant would then have nothing for three weeks. It is sold monthly
	// because that is how a restaurant thinks about a bill.
	//
	// ⚠️ **Added to the plan's own cap, never replacing it.** Buying more must
	// not be a downgrade for anybody above the smallest rung.
	AIExtra int `bson:"aiExtra,omitempty" json:"aiExtra,omitempty"`

	// How many televisions this restaurant pays for.
	//
	// ⚠️ **The count is the entitlement**: it grants the module and prices it
	// at the same time (billing.TVScreenMonthly). A separate on/off switch
	// beside a number is two facts that can disagree — "the module is on and
	// they have no screens", "they have four screens and the module is off" —
	// and the second one is a restaurant whose dining room went dark.
	TVScreens int `bson:"tvScreens,omitempty" json:"tvScreens,omitempty"`

	// Modules bought on top of the plan (billing.ModStock today).
	//
	// Kept even while the plan includes them: a customer who bought stock on
	// Start and moved to Pro should not lose it by moving back down, and the
	// pricing already skips what the rung covers.
	Addons []string `bson:"addons,omitempty" json:"addons,omitempty"`

	// ⚠️ **When it was switched on**, because the subscription is billed by the
	// day — the same field, for the same reason, as HideWatermarkSince. A till
	// enabled on the 24th that arrives as a full month on the 1st is an invoice
	// the customer disputes, and they are right.
	Since *time.Time `bson:"since,omitempty" json:"since,omitempty"`

	// What was negotiated, when the ladder's own arithmetic does not apply.
	//
	// ⚠️ Enterprise has no computed price on purpose, so without this field an
	// Enterprise invoice would be zero — the failure that looks like a working
	// system right up until somebody reconciles a quarter. Non-zero overrides
	// the ladder for any plan; a chain that argued its way to a number keeps it
	// when the price list moves.
	PriceOverride int `bson:"priceOverride,omitempty" json:"priceOverride,omitempty"`

	// How many branches are being charged for.
	//
	// ⚠️ **A billing figure, not a count of rows in the restaurant's database.**
	// Reading it live would let a customer change their own invoice by adding a
	// branch, and would re-price a closed month every time the aggregate ran.
	// It is what was agreed, typed by the person who agreed it.
	Branches int `bson:"branches,omitempty" json:"branches,omitempty"`

	// When the current paid period runs out — the date the counter's screens
	// count down to.
	//
	// ⚠️ **A date, not a flag.** A stored "subscription ok" boolean goes stale
	// the moment the clock passes it, and this one is read by a screen standing
	// in a restaurant with nobody watching it — the same lesson as
	// provisionStatus and lastEventAt. Absent means nothing is counted down to,
	// which is what a customer who pays by hand every month looks like until
	// somebody records the date.
	PaidUntil *time.Time `bson:"paidUntil,omitempty" json:"paidUntil,omitempty"`

	// Free text: who agreed what, and when. Read by the next person on the phone.
	Note string `bson:"note,omitempty" json:"note,omitempty"`

	UpdatedBy string     `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
	UpdatedAt *time.Time `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

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
	// How many people came to the site that day, and how many pages they
	// opened. Kept beside the order counts because together they answer the
	// question neither can alone: a day with traffic and no orders is a
	// broken checkout, and a day with neither is a marketing problem.
	Visitors int `bson:"visitors" json:"visitors"`
	Views    int `bson:"views" json:"views"`
	// Cancelled that day — counted, never billed.
	//
	// Kept beside Orders rather than folded into it because it answers a
	// different question: not "what do they owe" but "is something going
	// wrong over there". A customer whose cancellations are climbing is one
	// about to phone, and rows that only hold what we can invoice cannot show
	// it. Absent on rows written before this field existed, which reads as 0 —
	// correct enough, since nobody can recover what was not counted.
	Cancelled int `bson:"cancelled" json:"cancelled"`
	// ⚠️ Cancelled **after it had already been delivered** — and billed anyway.
	//
	// This is the one number here that is about us rather than about the
	// restaurant's day. Our fee is per order and cancellations are free, which
	// is right for a kitchen that stops an order before cooking it — and is an
	// open invitation if the fee can be removed *after* the food is handed over.
	// It is a single click in the panel, and the nightly recount would have paid
	// it out: the row is rewritten from the order's current status, so an order
	// delivered on Tuesday and cancelled on Wednesday came back uncharged.
	//
	// Billing no longer reads the current status alone (see aggregate.one), so
	// this cannot cost us money. It is recorded because it is still worth
	// **seeing**: one is a guest refusing at the door, twenty a night is a
	// conversation to have.
	Reversed int `bson:"reversed" json:"reversed"`
	// Cancelled after the kitchen had it (`queuedAt` set), but never delivered.
	//
	// The gap the rule above cannot close by itself: an order cancelled from
	// `on_the_way` never carries the word "delivered", so nothing in the data
	// proves the food left the building. Legitimate every day — the guest is not
	// home, the courier cannot find the address — and therefore never billed.
	// A restaurant where it is most of the traffic is one to phone, which is a
	// judgement a person makes, not a rule a cron job enforces.
	CancelledCooked int `bson:"cancelledCooked" json:"cancelledCooked"`
	// So'm taken through the platform that day, for the tenant's own curve.
	Revenue int `bson:"revenue" json:"revenue"`
	// What we charge for that day: Orders × the tenant's price at the time.
	Billable int `bson:"billable" json:"billable"`

	// ---- The counter ----
	//
	// ⚠️ **Recorded, never billed.** Everything above is priced per order; the
	// till is a monthly subscription, and a per-order fee on top of it charges
	// the same sale twice (see the note in aggregate's pipeline). These four
	// exist because "not billed" had silently become "not shown": a restaurant
	// doing its whole trade in the dining room appeared on the console as a
	// customer with no sales, which is the opposite of the truth and the worst
	// possible number to plan a phone call around.
	//
	// ⚠️ Keyed by the day the check **closed**, not the day it opened — see
	// tillByDay. A table that sits down at 23:40 pays on the next date, and
	// the money has to land on the day the drawer holds it.
	//
	// Absent on every row written before this field existed, which reads as 0.
	// Correct enough: the rolling window recollects recent days, and nothing
	// can recover a count that was never taken.
	TillChecks int `bson:"tillChecks,omitempty" json:"tillChecks"`
	// Guests seated across those checks. The number that turns takings into
	// an average bill, which is the figure an owner actually compares.
	TillGuests int `bson:"tillGuests,omitempty" json:"tillGuests"`
	// So'm actually taken at the counter: paid checks only. A check handed
	// over on credit closes as delivered and **unpaid**, and counting it here
	// would book money nobody has.
	TillRevenue int `bson:"tillRevenue,omitempty" json:"tillRevenue"`
	// Refunded back out afterwards. Its own column rather than subtracted
	// from TillRevenue: nine million sold with two refunded is a different
	// day from seven million sold, and only one of them is worth a call.
	TillRefunded int `bson:"tillRefunded,omitempty" json:"tillRefunded"`

	CollectedAt time.Time `bson:"collectedAt" json:"collectedAt"`
}

// How the money reached us.
//
// **Cash is not a fallback here, it is the MVP.** Keel starts before the MChJ
// is registered, and without a legal entity there is no contract, no invoice
// with a stamp on it, and no bank account to receive a transfer into. So the
// subscription is collected in cash, by hand, and the ledger has to record that
// honestly rather than pretend a bank was involved.
//
// `transfer` exists from day one anyway, unused, because the alternative is
// adding it later and discovering that every stored row has to be re-read to
// mean anything.
const (
	PayCash     = "cash"
	PayTransfer = "transfer"
)

// Invoice states. Deliberately three: an invoice is owed, settled, or was
// issued in error. "Partly paid" is not a state — see Invoice.Paid.
const (
	InvoiceOpen = "open"
	InvoicePaid = "paid"
	InvoiceVoid = "void"
)

// Invoice is one billing period, frozen, and what was collected against it.
//
// Two properties matter more than the fields:
//
//   - **The amount is frozen when the invoice is issued.** The daily rows go on
//     accruing, so an invoice that recomputed itself would change after the
//     customer agreed to it — and the number they were told would be gone. The
//     tenant's own numbers stay live in the dashboard; this is the copy that
//     was quoted.
//
//   - **A payment is a record, not a counter.** Who handed over the money, who
//     took it, on what day, and against which period. Cash with no name on it
//     becomes an argument three weeks later — the same reason the couriers'
//     cash settlements and the staff salary payments are written down rather
//     than decremented from a running total.
//
// Issuing an invoice does not switch anybody off. `suspended` stays the blunt
// "no money came" switch an operator flips on purpose; a ledger that suspended
// customers by itself would take a restaurant offline over a typo.
type Invoice struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID primitive.ObjectID `bson:"tenantId" json:"tenantId"`
	// Copied so the ledger still reads after a customer is closed and removed
	// from the list — which is exactly when somebody asks what they owed.
	Slug string `bson:"slug" json:"slug"`
	Name string `bson:"name" json:"name"`

	// What a human says on the phone: "KEEL-2026-08-0007".
	Number string `bson:"number" json:"number"`

	// The period, half-open [From, To) — the same convention as everywhere
	// else, so the day one invoice ends is the day the next begins and no
	// order is ever billed twice.
	From string `bson:"from" json:"from"`
	To   string `bson:"to" json:"to"`

	// The period's totals as they stood at issue.
	Orders  int `bson:"orders" json:"orders"`
	Revenue int `bson:"revenue" json:"revenue"`
	// What the customer owes: so'm, already multiplied by their own price.
	Amount int `bson:"amount" json:"amount"`
	// ⚠️ The add-on, kept as its own figure rather than folded into `Amount` alone.
	//
	// It is in the total as well — `Amount` is what they pay — but a bill that is three
	// million larger with nothing saying why is a bill somebody rings about. Zero on every
	// invoice for a restaurant that keeps the badge.
	WatermarkFee int `bson:"watermarkFee,omitempty" json:"watermarkFee,omitempty"`
	// ⚠️ The till subscription, on the **same invoice** and as its own figure.
	//
	// One bill a month, not two: a restaurant that pays for its website by the
	// order and its counter by the month has one supplier and one accountant,
	// and two invoices arriving on the same day is two reconciliations, two
	// payments and two chances for one of them to be missed. Kept separate
	// inside it because "why is this three million more than last month" has to
	// be answerable off the paper — the same reason WatermarkFee is not folded
	// into Amount either.
	TillFee int `bson:"tillFee,omitempty" json:"tillFee,omitempty"`
	// The rung the till fee was charged at, frozen. The customer may move plan
	// next month; the invoice must keep saying what was agreed for this one.
	TillPlan string `bson:"tillPlan,omitempty" json:"tillPlan,omitempty"`

	Status string `bson:"status" json:"status"`
	// Why it was voided. Required, for the same reason cancelling an order is:
	// an invoice that vanished without a sentence is one nobody can explain.
	VoidReason string `bson:"voidReason,omitempty" json:"voidReason,omitempty"`

	// Every payment against this invoice. A list rather than a total because
	// money arrives in pieces — half in cash on Friday, the rest next week —
	// and only the pieces answer "what did we actually receive".
	Paid []InvoicePayment `bson:"paid" json:"paid"`

	IssuedBy  string    `bson:"issuedBy" json:"issuedBy"`
	Note      string    `bson:"note,omitempty" json:"note,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// InvoicePayment is one handover of money.
type InvoicePayment struct {
	Amount int `bson:"amount" json:"amount"`
	// cash | transfer. Cash until the MChJ exists.
	Method string `bson:"method" json:"method"`
	// The Keel person who took it. Unsigned cash is how a ledger stops being
	// evidence.
	ReceivedBy string    `bson:"receivedBy" json:"receivedBy"`
	At         time.Time `bson:"at" json:"at"`
	Note       string    `bson:"note,omitempty" json:"note,omitempty"`
}

// Collected is the sum actually received against this invoice.
func (i Invoice) Collected() int {
	n := 0
	for _, p := range i.Paid {
		n += p.Amount
	}
	return n
}

// Outstanding is what is still owed, never negative: an overpayment is a
// conversation, not a negative debt on a list.
func (i Invoice) Outstanding() int {
	if i.Status == InvoiceVoid {
		return 0
	}
	if d := i.Amount - i.Collected(); d > 0 {
		return d
	}
	return 0
}

// CollectorDocID is the fixed key of the single collector-run document.
const CollectorDocID = "collector"

// CollectorRun is what the nightly aggregate did last time it ran.
//
// It exists because of a question that could not be answered from the screen:
// the overview showed an empty 30-day chart, an empty top-customers list and
// zeroes everywhere, and there was **no way to tell "nobody has ordered yet"
// from "the collector has never successfully run"**. Both look identical, and
// only one of them is somebody's problem.
//
// That is the same class of silent failure as a container running last month's
// image while reporting itself healthy: every indicator is calm and the thing
// is simply not happening. So the collector now says when it last ran, how many
// tenant databases it reached, and which ones it could not.
type CollectorRun struct {
	At time.Time `bson:"at" json:"at"`
	// How long it took. A collection that has started taking minutes is the
	// early warning that tenant databases are growing past this design.
	DurationMs int64 `bson:"durationMs" json:"durationMs"`
	// Tenants attempted, reached, and failed.
	Tenants int `bson:"tenants" json:"tenants"`
	OK      int `bson:"ok" json:"ok"`
	Failed  int `bson:"failed" json:"failed"`
	// Day-rows written. Zero with tenants > 0 is not an error — it is a
	// platform where nobody has ordered in the window — but it is the number
	// that makes an empty chart explainable rather than suspicious.
	Rows int `bson:"rows" json:"rows"`
	// Up to a handful of failures, verbatim. The useful sentence is almost
	// never ours: it is a database refusing a connection.
	Errors []string `bson:"errors,omitempty" json:"errors,omitempty"`
	// "schedule" | "manual" — an operator pressing "collect now" while
	// looking at an empty chart is the common case.
	Trigger string `bson:"trigger" json:"trigger"`
}

// RolloutDocID is the fixed key of the single rollout document.
//
// One document rather than one per run: what an operator needs is "what is
// happening now, or what happened last time". A history of rollouts would be a
// list nobody reads, and every run is already visible on each tenant's own
// provisionedAt.
const RolloutDocID = "current"

// Rollout is one pass of moving every live tenant onto the current image.
//
// Stored rather than kept in memory so the console can show progress from any
// browser, and so a finished run still explains itself the next morning —
// including the tenants it could not update, which is the half worth reading.
type Rollout struct {
	StartedAt  time.Time `bson:"startedAt" json:"startedAt"`
	FinishedAt time.Time `bson:"finishedAt,omitempty" json:"finishedAt,omitempty"`
	StartedBy  string    `bson:"startedBy" json:"startedBy"`

	// The tag rolled out, and the image id it named at the time. The id is
	// what matters: the tag will point elsewhere after the next deploy, and
	// then this record would claim a rollout that never happened.
	Image   string `bson:"image" json:"image"`
	ImageID string `bson:"imageId" json:"imageId"`

	// "running" | "done" | "aborted" | "failed"
	Status string `bson:"status" json:"status"`
	// Why it stopped, when that is not obvious. Shown as-is to the operator.
	Note string `bson:"note,omitempty" json:"note,omitempty"`
	// The tenant being worked on right now, so a stuck rollout names the
	// customer holding it up rather than a percentage.
	Current string `bson:"current,omitempty" json:"current,omitempty"`

	Total   int `bson:"total" json:"total"`
	Done    int `bson:"done" json:"done"`
	Updated int `bson:"updated" json:"updated"`
	Failed  int `bson:"failed" json:"failed"`

	Items []RolloutItem `bson:"items" json:"items"`
}

// RolloutItem is what happened to one tenant.
type RolloutItem struct {
	Slug string `bson:"slug" json:"slug"`
	Name string `bson:"name" json:"name"`
	// "updated" | "current" | "skipped" | "failed"
	Status string    `bson:"status" json:"status"`
	Note   string    `bson:"note,omitempty" json:"note,omitempty"`
	At     time.Time `bson:"at" json:"at"`
}

// User is a Keel employee who can open the dashboard. Deliberately minimal:
// this is us, not the customers.
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username     string             `bson:"username" json:"username"`
	PasswordHash string             `bson:"passwordHash" json:"-"`
	Name         string             `bson:"name" json:"name"`
	// owner | admin | manager | agent. ⚠️ An empty role reads as **owner**, because
	// the only account that predates this field is the one seeded at first boot —
	// treating it as an agent would lock the platform's owner out of their own
	// console on the deploy that introduced roles.
	Role string `bson:"role,omitempty" json:"role"`
	// Every role this account holds; permissions are their union. ⚠️ Absent on
	// accounts created before an account could hold several — `RolesOf` falls back
	// to `Role`, which is still written (as the widest role) for that reason.
	Roles       []string   `bson:"roles,omitempty" json:"roles,omitempty"`
	Phone       string     `bson:"phone,omitempty" json:"phone,omitempty"`
	IsActive    *bool      `bson:"isActive,omitempty" json:"isActive,omitempty"`
	CreatedBy   string     `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	CreatedAt   time.Time  `bson:"createdAt" json:"createdAt"`
	LastLoginAt *time.Time `bson:"lastLoginAt,omitempty" json:"lastLoginAt,omitempty"`
}

// Active reports whether this account may sign in. Absent means yes: every account
// created before the flag existed is a working account.
func (u User) Active() bool { return u.IsActive == nil || *u.IsActive }

// BriefingLog is one morning briefing we bought for one tenant.
//
// ⚠️ **Tokens, not a price.** A rate hard-coded today is a number that quietly
// stops being right, and the arithmetic is better done where whoever does it
// can see which rate they used.
type BriefingLog struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Slug         string             `bson:"slug" json:"slug"`
	At           time.Time          `bson:"at" json:"at"`
	InputTokens  int64              `bson:"inputTokens" json:"inputTokens"`
	CachedTokens int64              `bson:"cachedTokens" json:"cachedTokens"`
	OutputTokens int64              `bson:"outputTokens" json:"outputTokens"`
}

// AdsLog is one campaign plan we bought for one tenant.
//
// ⚠️ **Its own collection, not a kind on BriefingLog.** The advertising add-on
// is sold with its own daily number on it, and the assistant's usage screen
// counts rows — one table would make both numbers wrong, the cap in one
// direction and the cost report in the other.
type AdsLog struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Slug         string             `bson:"slug" json:"slug"`
	At           time.Time          `bson:"at" json:"at"`
	InputTokens  int64              `bson:"inputTokens" json:"inputTokens"`
	CachedTokens int64              `bson:"cachedTokens" json:"cachedTokens"`
	OutputTokens int64              `bson:"outputTokens" json:"outputTokens"`
}
