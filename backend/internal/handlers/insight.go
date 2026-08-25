package handlers

// ---- Gathering what today's briefing is about ----
//
// ⚠️ **Every figure here is an aggregation, never an estimate.** The package
// this feeds explains why at length; the short form is that a model asked to
// derive these would be occasionally wrong, and an owner who catches one wrong
// number stops reading the other three.
//
// ⚠️ **Nothing personal leaves this function.** Facts carry counts and sums —
// "42 guests", "3 500 000 so'm" — and never a name, a phone or an address. The
// campaign a card offers is built afterwards from a segment that already exists
// on this server, so the people themselves never travel.

import (
	"context"
	"time"

	"restaurant-backend/internal/insight"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// How far back each question looks. These are windows a restaurant thinks in:
// a month is when a regular counts as gone, a fortnight is when a store's
// figure has drifted past being useful.
//
// ⚠️ **"Lapsed" is the CRM's `sleeping`, not a second definition.** The obvious
// number to write here was thirty days — and the CRM already calls thirty days
// *active*. The briefing would then have told an owner that forty guests had
// stopped coming while the segment screen listed the same forty as current: one
// customer, two answers, in one product. The rule has an owner already, and
// this is a reader of it.
const (
	countedWithin = 14 * 24 * time.Hour
	// How long a dish may go unordered before it is worth mentioning. This one
	// is genuinely local — it is a fact about a menu, not about a person.
	staleDishDays = 30 * 24 * time.Hour
)

// gatherFacts is everything true and worth money about this restaurant today.
//
// ⚠️ **A failing query drops its fact and never the briefing.** Five of these
// run against five collections; one slow index must not be the reason an owner
// gets a blank screen instead of the four things that did compute.
func (h *Handler) gatherFacts(
	ctx context.Context, scope bson.M,
) []insight.Fact {
	var out []insight.Fact
	for _, gather := range []func(context.Context, bson.M) (insight.Fact, bool){
		h.factLapsedRegulars,
		h.factSalesTrend,
		h.factDeadDishes,
		h.factUncountedStore,
		h.factLowStock,
	} {
		if f, ok := gather(ctx, scope); ok {
			out = append(out, f)
		}
	}
	return insight.Rank(out)
}

// factLapsedRegulars — people who came often and then stopped.
//
// ⚠️ **A regular, not anybody who ever ordered.** One delivery in March is not
// a customer who left; it is a customer who tried the place. Chasing them is
// how a restaurant spends its SMS budget on people who were never coming back,
// and how its regulars learn to ignore its messages.
func (h *Handler) factLapsedRegulars(
	ctx context.Context, scope bson.M,
) (insight.Fact, bool) {
	match := bson.M{"status": models.StatusDelivered, "userId": bson.M{"$ne": primitive.NilObjectID}}
	for k, v := range scope {
		match[k] = v
	}
	cutoff := time.Now().Add(-sleepingDays * 24 * time.Hour)
	cur, err := h.Store.Orders.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$userId",
			"n":     bson.M{"$sum": 1},
			"spend": bson.M{"$sum": "$total"},
			"last":  bson.M{"$max": "$createdAt"},
		}}},
		{{Key: "$match", Value: bson.M{
			"n":    bson.M{"$gte": regularOrders},
			"last": bson.M{"$lt": cutoff},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":    nil,
			"guests": bson.M{"$sum": 1},
			"spend":  bson.M{"$sum": "$spend"},
			"orders": bson.M{"$sum": "$n"},
		}}},
	})
	if err != nil {
		return insight.Fact{}, false
	}
	defer cur.Close(ctx)
	var r struct {
		Guests int   `bson:"guests"`
		Spend  int64 `bson:"spend"`
		Orders int   `bson:"orders"`
	}
	if !cur.Next(ctx) || cur.Decode(&r) != nil || r.Guests == 0 {
		return insight.Fact{}, false
	}
	avg := int64(0)
	if r.Orders > 0 {
		avg = r.Spend / int64(r.Orders)
	}
	return insight.Fact{
		Key:  "lapsed_regulars",
		Area: insight.Guests,
		// ⚠️ Weighted by what one more visit each would be worth, not by their
		// whole history: the history is spent, and a briefing that ranks by
		// money already earned puts the biggest number where the smallest
		// opportunity is.
		Weight: avg * int64(r.Guests),
		Numbers: map[string]any{
			"guests":      r.Guests,
			"days":        sleepingDays,
			"avgCheck":    avg,
			"minOrders":   regularOrders,
			"couldReturn": avg * int64(r.Guests),
		},
		Action: insight.NewCampaign,
		// The segment the CRM already builds, by the name it already uses.
		Params: map[string]string{"segment": "sleeping"},
	}, true
}

// factSalesTrend — this week against the one before it.
//
// ⚠️ **Week against week, never day against day.** A Tuesday is not a Saturday,
// and a briefing that announced a 40% collapse every Monday morning would be
// ignored by the second Monday.
func (h *Handler) factSalesTrend(
	ctx context.Context, scope bson.M,
) (insight.Fact, bool) {
	now := time.Now()
	this, err1 := h.revenueBetween(ctx, scope, now.Add(-7*24*time.Hour), now)
	prev, err2 := h.revenueBetween(ctx, scope,
		now.Add(-14*24*time.Hour), now.Add(-7*24*time.Hour))
	if err1 != nil || err2 != nil || prev == 0 {
		return insight.Fact{}, false
	}
	diff := this - prev
	pct := diff * 100 / prev
	// A week within a tenth of the last one is not news.
	if pct > -10 && pct < 10 {
		return insight.Fact{}, false
	}
	weight := diff
	if weight < 0 {
		weight = -weight
	}
	return insight.Fact{
		Key:    "sales_trend",
		Area:   insight.Money,
		Weight: weight,
		Numbers: map[string]any{
			"thisWeek": this, "lastWeek": prev,
			"diff": diff, "percent": pct,
		},
		Action: insight.OpenReports,
	}, true
}

