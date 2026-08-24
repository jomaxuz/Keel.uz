package billing

// ---- The till subscription ----
//
// The website is billed per order (see the price ladder on the tenant). The
// till is not, and the difference is not a preference: a restaurant's counter
// takes money whether or not anybody ordered online, so a per-order price would
// bill a busy dining room nothing and a quiet delivery kitchen everything.
//
// ⚠️ **The unit is the branch, not the register.** Five terminals in one dining
// room are one kitchen, one printer setup, one stockroom and one manager; five
// terminals in five branches are five of each. Pricing the second case like the
// first hands a ten-branch chain a bill smaller than a single café's — which is
// exactly backwards, and it is the shape most of this file exists to prevent.
// Registers are an axis *inside* a branch, and they are capped rather than
// charged for.
//
// ⚠️ **The marginal price of a register never rises.** This is the one rule a
// ladder like this gets wrong, and it is worth stating as arithmetic rather
// than as intent: at 1 600 000 the third register cost a restaurant 750 000 on
// its own, so the per-register price went 450 000 → 425 000 → **533 000**, and
// a three-register restaurant paid more per register than a one-register one.
// The visible result is not a complaint about the invoice: it is a restaurant
// with a queue that does not buy the terminal that would clear it, and then
// blames us for a slow counter. Pro at 1 250 000 keeps the marginal price flat
// at 400 000 and lets the average fall the whole way down the ladder.

import (
	"strings"
	"time"
)

// Module ids. Stable strings, because they are stored on the tenant and read by
// the restaurant's own server: renaming one silently returns a paid module to
// everybody who bought it, or takes it away from everybody who did.
const (
	// Cost price, technical cards, stocktakes, supplier debt, ingredient ABC,
	// the shopping list.
	ModStock = "stock"
	// More than one brand or branch, and the switcher that moves between them.
	ModMultiBranch = "multibranch"
	// iiko / Syrve / Poster / Clopos / r_keeper.
	//
	// ⚠️ **Granted and no longer enforced.** Nothing gates on it any more (see
	// handlers/modulegate.go in the tenant): for a restaurant that never buys a
	// Keel till, pushing its online orders into the register it already runs is
	// the entire reason our website is worth having, and selling that back as an
	// upgrade priced the product's own value out of the plan most likely to need
	// it. Worse, it took the integration away from a website customer on the day
	// they bought a Start till.
	//
	// ⚠️ **The id stays, and it stays in the plans below.** Removing a stored
	// string is how a module silently returns to everybody or vanishes from
	// everybody, and tenants already hold this one. It costs nothing to keep
	// granting something nobody checks; it would cost a migration to stop.
	ModPOSIntegration = "posint"
	// Franchise management.
	ModFranchise = "franchise"
)

// ⚠️ **Analysis and the customer base are not modules, and that is deliberate.**
//
// They were rungs on this ladder once — reports at Standard, the CRM at Pro —
// and the defect only shows when you follow one customer through. A restaurant
// paying per order for the website has no subscription at all, so it uses the
// analytics and the call centre for months. The day it buys a Start till a
// subscription appears with an empty module list and those screens close:
// paying us more takes features away. That is the third-register cliff again in
// different clothes, and this time the customer would be right to call it a
// bait and switch. So the axes are scale — registers and branches — plus the
// modules that genuinely belong to a counter.

// Plan ids.
const (
	PlanStart      = "start"
	PlanStandard   = "standard"
	PlanPro        = "pro"
	PlanEnterprise = "enterprise"
)

// Plan is one rung of the till ladder.
//
// Prices are per **branch** per month, in whole so'm — the currency has no
// subunit anybody uses, and a plan priced in tiyin would be a plan whose
// arithmetic nobody can check by eye.
type Plan struct {
	ID string
	// Monthly price of the first branch. Later branches are discounted; see
	// BranchPrice.
	Monthly int
	// How many till screens one branch may bind. 0 means no cap.
	//
	// ⚠️ A cap, not a price. A restaurant that grows into the next rung should
	// meet a sentence and a button, never a register that stops selling in the
	// middle of a Friday — see SoftLimit in the handler.
	Registers int
	// What this plan includes without buying an add-on.
	Modules []string
	// Whether the price is negotiated per customer rather than read from here.
	Individual bool
}

