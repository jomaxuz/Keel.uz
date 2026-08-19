package handlers

import (
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"restaurant-backend/internal/httpx"
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
// ⚠️ **The terminal is not one of these**, and deliberately never will be. A
// bank terminal on the counter has its own receipt, its own settlement and its
// own money that never passes through this system; `card` records that it was
// used. Pretending we confirm it would put a green tick on the one payment we
// cannot see.

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
