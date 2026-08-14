package handlers

import (
	"testing"
	"time"

	"restaurant-backend/internal/models"
)

// The failures sealed here are all of the same shape: the report still draws,
// still looks like a month of business, and is wrong in a direction nobody
// checks.

func paidOrder(at time.Time, total int, typ string) models.Order {
	return models.Order{
		CreatedAt: at, Total: total, Type: typ,
		Status: models.StatusDelivered, PaymentMethod: "cash",
		Items: []models.OrderItem{{Qty: 1}},
	}
}

// ⚠️ Buckets are cut in **local** time.
//
// Mongo hands every date back in UTC whatever the server's zone (§ "Mongo'dan
// kelgan sana doim UTC"), so bucketing on the raw value puts a Tashkent
// restaurant's entire evening trade — everything after 19:00 local — into the
// next day. The chart still shows a plausible month; it is just the wrong day
// for the busiest half of the business.
func TestBucketsUseLocalDay(t *testing.T) {
	local, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	old := time.Local
	time.Local = local
	defer func() { time.Local = old }()

	// 22:30 on the 14th in Tashkent is 17:30 UTC on the 14th — but 20:00 local
	// is 15:00 UTC, and an order at 02:00 UTC on the 15th is 07:00 on the 15th
	// local. The one that matters: 21:00 UTC on the 14th is 02:00 on the 15th
	// local, and must not land on the 14th.
	evening := time.Date(2026, 8, 14, 19, 30, 0, 0, time.UTC) // 00:30 on the 15th local
	key, _ := bucketKey(evening, groupDay)
	if key != "2026-08-15" {
		t.Fatalf("bucketed a UTC timestamp into the wrong local day: got %s, want 2026-08-15", key)
	}
}

// A week runs Monday to Sunday, because that is the week the staff schedule
// and payroll already use. Two week boundaries in one panel is a
// reconciliation nobody wins.
func TestWeekStartsMonday(t *testing.T) {
	// 2026-08-14 is a Friday; 2026-08-16 is the Sunday of the same week.
	for _, day := range []time.Time{
		time.Date(2026, 8, 10, 12, 0, 0, 0, time.Local), // Monday
		time.Date(2026, 8, 14, 12, 0, 0, 0, time.Local), // Friday
		time.Date(2026, 8, 16, 23, 0, 0, 0, time.Local), // Sunday
	} {
		key, _ := bucketKey(day, groupWeek)
		if key != "2026-08-10" {
			t.Errorf("%s: week key %s, want the Monday 2026-08-10", day.Format("Mon 02"), key)
		}
	}
}

// ⚠️ The average bill divides takings by the orders the takings came from, not
// by every order placed. Dividing by all of them makes the average fall as the
// kitchen gets busier — the opposite of what an owner reads it for, and
// plausible enough that nobody questions it.
func TestAvgCheckDividesByCollectedOrders(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.Local)
	orders := []models.Order{
		paidOrder(now, 100_000, "delivery"),
		// Placed and still out: money, but not takings.
		{CreatedAt: now, Total: 100_000, Type: "delivery", Status: models.StatusConfirmed},
		{CreatedAt: now, Total: 100_000, Type: "delivery", Status: models.StatusConfirmed},
	}
	_, totals := summariseSales(orders, groupDay)

	if totals.AvgCheck != 100_000 {
		t.Fatalf("avg check %d, want 100000 — divided by orders placed, not collected", totals.AvgCheck)
	}
	if totals.Revenue != 100_000 {
		t.Fatalf("revenue %d, want 100000", totals.Revenue)
	}
	if totals.Pending != 200_000 {
		t.Fatalf("pending %d, want 200000 — placed but not collected is its own line", totals.Pending)
	}
}

// A cancelled order is work that arrived and nothing else. Counting it as
// pending would put money on the screen that will never come.
func TestCancelledIsCountedButNotOwed(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.Local)
	orders := []models.Order{
		{CreatedAt: now, Total: 90_000, Type: "delivery", Status: models.StatusCancelled,
			Items: []models.OrderItem{{Qty: 3}}},
	}
	_, totals := summariseSales(orders, groupDay)

	if totals.Orders != 1 || totals.Cancelled != 1 {
		t.Fatalf("orders %d cancelled %d, want 1/1", totals.Orders, totals.Cancelled)
	}
	if totals.Revenue != 0 || totals.Pending != 0 {
		t.Fatalf("revenue %d pending %d, want 0/0 — a cancellation owes nothing",
			totals.Revenue, totals.Pending)
	}
	// Nor was anything sold: only a cancellation un-sells a dish, and this is
	// the cancellation.
	if totals.Items != 0 {
		t.Fatalf("items %d, want 0 — a cancelled order sold nothing", totals.Items)
	}
	// And it is not in any fulfilment column: a money-shaped row next to an
	// order that will never produce money is how a split stops adding up.
	if totals.Delivery != 0 {
		t.Fatalf("delivery %d, want 0", totals.Delivery)
	}
}

// ⚠️ No baseline, no percentage. "Up 100%" from a month with nothing in it is
// not information, and dividing by zero is how a dashboard starts printing ∞
// next to a real figure.
func TestGrowthNeedsABaseline(t *testing.T) {
	if _, ok := percentChange(0, 500_000); ok {
		t.Fatal("reported growth from a zero baseline")
	}
	pct, ok := percentChange(400_000, 500_000)
	if !ok || pct != 25 {
		t.Fatalf("percentChange(400000, 500000) = %v, %v; want 25, true", pct, ok)
	}
}

// The comparison is skipped for an open-ended period rather than invented.
// "All time" has nothing before it, and a growth figure for it would look like
// every other growth figure on the screen.
func TestNoComparisonForOpenPeriod(t *testing.T) {
	c := &salesChange{}
	c.withPercent(salesTotals{Revenue: 100})
	if len(c.Percent) != 0 {
		t.Fatalf("percent map %v, want empty when the previous period is zero", c.Percent)
	}
}

// An unknown `group` draws days rather than erroring: the parameter arrives in
// a link an operator may have edited, and a report that refuses to draw is a
// worse answer than one drawn finely.
func TestUnknownGroupFallsBackToDays(t *testing.T) {
	for _, in := range []string{"", "hafta", "DAY", "garbage"} {
		if got := parseSalesGroup(in); got != groupDay && in != "DAY" {
			t.Errorf("parseSalesGroup(%q) = %q, want day", in, got)
		}
	}
	if parseSalesGroup("  MONTH ") != groupMonth {
		t.Error("group parsing should tolerate case and padding")
	}
}

// The busiest hour is summed across the period, and cancellations are left out
// of it: this number is read to decide when a second courier is needed, and
// orders nobody cooked do not need one.
func TestHoursSkipCancellations(t *testing.T) {
	at := time.Date(2026, 8, 14, 19, 0, 0, 0, time.Local)
	hours := salesByHour([]models.Order{
		paidOrder(at, 50_000, "delivery"),
		{CreatedAt: at, Total: 90_000, Status: models.StatusCancelled},
	})
	if hours[19].Orders != 1 {
		t.Fatalf("hour 19 orders %d, want 1", hours[19].Orders)
	}
	if hours[19].Revenue != 50_000 {
		t.Fatalf("hour 19 revenue %d, want 50000", hours[19].Revenue)
	}
}
