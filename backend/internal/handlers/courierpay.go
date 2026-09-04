package handlers

// ---- Paying a courier ----
//
// ⚠️ **The one cost the financial report admitted it could not see.**
// `courierEarning` has always worked out what a delivery owes its courier, so
// the arithmetic existed — but the money changed hands in the corridor and
// nothing recorded it. The report carried a comment saying couriers were
// "absent rather than guessed", which is honest and still a hole: a restaurant
// paying couriers every week had no screen that knew it, and a courier asking
// "was I paid for last week?" had only somebody's memory to go on.
//
// ⚠️ **Earned is derived, paid is recorded, and the two must stay apart.** What
// a courier earned comes out of the deliveries and their payout rule — it is a
// fact about work done. What they were handed is a fact about money, and only
// the second one can be corrected by writing a document. Deriving one from the
// other in either direction produces a figure that changes when a rule is
// edited, months after the notes were counted out.

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// courierPayments is what this courier has been handed, newest first.
func (h *Handler) courierPayments(
	r *http.Request, courierID primitive.ObjectID,
) []models.CourierPayment {
	opts := options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(50)
	cur, err := h.Store.CourierPayments.Find(r.Context(),
		bson.M{"courierId": courierID}, opts)
	if err != nil {
		return []models.CourierPayment{}
	}
	// ⚠️ Empty slice, never nil: Go marshals a nil slice as `null` and the
	// screen reads `.length` off it. This has bitten twice.
	out := []models.CourierPayment{}
	_ = cur.All(r.Context(), &out)
	return out
}

// courierPaidTotal is everything ever handed to this courier as pay.
//
// ⚠️ Summed from the ledger on every read rather than kept on the courier: a
// stored running total is a second copy of an answer the rows already contain,
// and it drifts the first time one is corrected — silently, in the figure
// somebody is owed.
func (h *Handler) courierPaidTotal(
	r *http.Request, courierID primitive.ObjectID,
) int {
	total, _, err := h.sumField(r.Context(), h.Store.CourierPayments,
		bson.M{"courierId": courierID}, "$amount")
	if err != nil {
		return 0
	}
	return total
}

// AdminPayCourier records money handed to a courier for their work.
func (h *Handler) AdminPayCourier(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var c models.Courier
	if err := h.Store.Couriers.FindOne(r.Context(), bson.M{"_id": id}).Decode(&c); err != nil {
		httpx.Error(w, http.StatusNotFound, "kuryer topilmadi")
		return
	}
	if err := h.requireBranchAccess(r, c.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req paymentPayload
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Amount <= 0 {
		httpx.Error(w, http.StatusBadRequest, "summani yozing")
		return
	}

	now := time.Now()
	// The window this settles. Left as typed when given, and otherwise this
	// calendar month — the case the button is pressed in nine times out of ten.
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).Format(dayLayout)
	to := now.Format(dayLayout)
	if v, err := parseDay(req.From); err == nil {
		from = v.Format(dayLayout)
	}
	if v, err := parseDay(req.To); err == nil {
		to = v.Format(dayLayout)
	}
	if to < from {
		from, to = to, from
	}

	p := models.CourierPayment{
		CourierID: c.ID,
		BranchID:  c.BranchID,
		Amount:    req.Amount,
		From:      from,
		To:        to,
		PaidBy:    h.adminName(r),
		Note:      clampText(req.Note, 200),
		At:        now,
	}
	res, err := h.Store.CourierPayments.InsertOne(r.Context(), p)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	p.ID = oidOf(res.InsertedID)

	// ⚠️ Asked, not assumed — a courier is as likely to be paid out of the
	// drawer they just handed cash into as out of the office box, and a safe
	// that guessed would be a confident figure about money still sitting there.
	if req.FromSafe {
		h.recordSafeMovement(r.Context(), models.SafeEntry{
			BranchID: c.BranchID, Kind: models.SafeOut, Amount: p.Amount,
			At: p.At, Category: "kuryer ish haqi", Note: c.Name, By: p.PaidBy,
			RefKind: models.SafeRefCourierPay, RefID: p.ID,
		})
	}

	h.logAction(r, "courier.pay", "courier", c.ID.Hex(), c.Name, formatSum(p.Amount))
	httpx.JSON(w, http.StatusCreated, p)
}

// AdminDeleteCourierPayment removes one entered by mistake.
//
// ⚠️ The safe's row stays: the notes left the box. Deleting the payment says
// "this was not pay", not "the money is back" — the same rule a deleted cost
// follows.
func (h *Handler) AdminDeleteCourierPayment(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "paymentId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var p models.CourierPayment
	if err := h.Store.CourierPayments.FindOne(r.Context(), bson.M{"_id": id}).Decode(&p); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	if err := h.requireBranchAccess(r, p.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	if _, err := h.Store.CourierPayments.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, "courier.pay.delete", "courier", p.CourierID.Hex(), "",
		formatSum(p.Amount))
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
