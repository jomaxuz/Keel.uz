package handlers

import (
	"net/http"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// The money report: what came in, what went out, and what is still owed to us.
//
// ⚠️ **This is not a profit and loss statement, and it must never be read as
// one.** There is no cost of goods in this system — no ingredient prices, no
// recipe cards — so the difference between takings and outgoings here is
// *cash movement*, not profit. A restaurant that treats it as profit is
// overstating by the entire cost of its food, which for a kitchen is most of
// the number.
//
// That is stated on the report itself rather than only here. A financial
// figure whose meaning lives in a code comment is a figure somebody will
// eventually put in a bank application.
//
// The three sections answer three different questions, and they are kept apart
// because mixing them is how the old dashboard came to call an uncooked order
// "revenue":
//
//   - **Kirim** — money actually in hand for the period.
//   - **Chiqim** — money actually handed out in the period.
//   - **Kutilayotgan** — placed, not cancelled, not yet collected. Real, and
//     deliberately outside the totals.

type finLine struct {
	Label  string `json:"label"`
	Amount int    `json:"amount"`
	// Orders/payments behind the figure, so a line can be checked rather than
	// believed. Zero means the line is not a count of anything.
	Count int `json:"count"`
	// "in" | "out" | "pending" — which of the three questions this answers.
	Kind string `json:"kind"`
	// Indented under the line above: the payment-method split under takings,
	// for instance. A flat list would read as if the parts and the whole were
	// peers and could be added together.
	Sub bool `json:"sub"`
}

// AdminFinanceReport is the period's money movement.
func (h *Handler) AdminFinanceReport(w http.ResponseWriter, r *http.Request) {
	lang := reportLang(r)
	from, to, err := parseRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	branchScope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	within := func(base bson.M, field string) bson.M {
		f := bson.M{}
		for k, v := range base {
			f[k] = v
		}
		rng := bson.M{}
		if from != nil {
			rng["$gte"] = *from
		}
		if to != nil {
			rng["$lt"] = *to
		}
		if len(rng) > 0 {
			f[field] = rng
		}
		return f
	}

	orderFilter := bson.M{}
	for k, v := range branchScope {
		orderFilter[k] = v
	}
	cur, err := h.Store.Orders.Find(r.Context(), within(orderFilter, "createdAt"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var orders []models.Order
	if err := cur.All(r.Context(), &orders); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	lines, in, out, pending := financeLines(orders, lang)

	// ---- What was handed out ----
	//
	// Read from the ledgers rather than recomputed: a wage is what was
	// actually paid, not what the schedule says was owed, and the two differ
	// exactly when somebody needs this report.
	payroll, payrollN := h.sumAmounts(r, h.Store.StaffPayments, within(bson.M{}, "at"))
	if payroll > 0 {
		lines = append(lines, finLine{
			Label:  tr{"Ishchilarga to'langan", "Выплаты сотрудникам", "Paid to staff"}.in(lang),
			Amount: payroll, Count: payrollN, Kind: "out"})
		out += payroll
	}
	// ⚠️ A courier settlement is **not** an outgoing: it is cash the courier
	// collected on our behalf moving into the till. Counting it here would
	// subtract the restaurant's own takings from itself.
	//
	// What the couriers actually cost is their pay, and that is not recorded
	// as a payment anywhere yet — so it is absent rather than guessed. See the
	// note on the report.

	external, externalN := externalDeliveryCost(orders)
	if external > 0 {
		lines = append(lines, finLine{
			Label:  tr{"Tashqi yetkazish xizmati", "Внешняя служба доставки", "External delivery service"}.in(lang),
			Amount: external, Count: externalN, Kind: "out"})
		out += external
	}

	rep := &Report{
		Title:   tr{"Moliyaviy hisobot", "Финансовый отчёт", "Financial report"}.in(lang),
		Slug:    "moliya",
		From:    dayOrAll(from, lang),
		To:      dayOrAll(to, lang),
		Note:    financeNote(lang),
		Columns: financeColumns(lang),
		Rows:    financeRows(lines, lang),
		Totals: map[string]any{
			"label":  tr{"Kirim − chiqim", "Приход − расход", "In − out"}.in(lang),
			"amount": in - out,
		},
	}
	if wantsExcel(r) {
		h.respondReport(w, r, rep)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"from": rep.From, "to": rep.To, "note": rep.Note,
		"lines": lines,
		"totals": map[string]int{
			"in": in, "out": out, "net": in - out, "pending": pending,
		},
	})
}

// financeLines turns a period's orders into the takings side of the report.
func financeLines(orders []models.Order, lang string) (lines []finLine, in, out, pending int) {
	var (
		revenue, delivered, pickup, dinein        int
		nDelivered, nPickup, nDinein              int
		byMethod                                  = map[string]int{}
		nByMethod                                 = map[string]int{}
		deliveryFees, discounts, points, refunded int
		nRefunded, nPending                       int
	)

	for _, o := range orders {
		switch {
		case o.PaymentStatus == models.PayRefunded:
			// Money that came in and went back out. Shown on its own line
			// rather than netted off silently: a month with ten refunds and a
			// month with none can otherwise look identical.
			refunded += o.Total
			nRefunded++
			continue
		case received(o):
			revenue += o.Total
			deliveryFees += o.DeliveryFee
			discounts += o.DiscountTotal
			points += o.PointsSpent
			byMethod[o.PaymentMethod] += o.Total
			nByMethod[o.PaymentMethod]++
			switch o.Type {
			case "pickup":
				pickup += o.Total
				nPickup++
			case "dinein":
				dinein += o.Total
				nDinein++
			default:
				delivered += o.Total
				nDelivered++
			}
		case o.Status != models.StatusCancelled:
			pending += o.Total
			nPending++
		}
	}

	lines = append(lines, finLine{
		Label:  tr{"Tushum", "Выручка", "Revenue"}.in(lang),
		Amount: revenue, Count: nDelivered + nPickup + nDinein, Kind: "in"})
	// The channel split, indented: these are parts of the line above, not
	// separate income.
	for _, c := range []struct {
		label  tr
		amount int
		n      int
	}{
		{tr{"— yetkazib berish", "— доставка", "— delivery"}, delivered, nDelivered},
		{tr{"— olib ketish", "— самовывоз", "— pickup"}, pickup, nPickup},
		{tr{"— stolda", "— за столом", "— dine-in"}, dinein, nDinein},
	} {
		if c.amount > 0 {
			lines = append(lines, finLine{Label: c.label.in(lang), Amount: c.amount, Count: c.n, Kind: "in", Sub: true})
		}
	}
	for _, m := range []string{
		models.ProviderCash, models.ProviderPayme, models.ProviderClick,
		models.ProviderUzum, models.ProviderAtmos,
	} {
		if byMethod[m] > 0 {
			lines = append(lines, finLine{
				Label: "— " + m, Amount: byMethod[m], Count: nByMethod[m], Kind: "in", Sub: true,
			})
		}
	}
	if deliveryFees > 0 {
		lines = append(lines, finLine{
			Label:  tr{"Shundan yetkazish yig'imi", "Из них сбор за доставку", "Of which delivery fees"}.in(lang),
			Amount: deliveryFees, Kind: "in", Sub: true})
	}
	in = revenue

	if refunded > 0 {
		lines = append(lines, finLine{
			Label:  tr{"Qaytarilgan to'lovlar", "Возвраты", "Refunds"}.in(lang),
			Amount: refunded, Count: nRefunded, Kind: "out"})
		out += refunded
	}
	// ⚠️ Discounts and points are **not** outgoings. No money left the till —
	// it never arrived. Listing them as costs would double-count against the
	// takings line, which is already net of them. They are here because an
	// owner does want to know what the campaigns gave away.
	if discounts > 0 {
		lines = append(lines, finLine{
			Label:  tr{"Chegirmalar (pul chiqmagan)", "Скидки (деньги не выходили)", "Discounts (no money left)"}.in(lang),
			Amount: discounts, Kind: "info"})
	}
	if points > 0 {
		lines = append(lines, finLine{
			Label:  tr{"Ballar bilan to'langan (pul chiqmagan)", "Оплачено баллами (деньги не выходили)", "Paid with points (no money left)"}.in(lang),
			Amount: points, Kind: "info"})
	}
	if pending > 0 {
		lines = append(lines, finLine{
			Label:  tr{"Kutilayotgan (hali olinmagan)", "Ожидается (ещё не получено)", "Still out (not collected yet)"}.in(lang),
			Amount: pending, Count: nPending, Kind: "pending"})
	}
	return lines, in, out, pending
}

func externalDeliveryCost(orders []models.Order) (total, n int) {
	for _, o := range orders {
		if o.ExternalDelivery != nil && o.ExternalDelivery.Cost > 0 {
			total += o.ExternalDelivery.Cost
			n++
		}
	}
	return total, n
}

// sumAmounts adds up the `amount` field of a ledger collection, in the database.
//
// Aggregated rather than fetched and summed here for the reason the dashboard
// already learned once: a report that loads rows to add them up is correct
// until the restaurant has more rows than the page expected, and then it is
// quietly wrong in the direction of understating.
func (h *Handler) sumAmounts(r *http.Request, coll *mongo.Collection, filter bson.M) (int, int) {
	cur, err := coll.Aggregate(r.Context(), mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.M{
			"_id": nil, "total": bson.M{"$sum": "$amount"}, "n": bson.M{"$sum": 1},
		}}},
	})
	if err != nil {
		return 0, 0
	}
	var rows []struct {
		Total int `bson:"total"`
		N     int `bson:"n"`
	}
	if err := cur.All(r.Context(), &rows); err != nil || len(rows) == 0 {
		return 0, 0
	}
	return rows[0].Total, rows[0].N
}

