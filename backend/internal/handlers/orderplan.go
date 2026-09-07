package handlers

// ---- How much to buy, and until when ----
//
// ⚠️ **The shopping list knew what had run out and nothing about how fast.** A
// minimum is a line an owner drew once, months ago, in a quiet week; it says
// "tell me when the shelf drops below nine kilos" and it cannot say whether
// nine kilos is tomorrow's cooking or next month's. So the list was a reminder
// rather than an order, and the quantity on it — the gap back up to the
// minimum — was the smallest defensible number rather than the right one.
//
// This is the other half: what the shelf actually consumes, how often the
// deliveries come, and therefore how much has to be there to reach the next
// one. Three measurements, all of them read from documents the restaurant
// already writes:
//
//  1. **Consumption by weekday** — from `stock_movement`, the same rows every
//     balance reads. ⚠️ **By weekday and not as an average**, because Monday is
//     not Saturday: a flat average under-orders for the weekend and over-orders
//     for the start of the week, every week, in exactly the businesses where a
//     weekend is most of the trade.
//  2. **The delivery rhythm** — the median gap between deliveries of that
//     ingredient. ⚠️ **Measured rather than asked for.** A "lead time" field
//     would be a form nobody fills in, and the honest question is not "how long
//     does the supplier take" but "how long until the next delivery normally
//     arrives", which the purchase history answers on its own.
//  3. **The shelf life** — the median gap between a delivery and its expiry
//     date, where the dates are entered. It **caps** the horizon: ordering
//     three weeks of cover for something that lasts five days is not a full
//     shelf, it is a write-off with a delay.
//
// ⚠️ **None of it varies by business type, and that is deliberate** — the same
// rule `models/businesstype.go` draws around the whole stock module. What
// separates a chemist from a kitchen here shows up in the *data* rather than in
// a branch of the code: a pharmacy's deliveries are fortnightly and its yoghurt
// expires, so its horizon comes out long and then gets capped; a kitchen's beef
// arrives twice a week and never carries a date. The one case that needs
// protecting from a forecast is a clothes shop, where a dress sells once and a
// confident "reorder 3" would be nonsense — and it protects itself: a row needs
// enough selling days before it is forecast at all, and a garment never has
// them. A rule written as `if grocery` would have been a rule nobody could
// check, in the one part of the product where a wrong number is silent.

import (
	"context"
	"math"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"restaurant-backend/internal/models"
)

// How far back the demand profile looks. Eight weeks: enough that every weekday
// has eight samples, short enough that a menu changed in the spring is not
// still ordering for itself in the autumn.
const demandWeeks = 8

// How far back the delivery rhythm looks. Half a year, because a fortnightly
// delivery needs a dozen of them before a median means anything.
const rhythmDays = 180

// How many days a row must have sold on before it is forecast at all.
//
// ⚠️ **The guard that keeps a clothes shop out of the forecast**, without the
// code ever asking what kind of shop it is. Three separate days of sales is a
// thing that sells; a dress that left the rail once has one, and its
// "consumption rate" would be a number invented from a single event and printed
// as an order. Below the line the ingredient falls back to the minimum rule —
// which is exactly what this screen did before.
const forecastMinDays = 3

// The horizon when no rhythm can be measured — a week, which is what somebody
// with no delivery history in front of them would say.
const coverWhenUnknown = 7

// demand is one ingredient's consumption, as the weekdays actually ran.
type demand struct {
	// Average consumption per occurrence of each weekday, in purchase units.
	// Indexed 0..6 with Sunday at 0, matching time.Weekday.
	byWeekday [7]float64
	// How many distinct days it moved at all — the "is this forecastable"
	// question, and nothing else.
	days int
}

// rhythm is how a shelf is refilled: how often, and how long it keeps.
type rhythm struct {
	// Median days between deliveries. Zero when there have been fewer than two.
	every float64
	// Median days between a delivery and its expiry date, where dates are
	// entered. Zero when none are.
	shelfLife float64
	// How many deliveries the two figures were measured from, so a screen can
	// say "from three deliveries" rather than presenting a guess as a fact.
	deliveries int
}