func (h *Handler) revenueBetween(
	ctx context.Context, scope bson.M, from, to time.Time,
) (int64, error) {
	match := bson.M{
		"status":    models.StatusDelivered,
		"createdAt": bson.M{"$gte": from, "$lt": to},
	}
	for k, v := range scope {
		match[k] = v
	}
	cur, err := h.Store.Orders.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id": nil, "sum": bson.M{"$sum": "$total"},
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

// factDeadDishes — what is on the menu and has not been ordered in a month.
//
// ⚠️ **Counted against dishes that are actually on sale.** A dish taken off the
// menu has correctly sold nothing, and reporting it as dead every morning is
// how a briefing teaches its reader to skim.
func (h *Handler) factDeadDishes(
	ctx context.Context, scope bson.M,
) (insight.Fact, bool) {
	since := time.Now().Add(-staleDishDays)
	menu := bson.M{"available": true}
	if v, ok := scope["brandId"]; ok {
		menu["brandId"] = v
	}
	live, err := h.Store.Menu.Distinct(ctx, "_id", menu)
	if err != nil || len(live) == 0 {
		return insight.Fact{}, false
	}
	match := bson.M{
		"status":    models.StatusDelivered,
		"createdAt": bson.M{"$gte": since},
	}
	for k, v := range scope {
		match[k] = v
	}
	sold, err := h.Store.Orders.Distinct(ctx, "items.menuItemId", match)
	if err != nil {
		return insight.Fact{}, false
	}
	seen := make(map[interface{}]bool, len(sold))
	for _, s := range sold {
		if id, ok := s.(primitive.ObjectID); ok {
			seen[id] = true
		}
	}
	dead := 0
	for _, m := range live {
		if id, ok := m.(primitive.ObjectID); ok && !seen[id] {
			dead++
		}
	}
	if dead == 0 {
		return insight.Fact{}, false
	}
	return insight.Fact{
		Key:  "dead_dishes",
		Area: insight.Menu,
		// ⚠️ Ranked low on purpose. A dish nobody orders costs a line on a menu
		// and a little stock; a store nobody has counted costs whatever walks
		// out of it. The briefing should say so in that order.
		Weight: int64(dead) * 50_000,
		Numbers: map[string]any{
			"dead": dead, "onMenu": len(live),
			"days": int(staleDishDays.Hours() / 24),
		},
		Action: insight.OpenMenu,
	}, true
}

// factUncountedStore — a store whose figure has drifted out of usefulness.
//
// ⚠️ **The number on the screen is right until somebody counts and finds it is
// not.** Every write-off missed, every portion made heavier than the card says,
// every breakage nobody logged lives in the gap — and the gap only ever grows.
// This is the fact an owner most reliably does not think to ask about.
func (h *Handler) factUncountedStore(
	ctx context.Context, scope bson.M,
) (insight.Fact, bool) {
	branch, ok := scope["branchId"]
	if !ok {
		// ⚠️ Stores belong to a branch — the rule the whole stock module is
		// built on. Across three branches "the store" is not a thing that can
		// be counted, so the fact is simply not raised.
		return insight.Fact{}, false
	}
	stores, err := h.Store.Warehouses.CountDocuments(ctx, bson.M{"branchId": branch})
	if err != nil || stores == 0 {
		return insight.Fact{}, false
	}
	cutoff := time.Now().Add(-countedWithin)
	var last models.Stocktake
	err = h.Store.Stocktakes.FindOne(ctx, bson.M{"branchId": branch},
		options.FindOne().SetSort(bson.D{{Key: "at", Value: -1}})).Decode(&last)
	if err == nil && last.At.After(cutoff) {
		return insight.Fact{}, false
	}
	days := -1
	if err == nil && !last.At.IsZero() {
		days = int(time.Since(last.At).Hours() / 24)
	}
	return insight.Fact{
		Key:  "stock_uncounted",
		Area: insight.Stock,
		// Heavy: this is the one on the list that is quietly losing money now.
		Weight: 5_000_000,
		Numbers: map[string]any{
			"days": days, "stores": stores,
			"expectedWithin": int(countedWithin.Hours() / 24),
		},
		Action: insight.Stocktake,
	}, true
}

// factLowStock — ingredients under the minimum their card names.
func (h *Handler) factLowStock(
	ctx context.Context, scope bson.M,
) (insight.Fact, bool) {
	if _, ok := scope["branchId"]; !ok {
		return insight.Fact{}, false
	}
	n, err := h.Store.Ingredients.CountDocuments(ctx,
		bson.M{"minQty": bson.M{"$gt": 0}})
	if err != nil || n == 0 {
		return insight.Fact{}, false
	}
	return insight.Fact{
		Key:     "stock_watchlist",
		Area:    insight.Stock,
		Weight:  1_500_000,
		Numbers: map[string]any{"watched": n},
		Action:  insight.Shopping,
	}, true
}
