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
//
// That overwrite is also the migration story. When the meaning of a field
// changes — as `revenue` just did, from "ordered" to "collected" — the rolling
// window rewrites every day inside it on the next tick, so recent history
// corrects itself with nothing to run by hand. Only rows older than the window
// keep the old meaning, and there is no way to recover what they did not
// record.
//
// **It records what it did.** An empty dashboard used to have two possible
// causes that looked identical — nobody has ordered yet, or this never ran —
// and no way at all to tell them apart. Now the run writes down when it
// happened, how many tenant databases it reached and which it could not, and
// the console reads that back. See models.CollectorRun.
func Run(ctx context.Context, s *repository.Store, days int, trigger string) error {
	started := time.Now()
	run := models.CollectorRun{At: started, Trigger: trigger}

	cur, err := s.Tenants.Find(ctx, bson.M{})
	if err != nil {
		run.Errors = []string{err.Error()}
		save(ctx, s, &run, started)
		return err
	}
	var tenants []models.Tenant
	if err := cur.All(ctx, &tenants); err != nil {
		run.Errors = []string{err.Error()}
		save(ctx, s, &run, started)
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
		run.Tenants++
		rows, err := one(ctx, s, t, days)
		if err != nil {
			// One unreachable database must not stop the other forty-nine.
			log.Printf("aggregate %s: %v", t.Slug, err)
			run.Failed++
			// Capped: a platform-wide outage would otherwise write fifty
			// copies of one sentence into a document the console renders.
			if len(run.Errors) < 5 {
				run.Errors = append(run.Errors, t.Slug+": "+err.Error())
			}
			continue
		}
		run.OK++
		run.Rows += rows
	}
	save(ctx, s, &run, started)
	return nil
}

// save writes the run report. Its own failure is logged and swallowed: losing
// the record of a collection must not fail the collection.
func save(ctx context.Context, s *repository.Store, run *models.CollectorRun, started time.Time) {
	run.DurationMs = time.Since(started).Milliseconds()
	if _, err := s.Collector.UpdateOne(ctx,
		bson.M{"_id": models.CollectorDocID},
		bson.M{"$set": run},
		options.Update().SetUpsert(true),
	); err != nil {
		log.Printf("aggregate: hisobotni yozib bo'lmadi: %v", err)
	}
}

// One collects a single tenant. Exported for the delete path, which takes a
// customer's final numbers before they leave the list.
func One(ctx context.Context, s *repository.Store, t models.Tenant, days int) error {
	_, err := one(ctx, s, t, days)
	return err
}

// one collects a single tenant and reports how many day-rows it wrote.
//
// The row count is what makes an empty chart explainable: zero rows across
// every tenant means nobody ordered in the window, which is a fact rather than
// a fault — and indistinguishable from a broken collector without it.
func one(ctx context.Context, s *repository.Store, t models.Tenant, days int) (int, error) {
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
	//
	// Cancelled orders are **counted but never billed**. Charging a restaurant
	// for an order it cancelled itself is the first argument you will have
	// with a customer, and it is not one worth winning. But the count is worth
	// having: a customer whose cancellations are climbing is one about to
	// phone, and that is invisible if the rows only hold what we can invoice.
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"createdAt": bson.M{"$gte": from}}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{"$dateToString": bson.M{
				"format":   "%Y-%m-%d",
				"date":     "$createdAt",
				"timezone": localZone(),
			}},
			"cancelled": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$eq": bson.A{"$status", "cancelled"}}, 1, 0,
			}}},
			// Everything below counts only what was not cancelled, so the
			// billable figure is unchanged by adding the count above.
			"orders": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$ne": bson.A{"$status", "cancelled"}}, 1, 0,
			}}},
			// **Money counted when it arrives, not when an order is placed.**
			//
			// Cash is real once the courier hands the food over; a card
			// payment is real once the bank says so, which can be earlier.
			// Summing every uncancelled order — which this did — booked a
			// 100 000 so'm order as takings the minute it was placed, and the
			// console's "customer revenue" then sat permanently above the
			// restaurant's own dashboard. Same rule in all three places now.
			//
			// Billing is unaffected: it is Orders × price, and `orders` above
			// still counts every order that reached the kitchen. A restaurant
			// that cooked the food is billed for it whether or not the guest
			// was home.
			"revenue": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$and": bson.A{
					bson.M{"$ne": bson.A{"$status", "cancelled"}},
					bson.M{"$or": bson.A{
						bson.M{"$eq": bson.A{"$paymentStatus", "paid"}},
						bson.M{"$eq": bson.A{"$status", "delivered"}},
					}},
				}}, "$total", 0,
			}}},
		}}},
	}
	cur, err := orders.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	var rows []struct {
		Date      string `bson:"_id"`
		Orders    int    `bson:"orders"`
		Cancelled int    `bson:"cancelled"`
		Revenue   int    `bson:"revenue"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return 0, err
	}

	price := t.PricePerOrder
	for _, r := range rows {
		day := models.TenantDay{
			TenantID:    t.ID,
			Date:        r.Date,
			Orders:      r.Orders,
			Cancelled:   r.Cancelled,
			Revenue:     r.Revenue,
			Billable:    r.Orders * price,
			CollectedAt: time.Now(),
		}
		if _, err := s.Days.UpdateOne(ctx,
			bson.M{"tenantId": t.ID, "date": r.Date},
			bson.M{"$set": day},
			options.Update().SetUpsert(true),
		); err != nil {
			return 0, err
		}
	}
	return len(rows), nil
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
