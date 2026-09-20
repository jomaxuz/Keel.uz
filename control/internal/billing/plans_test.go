package billing

import "testing"

// The defect the whole ladder was redrawn to remove: at 1 600 000 the third
// register cost more than the first, so a restaurant with a queue would not buy
// the terminal that clears it. Pinned as arithmetic because it is invisible in
// the price list — every rung there looks like a discount.
func TestRegisterPriceNeverRises(t *testing.T) {
	type rung struct {
		plan      string
		registers int
	}
	// The cheapest plan that covers each register count, walked upwards.
	steps := []rung{
		{PlanStart, 1}, {PlanStandard, 2}, {PlanPro, 3}, {PlanPro, 4}, {PlanPro, 5},
	}
	prevAvg, prevTotal := 1<<30, 0
	for _, s := range steps {
		p, ok := PlanByID(s.plan)
		if !ok {
			t.Fatalf("plan %q missing", s.plan)
		}
		total := TillMonthly(p, 1, nil, 0, 0)
		avg := total / s.registers
		if avg > prevAvg {
			t.Errorf("%s at %d registers: %d per register, up from %d — "+
				"a bigger restaurant must never pay more per till",
				s.plan, s.registers, avg, prevAvg)
		}
		if marginal := total - prevTotal; marginal < 0 {
			t.Errorf("%s: marginal price %d is negative", s.plan, marginal)
		}
		prevAvg, prevTotal = avg, total
	}
}

// The other half of the same rule, and the one the original draft got backwards:
// a chain must never pay less in total than a single restaurant.
func TestMoreBranchesCostMore(t *testing.T) {
	p, _ := PlanByID(PlanStart)
	prev := 0
	for n := 1; n <= 12; n++ {
		got := TillMonthly(p, n, nil, 0, 0)
		if got <= prev {
			t.Fatalf("%d branches costs %d, not more than %d for %d",
				n, got, prev, n-1)
		}
		// And the average still falls, which is the sentence we sell.
		if n > 1 && got/n > prev/(n-1) {
			t.Errorf("%d branches: %d per branch, up from %d",
				n, got/n, prev/(n-1))
		}
		prev = got
	}
}

// The commercial decision worth protecting from a later tidy-up: stock is
// buyable on Start. Folding it into Pro puts a 800 000 so'm cliff in front of
// the café that wants to know its food cost, which is the cliff that sends
// those customers to a competitor.
func TestStockIsBuyableOnTheCheapestPlan(t *testing.T) {
	start, _ := PlanByID(PlanStart)
	if start.Includes(ModStock) {
		t.Fatal("Start should not include stock — it is the add-on that pays for itself")
	}
	if AddonPrice(ModStock) <= 0 {
		t.Fatal("stock must have a standalone price, or Start cannot buy it")
	}
	with := TillMonthly(start, 1, []string{ModStock}, 0, 0)
	pro, _ := PlanByID(PlanPro)
	if with >= TillMonthly(pro, 1, nil, 0, 0) {
		t.Errorf("Start+stock (%d) costs at least as much as Pro (%d) — "+
			"then the add-on buys nothing", with, TillMonthly(pro, 1, nil, 0, 0))
	}
}

// A plan that includes a module must not also charge for it. The path is easy
// to miss because it only runs for a customer who bought the add-on *first* and
// moved up later — which is the customer most likely to be reading the invoice.
func TestUpgradeStopsBillingTheAddon(t *testing.T) {
	pro, _ := PlanByID(PlanPro)
	if a, b := TillMonthly(pro, 1, []string{ModStock}, 0, 0), TillMonthly(pro, 1, nil, 0, 0); a != b {
		t.Errorf("Pro billed %d with the stock add-on and %d without; Pro includes it", a, b)
	}
}

// Enterprise is negotiated. A number invented here would reach an invoice
// looking like something that had been agreed with the customer.
func TestEnterpriseHasNoAutomaticPrice(t *testing.T) {
	e, _ := PlanByID(PlanEnterprise)
	if !e.Individual {
		t.Fatal("Enterprise must be marked individual")
	}
	if got := TillMonthly(e, 3, []string{ModStock}, 0, 0); got != 0 {
		t.Errorf("TillMonthly(Enterprise, 0) = %d, want 0 — the price is agreed, not computed", got)
	}
}

// An id we do not recognise must not resolve to the cheapest rung: that
// silently takes a paying customer's modules away and refuses their registers.
func TestUnknownPlanIsNotStart(t *testing.T) {
	for _, id := range []string{"", "  ", "basic", "Start ", "STARTER"} {
		if p, ok := PlanByID(id); ok && p.ID == PlanStart && id != "Start " {
			t.Errorf("PlanByID(%q) resolved to Start", id)
		}
	}
	// Case and padding are tolerated, because they come from a form, not a bug.
	if p, ok := PlanByID(" PRO "); !ok || p.ID != PlanPro {
		t.Errorf("PlanByID(\" PRO \") = %v, %v", p, ok)
	}
}

