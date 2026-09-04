package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
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
		// ⚠️ How long it has been open, on every read. A till with an open
		// shift looks exactly like a till working normally, which is why this
		// has to be said rather than inferred — see handlers/shiftwatch.go.
		"age": h.shiftAge(r.Context(), shift),
	})
}

// cashFigures is the arithmetic behind "what should be in the drawer".
type cashFigures struct {
	OpeningFloat int `json:"openingFloat"`
	// Cash taken at the counter: dine-in and pickup orders settled in cash.
	CounterCash int `json:"counterCash"`
	CounterN    int `json:"counterOrders"`
	// Cash that arrived this shift for a sale closed in an earlier one — a
	// debt somebody came back and paid.
	//
	// ⚠️ **Part of CounterCash, not an addition to it**: the money is in the
	// drawer and already counted. It is broken out because otherwise the paper
	// says the drawer holds more cash than the shift sold, with nothing on it
	// to explain the difference — and an unexplained difference on a Z report
	// is an accusation aimed at whoever counted the drawer.
	DebtPaid  int `json:"debtPaid"`
	DebtPaidN int `json:"debtPaidCount"`
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

	// ⚠️ Identified by **when the sale closed**, not by a flag: a payment that
	// arrived after its own shift is the same event whether it was written on
	// the slate or confirmed late by a provider, and the drawer cannot tell
	// them apart either.
	repaid := bson.M{"paidAt": since, "check.closedAt": bson.M{"$lt": shift.OpenedAt}}
	for k, v := range counter {
		if k == "paidAt" {
			continue
		}
		repaid[k] = v
	}
	if total, n, err := h.sumField(ctx, h.Store.Orders, repaid, "$total"); err == nil {
		f.DebtPaid, f.DebtPaidN = total, n
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

	shiftPtr, status, err := h.openShiftFor(r, branchOf(branchScope),
		req.OpeningFloat, req.Note, h.adminName(r))
	if err != nil {
		httpx.Error(w, status, err.Error())
		return
	}
	shift := *shiftPtr
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

	name := h.adminName(r)
	figures, status, err := h.closeShiftFor(r, shift, req.Counted,
		req.VarianceNote, req.Note, name)
	if err != nil {
		httpx.Error(w, status, err.Error())
		return
	}
	h.logAction(r, ActCashShiftClose, "cash", shift.ID.Hex(),
		"Kassa smenasi yopildi", varianceLabel(req.Counted-figures.Expected))

	// ⚠️ **Only a shortfall, never a surplus.** A drawer with more in it than
	// expected is usually a sale rung on the wrong tender or change that was
	// not given, and it is worth looking at — but it is not worth a phone
	// buzzing in the evening, and mixing the two is how the useful message
	// becomes one of the two the owner scrolls past.
	if short := figures.Expected - req.Counted; short > 0 {
		if aset := h.alertSettingsOf(r.Context(), shift.BranchID); aset.Enabled &&
			short >= aset.CashShortFrom {
			h.raiseAlert(models.LossAlert{
				BranchID: shift.BranchID,
				Kind:     models.AlertCashShort,
				By:       name,
				Amount:   short,
				Reason:   clampText(req.VarianceNote, 200),
				RefID:    shift.ID,
			})
		}
	}

	// ⚠️ **The register's day is asked to end, not ended here.** The two shifts
	// are not the same shift — ours can turn over twice a day when staff change,
	// the register's is a tax day — and this panel is frequently a laptop that
	// cannot reach the register at all. So the request is recorded and whoever
	// is standing on the restaurant's network carries it out. See
	// handlers/fiscalday.go.
	//
	// ⚠️ **Never blocks the drawer count.** Counting cash is this endpoint's
	// job and it has already succeeded; refusing to record it because a PC in
	// the corner has unfiled receipts would lose the count and leave the money
	// unexplained. The reason comes back beside the shift instead, where the
	// manager can act on it.
	fiscalNote := ""
	if _, ferr := h.requestCloseDay(r.Context(), branchOf(branchScope), name, shift.ID); ferr != nil {
		fiscalNote = ferr.Error()
	}

	var saved models.CashShift
	_ = h.Store.CashShifts.FindOne(r.Context(), bson.M{"_id": shift.ID}).Decode(&saved)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"shift": saved, "figures": figures, "fiscalNote": fiscalNote,
	})
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
	var req cashEntryInput
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	req.By = h.adminName(r)
	entry, code, err := h.addCashEntry(r, shift, req)
	if err != nil {
		httpx.Error(w, code, err.Error())
		return
	}
	h.logAction(r, ActCashEntry, "cash", entry.ID.Hex(), entry.Category, entry.Kind)
	httpx.JSON(w, http.StatusOK, entry)
}

