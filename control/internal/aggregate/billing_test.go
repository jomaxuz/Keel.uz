package aggregate

import (
	"context"
	"os"
	"testing"
	"time"

	"keel-control/internal/models"
	"keel-control/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ⚠️ Against a real Mongo, because the rule being sealed **is** the pipeline.
//
// The decision that stops a restaurant from cancelling its way out of the bill
// lives in an aggregation expression, and a Go reimplementation of it in a test
// would prove only that two pieces of code written five minutes apart agree.
// Skipped when there is no Mongo to talk to, so the suite still runs on a
// machine that has none.
func mongoOrSkip(t *testing.T) (*repository.Store, string) {
	t.Helper()
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cli, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Skip("mongo yo'q:", err)
	}
	if err := cli.Ping(ctx, nil); err != nil {
		t.Skip("mongo javob bermadi:", err)
	}
	name := "keel_billing_test"
	t.Cleanup(func() {
		bg := context.Background()
		_ = cli.Database(name).Drop(bg)
		_ = cli.Database("t_" + name).Drop(bg)
		_ = cli.Disconnect(bg)
	})
	_ = cli.Database(name).Drop(ctx)
	_ = cli.Database("t_" + name).Drop(ctx)
	return repository.New(cli.Database(name), cli), name
}

// order builds one order as the tenant app writes it.
func order(status string, history []string, queued bool, total int) bson.M {
	now := time.Now()
	events := make([]bson.M, 0, len(history))
	for i, s := range history {
		events = append(events, bson.M{"status": s, "at": now.Add(time.Duration(i) * time.Minute)})
	}
	o := bson.M{
		"createdAt":     now,
		"status":        status,
		"statusHistory": events,
		"total":         total,
	}
	if queued {
		o["queuedAt"] = now
	}
	return o
}

// ⚠️ **The fee must not be removable after the food is handed over.**
//
// Our charge is per order and cancellations are free — which is correct for a
// kitchen that stops an order before cooking it, and an open invitation if the
// same word can be applied on Wednesday to an order delivered on Tuesday. The
// row is rewritten from the order every night, so that click used to pay for
// itself: one restaurant doing it to everything would have owed nothing while
// its guests, couriers and dashboard all behaved normally.
//
// The four orders below are the whole rule.
func TestDeliveredOrdersStayBillableAfterCancellation(t *testing.T) {
	store, name := mongoOrSkip(t)
	ctx := context.Background()
	tenant := models.Tenant{
		ID:            primitive.NewObjectID(),
		Slug:          name,
		PricePerOrder: 1000,
	}

	docs := []any{
		// Ordinary, delivered, paid for.
		order("delivered", []string{"confirmed", "preparing", "on_the_way", "delivered"}, true, 50_000),
		// The honest cancellation: stopped before anybody cooked anything. Free,
		// and it must stay free — this is the promise the rule is protecting.
		order("cancelled", []string{"pending", "cancelled"}, false, 40_000),
		// ⚠️ The attack: delivered, then marked cancelled. Billed anyway.
		order("cancelled", []string{"confirmed", "preparing", "on_the_way", "delivered", "cancelled"}, true, 60_000),
		// The genuine grey case: cooked, sent, never handed over. Not billed —
		// nothing in the data proves the food left the building, and a rule that
		// guessed would accuse a restaurant over a guest who was not home.
		order("cancelled", []string{"confirmed", "preparing", "on_the_way", "cancelled"}, true, 30_000),
	}
	if _, err := store.TenantDB(tenant.DBName()).Collection("order").InsertMany(ctx, docs); err != nil {
		t.Fatal(err)
	}

	if _, err := one(ctx, store, tenant, 1); err != nil {
		t.Fatal(err)
	}

	var day models.TenantDay
	if err := store.Days.FindOne(ctx, bson.M{"tenantId": tenant.ID}).Decode(&day); err != nil {
		t.Fatal(err)
	}

	// Two billable: the delivered one, and the delivered-then-cancelled one.
	if day.Orders != 2 {
		t.Errorf("billable orders = %d, want 2", day.Orders)
	}
	if day.Billable != 2000 {
		t.Errorf("billable = %d, want 2000", day.Billable)
	}
	// All three cancellations are still counted as cancellations: the customer's
	// own view of their day does not change because of how we bill.
	if day.Cancelled != 3 {
		t.Errorf("cancelled = %d, want 3", day.Cancelled)
	}
	// The one worth a person's attention.
	if day.Reversed != 1 {
		t.Errorf("reversed = %d, want 1", day.Reversed)
	}
	// Cooked and cancelled without delivery — visible, not charged.
	if day.CancelledCooked != 1 {
		t.Errorf("cancelledCooked = %d, want 1", day.CancelledCooked)
	}
	// Revenue is a different question and keeps its own rule: money counted when
	// it arrives. Only the delivered, uncancelled order brought any in.
	if day.Revenue != 50_000 {
		t.Errorf("revenue = %d, want 50000", day.Revenue)
	}
}
