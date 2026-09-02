package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/instore"
	"restaurant-backend/internal/models"
)

// ---- Paying at the counter with a phone ----
//
// ⚠️ **The guest pays from their own phone, and the till watches.** Payme,
// Click and Uzum all end in the same place: a page on the payer's device asking
// them to confirm. There is nothing a cashier can drive there — no card to
// swipe, no amount to type — so what the till can usefully do is put the link
// on the screen as a QR code and wait for the provider to tell the server the
// money arrived.
//
// ⚠️ **The check does not close until it is paid**, and that is the whole
// design. The obvious shortcut is to mark the sale paid when the cashier picks
// the provider and let the callback catch up; it works every time it is tested,
// because in a test the payment succeeds. In a queue it hands out food for a
// payment that was cancelled, expired, or made on somebody else's screen. The
// bank's word is the only evidence — the same rule the website has always
// followed: "a browser proves nothing".
//
// ⚠️ **The terminal is not one of these.** A bank terminal on the counter has
// its own receipt, its own settlement and its own money that never passes
// through this system; `card` records that it was used. Pretending we confirm
// it would put a green tick on the one payment we cannot see. No published
// protocol exists in Uzbekistan for driving one from a till — see
// internal/instore/terminal.go for who was asked and what they said.
//
// ⚠️ **But the retyping it caused is fixed elsewhere, and in the opposite
// direction.** handlers/tillscan.go charges a card by scanning a code on the
// guest's phone: the amount comes off the check, the answer arrives in the same
// request, and nobody types a total into a second machine. Two files, because
// they are two flows — here the guest scans us and we wait; there we scan the
// guest and the bank answers.

// tillOnlineMethods are the providers a till may ask a guest to pay with.
//
// ATMOS is absent on purpose: its link is minted by a request that can fail for
// network reasons, and a QR that sometimes does not appear is worse at a
// counter than one that is not offered — the cashier is left explaining an
// empty screen to somebody holding a phone.
var tillOnlineMethods = map[string]bool{
	models.ProviderPayme: true,
	models.ProviderClick: true,
	models.ProviderUzum:  true,
}

// TillPaymentMethods is what this till can actually take today.
//
// ⚠️ **Asked of the server rather than hard-coded on the screen**, for the same
// reason the website asks: a button leading to a bank page that rejects the
// merchant loses the sale and the restaurant gets the blame. Cash, card,
// transfer and the slate need no configuring and are always there.
func (h *Handler) TillPaymentMethods(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.tillStaff(w, r, models.PermWaiter); !ok {
		return
	}
	s := h.paymentSettings(r.Context())
	methods := []string{models.ProviderCash, "card", "transfer"}
	for _, p := range []string{
		models.ProviderPayme, models.ProviderClick, models.ProviderUzum,
	} {
		if s.Configured(p) {
			methods = append(methods, p)
		}
	}
	// The counter rails, before the slate. ⚠️ Listed separately from the three
	// above rather than folded into the same loop: those are offered when the
	// *website's* credentials are configured, and these have their own — a
	// restaurant can perfectly well take Click on its site and not at the
	// counter, or the reverse.
	for _, p := range []string{instore.ClickPass, instore.UzumFastPay} {
		if _, on := s.InStoreCreds(p); on && instore.Ready(p) {
			methods = append(methods, p)
		}
	}
	methods = append(methods, models.MethodDebt)
	httpx.JSON(w, http.StatusOK, map[string]any{"methods": methods})
}

type tillPayRequest struct {
	Provider string `json:"provider"`
}

