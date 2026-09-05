package handlers

// ---- The work nobody has picked up ----
//
// ⚠️ **The attention screen had two halves and only one of them buzzed.** The
// losses — a discount after the bill, a till short at the close — reach the
// owner's phone the evening they happen. The queue beside them did not: an
// order nobody accepted, a pre-order falling due with nothing started, a
// booking nobody answered. Those are computed when somebody opens the screen,
// which means the alert for "nobody is looking" required somebody to look.
//
// ⚠️ **Said once per thing, not once per tick.** These are states, not events:
// an order unaccepted at eleven is still unaccepted at midnight. A watcher that
// re-read the condition every five minutes would buzz every five minutes, and
// the second message is what gets an application muted — after which it misses
// the one that mattered. Every branch below stamps the document before it
// sends, exactly as `checkOpenShifts` does.
//
// ⚠️ **Not gated on the loss-alert switch.** That switch is opt-in because
// those messages are about people, and an owner should choose to watch their
// staff. This is about work: the same category as the new-order notification,
// which has never asked permission. An owner who has turned notifications off
// on the phone has already answered the question.
//
// ⚠️ **Managers hear these too** (`ownersOnly: false`), which is the opposite
// of the loss channel and for the same reason: a manager is who *acts* on an
// unaccepted order, and is not who an unaccepted order is about.

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/models"
)

// How often the queue is looked at.
//
// ⚠️ **Five minutes, not one.** The thing being measured is a person walking to
// a screen; a tighter loop would not change when anybody acts, and every tick
// is four queries against every tenant. It also bounds the worst case honestly:
// the message says an order has waited ten minutes, and it may have waited
// fifteen.
const queueWatchEvery = 5 * time.Minute

// How long an order may sit unaccepted before the phone is told.
//
// ⚠️ **Ten minutes, and it is deliberately not a setting.** The panel already
// rings the moment one arrives; this is the second line of defence for the
// hours when nobody has the panel open, and a threshold nobody has configured
// is the case that has to work. An owner who wants silence has the phone's own
// switch — which is the honest control, because it also stops everything else.
const orderWaitBefore = 10 * time.Minute

// How long a booking may go unanswered. Longer than an order because nobody is
// standing over it: a table for Saturday can wait half an hour for an answer,
// and a guest who ordered food cannot.
const bookingWaitBefore = 30 * time.Minute

