package handlers

import (
	"errors"
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
	lines, err := cleanPurchaseLines(in.Lines)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
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
	sc, _ := h.adminScope(r)
	// ⚠️ The name is frozen off the supplier, not taken from the form: a
	// renamed supplier must not rewrite last year's invoices, and a form that
	// posts both would let the two disagree from the first save.
	in.SupplierID, in.Supplier = h.supplierNameFor(r, sc, in.SupplierID, in.Supplier)
	in.Note = clampText(in.Note, 200)
	in.CreatedAt = now
	in.CreatedBy = h.adminName(r)
	if in.BranchID.IsZero() {
		in.BranchID = h.scopeBranch(r, sc)
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
		// ⚠️ **The labels this delivery makes worth printing — offered, never
		// printed.** A shop's goods arrive and go on a shelf, and until this
		// existed somebody had to hunt each one out of the catalogue by name to
		// get a sticker. Two hundred packets answering themselves with two
		// hundred stickers would be worse: the products and the counts are here,
		// so the panel puts one button in front of the person who just typed the
		// invoice. Empty for every restaurant, which is every install today.
		"labelsDue": h.labelsDueFor(r.Context(), in),
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
	// ⚠️ The brand inside the filter, like every other by-id read here: a
	// delivery line naming another brand's ingredient would rewrite its buying
	// price, which is the one edit nothing on that brand's screens explains.
	sc, _ := h.adminScope(r)
	changed := 0
	for _, l := range p.Lines {
		var ing models.Ingredient
		if err := h.Store.Ingredients.FindOne(r.Context(),
			sc.brandFilter(bson.M{"_id": l.IngredientID})).Decode(&ing); err != nil {
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
		hist := append(ing.History, models.PriceEntry{
			Price: l.Price, At: p.At, PurchaseID: p.ID,
		})
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

// AdminUpdatePurchase corrects a delivery that was entered wrongly.
//
// ⚠️ **The gap this closes is small and constant.** An invoice is forty numbers
// typed by somebody standing at a door, and the twenty-first is a transposition
// — 42 000 for 24 000, twelve kilos for twenty-one. Until now the only remedy
// was delete and retype, which loses the entry date, the person who took it in,
// and (because a delete deliberately leaves prices alone) leaves the wrong price
// standing in the history with a correct one beside it.
//
// ⚠️ **The prices this invoice claimed are withdrawn and re-applied**, which is
// only possible because each entry now records the delivery that wrote it. This
// is deliberately *not* what a delete does, and the difference is what the two
// actions mean: an edit says "the invoice should have said this", which is a
// claim about the price, so the old claim goes. A delete says only "this row
// should not be here" — it cannot distinguish a mis-entry from a delivery that
// was cancelled after the food was already costed, so it leaves the price to be
// corrected the way every other price is.
//
// ⚠️ Entries with no delivery behind them — hand edits, and everything from
// before the field existed — are never touched. Nobody can say which invoice
// they belonged to, and guessing would silently delete a deliberate correction.
func (h *Handler) AdminUpdatePurchase(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in models.Purchase
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
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
	var stored models.Purchase
	if err := h.Store.Purchases.FindOne(r.Context(), filter).Decode(&stored); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}

	lines, err := cleanPurchaseLines(in.Lines)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	at := in.At
	if at.IsZero() || at.After(now) {
		at = stored.At
	}
	total := in.Total
	if total <= 0 {
		for _, l := range lines {
			total += l.Sum()
		}
	}
	sc, _ := h.adminScope(r)
	supplierID, supplier := h.supplierNameFor(r, sc, in.SupplierID, in.Supplier)

	// ⚠️ Withdrawn **before** the new lines are written, and against the stored
	// version: an ingredient dropped from the invoice has to lose its price
	// claim too, and reading the incoming lines would leave that one behind.
	h.withdrawDeliveryPrices(r, stored)

	set := bson.M{
		"at":         at,
		"lines":      lines,
		"total":      total,
		"supplierId": supplierID,
		"supplier":   supplier,
		"note":       clampText(in.Note, 200),
	}
	if _, err := h.Store.Purchases.UpdateOne(r.Context(), filter,
		bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	stored.At, stored.Lines, stored.Total = at, lines, total
	stored.SupplierID, stored.Supplier = supplierID, supplier
	changed := h.applyDeliveryPrices(r, stored)
	h.logAction(r, "purchase.update", "purchase", id.Hex(), supplier, "")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"purchase": stored, "pricesChanged": changed,
	})
}

// withdrawDeliveryPrices removes the history entries one delivery wrote.
//
// ⚠️ **Matched on the delivery, not on the date or the amount.** Two invoices
// can land on one day, and an entry removed by date would take the other one's
// claim with it — which would show up weeks later as a dish that quietly
// changed price in a month nobody edited.
func (h *Handler) withdrawDeliveryPrices(r *http.Request, p models.Purchase) {
	if p.ID.IsZero() {
		return
	}
	for _, l := range p.Lines {
		var ing models.Ingredient
		if err := h.Store.Ingredients.FindOne(r.Context(),
			bson.M{"_id": l.IngredientID}).Decode(&ing); err != nil {
			continue
		}
		kept := make([]models.PriceEntry, 0, len(ing.History))
		for _, e := range ing.History {
			if e.PurchaseID == p.ID {
				continue
			}
			kept = append(kept, e)
		}
		if len(kept) == len(ing.History) {
			continue
		}
		set := bson.M{"history": kept, "updatedAt": time.Now()}
		// ⚠️ The headline price follows the latest entry that survives — or
		// stays where it is when nothing does. Zeroing it would make every dish
		// containing the ingredient cost nothing, which reads on a margin
		// report as very good news.
		if len(kept) > 0 {
			set["price"] = kept[len(kept)-1].Price
		}
		_, _ = h.Store.Ingredients.UpdateByID(r.Context(), ing.ID, bson.M{"$set": set})
	}
}

// cleanPurchaseLines drops the rows a form leaves behind and refuses an empty
// invoice.
func cleanPurchaseLines(in []models.PurchaseLine) ([]models.PurchaseLine, error) {
	var out []models.PurchaseLine
	for _, l := range in {
		if l.IngredientID.IsZero() || l.Qty <= 0 || l.Price < 0 {
			continue
		}
		out = append(out, l)
	}
	if len(out) == 0 {
		return nil, errNoPurchaseLines
	}
	return out, nil
}

var errNoPurchaseLines = errors.New("hech bo'lmasa bitta qator kerak")

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
