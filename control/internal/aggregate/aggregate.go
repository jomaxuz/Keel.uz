// Package aggregate walks the tenant databases once a night and writes one row
// per tenant per day.
//
// Why not query live: the overview screen asks "how are all our customers
// doing", and answering that by dialling every tenant database makes the page
// slower with every customer sold — the one curve that must not bend the wrong
// way. The same rows are what the monthly invoice is built from, so the number
// on the dashboard and the number on the bill cannot disagree.
package aggregate

import (
	"context"
	"log"
	"time"

	"keel-control/internal/models"
	"keel-control/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Run collects the given number of days back from today, for every tenant.
//
// Re-collecting is safe and expected: the aggregator crashed, the clock
// slipped, somebody wants yesterday recounted. Rows are keyed by
// (tenant, date) and overwritten, never appended.
func Run(ctx context.Context, s *repository.Store, days int) error {
	cur, err := s.Tenants.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	var tenants []models.Tenant
	if err := cur.All(ctx, &tenants); err != nil {
		return err
	}
	for _, t := range tenants {
		// A closed customer's numbers cannot change any more, and dialling its
		// database every hour forever is the cost that grows with every
		// customer who ever left. The final day is collected once, at the
		// moment of deletion, so nothing is lost by skipping it here.
		if t.Status == models.StatusDeleted {
			continue
		}
		if err := One(ctx, s, t, days); err != nil {
			// One unreachable database must not stop the other forty-nine.
			log.Printf("aggregate %s: %v", t.Slug, err)
		}
	}
	return nil
}

// One collects a single tenant.
func One(ctx context.Context, s *repository.Store, t models.Tenant, days int) error {
	if days < 1 {
		days = 1
	}
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).
		AddDate(0, 0, -(days - 1))

	orders := s.TenantDB(t.DBName()).Collection("order")
	// Grouped in the database. The alternative — pulling a month of orders per
	// tenant into this process — is the same mistake the dashboard stats made
	// before they were moved server-side.
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"createdAt": bson.M{"$gte": from},
			// Cancelled orders are not billed. Charging a restaurant for an
			// order it cancelled itself is the first argument you will have
			// with a customer, and it is not one worth winning.
			"status": bson.M{"$ne": "cancelled"},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{"$dateToString": bson.M{
				"format":   "%Y-%m-%d",
				"date":     "$createdAt",
				"timezone": localZone(),
			}},
			"orders":  bson.M{"$sum": 1},
			"revenue": bson.M{"$sum": "$total"},
		}}},
	}
	cur, err := orders.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	var rows []struct {
		Date    string `bson:"_id"`
		Orders  int    `bson:"orders"`
		Revenue int    `bson:"revenue"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return err
	}

	price := t.PricePerOrder
	for _, r := range rows {
		day := models.TenantDay{
			TenantID:    t.ID,
			Date:        r.Date,
			Orders:      r.Orders,
			Revenue:     r.Revenue,
			Billable:    r.Orders * price,
			CollectedAt: time.Now(),
		}
		if _, err := s.Days.UpdateOne(ctx,
			bson.M{"tenantId": t.ID, "date": r.Date},
			bson.M{"$set": day},
			options.Update().SetUpsert(true),
		); err != nil {
			return err
		}
	}
	return nil
}

// The schedule itself lives in cmd/server: collecting and sweeping have to
// happen in that order, on one clock, and splitting them across two tickers is
// how a demo gets switched off before its last day is counted.

// localZone gives Mongo the same day boundary the owner sees on a calendar.
// Without it every day rolls over five hours early and the last evening's
// orders land on tomorrow's invoice.
func localZone() string {
	name, _ := time.Now().Zone()
	if name == "" || name == "UTC" {
		return "UTC"
	}
	return time.Local.String()
}
