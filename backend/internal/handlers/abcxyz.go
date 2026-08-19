package handlers

import (
	"math"
	"net/http"
	"sort"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ABC/XYZ: which dishes earn the money, and which of them can be planned.
//
// Two questions that look alike and are not, which is exactly why they are
// crossed rather than merged:
//
//   - **ABC** — how much of the takings a dish is responsible for. Pareto:
//     the dishes making the first 80% are A, the next 15% B, the last 5% C.
//   - **XYZ** — how *steady* its demand is, measured as the coefficient of
//     variation of daily sales. X is predictable, Z is erratic.
//
// The cross is where the decisions live, and neither axis gives them alone:
//
//   - **AX** — earns well and sells steadily. The dishes to never run out of,
//     and the ones whose price change is felt immediately.
//   - **AZ** — earns well but unpredictably. Worth stocking for, and worth
//     asking why: a dish that sells forty portions one Friday and none for a
//     week is usually riding something other than the menu.
//   - **CZ** — little money, no pattern. The candidates to drop; a shorter
//     menu is faster to cook and cheaper to hold stock for.
//
// ⚠️ **Counted on dishes sold, not on money collected.** This answers "what
// sells", and a dish in a confirmed order has sold even if the courier has not
// come back with the cash — the same basis the dashboard's top-dish list uses,
// and deliberately not the takings basis used for revenue. Only a cancellation
// un-sells a dish.

// abcRow is one dish's place in the analysis.
type abcRow struct {
	Name string `json:"name"`
	Qty  int    `json:"qty"`
	// So'm, at the price actually charged (dish + option deltas).
	Revenue int `json:"revenue"`
	// This dish's share of the period's takings, 0–100.
	Share float64 `json:"share"`
	// Running total of Share down the sorted list — the number the A/B/C cut
	// is actually made on, and the one worth showing: it is what turns "this
	// dish is 3%" into "these nine dishes are 80%".
	Cumulative float64 `json:"cumulative"`
	// Coefficient of variation of daily sales, as a percentage.
	Variation float64 `json:"variation"`
	ABC       string  `json:"abc"`
	XYZ       string  `json:"xyz"`
	// "AX", "CZ" — the pair, precomputed so the panel can filter on one field.
	Class string `json:"class"`
	// How many days of the period this dish sold on at all. The honesty check
	// on XYZ: a dish sold on two days out of thirty has a variation figure
	// that is arithmetically true and means nothing.
	Days int `json:"days"`

	// What the portions sold cost the kitchen, and what was left.
	//
	// ⚠️ **Both are absent unless somebody typed a cost for this dish.** Zero
	// would read as "free", which would make the worst-margin dish on the menu
	// look like the best — and the owner acts on this screen.
	Cost   int  `json:"cost,omitempty"`
	Margin int  `json:"margin,omitempty"`
	Costed bool `json:"costed,omitempty"`
}

// XYZ thresholds, as coefficients of variation in percent.
//
// The textbook figures are 10% and 25%, and they come from manufacturing —
// where demand is smoothed by contracts. A restaurant's daily dish counts are
// small integers, and small integers are noisy: at an average of three
// portions a day, one quiet Tuesday is a 30% swing. Using 10% here would put
// almost the entire menu in Z and say nothing.
const (
	xyzSteady   = 25.0 // ≤ 25% — X, plannable
	xyzVariable = 60.0 // ≤ 60% — Y; above that Z
)

// ABC cut points, as cumulative share of revenue.
const (
	abcA = 80.0
	abcB = 95.0
)

// AdminABCXYZ classifies the menu for a period.
func (h *Handler) AdminABCXYZ(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// The same brand/branch lens every other counted screen reads through.
	branchScope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	filter := bson.M{"status": bson.M{"$ne": models.StatusCancelled}}
	for k, v := range branchScope {
		filter[k] = v
	}
	created := bson.M{}
	if from != nil {
		created["$gte"] = *from
	}
	if to != nil {
		created["$lt"] = *to
	}
	if len(created) > 0 {
		filter["createdAt"] = created
	}

	cur, err := h.Store.Orders.Find(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var orders []models.Order
	if err := cur.All(r.Context(), &orders); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// ⚠️ Every dish the period sold, asked for at once. The alternative — a
	// lookup per row — is one query per dish on a menu of two hundred, on a
	// screen an owner opens while thinking about something else.
	ids := make([]primitive.ObjectID, 0, 64)
	seen := map[primitive.ObjectID]bool{}
	for _, o := range orders {
		for _, it := range o.Items {
			if !it.MenuItemID.IsZero() && !seen[it.MenuItemID] {
				seen[it.MenuItemID] = true
				ids = append(ids, it.MenuItemID)
			}
		}
	}
	rows, total := classify(orders, h.costLedgerFor(r.Context(), ids))
	lang := reportLang(r)
	costedRows, costedRevenue := costCoverage(rows, total)
	rep := &Report{
		Title:   "ABC-XYZ",
		Slug:    "abc-xyz",
		From:    dayOrAll(from, lang),
		To:      dayOrAll(to, lang),
		Note:    abcNote(lang) + costNote(lang, costedRows, len(rows), costedRevenue),
		Columns: abcColumns(lang, costedRows > 0),
		Rows:    abcReportRows(rows),
		Totals: map[string]any{
			"name":    trTotal.in(lang),
			"qty":     sumQty(rows),
			"revenue": total,
			"share":   100.0,
		},
	}
	if wantsExcel(r) {
		h.respondReport(w, r, rep)
		return
	}
	// On screen the panel wants the typed rows (it filters and colours by
	// class), plus the report envelope so the same period line is shown.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"from": rep.From, "to": rep.To, "note": rep.Note,
		"total": total, "items": rows,
	})
}

