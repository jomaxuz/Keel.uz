// Package tenantstats reads one customer's database directly, for their own
// card in the console.
//
// This is deliberately the opposite choice from the overview.
//
// The overview asks "how are all our customers doing", and answering that by
// dialling every tenant database makes the page slower with every customer
// sold — the one curve that must not bend the wrong way. That is what the
// nightly aggregate exists for.
//
// One customer's card asks a different question: "what is happening at this
// restaurant right now". Most of the answer is not a time series at all — how
// many couriers they employ, how many are on shift, what is in the kitchen at
// this minute — and none of it can be pre-computed hourly without being wrong
// by the time anybody reads it. One database, one screen, opened on purpose:
// the cost is bounded by the operator's attention, not by the customer count.
//
// Everything here is read-only and defensive. A tenant that is stopped, broken
// or half-migrated must still render a card — a missing number is information,
// a 500 is not.
package tenantstats

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Snapshot is everything one customer's card shows that is not history.
type Snapshot struct {
	Today     DayFigures `json:"today"`
	Yesterday DayFigures `json:"yesterday"`

	// What is in the kitchen or on the road right now — the numbers that make
	// the card worth opening rather than reading yesterday's chart.
	Active ActiveOrders `json:"active"`

	People       People       `json:"people"`
	Staff        Staff        `json:"staff"`
	Couriers     Couriers     `json:"couriers"`
	Menu         Menu         `json:"menu"`
	Reservations Reservations `json:"reservations"`
	// Site traffic today and yesterday. Beside the orders because together
	// they answer what neither can alone: traffic with no orders is a broken
	// checkout, no traffic at all is a marketing problem.
	Traffic Traffic `json:"traffic"`

	// Top dishes over the last 30 days. Says what the restaurant actually
	// sells, which is the first thing anybody asks on a support call.
	TopItems []TopItem `json:"topItems"`

	CollectedAt time.Time `json:"collectedAt"`
	// Set when the tenant's database could not be read. The card renders
	// anyway; the operator needs to know the zeroes are ignorance, not calm.
	Error string `json:"error,omitempty"`
}

type DayFigures struct {
	Orders    int `json:"orders"`
	Cancelled int `json:"cancelled"`
	// Money actually in hand — see `received`. Not every order placed.
	Revenue int `json:"revenue"`
	// Placed, not cancelled, not yet collected.
	Pending int `json:"pending"`
	// How many orders the revenue came from.
	Paid int `json:"paid"`
	// Revenue / Paid, computed here so two screens cannot round it
	// differently.
	AvgOrder int `json:"avgOrder"`
	Delivery int `json:"delivery"`
	Pickup   int `json:"pickup"`
	DineIn   int `json:"dinein"`
}

// ActiveOrders is the kitchen's queue, by stage.
type ActiveOrders struct {
	Pending   int `json:"pending"`
	Confirmed int `json:"confirmed"`
	Preparing int `json:"preparing"`
	OnTheWay  int `json:"onTheWay"`
	Total     int `json:"total"`
}

type People struct {
	Customers int `json:"customers"`
	// Signed up in the last 30 days. Growth, as opposed to size.
	New30d int `json:"new30d"`
	// Ordered in the last 30 days. A base of two thousand accounts where forty
	// people order is a different business from one where four hundred do.
	Active30d int `json:"active30d"`
}

type Staff struct {
	Total  int `json:"total"`
	Active int `json:"active"`
	// Clocked in and not yet out. The one number that says whether the
	// restaurant is actually open for business right now.
	OnShift int `json:"onShift"`
}

type Couriers struct {
	Total  int `json:"total"`
	Active int `json:"active"`
	Free   int `json:"free"`
	Busy   int `json:"busy"`
	Off    int `json:"off"`
}

type Menu struct {
	Items      int `json:"items"`
	Available  int `json:"available"`
	Categories int `json:"categories"`
	Branches   int `json:"branches"`
	Brands     int `json:"brands"`
}

