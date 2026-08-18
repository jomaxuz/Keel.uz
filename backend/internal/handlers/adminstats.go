package handlers

import (
	"errors"
	"maps"
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Dashboard numbers for a period the operator chooses.
//
// Everything here is counted in Mongo rather than in the browser: the panel used
// to fetch the last 200 orders and add them up client-side, which quietly became
// wrong as soon as a restaurant had more than 200 orders — and could never
// answer "how many customers do we have" at all.

var (
	errBadDate  = errors.New("sana formati: YYYY-MM-DD")
	errBadRange = errors.New("oxirgi sana boshlanishidan oldin")
)

type statsPeriod struct {
	Orders int `json:"orders"`
	// Money actually in hand: cash handed over on delivery, or a card payment
	// the bank confirmed. **Not** every order that was placed — see received().
	Revenue int `json:"revenue"`
	// Placed, not cancelled, and not yet collected: the food in the kitchen and
	// on the road. Worth its own line — an owner does want to know what today
	// is still going to bring in — but it is not takings, and calling it that
	// is what this used to do.
	Pending  int `json:"pending"`
	AvgOrder int `json:"avgOrder"`
	// How many orders the revenue above came from. Shown so the average can be
	// checked, and so "12 orders, 3 collected" is visible rather than implied.
	Paid        int            `json:"paid"`
	Delivery    int            `json:"delivery"`
	Pickup      int            `json:"pickup"`
	DineIn      int            `json:"dineIn"`
	Cancelled   int            `json:"cancelled"`
	Delivered   int            `json:"delivered"`
	ByStatus    map[string]int `json:"byStatus"`
	DeliveryFee int            `json:"deliveryFee"`
	CashTotal   int            `json:"cashTotal"`
}

// received reports whether the money for an order is actually in the
// restaurant's hands.
//
// The dashboard used to count every order that was not cancelled, which meant
// a 100 000 so'm order placed a minute ago — nobody has cooked it, no courier
// has left, nobody has paid anything — appeared as takings immediately. The
// number was always right eventually and wrong all day.
//
// Two ways money actually arrives, and the order carries both:
//
//   - **the bank said so** (`paymentStatus: paid`) — a card payment is real
//     before the food moves, and stays real if delivery is still an hour away;
//   - **the courier came back with it** (`delivered`) — which is what cash on
//     delivery means, and is also the terminal state for pickup and dine-in.
//
// A cancelled order is never counted, even if it was paid: that money is owed
// back, and the refund moves `paymentStatus` off `paid`. Counting it would
// book takings the restaurant is about to hand over again.
func received(o models.Order) bool {
	if o.Status == models.StatusCancelled {
		return false
	}
	// ⚠️ **Refunded money is not takings, and the `delivered` half of this
	// test would otherwise keep counting it.** For a delivery the refund does
	// move `paymentStatus` off `paid` and the order drops out — which is what
	// the comment above assumed for every case. A dining-room sale is
	// `delivered` the moment it is closed, so a refunded table went on being
	// counted as revenue with nothing on any screen disagreeing: the guest has
	// the cash back, the drawer is short by it, and the dashboard is not.
	if o.PaymentStatus == models.PayRefunded {
		return false
	}
	return o.PaymentStatus == models.PayPaid || o.Status == models.StatusDelivered
}

type topDish struct {
	Name  string `json:"name"`
	Qty   int    `json:"qty"`
	Total int    `json:"total"`
}

// AdminStats answers the dashboard: order and money totals for the chosen
// period, plus the standing counts (customers, couriers, admins, menu) that do
// not depend on it.
//
// Period comes as ?from=&to= (RFC3339 or YYYY-MM-DD, `to` inclusive). Without
// them everything is counted from the beginning, which is what "all time" means.
func (h *Handler) AdminStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	from, to, err := parseRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	created := bson.M{}
	if from != nil {
		created["$gte"] = *from
	}
	if to != nil {
		created["$lt"] = *to
	}
	periodFilter := bson.M{}
	if len(created) > 0 {
		periodFilter["createdAt"] = created
	}

	// Everything countable is read through the panel's brand/branch lens.
	// Customers and panel accounts are not: they belong to the company, and a
	// samsa branch manager still shares the same customer base.
	branchScope, scope, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	orderFilter := bson.M{}
	for k, v := range periodFilter {
		orderFilter[k] = v
	}
	for k, v := range branchScope {
		orderFilter[k] = v
	}

	period := statsPeriod{ByStatus: map[string]int{}}
	cur, err := h.Store.Orders.Find(ctx, orderFilter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Customers who ordered in the period — "active", as opposed to registered.
	activeUsers := map[string]struct{}{}
	dishes := map[string]*topDish{}
	var orders []models.Order
	if err := cur.All(ctx, &orders); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, o := range orders {
		period.Orders++
		period.ByStatus[string(o.Status)]++
		switch o.Status {
		case models.StatusCancelled:
			period.Cancelled++
		case models.StatusDelivered:
			period.Delivered++
		}
		// Money is counted when it arrives, not when an order is placed.
		if received(o) {
			period.Revenue += o.Total
			period.Paid++
			period.DeliveryFee += o.DeliveryFee
			if o.PaymentMethod == "cash" {
				period.CashTotal += o.Total
			}
		} else if o.Status != models.StatusCancelled {
			period.Pending += o.Total
		}
		// Dishes are counted on a different basis on purpose: this list
		// answers "what sells", and a dish in a confirmed order has sold —
		// the money simply has not been handed over yet. Only a cancellation
		// un-sells it.
		if o.Status != models.StatusCancelled {
			for _, it := range o.Items {
				d, ok := dishes[it.Name]
				if !ok {
					d = &topDish{Name: it.Name}
					dishes[it.Name] = d
				}
				d.Qty += it.Qty
				d.Total += it.Price * it.Qty
			}
		}
		switch o.Type {
		case "pickup":
			period.Pickup++
		case "dinein":
			period.DineIn++
		default:
			period.Delivery++
		}
		if !o.UserID.IsZero() {
			activeUsers[o.UserID.Hex()] = struct{}{}
		}
	}
	// The average of what was collected, over the orders it was collected from.
	// Dividing takings by every non-cancelled order mixes two bases and drags
	// the figure down every time the kitchen is busy.
	if period.Paid > 0 {
		period.AvgOrder = period.Revenue / period.Paid
	}

	// Top dishes of the period, best sellers first.
	top := make([]topDish, 0, len(dishes))
	for _, d := range dishes {
		top = append(top, *d)
	}
	for i := 1; i < len(top); i++ {
		for j := i; j > 0 && top[j].Qty > top[j-1].Qty; j-- {
			top[j], top[j-1] = top[j-1], top[j]
		}
	}
	if len(top) > 8 {
		top = top[:8]
	}

	count := func(c *mongo.Collection, filter bson.M) int {
		n, err := c.CountDocuments(ctx, filter)
		if err != nil {
			return 0
		}
		return int(n)
	}

	// Registrations inside the period, next to the all-time total.
	newUsers := count(h.Store.Users, periodFilter)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"from":   from,
		"to":     to,
		"period": period,
		"top":    top,
		// Day by day, for the one chart worth drawing on a dashboard.
		// Built from the same orders already loaded rather than a second
		// query: a chart that disagrees with the totals above it is worse
		// than no chart.
		"series": dailySeries(orders, from, to),
		"users": map[string]int{
			"total":  count(h.Store.Users, bson.M{}),
			"new":    newUsers,
			"active": len(activeUsers),
			// Customers who have ever saved an address: these are the ones a
			// courier can be sent to without a phone call.
			"withAddress": count(h.Store.Users, bson.M{"addresses.0": bson.M{"$exists": true}}),
		},
		"couriers": map[string]int{
			"total":  count(h.Store.Couriers, scoped(branchScope)),
			"active": count(h.Store.Couriers, scoped(branchScope, "isActive", true)),
			"online": count(h.Store.Couriers, scoped(branchScope,
				"status", bson.M{"$in": []string{"free", "busy"}})),
		},
		"admins": map[string]int{
			"total":    count(h.Store.Admins, bson.M{}),
			"owners":   count(h.Store.Admins, bson.M{"role": "owner"}),
			"managers": count(h.Store.Admins, bson.M{"role": "manager"}),
		},
		"menu": map[string]int{
			"dishes":     count(h.Store.Menu, scope.brandFilter(bson.M{})),
			"available":  count(h.Store.Menu, scope.brandFilter(bson.M{"isAvailable": true})),
			"categories": count(h.Store.Categories, scope.brandFilter(bson.M{})),
		},
	})
}

