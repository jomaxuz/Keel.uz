package handlers

import (
	"net/http"
	"strings"

	"restaurant-backend/internal/auth"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserMe returns the authenticated customer's profile.
func (h *Handler) UserMe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	id, err := objectID(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var user models.User
	if err := h.Store.Users.FindOne(r.Context(), bson.M{"_id": id}).Decode(&user); err != nil {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

// UserOrders returns the authenticated user's order history.
func (h *Handler) UserOrders(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	id, err := objectID(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(100)
	cur, err := h.Store.Orders.Find(r.Context(), bson.M{"userId": id}, opts)
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

// optionalUserID reads a user JWT from the Authorization header if present and
// returns the user's ObjectID. Used by public endpoints (like order creation)
// to link a record to a logged-in customer without requiring auth.
func (h *Handler) optionalUserID(r *http.Request) (primitive.ObjectID, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return primitive.NilObjectID, false
	}
	claims, err := auth.Parse(h.Cfg.JWTSecret, strings.TrimPrefix(header, "Bearer "))
	if err != nil || claims.Role != "user" {
		return primitive.NilObjectID, false
	}
	id, err := objectID(claims.UserID)
	if err != nil {
		return primitive.NilObjectID, false
	}
	return id, true
}
