package handlers

import (
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- Deliveries, and the prices that come out of them ----
//
// ⚠️ **This is where a price stops being retyped.** An ingredient's price was a
// number somebody read off an invoice and entered by hand, which is exactly the
// step that does not happen after the fortieth delivery. Recording the delivery
// is work the restaurant already does; the price falls out of it.
//
// ⚠️ **Dated by the invoice, not by when it was entered.** That is what makes
// this different from editing a price by hand: an edit cannot tell "we typed it
// wrong" from "beef went up" and therefore counts from today, while a delivery
// is a measurement carrying its own date. Saying meat cost this much last
// Tuesday is a fact about last Tuesday, not a rewriting of it.

// AdminListPurchases returns recent deliveries, newest first.
func (h *Handler) AdminListPurchases(w http.ResponseWriter, r *http.Request) {
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
	cur, err := h.Store.Purchases.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var rows []models.Purchase
	_ = cur.All(r.Context(), &rows)
	if rows == nil {
		rows = []models.Purchase{}
	}
	spent := 0
	for _, p := range rows {
		spent += p.Total
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"purchases": rows, "spent": spent})
}

// AdminCreatePurchase records one delivery and updates what things cost.
func (h *Handler) AdminCreatePurchase(w http.ResponseWriter, r *http.Request) {
	var in models.Purchase
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	if in.At.IsZero() {
		in.At = now
	}
	// ⚠️ A delivery cannot be dated into the future. Prices from a date that
	// has not happened would apply to nothing today and to everything from
	// then on — a mistake with a delayed effect nobody would connect back to
	// this form.
	if in.At.After(now) {
		in.At = now
	}
	var lines []models.PurchaseLine
	for _, l := range in.Lines {
		if l.IngredientID.IsZero() || l.Qty <= 0 || l.Price < 0 {
			continue
		}
		lines = append(lines, l)
	}
	if len(lines) == 0 {
		httpx.Error(w, http.StatusBadRequest, "hech bo'lmasa bitta qator kerak")
		return
	}
	in.Lines = lines
	// ⚠️ The invoice's own total wins when it was given: a delivery charge or a
	// discount at the door is money the restaurant paid and no line explains
	// it. Only an empty total is computed, so the field is never a silent
	// second opinion about what the lines say.
	if in.Total <= 0 {
		for _, l := range lines {
			in.Total += l.Sum()
		}
	}
	in.Supplier = clampText(in.Supplier, 120)
	in.Note = clampText(in.Note, 200)
	in.CreatedAt = now
	in.CreatedBy = h.adminName(r)
	if scope, err := h.adminScope(r); err == nil && in.BranchID.IsZero() {
		in.BranchID = h.scopeBranch(r, scope)
	}

	res, err := h.Store.Purchases.InsertOne(r.Context(), in)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	in.ID = oidOf(res.InsertedID)

	changed := h.applyDeliveryPrices(r, in)
	h.logAction(r, "purchase.create", "purchase", in.ID.Hex(), in.Supplier, "")
	httpx.JSON(w, http.StatusCreated, map[string]any{
		"purchase": in,
		// How many ingredients now cost something different. Reported because
		// it is the part of this that changes other screens, and somebody
		// entering an invoice should see that it did.
		"pricesChanged": changed,
	})
}

// applyDeliveryPrices writes each line's price into its ingredient's history.
//
// ⚠️ **Inserted at the delivery's date and kept in order**, rather than
// appended: an invoice entered three days late describes those three days, and
// a history sorted by when somebody typed it would cost them at the wrong
// price — which is the exact failure the history exists to prevent, arriving
// through the door it was built for.
//
// ⚠️ Nothing is written when the price has not moved: an entry per delivery
// would bury the two or three that matter under a hundred that say the same
// number, and "when did this go up" is the question the list has to answer.
func (h *Handler) applyDeliveryPrices(r *http.Request, p models.Purchase) int {
	changed := 0
	for _, l := range p.Lines {
		var ing models.Ingredient
		if err := h.Store.Ingredients.FindOne(r.Context(),
			bson.M{"_id": l.IngredientID}).Decode(&ing); err != nil {
			continue
		}
		// A prep item is cooked, not delivered: its rate comes from its own
		// card, and a delivery line naming one is a mistake this must not
		// quietly act on.
		if ing.MadeInHouse() || l.Price <= 0 {
			continue
		}
		if ing.PriceAt(p.At) == l.Price {
			continue
		}
		hist := append(ing.History, models.PriceEntry{Price: l.Price, At: p.At})
		sort.SliceStable(hist, func(i, j int) bool { return hist[i].At.Before(hist[j].At) })
		set := bson.M{"history": hist, "updatedAt": time.Now()}
		// ⚠️ The headline price follows only the **latest** entry: an invoice
		// from last month tells us what last month cost and says nothing about
		// today, and letting it overwrite today's price is how a back-dated
		// correction quietly makes every dish cheaper.
		if len(hist) > 0 && hist[len(hist)-1].At.Equal(p.At) {
			set["price"] = l.Price
		}
		if _, err := h.Store.Ingredients.UpdateByID(r.Context(), ing.ID,
			bson.M{"$set": set}); err == nil {
			changed++
		}
	}
	return changed
}

// AdminDeletePurchase removes a delivery that was entered by mistake.
//
// ⚠️ **The prices it wrote are left alone**, and the screen says so. Unwinding
// them is not "put the old number back": later deliveries, hand edits and the
// dishes costed in between all sit on top of it, and a delete that quietly
// re-costed a month would be far worse than a wrong invoice row. The price is
// corrected the way every other price is — by the next delivery, or by hand.
func (h *Handler) AdminDeletePurchase(w http.ResponseWriter, r *http.Request) {
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
	res, err := h.Store.Purchases.DeleteOne(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.DeletedCount == 0 {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	h.logAction(r, "purchase.delete", "purchase", id.Hex(), "", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
