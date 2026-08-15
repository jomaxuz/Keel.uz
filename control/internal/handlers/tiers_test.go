package handlers

import (
	"testing"
	"time"

	"keel-control/internal/models"
)

// The volume ladder.
//
// It exists because of what a flat rate does to the best customer: at 400
// orders a day the bill is a developer's salary here, and that is where a
// chain's finance person stops reading the invoice and starts doing
// arithmetic. Tiers bend the average down so growth still pays us and the
// conversation changes shape.
//
// ⚠️ The bands are set against the competitor's published ladder, which prices
// the **whole** volume at the band rate (1 000 / 700 / 500). Ours are 20% under
// each of those, plus a fourth band that only a chain reaches. Because the
// shape now matches theirs, the comparison a customer makes on the two pricing
// pages is the comparison they get on the invoice.
//
// Everything below is money, so all of it is pinned.

var ladder = []models.PriceTier{
	{UpTo: 3000, Price: 800},
	{UpTo: 15000, Price: 560},
	{UpTo: 50000, Price: 400},
	{UpTo: 0, Price: 300}, // no limit — chains
}

func TestLadderTakesWhicheverBandIsCheapest(t *testing.T) {
	cases := map[int]int{
		0:      0,
		1:      800,
		1000:   800_000,    // base rate, nowhere near a band
		2099:   1_679_200,  // the last order still cheaper at the base rate
		2100:   1_680_000,  // buying the 3 000 band early breaks even here
		3000:   1_680_000,  // and inside it the rate is exactly 560
		6000:   3_360_000,  // 200/day
		12000:  6_000_000,  // 400/day — already cheaper to buy the 15 000 band
		15000:  6_000_000,  // exactly that band, average 400
		24000:  9_600_000,  // 800/day
		50000:  15_000_000, // the chain band, average 300
		100000: 30_000_000,
	}
	for orders, want := range cases {
		if got := models.PriceForOrders(orders, ladder, 800); got != want {
			t.Errorf("%d orders → %d, want %d", orders, got, want)
		}
	}
}

// The whole point of the shape: the average lands **on** a published band rate
// rather than above it. A marginal ladder can never do this, which is why the
// old one lost the comparison at exactly the volumes worth winning.
func TestAverageReachesTheBandRate(t *testing.T) {
	for _, c := range []struct{ orders, want int }{
		{3000, 560},
		{15000, 400},
		{50000, 300},
		{100000, 300},
	} {
		got := models.PriceForOrders(c.orders, ladder, 800) / c.orders
		if got != c.want {
			t.Errorf("%d orders → %d so'm/order on average, want %d", c.orders, got, c.want)
		}
	}
}

// The bill never falls as orders rise.
//
// Buying a band early creates flat stretches — inside one, the next order is
// free — and that is fine. What would not be fine is the bill going *down*,
// which is what a naive "cheapest band" rule does if a band can be entered
// without paying for it in full.
func TestBillNeverDropsAsOrdersGrow(t *testing.T) {
	prev := 0
	for orders := 0; orders <= 60000; orders += 137 {
		got := models.PriceForOrders(orders, ladder, 800)
		if got < prev {
			t.Fatalf("%d orders → %d, less than the %d before it", orders, got, prev)
		}
		prev = got
	}
}

// A small customer must not notice the ladder exists: below the point where the
// second band becomes worth buying, they pay the base rate and nothing else.
func TestSmallCustomersAreUnaffected(t *testing.T) {
	for _, perDay := range []int{10, 20, 50, 60} {
		orders := perDay * 30
		if got := models.PriceForOrders(orders, ladder, 800); got != orders*800 {
			t.Errorf("%d/day → %d, want the flat %d", perDay, got, orders*800)
		}
	}
}

// No tiers means the old behaviour, unchanged. Every tenant created before
// this existed relies on it.
func TestNoTiersIsFlat(t *testing.T) {
	if got := models.PriceForOrders(500, nil, 800); got != 400_000 {
		t.Fatalf("got %d, want the flat rate", got)
	}
}

// A tenant's own ladder beats the platform default — the same rule
// pricePerOrder follows, so a negotiated deal survives a platform-wide change.
// This is where a chain's individually agreed rate lives.
func TestTenantLadderBeatsTheDefault(t *testing.T) {
	now := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.Local)
	own := models.Tenant{
		PricePerOrder: 800,
		PriceTiers:    []models.PriceTier{{UpTo: 0, Price: 250}},
	}
	if got := own.ChargeForOrders(12000, ladder, 0, now); got != 12000*250 {
		t.Fatalf("got %d, want the tenant's own rate", got)
	}
	// And a tenant with none of its own picks up the platform's.
	plain := models.Tenant{PricePerOrder: 800}
	if got := plain.ChargeForOrders(12000, ladder, 0, now); got != 6_000_000 {
		t.Fatalf("got %d, want the platform ladder", got)
	}
}

