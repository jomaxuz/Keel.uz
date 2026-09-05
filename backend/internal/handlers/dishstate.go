package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- Where one dish has got to ----
//
// ⚠️ **The pass had one button for a whole ticket, and that was the bug.** A
// table of six was "ready" only when the last thing on it was — so for the
// twenty minutes before that the floor was told nothing, while the first five
// plates sat under the lamp. A cook finishes dishes one at a time; the screens
// downstream have to be able to say which ones.
//
// So two timestamps live on the line (`OrderItem.ReadyAt`, `OrderItem.ServedAt`)
// and this file is the only place that writes them:
//
//	kitchen ticks a dish  → ReadyAt   → it turns green on the floor, the till
//	                                    and the waiter's phone, with "5 daq
//	                                    oldin tayyor" beside it
//	waiter carries it out → ServedAt  → it stops being something to carry
//
// ⚠️ **The order-wide `readyAt` is derived, never pressed.** It is what makes a
// ticket leave the pass and what the waiter's and courier's notification hangs
// off, so it is computed here from the lines: set when every fired live line is
// ready, cleared the moment one stops being. Two places writing it would drift,
// and the direction of the drift is a ticket that never comes back.

// dishRef names one line of an order.
//
// ⚠️ **Two ways to name it, because two kinds of order arrive here.** A till
// check's lines carry a `lineId` — they are edited, split and moved, and their
// position shifts under whoever is looking. A website or Telegram order has no
// such id: its items are written once and never edited, so their position *is*
// their identity. Refusing the second kind would leave every delivery ticket
// with an unusable tick.
type dishRef struct {
	LineID string `json:"lineId"`
	// Position in `items`, used only when there is no line id. A pointer so
	// "not sent" is distinguishable from "the first one".
	Index *int `json:"index"`
}

// find returns the index of the referenced item, or -1.
func (d dishRef) find(items []models.OrderItem) int {
	if d.LineID != "" {
		for i := range items {
			if items[i].LineID == d.LineID {
				return i
			}
		}
		return -1
	}
	if d.Index == nil || *d.Index < 0 || *d.Index >= len(items) {
		return -1
	}
	// ⚠️ An index may not address a line that has one: a check where the
	// waiter has just split a course would otherwise be edited at the wrong
	// row, and the screen would show the tick jumping to another dish.
	if items[*d.Index].LineID != "" {
		return -1
	}
	return *d.Index
}

type kitchenItemRequest struct {
	dishRef
	// ⚠️ Both directions. A cook ticking the wrong dish on a wet screen is
	// ordinary, and a tick that cannot be undone is one nobody dares press —
	// which leaves the room back where it started, waiting for the whole
	// ticket.
	Ready bool `json:"ready"`
}