// plans is the ladder, cheapest first. Order matters: it is the order the
// console draws, and the order Upgrade walks.
var plans = []Plan{
	{
		ID: PlanStart, Monthly: 450_000, Registers: 1,
		// ⚠️ Deliberately empty, and deliberately not empty of *everything*:
		// what a Start till can do is defined by the never-restricted list
		// (fiscal receipts, X/Z, roles and PINs, the printer, the drawer, data
		// export, the manual stop list), which is not expressed here because it
		// is not for sale. This field holds only what a plan *adds*.
		Modules: nil,
	},
	{
		// ⚠️ Standard differs from Start by **register count alone**, and that
		// is a feature of the ladder rather than a gap in it: a second till is
		// exactly what a restaurant outgrowing Start needs, and inventing a
		// software difference to justify the rung would mean withholding
		// something from Start for no reason but the price list.
		ID: PlanStandard, Monthly: 850_000, Registers: 2,
		Modules: nil,
	},
	{
		ID: PlanPro, Monthly: 1_250_000, Registers: 5,
		Modules: []string{ModStock, ModMultiBranch, ModPOSIntegration},
	},
	{
		ID: PlanEnterprise, Monthly: 2_500_000, Registers: 0,
		Modules: []string{
			ModStock, ModMultiBranch, ModPOSIntegration, ModFranchise,
		},
		Individual: true,
	},
}

// Plans returns the ladder in display order.
func Plans() []Plan { return append([]Plan(nil), plans...) }

// PlanByID resolves a stored plan id.
//
// ⚠️ **An unknown id is not Start.** A plan string that fell out of a bad
// import or a renamed constant would otherwise silently downgrade a paying
// customer to the cheapest rung — their stock module gone, their second
// register refused — and the first report of it would come from the restaurant.
// The caller is told instead, and every caller here treats "unknown" as "till
// not sold", which fails towards *not billing* rather than towards taking
// something away.
func PlanByID(id string) (Plan, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, p := range plans {
		if p.ID == id {
			return p, true
		}
	}
	return Plan{}, false
}

// Includes reports whether the plan covers a module without an add-on.
func (p Plan) Includes(mod string) bool {
	for _, m := range p.Modules {
		if m == mod {
			return true
		}
	}
	return false
}

// AddonPrice is what a module costs per month when the plan does not include it.
//
// ⚠️ **Stock is an add-on and not a Pro-only feature, and that is the single
// most commercial decision in this file.** A small café that wants to know its
// food cost would otherwise have to jump 450 000 → 1 250 000, and iiko has
// exactly this cliff — it is why those customers end up on Poster. As an add-on
// the same café pays 740 000 and stays.
//
// Zero means the module cannot be bought on its own; it comes with a rung.
func AddonPrice(mod string) int {
	switch mod {
	case ModStock:
		return 290_000
	}
	return 0
}

// Branch discounts. A chain is more work per branch than a single restaurant,
// but not linearly more: the menu, the recipes, the customer base and most
// support questions are answered once for all of them.
const (
	branchDiscountFrom2 = 30 // percent off branches 2..4
	branchDiscountFrom5 = 40 // percent off branch 5 and up
)

// BranchPrice is the monthly price of the nth branch, n counted from 1.
func BranchPrice(monthly, n int) int {
	switch {
	case n <= 1:
		return monthly
	case n < 5:
		return percentOff(monthly, branchDiscountFrom2)
	default:
		return percentOff(monthly, branchDiscountFrom5)
	}
}

// percentOff rounds to the nearest so'm rather than truncating, for the reason
// WatermarkFee does: a figure that is consistently a hair low reads as sloppy
// arithmetic to the person checking it.
func percentOff(v, pct int) int {
	keep := 100 - pct
	return (v*keep + 50) / 100
}

// TillMonthly is the full monthly till price for a customer: every branch at
// its own rung of the branch discount, plus any add-on they bought.
//
// ⚠️ **Add-ons are priced once, not once per branch.** The stock module is one
// catalogue of ingredients and one set of technical cards for the whole company
// — charging a five-branch chain five times for one recipe book would be a
// bill nobody could defend on the phone.
//
// Returns 0 for a plan that is negotiated individually: an Enterprise number
// invented here would appear on an invoice as though it had been agreed.
func TillMonthly(p Plan, branches int, addons []string) int {
	if p.Individual {
		return 0
	}
	if branches < 1 {
		branches = 1
	}
	total := 0
	for n := 1; n <= branches; n++ {
		total += BranchPrice(p.Monthly, n)
	}
	for _, mod := range addons {
		if p.Includes(mod) {
			// Already paid for by the rung. Silently skipped rather than
			// refused: a customer who bought the module and later moved up to
			// Pro should stop being charged for it, not see an error.
			continue
		}
		total += AddonPrice(mod)
	}
	return total
}

// TillFee is what the till subscription costs for one billing period.
//
// The same day-counting as WatermarkFee, and deliberately the same function
// underneath: two prorations that drift apart is how one invoice ends up
// disagreeing with itself about where a month starts.
func TillFee(monthly int, from, to, since time.Time) int {
	return prorate(monthly, from, to, since)
}
