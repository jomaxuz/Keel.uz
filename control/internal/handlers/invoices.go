package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"keel-control/internal/billing"
	"keel-control/internal/httpx"
	"keel-control/internal/middleware"
	"keel-control/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The invoice ledger: what each customer was billed, and what we actually got.
//
// Until now the only financial state was `suspended`, which says "no money
// came" and nothing else — not how much, not for which month, not who was
// supposed to collect it. That is workable for three customers and stops being
// workable at ten, and it is completely unworkable for the way Keel actually
// takes money right now:
//
// **Cash, by hand.** Keel starts before the MChJ is registered, so there is no
// legal entity, no stamped invoice and no bank account to receive a transfer
// into. Somebody drives to the restaurant and is handed so'm. That is not a
// workaround to be tidied up later — it is the MVP, and the ledger records it
// as a first-class fact rather than pretending a bank was involved.
// `transfer` is in the model from day one, unused, so that adding it after
// registration does not mean re-reading every stored row to work out what it
// meant.
//
// The two rules that make this a ledger rather than a total:
//
//   - **The amount is frozen at issue.** Daily rows keep accruing; the number
//     the customer was told does not.
//   - **A payment is a record with a name on it.** Unsigned cash becomes an
//     argument three weeks later — the same reason courier settlements and
//     staff salaries are written down rather than decremented.

// InvoiceView is an invoice plus the two numbers every screen recomputes.
type InvoiceView struct {
	models.Invoice
	Collected   int `json:"collected"`
	Outstanding int `json:"outstanding"`
}

func view(i models.Invoice) InvoiceView {
	if i.Paid == nil {
		// Go marshals a nil slice as null, and the console would do
		// `invoice.paid.length` on it.
		i.Paid = []models.InvoicePayment{}
	}
	return InvoiceView{Invoice: i, Collected: i.Collected(), Outstanding: i.Outstanding()}
}

// ListInvoices is the ledger, newest first.
//
// The default is deliberately **open invoices across every customer**: the
// question this page exists to answer is "who owes us money", and a list that
// opens on everything ever issued buries it.
func (h *Handler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	filter := bson.M{}
	switch status := r.URL.Query().Get("status"); status {
	case models.InvoiceOpen, models.InvoicePaid, models.InvoiceVoid:
		filter["status"] = status
	case "all":
		// everything
	default:
		filter["status"] = models.InvoiceOpen
	}
	if id := r.URL.Query().Get("tenantId"); id != "" {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
			return
		}
		filter["tenantId"] = oid
		// One customer's card wants their whole history, not only what is open.
		delete(filter, "status")
	}

	cur, err := h.Store.Invoices.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "from", Value: -1}}).SetLimit(500))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var rows []models.Invoice
	if err := cur.All(r.Context(), &rows); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]InvoiceView, 0, len(rows))
	outstanding := 0
	for _, i := range rows {
		v := view(i)
		outstanding += v.Outstanding
		items = append(items, v)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"items": items,
		// The headline: what is owed across everything on screen.
		"outstanding": outstanding,
	})
}

