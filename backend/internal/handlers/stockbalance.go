package handlers

// ---- What is on the shelf, and how it got there ----
//
// ⚠️ **The screen the whole inventory module was missing.** Everything before
// this recorded *movements* — a delivery, a write-off, a count — and the one
// report that read them back (`stockreport.go`) is a flow: how much came in and
// went out over a period. Nobody had built the question an owner actually walks
// into the store with: **what is here now, and what is it worth.**
//
// ⚠️ **It is an estimate and every screen that shows it has to say so.** The
// figure is the last count, plus deliveries, less what the tech cards say the
// dishes used, less write-offs — four recorded facts, not a running balance
// this system has been keeping. It drifts exactly as far as the kitchen drifts
// from its cards, and as far back as the last count. A restaurant that believes
// a stock number and finds it wrong stops believing the panel entirely, so the
// number is always served beside the date it is measured from.
//
// ⚠️ **Per store.** The bar is counted on a Sunday and the kitchen on a
// Wednesday; one figure for the branch would be measured from a date nobody
// counted half of it on — see expectedStockByWarehouse.

import (
	"math"
	"net/http"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// balanceRow is one line of the store.
type balanceRow struct {
	IngredientID string `json:"ingredientId"`
	Name         string `json:"name"`
	Unit         string `json:"unit"`
	WarehouseID  string `json:"warehouseId"`
	// In purchase units — kilos, litres, pieces — because that is how it is
	// counted on a shelf and how it was bought.
	Qty float64 `json:"qty"`
	// What that much is worth at today's price. ⚠️ Today's, not the price it
	// came in at: this answers "what is standing in the store", and a store is
	// worth what it would cost to replace.
	Value int `json:"value"`
	// Warn below this, in purchase units. Zero means the owner never asked.
	MinQty float64 `json:"minQty,omitempty"`
	Low    bool    `json:"low,omitempty"`
	// ⚠️ **A prep item is not on a shelf as itself** — what it was made from is,
	// and that is counted directly. Marked rather than hidden: an owner looking
	// for "Sous" should find it here with the reason, not conclude the list is
	// incomplete.
	Made bool `json:"made,omitempty"`
}

// AdminStockBalances is the store, store by store.
func (h *Handler) AdminStockBalances(w http.ResponseWriter, r *http.Request) {
	scope, branch, brand, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	byWarehouse, since, err := h.expectedStockByWarehouse(r, scope, brand, branch, time.Now())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	ingredients := h.scopedIngredients(r.Context(), brand)
	placed := h.placementsIn(r.Context(), branch)
	rates := h.ingredientRates(r.Context())

	rows := make([]balanceRow, 0, len(ingredients))
	// ⚠️ Totalled per store rather than for the branch alone. "The bar holds
	// four million" is a sentence somebody can act on; the branch total is one
	// they can only nod at.
	value := map[primitive.ObjectID]int{}
	for _, in := range ingredients {
		store := placed[in.ID]
		qty := byWarehouse[store][in.ID]
		// The rate is per recipe unit (gram, millilitre, piece); the quantity
		// is in purchase units, so one has to be converted to meet the other.
		per := float64(models.PerUnit(in.Unit))
		worth := int(math.Round(qty * per * rates[in.ID]))
		if !in.MadeInHouse() {
			value[store] += worth
		}
		rows = append(rows, balanceRow{
			IngredientID: in.ID.Hex(),
			Name:         in.Name,
			Unit:         in.Unit,
			WarehouseID:  store.Hex(),
			Qty:          round3(qty),
			Value:        worth,
			MinQty:       in.MinQty,
			// ⚠️ Only where a minimum was set. Zero means "do not warn me" —
			// nobody tracks a minimum for cinnamon, and a list where every line
			// eventually turns red is a list nobody reads.
			Low:  in.MinQty > 0 && qty < in.MinQty,
			Made: in.MadeInHouse(),
		})
	}
	// Short first, then by name: the reason this screen gets opened is
	// something running out, and that answer must not be halfway down a list of
	// two hundred.
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Low != rows[j].Low {
			return rows[i].Low
		}
		return rows[i].Name < rows[j].Name
	})

	// The stores themselves, so the screen can name them without a second call.
	stores := []models.Warehouse{}
	filter := bson.M{}
	for k, v := range scope {
		filter[k] = v
	}
	if cur, err := h.Store.Warehouses.Find(r.Context(), filter, options.Find().
		SetSort(bson.D{{Key: "sort", Value: 1}})); err == nil {
		_ = cur.All(r.Context(), &stores)
	}

	// ⚠️ **Per store, and null where a store has never been counted.** That is
	// the difference between "measured from 3 March" and "everything that ever
	// arrived, less everything accounted for" — and the second is a number
	// nobody should order against without being told.
	measured := map[string]any{}
	total := map[string]int{}
	for id, t := range since {
		measured[id.Hex()] = t
	}
	for id, v := range value {
		total[id.Hex()] = v
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"rows":       rows,
		"warehouses": stores,
		"since":      measured,
		"value":      total,
	})
}

// ---- One ingredient, from one date to another ----
//
// ⚠️ **The report that answers "where did it go".** A balance says forty kilos
// are missing; only this says whether they were sold, thrown away, or never
// counted in the first place. It is the same four facts the balance is built
// from, taken apart instead of added up — which is why it cannot disagree with
// the number that sent somebody here.