// StartQueueWatch tells the owner about work that has been standing still.
func (h *Handler) StartQueueWatch(ctx context.Context) {
	go func() {
		// Two minutes in, for the reason the shift watch waits ninety seconds:
		// a restart during service must not put this in front of the first
		// request of the evening.
		timer := time.NewTimer(2 * time.Minute)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		ticker := time.NewTicker(queueWatchEvery)
		defer ticker.Stop()
		for {
			h.checkQueue(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (h *Handler) checkQueue(ctx context.Context) {
	now := time.Now()
	// ⚠️ **A till check is not an order in this sense.** An open check on table
	// six is a waiter's work in progress, not something waiting to be accepted
	// — the exact confusion that once made the panel ring for every table
	// somebody sat down at. The same `check` exclusion the bell learned.
	noTill := bson.M{"$exists": false}

	// 1. A delivery or pickup order nobody has accepted.
	//
	// ⚠️ `paymentStatus != pending`: an order still waiting on a bank is not
	// the kitchen's yet, and telling the owner it is unaccepted would be
	// telling them to accept something that may never be paid for.
	h.nudgeOrders(ctx, bson.M{
		"status":        string(models.StatusPending),
		"paymentStatus": bson.M{"$ne": models.PayPending},
		"check":         noTill,
		"queuedAt":      bson.M{"$exists": true, "$lte": now.Add(-orderWaitBefore)},
		"nudgedAt":      nil,
	}, func(o models.Order) func(string) (string, string) {
		return adminText("Buyurtma kutmoqda",
			fmt.Sprintf("#%s · %d so'm", o.Number, o.Total))
	})

	// 2. A pre-order whose hour has come, accepted, and nothing started.
	//
	// ⚠️ **Its own message, not folded into the one above.** "Somebody ordered
	// and nobody answered" and "the food you promised for seven is not being
	// cooked" are different failures with different clearing acts, and one
	// sentence covering both would be a sentence that names neither.
	//
	// ⚠️ **No grace period here.** The ten minutes above exist because the
	// order has just arrived and somebody may be reaching for it; a pre-order
	// has been known about for hours and its time is now.
	h.nudgeOrders(ctx, bson.M{
		"status":      string(models.StatusConfirmed),
		"check":       noTill,
		"scheduledAt": bson.M{"$ne": nil},
		"queuedAt":    bson.M{"$exists": true, "$lte": now},
		"nudgedAt":    nil,
	}, func(o models.Order) func(string) (string, string) {
		return adminText("Oldindan buyurtma vaqti keldi",
			fmt.Sprintf("#%s · %d so'm", o.Number, o.Total))
	})

	// 3. A booking nobody has confirmed.
	h.nudgeBookings(ctx, now)
}

// nudgeOrders sends one message per matching order and stamps it.
//
// ⚠️ **The caller hands over a finished `adminText`, not a title and a body.**
// The i18n test reads the syntax tree for literals passed to `adminText`, so a
// sentence assembled anywhere else is a sentence it cannot see — and an
// untranslated notification fails silently, as an Uzbek line arriving on a
// Russian phone and being swiped away. Two of these were invisible that way
// until the test was pointed at them.
func (h *Handler) nudgeOrders(
	ctx context.Context, filter bson.M,
	text func(models.Order) func(lang string) (title, body string),
) {
	// ⚠️ **Bounded.** A tenant coming back after a long outage can match
	// hundreds at once, and a hundred notifications arriving together is
	// indistinguishable from a broken application. The rest are stamped by the
	// following ticks, oldest first.
	cur, err := h.Store.Orders.Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "queuedAt", Value: 1}}).SetLimit(20))
	if err != nil {
		log.Printf("queue watch: %v", err)
		return
	}
	var orders []models.Order
	if err := cur.All(ctx, &orders); err != nil {
		log.Printf("queue watch: %v", err)
		return
	}
	now := time.Now()
	for _, o := range orders {
		// ⚠️ **Stamped before the send, never after.** The send is a network
		// round trip in its own goroutine; a crash between the two would leave
		// the order eligible again on the next tick, which is the repeating
		// message this whole file exists to avoid.
		if _, err := h.Store.Orders.UpdateByID(ctx, o.ID,
			bson.M{"$set": bson.M{"nudgedAt": now}}); err != nil {
			log.Printf("queue watch: %v", err)
			continue
		}
		h.notifyAdmins(o.BranchID, false, text(o),
			map[string]any{"type": "order", "orderId": o.ID.Hex(), "tab": "orders"})
	}
}

func (h *Handler) nudgeBookings(ctx context.Context, now time.Time) {
	cur, err := h.Store.Reservations.Find(ctx, bson.M{
		"status":    string(models.ReservationPending),
		"createdAt": bson.M{"$lte": now.Add(-bookingWaitBefore)},
		"nudgedAt":  nil,
		// ⚠️ **Only bookings still ahead of us.** A table asked for last
		// Tuesday that nobody ever answered is a fact about the past; a phone
		// buzzing about it teaches the owner that this channel reports things
		// they cannot act on.
		"at": bson.M{"$gte": now},
	}, options.Find().SetSort(bson.D{{Key: "at", Value: 1}}).SetLimit(20))
	if err != nil {
		log.Printf("queue watch: %v", err)
		return
	}
	var books []models.Reservation
	if err := cur.All(ctx, &books); err != nil {
		log.Printf("queue watch: %v", err)
		return
	}
	for _, b := range books {
		if _, err := h.Store.Reservations.UpdateByID(ctx, b.ID,
			bson.M{"$set": bson.M{"nudgedAt": now}}); err != nil {
			log.Printf("queue watch: %v", err)
			continue
		}
		h.notifyAdmins(b.BranchID, false, adminText("Bron javobsiz",
			fmt.Sprintf("#%s · %s", b.Number, b.At.In(time.Local).Format("02.01 15:04"))),
			map[string]any{"type": "reservation", "reservationId": b.ID.Hex()})
	}
}
