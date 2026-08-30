package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/instore"
	"restaurant-backend/internal/models"
)

// ---- Paying at the counter by scanning the guest's code ----
//
// ⚠️ **The opposite direction from tillpay.go, and that is the whole feature.**
// There, the till shows a QR and waits for a bank to call back; the cashier can
// do nothing but watch a spinner while a guest finds their phone, opens an app
// and confirms. Here the guest already has their code open, the cashier scans
// it, and the card is charged inside one request — the answer is on screen
// before the guest has put the phone down.
//
// ⚠️ **Which means the failure mode moves, and this file is mostly about that.**
// A callback that never arrives leaves a check open and obviously unpaid. A
// synchronous charge that times out leaves us not knowing whether the guest was
// charged, with the guest standing there. So: the payment is written to the
// order **before** the check is touched, the bank's id is stored whatever the
// outcome, and a timed-out attempt is answered by asking the bank rather than
// by trying again — see TillScanStatus.

// Why a rail cannot take money, kept apart because each sends the reader
// somewhere different: to the settings page, to the provider's cabinet, or to
// us. ⚠️ The last one is deliberately not "xatolik" — an owner told that a
// provider is not built yet stops looking for a mistake of their own.
var (
	errUnknownRail       = errors.New("noma'lum to'lov tizimi")
	errRailNotBuilt      = errors.New("bu to'lov tizimi uchun adapter hali yozilmagan")
	errRailNotConfigured = errors.New("bu to'lov tizimi hali sozlanmagan")
)

