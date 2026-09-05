package handlers

// ---- The platform, split by what our customers actually are ----
//
// ⚠️ **A restaurant and a shop are not one business and must not be one row.**
// They are sold different ladders at a third of each other's price, they trade
// differently — a dining room's money is in checks, a grocery's is in a few
// hundred scans an hour — and the only question worth asking of the pair is
// which of them is worth more to us per month. Averaged together that answer
// does not exist: one figure is the mean of two populations and describes
// neither, which is how a platform decides to sell more of the wrong thing.
//
// ⚠️ **Read entirely from the nightly rows and one Docker call.** No tenant
// database is dialled — the rule the overview screen already follows, and for
// the same reason: a page that opens a connection per customer gets slower with
// every customer sold, which is the one curve that must not bend the wrong way.
//
// ⚠️ **"Ours per month" is two figures, never one.** The till is a monthly
// subscription and the website is billed per order; adding a month of one to a
// window of the other produces a number that matches no invoice we have ever
// issued. They are reported separately, and the window defaults to thirty days
// so that the two can honestly be read side by side.

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"keel-control/internal/httpx"
	"keel-control/internal/models"
)

// tenantMini is one customer, small enough to sit in a "best and worst" pair.
type tenantMini struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Revenue  int    `json:"revenue"`
	Ops      int    `json:"ops"`
	Online   bool   `json:"online"`
	LastSale string `json:"lastSale,omitempty"`
}

// bizRow is one kind of business, totalled.
type bizRow struct {
	// "" is a restaurant, the same convention the business type itself uses.
	Type string `json:"type"`
	// How many customers of this kind, and how many are answering right now.
	//
	// ⚠️ **Online is read from Docker, not from a stored flag.** A saved
	// "ready" goes stale the moment a container dies, and the console showed a
	// customer as working with no container at all for a week — the lesson
	// `provisionStatus` taught this codebase.
	Tenants int `json:"tenants"`
	Online  int `json:"online"`
	// ⚠️ **Unknown is not "down".** Docker unreachable — a laptop, a socket
	// that moved — means nothing is known about anybody, and lighting up every
	// row is a platform-wide false alarm nobody reads twice.
	OnlineKnown bool `json:"onlineKnown"`
	// Customers of this kind that sold nothing at all in the window.
	Idle int `json:"idle"`

	// What they did.
	Orders     int `json:"orders"`
	TillChecks int `json:"tillChecks"`
	// Every sale, however it was rung up. ⚠️ The two are kept above as well:
	// summed alone they hide a restaurant whose online orders are flat while
	// its counter doubles, which is the shape of a customer growing.
	Ops      int `json:"ops"`
	Revenue  int `json:"revenue"`
	Visitors int `json:"visitors"`

	// What we earn. Two figures, never added — see the note at the top.
	Subscription int `json:"subscription"`
	PerOrder     int `json:"perOrder"`

	// The most recent day anybody of this kind sold anything, "YYYY-MM-DD".
	LastSale string `json:"lastSale,omitempty"`

	Top    *tenantMini `json:"top,omitempty"`
	Bottom *tenantMini `json:"bottom,omitempty"`
}

