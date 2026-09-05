package handlers

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/receipt"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	// The phone's own id for this tap, when it is resending something it
	// queued.
	//
	// ⚠️ **The one endpoint here that needs one.** Every other queued operation
	// states an absolute — `qty = 3`, `served = true`, "void this line id" — and
	// repeating it lands on the same answer. This one says "one more", and a
	// resend the phone could not know had already arrived is a guest charged
	// twice for food nobody ordered.
	//
	// ⚠️ Absent on every online tap, which is nearly all of them: an id here is
	// a phone saying "I am not sure you heard me", and a screen with a working
	// connection never is.
	OpID string `json:"opId"`
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

	// ⚠️ **Answered as success, not as an error, and that is the whole point.**
	// The phone resending this cannot tell "you never heard me" from "you heard
	// me and I lost the reply" — so a refusal would be read as a failure and
	// queued again, forever. Handing back the check as it now stands is the
	// truthful answer to both readings: the dishes are on it.
	if req.OpID != "" && slices.Contains(o.AppliedOps, req.OpID) {
		httpx.JSON(w, http.StatusOK, viewCheck(o, time.Now(), s.ID))
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
	// ⚠️ **The batch is checked against what is being added, not against a flag.**
	// The stop list above answers "has this dish already gone", which is one tap
	// too late: a limit of two, five taps on the tile, nothing sold yet, and five
	// go to the kitchen. See limitRefusal.
	if msg := h.limitRefusal(r.Context(), branch, lines); msg != "" {
		httpx.Error(w, http.StatusConflict, msg)
		return
	}

	// One check, one brand — the rule menuLines enforces within a request, held
	// across the several requests a table is built from.
	if !brandID.IsZero() && !o.BrandID.IsZero() && brandID != o.BrandID {
		httpx.Error(w, http.StatusBadRequest,
			"bu chekda boshqa brend taomi bor — alohida chek oching")
		return
	}
	// ⚠️ **The same dish tapped twice is one line of two, not two lines of
	// one.** A till is used by tapping: four coffees is the tile pressed four
	// times, and stacking four identical rows made a check nobody could read
	// back to a guest — and no way to correct a miscount except removing rows
	// one at a time.
	//
	// ⚠️ **Only into a line the kitchen has not seen.** Once a line is fired the
	// ticket at the pass names a quantity, and quietly growing it would leave
	// the paper and the screen disagreeing about the same dish. A fired line
	// therefore stays as it is and the new one lands beside it — which is also
	// the honest reading: two of those are cooking, one more has been asked for.
	//
	// ⚠️ **Same dish is not enough** — the options and the note have to match
	// too. "Osh (katta)" and "Osh (kichik)" are different food, and merging a
	// plain one into a line that says "piyozsiz" sends the wrong instruction to
	// the kitchen for both of them.
	// ⚠️ **Checked against the whole check, not against what is being added.**
	// The same bottle scanned twice is the failure this catches, and the second
	// scan usually arrives in a later request than the first.
	if msg, err := h.markingRefusal(r.Context(), append(append([]models.OrderItem{}, o.Items...), lines...)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	} else if msg != "" {
		httpx.Error(w, http.StatusConflict, msg)
		return
	}

	for _, line := range lines {
		// ⚠️ **A marked line never merges.** Merging is what makes four taps on
		// one tile a line of four, and it is right for every dish that is not a
		// physical object with its own code: two bottles are two codes, and a
		// line of two behind one code files one bottle and hands over two.
		if line.MarkCode == "" {
			if at := mergeableLine(o.Items, line); at >= 0 {
				o.Items[at].Qty += line.Qty
				continue
			}
		}
		line.LineID = lineID()
		o.Items = append(o.Items, line)
	}

	now := time.Now()
	set := bson.M{"updatedAt": now, "items": o.Items}
	if req.OpID != "" {
		// ⚠️ **Written in the same update as the dishes.** Recorded before, a
		// crash between the two loses the food and keeps the receipt; recorded
		// after, it loses the receipt and keeps the food — and the second resend
		// then adds everything twice. One write, or the guard is a guess.
		o.AppliedOps = appendOp(o.AppliedOps, req.OpID)
		set["appliedOps"] = o.AppliedOps
	}
	if o.BrandID.IsZero() && !brandID.IsZero() {
		// The brand of a check is decided by its first dish, exactly as a
		// basket's is.
		set["brandId"] = brandID
	}
	applyCheckTotals(o, set)

	if _, err := h.Store.Orders.UpdateOne(r.Context(), checkFilter(o.ID, s.BranchID),
		bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ **Here, not only at payment.** The dish that has just reached its batch
	// has to leave the menu now — the next waiter is already reaching for the
	// tile. Recomputing only when a check closed meant the last portions were
	// sold several times over while the tables that had them sat open.
	h.applyDailyLimits(r.Context(), s.BranchID)
	// ⚠️ **The shelf moves when the tile is tapped, not when the money
	// arrives.** That is what makes the last portion of osh un-promisable to a
	// second table — and it is what the derived arithmetic always counted, so
	// switching to written rows changed no figure. See handlers/stocksale.go.
	h.syncOrderStock(r.Context(), o)
	httpx.JSON(w, http.StatusOK, viewCheck(o, now, s.ID))
}

// appendOp records an applied operation, keeping only the recent ones.
//
// ⚠️ **Twenty, because the window a duplicate can arrive in is one outage, not
// one evening.** A phone reconnects after seconds or minutes and flushes what it
// queued; nothing resends an hour later. An unbounded list would grow with every
// tap of every busy table and be read back on every refresh of the room.
func appendOp(ops []string, id string) []string {
	const keep = 20
	ops = append(ops, id)
	if len(ops) > keep {
		ops = ops[len(ops)-keep:]
	}
	return ops
}

// mergeableLine finds the line an incoming dish should be added to, or -1.
//
// The rules are the three sentences above StaffAddCheckLines: not voided, not
// yet fired, same dish, same options, same note.
func mergeableLine(items []models.OrderItem, add models.OrderItem) int {
	for i := range items {
		it := items[i]
		if !it.Live() || it.FiredAt != nil {
			continue
		}
		if it.MenuItemID != add.MenuItemID || it.Comment != add.Comment {
			continue
		}
		// ⚠️ **Two guests ordering the same dish stay two lines**, and so do two
		// courses of it. This is the case the line model was always worried
		// about: merging them would hand one guest a bill for both, and send a
		// dessert to the pass with the starters.
		if it.Guest != add.Guest || it.Course != add.Course {
			continue
		}
		// ⚠️ **Half a loaf and a whole one are two lines.** They are different
		// food and different money, and merging them would hide a half inside
		// a quantity — the kitchen would make two whole ones and the guest
		// would be charged for one and a half.
		if it.Portion != add.Portion {
			continue
		}
		if !sameOptions(it.Options, add.Options) {
			continue
		}
		return i
	}
	return -1
}

// sameOptions reports whether two lines were ordered with the same choices.
//
// ⚠️ **Order-insensitive.** The choices arrive in whatever order the dialog
// listed the groups in, and a screen that lists "Hajm" before "Qo'shimcha" one
// day and after it the next would stop merging without anybody changing
// anything. Lines are short — a handful of choices — so the quadratic walk is
// cheaper than sorting a copy.
func sameOptions(a, b []models.OrderItemOption) bool {
	if len(a) != len(b) {
		return false
	}
	used := make([]bool, len(b))
	for _, want := range a {
		found := false
		for i, got := range b {
			if used[i] || got.Name != want.Name || got.Choice != want.Choice {
				continue
			}
			used[i], found = true, true
			break
		}
		if !found {
			return false
		}
	}
	return true
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
	payable := subtotal - o.DiscountTotal
	if payable < 0 {
		payable = 0
	}
	service := serviceOn(payable, o.ServicePercent)
	o.Subtotal, o.ServiceCharge, o.Total = subtotal, service, payable+service
	set["subtotal"], set["total"] = subtotal, o.Total
	set["serviceCharge"] = service
}

// serviceOn is the room's percentage of what the table actually pays.
//
// ⚠️ **After the discount, not before.** A guest who was given 20% off and then
// charged service on the full bill is being charged for a discount they were
// told they had — and it is the receipt they read most carefully, because
// somebody just did them a favour.
//
// ⚠️ **Rounded half-up to the som**, once, here. Rounding in two places is how
// the panel, the paper and the drawer end up one som apart, and one som apart
// is what somebody spends an evening looking for.
func serviceOn(payable, percent int) int {
	if payable <= 0 || percent <= 0 {
		return 0
	}
	return (payable*percent + 50) / 100
}

// ---- Sending to the kitchen ----

// fireRequest asks for one course instead of the whole check.
type fireRequest struct {
	Course *int `json:"course,omitempty"`
}

// StaffFireCheck sends what has not been sent to the pass — a course, or
// everything.
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
	// ⚠️ **Optional, and absent means everything** — which is what every screen
	// sent before courses existed, and what a counter selling coffee will send
	// forever. A body is not required at all.
	var req fireRequest
	if err := httpx.DecodeOptional(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now()
	fired := 0
	for i := range o.Items {
		if !o.Items[i].Live() || o.Items[i].FiredAt != nil {
			continue
		}
		// ⚠️ **Courses are sent one at a time, and that is the whole point of
		// them.** The starters go now and the mains twenty minutes later, when
		// the room is ready — firing everything at once is what a kitchen
		// screen cannot undo, because the food is already being made.
		if req.Course != nil && o.Items[i].Course != *req.Course {
			continue
		}
		at := now
		o.Items[i].FiredAt = &at
		fired++
	}
	if fired == 0 {
		// Not an error: a double tap on a slow tablet is ordinary, and telling
		// the waiter off for it teaches them to distrust the button. The check
		// comes back unchanged and the screen simply shows nothing pending.
		httpx.JSON(w, http.StatusOK, viewCheck(o, now, s.ID))
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
	// ⚠️ **The kitchen ticket prints itself**, and this is the whole reason a
	// restaurant buys a printer for the pass: the waiter presses "send" at the
	// table and the paper is already at the grill. Queued **after** the write,
	// so a printer nobody plugged in cannot be the reason an order fails to
	// reach the kitchen — the screen has it either way.
	h.queueKitchenTicket(r.Context(), s.BranchID, o, now)
	// Firing takes nothing extra off the shelf — the row was written when the
	// line was rung up. What it changes is whether removing the line later can
	// put anything back, which is the question `cookedLine` answers.
	h.syncOrderStock(r.Context(), o)
	httpx.JSON(w, http.StatusOK, viewCheck(o, now, s.ID))
}

// queueKitchenTicket sends the pass what has just been fired.
//
// ⚠️ **Only the lines fired by this press.** A second course sent twenty
// minutes later must not reprint the starters — the cook would make them again,
// and nothing on the paper would say they had already gone out.
func (h *Handler) queueKitchenTicket(
	ctx context.Context, branchID primitive.ObjectID, o *models.Order, at time.Time,
) {
	just := *o
	just.Items = nil
	for _, it := range o.Items {
		if it.Live() && it.FiredAt != nil && it.FiredAt.Equal(at) {
			just.Items = append(just.Items, it)
		}
	}
	if len(just.Items) == 0 {
		return
	}
	set := h.receiptSettingsOf(ctx, branchID)
	h.queueReceipt(ctx, branchID, receipt.Kitchen, set.Kitchen,
		h.checkReceiptOf(ctx, &just), o)
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
	// ⚠️ **The line above decided what this does.** An unfired line was dropped
	// from the check outright, so its row is reversed and the shelf is untouched
	// — nothing was ever cooked. A fired line stayed, carrying the answer to
	// "was it thrown away", and where it was the row stands: putting those
	// ingredients back would file real waste as a correction.
	h.syncOrderStock(r.Context(), o)
	httpx.JSON(w, http.StatusOK, viewCheck(o, now, s.ID))
}

// lineEditRequest is a change to one line that has not gone to the kitchen yet:
// the note against it, how many of it, or both.
//
// ⚠️ **Both fields are pointers, and that is the whole correctness of it.** The
// screens send one field at a time — the comment dialog sends a note, the
// quantity stepper sends a number — and a plain string would make "no comment
// in this request" indistinguishable from "clear the comment". A cashier
// pressing "+" would silently wipe the guest's "piyozsiz", and nothing on
// either screen would say so.
type lineEditRequest struct {
	Comment *string `json:"comment,omitempty"`
	Qty     *int    `json:"qty,omitempty"`
	// Which guest pays for this line, and which course it goes out with.
	Guest  *int `json:"guest,omitempty"`
	Course *int `json:"course,omitempty"`
}

// The most of one dish a single line may carry.
//
// ⚠️ A limit rather than none: the stepper is held down by a thumb on a
// touchscreen, and "128 lag'mon" reaching the kitchen is a real ticket somebody
// has to walk over and cancel. A table ordering more than ninety-nine of one
// thing is a banquet, and a banquet is a second line.
const maxLineQty = 99

// cooksAffected reports whether an edit changes what the kitchen has to make.
//
// Split out so the rule reads as one sentence in the handler and can be seen
// from the test: everything except which guest is paying.
func cooksAffected(req lineEditRequest) bool {
	return req.Qty != nil || req.Comment != nil || req.Course != nil
}

// applyLineEdit applies one screen's edit to one live line.
//
// Split out from the handler so the rules can be sealed in a test: this is the
// only place a line's quantity changes upwards, and the guard against zero is
// what stops "−" from becoming a void with no reason and no record.
func applyLineEdit(line *models.OrderItem, req lineEditRequest) error {
	if req.Guest != nil {
		// ⚠️ Guests are numbered from one and capped at the size of a table
		// anybody actually seats. Zero is legal and means the whole party —
		// putting a line back on the shared bill.
		if *req.Guest < 0 || *req.Guest > maxGuests {
			return errBadGuest
		}
		line.Guest = *req.Guest
	}
	if req.Course != nil {
		// Zero is legal here too: "send it with everything else".
		if *req.Course < 0 || *req.Course > maxCourse {
			return errBadCourse
		}
		line.Course = *req.Course
	}
	if req.Qty != nil {
		// ⚠️ Zero is refused rather than treated as "remove". Taking a line off
		// is a different act with a different record — an unfired line is
		// dropped outright, a fired one needs a cashier and a reason — and a
		// stepper that quietly performs a void at zero is how a till stops
		// being able to answer where the food went.
		if *req.Qty < 1 || *req.Qty > maxLineQty {
			return errBadLineQty
		}
		line.Qty = *req.Qty
	}
	if req.Comment != nil {
		line.Comment = clampText(*req.Comment, 200)
	}
	return nil
}

var errBadLineQty = errors.New("soni 1 dan " +
	strconv.Itoa(maxLineQty) + " gacha bo'lishi kerak")

// How many guests one check may be split between, and how many courses a meal
// may be sent in.
//
// ⚠️ Limits rather than none, for the same reason the quantity has one: both
// numbers come off a touchscreen, and a check split between two hundred guests
// is a screen of tabs nobody can use and a bill nobody can print.
const (
	maxGuests = 20
	maxCourse = 9
)

var errBadGuest = errors.New("mehmon raqami 1 dan " +
	strconv.Itoa(maxGuests) + " gacha bo'lishi kerak")
var errBadCourse = errors.New("kurs 1 dan " +
	strconv.Itoa(maxCourse) + " gacha bo'lishi kerak")

// ---- The room this screen belongs to ----

// StaffBranch is the room, read from the employee's own branch.
//
// ⚠️ **Not `GET /restaurant`, and that was a real bug.** The public profile
// answers "which branch is this *visitor* being served from" — the site's
// default, or whatever a cookie says. A till belongs to a branch by its token,
// and on a two-branch company the counter in Yunusobod was drawing Chilonzor's
// floor plan: the right number of tables, in the right shapes, for a different
// room. Nothing looked broken, and the first sign would have been a waiter
// unable to find table 7.
//
// ⚠️ **The plan comes through `bookingSettings`**, not raw: this screen has to
// draw a room, so it needs the sizes filled in — unlike the settings page,
// which saves what it is given and must not be handed invented defaults.
func (h *Handler) StaffBranch(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	branch, err := h.branchByID(r, s.BranchID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "filial topilmadi")
		return
	}
	// The currency is the company's, like the menu's prices — a branch does not
	// bill in a different one.
	currency := "UZS"
	var rest models.Restaurant
	if err := h.Store.Restaurant.FindOne(r.Context(), bson.M{}).Decode(&rest); err == nil &&
		rest.Currency != "" {
		currency = rest.Currency
	}
	biz := h.businessOf(r.Context(), branch)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"id":       branch.ID.Hex(),
		"name":     branch.Name,
		"currency": currency,
		"booking":  bookingSettings(branch.Booking),
		// ⚠️ The service rate reaches the device so a check opened during an
		// outage charges what the same table would have been charged a minute
		// earlier. Without it the till quietly billed two different amounts
		// depending on the wifi — and the guest who paid less is the one who
		// never finds out.
		"servicePercent": servicePercentOf(branch),
		// ⚠️ **Capabilities, not a business type.** The till needs to know
		// whether to open on a scanner and whether there is a room to draw; it
		// does not need to know the difference between a pharmacy and a flower
		// shop, and a screen that switched on the type itself would need
		// editing every time a type is added.
		//
		// ⚠️ **Read live, unlike `Defaults()`.** What a brand *offers* is a
		// preference and is copied onto it once, so an owner can change it.
		// What a counter *is* — scanned or tapped, with a floor plan or without
		// — is a description of the business rather than a preference, and a
		// stored copy of it would be a second answer that can go stale.
		"sellsGoods": biz.ScansToSell(),
		"hasTables":  biz.HasTables(),
		// ⚠️ **Only the port reaches the device.** The label layout is decoded
		// on the server (see tillbarcode.go), so a till that reads it here
		// would be a second decoder — and two decoders eventually disagree
		// about how many grams a packet holds.
		"scalePort": branch.Scale.Port,
	})
}

