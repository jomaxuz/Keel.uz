package handlers

import (
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- Today's sales, on the till ----
//
// ⚠️ **A cashier's questions about a closed check do not go through the
// panel.** "Where is the receipt for the table that just left", "did that one
// go through as cash", "print it again, they want it for work" — all of them
// happen minutes after the sale, at the counter, and the answer used to be a
// manager's login. What that produces is a panel session left open on a machine
// in the dining room, which is the customer base, the payment keys and the
// reports, guarded by nothing.
//
// ⚠️ **Today only, and this branch only.** Not paging back through the month:
// that is a question about the business, it belongs to the owner's screen, and
// a till that can read a year of sales is a till worth stealing. The window is
// the shift the person standing there worked.

// StaffClosedChecks lists sales closed today, newest first.
func (h *Handler) StaffClosedChecks(w http.ResponseWriter, r *http.Request) {
	// ⚠️ Cashier, not waiter. A closed check carries what was paid and how,
	// which is the drawer's business — and the waiter's screen already answers
	// the only question a waiter has about a closed table, which is that it is
	// gone.
	s, ok := h.tillStaff(w, r, models.PermCashier)
	if !ok {
		return
	}
	// From the open shift when there is one, and from midnight when there is
	// not. ⚠️ The shift is the better answer: a restaurant that works past
	// midnight would otherwise watch its own list empty halfway through the
	// night, in the middle of the busiest hour it has.
	from := startOfDay(time.Now())
	if shift, err := h.openCashShift(r, bson.M{"branchId": s.BranchID}); err == nil &&
		shift != nil && shift.OpenedAt.Before(from) {
		from = shift.OpenedAt
	}

	filter := bson.M{
		"branchId":       s.BranchID,
		"check.closedAt": bson.M{"$gte": from},
	}
	// ⚠️ Cancelled checks stay in the list. A table that was voided before
	// paying is exactly the row somebody comes looking for — "what happened to
	// table six" — and hiding it makes the till look like it lost a sale.
	cur, err := h.Store.Orders.Find(r.Context(), filter,
		options.Find().
			SetSort(bson.D{{Key: "check.closedAt", Value: -1}}).
			SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cur.Close(r.Context())

	now := time.Now()
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	out := make([]checkView, 0, 32)
	// What the list is worth reading for: the shift's own running total, how
	// much of it went back, and what left without being paid for.
	//
	// ⚠️ Computed over everything found, not over the rows the search left — a
	// filtered total nobody asked for is the number somebody writes down.
	//
	// ⚠️ **A debt is not in the total.** This figure sits at the top of a list
	// the cashier reads while standing at the drawer, and a slate folded into
	// it says money is here that is not. It gets its own line, as it does
	// everywhere else this figure is drawn.
	var total, refunded, owed, checks int
	for cur.Next(r.Context()) {
		var o models.Order
		if err := cur.Decode(&o); err != nil {
			continue
		}
		if o.Status != models.StatusCancelled {
			checks++
			switch {
			case paymentStatusOf(&o) == models.PayRefunded:
				refunded += o.Total
			case o.PaymentMethod == models.MethodDebt &&
				paymentStatusOf(&o) != models.PayPaid:
				owed += o.Total
			default:
				total += o.Total
			}
		}
		if q != "" && !matchesCheck(&o, q) {
			continue
		}
		out = append(out, viewCheck(&o, now))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"checks":   out,
		"total":    total,
		"refunded": refunded,
		"owed":     owed,
		"count":    checks,
		"from":     from,
	})
}

// matchesCheck is the counter's search: a number, a table, or a name.
//
// ⚠️ **In memory, over one shift's checks.** A text index for at most a few
// hundred documents that are already being read is a second thing to keep
// correct, and this list is short by construction.
func matchesCheck(o *models.Order, q string) bool {
	if strings.Contains(strings.ToLower(o.Number), q) {
		return true
	}
	if o.TableNumber != "" && strings.Contains(strings.ToLower(o.TableNumber), q) {
		return true
	}
	if o.Check != nil && strings.Contains(strings.ToLower(o.Check.ServerName), q) {
		return true
	}
	// The last digits of a phone: how a guest who has walked back in is found,
	// and the only thing they will offer without being asked twice.
	return o.Customer.Phone != "" && strings.Contains(o.Customer.Phone, q)
}
