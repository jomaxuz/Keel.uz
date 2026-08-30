package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/fiscal"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/instore"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Filing a till sale with the tax committee, through the one machine that can
// reach the cash register.
//
// # Why the browser makes the call
//
// The registered virtual cash register runs on a PC inside the restaurant, on
// an address like 192.168.14.65 with no authentication — see internal/fiscal.
// Our container cannot route to it. The cashier's tablet can: it is already on
// that network, because it is in that building.
//
// So the work is split at the only place it can be, and deliberately not one
// step further:
//
//	server  → builds the receipt from the order and serialises the request
//	browser → POSTs it to the register, verbatim, and posts the reply back
//	server  → parses the reply and records the filing
//
// ⚠️ **The browser never computes money and never decides anything.** It
// receives an opaque body and a URL and carries them across one network hop.
// Every alternative that felt simpler — letting the till assemble the basket,
// or trusting it to say "filed" — hands a tax document to the least trusted
// participant in the system. The pricing pipeline's rule, applied to a document
// with a heavier consequence than a wrong total.
//
// ⚠️ A till *can* still lie about the outcome, and that is accepted: it can only
// lie about its own restaurant's filings, which is a restaurant defrauding
// itself in a way the tax committee's own records contradict. The line worth
// defending is the contents of the receipt, and that is on this side.

// tillFiscalJob is what the till screen is asked to do.
type tillFiscalJob struct {
	// The full address to call — the branch's stored register address with the
	// adapter's path appended. Assembled here so no address handling lives in
	// the browser, which would otherwise have to know each provider's paths.
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
	// How long the till should wait before giving up. A cash register on a local
	// network answers in well under a second when it answers at all; a guest at
	// the counter should not watch a spinner for thirty.
	TimeoutMS int `json:"timeoutMs"`
}

// tillFiscalStatus is what the till screen shows about fiscalisation, and what
// it needs to decide whether to file at all.
type tillFiscalStatus struct {
	// Whether receipts are to be filed here. False is the ordinary state for a
	// restaurant that has not connected a register, and the till works normally
	// without one — it simply does not claim to have filed anything.
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider,omitempty"`
	Name     string `json:"name,omitempty"`
	// Whether the register is on the restaurant's network. The till screen shows
	// a different failure hint for each: a local register that does not answer
	// is a cable or an address, and neither is something we can check from here.
	Local bool `json:"local"`
	// Whether a relay on the register's PC is currently taking the filings. When
	// it is, this screen waits for a result instead of making the call — see
	// StaffFileReceipt.
	Relay bool `json:"relay"`
	// The connection check, as a job the till can run whenever the cashier asks.
	// Present only when a local provider is configured **and** no relay is
	// handling things: with a relay there is nothing for this browser to reach,
	// and a check that failed from here would say nothing about whether receipts
	// are being filed.
	Hello *tillFiscalJob `json:"hello,omitempty"`

	// ⚠️ **A timestamp, not a flag** — the most useful line on the settings page
	// and the most useful line here. "Connected" goes stale the moment the hour
	// moves past it; "last receipt filed at 19:42" cannot.
	LastReceiptAt *time.Time `json:"lastReceiptAt,omitempty"`
	LastErrorAt   *time.Time `json:"lastErrorAt,omitempty"`
	LastError     string     `json:"lastError,omitempty"`
}

// tillFiscal loads a branch's fiscal setup as the till needs it, along with the
// encoder when there is a usable one.
//
// A disabled or unconfigured branch is not an error: most restaurants will run
// the till without a register connected for as long as it takes to sign a
// contract, and every caller here has to handle that case anyway.
func (h *Handler) tillFiscal(
	ctx context.Context, branchID primitive.ObjectID,
) (*models.FiscalSettings, fiscal.Encoder, string) {
	s := h.fiscalSettingsOf(ctx, branchID)
	if !s.Enabled || s.Provider == "" {
		return s, nil, ""
	}
	enc, err := fiscal.EncoderFor(s.Provider, credsOf(s))
	if err != nil || enc == nil {
		return s, nil, ""
	}
	base := strings.TrimRight(strings.TrimSpace(credsOf(s).BaseURL), "/")
	return s, enc, base
}

