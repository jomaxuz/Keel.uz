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

// How much of the branch's trading days a line must sell on before a zero day
// is read as an empty shelf rather than as a quiet one.
//
// ⚠️ **Two thirds, and deliberately high.** Reading a zero as a stock-out is
// inventing demand that was never recorded; doing it to a line that genuinely
// sells three days a week would order for four days that never existed. Above
// this bar the opposite mistake is the likely one, and it is the expensive one:
// an item that ran out is ordered less, so it runs out again.
const regularSellerShare = 2.0 / 3.0

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
	// Days the branch traded but this one did not move, on a line that
	// otherwise moves nearly every day.
	//
	// ⚠️ **A shelf that was empty did not have zero demand — it had zero
	// supply, and the two are the same row in the data.** This is the defect
	// that makes a naive forecast permanently wrong in the one direction that
	// matters: an item that ran out for three days looks like an item that
	// nobody wanted for three days, so it is ordered *less*, so it runs out
	// again. Counted here, excluded from the divisor below, and reported so the
	// screen can say the forecast was measured without them.
	stockouts int
}

// rhythm is how a shelf is refilled: how often, and how long it keeps.
type rhythm struct {
	// Median days between deliveries. Zero when there have been fewer than two.
	every float64
	// Days since the last one arrived — which is what says whether the next is
	// due tomorrow or in a week.
	sinceLast float64
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

	// The days the branch traded at all, by weekday — the divisor.
	//
	// ⚠️ **Trading days, not calendar days.** A restaurant shut on Mondays
	// consumes nothing on eight of the window's fifty-six days; dividing by the
	// calendar would report its Monday demand as a real, measured zero and then
	// order for a Monday it does not open — or, worse, quietly halve a
	// Sunday-only line's average because the shop was closed the rest of the
	// week. A day nobody traded is not a zero, it is an absence of measurement.
	trading := h.tradingDays(ctx, scope, from, to)

	// Gather per ingredient before dividing: which weekdays it moved on, and
	// how many of that weekday's trading days it moved on.
	type acc struct {
		qty  [7]float64
		sold [7]int
	}
	got := map[primitive.ObjectID]*acc{}
	for _, row := range rows {
		if row.ID.WD < 1 || row.ID.WD > 7 {
			continue
		}
		wd := row.ID.WD - 1 // Mongo counts Sunday as 1; time.Weekday counts it 0.
		a := got[row.ID.Ing]
		if a == nil {
			a = &acc{}
			got[row.ID.Ing] = a
		}
		a.qty[wd] += row.Qty
		a.sold[wd] += row.Days
	}

	tradedTotal := 0
	for _, n := range trading {
		tradedTotal += n
	}
	for id, a := range got {
		soldTotal := 0
		for _, n := range a.sold {
			soldTotal += n
		}
		// ⚠️ **A line that normally sells every day, and did not, was
		// probably not on the shelf.** That is the only reading of a zero this
		// system can support — nothing records a stock-out — and it is safe
		// precisely because it is narrow: an item selling on two thirds of the
		// days the branch traded is not an item with quiet days, it is an item
		// with missing days. A weekend-only line falls under the bar and keeps
		// its honest zeros, which is what makes the weekday profile work at
		// all.
		regular := tradedTotal > 0 &&
			float64(soldTotal) >= regularSellerShare*float64(tradedTotal)
		d := demand{days: soldTotal}
		for wd := 0; wd < 7; wd++ {
			divisor := trading[wd]
			if regular {
				// Only the days it was actually available.
				divisor = a.sold[wd]
				d.stockouts += trading[wd] - a.sold[wd]
			}
			if divisor > 0 {
				d.byWeekday[wd] = a.qty[wd] / float64(divisor)
			}
		}
		if d.stockouts < 0 {
			d.stockouts = 0
		}
		out[id] = d
	}
	return out
}

