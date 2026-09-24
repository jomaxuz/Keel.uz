package handlers

// ---- The rest of the accountant's view: days, positions, fiscal, and a nudge ----
//
// Beside the money ledger (openmoney.go), all under `finance:read`:
//
//   - `GET /money/daily`  — the ledger summed per day and branch.
//   - `GET /balances`     — where the money is right now, and who owes whom.
//   - `GET /fiscal`       — receipts filed with the tax committee, and Z-reports.
//   - `money.day_changed` — a webhook naming a day whose money changed.
//
// ⚠️ **Every figure here is derived from the ledger or from the panel's own
// arithmetic, never written a second time.** A daily total that is its own
// query is a daily total that disagrees with the entries it claims to sum.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// openPeriod reads `from`, `to` and `branchId` the way `/money` does, or
// answers the error itself.
func openPeriod(w http.ResponseWriter, r *http.Request, maxDays int) (time.Time, time.Time, bson.M, bool) {
	q := r.URL.Query()
	if q.Get("from") == "" || q.Get("to") == "" {
		openFail(w, http.StatusBadRequest, openCodeInvalid, "from and to are required, e.g. from=2026-09-01&to=2026-09-30")
		return time.Time{}, time.Time{}, nil, false
	}
	from, to, err := parseRange(q.Get("from"), q.Get("to"))
	if err != nil || from == nil || to == nil {
		openFail(w, http.StatusBadRequest, openCodeInvalid, "from and to must be YYYY-MM-DD or RFC 3339, and from must not be after to")
		return time.Time{}, time.Time{}, nil, false
	}
	if to.Sub(*from) > time.Duration(maxDays)*24*time.Hour+time.Hour {
		openFail(w, http.StatusBadRequest, openCodeInvalid,
			fmt.Sprintf("a period is at most %d days", maxDays))
		return time.Time{}, time.Time{}, nil, false
	}
	branch, ok := openBranchFilter(w, r)
	return *from, *to, branch, ok
}

func openBranchFilter(w http.ResponseWriter, r *http.Request) (bson.M, bool) {
	branch := bson.M{}
	if v := strings.TrimSpace(r.URL.Query().Get("branchId")); v != "" {
		id, err := primitive.ObjectIDFromHex(v)
		if err != nil {
			openFail(w, http.StatusBadRequest, openCodeInvalid, "branchId is not a valid id")
			return nil, false
		}
		branch["branchId"] = id
	}
	return branch, true
}

// ---- GET /money/daily ----

// moneyDaily is one day of one branch. ⚠️ Published shape.
type moneyDaily struct {
	Day      string                     `json:"day"`
	BranchID string                     `json:"branchId"`
	ByClass  map[string]moneyClassTotal `json:"byClass"`
	// Revenue split by how it was paid — the line an accountant matches
	// against the bank statement and the drawer.
	RevenueByMethod map[string]int `json:"revenueByMethod"`
	PnLNet          int            `json:"pnlNet"`
}

// moneyDays groups ledger entries by (day, branch), in day order.
func moneyDays(entries []MoneyEntry) []moneyDaily {
	type key struct{ day, branch string }
	groups := map[key][]MoneyEntry{}
	for _, e := range entries {
		k := key{e.Day, e.BranchID}
		groups[k] = append(groups[k], e)
	}
	out := make([]moneyDaily, 0, len(groups))
	for k, es := range groups {
		byClass, net := moneyTotals(es)
		byMethod := map[string]int{}
		for _, e := range es {
			if e.Class == MoneyRevenue {
				m := e.Method
				if m == "" {
					m = "other"
				}
				byMethod[m] += e.Amount
			}
		}
		out = append(out, moneyDaily{
			Day: k.day, BranchID: k.branch, ByClass: byClass, RevenueByMethod: byMethod, PnLNet: net,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Day != out[j].Day {
			return out[i].Day < out[j].Day
		}
		return out[i].BranchID < out[j].BranchID
	})
	return out
}

// OpenMoneyDaily is the ledger summed per day and branch — up to 93 days,
// because a quarter is the period an accountant closes.
func (h *Handler) OpenMoneyDaily(w http.ResponseWriter, r *http.Request) {
	from, to, branch, ok := openPeriod(w, r, 93)
	if !ok {
		return
	}
	src, err := h.loadMoneySources(r.Context(), branch, from, to)
	if err != nil {
		openFail(w, http.StatusInternalServerError, openCodeInternal, "the ledger could not be read")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"from": from.UTC(), "to": to.UTC(), "currency": "UZS",
		"days": moneyDays(moneyEntries(src, from, to)),
	})
}

