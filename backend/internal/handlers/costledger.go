package handlers

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

// ---- Costing a period, day by day ----
//
// ⚠️ **A dish did not cost today's price last month.** The first version of
// this read every ingredient's current price and applied it across the whole
// report — which meant that putting up the price of beef this morning quietly
// changed March's margin, on a screen somebody had already read and quoted.
// A report that moves when nothing about that month moved is a report nobody
// can rely on.
//
// So the ledger resolves rates **as of the day of each sale**, and caches per
// day: a month's report resolves at most thirty-one times rather than once per
// order, and a year's at most a year — always in memory, from one read.
//
// ⚠️ **The recipe itself is not versioned**, and pretending otherwise would be
// the more expensive lie: a card edited today is applied to old sales as if it
// had always been that way. It is the honest limit of this design — the prices
// move constantly and are worth tracking, while a card changes when the dish
// changes, and a dish that changed is arguably a different dish. Said out loud
// here because the next person to notice deserves to find it written down.

type costLedger struct {
	// The dishes asked about: their cards and their typed fallback.
	dishes map[primitive.ObjectID]dishCosting
	// Every ingredient, with its price history.
	ingredients []models.Ingredient
	// Resolved rates, keyed by day.
	byDay map[string]map[primitive.ObjectID]float64
}

type dishCosting struct {
	recipe []models.RecipeLine
	typed  int
}

// costLedgerFor reads what is needed to cost the dishes named, once.
func (h *Handler) costLedgerFor(
	ctx context.Context, ids []primitive.ObjectID,
) *costLedger {
	led := &costLedger{
		dishes: map[primitive.ObjectID]dishCosting{},
		byDay:  map[string]map[primitive.ObjectID]float64{},
	}
	if len(ids) == 0 {
		return led
	}
	cur, err := h.Store.Menu.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err == nil {
		var rows []struct {
			ID     primitive.ObjectID  `bson:"_id"`
			Cost   int                 `bson:"cost"`
			Recipe []models.RecipeLine `bson:"recipe"`
		}
		if err := cur.All(ctx, &rows); err == nil {
			for _, r := range rows {
				led.dishes[r.ID] = dishCosting{recipe: r.Recipe, typed: r.Cost}
			}
		}
	}
	if cur, err := h.Store.Ingredients.Find(ctx, bson.M{}); err == nil {
		_ = cur.All(ctx, &led.ingredients)
	}
	return led
}

// rates gives what everything cost on that day, resolving once per day.
func (l *costLedger) rates(at time.Time) map[primitive.ObjectID]float64 {
	key := at.In(time.Local).Format("2006-01-02")
	if got, ok := l.byDay[key]; ok {
		return got
	}
	got := ratesAt(l.ingredients, at)
	l.byDay[key] = got
	return got
}

// Cost is what one portion of a dish cost on a given day.
//
// ⚠️ The card wins over the typed figure, and an incomplete card costs nothing
// at all rather than costing part of a dish — the same rule everywhere else:
// a card missing an ingredient would otherwise make the dish look cheaper,
// which reads as good news on every screen that shows a margin.
func (l *costLedger) Cost(dishID primitive.ObjectID, at time.Time) (int, bool) {
	d, ok := l.dishes[dishID]
	if !ok {
		return 0, false
	}
	if len(d.recipe) > 0 {
		rates := l.rates(at)
		if recipeComplete(d.recipe, rates) {
			if c := recipeCost(d.recipe, rates); c > 0 {
				return c, true
			}
		}
	}
	if d.typed > 0 {
		return d.typed, true
	}
	return 0, false
}

// fixedCosts is a ledger with one price per dish, for tests and for callers
// that genuinely have no period — the panel's own menu list, where "as of when"
// is always "now".
func fixedCosts(costs map[primitive.ObjectID]int) *costLedger {
	led := &costLedger{
		dishes: map[primitive.ObjectID]dishCosting{},
		byDay:  map[string]map[primitive.ObjectID]float64{},
	}
	for id, c := range costs {
		led.dishes[id] = dishCosting{typed: c}
	}
	return led
}
