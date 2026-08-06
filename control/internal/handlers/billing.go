package handlers

import (
	"context"
	"errors"
	"strings"
	"time"

	"keel-control/internal/billing"
	"keel-control/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Period is one tenant's current window, as the console shows it.
//
// `To` is exclusive, which makes it exactly the date of the next invoice —
// the one thing an operator on the phone is actually asked.
type Period struct {
	From string `json:"from"`
	To   string `json:"to"`
	// "subscription" — a paying customer's monthly cycle.
	// "trial" — an evaluation, which has an end rather than a cycle.
	Kind string `json:"kind"`
	// The recorded subscription day, "YYYY-MM-DD", empty when there is none.
	//
	// Sent as a plain local day rather than left to the browser to slice out of
	// the timestamp: `subscribedAt` marshals as UTC, so the first ten
	// characters of it are the *previous* day here — a date field that quietly
	// showed the 4th for a subscription made on the 5th, and saved it back.
	Anchor string `json:"anchor"`
	// False when the cycle is counted from the day the tenant was opened
	// because no subscription date was ever recorded. Surfaced rather than
	// hidden: the number is still the best available answer, but an operator
	// correcting a bill deserves to know which date it rests on.
	Anchored bool `json:"anchored"`
}

// tenantPeriod decides which days belong to a tenant's current period.
//
// Trials are deliberately not given a monthly cycle. A demo is a deadline, and
// showing "next invoice" for a customer who has not agreed to pay yet is how an
// operator ends up quoting a bill that does not exist.
func tenantPeriod(t models.Tenant, now time.Time) Period {
	if t.Status == models.StatusTrial && t.TrialEndsAt != nil {
		from := local(t.CreatedAt)
		if from.IsZero() {
			from = local(*t.TrialEndsAt)
		}
		to := local(*t.TrialEndsAt)
		// Bad data — an end before the beginning — must still produce a window
		// that can be summed, or the row simply vanishes from the table.
		if !to.After(from) {
			to = from.AddDate(0, 0, 1)
		}
		return Period{
			From: billing.Day(from), To: billing.Day(to),
			Kind: "trial", Anchored: true,
		}
	}

	anchor, anchored := local(t.CreatedAt), false
	if t.SubscribedAt != nil && !t.SubscribedAt.IsZero() {
		anchor, anchored = local(*t.SubscribedAt), true
	}
	if anchor.IsZero() {
		anchor = now
	}
	from, to := billing.Cycle(anchor, now)
	p := Period{
		From: billing.Day(from), To: billing.Day(to),
		Kind: "subscription", Anchored: anchored,
	}
	if anchored {
		p.Anchor = billing.Day(anchor)
	}
	return p
}

// local moves a stored timestamp into the zone every day boundary in this
// system is measured in.
//
// ⚠️ The Mongo driver decodes every date as **UTC**, whatever was written. Left
// alone, a subscription anchored at local midnight comes back as 19:00 the
// previous day, and the period — and the invoice — starts a day early. It fails
// silently: the window is still a valid month, just the wrong one, and only a
// customer counting their own orders would ever notice.
//
// This is the same trap that put every staff shift on the wrong day in the
// tenant app, arriving from a different direction.
func local(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.In(time.Local)
}

func startOfToday() time.Time { return startOfDay(time.Now()) }

// parseDay reads a subscription date typed by an operator.
//
// Local midnight, never UTC: the whole system buckets days the way the owner
// reads a wall calendar, and a date parsed five hours off would move a
// boundary — and therefore an invoice — by a day for anyone anchored on the
// 1st.
//
// The bounds are not paranoia about attackers (this endpoint is behind a Keel
// login); they catch the typo. A year entered as 2206 would push the anchor
// past every real date, and Cycle would then quietly hand that customer their
// "first period" forever.
func parseDay(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	var (
		t   time.Time
		err error
	)
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		if t, err = time.ParseInLocation(layout, s, time.Local); err == nil {
			break
		}
	}
	if err != nil {
		return time.Time{}, err
	}
	y, m, d := t.Date()
	t = time.Date(y, m, d, 0, 0, 0, 0, time.Local)

	if t.Before(time.Date(2020, time.January, 1, 0, 0, 0, 0, time.Local)) {
		return time.Time{}, errBadDate
	}
	// A little slack forward: a subscription agreed today but starting on the
	// first of next month is a real thing an operator writes down.
	if t.After(time.Now().AddDate(0, 0, 62)) {
		return time.Time{}, errBadDate
	}
	return t, nil
}

