package handlers

import (
	"testing"
	"time"

	"keel-control/internal/models"
)

// The volume ladder.
//
// It exists because of what a flat rate does to the best customer: at 400
// orders a day the bill is 12 million so'm a month, which is roughly a
// developer's salary here, and that is where a chain's finance person stops
// reading the invoice and starts doing arithmetic. Tiers keep the marginal
// rate positive while the average falls, so growth still pays us and the
// conversation changes shape.
//
// Everything below is money, so all of it is pinned.

var ladder = []models.PriceTier{
	{UpTo: 3000, Price: 1000},
	{UpTo: 10000, Price: 700},
	{UpTo: 0, Price: 500}, // no limit
}

func TestLadderChargesEachBandAtItsOwnRate(t *testing.T) {
	cases := map[int]int{
		0:     0,
		1:     1000,
		3000:  3_000_000,                        // exactly the first band
		3001:  3_000_000 + 700,                  // one order into the second
		6000:  3_000_000 + 3000*700,             // mid second band
		10000: 3_000_000 + 7000*700,             // exactly the second
		12000: 3_000_000 + 7000*700 + 2000*500,  // the 400/day case
		24000: 3_000_000 + 7000*700 + 14000*500, // 800/day
	}
	for orders, want := range cases {
		if got := models.PriceForOrders(orders, ladder, 1000); got != want {
			t.Errorf("%d orders → %d, want %d", orders, got, want)
		}
	}
}

// The headline number from the conversation that produced this: 400 a day used
// to cost 12 mln and now costs 8.9 mln, an average of 742 so'm an order.
func TestFourHundredADayCase(t *testing.T) {
	const orders = 400 * 30
	got := models.PriceForOrders(orders, ladder, 1000)
	if got != 8_900_000 {
		t.Fatalf("400/day → %d, want 8 900 000", got)
	}
	if avg := got / orders; avg != 741 && avg != 742 {
		t.Errorf("average %d so'm/order, want ~742", avg)
	}
	// The marginal rate stays positive: a cap would make the next order free,
	// which is the wrong incentive on both sides.
	next := models.PriceForOrders(orders+1, ladder, 1000)
	if next-got != 500 {
		t.Errorf("marginal order costs %d, want 500", next-got)
	}
}

// A small customer must not notice the ladder exists.
func TestSmallCustomersAreUnaffected(t *testing.T) {
	for _, perDay := range []int{10, 20, 50, 100} {
		orders := perDay * 30
		if got := models.PriceForOrders(orders, ladder, 1000); got != orders*1000 {
			t.Errorf("%d/day → %d, want the flat %d", perDay, got, orders*1000)
		}
	}
}

// No tiers means the old behaviour, unchanged. Every tenant created before
// this existed relies on it.
func TestNoTiersIsFlat(t *testing.T) {
	if got := models.PriceForOrders(500, nil, 1000); got != 500_000 {
		t.Fatalf("got %d, want the flat rate", got)
	}
}

// A tenant's own ladder beats the platform default — the same rule
// pricePerOrder follows, so a negotiated deal survives a platform-wide change.
func TestTenantLadderBeatsTheDefault(t *testing.T) {
	now := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.Local)
	own := models.Tenant{
		PricePerOrder: 1000,
		PriceTiers:    []models.PriceTier{{UpTo: 0, Price: 400}},
	}
	if got := own.ChargeForOrders(12000, ladder, 0, now); got != 12000*400 {
		t.Fatalf("got %d, want the tenant's own rate", got)
	}
	// And a tenant with none of its own picks up the platform's.
	plain := models.Tenant{PricePerOrder: 1000}
	if got := plain.ChargeForOrders(12000, ladder, 0, now); got != 8_900_000 {
		t.Fatalf("got %d, want the platform ladder", got)
	}
}