// cashEntryInput is one movement of cash by hand, from either screen.
type cashEntryInput struct {
	Kind     string `json:"kind"`
	Category string `json:"category"`
	Amount   int    `json:"amount"`
	Note     string `json:"note"`
	// Whether this movement is the drawer being emptied into the safe, or the
	// safe topping the drawer up.
	//
	// ⚠️ **Asked rather than inferred from the category.** "Inkassatsiya" is
	// free text — one restaurant writes it, the next writes "bankka", a third
	// writes nothing — and a safe balance built on guessing at words would be a
	// confident figure about a box nobody opened. Ticked, it writes one linked
	// row; the link makes writing it twice impossible.
	ToSafe bool `json:"toSafe"`
	// Who this money was paid to, when it was a wage.
	//
	// ⚠️ **The hole this closes was silent and expensive.** A cashier handing a
	// courier his month out of the drawer produced one document: a cash entry
	// reading "maosh". The drawer was right, and payroll went on saying the
	// courier was owed the whole amount — so at the end of the month the
	// restaurant either paid twice or spent an evening arguing about a Tuesday
	// nobody could remember. The wage is now a wage document as well, written
	// in the same call.
	//
	// ⚠️ **Picked from a list, never typed.** A name typed at a counter is
	// "Aziz", "aziz", "Азиз" and "Aziz kuryer" within a week, and none of them
	// can be matched to the person the payroll owes.
	PersonKind string             `json:"personKind"`
	PersonID   primitive.ObjectID `json:"-"`
	// Who is answering for it. Not decoded from the request — the panel takes
	// it from the session and the till from whoever's PIN was accepted, and a
	// name a client could choose is a name that means nothing on an audit line.
	By string `json:"-"`
}

// addCashEntry writes one movement, or says why it cannot.
//
// ⚠️ **Shared by the panel and the till on purpose.** Both screens record the
// same thing against the same drawer, and the rule that a payout cannot exceed
// what is in it has to be one rule: two copies would drift, and the one that
// had drifted would be the one on the counter taking real money out.
func (h *Handler) addCashEntry(
	r *http.Request, shift *models.CashShift, req cashEntryInput,
) (models.CashEntry, int, error) {
	var entry models.CashEntry
	if req.Kind != models.CashIn && req.Kind != models.CashOut {
		return entry, http.StatusBadRequest, errors.New("turi: in yoki out")
	}
	if req.Amount <= 0 {
		return entry, http.StatusBadRequest,
			errors.New("summa noldan katta bo'lishi kerak")
	}
	// The reason is required for the same reason the variance note is: cash
	// that moved without one is the entry that becomes an argument later.
	if strings.TrimSpace(req.Category) == "" {
		return entry, http.StatusBadRequest,
			errors.New("sababini tanlang yoki yozing")
	}

	// ⚠️ **You cannot take out money that is not in the drawer.**
	//
	// Nothing checked this, so a cashier could record a 500 000 payout against
	// an empty till and every screen accepted it: the entry saved, the expected
	// balance went negative, and the only place it surfaced was the count at
	// the end of the evening — as a difference nobody could explain, because
	// the entry itself looked perfectly ordinary. A shortfall that is discovered
	// six hours later is a shortfall that gets blamed on whoever counted.
	//
	// ⚠️ Checked against `Expected`, which is the drawer's own arithmetic
	// (float + counter cash + settlements + manual in − manual out) — the same
	// number the screen shows and the same one the close reconciles against.
	// Anything else here would be a second opinion about how much money is in
	// one box.
	//
	// ⚠️ **Only `out` is checked.** Money going in cannot overdraw anything, and
	// refusing an unexpected deposit would be refusing the one entry that fixes
	// a shortfall.
	if req.Kind == models.CashOut {
		figures, _, err := h.shiftFigures(r, shift)
		if err != nil {
			return entry, http.StatusInternalServerError, err
		}
		if msg := cashOutRefusal(req.Amount, figures.Expected); msg != "" {
			return entry, http.StatusBadRequest, errors.New(msg)
		}
	}

	entry = models.CashEntry{
		BranchID: shift.BranchID,
		ShiftID:  shift.ID,
		Kind:     req.Kind,
		Category: strings.TrimSpace(req.Category),
		Amount:   req.Amount,
		Note:     strings.TrimSpace(req.Note),
		By:       req.By,
		At:       time.Now(),
	}
	res, err := h.Store.CashEntries.InsertOne(r.Context(), entry)
	if err != nil {
		return entry, http.StatusInternalServerError, err
	}
	entry.ID = oidOf(res.InsertedID)

	// ⚠️ **The safe's side is the mirror, not a copy.** Money taken *out* of the
	// drawer and carried to the office goes *into* the safe; a float brought
	// back the other way comes out of it. Getting the direction wrong here would
	// double a balance instead of moving it, and both numbers would still look
	// like money.
	if req.ToSafe {
		kind := models.SafeIn
		if entry.Kind == models.CashIn {
			kind = models.SafeOut
		}
		h.recordSafeMovement(r.Context(), models.SafeEntry{
			BranchID: entry.BranchID, Kind: kind, Amount: entry.Amount,
			At: entry.At, Category: entry.Category, Note: entry.Note,
			By:      entry.By,
			RefKind: models.SafeRefCash, RefID: entry.ID,
		})
	}
	// ⚠️ **The wage document, written from the same call.** Two facts, not two
	// copies: the drawer got lighter *and* a person was paid. Either one alone
	// is a real record that leaves the other question unanswered — which is
	// exactly the state this used to be in.
	//
	// ⚠️ Failure here does not fail the entry. The money has already left the
	// drawer; refusing to record that because the payroll write failed would
	// lose the one fact we are certain of. It is logged, and the payroll row
	// can be corrected by hand.
	if req.Kind == models.CashOut && !req.PersonID.IsZero() {
		h.recordWagePayment(r, req, entry)
	}
	return entry, http.StatusOK, nil
}