// Traffic is how many people opened the site, not how many ordered.
type Traffic struct {
	TodayVisitors     int `json:"todayVisitors"`
	TodayViews        int `json:"todayViews"`
	YesterdayVisitors int `json:"yesterdayVisitors"`
	// Unique visitors over the last 30 days — the size of the audience, as
	// opposed to today's weather.
	Visitors30d int `json:"visitors30d"`
}

type Reservations struct {
	Today    int `json:"today"`
	Upcoming int `json:"upcoming"`
}

type TopItem struct {
	Name string `json:"name"`
	Qty  int    `json:"qty"`
}

// Collect reads a tenant database.
//
// Every section is independent and every failure is swallowed into a zero: a
// restaurant that has never opened the bookings screen has no `reservation`
// collection, and a card that refused to render over that would be useless
// exactly when a new customer is being set up.
func Collect(ctx context.Context, db *mongo.Database) Snapshot {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	now := time.Now()
	startToday := startOfDay(now)
	startYesterday := startToday.AddDate(0, 0, -1)
	last30 := startToday.AddDate(0, 0, -30)

	s := Snapshot{CollectedAt: now}

	orders := db.Collection("order")
	s.Today = dayFigures(ctx, orders, startToday, startToday.AddDate(0, 0, 1))
	s.Yesterday = dayFigures(ctx, orders, startYesterday, startToday)
	s.Active = activeOrders(ctx, orders)
	s.TopItems = topItems(ctx, orders, last30)

	s.People = People{
		Customers: count(ctx, db.Collection("user"), bson.M{}),
		New30d:    count(ctx, db.Collection("user"), bson.M{"createdAt": bson.M{"$gte": last30}}),
		Active30d: distinctOrderers(ctx, orders, last30),
	}
	s.Staff = Staff{
		Total:  count(ctx, db.Collection("staff"), bson.M{}),
		Active: count(ctx, db.Collection("staff"), bson.M{"isActive": true}),
		// An open shift is one with no clock-out. Matching on absence rather
		// than on a flag: the tenant app never writes "still working", it
		// simply has not written an end yet.
		OnShift: count(ctx, db.Collection("shift"), bson.M{
			"$or": []bson.M{{"out": bson.M{"$exists": false}}, {"out": ""}},
		}),
	}
	couriers := db.Collection("courier")
	s.Couriers = Couriers{
		Total:  count(ctx, couriers, bson.M{}),
		Active: count(ctx, couriers, bson.M{"isActive": true}),
		Free:   count(ctx, couriers, bson.M{"status": "free"}),
		Busy:   count(ctx, couriers, bson.M{"status": "busy"}),
		Off:    count(ctx, couriers, bson.M{"status": "off"}),
	}
	s.Menu = Menu{
		Items:      count(ctx, db.Collection("menu_item"), bson.M{}),
		Available:  count(ctx, db.Collection("menu_item"), bson.M{"isAvailable": true}),
		Categories: count(ctx, db.Collection("category"), bson.M{}),
		Branches:   count(ctx, db.Collection("branch"), bson.M{}),
		Brands:     count(ctx, db.Collection("brand"), bson.M{}),
	}
	reservations := db.Collection("reservation")
	s.Reservations = Reservations{
		Today: count(ctx, reservations, bson.M{
			"at":     bson.M{"$gte": startToday, "$lt": startToday.AddDate(0, 0, 1)},
			"status": bson.M{"$nin": []string{"cancelled"}},
		}),
		Upcoming: count(ctx, reservations, bson.M{
			"at":     bson.M{"$gte": now},
			"status": bson.M{"$nin": []string{"cancelled", "done"}},
		}),
	}
	s.Traffic = Traffic{
		TodayVisitors:     visitors(ctx, db, startToday, startToday.AddDate(0, 0, 1)),
		TodayViews:        views(ctx, db, startToday, startToday.AddDate(0, 0, 1)),
		YesterdayVisitors: visitors(ctx, db, startYesterday, startToday),
		Visitors30d:       visitors(ctx, db, last30, startToday.AddDate(0, 0, 1)),
	}
	return s
}

