package handlers

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

// Sales over time: the same period cut into days, weeks or months.
//
// The dashboard already answers "how are we doing right now" — one period, one
// set of numbers. This answers the question that comes immediately after it and
// that a single figure cannot: **is it going up or down**. A month's takings
// mean one thing after a better month and the opposite after a worse one, and
// an owner reading a lone total has no way to tell which they are looking at.
//
// So this report is deliberately two things at once:
//
//   - a **row per bucket** (day / week / month), which is the table and the
//     spreadsheet — the shape an accountant can sort and chart;
//   - a **comparison with the previous period of the same length**, which is
//     the sentence the owner actually reads.
//
// ⚠️ **Takings use the same rule as the dashboard** (`received`): money counts
// when it has arrived, not when the order was placed. Two screens that both say
// "tushum" and disagree are worse than one screen, and this system has already
// made that mistake once — see the note on `received`.
//
// ⚠️ **Buckets are cut in local time, in Go, not with `$dateToString` in
// Mongo.** The driver hands back every date in UTC (§ "Mongo'dan kelgan sana
// doim UTC"), so a Tashkent restaurant's evening trade — everything after
// 19:00 — would land in the next day's bucket. That failure is invisible: the
// chart still looks like a month of business, just the wrong month, shifted.
// Bucketing here, after `.In(time.Local)`, means the day boundary is the one
// the kitchen worked to.

// salesGroup is how the period is cut up.
type salesGroup string

const (
	groupDay   salesGroup = "day"
	groupWeek  salesGroup = "week"
	groupMonth salesGroup = "month"
)

// parseSalesGroup narrows the query parameter to a bucket size.
//
// An unknown value falls back to days rather than erroring: this parameter
// comes from a link an operator may have edited or bookmarked, and a report
// that refuses to draw is a worse answer than a report drawn at the finest
// granularity available.
func parseSalesGroup(s string) salesGroup {
	switch salesGroup(strings.ToLower(strings.TrimSpace(s))) {
	case groupWeek:
		return groupWeek
	case groupMonth:
		return groupMonth
	}
	return groupDay
}

// salesBucket is one row: a slice of time and what happened in it.
type salesBucket struct {
	// "2026-08-14" for a day, "2026-08-10" (the Monday) for a week,
	// "2026-08" for a month. Sortable as a string in every case, which is what
	// keeps the ordering right without a second date field.
	Key string `json:"key"`
	// What the panel prints. Separate from Key because "14.08" is readable and
	// "2026-08-14" is sortable, and asking one string to be both produces a
	// chart axis nobody can read.
	Label string `json:"label"`
	// Orders placed in the bucket, cancellations included — the count is
	// "how much work arrived", and a cancelled order still arrived.
	Orders    int `json:"orders"`
	Cancelled int `json:"cancelled"`
	// Orders whose money is in hand, and what it came to.
	Paid    int `json:"paid"`
	Revenue int `json:"revenue"`
	// Revenue ÷ Paid. Divided by the orders the money came from rather than by
	// every order placed — otherwise the average bill sinks as the kitchen gets
	// busier, which is the opposite of what it is being read for.
	AvgCheck int `json:"avgCheck"`
	// Still out: placed, not cancelled, not yet collected. Kept apart from
	// Revenue for the reason the dashboard learned the hard way — it is real
	// money, and it is not takings yet.
	Pending     int `json:"pending"`
	Discounts   int `json:"discounts"`
	DeliveryFee int `json:"deliveryFee"`
	Delivery    int `json:"delivery"`
	Pickup      int `json:"pickup"`
	DineIn      int `json:"dineIn"`
	// Dishes sold in the bucket. On the "sold" basis, like ABC/XYZ: a dish in
	// a confirmed order has sold even if the courier is still out with the
	// cash. Deliberately a different basis from Revenue, and the report says so.
	Items int `json:"items"`
}