// businessOf reports what kind of business a branch belongs to.
//
// ⚠️ **Missing brand means restaurant**, which is what an install that predates
// brands is. A branch whose brand has been deleted must not lose its floor
// plan — a till that silently turns into a shop counter is a restaurant that
// cannot open a table.
func (h *Handler) businessOf(ctx context.Context, branch *models.Branch) models.BusinessType {
	if branch.BrandID.IsZero() {
		return models.BizRestaurant
	}
	var b models.Brand
	if err := h.Store.Brands.FindOne(ctx, bson.M{"_id": branch.BrandID}).Decode(&b); err != nil {
		return models.BizRestaurant
	}
	return b.BusinessType
}

// ---- Today's bookings, from the till ----

// tillReservation is a booking as the floor screen needs it: who, when, how
// many, which table. Deliberately narrower than the panel's view — a waiter
// does not need the audit trail, and the phone number belongs to the office.
type tillReservation struct {
	ID          string    `json:"id"`
	Number      string    `json:"number"`
	Name        string    `json:"name"`
	At          time.Time `json:"at"`
	Guests      int       `json:"guests"`
	TableNumber string    `json:"tableNumber,omitempty"`
	Status      string    `json:"status"`
	Comment     string    `json:"comment,omitempty"`
}

// StaffReservations lists the bookings still to come today, for this branch.
//
// ⚠️ **The till is where a booking actually lands.** It is agreed on the phone
// and written in the panel, and then somebody at seven in the evening has to
// know that table 12 is spoken for at half past — on the screen they are
// standing at, not one they would have to go and open. A booked table that gets
// walked in on is a party turned away at the door of a restaurant that had a
// table for them.
//
// ⚠️ **Only what is still ahead**, and only today: a list that keeps this
// morning's finished lunches is a list nobody scrolls past.
func (h *Handler) StaffReservations(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	now := time.Now()
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0,
		now.Location()).AddDate(0, 0, 1)
	// ⚠️ Scoped to the employee's own branch, like every other till query: the
	// id in a token decides which room this is, never a parameter.
	filter := bson.M{
		"branchId": s.BranchID,
		"endsAt":   bson.M{"$gte": now},
		"at":       bson.M{"$lt": end},
		"status":   bson.M{"$nin": []string{"cancelled", "done"}},
	}
	cur, err := h.Store.Reservations.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "at", Value: 1}}).SetLimit(50))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var rows []models.Reservation
	if err := cur.All(r.Context(), &rows); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ Built as an empty slice, not a nil one: a nil slice marshals to `null`
	// and the screen maps over it — the trap this codebase has been bitten by
	// twice.
	out := make([]tillReservation, 0, len(rows))
	for _, b := range rows {
		out = append(out, tillReservation{
			ID:          b.ID.Hex(),
			Number:      b.Number,
			Name:        b.Customer.Name,
			At:          b.At,
			Guests:      b.Guests,
			TableNumber: b.TableNumber,
			Status:      string(b.Status),
			Comment:     b.Comment,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"reservations": out})
}

