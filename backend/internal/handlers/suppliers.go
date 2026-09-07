package handlers

import (
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

// Suppliers — see models/supplier.go for why the free-text field survives.

// AdminListSuppliers returns the brand's suppliers, in the owner's order.
func (h *Handler) AdminListSuppliers(w http.ResponseWriter, r *http.Request) {
	sc, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	cur, err := h.Store.Suppliers.Find(r.Context(), sc.brandFilter(bson.M{}),
		options.Find().SetSort(bson.D{{Key: "sort", Value: 1}, {Key: "name", Value: 1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.Supplier{}
	_ = cur.All(r.Context(), &rows)
	httpx.JSON(w, http.StatusOK, map[string]any{"suppliers": rows})
}

type supplierRequest struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	TIN      string `json:"tin"`
	Note     string `json:"note"`
	Sort     int    `json:"sort"`
	IsActive *bool  `json:"isActive"`
}

// AdminSaveSupplier creates or updates one.
func (h *Handler) AdminSaveSupplier(w http.ResponseWriter, r *http.Request) {
	sc, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req supplierRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	name := clampText(req.Name, 120)
	if name == "" {
		httpx.Error(w, http.StatusBadRequest, "yetkazib beruvchining nomi kerak")
		return
	}
	now := time.Now()
	set := bson.M{
		"name":      name,
		"phone":     clampText(req.Phone, 40),
		// ⚠️ Digits only, because that is what an electronic invoice carries:
		// "ИНН 302 936 161" typed with spaces would never match the number the
		// document names, and the failure would look like a missing supplier.
		"tin":       tinDigits(clampText(req.TIN, 20)),
		"note":      clampText(req.Note, 300),
		"sort":      req.Sort,
		"updatedAt": now,
	}
	if req.IsActive != nil {
		set["isActive"] = *req.IsActive
	}

	raw := chi.URLParam(r, "id")
	if raw == "" {
		set["isActive"] = true
		set["createdAt"] = now
		set["brandId"] = h.scopeBrand(r, sc)
		res, err := h.Store.Suppliers.InsertOne(r.Context(), set)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		id := oidOf(res.InsertedID)
		h.logAction(r, "supplier.create", "supplier", id.Hex(), name, "")
		h.oneSupplier(w, r, sc, id)
		return
	}
	id, err := objectID(raw)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	// The brand inside the filter: an id alone never selects a document.
	res, err := h.Store.Suppliers.UpdateOne(r.Context(),
		sc.brandFilter(bson.M{"_id": id}), bson.M{"$set": set})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.MatchedCount == 0 {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	h.logAction(r, "supplier.update", "supplier", id.Hex(), name, "")
	h.oneSupplier(w, r, sc, id)
}

func (h *Handler) oneSupplier(
	w http.ResponseWriter, r *http.Request, sc Scope, id primitive.ObjectID,
) {
	var row models.Supplier
	if err := h.Store.Suppliers.FindOne(r.Context(),
		sc.brandFilter(bson.M{"_id": id})).Decode(&row); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	httpx.JSON(w, http.StatusOK, row)
}

// AdminDeleteSupplier deactivates one.
//
// ⚠️ **Deactivated, not removed.** Every delivery ever entered points here, and
// a supplier deleted outright takes the meaning of a year of invoices with it —
// the same rule a warehouse follows. The deliveries keep their copy of the name
// either way, so history stays readable; what would be lost is the grouping.
func (h *Handler) AdminDeleteSupplier(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	sc, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.Store.Suppliers.UpdateOne(r.Context(),
		sc.brandFilter(bson.M{"_id": id}),
		bson.M{"$set": bson.M{"isActive": false, "updatedAt": time.Now()}})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.MatchedCount == 0 {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	h.logAction(r, "supplier.delete", "supplier", id.Hex(), "", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- What each supplier is costing, and what is still owed ----

type supplierTotal struct {
	SupplierID string `json:"supplierId"`
	Name       string `json:"name"`
	Phone      string `json:"phone,omitempty"`
	// Deliveries in the period and what they came to.
	Count int `json:"count"`
	Spent int `json:"spent"`
	// ⚠️ **Owed is not filtered by the period**, unlike everything beside it.
	// A March invoice is still a debt in May, and a figure that clears itself
	// when the month rolls over is not a debt at all — the same rule the
	// courier's cash in hand follows.
	Owed      int `json:"owed"`
	OwedCount int `json:"owedCount"`
}

// AdminSupplierReport is who we buy from, what it costs and what is unpaid.
func (h *Handler) AdminSupplierReport(w http.ResponseWriter, r *http.Request) {
	scope, sc, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	from, to, err := parseRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	named := map[primitive.ObjectID]models.Supplier{}
	if cur, err := h.Store.Suppliers.Find(r.Context(), sc.brandFilter(bson.M{})); err == nil {
		var rows []models.Supplier
		_ = cur.All(r.Context(), &rows)
		for _, s := range rows {
			named[s.ID] = s
		}
	}

	acc := map[primitive.ObjectID]*supplierTotal{}
	get := func(p models.Purchase) *supplierTotal {
		a, ok := acc[p.SupplierID]
		if ok {
			return a
		}
		// ⚠️ The stored name is the fallback, and an unnamed delivery groups
		// under the zero id rather than being dropped: a market run is real
		// spending, and a total that silently excludes it is worse than one
		// with an "unnamed" row in it.
		name := named[p.SupplierID].Name
		if name == "" {
			name = p.Supplier
		}
		a = &supplierTotal{
			SupplierID: p.SupplierID.Hex(),
			Name:       name,
			Phone:      named[p.SupplierID].Phone,
		}
		acc[p.SupplierID] = a
		return a
	}

	inPeriod := bson.M{}
	for k, v := range scope {
		inPeriod[k] = v
	}
	if rng := timeRange(from, to); len(rng) > 0 {
		inPeriod["at"] = rng
	}
	if cur, err := h.Store.Purchases.Find(r.Context(), inPeriod); err == nil {
		var rows []models.Purchase
		_ = cur.All(r.Context(), &rows)
		for _, p := range rows {
			a := get(p)
			a.Count++
			a.Spent += p.Total
		}
	}

	// Debts across the whole history — see the field note above.
	owedFilter := bson.M{"paid": bson.M{"$ne": true}}
	for k, v := range scope {
		owedFilter[k] = v
	}
	if cur, err := h.Store.Purchases.Find(r.Context(), owedFilter); err == nil {
		var rows []models.Purchase
		_ = cur.All(r.Context(), &rows)
		for _, p := range rows {
			a := get(p)
			a.Owed += p.Total
			a.OwedCount++
		}
	}

	out := make([]supplierTotal, 0, len(acc))
	for _, a := range acc {
		out = append(out, *a)
	}
	// ⚠️ Owed first, then by spend: the reason this page gets opened before a
	// delivery day is "who are we behind with", and that answer must not be
	// below the biggest supplier who is fully paid.
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].Owed > 0) != (out[j].Owed > 0) {
			return out[i].Owed > 0
		}
		if out[i].Owed != out[j].Owed {
			return out[i].Owed > out[j].Owed
		}
		return out[i].Spent > out[j].Spent
	})

	total, owed := 0, 0
	for _, a := range out {
		total += a.Spent
		owed += a.Owed
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"rows": out, "spent": total, "owed": owed,
	})
}

// AdminPayPurchase marks one delivery settled.
//
// ⚠️ Guarded by the unpaid filter rather than by id, exactly like a guest's
// slate: "paid" pressed on two screens must settle it once, and an invoice that
// is already settled has to answer 404 rather than quietly writing a second
// date over the first.
func (h *Handler) AdminPayPurchase(w http.ResponseWriter, r *http.Request) {
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
	// ⚠️ **Whether the notes came out of the safe is its own question.** A
	// delivery marked paid used to say the supplier was square and nothing at
	// all about which box got lighter — and paying a supplier at the door is the
	// most common way money leaves a restaurant's safe. Optional, because it is
	// as often settled by transfer or out of the drawer.
	var req struct {
		FromSafe bool `json:"fromSafe"`
	}
	// ⚠️ A decode failure is not an error here: this endpoint took no body at
	// all until now, and every screen that still sends none must keep working.
	_ = httpx.Decode(r, &req)

	filter := bson.M{"_id": id, "paid": bson.M{"$ne": true}}
	for k, v := range scope {
		filter[k] = v
	}
	now := time.Now()
	res, err := h.Store.Purchases.UpdateOne(r.Context(), filter,
		bson.M{"$set": bson.M{"paid": true, "paidAt": now}})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.MatchedCount == 0 {
		httpx.Error(w, http.StatusNotFound, "topilmadi yoki allaqachon to'langan")
		return
	}
	if req.FromSafe {
		var p models.Purchase
		if err := h.Store.Purchases.FindOne(r.Context(), bson.M{"_id": id}).Decode(&p); err == nil {
			h.recordSafeMovement(r.Context(), models.SafeEntry{
				BranchID: p.BranchID, Kind: models.SafeOut, Amount: p.Total,
				At: now, Category: "yetkazib berish", Note: p.Supplier,
				By: h.adminName(r), RefKind: models.SafeRefPurchase, RefID: p.ID,
			})
		}
	}
	h.logAction(r, "purchase.pay", "purchase", id.Hex(), "", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "paidAt": now})
}

// supplierNameFor freezes the name a delivery was entered against.
func (h *Handler) supplierNameFor(
	r *http.Request, sc Scope, id primitive.ObjectID, typed string,
) (primitive.ObjectID, string) {
	if id.IsZero() {
		return primitive.NilObjectID, clampText(typed, 120)
	}
	var row models.Supplier
	if err := h.Store.Suppliers.FindOne(r.Context(),
		sc.brandFilter(bson.M{"_id": id})).Decode(&row); err != nil {
		// Named a supplier this brand does not have: the id is dropped and the
		// typed text kept, so the delivery is still recorded.
		return primitive.NilObjectID, clampText(typed, 120)
	}
	return row.ID, strings.TrimSpace(row.Name)
}