// tradingDays counts, per weekday, the days this branch sold anything at all.
//
// ⚠️ **One question asked once for the whole catalogue.** Whether the doors
// were open is a fact about the branch, not about an ingredient, and asking it
// per ingredient would be two hundred aggregations for one answer.
func (h *Handler) tradingDays(
	ctx context.Context, scope bson.M, from, to time.Time,
) [7]int {
	var out [7]int
	match := bson.M{"reversedAt": nil, "at": bson.M{"$gte": from, "$lt": to}}
	for k, v := range scope {
		match[k] = v
	}
	tz := mongoTZ()
	cur, err := h.Store.StockMoves.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{"$dateToString": bson.M{
				"format": "%Y-%m-%d", "date": "$at", "timezone": tz,
			}},
			"wd": bson.M{"$first": bson.M{
				"$dayOfWeek": bson.M{"date": "$at", "timezone": tz},
			}},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id": "$wd", "days": bson.M{"$sum": 1},
		}}},
	})
	if err != nil {
		return out
	}
	defer cur.Close(ctx)
	var rows []struct {
		WD   int `bson:"_id"`
		Days int `bson:"days"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return out
	}
	for _, row := range rows {
		if row.WD >= 1 && row.WD <= 7 {
			out[row.WD-1] = row.Days
		}
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
		since := 0.0
		if prev != nil {
			since = time.Since(*prev).Hours() / 24
			if since < 0 {
				since = 0
			}
		}
		out[g.ID] = rhythm{
			every: median(gaps), sinceLast: since, shelfLife: median(lives),
			deliveries: len(g.Rows),
		}
	}
	return out
}

// coverDays is how far ahead one ingredient has to be bought for.
//
// ⚠️ **Three stretches of time, and leaving one out is what made the first
// version under-order.** An order placed this morning does not appear on the
// shelf this morning: it arrives with the next delivery, and then it has to
// last until the delivery after that. So the shelf must survive
//
//	until the next delivery  +  one whole delivery cycle  +  a margin
//
// The first version covered only the cycle, which is right on the one morning a
// delivery has just left and short by up to a week on every other morning. The
// classic periodic-review formula, with each of its three terms measured rather
// than typed in: lead time, review period, safety.
//
// ⚠️ **Capped by how long the goods keep.** Ordering three weeks of cover for
// something that lasts five days is not a full shelf, it is a write-off with a
// delay — and the cap is what makes the same arithmetic right for a pharmacy's
// yoghurt and its paracetamol without either being a special case.
//
// ⚠️ **Ordering twice in a day does not order twice**, which is what makes the
// long horizon safe: what is on the shelf and what is already on somebody's
// list are both subtracted, so the second reading of the list asks for what is
// still missing rather than for the same order again.
func coverDays(r rhythm) int {
	every := r.every
	if every <= 0 {
		every = coverWhenUnknown
	}
	// How long until the next delivery is due. ⚠️ Never negative and never more
	// than a whole cycle: a supplier who is three days late is due today, not
	// "minus three days", and one who has never delivered inside the window
	// must not push the horizon past a cycle.
	untilNext := every - r.sinceLast
	if untilNext < 0 {
		untilNext = 0
	}
	if untilNext > every {
		untilNext = every
	}
	// Half a cycle of margin, and never more than a week of it: a fortnightly
	// delivery does not need a fortnight of slack.
	margin := math.Ceil(every / 2)
	if margin > 7 {
		margin = 7
	}
	cover := int(math.Ceil(untilNext + every + margin))
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

// orderQty rounds a suggestion to a number somebody can actually buy.
//
// ⚠️ **Always up, never down.** This screen exists to stop a shelf running out;
// rounding 71.9 kilos down to 71 saves nothing and reintroduces exactly the
// failure. And pieces are whole: "order 2.4 bottles" is a quantity nobody can
// hand over, and the person reading it rounds — in whichever direction they
// feel like, which is the same as this screen not having decided.
func orderQty(qty float64, unit string) float64 {
	if qty <= 0 {
		return 0
	}
	if models.PerUnit(unit) == 1 {
		// Pieces, packets, bottles.
		return math.Ceil(qty)
	}
	// ⚠️ Coarser once the number is big: 0.1 of a kilo matters on a spice and
	// is noise on half a cow, and a list of "71.9" reads as a machine talking
	// to itself rather than as an order.
	if qty >= 10 {
		return math.Ceil(qty)
	}
	return math.Ceil(qty*10) / 10
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
	// ⚠️ **Shipped counts too, and only the market half counts at all.**
	// Something a buyer is carrying home has been asked for and is not on the
	// shelf yet, so it still has to come off the next suggestion — the whole
	// point of this function. But a request answered from the store room is
	// already *in* the balance this suggestion was computed from: subtracting it
	// as well would discount the same kilos twice and quietly stop reordering
	// the things the restaurant moves between its own rooms most often.
	filter := bson.M{
		"status": bson.M{"$in": []string{models.ShoppingSent, models.ShoppingShipped}},
		"source": bson.M{"$ne": models.SourceStore},
	}
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
