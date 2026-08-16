package handlers

// Asking the till what became of an order we sent it.
//
// Sending was only ever half the bridge. `SendOrder` returns the moment the POS
// has filed the order, and for two of the four providers that is emphatically
// not the moment the kitchen has it:
//
//   - **Poster** files it as an *incoming order* with `status: 0` and waits for
//     somebody at the counter to press accept. There is no API parameter that
//     skips this — the whole `createIncomingOrder` parameter list is spot_id,
//     client_id, names, phone, email, sex, birthday, address, comment,
//     products, payment, promotion. Auto-accepting is simply not offered.
//   - **iiko** creates asynchronously; `SendOrder` already waits a few seconds
//     for the command to resolve, but a slow terminal group outlives that wait.
//
// Until now nothing ever asked. Every adapter implements `OrderStatus`, and not
// one line of code called it — so the panel said "kassaga yuborildi" and then
// stopped having an opinion, forever. An order could sit unaccepted on the
// till's screen through an entire service and the only place that fact existed
// was the till itself, in another room.
//
// That is the failure this file exists to make visible, and the shape of it
// governs three decisions:
//
//   - **The till's answer is a separate field** (`pos.till`), never folded into
//     `pos.status`. The handover and the acceptance are different facts on
//     different clocks; one field for both would mean either lying about the
//     kitchen or losing the record of a successful send.
//   - **Backed off, not hammered.** An order is usually accepted within a
//     minute or two, so it is asked about every minute at first and every five
//     after that. A till nobody is watching must not cost 360 API calls.
//   - **It warns, it does not ring.** The clearing action for "the cashier has
//     not accepted this" is on the *till*, not in our panel, and an alarm whose
//     silencing button does not exist is one people learn to ignore (the same
//     rule AlertBell follows for every sound it plays).

import (
	"context"
	"errors"
	"log"
	"time"

	"restaurant-backend/internal/models"
	"restaurant-backend/internal/pos"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	// How often the loop wakes. Which orders it actually asks about is decided
	// per order by tillCheckDue — the ticker is the floor, not the rate.
	posTillTick = time.Minute
	// Fresh orders are asked about on every tick: acceptance normally happens
	// while the guest is still on the tracking page, and that is the window in
	// which knowing is worth anything.
	posTillCheckFast = time.Minute
	// After this the question stops being urgent and starts being a vigil.
	posTillBackoffAfter = 10 * time.Minute
	posTillCheckSlow    = 5 * time.Minute
	// Stop asking eventually. An order unaccepted six hours after it was sent
	// is not going to be accepted, and the record of that is already written.
	posTillWatchFor = 6 * time.Hour
	// How long an order may sit unaccepted before the panel says so. Short
	// enough to catch a missed ticket during service, long enough that a
	// counter dealing with a queue is not nagged about the order it is holding
	// the phone about.
	posTillAlertAfter = 5 * time.Minute
)

