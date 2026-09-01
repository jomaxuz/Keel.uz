package handlers

// What this restaurant is entitled to, and what happens when it runs out.
//
// The document is written by the console (see models.Subscription). This file
// is the reading side and the two things built on it: the gate that refuses a
// module nobody bought, and the countdown the counter's screens draw.

import (
	"context"
	"net/http"
	"sync"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// The subscription is read on nearly every request that draws a gated screen,
// and it changes when a human at Keel presses a button — minutes apart at the
// very fastest. Cached for two minutes, the same window and for the same reason
// as the unmapped-dish count: `/admin/alerts` runs every 15 seconds in every
// open tab.
const subscriptionTTL = 2 * time.Minute

var (
	subMu     sync.Mutex
	subCache  *models.Subscription
	subCached time.Time
)

// subscription reads the entitlements, or nil when there is no till sold.
//
// ⚠️ **A read error returns the cached answer rather than nothing.** Mongo
// being briefly unreachable is not the customer's subscription lapsing, and
// treating it as one would close the stock module in the middle of a stocktake.
// With nothing cached it returns nil, which is "not sold" — the safe direction
// on a cold start, and the one that cannot invent a paid module.
func (h *Handler) subscription(ctx context.Context) *models.Subscription {
	subMu.Lock()
	defer subMu.Unlock()
	if subCache != nil && time.Since(subCached) < subscriptionTTL {
		return subCache
	}
	var s models.Subscription
	err := h.Store.DB.Collection("subscription").
		FindOne(ctx, bson.M{"_id": models.SubscriptionID}).Decode(&s)
	if err != nil {
		// A missing document is a real answer and worth caching: an install
		// with no till would otherwise ask on every request forever.
		if err == mongo.ErrNoDocuments {
			subCache, subCached = nil, time.Now()
			return nil
		}
		return subCache
	}
	subCache, subCached = &s, time.Now()
	return &s
}

// requireModule refuses a screen the customer has not bought.
//
// ⚠️ **A filter at the handler, not a hidden button.** The panel does hide what
// it cannot use — a screen whose every control answers "no permission" teaches
// an owner their tools are broken — but hiding is courtesy and this is the
// rule. The same shape as `clampToAdmin` and `tenantScope`: the boundary is one
// call at the top of the handler, so the next person who adds a count, an
// aggregate or an export beside the list does not open a way around it.
//
// ⚠️ **402, not 403.** "You have not bought this" and "you are not allowed
// this" send the owner to two different people — us, and their own manager —
// and an owner told the second thing about the first will go looking for a
// permission that does not exist. The body carries the module and the cheapest
// rung that includes it, because the panel draws an upgrade button out of it.
func (h *Handler) requireModule(w http.ResponseWriter, r *http.Request, mod string) bool {
	if h.subscription(r.Context()).Has(mod) {
		return true
	}
	httpx.JSON(w, http.StatusPaymentRequired, map[string]any{
		"error":  httpx.T(w, "bu bo'lim tarifingizga kirmaydi"),
		"module": mod,
		"plan":   h.cheapestPlanWith(r.Context(), mod),
	})
	return false
}

// cheapestPlanWith names the lowest rung that includes a module, for the
// upgrade button. Empty when nothing does — which is a price list we shipped
// wrong, and an empty string draws a button with no destination rather than a
// button that lies about one.
func (h *Handler) cheapestPlanWith(ctx context.Context, mod string) string {
	s := h.subscription(ctx)
	if s == nil {
		return ""
	}
	for _, p := range s.Plans {
		for _, m := range p.Modules {
			if m == mod {
				return p.ID
			}
		}
	}
	return ""
}

// ---- The countdown ----

// Warning thresholds, in whole days remaining.
//
// ⚠️ **Three, and no more.** A banner that appears a month out is a banner
// people stop seeing by the time it matters, and the counter's screens have one
// corner to spend. A week is enough to arrange a payment, three days is enough
// to chase it, one day is the last honest warning before a panel closes.
var subscriptionWarnAt = []int{7, 3, 1}

// SubscriptionNotice is what a screen draws in its corner. Nil when there is
// nothing to say — the overwhelmingly common case, and the one that must cost
// nothing to render.
type SubscriptionNotice struct {
	// Whole days left; 0 on the last day, negative once it has passed.
	Days int `json:"days"`
	// "warn" a week out, "urgent" at three days and under, "expired" after.
	//
	// ⚠️ Computed here rather than from `days` at each screen: the counter, the
	// floor tablet, the lock screen and the panel all draw this, and four
	// copies of the same comparison is four chances for one of them to be a day
	// out of step with the others.
	Level string `json:"level"`
	// The date itself, "YYYY-MM-DD", already in local time.
	//
	// ⚠️ **A string built here, not a timestamp for the browser to slice.**
	// `time.Time` marshals as UTC, so `paidUntil.slice(0,10)` in Tashkent
	// returns the previous day — the trap this codebase records twice already.
	Until string `json:"until"`
}

const (
	noticeWarn    = "warn"
	noticeUrgent  = "urgent"
	noticeExpired = "expired"
)

// subscriptionNotice is the countdown for a screen, or nil for silence.
func (h *Handler) subscriptionNotice(ctx context.Context) *SubscriptionNotice {
	s := h.subscription(ctx)
	if s == nil || !s.Enabled {
		return nil
	}
	days, ok := s.DaysLeft(time.Now())
	if !ok {
		return nil
	}
	level := ""
	switch {
	case days < 0:
		level = noticeExpired
	case days <= 3:
		level = noticeUrgent
	case days <= 7:
		level = noticeWarn
	}
	if level == "" {
		return nil
	}
	return &SubscriptionNotice{
		Days:  days,
		Level: level,
		Until: s.PaidUntil.In(time.Local).Format("2006-01-02"),
	}
}

// AdminSubscription is what the panel reads to draw its plan section and its
// upgrade buttons.
//
// ⚠️ **Not owner-only.** A manager who runs into a closed module needs to know
// it is a plan and not a bug they should keep pressing — and the response
// carries no price the owner does not already see on their invoice. What the
// manager cannot do is change it, and nothing here changes it.
func (h *Handler) AdminSubscription(w http.ResponseWriter, r *http.Request) {
	s := h.subscription(r.Context())
	if s == nil {
		// ⚠️ A shaped answer rather than 404. The panel asks this on every
		// load, and an error status for the ordinary case — a restaurant with
		// no till — is an error in a console nobody should be reading.
		httpx.JSON(w, http.StatusOK, map[string]any{
			"enabled": false,
			"modules": []string{},
			"plans":   []models.SubscriptionPlan{},
		})
		return
	}
	if s.Modules == nil {
		s.Modules = []string{}
	}
	if s.Plans == nil {
		s.Plans = []models.SubscriptionPlan{}
	}
	if s.Addons == nil {
		s.Addons = []string{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"enabled":   s.Enabled,
		"plan":      s.Plan,
		"modules":   s.Modules,
		"addons":    s.Addons,
		"registers": s.Registers,
		"branches":  s.Branches,
		"monthly":   s.Monthly,
		// ⚠️ **A string, not the timestamp.** `time.Time` marshals as UTC, so
		// `paidUntil.slice(0,10)` in Tashkent returns the previous day — the
		// trap this codebase has written down twice. The notice already sends
		// its date this way; the account screen needs it even when there is
		// nothing to warn about, which is most of the month.
		"paidUntil": paidUntilString(s),
		"notice":    h.subscriptionNotice(r.Context()),
		"plans":     s.Plans,
	})
}

// paidUntilString is the paid-through date as the panel should print it, or ""
// when there is none to print.
func paidUntilString(s *models.Subscription) string {
	if s == nil || s.PaidUntil == nil || s.PaidUntil.IsZero() {
		return ""
	}
	return s.PaidUntil.In(time.Local).Format("2006-01-02")
}
