package handlers

// ---- The platform in one window ----
//
// The dashboard's front page answers "how are our customers doing **this
// month**", because that is the question an invoice is built from. This answers
// a different one: how is the whole platform doing over a window somebody
// chooses. They are not the same screen and must not be the same number — the
// month is a billing period with a start nobody picks, and the answer to "is
// this growing" needs a yesterday, a week and a year beside each other.
//
// ⚠️ **Read entirely from the nightly rows**, like everything else here: no
// tenant database is dialled. A platform overview that opened fifty connections
// would get slower with every customer sold, which is the one curve that must
// not bend the wrong way.

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"keel-control/internal/httpx"
	"keel-control/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// overviewPoint is one bucket of the platform's trade.
//
// ⚠️ **Online and counter stay in separate fields, never summed into one
// "orders".** They are settled differently — per order against a monthly
// subscription — so a single figure would be a number that matches neither
// invoice. It is also the distinction the operator reading this screen most
// needs: a platform whose order count is flat while its check count doubles is
// growing, and one figure would hide that completely.
type overviewPoint struct {
	// Bucket start, "YYYY-MM-DD". For a week or a month bucket this is the
	// first day inside it, which is what a chart axis should be labelled with.
	Date string `json:"date"`

	Visitors int `json:"visitors"`
	Views    int `json:"views"`

	Orders    int `json:"orders"`
	Cancelled int `json:"cancelled"`
	Revenue   int `json:"revenue"`
	Billable  int `json:"billable"`

	TillChecks   int `json:"tillChecks"`
	TillGuests   int `json:"tillGuests"`
	TillRevenue  int `json:"tillRevenue"`
	TillRefunded int `json:"tillRefunded"`
}

func (p *overviewPoint) add(o overviewPoint) {
	p.Visitors += o.Visitors
	p.Views += o.Views
	p.Orders += o.Orders
	p.Cancelled += o.Cancelled
	p.Revenue += o.Revenue
	p.Billable += o.Billable
	p.TillChecks += o.TillChecks
	p.TillGuests += o.TillGuests
	p.TillRevenue += o.TillRevenue
	p.TillRefunded += o.TillRefunded
}

// Named ranges the console offers as buttons, in days.
//
// ⚠️ **"1 day" is today, so it is one day and not zero.** An off-by-one here
// would draw an empty screen at nine in the morning and look like a broken
// collector rather than like a quiet start.
var overviewRanges = map[string]int{
	"1d":  1,
	"7d":  7,
	"30d": 30,
	"90d": 90,
	"1y":  365,
}