// classify does the whole analysis over a period's orders.
//
// Split out from the handler so it can be tested against a hand-built set of
// orders: every threshold below is a judgement call, and a judgement call that
// cannot be tested is one nobody dares change later.
func classify(orders []models.Order, costs *costLedger) ([]abcRow, int) {
	type acc struct {
		qty     int
		revenue int
		// What the portions sold cost, each at the prices of the day it was
		// sold. ⚠️ Accumulated rather than "cost × qty" at the end: a period
		// spanning a price rise has portions at two costs, and one figure
		// multiplied out would be wrong for both halves.
		cost int
		// Revenue of the portions that could actually be costed. ⚠️ The margin
		// is computed against **this**, not the dish's whole revenue: an order
		// line old enough to have no dish id cannot be costed, and charging its
		// revenue against the others' cost would report a margin nobody earned.
		costedRevenue int
		costed        bool
		// Portions per calendar day, for the variation figure.
		perDay map[string]int
	}
	dishes := map[string]*acc{}
	days := map[string]struct{}{}

	for _, o := range orders {
		day := o.CreatedAt.In(time.Local).Format("2006-01-02")
		days[day] = struct{}{}
		for _, it := range o.Items {
			d, ok := dishes[it.Name]
			if !ok {
				d = &acc{perDay: map[string]int{}}
				dishes[it.Name] = d
			}
			d.qty += it.Qty
			d.revenue += it.Price * it.Qty
			d.perDay[day] += it.Qty
			if costs != nil {
				if c, ok := costs.Cost(it.MenuItemID, o.CreatedAt); ok {
					d.cost += c * it.Qty
					d.costedRevenue += it.Price * it.Qty
					d.costed = true
				}
			}
		}
	}

	total := 0
	rows := make([]abcRow, 0, len(dishes))
	for name, d := range dishes {
		total += d.revenue
		row := abcRow{
			Name:      name,
			Qty:       d.qty,
			Revenue:   d.revenue,
			Days:      len(d.perDay),
			Variation: variation(d.perDay, len(days)),
		}
		if d.costed && d.cost > 0 {
			// ⚠️ Against the portions **sold**, not the menu price: a dish
			// given away at half price still cost the kitchen the same, and
			// that is exactly the case an owner is looking for here.
			row.Costed = true
			row.Cost = d.cost
			row.Margin = d.costedRevenue - d.cost
		}
		rows = append(rows, row)
	}

	// Best earner first: the cumulative share only means anything in this order.
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Revenue != rows[j].Revenue {
			return rows[i].Revenue > rows[j].Revenue
		}
		return rows[i].Name < rows[j].Name
	})

	running := 0.0
	for i := range rows {
		if total > 0 {
			rows[i].Share = float64(rows[i].Revenue) / float64(total) * 100
		}
		running += rows[i].Share
		rows[i].Cumulative = running

		// ⚠️ The cut is made on the cumulative share **before** this dish is
		// added, not after. Taking it after would push the dish that crosses
		// 80% into B — and on a short menu that dish is often a large part of
		// the takings, which is precisely the one an owner must not be told to
		// stop worrying about.
		switch {
		case running-rows[i].Share < abcA:
			rows[i].ABC = "A"
		case running-rows[i].Share < abcB:
			rows[i].ABC = "B"
		default:
			rows[i].ABC = "C"
		}

		switch {
		case rows[i].Variation <= xyzSteady:
			rows[i].XYZ = "X"
		case rows[i].Variation <= xyzVariable:
			rows[i].XYZ = "Y"
		default:
			rows[i].XYZ = "Z"
		}
		rows[i].Class = rows[i].ABC + rows[i].XYZ
	}
	return rows, total
}

