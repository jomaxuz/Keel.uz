package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The till: what should be in the drawer, what is, and the difference.
//
// ⚠️ **The difference is the product.** Everything else here — the float, the
// cash sales, the payouts — exists only to produce a number the till can be
// counted against. A screen that shows "expected: 1 240 000" and a box to type
// what was found, storing only the second, has recorded nothing at all: the
// shortfall it was built to surface has been overwritten by the person who
// might have caused it.
//
// So: `expected` is frozen at the moment of closing, `counted` is what the
// drawer held, `variance` is stored rather than recalculated on read, and a
// non-zero variance cannot be saved without a sentence explaining it.
//
// Where the cash comes from, and the one that is easy to get wrong:
//
//   - dine-in and pickup orders paid in cash — handed over at the till;
//   - **courier settlements** — cash the courier collected on delivery and has
//     now brought back. ⚠️ This is why delivered cash orders are *not* counted
//     directly: until the courier hands it over the money is in their pocket,
//     not in the drawer, and counting both would double every delivery.

// AdminCashShift returns the open shift with its running figures, or the last
// closed one when nothing is open.
func (h *Handler) AdminCashShift(w http.ResponseWriter, r *http.Request) {
	branchScope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	shift, err := h.openCashShift(r, branchScope)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if shift == nil {
		// No open shift. The last closed one is still the useful answer: it
		// says when the till was last counted and how it came out.
		var last models.CashShift
		opts := options.FindOne().SetSort(bson.D{{Key: "openedAt", Value: -1}})
		_ = h.Store.CashShifts.FindOne(r.Context(), scopeFilter(branchScope), opts).Decode(&last)
		httpx.JSON(w, http.StatusOK, map[string]any{
			"open": nil, "last": nilIfZero(&last),
		})
		return
	}

	sum, entries, err := h.shiftFigures(r, shift)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"open": shift, "figures": sum, "entries": entries,
	})
}

// cashFigures is the arithmetic behind "what should be in the drawer".
type cashFigures struct {
	OpeningFloat int `json:"openingFloat"`
	// Cash taken at the counter: dine-in and pickup orders settled in cash.
	CounterCash int `json:"counterCash"`
	CounterN    int `json:"counterOrders"`
	// Cash couriers have handed back during this shift.
	Settlements  int `json:"settlements"`
	SettlementsN int `json:"settlementCount"`
	ManualIn     int `json:"manualIn"`
	ManualOut    int `json:"manualOut"`
	// The sum of all of the above. What the till should hold right now.
	Expected int `json:"expected"`
	// Cash on deliveries that have gone out but not been settled — money that
	// is real, in a courier's pocket, and deliberately **not** in Expected.
	// Shown because an owner counting a short till usually wants this number
	// before they start asking anyone difficult questions.
	WithCouriers int `json:"withCouriers"`
}

func (h *Handler) shiftFigures(r *http.Request, shift *models.CashShift) (cashFigures, []models.CashEntry, error) {
	f := cashFigures{OpeningFloat: shift.OpeningFloat}
	ctx := r.Context()
	since := bson.M{"$gte": shift.OpenedAt}

	scope := bson.M{}
	if !shift.BranchID.IsZero() {
		scope["branchId"] = shift.BranchID
	}

	// Counter cash: paid, cash, and not a delivery. A delivery's cash reaches
	// the till through the courier, not directly.
	counter := bson.M{
		"paymentMethod": models.ProviderCash,
		"paymentStatus": models.PayPaid,
		"type":          bson.M{"$in": []string{"pickup", "dinein"}},
		"paidAt":        since,
	}
	for k, v := range scope {
		counter[k] = v
	}
	if total, n, err := h.sumField(ctx, h.Store.Orders, counter, "$total"); err == nil {
		f.CounterCash, f.CounterN = total, n
	}

	settle := bson.M{"at": since}
	if total, n, err := h.sumField(ctx, h.Store.Settlements, settle, "$amount"); err == nil {
		f.Settlements, f.SettlementsN = total, n
	}

	entries := []models.CashEntry{}
	cur, err := h.Store.CashEntries.Find(ctx, bson.M{"shiftId": shift.ID},
		options.Find().SetSort(bson.D{{Key: "at", Value: 1}}))
	if err == nil {
		_ = cur.All(ctx, &entries)
	}
	for _, e := range entries {
		if e.Kind == models.CashOut {
			f.ManualOut += e.Amount
		} else {
			f.ManualIn += e.Amount
		}
	}

	// Cash already collected by couriers and not yet handed over: everything
	// they collected on delivered cash orders, less everything they have
	// settled. Read the same way the courier screen reads it, so the two
	// cannot disagree about who is holding what.
	f.WithCouriers = h.cashWithCouriers(ctx, scope)

	f.Expected = f.OpeningFloat + f.CounterCash + f.Settlements + f.ManualIn - f.ManualOut
	return f, entries, nil
}

