package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// Warehouses: the bar, the kitchen, the cellar — see models/warehouse.go.

// AdminListWarehouses returns the stores of this branch, in the owner's order.
func (h *Handler) AdminListWarehouses(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := bson.M{}
	for k, v := range scope {
		filter[k] = v
	}
	cur, err := h.Store.Warehouses.Find(r.Context(), filter, options.Find().SetSort(
		bson.D{{Key: "sort", Value: 1}, {Key: "createdAt", Value: 1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.Warehouse{}
	_ = cur.All(r.Context(), &rows)
	httpx.JSON(w, http.StatusOK, map[string]any{"warehouses": rows})
}

type warehouseRequest struct {
	Name     string `json:"name"`
	Note     string `json:"note"`
	Sort     int    `json:"sort"`
	IsActive *bool  `json:"isActive"`
}

func (h *Handler) AdminCreateWarehouse(w http.ResponseWriter, r *http.Request) {
	_, sc, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req warehouseRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		httpx.Error(w, http.StatusBadRequest, "ombor nomi kerak")
		return
	}
	wh := models.Warehouse{
		Name:      name,
		Note:      strings.TrimSpace(req.Note),
		Sort:      req.Sort,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	// ⚠️ **The scope filter is not a branch id.** With a brand chosen but no
	// branch, `orderScope` puts `{$in: [...]}` here, the type assertion fails
	// and the store is saved with no branch at all — which Mongo omits, and an
	// `$in` filter does not match a missing field. The row went in and never
	// came back out of the list, so the owner created it again. `scopeBranch`
	// is what every sibling handler (a delivery, a write-off, a count) already
	// uses, and it falls back to the default branch.
	wh.BranchID = h.scopeBranch(r, sc)
	res, err := h.Store.Warehouses.InsertOne(r.Context(), wh)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	wh.ID = oidOf(res.InsertedID)
	h.logAction(r, ActWarehouseSave, "warehouse", wh.ID.Hex(), wh.Name, "")
	httpx.JSON(w, http.StatusCreated, wh)
}

func (h *Handler) AdminUpdateWarehouse(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	var req warehouseRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := bson.M{"updatedAt": time.Now(), "sort": req.Sort}
	if name := strings.TrimSpace(req.Name); name != "" {
		set["name"] = name
	}
	set["note"] = strings.TrimSpace(req.Note)
	if req.IsActive != nil {
		set["isActive"] = *req.IsActive
	}
	if _, err := h.Store.Warehouses.UpdateByID(r.Context(), id, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActWarehouseSave, "warehouse", id.Hex(), req.Name, "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// AdminDeleteWarehouse removes a store nothing is kept in.
//
// ⚠️ **Refused while any ingredient still points at it**, with the count, and
// the ingredient is not moved for you. A store deleted out from under its
// ingredients leaves them in a warehouse that does not exist, which is not the
// same as the default store — every count of the default store would silently
// start including the bar's vodka. Moving them automatically would be worse: it
// is a decision about where forty things live, made by a delete button.
func (h *Handler) AdminDeleteWarehouse(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	n, err := h.Store.Ingredients.CountDocuments(r.Context(), bson.M{"warehouseId": id})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n > 0 {
		httpx.Error(w, http.StatusConflict, refusalWarehouseInUse(int(n)))
		return
	}
	if _, err := h.Store.Warehouses.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActWarehouseDelete, "warehouse", id.Hex(), "", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// refusalWarehouseInUse is its own function so the wording is testable and the
// count cannot quietly drop out of it — "in use" without a number sends the
// owner hunting through a list of two hundred ingredients.
func refusalWarehouseInUse(n int) string {
	return "bu omborda " + itoa(n) + " ta masalliq bor — avval ularni boshqa omborga o'tkazing"
}