// IssueInvoice freezes a period into a bill.
//
// Defaults to the tenant's **previous** period, not the current one. Billing a
// month that has not finished asks a restaurant to pay for orders it has not
// taken yet, and the number would be wrong by tomorrow anyway.
func (h *Handler) IssueInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var t models.Tenant
	if err := h.Store.Tenants.FindOne(r.Context(), bson.M{"_id": id}).Decode(&t); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
		Note string `json:"note"`
	}
	_ = httpx.Decode(r, &req)

	from, to := strings.TrimSpace(req.From), strings.TrimSpace(req.To)
	if from == "" || to == "" {
		from, to = previousPeriod(t, time.Now())
	}
	if from >= to {
		httpx.Error(w, http.StatusBadRequest, "davr noto'g'ri")
		return
	}

	orders, revenue, billable, err := h.sumDays(r.Context(), t.ID, from, to)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// What the customer is actually asked for: the volume ladder over the
	// period's orders, then free terms, then any standing discount.
	//
	// ⚠️ **Recomputed from the order count, not summed from the daily rows.**
	// The tiers are a monthly ladder; applied per day they would reset every
	// midnight and a busy restaurant would never leave the first band. So
	// `sumDays` gives the raw daily estimate and this is the real price — the
	// two are allowed to differ, and this is the one that is right.
	//
	// Applied here, at issue, so the frozen amount is the amount that was
	// agreed: a discount changed next month must not silently re-price an
	// invoice already sent.
	//
	// A free customer still gets an invoice, for zero. It is the record that
	// the month happened and that it was deliberately not charged: an account
	// with a gap where its invoices should be is one nobody can explain later.
	raw := billable
	billable = t.ChargeForOrders(orders, h.Cfg.PriceTiers, h.Cfg.MinMonthly, time.Now())

	// ⚠️ The add-on is added **after** the order fee and its discounts, and never inside them.
	//
	// Free terms and a standing discount are about what the restaurant sells; removing the
	// badge is a thing they bought. Folding it into the same number would mean a free customer
	// getting the add-on for nothing and a discounted one paying a discounted price for it —
	// neither was agreed. Billed by the day, so a badge hidden yesterday is not a month's fee:
	// see billing.WatermarkFee.
	watermark := 0
	if t.HideWatermark {
		since := time.Time{}
		if t.HideWatermarkSince != nil {
			since = *t.HideWatermarkSince
		}
		// The package's own reader, not a second one: two readings of the same date format
		// is how two halves of one bill end up disagreeing about where a month starts.
		fromDay, err1 := parseDay(from)
		toDay, err2 := parseDay(to)
		if err1 == nil && err2 == nil {
			watermark = billing.WatermarkFee(h.Cfg.WatermarkPrice, fromDay, toDay, since)
			billable += watermark
		}
	}

	note := strings.TrimSpace(req.Note)
	if billable != raw && note == "" {
		note = billingNote(t, orders, raw, billable)
	}

	number, err := h.nextInvoiceNumber(r.Context(), from)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	now := time.Now()
	inv := models.Invoice{
		TenantID: t.ID,
		Slug:     t.Slug,
		Name:     t.Name,
		Number:   number,
		From:     from,
		To:       to,
		Orders:   orders,
		Revenue:  revenue,
		Amount:   billable,
		// In the total as well; kept separately so a bill three million larger says why.
		WatermarkFee: watermark,
		Status:       models.InvoiceOpen,
		Paid:         []models.InvoicePayment{},
		IssuedBy:     currentUser(r),
		Note:         note,

		CreatedAt: now,
		UpdatedAt: now,
	}
	res, err := h.Store.Invoices.InsertOne(r.Context(), inv)
	if mongo.IsDuplicateKeyError(err) {
		// Already billed. Returned rather than refused: a double click and a
		// retried request both land here, and the operator wants the invoice,
		// not an error about one that exists.
		var existing models.Invoice
		if err := h.Store.Invoices.FindOne(r.Context(), bson.M{
			"tenantId": t.ID, "from": from, "to": to,
		}).Decode(&existing); err == nil {
			httpx.JSON(w, http.StatusOK, view(existing))
			return
		}
		httpx.Error(w, http.StatusConflict, "bu davr uchun hisob allaqachon chiqarilgan")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	inv.ID = res.InsertedID.(primitive.ObjectID)
	httpx.JSON(w, http.StatusOK, view(inv))
}

