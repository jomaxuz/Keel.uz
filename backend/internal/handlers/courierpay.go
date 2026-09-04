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

// ---- Couriers on the payroll ----
//
// ⚠️ **They were the only paid people with no pay period.** A courier's money
// was thought of as per-delivery and therefore continuous, so the screens could
// say what one had ever earned — a figure that only grows — and never what was
// owed *now*. But a courier is paid once a month like everybody else, and
// "what do we owe him" needs a window before it has an answer at all.
//
// ⚠️ **The same shape as a cook's row on purpose.** An owner settling up at the
// end of the month should not have to read two different screens with two
// different meanings of "earned" and "paid"; the kitchen and the bikes are the
// same question about different people.

// courierPayrollRows is every courier's own period, earnings and balance.
func (h *Handler) courierPayrollRows(r *http.Request, now time.Time) []PayrollRow {
	scope, _, err := h.orderScope(r)
	if err != nil {
		return nil
	}
	cur, err := h.Store.Couriers.Find(r.Context(), scope,
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil
	}
	var couriers []models.Courier
	if err := cur.All(r.Context(), &couriers); err != nil {
		return nil
	}
	branchNames := h.branchNames(r)
	rows := make([]PayrollRow, 0, len(couriers))
	for i := range couriers {
		c := &couriers[i]
		pFrom, pTo := payPeriodBounds(c.PayPeriod, now)
		fromKey, toKey := pFrom.Format(dayLayout), pTo.Format(dayLayout)

		earned := c.MonthlyRate
		if c.PayoutMode != models.PayoutMonthly {
			earned = h.courierEarnedBetween(r, c, pFrom, pTo.AddDate(0, 0, 1))
		}
		paid := h.courierPaidBetween(r, c.ID, pFrom, pTo.AddDate(0, 0, 1))

		rate := c.PayoutPerOrder
		switch c.PayoutMode {
		case models.PayoutMonthly:
			rate = c.MonthlyRate
		case models.PayoutPercent:
			rate = c.PayoutPercent
		case models.PayoutDeliveryFee, "":
			rate = 0
		}
		rows = append(rows, PayrollRow{
			// ⚠️ **`courierId`, never `staffId`.** The pay button on this row
			// posts to a different endpoint, and a row that lied about which
			// kind of person it described would pay a cook whose id happened to
			// be typed in a courier's field.
			CourierID:  c.ID.Hex(),
			Name:       c.Name,
			Position:   "courier",
			BranchName: branchNames[c.BranchID],
			PayPeriod:  models.StaffPayPeriod(payPeriodOf(c.PayPeriod)),
			Rate:       rate, IsActive: c.IsActive,
			From: fromKey, To: toKey,
			Earned: earned, Paid: paid, Due: earned - paid,
		})
	}
	return rows
}

// payPeriodOf reads an empty period as monthly, the way the bounds do.
func payPeriodOf(p models.StaffPayPeriod) string {
	if p == "" {
		return string(models.PeriodMonthly)
	}
	return string(p)
}

// courierEarnedBetween is what the deliveries in a window owe this courier.
//
// ⚠️ Computed from the orders and the courier's own rule rather than stored:
// the rule is a fact about the arrangement, and a total kept on the courier
// would be a second copy that drifts the first time an order is corrected.
func (h *Handler) courierEarnedBetween(
	r *http.Request, c *models.Courier, from, to time.Time,
) int {
	cur, err := h.Store.Orders.Find(r.Context(), bson.M{
		"courierId": c.ID,
		"status":    models.StatusDelivered,
		"updatedAt": bson.M{"$gte": from, "$lt": to},
	})
	if err != nil {
		return 0
	}
	var orders []models.Order
	if err := cur.All(r.Context(), &orders); err != nil {
		return 0
	}
	sum := 0
	for i := range orders {
		sum += courierEarning(&orders[i], c)
	}
	return sum
}

// courierPaidBetween is what was handed to this courier inside a window.
func (h *Handler) courierPaidBetween(
	r *http.Request, id primitive.ObjectID, from, to time.Time,
) int {
	total, _, err := h.sumField(r.Context(), h.Store.CourierPayments, bson.M{
		"courierId": id,
		"at":        bson.M{"$gte": from, "$lt": to},
	}, "$amount")
	if err != nil {
		return 0
	}
	return total
}