// StartPOSOrderSync watches orders the till has not confirmed yet.
//
// Started from cmd/server alongside the stop-list poller and stopped with the
// process. Like that one, every failure is logged and nothing else: a till that
// is down must never take the panel with it, and the order is at the till (or
// not) regardless of whether we managed to ask.
func (h *Handler) StartPOSOrderSync(ctx context.Context) {
	go func() {
		// Offset from the stop-list poller's own 20s delay so the two do not
		// both dial somebody else's cloud in the same second on boot.
		timer := time.NewTimer(40 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		ticker := time.NewTicker(posTillTick)
		defer ticker.Stop()
		for {
			h.syncPOSOrderStates(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// syncPOSOrderStates asks about every order still awaiting a verdict.
func (h *Handler) syncPOSOrderStates(ctx context.Context) {
	now := time.Now()
	cur, err := h.Store.Orders.Find(ctx, pendingTillFilter(now))
	if err != nil {
		log.Printf("pos order status: %v", err)
		return
	}
	var orders []models.Order
	if err := cur.All(ctx, &orders); err != nil {
		log.Printf("pos order status: %v", err)
		return
	}

	// One provider per branch, not per order. iiko and Clopos exchange a token
	// on construction, so building one per order would turn a quiet branch with
	// four unaccepted tickets into four logins a minute.
	providers := map[primitive.ObjectID]pos.Provider{}
	for _, o := range orders {
		if o.POS == nil || !tillCheckDue(o.POS.Till, tillSentAt(o.POS, now), now) {
			continue
		}
		provider, ok := providers[o.BranchID]
		if !ok {
			provider, _, err = h.posFor(ctx, o.BranchID)
			if err != nil {
				log.Printf("pos order status (branch %s): %v", o.BranchID.Hex(), err)
			}
			providers[o.BranchID] = provider
		}
		if provider == nil {
			continue
		}
		h.checkOrderTill(ctx, provider, o)
	}
}

// pendingTillFilter is which orders are still worth asking about.
//
// ⚠️ **A missing `pos.till.state` must match**, and `$nin` gives that: an order
// sent before this file existed, or one never yet asked about, has no field
// there at all. This is the one place the "missing is not null" trap works in
// our favour rather than against us — spelled out because the next edit will be
// tempted to make it an explicit `$in` of the unresolved states, which would
// silently exclude every order that has never been checked.
func pendingTillFilter(now time.Time) bson.M {
	return bson.M{
		"pos.status":     models.POSSent,
		"pos.posOrderId": bson.M{"$ne": ""},
		"pos.sentAt":     bson.M{"$gte": now.Add(-posTillWatchFor)},
		"pos.till.state": bson.M{"$nin": []string{
			models.TillAccepted, models.TillCancelled, models.TillUnsupported,
		}},
		// A cancelled or delivered order has been settled by people; whatever
		// the till thinks is now history, not a question.
		"status": bson.M{"$nin": []models.OrderStatus{
			models.StatusCancelled, models.StatusDelivered,
		}},
	}
}

// checkOrderTill asks about one order and writes down the answer.
func (h *Handler) checkOrderTill(ctx context.Context, provider pos.Provider, o models.Order) {
	callCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	st, err := provider.OrderStatus(callCtx, o.POS.POSOrderID)

	till := tillUpdate(o.POS.Till, st, err, time.Now())
	if _, uerr := h.Store.Orders.UpdateByID(ctx, o.ID, bson.M{
		"$set": bson.M{"pos.till": till},
	}); uerr != nil {
		log.Printf("pos order status save (order %s): %v", o.Number, uerr)
	}
}

// tillSentAt is when the handover happened, for the backoff clock.
//
// Falls back to now when the timestamp is missing rather than to the zero time:
// a zero would read as "sent in year 1", i.e. maximally old, and put the order
// straight onto the slow schedule at the one moment the fast one matters.
func tillSentAt(p *models.OrderPOS, now time.Time) time.Time {
	if p != nil && p.SentAt != nil {
		return *p.SentAt
	}
	return now
}

// tillCheckDue decides whether this order is asked about on this tick.
//
// Pure, so the backoff can be tested without a till, a database or a clock —
// the two rules that matter (never-checked is always due, old orders slow down)
// are both the kind that look obviously right and go wrong by one comparison.
func tillCheckDue(till *models.OrderTill, sentAt, now time.Time) bool {
	if till == nil || till.CheckedAt == nil {
		return true
	}
	every := posTillCheckFast
	if now.Sub(sentAt) > posTillBackoffAfter {
		every = posTillCheckSlow
	}
	return now.Sub(*till.CheckedAt) >= every
}

// tillUpdate turns one answer into the record to store.
//
// Kept pure and separate from the call for the same reason as stoppedMenuItems:
// every branch here is a decision about what the panel will claim, and the
// wrong one is invisible until somebody is standing in a kitchen wondering why
// the screen disagrees with the till.
func tillUpdate(prev *models.OrderTill, st pos.Status, err error, now time.Time) *models.OrderTill {
	till := &models.OrderTill{CheckedAt: &now}
	if prev != nil {
		till.State, till.Raw, till.AcceptedAt = prev.State, prev.Raw, prev.AcceptedAt
	}

	switch {
	case errors.Is(err, pos.ErrUnsupported):
		// Not a failure and not a gap in the integration — this till has no
		// document to ask about. Recorded so the poller stops asking and the
		// panel can say so instead of showing a verdict that will never come.
		till.State, till.Raw, till.Error = models.TillUnsupported, "", ""
		return till
	case err != nil:
		// ⚠️ **The previous verdict is kept.** A till that is unreachable for a
		// minute has not un-accepted anything, and blanking the state would
		// turn every network blip into "waiting for the cashier" — an alarm
		// about the kitchen raised by our own connectivity.
		till.Error = clampText(err.Error(), 300)
		return till
	}

	till.Error = ""
	till.Raw = st.Raw
	switch st.State {
	case "accepted", "cooking", "ready", "closed":
		// Everything past acceptance counts as accepted: iiko answers with
		// where the order is in the kitchen, and treating "cooking" as
		// unconfirmed would keep warning about an order already being made.
		till.State = models.TillAccepted
		if till.AcceptedAt == nil {
			till.AcceptedAt = &now
		}
	case "cancelled":
		till.State = models.TillCancelled
	default:
		// "unknown" is Poster's `status: 0` and iiko's not-yet-resolved
		// command. Named honestly for the panel: nobody at the till has taken
		// it. Deliberately not left blank — blank means never asked, and the
		// difference is the entire point of the alert below.
		till.State = models.TillWaiting
	}
	return till
}

// failedPOSFilter is the harder failure: the order never reached the till at
// all, so the kitchen has no ticket and does not know it is missing one.
//
// ⚠️ **The most likely cause is one unmapped dish**, and it is invisible from
// both ends. `pos.CheckMapped` refuses the whole order by design — half a
// ticket is worse than none — but sending happens in the background on confirm,
// so the operator sees the order turn green and moves on. The guest is waiting,
// the panel looks healthy, and the kitchen is not cooking. Nothing anywhere
// said so until this count existed.
//
// No time bound, unlike the till check: an order that failed an hour ago and is
// still open is *more* worth showing, not less. Settled orders drop out on
// their own through the status filter.
func failedPOSFilter() bson.M {
	return bson.M{
		"pos.status": models.POSFailed,
		"status": bson.M{"$nin": []models.OrderStatus{
			models.StatusCancelled, models.StatusDelivered,
		}},
	}
}

// unacceptedTillFilter is the panel's warning: handed over, the till says a
// human still has not taken it, and it has been long enough to matter.
//
// ⚠️ **`TillWaiting` exactly, not "anything unresolved".** An order we have
// never managed to ask about, and one on a till that cannot answer, must not
// raise this — the warning claims something specific about a person at a
// counter, and claiming it from our own silence is how a panel teaches its
// owner to distrust warnings.
func unacceptedTillFilter(now time.Time) bson.M {
	return bson.M{
		"pos.status":     models.POSSent,
		"pos.till.state": models.TillWaiting,
		"pos.sentAt":     bson.M{"$lte": now.Add(-posTillAlertAfter)},
		"status": bson.M{"$nin": []models.OrderStatus{
			models.StatusCancelled, models.StatusDelivered,
		}},
	}
}
