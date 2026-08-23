package handlers

import (
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- What came in against what the food used ----
//
// ⚠️ **This is deliberately not a stock balance, and the report says so.**
// There is no opening count, no write-offs and no record of what was thrown
// away, so any "remaining" figure would be a number nobody could check and
// everybody would believe. What *can* be said honestly is a flow: this month
// forty kilos of beef came through the door and the dishes sold account for
// twenty-six of them. That difference is a question worth asking — and unlike a
// balance, every part of it is measured.
//
// ⚠️ **Prep items are expanded to what they are made of.** A dish using a sauce
// consumes tomatoes, not "sauce": leaving it unexpanded would show tomatoes
// arriving and never being used, which is the shape of a theft report and the
// substance of an accounting bug.

type stockRow struct {
	Name string `json:"name"`
	Unit string `json:"unit"`
	// Purchase units — kilos, litres, pieces — because that is how it was
	// bought and how the person reading this counts it.
	In   float64 `json:"in"`
	Used float64 `json:"used"`
	// Thrown away, spilled, eaten by the staff — recorded by hand.
	//
	// ⚠️ Its own column rather than folded into `used`: one is what the cards
	// say the dishes took and the other is what somebody wrote down, and a
	// single figure would hide which of the two a difference came from.
	Written float64 `json:"written"`
	Diff    float64 `json:"diff"`
	// What came in cost, from the invoices themselves.
	Spent int `json:"spent"`
}

// AdminStockReport is the period's ingredient flow.
func (h *Handler) AdminStockReport(w http.ResponseWriter, r *http.Request) {
	lang := reportLang(r)
	from, to, err := parseRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx := r.Context()

	var ingredients []models.Ingredient
	if cur, err := h.Store.Ingredients.Find(ctx, bson.M{}); err == nil {
		_ = cur.All(ctx, &ingredients)
	}
	byID := map[primitive.ObjectID]models.Ingredient{}
	for _, in := range ingredients {
		byID[in.ID] = in
	}

	in, spent := h.deliveredInPeriod(r, scope, from, to)
	used := h.consumedInPeriod(r, scope, from, to, ingredients)
	written, writtenValue := h.writtenOffInPeriod(r, scope, from, to)

	rows := make([]stockRow, 0, len(byID))
	for id, ing := range byID {
		// A prep item is never bought and never "used" as itself — its inputs
		// carry both sides. Listing it would be two empty columns with a name.
		if ing.MadeInHouse() {
			continue
		}
		got, out, off := in[id], used[id], written[id]
		if got == 0 && out == 0 && off == 0 {
			continue
		}
		rows = append(rows, stockRow{
			Name: ing.Name, Unit: ing.Unit,
			In: round3(got), Used: round3(out), Written: round3(off),
			// ⚠️ Write-offs come off the difference: that is the whole point of
			// recording them. What is left is the part nobody has accounted
			// for — which is the only honest thing this column can be.
			Diff:  round3(got - out - off),
			Spent: spent[id],
		})
	}
	// Biggest spend first: the question this answers is "where did the money
	// go", and an alphabetical list buries it under the garnishes.
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Spent != rows[j].Spent {
			return rows[i].Spent > rows[j].Spent
		}
		return rows[i].Name < rows[j].Name
	})

	rep := &Report{
		Title:   tr{"Masalliqlar harakati", "Движение ингредиентов", "Ingredient flow"}.in(lang),
		Slug:    "ombor",
		From:    dayOrAll(from, lang),
		To:      dayOrAll(to, lang),
		Note:    stockNote(lang),
		Columns: stockColumns(lang),
		Rows:    stockReportRows(rows),
		Totals: map[string]any{
			"name":  trTotal.in(lang),
			"spent": sumSpent(rows),
		},
		// What the write-offs were worth, at the prices of the days they
		// happened. ⚠️ Under the table rather than in a column: it is one
		// number for the period, and repeating a running total on every row
		// invites it to be added up.
	}
	if wantsExcel(r) {
		h.respondReport(w, r, rep)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"from": rep.From, "to": rep.To, "note": rep.Note, "rows": rows,
		"spent": sumSpent(rows), "writtenValue": writtenValue,
	})
}

// writtenOffInPeriod totals what was thrown away, per ingredient.
func (h *Handler) writtenOffInPeriod(
	r *http.Request, scope bson.M, from, to *time.Time,
) (map[primitive.ObjectID]float64, int) {
	qty := map[primitive.ObjectID]float64{}
	value := 0
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
		filter["at"] = rng
	}
	cur, err := h.Store.WriteOffs.Find(r.Context(), filter)
	if err != nil {
		return qty, value
	}
	var rows []models.WriteOff
	if err := cur.All(r.Context(), &rows); err != nil {
		return qty, value
	}
	for _, x := range rows {
		qty[x.IngredientID] += x.Qty
		value += x.Value
	}
	return qty, value
}

