package handlers

// Counting the registers a branch runs, and taking one away.
//
// ⚠️ **The cap is what the plan is sold by, so it has to be a fact.** Until
// there was a row per machine, "Standard: 2 kassa" was a sentence in a price
// list and nothing more: every monoblock carried an identical branch token, so
// a restaurant on the cheapest plan could bind ten and nothing anywhere would
// notice. Charging for a limit nobody enforces is worse than not having one —
// the customers who read the price list honestly end up subsidising the ones
// who did not.
//
// ⚠️ **But enforcement is at the door only, never on a machine already
// selling.** Refusing a *new* link is declining to add a register; killing a
// bound one is taking a working till off a restaurant, and the moment for that
// would inevitably be a Friday evening. So the cap is checked when a link is
// issued and never again — a till that is already paired keeps working for as
// long as its token lives, whatever happens to the plan afterwards.
//
// The refusal carries the count, the cap and the rung that lifts it, because
// "limit reached" alone leaves the manager with one action: phoning us to ask
// the question the message could have answered.

import (
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// tillDeviceFilter is every live device of a branch.
//
// ⚠️ **Version-matched, so rotating the branch key frees the slots too.** A
// manager who rotates after a theft has, in the same act, told us those
// machines are gone; leaving their rows counting against the cap would mean the
// replacement monoblock is refused by a limit filled with stolen hardware.
func tillDeviceFilter(branchID primitive.ObjectID, version int) bson.M {
	return bson.M{"branchId": branchID, "version": version}
}

// countTillDevices is how many registers this branch currently has bound.
func (h *Handler) countTillDevices(r *http.Request, branch models.Branch) (int, error) {
	n, err := h.Store.TillDevices.CountDocuments(
		r.Context(), tillDeviceFilter(branch.ID, branch.TillVersion))
	return int(n), err
}

// tillCapReached reports whether another register would exceed the plan, and
// what to tell whoever asked.
//
// ⚠️ **No subscription means no cap.** Every install that predates the till
// subscription, and every one we host without a counter, must keep binding
// screens exactly as before — the same rule the module gate follows, and for
// the same reason: a limit that switches itself on for customers who were never
// sold one empties their building on a deploy.
func (h *Handler) tillCapReached(r *http.Request, branch models.Branch) (bool, map[string]any) {
	s := h.subscription(r.Context())
	n, err := h.countTillDevices(r, branch)
	if err != nil {
		// ⚠️ **A failed count does not refuse.** Mongo being briefly
		// unreachable is not the restaurant exceeding its plan, and the visible
		// result of getting this backwards is a till that cannot be set up
		// during the incident somebody is already dealing with.
		return false, nil
	}
	if !tillCapExceeded(s, n) {
		return false, nil
	}
	return true, map[string]any{
		"error":     "tarifingizdagi kassalar soni to'lgan",
		"registers": n,
		"limit":     s.Registers,
		"plan":      nextPlanAbove(s, s.Registers),
	}
}

// tillCapExceeded decides whether one more register is one too many.
//
// Pure, so the two answers that must never be got backwards can be argued with
// in a test rather than in a restaurant: an install with no subscription and a
// plan with no cap both mean **no limit at all**.
func tillCapExceeded(s *models.Subscription, count int) bool {
	if s == nil || !s.Enabled || s.Registers <= 0 {
		return false
	}
	return count >= s.Registers
}

// nextPlanAbove names the cheapest rung that allows more registers than the
// current cap, for the upgrade button. Empty when there is none — the customer
// is already at the top, and a button offering nothing is worse than no button.
func nextPlanAbove(s *models.Subscription, limit int) string {
	if s == nil {
		return ""
	}
	for _, p := range s.Plans {
		if p.Registers == 0 || p.Registers > limit {
			return p.ID
		}
	}
	return ""
}

// AdminTillDevices lists the screens bound to a branch.
func (h *Handler) AdminTillDevices(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.requireBranchAccess(r, id); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	branch, err := h.branchByID(r, id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "filial topilmadi")
		return
	}
	cur, err := h.Store.TillDevices.Find(r.Context(),
		tillDeviceFilter(id, branch.TillVersion),
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ An empty slice, never nil: the panel reads `.length` off this to draw
	// "2 / 2 kassa", and Go marshals a nil slice as `null`.
	rows := []models.TillDevice{}
	if err := cur.All(r.Context(), &rows); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	limit := 0
	if s := h.subscription(r.Context()); s != nil && s.Enabled {
		limit = s.Registers
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"devices": rows,
		// 0 means no cap, and the panel says "cheksiz" rather than "0 kassa".
		"limit": limit,
	})
}

