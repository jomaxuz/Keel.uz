package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/instore"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Closing a check: discount, payment, and the two ways a table ends.
//
// ⚠️ **Nothing here teaches the cash drawer about a new kind of sale.** The
// shift already counts `dinein` orders that are cash, paid, and stamped inside
// the shift window (see shiftFigures) — so a check closed for cash lands in the
// expected-in-drawer figure with no change to that code at all. This is the
// payoff of the decision at the top of check.go: a till sale is an order, so
// every screen that already understood orders understood the till on day one.

// tillMethods are what a counter can be paid with.
//
// Deliberately not the website's list. Payme and Click take a guest to a bank
// page on their own phone, which is a checkout flow and not something a cashier
// can drive; "card" here means the bank terminal on the counter, whose receipt
// is the proof and whose money never passes through this system.
var tillMethods = map[string]bool{
	models.ProviderCash: true,
	"card":              true,
	"transfer":          true,
	// ⚠️ **Not a way of paying — a way of not paying yet**, and that is the
	// whole reason it is here rather than left to a note in a book. A regular
	// who eats today and settles on Friday is a real and ordinary thing in this
	// business; what is not ordinary is a sale that leaves no record, and that
	// is what the paper book by the till produces. The money is owed by a
	// **named** customer, it is not takings until it arrives, and it turns into
	// takings on the day the cashier records the repayment — see MethodDebt.
	models.MethodDebt: true,
	// ⚠️ **The online rails, and they are accepted here only once the bank has
	// already said yes.** The guest scans a QR on the till screen and pays on
	// their own phone; the provider tells the server, and the cashier's press
	// of "pay" then records a payment that has already happened. See
	// handlers/tillpay.go — closing one of these while it is still `pending`
	// is refused below, because a check closed on an unconfirmed payment is
	// food handed over for money that was cancelled.
	// ⚠️ **The marketplaces, whose money somebody else is holding.** A Yandex
	// Eats order is rung up here like any other and paid by nobody at the
	// counter: the guest paid the aggregator weeks before the transfer arrives.
	// Recording it as cash would put money in a drawer that never saw it; as
	// "card", it would vanish into the acquirer's balance and no screen could
	// say who owes what. Accepted by the server unconditionally, but only
	// *offered* on tills whose restaurant has switched the marketplace on —
	// a method the server refuses is a cashier who cannot close a check.
	models.ProviderYandexEats: true,
	models.ProviderUzumTezkor: true,
	models.ProviderPayme:      true,
	models.ProviderClick:      true,
	models.ProviderUzum:       true,
	// ⚠️ **The counter rails, and they are not the three above.** Same banks,
	// different evidence: there the guest paid on their own phone through a
	// checkout page and a callback confirmed it; here a cashier scanned a code
	// on the guest's phone and our own request charged the card. The refund
	// route differs (we call the bank by payment id), the settlement report
	// differs, and "who was standing there" has an answer in one case and not
	// the other — so a report that folded them together could answer none of
	// it. See internal/instore.
	instore.ClickPass:   true,
	instore.UzumFastPay: true,
}

// bankConfirmed reports whether this method may only be closed once a bank has
// said the money arrived.
//
// ⚠️ **One list, asked in three places.** It started as `tillOnlineMethods` and
// the counter rails would have had to be added to each caller by hand — which
// is the shape where the third caller is missed and a scanned card that was
// declined closes a check anyway. The question is "did somebody outside this
// building confirm it", and cash, the terminal, a transfer and the slate all
// answer no for entirely different reasons.
func bankConfirmed(method string) bool {
	return tillOnlineMethods[method] || instore.Known(method)
}

type closeCheckRequest struct {
	PaymentMethod string `json:"paymentMethod"`
	// Who owes it, when the method is debt. ⚠️ Required in that case: "somebody
	// will pay later" is exactly the record the paper book already keeps badly.
	UserID string `json:"userId"`
	// What was said at the counter — "to'yga, juma kuni", "direktor aytdi".
	// Optional, and worth having: a debt with no sentence beside it is the one
	// nobody can chase without ringing somebody to ask what it was.
	DebtNote string `json:"debtNote"`
	// A discount the cashier gives at the counter, in so'm off the subtotal.
	// Capped at the subtotal server-side: a till that can be talked into a
	// negative total is a till that can be talked into paying the guest.
	Discount int `json:"discount"`
	// Why. Required whenever the discount is non-zero, for the same reason a
	// void is: an untraceable discount and an untraceable void take money out
	// of a restaurant by exactly the same route.
	DiscountReason string `json:"discountReason"`
	// A code from somebody who may give discounts, when the person at the
	// screen may not.
	PIN string `json:"pin"`
}

