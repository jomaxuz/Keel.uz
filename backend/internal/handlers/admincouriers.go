package handlers

import (
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// courierPayload is what the admin form sends. Password is optional on update
// (empty = keep the current one).
type courierPayload struct {
	Name     string               `json:"name" validate:"required"`
	Phone    string               `json:"phone"`
	Username string               `json:"username" validate:"required"`
	Password string               `json:"password"`
	Vehicle  string               `json:"vehicle"`
	Status   models.CourierStatus `json:"status"`
	IsActive *bool                `json:"isActive"`

	PayoutMode     models.CourierPayout `json:"payoutMode"`
	PayoutPerOrder int                  `json:"payoutPerOrder"`
	PayoutPercent  int                  `json:"payoutPercent"`
}

// payoutOf normalises the payout rule coming from the form.
func payoutOf(req courierPayload) (models.CourierPayout, int, int) {
	mode := req.PayoutMode
	if mode != models.PayoutPerOrder && mode != models.PayoutPercent {
		mode = models.PayoutDeliveryFee
	}
	percent := req.PayoutPercent
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	perOrder := req.PayoutPerOrder
	if perOrder < 0 {
		perOrder = 0
	}
	return mode, perOrder, percent
}

// AdminListCouriers returns the selected branch's courier accounts with their
// last positions. Couriers belong to a branch: a rider in Samarqand must not
// appear on the dispatcher's list in Toshkent.
func (h *Handler) AdminListCouriers(w http.ResponseWriter, r *http.Request) {
	filter, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})
	cur, err := h.Store.Couriers.Find(r.Context(), filter, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	couriers := []models.Courier{}
	_ = cur.All(r.Context(), &couriers)
	httpx.JSON(w, http.StatusOK, couriers)
}

