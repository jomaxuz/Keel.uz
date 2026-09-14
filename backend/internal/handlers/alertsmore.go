package handlers

// ---- The money that leaves by the doors the first eight kinds did not watch ----
//
// A refund on a closed check, a phone order cancelled after the kitchen made
// it, a large withdrawal from the drawer, a large write-off and a large sum on
// the slate. Each one was already written down — a refund record, a cancel
// reason, a drawer entry, a write-off document, a debt — and each one reached
// the owner only if the owner went looking.
//
// ⚠️ **Same rules as every other kind** (models/alert.go): a single unusual
// event, above a threshold the restaurant sets, said as a fact and never as an
// accusation, recorded whether or not it is sent.
//
// ⚠️ **Not about the owner.** A refund, a write-off or a cancellation the owner
// makes in the panel is not news to the owner — the same rule the panel-action
// kind follows — and a channel that reports its reader to themselves is muted.

import (
	"context"

	"restaurant-backend/internal/models"
)

// alertOnRefund tells the owner money was handed back on a closed check.
func (h *Handler) alertOnRefund(o *models.Order, refund models.CheckRefund, byOwner bool) {
	if o == nil || byOwner {
		return
	}
	set := h.alertSettingsOf(context.Background(), o.BranchID)
	if !set.Enabled || refund.Amount < set.RefundFrom {
		return
	}
	h.raiseAlert(models.LossAlert{
		BranchID: o.BranchID,
		Kind:     models.AlertCheckRefunded,
		At:       refund.At,
		By:       refund.By,
		Amount:   refund.Amount,
		Reason:   refund.Reason,
		Number:   o.Number,
		Table:    o.TableNumber,
		RefID:    o.ID,
	})
}

// kitchenStarted reports whether an order had reached the kitchen before it was
// cancelled: it went through preparing or on the way, or was marked ready.
//
// ⚠️ **Read from the history, not the status**, because by the time this runs
// the status is already "cancelled" — the only thing that still says where the
// order had got to is the record of how it moved.
func kitchenStarted(o *models.Order) bool {
	if o.ReadyAt != nil {
		return true
	}
	for _, ev := range o.StatusHistory {
		if ev.Status == models.StatusPreparing || ev.Status == models.StatusOnTheWay {
			return true
		}
	}
	return false
}

// alertOnOrderCancelled tells the owner an order was cancelled after the
// kitchen had started on it.
//
// ⚠️ **Not a till check** — those have their own kind, with the cooked value of
// the lines rather than the order total. And **only once cooking had started**:
// an order cancelled while still pending is a guest who changed their mind, and
// alerting on it would put the commonest cancellation on somebody's phone
// several times a day.
func (h *Handler) alertOnOrderCancelled(o *models.Order, by string, byOwner bool) {
	if o == nil || byOwner || o.Check != nil || o.Total <= 0 || !kitchenStarted(o) {
		return
	}
	set := h.alertSettingsOf(context.Background(), o.BranchID)
	if !set.Enabled {
		return
	}
	h.raiseAlert(models.LossAlert{
		BranchID: o.BranchID,
		Kind:     models.AlertOrderCancelled,
		By:       by,
		Amount:   o.Total,
		Reason:   o.CancelReason,
		Number:   o.Number,
		RefID:    o.ID,
	})
}

// alertOnCashOut tells the owner about a large withdrawal from the drawer.
//
// ⚠️ **The category is the subject**, because it is the answer most of the
// time: "Non yetkazib beruvchiga", "Kuryer ish haqi". A large withdrawal with
// its reason beside it is usually explained before the owner finishes reading.
func (h *Handler) alertOnCashOut(entry models.CashEntry) {
	if entry.Kind != models.CashOut {
		return
	}
	set := h.alertSettingsOf(context.Background(), entry.BranchID)
	if !set.Enabled || entry.Amount < set.CashOutFrom {
		return
	}
	h.raiseAlert(models.LossAlert{
		BranchID: entry.BranchID,
		Kind:     models.AlertCashOut,
		At:       entry.At,
		ByID:     entry.ByID,
		By:       entry.By,
		Amount:   entry.Amount,
		Subject:  entry.Category,
		Reason:   entry.Note,
		RefID:    entry.ID,
	})
}

// alertOnWriteOff tells the owner stock worth a lot was written off.
func (h *Handler) alertOnWriteOff(wo models.WriteOff, ingredient string, byOwner bool) {
	if byOwner {
		return
	}
	set := h.alertSettingsOf(context.Background(), wo.BranchID)
	if !set.Enabled || wo.Value < set.WriteoffFrom {
		return
	}
	h.raiseAlert(models.LossAlert{
		BranchID: wo.BranchID,
		Kind:     models.AlertBigWriteoff,
		By:       wo.By,
		Amount:   wo.Value,
		Subject:  ingredient,
		Reason:   wo.Reason,
		RefID:    wo.ID,
	})
}

// alertOnDebt tells the owner a large sum was put on somebody's slate.
func (h *Handler) alertOnDebt(o *models.Order, set models.AlertSettings, by, note string) {
	if o == nil || !set.Enabled || o.Total < set.DebtFrom {
		return
	}
	h.raiseAlert(models.LossAlert{
		BranchID: o.BranchID,
		Kind:     models.AlertDebtWritten,
		By:       by,
		Amount:   o.Total,
		Reason:   clampText(note, 200),
		Number:   o.Number,
		Table:    o.TableNumber,
		RefID:    o.ID,
	})
}
