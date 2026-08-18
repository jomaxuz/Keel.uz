package handlers

import (
	"net/http"
	"sort"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Who did the work: one row per courier, one row per employee, for a period.
//
// Both of these already existed as **one person at a time** — a courier's card
// and an employee's calendar — and both are the right screen for the question
// "how is Aziz doing". Neither can answer the question an owner actually asks
// at the end of a month, which is "how are they doing *compared with each
// other*": that one needs every row on one page, sorted, and in a file the
// accountant can open.
//
// ⚠️ **The rules are not restated here.** A courier's earnings come from
// `courierEarning`, an employee's day from `buildDays`/`payForDay` — the same
// functions their own screens use. A report that re-derives pay is a report
// that will eventually disagree with the payslip, and when it does there is no
// way to tell which one the employee should be paid on.

// courierReportRow is one courier's period.
type courierReportRow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Orders actually delivered in the period.
	Delivered int `json:"delivered"`
	// What the courier earned, under their own payout rule.
	Earnings int `json:"earnings"`
	// Order value carried, for context: a courier on a fixed per-order rate
	// earns the same on a 40 000 and a 400 000 so'm order, and the difference
	// is the restaurant's risk, not theirs.
	Carried int `json:"carried"`
	// Cash collected in the period, and what is still on them right now.
	//
	// ⚠️ These two are on different clocks and that is deliberate: `Cash` is a
	// period figure, `CashInHand` is a fact about this moment (everything ever
	// collected less everything ever handed back). Making the second one
	// period-bounded would produce a "debt" that resets every month whether or
	// not the money came back.
	Cash       int `json:"cash"`
	CashInHand int `json:"cashInHand"`
	// Minutes from "on the way" to "delivered", averaged. The courier's own
	// leg of the journey, not the kitchen's: an order that sat unconfirmed for
	// forty minutes is a problem, and it is not this person's problem.
	//
	// Zero when no order in the period recorded both events — an honest zero,
	// which is why `TimedOrders` sits next to it.
	AvgMinutes  int `json:"avgMinutes"`
	TimedOrders int `json:"timedOrders"`
	// Still assigned and undelivered right now.
	Active bool `json:"active"`
}

// AdminCourierReport is every courier's period, on screen and in a spreadsheet.
func (h *Handler) AdminCourierReport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to, err := parseRange(q.Get("from"), q.Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	scope, s, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// ⚠️ The courier list is narrowed to the same branches as the orders.
	// `RequireRole` only proves the caller is *some* admin; a manager pinned to
	// one branch must not learn the other branches' couriers by name, let alone
	// what they earn.
	courierFilter := s.branchFilter(bson.M{}, nil)
	cur, err := h.Store.Couriers.Find(r.Context(), courierFilter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var couriers []models.Courier
	if err := cur.All(r.Context(), &couriers); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	orders, err := h.ordersInRange(r, scope, from, to)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	settled, err := h.settledByCourier(r)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Cash ever collected, over the whole history — the other half of "what do
	// they still owe us", and the reason that column cannot be computed from
	// the period's orders alone.
	collected, err := h.cashByCourier(r, scope)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	rows := courierReportRows(couriers, orders, settled, collected)

	lang := reportLang(r)
	rep := &Report{
		Title:   tr{"Kuryerlar hisoboti", "Отчёт по курьерам", "Courier report"}.in(lang),
		Slug:    "kuryerlar",
		From:    dayOrAll(from, lang),
		To:      dayOrAll(to, lang),
		Note:    courierReportNote(lang),
		Columns: courierReportColumns(lang),
		Rows:    courierReportSheet(rows),
		Totals:  courierReportTotals(rows, lang),
	}
	if wantsExcel(r) {
		h.respondReport(w, r, rep)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"from": rep.From, "to": rep.To, "note": rep.Note, "couriers": rows,
	})
}