// Overview is the platform's own numbers over a chosen window.
func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	from, to, err := overviewWindow(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	cur, err := h.Store.Days.Aggregate(ctx, mongo.Pipeline{
		// Both ends inclusive: these are calendar days a person typed into two
		// boxes, and a half-open range would silently drop the day they named
		// as the end — the single most confusing possible off-by-one on a
		// screen whose whole job is a date range.
		{{Key: "$match", Value: bson.M{"date": bson.M{"$gte": from, "$lte": to}}}},
		{{Key: "$group", Value: bson.M{
			"_id":          "$date",
			"visitors":     bson.M{"$sum": "$visitors"},
			"views":        bson.M{"$sum": "$views"},
			"orders":       bson.M{"$sum": "$orders"},
			"cancelled":    bson.M{"$sum": "$cancelled"},
			"revenue":      bson.M{"$sum": "$revenue"},
			"billable":     bson.M{"$sum": "$billable"},
			"tillChecks":   bson.M{"$sum": "$tillChecks"},
			"tillGuests":   bson.M{"$sum": "$tillGuests"},
			"tillRevenue":  bson.M{"$sum": "$tillRevenue"},
			"tillRefunded": bson.M{"$sum": "$tillRefunded"},
			// How many customers traded at all that day. ⚠️ Counted per day
			// and **never summed** across the window — see `activeTenants`.
			"tenants": bson.M{"$addToSet": "$tenantId"},
		}}},
		{{Key: "$sort", Value: bson.M{"_id": 1}}},
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var rows []struct {
		Date         string `bson:"_id"`
		Visitors     int    `bson:"visitors"`
		Views        int    `bson:"views"`
		Orders       int    `bson:"orders"`
		Cancelled    int    `bson:"cancelled"`
		Revenue      int    `bson:"revenue"`
		Billable     int    `bson:"billable"`
		TillChecks   int    `bson:"tillChecks"`
		TillGuests   int    `bson:"tillGuests"`
		TillRevenue  int    `bson:"tillRevenue"`
		TillRefunded int    `bson:"tillRefunded"`
		Tenants      []any  `bson:"tenants"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	byDate := make(map[string]overviewPoint, len(rows))
	// ⚠️ **Distinct over the whole window, not the sum of the daily counts.**
	// A restaurant that traded on all thirty days would otherwise be counted
	// thirty times, and "45 active customers" on a platform of 12 is a number
	// somebody quotes in a meeting before anybody checks it.
	active := map[string]bool{}
	for _, r := range rows {
		byDate[r.Date] = overviewPoint{
			Date: r.Date, Visitors: r.Visitors, Views: r.Views,
			Orders: r.Orders, Cancelled: r.Cancelled,
			Revenue: r.Revenue, Billable: r.Billable,
			TillChecks: r.TillChecks, TillGuests: r.TillGuests,
			TillRevenue: r.TillRevenue, TillRefunded: r.TillRefunded,
		}
		for _, id := range r.Tenants {
			active[hexOf(id)] = true
		}
	}

	// ⚠️ **Every day in the window, including the empty ones.** Days with no
	// trade are missing from the aggregate, and a chart that simply skips them
	// draws a busy week and a quiet week the same width — the same rule
	// dailySeries follows, and the same reason the ABC report counts a day with
	// no sales as a zero rather than as a missing observation.
	start, _ := time.ParseInLocation("2006-01-02", from, time.Local)
	end, _ := time.ParseInLocation("2006-01-02", to, time.Local)
	daily := []overviewPoint{}
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		if p, ok := byDate[key]; ok {
			daily = append(daily, p)
		} else {
			daily = append(daily, overviewPoint{Date: key})
		}
	}

	var total overviewPoint
	for _, p := range daily {
		total.add(p)
	}
	total.Date = ""

	bucket, series := bucketSeries(daily)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"from": from,
		"to":   to,
		// Which grain the series came back at. ⚠️ Sent rather than inferred:
		// the browser would have to re-derive the same rule from the row count,
		// and the day the two disagree the axis is labelled in the wrong unit.
		"bucket": bucket,
		"series": series,
		"total":  total,
		// Customers that traded at all in the window — the denominator for
		// everything above. Without it, "8 000 orders" is unreadable: it is a
		// different platform at three customers than at eighty.
		"activeTenants": len(active),
		// The collector's own report, for the same reason the front page
		// carries it: an empty chart otherwise has two indistinguishable
		// causes, and no way at all to tell them apart.
		"collector": h.lastCollectorRun(ctx),
	})
}

// bucketSeries groups a daily series so a chart stays readable.
//
// ⚠️ **A year is 365 bars, and 365 bars is a smear.** The point of the year
// button is the shape of the year, and at one bar per day the shape is noise —
// every weekend dip drawn at a width of half a pixel. Weeks and months are the
// grains people actually reason in ("last March", "the week of the holiday"),
// so the thresholds are set where the axis stops being readable rather than at
// a round number.
//
// ⚠️ Buckets are **sums, never averages**. An average would make a chart whose
// bars cannot be added up to the total printed above it, and somebody will try.
func bucketSeries(daily []overviewPoint) (string, []overviewPoint) {
	switch {
	case len(daily) <= 45:
		return "day", daily
	case len(daily) <= 200:
		return "week", groupBy(daily, func(t time.Time) string {
			// ⚠️ Monday-start weeks. Go's Sunday-start `Weekday()` would put
			// Sunday alone in the previous bucket, and in a restaurant business
			// the weekend is the point of the chart — splitting it across two
			// bars is the one grouping that hides what somebody opened this to
			// see.
			off := (int(t.Weekday()) + 6) % 7
			return t.AddDate(0, 0, -off).Format("2006-01-02")
		})
	default:
		return "month", groupBy(daily, func(t time.Time) string {
			return t.Format("2006-01") + "-01"
		})
	}
}

// groupBy folds daily points into buckets named by `key`, preserving order.
func groupBy(daily []overviewPoint, key func(time.Time) string) []overviewPoint {
	out := []overviewPoint{}
	at := map[string]int{}
	for _, p := range daily {
		t, err := time.ParseInLocation("2006-01-02", p.Date, time.Local)
		if err != nil {
			continue
		}
		k := key(t)
		i, ok := at[k]
		if !ok {
			out = append(out, overviewPoint{Date: k})
			i = len(out) - 1
			at[k] = i
		}
		out[i].add(p)
	}
	return out
}

// overviewWindow resolves the requested period into two inclusive local dates.
//
// ⚠️ **Local dates, never UTC.** A day here is what somebody sees on a wall
// calendar — the same convention the nightly rows are keyed by — and reading
// "today" in UTC would start the window five hours late in Tashkent, so the
// one-day button would show an empty screen for most of the working morning.
func overviewWindow(r *http.Request) (from, to string, err error) {
	q := r.URL.Query()
	now := time.Now()

	// Hand-typed dates win over the shorthand: somebody who filled both boxes
	// asked a specific question, and silently answering a different one is
	// worse than refusing.
	rawFrom := strings.TrimSpace(q.Get("from"))
	rawTo := strings.TrimSpace(q.Get("to"))
	if rawFrom != "" || rawTo != "" {
		f, e1 := time.ParseInLocation("2006-01-02", rawFrom, time.Local)
		t, e2 := time.ParseInLocation("2006-01-02", rawTo, time.Local)
		if e1 != nil || e2 != nil {
			return "", "", errBadRange
		}
		// ⚠️ Swapped dates are corrected rather than refused. Two date inputs
		// side by side get filled in the wrong order constantly, and the answer
		// the operator wanted is unambiguous — refusing teaches nothing and
		// costs a retype.
		if t.Before(f) {
			f, t = t, f
		}
		// A window nobody can draw. The cap is generous on purpose: this is a
		// guard against a typo'd year, not a policy about how far back somebody
		// may look.
		if t.Sub(f) > 5*365*24*time.Hour {
			return "", "", errRangeTooLong
		}
		return f.Format("2006-01-02"), t.Format("2006-01-02"), nil
	}

	days, ok := overviewRanges[strings.ToLower(strings.TrimSpace(q.Get("range")))]
	if !ok {
		// ⚠️ An unknown shorthand falls back to 30 days rather than failing.
		// This value arrives from a query string, so it can be anything; the
		// screen it feeds is a dashboard, and a dashboard that answers a
		// mistyped URL with an error is a dashboard somebody stops bookmarking.
		days = 30
	}
	if n, e := strconv.Atoi(q.Get("days")); e == nil && n > 0 && n <= 1825 {
		days = n
	}
	end := now
	return end.AddDate(0, 0, -(days - 1)).Format("2006-01-02"),
		end.Format("2006-01-02"), nil
}

var (
	errBadRange     = errString("sana noto'g'ri — YYYY-MM-DD kutilyapti")
	errRangeTooLong = errString("davr juda uzun")
)

type errString string

func (e errString) Error() string { return string(e) }

// compile-time reminder that this file reads the same rows the invoice does.
var _ = models.TenantDay{}
