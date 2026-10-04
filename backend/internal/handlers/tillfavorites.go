package handlers

// ---- The person's own corner of the menu ----
//
// Two lists above the categories on the till and the floor screen: what this
// person rang most in the last month, and what they pinned themselves.
//
// ⚠️ **Per person, because the habit is per person.** The bar sells tea and
// beer, the waiter on the terrace sells shashlik, the cashier sells takeaway
// samsa — a restaurant-wide "popular" list is the same twelve dishes for all of
// them and saves nobody a tap.

import (
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	// How far back "most used" looks. ⚠️ A month: long enough that one
	// banquet does not decide it, short enough that last season's menu does
	// not linger at the top.
	tillTopWindow = 30 * 24 * time.Hour
	tillTopCount  = 16
	// A pinned list is a shortlist; past this it is a second menu.
	tillMaxFavorites = 40
)

// StaffMenuMine is this person's favourites and most-rung dishes.
func (h *Handler) StaffMenuMine(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	favs := make([]string, 0, len(s.MenuFavorites))
	for _, id := range s.MenuFavorites {
		favs = append(favs, id.Hex())
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"favorites": favs,
		"top":       h.tillTopFor(r, s),
	})
}

// tillTopFor is what this person sold most, newest month, most first.
//
// ⚠️ **Who served or who closed, not who typed.** A line does not record who
// added it; the check does record its waiter and its cashier, and between them
// they are "the person whose hands this went through". A cashier closing the
// whole room's checks gets the room's favourites, which is also what they ring.
//
// ⚠️ Never nil — the empty-array rule every list here follows.
func (h *Handler) tillTopFor(r *http.Request, s models.Staff) []string {
	out := []string{}
	pipe := []bson.M{
		{"$match": bson.M{
			"branchId":       s.BranchID,
			"status":         bson.M{"$ne": models.StatusCancelled},
			"check.closedAt": bson.M{"$gte": time.Now().Add(-tillTopWindow)},
			"$or": []bson.M{
				{"check.serverId": s.ID},
				{"check.closedById": s.ID},
			},
		}},
		{"$unwind": "$items"},
		{"$match": bson.M{"items.void": bson.M{"$exists": false}}},
		{"$group": bson.M{"_id": "$items.menuItemId", "n": bson.M{"$sum": "$items.qty"}}},
		{"$sort": bson.D{{Key: "n", Value: -1}, {Key: "_id", Value: 1}}},
		{"$limit": tillTopCount},
	}
	cur, err := h.Store.Orders.Aggregate(r.Context(), pipe)
	if err != nil {
		return out
	}
	defer cur.Close(r.Context())
	for cur.Next(r.Context()) {
		var row struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if cur.Decode(&row) == nil && !row.ID.IsZero() {
			out = append(out, row.ID.Hex())
		}
	}
	return out
}

type tillFavoritesRequest struct {
	Favorites []string `json:"favorites"`
}

// StaffSaveMenuFavorites replaces this person's pinned dishes.
//
// ⚠️ **The whole list, not a toggle.** Two screens toggling one dish at once
// would each send "flip it" and leave it where it started; the list is what
// the screen is showing, so the list is what is saved.
func (h *Handler) StaffSaveMenuFavorites(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	var req tillFavoritesRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ids := []primitive.ObjectID{}
	seen := map[primitive.ObjectID]bool{}
	for _, raw := range req.Favorites {
		id, err := primitive.ObjectIDFromHex(raw)
		if err != nil || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
		if len(ids) >= tillMaxFavorites {
			break
		}
	}
	if _, err := h.Store.Staff.UpdateByID(r.Context(), s.ID,
		bson.M{"$set": bson.M{"menuFavorites": ids}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.Hex())
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"favorites": out})
}
