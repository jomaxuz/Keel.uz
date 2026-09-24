package handlers

// ---- The money ledger: every so'm that moved, for an accountant's program ----
//
// `GET /api/open/v1/money` (scope `finance:read`). Built for accounting
// services a restaurant contracts with itself (the first is Finze AI): they
// work with numbers, so they want every movement of money, and they have none
// of this system's hard-won knowledge of which movements are *not* profit or
// loss. See docs/DECISIONS.md → "Pul daftari (`/money`)" and docs/open-api.md.
//
// ⚠️ **The class is the product.** Handed the raw documents, a model books a
// collection to the bank as an expense, an advance to the market buyer as an
// expense, an aggregator's transfer as revenue a second time — each one a rule
// written down in CLAUDE.md after this system got it wrong once. Every entry
// here says which it is (`class`) and whether it belongs in profit and loss
// (`pnl`), so nobody downstream has to guess.
//
// ⚠️ **The P&L entries add up to the panel's money report, and a test holds
// that** (TestMoneyLedgerAgreesWithFinanceReport). An accountant's figure that
// differs from the owner's screen is an argument nobody can settle.
//
// ⚠️ **Computed on read, never stored.** No second copy of the money to drift
// from the documents, and nothing to backfill: an expense deleted yesterday is
// simply not in today's answer. The price is that a period is re-read rather
// than streamed — the contract says so: fetch a period again and *replace* it.
//
// ⚠️ **No customer is named anywhere in this file.** Amounts, methods,
// suppliers and staff; never a guest's name or phone. That is `orders:read`,
// a different key, and an accountant does not need it.

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// Money classes. ⚠️ Published strings, like the scopes.
const (
	// Profit and loss.
	MoneyRevenue    = "revenue"    // a sale, paid
	MoneyRefund     = "refund"     // a sale's money handed back
	MoneyCost       = "cost"       // purchases, expenses, outside delivery
	MoneyPayroll    = "payroll"    // wages paid, staff and couriers
	MoneyCommission = "commission" // what an aggregator or acquirer kept

	// Not profit and loss — the reason this ledger exists.
	MoneyTransfer = "transfer" // the same money in another place
	MoneyAdvance  = "advance"  // cash handed to somebody to spend for us
	MoneyManual   = "manual"   // cash in/out of a drawer or safe, by hand
	MoneyVariance = "variance" // a drawer counted over or under
)

// moneyPnL says which classes belong in profit and loss.
var moneyPnL = map[string]bool{
	MoneyRevenue: true, MoneyRefund: true, MoneyCost: true,
	MoneyPayroll: true, MoneyCommission: true,
}

// The longest period one request may ask for. A month of a busy restaurant is
// ten thousand sales; more than that in one answer is a request that should
// have been two.
const moneyMaxDays = 31

