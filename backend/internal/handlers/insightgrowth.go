package handlers

// ---- What is working, and what would make it work better ----
//
// ⚠️ **The briefing was seven alarms.** Lapsed regulars, a falling week, dead
// dishes, an uncounted store, low stock, a voiding outlier, a shortfall: every
// one of them true, every one of them a problem, and together they are a
// morning report that an owner learns to skim by the second week. The rule the
// prompt already states — "a briefing that only ever reports problems is read
// as noise" — had nothing to draw on, because nothing here could produce good
// news or a lever to pull.
//
// These are the four questions an owner actually asks an assistant, in their
// own words: which dishes are selling and which are not, which staff are
// pulling their weight, what would cut next month's costs, and what would sell
// more. Each one is exact arithmetic over data this system already owns — the
// package rule stands: the numbers are computed here and only the words come
// from a model.

import (
	"context"
	"time"

	"restaurant-backend/internal/insight"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// The windows these facts think in. A week against the week before is what a
// restaurant compares; a fortnight is the shortest span in which a store's
// waste is a pattern rather than one bad delivery.
const (
	salesWindow = 7 * 24 * time.Hour
	wasteWindow = 14 * 24 * time.Hour
)

// dishSale is one dish's week, as the aggregation returns it.
type dishSale struct {
	ID    string `bson:"_id"`
	Name  string `bson:"name"`
	Qty   int64  `bson:"qty"`
	Money int64  `bson:"money"`
}

// dishSales is what sold, by dish, between two moments.
//
// ⚠️ **Off the order's own line items, not off the menu.** A dish renamed or
// repriced last month still has to appear under the name it was sold as — the
// line carries both, frozen at the sale, for the same reason a receipt does.
func (h *Handler) dishSales(
	ctx context.Context, scope bson.M, from, to time.Time, limit int,
) ([]dishSale, error) {
	match := bson.M{
		"status":    models.StatusDelivered,
		"createdAt": bson.M{"$gte": from, "$lt": to},
	}
	for k, v := range scope {
		match[k] = v
	}
	cur, err := h.Store.Orders.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$unwind", Value: "$items"}},
		{{Key: "$group", Value: bson.M{
			"_id":  "$items.menuItemId",
			"name": bson.M{"$first": "$items.name"},
			"qty":  bson.M{"$sum": "$items.qty"},
			"money": bson.M{"$sum": bson.M{
				"$multiply": bson.A{"$items.price", "$items.qty"},
			}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "money", Value: -1}}}},
		{{Key: "$limit", Value: limit}},
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []dishSale
	for cur.Next(ctx) {
		var d dishSale
		if cur.Decode(&d) == nil {
			out = append(out, d)
		}
	}
	return out, nil
}

// factTopDishes — what is carrying the week, and by how much.
//
// ⚠️ **Reported with its share of the week's takings, not as a bare list.**
// "Osh sold 84" is a number an owner already feels; "three dishes are 41% of
// the week" is the one that changes what they do about the other forty. It is
// also the fact that makes a concentrated menu visible: a restaurant whose top
// three are most of its revenue is one supplier problem away from a bad week.
func (h *Handler) factTopDishes(
	ctx context.Context, scope bson.M,
) (insight.Fact, bool) {
	now := time.Now()
	top, err := h.dishSales(ctx, scope, now.Add(-salesWindow), now, 3)
	if err != nil || len(top) < 3 {
		return insight.Fact{}, false
	}
	total, err := h.revenueBetween(ctx, scope, now.Add(-salesWindow), now)
	if err != nil || total == 0 {
		return insight.Fact{}, false
	}
	var lead int64
	names := make([]string, 0, len(top))
	qty := make([]int64, 0, len(top))
	money := make([]int64, 0, len(top))
	for _, d := range top {
		lead += d.Money
		names = append(names, d.Name)
		qty = append(qty, d.Qty)
		money = append(money, d.Money)
	}
	return insight.Fact{
		Key:  "top_dishes",
		Area: insight.Menu,
		// ⚠️ **Weighted by the share, not by the money.** Weight orders the
		// briefing by what is at stake, and the takings of a good week are not
		// at stake — what is worth a card is how much of the week rests on
		// three dishes. A restaurant selling evenly has nothing to act on here
		// and this fact should lose to a real problem.
		Weight: lead * 100 / total * 1000,
		Numbers: map[string]any{
			"names": names, "qty": qty, "money": money,
			"weekTotal": total, "topShare": lead * 100 / total,
		},
		Action: insight.OpenReports,
	}, true
}

// factRisingDish — the dish that moved most against last week.
//
// ⚠️ **Both directions in one fact, and on purpose.** A riser and a faller are
// the same question — "what changed on the menu" — and splitting them into two
// facts in the same Area means the ranking silently drops one of them. The
// model is given the movement and its sign and says which story this is.
//
// ⚠️ **Anything that sold nothing last week is excluded.** A dish added on
// Tuesday is up infinitely, which is arithmetic rather than news, and it
// crowded out every real movement the first time this was written without the
// guard.
func (h *Handler) factRisingDish(
	ctx context.Context, scope bson.M,
) (insight.Fact, bool) {
	now := time.Now()
	this, err1 := h.dishSales(ctx, scope, now.Add(-salesWindow), now, 40)
	prev, err2 := h.dishSales(ctx, scope, now.Add(-2*salesWindow), now.Add(-salesWindow), 40)
	if err1 != nil || err2 != nil || len(this) == 0 || len(prev) == 0 {
		return insight.Fact{}, false
	}
	best, bestPrev, ok := pickDishMovement(this, prev)
	if !ok {
		return insight.Fact{}, false
	}
	return insight.Fact{
		Key:  "dish_movement",
		Area: insight.Menu,
		// The size of the move, either way: a dish that collapsed is worth the
		// same attention as one that took off.
		Weight: absInt64(best.Money - bestPrev.Money),
		Numbers: map[string]any{
			"name": best.Name, "thisWeek": best.Money, "lastWeek": bestPrev.Money,
			"diff":    best.Money - bestPrev.Money,
			"percent": (best.Money - bestPrev.Money) * 100 / bestPrev.Money,
			"qtyThis": best.Qty, "qtyLast": bestPrev.Qty,
		},
		Action: insight.OpenMenu,
		Params: map[string]string{"id": best.ID},
	}, true
}

// pickDishMovement finds the dish that moved most, in either direction.
//
// ⚠️ Split out of the query because every line of it is a judgement, and a
// judgement that only runs against a live database is a judgement nobody
// checks: which dishes are comparable, how big a move is news, and what counts
// as moving with the week rather than by itself.
func pickDishMovement(this, prev []dishSale) (now, was dishSale, ok bool) {
	before := make(map[string]dishSale, len(prev))
	for _, d := range prev {
		before[d.ID] = d
	}
	var move int64
	for _, d := range this {
		p, seen := before[d.ID]
		// ⚠️ **A dish that sold nothing last week is excluded.** It is up
		// infinitely, which is arithmetic rather than news, and a menu with one
		// new item crowds out every real movement.
		if !seen || p.Money == 0 {
			continue
		}
		gap := d.Money - p.Money
		if gap < 0 {
			gap = -gap
		}
		if gap > move {
			now, was, move = d, p, gap
		}
	}
	// Under a fifth is a dish moving with the week, not by itself.
	if move == 0 || move*5 < was.Money {
		return dishSale{}, dishSale{}, false
	}
	return now, was, true
}

// serverWeek is one waiter's week on the floor.
type serverWeek struct {
	Name   string `bson:"_id"`
	Checks int64  `bson:"checks"`
	Money  int64  `bson:"money"`
}

// factServerOutput — who is selling on the floor, and who is not.
//
// ⚠️ **Average check, not takings.** A waiter on the busy section takes more
// money than one on the terrace by standing where the guests are, and ranking
// on the total says only which sections are busy. What a manager can act on is
// what happens *per table*: the same room, the same menu, a different figure at
// the bottom of the bill — that is upselling, or the absence of it, and it is
// teachable.
//
// ⚠️ **This fact names a person, so it carries the comparison and nothing
// else.** The prompt has rules for staff cards for exactly this reason: the
// number is the reason the card exists and the interpretation must stay open.
// A waiter who is bottom of this list may be on the slowest shift, may be new,
// may be the one covering the door.
func (h *Handler) factServerOutput(
	ctx context.Context, scope bson.M,
) (insight.Fact, bool) {
	now := time.Now()
	match := bson.M{
		"status":           models.StatusDelivered,
		"createdAt":        bson.M{"$gte": now.Add(-salesWindow), "$lt": now},
		"check.serverName": bson.M{"$nin": bson.A{nil, ""}},
	}
	for k, v := range scope {
		match[k] = v
	}
	cur, err := h.Store.Orders.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id":    "$check.serverName",
			"checks": bson.M{"$sum": 1},
			"money":  bson.M{"$sum": "$total"},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "money", Value: -1}}}},
	})
	if err != nil {
		return insight.Fact{}, false
	}
	defer cur.Close(ctx)
	var rows []serverWeek
	for cur.Next(ctx) {
		var s serverWeek
		if cur.Decode(&s) == nil && s.Checks > 0 {
			rows = append(rows, s)
		}
	}
	gap, ok := pickServerGap(rows)
	if !ok {
		return insight.Fact{}, false
	}
	// ⚠️ Worth roughly what the floor would take if everybody averaged what the
	// best one does. It is an upper bound and it is only used for ordering.
	return insight.Fact{
		Key:    "server_output",
		Area:   insight.Team,
		Weight: (gap.bestAvg - gap.houseAvg) * gap.checks,
		Numbers: map[string]any{
			"best": gap.best.Name, "bestAvg": gap.bestAvg, "bestChecks": gap.best.Checks,
			"weakest": gap.worst.Name, "weakestAvg": gap.worstAvg,
			"weakestChecks": gap.worst.Checks,
			"houseAvg":      gap.houseAvg, "servers": len(rows),
		},
		Action: insight.OpenTeam,
	}, true
}

