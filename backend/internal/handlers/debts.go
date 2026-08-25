package handlers

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

// ---- What guests owe ----
//
// ⚠️ **A debt is the sale itself, not a second document.** The obvious design
// is a ledger beside the orders — and it produces two numbers for one meal that
// disagree the first time anybody edits either. A check closed as a debt is
// `delivered` and `unpaid`: the food went out, the money did not arrive, and
// every screen that already understands orders understands this one.
//
// ⚠️ **The repayment is dated when the money arrives, not when the meal was
// eaten.** Tuesday's dinner settled on Friday is Friday's takings and lands in
// Friday's drawer — anything else would leave a cashier counting cash against a
// figure from three days ago.

type debtRow struct {
	OrderID string    `json:"orderId"`
	Number  string    `json:"number"`
	At      time.Time `json:"at"`
	Total   int       `json:"total"`
	Note    string    `json:"note,omitempty"`
	Table   string    `json:"table,omitempty"`
	// Who let it go on the slate. The name is the point of the record: a debt
	// nobody signed is one nobody can ask about.
	By string `json:"by,omitempty"`
}

// debtFilter is every unpaid debt in scope.
func debtFilter(scope bson.M) bson.M {
	f := bson.M{
		"paymentMethod": models.MethodDebt,
		"paymentStatus": bson.M{"$ne": models.PayPaid},
		// ⚠️ A cancelled check is not owed. It is the one case where the food
		// never left, and chasing somebody for it is how a restaurant loses a
		// regular over its own bookkeeping.
		"status": bson.M{"$ne": models.StatusCancelled},
	}
	for k, v := range scope {
		f[k] = v
	}
	return f
}

// AdminDebts lists what is owed, newest first — optionally for one customer.
func (h *Handler) AdminDebts(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := debtFilter(scope)
	if raw := r.URL.Query().Get("userId"); raw != "" {
		id, err := objectID(raw)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid userId")
			return
		}
		filter["userId"] = id
	}
	cur, err := h.Store.Orders.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var orders []models.Order
	_ = cur.All(r.Context(), &orders)

	rows := make([]debtRow, 0, len(orders))
	total := 0
	byUser := map[string]int{}
	for _, o := range orders {
		row := debtRow{
			OrderID: o.ID.Hex(), Number: o.Number, Total: o.Total,
			Note: o.DebtNote, Table: o.TableNumber,
			At: o.CreatedAt.In(time.Local),
		}
		if o.Check != nil {
			row.By = o.Check.ClosedBy
			if o.Check.ClosedAt != nil {
				row.At = o.Check.ClosedAt.In(time.Local)
			}
		}
		rows = append(rows, row)
		total += o.Total
		if !o.UserID.IsZero() {
			byUser[o.UserID.Hex()] += o.Total
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"debts": rows,
		"total": total,
		// Per customer, so the list of people can be drawn without asking for
		// every debt again. ⚠️ Money, not a count: "four debts" is not a number
		// anybody chases.
		"byUser": byUser,
	})
}

type debtPaymentRequest struct {
	// How the money actually arrived. ⚠️ Never "debt" again: that would leave
	// the sale unpaid and produce a repayment that changed nothing.
	Method string `json:"method"`
	Note   string `json:"note"`
}

// AdminPayDebt records that a debt was settled.
//
// ⚠️ **The sale becomes paid, dated now.** It does not become a new sale: the
// dinner happened on Tuesday and appears in Tuesday's covers and dish counts,
// while the money appears on Friday — in Friday's takings, Friday's drawer and
// Friday's shift, because that is the day somebody has to count it.
func (h *Handler) AdminPayDebt(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req debtPaymentRequest
	if err := httpx.DecodeOptional(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	method := req.Method
	if method == "" {
		method = models.ProviderCash
	}
	if method == models.MethodDebt || !tillMethods[method] {
		httpx.Error(w, http.StatusBadRequest, "noma'lum to'lov turi")
		return
	}

	filter := debtFilter(scope)
	filter["_id"] = id
	now := time.Now()
	set := bson.M{
		"paymentMethod": method,
		"paymentStatus": models.PayPaid,
		"paidAt":        now,
		"updatedAt":     now,
	}
	if note := clampText(req.Note, 200); note != "" {
		set["debtNote"] = note
	}
	// ⚠️ Guarded by the debt filter rather than by the id alone: two people
	// pressing "paid" on two screens must take the money once, and a debt that
	// was cancelled or already settled must refuse rather than quietly move a
	// second payment into today's drawer.
	res, err := h.Store.Orders.UpdateOne(r.Context(), filter, bson.M{"$set": set})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.MatchedCount == 0 {
		httpx.Error(w, http.StatusNotFound, "qarz topilmadi yoki allaqachon yopilgan")
		return
	}
	h.logAction(r, "debt.paid", "order", id.Hex(), method, req.Note)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "paidAt": now})
}

// debtsOf is what one customer owes, for their card.
func (h *Handler) debtsOf(
	r *http.Request, userID primitive.ObjectID,
) (int, int) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		return 0, 0
	}
	filter := debtFilter(scope)
	filter["userId"] = userID
	cur, err := h.Store.Orders.Find(r.Context(), filter)
	if err != nil {
		return 0, 0
	}
	var orders []models.Order
	if err := cur.All(r.Context(), &orders); err != nil {
		return 0, 0
	}
	total := 0
	for _, o := range orders {
		total += o.Total
	}
	return total, len(orders)
}