// visitors counts people, not page views: one row per visitor per day, so a
// count of rows over several days counts a returning visitor once per day.
//
// ⚠️ Which means the 30-day figure is **visitor-days, not distinct people** —
// the row key is hashed with the date on purpose, so the same browser is
// unrecognisable across days. That is a deliberate trade: an audience number
// that is slightly high is worth more than a system that can follow somebody.
func visitors(ctx context.Context, db *mongo.Database, from, to time.Time) int {
	return count(ctx, db.Collection("visit"), bson.M{
		"date": bson.M{"$gte": from.Format("2006-01-02"), "$lt": to.Format("2006-01-02")},
	})
}

func views(ctx context.Context, db *mongo.Database, from, to time.Time) int {
	cur, err := db.Collection("visit").Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"date": bson.M{"$gte": from.Format("2006-01-02"), "$lt": to.Format("2006-01-02")},
		}}},
		{{Key: "$group", Value: bson.M{"_id": nil, "n": bson.M{"$sum": "$views"}}}},
	})
	if err != nil {
		return 0
	}
	var rows []struct {
		N int `bson:"n"`
	}
	if err := cur.All(ctx, &rows); err != nil || len(rows) == 0 {
		return 0
	}
	return rows[0].N
}

// dayFigures totals one window of orders, split the ways an operator asks
// about them.
func dayFigures(ctx context.Context, orders *mongo.Collection, from, to time.Time) DayFigures {
	cur, err := orders.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"createdAt": bson.M{"$gte": from, "$lt": to}}}},
		{{Key: "$group", Value: bson.M{
			"_id": nil,
			"cancelled": bson.M{"$sum": cond(
				bson.M{"$eq": bson.A{"$status", "cancelled"}}, 1, 0)},
			"orders": bson.M{"$sum": cond(
				bson.M{"$ne": bson.A{"$status", "cancelled"}}, 1, 0)},
			// **Money counted when it arrives, not when an order is placed.**
			// Cash is real once the courier hands the food over; a card
			// payment is real once the bank says so, which can be earlier.
			// Adding up every uncancelled order — which this did — showed a
			// 100 000 so'm order placed a minute ago as takings immediately.
			// The same rule as the restaurant's own dashboard, so the two
			// screens cannot disagree about the same day.
			"revenue": bson.M{"$sum": cond(received, "$total", 0)},
			// How many orders that money came from. Needed for the average:
			// dividing takings by every uncancelled order mixes two bases and
			// drags the figure down whenever the kitchen is busy.
			"paid": bson.M{"$sum": cond(received, 1, 0)},
			// Placed, not cancelled, not yet collected: the kitchen's workload
			// and the money still to come. Beside the takings, never inside.
			"pending": bson.M{"$sum": cond(
				bson.M{"$and": bson.A{
					bson.M{"$ne": bson.A{"$status", "cancelled"}},
					bson.M{"$not": received},
				}}, "$total", 0)},
			"delivery": bson.M{"$sum": cond(
				bson.M{"$eq": bson.A{"$type", "delivery"}}, 1, 0)},
			"pickup": bson.M{"$sum": cond(
				bson.M{"$eq": bson.A{"$type", "pickup"}}, 1, 0)},
			"dinein": bson.M{"$sum": cond(
				bson.M{"$eq": bson.A{"$type", "dinein"}}, 1, 0)},
		}}},
	})
	if err != nil {
		return DayFigures{}
	}
	var rows []DayFigures
	if err := cur.All(ctx, &rows); err != nil || len(rows) == 0 {
		return DayFigures{}
	}
	d := rows[0]
	if d.Paid > 0 {
		d.AvgOrder = d.Revenue / d.Paid
	}
	return d
}