var errBadDate = errors.New("sana chegaradan tashqarida")

// Attention values — what a tenant needs from a human, in the order they get
// urgent. Computed from dates on every request, never stored: a saved flag goes
// stale the moment the clock passes it, while a date does not.
const (
	// A demo with three days or fewer left. The call has to happen now, while
	// the customer is still using the thing they would be paying for.
	AttentionTrialEnding = "trial_ending"
	// A demo whose deadline has passed. Either it becomes a paying customer or
	// it is switched off — leaving it running is how a demo becomes free
	// forever.
	AttentionTrialExpired = "trial_expired"
	// Switched off for non-payment. Still on the list on purpose: a suspended
	// customer is a customer to phone, not a customer to forget.
	AttentionUnpaid = "suspended"
)

// trialEndingDays is how much warning is worth giving. Short enough that the
// badge means something, long enough to actually reach an owner who is running
// a kitchen.
const trialEndingDays = 3

// Attention is the row's warning, if it has one.
type Attention struct {
	// One of the constants above, empty when nothing is wrong.
	Kind string `json:"kind"`
	// Whole days left (ending) or days past the deadline (expired). Computed
	// here so the browser never does date arithmetic on a timestamp that
	// marshals as UTC.
	Days int `json:"days"`
}

func tenantAttention(t models.Tenant, now time.Time) Attention {
	// A customer on free terms is never on the "call them" list, whatever
	// their trial says. The whole point of the flag is that the conversation
	// about money has already happened and ended differently — putting them in
	// the queue every morning would teach the operator to ignore the queue.
	if t.FreeAt(now) && t.Status != models.StatusSuspended {
		return Attention{}
	}
	switch t.Status {
	case models.StatusSuspended:
		return Attention{Kind: AttentionUnpaid}
	case models.StatusTrial:
		if t.TrialEndsAt == nil {
			return Attention{}
		}
		// Whole days between calendar days, not between instants: a demo ending
		// tonight and one ending tomorrow morning are "0 days" and "1 day", not
		// both "0" because fewer than 24 hours separate them.
		ends := startOfDay(local(*t.TrialEndsAt))
		days := int(ends.Sub(startOfDay(now)).Hours() / 24)
		if days <= 0 {
			return Attention{Kind: AttentionTrialExpired, Days: -days}
		}
		if days <= trialEndingDays {
			return Attention{Kind: AttentionTrialEnding, Days: days}
		}
	}
	return Attention{}
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// attentionFilter turns the console's warning buttons into a query.
//
// Returned as a clause to be ANDed with whatever else is being asked, never
// merged into the filter directly: both this and the search box want `$or`, and
// a map assignment would let one silently replace the other — a search that
// quietly stopped searching and returned tenants nobody asked for.
func attentionFilter(kind string, now time.Time) (bson.M, bool) {
	today := startOfDay(now)
	expired := bson.M{"status": models.StatusTrial, "trialEndsAt": bson.M{"$lte": today}}
	ending := bson.M{
		"status": models.StatusTrial,
		"trialEndsAt": bson.M{
			"$gt":  today,
			"$lte": today.AddDate(0, 0, trialEndingDays),
		},
	}
	unpaid := bson.M{"status": models.StatusSuspended}

	switch kind {
	case AttentionTrialExpired:
		return expired, true
	case AttentionTrialEnding:
		return ending, true
	case AttentionUnpaid:
		return unpaid, true
	case "any":
		return bson.M{"$or": []bson.M{expired, ending, unpaid}}, true
	}
	return nil, false
}

// periodTotals sums each tenant over its own window in a single round trip.
//
// Every tenant needs a different range, so the match is an $or of one clause
// per tenant — each one served by the (tenantId, date) index the aggregate rows
// already carry. The alternative, one query per tenant, turns the customer list
// into N round trips and grows the page's load time with every sale: the same
// curve the nightly aggregate exists to avoid.
//
// Only IDs read from our own collection ever reach this filter. Nothing a
// browser sends is interpolated into it.
func (h *Handler) periodTotals(ctx context.Context, tenants []models.Tenant, now time.Time) (map[string]Totals, map[string]Period, error) {
	periods := make(map[string]Period, len(tenants))
	clauses := make([]bson.M, 0, len(tenants))
	for _, t := range tenants {
		p := tenantPeriod(t, now)
		periods[t.ID.Hex()] = p
		clauses = append(clauses, bson.M{
			"tenantId": t.ID,
			// Half-open, exactly like the window itself: the day an invoice
			// closes is the first day of the next one, and counting it twice
			// is a charge the customer did not make.
			"date": bson.M{"$gte": p.From, "$lt": p.To},
		})
	}
	out := map[string]Totals{}
	if len(clauses) == 0 {
		return out, periods, nil
	}

	cur, err := h.Store.Days.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"$or": clauses}}},
		{{Key: "$group", Value: bson.M{
			"_id":      "$tenantId",
			"orders":   bson.M{"$sum": "$orders"},
			"revenue":  bson.M{"$sum": "$revenue"},
			"billable": bson.M{"$sum": "$billable"},
		}}},
	})
	if err != nil {
		return nil, nil, err
	}
	var rows []struct {
		ID       any `bson:"_id"`
		Orders   int `bson:"orders"`
		Revenue  int `bson:"revenue"`
		Billable int `bson:"billable"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, nil, err
	}
	// The period's price, recomputed from its order count.
	//
	// ⚠️ Deliberately **not** the sum of the daily `billable` estimates. The
	// volume ladder is monthly; adding up days would apply the first band
	// thirty times and a busy restaurant would never reach a cheaper one — the
	// customer list would then quote a number the invoice does not agree with,
	// which is the one disagreement nobody forgives.
	byID := make(map[string]models.Tenant, len(tenants))
	for _, t := range tenants {
		byID[t.ID.Hex()] = t
	}
	for _, r := range rows {
		id := hexOf(r.ID)
		total := Totals{Orders: r.Orders, Revenue: r.Revenue, Billable: r.Billable}
		if t, ok := byID[id]; ok {
			total.Billable = t.ChargeForOrders(r.Orders, h.Cfg.PriceTiers, now)
		}
		out[id] = withShare(total)
	}
	return out, periods, nil
}

// lifetimeTotals sums every day the given tenants have ever had.
//
// Shown beside the current period because the two answer different questions —
// "what do they owe now" and "how much is this customer worth" — and an
// operator deciding whether to chase a late payment wants both on one line.
//
// Scoped to the tenants actually on screen rather than grouping the whole
// collection: one customer's card would otherwise pay the cost of every other
// customer's history, which is the exact shape of slowdown the nightly
// aggregate exists to prevent.
func (h *Handler) lifetimeTotals(ctx context.Context, tenants []models.Tenant) (map[string]Totals, error) {
	out := map[string]Totals{}
	if len(tenants) == 0 {
		return out, nil
	}
	ids := make([]primitive.ObjectID, 0, len(tenants))
	for _, t := range tenants {
		ids = append(ids, t.ID)
	}
	cur, err := h.Store.Days.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"tenantId": bson.M{"$in": ids}}}},
		{{Key: "$group", Value: bson.M{
			"_id":      "$tenantId",
			"orders":   bson.M{"$sum": "$orders"},
			"revenue":  bson.M{"$sum": "$revenue"},
			"billable": bson.M{"$sum": "$billable"},
		}}},
	})
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID       any `bson:"_id"`
		Orders   int `bson:"orders"`
		Revenue  int `bson:"revenue"`
		Billable int `bson:"billable"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[hexOf(r.ID)] = Totals{Orders: r.Orders, Revenue: r.Revenue, Billable: r.Billable}
	}
	return out, nil
}
