package handlers

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

func TestHandoverNeverExceedsWhatTheCourierOwes(t *testing.T) {
	for _, c := range []struct{ total, owes, want int }{
		{245_000, 1_000_000, 245_000}, // the ordinary case
		{245_000, 100_000, 100_000},   // part of it was handed over already
		{245_000, 0, 0},               // the panel closed the whole balance
		{245_000, -8_373_000, 0},      // an old double handover is not repeated
	} {
		if got := handoverAmount(c.total, c.owes); got != c.want {
			t.Errorf("handoverAmount(%d, %d) = %d, want %d", c.total, c.owes, got, c.want)
		}
	}
}

// The b5somsa sequence on a real database (skipped without one): orders
// delivered in the morning, the whole balance closed in the panel in the
// evening, then each order "paid" at the till. What the courier owes must go to
// zero and stay there — not below.
func TestCourierOwesAfterAPanelSettlementLive(t *testing.T) {
	h, branch := liveHandler(t)
	ctx := context.Background()
	courier := primitive.NewObjectID()
	var totals []int
	for _, total := range []int{245_000, 180_000, 532_000} {
		_, err := h.Store.Orders.InsertOne(ctx, models.Order{
			BranchID: branch, CourierID: courier, Type: "delivery", Total: total,
			Status: models.StatusDelivered, PaymentMethod: models.ProviderCash, CreatedAt: time.Now(),
		})
		if err != nil {
			t.Fatal(err)
		}
		totals = append(totals, total)
	}
	owes, err := h.courierOwes(ctx, courier)
	if err != nil || owes != 957_000 {
		t.Fatalf("before any handover: %d, %v", owes, err)
	}
	// The panel: the whole balance, as one sum.
	if _, err := h.Store.Settlements.InsertOne(ctx, models.CourierSettlement{
		CourierID: courier, Amount: owes, At: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	// The till: each order, as it would now record it.
	for _, total := range totals {
		owes, _ := h.courierOwes(ctx, courier)
		if amount := handoverAmount(total, owes); amount > 0 {
			t.Fatalf("the till would hand over %d again", amount)
		}
	}
	if owes, _ := h.courierOwes(ctx, courier); owes != 0 {
		t.Fatalf("after both: %d, want 0", owes)
	}
}
