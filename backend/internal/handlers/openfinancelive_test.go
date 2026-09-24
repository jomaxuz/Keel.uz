package handlers

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"restaurant-backend/internal/models"
)

// The watcher against a real database (skipped without one): a baseline on the
// first look, one event per changed day after it, silence when nothing moved —
// and a deletion is a change.
func TestMoneyWatchLive(t *testing.T) {
	h, branch := liveHandler(t)
	ctx := context.Background()
	ep := models.WebhookEndpoint{
		URL: "https://example.com/hook", Events: []string{models.EventMoneyDayChanged},
		Secret: "whsec_x", Enabled: true, CreatedAt: time.Now().Add(-time.Hour),
	}
	if _, err := h.Store.WebhookEndpoints.InsertOne(ctx, ep); err != nil {
		t.Fatal(err)
	}
	queued := func() int64 {
		n, _ := h.Store.WebhookDeliveries.CountDocuments(ctx, bson.M{"event": models.EventMoneyDayChanged})
		return n
	}

	h.watchMoney(ctx)
	if n := queued(); n != 0 {
		t.Fatalf("the baseline announced %d days", n)
	}
	res, err := h.Store.Expenses.InsertOne(ctx, models.Expense{
		BranchID: branch, At: time.Now(), Amount: 5_000_000, Category: "ijara",
	})
	if err != nil {
		t.Fatal(err)
	}
	h.watchMoney(ctx)
	if n := queued(); n != 1 {
		t.Fatalf("after an expense: %d events, want 1", n)
	}
	h.watchMoney(ctx)
	if n := queued(); n != 1 {
		t.Fatalf("nothing changed, but %d events", n)
	}
	if _, err := h.Store.Expenses.DeleteOne(ctx, bson.M{"_id": res.InsertedID}); err != nil {
		t.Fatal(err)
	}
	h.watchMoney(ctx)
	if n := queued(); n != 2 {
		t.Fatalf("after the deletion: %d events, want 2", n)
	}
}

// The ledger's queries run on a real database — the refund filter in
// particular, which unit tests cannot reach.
func TestMoneyLoadLive(t *testing.T) {
	h, branch := liveHandler(t)
	ctx := context.Background()
	now := time.Now()
	sold := models.Order{BranchID: branch, Number: "L1", Total: 100, Status: models.StatusDelivered,
		PaymentStatus: models.PayPaid, PaymentMethod: "cash", CreatedAt: now.AddDate(0, -2, 0), UpdatedAt: now}
	refundedLater := sold
	refundedLater.Number, refundedLater.PaymentStatus = "L2", models.PayRefunded
	refundedLater.Refund = &models.CheckRefund{At: now, Amount: 100}
	for _, o := range []models.Order{sold, refundedLater} {
		if _, err := h.Store.Orders.InsertOne(ctx, o); err != nil {
			t.Fatal(err)
		}
	}
	from, to := now.AddDate(0, 0, -1), now.AddDate(0, 0, 1)
	src, err := h.loadMoneySources(ctx, bson.M{"branchId": branch}, from, to)
	if err != nil {
		t.Fatal(err)
	}
	e := moneyEntries(src, from, to)
	if len(e) != 1 || e[0].Class != MoneyRefund {
		t.Fatalf("want only this week's refund of a two-month-old sale, got %+v", e)
	}
}