// variation is the coefficient of variation of a dish's daily sales, in
// percent: the standard deviation over the mean.
//
// ⚠️ **Days the dish sold nothing count as zero**, which is why the number of
// days in the period is passed in rather than taken from the dish's own map.
// Averaging only the days it sold would make a dish that sells twenty portions
// one day a month look perfectly steady — the most erratic item on the menu
// reported as the most plannable.
func variation(perDay map[string]int, periodDays int) float64 {
	if periodDays <= 1 {
		// One day is not a series. Reporting 0% would read as "perfectly
		// steady" on the strength of a single observation.
		return 0
	}
	sum := 0
	for _, q := range perDay {
		sum += q
	}
	mean := float64(sum) / float64(periodDays)
	if mean == 0 {
		return 0
	}
	// Every day of the period contributes, including the silent ones.
	variance := 0.0
	for _, q := range perDay {
		d := float64(q) - mean
		variance += d * d
	}
	silent := periodDays - len(perDay)
	variance += float64(silent) * mean * mean
	variance /= float64(periodDays)

	return math.Sqrt(variance) / mean * 100
}

func sumQty(rows []abcRow) int {
	n := 0
	for _, r := range rows {
		n += r.Qty
	}
	return n
}

// abcColumns is the sheet's shape.
//
// ⚠️ **The cost columns appear only when somebody has typed a cost.** On the
// restaurants that never will — most of them, at first — two permanently empty
// columns are two columns to scroll past on every report, and an empty money
// column reads as zero rather than as unknown.
func abcColumns(lang string, costed bool) []Column {
	cols := []Column{
		{Key: "name", Title: tr{"Taom", "Блюдо", "Dish"}.in(lang), Kind: ColText},
		{Key: "qty", Title: tr{"Sotilgan", "Продано", "Sold"}.in(lang), Kind: ColInt},
		{Key: "revenue", Title: tr{"Tushum", "Выручка", "Revenue"}.in(lang), Kind: ColMoney},
		{Key: "share", Title: tr{"Ulushi", "Доля", "Share"}.in(lang), Kind: ColPercent},
		{Key: "cumulative", Title: tr{"Jamlanma", "Накопительно", "Cumulative"}.in(lang), Kind: ColPercent},
		{Key: "abc", Title: "ABC", Kind: ColText},
		{Key: "variation", Title: tr{"Tebranish", "Колебание", "Variation"}.in(lang), Kind: ColPercent},
		{Key: "xyz", Title: "XYZ", Kind: ColText},
		{Key: "class", Title: tr{"Sinf", "Класс", "Class"}.in(lang), Kind: ColText},
		{Key: "days", Title: tr{"Sotilgan kun", "Дней с продажами", "Days sold"}.in(lang), Kind: ColInt},
	}
	if costed {
		// After the revenue, where the eye already is.
		cols = append(cols[:3], append([]Column{
			{Key: "cost", Title: tr{"Tannarx", "Себестоимость", "Cost"}.in(lang), Kind: ColMoney},
			{Key: "margin", Title: tr{"Yalpi foyda", "Валовая прибыль", "Gross margin"}.in(lang), Kind: ColMoney},
		}, cols[3:]...)...)
	}
	return cols
}

