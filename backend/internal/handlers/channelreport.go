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

// Channel analytics: which door the orders came in through.
//
// The restaurant pays for several of these doors separately — a bot token, a
// batch of printed table QR codes, an operator's shift, a site that has to be
// found in search — and until now the panel could not tell it whether any of
// them worked. "Half our orders come from Telegram" and "nobody has ever used
// it" are the same restaurant with the same bot, and they lead to opposite
// decisions about next month's money.
//
// ⚠️ **Two questions, not one, and they are deliberately kept apart:**
//
//   - **Kanal** (`order.channel`) — how the order was *placed*: the site, the
//     Telegram mini app, or an operator typing it in. This is what the
//     marketing spend maps onto.
//   - **Turi** (`order.type`) — how it was *fulfilled*: delivery, pickup, or a
//     table. This is what the kitchen and the couriers map onto.
//
// Collapsing them into one list would produce rows like "Telegram" next to
// "olib ketish" that overlap without saying so — a guest can, and often does,
// order pickup through the bot. Every such row is double-counted and the
// column adds up to more than the business did.
//
// ⚠️ **`channel` is attribution, not authorisation** (see the field's own
// note): the mini app *is* the site rendered inside Telegram, so the client
// says which it is. A guest who lies mislabels their own order and nothing
// else — which is fine for a report and would not be fine for a permission.
// The one value never taken on trust is `operator`, set server-side.

// channelRow is one door and what came through it.
type channelRow struct {
	// Stable id ("web", "telegram", "android", "operator", "unknown"); the panel
	// translates it. Not the label, for the same reason audit-log actions are
	// ids: a report filtered by a translated string breaks the day somebody
	// switches the panel to Russian.
	Key    string `json:"key"`
	Orders int    `json:"orders"`
	// Money in hand from this channel, on the `received` basis — the same rule
	// as everywhere else that says "tushum".
	Revenue int `json:"revenue"`
	Paid    int `json:"paid"`
	// Revenue ÷ Paid. The number that makes this report worth reading: a
	// channel bringing a third of the orders at half the average bill is a
	// different business decision from one bringing a third at double.
	AvgCheck  int `json:"avgCheck"`
	Cancelled int `json:"cancelled"`
	// Share of the period's orders, 0–100. On orders rather than on takings
	// because a channel's job is to bring people; what they then spend is the
	// next column along and reading them together is the point.
	Share float64 `json:"share"`
	// Distinct customers who ordered through this channel in the period, by
	// account. Zero for guests without one, which is why it sits next to
	// Orders rather than replacing it.
	Customers int `json:"customers"`
	// First-time customers: their first order in the whole history falls
	// inside this period. The closest this system gets to "what did this
	// channel actually win us", and the reason the acquisition question is
	// answerable at all.
	NewCustomers int `json:"newCustomers"`
}

// AdminChannelReport is the period's trade split by channel and by type.
func (h *Handler) AdminChannelReport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to, err := parseRange(q.Get("from"), q.Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
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

	// Who was new: decided from each customer's *first ever* order, not from
	// the first one inside the window. Otherwise every customer looks new in
	// every period, and the most loyal restaurant in the city reads as one
	// that cannot keep anybody.
	firstOrder, err := h.firstOrderTimes(r, branchScope)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	channels := channelRows(orders, orderChannelKey, firstOrder, from)
	types := channelRows(orders, orderTypeKey, firstOrder, from)

	lang := reportLang(r)
	rep := &Report{
		Title:   tr{"Kanal analitikasi", "Аналитика каналов", "Channel analytics"}.in(lang),
		Slug:    "channels",
		From:    dayOrAll(from, lang),
		To:      dayOrAll(to, lang),
		Note:    channelNote(lang),
		Columns: channelColumns(lang),
		Rows:    channelReportRows(channels, types, lang),
	}
	if wantsExcel(r) {
		h.respondReport(w, r, rep)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"from": rep.From, "to": rep.To, "note": rep.Note,
		"channels": channels,
		"types":    types,
	})
}

// orderChannelKey is how the order was placed.
//
// Empty means the order predates the field. Reported as "unknown" rather than
// quietly folded into "web": most of them probably were the site, but a report
// that guesses is one whose oldest rows cannot be told from its newest, and the
// owner comparing this month with last year would never know where the line is.
func orderChannelKey(o models.Order) string {
	switch o.Channel {
	case "web", "telegram", "operator", "android":
		return o.Channel
	case "":
		// An operator-taken order from before the field existed still names
		// itself, because the operator's name was always written down.
		if o.TakenBy != "" {
			return "operator"
		}
		return "unknown"
	}
	return o.Channel
}

