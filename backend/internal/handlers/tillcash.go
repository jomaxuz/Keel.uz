package handlers

import (
	"errors"
	"net/http"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

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
	httpx.JSON(w, http.StatusOK, map[string]any{
		"shift": saved, "figures": figures, "fiscalNote": fiscalNote,
	})
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
