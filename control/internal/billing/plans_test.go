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
		total := TillMonthly(p, 1, nil)
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
		got := TillMonthly(p, n, nil)
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
	with := TillMonthly(start, 1, []string{ModStock})
	pro, _ := PlanByID(PlanPro)
	if with >= TillMonthly(pro, 1, nil) {
		t.Errorf("Start+stock (%d) costs at least as much as Pro (%d) — "+
			"then the add-on buys nothing", with, TillMonthly(pro, 1, nil))
	}
}

// A plan that includes a module must not also charge for it. The path is easy
// to miss because it only runs for a customer who bought the add-on *first* and
// moved up later — which is the customer most likely to be reading the invoice.
func TestUpgradeStopsBillingTheAddon(t *testing.T) {
	pro, _ := PlanByID(PlanPro)
	if a, b := TillMonthly(pro, 1, []string{ModStock}), TillMonthly(pro, 1, nil); a != b {
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
	if got := TillMonthly(e, 3, []string{ModStock}); got != 0 {
		t.Errorf("TillMonthly(Enterprise) = %d, want 0 — the price is agreed, not computed", got)
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