// ⚠️ **Buying from us must never take a screen away.**
//
// Analysis and the customer base were rungs on this ladder once. The defect
// only appears when you follow one customer through: a restaurant paying per
// order for the website has no subscription, so the module gate lets everything
// through and it uses the reports and the call centre for months. The day it
// buys the cheapest till a subscription appears — and with those as modules,
// its empty list closes screens the restaurant already had.
//
// Pinned as a property of the whole ladder rather than as a list of ids: the
// cheapest rung must not be missing anything a customer without any plan can
// already use. Anything genuinely sold has to be a *counter* capability.
func TestCheapestPlanWithholdsNothingAWebsiteCustomerAlreadyHas(t *testing.T) {
	// What a customer with no till subscription can reach. Mirrors the tenant
	// server's module gate, which lets everything through when nothing is sold.
	ungated := []string{"reports", "crm"}
	for _, p := range Plans() {
		for _, mod := range p.Modules {
			for _, u := range ungated {
				if mod == u {
					t.Errorf("plan %q sells %q, which every website customer "+
						"already has — buying a till would remove it", p.ID, mod)
				}
			}
		}
	}
	// And the module list that *is* sold stays about the counter.
	start, _ := PlanByID(PlanStart)
	if len(start.Modules) != 0 {
		t.Errorf("Start carries modules %v; the cheapest rung should differ by "+
			"scale, not by withheld software", start.Modules)
	}
}

// ⚠️ **The assistant blocks were sold on one screen and billed from another.**
// The console priced them into the figure an operator agreed with a customer;
// this function — which the invoice is issued from and which is mirrored into
// the restaurant's own panel — left them out. So the quote said 1 800 000, the
// bill said 1 500 000, and the owner's app showed the second number. The
// pricing helper existed and was called from nowhere at all.
func TestBoughtAssistantBlocksAreInTheMonthlyPrice(t *testing.T) {
	p, _ := PlanByID(PlanStandard)
	base := TillMonthly(p, 1, nil, 0, 0)
	with := TillMonthly(p, 1, nil, 3, 0)
	if with != base+3*AIExtraMonthly {
		t.Fatalf("three blocks cost %d, want %d", with-base, 3*AIExtraMonthly)
	}
	// A rung that includes the assistant still pays for extra allowance: the
	// blocks are more of it, not the thing itself.
	pro, _ := PlanByID(PlanPro)
	if TillMonthly(pro, 1, nil, 1, 0) != TillMonthly(pro, 1, nil, 0, 0)+AIExtraMonthly {
		t.Fatal("a Pro customer's extra allowance was given away")
	}
}

// ⚠️ **The televisions are priced per screen, and nothing on the ladder covers
// them.** Every other line here is priced by the branch — one kitchen, one
// stockroom, one manager — and a screen is the one thing a restaurant buys more
// of inside the same room. Charging four like one gives away the case the
// module exists for; charging them by branch bills a chain for televisions it
// does not have.
func TestScreensArePricedOneByOne(t *testing.T) {
	pro, _ := PlanByID(PlanPro)
	base := TillMonthly(pro, 1, nil, 0, 0)

	if got := TillMonthly(pro, 1, nil, 0, 4); got != base+4*TVScreenMonthly {
		t.Errorf("four screens = %d, want %d", got, base+4*TVScreenMonthly)
	}
	// ⚠️ No rung includes them, so the top of the ladder pays for its screens
	// exactly like the bottom. A plan that quietly covered them would be a plan
	// whose invoice nobody can check by eye.
	start, _ := PlanByID(PlanStart)
	startBase := TillMonthly(start, 1, nil, 0, 0)
	if got := TillMonthly(start, 1, nil, 0, 4); got != startBase+4*TVScreenMonthly {
		t.Errorf("screens on Start = %d, want %d", got, startBase+4*TVScreenMonthly)
	}

	// ⚠️ **Not multiplied by branches.** The screens are counted, not derived:
	// a three-branch chain with one television pays for one television.
	if got := TillMonthly(pro, 3, nil, 0, 1); got != TillMonthly(pro, 3, nil, 0, 0)+TVScreenMonthly {
		t.Error("screens are being charged per branch")
	}

	if TVMonthlyFor(0) != 0 || TVMonthlyFor(-2) != 0 {
		t.Error("no screens must cost nothing, and a negative is a typo")
	}
}

// ---- The shop ladder ----