// ---- GET /balances ----

type openPlace struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Amount int    `json:"amount"`
	// Counted (a drawer at close, a balance read off a bank app — goes
	// stale) or added up from documents (goes wrong when one is missing).
	Counted bool       `json:"counted"`
	At      *time.Time `json:"at,omitempty"`
	Note    string     `json:"note,omitempty"`
}

type openOwed struct {
	Amount int `json:"amount"`
	Count  int `json:"count"`
}

func openPlaces(in []models.MoneyPlace) []openPlace {
	out := make([]openPlace, 0, len(in))
	for _, p := range in {
		out = append(out, openPlace{
			Kind: p.Kind, Name: p.Name, Amount: p.Amount, Counted: p.Counted, At: p.At, Note: p.Note,
		})
	}
	return out
}

// OpenBalances is where the money is right now — the panel's "Pul qayerda"
// screen (moneyPosition), plus what is owed each way.
//
// ⚠️ **Three totals, and no fourth.** Cash can be spent tonight, the bank this
// week, and money an aggregator holds when they decide. There is no "total"
// field on purpose — see handlers/money.go.
func (h *Handler) OpenBalances(w http.ResponseWriter, r *http.Request) {
	branch, ok := openBranchFilter(w, r)
	if !ok {
		return
	}
	pos := h.moneyPosition(r, branch)

	// Owed by the restaurant: deliveries not yet paid for.
	unpaid := scopeFilter(branch)
	unpaid["paid"] = bson.M{"$ne": true}
	suppliers, suppliersN, _ := h.sumField(r.Context(), h.Store.Purchases, unpaid, "$total")
	// Owed to the restaurant: meals eaten on the slate and not yet paid.
	slate := scopeFilter(branch)
	slate["paymentMethod"] = models.MethodDebt
	slate["paymentStatus"] = bson.M{"$ne": models.PayPaid}
	slate["status"] = bson.M{"$ne": models.StatusCancelled}
	guests, guestsN, _ := h.sumField(r.Context(), h.Store.Orders, slate, "$total")

	httpx.JSON(w, http.StatusOK, map[string]any{
		"asOf": time.Now().UTC(), "currency": "UZS", "branchId": hexOrEmpty(branchIDOf(branch)),
		"cash": openPlaces(pos.Cash), "cashTotal": pos.CashTotal,
		"bank": openPlaces(pos.Bank), "bankTotal": pos.BankTotal,
		"inTransit": openPlaces(pos.Rails), "inTransitTotal": pos.RailsTotal,
		"cashLimit": pos.CashLimit, "overCashLimit": pos.OverLimit,
		"payables":    map[string]openOwed{"suppliers": {suppliers, suppliersN}},
		"receivables": map[string]openOwed{"guestDebt": {guests, guestsN}},
	})
}

func branchIDOf(branch bson.M) primitive.ObjectID {
	id, _ := branch["branchId"].(primitive.ObjectID)
	return id
}

// ---- GET /fiscal ----

type openFiscalReceipt struct {
	OrderID     string     `json:"orderId"`
	OrderNumber string     `json:"orderNumber"`
	BranchID    string     `json:"branchId"`
	Kind        string     `json:"kind"` // "sale" | "refund"
	Total       int        `json:"total"`
	Method      string     `json:"method"`
	Status      string     `json:"status"`
	Provider    string     `json:"provider"`
	FiscalSign  string     `json:"fiscalSign,omitempty"`
	ReceiptID   string     `json:"receiptId,omitempty"`
	QRText      string     `json:"qrText,omitempty"`
	FiledAt     *time.Time `json:"filedAt,omitempty"`
	Error       string     `json:"error,omitempty"`
	SoldAt      time.Time  `json:"soldAt"`
}

type openZReport struct {
	ShiftID     string    `json:"shiftId"`
	BranchID    string    `json:"branchId"`
	Number      string    `json:"number,omitempty"`
	SaleCash    int       `json:"saleCash"`
	SaleCard    int       `json:"saleCard"`
	SaleTotal   int       `json:"saleTotal"`
	SaleCount   int       `json:"saleCount"`
	RefundTotal int       `json:"refundTotal"`
	ClosedAt    time.Time `json:"closedAt"`
	Error       string    `json:"error,omitempty"`
}