// TillStartPayment puts a check in front of a provider and hands back the link.
//
// ⚠️ **The amount is frozen here.** The link carries a sum, and a dish added
// after the QR went up would leave the guest paying the old total against a
// bigger bill — so the check's totals are written now and the line is what the
// bank is asked for. Adding a dish afterwards is not blocked (the kitchen is
// the guest's business, not the payment rail's); the cashier simply asks for a
// new code, and the old one is replaced.
func (h *Handler) TillStartPayment(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermCashier)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok || !requireOpen(w, o) {
		return
	}
	var req tillPayRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if !tillOnlineMethods[req.Provider] {
		httpx.Error(w, http.StatusBadRequest, "noma'lum to'lov tizimi")
		return
	}
	if !h.paymentSettings(r.Context()).Configured(req.Provider) {
		// ⚠️ Refused rather than shown as a dead QR: a code that leads to the
		// provider's own error page is indistinguishable, to everyone standing
		// there, from a restaurant whose till is broken.
		httpx.Error(w, http.StatusServiceUnavailable,
			"bu to'lov tizimi hali sozlanmagan")
		return
	}
	if len(o.LiveItems()) == 0 {
		httpx.Error(w, http.StatusBadRequest, "chek bo'sh")
		return
	}

	now := time.Now()
	set := bson.M{"updatedAt": now, "paymentMethod": req.Provider}
	// ⚠️ `pending`, never `paid`: this records that we asked, not that anybody
	// answered. `received()` reads it as money not yet in hand, which is
	// exactly what it is while the guest is still finding their phone.
	set["paymentStatus"] = models.PayPending
	applyCheckTotals(o, set)
	if _, err := h.Store.Orders.UpdateOne(r.Context(),
		bson.M{"_id": o.ID}, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	o.PaymentMethod = req.Provider

	url := h.payURL(r.Context(), o, "")
	if url == "" {
		httpx.Error(w, http.StatusServiceUnavailable,
			"to'lov havolasi olinmadi")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"url":      url,
		"provider": req.Provider,
		"total":    o.Total,
		// The number is on the QR's destination and on the paper. ⚠️ Worth
		// returning: when the automatic path fails — a guest who paid on a
		// phone with no signal, a callback the provider retries in ten
		// minutes — somebody finds the payment in the provider's cabinet by
		// this number and nothing else.
		"number": o.Number,
	})
}

// TillPaymentStatus is the till asking whether the money has arrived.
//
// ⚠️ **Polled, not pushed.** A socket for one screen that is waiting for at
// most a minute would be a second transport to keep alive across the network
// outages this till is built to survive — and the panel already answers every
// question this way (see AlertBell). The screen stops asking when the check
// closes or the cashier gives up, which is the only two ways this ends.
func (h *Handler) TillPaymentStatus(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok {
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"status": paymentStatusOf(o),
		"method": o.PaymentMethod,
		"paid":   paymentStatusOf(o) == models.PayPaid,
	})
}

// ---- A debt paid back at the counter ----
//
// ⚠️ **The person who takes the money is the person with the drawer.** Settling
// from the panel works and is kept (an owner reconciling a card by phone), but
// the ordinary case is a regular walking in on Friday with cash for Tuesday —
// and sending the cashier to find somebody with a panel login, in front of that
// guest, is how a manager's password ends up written by the till.
//
// The money lands in **today's** drawer, because today is the drawer somebody
// counts tonight. The sale keeps its own date: Tuesday's covers and Tuesday's
// dish counts do not move. See AdminPayDebt, which is the same rule.

type tillDebtRow struct {
	OrderID string    `json:"orderId"`
	Number  string    `json:"number"`
	At      time.Time `json:"at"`
	Total   int       `json:"total"`
	Note    string    `json:"note,omitempty"`
	Table   string    `json:"table,omitempty"`
}

// TillDebts finds what a guest owes, by their phone number.
//
// ⚠️ **By phone, and only by phone.** It is the one thing a cashier can ask for
// and a guest will answer; a name is not unique and nobody knows their customer
// id. An empty query returns nothing rather than everybody: a list of every
// debtor in the restaurant, on a screen in the dining room, is the customer
// base on display to whoever is standing at the counter.
func (h *Handler) TillDebts(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermCashier)
	if !ok {
		return
	}
	phone, valid := normalizePhone(r.URL.Query().Get("phone"))
	if !valid {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"debts": []tillDebtRow{}, "total": 0,
		})
		return
	}
	var user models.User
	if err := h.Store.Users.FindOne(r.Context(),
		bson.M{"phone": phone}).Decode(&user); err != nil {
		// Not an error: a guest with no account has no slate, and a till that
		// says "not found" for that teaches the cashier the search is broken.
		httpx.JSON(w, http.StatusOK, map[string]any{
			"debts": []tillDebtRow{}, "total": 0,
		})
		return
	}

	// ⚠️ The branch comes from the employee, as everywhere else at the till: a
	// cashier in one dining room must not be able to settle — or read — another
	// branch's slate.
	filter := debtFilter(bson.M{"branchId": s.BranchID})
	filter["userId"] = user.ID
	cur, err := h.Store.Orders.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(50))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var orders []models.Order
	_ = cur.All(r.Context(), &orders)

	rows := make([]tillDebtRow, 0, len(orders))
	total := 0
	for _, o := range orders {
		row := tillDebtRow{
			OrderID: o.ID.Hex(), Number: o.Number, Total: o.Total,
			Note: o.DebtNote, Table: o.TableNumber,
			At: o.CreatedAt.In(time.Local),
		}
		if o.Check != nil && o.Check.ClosedAt != nil {
			row.At = o.Check.ClosedAt.In(time.Local)
		}
		rows = append(rows, row)
		total += o.Total
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"name":  strings.TrimSpace(user.FirstName + " " + user.LastName),
		"phone": user.Phone,
		"debts": rows,
		"total": total,
	})
}

