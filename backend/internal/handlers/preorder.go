package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Ordering ahead of time ("predzakaz").
//
// A guest picks a time instead of being cooked for the moment they tap, and the
// restaurant is told about it twice: once when it is placed — somebody has to
// buy the meat — and again shortly before it is due, which is when the kitchen
// actually starts.
//
// The second bell is the whole feature, and it is deliberately **not** a
// scheduler, a job queue or a new order status. A scheduled order is written
// with its `queuedAt` set into the future: "when this becomes the kitchen's
// problem" already meant exactly that, and every screen that matters — the
// pass, the chime, the waiting counts — reads that one timestamp. So the only
// change those screens needed was to stop treating a future timestamp as past.
//
// The cost of the alternative is worth naming: a background sweeper flipping
// orders at their due time is a second writer, it needs a lock, it stops when
// the container restarts, and a restaurant only finds out it stopped when a
// pre-order is never cooked.

// preorderSettings fills in the defaults for a branch that has never had this
// section opened. `Enabled` is deliberately left alone — the zero value is off,
// which is what every install that predates the feature already does.
func preorderSettings(p models.PreorderSettings) models.PreorderSettings {
	if p.LeadMinutes <= 0 {
		// An hour: long enough for a kitchen to start something slow, short
		// enough that the ticket is still about the next thing to cook.
		p.LeadMinutes = 60
	}
	if p.MinMinutes <= 0 {
		p.MinMinutes = 60
	}
	if p.MaxDays <= 0 {
		p.MaxDays = 7
	}
	if p.SlotMinutes <= 0 {
		p.SlotMinutes = 30
	}
	return p
}

// Bounds, so a mistyped settings form cannot produce a restaurant nobody can
// order from: a lead longer than the notice would make every slot the guest is
// allowed to pick one the kitchen is already late for.
const (
	maxPreorderLeadMinutes = 24 * 60
	maxPreorderDays        = 60
)

// clampPreorder narrows what the settings form sent to something a kitchen can
// actually run on. Called on save rather than on read: an owner who typed 5000
// minutes should see the number that was kept.
func clampPreorder(p models.PreorderSettings) models.PreorderSettings {
	p = preorderSettings(p)
	if p.LeadMinutes > maxPreorderLeadMinutes {
		p.LeadMinutes = maxPreorderLeadMinutes
	}
	if p.MinMinutes > maxPreorderLeadMinutes {
		p.MinMinutes = maxPreorderLeadMinutes
	}
	if p.MaxDays > maxPreorderDays {
		p.MaxDays = maxPreorderDays
	}
	if p.SlotMinutes > 240 {
		p.SlotMinutes = 240
	}
	return p
}

var errPreorderOff = errors.New("bu filial oldindan buyurtma qabul qilmaydi")

// parsePreorderTime reads what the client sent. RFC 3339 with an offset, which
// is what `Date.toISOString()` produces — the browser's clock names the instant
// and the server never has to guess which timezone a bare "19:00" was written
// in. Returned in local time, because every rule below is about the wall clock
// in the restaurant.
func parsePreorderTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errors.New("vaqt ko'rsatilmagan")
	}
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, errors.New("vaqt formati noto'g'ri")
	}
	return at.In(time.Local).Truncate(time.Minute), nil
}

// resolvePreorder validates a requested time against the branch that will cook
// it, and answers with the instant to store.
//
// ⚠️ **An operator is exempt from the timing rules, and only from those.** The
// same exemption the panel's own bookings have, for the same reason: "in twenty
// minutes" and "for the wedding in six weeks" are both normal things to say on
// the phone, and a rule written for a web form must not make the restaurant
// unable to take an order it is perfectly willing to cook. What the operator is
// *not* exempt from is the branch having the feature switched on at all — that
// is the owner's decision about their own kitchen, not a form validation.
func resolvePreorder(
	branch *models.Branch, raw string, byOperator bool, now time.Time,
) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	p := preorderSettings(branch.Preorder)
	if !p.Enabled {
		return nil, errPreorderOff
	}
	at, err := parsePreorderTime(raw)
	if err != nil {
		return nil, err
	}
	now = now.In(time.Local)
	if byOperator {
		// One rule still applies to everybody: the past is not a time you can
		// cook for, and an order dated backwards would be due the instant it
		// was written — the exact behaviour a pre-order exists to avoid.
		if at.Before(now) {
			return nil, errors.New("o'tgan vaqtga buyurtma berib bo'lmaydi")
		}
		return &at, nil
	}

	earliest := now.Add(time.Duration(p.MinMinutes) * time.Minute)
	if at.Before(earliest) {
		return nil, fmt.Errorf(
			"oldindan buyurtma kamida %d daqiqa oldin beriladi", p.MinMinutes)
	}
	// Midnight after the last allowed day: "3 days ahead" means the whole of
	// the third day is open, not the same hour on it.
	last := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).
		AddDate(0, 0, p.MaxDays+1)
	if !at.Before(last) {
		return nil, fmt.Errorf("eng ko'pi bilan %d kun oldin buyurtma berish mumkin", p.MaxDays)
	}
	// The restaurant has to be open when the guest wants it. Checked against the
	// **branch's** hours rather than the company's: with several branches those
	// are genuinely different, and the guest picked a time from one of them.
	if len(branch.WorkingHours) > 0 && !isOpenNow(branch.WorkingHours, at) {
		return nil, errors.New("bu vaqtda restoran yopiq — boshqa vaqtni tanlang")
	}
	return &at, nil
}

// preorderQueueAt is the moment a scheduled order stops being a note in the
// database and becomes the kitchen's: its requested time, minus the lead the
// branch asked for.
//
// Never later than the scheduled time and never earlier than the order itself —
// a lead longer than the notice (an operator taking one for "in ten minutes")
// would otherwise produce a queue time in the past, and the order would look
// like it had been waiting since before it existed.
// preorderQueueOf answers, for a stored order, when the kitchen should be told
// about it. `false` means "not a pre-order" — the caller keeps its own answer.
//
// The lead is read from the branch now rather than frozen onto the order: an
// owner who shortens it while a pre-order is still waiting means it for that
// order too, which is the only reading that matches what they just typed.
func (h *Handler) preorderQueueOf(
	r *http.Request, orderID primitive.ObjectID, now time.Time,
) (time.Time, bool) {
	var o models.Order
	err := h.Store.Orders.FindOne(r.Context(), bson.M{"_id": orderID}).Decode(&o)
	if err != nil || o.ScheduledAt == nil {
		return time.Time{}, false
	}
	branch, err := h.branchByID(r, o.BranchID)
	if err != nil {
		// The branch is gone or unreadable. Queue it now: an order the kitchen
		// sees early is a nuisance, one it never sees is a guest waiting for
		// food nobody cooked.
		return now, true
	}
	return preorderQueueAt(branch, *o.ScheduledAt, now), true
}

func preorderQueueAt(branch *models.Branch, scheduled, placed time.Time) time.Time {
	p := preorderSettings(branch.Preorder)
	at := scheduled.Add(-time.Duration(p.LeadMinutes) * time.Minute)
	if at.Before(placed) {
		return placed
	}
	return at
}
