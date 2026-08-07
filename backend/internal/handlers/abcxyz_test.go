package handlers

import (
	"math"
	"testing"
	"time"

	"restaurant-backend/internal/models"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 12, 0, 0, 0, time.Local)
}

func order(at time.Time, items ...models.OrderItem) models.Order {
	return models.Order{CreatedAt: at, Items: items}
}

func dish(name string, price, qty int) models.OrderItem {
	return models.OrderItem{Name: name, Price: price, Qty: qty}
}

// The Pareto cut, and the boundary case that decides whether it is useful.
func TestABCCutsOnCumulativeShare(t *testing.T) {
	// Three dishes: 80%, 15%, 5% of the takings exactly.
	rows, total := classify([]models.Order{
		order(day(2026, time.August, 1), dish("Lag'mon", 80_000, 1)),
		order(day(2026, time.August, 1), dish("Somsa", 15_000, 1)),
		order(day(2026, time.August, 1), dish("Choy", 5_000, 1)),
	})
	if total != 100_000 {
		t.Fatalf("total = %d", total)
	}
	want := map[string]string{"Lag'mon": "A", "Somsa": "B", "Choy": "C"}
	for _, r := range rows {
		if r.ABC != want[r.Name] {
			t.Errorf("%s: got %s, want %s (share %.1f, cum %.1f)",
				r.Name, r.ABC, want[r.Name], r.Share, r.Cumulative)
		}
	}

	// ⚠️ The dish that *crosses* 80% belongs to A, not B.
	//
	// Two dishes at 60% and 40%: the second one takes the running total from
	// 60 to 100, crossing the line. Cutting on the total *after* adding it
	// would call it B — and on a short menu that is a dish carrying 40% of the
	// takings being reported as second-tier.
	rows, _ = classify([]models.Order{
		order(day(2026, time.August, 1), dish("Katta", 60_000, 1)),
		order(day(2026, time.August, 1), dish("Kichik", 40_000, 1)),
	})
	for _, r := range rows {
		if r.ABC != "A" {
			t.Errorf("%s crossed the 80%% line and was demoted to %s", r.Name, r.ABC)
		}
	}
}

// Best earner first — the cumulative column is meaningless in any other order.
func TestABCSortsByRevenueNotQuantity(t *testing.T) {
	rows, _ := classify([]models.Order{
		order(day(2026, time.August, 1), dish("Choy", 2_000, 50)), // 100 000
		order(day(2026, time.August, 1), dish("Osh", 40_000, 4)),  // 160 000
	})
	if rows[0].Name != "Osh" {
		t.Errorf("first row is %q; the biggest earner must lead", rows[0].Name)
	}
	// And the cumulative share must actually accumulate.
	if rows[len(rows)-1].Cumulative < 99.9 {
		t.Errorf("cumulative ends at %.1f, want 100", rows[len(rows)-1].Cumulative)
	}
}

// ⚠️ The rule that makes XYZ mean anything: a day with no sales is a zero, not
// a missing observation.
func TestXYZCountsSilentDaysAsZero(t *testing.T) {
	// Ten days of trading. "Bayram" sells twenty portions on one of them and
	// nothing on the other nine.
	var orders []models.Order
	for d := 1; d <= 10; d++ {
		orders = append(orders, order(day(2026, time.August, d), dish("Non", 3_000, 5)))
	}
	orders = append(orders, order(day(2026, time.August, 3), dish("Bayram", 50_000, 20)))

	rows, _ := classify(orders)
	byName := map[string]abcRow{}
	for _, r := range rows {
		byName[r.Name] = r
	}

	// Five portions every single day is as steady as it gets.
	if got := byName["Non"]; got.XYZ != "X" || got.Variation > 0.01 {
		t.Errorf("Non: %s at %.1f%%, want X at ~0", got.XYZ, got.Variation)
	}
	// One spike in ten days is the most erratic thing on the menu. Averaging
	// only the days it sold would have reported it as perfectly steady — the
	// exact inversion this test exists to prevent.
	if got := byName["Bayram"]; got.XYZ != "Z" {
		t.Errorf("Bayram: %s at %.1f%%, want Z", got.XYZ, got.Variation)
	}
	if got := byName["Bayram"]; got.Days != 1 {
		t.Errorf("Bayram sold on %d days, want 1", got.Days)
	}
}

// A single day is not a series, and must not be reported as perfect steadiness.
func TestXYZRefusesToJudgeOneDay(t *testing.T) {
	rows, _ := classify([]models.Order{
		order(day(2026, time.August, 1), dish("Lag'mon", 30_000, 3)),
	})
	if rows[0].Variation != 0 {
		t.Errorf("variation = %.1f on a one-day period", rows[0].Variation)
	}
	// It still classifies — the owner asked for a day and gets a day — but the
	// Days column is what says how much the XYZ letter is worth.
	if rows[0].Days != 1 {
		t.Errorf("days = %d", rows[0].Days)
	}
}

// The coefficient of variation itself, against a hand-computed value.
func TestVariationMatchesTheFormula(t *testing.T) {
	// Sales of 2, 4, 6 over three days: mean 4, population sd √(8/3).
	got := variation(map[string]int{"a": 2, "b": 4, "c": 6}, 3)
	want := math.Sqrt(8.0/3.0) / 4 * 100
	if math.Abs(got-want) > 0.01 {
		t.Errorf("got %.4f, want %.4f", got, want)
	}
}

// An empty period must produce an empty analysis, not a division by zero.
func TestClassifyEmptyPeriod(t *testing.T) {
	rows, total := classify(nil)
	if len(rows) != 0 || total != 0 {
		t.Fatalf("rows=%d total=%d", len(rows), total)
	}
	// And the report built from it must still be a valid, downloadable sheet
	// rather than a nil-row panic — an owner opening a quiet week gets a file
	// that says nothing sold, which is the answer.
	if got := abcReportRows(rows); got == nil {
		t.Error("nil rows would marshal as null")
	}
}

// Cancelled orders never reach classify, but a cancelled *dish* does not exist
// — so quantities are the sum of what was ordered, options included in price.
func TestClassifyUsesChargedPrice(t *testing.T) {
	rows, total := classify([]models.Order{
		// 30 000 base + 5 000 option delta was already folded into Price.
		order(day(2026, time.August, 1), dish("Lag'mon", 35_000, 2)),
	})
	if total != 70_000 || rows[0].Revenue != 70_000 || rows[0].Qty != 2 {
		t.Errorf("total=%d revenue=%d qty=%d", total, rows[0].Revenue, rows[0].Qty)
	}
}