func (j tillFiscalJob) withBase(base string, r fiscal.Request) tillFiscalJob {
	j.URL = base + r.Path
	j.Method = r.Method
	j.Headers = r.Headers
	j.Body = r.Body
	j.TimeoutMS = 15000
	return j
}

// StaffFiscalStatus tells the till screen whether and how to file.
func (h *Handler) StaffFiscalStatus(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	set, enc, base := h.tillFiscal(r.Context(), s.BranchID)
	out := tillFiscalStatus{
		Enabled:       set.Enabled && enc != nil && base != "",
		Provider:      set.Provider,
		Name:          fiscal.Name(set.Provider),
		Local:         fiscal.IsLocal(set.Provider),
		Relay:         agentUsable(set),
		LastReceiptAt: set.LastReceiptAt,
		LastErrorAt:   set.LastErrorAt,
		LastError:     set.LastError,
	}
	if out.Enabled && !out.Relay {
		if req, err := enc.Hello(); err == nil {
			job := tillFiscalJob{}.withBase(base, req)
			out.Hello = &job
		}
	}
	// With a relay running, filing does not depend on this browser reaching
	// anything — so the screen must not be switched off just because the
	// register's address is blank here. The relay defaults to its own machine.
	if !out.Enabled && out.Relay && enc != nil {
		out.Enabled = true
	}
	if set.Provider == "" {
		out.Name = ""
	}
	httpx.JSON(w, http.StatusOK, out)
}

// StaffUnfiledChecks lists the sales that took money and have no receipt.
//
// ⚠️ **This is the screen the whole feature needs to be honest.** Everything
// else records a filing at the moment it happens, which works right up until it
// does not — and a failed filing is invisible by nature: the guest has eaten
// and gone, the money is in the drawer, and nothing on any screen looks wrong.
// A retry button promised in a comment and reachable from nowhere is not a
// retry button.
//
// On the till rather than only in the panel because the person who can fix most
// of these — open the shift, switch the register's PC back on — is standing
// next to it.
func (h *Handler) StaffUnfiledChecks(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermCashier)
	if !ok {
		return
	}
	now := time.Now()
	filter := unfiledFiscalFilter(now)
	filter["branchId"] = s.BranchID

	cur, err := h.Store.Orders.Find(r.Context(), filter,
		// Oldest first, as the kitchen screen orders its tickets: the receipt
		// that has waited longest is the one closest to being a real problem.
		options.Find().SetSort(bson.M{"check.closedAt": 1}).SetLimit(50))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var orders []models.Order
	if err := cur.All(r.Context(), &orders); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]checkView, 0, len(orders))
	for i := range orders {
		out = append(out, viewCheck(&orders[i], now, s.ID))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"checks": out})
}

type fiscalReplyRequest struct {
	// What the till saw. Both are needed: this register reports business
	// refusals with a 500 and a readable body, so neither half is the answer on
	// its own.
	Status int    `json:"status"`
	Body   string `json:"body"`
	// Set when the call never completed at all — a timeout, a refused
	// connection, a browser that blocked it. Kept distinct from a rejection
	// because the two send a cashier to entirely different places, and "the
	// register said no" is the only one of them that is about the receipt.
	NetworkError string `json:"networkError"`
}

// StaffFiscalCheck records the outcome of a connection check run by the till.
//
// The check lives here rather than on the settings page because the settings
// page is frequently open on a laptop at home, where the register is
// unreachable for a reason that says nothing about the restaurant.
func (h *Handler) StaffFiscalCheck(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermCashier)
	if !ok {
		return
	}
	var req fiscalReplyRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set, enc, _ := h.tillFiscal(r.Context(), s.BranchID)
	if enc == nil {
		httpx.Error(w, http.StatusBadRequest, "fiskal kassa ulanmagan")
		return
	}

	msg, err := describeReply(set.Provider, enc, req)
	now := time.Now()
	update := bson.M{"lastCheckAt": now, "lastCheckOk": err == nil}
	if err != nil {
		update["lastCheck"] = err.Error()
	} else {
		update["lastCheck"] = msg
	}
	_, _ = h.Store.FiscalSettings.UpdateOne(r.Context(),
		bson.M{"branchId": set.BranchID}, bson.M{"$set": update},
		options.Update().SetUpsert(true))

	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": false, "message": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "message": msg})
}