// MoneyEntry is one movement. ⚠️ Published shape.
type MoneyEntry struct {
	// Stable across requests: `<source>_<document id>`, so a period fetched
	// twice can be matched line by line.
	ID     string `json:"id"`
	Source string `json:"source"`
	Class  string `json:"class"`
	PnL    bool   `json:"pnl"`
	// "in" or "out", from the restaurant's side. A transfer is "in" to `to`.
	Direction  string    `json:"direction"`
	Amount     int       `json:"amount"`
	OccurredAt time.Time `json:"occurredAt"`
	// ⚠️ The restaurant's own calendar day, computed here. `occurredAt` is UTC
	// and slicing it gives the day before for everything after 19:00 local —
	// see CLAUDE.md §10.
	Day      string `json:"day"`
	BranchID string `json:"branchId"`

	// How the money moved: cash, card, transfer, payme, click, uzum, …
	Method string `json:"method,omitempty"`
	// The till button's own name ("Humo terminal"), when a sale has one.
	MethodName string `json:"methodName,omitempty"`
	// The restaurant's own word for it — free text, never an enum.
	Category string `json:"category,omitempty"`
	// Supplier, employee or aggregator. Never a guest.
	Counterparty string `json:"counterparty,omitempty"`
	// For transfers: where the money was and where it went (till, safe, bank,
	// aggregator, courier, staff).
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`

	Ref  moneyRef   `json:"ref"`
	Sale *moneySale `json:"sale,omitempty"`
	Paid *moneyPaid `json:"paid,omitempty"`
	Note string     `json:"note,omitempty"`
}

type moneyRef struct {
	Type   string `json:"type"`
	ID     string `json:"id"`
	Number string `json:"number,omitempty"`
}

// moneySale is the arithmetic inside a sale's amount. `amount` is what was
// charged; these are its parts, not further money.
type moneySale struct {
	Type          string `json:"type"`
	Channel       string `json:"channel,omitempty"`
	Subtotal      int    `json:"subtotal"`
	DiscountTotal int    `json:"discountTotal"`
	PointsSpent   int    `json:"pointsSpent"`
	DeliveryFee   int    `json:"deliveryFee"`
	ServiceCharge int    `json:"serviceCharge"`
}

// moneyPaid is a purchase's settlement: booked when the goods came, paid when
// the supplier was paid — which is often not the same day, or the same month.
type moneyPaid struct {
	Paid   bool       `json:"paid"`
	PaidAt *time.Time `json:"paidAt,omitempty"`
}

// moneySources is everything the ledger reads, loaded for one period.
type moneySources struct {
	Orders      []models.Order
	Purchases   []models.Purchase
	Expenses    []models.Expense
	StaffPays   []models.StaffPayment
	CourierPays []models.CourierPayment
	Payouts     []models.Payout
	Collections []models.Collection
	Advances    []models.StaffAdvance
	CashEntries []models.CashEntry
	SafeEntries []models.SafeEntry
	Settlements []models.CourierSettlement
	Shifts      []models.CashShift
}

func moneyDay(t time.Time) string { return t.In(time.Local).Format("2006-01-02") }

// moneyEntries builds the ledger for [from, to) from its sources. Pure, so the
// rules can be tested without a database.
func moneyEntries(src *moneySources, from, to time.Time) []MoneyEntry {
	out := []MoneyEntry{}
	add := func(e MoneyEntry) {
		if e.Amount <= 0 || e.OccurredAt.Before(from) || !e.OccurredAt.Before(to) {
			return
		}
		e.PnL = moneyPnL[e.Class]
		e.Day = moneyDay(e.OccurredAt)
		out = append(out, e)
	}

	seen := map[primitive.ObjectID]bool{}
	for i := range src.Orders {
		o := &src.Orders[i]
		if seen[o.ID] {
			continue
		}
		seen[o.ID] = true
		ref := moneyRef{Type: "order", ID: o.ID.Hex(), Number: o.Number}
		refunded := o.PaymentStatus == models.PayRefunded
		// ⚠️ The same test the money report and the dashboard use — see
		// `received`. A refunded sale was money in and is money out, so it is
		// both: the sale on its day, the refund on its own.
		if received(*o) || refunded {
			add(MoneyEntry{
				ID: "sale_" + o.ID.Hex(), Source: "sale", Class: MoneyRevenue, Direction: "in",
				Amount: o.Total, OccurredAt: o.CreatedAt, BranchID: hexOrEmpty(o.BranchID),
				Method: o.PaymentMethod, MethodName: o.PaymentOptionName, Ref: ref,
				Sale: &moneySale{
					Type: o.Type, Channel: o.Channel, Subtotal: o.Subtotal,
					DiscountTotal: o.DiscountTotal, PointsSpent: o.PointsSpent,
					DeliveryFee: o.DeliveryFee, ServiceCharge: o.ServiceCharge,
				},
			})
		}
		if refunded {
			at, method, note := o.UpdatedAt, o.PaymentMethod, ""
			if o.Refund != nil {
				at, note = o.Refund.At, o.Refund.Reason
				if o.Refund.Method != "" {
					method = o.Refund.Method
				}
			}
			amount := o.Total
			if o.Refund != nil && o.Refund.Amount > 0 {
				amount = o.Refund.Amount
			}
			add(MoneyEntry{
				ID: "refund_" + o.ID.Hex(), Source: "refund", Class: MoneyRefund, Direction: "out",
				Amount: amount, OccurredAt: at, BranchID: hexOrEmpty(o.BranchID),
				Method: method, Ref: ref, Note: note,
			})
		}
		// Somebody else's courier, paid per order — the money report counts it
		// on the order's day, and so does this.
		if o.ExternalDelivery != nil && o.ExternalDelivery.Cost > 0 {
			add(MoneyEntry{
				ID: "delivery_" + o.ID.Hex(), Source: "delivery_service", Class: MoneyCost,
				Direction: "out", Amount: o.ExternalDelivery.Cost, OccurredAt: o.CreatedAt,
				BranchID: hexOrEmpty(o.BranchID), Counterparty: o.ExternalDelivery.ProviderName,
				Ref: ref,
			})
		}
	}

	// ⚠️ Booked on the day the goods came (`at`), as the money report does;
	// whether the supplier has been paid rides along in `paid`. An unpaid
	// delivery is still food that was bought.
	for _, p := range src.Purchases {
		e := MoneyEntry{
			ID: "purchase_" + p.ID.Hex(), Source: "purchase", Class: MoneyCost, Direction: "out",
			Amount: p.Total, OccurredAt: p.At, BranchID: hexOrEmpty(p.BranchID),
			Counterparty: p.Supplier, Ref: moneyRef{Type: "purchase", ID: p.ID.Hex()},
			Paid: &moneyPaid{Paid: p.Paid, PaidAt: p.PaidAt}, Note: p.Note,
		}
		add(e)
	}
	for _, x := range src.Expenses {
		add(MoneyEntry{
			ID: "expense_" + x.ID.Hex(), Source: "expense", Class: MoneyCost, Direction: "out",
			Amount: x.Amount, OccurredAt: x.At, BranchID: hexOrEmpty(x.BranchID),
			Method: x.Method, Category: x.Category, Note: x.Note,
			Ref: moneyRef{Type: "expense", ID: x.ID.Hex()},
		})
	}
	for _, p := range src.StaffPays {
		add(MoneyEntry{
			ID: "salary_" + p.ID.Hex(), Source: "salary", Class: MoneyPayroll, Direction: "out",
			Amount: p.Amount, OccurredAt: p.At, BranchID: hexOrEmpty(p.BranchID),
			Ref: moneyRef{Type: "staff_payment", ID: p.ID.Hex()}, Note: p.Note,
		})
	}
	for _, p := range src.CourierPays {
		add(MoneyEntry{
			ID: "courierpay_" + p.ID.Hex(), Source: "courier_pay", Class: MoneyPayroll, Direction: "out",
			Amount: p.Amount, OccurredAt: p.At, BranchID: hexOrEmpty(p.BranchID),
			Ref: moneyRef{Type: "courier_payment", ID: p.ID.Hex()}, Note: p.Note,
		})
	}

	// ⚠️ **One transfer, two entries.** What the aggregator kept is a cost; what
	// arrived is the guest's money changing place — the sale was already counted
	// the day the guest paid. `net` is theirs, never `gross − commission`: the
	// difference is a real event (a chargeback, a penalty) and must stay visible.
	for _, p := range src.Payouts {
		ref := moneyRef{Type: "payout", ID: p.ID.Hex()}
		name := p.ProviderName
		if name == "" {
			name = p.Provider
		}
		add(MoneyEntry{
			ID: "commission_" + p.ID.Hex(), Source: "payout", Class: MoneyCommission, Direction: "out",
			Amount: p.Commission, OccurredAt: p.ReceivedAt, BranchID: hexOrEmpty(p.BranchID),
			Counterparty: name, Ref: ref,
		})
		add(MoneyEntry{
			ID: "payout_" + p.ID.Hex(), Source: "payout", Class: MoneyTransfer, Direction: "in",
			Amount: p.Net, OccurredAt: p.ReceivedAt, BranchID: hexOrEmpty(p.BranchID),
			Counterparty: name, From: "aggregator", To: "bank", Ref: ref, Note: p.Note,
		})
	}
	for _, c := range src.Collections {
		to := c.To
		if to == "" {
			to = models.CollectionToBank
		}
		add(MoneyEntry{
			ID: "collection_" + c.ID.Hex(), Source: "collection", Class: MoneyTransfer, Direction: "in",
			Amount: c.Amount, OccurredAt: c.At, BranchID: hexOrEmpty(c.BranchID),
			From: "till", To: to, Ref: moneyRef{Type: "collection", ID: c.ID.Hex()}, Note: c.Note,
		})
	}
	// ⚠️ An advance is spent when it buys something, and that is a purchase —
	// already a cost above. Counting the advance too books the market twice.
	for _, a := range src.Advances {
		e := MoneyEntry{
			ID: "advance_" + a.ID.Hex(), Source: "advance", Class: MoneyAdvance,
			Amount: a.Amount, OccurredAt: a.At, BranchID: hexOrEmpty(a.BranchID),
			Counterparty: a.StaffName, Ref: moneyRef{Type: "staff_advance", ID: a.ID.Hex()}, Note: a.Note,
		}
		if a.Kind == models.AdvanceBack {
			e.Direction, e.From = "in", "staff"
		} else {
			e.Direction, e.To = "out", "staff"
		}
		add(e)
	}
	// Cash put into or taken out of a drawer by hand. ⚠️ Not P&L in the money
	// report, and not here: the category is the cashier's own word, and
	// "mahsulot" taken from the drawer is usually the same food a purchase
	// already records.
	for _, c := range src.CashEntries {
		add(MoneyEntry{
			ID: "cash_" + c.ID.Hex(), Source: "cash_entry", Class: MoneyManual,
			Direction: moneyDir(c.Kind), Amount: c.Amount, OccurredAt: c.At,
			BranchID: hexOrEmpty(c.BranchID), Category: c.Category, Note: c.Note,
			From: moneyIf(c.Kind != models.CashIn, "till"), To: moneyIf(c.Kind == models.CashIn, "till"),
			Ref: moneyRef{Type: "cash_entry", ID: c.ID.Hex()},
		})
	}
	// ⚠️ **Only safe entries made by hand.** One with a `refKind` is the other
	// half of a document already above — a wage, an expense, a collection paid
	// out of or into the safe — and listing it again would count that money
	// twice.
	for _, s := range src.SafeEntries {
		if s.RefKind != "" {
			continue
		}
		add(MoneyEntry{
			ID: "safe_" + s.ID.Hex(), Source: "safe_entry", Class: MoneyManual,
			Direction: moneyDir(s.Kind), Amount: s.Amount, OccurredAt: s.At,
			BranchID: hexOrEmpty(s.BranchID), Category: s.Category, Note: s.Note,
			From: moneyIf(s.Kind != models.SafeIn, "safe"), To: moneyIf(s.Kind == models.SafeIn, "safe"),
			Ref: moneyRef{Type: "safe_entry", ID: s.ID.Hex()},
		})
	}
	// Cash a courier collected on our behalf, handed into the till. ⚠️ Not
	// income: the sale it came from was counted when it was delivered.
	for _, s := range src.Settlements {
		add(MoneyEntry{
			ID: "settlement_" + s.ID.Hex(), Source: "courier_settlement", Class: MoneyTransfer,
			Direction: "in", Amount: s.Amount, OccurredAt: s.At, From: "courier", To: "till",
			Ref: moneyRef{Type: "courier_settlement", ID: s.ID.Hex()}, Note: s.Note,
		})
	}
	// A drawer that counted over or under. ⚠️ Missing money is not spent money
	// — nothing was bought and no document exists.
	for _, s := range src.Shifts {
		if s.ClosedAt == nil || s.Variance == 0 {
			continue
		}
		dir, amount := "in", s.Variance
		if amount < 0 {
			dir, amount = "out", -amount
		}
		add(MoneyEntry{
			ID: "variance_" + s.ID.Hex(), Source: "shift_variance", Class: MoneyVariance,
			Direction: dir, Amount: amount, OccurredAt: *s.ClosedAt, BranchID: hexOrEmpty(s.BranchID),
			From: moneyIf(dir == "out", "till"), To: moneyIf(dir == "in", "till"),
			Ref: moneyRef{Type: "cash_shift", ID: s.ID.Hex()}, Note: s.VarianceNote,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].OccurredAt.Equal(out[j].OccurredAt) {
			return out[i].OccurredAt.Before(out[j].OccurredAt)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func moneyDir(kind string) string {
	if kind == "in" {
		return "in"
	}
	return "out"
}

func moneyIf(ok bool, s string) string {
	if ok {
		return s
	}
	return ""
}

type moneyClassTotal struct {
	In    int `json:"in"`
	Out   int `json:"out"`
	Count int `json:"count"`
}

// moneyTotals sums the entries per class, and the profit-and-loss net — the
// figure the panel's money report calls "Kirim − chiqim".
func moneyTotals(entries []MoneyEntry) (map[string]moneyClassTotal, int) {
	by := map[string]moneyClassTotal{}
	net := 0
	for _, e := range entries {
		t := by[e.Class]
		t.Count++
		if e.Direction == "in" {
			t.In += e.Amount
		} else {
			t.Out += e.Amount
		}
		by[e.Class] = t
		if e.PnL {
			if e.Direction == "in" {
				net += e.Amount
			} else {
				net -= e.Amount
			}
		}
	}
	return by, net
}

// loadMoneySources reads every document that can put an entry in [from, to).
func (h *Handler) loadMoneySources(ctx context.Context, branch bson.M, from, to time.Time) (*moneySources, error) {
	in := func(field string) bson.M {
		f := bson.M{field: bson.M{"$gte": from, "$lt": to}}
		for k, v := range branch {
			f[k] = v
		}
		return f
	}
	src := &moneySources{}
	load := func(c *mongo.Collection, filter bson.M, into any) error {
		cur, err := c.Find(ctx, filter)
		if err != nil {
			return err
		}
		return cur.All(ctx, into)
	}
	// Sales by the day they were made; refunds by the day they were given back,
	// which can be a later period than the sale.
	refunds := bson.M{"paymentStatus": models.PayRefunded, "$or": bson.A{
		bson.M{"refund.at": bson.M{"$gte": from, "$lt": to}},
		bson.M{"refund": nil, "updatedAt": bson.M{"$gte": from, "$lt": to}},
	}}
	for k, v := range branch {
		refunds[k] = v
	}
	var refunded []models.Order
	steps := []error{
		load(h.Store.Orders, in("createdAt"), &src.Orders),
		load(h.Store.Orders, refunds, &refunded),
		load(h.Store.Purchases, in("at"), &src.Purchases),
		load(h.Store.Expenses, in("at"), &src.Expenses),
		load(h.Store.StaffPayments, in("at"), &src.StaffPays),
		load(h.Store.CourierPayments, in("at"), &src.CourierPays),
		load(h.Store.Payouts, in("receivedAt"), &src.Payouts),
		load(h.Store.Collections, in("at"), &src.Collections),
		load(h.Store.Advances, in("at"), &src.Advances),
		load(h.Store.CashEntries, in("at"), &src.CashEntries),
		load(h.Store.SafeEntries, in("at"), &src.SafeEntries),
		load(h.Store.CashShifts, in("closedAt"), &src.Shifts),
	}
	// ⚠️ A settlement has no branch of its own — it belongs to a courier. Only
	// read when the whole company is asked for, rather than attributed to a
	// branch it may not belong to.
	if len(branch) == 0 {
		steps = append(steps, load(h.Store.Settlements, bson.M{"at": bson.M{"$gte": from, "$lt": to}}, &src.Settlements))
	}
	for _, err := range steps {
		if err != nil {
			return nil, err
		}
	}
	src.Orders = append(src.Orders, refunded...)
	return src, nil
}

// OpenMoney is the ledger for a period of at most 31 days.
//
// `from` / `to` are the restaurant's calendar days (`2026-09-01`), both
// inclusive, or RFC 3339 instants. `branchId` narrows to one branch.
func (h *Handler) OpenMoney(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("from") == "" || q.Get("to") == "" {
		openFail(w, http.StatusBadRequest, openCodeInvalid, "from and to are required, e.g. from=2026-09-01&to=2026-09-30")
		return
	}
	from, to, err := parseRange(q.Get("from"), q.Get("to"))
	if err != nil || from == nil || to == nil {
		openFail(w, http.StatusBadRequest, openCodeInvalid, "from and to must be YYYY-MM-DD or RFC 3339, and from must not be after to")
		return
	}
	if to.Sub(*from) > moneyMaxDays*24*time.Hour+time.Hour {
		openFail(w, http.StatusBadRequest, openCodeInvalid, "a period is at most 31 days — ask for a month at a time")
		return
	}
	branch := bson.M{}
	if v := strings.TrimSpace(q.Get("branchId")); v != "" {
		id, err := primitive.ObjectIDFromHex(v)
		if err != nil {
			openFail(w, http.StatusBadRequest, openCodeInvalid, "branchId is not a valid id")
			return
		}
		branch["branchId"] = id
	}
	src, err := h.loadMoneySources(r.Context(), branch, *from, *to)
	if err != nil {
		openFail(w, http.StatusInternalServerError, openCodeInternal, "the ledger could not be read")
		return
	}
	entries := moneyEntries(src, *from, *to)
	byClass, net := moneyTotals(entries)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"from": from.UTC(), "to": to.UTC(), "currency": "UZS",
		"entries": entries,
		"totals":  map[string]any{"byClass": byClass, "pnlNet": net},
	})
}
