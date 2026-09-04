package handlers

// ---- Perechisleniye: the money a rail is still holding ----
//
// ⚠️ **The whole point is the comparison, not the ledger.** Recording that
// 9 500 000 arrived from Uzum Tezkor in March is bookkeeping; putting it beside
// "you sold 12 000 000 through Uzum Tezkor in March, and they kept 1 800 000"
// is the sentence that catches an underpayment. One number is a receipt, three
// are a control.
//
// ⚠️ **What arrives is not revenue.** The sale was counted the day the guest
// paid — this is that same money changing location, exactly like cash carried
// from the drawer to the safe. Only the commission is a cost, and only the
// commission reaches the financial report. Guarded by a test, because a doubled
// revenue figure looks entirely plausible.

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// payoutRails is every payment method whose money somebody else holds for a
// while.
//
// ⚠️ **Cash, transfer and the slate are deliberately absent.** Cash is in the
// drawer, a transfer lands directly, and a debt is owed by a named guest and
// already has its own line. Listing them here would invent a payout that is
// never coming and leave a balance nobody can ever clear.
func (h *Handler) payoutRails(r *http.Request) []models.AggregatorAccount {
	s := h.paymentSettings(r.Context())
	rails := []models.AggregatorAccount{
		{ID: models.MethodCard, Name: "Terminal"},
	}
	for _, p := range []struct{ id, name string }{
		{models.ProviderClick, "Click"},
		{models.ProviderPayme, "Payme"},
		{models.ProviderUzum, "Uzum"},
		{models.ProviderAtmos, "ATMOS"},
	} {
		if s.Configured(p.id) {
			rails = append(rails, models.AggregatorAccount{ID: p.id, Name: p.name})
		}
	}
	return append(rails, s.EnabledAggregators()...)
}

// AdminPayouts is what has arrived, and what has not.
func (h *Handler) AdminPayouts(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	cur, err := h.Store.Payouts.Find(r.Context(), scopeFilter(scope),
		options.Find().SetSort(bson.D{{Key: "receivedAt", Value: -1}}).SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.Payout{}
	_ = cur.All(r.Context(), &rows)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"payouts":  rows,
		"balances": h.payoutBalances(r, scope, rows),
		"rails":    h.payoutRails(r),
	})
}

// payoutBalances answers "who still owes us what", per rail.
func (h *Handler) payoutBalances(
	r *http.Request, scope bson.M, rows []models.Payout,
) []models.PayoutBalance {
	// Everything already recorded, folded per rail.
	type agg struct {
		received, commission int
		through              string
		last                 *time.Time
	}
	seen := map[string]*agg{}
	for i := range rows {
		p := rows[i]
		a := seen[p.Provider]
		if a == nil {
			a = &agg{}
			seen[p.Provider] = a
		}
		a.received += p.Net
		a.commission += p.Commission
		// ⚠️ The **latest** period end, not the latest payout: statements
		// arrive out of order often enough (a corrected March lands after
		// April), and taking the newest document's period would reopen sales
		// that are already settled.
		if p.PeriodTo > a.through {
			a.through = p.PeriodTo
		}
		if a.last == nil || p.ReceivedAt.After(*a.last) {
			at := p.ReceivedAt
			a.last = &at
		}
	}

	out := []models.PayoutBalance{}
	for _, rail := range h.payoutRails(r) {
		a := seen[rail.ID]
		if a == nil {
			a = &agg{}
		}
		sold, n := h.soldThrough(r, scope, rail.ID, a.through)
		// ⚠️ A rail with no sales and no payouts is left off the screen
		// entirely: a restaurant that has never taken a card should not be
		// looking at a row of zeroes trying to work out what it means.
		if sold == 0 && a.received == 0 && a.commission == 0 {
			continue
		}
		out = append(out, models.PayoutBalance{
			Provider: rail.ID, Name: rail.Name,
			Sold: sold, Count: n, SettledThrough: a.through,
			Received: a.received, Commission: a.commission, LastAt: a.last,
		})
	}
	return out
}

// soldThrough is what was sold on a rail after the last settled period.
//
// ⚠️ **Paid orders only, and cancelled ones never.** A pending online payment
// is a guest still holding their phone; counting it as money a provider owes us
// would put a debt on somebody who was never given anything.
func (h *Handler) soldThrough(
	r *http.Request, scope bson.M, provider, through string,
) (int, int) {
	filter := scopeFilter(scope)
	filter["paymentMethod"] = provider
	filter["paymentStatus"] = models.PayPaid
	filter["status"] = bson.M{"$ne": models.StatusCancelled}
	if through != "" {
		if day, err := parseDay(through); err == nil {
			// The period end is inclusive, so the boundary is the next midnight.
			filter["createdAt"] = bson.M{"$gte": day.AddDate(0, 0, 1)}
		}
	}
	total, n, err := h.sumField(r.Context(), h.Store.Orders, filter, "$total")
	if err != nil {
		return 0, 0
	}
	return total, n
}

