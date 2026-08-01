package handlers

import (
	"fmt"
	"net/http"
	"time"

	"restaurant-backend/internal/auth"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// ---- Auth ----

type courierLoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// CourierLogin issues a JWT with the "courier" role. Accounts are created by
// the restaurant in the admin panel; couriers cannot sign themselves up.
func (h *Handler) CourierLogin(w http.ResponseWriter, r *http.Request) {
	var req courierLoginRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var c models.Courier
	if err := h.Store.Couriers.FindOne(r.Context(), bson.M{"username": req.Username}).Decode(&c); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "login yoki parol noto'g'ri")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(c.PasswordHash), []byte(req.Password)) != nil {
		httpx.Error(w, http.StatusUnauthorized, "login yoki parol noto'g'ri")
		return
	}
	if !c.IsActive {
		httpx.Error(w, http.StatusForbidden, "hisob o'chirilgan — restoran bilan bog'laning")
		return
	}
	token, err := auth.Generate(h.Cfg.JWTSecret, c.ID.Hex(), "courier")
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"token": token, "courier": c})
}

// courierFromCtx loads the courier behind the request's JWT.
func (h *Handler) courierFromCtx(r *http.Request) (models.Courier, bool) {
	claims := middleware.ClaimsFrom(r.Context())
	if claims == nil {
		return models.Courier{}, false
	}
	id, err := objectID(claims.UserID)
	if err != nil {
		return models.Courier{}, false
	}
	var c models.Courier
	if err := h.Store.Couriers.FindOne(r.Context(), bson.M{"_id": id}).Decode(&c); err != nil {
		return models.Courier{}, false
	}
	return c, true
}