// activeOrders counts what has not finished yet.
func activeOrders(ctx context.Context, orders *mongo.Collection) ActiveOrders {
	var a ActiveOrders
	cur, err := orders.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"status": bson.M{"$in": []string{"pending", "confirmed", "preparing", "on_the_way"}},
		}}},
		{{Key: "$group", Value: bson.M{"_id": "$status", "n": bson.M{"$sum": 1}}}},
	})
	if err != nil {
		return a
	}
	var rows []struct {
		Status string `bson:"_id"`
		N      int    `bson:"n"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return a
	}
	for _, r := range rows {
		switch r.Status {
		case "pending":
			a.Pending = r.N
		case "confirmed":
			a.Confirmed = r.N
		case "preparing":
			a.Preparing = r.N
		case "on_the_way":
			a.OnTheWay = r.N
		}
		a.Total += r.N
	}
	return a
}

// topItems is what this restaurant actually sells.
func topItems(ctx context.Context, orders *mongo.Collection, since time.Time) []TopItem {
	out := []TopItem{}
	cur, err := orders.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"createdAt": bson.M{"$gte": since},
			"status":    bson.M{"$ne": "cancelled"},
		}}},
		{{Key: "$unwind", Value: "$items"}},
		{{Key: "$group", Value: bson.M{"_id": "$items.name", "qty": bson.M{"$sum": "$items.qty"}}}},
		{{Key: "$sort", Value: bson.D{{Key: "qty", Value: -1}}}},
		{{Key: "$limit", Value: 8}},
	})
	if err != nil {
		return out
	}
	var rows []struct {
		Name string `bson:"_id"`
		Qty  int    `bson:"qty"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return out
	}
	for _, r := range rows {
		if r.Name != "" {
			out = append(out, TopItem{Name: r.Name, Qty: r.Qty})
		}
	}
	return out
}

// distinctOrderers counts people who ordered, not orders placed.
//
// Keyed by phone rather than by account: a guest who ordered without
// registering, or from a family member's number, is still one customer — the
// same reasoning the call centre's lookup follows.
func distinctOrderers(ctx context.Context, orders *mongo.Collection, since time.Time) int {
	cur, err := orders.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"createdAt": bson.M{"$gte": since},
			"status":    bson.M{"$ne": "cancelled"},
		}}},
		{{Key: "$group", Value: bson.M{"_id": "$customer.phone"}}},
		{{Key: "$count", Value: "n"}},
	})
	if err != nil {
		return 0
	}
	var rows []struct {
		N int `bson:"n"`
	}
	if err := cur.All(ctx, &rows); err != nil || len(rows) == 0 {
		return 0
	}
	return rows[0].N
}

// count is CountDocuments with the error folded into zero.
//
// A collection that does not exist yet — bookings at a restaurant that has
// never opened that screen — is the normal state of a new customer, not a
// failure worth breaking a page over.
func count(ctx context.Context, c *mongo.Collection, filter bson.M) int {
	n, err := c.CountDocuments(ctx, filter, options.Count().SetMaxTime(5*time.Second))
	if err != nil {
		return 0
	}
	return int(n)
}

// received is the aggregation's version of "this money is in hand".
//
// Two ways it arrives, and an order carries both: the bank confirmed a card
// payment, or the courier came back — which is what cash on delivery means and
// is also the terminal state for pickup and dine-in. A cancelled order never
// counts, even if it was paid: that is a refund waiting to happen.
var received = bson.M{"$and": bson.A{
	bson.M{"$ne": bson.A{"$status", "cancelled"}},
	bson.M{"$or": bson.A{
		bson.M{"$eq": bson.A{"$paymentStatus", "paid"}},
		bson.M{"$eq": bson.A{"$status", "delivered"}},
	}},
}}

func cond(test any, yes, no any) bson.M {
	return bson.M{"$cond": bson.A{test, yes, no}}
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}