// AdminOpenCashShift starts a till session.
func (h *Handler) AdminOpenCashShift(w http.ResponseWriter, r *http.Request) {
	branchScope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		OpeningFloat int    `json:"openingFloat"`
		Note         string `json:"note"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.OpeningFloat < 0 {
		httpx.Error(w, http.StatusBadRequest, "boshlang'ich qoldiq manfiy bo'la olmaydi")
		return
	}

	// ⚠️ One open shift per branch, refused rather than merged.
	//
	// Two open shifts make "what should be in the drawer" unanswerable: the
	// same counter sale belongs to both, and closing either one produces a
	// variance that is arithmetic rather than a fact about money. The same
	// rule the staff clock-in follows, for the same reason.
	if existing, err := h.openCashShift(r, branchScope); err == nil && existing != nil {
		httpx.Error(w, http.StatusConflict, "smena allaqachon ochiq")
		return
	}

	name := h.adminName(r)
	now := time.Now()
	shift := models.CashShift{
		BranchID:     branchOf(branchScope),
		OpenedAt:     now,
		OpenedBy:     name,
		OpeningFloat: req.OpeningFloat,
		Note:         strings.TrimSpace(req.Note),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	res, err := h.Store.CashShifts.InsertOne(r.Context(), shift)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	shift.ID = oidOf(res.InsertedID)
	h.logAction(r, ActCashShiftOpen, "cash", shift.ID.Hex(), "Kassa smenasi ochildi", "")
	httpx.JSON(w, http.StatusOK, shift)
}

// AdminCloseCashShift counts the till and records the difference.
func (h *Handler) AdminCloseCashShift(w http.ResponseWriter, r *http.Request) {
	branchScope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	shift, err := h.openCashShift(r, branchScope)
	if err != nil || shift == nil {
		httpx.Error(w, http.StatusBadRequest, "ochiq smena yo'q")
		return
	}
	var req struct {
		Counted      int    `json:"counted"`
		VarianceNote string `json:"varianceNote"`
		Note         string `json:"note"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Counted < 0 {
		httpx.Error(w, http.StatusBadRequest, "sanalgan summa manfiy bo'la olmaydi")
		return
	}

	figures, _, err := h.shiftFigures(r, shift)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	variance := req.Counted - figures.Expected

	// ⚠️ A difference cannot be saved without a sentence.
	//
	// Refused rather than defaulted, the same rule as cancelling an order and
	// voiding an invoice: the entry that removes money has to explain itself.
	// A drawer 40 000 short with an empty note is a record nobody can act on a
	// week later, and by then the person who could explain it has gone home.
	if variance != 0 && strings.TrimSpace(req.VarianceNote) == "" {
		httpx.Error(w, http.StatusBadRequest, "farq bor — sababini yozing")
		return
	}

	name := h.adminName(r)
	now := time.Now()
	update := bson.M{"$set": bson.M{
		"closedAt": now, "closedBy": name,
		// Frozen, not derived on read: changing how expected cash is computed
		// must never silently rewrite last month's shortfalls.
		"expected":     figures.Expected,
		"counted":      req.Counted,
		"variance":     variance,
		"varianceNote": strings.TrimSpace(req.VarianceNote),
		"note":         strings.TrimSpace(req.Note),
		"updatedAt":    now,
	}}
	// Guarded by "still open", so two people closing at once cannot both write
	// a count — the second would overwrite the first with a different drawer.
	res, err := h.Store.CashShifts.UpdateOne(r.Context(),
		bson.M{"_id": shift.ID, "closedAt": bson.M{"$exists": false}}, update)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.MatchedCount == 0 {
		httpx.Error(w, http.StatusConflict, "smena allaqachon yopilgan")
		return
	}
	h.logAction(r, ActCashShiftClose, "cash", shift.ID.Hex(),
		"Kassa smenasi yopildi", varianceLabel(variance))

	var saved models.CashShift
	_ = h.Store.CashShifts.FindOne(r.Context(), bson.M{"_id": shift.ID}).Decode(&saved)
	httpx.JSON(w, http.StatusOK, map[string]any{"shift": saved, "figures": figures})
}

