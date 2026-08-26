package handlers

import (
	"errors"
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
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
	models.ProviderPayme: true,
	models.ProviderClick: true,
	models.ProviderUzum:  true,
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
		if err := h.Store.Users.FindOne(r.Context(), bson.M{"_id": id}).
			Err(); err != nil {
			httpx.Error(w, http.StatusBadRequest, "mijoz topilmadi")
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
	if tillOnlineMethods[method] && paymentStatusOf(o) != models.PayPaid {
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
	httpx.JSON(w, http.StatusOK, viewCheck(o, now))
}

type cancelCheckRequest struct {
	Reason string `json:"reason"`
}

// StaffCancelCheck ends a check without taking money.
//
// ⚠️ **Not the same event as an empty check being abandoned**, even though both
// end here. A reason is required exactly as it is on the panel's cancel button,
// and for the sharper version of the same argument: a check that reached the
// kitchen and was then cancelled is food the restaurant paid for, and "why"
// is the only question worth asking about it.
func (h *Handler) StaffCancelCheck(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermCashier)
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
	if reason == "" {
		httpx.Error(w, http.StatusBadRequest, "bekor qilish sababini yozing")
		return
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
	httpx.JSON(w, http.StatusOK, viewCheck(o, now))
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
	httpx.JSON(w, http.StatusOK, viewCheck(o, now))
}