func fiscalOf(o *models.Order, kind string, f *models.FiscalReceipt) openFiscalReceipt {
	return openFiscalReceipt{
		OrderID: o.ID.Hex(), OrderNumber: o.Number, BranchID: hexOrEmpty(o.BranchID), Kind: kind,
		Total: o.Total, Method: o.PaymentMethod, Status: string(f.Status), Provider: f.Provider,
		FiscalSign: f.FiscalSign, ReceiptID: f.ReceiptID, QRText: f.QRText, FiledAt: f.FiledAt,
		Error: f.Error, SoldAt: o.CreatedAt,
	}
}

// OpenFiscal lists the receipts filed with the tax committee for sales made in
// the period, and the Z-reports of the shifts closed in it.
//
// ⚠️ **Every status, not only the filed ones.** A receipt still pending or
// refused is exactly what an accountant has to chase before the month closes;
// a list of the successful ones hides the work.
func (h *Handler) OpenFiscal(w http.ResponseWriter, r *http.Request) {
	from, to, branch, ok := openPeriod(w, r, moneyMaxDays)
	if !ok {
		return
	}
	ctx := r.Context()
	filter := scopeFilter(branch)
	filter["createdAt"] = bson.M{"$gte": from, "$lt": to}
	filter["$or"] = bson.A{bson.M{"fiscal": bson.M{"$ne": nil}}, bson.M{"fiscalRefund": bson.M{"$ne": nil}}}
	cur, err := h.Store.Orders.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: 1}}).
		SetProjection(bson.M{"items": 0, "statusHistory": 0, "customer": 0, "address": 0, "check": 0}))
	if err != nil {
		openFail(w, http.StatusInternalServerError, openCodeInternal, "receipts could not be read")
		return
	}
	var orders []models.Order
	if err := cur.All(ctx, &orders); err != nil {
		openFail(w, http.StatusInternalServerError, openCodeInternal, "receipts could not be read")
		return
	}
	receipts := []openFiscalReceipt{}
	for i := range orders {
		o := &orders[i]
		if o.Fiscal != nil {
			receipts = append(receipts, fiscalOf(o, "sale", o.Fiscal))
		}
		if o.FiscalRefund != nil {
			receipts = append(receipts, fiscalOf(o, "refund", o.FiscalRefund))
		}
	}

	shiftFilter := scopeFilter(branch)
	shiftFilter["closedAt"] = bson.M{"$gte": from, "$lt": to}
	shiftFilter["fiscal"] = bson.M{"$ne": nil}
	days := []openZReport{}
	if cur, err := h.Store.CashShifts.Find(ctx, shiftFilter,
		options.Find().SetSort(bson.D{{Key: "closedAt", Value: 1}})); err == nil {
		var shifts []models.CashShift
		if cur.All(ctx, &shifts) == nil {
			for _, s := range shifts {
				f := s.Fiscal
				days = append(days, openZReport{
					ShiftID: s.ID.Hex(), BranchID: hexOrEmpty(s.BranchID), Number: f.Number,
					SaleCash: f.SaleCash, SaleCard: f.SaleCard, SaleTotal: f.SaleTotal,
					SaleCount: f.SaleCount, RefundTotal: f.RefundTotal, ClosedAt: f.ClosedAt, Error: f.Error,
				})
			}
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"from": from.UTC(), "to": to.UTC(), "receipts": receipts, "zReports": days,
	})
}

// ---- money.day_changed: the watcher ----

// How far back the watcher looks. A month and a bit: the month an accountant
// is closing, and the days before it that get corrected while they do.
const moneyWatchDays = 35

// moneyFingerprints hashes each (day, branch) of the ledger. Anything that
// changes a figure changes the hash: a new entry, a deleted one, an edited
// amount, a re-dated one.
func moneyFingerprints(entries []MoneyEntry) map[string]string {
	lines := map[string][]string{}
	for _, e := range entries {
		k := e.Day + "|" + e.BranchID
		lines[k] = append(lines[k], fmt.Sprintf("%s:%s:%s:%d:%d",
			e.ID, e.Class, e.Direction, e.Amount, e.OccurredAt.UnixMilli()))
	}
	out := map[string]string{}
	for k, ls := range lines {
		sort.Strings(ls)
		sum := sha256.Sum256([]byte(strings.Join(ls, "\n")))
		out[k] = hex.EncodeToString(sum[:])
	}
	return out
}

