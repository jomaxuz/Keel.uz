package handlers

import (
	"net/http"
	"strconv"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// courierEarning is what the courier is paid for one delivered order, under
// the payout rule configured for them. Pickup orders earn nothing — there is
// no delivery to make.
func courierEarning(o *models.Order, c *models.Courier) int {
	if o.Type != "delivery" {
		return 0
	}
	switch c.PayoutMode {
	case models.PayoutMonthly:
		// ⚠️ A salaried courier earns nothing *per delivery* — the wage is the
		// wage. Returning the fee here as well would pay them twice, and the
		// doubled figure would look exactly like a busy month.
		return 0
	case models.PayoutPerOrder:
		return c.PayoutPerOrder
	case models.PayoutPercent:
		return o.DeliveryFee * c.PayoutPercent / 100
	default: // PayoutDeliveryFee, and empty on older documents
		return o.DeliveryFee
	}
}

// CourierPeriod aggregates one time window of a courier's work.
type CourierPeriod struct {
	Orders   int `json:"orders"`
	Earnings int `json:"earnings"`
	// Cash the courier physically collected — what has to be handed back.
	Cash  int `json:"cash"`
	Total int `json:"total"` // order value carried, for context
}

// CourierStats is the earnings board shown to the courier and to the admin.
type CourierStats struct {
	Today CourierPeriod `json:"today"`
	Week  CourierPeriod `json:"week"`
	Month CourierPeriod `json:"month"`
	All   CourierPeriod `json:"all"`
	// Currently assigned and not yet delivered.
	Active int `json:"active"`
	// Cash actually still on the courier: everything they ever collected, less
	// what they have handed back. `All.Cash` alone only ever grows, so it
	// answers "how much have they carried", not "how much do they owe" — and
	// the second question is the one anybody asks.
	CashInHand int `json:"cashInHand"`
	// What has been handed over in total, for the courier's own record.
	CashSettled int `json:"cashSettled"`
}

// deliveredOrders returns every order this courier actually delivered, newest
// first. Cancelled ones never count towards earnings.
func (h *Handler) deliveredOrders(r *http.Request, courierID primitive.ObjectID, limit int64) ([]models.Order, error) {
	opts := options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}})
	if limit > 0 {
		opts.SetLimit(limit)
	}
	cur, err := h.Store.Orders.Find(r.Context(), bson.M{
		"courierId": courierID,
		"status":    models.StatusDelivered,
	}, opts)
	if err != nil {
		return nil, err
	}
	var orders []models.Order
	if err := cur.All(r.Context(), &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

// deliveredAt is when the order reached "delivered" — taken from the status
// history so the stats line up with reality even if the record is touched
// later for another reason.
func deliveredAt(o *models.Order) time.Time {
	for i := len(o.StatusHistory) - 1; i >= 0; i-- {
		if o.StatusHistory[i].Status == models.StatusDelivered {
			return o.StatusHistory[i].At
		}
	}
	return o.UpdatedAt
}

// buildCourierStats sums the periods from a courier's delivered orders.
func buildCourierStats(orders []models.Order, c *models.Courier, active int) CourierStats {
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := dayStart.AddDate(0, 0, -6)   // today + 6 previous days
	monthStart := dayStart.AddDate(0, 0, -29) // rolling 30 days

	var stats CourierStats
	stats.Active = active

	add := func(p *CourierPeriod, o *models.Order, earned int) {
		p.Orders++
		p.Earnings += earned
		p.Total += o.Total
		if o.PaymentMethod == "cash" {
			p.Cash += o.Total
		}
	}

	for i := range orders {
		o := &orders[i]
		earned := courierEarning(o, c)
		at := deliveredAt(o)
		add(&stats.All, o, earned)
		if at.After(monthStart) {
			add(&stats.Month, o, earned)
		}
		if at.After(weekStart) {
			add(&stats.Week, o, earned)
		}
		if at.After(dayStart) {
			add(&stats.Today, o, earned)
		}
	}
	return stats
}

// courierOrderRow is a delivered order plus what the courier earned on it, so
// the history table does not have to re-derive the payout rule.
type courierOrderRow struct {
	models.Order
	Earned      int       `json:"earned"`
	DeliveredAt time.Time `json:"deliveredAt"`
}

func rowsFor(orders []models.Order, c *models.Courier) []courierOrderRow {
	rows := make([]courierOrderRow, 0, len(orders))
	for i := range orders {
		o := orders[i]
		rows = append(rows, courierOrderRow{
			Order:       o,
			Earned:      courierEarning(&o, c),
			DeliveredAt: deliveredAt(&o),
		})
	}
	return rows
}

func (h *Handler) activeCount(r *http.Request, courierID primitive.ObjectID) int {
	n, err := h.Store.Orders.CountDocuments(r.Context(), bson.M{
		"courierId": courierID,
		"status": bson.M{"$nin": []models.OrderStatus{
			models.StatusDelivered, models.StatusCancelled,
		}},
	})
	if err != nil {
		return 0
	}
	return int(n)
}

// CourierMyStats is the courier's own earnings board.
func (h *Handler) CourierMyStats(w http.ResponseWriter, r *http.Request) {
	c, ok := h.courierFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	orders, err := h.deliveredOrders(r, c.ID, 0)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	stats := h.withSettlements(r, c.ID, buildCourierStats(orders, &c, h.activeCount(r, c.ID)))
	httpx.JSON(w, http.StatusOK, stats)
}

// CourierMyHistory returns the courier's delivered orders with the amount
// earned on each.
func (h *Handler) CourierMyHistory(w http.ResponseWriter, r *http.Request) {
	c, ok := h.courierFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	orders, err := h.deliveredOrders(r, c.ID, 100)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, rowsFor(orders, &c))
}

