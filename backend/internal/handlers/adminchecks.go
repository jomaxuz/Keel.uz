package handlers

import (
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/receipt"
)

// ---- Dining room and counter sales ----
//
// The orders board deliberately leaves till checks out (see AdminListOrders):
// a room full of open tables would bury the delivery orders somebody has to
// accept, and the till itself is the better screen for what is open right now.
//
// ⚠️ **But "not on that board" turned into "nowhere at all".** The money was
// never lost — the statistics, the sales report and the financial report have
// counted till sales from the first day, because they filter on the period and
// the scope and nothing else. What was missing was the list: an owner could
// see that Tuesday took 4.2M and could not see *which sales those were*, which
// is the question every argument about a shift starts with. This page is that
// list, and it is deliberately a sibling of the orders board rather than a tab
// on it — the two answer different questions and are read by different people.
//
// ⚠️ The period is cut on **`createdAt`**, the same field the reports use, and
// that matters more than picking the "better" timestamp. A list whose month is
// bounded differently from the report's month is two answers to one question,
// and the first person to add them up finds a difference nobody can explain.
// (`closedAt` is the tempting one — a check opened at 23:50 and paid at 00:20
// is Tuesday's sale by the drawer and Monday's by this list — but the till's
// own answer to that is the cash shift, which is a separate screen.)

// checkRow is one sale as this page shows it.
//
// A narrow struct of its own rather than the order: the order carries the
// customer record, the address, the courier and the status history, and none
// of that means anything for a table. A list that ships all of it invites the
// next field to be added by accident.
type checkRow struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	// Empty for a counter sale — that is the only thing distinguishing the two,
	// and it is what `place` filters on.
	Table  string `json:"table,omitempty"`
	Guests int    `json:"guests,omitempty"`
	// Who the check belongs to, and who took the money. Usually the same
	// person; on a busy night routinely not, and "who closed this" is the
	// question a till exists to be able to answer.
	Server   string     `json:"server,omitempty"`
	ClosedBy string     `json:"closedBy,omitempty"`
	OpenedAt time.Time  `json:"openedAt"`
	ClosedAt *time.Time `json:"closedAt,omitempty"`
	// Dishes, not lines: two portions of one dish is two.
	Items         int    `json:"items"`
	Subtotal      int    `json:"subtotal"`
	Discount      int    `json:"discount,omitempty"`
	Total         int    `json:"total"`
	PaymentMethod string `json:"paymentMethod,omitempty"`
	// "", "pending", "ok" or "error" — the panel draws the same badge the till
	// does, so an unfiled sale is visible to the owner as well as the cashier.
	Fiscal string `json:"fiscal,omitempty"`
	Open   bool   `json:"open"`
	// Part of a bill that was divided at the table. ⚠️ Its money is real and
	// counts; its **existence** does not — see checkTotals.Checks.
	Split bool `json:"split,omitempty"`
}

// checkTotals is the whole filtered set, never the page.
//
// ⚠️ Computed over every matching sale even when only a hundred rows are
// returned. A footer that adds up the page is a number that changes when you
// press "next", and it is the number that gets copied into a message.
type checkTotals struct {
	// ⚠️ **Tables, not pieces of paper.** A party that asked for four bills had
	// one dinner: counting four would show a busier night than the room had,
	// and would quietly drag the average check down towards a quarter of it.
	// The money from every half is in `Sales` — it was all taken.
	Checks int `json:"checks"`
	// How many of those bills were halves. Shown so the two numbers can be
	// reconciled by anybody who counts the rows on screen.
	Splits   int `json:"splits"`
	Open     int `json:"open"`
	Guests   int `json:"guests"`
	Sales    int `json:"sales"`
	Discount int `json:"discount"`
	Cash     int `json:"cash"`
	Card     int `json:"card"`
	Other    int `json:"other"`
	// Average over **closed** checks only: an open table has taken no money
	// yet, and dividing by it makes every busy evening look cheap.
	AvgCheck int `json:"avgCheck"`
	// Per guest, over the closed checks that said how many people were sitting
	// there. Zero when nobody filled it in, rather than a figure computed from
	// the few that did — a dining room's most-quoted number must not quietly
	// mean "the tables where somebody remembered".
	AvgGuest int `json:"avgGuest"`
	Hall     int `json:"hall"`
	Counter  int `json:"counter"`
}

