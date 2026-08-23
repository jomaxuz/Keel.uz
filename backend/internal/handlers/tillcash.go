package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/receipt"

	"go.mongodb.org/mongo-driver/bson"
)

// The cash drawer, opened and closed from the till itself.
//
// ⚠️ **This is where it belongs.** The drawer is counted by the person standing
// in front of it at the end of the evening, and until now the only way to do
// that was the admin panel — which meant either giving a cashier a panel login
// (everything: the customer base, the payment keys, the reports) or having the
// manager come in the next morning to count a drawer somebody else emptied.
// Both are worse than the thing they were avoiding.
//
// ⚠️ **The arithmetic is not repeated here.** openShiftFor and closeShiftFor
// live in cash.go and are shared with the panel: two implementations of "what
// should be in the drawer" would eventually disagree, and then the restaurant
// has two answers about missing money and no way to tell which is right.

// StaffCashShift is what the till shows about the drawer.
func (h *Handler) StaffCashShift(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	scope := bson.M{"branchId": s.BranchID}
	shift, err := h.openCashShift(r, scope)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if shift == nil {
		// ⚠️ Not an error and not an empty response: "no shift is open" is the
		// answer, and the screen draws the button that opens one.
		httpx.JSON(w, http.StatusOK, map[string]any{
			"open": nil,
			// Whether this person may open one, so the screen can decide
			// between a button and a note. ⚠️ Courtesy only — the server asks
			// again on the way in, and asks a manager if the answer is no.
			"canShift": s.Can(models.PermShift),
		})
		return
	}
	figures, entries, err := h.shiftFigures(r, shift)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"open": shift, "figures": figures, "entries": entries,
		"canShift": s.Can(models.PermShift),
	})
}

type tillOpenShiftRequest struct {
	OpeningFloat int    `json:"openingFloat"`
	Note         string `json:"note"`
	// A code from somebody who may open the drawer, when this person may not.
	PIN string `json:"pin"`
}