// courierReportRows folds the period's orders onto the couriers who carried them.
func courierReportRows(
	couriers []models.Courier,
	orders []models.Order,
	settled, collected map[string]int,
) []courierReportRow {
	byID := map[string]*models.Courier{}
	rows := map[string]*courierReportRow{}
	for i := range couriers {
		c := &couriers[i]
		id := c.ID.Hex()
		byID[id] = c
		rows[id] = &courierReportRow{ID: id, Name: c.Name}
	}

	// Minutes on the road, accumulated so the average can be taken at the end.
	minutes := map[string]int{}

	for i := range orders {
		o := &orders[i]
		if o.CourierID.IsZero() {
			continue
		}
		id := o.CourierID.Hex()
		row := rows[id]
		c := byID[id]
		if row == nil || c == nil {
			// A courier outside this admin's branches, or one whose account was
			// deleted. Skipped rather than shown under a blank name: a row
			// nobody can act on is a row that makes the total look wrong.
			continue
		}
		if o.Status != models.StatusDelivered {
			row.Active = true
			continue
		}

		row.Delivered++
		row.Earnings += courierEarning(o, c)
		row.Carried += o.Total
		if o.PaymentMethod == models.ProviderCash {
			row.Cash += o.Total
		}
		if mins, ok := onTheRoadMinutes(o); ok {
			minutes[id] += mins
			row.TimedOrders++
		}
	}

	out := make([]courierReportRow, 0, len(rows))
	for id, row := range rows {
		if row.TimedOrders > 0 {
			row.AvgMinutes = minutes[id] / row.TimedOrders
		}
		row.CashInHand = collected[id] - settled[id]
		out = append(out, *row)
	}
	// Most deliveries first: this is a comparison, and the person who did the
	// most work is the first row of it.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Delivered != out[j].Delivered {
			return out[i].Delivered > out[j].Delivered
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// onTheRoadMinutes is how long the courier's own leg took.
//
// Measured from `on_the_way` to `delivered` in the status history, and only
// when both are there. The alternative — order creation to delivery — folds in
// the time the ticket spent waiting to be confirmed and the time it spent in
// the kitchen, and then reports the total under a courier's name.
//
// ⚠️ The history is scanned for the **last** of each: an order pushed back and
// re-dispatched has two `on_the_way` events, and taking the first would bill
// the courier for the gap in between.
func onTheRoadMinutes(o *models.Order) (int, bool) {
	var out, done *time.Time
	for i := range o.StatusHistory {
		ev := o.StatusHistory[i]
		switch ev.Status {
		case models.StatusOnTheWay:
			at := ev.At
			out = &at
			// A re-dispatch invalidates the earlier delivery, if any.
			done = nil
		case models.StatusDelivered:
			at := ev.At
			done = &at
		}
	}
	if out == nil || done == nil || !done.After(*out) {
		return 0, false
	}
	return int(done.Sub(*out).Minutes()), true
}

// settledByCourier is how much cash each courier has handed back, ever.
func (h *Handler) settledByCourier(r *http.Request) (map[string]int, error) {
	return h.sumByObjectID(r, h.Store.Settlements, bson.M{}, "$courierId", "$amount")
}

// cashByCourier is how much cash each courier has collected, ever.
//
// Over the whole history rather than the period, because it is one half of
// "what do they still owe us" and the other half — the settlements — is also
// lifetime. Two figures on different clocks subtract into a number that means
// nothing.
func (h *Handler) cashByCourier(r *http.Request, scope bson.M) (map[string]int, error) {
	filter := bson.M{
		"status":        models.StatusDelivered,
		"paymentMethod": models.ProviderCash,
		"courierId":     bson.M{"$exists": true},
	}
	for k, v := range scope {
		filter[k] = v
	}
	return h.sumByObjectID(r, h.Store.Orders, filter, "$courierId", "$total")
}

// sumByObjectID groups a collection by an ObjectID field and sums another.
//
// In the database rather than in Go: this reads a restaurant's entire delivery
// history, and that is precisely the query that must not come back as
// documents (§ "VPS resurslari").
func (h *Handler) sumByObjectID(
	r *http.Request, coll *mongo.Collection, filter bson.M, groupBy, sum string,
) (map[string]int, error) {
	cur, err := coll.Aggregate(r.Context(), mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.M{"_id": groupBy, "total": bson.M{"$sum": sum}}}},
	})
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID    any `bson:"_id"`
		Total int `bson:"total"`
	}
	if err := cur.All(r.Context(), &rows); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		if id, ok := row.ID.(interface{ Hex() string }); ok {
			out[id.Hex()] = row.Total
		}
	}
	return out, nil
}

func courierReportColumns(lang string) []Column {
	return []Column{
		{Key: "name", Title: tr{"Kuryer", "Курьер", "Courier"}.in(lang), Kind: ColText},
		{Key: "delivered", Title: tr{"Yetkazgan", "Доставлено", "Delivered"}.in(lang), Kind: ColInt},
		{Key: "avgMinutes", Title: tr{"O'rtacha vaqt (daq)", "Среднее время (мин)", "Average time (min)"}.in(lang), Kind: ColInt},
		{Key: "earnings", Title: tr{"Daromadi", "Заработок", "Earnings"}.in(lang), Kind: ColMoney},
		{Key: "carried", Title: tr{"Tashigan summa", "Сумма заказов", "Order value carried"}.in(lang), Kind: ColMoney},
		{Key: "cash", Title: tr{"Naqd yig'gan", "Собрано наличными", "Cash collected"}.in(lang), Kind: ColMoney},
		{Key: "cashInHand", Title: tr{"Qo'lida (hozir)", "На руках (сейчас)", "In hand (now)"}.in(lang), Kind: ColMoney},
	}
}