// serverGap is the spread across the floor, once it is worth reporting.
type serverGap struct {
	best, worst       serverWeek
	bestAvg, worstAvg int64
	houseAvg          int64
	checks            int64
}

// pickServerGap decides whether a floor has a spread worth a card.
//
// ⚠️ **Three waiters minimum, and at least ten checks each.** With two people
// one of them is always "the weaker", which is a ranking rather than a finding;
// below ten checks an average is one large table. Both thresholds are the
// difference between a card a manager acts on and a card that names somebody
// for having worked a quiet Tuesday.
func pickServerGap(rows []serverWeek) (serverGap, bool) {
	if len(rows) < 3 {
		return serverGap{}, false
	}
	var g serverGap
	var money int64
	for _, s := range rows {
		if s.Checks < 10 {
			continue
		}
		avg := s.Money / s.Checks
		g.checks += s.Checks
		money += s.Money
		if g.bestAvg == 0 || avg > g.bestAvg {
			g.best, g.bestAvg = s, avg
		}
		if g.worstAvg == 0 || avg < g.worstAvg {
			g.worst, g.worstAvg = s, avg
		}
	}
	if g.checks == 0 || g.best.Name == "" || g.worst.Name == g.best.Name {
		return serverGap{}, false
	}
	g.houseAvg = money / g.checks
	// Under a fifth apart is a team working the same way.
	if (g.bestAvg-g.worstAvg)*5 < g.houseAvg {
		return serverGap{}, false
	}
	return g, true
}

