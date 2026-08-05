package handlers

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"restaurant-backend/internal/models"
)

// Click SHOP API.
//
// Two calls, in order: **Prepare** (action 0) asks whether the bill exists and
// may be paid, **Complete** (action 1) says the money moved. Both are signed
// with an MD5 over the request fields plus the merchant's secret key, and both
// carry the shop's own bill id in `merchant_trans_id` — for us, the order
// number on the receipt.
//
// The signature is the only authentication Click offers, which makes verifying
// it the whole of the security here. Note that the two signatures are *not* the
// same formula: Complete has merchant_prepare_id spliced into the middle. An
// implementation that reuses one formula for both passes Prepare, fails
// Complete, and looks like a Click outage.
//
// Amounts arrive in so'm with decimals ("78000.00"), unlike Payme and Uzum.

// Click error codes, from the vendor's own reference implementation.
const (
	clickOK = 0
	// The signature did not match.
	clickErrSign = -1
	// The amount is not the bill's.
	clickErrAmount = -2
	// action was neither 0 nor 1.
	clickErrAction = -3
	// This bill is already settled.
	clickErrAlreadyPaid = -4
	// No such bill.
	clickErrNoOrder = -5
	// No such transaction (Complete without a Prepare we know about).
	clickErrNoTxn = -6
	// Required fields missing from the request.
	clickErrRequest = -8
	// Cancelled — also what we answer when Click itself reports a failure.
	clickErrCancelled = -9
)

type clickCallback struct {
	ClickTransID      string `json:"click_trans_id"`
	ServiceID         string `json:"service_id"`
	ClickPaydocID     string `json:"click_paydoc_id"`
	MerchantTransID   string `json:"merchant_trans_id"`
	MerchantPrepareID string `json:"merchant_prepare_id"`
	Amount            string `json:"amount"`
	Action            string `json:"action"`
	Error             string `json:"error"`
	ErrorNote         string `json:"error_note"`
	SignTime          string `json:"sign_time"`
	SignString        string `json:"sign_string"`
}

// ClickPrepare is action 0: Click is asking whether this bill can be paid.
func (h *Handler) ClickPrepare(w http.ResponseWriter, r *http.Request) {
	h.clickHandle(w, r, "0")
}

// ClickComplete is action 1: the money has moved (or Click is reporting that it
// did not).
func (h *Handler) ClickComplete(w http.ResponseWriter, r *http.Request) {
	h.clickHandle(w, r, "1")
}

func (h *Handler) clickHandle(w http.ResponseWriter, r *http.Request, expected string) {
	req, err := parseClickCallback(r)
	if err != nil {
		clickFail(w, nil, clickErrRequest, "Error in request from click")
		return
	}
	s := h.paymentSettings(r.Context())
	if !s.Click.Enabled || s.Click.SecretKey == "" {
		clickFail(w, req, clickErrSign, "SIGN CHECK FAILED!")
		return
	}
	// The endpoint the request arrived at decides the action, not the field:
	// Click posts both to distinct URLs, and trusting the body would let a
	// Prepare-shaped request settle a bill.
	if req.Action != expected {
		clickFail(w, req, clickErrAction, "Action not found")
		return
	}
	if !clickSignValid(req, s.Click.SecretKey) {
		clickFail(w, req, clickErrSign, "SIGN CHECK FAILED!")
		return
	}
	if req.ServiceID != s.Click.ServiceID {
		clickFail(w, req, clickErrRequest, "Error in request from click")
		return
	}

	order, err := h.orderByNumber(r.Context(), req.MerchantTransID)
	if err != nil {
		clickFail(w, req, clickErrNoOrder, "Order does not exist")
		return
	}
	if order.PaymentMethod != models.ProviderClick {
		clickFail(w, req, clickErrNoOrder, "Order does not exist")
		return
	}
	if order.PaymentStatus == models.PayPaid {
		clickFail(w, req, clickErrAlreadyPaid, "Already paid")
		return
	}
	if order.Status == models.StatusCancelled {
		clickFail(w, req, clickErrCancelled, "Transaction cancelled")
		return
	}
	// Click sends so'm with decimals; the order is whole so'm. A tolerance of
	// half a tiyin covers the float round-trip without letting a real
	// difference through.
	amount, convErr := strconv.ParseFloat(req.Amount, 64)
	if convErr != nil || math.Abs(amount-float64(order.Total)) > 0.01 {
		clickFail(w, req, clickErrAmount, "Incorrect parameter amount")
		return
	}

	if expected == "0" {
		h.clickDoPrepare(w, r, req, order)
		return
	}
	h.clickDoComplete(w, r, req, order)
}

