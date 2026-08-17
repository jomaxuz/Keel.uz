package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"restaurant-backend/internal/fiscal"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Ending the register's day, and tying it to the drawer count.
//
// # Two shifts, and they are not the same shift
//
// ⚠️ The cash shift is **ours**: a manager opens it, counts the drawer and
// closes it, and it can happen twice in one day when staff change over. The
// fiscal day is the **register's**: it is opened by the first sale (see the
// #2D handling in tillfiscal.go) and ended by a Z-report, which is a tax
// document totalling everything filed since.
//
// Conflating them would break in both directions. Closing the fiscal day at
// every staff handover would split one tax day into two and file a Z-report at
// four in the afternoon; leaving it to the register alone would mean the
// drawer count never has the register's own figures beside it.
//
// So the link is deliberately loose and one-directional: **closing the cash
// shift asks for the day to end**, and the answer is written back onto that
// shift when it arrives.
//
// # ⚠️ Why it is a request and not a call
//
// The cash shift is closed from the panel, which is very often a laptop at
// home. The register is a PC on the restaurant's network. The panel cannot
// reach it and never will — so the close is recorded as a request, and whoever
// *can* reach the register carries it out: the relay, or the till screen the
// next time somebody opens it.
//
// The ordering falls out for free. The relay asks for filings first and only
// then for the day-end, so a Z-report can never overtake a receipt that has not
// been filed yet.

// ErrUnfiledReceipts is why a day cannot be ended.
var errUnfiledReceipts = errors.New(
	"fiskallashtirilmagan cheklar bor — avval ularni yuboring")

// requestCloseDay records that the register's day should end.
//
// ⚠️ **Refused while any sale is still unfiled**, and this is the guard the
// whole file exists for. A Z-report totals the day and hands that total to the
// state; a receipt filed after it belongs to the *next* day. Ending the day
// with receipts outstanding either loses them or moves them onto tomorrow's
// figures, and neither is something a restaurant can correct afterwards.
//
// Returns false when there is nothing to do — no register, or no adapter —
// which is the ordinary case and not an error.
func (h *Handler) requestCloseDay(
	ctx context.Context, branchID primitive.ObjectID, by string, shiftID primitive.ObjectID,
) (bool, error) {
	set, enc, _ := h.tillFiscal(ctx, branchID)
	if enc == nil || !set.Enabled {
		return false, nil
	}
	if _, ok := enc.(fiscal.ShiftCloser); !ok {
		// This register ends its own day. Nothing to ask for, and asking would
		// leave a request nothing ever clears.
		return false, nil
	}

	// ⚠️ anyUnfiledFilter, **not** the alert's — no grace period. A receipt still
	// in flight when the day is totalled is precisely the one at risk, and the
	// last sales of the evening are exactly the ones that would be inside a
	// five-minute window at closing time.
	n, err := h.Store.Orders.CountDocuments(ctx, withBranch(anyUnfiledFilter(), branchID))
	if err != nil {
		return false, err
	}
	if n > 0 {
		return false, errUnfiledReceipts
	}

	now := time.Now()
	_, err = h.Store.FiscalSettings.UpdateOne(ctx,
		bson.M{"branchId": branchID},
		bson.M{"$set": bson.M{
			"closeDayRequestedAt": now,
			"closeDayBy":          by,
			"closeDayShiftId":     shiftID,
		}},
		options.Update().SetUpsert(true))
	return err == nil, err
}

// withBranch scopes a filter to one branch.
//
// ⚠️ Copies rather than mutating: unfiledFiscalFilter is shared with the alert
// and the till list, and a filter that quietly acquired a branch from whoever
// called it last is the kind of bug that shows one restaurant another's counts.
func withBranch(filter bson.M, branchID primitive.ObjectID) bson.M {
	out := bson.M{"branchId": branchID}
	for k, v := range filter {
		out[k] = v
	}
	return out
}

// closeDayJob builds the Z-report call, when one has been asked for.
func closeDayJob(set *models.FiscalSettings, enc fiscal.Encoder) (tillFiscalJob, bool) {
	if set.CloseDayRequestedAt == nil {
		return tillFiscalJob{}, false
	}
	closer, ok := enc.(fiscal.ShiftCloser)
	if !ok {
		return tillFiscalJob{}, false
	}
	req, err := closer.CloseShift(set.CloseDayBy, time.Now())
	if err != nil {
		return tillFiscalJob{}, false
	}
	return tillFiscalJob{}.withBase(agentBase(set), req), true
}

