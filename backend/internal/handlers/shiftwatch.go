package handlers

// ---- A drawer nobody has counted for days ----
//
// ⚠️ **The failure here is that nothing looks wrong.** A till with an open shift
// behaves exactly like a till working normally: checks open, money is taken,
// receipts print. What quietly stops being true is the one number the whole cash
// module exists to produce. "What should be in the drawer" becomes the sum of
// four evenings, so the variance at the eventual close cannot be attributed to a
// shift, a person or a day — and by then it is a figure somebody is asked to
// explain about a week they cannot remember.
//
// ⚠️ **It is not closed automatically, and that is the important decision.**
// Closing a shift writes a *counted* figure and the variance against it. An
// automatic close would have to either invent a count — declaring money the
// restaurant may not have — or file a close with none, which destroys the only
// measurement the shift was keeping. The same reasoning `saveStocktake` follows:
// a count is a measurement, and a measurement nobody made must not be written.
//
// So the shift stays open, every screen that shows it says how long it has been
// open, and the owner is told once when it crosses the line. Telling somebody is
// the only action here that cannot make the figures worse.

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/models"
)

// How often the open shifts are looked at. ⚠️ Hourly rather than by the minute:
// the thing being watched is measured in days, and the message says how many
// hours it has been — a tighter loop would change nothing except the load.
const shiftWatchEvery = time.Hour

// StartShiftWatch tells the owner about a cash shift that has stopped meaning
// anything.
func (h *Handler) StartShiftWatch(ctx context.Context) {
	go func() {
		// A minute in, so a restart during service does not put this in front
		// of the first request of the morning.
		timer := time.NewTimer(90 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		ticker := time.NewTicker(shiftWatchEvery)
		defer ticker.Stop()
		for {
			h.checkOpenShifts(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (h *Handler) checkOpenShifts(ctx context.Context) {
	cur, err := h.Store.CashShifts.Find(ctx, bson.M{
		"closedAt": nil,
		// ⚠️ Only the ones nobody has been told about yet. The state persists
		// for days; a message every hour is the noise models/alert.go opens by
		// warning about, and the second one is what gets the channel muted.
		"overdueAt": nil,
	}, options.Find().SetLimit(200))
	if err != nil {
		log.Printf("shift watch: %v", err)
		return
	}
	var shifts []models.CashShift
	if err := cur.All(ctx, &shifts); err != nil {
		log.Printf("shift watch: %v", err)
		return
	}

	now := time.Now()
	for _, s := range shifts {
		set := h.alertSettingsOf(ctx, s.BranchID)
		if int(now.Sub(s.OpenedAt).Hours()) < set.ShiftMaxHours {
			continue
		}
		// ⚠️ **Stamped before the alert is raised, not after.** Raising it is a
		// network round trip in its own goroutine; a crash in between would
		// otherwise leave the shift eligible again on the next tick, and the
		// owner would get the same message every hour until somebody closed the
		// drawer.
		if _, err := h.Store.CashShifts.UpdateByID(ctx, s.ID,
			bson.M{"$set": bson.M{"overdueAt": now}}); err != nil {
			log.Printf("shift watch: %v", err)
			continue
		}
		h.raiseAlert(models.LossAlert{
			BranchID: s.BranchID,
			Kind:     models.AlertShiftOverdue,
			At:       now,
			By:       s.OpenedBy,
			// ⚠️ **Hours, not money.** Nothing has been lost yet and saying a
			// figure would read as an accusation about one; what has been lost
			// is the ability to check, and the number that says so is time.
			Amount:  int(now.Sub(s.OpenedAt).Hours()),
			Subject: s.OpenedAt.In(time.Local).Format("02.01 15:04"),
			RefID:   s.ID,
		})
	}
}

// ShiftAge is how long a shift has been open and whether that is too long.
//
// ⚠️ **Computed on every read rather than stored.** A saved "overdue" flag goes
// stale the moment the clock passes it — the lesson `provisionStatus` and
// `IsLimitSoldOut` each taught this codebase — and the screens that read this
// are the ones somebody opens to decide whether to go and count the drawer.
type ShiftAge struct {
	Hours   int  `json:"hours"`
	Overdue bool `json:"overdue"`
	// The line this branch was measured against, so a screen can say "18 soat"
	// rather than only "too long".
	MaxHours int `json:"maxHours"`
}

func (h *Handler) shiftAge(ctx context.Context, s *models.CashShift) *ShiftAge {
	if s == nil || s.ClosedAt != nil || s.OpenedAt.IsZero() {
		return nil
	}
	max := h.alertSettingsOf(ctx, s.BranchID).ShiftMaxHours
	hours := int(time.Since(s.OpenedAt).Hours())
	return &ShiftAge{Hours: hours, Overdue: hours >= max, MaxHours: max}
}