// describeReply turns a connection check's raw reply into one readable line.
func describeReply(provider string, enc fiscal.Encoder, req fiscalReplyRequest) (string, error) {
	if err := replyFailure(req); err != nil {
		return "", err
	}
	if _, err := enc.Parse(req.Status, []byte(req.Body)); err != nil {
		return "", err
	}
	if msg := fiscal.DescribeFor(provider, []byte(req.Body)); msg != "" {
		return msg, nil
	}
	// Reached something that answered correctly but did not name itself. Better
	// than a failure and worse than an identification, and said as such rather
	// than dressed up as "connected".
	return "kassa javob berdi", nil
}

// replyFailure reports a call that never reached the register.
//
// ⚠️ Its own case, and the message names the till rather than the register: a
// cashier told "the cash register refused" will go and restart the cash
// register, which was never running the software that failed.
func replyFailure(req fiscalReplyRequest) error {
	if e := strings.TrimSpace(req.NetworkError); e != "" {
		return errors.New("kassa dasturiga ulanib bo'lmadi — manzilni va planshet " +
			"kassa bilan bir tarmoqda ekanini tekshiring (" + clampText(e, 120) + ")")
	}
	if strings.TrimSpace(req.Body) == "" && req.Status == 0 {
		return errors.New("kassa dasturi javob bermadi")
	}
	return nil
}

// StaffFileReceipt builds the filing for a closed check.
//
// ⚠️ **Called after the money is taken, never before.** Filing first would
// register a sale that a card decline can still undo, and an over-filed receipt
// is corrected by a refund document rather than by deleting anything — the
// expensive direction of a mistake. The reverse gap, a paid check not yet
// filed, is visible, retryable and exactly what fiscal.status "pending" is for.
func (h *Handler) StaffFileReceipt(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermCashier)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok {
		return
	}
	if o.Check == nil || o.Check.ClosedAt == nil || o.PaymentStatus != models.PayPaid {
		httpx.Error(w, http.StatusBadRequest, "chek hali yopilmagan")
		return
	}
	// ⚠️ **A refunded check has two documents on it**, and once the sale is
	// filed the work left on this button is the reversal. Without this the
	// button below would answer "already filed" for the rest of the check's
	// life and a reversal queued at a restaurant with no relay would never be
	// filed by anybody — the sale would stand with the tax committee while the
	// money sat back in the guest's hand.
	isRefund := pendingReversal(o)

	if !isRefund && o.Fiscal != nil && o.Fiscal.Status == models.FiscalFiled {
		// Already filed. Not an error — the till retries after a lost reply, and
		// this is the case that keeps a retry from filing a second document for
		// one sale. Same guard as the POS bridge's, for a sharper reason: a
		// duplicate fiscal receipt is a sale the restaurant is taxed on twice.
		httpx.JSON(w, http.StatusOK, map[string]any{
			"filed": true, "fiscal": o.Fiscal,
		})
		return
	}

	set, enc, base := h.tillFiscal(r.Context(), s.BranchID)
	if enc == nil || base == "" {
		// No register connected. The sale stands; we simply do not pretend it
		// was registered. The till shows nothing rather than a broken badge.
		httpx.JSON(w, http.StatusOK, map[string]any{"skip": true})
		return
	}

	codes := h.menuFiscal(r.Context(), o)
	sale := receiptFor(o, set, s.Name, codes)
	if isRefund {
		sale = refundFor(o, set, s.Name, codes)
	}
	req, err := enc.Sale(fiscal.Build(sale))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Marked pending before the till is handed the job, so a sale whose reply
	// never comes back is a visible unfinished filing rather than an order that
	// looks like it was never meant to have one.
	now := time.Now()
	prev, field := o.Fiscal, "fiscal"
	if isRefund {
		prev, field = o.FiscalRefund, "fiscalRefund"
	}
	pending := models.FiscalReceipt{
		Status:   models.FiscalPending,
		Provider: set.Provider,
		Attempts: attemptsOf(prev) + 1,
	}
	if _, err := h.Store.Orders.UpdateOne(r.Context(),
		checkFilter(o.ID, s.BranchID),
		bson.M{"$set": bson.M{field: pending, "updatedAt": now}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// ⚠️ **If a relay is listening, the filing is left for it.** The order is
	// already marked pending, which is the whole queue — the agent finds it on
	// its next request for work. The till is told to wait rather than being
	// handed the job as well: two callers filing one sale is a receipt
	// registered twice, and a duplicate fiscal document is corrected by paperwork
	// rather than by us.
	if agentUsable(set) {
		httpx.JSON(w, http.StatusOK, map[string]any{"queued": true})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"job": tillFiscalJob{}.withBase(base, req),
	})
}