// deliveredInPeriod totals what arrived, per ingredient.
func (h *Handler) deliveredInPeriod(
	r *http.Request, scope bson.M, from, to *time.Time,
) (map[primitive.ObjectID]float64, map[primitive.ObjectID]int) {
	qty := map[primitive.ObjectID]float64{}
	money := map[primitive.ObjectID]int{}
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
		filter["at"] = rng
	}
	cur, err := h.Store.Purchases.Find(r.Context(), filter)
	if err != nil {
		return qty, money
	}
	var rows []models.Purchase
	if err := cur.All(r.Context(), &rows); err != nil {
		return qty, money
	}
	for _, p := range rows {
		for _, l := range p.Lines {
			qty[l.IngredientID] += l.Qty
			money[l.IngredientID] += l.Sum()
		}
	}
	return qty, money
}

// consumedInPeriod totals what the dishes sold should have used.
//
// ⚠️ **"Should have", and the report says so.** It is what the cards describe,
// not what the kitchen did: a heavy hand, a dropped tray and a portion given to
// a regular are all real and none of them are here. That gap is the entire
// point of the difference column — it is the question, not the answer.
func (h *Handler) consumedInPeriod(
	r *http.Request, scope bson.M, from, to *time.Time, ingredients []models.Ingredient,
) map[primitive.ObjectID]float64 {
	used := map[primitive.ObjectID]float64{}
	orders, err := h.ordersInRange(r, scope, from, to)
	if err != nil {
		return used
	}
	sold := map[primitive.ObjectID]int{}
	// ⚠️ **What was poured, not just what was ordered.** A bar sells one vodka
	// in three measures, and until this was counted a hundred 100 ml pours took
	// exactly as much off the shelf as a hundred 40 ml ones — the price already
	// varied, only the stock did not, and the difference showed up once a month
	// as an unexplained shortfall. Keyed by dish **and** the choices that were
	// made, because that pair is what decides how much left the store.
	poured := map[optionKey]int{}
	for _, o := range orders {
		// A cancelled order was not cooked — the same basis the ABC report
		// counts on, so the two cannot disagree about what sold.
		if o.Status == models.StatusCancelled {
			continue
		}
		for _, it := range o.Items {
			if it.Live() && !it.MenuItemID.IsZero() {
				sold[it.MenuItemID] += it.Qty
				poured[optionKeyOf(it)] += it.Qty
			}
		}
	}
	if len(sold) == 0 {
		return used
	}
	ids := make([]primitive.ObjectID, 0, len(sold))
	for id := range sold {
		ids = append(ids, id)
	}
	var dishes []struct {
		ID      primitive.ObjectID  `bson:"_id"`
		Recipe  []models.RecipeLine `bson:"recipe"`
		Options []models.MenuOption `bson:"options"`
	}
	if cur, err := h.Store.Menu.Find(r.Context(),
		bson.M{"_id": bson.M{"$in": ids}}); err == nil {
		_ = cur.All(r.Context(), &dishes)
	}
	raw := rawInputs(ingredients)
	// Recipe units → purchase units: grams to kilos, millilitres to litres,
	// pieces to pieces.
	take := func(lines []models.RecipeLine, portions float64) {
		for _, l := range lines {
			for id, per := range raw[l.IngredientID] {
				div := float64(models.PerUnit(byUnit(ingredients, id)))
				used[id] += per * l.Qty * portions / div
			}
		}
	}
	for _, d := range dishes {
		take(d.Recipe, float64(sold[d.ID]))
		// ⚠️ The choices are looked up on the **current** menu rather than
		// frozen on the order, the same way the fiscal code is: a card corrected
		// this morning has to be right for the count taken this evening, and an
		// order carries the choice's name, not its recipe.
		if len(d.Options) == 0 {
			continue
		}
		for key, n := range poured {
			if key.dish != d.ID || n == 0 {
				continue
			}
			for _, line := range chosenRecipes(d.Options, key.choices) {
				take(line, float64(n))
			}
		}
	}
	return used
}

// optionKey names one dish sold with one particular set of choices.
//
// ⚠️ A string of the choices rather than the slice itself, because this is a map
// key: two lines of the same pour have to add together, and Go cannot compare
// slices. The group is included, not only the choice — "50" under "Hajm" and
// "50" under "Muzli" are different answers that happen to share a word.
type optionKey struct {
	dish    primitive.ObjectID
	choices string
}

func optionKeyOf(it models.OrderItem) optionKey {
	parts := make([]string, 0, len(it.Options))
	for _, o := range it.Options {
		parts = append(parts, o.Name+"\x00"+o.Choice)
	}
	// ⚠️ Sorted, so the same two choices ticked in a different order are one
	// key. A cashier and a waiter tick them in whatever order the screen shows,
	// and two keys for one pour would still add up correctly here — but the
	// stop list compares the same pair, and there it would not.
	sort.Strings(parts)
	return optionKey{dish: it.MenuItemID, choices: strings.Join(parts, "\x01")}
}