// Free and the discount still apply on top, in that order.
func TestFreeAndDiscountApplyAfterTheLadder(t *testing.T) {
	now := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.Local)

	free := models.Tenant{PricePerOrder: 1000, Free: true, FreeReason: "anchor"}
	if got := free.ChargeForOrders(12000, ladder, 0, now); got != 0 {
		t.Errorf("free customer charged %d", got)
	}
	disc := models.Tenant{PricePerOrder: 1000, DiscountPercent: 10}
	if got := disc.ChargeForOrders(12000, ladder, 0, now); got != 8_010_000 {
		t.Errorf("got %d, want 10%% off 8 900 000", got)
	}
}

// A malformed ladder must not bill orders at zero. Free is a decision, never a
// gap in a table.
func TestOrdersPastTheLastBandAreStillBilled(t *testing.T) {
	short := []models.PriceTier{{UpTo: 100, Price: 900}}
	got := models.PriceForOrders(300, short, 1000)
	if got != 100*900+200*900 {
		t.Fatalf("got %d, want the last rate applied to the remainder", got)
	}
}

// The floor, and the four ways it must not misfire.
//
// The ladder bends the top of the curve down so the largest customer's invoice
// does not invite a negotiation. This is the other end: a restaurant doing five
// orders a day bills ~150 000 so'm a month and needs exactly as much support as
// one doing four hundred. Every rule below is money, so every rule is pinned.
func TestMinimumMonthly(t *testing.T) {
	now := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.Local)
	const floor = 300_000
	small := models.Tenant{PricePerOrder: 1000}

	// 150 orders → 150 000 by the ladder, lifted to the floor.
	if got := small.ChargeForOrders(150, ladder, floor, now); got != floor {
		t.Errorf("below the floor: got %d, want %d", got, floor)
	}
	// A customer already above it is untouched — the floor is a minimum, not a
	// surcharge.
	if got := small.ChargeForOrders(1000, ladder, floor, now); got != 1_000_000 {
		t.Errorf("above the floor: got %d, want 1 000 000", got)
	}

	// ⚠️ A period with no orders is never floored. Zero orders means the site
	// was not live or the restaurant was closed; an invoice for a month the
	// customer did not use is the fastest way to lose one.
	if got := small.ChargeForOrders(0, ladder, floor, now); got != 0 {
		t.Errorf("unused period charged %d, want 0", got)
	}

	// Free wins over the floor. An account promised free terms is free, and a
	// minimum that survived that promise would be the one billing mistake that
	// costs a customer rather than money.
	free := models.Tenant{PricePerOrder: 1000, Free: true, FreeReason: "anchor"}
	if got := free.ChargeForOrders(150, ladder, floor, now); got != 0 {
		t.Errorf("free customer charged %d", got)
	}

	// The discount applies to the floor, not underneath it: a negotiated
	// percentage that could not move the minimum would be a discount that does
	// nothing for exactly the customers small enough to have asked for one.
	disc := models.Tenant{PricePerOrder: 1000, DiscountPercent: 20}
	if got := disc.ChargeForOrders(150, ladder, floor, now); got != 240_000 {
		t.Errorf("discounted floor: got %d, want 240 000", got)
	}

	// Unconfigured means off. A floor changes what real customers owe, so it
	// must never arrive as the side effect of a deploy.
	if got := small.ChargeForOrders(150, ladder, 0, now); got != 150_000 {
		t.Errorf("no floor configured: got %d, want 150 000", got)
	}
}

// A tenant's own floor beats the platform's, in both directions.
//
// Same rule as the price and the tiers: terms agreed before a floor existed
// survive one being introduced, and raising the platform default must not
// silently reprice everybody who already said yes to something else.
func TestTenantMinimumOverridesPlatform(t *testing.T) {
	now := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.Local)
	own := models.Tenant{PricePerOrder: 1000, MinMonthly: 500_000}
	if got := own.ChargeForOrders(150, ladder, 300_000, now); got != 500_000 {
		t.Errorf("own floor: got %d, want 500 000", got)
	}
	// And 0 on the tenant means "use the platform's", not "no floor" — the
	// field is absent on every row written before it existed.
	plain := models.Tenant{PricePerOrder: 1000}
	if got := plain.Minimum(300_000); got != 300_000 {
		t.Errorf("fallback: got %d, want the platform floor", got)
	}
}