// StaffFileReceiptResult records what the register said.
func (h *Handler) StaffFileReceiptResult(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermCashier)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok {
		return
	}
	var req fiscalReplyRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set, enc, _ := h.tillFiscal(r.Context(), s.BranchID)
	if enc == nil {
		httpx.Error(w, http.StatusBadRequest, "fiskal kassa ulanmagan")
		return
	}

	// ⚠️ **"The shift is not open" is answered, not reported.** It is the one
	// refusal that happens every single morning — the first sale of the day
	// hits a register whose day has not been started — and it is entirely
	// mechanical to fix. Handing it to the cashier as an error would make
	// opening the shift a thing they must learn, remember, and do while
	// somebody waits at the counter.
	//
	// The shift is opened here rather than pre-emptively because the register
	// itself is the only reliable source of whether one is needed. Asking in
	// advance costs a round trip per receipt; opening one speculatively files a
	// document with the tax committee for a day that was already open.
	if job, ok := h.shiftJobFor(set, enc, req, s.Name); ok {
		httpx.JSON(w, http.StatusOK, map[string]any{"openShift": job})
		return
	}

	// Which document this answer belongs to is read from the order rather than
	// sent by the till — same rule as the relay's reply, and for the same
	// reason: the reply carries an order id, and the order knows which of its
	// two filings is in flight.
	// recordFilingInto writes the record onto `o` itself, so the view below is
	// already the updated check.
	_, err := h.recordFilingInto(r.Context(), o, set, enc, req, pendingReversal(o))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, viewCheck(o, time.Now(), s.ID))
}

// shiftJobFor returns the "open the day" call when that is what the register
// just complained about.
//
// ⚠️ **The filing is deliberately not recorded as failed in this case.** The
// sale is still pending, which is the truth: nothing was refused about the
// receipt itself, only about the register's state. Writing a failure here would
// put every restaurant's first sale of the day into the unfiled alert every
// morning, and an alert that is always red on opening is an alert nobody reads
// by the end of the week.
func (h *Handler) shiftJobFor(
	set *models.FiscalSettings, enc fiscal.Encoder,
	reply fiscalReplyRequest, cashier string,
) (tillFiscalJob, bool) {
	if replyFailure(reply) != nil {
		return tillFiscalJob{}, false
	}
	_, err := enc.Parse(reply.Status, []byte(reply.Body))
	if !fiscal.NeedsShift(set.Provider, err) {
		return tillFiscalJob{}, false
	}
	opener, ok := enc.(fiscal.ShiftOpener)
	if !ok {
		return tillFiscalJob{}, false
	}
	req, oerr := opener.OpenShift(cashier, time.Now())
	if oerr != nil {
		return tillFiscalJob{}, false
	}
	base := strings.TrimRight(strings.TrimSpace(credsOf(set).BaseURL), "/")
	if base == "" {
		base = "http://localhost:8080"
	}
	return tillFiscalJob{}.withBase(base, req), true
}