func financeColumns(lang string) []Column {
	return []Column{
		{Key: "label", Title: tr{"Modda", "Статья", "Item"}.in(lang), Kind: ColText},
		{Key: "amount", Title: tr{"Summa", "Сумма", "Amount"}.in(lang), Kind: ColMoney},
		{Key: "count", Title: tr{"Soni", "Количество", "Count"}.in(lang), Kind: ColInt},
		{Key: "kind", Title: tr{"Turi", "Тип", "Kind"}.in(lang), Kind: ColText},
	}
}

func financeRows(lines []finLine, lang string) []map[string]any {
	out := make([]map[string]any, 0, len(lines))
	for _, l := range lines {
		row := map[string]any{"label": l.Label, "amount": l.Amount, "kind": financeKindLabel(l.Kind, lang)}
		if l.Count > 0 {
			row["count"] = l.Count
		}
		out = append(out, row)
	}
	return out
}

func financeKindLabel(kind, lang string) string {
	switch kind {
	case "in":
		return tr{"kirim", "приход", "in"}.in(lang)
	case "out":
		return tr{"chiqim", "расход", "out"}.in(lang)
	case "pending":
		return tr{"kutilmoqda", "ожидается", "pending"}.in(lang)
	}
	return tr{"ma'lumot", "справочно", "info"}.in(lang)
}

// financeNote is the sentence that keeps this report honest.
func financeNote(lang string) string {
	return tr{
		"⚠️ Bu foyda hisoboti EMAS: tizimda taom tannarxi yo'q, " +
			"shuning uchun \"kirim − chiqim\" pul harakati, foyda emas. " +
			"Tushum pul kelganda hisoblanadi (naqd topshirilgan yoki bank tasdiqlagan).",
		"⚠️ Это НЕ отчёт о прибыли: себестоимости блюд в системе нет, поэтому " +
			"«приход − расход» — движение денег, а не прибыль. " +
			"Выручка считается, когда деньги получены (сданы наличными или подтверждены банком).",
		"⚠️ This is NOT a profit report: the system holds no dish cost, so " +
			"\"in − out\" is cash movement, not profit. " +
			"Revenue is counted when the money arrives (cash handed in or confirmed by the bank).",
	}.in(lang)
}
