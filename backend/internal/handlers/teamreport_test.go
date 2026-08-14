package handlers

import (
	"testing"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func clockAt(hour, min int) time.Time {
	return time.Date(2026, 8, 14, hour, min, 0, 0, time.Local)
}

// The courier's leg is measured from "on the way", not from the order.
//
// Creation-to-delivery folds in the wait for confirmation and the time in the
// kitchen, and then files the total under a courier's name. On a busy Friday
// that is the difference between a courier who is slow and a kitchen that is.
func TestCourierTimeExcludesTheKitchen(t *testing.T) {
	o := &models.Order{
		CreatedAt: clockAt(18, 0),
		StatusHistory: []models.StatusEvent{
			{Status: models.StatusConfirmed, At: clockAt(18, 25)},
			{Status: models.StatusPreparing, At: clockAt(18, 30)},
			{Status: models.StatusOnTheWay, At: clockAt(19, 0)},
			{Status: models.StatusDelivered, At: clockAt(19, 20)},
		},
	}
	mins, ok := onTheRoadMinutes(o)
	if !ok {
		t.Fatal("both events present, expected a timing")
	}
	if mins != 20 {
		t.Fatalf("minutes %d, want 20 — the kitchen's hour must not be billed to the courier", mins)
	}
}

// ⚠️ A re-dispatched order has two "on the way" events. Taking the first would
// bill the courier for everything in between — including the time the order
// spent back in the kitchen after being pushed back.
func TestRedispatchTimesTheSecondRun(t *testing.T) {
	o := &models.Order{
		StatusHistory: []models.StatusEvent{
			{Status: models.StatusOnTheWay, At: clockAt(18, 0)},
			// Pushed back: wrong address, back to the kitchen.
			{Status: models.StatusPreparing, At: clockAt(18, 20)},
			{Status: models.StatusOnTheWay, At: clockAt(19, 0)},
			{Status: models.StatusDelivered, At: clockAt(19, 15)},
		},
	}
	mins, ok := onTheRoadMinutes(o)
	if !ok || mins != 15 {
		t.Fatalf("minutes %d (ok=%v), want 15 — only the run that delivered counts", mins, ok)
	}
}

// An order still out has no delivery time, and says so rather than reporting
// zero. A 0 in an average column reads as instant delivery.
func TestUndeliveredHasNoTiming(t *testing.T) {
	o := &models.Order{StatusHistory: []models.StatusEvent{
		{Status: models.StatusOnTheWay, At: clockAt(19, 0)},
	}}
	if _, ok := onTheRoadMinutes(o); ok {
		t.Fatal("timed an order that has not arrived")
	}
	// And an order whose history was written out of order does not produce a
	// negative average.
	backwards := &models.Order{StatusHistory: []models.StatusEvent{
		{Status: models.StatusOnTheWay, At: clockAt(19, 30)},
		{Status: models.StatusDelivered, At: clockAt(19, 0)},
	}}
	if _, ok := onTheRoadMinutes(backwards); ok {
		t.Fatal("timed a delivery that arrived before it left")
	}
}

// Cash in hand is lifetime, not the period: collected-ever minus
// handed-back-ever. A period-bounded version would produce a debt that clears
// itself every month whether or not the money came back.
func TestCashInHandIsLifetime(t *testing.T) {
	id := primitive.NewObjectID()
	couriers := []models.Courier{{ID: id, Name: "Aziz", PayoutMode: models.PayoutPerOrder, PayoutPerOrder: 8000}}

	// One delivery inside the period, worth 100 000 in cash.
	orders := []models.Order{{
		CourierID: id, Status: models.StatusDelivered, Type: "delivery",
		Total: 100_000, PaymentMethod: models.ProviderCash, DeliveryFee: 15_000,
		StatusHistory: []models.StatusEvent{
			{Status: models.StatusOnTheWay, At: clockAt(19, 0)},
			{Status: models.StatusDelivered, At: clockAt(19, 30)},
		},
	}}
	// Lifetime: they have collected 500 000 and handed back 300 000.
	collected := map[string]int{id.Hex(): 500_000}
	settled := map[string]int{id.Hex(): 300_000}

	rows := courierReportRows(couriers, orders, settled, collected)
	if len(rows) != 1 {
		t.Fatalf("rows %d, want 1", len(rows))
	}
	r := rows[0]
	if r.Cash != 100_000 {
		t.Fatalf("period cash %d, want 100000", r.Cash)
	}
	if r.CashInHand != 200_000 {
		t.Fatalf("cash in hand %d, want 200000 (lifetime collected − settled)", r.CashInHand)
	}
	// The payout rule is the courier's own, not the delivery fee.
	if r.Earnings != 8000 {
		t.Fatalf("earnings %d, want 8000 — per-order rate, not the delivery fee", r.Earnings)
	}
	if r.AvgMinutes != 30 || r.TimedOrders != 1 {
		t.Fatalf("avg %d over %d orders, want 30/1", r.AvgMinutes, r.TimedOrders)
	}
}

// A courier with no work in the period still gets a row — at zero. Dropping
// them would make the report read as a list of couriers rather than a
// comparison, and "who did nothing this month" is a question it should answer.
func TestIdleCourierStillHasARow(t *testing.T) {
	id := primitive.NewObjectID()
	rows := courierReportRows(
		[]models.Courier{{ID: id, Name: "Bekzod"}},
		nil, nil, nil,
	)
	if len(rows) != 1 || rows[0].Delivered != 0 {
		t.Fatalf("rows %+v, want one row at zero", rows)
	}
	// And no average is claimed for them.
	sheet := courierReportSheet(rows)
	if _, has := sheet[0]["avgMinutes"]; has {
		t.Fatal("an average was written for a courier who delivered nothing")
	}
}

// An order carried by a courier outside this admin's branches is skipped
// rather than filed under a blank name — a row nobody can act on makes the
// total look wrong.
func TestForeignCourierOrdersAreNotCounted(t *testing.T) {
	mine, theirs := primitive.NewObjectID(), primitive.NewObjectID()
	rows := courierReportRows(
		[]models.Courier{{ID: mine, Name: "Aziz"}},
		[]models.Order{{CourierID: theirs, Status: models.StatusDelivered, Total: 90_000}},
		nil, nil,
	)
	if len(rows) != 1 {
		t.Fatalf("rows %d, want 1", len(rows))
	}
	if rows[0].Delivered != 0 || rows[0].Carried != 0 {
		t.Fatal("another branch's order was counted")
	}
}