// scoped copies a scope filter and adds key/value pairs to the copy, so the
// shared scope map is never mutated by one of the counts built from it.
func scoped(base bson.M, kv ...any) bson.M {
	out := bson.M{}
	for k, v := range base {
		out[k] = v
	}
	for i := 0; i+1 < len(kv); i += 2 {
		key, _ := kv[i].(string)
		out[key] = kv[i+1]
	}
	return out
}

// parseRange reads the ?from=/?to= pair. Both are optional; a bare date means
// the whole day, so `to` is pushed to the following midnight — an operator
// asking for "1–7 July" means the 7th included.
func parseRange(fromRaw, toRaw string) (*time.Time, *time.Time, error) {
	parse := func(s string, endOfDay bool) (*time.Time, error) {
		if s == "" {
			return nil, nil
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return &t, nil
		}
		t, err := time.ParseInLocation("2006-01-02", s, time.Local)
		if err != nil {
			return nil, errBadDate
		}
		if endOfDay {
			t = t.AddDate(0, 0, 1)
		}
		return &t, nil
	}
	from, err := parse(fromRaw, false)
	if err != nil {
		return nil, nil, err
	}
	to, err := parse(toRaw, true)
	if err != nil {
		return nil, nil, err
	}
	if from != nil && to != nil && to.Before(*from) {
		return nil, nil, errBadRange
	}
	return from, to, nil
}