// AdminCreateCourier adds a courier account by hand — there is no self-signup.
func (h *Handler) AdminCreateCourier(w http.ResponseWriter, r *http.Request) {
	var req courierPayload
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	username := strings.TrimSpace(strings.ToLower(req.Username))
	if len(req.Password) < 5 {
		httpx.Error(w, http.StatusBadRequest, "parol kamida 5 belgi bo'lishi kerak")
		return
	}
	n, _ := h.Store.Couriers.CountDocuments(r.Context(), bson.M{"username": username})
	if n > 0 {
		httpx.Error(w, http.StatusConflict, "bu login band")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	mode, perOrder, percent := payoutOf(req)
	// A new rider joins the branch the panel is looking at.
	var branchID primitive.ObjectID
	if scope, err := h.adminScope(r); err == nil {
		branchID = scope.BranchID
		if branchID.IsZero() {
			if b, err := h.defaultBranch(r, scope.BrandID); err == nil {
				branchID = b.ID
			}
		}
	}
	now := time.Now()
	c := models.Courier{
		BranchID:       branchID,
		PayoutMode:     mode,
		PayoutPerOrder: perOrder,
		PayoutPercent:  percent,
		Name:           strings.TrimSpace(req.Name),
		Phone:          strings.TrimSpace(req.Phone),
		Username:       username,
		PasswordHash:   string(hash),
		Status:         models.CourierOff,
		Vehicle:        req.Vehicle,
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	res, err := h.Store.Couriers.InsertOne(r.Context(), c)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	c.ID = oidOf(res.InsertedID)
	h.logAction(r, ActCourierCreate, "courier", c.ID.Hex(), c.Name, c.Phone)
	httpx.JSON(w, http.StatusCreated, c)
}

// AdminUpdateCourier edits the profile; an empty password keeps the old one.
func (h *Handler) AdminUpdateCourier(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req courierPayload
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	username := strings.TrimSpace(strings.ToLower(req.Username))
	n, _ := h.Store.Couriers.CountDocuments(r.Context(),
		bson.M{"username": username, "_id": bson.M{"$ne": id}})
	if n > 0 {
		httpx.Error(w, http.StatusConflict, "bu login band")
		return
	}

	mode, perOrder, percent := payoutOf(req)
	set := bson.M{
		"name":           strings.TrimSpace(req.Name),
		"phone":          strings.TrimSpace(req.Phone),
		"username":       username,
		"vehicle":        req.Vehicle,
		"payoutMode":     mode,
		"payoutPerOrder": perOrder,
		"payoutPercent":  percent,
		"updatedAt":      time.Now(),
	}
	if req.Status != "" {
		set["status"] = req.Status
	}
	if req.IsActive != nil {
		set["isActive"] = *req.IsActive
	}
	if req.Password != "" {
		if len(req.Password) < 5 {
			httpx.Error(w, http.StatusBadRequest, "parol kamida 5 belgi bo'lishi kerak")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		set["passwordHash"] = string(hash)
	}
	if _, err := h.Store.Couriers.UpdateByID(r.Context(), id, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var c models.Courier
	_ = h.Store.Couriers.FindOne(r.Context(), bson.M{"_id": id}).Decode(&c)
	details := ""
	if req.Password != "" {
		details = "parol o'zgartirildi"
	}
	h.logAction(r, ActCourierUpdate, "courier", id.Hex(), c.Name, details)
	httpx.JSON(w, http.StatusOK, c)
}

// AdminDeleteCourier removes the account. Past orders keep `courierName`, so
// the receipts still read correctly.
func (h *Handler) AdminDeleteCourier(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	n, _ := h.Store.Orders.CountDocuments(r.Context(), bson.M{
		"courierId": id,
		"status": bson.M{"$nin": []models.OrderStatus{
			models.StatusDelivered, models.StatusCancelled,
		}},
	})
	if n > 0 {
		httpx.Error(w, http.StatusConflict,
			"kuryerda tugallanmagan buyurtma bor — avval uni boshqa kuryerga bering")
		return
	}
	var removed models.Courier
	_ = h.Store.Couriers.FindOne(r.Context(), bson.M{"_id": id}).Decode(&removed)
	if _, err := h.Store.Couriers.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActCourierDelete, "courier", id.Hex(), removed.Name, removed.Phone)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// AdminAssignCourier attaches (or, with an empty id, detaches) a courier to an
// order. Assigning marks the courier busy; detaching re-evaluates their state.
func (h *Handler) AdminAssignCourier(w http.ResponseWriter, r *http.Request) {
	orderID, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		CourierID string `json:"courierId"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	var previous models.Order
	_ = h.Store.Orders.FindOne(r.Context(), bson.M{"_id": orderID}).Decode(&previous)

	if strings.TrimSpace(req.CourierID) == "" {
		if _, err := h.Store.Orders.UpdateByID(r.Context(), orderID, bson.M{
			"$unset": bson.M{"courierId": "", "courierName": ""},
			"$set":   bson.M{"updatedAt": time.Now()},
		}); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !previous.CourierID.IsZero() {
			h.syncCourierBusy(r, previous.CourierID)
		}
		h.logAction(r, ActOrderCourier, "order", orderID.Hex(), "#"+previous.Number,
			"kuryer yechildi: "+previous.CourierName)
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	courierID, err := objectID(req.CourierID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid courier id")
		return
	}
	var c models.Courier
	if err := h.Store.Couriers.FindOne(r.Context(), bson.M{"_id": courierID}).Decode(&c); err != nil {
		httpx.Error(w, http.StatusNotFound, "kuryer topilmadi")
		return
	}
	// A rider can only carry their own branch's orders. Without this an owner
	// looking at "all branches" could hand a Samarqand order to a Toshkent
	// courier, who would then be blocked from ever closing it (the arrival
	// check would put them 300 km away).
	if !c.BranchID.IsZero() && !previous.BranchID.IsZero() && c.BranchID != previous.BranchID {
		httpx.Error(w, http.StatusBadRequest, "bu kuryer boshqa filialga biriktirilgan")
		return
	}
	if _, err := h.Store.Orders.UpdateByID(r.Context(), orderID, bson.M{
		"$set": bson.M{
			"courierId":   c.ID,
			"courierName": c.Name,
			"updatedAt":   time.Now(),
		},
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.syncCourierBusy(r, c.ID)
	if !previous.CourierID.IsZero() && previous.CourierID != c.ID {
		h.syncCourierBusy(r, previous.CourierID)
	}
	h.logAction(r, ActOrderCourier, "order", orderID.Hex(), "#"+previous.Number,
		"kuryer: "+c.Name)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "courier": c})
}
