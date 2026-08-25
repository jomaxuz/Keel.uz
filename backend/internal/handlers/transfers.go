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

// ---- Moving stock between stores ----
//
// See models/transfer.go for why this is a movement of its own rather than a
// write-off paired with a delivery.

// AdminListTransfers returns recent moves, newest first.
func (h *Handler) AdminListTransfers(w http.ResponseWriter, r *http.Request) {
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
	cur, err := h.Store.Transfers.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.StockTransfer{}
	_ = cur.All(r.Context(), &rows)
	moved := 0
	for _, x := range rows {
		moved += x.Value
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"transfers": rows, "moved": moved})
}

// AdminCreateTransfer records one move between two stores.
func (h *Handler) AdminCreateTransfer(w http.ResponseWriter, r *http.Request) {
	var in models.StockTransfer
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	if in.At.IsZero() || in.At.After(now) {
		// Never into the future, like a delivery and a write-off: a movement
		// dated forward sits outside every report until the day arrives and
		// then changes a month somebody has already read.
		in.At = now
	}
	if in.Qty <= 0 {
		httpx.Error(w, http.StatusBadRequest, "miqdor noldan katta bo'lsin")
		return
	}
	// ⚠️ A move happens inside one building, so the branch has to be settled
	// before either end can be checked: "different stores" is only meaningful
	// against one branch's placements.
	_, branch, brand, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sc := Scope{BrandID: brand, BranchID: branch}
	src, dst, err := h.transferEnds(r, sc, branch, in.FromID, in.ToID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// ⚠️ Valued at the **source's** price on that day, and frozen. The two
	// shelves may carry different prices for the same thing (the bar's bottle
	// was bought in a hurry), and what moved is what left the shelf it left.
	rates := h.ingredientRates(r.Context())
	in.Value = writeOffValue(src, in.Qty, in.At, rates)
	in.By = h.adminName(r)
	in.CreatedAt = now
	if in.BranchID.IsZero() {
		in.BranchID = branch
	}

	res, err := h.Store.Transfers.InsertOne(r.Context(), in)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	in.ID = oidOf(res.InsertedID)
	h.logAction(r, "transfer.create", "transfer", in.ID.Hex(),
		src.Name+" → "+dst.Name, "")
	httpx.JSON(w, http.StatusCreated, in)
}

// transferEnds loads the two shelves and refuses the moves that cannot mean
// anything.
//
// ⚠️ **The units have to match.** Both sides are counted in purchase units, and
// a quantity moved out of a kilo into a litre is a number with no meaning that
// both balances would nevertheless accept. Nothing downstream could detect it
// afterwards: each store's figure stays arithmetically consistent and both are
// wrong.
//
// ⚠️ **A prep item cannot be either end.** It is not on a shelf as itself — it
// is a pot made this morning, and its ingredients are what is counted. Moving
// one would subtract from a balance that is derived rather than held.
//
// ⚠️ **The two ends must be different.** A move to the same shelf nets to zero
// today, but it is always a mis-click, and stored it becomes a row somebody
// later tries to explain.
func (h *Handler) transferEnds(
	r *http.Request, sc Scope, branch, fromID, toID primitive.ObjectID,
) (models.Ingredient, models.Ingredient, error) {
	var src, dst models.Ingredient
	if fromID.IsZero() || toID.IsZero() {
		return src, dst, errTransferEnds
	}
	if fromID == toID {
		return src, dst, errTransferSameShelf
	}
	// The brand inside the filter: an id alone never selects a document.
	if err := h.Store.Ingredients.FindOne(r.Context(),
		sc.brandFilter(bson.M{"_id": fromID})).Decode(&src); err != nil {
		return src, dst, errTransferEnds
	}
	if err := h.Store.Ingredients.FindOne(r.Context(),
		sc.brandFilter(bson.M{"_id": toID})).Decode(&dst); err != nil {
		return src, dst, errTransferEnds
	}
	placed := h.placementsIn(r.Context(), branch)
	return src, dst, transferRefusal(src, dst, placed[fromID], placed[toID])
}

// transferRefusal is every reason a move cannot mean anything, in one pure
// function so the rules can be sealed by a test rather than only by reading.
//
// The same shape `purgeRefusal` takes, and for the same reason: these are the
// checks that stop being run the moment somebody adds a second way in.
func transferRefusal(src, dst models.Ingredient, from, to primitive.ObjectID) error {
	if src.DerivedOnly() || dst.DerivedOnly() {
		return errTransferPrep
	}
	if src.Unit != dst.Unit {
		return errTransferUnits
	}
	// ⚠️ Compared on **this branch's** placements: the same two ingredients can
	// sit in one room here and two rooms there, and the ingredient itself no
	// longer carries an answer.
	if from == to {
		return errTransferSameStore
	}
	return nil
}

// AdminDeleteTransfer removes a move entered by mistake.
//
// ⚠️ Unlike a delivery, deleting this really does undo it: a transfer writes no
// price and nothing is layered on top of it. Both balances simply go back to
// what they were.
func (h *Handler) AdminDeleteTransfer(w http.ResponseWriter, r *http.Request) {
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
	res, err := h.Store.Transfers.DeleteOne(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.DeletedCount == 0 {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	h.logAction(r, "transfer.delete", "transfer", id.Hex(), "", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// transferredInPeriod is what arrived at and left each shelf by being moved.
//
// ⚠️ **Two maps, never one net figure.** "Twelve kilos moved out and eleven
// moved in" and "one kilo moved out" produce the same balance and answer
// different questions, and the movement report exists to tell them apart.
func (h *Handler) transferredInPeriod(
	r *http.Request, scope bson.M, from, to *time.Time,
) (in, out map[primitive.ObjectID]float64) {
	in, out = map[primitive.ObjectID]float64{}, map[primitive.ObjectID]float64{}
	filter := bson.M{}
	for k, v := range scope {
		filter[k] = v
	}
	if rng := timeRange(from, to); len(rng) > 0 {
		filter["at"] = rng
	}
	cur, err := h.Store.Transfers.Find(r.Context(), filter)
	if err != nil {
		return in, out
	}
	var rows []models.StockTransfer
	if err := cur.All(r.Context(), &rows); err != nil {
		return in, out
	}
	for _, x := range rows {
		out[x.FromID] += x.Qty
		in[x.ToID] += x.Qty
	}
	return in, out
}

// ---- Why a move can be refused ----
//
// Named rather than inline, because each one sends the person at the screen
// somewhere different: to pick a shelf, to fix a unit, or to stop trying to
// move a sauce.
var (
	errTransferEnds      = errors.New("qaysi javondan qaysi javonga ko'chirilishini tanlang")
	errTransferSameShelf = errors.New("bir masalliqni o'ziga ko'chirib bo'lmaydi")
	errTransferSameStore = errors.New("ikkalasi ham bitta omborda — ko'chirishga hojat yo'q")
	errTransferPrep      = errors.New("yarim tayyor mahsulot javonda o'zi bo'lib turmaydi: uni tashkil qilgan masalliqlarni ko'chiring")
	errTransferUnits     = errors.New("o'lchov birliklari har xil — kilogrammni litrga ko'chirib bo'lmaydi")
)

// timeRange is the `at` filter a period produces, or nothing for an open one.
//
// ⚠️ Half-open at the top (`$lt`): `parseRange` already moves `to` to the start
// of the next day, so `$lte` would take in the first instant of a day the
// report does not cover.
func timeRange(from, to *time.Time) bson.M {
	rng := bson.M{}
	if from != nil {
		rng["$gte"] = *from
	}
	if to != nil {
		rng["$lt"] = *to
	}
	return rng
}
