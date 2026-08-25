package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- Making a batch, in a central kitchen ----
//
// See models/production.go for why this document exists at all, and why it and
// `Ingredient.Batched` are two halves of one thing.

var (
	errBatchItem = errors.New(
		"faqat partiya bilan tayyorlanadigan yarim tayyor mahsulot tanlanadi")
	errBatchStore = errors.New(
		"partiya faqat ishlab chiqarish omborida tayyorlanadi (tsex)")
	errBatchInputs = errors.New(
		"texkarta to'liq emas — masalliqlaridan biri yo'q")
)

// AdminListProductions returns recent batches, newest first.
func (h *Handler) AdminListProductions(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := bson.M{}
	for k, v := range scope {
		filter[k] = v
	}
	from, to, err := parseRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if rng := timeRange(from, to); len(rng) > 0 {
		filter["at"] = rng
	}
	cur, err := h.Store.Productions.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.Production{}
	_ = cur.All(r.Context(), &rows)
	made := 0
	for _, x := range rows {
		made += x.Value
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"productions": rows, "made": made})
}

// AdminCreateProduction records one batch.
func (h *Handler) AdminCreateProduction(w http.ResponseWriter, r *http.Request) {
	var in models.Production
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	if in.At.IsZero() || in.At.After(now) {
		// Never into the future — the same rule a delivery, a write-off and a
		// transfer follow, and for the same reason: a document dated forward
		// sits outside every report until the day arrives and then changes a
		// month somebody has already read and reconciled.
		in.At = now
	}
	if in.Qty <= 0 {
		httpx.Error(w, http.StatusBadRequest, "miqdor noldan katta bo'lsin")
		return
	}
	// ⚠️ One branch, settled first: the store the batch is cooked in and the
	// shelves its inputs come off are both facts about one building.
	_, branch, brand, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sc := Scope{BrandID: brand, BranchID: branch}

	var item models.Ingredient
	if err := h.Store.Ingredients.FindOne(r.Context(),
		sc.brandFilter(bson.M{"_id": in.IngredientID})).Decode(&item); err != nil {
		httpx.Error(w, http.StatusBadRequest, "mahsulot topilmadi")
		return
	}
	// ⚠️ **Only a batched prep item.** Producing something bought is a delivery;
	// producing a prep item that is *not* batched would subtract its inputs
	// here and again through the card when a dish is sold, and the shortfall
	// would surface weeks later at a count as somebody's fault.
	if !item.MadeInHouse() || !item.Batched {
		httpx.Error(w, http.StatusBadRequest, errBatchItem.Error())
		return
	}

	var store models.Warehouse
	if err := h.Store.Warehouses.FindOne(r.Context(),
		bson.M{"_id": in.WarehouseID, "branchId": branch}).Decode(&store); err != nil {
		httpx.Error(w, http.StatusBadRequest, "ombor topilmadi")
		return
	}
	if !store.IsProduction() {
		httpx.Error(w, http.StatusBadRequest, errBatchStore.Error())
		return
	}

	lines, value, err := h.batchInputs(r, item, in.Qty, in.At)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	in.Lines = lines
	in.Value = value
	in.By = h.adminName(r)
	in.CreatedAt = now
	if in.BranchID.IsZero() {
		in.BranchID = branch
	}

	res, err := h.Store.Productions.InsertOne(r.Context(), in)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	in.ID = oidOf(res.InsertedID)
	h.logAction(r, "production.create", "production", in.ID.Hex(), item.Name, "")
	httpx.JSON(w, http.StatusCreated, in)
}

// batchInputs works out what a batch of this size took, and what that was worth.
//
// ⚠️ **Computed once, here, and then frozen onto the document.** The card is
// corrected as recipes change, and a batch made in March took what it took in
// March; re-deriving it later would rewrite a month that has already been
// counted. Same rule as a write-off's value.
//
// ⚠️ **Purchase units on both sides.** The card speaks grams and millilitres;
// every balance in this system speaks kilos and litres, and mixing them is the
// kind of error that stays arithmetically consistent while being a thousand
// times wrong.
func (h *Handler) batchInputs(
	r *http.Request, item models.Ingredient, qty float64, at time.Time,
) ([]models.ProductionLine, int, error) {
	rates := h.ingredientRates(r.Context())
	// How many recipe-units of output this batch is, in the card's terms.
	//
	// ⚠️ `Output` is in **recipe** units (2 kg of sauce is 2000), and `qty` is
	// in purchase units — so the batch is scaled by both. Getting this wrong
	// produces a plausible number a thousand times off, which is exactly the
	// size of error nobody spots on a screen full of kilos.
	perUnit := float64(models.PerUnit(item.Unit))
	batches := qty * perUnit / item.Output

	lines := make([]models.ProductionLine, 0, len(item.Recipe))
	value := 0
	for _, l := range item.Recipe {
		var input models.Ingredient
		if err := h.Store.Ingredients.FindOne(r.Context(),
			bson.M{"_id": l.IngredientID}).Decode(&input); err != nil {
			// ⚠️ Refused rather than skipped. A missing input silently makes a
			// batch cheaper and takes nothing off the shelf — and a screen
			// showing margin reads that as good news.
			return nil, 0, errBatchInputs
		}
		took := l.Qty * batches / float64(models.PerUnit(input.Unit))
		lines = append(lines, models.ProductionLine{
			IngredientID: input.ID,
			Name:         input.Name,
			Qty:          round3(took),
		})
		value += writeOffValue(input, took, at, rates)
	}
	if len(lines) == 0 {
		return nil, 0, errBatchInputs
	}
	return lines, value, nil
}

// AdminDeleteProduction removes a batch entered by mistake.
//
// ⚠️ Like a transfer and unlike a delivery, deleting this really does undo it:
// a batch writes no price into any history and nothing is layered on top of it.
// Both sides — the inputs and the output — simply go back to what they were.
func (h *Handler) AdminDeleteProduction(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := bson.M{"_id": id}
	for k, v := range scope {
		filter[k] = v
	}
	res, err := h.Store.Productions.DeleteOne(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.DeletedCount == 0 {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	h.logAction(r, "production.delete", "production", id.Hex(), "", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// producedInPeriod is the sixth fact the balance needs: what a batch put on the
// shelf, and what it took off.
//
// ⚠️ **Both halves, from one document.** Counting only the output would make a
// central kitchen a machine that creates sauce out of nothing; counting only
// the inputs would make it a write-off with no waste. The pair is the whole
// point of the document.
func (h *Handler) producedInPeriod(
	r *http.Request, scope bson.M, from, to *time.Time,
) (made, took map[primitive.ObjectID]float64) {
	made = map[primitive.ObjectID]float64{}
	took = map[primitive.ObjectID]float64{}
	filter := bson.M{}
	for k, v := range scope {
		filter[k] = v
	}
	if rng := timeRange(from, to); len(rng) > 0 {
		filter["at"] = rng
	}
	cur, err := h.Store.Productions.Find(r.Context(), filter)
	if err != nil {
		return made, took
	}
	var rows []models.Production
	if err := cur.All(r.Context(), &rows); err != nil {
		return made, took
	}
	for _, p := range rows {
		made[p.IngredientID] += p.Qty
		for _, l := range p.Lines {
			took[l.IngredientID] += l.Qty
		}
	}
	return made, took
}