// Free and the discount still apply on top, in that order.
func TestFreeAndDiscountApplyAfterTheLadder(t *testing.T) {
	now := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.Local)

	free := models.Tenant{PricePerOrder: 800, Free: true, FreeReason: "anchor"}
	if got := free.ChargeForOrders(12000, ladder, 0, now); got != 0 {
		t.Errorf("free customer charged %d", got)
	}
	disc := models.Tenant{PricePerOrder: 800, DiscountPercent: 10}
	if got := disc.ChargeForOrders(12000, ladder, 0, now); got != 5_400_000 {
		t.Errorf("got %d, want 10%% off 6 000 000", got)
	}
}

// A malformed ladder must not bill orders at zero. Free is a decision, never a
// gap in a table.
func TestOrdersPastTheLastBandAreStillBilled(t *testing.T) {
	short := []models.PriceTier{{UpTo: 100, Price: 900}}
	got := models.PriceForOrders(300, short, 800)
	if got != 300*900 {
		t.Fatalf("got %d, want the last rate applied to every order", got)
	}
}

// The floor, and the four ways it must not misfire.
//
// The ladder bends the top of the curve down so the largest customer's invoice
// does not invite a negotiation. This is the other end: a restaurant doing five
// orders a day bills ~120 000 so'm a month and needs exactly as much support as
// one doing four hundred. Every rule below is money, so every rule is pinned.
func TestMinimumMonthly(t *testing.T) {
	now := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.Local)
	const floor = 300_000
	small := models.Tenant{PricePerOrder: 800}

	// 150 orders → 120 000 by the ladder, lifted to the floor.
	if got := small.ChargeForOrders(150, ladder, floor, now); got != floor {
		t.Errorf("below the floor: got %d, want %d", got, floor)
	}
	// A customer already above it is untouched — the floor is a minimum, not a
	// surcharge.
	if got := small.ChargeForOrders(1000, ladder, floor, now); got != 800_000 {
		t.Errorf("above the floor: got %d, want 800 000", got)
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
	free := models.Tenant{PricePerOrder: 800, Free: true, FreeReason: "anchor"}
	if got := free.ChargeForOrders(150, ladder, floor, now); got != 0 {
		t.Errorf("free customer charged %d", got)
	}

	// The discount applies to the floor, not underneath it: a negotiated
	// percentage that could not move the minimum would be a discount that does
	// nothing for exactly the customers small enough to have asked for one.
	disc := models.Tenant{PricePerOrder: 800, DiscountPercent: 20}
	if got := disc.ChargeForOrders(150, ladder, floor, now); got != 240_000 {
		t.Errorf("discounted floor: got %d, want 240 000", got)
	}

	// Unconfigured means off. A floor changes what real customers owe, so it
	// must never arrive as the side effect of a deploy.
	//
	// ⚠️ And it stays off: the published promise is that a month with no
	// orders costs nothing, which is the one line the competitor cannot say.
	if got := small.ChargeForOrders(150, ladder, 0, now); got != 120_000 {
		t.Errorf("no floor configured: got %d, want 120 000", got)
	}
}

// A tenant's own floor beats the platform's, in both directions.
//
// Same rule as the price and the tiers: terms agreed before a floor existed
// survive one being introduced, and raising the platform default must not
// silently reprice everybody who already said yes to something else.
func TestTenantMinimumOverridesPlatform(t *testing.T) {
	now := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.Local)
	own := models.Tenant{PricePerOrder: 800, MinMonthly: 500_000}
	if got := own.ChargeForOrders(150, ladder, 300_000, now); got != 500_000 {
		t.Errorf("own floor: got %d, want 500 000", got)
	}
	// And 0 on the tenant means "use the platform's", not "no floor" — the
	// field is absent on every row written before it existed.
	plain := models.Tenant{PricePerOrder: 800}
	if got := plain.Minimum(300_000); got != 300_000 {
		t.Errorf("fallback: got %d, want the platform floor", got)
	}
}

// The claim the landing page makes, pinned against the competitor's published
// ladder. ⚠️ Their bands are by orders **per day** and apply to the whole
// volume; ours are monthly, so the comparison is done in monthly orders.
//
// If a band ever moves, this test says which volume stopped being 20% cheaper —
// the sentence "we are cheaper at every volume" is either true or it is a claim
// we should not be printing.
func TestTwentyPercentUnderTheCompetitorAtEveryVolume(t *testing.T) {
	zoomda := func(orders int) int {
		perDay := orders / 30
		switch {
		case perDay >= 500:
			return orders * 500
		case perDay >= 100:
			return orders * 700
		default:
			return orders * 1000
		}
	}
	for _, orders := range []int{300, 1000, 2100, 3000, 6000, 12000, 15000, 24000, 50000, 100000} {
		ours, theirs := models.PriceForOrders(orders, ladder, 800), zoomda(orders)
		if ours*100 > theirs*80 {
			t.Errorf("%d orders: ours %d vs theirs %d — less than 20%% under", orders, ours, theirs)
		}
	}
}