// salesTotals is the period as one row, plus the comparison that gives it a
// direction.
type salesTotals struct {
	Orders      int `json:"orders"`
	Cancelled   int `json:"cancelled"`
	Paid        int `json:"paid"`
	Revenue     int `json:"revenue"`
	AvgCheck    int `json:"avgCheck"`
	Pending     int `json:"pending"`
	Discounts   int `json:"discounts"`
	DeliveryFee int `json:"deliveryFee"`
	Delivery    int `json:"delivery"`
	Pickup      int `json:"pickup"`
	DineIn      int `json:"dineIn"`
	Items       int `json:"items"`
}

// salesChange is "compared with the period before this one".
type salesChange struct {
	// The window that was compared against, so the reader can check what "the
	// previous period" was rather than infer it. A comparison whose baseline is
	// implied is one nobody can argue with, which is not the same as one that
	// is right.
	From string `json:"from"`
	To   string `json:"to"`
	// The previous period's figures, whole — the panel decides what to show.
	Previous salesTotals `json:"previous"`
	// Percent change, one entry per figure that has one. Absent when the
	// previous period was zero: "up 100%" from nothing is not information, and
	// dividing by it is how a dashboard starts printing ∞.
	Percent map[string]float64 `json:"percent"`
}

// salesHour is one hour of the day, summed across the whole period.
type salesHour struct {
	Hour    int `json:"hour"`
	Orders  int `json:"orders"`
	Revenue int `json:"revenue"`
}