// orderTypeKey is how the order was fulfilled.
func orderTypeKey(o models.Order) string {
	switch o.Type {
	case "pickup", "dinein":
		return o.Type
	}
	return "delivery"
}

// channelRows folds orders into rows by whatever key function is given.
func channelRows(
	orders []models.Order,
	key func(models.Order) string,
	firstOrder map[string]time.Time,
	from *time.Time,
) []channelRow {
	type acc struct {
		row channelRow
		// Distinct customers, counted by account id — and by phone for guests
		// who ordered without one, so a table QR that nobody signs in through
		// does not report zero customers on a hundred orders.
		seen map[string]bool
		new  map[string]bool
	}
	byKey := map[string]*acc{}
	total := 0

	for _, o := range orders {
		k := key(o)
		a := byKey[k]
		if a == nil {
			a = &acc{row: channelRow{Key: k}, seen: map[string]bool{}, new: map[string]bool{}}
			byKey[k] = a
		}
		a.row.Orders++
		total++

		if id := customerKey(o); id != "" {
			a.seen[id] = true
			if first, ok := firstOrder[id]; ok && (from == nil || !first.Before(*from)) {
				a.new[id] = true
			}
		}

		switch {
		case o.Status == models.StatusCancelled:
			a.row.Cancelled++
		case o.PaymentStatus == models.PayRefunded:
			// Neither takings nor owed: it belongs on the finance report.
		case received(o):
			a.row.Paid++
			a.row.Revenue += o.Total
		}
	}

	out := make([]channelRow, 0, len(byKey))
	for _, a := range byKey {
		a.row.Customers = len(a.seen)
		a.row.NewCustomers = len(a.new)
		a.row.AvgCheck = avgCheck(a.row.Revenue, a.row.Paid)
		if total > 0 {
			a.row.Share = float64(a.row.Orders) / float64(total) * 100
		}
		out = append(out, a.row)
	}
	// Busiest first: this is a ranking, and the first row is the answer to
	// "where do our orders come from".
	sort.Slice(out, func(i, j int) bool {
		if out[i].Orders != out[j].Orders {
			return out[i].Orders > out[j].Orders
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// customerKey identifies the person behind an order.
//
// ⚠️ The account id when there is one, the phone number otherwise — and never
// a bare `o.UserID` truthiness test. An unset `ObjectID` marshals to
// "000000000000000000000000" rather than disappearing (§ "Bo'sh ObjectID
// JSON'da yo'qolmaydi"), so every guest order would otherwise be counted as
// the same one very busy customer.
func customerKey(o models.Order) string {
	if !o.UserID.IsZero() {
		return o.UserID.Hex()
	}
	return o.Customer.Phone
}

// firstOrderTimes is each customer's first order, over the whole history.
//
// Aggregated in Mongo rather than loaded: this reads every order the branch has
// ever taken, and that is exactly the query that must not come back as
// documents. Scoped to the same branches as the report — a manager pinned to
// one branch must not learn when a customer first ordered from another.
func (h *Handler) firstOrderTimes(r *http.Request, scope bson.M) (map[string]time.Time, error) {
	match := bson.M{"status": bson.M{"$ne": models.StatusCancelled}}
	for k, v := range scope {
		match[k] = v
	}
	// Group by account when there is one and by phone otherwise — the same
	// identity `customerKey` uses, so the two cannot drift into disagreeing
	// about who is new.
	cur, err := h.Store.Orders.Aggregate(r.Context(), mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{"$cond": bson.A{
				bson.M{"$ifNull": bson.A{"$userId", false}},
				bson.M{"$toString": "$userId"},
				"$customer.phone",
			}},
			"first": bson.M{"$min": "$createdAt"},
		}}},
	})
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID    string    `bson:"_id"`
		First time.Time `bson:"first"`
	}
	if err := cur.All(r.Context(), &rows); err != nil {
		return nil, err
	}
	out := make(map[string]time.Time, len(rows))
	for _, row := range rows {
		if row.ID == "" {
			continue
		}
		// ⚠️ Mongo hands every date back in UTC regardless of the process's
		// zone, and this is compared against a local-time period bound.
		out[row.ID] = row.First.In(time.Local)
	}
	return out, nil
}