// AdminListChecks lists till sales — the dining room and the counter.
func (h *Handler) AdminListChecks(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	q := r.URL.Query()
	from, to, err := parseRange(q.Get("from"), q.Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := bson.M{}
	for k, v := range scope {
		filter[k] = v
	}
	// The one thing that makes a sale a till sale. Same field the orders board
	// excludes on, from the opposite side — so a check is on exactly one of the
	// two screens and never on neither.
	filter["check"] = bson.M{"$exists": true}
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
	switch q.Get("state") {
	case "open":
		filter["check.closedAt"] = bson.M{"$exists": false}
	case "closed":
		filter["check.closedAt"] = bson.M{"$exists": true}
	}
	if m := q.Get("method"); m != "" {
		filter["paymentMethod"] = m
	}
	if id := q.Get("serverId"); id != "" {
		oid, err := objectID(id)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid serverId")
			return
		}
		filter["check.serverId"] = oid
	}
	switch q.Get("place") {
	case "hall":
		filter["tableId"] = bson.M{"$nin": bson.A{nil, ""}}
	case "counter":
		filter["tableId"] = bson.M{"$in": bson.A{nil, ""}}
	}
	if s := strings.TrimSpace(q.Get("q")); s != "" {
		s = strings.TrimPrefix(s, "#")
		rx := bson.M{"$regex": regexp.QuoteMeta(s), "$options": "i"}
		filter["$or"] = []bson.M{
			{"number": rx}, {"tableNumber": rx},
			{"check.serverName": rx}, {"check.closedBy": rx},
		}
	}

	orders, err := h.checksMatching(r, filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	rows := make([]checkRow, 0, len(orders))
	for i := range orders {
		rows = append(rows, checkRowOf(&orders[i]))
	}
	// Newest first, and by when the check was opened: the id would sort almost
	// the same way and would be wrong for a sale the till took offline and
	// handed over hours later.
	sort.SliceStable(rows, func(a, b int) bool {
		return rows[a].OpenedAt.After(rows[b].OpenedAt)
	})
	totals := totalsOf(rows)

	limit := 100
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 && v <= 500 {
		limit = v
	}
	skip, _ := strconv.Atoi(q.Get("skip"))
	if skip < 0 || skip > len(rows) {
		skip = 0
	}
	page := rows[skip:min(skip+limit, len(rows))]
	if page == nil {
		page = []checkRow{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"rows":   page,
		"total":  len(rows),
		"totals": totals,
	})
}

// checksMatching reads the period. Bounded by the filter, like the reports.
func (h *Handler) checksMatching(r *http.Request, filter bson.M) ([]models.Order, error) {
	cur, err := h.Store.Orders.Find(r.Context(), filter)
	if err != nil {
		return nil, err
	}
	var orders []models.Order
	if err := cur.All(r.Context(), &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

func checkRowOf(o *models.Order) checkRow {
	c := o.Check
	if c == nil {
		c = &models.OrderCheck{}
	}
	items := 0
	for _, it := range o.Items {
		// Voided lines are not sold. They are worth keeping on the check —
		// that is the whole point of a void — but counting them here would
		// make a cancelled starter look like food that went out.
		if it.Live() {
			items += it.Qty
		}
	}
	row := checkRow{
		ID:            o.ID.Hex(),
		Number:        o.Number,
		Table:         o.TableNumber,
		Guests:        c.Guests,
		Server:        c.ServerName,
		ClosedBy:      c.ClosedBy,
		OpenedAt:      c.OpenedAt.In(time.Local),
		Items:         items,
		Subtotal:      o.Subtotal,
		Discount:      o.DiscountTotal,
		Total:         o.Total,
		PaymentMethod: o.PaymentMethod,
		Open:          c.IsOpen(),
		Split:         !c.SplitFromID.IsZero(),
	}
	if row.OpenedAt.IsZero() {
		row.OpenedAt = o.CreatedAt.In(time.Local)
	}
	if c.ClosedAt != nil {
		// ⚠️ The driver hands every time back in UTC, so a sale closed at
		// half past midnight reads as the previous evening unless it is moved
		// back into the restaurant's own zone first.
		at := c.ClosedAt.In(time.Local)
		row.ClosedAt = &at
	}
	if o.Fiscal != nil {
		row.Fiscal = string(o.Fiscal.Status)
	}
	return row
}

func totalsOf(rows []checkRow) checkTotals {
	var t checkTotals
	closed, guests := 0, 0
	for _, row := range rows {
		if row.Split {
			t.Splits++
		} else {
			t.Checks++
		}
		if row.Table != "" {
			t.Hall++
		} else {
			t.Counter++
		}
		t.Guests += row.Guests
		if row.Open {
			t.Open++
			continue
		}
		if !row.Split {
			closed++
		}
		t.Sales += row.Total
		t.Discount += row.Discount
		switch row.PaymentMethod {
		case "cash":
			t.Cash += row.Total
		case "card":
			t.Card += row.Total
		default:
			t.Other += row.Total
		}
		if row.Guests > 0 {
			guests += row.Guests
		}
	}
	if closed > 0 {
		t.AvgCheck = t.Sales / closed
	}
	if guests > 0 {
		t.AvgGuest = t.Sales / guests
	}
	return t
}

// ---- One sale, opened ----
//
// ⚠️ **A narrow shape again, and for a sharper reason than the list's.** The
// order document carries the customer record, the address, the courier and the
// delivery fee; for a table those fields are either empty or meaningless, and a
// screen that renders whatever arrives eventually shows one of them filled in
// by something unrelated. What a dining-room sale has to explain is different
// from what a delivery has to explain: who was sitting there, what went to the
// kitchen and when, what was taken off the bill and by whom, and how the money
// was settled.
//
// ⚠️ **Voided lines are in the response.** They are the single most important
// thing on this screen — a void that leaves no trace is the oldest way to take
// money out of a restaurant, which is exactly why the line stays on the
// document. Sending only the live lines would make the detail view agree with a
// dishonest check and disagree with the kitchen.

type checkLineView struct {
	Name  string `json:"name"`
	Qty   int    `json:"qty"`
	Price int    `json:"price"`
	// Zero for a voided line: it is on the bill's face and not in its total.
	Sum     int                      `json:"sum"`
	Options []models.OrderItemOption `json:"options,omitempty"`
	Comment string                   `json:"comment,omitempty"`
	// Zero means the table — one bill for the party, which is how most meals
	// end and how every check written before splitting existed reads.
	Guest int `json:"guest,omitempty"`
	// Zero means "with everything else".
	Course  int        `json:"course,omitempty"`
	FiredAt *time.Time `json:"firedAt,omitempty"`
	// Who took it off and why. The reason is the point of the record.
	VoidedBy   string     `json:"voidedBy,omitempty"`
	VoidReason string     `json:"voidReason,omitempty"`
	VoidedAt   *time.Time `json:"voidedAt,omitempty"`
	// Whether the food had actually been made. A kitchen that caught it in
	// time and a plate that went in the bin are different losses.
	Wasted bool `json:"wasted,omitempty"`
}

type checkDetail struct {
	checkRow
	Lines      []checkLineView        `json:"lines"`
	Discounts  []models.OrderDiscount `json:"discounts,omitempty"`
	OpenedBy   string                 `json:"openedBy,omitempty"`
	PrecheckAt *time.Time             `json:"precheckAt,omitempty"`
	// The register's own words when a filing failed. They usually name
	// something fixable in seconds, and a summary would turn an instruction
	// into a category.
	FiscalError string `json:"fiscalError,omitempty"`
	// The tax authority's own sign, not a number of ours: it is what a guest
	// or an inspector checks the sale against.
	FiscalSign string `json:"fiscalSign,omitempty"`
}

// AdminGetCheck opens one dining-room or counter sale.
func (h *Handler) AdminGetCheck(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	// The branch lives inside the filter, not beside it: an id alone must
	// never select a document, or a manager pinned to one kitchen reads
	// another one's takings by pasting an id from the list they can see.
	filter, err := h.scopedOrderFilter(r, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// Out of scope and "not a till sale" are the same 404 — a manager should
	// not learn that either kind of document exists.
	filter["check"] = bson.M{"$exists": true}
	var o models.Order
	if err := h.Store.Orders.FindOne(r.Context(), filter).Decode(&o); err != nil {
		httpx.Error(w, http.StatusNotFound, "check not found")
		return
	}

	d := checkDetail{checkRow: checkRowOf(&o), Lines: []checkLineView{}}
	if o.Check != nil {
		d.OpenedBy = o.Check.OpenedBy
		if at := o.Check.PrecheckAt; at != nil {
			t := at.In(time.Local)
			d.PrecheckAt = &t
		}
	}
	if o.Fiscal != nil {
		d.FiscalError = o.Fiscal.Error
		d.FiscalSign = o.Fiscal.FiscalSign
	}
	if len(o.Discounts) > 0 {
		d.Discounts = o.Discounts
	}
	for _, it := range o.Items {
		line := checkLineView{
			Name:    it.Name,
			Qty:     it.Qty,
			Price:   it.Price,
			Options: it.Options,
			Comment: it.Comment,
			Guest:   it.Guest,
			Course:  it.Course,
		}
		if it.FiredAt != nil {
			at := it.FiredAt.In(time.Local)
			line.FiredAt = &at
		}
		if it.Void != nil {
			line.VoidedBy = it.Void.By
			line.VoidReason = it.Void.Reason
			line.Wasted = it.Void.Wasted
			at := it.Void.At.In(time.Local)
			line.VoidedAt = &at
		} else {
			line.Sum = it.Price * it.Qty
		}
		d.Lines = append(d.Lines, line)
	}
	httpx.JSON(w, http.StatusOK, d)
}

// ---- Printing a sale from the panel ----
//
// ⚠️ **The same renderer as the till, never a second layout.** A guest ringing
// about a bill is read to from this screen while they hold the paper; a copy
// that lays the lines out differently makes that conversation about the two
// documents instead of about the meal. The reports follow the same rule for the
// same reason: one calculation, two outputs.
//
// ⚠️ **The browser prints by default, and the restaurant's printer only when
// asked.** Whoever opens the panel is usually not in the building — paper
// appearing at a counter nobody is standing at is confusing at best, and on a
// busy evening it is a slip somebody has to work out the meaning of. The
// browser's own dialog is also where "save as PDF" lives, which is what this is
// wanted for most of the time.

type adminPrintRequest struct {
	// Send it to the branch's own printers as well.
	ToPrinter bool `json:"toPrinter"`
}

// AdminPrintCheck lays out one sale's guest receipt.
func (h *Handler) AdminPrintCheck(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	filter, err := h.scopedOrderFilter(r, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter["check"] = bson.M{"$exists": true}
	var o models.Order
	if err := h.Store.Orders.FindOne(r.Context(), filter).Decode(&o); err != nil {
		httpx.Error(w, http.StatusNotFound, "check not found")
		return
	}
	var req adminPrintRequest
	if r.ContentLength > 0 {
		if err := httpx.Decode(r, &req); err != nil {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	// ⚠️ The **guest's** copy, and the branch's own template for it: the paper
	// this is compared against was printed with the header, the width and the
	// footer that branch set. The kitchen and till copies are working
	// documents and mean nothing to somebody holding a bill.
	settings := h.receiptSettingsOf(r.Context(), o.BranchID)
	tpl := settings.Customer
	data := h.checkReceiptOf(r.Context(), &o)

	// How many of the branch's printers took it. Zero is a real answer — a
	// branch with none is the normal case on the first evening — and the
	// screen says "sent" only when something was.
	queued := 0
	if req.ToPrinter {
		queued = h.queueReceipt(r.Context(), o.BranchID, receipt.Customer, tpl, data, &o)
	}
	logo := ""
	if tpl.Logo {
		logo = h.logoURL(r.Context())
	}
	h.logAction(r, "check.print", "order", o.ID.Hex(), o.Number, "")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"lines":   receipt.Render(receipt.Customer, tpl, data),
		"widthMM": tpl.WidthMM,
		"logoUrl": logo,
		"queued":  queued,
	})
}