// recordFiling writes down what the register said.
//
// ⚠️ **Shared by both transports** — the till carrying the call itself, and the
// relay on the register's PC doing it — because they differ only in who held
// the socket. Two copies would drift on the thing that matters most here: which
// outcomes count as filed. A path that recorded a filing more loosely than the
// other would produce receipts the tax committee has no record of, and the
// discrepancy would surface as an inspection rather than as a bug.
//
// The error return is for storage failures only. A register that refused is not
// an error here — it is a recorded outcome, which is the entire point.
func (h *Handler) recordFiling(
	ctx context.Context,
	o *models.Order,
	set *models.FiscalSettings,
	enc fiscal.Encoder,
	reply fiscalReplyRequest,
) (*models.FiscalReceipt, error) {
	return h.recordFilingInto(ctx, o, set, enc, reply, false)
}

// recordFilingInto writes the answer to whichever of the two filings it belongs.
//
// ⚠️ **Which field is decided by the caller, not guessed from the order.** A
// refunded sale has both records, and reading the state to work out which one a
// reply is about would get it wrong on exactly the retry that matters — a
// reversal that failed once and is being tried again, against a sale that was
// filed months ago.
func (h *Handler) recordFilingInto(
	ctx context.Context,
	o *models.Order,
	set *models.FiscalSettings,
	enc fiscal.Encoder,
	reply fiscalReplyRequest,
	isRefund bool,
) (*models.FiscalReceipt, error) {
	now := time.Now()
	prev := o.Fiscal
	field := "fiscal"
	if isRefund {
		prev = o.FiscalRefund
		field = "fiscalRefund"
	}
	rec := models.FiscalReceipt{
		Provider: set.Provider,
		Attempts: attemptsOf(prev),
	}

	err := replyFailure(reply)
	var res fiscal.Result
	if err == nil {
		res, err = enc.Parse(reply.Status, []byte(reply.Body))
	}

	settings := bson.M{}
	if err != nil {
		rec.Status = models.FiscalFailed
		rec.Error = clampText(err.Error(), 300)
		// ⚠️ Recorded on the branch as well, because this is the question nobody
		// watches: the owner is not standing at the till, and a register that
		// stopped filing this morning is otherwise discovered by an inspector.
		settings["lastErrorAt"] = now
		settings["lastError"] = rec.Error
	} else {
		rec.Status = models.FiscalFiled
		rec.FiscalSign = res.FiscalSign
		rec.QRText = res.QRText
		rec.ReceiptID = res.ReceiptID
		rec.FiledAt = &now
		settings["lastReceiptAt"] = now
		// A success clears the standing failure: a page showing both a working
		// register and an old error at once answers neither question, and the
		// stale half is the frightening one.
		settings["lastErrorAt"] = nil
		settings["lastError"] = ""
	}

	if _, uerr := h.Store.Orders.UpdateOne(ctx,
		checkFilter(o.ID, o.BranchID),
		bson.M{"$set": bson.M{field: rec, "updatedAt": now}}); uerr != nil {
		return nil, uerr
	}
	_, _ = h.Store.FiscalSettings.UpdateOne(ctx,
		bson.M{"branchId": set.BranchID}, bson.M{"$set": settings},
		options.Update().SetUpsert(true))

	// ⚠️ **The paper comes out here, not at the close**, because this is where
	// the fiscal sign first exists — and the sign is the only part of a receipt
	// the guest can check. Queued whatever the register said: a refusal is the
	// restaurant's problem to fix (the unfiled alert names it), and the person
	// waiting at the counter is owed their bill either way.
	if isRefund {
		// ⚠️ **A reversal does not reprint the guest's bill.** The paper below
		// is the sale's receipt, and printing it again with the reversal's sign
		// on it would hand somebody a document that looks like a second sale.
		// What a refund is owed on paper is the register's own return slip,
		// which it prints itself when it accepts the operation.
		o.FiscalRefund = &rec
		return &rec, nil
	}
	o.Fiscal = &rec
	h.queueSaleReceipts(ctx, o)
	h.sendCounterFiscalLink(ctx, o, rec)
	return &rec, nil
}

