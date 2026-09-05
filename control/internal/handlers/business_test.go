package handlers

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"keel-control/internal/repository"
)

// ⚠️ Against a real Mongo, because the rule being sealed **is** the pipeline —
// the same reasoning the billing aggregation's test is written under. Skipped
// where there is no Mongo, so the suite still runs on a machine without one.
func breakdownStore(t *testing.T) *repository.Store {
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
	name := "keel_business_test"
	t.Cleanup(func() {
		bg := context.Background()
		_ = cli.Database(name).Drop(bg)
		_ = cli.Disconnect(bg)
	})
	_ = cli.Database(name).Drop(ctx)
	return repository.New(cli.Database(name), cli)
}

// ⚠️ **"When did they last sell anything" must skip the quiet days.**
//
// The collector writes a row for every day it walks, including the ones where
// nothing happened. So the newest *row* is always yesterday — and a screen
// reading it would tell every customer on the list that they sold something
// last night, including the one that has been dark for a month. That customer
// is the entire reason the column exists.
func TestTheLastSaleSkipsTheQuietDays(t *testing.T) {
	store := breakdownStore(t)
	h := &Handler{Store: store}
	ctx := context.Background()

	quiet := primitive.NewObjectID()
	busy := primitive.NewObjectID()
	rows := []any{
		// Sold on the 1st, then nothing — but the collector kept writing.
		bson.M{"tenantId": quiet, "date": "2026-09-01", "orders": 3, "revenue": 300000},
		bson.M{"tenantId": quiet, "date": "2026-09-02", "orders": 0},
		bson.M{"tenantId": quiet, "date": "2026-09-03", "orders": 0, "tillChecks": 0},
		// Sells at the counter only, which must count as selling.
		bson.M{"tenantId": busy, "date": "2026-09-02", "tillChecks": 12, "tillRevenue": 900000},
		bson.M{"tenantId": busy, "date": "2026-09-03", "tillChecks": 4, "tillRevenue": 250000},
	}
	if _, err := store.Days.InsertMany(ctx, rows); err != nil {
		t.Fatal(err)
	}

	sums, err := h.tenantSums(ctx, "2026-09-01", "2026-09-03")
	if err != nil {
		t.Fatal(err)
	}

	if got := sums[quiet].LastSale; got != "2026-09-01" {
		t.Errorf("the quiet customer's last sale reads %q, want 2026-09-01 — "+
			"a screen saying otherwise hides the one customer worth ringing", got)
	}
	// ⚠️ And a counter-only customer has sold: a shop with no website would
	// otherwise look dark on every screen we have.
	if got := sums[busy].LastSale; got != "2026-09-03" {
		t.Errorf("the counter-only customer's last sale reads %q, want 2026-09-03", got)
	}
	if got := sums[busy].TillChecks; got != 16 {
		t.Errorf("till checks summed to %d, want 16", got)
	}
	if got := sums[quiet].Orders; got != 3 {
		t.Errorf("orders summed to %d, want 3", got)
	}
	// ⚠️ Both kinds of takings are kept apart in the row and added only where
	// the screen says "revenue": one is billed per order and the other is not.
	if got := sums[busy].Revenue; got != 0 {
		t.Errorf("online revenue reads %d for a counter-only customer, want 0", got)
	}
	if got := sums[busy].TillRevenue; got != 1150000 {
		t.Errorf("counter takings summed to %d, want 1 150 000", got)
	}
}

// ⚠️ **A customer outside the window has no row at all**, and the caller reads
// that as "sold nothing" rather than crashing on a missing key — which is what
// makes the idle count possible.
func TestACustomerWithNoRowsIsSimplyIdle(t *testing.T) {
	store := breakdownStore(t)
	h := &Handler{Store: store}
	ctx := context.Background()

	id := primitive.NewObjectID()
	if _, err := store.Days.InsertOne(ctx, bson.M{
		"tenantId": id, "date": "2026-08-01", "orders": 9,
	}); err != nil {
		t.Fatal(err)
	}
	sums, err := h.tenantSums(ctx, "2026-09-01", "2026-09-30")
	if err != nil {
		t.Fatal(err)
	}
	s := sums[id]
	if s.Orders != 0 || s.LastSale != "" {
		t.Errorf("a customer with nothing in the window reads %+v", s)
	}
}
