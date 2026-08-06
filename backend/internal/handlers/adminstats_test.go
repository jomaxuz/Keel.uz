package handlers

import (
	"testing"

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