// AdminSalesReport is sales over time, on screen and in a spreadsheet.
func (h *Handler) AdminSalesReport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to, err := parseRange(q.Get("from"), q.Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	group := parseSalesGroup(q.Get("group"))

	branchScope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	orders, err := h.ordersInRange(r, branchScope, from, to)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	buckets, totals := summariseSales(orders, group)
	hours := salesByHour(orders)

	lang := reportLang(r)
	rep := &Report{
		Title:   salesTitle(group, lang),
		Slug:    "savdo-" + string(group),
		From:    dayOrAll(from, lang),
		To:      dayOrAll(to, lang),
		Note:    salesNote(lang),
		Columns: salesColumns(group, lang),
		Rows:    salesRows(buckets),
		Totals:  salesTotalsRow(totals, lang),
	}
	if wantsExcel(r) {
		h.respondReport(w, r, rep)
		return
	}

	// The comparison is screen-only on purpose. A spreadsheet the accountant
	// sorts and filters must be one period of one thing; a second period's
	// figures sitting in it as extra rows would be added into every column
	// total the moment somebody selected the sheet.
	resp := map[string]any{
		"from":    rep.From,
		"to":      rep.To,
		"group":   string(group),
		"note":    rep.Note,
		"buckets": buckets,
		"totals":  totals,
		"byHour":  hours,
		"best":    bestBucket(buckets),
	}
	if cmp := h.compareSales(r, branchScope, from, to); cmp != nil {
		resp["compare"] = cmp.withPercent(totals)
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// ordersInRange loads the period's orders for a branch scope.
//
// Loaded rather than aggregated because every figure below needs a different
// slice of the same order — the type split, the money basis, the hour it
// arrived, the dishes on it — and five aggregations over the same documents
// cost more than one read. The period is bounded, which is what keeps this
// honest: the mistake this codebase already made was loading the *last 200*
// orders and calling it a period.
func (h *Handler) ordersInRange(r *http.Request, scope bson.M, from, to *time.Time) ([]models.Order, error) {
	return h.ordersBetween(r.Context(), scope, from, to)
}

// ordersBetween is the same read with no request behind it, for the callers
// that have none — the morning briefing gathers its facts on a context alone.
//
// ⚠️ **The filter lives here and only here.** It is the reason a dining-room
// sale appears in the sales report, the channel report and the financial report
// for the same reason a delivery does: the period and the scope, and nothing
// else. Adding `check` to it — the obvious "let us make the screens consistent"
// move, since the orders board rightly has that line — would take the room out
// of the restaurant's own revenue, silently, on the number an owner carries to
// the bank.
func (h *Handler) ordersBetween(
	ctx context.Context, scope bson.M, from, to *time.Time,
) ([]models.Order, error) {
	filter := bson.M{}
	for k, v := range scope {
		filter[k] = v
	}
	rng := bson.M{}
	if from != nil {
		rng["$gte"] = *from
	}
	if to != nil {
		rng["$lt"] = *to
	}
	if len(rng) > 0 {
		filter["createdAt"] = rng
	}
	cur, err := h.Store.Orders.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	var orders []models.Order
	if err := cur.All(ctx, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

// bucketKey places a moment in its bucket, in local time.
func bucketKey(t time.Time, group salesGroup) (key, label string) {
	local := t.In(time.Local)
	switch group {
	case groupMonth:
		return local.Format("2006-01"), local.Format("01.2006")
	case groupWeek:
		// The Monday of that week. Monday rather than Sunday because that is
		// the week a restaurant's staff schedule and payroll already run on,
		// and two different week boundaries in one panel is a reconciliation
		// nobody can win.
		offset := (int(local.Weekday()) + 6) % 7
		monday := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local).
			AddDate(0, 0, -offset)
		return monday.Format("2006-01-02"), monday.Format("02.01")
	}
	return local.Format("2006-01-02"), local.Format("02.01")
}

// summariseSales folds the orders into buckets and a period total.
func summariseSales(orders []models.Order, group salesGroup) ([]salesBucket, salesTotals) {
	byKey := map[string]*salesBucket{}
	var totals salesTotals

	for _, o := range orders {
		key, label := bucketKey(o.CreatedAt, group)
		b := byKey[key]
		if b == nil {
			b = &salesBucket{Key: key, Label: label}
			byKey[key] = b
		}

		b.Orders++
		totals.Orders++

		// Dishes sold: the "sold" basis, so only a cancellation un-sells them.
		if o.Status != models.StatusCancelled {
			for _, it := range o.Items {
				b.Items += it.Qty
				totals.Items += it.Qty
			}
		}

		switch {
		case o.Status == models.StatusCancelled:
			b.Cancelled++
			totals.Cancelled++
			// A cancelled order is counted as work that arrived and as nothing
			// else: no takings, no pending, no channel split. Splitting it by
			// type would put money-shaped rows next to an order that will
			// never produce any.
			continue
		case o.PaymentStatus == models.PayRefunded:
			// Came in and went back out. Not takings, and not still owed to us
			// either — it belongs on the finance report's refund line, not here.
			continue
		case received(o):
			b.Paid++
			b.Revenue += o.Total
			b.Discounts += o.DiscountTotal
			b.DeliveryFee += o.DeliveryFee
			totals.Paid++
			totals.Revenue += o.Total
			totals.Discounts += o.DiscountTotal
			totals.DeliveryFee += o.DeliveryFee
		default:
			b.Pending += o.Total
			totals.Pending += o.Total
		}

		switch o.Type {
		case "pickup":
			b.Pickup++
			totals.Pickup++
		case "dinein":
			b.DineIn++
			totals.DineIn++
		default:
			b.Delivery++
			totals.Delivery++
		}
	}

	out := make([]salesBucket, 0, len(byKey))
	for _, b := range byKey {
		b.AvgCheck = avgCheck(b.Revenue, b.Paid)
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })

	totals.AvgCheck = avgCheck(totals.Revenue, totals.Paid)
	return out, totals
}

// avgCheck divides takings by the orders the takings came from.
func avgCheck(revenue, paid int) int {
	if paid == 0 {
		return 0
	}
	return revenue / paid
}

// salesByHour is the period's trade by hour of day, in local time.
//
// Its use is a staffing question, not a money one: "when do we need a second
// courier on". So it is summed across the whole period rather than shown per
// day — one Friday says nothing, thirty Fridays say when the rush is.
func salesByHour(orders []models.Order) []salesHour {
	hours := make([]salesHour, 24)
	for i := range hours {
		hours[i].Hour = i
	}
	for _, o := range orders {
		if o.Status == models.StatusCancelled {
			continue
		}
		h := o.CreatedAt.In(time.Local).Hour()
		hours[h].Orders++
		if received(o) {
			hours[h].Revenue += o.Total
		}
	}
	return hours
}

// bestBucket is the period's best bucket by takings.
//
// Returned rather than left to the panel because "your best day was Saturday
// the 9th" is the one line of this report somebody repeats out loud, and a
// panel that has to scan the array to find it will eventually scan it by a
// different column than the label claims.
func bestBucket(buckets []salesBucket) *salesBucket {
	var best *salesBucket
	for i := range buckets {
		if best == nil || buckets[i].Revenue > best.Revenue {
			best = &buckets[i]
		}
	}
	if best == nil || best.Revenue == 0 {
		return nil
	}
	b := *best
	return &b
}

// compareSales totals the period of the same length immediately before this one.
//
// Returns nil when the period is open-ended: "all time" has nothing before it,
// and inventing a baseline for it would produce a growth figure that looks like
// every other growth figure on the screen and means nothing.
func (h *Handler) compareSales(r *http.Request, scope bson.M, from, to *time.Time) *salesChange {
	if from == nil || to == nil {
		return nil
	}
	span := to.Sub(*from)
	if span <= 0 {
		return nil
	}
	prevTo := *from
	prevFrom := from.Add(-span)

	orders, err := h.ordersInRange(r, scope, &prevFrom, &prevTo)
	if err != nil {
		// A comparison is an extra, not the report. Failing the whole request
		// because the baseline query fell over would hide the numbers the
		// owner actually asked for.
		return nil
	}
	_, prev := summariseSales(orders, groupDay)

	return &salesChange{
		From:     prevFrom.In(time.Local).Format("2006-01-02"),
		To:       prevTo.In(time.Local).AddDate(0, 0, -1).Format("2006-01-02"),
		Previous: prev,
	}
}

// percentChange is the growth from old to new, or nothing when there is no
// baseline to grow from.
func percentChange(old, now int) (float64, bool) {
	if old == 0 {
		return 0, false
	}
	return float64(now-old) / float64(old) * 100, true
}

// withPercent fills in the change map for the figures that have a baseline.
func (c *salesChange) withPercent(now salesTotals) *salesChange {
	c.Percent = map[string]float64{}
	for key, pair := range map[string][2]int{
		"orders":    {c.Previous.Orders, now.Orders},
		"revenue":   {c.Previous.Revenue, now.Revenue},
		"paid":      {c.Previous.Paid, now.Paid},
		"avgCheck":  {c.Previous.AvgCheck, now.AvgCheck},
		"items":     {c.Previous.Items, now.Items},
		"cancelled": {c.Previous.Cancelled, now.Cancelled},
	} {
		if pct, ok := percentChange(pair[0], pair[1]); ok {
			c.Percent[key] = pct
		}
	}
	return c
}

func salesTitle(group salesGroup, lang string) string {
	switch group {
	case groupWeek:
		return tr{"Savdo hisoboti (haftalik)", "Отчёт о продажах (по неделям)",
			"Sales report (weekly)"}.in(lang)
	case groupMonth:
		return tr{"Savdo hisoboti (oylik)", "Отчёт о продажах (по месяцам)",
			"Sales report (monthly)"}.in(lang)
	}
	return tr{"Savdo hisoboti (kunlik)", "Отчёт о продажах (по дням)",
		"Sales report (daily)"}.in(lang)
}

func salesColumns(group salesGroup, lang string) []Column {
	period := tr{"Kun", "День", "Day"}
	switch group {
	case groupWeek:
		period = tr{"Hafta", "Неделя", "Week"}
	case groupMonth:
		period = tr{"Oy", "Месяц", "Month"}
	}
	return []Column{
		{Key: "period", Title: period.in(lang), Kind: ColText},
		{Key: "orders", Title: tr{"Buyurtma", "Заказы", "Orders"}.in(lang), Kind: ColInt},
		{Key: "paid", Title: tr{"To'langan", "Оплачено", "Paid"}.in(lang), Kind: ColInt},
		{Key: "cancelled", Title: tr{"Bekor", "Отменено", "Cancelled"}.in(lang), Kind: ColInt},
		{Key: "revenue", Title: tr{"Tushum", "Выручка", "Revenue"}.in(lang), Kind: ColMoney},
		{Key: "avgCheck", Title: tr{"O'rtacha chek", "Средний чек", "Average bill"}.in(lang), Kind: ColMoney},
		{Key: "discounts", Title: tr{"Chegirma", "Скидки", "Discounts"}.in(lang), Kind: ColMoney},
		{Key: "deliveryFee", Title: tr{"Yetkazish yig'imi", "Сбор за доставку", "Delivery fees"}.in(lang), Kind: ColMoney},
		{Key: "delivery", Title: tr{"Yetkazish", "Доставка", "Delivery"}.in(lang), Kind: ColInt},
		{Key: "pickup", Title: tr{"Olib ketish", "Самовывоз", "Pickup"}.in(lang), Kind: ColInt},
		{Key: "dineIn", Title: tr{"Stolda", "За столом", "Dine-in"}.in(lang), Kind: ColInt},
		{Key: "items", Title: tr{"Taom soni", "Блюд продано", "Dishes sold"}.in(lang), Kind: ColInt},
	}
}

func salesRows(buckets []salesBucket) []map[string]any {
	rows := make([]map[string]any, 0, len(buckets))
	for _, b := range buckets {
		rows = append(rows, map[string]any{
			"period": b.Label, "orders": b.Orders, "paid": b.Paid,
			"cancelled": b.Cancelled, "revenue": b.Revenue, "avgCheck": b.AvgCheck,
			"discounts": b.Discounts, "deliveryFee": b.DeliveryFee,
			"delivery": b.Delivery, "pickup": b.Pickup, "dineIn": b.DineIn,
			"items": b.Items,
		})
	}
	return rows
}

func salesTotalsRow(t salesTotals, lang string) map[string]any {
	return map[string]any{
		"period": trTotal.in(lang), "orders": t.Orders, "paid": t.Paid,
		"cancelled": t.Cancelled, "revenue": t.Revenue, "avgCheck": t.AvgCheck,
		"discounts": t.Discounts, "deliveryFee": t.DeliveryFee,
		"delivery": t.Delivery, "pickup": t.Pickup, "dineIn": t.DineIn,
		"items": t.Items,
	}
}

// salesNote says which basis each column is on, because two of them differ and
// the difference has already misled this dashboard once.
func salesNote(lang string) string {
	return tr{
		"Tushum pul kelganda hisoblanadi (naqd topshirilgan yoki bank tasdiqlagan), " +
			"buyurtma esa tushgan kuni sanaladi. O'rtacha chek — tushumni to'langan " +
			"buyurtmalar soniga bo'lish. Taom soni bekor qilinmagan buyurtmalar bo'yicha.",
		"Выручка считается, когда деньги получены (курьер сдал наличные или банк " +
			"подтвердил оплату), а заказ — в день оформления. Средний чек — выручка, " +
			"делённая на число оплаченных заказов. Блюда — по неотменённым заказам.",
		"Revenue is counted when the money arrives (cash handed in or confirmed by " +
			"the bank); an order is counted on the day it was placed. The average bill " +
			"is revenue divided by paid orders. Dishes exclude cancelled orders.",
	}.in(lang)
}