func courierReportSheet(rows []courierReportRow) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, c := range rows {
		row := map[string]any{
			"name": c.Name, "delivered": c.Delivered, "earnings": c.Earnings,
			"carried": c.Carried, "cash": c.Cash, "cashInHand": c.CashInHand,
		}
		// ⚠️ Left blank rather than written as 0 when nothing was timed. A 0
		// in an average column reads as "instant delivery", which is the one
		// thing it certainly does not mean.
		if c.TimedOrders > 0 {
			row["avgMinutes"] = c.AvgMinutes
		}
		out = append(out, row)
	}
	return out
}

func courierReportTotals(rows []courierReportRow, lang string) map[string]any {
	var delivered, earnings, carried, cash, inHand int
	for _, c := range rows {
		delivered += c.Delivered
		earnings += c.Earnings
		carried += c.Carried
		cash += c.Cash
		inHand += c.CashInHand
	}
	// No average in the totals row: an average of averages is not the average,
	// and it would be wrong in proportion to how unevenly the work was shared.
	return map[string]any{
		"name": trTotal.in(lang), "delivered": delivered, "earnings": earnings,
		"carried": carried, "cash": cash, "cashInHand": inHand,
	}
}

func courierReportNote(lang string) string {
	return tr{
		"O'rtacha vaqt — \"yo'lga chiqdi\" dan \"yetkazildi\" gacha, ya'ni faqat " +
			"kuryerning yo'li (oshxonada kutgan vaqt kirmaydi). " +
			"\"Qo'lida\" — butun tarix bo'yicha yig'ilgan naqd minus topshirilgani, " +
			"ya'ni davrga bog'liq emas: hozirgi qarz.",
		"Среднее время — от «в пути» до «доставлен», то есть только дорога курьера " +
			"(ожидание на кухне не входит). «На руках» — вся собранная за всю историю " +
			"наличность минус сданная, поэтому от периода не зависит: это долг на сейчас.",
		"Average time is from \"on the way\" to \"delivered\" — the courier's own leg " +
			"only, with the wait in the kitchen excluded. \"In hand\" is all cash ever " +
			"collected less all cash ever handed in, so it does not depend on the " +
			"period: it is what is owed right now.",
	}.in(lang)
}

// ---- Employees ----

// staffReportRow is one employee's period.
type staffReportRow struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position string `json:"position"`
	// Days with any attendance, and rostered days nobody came in for.
	Days   int `json:"days"`
	Absent int `json:"absent"`
	// Minutes: what the roster asked for, and what was clocked.
	Expected int `json:"expected"`
	Worked   int `json:"worked"`
	Overtime int `json:"overtime"`
	Shortage int `json:"shortage"`
	// Pay the period earned under the employee's own rule, and what has
	// actually been handed over in it.
	//
	// ⚠️ Two different facts, kept apart. `Pay` is a calculation from the
	// calendar; `Paid` is a ledger of money that left the till. An owner asking
	// "did we settle up with the kitchen in March" needs the second, and a
	// report that showed only the first would answer a question nobody asked
	// while looking like it had answered theirs.
	Pay  int `json:"pay"`
	Paid int `json:"paid"`
}

// AdminStaffReport is every employee's attendance and pay for a period.
func (h *Handler) AdminStaffReport(w http.ResponseWriter, r *http.Request) {
	from, to, err := dateRange(r.URL.Query())
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	_, s, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	cur, err := h.Store.Staff.Find(r.Context(), s.branchFilter(bson.M{}, nil))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var people []models.Staff
	if err := cur.All(r.Context(), &people); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	fromKey, toKey := dayKey(from), dayKey(to)
	rows := make([]staffReportRow, 0, len(people))
	for i := range people {
		p := &people[i]
		shifts, err := h.shiftsInRange(r.Context(), p.ID, fromKey, toKey)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		// The same builder the employee's own calendar uses, so the report and
		// the calendar cannot disagree about a day.
		totals := sumDays(buildDays(p, shifts, from, to))
		paid, err := h.paidBetween(r.Context(), p.ID, fromKey, toKey)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		rows = append(rows, staffReportRow{
			ID: p.ID.Hex(), Name: p.Name, Position: p.Position,
			Days: totals.Days, Absent: totals.Absent,
			Expected: totals.Expected, Worked: totals.Worked,
			Overtime: totals.Overtime, Shortage: totals.Shortage,
			Pay: totals.Pay, Paid: paid,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Worked != rows[j].Worked {
			return rows[i].Worked > rows[j].Worked
		}
		return rows[i].Name < rows[j].Name
	})

	lang := reportLang(r)
	rep := &Report{
		Title:   tr{"Ishchilar hisoboti", "Отчёт по сотрудникам", "Staff report"}.in(lang),
		Slug:    "ishchilar",
		From:    fromKey,
		To:      toKey,
		Note:    staffReportNote(lang),
		Columns: staffReportColumns(lang),
		Rows:    staffReportSheet(rows),
		Totals:  staffReportTotals(rows, lang),
	}
	if wantsExcel(r) {
		h.respondReport(w, r, rep)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"from": rep.From, "to": rep.To, "note": rep.Note, "staff": rows,
	})
}

