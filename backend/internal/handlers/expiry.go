package handlers

// ---- What goes out of date, and when ----
//
// ⚠️ **A pharmacy is inspected on this, and a grocery loses money to it.** The
// question is not "how much is on the shelf" — the stockroom answers that — but
// "what did we take in that stops being sellable, and how soon". Those are
// different questions with different documents behind them, and a pharmacy that
// cannot answer the second one cannot open.
//
// ⚠️ **Built from deliveries, not from balances, and it says so on the screen.**
// Consumption keys on the ingredient rather than on the box — the rule the whole
// stockroom is built on, and the one thing the shop work was told not to change.
// So this cannot claim "three of these four boxes are still here"; it reports
// what came in and when it expires, which is what the invoice and the package
// actually say. Inventing the subtraction would produce a figure that is right
// most of the time, and a pharmacist who finds one stock number wrong stops
// believing every other number on the screen.
//
// ⚠️ **Already-expired comes first, not last.** A list sorted only by date
// buries the boxes that are a problem *today* under the ones that will be a
// problem in November — and today's are the only ones somebody has to walk to a
// shelf about.

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// expiringRow is one delivery line that runs out.
type expiringRow struct {
	IngredientID string    `json:"ingredientId"`
	Name         string    `json:"name"`
	Unit         string    `json:"unit"`
	Series       string    `json:"series,omitempty"`
	ExpiresAt    time.Time `json:"expiresAt"`
	// Whole days from today. Negative means it has already passed.
	DaysLeft int `json:"daysLeft"`
	// As delivered. ⚠️ **Not what is left** — see the note at the top.
	Qty float64 `json:"qty"`
	// The delivery it came in on, so somebody can open it.
	PurchaseID string    `json:"purchaseId"`
	At         time.Time `json:"at"`
	Supplier   string    `json:"supplier,omitempty"`
}

// AdminExpiring lists deliveries that are past their date or close to it.
func (h *Handler) AdminExpiring(w http.ResponseWriter, r *http.Request) {
	scope, _, _, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// How far ahead to look. ⚠️ Sixty days by default because that is roughly
	// when a pharmacy can still return a box to its supplier; past that the
	// only options are a discount or a write-off.
	days := 60
	if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 3650 {
			days = n
		}
	}
	now := time.Now()
	until := now.AddDate(0, 0, days)

	filter := bson.M{}
	for k, v := range scope {
		filter[k] = v
	}
	// ⚠️ Only deliveries that carry a date at all. Every purchase written
	// before this field existed has none, and a missing date is "we did not
	// record it" — not "it never expires", and not a row to put on a screen
	// that is read as a to-do list.
	filter["lines.expiresAt"] = bson.M{"$ne": nil, "$lte": until}

	cur, err := h.Store.Purchases.Find(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var purchases []models.Purchase
	if err := cur.All(r.Context(), &purchases); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	names, units := h.stockNames(r, purchases)

	rows := []expiringRow{}
	for _, p := range purchases {
		for _, l := range p.Lines {
			// ⚠️ Re-checked per line: the query matched the *document* because
			// one of its lines expires soon, and a delivery of forty items
			// usually has thirty-nine that do not.
			if l.ExpiresAt == nil || l.ExpiresAt.After(until) {
				continue
			}
			rows = append(rows, expiringRow{
				IngredientID: l.IngredientID.Hex(),
				Name:         names[l.IngredientID],
				Unit:         units[l.IngredientID],
				Series:       l.Series,
				ExpiresAt:    *l.ExpiresAt,
				DaysLeft:     daysBetween(now, *l.ExpiresAt),
				Qty:          l.Qty,
				PurchaseID:   p.ID.Hex(),
				At:           p.At,
				Supplier:     p.Supplier,
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].ExpiresAt.Before(rows[j].ExpiresAt)
	})

	httpx.JSON(w, http.StatusOK, map[string]any{
		"rows": rows,
		"days": days,
		// ⚠️ Sent rather than computed in the browser: "today" on a tablet whose
		// clock is a day out would silently shift every countdown on the screen.
		"now": now,
	})
}

// stockNames answers what each ingredient on these deliveries is called.
//
// ⚠️ **A shop's product name wins over its stock row's.** The row behind a
// product is written by the server and named after the product, but an owner
// who renames the product expects the new name everywhere — and the row is not
// something they can see or edit to fix it.
func (h *Handler) stockNames(
	r *http.Request, purchases []models.Purchase,
) (names map[primitive.ObjectID]string, units map[primitive.ObjectID]string) {
	names = map[primitive.ObjectID]string{}
	units = map[primitive.ObjectID]string{}
	ids := []primitive.ObjectID{}
	for _, p := range purchases {
		for _, l := range p.Lines {
			if _, ok := names[l.IngredientID]; ok {
				continue
			}
			names[l.IngredientID] = ""
			ids = append(ids, l.IngredientID)
		}
	}
	if len(ids) == 0 {
		return names, units
	}
	cur, err := h.Store.Ingredients.Find(r.Context(), bson.M{"_id": bson.M{"$in": ids}})
	if err == nil {
		var ing []models.Ingredient
		if err := cur.All(r.Context(), &ing); err == nil {
			for _, i := range ing {
				names[i.ID] = i.Name
				units[i.ID] = i.Unit
			}
		}
	}
	menuCur, err := h.Store.Menu.Find(r.Context(), bson.M{"stockId": bson.M{"$in": ids}})
	if err == nil {
		var items []models.MenuItem
		if err := menuCur.All(r.Context(), &items); err == nil {
			for _, m := range items {
				name := m.Name
				if len(m.Variant) > 0 {
					// The size is part of what expires: "Ko'ylak" on a list of
					// twelve rows names none of them.
					name += " · " + joinVariant(m.Variant)
				}
				names[m.StockID] = name
			}
		}
	}
	return names, units
}

func joinVariant(v []string) string {
	out := ""
	for i, s := range v {
		if i > 0 {
			out += " / "
		}
		out += s
	}
	return out
}

// daysBetween counts whole days from a to b, negative when b has passed.
//
// ⚠️ **Counted from midnight to midnight, in the restaurant's own zone.** A
// subtraction of timestamps says a box expiring this evening has "0 days left"
// at nine in the morning and "-1" at ten at night, which reads as two different
// facts about the same box on one working day.
func daysBetween(a, b time.Time) int {
	loc := time.Local
	da := time.Date(a.In(loc).Year(), a.In(loc).Month(), a.In(loc).Day(), 0, 0, 0, 0, loc)
	db := time.Date(b.In(loc).Year(), b.In(loc).Month(), b.In(loc).Day(), 0, 0, 0, 0, loc)
	return int(db.Sub(da).Hours() / 24)
}