type movementDoc struct {
	At   time.Time `json:"at"`
	Kind string    `json:"kind"`
	Qty  float64   `json:"qty"`
	Note string    `json:"note,omitempty"`
}

// AdminStockMovement is one ingredient's opening, ins, outs and closing.
func (h *Handler) AdminStockMovement(w http.ResponseWriter, r *http.Request) {
	scope, branch, brand, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := objectID(r.URL.Query().Get("ingredientId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "masalliq tanlanmagan")
		return
	}
	from, to, err := parseRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// ⚠️ **An open period is not a window.** Both ends are needed here: the
	// opening balance is the arithmetic run to a date, and a nil one would mean
	// "since the beginning of time", which is a different report.
	if from == nil || to == nil {
		httpx.Error(w, http.StatusBadRequest, "davrni tanlang")
		return
	}

	// ⚠️ The brand lives **inside** the filter: an id alone never selects a
	// document, or a manager reads another brand's buying prices by pasting one.
	// Out of scope is a 404 — they should not learn it exists.
	var ing models.Ingredient
	if err := h.Store.Ingredients.FindOne(r.Context(),
		Scope{BrandID: brand}.brandFilter(bson.M{"_id": id})).Decode(&ing); err != nil {
		httpx.Error(w, http.StatusNotFound, "masalliq topilmadi")
		return
	}

	ingredients := h.scopedIngredients(r.Context(), brand)
	store := h.placementsIn(r.Context(), branch)[id]

	// ⚠️ **The opening balance is the closing one of everything before it**, not
	// a stored figure. There is no running balance in this system, so "what was
	// here on the first" is the same arithmetic run to that date — which is what
	// keeps this report and the balance screen from ever disagreeing.
	openingAll, _, err := h.expectedStockByWarehouse(r, scope, brand, branch, *from)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	closingAll, _, err := h.expectedStockByWarehouse(r, scope, brand, branch, *to)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	in, _ := h.deliveredInPeriod(r, scope, from, to)
	used := h.consumedInPeriod(r, scope, from, to, ingredients)
	written, _ := h.writtenOffInPeriod(r, scope, from, to)
	movedIn, movedOut := h.transferredInPeriod(r, scope, from, to)

	// The documents behind those totals, so a difference has somewhere to be
	// looked at rather than only being reported.
	docs := []movementDoc{}
	purchaseFilter := bson.M{"at": bson.M{"$gte": *from, "$lt": *to}, "lines.ingredientId": id}
	for k, v := range scope {
		purchaseFilter[k] = v
	}
	if cur, err := h.Store.Purchases.Find(r.Context(), purchaseFilter); err == nil {
		var rows []models.Purchase
		_ = cur.All(r.Context(), &rows)
		for _, p := range rows {
			for _, l := range p.Lines {
				if l.IngredientID == id {
					docs = append(docs, movementDoc{
						At: p.At, Kind: "purchase", Qty: l.Qty, Note: p.Supplier,
					})
				}
			}
		}
	}
	writeFilter := bson.M{"at": bson.M{"$gte": *from, "$lt": *to}, "ingredientId": id}
	for k, v := range scope {
		writeFilter[k] = v
	}
	if cur, err := h.Store.WriteOffs.Find(r.Context(), writeFilter); err == nil {
		var rows []models.WriteOff
		_ = cur.All(r.Context(), &rows)
		for _, x := range rows {
			docs = append(docs, movementDoc{
				At: x.At, Kind: "writeoff", Qty: -x.Qty, Note: x.Reason,
			})
		}
	}
	// ⚠️ Both directions listed, and named as such. A shelf that received
	// twelve kilos and sent eleven away is not the same shelf as one that
	// received one, and the netted figure cannot tell them apart.
	transferFilter := bson.M{"at": bson.M{"$gte": *from, "$lt": *to},
		"$or": []bson.M{{"fromId": id}, {"toId": id}}}
	for k, v := range scope {
		transferFilter[k] = v
	}
	if cur, err := h.Store.Transfers.Find(r.Context(), transferFilter); err == nil {
		var rows []models.StockTransfer
		_ = cur.All(r.Context(), &rows)
		for _, x := range rows {
			qty := x.Qty
			if x.FromID == id {
				qty = -qty
			}
			docs = append(docs, movementDoc{
				At: x.At, Kind: "transfer", Qty: qty, Note: x.Note,
			})
		}
	}
	sort.SliceStable(docs, func(i, j int) bool { return docs[i].At.Before(docs[j].At) })

	httpx.JSON(w, http.StatusOK, map[string]any{
		"ingredient": map[string]any{
			"id": ing.ID.Hex(), "name": ing.Name, "unit": ing.Unit,
			"warehouseId": store.Hex(),
		},
		"from":    *from,
		"to":      *to,
		"opening": round3(openingAll[store][id]),
		"in":      round3(in[id]),
		// ⚠️ Reported as a positive number with its own name rather than a
		// negative "in": "sold" and "thrown away" are different questions about
		// the same missing kilo, and a single column would make them one.
		"used":     round3(used[id]),
		"written":  round3(written[id]),
		"movedIn":  round3(movedIn[id]),
		"movedOut": round3(movedOut[id]),
		"closing":  round3(closingAll[store][id]),
		"docs":     docs,
	})
}