// sendCounterFiscalLink hands the bank the receipt the guest can open.
//
// Uzum FastPay shows the fiscal receipt inside its own app, next to the payment
// — but only if we tell it where the receipt is. ⚠️ **Which cannot be done at
// the moment of payment**, and that is why this call lives here rather than in
// tillscan.go: at payment time the receipt does not exist yet. It comes into
// existence when the register answers, which is this function's caller, and
// which is minutes later over a different network.
//
// ⚠️ **It can fail, and nothing about the sale changes if it does.** The money
// is taken, the receipt is filed with the state, the paper is printed, and the
// guest is owed nothing further — what is lost is a convenience link inside an
// app. So it is fire-and-forget with a timestamp when it worked, and the one
// thing it must never do is look like a failure of the filing it follows.
func (h *Handler) sendCounterFiscalLink(
	ctx context.Context, o *models.Order, rec models.FiscalReceipt,
) {
	pay := o.CounterPay
	if pay == nil || pay.Status != instore.StatusPaid || pay.PaymentID == "" {
		return
	}
	if pay.FiscalSentAt != nil || rec.Status != models.FiscalFiled {
		return
	}
	// The QR's text is the link the guest scans off the paper — the same
	// address, which is exactly what the bank wants to show them.
	link := strings.TrimSpace(rec.QRText)
	if link == "" {
		return
	}
	charger, err := h.counterCharger(ctx, pay.Provider, o.BranchID)
	if err != nil {
		return
	}
	if err := charger.Fiscal(ctx, pay.PaymentID, link); err != nil {
		log.Printf("counter pay: fiscal link not delivered for %s: %v", o.Number, err)
		return
	}
	now := time.Now()
	_, _ = h.Store.Orders.UpdateOne(ctx, bson.M{"_id": o.ID},
		bson.M{"$set": bson.M{"counterPay.fiscalSentAt": now}})
}

// fiscalStuckAfter is how long a sale may sit unregistered before it is worth
// naming on the dashboard.
//
// Long enough that the ordinary path — close the check, file, show the QR —
// never trips it, and short enough that a cashier is still in the building.
// A register that refused outright does not wait for this: that is already a
// finished answer, and the count below reports it immediately.
const fiscalStuckAfter = 5 * time.Minute

// unfiledFiscalFilter matches sales that took money and have no tax receipt.
//
// ⚠️ **`$in` on the statuses, not `$nin` on "filed"** — the exact opposite of
// pendingTillFilter next door, and for the same underlying Mongo behaviour read
// from the other side. A missing field *matches* `$nin`, so "not filed" would
// sweep in every till sale ever rung up before a register was connected, and
// every sale at a restaurant that has none. Those are not unfiled receipts;
// they are sales that never owed one. What we want is the sales we know owe a
// receipt, and those all carry the flag because StaffCloseCheck writes it when
// the money is taken.
//
// ⚠️ **No upper bound on age, deliberately.** A receipt that failed an hour ago
// and is still unfiled deserves more attention than a fresh one, not less —
// the same reasoning as the POS failure banner. It leaves this filter by being
// filed, which is the only thing that actually resolves it.
func unfiledFiscalFilter(now time.Time) bson.M {
	return bson.M{
		"check": bson.M{"$exists": true},
		"$or": []bson.M{
			// Refused or unreachable: a finished answer, reported at once.
			{"fiscal.status": models.FiscalFailed},
			// Still waiting, and long enough that nobody is mid-transaction.
			{
				"fiscal.status":  models.FiscalPending,
				"check.closedAt": bson.M{"$lt": now.Add(-fiscalStuckAfter)},
			},
			// ⚠️ **A reversal counts the same, and its clock starts at the
			// refund**, not at the close: the check it undoes may have been
			// paid days ago, and dating the grace period from that close would
			// make every reversal overdue the moment it is queued.
			{"fiscalRefund.status": models.FiscalFailed},
			{
				"fiscalRefund.status": models.FiscalPending,
				"refund.at":           bson.M{"$lt": now.Add(-fiscalStuckAfter)},
			},
		},
	}
}