// AdminPayoutExpected is what our own records say a rail collected in a window.
//
// ⚠️ **This is the integration that actually exists today, and it is better
// than the one everybody asks for.** For Click, Payme, Uzum and ATMOS the
// provider already calls this server to confirm every payment — so what they
// collected is not something we need an API to be told; we watched it happen,
// order by order. The statement can therefore be checked against our own
// evidence rather than typed from theirs and believed.
//
// ⚠️ **A suggestion, never a substitute.** It fills the form; the owner still
// types what the statement says, and the difference between the two is the
// whole point. Writing our own figure into `gross` would produce a payout that
// always reconciles perfectly and never catches anything.
//
// For the marketplaces there is nothing to reconcile against yet: a Yandex Eats
// order reaches this system only if somebody rings it up here, so the figure is
// as good as the counter's discipline. Said on screen rather than hidden.
func (h *Handler) AdminPayoutExpected(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	q := r.URL.Query()
	provider := strings.TrimSpace(q.Get("provider"))
	if provider == "" {
		httpx.Error(w, http.StatusBadRequest, "qaysi tizimdan kelganini tanlang")
		return
	}
	filter := scopeFilter(scope)
	filter["paymentMethod"] = provider
	filter["paymentStatus"] = models.PayPaid
	filter["status"] = bson.M{"$ne": models.StatusCancelled}
	rng := bson.M{}
	if day, err := parseDay(q.Get("from")); err == nil {
		rng["$gte"] = day
	}
	if day, err := parseDay(q.Get("to")); err == nil {
		// The period end is inclusive — a statement for "1–31 March" means the
		// whole of the 31st.
		rng["$lt"] = day.AddDate(0, 0, 1)
	}
	if len(rng) > 0 {
		filter["createdAt"] = rng
	}
	total, n, err := h.sumField(r.Context(), h.Store.Orders, filter, "$total")
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Whether our own records are evidence or merely bookkeeping for this rail.
	watched := provider == models.ProviderPayme || provider == models.ProviderClick ||
		provider == models.ProviderUzum || provider == models.ProviderAtmos
	httpx.JSON(w, http.StatusOK, map[string]any{
		"gross": total, "count": n, "watched": watched,
	})
}

// AdminCreatePayout records one transfer that arrived.
func (h *Handler) AdminCreatePayout(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		Provider   string `json:"provider"`
		PeriodFrom string `json:"periodFrom"`
		PeriodTo   string `json:"periodTo"`
		Gross      int    `json:"gross"`
		Commission int    `json:"commission"`
		Net        int    `json:"net"`
		ReceivedAt string `json:"receivedAt"`
		Account    string `json:"account"`
		Note       string `json:"note"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	provider := strings.TrimSpace(req.Provider)
	if provider == "" {
		httpx.Error(w, http.StatusBadRequest, "qaysi tizimdan kelganini tanlang")
		return
	}
	if req.Net <= 0 && req.Gross <= 0 {
		httpx.Error(w, http.StatusBadRequest, "summani yozing")
		return
	}
	branch := h.scopeBranch(r, scope)
	if err := h.requireBranchAccess(r, branch); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}

	now := time.Now()
	at := now
	if day, err := parseDay(req.ReceivedAt); err == nil {
		at = day
	}
	if at.After(now) {
		at = now
	}
	from, to := strings.TrimSpace(req.PeriodFrom), strings.TrimSpace(req.PeriodTo)
	if to < from {
		from, to = to, from
	}

	name := provider
	for _, rail := range h.payoutRails(r) {
		if rail.ID == provider {
			name = rail.Name
			break
		}
	}

	p := models.Payout{
		BranchID: branch, Provider: provider, ProviderName: name,
		PeriodFrom: from, PeriodTo: to,
		Gross: req.Gross, Commission: req.Commission, Net: req.Net,
		ReceivedAt: at,
		Account:    clampText(strings.TrimSpace(req.Account), 80),
		Note:       clampText(strings.TrimSpace(req.Note), 200),
		CreatedBy:  h.adminName(r), CreatedAt: now,
	}
	res, err := h.Store.Payouts.InsertOne(r.Context(), p)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	p.ID = oidOf(res.InsertedID)

	// ⚠️ **No safe row, ever.** A transfer lands in a bank account; the office
	// box is not involved, and offering the tick would invite somebody to add
	// money to a safe that never received any.
	h.logAction(r, "payout.create", "payout", p.ID.Hex(), name, formatSum(p.Net))
	httpx.JSON(w, http.StatusCreated, p)
}

// AdminDeletePayout removes one entered by mistake.
func (h *Handler) AdminDeletePayout(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var p models.Payout
	if err := h.Store.Payouts.FindOne(r.Context(), bson.M{"_id": id}).Decode(&p); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	if err := h.requireBranchAccess(r, p.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	if _, err := h.Store.Payouts.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, "payout.delete", "payout", id.Hex(), p.ProviderName,
		formatSum(p.Net))
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