// PayInvoice records money that arrived.
//
// Appends rather than sets: money comes in pieces, and the pieces are what
// answer "what did we actually receive". The invoice closes itself once the
// total covers the amount — an operator should not have to remember to flip a
// status after counting notes.
func (h *Handler) PayInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var req struct {
		Amount     int    `json:"amount"`
		Method     string `json:"method"`
		ReceivedBy string `json:"receivedBy"`
		Note       string `json:"note"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Amount <= 0 {
		httpx.Error(w, http.StatusBadRequest, "summa noldan katta bo'lishi kerak")
		return
	}
	method := strings.TrimSpace(req.Method)
	if method == "" {
		// Cash is the default because cash is how this is actually collected
		// until the MChJ exists.
		method = models.PayCash
	}
	if method != models.PayCash && method != models.PayTransfer {
		httpx.Error(w, http.StatusBadRequest, "noma'lum to'lov usuli")
		return
	}
	// Who took the money. Defaulted to the signed-in operator rather than left
	// blank: cash with nobody's name on it is the thing this ledger exists to
	// prevent, and a field an operator can skip is a field they will skip.
	receivedBy := strings.TrimSpace(req.ReceivedBy)
	if receivedBy == "" {
		receivedBy = currentUser(r)
	}

	var inv models.Invoice
	if err := h.Store.Invoices.FindOne(r.Context(), bson.M{"_id": id}).Decode(&inv); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	if inv.Status == models.InvoiceVoid {
		httpx.Error(w, http.StatusBadRequest, "bekor qilingan hisobga to'lov yozib bo'lmaydi")
		return
	}

	now := time.Now()
	payment := models.InvoicePayment{
		Amount: req.Amount, Method: method,
		ReceivedBy: receivedBy, At: now, Note: strings.TrimSpace(req.Note),
	}
	set := bson.M{"updatedAt": now}
	if inv.Collected()+req.Amount >= inv.Amount {
		set["status"] = models.InvoicePaid
	}
	if _, err := h.Store.Invoices.UpdateByID(r.Context(), id, bson.M{
		"$push": bson.M{"paid": payment},
		"$set":  set,
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	inv.Paid = append(inv.Paid, payment)
	if s, ok := set["status"].(string); ok {
		inv.Status = s
	}
	httpx.JSON(w, http.StatusOK, view(inv))
}

// VoidInvoice cancels a bill that should not have been issued.
//
// The reason is required, exactly as it is for cancelling an order in the
// tenant app: an invoice that disappeared without a sentence is one nobody can
// explain when the customer asks. Voiding never deletes — the row stays, with
// why on it.
func (h *Handler) VoidInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = httpx.Decode(r, &req)
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		httpx.Error(w, http.StatusBadRequest, "bekor qilish sababi kerak")
		return
	}
	res := h.Store.Invoices.FindOneAndUpdate(r.Context(),
		bson.M{"_id": id},
		bson.M{"$set": bson.M{
			"status": models.InvoiceVoid, "voidReason": reason, "updatedAt": time.Now(),
		}},
		options.FindOneAndUpdate().SetReturnDocument(options.After))
	var inv models.Invoice
	if err := res.Decode(&inv); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	httpx.JSON(w, http.StatusOK, view(inv))
}

// sumDays totals the aggregate rows inside a half-open window.
//
// The same rows and the same day keys the dashboard reads, so an invoice can
// never disagree with the number the customer is looking at.
func (h *Handler) sumDays(ctx context.Context, id primitive.ObjectID, from, to string) (
	orders, revenue, billable int, err error,
) {
	cur, err := h.Store.Days.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"tenantId": id,
			"date":     bson.M{"$gte": from, "$lt": to},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":      nil,
			"orders":   bson.M{"$sum": "$orders"},
			"revenue":  bson.M{"$sum": "$revenue"},
			"billable": bson.M{"$sum": "$billable"},
		}}},
	})
	if err != nil {
		return 0, 0, 0, err
	}
	var rows []struct {
		Orders   int `bson:"orders"`
		Revenue  int `bson:"revenue"`
		Billable int `bson:"billable"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return 0, 0, 0, err
	}
	if len(rows) == 0 {
		return 0, 0, 0, nil
	}
	return rows[0].Orders, rows[0].Revenue, rows[0].Billable, nil
}