// anyUnfiledFilter matches every till sale that still owes a receipt, with no
// grace period at all.
//
// ⚠️ **Deliberately stricter than unfiledFiscalFilter, and the difference is the
// point.** That one waits five minutes before calling a pending filing a
// problem, because it drives an alert and a sale filed thirty seconds ago is not
// news. This one guards the Z-report, where the same five minutes is a hole: a
// receipt still in flight when the day is totalled either vanishes from the
// day's figures or lands on tomorrow's, and neither can be corrected afterwards.
//
// Reusing the alert's filter here read as obviously right and was quietly wrong
// for exactly the sales most likely to exist at closing time — the last ones.
func anyUnfiledFilter() bson.M {
	unfinished := bson.M{"$in": []string{models.FiscalPending, models.FiscalFailed}}
	return bson.M{
		"check": bson.M{"$exists": true},
		"$or": []bson.M{
			{"fiscal.status": unfinished},
			// A reversal missing from the day's total is the same hole read from
			// the other side: the sale stays in the figures and the money that
			// went back out of the drawer does not.
			{"fiscalRefund.status": unfinished},
		},
	}
}

func attemptsOf(f *models.FiscalReceipt) int {
	if f == nil {
		return 0
	}
	return f.Attempts
}

// receiptFor turns a closed check into the sale to file.
//
// ⚠️ **The classifier codes are not read from the order's stored lines — they
// are read from the menu, now** (menuFiscal, and the note on MenuItem.Ikpu).
// The dish's name and price are what the guest agreed to and are frozen onto
// the order for that reason; ИКПУ, packaging and VAT rate are facts about the
// **product**, so an accountant correcting a wrong code has to affect the sale
// being rung up this minute rather than only dishes added afterwards.
// refundFor is the same document, reversed.
//
// ⚠️ **The same lines, at the same prices, and the original's identity.** A
// reversal is not a new sale with a minus in front of it: the register needs to
// find the document being undone, which is what `Original` carries. Built from
// `o.Fiscal` — the register's own words about a receipt it already issued —
// and never recomputed, because nothing here could derive a fiscal sign.
func refundFor(
	o *models.Order, s *models.FiscalSettings, cashier string,
	codes map[primitive.ObjectID]menuFiscalInfo,
) fiscal.Sale {
	sale := receiptFor(o, s, cashier, codes)
	sale.IsRefund = true
	if o.Fiscal != nil {
		sale.Original = fiscal.OriginalReceipt{
			Sign:       o.Fiscal.FiscalSign,
			SaleID:     o.Fiscal.ReceiptID,
			Seq:        o.Fiscal.ReceiptID,
			TerminalID: fiscalRegisterID(s),
		}
		if o.Fiscal.FiledAt != nil {
			sale.Original.At = *o.Fiscal.FiledAt
		}
	}
	// ⚠️ The refund's own moment, not the sale's. The sale's timestamp belongs
	// to `Original`; using it here would file a reversal dated to the day the
	// meal was eaten.
	sale.Time = time.Now()
	if o.Refund != nil {
		sale.Time = o.Refund.At
	}
	return sale
}

// fiscalRegisterID is the terminal the branch files through, where it is known.
func fiscalRegisterID(s *models.FiscalSettings) string {
	return strings.TrimSpace(credsOf(s).RegisterID)
}