// The same arithmetic the restaurant ladder is pinned by, and it matters more
// here: a shop buys a second register to clear a queue at the door, and a rung
// that makes the second one dearer than the first is a rung that keeps the
// queue.
func TestShopRegisterPriceNeverRises(t *testing.T) {
	steps := []struct {
		plan      string
		registers int
	}{
		{PlanShopStart, 1}, {PlanShopStandard, 2},
		{PlanShopPro, 3}, {PlanShopPro, 4}, {PlanShopPro, 5},
	}
	prevAvg, prevTotal := 1<<30, 0
	for _, s := range steps {
		p, ok := PlanByID(s.plan)
		if !ok {
			t.Fatalf("plan %q missing", s.plan)
		}
		total := TillMonthly(p, 1, nil, 0, 0)
		if avg := total / s.registers; avg > prevAvg {
			t.Errorf("%s at %d registers: %d each, up from %d",
				s.plan, s.registers, avg, prevAvg)
		} else {
			prevAvg = avg
		}
		if marginal := total - prevTotal; marginal < 0 {
			t.Errorf("%s: marginal price %d is negative", s.plan, marginal)
		}
		prevTotal = total
	}
}

// ⚠️ **A shop that cannot open its balances cannot do the thing it bought
// this for.** Stock is an add-on on the restaurant ladder because a café can
// genuinely want a till and not want its food cost; there is no such shop. Sold
// separately here it would be a till that cannot count what it sells.
func TestEveryShopRungIncludesTheStockroom(t *testing.T) {
	for _, p := range PlansFor("grocery") {
		if !p.Includes(ModStock) {
			t.Errorf("shop plan %q does not include the stockroom", p.ID)
		}
	}
}

// ⚠️ **The cheap ladder must stay behind the business type.** The shop rungs
// are a third of the restaurant ones; a restaurant reaching them — through the
// console's list or through a hand-written request — pays 149 000 for what was
// agreed at 450 000, and every screen downstream agrees with the wrong number.
func TestTheLaddersDoNotCross(t *testing.T) {
	for _, biz := range []string{"", "fastfood", "restoran", "nonsense"} {
		for _, p := range PlansFor(biz) {
			if p.Kind == KindShop {
				t.Errorf("business %q is offered shop plan %q", biz, p.ID)
			}
		}
		shop, _ := PlanByID(PlanShopStart)
		if PlanFitsBusiness(shop, biz) {
			t.Errorf("business %q may be put on %q", biz, shop.ID)
		}
	}
	for _, biz := range []string{
		"grocery", "butcher", "clothing", "cosmetics", "flowers", "pharmacy",
		"hardware", "bakery", "coffee", "pastry",
	} {
		list := PlansFor(biz)
		if len(list) == 0 {
			t.Fatalf("business %q is offered no plans at all", biz)
		}
		for _, p := range list {
			if p.Kind != KindShop {
				t.Errorf("shop %q is offered restaurant plan %q", biz, p.ID)
			}
		}
		start, _ := PlanByID(PlanStart)
		if PlanFitsBusiness(start, biz) {
			t.Errorf("shop %q may be put on the restaurant ladder", biz)
		}
	}
}

// ⚠️ **`shopLadder` here is a list that lives in another repository**, because
// the control plane does not import the restaurant's packages. The compiler
// cannot join them, so this does: the list below is every business type in
// `models.BusinessTypes`, and a type added there and forgotten here is a
// customer quietly billed on the wrong ladder — three times the agreed price,
// or a third of it, with an invoice that agrees with itself either way.
func TestTheBusinessTypesMatchTheTenants(t *testing.T) {
	cheap := []string{
		// Shops: what was delivered is what is sold.
		"grocery", "butcher", "clothing", "cosmetics",
		"flowers", "pharmacy", "hardware",
		// Makers: they compose everything they sell and are still counters.
		"bakery", "coffee", "pastry",
		// A shop with no room: it buys the catalogue half of the product and
		// none of the dining room half the full ladder is priced for.
		"ecommerce",
	}
	full := []string{"", "fastfood"}
	for _, b := range cheap {
		if !shopLadder(b) {
			t.Errorf("%q is a counter on the tenant and a restaurant here", b)
		}
	}
	for _, b := range full {
		if shopLadder(b) {
			t.Errorf("%q is a restaurant on the tenant and a counter here", b)
		}
	}
	// Padding and case come from a form, not from a bug.
	if !shopLadder(" Grocery ") {
		t.Error("a padded business type stopped being a shop")
	}
}

// The number the whole ladder exists to hit: our entry price has to sit in the
// band a shop actually shops in (REGOS 149 000, YesPOS 100 000, BILLZ 299 000).
// Pinned because it is the one figure a later tidy-up would "round up".
func TestTheShopEntryPriceStaysInTheMarket(t *testing.T) {
	start, _ := PlanByID(PlanShopStart)
	if start.Monthly > 199_000 {
		t.Errorf("shop entry is %d — above the band a shop compares in, "+
			"which does not make us expensive, it makes us unconsidered",
			start.Monthly)
	}
}