// AdminAlerts is what the panel polls to know whether something new arrived:
// how many orders and bookings are waiting, and when the newest of each was
// created. The panel compares those timestamps with what it last saw and, if
// they moved, plays a sound — so nobody has to watch the screen.
//
// Deliberately tiny: it runs every few seconds on every open panel tab.
func (h *Handler) AdminAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Reads whichever timestamp it was asked to sort by, rather than assuming
	// createdAt: orders are watched by queuedAt (when the kitchen may start),
	// bookings by createdAt.
	newest := func(coll *mongo.Collection, filter bson.M, field string) *time.Time {
		opts := options.FindOne().SetSort(bson.D{{Key: field, Value: -1}})
		var doc bson.M
		if err := coll.FindOne(ctx, filter, opts).Decode(&doc); err != nil {
			return nil
		}
		if ts, ok := doc[field].(primitive.DateTime); ok {
			at := ts.Time()
			return &at
		}
		return nil
	}
	count := func(coll *mongo.Collection, filter bson.M) int {
		n, err := coll.CountDocuments(ctx, filter)
		if err != nil {
			return 0
		}
		return int(n)
	}

	// Scoped like everything else: a branch must not be made to hear the bell
	// for an order another branch is cooking.
	branchScope, scope, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// An order still waiting on a bank is not the kitchen's yet: it must not be
	// counted as waiting, and it must not ring the bell. When the money lands,
	// queuedAt is set and it becomes a normal new order — which is the moment
	// the chime is supposed to fire.
	pendingOrders := scoped(branchScope, "status", string(models.StatusPending))
	pendingOrders["paymentStatus"] = bson.M{"$ne": models.PayPending}
	// ⚠️ **The bell must not ring for the dining room, and this is the second
	// half of a fix that was only half done.**
	//
	// Excluding till checks from the orders *list* was not enough: the alert is
	// a different query, and it still counted them. An open check is stored
	// `status: pending, paymentStatus: "unpaid"`, and `"unpaid" != "pending"`
	// matches — so every table a waiter opened rang the panel.
	//
	// The result was worse than either failure alone: the list was empty and
	// the sound played, so the panel called an operator to something they could
	// not find and could not silence.
	//
	// ⚠️ Applied only to the alerts that mean **"a delivery order needs
	// accepting"**. `pos.failed`, `pos.unaccepted` and `fiscal.unfiled` are
	// about till sales too and must keep counting them.
	noTill := bson.M{"$exists": false}
	pendingOrders["check"] = noTill
	now := time.Now()
	// ⚠️ **`$lte` now, not merely "set".** A pre-order is stored with its
	// `queuedAt` in the future, so without this bound the bell would ring the
	// moment one was placed — for an order due next Saturday — and then stay
	// silent at the time it actually matters, because by then the timestamp is
	// no longer new. Exactly backwards, and it would look like a broken chime
	// rather than a missing bound.
	// Ordinary orders and pre-orders ring separately, because they ask for
	// different things. A pre-order arriving is news ("buy the meat"); a
	// pre-order falling due is an instruction ("start cooking"), and it arrives
	// hours later with nobody having touched anything. One timestamp cannot
	// carry both events, and merging them would mean the kitchen hears the same
	// sound for "in a week" and "now".
	plain := scoped(branchScope)
	plain["check"] = noTill
	plain["queuedAt"] = bson.M{"$exists": true, "$lte": now}
	plain["scheduledAt"] = nil
	placed := scoped(branchScope)
	placed["check"] = noTill
	placed["scheduledAt"] = bson.M{"$ne": nil}
	due := scoped(branchScope)
	due["check"] = noTill
	due["scheduledAt"] = bson.M{"$ne": nil}
	due["queuedAt"] = bson.M{"$exists": true, "$lte": now}
	// Due, accepted, and still nobody cooking it — the repeating alarm's
	// condition, and the counterpart of `orders.pending`.
	//
	// ⚠️ **`confirmed` only, deliberately.** A due pre-order still sitting in
	// `pending` is already counted above, and counting it twice would mean two
	// alarms for one order with two different sounds and only one action that
	// silences either. Split this way each alarm has exactly one clearing act:
	// "Qabul qilish" for one, "Tayyorlashni boshlash" for the other.
	dueWaiting := scoped(branchScope, "status", string(models.StatusConfirmed))
	dueWaiting["check"] = noTill
	dueWaiting["scheduledAt"] = bson.M{"$ne": nil}
	dueWaiting["queuedAt"] = bson.M{"$exists": true, "$lte": now}
	// Still ahead of the restaurant: what the panel's pre-order tab holds, and
	// the number worth knowing before ordering stock.
	upcoming := scoped(branchScope)
	upcoming["check"] = noTill
	upcoming["scheduledAt"] = bson.M{"$gt": now}
	upcoming["status"] = bson.M{"$ne": string(models.StatusCancelled)}
	pendingBookings := scoped(branchScope, "status", string(models.ReservationPending))
	// Handed to the till, and the till says a human there still has not taken
	// it. ⚠️ **No sound is attached to this one, deliberately.** Every other
	// alarm on this endpoint has exactly one button in this panel that clears
	// it; this one is cleared by walking to the counter and pressing accept on
	// the POS. A chime nobody in this app can silence is a chime people learn
	// to ignore — and it would train them to ignore the two that matter.
	unaccepted := scoped(branchScope)
	maps.Copy(unaccepted, unacceptedTillFilter(now))
	// Never reached the till at all — usually one dish with no mapping. Worse
	// than the above: there the ticket is on somebody's screen, here the
	// kitchen has nothing and no reason to suspect it.
	posFailed := scoped(branchScope)
	maps.Copy(posFailed, failedPOSFilter())
	// Money taken at our own counter with no tax receipt behind it. ⚠️ A heavier
	// failure than either POS alert above: those are about a kitchen not seeing
	// a ticket, which the restaurant discovers within the hour because a guest
	// is waiting. This one nothing discovers — the food goes out, the guest
	// leaves happy, and the gap surfaces at an inspection.
	fiscalUnfiled := scoped(branchScope)
	maps.Copy(fiscalUnfiled, unfiledFiscalFilter(now))

	httpx.JSON(w, http.StatusOK, map[string]any{
		"orders": map[string]any{
			"pending": count(h.Store.Orders, pendingOrders),
			// Only orders wanted now. A pre-order gets its own key below, so
			// the panel can name what it just heard.
			"newestAt": newest(h.Store.Orders, plain, "queuedAt"),
		},
		"preorders": map[string]any{
			// How many are still ahead — for the badge, not for the sound.
			"upcoming": count(h.Store.Orders, upcoming),
			// A new one was placed (watched by when it was written).
			"newestAt": newest(h.Store.Orders, placed, "createdAt"),
			// One is due now (watched by when it joined the queue).
			"dueAt": newest(h.Store.Orders, due, "queuedAt"),
			// Due, accepted, nobody cooking it yet: the panel keeps ringing
			// while this is above zero.
			"dueWaiting": count(h.Store.Orders, dueWaiting),
		},
		"reservations": map[string]any{
			"pending":  count(h.Store.Reservations, pendingBookings),
			"newestAt": newest(h.Store.Reservations, scoped(branchScope), "createdAt"),
		},
		"pos": map[string]any{
			// A count, shown as a banner. See above for why it is silent.
			"unaccepted": count(h.Store.Orders, unaccepted),
			"afterMins":  int(posTillAlertAfter / time.Minute),
			"failed":     count(h.Store.Orders, posFailed),
			// Prevention rather than a report: dishes the till has no id for.
			// The gap is created by ordinary work — somebody adds a dish weeks
			// after a correct setup — and it stays invisible until the first
			// order containing it fails as a whole. Cached, because this
			// endpoint runs every 15s on every open tab.
			"unmapped": h.unmappedDishes(ctx, h.scopeBranch(r, scope)),
		},
		// ⚠️ **The quietest failure in the system gets a banner, not a sound.**
		// A kitchen ticket that never printed leaves no trace anywhere else:
		// the order is on the screen, the sale is in the reports, and the only
		// symptom is a plate nobody made. But the clearing act is at the
		// printer — load paper, plug it back in — and an alarm nobody in this
		// app can silence is one people learn to ignore, which is a habit that
		// spreads to the two that must never be ignored.
		//
		// ⚠️ Bounded to the last 12 hours, unlike `pos.failed`. A printer that
		// has been unplugged for a week would otherwise show a number in the
		// hundreds that says nothing about tonight — and a count nobody can
		// bring back to zero is a count people stop reading.
		"print": map[string]any{
			"failed": count(h.Store.PrintJobs, printFailedFilter(branchScope, now)),
		},
		"fiscal": map[string]any{
			// ⚠️ A count and a banner, and **no sound** — the same judgement as
			// `pos.unaccepted`. The clearing act is pressing retry, which may
			// well fail again for as long as the register's PC is off; an alarm
			// that cannot be silenced by any action in this app is one people
			// learn to ignore, and that habit spreads to the two alarms that
			// must never be ignored.
			"unfiled": count(h.Store.Orders, fiscalUnfiled),
		},
	})
}