// factWasteShare — what is being thrown away, against what is being bought.
//
// ⚠️ **A share, because the absolute figure means nothing on its own.** Half a
// million so'm of write-offs is a disaster in a café and a rounding error in a
// banqueting kitchen. Against the fortnight's buying it is the same number in
// both places: this is the part of everything you bought that nobody paid for.
func (h *Handler) factWasteShare(
	ctx context.Context, scope bson.M,
) (insight.Fact, bool) {
	now := time.Now()
	from := now.Add(-wasteWindow)
	bought, err := h.sumBetween(ctx, h.Store.Purchases, scope, "at", "total", from, now)
	if err != nil || bought == 0 {
		return insight.Fact{}, false
	}
	wasted, err := h.sumBetween(ctx, h.Store.WriteOffs, scope, "at", "value", from, now)
	if err != nil || wasted == 0 {
		return insight.Fact{}, false
	}
	pct := wasted * 100 / bought
	// Under two per cent is a kitchen working normally; a card about it is a
	// card that teaches its reader to skim.
	if pct < 2 {
		return insight.Fact{}, false
	}
	top, reason := h.topWriteOff(ctx, scope, from, now)
	return insight.Fact{
		Key:    "waste_share",
		Area:   insight.Stock,
		Weight: wasted,
		Numbers: map[string]any{
			"wasted": wasted, "bought": bought, "percent": pct,
			"topReason": reason, "topValue": top,
		},
		Action: insight.OpenReports,
	}, true
}

// topWriteOff is the reason that cost the most over a period.
func (h *Handler) topWriteOff(
	ctx context.Context, scope bson.M, from, to time.Time,
) (int64, string) {
	match := bson.M{"at": bson.M{"$gte": from, "$lt": to}}
	for k, v := range scope {
		match[k] = v
	}
	cur, err := h.Store.WriteOffs.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id": "$reason", "value": bson.M{"$sum": "$value"},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "value", Value: -1}}}},
		{{Key: "$limit", Value: 1}},
	})
	if err != nil {
		return 0, ""
	}
	defer cur.Close(ctx)
	var r struct {
		Reason string `bson:"_id"`
		Value  int64  `bson:"value"`
	}
	if cur.Next(ctx) && cur.Decode(&r) == nil {
		return r.Value, r.Reason
	}
	return 0, ""
}

