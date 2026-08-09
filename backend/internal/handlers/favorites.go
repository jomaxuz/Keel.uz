package handlers

import (
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Dishes a guest marked to come back to.
//
// ⚠️ **On the account, not in the browser.** A heart kept in `localStorage` is a heart
// that disappears when they order from their husband's phone, and the whole point of it
// is the second visit. It is also the one piece of data here that says what somebody
// wants rather than what they bought — which is why it sits beside the addresses and the
// points rather than in a cookie.
//
// ⚠️ **A toggle, not two endpoints.** "Add" and "remove" as separate calls means a
// double tap can add twice or remove something already gone, and the client then has to
// know which it is. `$addToSet` and `$pull` make either direction idempotent, and the
// server answers with the list — so the button renders from the truth rather than from
// what it hoped happened.

// UserFavorites returns the guest's saved dishes, as menu items.
//
// The items themselves rather than their ids: the profile draws cards, and a list of ids
// would be a second round trip on the one screen that exists to show them.
func (h *Handler) UserFavorites(w http.ResponseWriter, r *http.Request) {
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
	ids := user.Favorites
	if len(ids) == 0 {
		// ⚠️ An empty array, never nil: a nil slice marshals as `null`, and
		// `null.length` in the browser is the crash this codebase has already shipped
		// twice.
		httpx.JSON(w, http.StatusOK, []models.MenuItem{})
		return
	}
	cur, err := h.Store.Menu.Find(r.Context(), bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := []models.MenuItem{}
	if err := cur.All(r.Context(), &items); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ A dish that was deleted or hidden simply is not in the answer. The id stays on
	// the account rather than being cleaned up: a menu item comes back after a
	// seasonal break, and quietly forgetting a guest's favourite is worse than showing
	// one fewer card for a month.
	//
	// Combos are decorated like anywhere else — their price and contents are computed,
	// never stored, so a card drawn from a raw document would show a set with no price.
	// nil branch: the profile lists dishes, not what one kitchen has left tonight —
	// "sold out here today" belongs on the menu page, where the guest is choosing.
	h.decorateCombos(r.Context(), items, nil)
	httpx.JSON(w, http.StatusOK, items)
}

// ToggleFavorite adds or removes one dish, and answers with the new list of ids.
func (h *Handler) ToggleFavorite(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	userID, err := objectID(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	itemID, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "taom id noto'g'ri")
		return
	}
	// The dish has to exist. A heart on an id nobody can order is a row in the
	// database and a card that never renders.
	if n, err := h.Store.Menu.CountDocuments(r.Context(),
		bson.M{"_id": itemID}); err != nil || n == 0 {
		httpx.Error(w, http.StatusNotFound, "taom topilmadi")
		return
	}

	var user models.User
	if err := h.Store.Users.FindOne(r.Context(), bson.M{"_id": userID}).Decode(&user); err != nil {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	on := !containsID(user.Favorites, itemID)
	op := "$pull"
	if on {
		op = "$addToSet"
	}
	if _, err := h.Store.Users.UpdateByID(r.Context(), userID, bson.M{
		op:     bson.M{"favorites": itemID},
		"$set": bson.M{"updatedAt": time.Now()},
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	next := make([]primitive.ObjectID, 0, len(user.Favorites)+1)
	for _, id := range user.Favorites {
		if id != itemID {
			next = append(next, id)
		}
	}
	if on {
		next = append(next, itemID)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"on": on, "favorites": next})
}

func containsID(list []primitive.ObjectID, id primitive.ObjectID) bool {
	for _, x := range list {
		if x == id {
			return true
		}
	}
	return false
}