// changedDays compares two fingerprint sets over the watched window. A key
// that vanished (every entry of a day deleted) is a change too.
func changedDays(prev, now map[string]string, oldest string) []string {
	var out []string
	for k, h := range now {
		if prev[k] != h {
			out = append(out, k)
		}
	}
	for k := range prev {
		if _, ok := now[k]; !ok && k >= oldest {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// StartMoneyWatch looks for changed days every ten minutes.
//
// ⚠️ **Why a watcher, not a call at every write.** Money is written in a dozen
// handlers — expenses, purchases and their edits, wages, payouts, collections,
// advances, drawer and safe entries, shift closes, sales, refunds — and each
// can also be deleted. The order webhook holds its dozen with a test; this one
// would need twice that, and the one forgotten is a receiver whose books are
// silently wrong. Hashing the ledger catches every change wherever it was made,
// and the ledger is already how the money is read.
//
// ⚠️ **Does nothing at all unless an endpoint subscribes.** No endpoint, no
// read — so it costs every other restaurant nothing.
func (h *Handler) StartMoneyWatch(ctx context.Context) {
	go func() {
		tick := time.NewTicker(10 * time.Minute)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				h.watchMoney(ctx)
			}
		}
	}()
}

func (h *Handler) watchMoney(ctx context.Context) {
	eps, err := h.liveWebhookEndpoints(ctx)
	if err != nil {
		return
	}
	var subscribed []models.WebhookEndpoint
	for _, ep := range eps {
		if ep.Subscribes(models.EventMoneyDayChanged) {
			subscribed = append(subscribed, ep)
		}
	}
	if len(subscribed) == 0 {
		return
	}
	now := time.Now()
	y, m, d := now.In(time.Local).Date()
	to := time.Date(y, m, d+1, 0, 0, 0, 0, time.Local)
	from := to.AddDate(0, 0, -moneyWatchDays)
	src, err := h.loadMoneySources(ctx, bson.M{}, from, to)
	if err != nil {
		log.Printf("money watch: %v", err)
		return
	}
	entries := moneyEntries(src, from, to)
	prints := moneyFingerprints(entries)

	var state models.MoneyWatch
	first := h.Store.MoneyWatch.FindOne(ctx, bson.M{"_id": "state"}).Decode(&state) != nil
	save := func() {
		_, _ = h.Store.MoneyWatch.UpdateOne(ctx, bson.M{"_id": "state"},
			bson.M{"$set": bson.M{"days": prints, "at": now}}, options.Update().SetUpsert(true))
	}
	// ⚠️ **The first look is a baseline, not news.** Announcing thirty-five
	// days of "changed" the moment somebody subscribes would tell the receiver
	// nothing it did not get by reading the period — which it has to do anyway.
	if first {
		save()
		return
	}
	daily := map[string]moneyDaily{}
	for _, dd := range moneyDays(entries) {
		daily[dd.Day+"|"+dd.BranchID] = dd
	}
	queued := false
	for _, k := range changedDays(state.Days, prints, from.In(time.Local).Format("2006-01-02")) {
		day, branch, _ := strings.Cut(k, "|")
		hash := prints[k]
		if hash == "" {
			hash = "empty"
		}
		dd := daily[k]
		count := 0
		for _, c := range dd.ByClass {
			count += c.Count
		}
		env := webhookEnvelope{
			// Content-addressed: the same change reported twice is one event,
			// and a change back and forth is two.
			ID:   "evt_money_" + day + "_" + branch + "_" + hash[:min(12, len(hash))],
			Type: models.EventMoneyDayChanged, CreatedAt: now,
			Data: map[string]any{"day": day, "branchId": branch, "pnlNet": dd.PnLNet, "entries": count},
		}
		for i := range subscribed {
			// Only endpoints that existed before this look — a new one starts
			// from its own reading of the period.
			if subscribed[i].CreatedAt.After(state.At) {
				continue
			}
			if h.queueDelivery(ctx, &subscribed[i], env, day, now) {
				queued = true
			}
		}
	}
	save()
	if queued {
		kickWebhooks()
	}
}
