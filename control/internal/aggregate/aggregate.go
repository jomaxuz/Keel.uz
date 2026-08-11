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
	"os"
	"strings"
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
	//
	// ⚠️ **"Cancelled" is read from the order's history, not from its current
	// status, and that difference is what stops the obvious way to not pay us.**
	//
	// The fee is per order and this row is rewritten on every nightly pass, from
	// whatever the status says at that moment. So an order delivered on Tuesday
	// and marked cancelled on Wednesday used to come back uncharged — one click
	// in the panel, no evidence left on the row, and the money simply gone. A
	// restaurant doing it to every order would have paid nothing at all while
	// its own dashboard, its couriers and its guests all carried on normally.
	//
	// So an order that **reached `delivered`** is billable for ever, whatever
	// happens to it afterwards. Nothing honest is lost: a genuine cancellation
	// happens before the food is handed over and never touches that state, so
	// the promise "cancelled orders are free" holds exactly as it did. What
	// changes is only that the word cannot be applied retroactively to an order
	// the kitchen already completed.
	//
	// ⚠️ This is not a fraud detector and must not be mistaken for one. It
	// removes the *reward*, which is the only part worth automating: a rule that
	// tried to judge intent would eventually accuse a restaurant of cheating for
	// a guest who refused their food at the door. What it cannot see is recorded
	// beside it (`reversed`, `cancelledCooked`) for a person to look at.
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"createdAt": bson.M{"$gte": from}}}},
		// Did this order ever reach the customer? Derived once, from the history
		// the tenant appends to on every status change, so the three sums below
		// all agree about the same order.
		//
		// ⚠️ `statusHistory` rather than `status`: the current status is a field
		// the restaurant can set back and forth, while the history is a record of
		// what happened. The panel only appends to it — there is no screen that
		// edits it — so "was delivered" survives a later cancellation.
		{{Key: "$addFields", Value: bson.M{
			"delivered": bson.M{"$in": bson.A{"delivered", bson.M{
				"$ifNull": bson.A{"$statusHistory.status", bson.A{}},
			}}},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{"$dateToString": bson.M{
				"format":   "%Y-%m-%d",
				"date":     "$createdAt",
				"timezone": localZone(),
			}},
			"cancelled": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$eq": bson.A{"$status", "cancelled"}}, 1, 0,
			}}},
			// Billable: not cancelled, **or** cancelled after having been
			// delivered. `$delivered` is computed above from the status history,
			// which the panel appends to and cannot rewrite.
			"orders": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$or": bson.A{
					bson.M{"$ne": bson.A{"$status", "cancelled"}},
					"$delivered",
				}}, 1, 0,
			}}},
			// Delivered, then cancelled. Billed by the line above; counted here
			// because it is the shape of both a returned order and a fee being
			// wished away, and only a person can tell which.
			"reversed": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$and": bson.A{
					bson.M{"$eq": bson.A{"$status", "cancelled"}},
					"$delivered",
				}}, 1, 0,
			}}},
			// Cancelled after the kitchen had it, but never delivered. Not
			// billed — the food may genuinely have come back.
			"cancelledCooked": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$and": bson.A{
					bson.M{"$eq": bson.A{"$status", "cancelled"}},
					bson.M{"$not": bson.A{"$delivered"}},
					// ⚠️ `$ifNull` is not decoration: in an aggregation
					// expression a **missing** field is "missing", not null, so
					// a plain `$ne: ["$queuedAt", null]` answers *true* for an
					// order that never reached the kitchen — and every honest
					// cancellation would have been counted as one cancelled
					// after cooking.
					bson.M{"$ne": bson.A{bson.M{"$ifNull": bson.A{"$queuedAt", nil}}, nil}},
				}}, 1, 0,
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
		Date            string `bson:"_id"`
		Orders          int    `bson:"orders"`
		Cancelled       int    `bson:"cancelled"`
		Reversed        int    `bson:"reversed"`
		CancelledCooked int    `bson:"cancelledCooked"`
		Revenue         int    `bson:"revenue"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return 0, err
	}

	// Site traffic for the same window. Its own collection and its own shape,
	// so it is its own pass rather than a join: one row per visitor per day
	// means unique visitors is a count and page views is a sum.
	visitors, views := visitsByDay(ctx, s, t, from)

	price := t.PricePerOrder
	for _, r := range rows {
		day := models.TenantDay{
			TenantID:        t.ID,
			Date:            r.Date,
			Orders:          r.Orders,
			Cancelled:       r.Cancelled,
			Reversed:        r.Reversed,
			CancelledCooked: r.CancelledCooked,
			Visitors:        visitors[r.Date],
			Views:           views[r.Date],
			Revenue:         r.Revenue,
			Billable:        r.Orders * price,
			CollectedAt:     time.Now(),
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

// visitsByDay reads the tenant's own visit rows.
//
// Failures are swallowed into empty maps on purpose: a restaurant whose site
// has never been opened has no such collection, and an install from before
// visit tracking existed has none either. Neither is a reason to lose the
// day's order figures, which is what returning an error here would do.
func visitsByDay(ctx context.Context, s *repository.Store, t models.Tenant, from time.Time) (
	visitors, views map[string]int,
) {
	visitors, views = map[string]int{}, map[string]int{}
	day := from.Format("2006-01-02")
	cur, err := s.TenantDB(t.DBName()).Collection("visit").Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"date": bson.M{"$gte": day}}}},
		{{Key: "$group", Value: bson.M{
			"_id": "$date",
			// One row per visitor, so counting rows counts people.
			"visitors": bson.M{"$sum": 1},
			"views":    bson.M{"$sum": "$views"},
		}}},
	})
	if err != nil {
		return visitors, views
	}
	var rows []struct {
		Date     string `bson:"_id"`
		Visitors int    `bson:"visitors"`
		Views    int    `bson:"views"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return visitors, views
	}
	for _, r := range rows {
		visitors[r.Date], views[r.Date] = r.Visitors, r.Views
	}
	return visitors, views
}

// The schedule itself lives in cmd/server: collecting and sweeping have to
// happen in that order, on one clock, and splitting them across two tickers is
// how a demo gets switched off before its last day is counted.

// localZone gives Mongo the same day boundary the owner sees on a calendar.
// Without it every day rolls over five hours early and the last evening's
// orders land on tomorrow's invoice.
//
// ⚠️ **It must be a name Mongo knows, and the obvious way to build it is not.**
// `time.Local.String()` is the IANA identifier ("Asia/Tashkent") only when Go
// managed to load a named zone; on a machine where it did not it is the literal
// string **"Local"**, and Mongo answers `unrecognized time zone identifier` and
// refuses the whole pipeline. That is not a wrong day boundary — it is **no
// aggregation at all**: no rows, no dashboard, no invoice, for every tenant,
// with the only evidence in a log line the collector swallows per tenant.
//
// The same tzdata trap as the shifts bug in the tenant app, arriving from the
// other side. UTC is the fallback because a day boundary five hours out is a
// small, visible error, while a failed pipeline is a silent total one.
func localZone() string {
	if name := time.Local.String(); name != "" && name != "Local" && name != "UTC" {
		return name
	}
	// Set in every deployment, and the honest second source: if the process was
	// told what zone it is in, use that — but only if it names a zone that
	// actually loads, so a typo falls back rather than breaking the pipeline.
	if tz := strings.TrimSpace(os.Getenv("TZ")); tz != "" && tz != "Local" {
		if _, err := time.LoadLocation(tz); err == nil {
			return tz
		}
	}
	return "UTC"
}