// CourierMe returns the signed-in courier's own profile.
func (h *Handler) CourierMe(w http.ResponseWriter, r *http.Request) {
	c, ok := h.courierFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

// ---- Shift status ----

type courierStatusRequest struct {
	Status models.CourierStatus `json:"status" validate:"required,oneof=off free busy"`
}

// CourierSetStatus flips the courier between off / free / busy.
func (h *Handler) CourierSetStatus(w http.ResponseWriter, r *http.Request) {
	c, ok := h.courierFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var req courierStatusRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	_, err := h.Store.Couriers.UpdateByID(r.Context(), c.ID, bson.M{
		"$set": bson.M{"status": req.Status, "updatedAt": time.Now()},
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"status": req.Status})
}

// ---- Location ----

type locationPoint struct {
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Accuracy float64 `json:"accuracy"`
	At       int64   `json:"at"` // unix ms; 0 = now
}

// courierLocationRequest accepts either a single point or a batch. The app
// buffers positions while offline and flushes them in one call, so only the
// newest point is kept — we store the last known position, not a track.
type courierLocationRequest struct {
	locationPoint
	Points []locationPoint `json:"points"`
}

// CourierUpdateLocation stores the courier's latest position.
func (h *Handler) CourierUpdateLocation(w http.ResponseWriter, r *http.Request) {
	c, ok := h.courierFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var req courierLocationRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	points := req.Points
	if len(points) == 0 {
		points = []locationPoint{req.locationPoint}
	}
	// Keep the freshest point of the batch.
	var newest *locationPoint
	for i := range points {
		p := points[i]
		if p.Lat == 0 && p.Lng == 0 {
			continue
		}
		if newest == nil || p.At > newest.At {
			newest = &points[i]
		}
	}
	if newest == nil {
		httpx.Error(w, http.StatusBadRequest, "koordinata yo'q")
		return
	}

	at := time.Now()
	if newest.At > 0 {
		at = time.UnixMilli(newest.At)
	}
	loc := models.CourierLocation{
		Lat:      newest.Lat,
		Lng:      newest.Lng,
		Accuracy: newest.Accuracy,
		At:       at,
	}
	if _, err := h.Store.Couriers.UpdateByID(r.Context(), c.ID, bson.M{
		"$set": bson.M{"location": loc, "updatedAt": time.Now()},
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "at": at})
}

// ---- Orders ----

// CourierOrders lists the orders assigned to this courier. `?all=1` includes
// finished ones (delivery history); by default only the active ones.
func (h *Handler) CourierOrders(w http.ResponseWriter, r *http.Request) {
	c, ok := h.courierFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	filter := bson.M{"courierId": c.ID}
	if r.URL.Query().Get("all") != "1" {
		filter["status"] = bson.M{"$nin": []models.OrderStatus{
			models.StatusDelivered, models.StatusCancelled,
		}}
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(100)
	cur, err := h.Store.Orders.Find(r.Context(), filter, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var orders []models.Order
	_ = cur.All(r.Context(), &orders)
	if orders == nil {
		orders = []models.Order{}
	}
	httpx.JSON(w, http.StatusOK, orders)
}

// CourierAdvanceOrder lets the courier move their own order forward: picked up
// ("on_the_way") and handed over ("delivered"). Nothing else is allowed — the
// kitchen owns the earlier stages.
func (h *Handler) CourierAdvanceOrder(w http.ResponseWriter, r *http.Request) {
	c, ok := h.courierFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Status models.OrderStatus `json:"status" validate:"required,oneof=on_the_way delivered"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	var order models.Order
	if err := h.Store.Orders.FindOne(r.Context(),
		bson.M{"_id": id, "courierId": c.ID}).Decode(&order); err != nil {
		httpx.Error(w, http.StatusNotFound, "buyurtma topilmadi")
		return
	}

	// "Delivered" may only be pressed at the customer's door. The client
	// disables the button too, but that is only a hint — this is the rule.
	if req.Status == models.StatusDelivered {
		if msg := h.arrivalBlocked(r, &order, &c); msg != "" {
			httpx.Error(w, http.StatusBadRequest, msg)
			return
		}
	}

	now := time.Now()
	if _, err := h.Store.Orders.UpdateByID(r.Context(), id, bson.M{
		"$set":  bson.M{"status": req.Status, "updatedAt": now},
		"$push": bson.M{"statusHistory": models.StatusEvent{Status: req.Status, At: now}},
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Delivering frees the courier up again, unless they still hold others.
	if req.Status == models.StatusDelivered {
		h.syncCourierBusy(r, c.ID)
		// The courier closing the order is the usual way it completes, so the
		// cashback is paid here too — not only when an admin does it by hand.
		var order models.Order
		if err := h.Store.Orders.FindOne(r.Context(),
			bson.M{"_id": id}).Decode(&order); err == nil {
			h.awardPoints(r.Context(), &order)
		}
	} else {
		_, _ = h.Store.Couriers.UpdateByID(r.Context(), c.ID,
			bson.M{"$set": bson.M{"status": models.CourierBusy, "updatedAt": now}})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"status": req.Status, "at": now})
}

// syncCourierBusy sets the courier to busy or free depending on whether any
// active order is still assigned to them. Off-shift couriers are left alone.
func (h *Handler) syncCourierBusy(r *http.Request, courierID primitive.ObjectID) {
	var c models.Courier
	if err := h.Store.Couriers.FindOne(r.Context(), bson.M{"_id": courierID}).Decode(&c); err != nil {
		return
	}
	if c.Status == models.CourierOff {
		return
	}
	n, err := h.Store.Orders.CountDocuments(r.Context(), bson.M{
		"courierId": courierID,
		"status": bson.M{"$nin": []models.OrderStatus{
			models.StatusDelivered, models.StatusCancelled,
		}},
	})
	if err != nil {
		return
	}
	status := models.CourierFree
	if n > 0 {
		status = models.CourierBusy
	}
	_, _ = h.Store.Couriers.UpdateByID(r.Context(), courierID,
		bson.M{"$set": bson.M{"status": status, "updatedAt": time.Now()}})
}

// maxLocationAge is how stale a courier's reported position may be before the
// arrival check treats it as unknown.
const maxLocationAge = 10 * time.Minute

// arrivalBlocked returns a message when the courier is not allowed to mark this
// order delivered yet, or "" when they are. Pickup orders and orders without
// coordinates are never blocked, and the restaurant can switch the check off by
// setting arrivalRadiusM to 0.
//
// Note the deliberate escape hatch: this only guards the courier app. If GPS
// fails on the road, the admin can still close the order from the panel.
func (h *Handler) arrivalBlocked(r *http.Request, order *models.Order, c *models.Courier) string {
	if order.Type != "delivery" || (order.Address.Lat == 0 && order.Address.Lng == 0) {
		return ""
	}
	// The rule belongs to the branch that took the order — a city-centre branch
	// may want 100 m, a village one 500.
	branch, err := h.branchByID(r, order.BranchID)
	if err != nil {
		return ""
	}
	radius := branch.Delivery.ArrivalRadiusM
	if radius <= 0 {
		return ""
	}
	if c.Location == nil || (c.Location.Lat == 0 && c.Location.Lng == 0) {
		return "Joylashuv aniqlanmadi — ilovada GPS yoqilganini tekshiring"
	}
	if time.Since(c.Location.At) > maxLocationAge {
		return "Joylashuv eskirgan — ilovani ochib, GPS yangilanishini kuting"
	}
	meters := haversineKm(
		c.Location.Lat, c.Location.Lng,
		order.Address.Lat, order.Address.Lng,
	) * 1000
	if meters > float64(radius) {
		return fmt.Sprintf(
			"Mijoz manzilidan %.0f m uzoqdasiz — yetkazildi deb belgilash uchun %d m ichida bo'lishingiz kerak",
			meters, radius)
	}
	return ""
}
