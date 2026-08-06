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
	Revenue   int `json:"revenue"`
	// Revenue / Orders, computed here so two screens cannot round it
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
	return s
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
			// Cancelled orders bring no money — the same rule the invoice
			// follows, so the card and the bill cannot disagree.
			"revenue": bson.M{"$sum": cond(
				bson.M{"$ne": bson.A{"$status", "cancelled"}}, "$total", 0)},
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
	if d.Orders > 0 {
		d.AvgOrder = d.Revenue / d.Orders
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

func cond(test any, yes, no any) bson.M {
	return bson.M{"$cond": bson.A{test, yes, no}}
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}
