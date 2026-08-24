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
	// ⚠️ **A set has no card of its own** — what it costs is what the dishes
	// in it cost. Without this a combo answered "not known" and dropped out of
	// the cost of goods sold entirely, which reads on the financial report as
	// a healthier margin than the kitchen has.
	combo []models.ComboLine
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
			ID         primitive.ObjectID  `bson:"_id"`
			Cost       int                 `bson:"cost"`
			Recipe     []models.RecipeLine `bson:"recipe"`
			ComboItems []models.ComboLine  `bson:"comboItems"`
		}
		if err := cur.All(ctx, &rows); err == nil {
			for _, r := range rows {
				led.dishes[r.ID] = dishCosting{
					recipe: r.Recipe, typed: r.Cost, combo: r.ComboItems,
				}
			}
		}
		// ⚠️ A set's members are costed too, and they were not in the list
		// asked about: nothing sold them by name. Loaded in a second pass
		// rather than a second query per set — a month of orders can hold
		// every combo on the menu.
		led.loadComboMembers(ctx, h)
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
	// ⚠️ **A set costs what is in it**, and only when *all* of it is known.
	// A part-costed combo would report the price of the two dishes somebody
	// happened to write cards for and call it the cost of the set — the same
	// rule an incomplete card follows, for the same reason: a cost that is too
	// low reads as good news on every screen showing a margin.
	if len(d.combo) > 0 {
		if c, ok := l.comboCost(d.combo, at); ok {
			return c, true
		}
	}
	if d.typed > 0 {
		return d.typed, true
	}
	return 0, false
}

// comboCost adds up what a set's dishes cost that day.
//
// ⚠️ **The typed fallback of a member counts**, because it is a real answer
// about that dish; only "nothing known at all" makes the set unpriced.
func (l *costLedger) comboCost(lines []models.ComboLine, at time.Time) (int, bool) {
	total := 0
	for _, c := range lines {
		member, ok := l.dishes[c.MenuItemID]
		if !ok {
			return 0, false
		}
		// A set inside a set cannot happen (validateCombo refuses it), so this
		// reads one level and does not recurse.
		one, known := l.plainCost(member, at)
		if !known {
			return 0, false
		}
		total += one * atLeastOne(c.Qty)
	}
	return total, total > 0
}

// plainCost is one dish's cost, without the set arithmetic above.
func (l *costLedger) plainCost(d dishCosting, at time.Time) (int, bool) {
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

// loadComboMembers pulls in the dishes that sets are made of.
//
// Nothing sold them by name, so they are absent from the list the ledger was
// built for — and a set whose members are unknown is a set with no cost.
func (l *costLedger) loadComboMembers(ctx context.Context, h *Handler) {
	need := []primitive.ObjectID{}
	for _, d := range l.dishes {
		for _, c := range d.combo {
			if _, have := l.dishes[c.MenuItemID]; !have && !c.MenuItemID.IsZero() {
				need = append(need, c.MenuItemID)
			}
		}
	}
	if len(need) == 0 {
		return
	}
	cur, err := h.Store.Menu.Find(ctx, bson.M{"_id": bson.M{"$in": need}})
	if err != nil {
		return
	}
	var rows []struct {
		ID     primitive.ObjectID  `bson:"_id"`
		Cost   int                 `bson:"cost"`
		Recipe []models.RecipeLine `bson:"recipe"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return
	}
	for _, r := range rows {
		l.dishes[r.ID] = dishCosting{recipe: r.Recipe, typed: r.Cost}
	}
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
