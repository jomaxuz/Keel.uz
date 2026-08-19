package handlers

import (
	"math"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- Counting the store ----
//
// ⚠️ **The difference is the product**, exactly as it is for the cash drawer.
// A count that stores only what was found has recorded nothing at all: the
// shortfall it exists to surface has been quietly overwritten by the person who
// might have caused it. So the expected figure is computed here, frozen when
// the count is saved, and a count that disagrees cannot be saved without a
// sentence.
//
// ⚠️ **Expected is measured, not assumed.** It is the previous count plus every
// delivery since, less what the cards say the dishes sold used and what was
// written off — four recorded facts. Where there is no previous count it starts
// from zero and the screen says so: "everything that ever arrived, less
// everything accounted for" is a meaningful number the first time, and the
// count itself is what makes the next one exact.

// AdminStocktakeSheet is what to count and what should be there.
func (h *Handler) AdminStocktakeSheet(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	expected, since, err := h.expectedStock(r, scope, time.Now())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var ingredients []models.Ingredient
	if cur, err := h.Store.Ingredients.Find(r.Context(), bson.M{}); err == nil {
		_ = cur.All(r.Context(), &ingredients)
	}
	type sheetRow struct {
		IngredientID string  `json:"ingredientId"`
		Name         string  `json:"name"`
		Unit         string  `json:"unit"`
		Expected     float64 `json:"expected"`
	}
	rows := make([]sheetRow, 0, len(ingredients))
	for _, in := range ingredients {
		// A prep item is not counted on a shelf as itself — it is a pot of
		// sauce made this morning, and what it was made from is already in the
		// count of its ingredients. Counting both would subtract the tomatoes
		// twice.
		if in.MadeInHouse() {
			continue
		}
		rows = append(rows, sheetRow{
			IngredientID: in.ID.Hex(), Name: in.Name, Unit: in.Unit,
			Expected: round3(expected[in.ID]),
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"rows": rows,
		// When the last count was, so the screen can say what "expected" is
		// measured from — without it the figure looks like a stock balance the
		// system has been keeping all along.
		"since": since,
	})
}

// AdminSaveStocktake records a count and what it was out by.
func (h *Handler) AdminSaveStocktake(w http.ResponseWriter, r *http.Request) {
	var in models.Stocktake
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	if in.At.IsZero() || in.At.After(now) {
		in.At = now
	}
	expected, _, err := h.expectedStock(r, scope, in.At)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rates := h.ingredientRates(r.Context())
	var ingredients []models.Ingredient
	if cur, err := h.Store.Ingredients.Find(r.Context(), bson.M{}); err == nil {
		_ = cur.All(r.Context(), &ingredients)
	}
	byID := map[primitive.ObjectID]models.Ingredient{}
	for _, x := range ingredients {
		byID[x.ID] = x
	}

	lines := make([]models.StocktakeLine, 0, len(in.Lines))
	total, off := 0, false
	for _, l := range in.Lines {
		ing, ok := byID[l.IngredientID]
		if !ok || ing.MadeInHouse() {
			continue
		}
		// ⚠️ The expected figure is the server's, never the browser's. A
		// count whose own baseline came from the screen that recorded it can
		// be made to agree with anything.
		exp := round3(expected[l.IngredientID])
		diff := round3(l.Counted - exp)
		value := writeOffValue(ing, math.Abs(diff), in.At, rates)
		if diff < 0 {
			value = -value
		}
		if diff != 0 {
			off = true
		}
		total += value
		lines = append(lines, models.StocktakeLine{
			IngredientID: l.IngredientID, Counted: l.Counted,
			Expected: exp, Diff: diff, Value: value,
		})
	}
	if len(lines) == 0 {
		httpx.Error(w, http.StatusBadRequest, "hech bo'lmasa bitta masalliq kerak")
		return
	}
	in.Note = clampText(in.Note, 400)
	// ⚠️ A discrepancy cannot be saved silently — the cash drawer's rule, for
	// the same reason: a variance nobody explained is a variance nobody can
	// use, and the explanation is only available on the day.
	if off && in.Note == "" {
		httpx.Error(w, http.StatusBadRequest, "farq bor — sababini yozing")
		return
	}
	in.Lines = lines
	in.Value = total
	in.By = h.adminName(r)
	in.CreatedAt = now
	if s, err := h.adminScope(r); err == nil && in.BranchID.IsZero() {
		in.BranchID = h.scopeBranch(r, s)
	}
	res, err := h.Store.Stocktakes.InsertOne(r.Context(), in)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	in.ID = oidOf(res.InsertedID)
	h.logAction(r, "stocktake.create", "stocktake", in.ID.Hex(), "", in.Note)
	httpx.JSON(w, http.StatusCreated, in)
}

// AdminListStocktakes returns past counts, newest first.
func (h *Handler) AdminListStocktakes(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := bson.M{}
	for k, v := range scope {
		filter[k] = v
	}
	cur, err := h.Store.Stocktakes.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(50))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var rows []models.Stocktake
	_ = cur.All(r.Context(), &rows)
	if rows == nil {
		rows = []models.Stocktake{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"stocktakes": rows})
}

// expectedStock is what should be on the shelves at a moment, and since when.
//
// ⚠️ Every part of it is a recorded fact: the last count, the deliveries after
// it, what the cards say the dishes sold used, and what was written off. None
// of it is a running balance the system has been keeping — which is why the
// screen is told when the measurement starts.
func (h *Handler) expectedStock(
	r *http.Request, scope bson.M, at time.Time,
) (map[primitive.ObjectID]float64, *time.Time, error) {
	out := map[primitive.ObjectID]float64{}

	filter := bson.M{}
	for k, v := range scope {
		filter[k] = v
	}
	filter["at"] = bson.M{"$lte": at}
	var last models.Stocktake
	var since *time.Time
	err := h.Store.Stocktakes.FindOne(r.Context(), filter,
		options.FindOne().SetSort(bson.D{{Key: "at", Value: -1}})).Decode(&last)
	if err == nil {
		for _, l := range last.Lines {
			out[l.IngredientID] = l.Counted
		}
		t := last.At
		since = &t
	}

	var ingredients []models.Ingredient
	if cur, err := h.Store.Ingredients.Find(r.Context(), bson.M{}); err == nil {
		_ = cur.All(r.Context(), &ingredients)
	}
	in, _ := h.deliveredInPeriod(r, scope, since, &at)
	used := h.consumedInPeriod(r, scope, since, &at, ingredients)
	written, _ := h.writtenOffInPeriod(r, scope, since, &at)
	for id, q := range in {
		out[id] += q
	}
	for id, q := range used {
		out[id] -= q
	}
	for id, q := range written {
		out[id] -= q
	}
	return out, since, nil
}
