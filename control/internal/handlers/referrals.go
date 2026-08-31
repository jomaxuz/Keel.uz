package handlers

// ---- Who sends us customers, and what we owe them ----
//
// ⚠️ **This exists so an arrangement can be honoured, not so it can be
// tracked.** A firm that sells fiscal registers walks into twenty kitchens a
// week and is not competing with us; the only thing that makes them send one
// our way is being paid for the last one. A referral nobody can count is a
// referral nobody can pay for, and a partner who is not paid in the second
// month stops sending anybody in the third.
//
// ⚠️ **Money, so owner and admin only.** An agent must not read what another
// channel earns — the same boundary the invoices already draw, for the same
// reason: knowing what everyone else is paid is not part of selling.
//
// ⚠️ **The commission is computed, never stored.** Storing it would mean a
// number that agrees with the invoices on the day it is written and drifts from
// them afterwards — a payment voided, an invoice corrected — and the drift is
// invisible because both figures look equally official. It is arithmetic over
// the invoices, done on every read, the way `Collected()` already is.

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"keel-control/internal/httpx"
	"keel-control/internal/models"
)

// referrerView is one channel with what it has actually produced.
type referrerView struct {
	models.Referrer
	// Customers attributed to them, and how many of those ever started paying.
	// ⚠️ Both, because the gap between them is the whole quality of a channel:
	// a partner who sends twenty trials and no subscribers is sending the
	// wrong twenty.
	Tenants int `json:"tenants"`
	Paying  int `json:"paying"`
	// What those customers paid us inside the commission window, and the share
	// of it this referrer has earned.
	Collected  int `json:"collected"`
	Commission int `json:"commission"`
}

// referrerTenantView is one customer's line under a referrer.
type referrerTenantView struct {
	ID   primitive.ObjectID `json:"id"`
	Slug string             `json:"slug"`
	Name string             `json:"name"`
	// nil while they are still evaluating — a trial earns nobody anything.
	SubscribedAt *time.Time `json:"subscribedAt,omitempty"`
	Status       string     `json:"status"`
	// When their commission window closes. Zero means it does not.
	WindowEndsAt *time.Time `json:"windowEndsAt,omitempty"`
	Collected    int        `json:"collected"`
	Commission   int        `json:"commission"`
}

// ListReferrers answers the channel list with its figures.
func (h *Handler) ListReferrers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.referrers(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]referrerView, 0, len(rows))
	for _, rf := range rows {
		view, _, err := h.referrerFigures(r.Context(), rf)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, view)
	}
	// Most owed first: the list is read to decide who to pay this week.
	sort.SliceStable(out, func(i, j int) bool { return out[i].Commission > out[j].Commission })
	httpx.JSON(w, http.StatusOK, map[string]any{"referrers": out})
}

// GetReferrer is one channel with the customers behind its number.
//
// ⚠️ **The customers are named, not counted.** "You are owed 1 240 000" is a
// figure to argue with; "these four restaurants, this is what each paid" is a
// figure to agree on, and the conversation this screen exists for is a
// conversation about money with somebody outside the company.
func (h *Handler) GetReferrer(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var rf models.Referrer
	if err := h.Store.Referrers.FindOne(r.Context(), bson.M{"_id": id}).Decode(&rf); err != nil {
		httpx.Error(w, http.StatusNotFound, "hamkor topilmadi")
		return
	}
	view, tenants, err := h.referrerFigures(r.Context(), rf)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"referrer": view,
		"tenants":  tenants,
	})
}

type referrerPayload struct {
	Name    string `json:"name"`
	Code    string `json:"code"`
	Contact string `json:"contact"`
	Note    string `json:"note"`
	Percent int    `json:"percent"`
	Months  int    `json:"months"`
	// A pointer so "not sent" keeps what is stored — the same guard the tenant
	// form needs: an older tab saving a note must not switch a channel off.
	IsActive *bool `json:"isActive"`
}

// CreateReferrer adds a channel.
func (h *Handler) CreateReferrer(w http.ResponseWriter, r *http.Request) {
	var req referrerPayload
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	rf, err := h.referrerFrom(req, models.Referrer{})
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	rf.CreatedAt = time.Now()
	rf.UpdatedAt = rf.CreatedAt
	res, err := h.Store.Referrers.InsertOne(r.Context(), rf)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			httpx.Error(w, http.StatusConflict, "bu kod band")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rf.ID, _ = res.InsertedID.(primitive.ObjectID)
	h.logConsole(r.Context(), h.actorOrNil(r), "referrer.create", rf.Name, rf.Code)
	httpx.JSON(w, http.StatusCreated, rf)
}

// UpdateReferrer edits one.
func (h *Handler) UpdateReferrer(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var stored models.Referrer
	if err := h.Store.Referrers.FindOne(r.Context(), bson.M{"_id": id}).Decode(&stored); err != nil {
		httpx.Error(w, http.StatusNotFound, "hamkor topilmadi")
		return
	}
	var req referrerPayload
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	next, err := h.referrerFrom(req, stored)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := bson.M{
		"name":      next.Name,
		"code":      next.Code,
		"contact":   next.Contact,
		"note":      next.Note,
		"percent":   next.Percent,
		"months":    next.Months,
		"isActive":  next.IsActive,
		"updatedAt": time.Now(),
	}
	if _, err := h.Store.Referrers.UpdateByID(r.Context(), id, bson.M{"$set": set}); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			httpx.Error(w, http.StatusConflict, "bu kod band")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ The code on the customers already attributed is **not** rewritten. It
	// is a frozen copy of what was on the leaflet that brought them, and the
	// link between the two is the id. Rewriting it would quietly restate what
	// somebody agreed to last spring.
	h.logConsole(r.Context(), h.actorOrNil(r), "referrer.update", next.Name, next.Code)
	next.ID = id
	httpx.JSON(w, http.StatusOK, next)
}