// ---- Moving lines between checks ----

type moveLinesRequest struct {
	LineIDs []string `json:"lineIds" validate:"required,min=1"`
	ToID    string   `json:"toCheckId" validate:"required"`
	// A manager's code, when the destination belongs to somebody else — see
	// StaffMoveCheckLines.
	PIN string `json:"pin"`
}

// StaffMoveCheckLines moves dishes from one open check to another.
//
// ⚠️ **Two tables that turned out to be one, or one that turned out to be two.**
// A party moves to a bigger table halfway through, a guest joins their friends,
// four people at the counter turn out to be paying separately. Every dining room
// does this several times an evening, and a till that cannot do it makes the
// waiter void the food and ring it in again — which throws away the times, the
// audit and, if it was already cooked, the money.
//
// ⚠️ **The lines keep everything except which check they are on**: what was
// fired stays fired, the void history, the note, the guest. Re-creating them
// instead would tell the kitchen to cook a second dinner.
func (h *Handler) StaffMoveCheckLines(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	from, ok := h.loadCheck(w, r, s)
	if !ok || !requireOpen(w, from) {
		return
	}
	var req moveLinesRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	toID, err := primitive.ObjectIDFromHex(req.ToID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "chek topilmadi")
		return
	}
	if toID == from.ID {
		// Not an error worth a refusal page: the screen offered the check it is
		// already on, and moving a line to where it is is a tap that changes
		// nothing.
		httpx.JSON(w, http.StatusOK, viewCheck(from, time.Now(), s.ID))
		return
	}

	// ⚠️ The destination is loaded through the **same branch filter** as the
	// source. Without it a waiter could type another branch's check id and move
	// a table's food onto a bill in a different building.
	var to models.Order
	if err := h.Store.Orders.FindOne(r.Context(),
		checkFilter(toID, s.BranchID)).Decode(&to); err != nil {
		httpx.Error(w, http.StatusNotFound, "chek topilmadi")
		return
	}
	if !requireOpen(w, &to) {
		return
	}
	// ---- Whose table it is going to ----
	//
	// ⚠️ **A waiter may move food onto their own tables and nobody else's.**
	// Moving a line changes what another person's guest is going to be asked
	// to pay, on a bill that waiter is standing at and answering for — and the
	// first they would know about it is the guest disputing a dish they never
	// ordered. It is also the shape every quiet transfer of food takes: three
	// dishes onto a table that is about to be closed by somebody who is not
	// looking.
	//
	// ⚠️ **Not refused outright, asked for.** The cashier's permission covers
	// it, and anybody without it gets the manager's-code dialog — the same
	// answer a void and a discount give. Refusing flatly would mean a party
	// that genuinely moved between two waiters' sections could not be served
	// without a manager doing it on their own screen.
	//
	// ⚠️ The check's own server, not the table's: a table is furniture, and
	// the person answering for a bill is the one whose name is on it.
	if !to.Check.ServerID.IsZero() && to.Check.ServerID != s.ID {
		if _, err := h.resolveActor(r.Context(), s, models.PermCashier, req.PIN); err != nil {
			if errors.Is(err, errNeedsOverride) {
				overrideDenied(w, models.PermCashier)
			} else {
				httpx.Error(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
	}

	wanted := map[string]bool{}
	for _, id := range req.LineIDs {
		wanted[id] = true
	}
	var moved []models.OrderItem
	var kept []models.OrderItem
	for _, it := range from.Items {
		// ⚠️ A voided line does not move. It is the record of food written off
		// this check, and carrying it across would move the blame with it.
		if wanted[it.LineID] && it.Live() {
			moved = append(moved, it)
			continue
		}
		kept = append(kept, it)
	}
	if len(moved) == 0 {
		httpx.Error(w, http.StatusNotFound, "qator topilmadi")
		return
	}

	now := time.Now()
	from.Items = kept
	to.Items = append(to.Items, moved...)

	fromSet := bson.M{"items": from.Items, "updatedAt": now}
	applyCheckTotals(from, fromSet)
	toSet := bson.M{"items": to.Items, "updatedAt": now}
	applyCheckTotals(&to, toSet)
	// ⚠️ **Food that was already cooking makes the destination the kitchen's
	// too.** Otherwise a check that has never been fired can end up holding
	// fired lines while the kitchen screen has never heard of it — the same
	// reasoning as firing, one level along.
	if to.QueuedAt == nil {
		for _, it := range moved {
			if it.FiredAt != nil {
				toSet["queuedAt"] = now
				break
			}
		}
	}

	if _, err := h.Store.Orders.UpdateOne(r.Context(),
		checkFilter(from.ID, s.BranchID), bson.M{"$set": fromSet}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := h.Store.Orders.UpdateOne(r.Context(),
		checkFilter(to.ID, s.BranchID), bson.M{"$set": toSet}); err != nil {
		// ⚠️ The source has already been written. Put the lines back rather
		// than leaving them nowhere: a half-finished move loses food off both
		// bills, and the guest is charged for neither.
		from.Items = append(kept, moved...)
		back := bson.M{"items": from.Items, "updatedAt": now}
		applyCheckTotals(from, back)
		_, _ = h.Store.Orders.UpdateOne(r.Context(),
			checkFilter(from.ID, s.BranchID), bson.M{"$set": back})
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Both checks, because a moved line leaves one shelf ledger and joins
	// another. The food did not move — the same branch cooked it — so the two
	// reconciles cancel out in the balance and leave a readable trail in the
	// audit, which is exactly what a moved line is.
	h.syncOrderStock(r.Context(), from)
	h.syncOrderStock(r.Context(), &to)
	httpx.JSON(w, http.StatusOK, viewCheck(from, now, s.ID))
}

// StaffEditCheckLine changes a line: its note, its quantity, or both.
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
func (h *Handler) StaffEditCheckLine(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok || !requireOpen(w, o) {
		return
	}
	var req lineEditRequest
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
	// ⚠️ **Which edits survive the kitchen is not one rule but two.**
	//
	// The quantity, the note and the course describe *food that has to be made*
	// — and once the ticket has printed, the paper at the pass carries the old
	// version. Software cannot change it, so a silent edit leaves the screen and
	// the kitchen disagreeing about the same dish, with the guest finding out.
	//
	// Which guest pays for it is a fact about the **bill**, and a table decides
	// how to split it when the plates are cleared. Refusing it after firing
	// would mean the one moment splitting is ever asked for is the moment it
	// stops working — and the waiter's workaround is a pen.
	if o.Items[idx].FiredAt != nil && cooksAffected(req) {
		httpx.Error(w, http.StatusConflict,
			"bu taom allaqachon oshxonaga yuborilgan — oshxonaga o'zingiz ayting "+
				"yoki qatorni olib tashlab qaytadan qo'shing")
		return
	}

	if err := applyLineEdit(&o.Items[idx], req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now()
	// ⚠️ The money is recomputed here and it was not before: a comment costs
	// nothing, a quantity is the bill. Left out, the stored total drifts from
	// the lines the moment somebody presses "+", and the screen keeps looking
	// right because it adds the lines up itself — the reports do not.
	set := bson.M{"items": o.Items, "updatedAt": now}
	applyCheckTotals(o, set)
	if _, err := h.Store.Orders.UpdateOne(r.Context(),
		checkFilter(o.ID, s.BranchID),
		bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ Three down to two scales the row rather than re-reading the card: the
	// recipe is frozen per portion precisely so a quantity correction cannot
	// reprice a sale against a recipe edited since.
	h.syncOrderStock(r.Context(), o)
	httpx.JSON(w, http.StatusOK, viewCheck(o, now, s.ID))
}

// servicePercentOf is the rate a table at this branch is charged.
//
// ⚠️ Zero when the setting is off, so the device has one number to store and
// no second way to express "no service charge" — two ways is a setting that
// looks on and charges nothing.
func servicePercentOf(b *models.Branch) int {
	if b == nil || !b.Service.Enabled {
		return 0
	}
	return b.Service.Percent
}
