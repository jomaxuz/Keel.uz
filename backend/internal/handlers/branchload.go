package handlers

import (
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// How busy each kitchen is, and moving an order between them.
//
// ⚠️ **Deliberately not automatic.** The obvious idea — route a new order away
// from a branch that is behind — is worse than it looks: the next branch is
// further from the guest, so ten minutes saved in the kitchen come back as
// twenty on the road, and the food arrives colder for having been "balanced".
// It also punishes the branch that is quick by handing it everybody's work, and
// makes the same basket ordered twice in five minutes go to two different
// kitchens for reasons the guest cannot see. Whether to move an order at seven
// on a Friday depends on how many couriers are actually out, which no ticket
// count knows.
//
// So the panel is given the two things a person needs to make that call: what
// each kitchen is holding right now, and a way to move one order. The decision
// stays with whoever can see the room.

// branchLoad is one kitchen's queue, as the panel shows it.
type branchLoad struct {
	BranchID string `json:"branchId"`
	Name     string `json:"name"`
	// Accepted and being cooked right now: exactly what the pass is showing
	// (see StaffKitchen), so the office and the kitchen never disagree about
	// how much work is out.
	Cooking int `json:"cooking"`
	// Placed and not yet accepted by anybody.
	Pending int `json:"pending"`
	// How long the oldest of those has been waiting, in minutes. The number
	// that actually says "behind" — five tickets just accepted is a normal
	// evening, one ticket waiting forty minutes is not.
	OldestMin int `json:"oldestMin"`
}

// AdminBranchLoad answers "who is behind right now", per branch.
func (h *Handler) AdminBranchLoad(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx := r.Context()

	filter := bson.M{"isActive": true}
	if !scope.BrandID.IsZero() {
		filter["brandId"] = scope.BrandID
	}
	// A manager pinned to one branch sees one row. Not a restriction that
	// matters much here, but the rule lives in one place and this is not the
	// screen to make an exception on.
	if !scope.BranchID.IsZero() {
		filter["_id"] = scope.BranchID
	}
	branches, err := h.listBranchesFiltered(r, filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	now := time.Now()
	out := make([]branchLoad, 0, len(branches))
	for i := range branches {
		b := &branches[i]
		row := branchLoad{BranchID: b.ID.Hex(), Name: b.Name}

		// The pass's own filter, word for word: accepted, paid for or cash,
		// due now, not yet marked ready. A pre-order that is not due is not
		// work anybody is doing.
		cooking := bson.M{
			"branchId": b.ID,
			"status":   bson.M{"$in": bson.A{models.StatusConfirmed, models.StatusPreparing}},
			"queuedAt": bson.M{"$ne": nil, "$lte": now},
			"readyAt":  nil,
		}
		if n, err := h.Store.Orders.CountDocuments(ctx, cooking); err == nil {
			row.Cooking = int(n)
		}
		pending := bson.M{
			"branchId":      b.ID,
			"status":        string(models.StatusPending),
			"paymentStatus": bson.M{"$ne": models.PayPending},
			"queuedAt":      bson.M{"$ne": nil, "$lte": now},
		}
		if n, err := h.Store.Orders.CountDocuments(ctx, pending); err == nil {
			row.Pending = int(n)
		}

		// The oldest thing still waiting on this branch, whether it has been
		// accepted or not.
		waiting := bson.M{
			"branchId": b.ID,
			"status": bson.M{"$in": bson.A{
				models.StatusPending, models.StatusConfirmed, models.StatusPreparing,
			}},
			"queuedAt": bson.M{"$ne": nil, "$lte": now},
			"readyAt":  nil,
		}
		opts := options.FindOne().SetSort(bson.D{{Key: "queuedAt", Value: 1}})
		var oldest models.Order
		if err := h.Store.Orders.FindOne(ctx, waiting, opts).Decode(&oldest); err == nil &&
			oldest.QueuedAt != nil {
			row.OldestMin = int(now.Sub(*oldest.QueuedAt).Minutes())
			if row.OldestMin < 0 {
				row.OldestMin = 0
			}
		}
		out = append(out, row)
	}
	httpx.JSON(w, http.StatusOK, out)
}

type moveBranchRequest struct {
	BranchID string `json:"branchId" validate:"required"`
}

// AdminMoveOrderBranch hands one order to another kitchen.
//
// The manual half of the decision above. Guarded rather than trusted, because
// the ways this goes wrong are all silent:
//
//   - **The money does not change.** The fee and the total were agreed with the
//     guest; re-pricing them because the office moved the ticket would change
//     what somebody already said yes to. Same rule as correcting an address by
//     hand — the zone and the distance are facts worth updating, the bill is
//     not.
//   - **The order number keeps its old branch prefix.** It is printed on the
//     receipt and it is the guest's tracking link; renaming it to match the new
//     branch would break the one address they were given.
//   - **A courier is unassigned.** Couriers belong to a branch, and one from
//     the branch that no longer has the food cannot carry it.
//   - **"Ready" is cleared.** A new kitchen has not cooked this, and a ticket
//     that arrives already marked done never gets made.
//   - **A branch that has run out is refused**, with the dish named. Sending a
//     ticket to a kitchen that cannot fill it is the exact failure this whole
//     screen exists to avoid; the counter can lift the stop there first if they
//     mean it.
func (h *Handler) AdminMoveOrderBranch(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req moveBranchRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	target, err := objectID(req.BranchID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "filial noto'g'ri")
		return
	}
	ctx := r.Context()

	var order models.Order
	if err := h.Store.Orders.FindOne(ctx, bson.M{"_id": id}).Decode(&order); err != nil {
		httpx.Error(w, http.StatusNotFound, "buyurtma topilmadi")
		return
	}
	if order.Status == models.StatusDelivered || order.Status == models.StatusCancelled {
		httpx.Error(w, http.StatusConflict, "yakunlangan buyurtmani ko'chirib bo'lmaydi")
		return
	}
	if order.BranchID == target {
		httpx.JSON(w, http.StatusOK, order)
		return
	}
	// Both the branch being left and the one being joined have to be the
	// admin's to touch: a manager must not push their queue onto somebody
	// else's kitchen, nor pull work off it.
	if err := h.requireBranchAccess(r, order.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	if err := h.requireBranchAccess(r, target); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}

	branch, err := h.branchByID(r, target)
	if err != nil || !branch.IsActive {
		httpx.Error(w, http.StatusBadRequest, "filial topilmadi")
		return
	}
	// ⚠️ The menu belongs to the brand, so an order cannot cross into one whose
	// kitchen has never seen these dishes.
	if !order.BrandID.IsZero() && branch.BrandID != order.BrandID {
		httpx.Error(w, http.StatusBadRequest, "bu filial boshqa brendga tegishli")
		return
	}
	if names := h.soldOutAt(ctx, branch, order.Items); len(names) > 0 {
		httpx.Error(w, http.StatusConflict, names[0]+" bu filialda tugagan")
		return
	}

	set := bson.M{"branchId": target, "updatedAt": time.Now()}
	unset := bson.M{"readyAt": "", "courierId": "", "courierName": ""}
	if _, err := h.Store.Orders.UpdateByID(ctx, id, bson.M{
		"$set": set, "$unset": unset,
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ The move detaches the courier (`unset` above), so the rider who was
	// carrying it has to be told — otherwise they keep riding to an address
	// that now belongs to another kitchen.
	if !order.CourierID.IsZero() {
		h.syncCourierBusy(r, order.CourierID)
		h.courierLostOrder(order.CourierID, &order)
	}
	// The courier that was carrying it may now be free.
	if !order.CourierID.IsZero() {
		h.syncCourierBusy(r, order.CourierID)
	}
	h.logAction(r, ActOrderMoveBranch, "order", id.Hex(), "#"+order.Number, branch.Name)

	var updated models.Order
	_ = h.Store.Orders.FindOne(ctx, bson.M{"_id": id}).Decode(&updated)
	httpx.JSON(w, http.StatusOK, updated)
}