// recordWagePayment writes the payroll side of a wage paid out of the drawer.
func (h *Handler) recordWagePayment(
	r *http.Request, req cashEntryInput, entry models.CashEntry,
) {
	now := entry.At
	// The window it settles: the person's own current pay period, which is the
	// one the payroll screen is showing when somebody decides to pay them.
	from, to := payPeriodBounds(models.PeriodMonthly, now)
	switch req.PersonKind {
	case "courier":
		var c models.Courier
		if err := h.Store.Couriers.FindOne(r.Context(),
			bson.M{"_id": req.PersonID}).Decode(&c); err == nil {
			from, to = payPeriodBounds(c.PayPeriod, now)
		}
		_, err := h.Store.CourierPayments.InsertOne(r.Context(), models.CourierPayment{
			CourierID: req.PersonID, BranchID: entry.BranchID, Amount: entry.Amount,
			From: from.Format(dayLayout), To: to.Format(dayLayout),
			PaidBy: entry.By, Note: entry.Note, At: now,
		})
		if err != nil {
			log.Printf("wage payment (courier %s): %v", req.PersonID.Hex(), err)
		}
	case "staff":
		var st models.Staff
		if err := h.Store.Staff.FindOne(r.Context(),
			bson.M{"_id": req.PersonID}).Decode(&st); err == nil {
			from, to = payPeriodBounds(st.PayPeriod, now)
		}
		_, err := h.Store.StaffPayments.InsertOne(r.Context(), models.StaffPayment{
			StaffID: req.PersonID, BranchID: entry.BranchID, Amount: entry.Amount,
			From: from.Format(dayLayout), To: to.Format(dayLayout),
			PaidBy: entry.By, Note: entry.Note, At: now,
		})
		if err != nil {
			log.Printf("wage payment (staff %s): %v", req.PersonID.Hex(), err)
		}
	}
}

