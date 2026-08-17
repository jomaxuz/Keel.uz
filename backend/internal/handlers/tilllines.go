package handlers

import (
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
)

// Editing an open check: adding dishes, sending them to the kitchen, taking
// them off again.
//
// The one idea worth stating up front is that **adding a dish and cooking it
// are two events, not one**. Everywhere else in this codebase an order arrives
// complete and goes straight to the pass. At a table the waiter types the
// starters, walks away, comes back for the mains twenty minutes later, and the
// kitchen must see each course when the room is ready for it and not before.
// That is why lines carry their own FiredAt and why the two verbs below are
// separate endpoints rather than one "save".

// lineID mints a stable identity for one line of a check.
//
// Short on purpose: it travels in a URL that a cashier's tablet builds on a bad
// connection, and it needs to be unique within a single check, not the
// universe. Collision inside one table's dozen lines is not a real risk.
func lineID() string {
	b := make([]byte, 6)
	if _, err := crand.Read(b); err != nil {
		panic("line id: no randomness: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// ---- Adding dishes ----

type addLinesRequest struct {
	Items []models.OrderItem `json:"items" validate:"required,min=1,dive"`
}

// StaffAddCheckLines puts dishes on an open check.
//
// The lines land **unfired**: typing is not ordering. The waiter reads the
// table back, corrects what they misheard, and only then sends it — which is
// the whole reason a till is faster than shouting through a hatch.
func (h *Handler) StaffAddCheckLines(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok || !requireOpen(w, o) {
		return
	}
	var req addLinesRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Priced against the live menu by the same code the website and the call
	// centre run — see menuLines. The tablet says which dish, never what it costs.
	lines, _, brandID, status, err := h.menuLines(r.Context(), req.Items)
	if err != nil {
		httpx.Error(w, status, err.Error())
		return
	}

	// ⚠️ Sold-out is checked **here**, not at payment. The branch is already
	// known — it is the waiter's own — so there is no reason to repeat the
	// website's compromise of finding out at checkout. A waiter told "lag'mon
	// tugadi" while still standing at the table can offer something else; the
	// same sentence at the till, after the guest has eaten, is an argument.
	branch, err := h.branchByID(r, s.BranchID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, line := range lines {
		if branch.IsSoldOut(line.MenuItemID) {
			httpx.Error(w, http.StatusConflict, line.Name+" bugun tugadi")
			return
		}
	}

	// One check, one brand — the rule menuLines enforces within a request, held
	// across the several requests a table is built from.
	if !brandID.IsZero() && !o.BrandID.IsZero() && brandID != o.BrandID {
		httpx.Error(w, http.StatusBadRequest,
			"bu chekda boshqa brend taomi bor — alohida chek oching")
		return
	}
	for i := range lines {
		lines[i].LineID = lineID()
	}

	now := time.Now()
	set := bson.M{"updatedAt": now}
	if o.BrandID.IsZero() && !brandID.IsZero() {
		// The brand of a check is decided by its first dish, exactly as a
		// basket's is.
		set["brandId"] = brandID
	}
	o.Items = append(o.Items, lines...)
	applyCheckTotals(o, set)

	if _, err := h.Store.Orders.UpdateOne(r.Context(), checkFilter(o.ID, s.BranchID), bson.M{
		"$push": bson.M{"items": bson.M{"$each": lines}},
		"$set":  set,
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, viewCheck(o, now))
}

// applyCheckTotals recomputes the money on a check from its live lines and adds
// it to a Mongo update.
//
// ⚠️ **Always from the lines, never incrementally.** A running total nudged by
// each edit drifts the first time an error path returns early, and the drift is
// invisible: the receipt still adds up on screen because the screen computes it
// the honest way. The stored figures exist for the reports, so they have to be
// the same figures.
func applyCheckTotals(o *models.Order, set bson.M) {
	subtotal := 0
	for _, it := range o.Items {
		if it.Live() {
			subtotal += it.Price * it.Qty
		}
	}
	total := subtotal - o.DiscountTotal
	if total < 0 {
		total = 0
	}
	o.Subtotal, o.Total = subtotal, total
	set["subtotal"], set["total"] = subtotal, total
}

// ---- Sending to the kitchen ----

// StaffFireCheck sends everything not yet sent to the pass.
//
// ⚠️ This is where a check first becomes the kitchen's problem, and it sets
// `queuedAt` to say so — the same timestamp an online order gets when the bank
// confirms. Everything downstream that already reasoned about "is this the
// kitchen's yet" therefore needed no changes at all: the KDS query, the
// new-order chime, the waiting counts, the POS bridge.
func (h *Handler) StaffFireCheck(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok || !requireOpen(w, o) {
		return
	}

	now := time.Now()
	fired := 0
	for i := range o.Items {
		if o.Items[i].Live() && o.Items[i].FiredAt == nil {
			at := now
			o.Items[i].FiredAt = &at
			fired++
		}
	}
	if fired == 0 {
		// Not an error: a double tap on a slow tablet is ordinary, and telling
		// the waiter off for it teaches them to distrust the button. The check
		// comes back unchanged and the screen simply shows nothing pending.
		httpx.JSON(w, http.StatusOK, viewCheck(o, now))
		return
	}

	set := bson.M{"items": o.Items, "updatedAt": now}
	update := bson.M{"$set": set}
	// ⚠️ **Firing clears `readyAt`.** A table's second course arrives at the
	// pass long after the cook marked the first one done, and a check still
	// carrying "ready" is filtered out of the kitchen screen — so the mains
	// would have been ordered, charged for, and never shown to anybody.
	//
	// The rule already existed for orders pushed back a stage, and it is the
	// same fact in both cases: food was asked for that has not been made.
	if o.ReadyAt != nil {
		o.ReadyAt = nil
		set["readyAt"] = nil
	}
	if o.QueuedAt == nil {
		o.QueuedAt = &now
		set["queuedAt"] = now
		// The first course leaving for the kitchen is what confirms a check:
		// before it, nothing has been committed to and the table can get up
		// and leave with no trace beyond an empty check.
		if o.Status == models.StatusPending {
			o.Status = models.StatusConfirmed
			set["status"] = models.StatusConfirmed
			update["$push"] = bson.M{"statusHistory": models.StatusEvent{
				Status: models.StatusConfirmed, At: now,
			}}
		}
	}
	if _, err := h.Store.Orders.UpdateOne(r.Context(), checkFilter(o.ID, s.BranchID), update); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, viewCheck(o, now))
}

// ---- Taking a dish off ----

type voidLineRequest struct {
	// A code from somebody who may write off cooked food, when the person at
	// the screen may not. Empty is the ordinary case.
	PIN string `json:"pin"`

	// How many of the line to remove. Zero or more than the line holds means
	// all of it — a waiter tapping "remove" without thinking about quantity is
	// removing the line, which is what they meant.
	Qty    int    `json:"qty"`
	Reason string `json:"reason"`
	// Whether the food was made and thrown away. Asked because it is the only
	// input the waste report has, and only the person standing at the pass
	// knows the answer.
	Wasted bool `json:"wasted"`
}

// StaffVoidCheckLine removes a line from an open check.
//
// ⚠️ **Two different operations behind one button**, and the difference is
// whether the kitchen has seen the line:
//
//   - **Not fired** — a typo being corrected. It disappears; asking a waiter to
//     justify fixing their own mistyping trains them to type "." in the box and
//     makes the reasons on the real voids worthless.
//   - **Fired** — food that exists. The line stays on the document with who,
//     when and why, and only a cashier may do it. A void that leaves no trace
//     is the oldest way to take money out of a restaurant, and a till that
//     permits it silently is not a control at all.
func (h *Handler) StaffVoidCheckLine(w http.ResponseWriter, r *http.Request) {
	// Authenticated as a waiter first; the stricter permission is demanded
	// below only if the line turns out to be fired. Requiring a cashier up
	// front would mean calling one over to undo a typo.
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok || !requireOpen(w, o) {
		return
	}
	var req voidLineRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	wanted := chi.URLParam(r, "lineId")
	idx := -1
	for i := range o.Items {
		if o.Items[i].LineID == wanted && o.Items[i].Live() {
			idx = i
			break
		}
	}
	if idx < 0 {
		httpx.Error(w, http.StatusNotFound, "qator topilmadi")
		return
	}
	line := o.Items[idx]

	now := time.Now()
	if line.FiredAt == nil {
		// Never cooked: drop it outright.
		o.Items = append(o.Items[:idx], o.Items[idx+1:]...)
	} else {
		// ⚠️ **Not refused when the permission is missing — a manager's code is
		// asked for instead.** Refusing is what teaches a room to share one
		// PIN, and after that every void in the journal carries the same name.
		// See handlers/tilloverride.go.
		who, err := h.resolveActor(r.Context(), s, models.PermVoid, req.PIN)
		if err != nil {
			if errors.Is(err, errNeedsOverride) {
				overrideDenied(w, models.PermVoid)
			} else {
				httpx.Error(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		reason := clampText(req.Reason, 200)
		if reason == "" {
			// The one field this whole record exists for. A void with no reason
			// answers none of the questions it will be read for a month from
			// now, and "sababsiz" is not a category anybody can act on.
			httpx.Error(w, http.StatusBadRequest, "sababini yozing")
			return
		}
		void := &models.CheckLineVoid{
			At: now, ByID: who.ByID, By: who.By,
			AuthByID: who.AuthByID, AuthBy: who.AuthBy,
			Reason: reason, Wasted: req.Wasted,
		}
		if req.Qty > 0 && req.Qty < line.Qty {
			// Part of a line. The remainder stays live and the voided share
			// becomes its own line, so the receipt still adds up and the audit
			// still names an amount.
			o.Items[idx].Qty = line.Qty - req.Qty
			voided := line
			// A fresh id: from here the two halves are separate lines and
			// nothing should be tempted to pair them up by name.
			voided.LineID = lineID()
			voided.Qty = req.Qty
			voided.Void = void
			o.Items = append(o.Items, voided)
		} else {
			o.Items[idx].Void = void
		}
	}

	set := bson.M{"items": o.Items, "updatedAt": now}
	applyCheckTotals(o, set)
	if _, err := h.Store.Orders.UpdateOne(r.Context(),
		checkFilter(o.ID, s.BranchID), bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, viewCheck(o, now))
}

type lineCommentRequest struct {
	Comment string `json:"comment"`
}

// StaffCommentCheckLine writes "no onion" against a dish.
//
// ⚠️ **Only before the line is fired, and refused after.** Once the ticket has
// printed, the paper at the pass carries the old text and nothing in software
// can change that — a silent edit would leave the screen and the kitchen
// disagreeing about the same dish, with the guest finding out. Refused with the
// one instruction that actually works: tell the kitchen, or take the line off
// and add it again.
//
// ⚠️ No permission beyond waiter and no reason asked for. A comment costs the
// restaurant nothing and is the reason this screen exists rather than shouting
// across the room; guarding it would put a manager between a waiter and the
// ordinary business of taking an order.
func (h *Handler) StaffCommentCheckLine(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok || !requireOpen(w, o) {
		return
	}
	var req lineCommentRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	wanted := chi.URLParam(r, "lineId")
	idx := -1
	for i := range o.Items {
		if o.Items[i].LineID == wanted && o.Items[i].Live() {
			idx = i
			break
		}
	}
	if idx < 0 {
		httpx.Error(w, http.StatusNotFound, "qator topilmadi")
		return
	}
	if o.Items[idx].FiredAt != nil {
		httpx.Error(w, http.StatusConflict,
			"bu taom allaqachon oshxonaga yuborilgan — oshxonaga o'zingiz ayting "+
				"yoki qatorni olib tashlab qaytadan qo'shing")
		return
	}

	o.Items[idx].Comment = clampText(req.Comment, 200)
	now := time.Now()
	if _, err := h.Store.Orders.UpdateOne(r.Context(),
		checkFilter(o.ID, s.BranchID),
		bson.M{"$set": bson.M{"items": o.Items, "updatedAt": now}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, viewCheck(o, now))
}