// AdminGetCourier is the dispatcher's view of one courier: profile, earnings
// and the full delivery history with receipts.
func (h *Handler) AdminGetCourier(w http.ResponseWriter, r *http.Request) {
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
	orders, err := h.deliveredOrders(r, c.ID, 200)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"courier":     c,
		"stats":       h.withSettlements(r, c.ID, buildCourierStats(orders, &c, h.activeCount(r, c.ID))),
		"settlements": h.settlementsOf(r, c.ID),
		// ⚠️ **Earned and paid, side by side, never one derived from the other.**
		// What a courier earned comes from their deliveries and the payout rule;
		// what they were handed is a document. Deriving either direction gives a
		// figure that moves when a rule is edited, months after the notes were
		// counted out.
		"payments": h.courierPayments(r, c.ID),
		"paid":     h.courierPaidTotal(r, c.ID),
		"orders":   rowsFor(orders, &c),
	})
}

// ---- Cash settlement ----

// settledCash is how much of a courier's collected cash has been handed back.
func (h *Handler) settledCash(r *http.Request, courierID primitive.ObjectID) int {
	cur, err := h.Store.Settlements.Find(r.Context(), bson.M{"courierId": courierID})
	if err != nil {
		return 0
	}
	var rows []models.CourierSettlement
	if err := cur.All(r.Context(), &rows); err != nil {
		return 0
	}
	sum := 0
	for _, s := range rows {
		sum += s.Amount
	}
	return sum
}

// withSettlements fills in the two cash figures that need the ledger.
func (h *Handler) withSettlements(r *http.Request, courierID primitive.ObjectID, st CourierStats) CourierStats {
	st.CashSettled = h.settledCash(r, courierID)
	st.CashInHand = st.All.Cash - st.CashSettled
	if st.CashInHand < 0 {
		// Handing over more than was collected is an accounting mistake, not a
		// debt the restaurant owes the courier. Show zero rather than a
		// negative that invites a second wrong correction.
		st.CashInHand = 0
	}
	return st
}

// settlementsOf lists a courier's handovers, newest first.
func (h *Handler) settlementsOf(r *http.Request, courierID primitive.ObjectID) []models.CourierSettlement {
	opts := options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(100)
	cur, err := h.Store.Settlements.Find(r.Context(), bson.M{"courierId": courierID}, opts)
	if err != nil {
		return []models.CourierSettlement{}
	}
	out := []models.CourierSettlement{}
	_ = cur.All(r.Context(), &out)
	return out
}

type settleRequest struct {
	Amount int    `json:"amount"`
	Note   string `json:"note"`
}

// AdminSettleCourierCash records the courier handing money over.
//
// Written as a ledger entry rather than by resetting a counter: "how much did
// Aziz hand in last Tuesday?" has to be answerable, and a counter that was
// zeroed cannot answer it.
func (h *Handler) AdminSettleCourierCash(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req settleRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Amount <= 0 {
		httpx.Error(w, http.StatusBadRequest, "summani yozing")
		return
	}
	var c models.Courier
	if err := h.Store.Couriers.FindOne(r.Context(), bson.M{"_id": id}).Decode(&c); err != nil {
		httpx.Error(w, http.StatusNotFound, "kuryer topilmadi")
		return
	}

	admin, _ := h.adminUser(r)
	takenBy := ""
	if admin != nil {
		takenBy = admin.Username
	}
	entry := models.CourierSettlement{
		CourierID: id,
		Amount:    req.Amount,
		TakenBy:   takenBy,
		Note:      clampText(req.Note, 200),
		At:        time.Now(),
	}
	if _, err := h.Store.Settlements.InsertOne(r.Context(), entry); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// The courier's own screen shows what they still owe; a number that drops
	// with no explanation is a number they come back and ask about.
	h.courierCashTaken(id, req.Amount)
	h.logAction(r, ActCourierSettle, "courier", id.Hex(), c.Name,
		formatSum(req.Amount))
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// formatSum is the amount as it reads in the activity log.
func formatSum(v int) string {
	return strconv.Itoa(v) + " so'm"
}