// recordCloseDay writes what the register said about the day.
//
// ⚠️ **The request is cleared whatever the answer was**, success or refusal.
// A request that survived a failure would have the relay filing a Z-report on
// every poll for as long as the condition lasted — and a Z-report is not an
// operation to retry in a loop. The failure is recorded on the shift instead,
// where somebody will read it.
func (h *Handler) recordCloseDay(
	ctx context.Context, set *models.FiscalSettings, enc fiscal.Encoder,
	reply fiscalReplyRequest,
) *models.FiscalDay {
	now := time.Now()
	day := models.FiscalDay{ClosedAt: now}

	if err := replyFailure(reply); err != nil {
		day.Error = clampText(err.Error(), 300)
	} else if z, ok := fiscal.ParseZReport([]byte(reply.Body)); ok {
		day.Number = z.Number
		day.SaleCash, day.SaleCard = z.SaleCash, z.SaleCard
		day.SaleTotal, day.SaleCount = z.SaleTotal, z.SaleCount
		day.RefundTotal = z.RefundTotal
	} else {
		// Not a Z-report. Usually the register refusing — "#2B: no operations
		// since the day was opened", which is what a day with no sales answers.
		// ⚠️ Reported rather than swallowed: an empty day and a failed Z-report
		// look identical in the totals and mean opposite things.
		_, perr := enc.Parse(reply.Status, []byte(reply.Body))
		if perr != nil {
			day.Error = clampText(perr.Error(), 300)
		} else {
			day.Error = "kassa kun yakunini qaytarmadi"
		}
	}

	if !set.CloseDayShiftID.IsZero() {
		_, _ = h.Store.CashShifts.UpdateOne(ctx,
			bson.M{"_id": set.CloseDayShiftID},
			bson.M{"$set": bson.M{"fiscal": day, "updatedAt": now}})
	}
	_, _ = h.Store.FiscalSettings.UpdateOne(ctx,
		bson.M{"branchId": set.BranchID},
		bson.M{"$unset": bson.M{
			"closeDayRequestedAt": "", "closeDayBy": "", "closeDayShiftId": "",
		}})
	return &day
}

// StaffCloseFiscalDay is the till screen ending the register's day.
//
// Two uses, and both are real: the cashier doing it deliberately at closing
// time, and the till picking up a request the panel made when the drawer was
// counted. Either way the call has to be made from here, because this is where
// the register is reachable.
func (h *Handler) StaffCloseFiscalDay(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermCashier)
	if !ok {
		return
	}
	set, enc, _ := h.tillFiscal(r.Context(), s.BranchID)
	if enc == nil {
		httpx.Error(w, http.StatusBadRequest, "fiskal kassa ulanmagan")
		return
	}

	// Asked for here rather than assumed: a cashier pressing this at eleven at
	// night must meet the same guard the panel does.
	if set.CloseDayRequestedAt == nil {
		if _, err := h.requestCloseDay(r.Context(), s.BranchID, s.Name, h.openShiftID(r, s.BranchID)); err != nil {
			httpx.Error(w, http.StatusConflict, err.Error())
			return
		}
		set = h.fiscalSettingsOf(r.Context(), s.BranchID)
	}
	if agentUsable(set) {
		// The relay will take it on its next poll. Handing the job out here as
		// well would file two Z-reports for one day.
		httpx.JSON(w, http.StatusOK, map[string]any{"queued": true})
		return
	}
	job, ok := closeDayJob(set, enc)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "bu kassa kun yakunini qo'llab-quvvatlamaydi")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"job": job})
}

// StaffCloseFiscalDayResult records the Z-report the till just obtained.
func (h *Handler) StaffCloseFiscalDayResult(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermCashier)
	if !ok {
		return
	}
	var req fiscalReplyRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set, enc, _ := h.tillFiscal(r.Context(), s.BranchID)
	if enc == nil {
		httpx.Error(w, http.StatusBadRequest, "fiskal kassa ulanmagan")
		return
	}
	httpx.JSON(w, http.StatusOK, h.recordCloseDay(r.Context(), set, enc, req))
}

// openShiftID is the cash shift a Z-report should be filed against, if there is
// one open. Zero is fine: a register's day can be ended without our drawer
// being open, and the report is then simply not attached to one.
func (h *Handler) openShiftID(r *http.Request, branchID primitive.ObjectID) primitive.ObjectID {
	var shift models.CashShift
	if err := h.Store.CashShifts.FindOne(r.Context(), bson.M{
		"branchId": branchID,
		"closedAt": bson.M{"$exists": false},
	}).Decode(&shift); err != nil {
		return primitive.NilObjectID
	}
	return shift.ID
}