func receiptFor(
	o *models.Order, s *models.FiscalSettings, cashier string,
	codes map[primitive.ObjectID]menuFiscalInfo,
) fiscal.Sale {
	sale := fiscal.Sale{
		OrderNumber: o.Number,
		TIN:         s.TIN,
		Cashier:     cashier,
		Time:        time.Now(),
		Discount:    o.DiscountTotal,
		VatPercent:  vatRateOf(s),
	}
	if o.PaidAt != nil {
		sale.Time = *o.PaidAt
	}
	for _, it := range o.LiveItems() {
		code := codes[it.MenuItemID]
		sale.Lines = append(sale.Lines, fiscal.Line{
			Name:        it.Name,
			Price:       it.Price,
			Qty:         it.Qty,
			SPIC:        code.Ikpu,
			PackageCode: code.PackageCode,
			Units:       code.UnitCode,
			VatPercent:  code.VatPercent,
			// ⚠️ **This one is read from the line and not from the menu**,
			// which is the opposite of every field above it. Those describe a
			// product and are corrected by an accountant; this describes the
			// bottle that was handed over, and there is nothing on the menu
			// that could be corrected without withdrawing an item nobody sold.
			MarkCode: it.MarkCode,
		})
	}

	// ⚠️ Cash and card are taken from the payment method rather than split,
	// because the till takes one payment per check today. When it learns to
	// split a bill, this is the line that has to learn with it — and Build's
	// third rule catches a wrong split rather than filing one.
	if o.PaymentMethod == models.ProviderCash {
		sale.Cash = o.Total
	} else {
		sale.Card = o.Total
	}
	return sale
}

// vatRateOf reads the branch's declared rate.
//
// nil cannot happen on an enabled register — fiscalEnableRefusal will not let
// it be switched on without one — but a nil here would silently file every line
// as exempt, so it is answered explicitly rather than by dereference.
func vatRateOf(s *models.FiscalSettings) int {
	if s.VatPercent == nil {
		return 0
	}
	return *s.VatPercent
}

// menuFiscalInfo is everything a fiscal receipt needs about a dish that is not
// already frozen onto the order.
type menuFiscalInfo struct {
	Ikpu        string `bson:"ikpu"`
	PackageCode string `bson:"packageCode"`
	UnitCode    int    `bson:"unitCode"`
	// nil when the dish does not override the branch's rate, which is almost
	// every dish. A pointer for the reason MenuItem.VatPercent is one: zero is a
	// real answer and "unset" is a different one.
	VatPercent *int `bson:"vatPercent"`
}

// menuFiscal loads the fiscal facts for an order's dishes.
//
// ⚠️ **One lookup for both paths that file receipts** — ATMOS's invoice and the
// till's own filing. They ask the same question about the same documents, and a
// second copy would drift in the direction that matters: an accountant fixing a
// code in the menu, seeing the online receipt corrected, and the counter's
// receipt still carrying the old one.
//
// A failure is not fatal. A receipt missing its codes is worse than one with
// them and far better than a guest who cannot pay at all.
func (h *Handler) menuFiscal(
	ctx context.Context, order *models.Order,
) map[primitive.ObjectID]menuFiscalInfo {
	out := map[primitive.ObjectID]menuFiscalInfo{}
	ids := make([]primitive.ObjectID, 0, len(order.Items))
	for _, it := range order.Items {
		if !it.MenuItemID.IsZero() {
			ids = append(ids, it.MenuItemID)
		}
	}
	if len(ids) == 0 {
		return out
	}
	cur, err := h.Store.Menu.Find(ctx,
		bson.M{"_id": bson.M{"$in": ids}},
		options.Find().SetProjection(bson.M{
			"ikpu": 1, "packageCode": 1, "unitCode": 1, "vatPercent": 1,
		}))
	if err != nil {
		log.Printf("fiscal: menyu kodlarini o'qib bo'lmadi: %v", err)
		return out
	}
	var rows []struct {
		ID             primitive.ObjectID `bson:"_id"`
		menuFiscalInfo `bson:",inline"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		log.Printf("fiscal: menyu kodlarini o'qib bo'lmadi: %v", err)
		return out
	}
	for _, r := range rows {
		out[r.ID] = r.menuFiscalInfo
	}
	return out
}

// pendingReversal says whether the work outstanding on this check is the
// reversal rather than the sale.
//
// ⚠️ **The sale must already be filed.** A reversal names its sale by that
// sale's fiscal sign; while the sale itself is still pending there is no sign
// to name, so the sale is always the job in front.
func pendingReversal(o *models.Order) bool {
	return o.Fiscal != nil && o.Fiscal.Status == models.FiscalFiled &&
		o.Refund != nil &&
		o.FiscalRefund != nil && o.FiscalRefund.Status != models.FiscalFiled
}
