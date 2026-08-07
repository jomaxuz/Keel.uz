package handlers

import (
	"testing"
	"time"

	"restaurant-backend/internal/models"
)

// When an order counts as money.
//
// The dashboard used to add up every order that was not cancelled, so a
// 100 000 so'm order placed a minute ago — uncooked, undelivered, unpaid —
// showed as takings straight away. The figure was always right eventually and
// wrong all day, which is the worst shape a number on a dashboard can have:
// nobody distrusts it, they just plan around it.

func TestPlacedIsNotCollected(t *testing.T) {
	for _, status := range []models.OrderStatus{
		models.StatusPending,
		"confirmed",
		"preparing",
		models.StatusOnTheWay,
	} {
		o := models.Order{Status: status, PaymentMethod: "cash", PaymentStatus: models.PayUnpaid}
		if received(o) {
			t.Errorf("%s: counted as money before anybody paid", status)
		}
	}
}

// Cash arrives when the courier hands the food over — which is also the
// terminal state for pickup and dine-in.
func TestCashCountsOnDelivery(t *testing.T) {
	o := models.Order{
		Status: models.StatusDelivered, PaymentMethod: "cash",
		PaymentStatus: models.PayUnpaid,
	}
	if !received(o) {
		t.Fatal("delivered cash order is money in hand")
	}
}

// A card payment is real before the food moves, and stays real while delivery
// is still an hour away. Waiting for `delivered` would under-report a
// restaurant that takes payment up front.
func TestCardCountsWhenTheBankConfirms(t *testing.T) {
	o := models.Order{
		Status: "preparing", PaymentMethod: "card", PaymentStatus: models.PayPaid,
	}
	if !received(o) {
		t.Fatal("a confirmed card payment is money in hand")
	}
}

// The one that is easy to get wrong in the other direction: a paid order that
// was then cancelled is money owed back, not takings. The refund moves
// paymentStatus off `paid`, but it must not count in the window before that.
func TestCancelledNeverCountsEvenIfPaid(t *testing.T) {
	o := models.Order{
		Status: models.StatusCancelled, PaymentMethod: "card",
		PaymentStatus: models.PayPaid,
	}
	if received(o) {
		t.Fatal("a cancelled paid order is a refund waiting to happen")
	}
}

// ⚠️ Every day in the range appears, including the ones with nothing.
//
// Charting only the days that had orders draws a straight line across a closed
// week: the reader sees a smooth trend where there was a gap, and a quiet
// Monday becomes invisible rather than obvious.
func TestDailySeriesFillsSilentDays(t *testing.T) {
	mk := func(d int, total int, delivered bool) models.Order {
		o := models.Order{
			CreatedAt: time.Date(2026, time.August, d, 12, 0, 0, 0, time.Local),
			Total:     total, Status: models.StatusPreparing,
		}
		if delivered {
			o.Status = models.StatusDelivered
			o.PaymentStatus = models.PayPaid
		}
		return o
	}
	// Orders on the 1st and the 4th only.
	series := dailySeries([]models.Order{
		mk(1, 50_000, true),
		mk(4, 30_000, true),
		// Cancelled: not a data point at all.
		{CreatedAt: time.Date(2026, time.August, 2, 12, 0, 0, 0, time.Local),
			Total: 900_000, Status: models.StatusCancelled},
	}, nil, nil)

	if len(series) != 4 {
		t.Fatalf("got %d points, want 1–4 August inclusive: %+v", len(series), series)
	}
	if series[0].Date != "2026-08-01" || series[3].Date != "2026-08-04" {
		t.Errorf("range = %s .. %s", series[0].Date, series[3].Date)
	}
	// The quiet days are present and zero, not missing.
	if series[1].Orders != 0 || series[1].Revenue != 0 {
		t.Errorf("2 August should be empty, got %+v", series[1])
	}
	// And the cancelled order contributed nothing to it.
	if series[1].Revenue != 0 {
		t.Error("a cancelled order put money on the chart")
	}
	if series[0].Revenue != 50_000 || series[3].Revenue != 30_000 {
		t.Errorf("revenue = %d / %d", series[0].Revenue, series[3].Revenue)
	}
}

// The chart is on the same basis as the numbers above it: money counts when it
// is collected, not when the order is placed.
func TestDailySeriesUsesCollectedBasis(t *testing.T) {
	placed := models.Order{
		CreatedAt: time.Date(2026, time.August, 1, 12, 0, 0, 0, time.Local),
		Total:     100_000, Status: models.StatusPreparing,
	}
	series := dailySeries([]models.Order{placed}, nil, nil)
	if len(series) != 1 {
		t.Fatalf("points: %d", len(series))
	}
	if series[0].Orders != 1 {
		t.Error("the order should be counted")
	}
	if series[0].Revenue != 0 {
		t.Errorf("revenue = %d; nobody has paid yet", series[0].Revenue)
	}
}