// chosenRecipes is what the ticked choices take out of the store.
//
// ⚠️ Matched by name, which is how an order stores a choice everywhere else in
// this system. Renaming a choice therefore stops accounting for pours sold
// under the old name — said plainly rather than worked around, because the
// alternative is freezing a recipe onto every order line and then having
// yesterday's correction never reach yesterday's count.
func chosenRecipes(groups []models.MenuOption, key string) [][]models.RecipeLine {
	if key == "" {
		return nil
	}
	picked := map[string]bool{}
	for _, p := range strings.Split(key, "\x01") {
		picked[p] = true
	}
	out := [][]models.RecipeLine{}
	for _, g := range groups {
		for _, c := range g.Choices {
			if len(c.Recipe) > 0 && picked[g.Name+"\x00"+c.Name] {
				out = append(out, c.Recipe)
			}
		}
	}
	return out
}

// rawInputs maps every ingredient to the bought things it is made of.
//
// ⚠️ **A prep item resolves to its inputs, however deep.** A dish uses sauce,
// the sauce uses stock, the stock uses bones: the bones are what left the
// store. Without this the tomatoes would arrive every week and never be used —
// a difference column that looks like theft and is arithmetic.
//
// ⚠️ Bounded like the rate resolver, and for the same reason: two cards can
// name each other. What cannot be resolved is simply absent, which shows up as
// an ingredient with deliveries and no usage — visible, and not a made-up
// number.
func rawInputs(ingredients []models.Ingredient) map[primitive.ObjectID]map[primitive.ObjectID]float64 {
	out := map[primitive.ObjectID]map[primitive.ObjectID]float64{}
	var made []models.Ingredient
	for _, in := range ingredients {
		if in.MadeInHouse() {
			made = append(made, in)
			continue
		}
		// A bought thing is one unit of itself.
		out[in.ID] = map[primitive.ObjectID]float64{in.ID: 1}
	}
	for range made {
		progress := false
		for _, in := range made {
			if _, done := out[in.ID]; done {
				continue
			}
			ready := true
			for _, l := range in.Recipe {
				if _, ok := out[l.IngredientID]; !ok {
					ready = false
					break
				}
			}
			if !ready || in.Output <= 0 {
				continue
			}
			mix := map[primitive.ObjectID]float64{}
			for _, l := range in.Recipe {
				for id, per := range out[l.IngredientID] {
					// Per one recipe unit of this prep item.
					mix[id] += per * l.Qty / in.Output
				}
			}
			out[in.ID] = mix
			progress = true
		}
		if !progress {
			break
		}
	}
	return out
}

func byUnit(ingredients []models.Ingredient, id primitive.ObjectID) string {
	for _, in := range ingredients {
		if in.ID == id {
			return in.Unit
		}
	}
	return models.UnitPcs
}

func round3(f float64) float64 { return math.Round(f*1000) / 1000 }

func sumSpent(rows []stockRow) int {
	n := 0
	for _, r := range rows {
		n += r.Spent
	}
	return n
}

func stockColumns(lang string) []Column {
	return []Column{
		{Key: "name", Title: tr{"Masalliq", "Ингредиент", "Ingredient"}.in(lang), Kind: ColText},
		{Key: "unit", Title: tr{"Birlik", "Единица", "Unit"}.in(lang), Kind: ColText},
		{Key: "in", Title: tr{"Kelgan", "Приход", "Delivered"}.in(lang), Kind: ColQty},
		{Key: "used", Title: tr{"Sarflangan (hisob bo'yicha)", "Расход (по расчёту)", "Used (by the cards)"}.in(lang), Kind: ColQty},
		{Key: "written", Title: tr{"Hisobdan chiqarilgan", "Списано", "Written off"}.in(lang), Kind: ColQty},
		{Key: "diff", Title: tr{"Farq", "Разница", "Difference"}.in(lang), Kind: ColQty},
		{Key: "spent", Title: tr{"Sarflangan pul", "Потрачено", "Spent"}.in(lang), Kind: ColMoney},
	}
}

func stockReportRows(rows []stockRow) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"name": r.Name, "unit": r.Unit, "in": r.In,
			"used": r.Used, "written": r.Written, "diff": r.Diff, "spent": r.Spent,
		})
	}
	return out
}

// stockNote is the sentence that keeps this from being read as a stock balance.
func stockNote(lang string) string {
	return tr{
		"⚠️ Bu ombor qoldig'i EMAS: boshlang'ich qoldiq va inventarizatsiya tizimda " +
			"yo'q. \"Sarflangan\" — sotilgan taomlarning texkartasi bo'yicha hisob, " +
			"oshxona haqiqatda ishlatgani emas. Farq — hisobdan chiqarilganidan keyin " +
			"ham hech kim tushuntirmagan qism.",
		"⚠️ Это НЕ остатки склада: начальных остатков и инвентаризации в системе нет. " +
			"«Расход» — расчёт по техкартам проданных блюд, а не то, что кухня " +
			"действительно израсходовала. Разница — то, что осталось необъяснённым " +
			"даже после списаний.",
		"⚠️ This is NOT a stock balance: there is no opening count and no stocktake " +
			"in the system. \"Used\" is what the cards of the dishes sold describe, not " +
			"what the kitchen actually consumed. The difference is what nobody has " +
			"accounted for, even after the write-offs.",
	}.in(lang)
}
