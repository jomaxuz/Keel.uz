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
	scope, sc, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	h.stocktakeSheet(w, r, scope, sc.BrandID)
}

// stocktakeSheet is the sheet itself, shared by the panel and the phone.
//
// ⚠️ **One arithmetic, two doors.** The person counting the bar with a phone in
// their hand and the owner checking it afterwards on the panel have to be shown
// the same expected figure, or the variance they end up arguing about is
// between two of our screens rather than between the shelf and the books. The
// same reason `composeOrder` is shared by the website and the call centre.
func (h *Handler) stocktakeSheet(
	w http.ResponseWriter, r *http.Request, scope bson.M, brand primitive.ObjectID,
) {
	// ⚠️ **A count is one room, so the sheet is one store.** Handing somebody
	// walking into the bar a list that also has forty kitchen ingredients on it
	// is how counts get abandoned halfway and saved anyway — and a half-counted
	// list saves zeros for everything nobody reached, which reads as a
	// catastrophic shortfall the next morning.
	warehouse, err := optionalObjectID(r.URL.Query().Get("warehouseId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "ombor noto'g'ri")
		return
	}
	byWarehouse, sinceOf, err := h.expectedStockByWarehouse(r, scope, brand, time.Now())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	expected := byWarehouse[warehouse]
	since := sinceOf[warehouse]
	ingredients := h.scopedIngredients(r.Context(), brand)
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
		// Only this store's shelves. An ingredient nobody has filed lives in
		// the undivided store, which is what the zero id means.
		if in.WarehouseID != warehouse {
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
	scope, sc, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.BranchID.IsZero() {
		in.BranchID = h.scopeBranch(r, sc)
	}
	h.saveStocktake(w, r, in, scope, sc.BrandID, h.adminName(r))
}

// saveStocktake records a count, whoever took it and on whichever screen.
//
// ⚠️ Shared for the same reason the sheet is: the frozen expected figure, the
// variance and the rule that a difference needs a sentence have to be one
// implementation. Two copies would disagree the first time either was touched,
// and the disagreement would surface as a count that "saved differently on the
// tablet".
func (h *Handler) saveStocktake(
	w http.ResponseWriter, r *http.Request, in models.Stocktake,
	scope bson.M, brand primitive.ObjectID, by string,
) {
	now := time.Now()
	if in.At.IsZero() || in.At.After(now) {
		in.At = now
	}
	byWarehouse, _, err := h.expectedStockByWarehouse(r, scope, brand, in.At)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ The store comes off the posted count, and every line is checked
	// against **that** store's figures. A line for an ingredient kept somewhere
	// else is dropped below rather than counted here: a count of the bar that
	// silently accepted a kitchen ingredient would reset the kitchen's starting
	// point to a number nobody walked in and looked at.
	expected := byWarehouse[in.WarehouseID]
	rates := h.ingredientRates(r.Context())
	ingredients := h.scopedIngredients(r.Context(), brand)
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
		// Not kept in the store being counted — see above.
		if ing.WarehouseID != in.WarehouseID {
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
	in.By = by
	in.CreatedAt = now
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
//
// ⚠️ **Each store is measured from its own last count.** The bar is counted on
// a Sunday and the kitchen on a Wednesday, so one "since" for the branch would
// measure half the ingredients from a date nobody counted them on — and the
// error lands entirely on whichever store was counted less recently, which is
// the store the owner is least sure about already. `since` is returned as the
// **oldest** of them, because it is the honest answer to the one question the
// screen asks with it: how far back does any of this reach.
func (h *Handler) expectedStock(
	r *http.Request, scope bson.M, brand primitive.ObjectID, at time.Time,
) (map[primitive.ObjectID]float64, *time.Time, error) {
	byWarehouse, since, err := h.expectedStockByWarehouse(r, scope, brand, at)
	if err != nil {
		return nil, nil, err
	}
	out := map[primitive.ObjectID]float64{}
	for _, m := range byWarehouse {
		for id, q := range m {
			out[id] = q
		}
	}
	var oldest *time.Time
	for _, t := range since {
		if t == nil {
			// ⚠️ A store that has never been counted reaches all the way back,
			// and saying so is the point: the screen's caveat has to describe
			// the weakest half of the answer, not the strongest.
			return out, nil, nil
		}
		if oldest == nil || t.Before(*oldest) {
			oldest = t
		}
	}
	return out, oldest, nil
}

// expectedStockByWarehouse is the same arithmetic, kept per store.
//
// The map keys are warehouse ids, with the zero id standing for the undivided
// store — every ingredient on a restaurant that has not split one, and every
// ingredient nobody has filed yet.
func (h *Handler) expectedStockByWarehouse(
	r *http.Request, scope bson.M, brand primitive.ObjectID, at time.Time,
) (map[primitive.ObjectID]map[primitive.ObjectID]float64,
	map[primitive.ObjectID]*time.Time, error) {

	// ⚠️ Narrowed to the brand in view: an unfiltered read put another brand's
	// stores into `stores` and its ingredients onto the balance screen.
	ingredients := h.scopedIngredients(r.Context(), brand)
	// Which store each ingredient is kept in, and which stores exist at all.
	home := map[primitive.ObjectID]primitive.ObjectID{}
	stores := map[primitive.ObjectID]bool{primitive.NilObjectID: true}
	for _, in := range ingredients {
		home[in.ID] = in.WarehouseID
		stores[in.WarehouseID] = true
	}

	out := map[primitive.ObjectID]map[primitive.ObjectID]float64{}
	since := map[primitive.ObjectID]*time.Time{}
	for wh := range stores {
		out[wh] = map[primitive.ObjectID]float64{}
		since[wh] = nil
	}

	// ---- Each store's own last count ----
	base := bson.M{}
	for k, v := range scope {
		base[k] = v
	}
	base["at"] = bson.M{"$lte": at}
	for wh := range stores {
		filter := bson.M{}
		for k, v := range base {
			filter[k] = v
		}
		// ⚠️ The undivided store matches counts with no warehouse **and** counts
		// saved before the field existed — a Mongo filter on the zero id would
		// not match a document that has no such field, so every count taken
		// before this shipped would stop being anybody's starting point.
		if wh.IsZero() {
			filter["$or"] = []bson.M{
				{"warehouseId": bson.M{"$exists": false}},
				{"warehouseId": primitive.NilObjectID},
			}
		} else {
			filter["warehouseId"] = wh
		}
		var last models.Stocktake
		if err := h.Store.Stocktakes.FindOne(r.Context(), filter,
			options.FindOne().SetSort(bson.D{{Key: "at", Value: -1}})).
			Decode(&last); err != nil {
			continue
		}
		for _, l := range last.Lines {
			// A count may list an ingredient that has since moved store; it
			// belongs to wherever it lives now, not to wherever it was counted.
			if home[l.IngredientID] == wh {
				out[wh][l.IngredientID] = l.Counted
			}
		}
		t := last.At
		since[wh] = &t
	}

	// ---- Everything that moved since ----
	//
	// ⚠️ Read once per store rather than once per ingredient: these are
	// aggregations over every order, delivery and write-off in the window, and
	// a restaurant with four stores would otherwise run all of it four times on
	// a screen that is opened all day.
	for wh := range stores {
		from := since[wh]
		in, _ := h.deliveredInPeriod(r, scope, from, &at)
		used := h.consumedInPeriod(r, scope, from, &at, ingredients)
		written, _ := h.writtenOffInPeriod(r, scope, from, &at)
		// ⚠️ Moved stock is the fifth fact, and without it a transfer looks
		// exactly like a theft from one store and a miscount in the other —
		// which is the pair of numbers a stocktake exists to rule out.
		movedIn, movedOut := h.transferredInPeriod(r, scope, from, &at)
		add := func(m map[primitive.ObjectID]float64, sign float64) {
			for id, q := range m {
				if home[id] != wh {
					continue
				}
				out[wh][id] += sign * q
			}
		}
		add(in, 1)
		add(movedIn, 1)
		add(used, -1)
		add(written, -1)
		add(movedOut, -1)
	}
	return out, since, nil
}