// BusinessBreakdown answers "which kind of customer is worth what".
func (h *Handler) BusinessBreakdown(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ⚠️ Thirty days by default so the per-order figure and the monthly
	// subscription beside it describe the same length of time.
	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 366 {
			days = n
		}
	}
	to := time.Now().In(time.Local)
	from := to.AddDate(0, 0, -(days - 1))
	fromKey, toKey := from.Format("2006-01-02"), to.Format("2006-01-02")

	// ⚠️ **Suspended and deleted customers are left out entirely.** They are
	// sites we switched off; counting them as "idle" would put our own decision
	// in a column that reads as a customer's problem, and every row would need
	// explaining before it could be used.
	cur, err := h.Store.Tenants.Find(ctx, bson.M{
		"status": bson.M{"$nin": []string{models.StatusSuspended, models.StatusDeleted}},
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var tenants []models.Tenant
	if err := cur.All(ctx, &tenants); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	sums, err := h.tenantSums(ctx, fromKey, toKey)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	states := h.containerStates(ctx)
	known := len(states) > 0

	rows := map[string]*bizRow{}
	for _, t := range tenants {
		row := rows[t.BusinessType]
		if row == nil {
			row = &bizRow{Type: t.BusinessType, OnlineKnown: known}
			rows[t.BusinessType] = row
		}
		row.Tenants++
		up := stateOf(states, t.Slug) == "running"
		if up {
			row.Online++
		}

		s := sums[t.ID]
		row.Orders += s.Orders
		row.TillChecks += s.TillChecks
		row.Revenue += s.Revenue + s.TillRevenue
		row.Visitors += s.Visitors
		row.PerOrder += s.Billable
		row.Subscription += TillMonthlyFor(t)

		ops := s.Orders + s.TillChecks
		row.Ops += ops
		if ops == 0 {
			row.Idle++
		}
		if s.LastSale > row.LastSale {
			row.LastSale = s.LastSale
		}

		// ⚠️ **Best and worst are picked among customers that traded.** A shop
		// that opened yesterday would otherwise always be "the worst", which is
		// a fact about its age rather than about its trade — and the row exists
		// to start a phone call.
		if ops == 0 {
			continue
		}
		mini := &tenantMini{
			ID: t.ID.Hex(), Slug: t.Slug, Name: t.Name,
			Revenue: s.Revenue + s.TillRevenue, Ops: ops,
			Online: up, LastSale: s.LastSale,
		}
		if row.Top == nil || mini.Revenue > row.Top.Revenue {
			row.Top = mini
		}
		if row.Bottom == nil || mini.Revenue < row.Bottom.Revenue {
			row.Bottom = mini
		}
	}

	out := make([]bizRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, *r)
	}
	// Biggest earner first: the screen exists to answer which kind of customer
	// to sell more of.
	sort.Slice(out, func(i, j int) bool {
		return out[i].Subscription+out[i].PerOrder > out[j].Subscription+out[j].PerOrder
	})

	httpx.JSON(w, http.StatusOK, map[string]any{
		"rows": out,
		"from": fromKey,
		"to":   toKey,
		"days": days,
		// ⚠️ Sent rather than computed in the browser: "today" on a machine
		// whose clock is a day out would shift every "how long ago" on screen.
		"now": time.Now(),
	})
}

// tenantSum is one customer's window, out of the nightly rows.
type tenantSum struct {
	Orders      int
	TillChecks  int
	Revenue     int
	TillRevenue int
	Visitors    int
	Billable    int
	// The last date this customer sold anything at all.
	LastSale string
}

func (h *Handler) tenantSums(
	ctx context.Context, from, to string,
) (map[primitive.ObjectID]tenantSum, error) {
	cur, err := h.Store.Days.Aggregate(ctx, mongo.Pipeline{
		// Both ends inclusive, the same rule the overview follows: these are
		// calendar days somebody typed into two boxes.
		{{Key: "$match", Value: bson.M{"date": bson.M{"$gte": from, "$lte": to}}}},
		{{Key: "$group", Value: bson.M{
			"_id":         "$tenantId",
			"orders":      bson.M{"$sum": "$orders"},
			"tillChecks":  bson.M{"$sum": "$tillChecks"},
			"revenue":     bson.M{"$sum": "$revenue"},
			"tillRevenue": bson.M{"$sum": "$tillRevenue"},
			"visitors":    bson.M{"$sum": "$visitors"},
			"billable":    bson.M{"$sum": "$billable"},
			// ⚠️ **The last day with a sale on it, not the last row.** The
			// collector writes a row for every day it walks, including the
			// quiet ones — so "the newest row" is always yesterday and would
			// tell every customer they sold something last night.
			// ⚠️ **`$ifNull` on both, and this is not defensive typing.**
			// `tillChecks` is `omitempty`, so it is simply absent on every row
			// written for a customer with no counter — and `$add` with a
			// missing field is **null**, not the other number. Without this the
			// column read "never sold" for every website-only customer on the
			// list, which is exactly the row an operator would have rung about.
			"lastSale": bson.M{"$max": bson.M{"$cond": []any{
				bson.M{"$gt": []any{
					bson.M{"$add": []any{
						bson.M{"$ifNull": []any{"$orders", 0}},
						bson.M{"$ifNull": []any{"$tillChecks", 0}},
					}}, 0,
				}},
				"$date", "",
			}}},
		}}},
	})
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID          primitive.ObjectID `bson:"_id"`
		Orders      int                `bson:"orders"`
		TillChecks  int                `bson:"tillChecks"`
		Revenue     int                `bson:"revenue"`
		TillRevenue int                `bson:"tillRevenue"`
		Visitors    int                `bson:"visitors"`
		Billable    int                `bson:"billable"`
		LastSale    string             `bson:"lastSale"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make(map[primitive.ObjectID]tenantSum, len(rows))
	for _, r := range rows {
		out[r.ID] = tenantSum{
			Orders: r.Orders, TillChecks: r.TillChecks,
			Revenue: r.Revenue, TillRevenue: r.TillRevenue,
			Visitors: r.Visitors, Billable: r.Billable,
			LastSale: r.LastSale,
		}
	}
	return out, nil
}