func errBadRequest(msg string) error {
	return &httpError{code: http.StatusBadRequest, msg: msg}
}

// actorOrNil is the signed-in console user, or nil. ⚠️ The log takes a nil and
// writes nothing rather than an unsigned row: a journal entry with no author is
// the shape of an entry somebody would like to have been able to write.
func (h *Handler) actorOrNil(r *http.Request) *models.User {
	u, err := h.actor(r)
	if err != nil {
		return nil
	}
	return u
}

// referrerFrom validates a payload onto a stored row.
func (h *Handler) referrerFrom(req referrerPayload, stored models.Referrer) (models.Referrer, error) {
	out := stored
	out.Name = strings.TrimSpace(req.Name)
	if out.Name == "" {
		return out, errBadRequest("hamkor nomini yozing")
	}
	code := models.NormalizeReferrerCode(req.Code)
	if !models.ValidReferrerCode(code) {
		return out, errBadRequest(
			"kod faqat kichik lotin harflari, raqam va tire bo'lishi mumkin (2–24 belgi)")
	}
	out.Code = code
	out.Contact = strings.TrimSpace(req.Contact)
	out.Note = strings.TrimSpace(req.Note)
	// ⚠️ Capped rather than trusted. A typo of 500 in a percentage field is a
	// commission larger than the revenue, and it would be discovered by paying
	// it. Fifty is already a partnership nobody would sign twice.
	if req.Percent < 0 || req.Percent > 50 {
		return out, errBadRequest("foiz 0 dan 50 gacha bo'lishi kerak")
	}
	out.Percent = req.Percent
	if req.Months < 0 || req.Months > 60 {
		return out, errBadRequest("muddat 0 dan 60 oygacha bo'lishi kerak")
	}
	out.Months = req.Months
	if req.IsActive != nil {
		out.IsActive = *req.IsActive
	} else if stored.ID.IsZero() {
		// A new channel is on: nobody creates one in order to leave it off.
		out.IsActive = true
	}
	return out, nil
}

func (h *Handler) referrers(ctx context.Context) ([]models.Referrer, error) {
	cur, err := h.Store.Referrers.Find(ctx, bson.M{},
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var rows []models.Referrer
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// ReferrerByCode resolves a code from a link. Inactive channels resolve too:
// the leaflets are already printed, and a visitor arriving on a dead link
// should still see the site.
func (h *Handler) ReferrerByCode(ctx context.Context, code string) (models.Referrer, bool) {
	code = models.NormalizeReferrerCode(code)
	if !models.ValidReferrerCode(code) {
		return models.Referrer{}, false
	}
	var rf models.Referrer
	if err := h.Store.Referrers.FindOne(ctx, bson.M{"code": code}).Decode(&rf); err != nil {
		return models.Referrer{}, false
	}
	return rf, true
}

// referrerFigures totals one channel from the invoices.
//
// ⚠️ **Payments, not invoices, and each payment judged by its own date.** A
// customer who settles March in June is inside the window for March's money and
// outside it for June's, and the difference is real money owed to somebody
// outside the company. Summing an invoice by its period would pay commission on
// a bill that has not been collected; summing by the invoice's date would pay
// it on a period that had already ended.
func (h *Handler) referrerFigures(
	ctx context.Context, rf models.Referrer,
) (referrerView, []referrerTenantView, error) {
	view := referrerView{Referrer: rf}

	cur, err := h.Store.Tenants.Find(ctx, bson.M{"referrerId": rf.ID},
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return view, nil, err
	}
	var tenants []models.Tenant
	if err := cur.All(ctx, &tenants); err != nil {
		return view, nil, err
	}

	lines := make([]referrerTenantView, 0, len(tenants))
	for _, t := range tenants {
		line := referrerTenantView{
			ID:           t.ID,
			Slug:         t.Slug,
			Name:         t.Name,
			SubscribedAt: t.SubscribedAt,
			Status:       t.Status,
		}
		if end := rf.CommissionWindow(t.SubscribedAt); !end.IsZero() {
			e := end
			line.WindowEndsAt = &e
		}
		collected, err := h.collectedFor(ctx, t.ID, rf.CommissionWindow(t.SubscribedAt))
		if err != nil {
			return view, nil, err
		}
		line.Collected = collected
		line.Commission = rf.Commission(collected)

		view.Tenants++
		if t.SubscribedAt != nil {
			view.Paying++
		}
		view.Collected += collected
		view.Commission += line.Commission
		lines = append(lines, line)
	}
	return view, lines, nil
}

// collectedFor sums the payments received from one tenant up to `until`.
// A zero `until` means everything ever received.
func (h *Handler) collectedFor(
	ctx context.Context, tenantID primitive.ObjectID, until time.Time,
) (int, error) {
	cur, err := h.Store.Invoices.Find(ctx, bson.M{"tenantId": tenantID})
	if err != nil {
		return 0, err
	}
	var invoices []models.Invoice
	if err := cur.All(ctx, &invoices); err != nil {
		return 0, err
	}
	total := 0
	for _, inv := range invoices {
		// ⚠️ A voided invoice's payments are still money that changed hands;
		// what a void says is that the bill was wrong, and a refund is recorded
		// as its own negative payment. Reading the status here would drop
		// collected cash from the total on the day somebody corrects a bill.
		for _, p := range inv.Paid {
			if !until.IsZero() && p.At.After(until) {
				continue
			}
			total += p.Amount
		}
	}
	return total, nil
}