// TillCustomerLookup finds the guest a debt is being written against.
//
// ⚠️ **Its own endpoint, and deliberately a narrow one.** The till used to call
// `/admin/lookup` for this, which is wrong twice over. It needs an
// administrator's token, which a monoblock does not have and never should — so
// on the desktop app the call simply failed, the error was swallowed, and the
// cashier got an empty panel with no explanation. In a browser it *worked*,
// which was worse: it worked only because somebody had signed into the panel
// on that machine, so the feature's behaviour depended on whose browser it was.
//
// ⚠️ And `/admin/lookup` answers a completely different question. It returns
// the whole customer card — every order, every address, every complaint, every
// call — which is the call-centre's screen and is documented as the largest
// leak this system has. A counter needs a name to write on a slate, so that is
// all this returns.
func (h *Handler) TillCustomerLookup(w http.ResponseWriter, r *http.Request) {
	// The cashier's permission, because writing a debt is a cashier's act — the
	// same one that takes the money for it.
	if _, ok := h.tillStaff(w, r, models.PermCashier); !ok {
		return
	}
	phone, valid := normalizePhone(r.URL.Query().Get("phone"))
	if !valid {
		// ⚠️ Not an error. A half-typed number is the ordinary state of this
		// field, and a red message on every third keystroke is a message
		// nobody reads by the fourth.
		httpx.JSON(w, http.StatusOK, map[string]any{"user": nil})
		return
	}
	var user models.User
	if err := h.Store.Users.FindOne(r.Context(),
		bson.M{"phone": phone}).Decode(&user); err != nil {
		// A guest with no account is not a failure either — it is the answer,
		// and the screen says so in words.
		httpx.JSON(w, http.StatusOK, map[string]any{"user": nil})
		return
	}
	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		// ⚠️ The number stands in for a missing name rather than an empty
		// string: a slate reading "— owes 240 000" is the notebook again.
		name = user.Phone
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id": user.ID.Hex(), "name": name, "phone": user.Phone,
			// ⚠️ Sent so the till can say *why* before the cashier types a
			// note and presses close: the refusal is enforced when the check
			// is closed (see StaffCloseCheck), and a rule a screen only
			// discovers at the last press is one that reads as a bug, in
			// front of the guest it is about.
			"creditAllowed": user.CreditAllowed,
		},
	})
}

// TillPayDebt takes the money for a debt at the counter.
func (h *Handler) TillPayDebt(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermCashier)
	if !ok {
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var req debtPaymentRequest
	if err := httpx.DecodeOptional(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	method := req.Method
	if method == "" {
		method = models.ProviderCash
	}
	// ⚠️ Cash, card or transfer — never a provider rail and never the slate.
	// A QR here would need the guest to be standing over an old check while a
	// new callback lands on it, and paying a debt with a debt is a repayment
	// that changes nothing.
	if method == models.MethodDebt || bankConfirmed(method) || !tillMethods[method] {
		httpx.Error(w, http.StatusBadRequest, "noma'lum to'lov turi")
		return
	}

	filter := debtFilter(bson.M{"branchId": s.BranchID})
	filter["_id"] = id
	now := time.Now()
	set := bson.M{
		"paymentMethod": method,
		"paymentStatus": models.PayPaid,
		// Today's drawer, deliberately — see the note at the top of this block.
		"paidAt":    now,
		"updatedAt": now,
	}
	// ⚠️ Guarded by the debt filter rather than by the id: the panel and the
	// till can both be looking at this debt, and the money must be taken once.
	res, err := h.Store.Orders.UpdateOne(r.Context(), filter, bson.M{"$set": set})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.MatchedCount == 0 {
		httpx.Error(w, http.StatusNotFound, "qarz topilmadi yoki allaqachon yopilgan")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "paidAt": now})
}