// AdminRemoveTillDevice unbinds one screen, freeing its slot.
//
// ⚠️ **One machine, unlike rotating the branch key.** Rotation exists for a
// monoblock that walked out of the building and kills every till at once,
// because before this there was nothing finer. This is the ordinary case —
// a machine replaced, a tablet retired, a link issued by mistake — and doing it
// with the blunt tool means a manager retiring one tablet takes the whole
// restaurant offline and then walks to every counter with a new link.
func (h *Handler) AdminRemoveTillDevice(w http.ResponseWriter, r *http.Request) {
	branchID, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	devID, err := objectID(chi.URLParam(r, "deviceId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	// ⚠️ The branch is in the filter, not merely checked above: `_id` alone
	// must never select a document — the rule the whole security section of
	// CLAUDE.md is built on. A manager typing another branch's device id gets
	// nothing rather than unbinding somebody else's counter.
	var dev models.TillDevice
	err = h.Store.TillDevices.FindOneAndDelete(r.Context(), bson.M{
		"_id": devID, "branchId": branchID,
	}).Decode(&dev)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "qurilma topilmadi")
		return
	}
	h.logAction(r, ActSettingsUpdate, "branch", branchID.Hex(), dev.Name,
		"kassa qurilmasi uzildi")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// StaffTillUnbind is the counter's own way out: the screen standing in the
// restaurant retires itself.
//
// ⚠️ **Not for the person running the till**, and that is the whole reason it
// is a separate endpoint rather than part of the lock button. A cashier or a
// waiter who pressed this would take the machine out of service in the middle
// of a shift, and getting it back means somebody with a panel login fetching a
// fresh link — during service, from wherever they are. The people who should be
// able to do it are the ones who could already stop a sale on their own
// judgement, so it is gated on the permission that already draws that line.
//
// ⚠️ It removes the device row as well as the token, so the register slot is
// returned. A screen retired without freeing its slot would leave a restaurant
// at its cap with a machine nobody can find.
func (h *Handler) StaffTillUnbind(w http.ResponseWriter, r *http.Request) {
	// ⚠️ The permission is checked against the person who unlocked the screen,
	// not against the device token — the device is the thing being removed and
	// cannot authorise its own removal.
	s, ok := h.tillStaff(w, r, models.PermVoid)
	if !ok {
		return
	}
	claims := middleware.ClaimsFrom(r.Context())
	if claims == nil || claims.Dev == "" {
		// A screen signed in with a staff login rather than bound as a device.
		// Nothing to unbind: the app signs out on its side, which is a
		// different and much cheaper act.
		httpx.JSON(w, http.StatusOK, map[string]any{"unbound": false})
		return
	}
	devID, err := objectID(claims.Dev)
	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"unbound": false})
		return
	}
	var dev models.TillDevice
	if err := h.Store.TillDevices.FindOneAndDelete(r.Context(), bson.M{
		"_id": devID, "branchId": s.BranchID,
	}).Decode(&dev); err != nil {
		// Already gone — removed from the panel a moment ago, or this token
		// predates the registry. The screen still logs itself out, because the
		// person pressed the button and the answer must not be "nothing
		// happened".
		httpx.JSON(w, http.StatusOK, map[string]any{"unbound": false})
		return
	}
	h.logAction(r, ActSettingsUpdate, "branch", s.BranchID.Hex(), dev.Name,
		"kassa ekranidan chiqildi")
	httpx.JSON(w, http.StatusOK, map[string]any{"unbound": true})
}

// tillHost is the computer name the till application sends about itself.
//
// ⚠️ **Trimmed and capped, because it is a string from a machine we do not
// administer.** It is drawn in the panel beside a button that unbinds a
// register, and a hostname long enough to push that button off the row is a
// hostname that decides something.
//
// ⚠️ Empty when the screen was paired from a browser rather than from the
// Windows application — a distinction worth keeping rather than filling in:
// a row with no machine behind it is usually the abandoned link.
func tillHost(r *http.Request) string {
	name := strings.TrimSpace(r.Header.Get("X-Till-Host"))
	// One line, always: a header with a newline in it would be drawn as two
	// rows in the panel and read as two machines.
	name = strings.Map(func(c rune) rune {
		if c < ' ' {
			return -1
		}
		return c
	}, name)
	if len(name) > 60 {
		name = name[:60]
	}
	return name
}

// touchTillDevice records that a bound screen was seen, and from where.
//
// ⚠️ **Throttled to once an hour, and it never fails a request.** A till polls
// several times a minute; writing on each would be a write per poll per machine
// for fields whose only job is telling a manager which row is the abandoned one
// and which machine each of the others is. An hour is far finer than either
// question needs.
//
// ⚠️ **The machine is written here rather than only when the link is issued**,
// because the two are often not the same computer at all: a manager who pairs a
// till by scanning the panel's QR code with the monoblock is the good case, and
// a link issued on a laptop and opened later on the counter is the ordinary
// one. Whoever is actually calling is the answer to "which machine is this",
// and this is the only place that sees them.
func (h *Handler) touchTillDevice(r *http.Request, devID primitive.ObjectID) {
	now := time.Now()
	set := bson.M{"lastSeenAt": now}
	if ip := clientIP(r); ip != "" {
		set["ip"] = ip
	}
	if host := tillHost(r); host != "" {
		// ⚠️ Only when it is sent. A till that has been upgraded to the Windows
		// application and a browser tab on the same machine take turns calling
		// these endpoints, and blanking the name on every browser call would
		// leave the column empty exactly as often as it was filled.
		set["host"] = host
	}
	_, _ = h.Store.TillDevices.UpdateOne(r.Context(), bson.M{
		"_id":        devID,
		"lastSeenAt": bson.M{"$not": bson.M{"$gt": now.Add(-time.Hour)}},
	}, bson.M{"$set": set})
}

// deviceName cleans what the panel typed, or names the machine by its number.
func deviceName(in string, n int) string {
	name := strings.TrimSpace(in)
	if name == "" {
		// ⚠️ Numbered rather than left blank: a list of unnamed rows is a list
		// nobody can act on, and the one moment somebody reads it is when a
		// machine has gone missing.
		return "Kassa " + itoa(n+1)
	}
	if len(name) > 40 {
		name = name[:40]
	}
	return name
}