// recentClosedShifts is the last few evenings, newest first.
//
// ⚠️ Deliberately short and branch-scoped: this answers "print yesterday's
// again" at a counter, not "how did March go" — that question belongs to the
// panel, which has the period picker and the export.
func (h *Handler) recentClosedShifts(
	ctx context.Context, branchID primitive.ObjectID, limit int64,
) ([]models.CashShift, error) {
	cur, err := h.Store.CashShifts.Find(ctx, bson.M{
		"branchId": branchID,
		"closedAt": bson.M{"$exists": true},
	}, options.Find().SetSort(bson.D{{Key: "closedAt", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	rows := []models.CashShift{}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
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

// ---- Shared between the panel and the till ----
//
// ⚠️ **One implementation, deliberately.** The panel and the till both count the
// same drawer, and two copies of "what should be in it" would eventually
// disagree — at which point the restaurant has two answers about missing money
// and no way to tell which is right. Same rule as the pricing pipeline and
// recordFiling.

// openShiftFor starts a till session for a branch.
//
// Returns the HTTP status to answer with alongside the error, because the two
// refusals here mean different things: a negative float is a typo (400), an
// already-open shift is a race with somebody else (409).
func (h *Handler) openShiftFor(
	r *http.Request, branchID primitive.ObjectID,
	openingFloat int, note, byName string,
) (*models.CashShift, int, error) {
	if openingFloat < 0 {
		return nil, http.StatusBadRequest,
			errors.New("boshlang'ich qoldiq manfiy bo'la olmaydi")
	}
	scope := bson.M{"branchId": branchID}
	// ⚠️ One open shift per branch, refused rather than merged. Two open shifts
	// make "what should be in the drawer" unanswerable: the same counter sale
	// belongs to both, and closing either produces a variance that is
	// arithmetic rather than a fact about money.
	if existing, err := h.openCashShift(r, scope); err == nil && existing != nil {
		return nil, http.StatusConflict, errors.New("smena allaqachon ochiq")
	}
	now := time.Now()
	shift := models.CashShift{
		BranchID:     branchID,
		OpenedAt:     now,
		OpenedBy:     byName,
		OpeningFloat: openingFloat,
		Note:         strings.TrimSpace(note),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	res, err := h.Store.CashShifts.InsertOne(r.Context(), shift)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	shift.ID = oidOf(res.InsertedID)
	return &shift, http.StatusOK, nil
}

// closeShiftFor counts the drawer and records the difference.
//
// ⚠️ **The difference is the product**, which is why `expected` is frozen here
// rather than derived on read: changing how expected cash is computed must
// never silently rewrite last month's shortfalls.
func (h *Handler) closeShiftFor(
	r *http.Request, shift *models.CashShift,
	counted int, varianceNote, note, byName string,
) (cashFigures, int, error) {
	if counted < 0 {
		return cashFigures{}, http.StatusBadRequest,
			errors.New("sanalgan summa manfiy bo'la olmaydi")
	}
	figures, _, err := h.shiftFigures(r, shift)
	if err != nil {
		return cashFigures{}, http.StatusInternalServerError, err
	}
	variance := counted - figures.Expected

	// ⚠️ A difference cannot be saved without a sentence. The same rule as
	// cancelling an order and voiding a dish: the entry that removes money has
	// to explain itself. A drawer 40 000 short with an empty note is a record
	// nobody can act on a week later, and by then the person who could explain
	// it has gone home.
	if variance != 0 && strings.TrimSpace(varianceNote) == "" {
		return cashFigures{}, http.StatusBadRequest, errors.New("farq bor — sababini yozing")
	}

	now := time.Now()
	update := bson.M{"$set": bson.M{
		"closedAt": now, "closedBy": byName,
		"expected": figures.Expected,
		// ⚠️ Frozen with it: the fiscal register reports cash *sales*, not the
		// drawer, so this is the only figure of ours it can honestly be set
		// beside. Without it the two-source comparison cannot be made.
		"counterCash":  figures.CounterCash,
		"counted":      counted,
		"variance":     variance,
		"varianceNote": strings.TrimSpace(varianceNote),
		"note":         strings.TrimSpace(note),
		"updatedAt":    now,
	}}
	// ⚠️ Guarded by "still open", so two people closing at once cannot both
	// write a count — the second would overwrite the first with a different
	// drawer, and the shortfall would belong to neither of them.
	res, err := h.Store.CashShifts.UpdateOne(r.Context(),
		bson.M{"_id": shift.ID, "closedAt": bson.M{"$exists": false}}, update)
	if err != nil {
		return cashFigures{}, http.StatusInternalServerError, err
	}
	if res.MatchedCount == 0 {
		return cashFigures{}, http.StatusConflict, errors.New("smena allaqachon yopilgan")
	}
	// The day is over as far as this branch is concerned. ⚠️ After the write
	// and never inside it: the close is the thing that must not fail.
	h.summariseDay(r.Context(), shift.BranchID)
	return figures, http.StatusOK, nil
}

// cashOutRefusal says why this payout cannot be recorded, or "" if it can.
//
// ⚠️ A pure function so the rule can be sealed by a test: it is one comparison,
// and one comparison written inline in a handler is the kind of thing a later
// edit reorders without noticing that the drawer stopped being checked.
//
// ⚠️ **The amount is said out loud.** "Not enough" on its own sends the cashier
// to guess, and the guess is a second rejected attempt with a guest waiting —
// the number is also the answer to the question they are about to take to a
// manager.
func cashOutRefusal(amount, inDrawer int) string {
	if amount <= inDrawer {
		return ""
	}
	return fmt.Sprintf("kassada buncha pul yo'q — hozir %d so'm bor", inDrawer)
}
