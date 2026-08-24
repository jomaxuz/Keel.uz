package handlers

import (
	"net/http/httptest"
	"testing"
	"time"
)

// ⚠️ **"1 kun" is today, not yesterday-to-today.** An off-by-one here draws an
// empty screen at nine in the morning, which reads as a broken collector rather
// than as a quiet start — and the first thing anybody does about a broken
// collector is press "collect now" and wait.
func TestOneDayWindowIsToday(t *testing.T) {
	from, to, err := overviewWindow(httptest.NewRequest("GET", "/?range=1d", nil))
	if err != nil {
		t.Fatal(err)
	}
	today := time.Now().Format("2006-01-02")
	if from != today || to != today {
		t.Errorf("1d = %s..%s, want %s..%s", from, to, today, today)
	}
}

// Typed dates beat the shorthand, and a pair filled in backwards is corrected
// rather than refused: two date boxes side by side get swapped constantly, and
// the question the operator meant is unambiguous.
func TestCustomWindowWinsAndSortsItself(t *testing.T) {
	r := httptest.NewRequest("GET", "/?range=1y&from=2026-03-31&to=2026-03-01", nil)
	from, to, err := overviewWindow(r)
	if err != nil {
		t.Fatal(err)
	}
	if from != "2026-03-01" || to != "2026-03-31" {
		t.Errorf("got %s..%s, want 2026-03-01..2026-03-31", from, to)
	}
}

// ⚠️ A mistyped shorthand must not error. This value arrives from a query
// string and the screen it feeds is a dashboard: answering a bad bookmark with
// a red box is how a bookmark stops being used.
func TestUnknownRangeFallsBackRatherThanFailing(t *testing.T) {
	from, to, err := overviewWindow(httptest.NewRequest("GET", "/?range=banana", nil))
	if err != nil {
		t.Fatalf("a mistyped range was refused: %v", err)
	}
	f, _ := time.ParseInLocation("2006-01-02", from, time.Local)
	e, _ := time.ParseInLocation("2006-01-02", to, time.Local)
	if days := int(e.Sub(f).Hours()/24) + 1; days != 30 {
		t.Errorf("fallback window = %d days, want 30", days)
	}
}

// ⚠️ Buckets are sums. Somebody will add the bars up and compare them with the
// total printed above the chart, and an average would make that fail silently.
func TestBucketsSumRatherThanAverage(t *testing.T) {
	// 60 days forces the weekly grain.
	daily := make([]overviewPoint, 0, 60)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	for i := 0; i < 60; i++ {
		daily = append(daily, overviewPoint{
			Date:   start.AddDate(0, 0, i).Format("2006-01-02"),
			Orders: 1, TillChecks: 2, Revenue: 100,
		})
	}
	grain, series := bucketSeries(daily)
	if grain != "week" {
		t.Fatalf("60 days grouped as %q, want week", grain)
	}
	var orders, checks, revenue int
	for _, p := range series {
		orders += p.Orders
		checks += p.TillChecks
		revenue += p.Revenue
	}
	if orders != 60 || checks != 120 || revenue != 6000 {
		t.Errorf("buckets lost or averaged data: orders=%d checks=%d revenue=%d", orders, checks, revenue)
	}
	// ⚠️ Monday-start weeks: in a restaurant business the weekend is the point
	// of the chart, and Go's Sunday-start weekday would split it across two
	// bars — hiding exactly what somebody opened this to see.
	first, err := time.ParseInLocation("2006-01-02", series[0].Date, time.Local)
	if err != nil {
		t.Fatal(err)
	}
	if first.Weekday() != time.Monday {
		t.Errorf("week buckets start on %s, want Monday", first.Weekday())
	}
}

// A year must not come back as 365 bars: the point of the year button is the
// shape of the year, and at one bar per day the shape is noise.
func TestYearIsGroupedByMonth(t *testing.T) {
	daily := make([]overviewPoint, 0, 365)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	for i := 0; i < 365; i++ {
		daily = append(daily, overviewPoint{Date: start.AddDate(0, 0, i).Format("2006-01-02"), Orders: 1})
	}
	grain, series := bucketSeries(daily)
	if grain != "month" || len(series) != 12 {
		t.Errorf("a year came back as %d %s buckets, want 12 month buckets", len(series), grain)
	}
}
