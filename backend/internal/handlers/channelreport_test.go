package handlers

import (
	"testing"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ⚠️ The zero-ObjectID trap, in the one place where it produces a plausible
// number instead of a crash.
//
// An unset `userId` does not disappear — it is a `[12]byte`, so `omitempty`
// leaves it as twelve zero bytes (§ "Bo'sh ObjectID JSON'da yo'qolmaydi").
// Keying customers off it directly would count every guest order in the
// restaurant's history as the same one extremely loyal customer, and the
// "Mijozlar" column would read 1 next to four hundred orders — low enough to
// look like a quiet channel rather than a bug.
func TestGuestOrdersAreNotOneCustomer(t *testing.T) {
	a := models.Order{Customer: models.OrderCustomer{Phone: "998901112233"}}
	b := models.Order{Customer: models.OrderCustomer{Phone: "998904445566"}}
	if customerKey(a) == customerKey(b) {
		t.Fatal("two guests with different phones collapsed into one customer")
	}

	id := primitive.NewObjectID()
	signedIn := models.Order{UserID: id, Customer: models.OrderCustomer{Phone: "998901112233"}}
	if customerKey(signedIn) != id.Hex() {
		t.Fatal("a signed-in customer must be counted by account, not by phone")
	}
	// And the same person ordering twice is one customer, not two.
	if customerKey(a) != customerKey(models.Order{Customer: models.OrderCustomer{Phone: "998901112233"}}) {
		t.Fatal("the same phone must be the same customer")
	}
}

// An order from before `channel` existed reports itself as unknown rather than
// being folded into "web". Most of them probably were the site — but a report
// that guesses cannot be compared with one that knows, and the owner reading
// this year against last would never see where the line is.
func TestOldOrdersDoNotClaimAChannel(t *testing.T) {
	if got := orderChannelKey(models.Order{}); got != "unknown" {
		t.Fatalf("channel %q for an order that predates the field, want unknown", got)
	}
	// Except when it names itself: the operator's name was always written
	// down, so those orders were never actually ambiguous.
	if got := orderChannelKey(models.Order{TakenBy: "Dilnoza"}); got != "operator" {
		t.Fatalf("channel %q for an operator-taken order, want operator", got)
	}
	if got := orderChannelKey(models.Order{Channel: "telegram"}); got != "telegram" {
		t.Fatalf("channel %q, want telegram", got)
	}
}

// An order with no `type` is a delivery — that is what every order was before
// pickup and dine-in existed, and reading the zero value any other way would
// reclassify every old order in the archive.
func TestEmptyTypeIsDelivery(t *testing.T) {
	if got := orderTypeKey(models.Order{}); got != "delivery" {
		t.Fatalf("type %q, want delivery", got)
	}
}

// ⚠️ "New customer" is decided from the customer's **first ever** order, not
// from the first one inside the window. Otherwise everybody is new in every
// period, and the most loyal restaurant in the city reports that it cannot
// keep anyone.
func TestNewMeansFirstEverOrder(t *testing.T) {
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)
	inPeriod := time.Date(2026, 8, 14, 12, 0, 0, 0, time.Local)

	regular := "998901112233"
	newcomer := "998904445566"
	firstOrder := map[string]time.Time{
		// Ordered for the first time two years ago; ordering again now does
		// not make them new.
		regular: time.Date(2024, 3, 1, 12, 0, 0, 0, time.Local),
		// First order is the one in this period.
		newcomer: inPeriod,
	}

	orders := []models.Order{
		{CreatedAt: inPeriod, Channel: "telegram", Status: models.StatusDelivered,
			Total: 100_000, Customer: models.OrderCustomer{Phone: regular}},
		{CreatedAt: inPeriod, Channel: "telegram", Status: models.StatusDelivered,
			Total: 50_000, Customer: models.OrderCustomer{Phone: newcomer}},
	}
	rows := channelRows(orders, orderChannelKey, firstOrder, &from)

	if len(rows) != 1 {
		t.Fatalf("rows %d, want 1", len(rows))
	}
	r := rows[0]
	if r.Customers != 2 {
		t.Fatalf("customers %d, want 2", r.Customers)
	}
	if r.NewCustomers != 1 {
		t.Fatalf("new customers %d, want 1 — a returning customer is not new", r.NewCustomers)
	}
	if r.Revenue != 150_000 || r.AvgCheck != 75_000 {
		t.Fatalf("revenue %d avg %d, want 150000 / 75000", r.Revenue, r.AvgCheck)
	}
}

// The two cuts each add up to the whole period on their own. They overlap —
// a guest can order pickup through the bot — which is exactly why they are
// reported separately and never summed together.
func TestEachCutCoversThePeriodOnce(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.Local)
	orders := []models.Order{
		{CreatedAt: now, Channel: "telegram", Type: "pickup", Status: models.StatusDelivered, Total: 40_000},
		{CreatedAt: now, Channel: "telegram", Type: "delivery", Status: models.StatusDelivered, Total: 60_000},
		{CreatedAt: now, Channel: "web", Type: "pickup", Status: models.StatusDelivered, Total: 30_000},
	}
	sum := func(rows []channelRow) (orders int, share float64) {
		for _, r := range rows {
			orders += r.Orders
			share += r.Share
		}
		return orders, share
	}

	byChannel, byType := channelRows(orders, orderChannelKey, nil, nil), channelRows(orders, orderTypeKey, nil, nil)
	for name, rows := range map[string][]channelRow{"channel": byChannel, "type": byType} {
		n, share := sum(rows)
		if n != 3 {
			t.Errorf("%s: orders %d, want 3", name, n)
		}
		if share < 99.9 || share > 100.1 {
			t.Errorf("%s: shares add to %.1f, want 100", name, share)
		}
	}
	// Busiest first — the first row is the answer to "where do our orders
	// come from", so the ordering is part of the report, not a detail.
	if byChannel[0].Key != "telegram" {
		t.Fatalf("first channel row %q, want telegram (2 orders vs 1)", byChannel[0].Key)
	}
}
