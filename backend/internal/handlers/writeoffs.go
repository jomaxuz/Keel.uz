package handlers

import (
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- Food that left without being sold ----
//
// ⚠️ **Without this the flow report's difference column cannot become an
// answer.** Food leaves a restaurant three ways: sold, eaten by the staff, or
// thrown away. The tech cards account for the first; until the other two are
// recorded, the gap between what came in and what was sold is unreadable — it
// could be waste, theft, or a delivery still sitting in the fridge.
//
// ⚠️ **A reason is required**, as everywhere else in this system where
// something disappears: a void, a refund, a cancelled order. "Twelve kilos of
// beef, written off" with no sentence beside it is the line every argument
// starts from, and by then the person who could answer has gone home.

// AdminListWriteOffs returns recent write-offs, newest first.
func (h *Handler) AdminListWriteOffs(w http.ResponseWriter, r *http.Request) {
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
	rng := bson.M{}
	if from != nil {
		rng["$gte"] = *from
	}
	if to != nil {
		rng["$lt"] = *to
	}
	if len(rng) > 0 {
		filter["at"] = rng
	}
	cur, err := h.Store.WriteOffs.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var rows []models.WriteOff
	_ = cur.All(r.Context(), &rows)
	if rows == nil {
		rows = []models.WriteOff{}
	}
	value := 0
	for _, x := range rows {
		value += x.Value
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"writeOffs": rows,
		"value":     value,
		"reasons":   reasonTotals(rows),
	})
}

// AdminCreateWriteOff records food that left without being sold.
func (h *Handler) AdminCreateWriteOff(w http.ResponseWriter, r *http.Request) {
	var in models.WriteOff
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	in.Reason = clampText(in.Reason, 200)
	if in.Reason == "" {
		httpx.Error(w, http.StatusBadRequest, "sababini yozing")
		return
	}
	if in.IngredientID.IsZero() || in.Qty <= 0 {
		httpx.Error(w, http.StatusBadRequest, "masalliq va miqdorni tanlang")
		return
	}
	now := time.Now()
	if in.At.IsZero() || in.At.After(now) {
		// ⚠️ Never into the future, like a delivery: a write-off dated forward
		// would sit outside every report until the day arrives and then change
		// a month that had already been read.
		in.At = now
	}

	// ⚠️ The brand is inside the filter: an id alone never selects a document,
	// and a write-off is allowed to name any ingredient the poster can guess at.
	sc, _ := h.adminScope(r)
	var ing models.Ingredient
	if err := h.Store.Ingredients.FindOne(r.Context(),
		sc.brandFilter(bson.M{"_id": in.IngredientID})).Decode(&ing); err != nil {
		httpx.Error(w, http.StatusBadRequest, "masalliq topilmadi")
		return
	}
	// ⚠️ Valued at the prices of **that day**, and frozen. This is the figure
	// that changes behaviour — "3 200 000 thrown away this month" moves an
	// owner in a way "eleven write-offs" does not — and it must not shift under
	// them when the next delivery arrives at a different price.
	in.Value = writeOffValue(ing, in.Qty, in.At, h.ingredientRates(r.Context()))
	in.By = h.adminName(r)
	in.CreatedAt = now
	if in.BranchID.IsZero() {
		in.BranchID = h.scopeBranch(r, sc)
	}

	res, err := h.Store.WriteOffs.InsertOne(r.Context(), in)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	in.ID = oidOf(res.InsertedID)
	h.logAction(r, "writeoff.create", "writeoff", in.ID.Hex(), ing.Name, in.Reason)
	httpx.JSON(w, http.StatusCreated, in)
}

// writeOffValue is what the quantity was worth on that day.
//
// ⚠️ A prep item is valued from its card — it was never bought, so it has no
// price of its own, and refusing to value it would leave the most expensive
// thing a kitchen throws away (a ruined batch of sauce) recorded as costing
// nothing.
func writeOffValue(
	ing models.Ingredient, qty float64, at time.Time,
	rates map[primitive.ObjectID]float64,
) int {
	if ing.MadeInHouse() {
		// The rate is per recipe unit; the quantity is in purchase units.
		return int(math.Round(rates[ing.ID] * qty * float64(models.PerUnit(ing.Unit))))
	}
	return int(math.Round(float64(ing.PriceAt(at)) * qty))
}

// AdminDeleteWriteOff removes one entered by mistake.
func (h *Handler) AdminDeleteWriteOff(w http.ResponseWriter, r *http.Request) {
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
	res, err := h.Store.WriteOffs.DeleteOne(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.DeletedCount == 0 {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	h.logAction(r, "writeoff.delete", "writeoff", id.Hex(), "", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- What is being thrown away, and why ----
//
// ⚠️ **The reason is the only field here that answers a question worth acting
// on.** "3 200 000 written off this month" tells an owner something is wrong;
// "1 900 000 of it is staff meals" tells them what to do about it — and those
// are two entirely different conversations, one with the kitchen and one with
// the rota.
//
// ⚠️ **Grouped on a normalised reason, displayed as it was typed.** The field
// is free text on purpose (every kitchen throws away something the next one
// does not), which means "buzildi", "Buzildi" and "buzildi " arrive as three
// reasons that are one thing. Matching on the trimmed lower-case form gathers
// them; showing the spelling somebody actually used keeps the list in the
// restaurant's own words rather than in a normalised transcription of them.

type reasonTotal struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
	Value  int    `json:"value"`
}

func reasonTotals(rows []models.WriteOff) []reasonTotal {
	type acc struct {
		label string
		count int
		value int
	}
	byKey := map[string]*acc{}
	for _, x := range rows {
		key := strings.ToLower(strings.TrimSpace(x.Reason))
		if key == "" {
			continue
		}
		a, ok := byKey[key]
		if !ok {
			a = &acc{label: strings.TrimSpace(x.Reason)}
			byKey[key] = a
		}
		a.count++
		a.value += x.Value
	}
	out := make([]reasonTotal, 0, len(byKey))
	for _, a := range byKey {
		out = append(out, reasonTotal{Reason: a.label, Count: a.count, Value: a.value})
	}
	// ⚠️ Costliest first, not most frequent: twelve spilled coffees and one
	// ruined tray of meat are the same length of list and not the same problem.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Value != out[j].Value {
			return out[i].Value > out[j].Value
		}
		return out[i].Reason < out[j].Reason
	})
	return out
}
