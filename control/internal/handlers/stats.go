package handlers

import (
	"context"
	"net/http"
	"time"

	"keel-control/internal/httpx"
	"keel-control/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Totals is one tenant's or the platform's numbers for a period.
type Totals struct {
	Orders   int `json:"orders"`
	Revenue  int `json:"revenue"`
	Billable int `json:"billable"`
}

// monthTotals sums the current calendar month per tenant, from the nightly
// rows rather than from the tenant databases — see package aggregate for why.
func (h *Handler) monthTotals(ctx context.Context) (map[string]Totals, error) {
	from := monthStart()
	cur, err := h.Store.Days.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"date": bson.M{"$gte": from}}}},
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
	out := map[string]Totals{}
	for _, r := range rows {
		out[hexOf(r.ID)] = Totals{r.Orders, r.Revenue, r.Billable}
	}
	return out, nil
}

// Stats is the dashboard's front page.
//
// Everything here is read from the aggregate rows and the tenant list — two
// collections in the control database, no tenant dialled. That is what keeps
// the page the same speed at five customers and at five hundred.
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	counts := map[string]int{}
	total := 0
	cur, err := h.Store.Tenants.Find(ctx, bson.M{})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var tenants []models.Tenant
	if err := cur.All(ctx, &tenants); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Who needs a phone call today, counted alongside the plain status tally.
	//
	// The two answer different questions: "trial" is how many customers are
	// evaluating, while "expired" is how many of those have run out and are
	// still running for free. Only the second one costs money every day it
	// goes unnoticed.
	watermarkRemoved := 0
	attention := map[string]int{}
	now := time.Now()
	for _, t := range tenants {
		counts[t.Status]++
		// Closed customers are counted on their own line and left out of the
		// total. "We have 40 customers" has to mean forty live businesses, or
		// the number grows on its own every time one leaves.
		if t.Status == models.StatusDeleted {
			continue
		}
		total++
		if t.HideWatermark {
			watermarkRemoved++
		}
		if a := tenantAttention(t, now); a.Kind != "" {
			attention[a.Kind]++
		}
	}

	month, err := h.monthTotals(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var m Totals
	for _, v := range month {
		m.Orders += v.Orders
		m.Revenue += v.Revenue
		m.Billable += v.Billable
	}

	// Thirty days of platform-wide orders, for the one chart worth drawing.
	series, err := h.dailySeries(ctx, 30)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// The five biggest customers this month. Useful for the opposite reason it
	// looks useful: it shows how much of the month's income rests on how few.
	top := topTenants(tenants, month, 5)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenants": map[string]int{
			"total":            total,
			"active":           counts[models.StatusActive],
			"trial":            counts[models.StatusTrial],
			"suspended":        counts[models.StatusSuspended],
			"deleted":          counts[models.StatusDeleted],
			"watermarkRemoved": watermarkRemoved,
		},
		"attention": map[string]int{
			"trialEnding":  attention[AttentionTrialEnding],
			"trialExpired": attention[AttentionTrialExpired],
			"unpaid":       attention[AttentionUnpaid],
		},
		"month":  m,
		"series": series,
		"top":    top,
	})
}

type point struct {
	Date     string `json:"date"`
	Orders   int    `json:"orders"`
	Revenue  int    `json:"revenue"`
	Billable int    `json:"billable"`
}

func (h *Handler) dailySeries(ctx context.Context, days int) ([]point, error) {
	from := time.Now().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	cur, err := h.Store.Days.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"date": bson.M{"$gte": from}}}},
		{{Key: "$group", Value: bson.M{
			"_id":      "$date",
			"orders":   bson.M{"$sum": "$orders"},
			"revenue":  bson.M{"$sum": "$revenue"},
			"billable": bson.M{"$sum": "$billable"},
		}}},
		{{Key: "$sort", Value: bson.M{"_id": 1}}},
	})
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Date     string `bson:"_id"`
		Orders   int    `bson:"orders"`
		Revenue  int    `bson:"revenue"`
		Billable int    `bson:"billable"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	// Days with no orders are missing from the aggregate; a chart that simply
	// skips them draws a busy week and a quiet week the same width.
	have := map[string]point{}
	for _, r := range rows {
		have[r.Date] = point{r.Date, r.Orders, r.Revenue, r.Billable}
	}
	out := make([]point, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		if p, ok := have[d]; ok {
			out = append(out, p)
		} else {
			out = append(out, point{Date: d})
		}
	}
	return out, nil
}

type topRow struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Orders   int    `json:"orders"`
	Billable int    `json:"billable"`
}

func topTenants(tenants []models.Tenant, month map[string]Totals, n int) []topRow {
	rows := make([]topRow, 0, len(tenants))
	for _, t := range tenants {
		m := month[t.ID.Hex()]
		if m.Orders == 0 {
			continue
		}
		rows = append(rows, topRow{t.ID.Hex(), t.Name, t.Slug, m.Orders, m.Billable})
	}
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j].Orders > rows[j-1].Orders; j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
	if len(rows) > n {
		rows = rows[:n]
	}
	return rows
}

func monthStart() string {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).Format("2006-01-02")
}

func hexOf(v any) string {
	type hexer interface{ Hex() string }
	if h, ok := v.(hexer); ok {
		return h.Hex()
	}
	return ""
}