func channelColumns(lang string) []Column {
	return []Column{
		{Key: "group", Title: tr{"Kesim", "Разрез", "Cut"}.in(lang), Kind: ColText},
		{Key: "name", Title: tr{"Nomi", "Название", "Name"}.in(lang), Kind: ColText},
		{Key: "orders", Title: tr{"Buyurtma", "Заказы", "Orders"}.in(lang), Kind: ColInt},
		{Key: "share", Title: tr{"Ulushi", "Доля", "Share"}.in(lang), Kind: ColPercent},
		{Key: "revenue", Title: tr{"Tushum", "Выручка", "Revenue"}.in(lang), Kind: ColMoney},
		{Key: "avgCheck", Title: tr{"O'rtacha chek", "Средний чек", "Average bill"}.in(lang), Kind: ColMoney},
		{Key: "cancelled", Title: tr{"Bekor", "Отменено", "Cancelled"}.in(lang), Kind: ColInt},
		{Key: "customers", Title: tr{"Mijozlar", "Клиенты", "Customers"}.in(lang), Kind: ColInt},
		{Key: "newCustomers", Title: tr{"Yangi mijoz", "Новые клиенты", "New customers"}.in(lang), Kind: ColInt},
	}
}

// channelLabels are the spreadsheet's words. The panel has its own translated
// versions; these exist because a downloaded file is read outside the panel,
// often by somebody who has never seen it — and it is read in the language the
// panel was set to when it was exported.
var channelLabels = map[string]tr{
	"web":      {"Sayt", "Сайт", "Website"},
	"telegram": {"Telegram", "Telegram", "Telegram"},
	// ⚠️ Named by what a guest holds, not by the platform: "Android" is a word
	// an owner reads as "the phone app", which is the thing they bought.
	"android": {"Ilova (Android)", "Приложение (Android)", "App (Android)"},
	"operator": {"Operator (telefon)", "Оператор (телефон)", "Operator (phone)"},
	"unknown":  {"Noma'lum (eski buyurtmalar)", "Неизвестно (старые заказы)", "Unknown (older orders)"},
	"delivery": {"Yetkazib berish", "Доставка", "Delivery"},
	"pickup":   {"Olib ketish", "Самовывоз", "Pickup"},
	"dinein":   {"Stolda (QR)", "За столом (QR)", "Dine-in (QR)"},
}

func channelLabel(key, lang string) string {
	if l, ok := channelLabels[key]; ok {
		return l.in(lang)
	}
	return key
}

// channelReportRows puts both cuts in one sheet, each under its own heading.
//
// One sheet rather than two because a spreadsheet with two tabs is one whose
// second tab is never opened; the "Kesim" column keeps the two from being
// summed together by anyone who selects the whole range.
func channelReportRows(channels, types []channelRow, lang string) []map[string]any {
	rows := make([]map[string]any, 0, len(channels)+len(types))
	add := func(group string, list []channelRow) {
		for _, c := range list {
			rows = append(rows, map[string]any{
				"group": group, "name": channelLabel(c.Key, lang), "orders": c.Orders,
				"share": c.Share, "revenue": c.Revenue, "avgCheck": c.AvgCheck,
				"cancelled": c.Cancelled, "customers": c.Customers,
				"newCustomers": c.NewCustomers,
			})
		}
	}
	add(tr{"Kanal", "Канал", "Channel"}.in(lang), channels)
	add(tr{"Turi", "Тип", "Type"}.in(lang), types)
	return rows
}

func channelNote(lang string) string {
	return tr{
		"Kanal — buyurtma qayerdan berilgani (sayt, Telegram, operator); " +
			"turi — qanday yetkazilgani (yetkazish, olib ketish, stolda). " +
			"Ikkisi bir-birini kesadi, shuning uchun alohida sanaladi — qo'shib bo'lmaydi. " +
			"\"Yangi mijoz\" — birinchi buyurtmasi shu davrga tushgan mijoz.",
		"Канал — откуда пришёл заказ (сайт, Telegram, оператор); " +
			"тип — как он выдан (доставка, самовывоз, за столом). " +
			"Они пересекаются, поэтому считаются отдельно и не складываются. " +
			"«Новый клиент» — тот, чей самый первый заказ попал в этот период.",
		"Channel is where the order came from (website, Telegram, operator); " +
			"type is how it was fulfilled (delivery, pickup, dine-in). " +
			"They overlap, so they are counted separately and must not be added together. " +
			"\"New customer\" means their very first order falls in this period.",
	}.in(lang)
}