// AdminAddCashEntry records money put in or taken out by hand.
func (h *Handler) AdminAddCashEntry(w http.ResponseWriter, r *http.Request) {
	branchScope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	shift, err := h.openCashShift(r, branchScope)
	if err != nil || shift == nil {
		httpx.Error(w, http.StatusBadRequest, "ochiq smena yo'q")
		return
	}
	var req struct {
		Kind     string `json:"kind"`
		Category string `json:"category"`
		Amount   int    `json:"amount"`
		Note     string `json:"note"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Kind != models.CashIn && req.Kind != models.CashOut {
		httpx.Error(w, http.StatusBadRequest, "turi: in yoki out")
		return
	}
	if req.Amount <= 0 {
		httpx.Error(w, http.StatusBadRequest, "summa noldan katta bo'lishi kerak")
		return
	}
	// The reason is required for the same reason the variance note is: cash
	// that moved without one is the entry that becomes an argument later.
	if strings.TrimSpace(req.Category) == "" {
		httpx.Error(w, http.StatusBadRequest, "sababini tanlang yoki yozing")
		return
	}

	name := h.adminName(r)
	entry := models.CashEntry{
		BranchID: shift.BranchID,
		ShiftID:  shift.ID,
		Kind:     req.Kind,
		Category: strings.TrimSpace(req.Category),
		Amount:   req.Amount,
		Note:     strings.TrimSpace(req.Note),
		By:       name,
		At:       time.Now(),
	}
	res, err := h.Store.CashEntries.InsertOne(r.Context(), entry)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	entry.ID = oidOf(res.InsertedID)
	h.logAction(r, ActCashEntry, "cash", entry.ID.Hex(), entry.Category, req.Kind)
	httpx.JSON(w, http.StatusOK, entry)
}

// AdminCashReport lists closed shifts for a period — the till's history, and
// the report an owner actually reads: every difference, with its explanation.
func (h *Handler) AdminCashReport(w http.ResponseWriter, r *http.Request) {
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
	filter := scopeFilter(branchScope)
	filter["closedAt"] = bson.M{"$exists": true}
	rng := bson.M{}
	if from != nil {
		rng["$gte"] = *from
	}
	if to != nil {
		rng["$lt"] = *to
	}
	if len(rng) > 0 {
		filter["openedAt"] = rng
	}

	cur, err := h.Store.CashShifts.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "openedAt", Value: -1}}).SetLimit(500))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	shifts := []models.CashShift{}
	if err := cur.All(r.Context(), &shifts); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	rows := make([]map[string]any, 0, len(shifts))
	var expected, counted, variance int
	for _, s := range shifts {
		expected += s.Expected
		counted += s.Counted
		variance += s.Variance
		closed := ""
		if s.ClosedAt != nil {
			closed = s.ClosedAt.In(time.Local).Format("2006-01-02 15:04")
		}
		rows = append(rows, map[string]any{
			"opened": s.OpenedAt.In(time.Local).Format("2006-01-02 15:04"),
			"closed": closed, "openedBy": s.OpenedBy, "closedBy": s.ClosedBy,
			"float": s.OpeningFloat, "expected": s.Expected,
			"counted": s.Counted, "variance": s.Variance,
			"varianceNote": s.VarianceNote,
		})
	}

	lang := reportLang(r)
	rep := &Report{
		Title: tr{"Kassa hisoboti", "Отчёт по кассе", "Till report"}.in(lang),
		Slug:  "kassa",
		From:  dayOrAll(from, lang), To: dayOrAll(to, lang),
		Note: tr{
			"Farq = sanalgan − kutilgan. Manfiy son — kamomad. " +
				"Yetkazib berishdagi naqd kuryer topshirgandan keyin kassaga kiradi.",
			"Разница = посчитано − ожидалось. Отрицательное число — недостача. " +
				"Наличные с доставки попадают в кассу после того, как курьер их сдал.",
			"Variance = counted − expected. A negative figure is a shortfall. " +
				"Cash from deliveries reaches the till only once the courier hands it in.",
		}.in(lang),
		Columns: cashColumns(lang),
		Rows:    rows,
		Totals: map[string]any{
			"opened": trTotal.in(lang), "expected": expected,
			"counted": counted, "variance": variance,
		},
	}
	if wantsExcel(r) {
		h.respondReport(w, r, rep)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"from": rep.From, "to": rep.To, "note": rep.Note,
		"shifts": shifts,
		"totals": map[string]int{"expected": expected, "counted": counted, "variance": variance},
	})
}

func cashColumns(lang string) []Column {
	return []Column{
		{Key: "opened", Title: tr{"Ochilgan", "Открыта", "Opened"}.in(lang), Kind: ColText},
		{Key: "closed", Title: tr{"Yopilgan", "Закрыта", "Closed"}.in(lang), Kind: ColText},
		{Key: "openedBy", Title: tr{"Ochdi", "Кто открыл", "Opened by"}.in(lang), Kind: ColText},
		{Key: "closedBy", Title: tr{"Yopdi", "Кто закрыл", "Closed by"}.in(lang), Kind: ColText},
		{Key: "float", Title: tr{"Boshlang'ich", "Начальный остаток", "Opening float"}.in(lang), Kind: ColMoney},
		{Key: "expected", Title: tr{"Kutilgan", "Ожидалось", "Expected"}.in(lang), Kind: ColMoney},
		{Key: "counted", Title: tr{"Sanalgan", "Посчитано", "Counted"}.in(lang), Kind: ColMoney},
		{Key: "variance", Title: tr{"Farq", "Разница", "Variance"}.in(lang), Kind: ColMoney},
		{Key: "varianceNote", Title: tr{"Farq sababi", "Причина разницы", "Reason for variance"}.in(lang), Kind: ColText},
	}
}

// ---- helpers ----

func (h *Handler) openCashShift(r *http.Request, branchScope bson.M) (*models.CashShift, error) {
	filter := scopeFilter(branchScope)
	filter["closedAt"] = bson.M{"$exists": false}
	var s models.CashShift
	err := h.Store.CashShifts.FindOne(r.Context(), filter,
		options.FindOne().SetSort(bson.D{{Key: "openedAt", Value: -1}})).Decode(&s)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func scopeFilter(branchScope bson.M) bson.M {
	f := bson.M{}
	for k, v := range branchScope {
		f[k] = v
	}
	return f
}

func branchOf(branchScope bson.M) primitive.ObjectID {
	if v, ok := branchScope["branchId"]; ok {
		if id, ok := v.(primitive.ObjectID); ok {
			return id
		}
	}
	return primitive.NilObjectID
}

func nilIfZero(s *models.CashShift) any {
	if s == nil || s.ID.IsZero() {
		return nil
	}
	return s
}

func varianceLabel(v int) string {
	if v == 0 {
		return "farqsiz"
	}
	if v < 0 {
		return "kamomad"
	}
	return "ortiqcha"
}

// sumField aggregates one numeric field over a collection.
func (h *Handler) sumField(ctx context.Context, coll *mongo.Collection, filter bson.M, field string) (int, int, error) {
	cur, err := coll.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.M{
			"_id": nil, "total": bson.M{"$sum": field}, "n": bson.M{"$sum": 1},
		}}},
	})
	if err != nil {
		return 0, 0, err
	}
	var rows []struct {
		Total int `bson:"total"`
		N     int `bson:"n"`
	}
	if err := cur.All(ctx, &rows); err != nil || len(rows) == 0 {
		return 0, 0, err
	}
	return rows[0].Total, rows[0].N, nil
}

// cashWithCouriers is money collected on delivered cash orders that has not
// been handed over yet.
//
// ⚠️ Deliberately outside the till's expected balance. It is real cash and it
// is somebody's responsibility, but it is not in the drawer — and an owner
// counting a short till wants this number before they start asking anyone
// difficult questions.
func (h *Handler) cashWithCouriers(ctx context.Context, scope bson.M) int {
	collected := bson.M{
		"paymentMethod": models.ProviderCash,
		"status":        models.StatusDelivered,
		"courierId":     bson.M{"$exists": true, "$ne": primitive.NilObjectID},
	}
	for k, v := range scope {
		collected[k] = v
	}
	total, _, err := h.sumField(ctx, h.Store.Orders, collected, "$total")
	if err != nil {
		return 0
	}
	settled, _, err := h.sumField(ctx, h.Store.Settlements, bson.M{}, "$amount")
	if err != nil {
		return 0
	}
	// Handing over more than was collected is a bookkeeping mistake, not a
	// negative amount of cash — the same clamp the courier card applies.
	if settled > total {
		return 0
	}
	return total - settled
}