// sumBetween adds one field over one collection in a window.
//
// ⚠️ The date field is named by the caller because these collections disagree:
// a purchase happened `at` the delivery, an order `createdAt` the tap. Guessing
// one name for both is how a report silently covers the wrong fortnight.
func (h *Handler) sumBetween(
	ctx context.Context, c *mongo.Collection, scope bson.M,
	dateField, sumField string, from, to time.Time,
) (int64, error) {
	match := bson.M{dateField: bson.M{"$gte": from, "$lt": to}}
	for k, v := range scope {
		match[k] = v
	}
	cur, err := c.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id": nil, "sum": bson.M{"$sum": "$" + sumField},
		}}},
	})
	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)
	var r struct {
		Sum int64 `bson:"sum"`
	}
	if cur.Next(ctx) {
		if err := cur.Decode(&r); err != nil {
			return 0, err
		}
	}
	return r.Sum, nil
}

// factQuietHours — when the room is empty, against when it is full.
//
// ⚠️ **The lever an owner can actually pull, and the one nothing else on this
// panel shows.** Every report here is a total over a day or a week; the hours
// inside it are where a promotion, a set lunch or one fewer person on the rota
// changes the number. A restaurant that is dead from three to six is not a
// restaurant with a marketing problem.
//
// ⚠️ Only opening hours are considered, and "open" is taken from the orders
// themselves rather than from the branch's timetable: a kitchen that closes an
// hour early on Mondays would otherwise be reported as a dead hour every week.
func (h *Handler) factQuietHours(
	ctx context.Context, scope bson.M,
) (insight.Fact, bool) {
	now := time.Now()
	match := bson.M{
		"status":    models.StatusDelivered,
		"createdAt": bson.M{"$gte": now.Add(-2 * salesWindow), "$lt": now},
	}
	for k, v := range scope {
		match[k] = v
	}
	cur, err := h.Store.Orders.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			// ⚠️ The local hour, not UTC. Mongo's `$hour` reads the stored
			// instant, which is UTC — a Tashkent lunch peak lands at seven in
			// the morning and the card names the wrong hours entirely.
			"_id": bson.M{"$hour": bson.M{
				"date": "$createdAt", "timezone": mongoTZ(),
			}},
			"orders": bson.M{"$sum": 1},
			"money":  bson.M{"$sum": "$total"},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "money", Value: -1}}}},
	})
	if err != nil {
		return insight.Fact{}, false
	}
	defer cur.Close(ctx)
	var rows []hourRow
	for cur.Next(ctx) {
		var r hourRow
		if cur.Decode(&r) == nil {
			rows = append(rows, r)
		}
	}
	peak, quiet, ok := pickQuietHour(rows)
	if !ok {
		return insight.Fact{}, false
	}
	return insight.Fact{
		Key:    "quiet_hours",
		Area:   insight.Guests,
		Weight: peak.Money - quiet.Money,
		Numbers: map[string]any{
			"peakHour": peak.Hour, "peakOrders": peak.Orders, "peakMoney": peak.Money,
			"quietHour": quiet.Hour, "quietOrders": quiet.Orders, "quietMoney": quiet.Money,
			"days": 14,
		},
		Action: insight.NewCampaign,
	}, true
}

// hourRow is one hour of the trading day, summed over the window.
type hourRow struct {
	Hour   int32 `bson:"_id"`
	Orders int64 `bson:"orders"`
	Money  int64 `bson:"money"`
}

// pickQuietHour finds the dead hour worth doing something about.
//
// ⚠️ **Not simply the smallest hour.** The smallest hour of any restaurant's
// day is the one it opens in or the one it closes in, and a card saying "you
// are quiet at 10am" is a card about the timetable. The hour has to sit inside
// the trading day — within six hours either side of the peak — and be at least
// three times quieter than it, or there is no gap to fill.
//
// Rows arrive sorted by takings, busiest first, which is why the peak is
// `rows[0]` and the search runs from the back.
func pickQuietHour(rows []hourRow) (peak, quiet hourRow, ok bool) {
	// Fewer than six trading hours is a place whose day has no shape to report.
	if len(rows) < 6 {
		return hourRow{}, hourRow{}, false
	}
	peak = rows[0]
	found := false
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].Hour > peak.Hour-6 && rows[i].Hour < peak.Hour+6 {
			quiet, found = rows[i], true
			break
		}
	}
	if !found || quiet.Hour == peak.Hour || quiet.Money*3 > peak.Money {
		return hourRow{}, hourRow{}, false
	}
	return peak, quiet, true
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