// dayPoint is one day of the dashboard chart.
type dayPoint struct {
	Date string `json:"date"`
	// Orders placed that day, cancellations excluded.
	Orders int `json:"orders"`
	// Money actually collected, on the same basis as the period figures —
	// `received()`, not "placed". A chart on a different basis from the
	// numbers above it is two answers to one question.
	Revenue int `json:"revenue"`
}

// dailySeries turns the period's orders into a day-by-day series.
//
// ⚠️ **Every day in the range appears, including the silent ones.** Charting
// only the days that had orders draws a line straight across a closed week and
// makes a quiet Monday invisible — the reader sees a smooth trend where there
// was a gap. The same reasoning as the XYZ analysis counting silent days as
// zero, arriving from the other side.
func dailySeries(orders []models.Order, from, to *time.Time) []dayPoint {
	byDay := map[string]*dayPoint{}
	var first, last time.Time

	for _, o := range orders {
		if o.Status == models.StatusCancelled {
			continue
		}
		local := o.CreatedAt.In(time.Local)
		key := local.Format("2006-01-02")
		d, ok := byDay[key]
		if !ok {
			d = &dayPoint{Date: key}
			byDay[key] = d
		}
		d.Orders++
		if received(o) {
			d.Revenue += o.Total
		}
		day := startOfLocalDay(local)
		if first.IsZero() || day.Before(first) {
			first = day
		}
		if last.IsZero() || day.After(last) {
			last = day
		}
	}
	if len(byDay) == 0 {
		return []dayPoint{}
	}
	// An explicit range wins over what the data happens to cover: an owner who
	// asked for thirty days should see thirty columns, and the empty ones are
	// the answer as much as the busy ones.
	if from != nil {
		first = startOfLocalDay(from.In(time.Local))
	}
	if to != nil {
		if end := startOfLocalDay(to.In(time.Local).Add(-time.Second)); end.After(last) {
			last = end
		}
	}

	out := []dayPoint{}
	// Capped so a request for "all time" on a three-year-old restaurant does
	// not return a thousand points nobody can read on a 900px chart.
	const maxDays = 120
	if last.Sub(first) > maxDays*24*time.Hour {
		first = last.AddDate(0, 0, -maxDays)
	}
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		if p, ok := byDay[key]; ok {
			out = append(out, *p)
		} else {
			out = append(out, dayPoint{Date: key})
		}
	}
	return out
}

func startOfLocalDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

// printFailedFilter is receipts the queue gave up on and nobody has re-sent.
//
// ⚠️ **`$gte: MaxPrintTries`, not `== `**: a job handed out one more time by a
// second agent would slip past an equality test, and the row it names is the
// one somebody has to act on.
func printFailedFilter(branchScope bson.M, now time.Time) bson.M {
	f := bson.M{
		"doneAt":    bson.M{"$exists": false},
		"tries":     bson.M{"$gte": models.MaxPrintTries},
		"createdAt": bson.M{"$gte": now.Add(-12 * time.Hour)},
	}
	for k, v := range branchScope {
		f[k] = v
	}
	return f
}