// StaffCloseCheck takes payment and closes a check.
func (h *Handler) StaffCloseCheck(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermCashier)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok || !requireOpen(w, o) {
		return
	}
	var req closeCheckRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	method := req.PaymentMethod
	if method == "" {
		method = models.ProviderCash
	}
	if !tillMethods[method] {
		httpx.Error(w, http.StatusBadRequest, "noma'lum to'lov turi")
		return
	}

	// ⚠️ A debt has to be owed by somebody. Without a customer this is a sale
	// that vanished: nothing to chase, nothing on anybody's card, and a total
	// that quietly stops adding up at the end of the month.
	var debtor primitive.ObjectID
	if method == models.MethodDebt {
		id, err := objectID(req.UserID)
		if err != nil || id.IsZero() {
			httpx.Error(w, http.StatusBadRequest, "qarzni kim olayotganini tanlang")
			return
		}
		var debtorUser models.User
		if err := h.Store.Users.FindOne(r.Context(), bson.M{"_id": id}).
			Decode(&debtorUser); err != nil {
			httpx.Error(w, http.StatusBadRequest, "mijoz topilmadi")
			return
		}
		// ⚠️ **Whether this guest may owe is the owner's decision, and this is
		// where it is enforced.** Writing a debt is a cashier's act; choosing
		// who is trusted with one cannot be, or the two are the same person —
		// and then the oldest trick at a counter works: put the evening's
		// shortfall on a name found by phone, and the drawer counts correct.
		// The screen hides the button too, and the screen is not the gate.
		if !debtorUser.CreditAllowed {
			httpx.Error(w, http.StatusForbidden,
				"bu mijozga qarz yozib bo'lmaydi — ruxsatni ega beradi")
			return
		}
		debtor = id
	}

	// ⚠️ **A provider's word, not the cashier's.** Marking these paid on the
	// press of a button would work in every test — in a test the payment
	// succeeds — and in a queue would hand out food for a payment that was
	// cancelled, expired, or made on somebody else's screen. Same rule the
	// website has always followed: the browser proves nothing, only the
	// server-to-server call does.
	if bankConfirmed(method) && paymentStatusOf(o) != models.PayPaid {
		httpx.Error(w, http.StatusConflict,
			"to'lov hali tasdiqlanmadi — mijoz to'laganini kuting")
		return
	}

	// ⚠️ **Asked again here, and this is the gate that matters.** A line can
	// predate the flag being turned on, and the receipt is what the law is
	// about: a bottle sold without its code filed is never withdrawn from
	// circulation, and nothing after the money is taken can put that right.
	if msg, err := h.markingRefusal(r.Context(), o.Items); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	} else if msg != "" {
		httpx.Error(w, http.StatusConflict, msg)
		return
	}

	// ⚠️ **The second question, and only where a shop has said it wants it
	// asked**: is this bottle one we actually took in? Silent on every install
	// that scans only at the till, which is all of them today — see
	// handlers/markinginbound.go.
	if msg, err := h.markInboundRefusal(r.Context(), o.BranchID, o.Items); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	} else if msg != "" {
		httpx.Error(w, http.StatusConflict, msg)
		return
	}

	live := o.LiveItems()
	if len(live) == 0 {
		// An empty check was never a sale. Closing it as one would put a zero
		// into the average-check figure every time a table changed its mind at
		// the door — see cancel below, which is the honest end for this.
		httpx.Error(w, http.StatusBadRequest,
			"chek bo'sh — to'lash o'rniga bekor qiling")
		return
	}

	now := time.Now()
	set := bson.M{"updatedAt": now}
	update := bson.M{"$set": set}

	// ⚠️ **Anything still unfired is sent to the kitchen now.** Paying for food
	// is the strongest possible commitment to cooking it, and the alternative —
	// refusing to close — strands a counter sale where the cashier rings up two
	// samsa and takes the cash in one movement. A guest who has paid for a dish
	// the pass never heard about is the one outcome this must not produce.
	for i := range o.Items {
		if o.Items[i].Live() && o.Items[i].FiredAt == nil {
			at := now
			o.Items[i].FiredAt = &at
		}
	}
	set["items"] = o.Items
	if o.QueuedAt == nil {
		o.QueuedAt = &now
		set["queuedAt"] = now
	}

	// Discount before the total, and capped: the pricing pipeline's rule that
	// a total never goes below zero, applied at the counter.
	if req.Discount > 0 {
		// ⚠️ Its own permission, separate from taking payment: a restaurant
		// that trusts somebody with the drawer may not trust them to decide
		// what a table owes. Missing it asks for a manager's code rather than
		// refusing — see handlers/tilloverride.go.
		who, err := h.resolveActor(r.Context(), s, models.PermDiscount, req.PIN)
		if err != nil {
			if errors.Is(err, errNeedsOverride) {
				overrideDenied(w, models.PermDiscount)
			} else {
				httpx.Error(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		reason := clampText(req.DiscountReason, 200)
		if reason == "" {
			httpx.Error(w, http.StatusBadRequest, "chegirma sababini yozing")
			return
		}
		// The authorising name goes into the discount's own label, because
		// that label is what the receipt shows a month later.
		if who.AuthBy != "" {
			reason += " (" + who.AuthBy + " tasdiqladi)"
		}
		subtotal := 0
		for _, it := range live {
			subtotal += it.Price * it.Qty
		}
		amount := min(req.Discount, subtotal)
		o.DiscountTotal = amount
		set["discountTotal"] = amount
		// Copied onto the order by name and amount, as every other discount is,
		// so the receipt explains itself a month later.
		set["discounts"] = []models.OrderDiscount{{
			Name:   "Kassa chegirmasi: " + reason,
			Amount: amount,
			// ⚠️ **Both names as fields, not only inside the label.** The label
			// is what the guest's receipt says a month later and keeps its
			// sentence; these are for the question nobody could previously ask
			// — who takes money off tables, and how often.
			ByID:     who.ByID,
			By:       who.By,
			AuthByID: who.AuthByID,
			AuthBy:   who.AuthBy,
			Reason:   clampText(req.DiscountReason, 200),
		}}
		// ⚠️ **Written to the order in memory as well as to the update, and
		// this line is the whole reason the alert never fired.**
		//
		// Everything else here writes into `set`, which is correct — it is what
		// reaches Mongo. But `alertOnDiscount` runs a few lines below and reads
		// `o.Discounts`, and `o` had never been told. The database had the
		// discount, the response had it, the receipt printed it, and the one
		// thing that was supposed to notice it was handed an empty slice.
		//
		// Silent in the worst way: a restaurant took 585 000 so'm off two bills
		// and the owner was told nothing, while every other trace of it was
		// perfectly correct.
		if d, ok := set["discounts"].([]models.OrderDiscount); ok {
			o.Discounts = d
		}
	}
	applyCheckTotals(o, set)

	// `delivered` is the terminal state for a dine-in order and always has
	// been — the guest has the food, there is nothing left to carry. Reusing it
	// is what keeps the till out of the statistics, the reports and the three
	// dictionaries that already know how to read an order's life.
	set["status"] = models.StatusDelivered
	set["paymentMethod"] = method
	// ⚠️ **A debt is closed but not paid**, and every screen downstream is
	// built on that distinction already: `received()` asks whether the money is
	// in the restaurant's hands, the drawer sums cash sales *paid* inside the
	// shift, and the financial report's "still out" line is exactly this.
	// Marking it paid because the guest walked out with the food would book
	// takings that may never arrive — the mistake this system was fixed of
	// once, at the dashboard.
	if method == models.MethodDebt {
		set["paymentStatus"] = models.PayUnpaid
		set["userId"] = debtor
		if note := clampText(req.DebtNote, 200); note != "" {
			set["debtNote"] = note
		}
	} else {
		set["paymentStatus"] = models.PayPaid
		// ⚠️ For an online payment this is already set, by the callback, to the
		// minute the bank confirmed — and that minute is the one that belongs
		// on the sale. Overwriting it with "when the cashier got round to
		// pressing the button" would move money between shifts at exactly the
		// hour a shift changes.
		if o.PaidAt == nil {
			set["paidAt"] = now
		}
	}
	set["readyAt"] = now
	set["check.closedAt"] = now
	set["check.closedById"] = s.ID
	set["check.closedBy"] = s.Name

	// ⚠️ **The sale is marked as owing a fiscal receipt here, at the moment the
	// money is taken** — not when somebody gets around to asking for the filing.
	//
	// Two things fall out of that, and both are the difference between a system
	// that files receipts and one that usually does:
	//
	//   - The relay on the register's PC polls for exactly this state, so a sale
	//     is filed **even if no browser is involved at all**. The cashier's tab
	//     crashing between taking the cash and filing used to lose the receipt
	//     silently, and that gap closed by itself.
	//   - "Paid but not filed" becomes an exact question the panel can ask. If
	//     the flag were written only when a filing was attempted, the sales that
	//     never got that far — the ones most worth finding — would be the ones
	//     invisible to the alert.
	//
	// ⚠️ Gated on there being a working adapter, not merely on the setting: a
	// provider we cannot build requests for would mark every sale as owing a
	// receipt that nothing will ever file, and the alert built on this would be
	// a permanent red badge — the kind people switch off and stop reading.
	if fset, enc, _ := h.tillFiscal(r.Context(), s.BranchID); enc != nil && fset.Enabled {
		set["fiscal"] = models.FiscalReceipt{
			Status:   models.FiscalPending,
			Provider: fset.Provider,
		}
	}
	update["$push"] = bson.M{"statusHistory": models.StatusEvent{
		Status: models.StatusDelivered, At: now,
	}}

	if _, err := h.Store.Orders.UpdateOne(r.Context(),
		checkFilter(o.ID, s.BranchID), update); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	o.Status = models.StatusDelivered
	o.Check.ClosedAt = &now
	o.Check.ClosedBy = s.Name
	// Nothing new leaves the shelf at payment — every line was written when it
	// was rung up. This is the last moment anybody touches the check, so it is
	// where a row lost to a blip earlier in the evening gets written.
	h.syncOrderStock(r.Context(), o)

	// ⚠️ **Printed here only when nothing will file it.** A restaurant with a
	// register owes the guest a receipt carrying a fiscal sign, and the sign
	// does not exist until the register answers — printing now would hand over
	// a slip that is missing the one thing the guest is entitled to check. With
	// no register there is nothing to wait for, and the paper is due
	// immediately: somebody is standing at the counter with their money out.
	if o.Fiscal == nil || o.Fiscal.Status != models.FiscalPending {
		h.queueSaleReceipts(r.Context(), o)
	}
	// ⚠️ **On closing, not on opening the check.** A table sitting with osh on
	// it has not sold it — the guests could leave, the check could be
	// cancelled, and a dish stopped by a table that never paid is a dish
	// refused to somebody standing at the counter with money out.
	h.applyDailyLimits(r.Context(), o.BranchID)

	// ⚠️ **After the sale is filed, and best effort.** The receipt is the legal
	// record and the money is taken; a store row that failed to update must not
	// unwind either. Its absence is the safe direction — the code reads as
	// unsold and the next scan of it is refused.
	h.markUnitsSold(r.Context(), o.ID, o.Items)

	// ⚠️ **On closing, and only on closing.** A line removed while a table is
	// still eating is ordinary work being done — the guest changed their mind,
	// the kitchen ran out. What the owner is told about is the shape of the
	// *finished* sale: what the guest was shown, and what they were charged.
	// Raised as a side effect and never able to fail this response — a cashier
	// must not be unable to take money because Telegram is slow.
	if aset := h.alertSettingsOf(r.Context(), o.BranchID); aset.Enabled {
		h.alertOnVoidsAfterPrecheck(o, aset)
		h.alertOnDiscount(o, aset)
	}
	httpx.JSON(w, http.StatusOK, viewCheck(o, now, s.ID))
}

type cancelCheckRequest struct {
	Reason string `json:"reason"`
	// Somebody else's code, when the person cancelling may not.
	PIN string `json:"pin,omitempty"`
}

// StaffCancelCheck ends a check without taking money.
//
// ⚠️ **Not the same event as an empty check being abandoned**, even though both
// end here. A reason is required exactly as it is on the panel's cancel button,
// and for the sharper version of the same argument: a check that reached the
// kitchen and was then cancelled is food the restaurant paid for, and "why"
// is the only question worth asking about it.
func (h *Handler) StaffCancelCheck(w http.ResponseWriter, r *http.Request) {
	// ⚠️ **Reachable by a waiter, then gated on `void` with an override** —
	// rather than refused outright to anybody without `cashier`, which is what
	// it did.
	//
	// Two things were wrong with the old gate. It asked for the wrong
	// permission: cancelling a check is voiding all of it, and the permission
	// for taking cooked food off a bill is `void`. And it *refused*, which is
	// the failure tilloverride.go is written against — the waiter at nine on a
	// Friday does not fetch the manager, they learn the manager's code, and
	// within a fortnight every cancellation in the journal carries one name.
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok || !requireOpen(w, o) {
		return
	}
	var req cancelCheckRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	reason := clampText(req.Reason, 200)
	// ⚠️ **An empty check has nothing to explain, and asking anyway is not
	// harmless.** A table opened by mistake — a wrong button, a guest who left
	// before ordering, a check opened twice — is the commonest cancellation
	// there is, and it costs the restaurant nothing: no dish was cooked and no
	// total existed. Demanding a sentence and a manager's PIN for it is the
	// friction `tilloverride.go` is written against: it does not stop anything,
	// it teaches a waiter to keep the manager's code in their head for the one
	// case that matters.
	//
	// Nothing cooked is the same test the loss alert uses, deliberately: the
	// question "was this cancellation worth anybody's attention" must have one
	// answer in this file.
	empty := cookedValue(o) == 0
	if reason == "" {
		if !empty {
			httpx.Error(w, http.StatusBadRequest, "bekor qilish sababini yozing")
			return
		}
		// Written rather than left blank: the journal reads "why was this
		// cancelled", and a row with nothing in it looks like a lost record
		// instead of an answer.
		reason = "bo'sh chek"
	}
	// ⚠️ The override follows the same line. Voiding a cooked dish still needs
	// somebody who may, because that is food and money leaving the building;
	// closing an empty table is not that act, whatever it is called in the code.
	perm := models.PermVoid
	if empty {
		perm = models.PermWaiter
	}
	who, err := h.resolveActor(r.Context(), s, perm, req.PIN)
	if err != nil {
		if errors.Is(err, errNeedsOverride) {
			overrideDenied(w, models.PermVoid)
		} else {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	// Both names on the record, the sentence this till exists to be able to
	// write a month later: "Aziz bekor qildi · Dilnoza tasdiqladi".
	if who.AuthBy != "" {
		reason += " (" + who.AuthBy + " tasdiqladi)"
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":           models.StatusCancelled,
			"cancelReason":     reason,
			"check.closedAt":   now,
			"check.closedById": s.ID,
			"check.closedBy":   s.Name,
			"updatedAt":        now,
		},
		"$push": bson.M{"statusHistory": models.StatusEvent{
			Status: models.StatusCancelled, At: now,
		}},
	}
	if _, err := h.Store.Orders.UpdateOne(r.Context(),
		checkFilter(o.ID, s.BranchID), update); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	o.Status = models.StatusCancelled
	o.Check.ClosedAt = &now
	// ⚠️ **Line by line, on the same test the alert above uses.** A table opened
	// by mistake cooked nothing and puts everything back; a cancelled check with
	// food on the pass leaves its rows standing, marked as waste. `cookedValue`
	// draws that line for the PIN and the alert, and the shelf must not draw a
	// second one — see handlers/stocksale.go.
	h.syncOrderStock(r.Context(), o)
	h.alertOnCancelledCheck(o, who, req)
	httpx.JSON(w, http.StatusOK, viewCheck(o, now, s.ID))
}

// cookedValue is what the kitchen actually made on this check.
//
// ⚠️ **One definition of "was anything lost here", used by both the alert and
// the cancellation gate.** A line that was never sent to the kitchen is a line
// nobody cooked: it can be removed, and a check made only of those is a table
// opened by mistake. Two copies of this test would eventually disagree, and
// then a cancellation would be worth a manager's PIN and not worth an alert, or
// the other way round.
func cookedValue(o *models.Order) int {
	if o == nil {
		return 0
	}
	value := 0
	for _, it := range o.Items {
		if it.Live() && it.FiredAt != nil {
			value += it.Price * it.Qty
		}
	}
	return value
}

// alertOnCancelledCheck tells the owner a check ended with food on it and no
// money.
//
// ⚠️ **The case this feature was asked for, and the one it shipped without.**
// "Take the cash and cancel the check as a mistake" is the first thing anybody
// describes when asked how a cashier steals — and every trigger was hung on the
// *close* path, which a cancelled check never reaches. Six kinds of alert, and
// the headline one was missing.
func (h *Handler) alertOnCancelledCheck(o *models.Order, who actor, req cancelCheckRequest) {
	if o == nil || o.Check == nil {
		return
	}
	// ⚠️ **Only a check something was actually cooked for.** A table opened by
	// mistake and closed again is the commonest cancellation in any restaurant,
	// and alerting on it would put a message on somebody's phone several times
	// a day — which is how the ones that matter stop being read.
	value := cookedValue(o)
	if value == 0 {
		return
	}
	set := h.alertSettingsOf(context.Background(), o.BranchID)
	if !set.Enabled || value < set.VoidFrom {
		return
	}
	// ⚠️ **Whether the guest had been shown the bill is carried in the words**,
	// because it is the difference between a table that changed its mind and a
	// total that existed and then did not. The same fact the void alert is
	// built on, and it belongs here more.
	h.raiseAlert(models.LossAlert{
		BranchID: o.BranchID,
		Kind:     models.AlertCheckCancelled,
		ByID:     who.ByID, By: who.By, AuthBy: who.AuthBy,
		Amount: value,
		// ⚠️ The reason as it was typed, without the "(X tasdiqladi)" the
		// receipt's label carries. Who approved it is a field on this alert and
		// is written in the group's own language; gluing an Uzbek word onto the
		// reason put it into Russian messages, and it was never the reason
		// anyway — it was decoration for a slip of paper.
		Reason: clampText(req.Reason, 200),
		// ⚠️ Whether the guest had been shown the bill, as a fact rather than a
		// sentence. It is the difference between a table that changed its mind
		// and a total that existed and then did not — and it has to read in the
		// language the group reads.
		AfterPrecheck: o.Check.PrecheckAt != nil,
		Number:        o.Number,
		Table:         o.TableNumber,
		RefID:         o.ID,
	})
}

// ---- Editing the check itself ----

type updateCheckRequest struct {
	Guests   *int    `json:"guests"`
	ServerID string  `json:"serverId"`
	TableID  *string `json:"tableId"`
	Comment  *string `json:"comment"`
}

// StaffUpdateCheck changes who and where a check is for.
//
// Pointers throughout: this is a partial update, and the panel's own hard-won
// rule applies — a form that sends only the field it edited must not blank the
// ones it never showed.
func (h *Handler) StaffUpdateCheck(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok || !requireOpen(w, o) {
		return
	}
	var req updateCheckRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now()
	set := bson.M{"updatedAt": now}
	if req.Guests != nil && *req.Guests >= 0 {
		o.Check.Guests = *req.Guests
		set["check.guests"] = *req.Guests
	}
	if req.Comment != nil {
		c := clampText(*req.Comment, 300)
		o.Address.Comment = c
		set["address.comment"] = c
	}
	if req.ServerID != "" {
		other, err := h.staffInBranch(r.Context(), req.ServerID, s.BranchID)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "ofitsiant topilmadi")
			return
		}
		o.Check.ServerID, o.Check.ServerName = other.ID, other.Name
		set["check.serverId"], set["check.serverName"] = other.ID, other.Name
	}
	// Moving a party to another table. Guarded by the same one-open-check rule
	// as opening: the destination must be free, or two waiters end up writing
	// onto one bill.
	if req.TableID != nil && *req.TableID != o.TableID {
		if *req.TableID == "" {
			o.TableID, o.TableNumber = "", ""
			set["tableId"], set["tableNumber"] = "", ""
		} else {
			branch, err := h.branchByID(r, s.BranchID)
			if err != nil {
				httpx.Error(w, http.StatusInternalServerError, err.Error())
				return
			}
			t, found := branchTable(branch, *req.TableID)
			if !found {
				httpx.Error(w, http.StatusBadRequest, "bunday stol xaritada yo'q — ekranni yangilang")
				return
			}
			busy, err := h.openCheckOnTable(r.Context(), s.BranchID, t.ID)
			if err != nil {
				httpx.Error(w, http.StatusInternalServerError, err.Error())
				return
			}
			if busy != nil && busy.ID != o.ID {
				httpx.Error(w, http.StatusConflict,
					t.Number+"-stolda ochiq chek bor ("+busy.Number+")")
				return
			}
			o.TableID, o.TableNumber = t.ID, t.Number
			set["tableId"], set["tableNumber"] = t.ID, t.Number
			// The receipt's name follows the table, since that is what it is.
			o.Customer.Name = guestLabel(t.Number)
			set["customer.name"] = o.Customer.Name
		}
	}

	if _, err := h.Store.Orders.UpdateOne(r.Context(),
		checkFilter(o.ID, s.BranchID), bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, viewCheck(o, now, s.ID))
}