// previousPeriod is the last window that has finished.
//
// Derived from the tenant's own anchor, so it lines up exactly with what the
// dashboard called "the period before this one" — two different notions of a
// month would put a day's orders on two invoices or on none.
func previousPeriod(t models.Tenant, now time.Time) (from, to string) {
	cur := tenantPeriod(t, now)
	anchor := local(t.CreatedAt)
	if t.SubscribedAt != nil && !t.SubscribedAt.IsZero() {
		anchor = local(*t.SubscribedAt)
	}
	if anchor.IsZero() {
		anchor = now
	}
	start, err := time.ParseInLocation("2006-01-02", cur.From, time.Local)
	if err != nil {
		return cur.From, cur.To
	}
	// One month back from the start of the current window. Using AddMonths on
	// the boundary rather than subtracting 30 days keeps the day-of-month
	// clamp — a customer anchored on the 31st is not walked backwards.
	prevStart := billing.AddMonths(start, -1)
	return billing.Day(prevStart), cur.From
}

// nextInvoiceNumber is what somebody reads out on the phone.
//
// Scoped to the period's month rather than global, so the number says when as
// well as which — "KEEL-2026-08-0007" is a sentence an operator can act on and
// a bare counter is not.
func (h *Handler) nextInvoiceNumber(ctx context.Context, from string) (string, error) {
	month := from
	if len(month) >= 7 {
		month = month[:7]
	}
	prefix := "KEEL-" + month + "-"
	n, err := h.Store.Invoices.CountDocuments(ctx,
		bson.M{"number": bson.M{"$regex": "^" + prefix}})
	if err != nil {
		return "", err
	}
	// A collision is possible if two are issued at once; the unique index
	// catches it and the caller retries, which is cheaper than a counter
	// document that has to be kept correct forever.
	for i := int64(1); i <= 50; i++ {
		candidate := fmt.Sprintf("%s%04d", prefix, n+i)
		count, err := h.Store.Invoices.CountDocuments(ctx, bson.M{"number": candidate})
		if err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("hisob raqamini yasab bo'lmadi")
}

// billingNote explains a number that is not the flat total, on the invoice
// itself.
//
// Three separate reasons an invoice can come in under `orders × price`, and
// the note has to name the right one: without it the amount disagrees with the
// day rows and neither the customer nor an operator a year later can see why.
func billingNote(t models.Tenant, orders, flat, charged int) string {
	if t.FreeAt(time.Now()) {
		n := "bepul xizmat"
		if t.FreeReason != "" {
			n += ": " + t.FreeReason
		}
		return n
	}
	// The floor, and it is checked before the discount because it is the line
	// the customer did not expect. A bill that is *larger* than the orders
	// account for needs a sentence on the invoice itself: the restaurant can
	// count its own orders, and an unexplained gap between their arithmetic and
	// ours is a phone call at best.
	//
	// Detected from the numbers rather than from the configured minimum:
	// nothing else in this pipeline can raise a charge, so `charged > flat` is
	// the floor by construction and cannot drift out of step with it.
	if orders > 0 && charged > flat {
		return fmt.Sprintf("minimal oylik to'lov (%d buyurtma bo'yicha %s so'm)",
			orders, thousands(flat))
	}
	if t.DiscountPercent > 0 {
		return fmt.Sprintf("chegirma %d%% (to'liq summa %s)",
			t.DiscountPercent, thousands(flat))
	}
	// The volume ladder. Stated as the average, because that is the number a
	// restaurant repeats to itself — "we are paying 742 a order now".
	if orders > 0 && charged < flat {
		return fmt.Sprintf("pog'onali narx: %d buyurtma, o'rtacha %s so'm/buyurtma",
			orders, thousands(charged/orders))
	}
	return ""
}

// thousands groups so'm the way every screen in this system does.
func thousands(n int) string {
	s := fmt.Sprintf("%d", n)
	out := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += " "
		}
		out += string(c)
	}
	return out
}

func currentUser(r *http.Request) string {
	if c := middleware.From(r.Context()); c != nil {
		return c.Username
	}
	return ""
}