// TillScanPay charges the card behind a scanned code.
//
// ⚠️ **It does not close the check.** Closing is StaffCloseCheck's job and
// stays there: it fires the kitchen, applies the discount with its override,
// files the receipt, queues the paper and raises the alerts, and a second
// half-copy of that inside a payment handler is the copy that goes stale. What
// this does is turn "the guest wants to pay by card" into "the money is in",
// after which the ordinary close runs exactly as it does for an online QR
// payment the bank has already confirmed.
func (h *Handler) TillScanPay(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermCashier)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok || !requireOpen(w, o) {
		return
	}
	var req struct {
		Provider string `json:"provider"`
		Code     string `json:"code"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		httpx.Error(w, http.StatusBadRequest, "QR kod o'qilmadi — qayta skanerlang")
		return
	}
	if len(o.LiveItems()) == 0 {
		httpx.Error(w, http.StatusBadRequest, "chek bo'sh")
		return
	}

	charger, err := h.counterCharger(r.Context(), req.Provider, s.BranchID)
	if err != nil {
		httpx.Error(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	// ⚠️ **The amount is frozen here, from the lines, before anything is sent.**
	// Two reasons, and the second is the one that bites: a stale `total` on the
	// order would charge yesterday's figure, and — because both providers key
	// on our order number — a second attempt after a dish was added would be
	// refused as a duplicate rather than charging the new total. The totals are
	// recomputed and written first so what the bank is asked for is what the
	// check says.
	now := time.Now()
	set := bson.M{"updatedAt": now}
	applyCheckTotals(o, set)
	if _, err := h.Store.Orders.UpdateOne(r.Context(),
		bson.M{"_id": o.ID}, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	pay := models.CounterPayment{
		Provider: req.Provider,
		// ⚠️ Fresh per attempt, never per order. A retry after a timeout is a
		// different attempt against the same sale, and reusing the id would
		// have the bank answer about the first one — which is exactly the
		// attempt we do not know the outcome of.
		TxnID:  primitive.NewObjectID().Hex(),
		Status: instore.StatusPending,
		At:     now,
	}
	// Written before the bank is called, so an attempt that never comes back
	// still leaves a record of having been made. Nothing else on the order
	// changes: the check is still open and still unpaid.
	h.storeCounterPay(r.Context(), o.ID, pay)

	res, err := charger.Charge(r.Context(), instore.Charge{
		Amount:  o.Total,
		OTPData: code,
		OrderID: o.Number,
		TxnID:   pay.TxnID,
	})
	pay.PaymentID = res.PaymentID
	pay.CardMask = res.CardMask
	pay.Processing = res.Processing
	pay.Status = res.Status
	if err != nil {
		pay.Error = err.Error()
		if pay.Status == "" {
			// A transport failure, not a refusal: we do not know. Left pending
			// so the screen offers "tekshirish" rather than "qayta urinish" —
			// see the note at the top of this file.
			pay.Status = instore.StatusPending
		}
	}
	h.storeCounterPay(r.Context(), o.ID, pay)

	if pay.Status != instore.StatusPaid {
		h.finishScan(w, o, pay, err)
		return
	}

	// The money is in. Recorded on the order the same way a confirmed callback
	// records it, so StaffCloseCheck's gate — "a provider's word, not the
	// cashier's" — is satisfied by the same field it already reads.
	paid := bson.M{
		"paymentMethod": req.Provider,
		"paymentStatus": models.PayPaid,
		"paidAt":        time.Now(),
		"updatedAt":     time.Now(),
	}
	if _, err := h.Store.Orders.UpdateOne(r.Context(),
		bson.M{"_id": o.ID}, bson.M{"$set": paid}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.finishScan(w, o, pay, nil)
}

// TillScanStatus asks the bank again about an attempt whose answer never came.
//
// ⚠️ **The only correct move after a timeout**, and the reason the screen does
// not offer "qayta urinish" for one. Retrying a charge that may have succeeded
// is how a guest is charged twice; both providers guard against it by refusing
// a repeated order id, but relying on somebody else's guard for our own
// double-charge is not a design, it is luck. Asking is free and answers the
// actual question.
func (h *Handler) TillScanStatus(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermCashier)
	if !ok {
		return
	}
	o, ok := h.loadCheck(w, r, s)
	if !ok {
		return
	}
	if o.CounterPay == nil || o.CounterPay.PaymentID == "" {
		// Nothing was ever sent, or it failed before the bank issued an id.
		// Not an error: the cashier's next move is to scan again, and the
		// screen says so.
		httpx.JSON(w, http.StatusOK, map[string]any{"status": "", "paid": false})
		return
	}
	pay := *o.CounterPay
	charger, err := h.counterCharger(r.Context(), pay.Provider, s.BranchID)
	if err != nil {
		httpx.Error(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	res, err := charger.Status(r.Context(), pay.PaymentID, o.Number)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	pay.Status = res.Status
	if res.CardMask != "" {
		pay.CardMask = res.CardMask
	}
	if res.Status == instore.StatusPaid {
		pay.Error = ""
	}
	h.storeCounterPay(r.Context(), o.ID, pay)
	if res.Status == instore.StatusPaid && paymentStatusOf(o) != models.PayPaid {
		now := time.Now()
		_, _ = h.Store.Orders.UpdateOne(r.Context(), bson.M{"_id": o.ID},
			bson.M{"$set": bson.M{
				"paymentMethod": pay.Provider,
				"paymentStatus": models.PayPaid,
				"paidAt":        now,
				"updatedAt":     now,
			}})
	}
	h.finishScan(w, o, pay, nil)
}

// finishScan is the one answer shape both handlers give the till.
//
// ⚠️ **A refused card is a 200, not a 4xx.** The request worked; the bank said
// no. Answering with an error status would have the screen's generic handler
// paint "xatolik" over a sentence the cashier can act on — "mijozdan yangi QR
// so'rang" — and the difference between those two is whether the queue moves.
func (h *Handler) finishScan(
	w http.ResponseWriter, o *models.Order, pay models.CounterPayment, err error,
) {
	msg := pay.Error
	if err != nil && msg == "" {
		msg = err.Error()
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"status":     pay.Status,
		"paid":       pay.Status == instore.StatusPaid,
		"paymentId":  pay.PaymentID,
		"cardMask":   pay.CardMask,
		"processing": pay.Processing,
		"provider":   pay.Provider,
		"error":      msg,
		"total":      o.Total,
	})
}

// storeCounterPay writes the bank's answer onto the order.
//
// A storage failure is logged and swallowed on purpose: it must never be the
// reason a cashier is told a successful charge failed. What is lost is the
// record, which is recoverable from the provider's cabinet by order number —
// the money is not.
func (h *Handler) storeCounterPay(
	ctx context.Context, id primitive.ObjectID, pay models.CounterPayment,
) {
	if _, err := h.Store.Orders.UpdateOne(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"counterPay": pay, "updatedAt": time.Now()}}); err != nil {
		log.Printf("counter pay: not recorded on %s: %v", id.Hex(), err)
	}
}

// counterCharger builds the driver for a branch, or says why it cannot.
//
// ⚠️ **Three separate refusals, each with its own sentence**, because they send
// whoever reads them to three different places: the panel's settings page, the
// provider's cabinet, or us. A single "sozlanmagan" for all of them is a
// support call.
func (h *Handler) counterCharger(
	ctx context.Context, provider string, branchID primitive.ObjectID,
) (instore.Charger, error) {
	if !instore.Known(provider) {
		return nil, errUnknownRail
	}
	if !instore.Ready(provider) {
		return nil, errRailNotBuilt
	}
	creds, on := h.paymentSettings(ctx).InStoreCreds(provider)
	if !on {
		return nil, errRailNotConfigured
	}
	return instore.New(instore.Config{
		Provider:  provider,
		ServiceID: creds.ServiceID,
		UserID:    creds.UserID,
		SecretKey: creds.SecretKey,
		BaseURL:   creds.BaseURL,
		Cashbox:   h.cashboxCode(ctx, branchID),
	})
}

// cashboxCode is what this till is called in the bank's cabinet.
//
// ⚠️ **It is read by a human, in a settlement report, weeks later.** So it is
// the branch's name where there is one, and the fiscal register's id where the
// branch has one — a merchant reconciling two branches' takings needs the two
// figures to be labelled with something they recognise. An object id would be
// technically perfect and useless for the only job this string has.
func (h *Handler) cashboxCode(ctx context.Context, branchID primitive.ObjectID) string {
	if id := fiscalRegisterID(h.fiscalSettingsOf(ctx, branchID)); id != "" {
		return id
	}
	if b, err := h.branchByIDCtx(ctx, branchID); err == nil && b != nil {
		if name := strings.TrimSpace(b.Name); name != "" {
			return name
		}
	}
	if branchID.IsZero() {
		return "KASSA"
	}
	return "KASSA-" + branchID.Hex()[:6]
}