func (h *Handler) clickDoPrepare(
	w http.ResponseWriter, r *http.Request, req *clickCallback, order *models.Order,
) {
	txn, err := h.createTxn(r.Context(), models.ProviderClick, req.ClickTransID, order, 0)
	if err != nil {
		clickFail(w, req, clickErrRequest, err.Error())
		return
	}
	if txn.State != models.TxnCreated {
		clickFail(w, req, clickErrCancelled, "Transaction cancelled")
		return
	}
	// merchant_prepare_id is ours to choose and comes back inside the Complete
	// signature. The ledger row's id is the natural answer: it is unique, and it
	// ties the two calls together without a second lookup.
	clickOKResp(w, req, txn.ID.Hex())
}

func (h *Handler) clickDoComplete(
	w http.ResponseWriter, r *http.Request, req *clickCallback, order *models.Order,
) {
	txn, err := h.findTxn(r.Context(), models.ProviderClick, req.ClickTransID)
	if err != nil {
		clickFail(w, req, clickErrNoTxn, "Transaction does not exist")
		return
	}
	// The prepare id has to be the one we handed out, or this Complete belongs
	// to a different attempt.
	if !strings.EqualFold(req.MerchantPrepareID, txn.ID.Hex()) {
		clickFail(w, req, clickErrNoTxn, "Transaction does not exist")
		return
	}
	if txn.OrderID != order.ID {
		clickFail(w, req, clickErrNoTxn, "Transaction does not exist")
		return
	}

	// Click reporting its own failure: the bill goes back to unpaid so the
	// guest can try again with another card.
	if code, _ := strconv.Atoi(req.Error); code < 0 {
		_ = h.cancelTxn(r.Context(), txn, code)
		clickFail(w, req, clickErrCancelled, "Transaction cancelled")
		return
	}
	if err := h.performTxn(r.Context(), txn); err != nil {
		clickFail(w, req, clickErrCancelled, err.Error())
		return
	}
	h.afterPaid(r, txn)
	clickOKResp(w, req, txn.ID.Hex())
}

// parseClickCallback reads either form-encoded or JSON: Click posts forms, but
// its own test tools and some cabinets send JSON.
func parseClickCallback(r *http.Request) (*clickCallback, error) {
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			return nil, err
		}
		var req clickCallback
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, err
		}
		return &req, nil
	}
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	f := r.PostForm
	req := clickCallback{
		ClickTransID:      f.Get("click_trans_id"),
		ServiceID:         f.Get("service_id"),
		ClickPaydocID:     f.Get("click_paydoc_id"),
		MerchantTransID:   f.Get("merchant_trans_id"),
		MerchantPrepareID: f.Get("merchant_prepare_id"),
		Amount:            f.Get("amount"),
		Action:            f.Get("action"),
		Error:             f.Get("error"),
		ErrorNote:         f.Get("error_note"),
		SignTime:          f.Get("sign_time"),
		SignString:        f.Get("sign_string"),
	}
	if req.ClickTransID == "" || req.MerchantTransID == "" || req.SignString == "" {
		return nil, fmt.Errorf("missing fields")
	}
	return &req, nil
}

// clickSignValid checks the MD5. Complete splices merchant_prepare_id in after
// merchant_trans_id; Prepare leaves that slot empty.
func clickSignValid(req *clickCallback, secret string) bool {
	prepareID := ""
	if req.Action == "1" {
		prepareID = req.MerchantPrepareID
	}
	raw := req.ClickTransID + req.ServiceID + secret + req.MerchantTransID +
		prepareID + req.Amount + req.Action + req.SignTime
	sum := md5.Sum([]byte(raw))
	return strings.EqualFold(hex.EncodeToString(sum[:]), req.SignString)
}

func clickOKResp(w http.ResponseWriter, req *clickCallback, prepareID string) {
	writeJSON(w, map[string]any{
		"click_trans_id":      req.ClickTransID,
		"merchant_trans_id":   req.MerchantTransID,
		"merchant_prepare_id": prepareID,
		"merchant_confirm_id": prepareID,
		"error":               clickOK,
		"error_note":          "Success",
	})
}

func clickFail(w http.ResponseWriter, req *clickCallback, code int, note string) {
	out := map[string]any{"error": code, "error_note": note}
	if req != nil {
		out["click_trans_id"] = req.ClickTransID
		out["merchant_trans_id"] = req.MerchantTransID
	}
	writeJSON(w, out)
}

// clickCheckoutURL is the hosted payment page. `transaction_param` is what
// comes back as merchant_trans_id, so it carries the order number.
func clickCheckoutURL(s *models.PaymentSettings, order *models.Order, returnTo string) string {
	q := url.Values{}
	q.Set("service_id", s.Click.ServiceID)
	q.Set("merchant_id", s.Click.MerchantID)
	q.Set("amount", strconv.Itoa(order.Total))
	q.Set("transaction_param", order.Number)
	if back := paymeReturnURL(returnTo, order.Number); back != "" {
		q.Set("return_url", back)
	}
	return "https://my.click.uz/services/pay?" + q.Encode()
}
