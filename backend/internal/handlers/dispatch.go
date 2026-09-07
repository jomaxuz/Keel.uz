package handlers

// ---- The central store's van ----
//
// ⚠️ **The movement a chain has all day and this system could not record.**
// Everything else moves stock inside one branch — a delivery arrives at it, a
// sale leaves it, a batch is made in it, a transfer carries a case from its
// cellar to its bar. A central kitchen that buys the meat, marinates it on
// Monday and sends it to four branches had no document, and both ways of faking
// it lie: a write-off puts food nobody wasted on the waste report, and a
// delivery at the far end writes a purchase price into the price history and
// costs every dish the ingredient goes into.
//
// ⚠️ **Two shelves and a van between them.** The sender loses what was loaded,
// the receiver gains what was counted off, and the difference belongs to
// neither — which is the whole reason the paper this replaces carries three
// signatures. See models/dispatch.go.
//
// ⚠️ **The sending branch is the one in view, never one named in the body.**
// The store screens already resolve exactly one branch (`stockBranch`) and a
// manager is clamped to theirs; taking a sender from the request would let one
// branch empty another's shelves by pasting an id — the hole scope.go describes
// at length.

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// How far back the dispatch list looks when nobody says. A fortnight: long
// enough to find the slip somebody is arguing about, short enough that the
// screen opens on this week's vans.
const dispatchDays = 14

var (
	errDispatchSame  = errors.New("jo'natuvchi va qabul qiluvchi filial bir xil")
	errDispatchEmpty = errors.New("hech bo'lmasa bitta mahsulot kerak")
	errDispatchTo    = errors.New("qabul qiluvchi filialni tanlang")
	errDispatchPrep  = errors.New("yarim tayyor mahsulot javonda o'zi bo'lib turmaydi: uni tashkil qilgan masalliqlarni jo'nating")
)

