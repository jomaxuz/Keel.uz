package handlers

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

// ---- What a plate costs the kitchen ----
//
// ⚠️ **The field is `json:"-"` on the model, and that is the whole design.**
// A dish is serialised to the public by a dozen handlers — the menu, one dish,
// the recommendations, favourites, the basket, combo contents, the Telegram
// mini app — and a guard added to each of them is a guard somebody forgets when
// they write the thirteenth. What a plate costs is the one number in that
// document a competitor across the street would pay for, so the safe default is
// "never sent" and the panel opts in explicitly, here.
//
// ⚠️ **Zero means "not known", never "free".** Nothing in this system can work
// a cost out — there are no recipes and no stock — so it is typed by hand, and
// most restaurants will type it for the ten dishes that matter and never for
// the rest. Every screen that adds these up has to say how much of the menu it
// covered.

// menuItemIO is a dish as the **panel** reads and writes it: the public shape
// plus the cost.
//
// The embedded field is `json:"-"`, so the outer one is the only way in or out
// — there is no second spelling to get wrong.
type menuItemIO struct {
	models.MenuItem
	// A pointer on the way in: "not sent" and "set to zero" are different
	// answers, and a form that omits the field must not wipe what is stored
	// (the same trap `soldOut` and `kioskSecret` are guarded against).
	Cost *int `json:"cost"`
}

// withCost is one dish on its way to the panel.
func withCost(m models.MenuItem) menuItemIO {
	c := m.Cost
	return menuItemIO{MenuItem: m, Cost: &c}
}

// withCosts is a list of them.
func withCosts(items []models.MenuItem) []menuItemIO {
	out := make([]menuItemIO, 0, len(items))
	for _, m := range items {
		out = append(out, withCost(m))
	}
	return out
}

// keepCost decides what a save should store.
//
// ⚠️ **A missing field keeps the stored value.** `UpdateMenuItem` replaces the
// whole document, so a panel that has not been taught about costs — or a script,
// or an older tab left open — would otherwise erase every cost in the menu by
// saving a dish's name.
func (h *Handler) keepCost(
	ctx context.Context, id primitive.ObjectID, incoming *int,
) int {
	if incoming != nil {
		if *incoming < 0 {
			return 0
		}
		return *incoming
	}
	if id.IsZero() {
		return 0
	}
	var row struct {
		Cost int `bson:"cost"`
	}
	if err := h.Store.Menu.FindOne(ctx, bson.M{"_id": id}).Decode(&row); err == nil {
		return row.Cost
	}
	return 0
}

// ---- The report side ----
//
// ⚠️ **The cost is read from the menu, not frozen onto the order**, and that is
// the opposite of the rule the price follows. The reason is the same one that
// puts ИКПУ on the menu and the price on the order: the price is what the guest
// agreed to and must never move afterwards, while the cost is a fact about the
// **product** — when the accountant corrects a wrong figure, every report built
// on it should correct with it, including last month's.
//
// The cost of ingredients also moves constantly, and freezing a wrong number
// onto twelve thousand order lines would leave nothing anybody could fix.

// dishCosts reads the per-portion cost of the dishes named.
func (h *Handler) dishCosts(
	ctx context.Context, ids []primitive.ObjectID,
) map[primitive.ObjectID]int {
	out := map[primitive.ObjectID]int{}
	if len(ids) == 0 {
		return out
	}
	cur, err := h.Store.Menu.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return out
	}
	var rows []struct {
		ID   primitive.ObjectID `bson:"_id"`
		Cost int                `bson:"cost"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return out
	}
	for _, r := range rows {
		// ⚠️ Zero is left out of the map entirely, so "not known" stays
		// distinguishable from "known to be nothing" everywhere downstream —
		// which is what lets a screen say how much of the menu it covered
		// instead of quietly reporting a margin of 100% on the rest.
		if r.Cost > 0 {
			out[r.ID] = r.Cost
		}
	}
	return out
}
