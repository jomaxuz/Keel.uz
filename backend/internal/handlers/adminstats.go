package handlers

import (
	"errors"
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
	branchScope, _, err := h.orderScope(r)
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
	queued := scoped(branchScope)
	queued["queuedAt"] = bson.M{"$exists": true}
	pendingBookings := scoped(branchScope, "status", string(models.ReservationPending))

	httpx.JSON(w, http.StatusOK, map[string]any{
		"orders": map[string]any{
			"pending":  count(h.Store.Orders, pendingOrders),
			"newestAt": newest(h.Store.Orders, queued, "queuedAt"),
		},
		"reservations": map[string]any{
			"pending":  count(h.Store.Reservations, pendingBookings),
			"newestAt": newest(h.Store.Reservations, scoped(branchScope), "createdAt"),
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