func abcReportRows(rows []abcRow) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"name": r.Name, "qty": r.Qty, "revenue": r.Revenue,
			"share": round1(r.Share), "cumulative": round1(r.Cumulative),
			"abc": r.ABC, "variation": round1(r.Variation),
			"xyz": r.XYZ, "class": r.Class, "days": r.Days,
		})
		// ⚠️ Absent, not zero, when nobody typed a cost for this dish. A zero
		// in a money column is read as "free", which would make the worst
		// margin on the menu look like the best — and this is a screen people
		// act on.
		if r.Costed {
			out[len(out)-1]["cost"] = r.Cost
			out[len(out)-1]["margin"] = r.Margin
		}
	}
	return out
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }

// dayOrAll renders a period bound, or "all of it" when it is open.
//
// An empty string would leave the sheet's period line reading " — 2026-08-07",
// which is the kind of half-answer somebody prints and then has to explain.
func dayOrAll(t *time.Time, lang string) string {
	if t == nil {
		return tr{"hammasi", "всё время", "all time"}.in(lang)
	}
	return t.In(time.Local).Format("2006-01-02")
}

// abcNote is the sentence under the title. It says what was counted, because
// "sold" and "collected" are different numbers here and the difference has
// already misled this dashboard once.
// costCoverage is how much of the report the cost figures actually cover.
//
// ⚠️ **The number nobody would think to ask for, and the one that decides
// whether the margin means anything.** A restaurant that priced ten dishes out
// of two hundred has a gross-margin column that is arithmetically correct and
// describes 6% of the evening. Saying so is the difference between a report and
// a misleading report.
func costCoverage(rows []abcRow, total int) (n int, revenueShare float64) {
	costed := 0
	for _, r := range rows {
		if r.Costed {
			n++
			costed += r.Revenue
		}
	}
	if total > 0 {
		revenueShare = float64(costed) / float64(total) * 100
	}
	return n, revenueShare
}

func costNote(lang string, costed, all int, revenueShare float64) string {
	if costed == 0 {
		// ⚠️ Said even when nothing is costed, because the absence is the
		// question: an owner looking for a profit column has to be told where
		// the number comes from, not left to conclude the feature is missing.
		return " " + tr{
			"Tannarx kiritilmagan — yalpi foyda ko'rsatilmaydi (menyu formasidagi «Tannarx» maydoni).",
			"Себестоимость не заполнена — валовая прибыль не показывается (поле «Себестоимость» в блюде).",
			"No costs entered — gross margin is not shown (the dish form's cost field).",
		}.in(lang)
	}
	if costed == all {
		return ""
	}
	return " " + tr{
		"Tannarx " + itoa(costed) + "/" + itoa(all) + " taomda kiritilgan — yalpi foyda tushumning " + itoa(int(revenueShare+0.5)) + "% ini qamraydi.",
		"Себестоимость указана у " + itoa(costed) + " из " + itoa(all) + " блюд — прибыль охватывает " + itoa(int(revenueShare+0.5)) + "% выручки.",
		"Cost is set on " + itoa(costed) + " of " + itoa(all) + " dishes — the margin covers " + itoa(int(revenueShare+0.5)) + "% of revenue.",
	}.in(lang)
}

func abcNote(lang string) string {
	return tr{
		"Bekor qilinganlardan tashqari sotilgan taomlar bo'yicha. " +
			"Pul olingani emas, sotilgani hisoblanadi.",
		"По проданным блюдам, кроме отменённых заказов. " +
			"Считается проданное, а не полученные деньги.",
		"Dishes sold, excluding cancelled orders. " +
			"Counted on what was sold, not on money collected.",
	}.in(lang)
}