// StaffKitchenItem marks one dish cooked, or puts it back.
func (h *Handler) StaffKitchenItem(w http.ResponseWriter, r *http.Request) {
	// ⚠️ The same guard as the ticket action, for the same reason: this makes a
	// dish disappear from the pass and tells the floor to come and get it.
	s, ok := h.kitchenStaff(w, r)
	if !ok {
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var req kitchenItemRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// The filter is the permission: this branch, a ticket the kitchen may still
	// act on. An id alone never selects a document here.
	filter := bson.M{
		"_id":      id,
		"branchId": s.BranchID,
		"status":   bson.M{"$in": bson.A{models.StatusConfirmed, models.StatusPreparing}},
		"queuedAt": bson.M{"$ne": nil},
	}
	var o models.Order
	if err := h.Store.Orders.FindOne(r.Context(), filter).Decode(&o); err != nil {
		httpx.Error(w, http.StatusConflict, "bu buyurtma o'zgargan — ro'yxat yangilandi")
		return
	}
	i := req.find(o.Items)
	if i < 0 || !o.Items[i].Live() {
		httpx.Error(w, http.StatusNotFound, "taom topilmadi — ro'yxat yangilandi")
		return
	}
	// A line the waiter has not sent is not the kitchen's yet, and the pass
	// cannot see it — so it cannot be ticked either.
	if o.Check != nil && o.Items[i].FiredAt == nil {
		httpx.Error(w, http.StatusConflict, "bu taom hali oshxonaga yuborilmagan")
		return
	}

	now := time.Now()
	if req.Ready {
		at := now
		o.Items[i].ReadyAt = &at
	} else {
		o.Items[i].ReadyAt = nil
	}

	set := bson.M{"items": o.Items, "updatedAt": now}
	// Ticking anything means the kitchen has started, whatever the ticket said.
	if o.Status == models.StatusConfirmed {
		set["status"] = models.StatusPreparing
	}
	all := allDishesReady(o.Items)
	switch {
	case all && o.ReadyAt == nil:
		set["readyAt"] = now
	case !all && o.ReadyAt != nil:
		// ⚠️ Untick returns the ticket to the pass. Without this the order
		// stays filtered out of the kitchen screen and the dish the cook just
		// took back is invisible to everybody, including them.
		set["readyAt"] = nil
	}
	if _, err := h.Store.Orders.UpdateOne(r.Context(), filter, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ **After the write, always** — the kitchen's work is recorded whether or
	// not anybody can be told, and `notifyStaff` is fire-and-forget for the same
	// reason.
	//
	// ⚠️ **One message, never two.** On an order with a single cookable dish,
	// ticking it makes that dish ready *and* the whole order ready at the same
	// instant — so sending both would put two notifications about one plate on
	// the waiter's phone, which is exactly the habit that teaches people to swipe
	// without reading. The last dish is announced as the order; every dish before
	// it is announced by name.
	switch {
	case !req.Ready:
		// Unticking is a correction inside the kitchen. Nobody is waiting on it,
		// and "that dish is not ready after all" is a message a waiter can do
		// nothing with while carrying plates.
	case all && o.ReadyAt == nil:
		h.notifyReady(r.Context(), id, s.BranchID)
	default:
		h.notifyDishReady(r.Context(), id, s.BranchID, o.Items[i].Name)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok": true, "items": kitchenItems(&o), "allReady": all,
	})
}

// allDishesReady reports whether every dish the kitchen owns has been ticked.
//
// ⚠️ **Unfired lines do not count and voided ones do not either.** A table
// whose mains are still a draft on the waiter's tablet is not a table waiting
// for nothing, and a dish that was sent back was never going to be cooked. An
// order with nothing cookable on it — every line voided — is not "ready": it
// has nothing to be ready.
func allDishesReady(items []models.OrderItem) bool {
	fired := hasFired(items)
	n := 0
	for _, it := range items {
		if !it.Live() || (fired && it.FiredAt == nil) {
			continue
		}
		n++
		if it.ReadyAt == nil {
			return false
		}
	}
	return n > 0
}

// hasFired reports whether this order is a till check whose lines are fired one
// course at a time. A website order has no fired marks at all and every live
// line counts.
func hasFired(items []models.OrderItem) bool {
	for _, it := range items {
		if it.FiredAt != nil {
			return true
		}
	}
	return false
}

// markAllDishesReady stamps every dish the kitchen owns, for the ticket-wide
// button.
//
// ⚠️ **Kept, and now it writes the lines too.** A big order finished in one go
// is a real thing — one press instead of eight on a screen somebody is touching
// with the back of a wrist — but if it left the lines unstamped, the floor
// would see "the order is ready" with no dish on it green, which is the
// contradiction that makes a room stop trusting the colour.
func markAllDishesReady(items []models.OrderItem, at time.Time) {
	fired := hasFired(items)
	for i := range items {
		if !items[i].Live() || (fired && items[i].FiredAt == nil) {
			continue
		}
		if items[i].ReadyAt == nil {
			stamp := at
			items[i].ReadyAt = &stamp
		}
	}
}

type serveRequest struct {
	dishRef
	// ⚠️ Undoable, like the kitchen's tick: a runner marking the wrong line on
	// a moving tray is the common case, and the correction has to be cheaper
	// than asking the guest.
	Served bool `json:"served"`
}

// StaffServeLine records that a dish reached the table.
//
// ⚠️ **The waiter's half, and it is deliberately not the kitchen's.** "Cooked"
// and "carried out" are the two states a runner works between; collapsing them
// would mean a plate under the lamp and a plate in front of a guest look the
// same on every screen — which is how one is carried out twice and another
// never at all.
func (h *Handler) StaffServeLine(w http.ResponseWriter, r *http.Request) {
	// A waiter's permission: this is the floor's own act, not the till's.
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req serveRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// The line id may also come from the path, which is how the floor screen
	// addresses it.
	if req.LineID == "" {
		req.LineID = chi.URLParam(r, "lineId")
	}

	var o models.Order
	if err := h.Store.Orders.FindOne(r.Context(),
		checkFilter(id, s.BranchID)).Decode(&o); err != nil {
		httpx.Error(w, http.StatusNotFound, "chek topilmadi")
		return
	}
	// ⚠️ **A closed check is history.** Editing one would move a number in a
	// shift that has already been counted, and "the guest got it" is not a
	// number anybody goes back to correct.
	if o.Check == nil || o.Check.ClosedAt != nil {
		httpx.Error(w, http.StatusConflict, "chek yopilgan")
		return
	}
	i := req.find(o.Items)
	if i < 0 || !o.Items[i].Live() {
		httpx.Error(w, http.StatusNotFound, "qator topilmadi")
		return
	}
	// ⚠️ **Fired, but not necessarily ready.** A glass poured at the bar never
	// passes a kitchen screen, and refusing to let the waiter mark it served
	// would leave a permanent unserved line on every table that ordered a
	// drink.
	if o.Items[i].FiredAt == nil {
		httpx.Error(w, http.StatusConflict, "bu taom hali oshxonaga yuborilmagan")
		return
	}

	now := time.Now()
	if req.Served {
		at := now
		o.Items[i].ServedAt = &at
	} else {
		o.Items[i].ServedAt = nil
	}
	if _, err := h.Store.Orders.UpdateOne(r.Context(), checkFilter(id, s.BranchID),
		bson.M{"$set": bson.M{"items": o.Items, "updatedAt": now}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, viewCheck(&o, now, s.ID))
}

// servedCount is how many of a check's live fired lines have reached the table.
// Used by the floor screen's badge, which answers "is anything still standing
// at the pass" without opening the check.
func servedCount(items []models.OrderItem) (served, waiting int) {
	for _, it := range items {
		if !it.Live() || it.FiredAt == nil {
			continue
		}
		if it.ServedAt != nil {
			served++
			continue
		}
		if it.ReadyAt != nil {
			waiting++
		}
	}
	return served, waiting
}

// hasOpenShift reports whether this employee is clocked in right now.
//
// ⚠️ **A database that will not answer means "yes".** The question guards a
// till, and the failure this protects against — an evening sold with nobody
// clocked in — costs a payroll correction. Refusing the whole counter because
// one read timed out costs the restaurant its service, with a queue standing
// there and nothing anybody at the counter can do about it. So the doubtful
// case opens the till and the honest case is the only one that closes it.
func (h *Handler) hasOpenShift(ctx context.Context, staffID primitive.ObjectID) bool {
	return shiftAllows(h.openShift(ctx, staffID))
}

// shiftAllows is that rule on its own, so the direction it fails in is written
// down in a test rather than in a sentence above a database call.
func shiftAllows(shift *models.Shift, err error) bool {
	if err != nil {
		// Only "there is no such row" is an answer. Anything else is the
		// database failing to speak, and a till that closes on that takes the
		// restaurant's evening with it.
		return !errors.Is(err, mongo.ErrNoDocuments)
	}
	return shift != nil
}