// StaffOpenCashShift starts the evening.
func (h *Handler) StaffOpenCashShift(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	var req tillOpenShiftRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	who, err := h.resolveActor(r.Context(), s, models.PermShift, req.PIN)
	if err != nil {
		if errors.Is(err, errNeedsOverride) {
			overrideDenied(w, models.PermShift)
		} else {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	shift, status, err := h.openShiftFor(
		r, s.BranchID, req.OpeningFloat, req.Note, shiftActorName(who))
	if err != nil {
		httpx.Error(w, status, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, shift)
}

type tillCloseShiftRequest struct {
	Counted      int    `json:"counted"`
	VarianceNote string `json:"varianceNote"`
	Note         string `json:"note"`
	PIN          string `json:"pin"`
}

// StaffCloseCashShift counts the drawer and records the difference.
//
// ⚠️ **The register's day is asked to end alongside**, exactly as it is from the
// panel — and from here it is better placed to actually happen, because this
// screen is the one standing on the restaurant's network. The request is
// refused while any sale is unfiled; the drawer count succeeds either way,
// because counting cash is this endpoint's job and it has already been done.
func (h *Handler) StaffCloseCashShift(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	var req tillCloseShiftRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	who, err := h.resolveActor(r.Context(), s, models.PermShift, req.PIN)
	if err != nil {
		if errors.Is(err, errNeedsOverride) {
			overrideDenied(w, models.PermShift)
		} else {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	shift, err := h.openCashShift(r, bson.M{"branchId": s.BranchID})
	if err != nil || shift == nil {
		httpx.Error(w, http.StatusBadRequest, "ochiq smena yo'q")
		return
	}
	figures, status, err := h.closeShiftFor(r, shift, req.Counted,
		req.VarianceNote, req.Note, shiftActorName(who))
	if err != nil {
		httpx.Error(w, status, err.Error())
		return
	}

	fiscalNote := ""
	if _, ferr := h.requestCloseDay(r.Context(), s.BranchID,
		shiftActorName(who), shift.ID); ferr != nil {
		fiscalNote = ferr.Error()
	}

	var saved models.CashShift
	_ = h.Store.CashShifts.FindOne(r.Context(), bson.M{"_id": shift.ID}).Decode(&saved)
	// ⚠️ **The Z report comes back with the close, not from a second button.**
	// It is the paper for the shift that has just ended, and a screen that
	// closes the drawer and then asks somebody to remember to print it is a
	// screen that produces evenings with no Z report at all.
	body := map[string]any{
		"shift": saved, "figures": figures, "fiscalNote": fiscalNote,
	}
	if data, tpl, err := h.shiftReportData(r.Context(), &saved, figures, "Z"); err == nil {
		body["lines"] = receipt.RenderShift(tpl, data)
		body["widthMM"] = tpl.WidthMM
	}
	httpx.JSON(w, http.StatusOK, body)
}

// shiftActorName is the name recorded on the shift.
//
// ⚠️ **Both names when a manager authorised it**, in one string, because
// CashShift stores who opened and closed it as text rather than as a pair of
// ids — and the honest answer to "who counted this drawer" is "the cashier,
// with the manager's permission". Recording only the manager would credit them
// with a count they never made; only the cashier would hide that they needed
// permission to make it.
func shiftActorName(who actor) string {
	if who.AuthBy == "" {
		return who.By
	}
	return who.By + " (" + who.AuthBy + ")"
}

// ---- Money in and out of the drawer, from the till ----
//
// ⚠️ **The cashier is the one who does this, so the till is where it belongs.**
// A supplier paid in cash at the door, change brought in at the start of the
// evening, a courier's float — every one of them happens at the counter, and
// until now the only place to record one was the admin panel. That meant either
// a panel login on the till (the customer base, the payment keys, the reports)
// or a drawer whose figure nobody could reconcile because half its movements
// were never written down.
//
// ⚠️ **The same rule as the panel's, from the same function.** A payout may not
// exceed what is in the drawer — see cashOutRefusal in cash.go. Two copies of
// that check would eventually disagree, and the one that had drifted would be
// the one on the counter.

type tillCashEntryRequest struct {
	// "in" or "out".
	Kind     string `json:"kind"`
	Category string `json:"category"`
	Amount   int    `json:"amount"`
	Note     string `json:"note"`
	// A code from somebody who may move cash, when this person may not.
	PIN string `json:"pin"`
}

// StaffAddCashEntry records money put in or taken out at the counter.
func (h *Handler) StaffAddCashEntry(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	var req tillCashEntryRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// ⚠️ **The drawer's own permission, not the waiter's.** Taking money out of
	// a till is the cashier's act — the same one that opens and closes the
	// shift — and a floor tablet must not be able to do it because it happens
	// to be signed in. A waiter who genuinely needs to is asked for a manager's
	// code, exactly as they are for a void.
	who, err := h.resolveActor(r.Context(), s, models.PermShift, req.PIN)
	if err != nil {
		if errors.Is(err, errNeedsOverride) {
			overrideDenied(w, models.PermShift)
		} else {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	shift, err := h.openCashShift(r, bson.M{"branchId": s.BranchID})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if shift == nil {
		// ⚠️ Refused rather than filed against no shift. Cash that moved
		// outside a shift belongs to no count at the end of the evening — the
		// same silence the shift gate exists to prevent.
		httpx.Error(w, http.StatusBadRequest, "ochiq smena yo'q")
		return
	}
	entry, code, err := h.addCashEntry(r, shift, cashEntryInput{
		Kind: req.Kind, Category: req.Category, Amount: req.Amount,
		Note: req.Note, By: who.By,
	})
	if err != nil {
		httpx.Error(w, code, err.Error())
		return
	}
	figures, entries, err := h.shiftFigures(r, shift)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// The whole drawer comes back, not just the row: the screen shows a running
	// expected total, and a client that added the number itself would be a
	// second opinion about what is in the till.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"entry": entry, "figures": figures, "entries": entries,
	})
}

// ---- Yesterday's paper ----
//
// ⚠️ **A Z report is printed once, and once is not always enough.** The roll
// jams, the paper runs out, the accountant asks for Tuesday's in March. Until
// now the only copy was the one that came out of the printer at the moment the
// shift closed, and a cashier who lost it had nowhere to go — the panel shows
// the figures but not the paper, and re-deriving them by hand is how two
// different answers about one evening start existing.

// StaffClosedShifts lists recent closed shifts for this branch.
func (h *Handler) StaffClosedShifts(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	// ⚠️ **A short list, and the till is not a reporting screen.** The question
	// asked at a counter is "print yesterday's again", not "how did March go" —
	// that one belongs to the panel, where the period picker and the export
	// are. A till that scrolled a year of shifts would be a till worth reading
	// somebody else's takings off.
	shifts, err := h.recentClosedShifts(r.Context(), s.BranchID, 10)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"shifts": shifts})
}

// StaffShiftZReport re-renders the Z report of one closed shift.
func (h *Handler) StaffShiftZReport(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "smena topilmadi")
		return
	}
	var shift models.CashShift
	// ⚠️ Filtered by branch, not only by id. Without it a cashier could type
	// another branch's shift id and print a building they do not work in — the
	// same rule every single-document read on this screen follows.
	if err := h.Store.CashShifts.FindOne(r.Context(), bson.M{
		"_id": id, "branchId": s.BranchID,
	}).Decode(&shift); err != nil {
		httpx.Error(w, http.StatusNotFound, "smena topilmadi")
		return
	}
	if shift.ClosedAt == nil {
		// The open shift's paper is the X report, and it has its own button.
		httpx.Error(w, http.StatusBadRequest, "smena hali yopilmagan")
		return
	}
	// ⚠️ **Rebuilt from the shift, not stored as text.** A Z report is the
	// figures of a closed shift, and those cannot change — so keeping a copy of
	// the paper would be a second version of the same facts, differing the
	// first time the receipt template is edited.
	figures, _, err := h.shiftFigures(r, &shift)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	data, tpl, err := h.shiftReportData(r.Context(), &shift, figures, "Z")
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"lines":   receipt.RenderShift(tpl, data),
		"widthMM": tpl.WidthMM,
	})
}