// demandProfile reads what each ingredient consumes, by weekday.
//
// ⚠️ **Aggregated in the database and grouped twice.** The first group collapses
// a day's checks into one row per ingredient per day; the second collapses the
// days into one row per ingredient per weekday. What comes back is at most seven
// rows per ingredient rather than eight weeks of movements, on a screen an owner
// opens every morning.
//
// ⚠️ **The day boundary is the kitchen's**, passed to Mongo explicitly: the
// driver speaks UTC, so a Tashkent restaurant's evening trade would otherwise be
// filed under tomorrow — and here that does not merely shift a total, it moves
// Saturday night's cooking onto Sunday's profile.
func (h *Handler) demandProfile(
	ctx context.Context, scope bson.M, from, to time.Time,
) map[primitive.ObjectID]demand {
	out := map[primitive.ObjectID]demand{}
	match := bson.M{"reversedAt": nil, "at": bson.M{"$gte": from, "$lt": to}}
	for k, v := range scope {
		match[k] = v
	}
	tz := mongoTZ()
	day := bson.M{"$dateToString": bson.M{
		"format": "%Y-%m-%d", "date": "$at", "timezone": tz,
	}}
	weekday := bson.M{"$dayOfWeek": bson.M{"date": "$at", "timezone": tz}}
	cur, err := h.Store.StockMoves.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$unwind", Value: "$lines"}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"ing": "$lines.ingredientId", "day": day, "wd": weekday,
			},
			"qty": bson.M{"$sum": bson.M{"$multiply": []any{"$lines.qty", "$qty"}}},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":  bson.M{"ing": "$_id.ing", "wd": "$_id.wd"},
			"qty":  bson.M{"$sum": "$qty"},
			"days": bson.M{"$sum": 1},
		}}},
	})
	if err != nil {
		return out
	}
	defer cur.Close(ctx)
	var rows []struct {
		ID struct {
			Ing primitive.ObjectID `bson:"ing"`
			// Mongo's $dayOfWeek is 1..7 with Sunday at 1.
			WD int `bson:"wd"`
		} `bson:"_id"`
		Qty  float64 `bson:"qty"`
		Days int     `bson:"days"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return out
	}

	// How many times each weekday occurred in the window — the divisor.
	//
	// ⚠️ **Calendar occurrences, never the days it sold on.** A restaurant that
	// is shut on Mondays consumes nothing on eight of the fifty-six days, and
	// dividing by "the Mondays it sold something" would report its Monday
	// demand as if it opened — which is the day the order would then arrive
	// for. A zero day is a real zero.
	occurrences := [7]float64{}
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		occurrences[int(d.Weekday())]++
	}

	for _, row := range rows {
		if row.ID.WD < 1 || row.ID.WD > 7 {
			continue
		}
		wd := row.ID.WD - 1 // Mongo counts Sunday as 1; time.Weekday counts it 0.
		d := out[row.ID.Ing]
		if occurrences[wd] > 0 {
			d.byWeekday[wd] = row.Qty / occurrences[wd]
		}
		d.days += row.Days
		out[row.ID.Ing] = d
	}
	return out
}

// forecast is what this ingredient will take over the next `cover` days.
//
// ⚠️ **The days ahead, named**, rather than a daily average multiplied out. A
// three-day cover taken on a Thursday covers Friday, Saturday and Sunday, and
// in most restaurants that is nearly double an average three days. Multiplying
// is what makes a shopping list run out at exactly the moment it matters.
func (d demand) forecast(now time.Time, cover int) float64 {
	total := 0.0
	for i := 1; i <= cover; i++ {
		total += d.byWeekday[int(now.AddDate(0, 0, i).Weekday())]
	}
	return total
}

// deliveryRhythm reads how often each ingredient arrives, and how long it keeps.
func (h *Handler) deliveryRhythm(
	ctx context.Context, scope bson.M, from time.Time,
) map[primitive.ObjectID]rhythm {
	out := map[primitive.ObjectID]rhythm{}
	match := bson.M{"at": bson.M{"$gte": from}}
	for k, v := range scope {
		match[k] = v
	}
	cur, err := h.Store.Purchases.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$unwind", Value: "$lines"}},
		{{Key: "$project", Value: bson.M{
			"ing": "$lines.ingredientId", "at": 1, "exp": "$lines.expiresAt",
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "at", Value: 1}}}},
		{{Key: "$group", Value: bson.M{
			"_id":  "$ing",
			"rows": bson.M{"$push": bson.M{"at": "$at", "exp": "$exp"}},
		}}},
	})
	if err != nil {
		return out
	}
	defer cur.Close(ctx)
	var groups []struct {
		ID   primitive.ObjectID `bson:"_id"`
		Rows []struct {
			At  time.Time  `bson:"at"`
			Exp *time.Time `bson:"exp"`
		} `bson:"rows"`
	}
	if err := cur.All(ctx, &groups); err != nil {
		return out
	}
	for _, g := range groups {
		var gaps, lives []float64
		var prev *time.Time
		for i := range g.Rows {
			// ⚠️ Local, because every one of these is a date the driver handed
			// back in UTC — the trap CLAUDE.md pays for twice. A gap in whole
			// days measured across the five-hour offset is a day out.
			at := local(g.Rows[i].At)
			if prev != nil {
				gap := at.Sub(*prev).Hours() / 24
				// ⚠️ Two deliveries on one day are one delivery for this
				// purpose — a butcher who sends a second van in the afternoon
				// must not read as "arrives every zero days", which would make
				// the horizon zero and the order nothing.
				if gap >= 0.5 {
					gaps = append(gaps, gap)
				}
			}
			p := at
			prev = &p
			if e := g.Rows[i].Exp; e != nil {
				if life := local(*e).Sub(at).Hours() / 24; life > 0 {
					lives = append(lives, life)
				}
			}
		}
		out[g.ID] = rhythm{
			every: median(gaps), shelfLife: median(lives),
			deliveries: len(g.Rows),
		}
	}
	return out
}

// coverDays is how far ahead one ingredient has to be bought for.
//
// ⚠️ **The rhythm plus a margin, capped by how long the goods keep.** Buying
// exactly to the next delivery leaves nothing for the delivery that comes a day
// late, which is the ordinary case rather than the exception; buying past the
// shelf life is throwing money away on a schedule.
func coverDays(r rhythm) int {
	every := r.every
	if every <= 0 {
		every = coverWhenUnknown
	}
	// Half the rhythm as a margin, and never more than a week of it: a
	// fortnightly delivery does not need a fortnight of slack.
	margin := math.Ceil(every / 2)
	if margin > 7 {
		margin = 7
	}
	cover := int(math.Ceil(every) + margin)
	if r.shelfLife > 0 && float64(cover) > r.shelfLife {
		cover = int(math.Floor(r.shelfLife))
	}
	if cover < 1 {
		cover = 1
	}
	if cover > 60 {
		cover = 60
	}
	return cover
}

// requestedQty is what is already on a shopping list somebody is out with.
//
// ⚠️ **Subtracted from every suggestion, and its absence was a real defect.**
// The list is read in the morning and read again after lunch; without this the
// second reading suggests everything the buyer is at that moment standing in a
// market holding, and the shelf ends up with twice what it needed — which for
// anything with a date on it is a write-off in a week.
func (h *Handler) requestedQty(
	ctx context.Context, branch primitive.ObjectID,
) map[primitive.ObjectID]float64 {
	out := map[primitive.ObjectID]float64{}
	filter := bson.M{"status": models.ShoppingSent}
	if !branch.IsZero() {
		filter["branchId"] = branch
	}
	cur, err := h.Store.BuyOrders.Find(ctx, filter)
	if err != nil {
		return out
	}
	var rows []models.ShoppingOrder
	if err := cur.All(ctx, &rows); err != nil {
		return out
	}
	for _, o := range rows {
		for _, l := range o.Lines {
			if l.IngredientID.IsZero() {
				continue
			}
			out[l.IngredientID] += l.Qty
		}
	}
	return out
}

// median is the middle value, or zero when there is nothing to take one of.
//
// ⚠️ **Median rather than mean, for both figures it serves.** One delivery
// missed over a holiday is a forty-day gap in a list of threes, and an average
// would let that single week decide how much a restaurant buys for the rest of
// the year.
func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return s[mid]
	}
	return (s[mid-1] + s[mid]) / 2
}