// AdminCreateDispatch loads a van: what leaves this store, and for whom.
func (h *Handler) AdminCreateDispatch(w http.ResponseWriter, r *http.Request) {
	scope, branch, brand, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = scope
	var req struct {
		ToBranchID string    `json:"toBranchId"`
		At         time.Time `json:"at"`
		Driver     string    `json:"driver"`
		Note       string    `json:"note"`
		Lines      []struct {
			IngredientID string  `json:"ingredientId"`
			Qty          float64 `json:"qty"`
		} `json:"lines"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	to, err := objectID(req.ToBranchID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, errDispatchTo.Error())
		return
	}
	if to == branch {
		httpx.Error(w, http.StatusBadRequest, errDispatchSame.Error())
		return
	}
	// ⚠️ **The receiving branch has to be one of ours, and of the same brand.**
	// An ingredient id means "the beef in this brand's catalogue"; sending it to
	// another brand's branch would put a line on a shelf whose counts, recipes
	// and prices come from a different list, and every screen at the far end
	// would show a name it has no row for.
	var dest models.Branch
	filter := bson.M{"_id": to}
	if !brand.IsZero() {
		filter["brandId"] = brand
	}
	if err := h.Store.Branches.FindOne(r.Context(), filter).Decode(&dest); err != nil {
		httpx.Error(w, http.StatusNotFound, "filial topilmadi")
		return
	}

	ings := map[primitive.ObjectID]models.Ingredient{}
	for _, in := range h.scopedIngredients(r.Context(), brand) {
		ings[in.ID] = in
	}
	rates := h.ingredientRates(r.Context())
	at := req.At
	if at.IsZero() || at.After(time.Now()) {
		at = time.Now()
	}

	lines := make([]models.DispatchLine, 0, len(req.Lines))
	value := 0.0
	for _, l := range req.Lines {
		id, err := objectID(l.IngredientID)
		if err != nil || l.Qty <= 0 {
			continue
		}
		in, ok := ings[id]
		if !ok {
			continue
		}
		// ⚠️ **A sauce made as the kitchen goes is not on a shelf as itself**,
		// so it cannot be loaded onto a van — what it was made from is what
		// moves. A batched prep item *is* on a shelf (it is a tub in a fridge),
		// and that distinction is exactly why `DerivedOnly` exists.
		if in.DerivedOnly() {
			httpx.Error(w, http.StatusBadRequest, errDispatchPrep.Error())
			return
		}
		qty := round3(l.Qty)
		lines = append(lines, models.DispatchLine{
			IngredientID: id, Name: in.Name, Unit: in.Unit, Qty: qty,
		})
		// ⚠️ **Value is carried, not created** — the same rule a transfer
		// follows. Nothing was bought and nothing was lost, so this figure
		// never reaches the financial report's expenses; it is here so a
		// storekeeper can see what is on the van in money as well as in kilos.
		value += qty * float64(models.PerUnit(in.Unit)) * rates[id]
	}
	if len(lines) == 0 {
		httpx.Error(w, http.StatusBadRequest, errDispatchEmpty.Error())
		return
	}

	doc := models.Dispatch{
		BrandID:      brand,
		FromBranchID: branch,
		ToBranchID:   to,
		Number:       h.dispatchNumber(r.Context(), branch, at),
		At:           at,
		Lines:        lines,
		Value:        int(math.Round(value)),
		Driver:       clampText(req.Driver, 120),
		Note:         clampText(req.Note, 400),
		By:           h.adminName(r),
		CreatedAt:    time.Now(),
	}
	res, err := h.Store.Dispatches.InsertOne(r.Context(), doc)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	doc.ID = oidOf(res.InsertedID)
	h.logAction(r, "dispatch.create", "dispatch", doc.ID.Hex(), "", doc.Number)
	httpx.JSON(w, http.StatusCreated, doc)
}

// dispatchNumber is what the slip calls itself: the day, and that day's count.
//
// ⚠️ **Per sending branch and per day, not a running total.** The number exists
// so somebody holding two slips can tell them apart, and "07.09/3" says
// everything they need at the moment they are holding them — where a global
// sequence would say 1174 and mean nothing to anybody.
//
// ⚠️ Two vans loaded in the same second could take the same number, and that is
// deliberate: this is a label on paper, not an identity. The document's id is
// its identity, and making a person wait for a lock to print a slip is a worse
// trade than two "07.09/3" in a folder.
func (h *Handler) dispatchNumber(
	ctx context.Context, branch primitive.ObjectID, at time.Time,
) string {
	day := local(at)
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.Local)
	n, err := h.Store.Dispatches.CountDocuments(ctx, bson.M{
		"fromBranchId": branch,
		"at":           bson.M{"$gte": start, "$lt": start.AddDate(0, 0, 1)},
	})
	if err != nil {
		n = 0
	}
	return fmt.Sprintf("%s/%d", day.Format("02.01"), n+1)
}

// AdminDispatches lists the vans this branch sent and the ones it is owed.
func (h *Handler) AdminDispatches(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	from, to, err := parseRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if from == nil {
		f := time.Now().AddDate(0, 0, -dispatchDays)
		from = &f
	}
	rng := timeRange(from, to)

	// ⚠️ **Both ends of the van in one list.** A branch is a sender in the
	// morning and a receiver in the afternoon, and two screens for the two
	// halves would mean the person chasing a missing crate has to know which
	// half they are in before they can look for it.
	filter := bson.M{"$or": dispatchEnds(scope)}
	if len(rng) > 0 {
		filter["at"] = rng
	}
	cur, err := h.Store.Dispatches.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var rows []models.Dispatch
	if err := cur.All(r.Context(), &rows); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rows == nil {
		rows = []models.Dispatch{}
	}
	names := h.branchNames(r)
	out := make([]map[string]any, 0, len(rows))
	for _, d := range rows {
		out = append(out, map[string]any{
			"dispatch": d,
			// Named here so the slip and the list can print a branch without a
			// second call, and so a branch renamed tomorrow reads correctly on
			// the screen while the signed paper keeps its own words.
			"from": names[d.FromBranchID],
			"to":   names[d.ToBranchID],
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"rows": out})
}

// AdminAcceptDispatch records what the receiving branch counted off the van.
func (h *Handler) AdminAcceptDispatch(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var req struct {
		Lines []struct {
			IngredientID string  `json:"ingredientId"`
			Got          float64 `json:"got"`
		} `json:"lines"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	got := map[primitive.ObjectID]float64{}
	for _, l := range req.Lines {
		if lid, err := objectID(l.IngredientID); err == nil && l.Got >= 0 {
			got[lid] = round3(l.Got)
		}
	}

	// ⚠️ **Only the branch it was sent to may sign for it**, and the branch
	// lives inside the filter rather than in a check afterwards: an id alone
	// must never select a document (scope.go). A branch signing for somebody
	// else's van would move goods onto its own shelves and off nobody's.
	filter := bson.M{"_id": id, "acceptedAt": bson.M{"$exists": false}}
	if cond, ok := branchCond(scope); ok {
		filter["toBranchId"] = cond
	}
	var d models.Dispatch
	if err := h.Store.Dispatches.FindOne(r.Context(), filter).Decode(&d); err != nil {
		// Either it is not ours to sign for, or somebody has signed already.
		httpx.Error(w, http.StatusConflict, "bu jo'natma allaqachon qabul qilingan")
		return
	}
	now := time.Now()
	lines := make([]models.DispatchLine, 0, len(d.Lines))
	for _, l := range d.Lines {
		if v, ok := got[l.IngredientID]; ok && v != l.Qty {
			q := v
			l.Got = &q
		}
		lines = append(lines, l)
	}
	_, err = h.Store.Dispatches.UpdateOne(r.Context(), bson.M{"_id": d.ID},
		bson.M{"$set": bson.M{
			"lines": lines, "acceptedAt": now, "acceptedBy": h.adminName(r),
		}})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, "dispatch.accept", "dispatch", d.ID.Hex(), "", d.Number)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- What the van does to two balances ----

// branchCond lifts the branch condition out of a scope filter.
//
// ⚠️ **A dispatch names two branches, so the scope's own key cannot be used.**
// Every other collection stores the branch as `branchId` and the filter is
// copied across as it stands; here the same condition has to be asked of
// `fromBranchId` and of `toBranchId` in turn. Lifted rather than rebuilt, so a
// manager's clamp travels with it exactly as it was computed.
func branchCond(scope bson.M) (any, bool) {
	v, ok := scope["branchId"]
	return v, ok
}

// dispatchEnds is "this branch was one end of it or the other".
func dispatchEnds(scope bson.M) []bson.M {
	cond, ok := branchCond(scope)
	if !ok {
		return []bson.M{{}}
	}
	return []bson.M{{"fromBranchId": cond}, {"toBranchId": cond}}
}

// dispatchedInPeriod is what left this branch on vans and what arrived on them.
//
// ⚠️ **Loaded on the way out, counted on the way in.** The sender's shelf loses
// what the storekeeper put on the van; the receiver's gains what the branch
// counted off it. Where those differ the difference is on neither shelf — it is
// a finding about the journey, and quietly balancing the two would erase the
// one number the three signatures on the paper exist to produce.
func (h *Handler) dispatchedInPeriod(
	r *http.Request, scope bson.M, from, to *time.Time,
) (out, in map[primitive.ObjectID]float64) {
	out, in = map[primitive.ObjectID]float64{}, map[primitive.ObjectID]float64{}
	cond, ok := branchCond(scope)
	if !ok {
		return out, in
	}
	filter := bson.M{"$or": []bson.M{
		{"fromBranchId": cond}, {"toBranchId": cond},
	}}
	if rng := timeRange(from, to); len(rng) > 0 {
		filter["at"] = rng
	}
	cur, err := h.Store.Dispatches.Find(r.Context(), filter)
	if err != nil {
		return out, in
	}
	var rows []models.Dispatch
	if err := cur.All(r.Context(), &rows); err != nil {
		return out, in
	}
	// Which branches the caller is actually looking at, so a van between two of
	// them counts on both shelves and one to a branch out of view counts only
	// on ours.
	mine := matcher(cond)
	for _, d := range rows {
		for _, l := range d.Lines {
			if mine(d.FromBranchID) {
				out[l.IngredientID] += l.Qty
			}
			if mine(d.ToBranchID) {
				// ⚠️ **Nothing arrives before somebody signs for it.** Until
				// then the goods are on the van: counting them onto the
				// receiving shelf early would show a branch stock it cannot
				// cook with, and the stop list would let it be sold.
				if d.Accepted() {
					in[l.IngredientID] += l.Arrived()
				}
			}
		}
	}
	return out, in
}

// matcher turns a scope's branch condition into a question about one id.
func matcher(cond any) func(primitive.ObjectID) bool {
	switch v := cond.(type) {
	case primitive.ObjectID:
		return func(id primitive.ObjectID) bool { return id == v }
	case bson.M:
		list, _ := v["$in"].([]primitive.ObjectID)
		set := map[primitive.ObjectID]bool{}
		for _, id := range list {
			set[id] = true
		}
		return func(id primitive.ObjectID) bool { return set[id] }
	}
	return func(primitive.ObjectID) bool { return false }
}

// AdminDispatchStock is what this branch can actually put on a van.
//
// ⚠️ **The shelf, not the catalogue.** A list of two hundred ingredients with
// nothing on most of them is a list somebody scrolls past; what a storekeeper
// loading a van needs is the twenty rows their store is holding, with the
// quantity beside each so they can see what is left after they promise thirty
// kilos to Sergili.
func (h *Handler) AdminDispatchStock(w http.ResponseWriter, r *http.Request) {
	scope, branch, brand, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	byWarehouse, _, err := h.expectedStockByWarehouse(r, scope, brand, branch, time.Now())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	placed := h.placementsIn(r.Context(), branch)
	type row struct {
		ID   string  `json:"id"`
		Name string  `json:"name"`
		Unit string  `json:"unit"`
		Qty  float64 `json:"qty"`
	}
	rows := []row{}
	for _, in := range h.scopedIngredients(r.Context(), brand) {
		if in.DerivedOnly() {
			continue
		}
		rows = append(rows, row{
			ID: in.ID.Hex(), Name: in.Name, Unit: in.Unit,
			Qty: round3(byWarehouse[placed[in.ID]][in.ID]),
		})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	httpx.JSON(w, http.StatusOK, map[string]any{"rows": rows})
}
