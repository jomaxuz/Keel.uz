package handlers

import (
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- Joining two checks ----
//
// ⚠️ **The other half of splitting, and just as ordinary.** Two friends at the
// counter move to a table; a couple is joined by four more; two tables push
// together for a birthday. Without this the waiter reads one check out loud
// while retyping it into another — which loses the times, the courses and the
// audit trail, and re-fires food the kitchen has already cooked.
//
// ⚠️ **The absorbed check is cancelled, not deleted.** It carries voided lines,
// a number that may already be on a printed bill, and the record of who opened
// it. Deleting it would erase all three; cancelling leaves a document that
// explains itself and — since a cancelled check counts as nothing anywhere —
// keeps the evening's takings and covers correct.

type mergeRequest struct {
	// The check that survives. Everything moves onto it.
	IntoID string `json:"intoId"`
}

// StaffMergeChecks moves a whole check onto another and closes the empty one.
func (h *Handler) StaffMergeChecks(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	from, ok := h.loadCheck(w, r, s)
	if !ok || !requireOpen(w, from) {
		return
	}
	var req mergeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	toID, err := objectID(req.IntoID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "chek topilmadi")
		return
	}
	if toID == from.ID {
		httpx.Error(w, http.StatusBadRequest, "chekni o'ziga qo'shib bo'lmaydi")
		return
	}
	// ⚠️ The destination is loaded through the same branch filter as the
	// source: without it a waiter could type another branch's check id and
	// move a table's food onto a bill in a different building.
	var to models.Order
	if err := h.Store.Orders.FindOne(r.Context(),
		checkFilter(toID, s.BranchID)).Decode(&to); err != nil {
		httpx.Error(w, http.StatusNotFound, "chek topilmadi")
		return
	}
	if !requireOpen(w, &to) {
		return
	}

	now := time.Now()
	var moved []models.OrderItem
	for _, it := range from.Items {
		// ⚠️ Voided lines stay behind, on the check they were written off.
		// Carrying them would move the blame onto a bill whose waiter never
		// took that dish off — and the cancelled document keeps the record.
		if it.Live() {
			moved = append(moved, it)
		}
	}
	to.Items = append(to.Items, moved...)
	// ⚠️ Covers add up: two tables pushed together seat both parties, and the
	// number this feeds — average per guest — is one of the two a dining room
	// is actually run on.
	if from.Check != nil && to.Check != nil {
		to.Check.Guests += from.Check.Guests
	}
	toSet := bson.M{
		"items":        to.Items,
		"check.guests": to.Check.Guests,
		"updatedAt":    now,
	}
	applyCheckTotals(&to, toSet)
	// Food already cooking makes the destination the kitchen's, exactly as
	// moving lines does.
	if to.QueuedAt == nil {
		for _, it := range moved {
			if it.FiredAt != nil {
				toSet["queuedAt"] = now
				break
			}
		}
	}
	if _, err := h.Store.Orders.UpdateOne(r.Context(),
		checkFilter(to.ID, s.BranchID), bson.M{"$set": toSet}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// ⚠️ Emptied **after** the destination is written, and its own lines are
	// left on it as the record. If this write fails the food is on two checks
	// and a human can see both; the other order would leave it on none.
	fromSet := bson.M{
		"status":    models.StatusCancelled,
		"updatedAt": now,
		// The reason a month later, in the words somebody will need: which
		// bill this table's food ended up on.
		"cancelReason": "birlashtirildi → " + to.Number,
		"mergedIntoId": to.ID,
	}
	if _, err := h.Store.Orders.UpdateOne(r.Context(),
		checkFilter(from.ID, s.BranchID), bson.M{
			"$set": fromSet,
			"$push": bson.M{"statusHistory": models.StatusEvent{
				Status: models.StatusCancelled, At: now,
			}},
		}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.JSON(w, http.StatusOK, viewCheck(&to, now))
}
