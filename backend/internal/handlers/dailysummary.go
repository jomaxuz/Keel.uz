package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/i18n"
	"restaurant-backend/internal/models"
)

// ---- The day's totals, once, when the till is closed ----
//
// ⚠️ **The trigger is the close, not a clock.** A summary at a fixed hour is
// wrong for the restaurant that is still serving at one in the morning and
// wrong again for the one that shut at nine — and it would be sent while the
// drawer is still open, which means the figures move after the message. The
// close is the restaurant's own statement that the day is over, and it is the
// moment the numbers stop changing.
//
// ⚠️ **One message per day, not one per register.** A place with two tills
// closes them minutes apart; the summary is sent by whichever close leaves no
// open shift behind, so the owner is told the day's figure rather than half of
// it twice.
//
// ⚠️ **Push only.** The loss alerts go to Telegram as well, because they are a
// record somebody may need to search a month later. This is the opposite kind
// of message: it is true for one evening, it is read once, and a group chat
// that receives it every night is a group chat people leave.

// summariseDay tells the owner how the day went, once the last till is closed.
func (h *Handler) summariseDay(ctx context.Context, branchID primitive.ObjectID) {
	// ⚠️ Detached from the request that closed the shift: the cashier is
	// waiting for the drawer count to save, and a push relay in another country
	// must never be why that takes four seconds.
	go func() {
		c, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()

		// Another register still open means the day is not over yet.
		open := bson.M{"closedAt": bson.M{"$exists": false}}
		if !branchID.IsZero() {
			open["branchId"] = branchID
		}
		if n, err := h.Store.CashShifts.CountDocuments(c, open); err != nil || n > 0 {
			return
		}

		// ⚠️ **The local day, and the numbers are the day's rather than the
		// shift's.** A shift that opened at four in the afternoon and one that
		// opened at eight in the morning both close at the end of the same day,
		// and an owner asking "how did today go" means the day.
		now := time.Now().In(time.Local)
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		filter := bson.M{"createdAt": bson.M{"$gte": start, "$lt": now}}
		if !branchID.IsZero() {
			filter["branchId"] = branchID
		}
		cur, err := h.Store.Orders.Find(c, filter)
		if err != nil {
			log.Printf("daily summary: %v", err)
			return
		}
		var orders []models.Order
		if err := cur.All(c, &orders); err != nil {
			return
		}

		revenue, checks := 0, 0
		for _, o := range orders {
			// ⚠️ The same test the dashboard uses, rather than a second one
			// written here. Money is counted when it arrives — a debt walked
			// out with the food and is not takings — and two definitions of
			// revenue in one product is the pair that eventually disagrees in
			// front of the person who pays us.
			if received(o) {
				revenue += o.Total
				checks++
			}
		}
		// Nothing sold, nothing to say. A restaurant that was closed today does
		// not need to be told so at midnight.
		if checks == 0 {
			return
		}
		average := revenue / checks

		h.notifyAdmins(branchID, true, func(lang string) (string, string) {
			return i18n.Localize(lang, "Kunlik yakun"),
				i18n.Localize(lang, fmt.Sprintf(
					"Tushum %d so'm · %d ta chek · o'rtacha %d so'm",
					revenue, checks, average))
		}, map[string]any{"type": "summary"})
	}()
}