func staffReportColumns(lang string) []Column {
	return []Column{
		{Key: "name", Title: tr{"Ishchi", "Сотрудник", "Employee"}.in(lang), Kind: ColText},
		{Key: "position", Title: tr{"Lavozimi", "Должность", "Position"}.in(lang), Kind: ColText},
		{Key: "days", Title: tr{"Ishlagan kun", "Отработано дней", "Days worked"}.in(lang), Kind: ColInt},
		{Key: "absent", Title: tr{"Chiqmagan", "Не вышел", "Absent"}.in(lang), Kind: ColInt},
		// Hours rather than minutes in the sheet: minutes are what the code
		// stores and hours are what a wage is discussed in.
		{Key: "expectedH", Title: tr{"Grafik (soat)", "График (часы)", "Scheduled (hours)"}.in(lang), Kind: ColInt},
		{Key: "workedH", Title: tr{"Ishlangan (soat)", "Отработано (часы)", "Worked (hours)"}.in(lang), Kind: ColInt},
		{Key: "overtimeH", Title: tr{"Ortiqcha (soat)", "Переработка (часы)", "Overtime (hours)"}.in(lang), Kind: ColInt},
		{Key: "shortageH", Title: tr{"Kam (soat)", "Недоработка (часы)", "Short (hours)"}.in(lang), Kind: ColInt},
		{Key: "pay", Title: tr{"Hisoblangan", "Начислено", "Accrued"}.in(lang), Kind: ColMoney},
		{Key: "paid", Title: tr{"To'langan", "Выплачено", "Paid"}.in(lang), Kind: ColMoney},
	}
}

// hours rounds minutes to whole hours for the spreadsheet.
func hours(minutes int) int { return minutes / 60 }

func staffReportSheet(rows []staffReportRow) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, s := range rows {
		out = append(out, map[string]any{
			"name": s.Name, "position": s.Position, "days": s.Days, "absent": s.Absent,
			"expectedH": hours(s.Expected), "workedH": hours(s.Worked),
			"overtimeH": hours(s.Overtime), "shortageH": hours(s.Shortage),
			"pay": s.Pay, "paid": s.Paid,
		})
	}
	return out
}

func staffReportTotals(rows []staffReportRow, lang string) map[string]any {
	var days, absent, expected, worked, overtime, shortage, pay, paid int
	for _, s := range rows {
		days += s.Days
		absent += s.Absent
		expected += s.Expected
		worked += s.Worked
		overtime += s.Overtime
		shortage += s.Shortage
		pay += s.Pay
		paid += s.Paid
	}
	return map[string]any{
		"name": trTotal.in(lang), "days": days, "absent": absent,
		"expectedH": hours(expected), "workedH": hours(worked),
		"overtimeH": hours(overtime), "shortageH": hours(shortage),
		"pay": pay, "paid": paid,
	}
}

func staffReportNote(lang string) string {
	return tr{
		"\"Hisoblangan\" — davomat va ish haqi qoidasidan chiqqan summa; " +
			"\"To'langan\" — shu davrda haqiqatan berilgan pul. Ikkisi teng bo'lishi shart emas. " +
			"Soatlar butun songacha yaxlitlangan, hisob-kitob esa daqiqada yuritiladi.",
		"«Начислено» — сумма, посчитанная по посещаемости и правилу оплаты; " +
			"«Выплачено» — деньги, реально выданные за этот период. Они не обязаны совпадать. " +
			"Часы округлены до целых, а расчёт ведётся в минутах.",
		"\"Accrued\" is what the attendance and the pay rule add up to; \"Paid\" is the " +
			"money actually handed over in this period. The two need not match. " +
			"Hours are rounded to whole numbers; the calculation itself runs in minutes.",
	}.in(lang)
}
