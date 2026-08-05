package handlers

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// Uzum Bank merchant API.
//
// Five endpoints the bank calls on us — check, create, confirm, reverse,
// status — behind HTTP Basic auth with a login and password the bank issues.
// Same two-phase shape as Click underneath: `create` reserves, `confirm`
// settles, `reverse` unwinds.
//
// Amounts arrive in tiyin, like Payme. The bill is named inside `params`, whose
// field names are configured in the bank's cabinet — ours carries the order
// number.
//
// Uzum answers in plain JSON with a `status` string and, on failure, a numeric
// `errorCode`; there is no JSON-RPC envelope and no signature beyond the Basic
// credentials.

// Uzum error codes.
const (
	uzumErrAuth = 10001
	uzumErrJSON = 10002
	// The operation is not one of the five.
	uzumErrUnknownOp = 10003
	uzumErrNotEnough = 10005
	uzumErrServiceID = 10006
	// The transaction has already been settled or reversed.
	uzumErrAlreadyDone = 10007
	// No such transaction / the bill named in params does not exist.
	uzumErrNotFound  = 10008
	uzumErrCancelled = 10009
	// Anything wrong with the bill itself: missing, unpayable, wrong amount.
	uzumErrCheckData = 99999
)

// Uzum status strings.
const (
	uzumStatusOK        = "OK"
	uzumStatusFailed    = "FAILED"
	uzumStatusCreated   = "CREATED"
	uzumStatusConfirmed = "CONFIRMED"
	uzumStatusReversed  = "REVERSED"
)

type uzumRequest struct {
	ServiceID any            `json:"serviceId"`
	Timestamp int64          `json:"timestamp"`
	TransID   string         `json:"transId"`
	Amount    int64          `json:"amount"`
	Params    map[string]any `json:"params"`
}

// uzumAccountField is the name of the parameter carrying the order number,
// as configured in the bank's cabinet.
func uzumAccountField(s *models.PaymentSettings) string {
	if f := strings.TrimSpace(s.Uzum.AccountField); f != "" {
		return f
	}
	return "order_id"
}

// uzumGuard authenticates and decodes. Returns nil when it has already answered.
func (h *Handler) uzumGuard(w http.ResponseWriter, r *http.Request) (*uzumRequest, *models.PaymentSettings) {
	s := h.paymentSettings(r.Context())
	if !s.Uzum.Enabled || s.Uzum.Login == "" || s.Uzum.Password == "" {
		uzumFail(w, nil, uzumErrAuth)
		return nil, nil
	}
	if !uzumAuthorised(r, s) {
		uzumFail(w, nil, uzumErrAuth)
		return nil, nil
	}
	var req uzumRequest
	if err := httpx.Decode(r, &req); err != nil {
		uzumFail(w, nil, uzumErrJSON)
		return nil, nil
	}
	// The service id identifies which of the bank's services is calling. A
	// mismatch means this request is for somebody else's shop.
	if uzumString(req.ServiceID) != s.Uzum.ServiceID {
		uzumFail(w, &req, uzumErrServiceID)
		return nil, nil
	}
	return &req, s
}

func uzumAuthorised(r *http.Request, s *models.PaymentSettings) bool {
	raw := r.Header.Get("Authorization")
	const prefix = "Basic "
	if !strings.HasPrefix(raw, prefix) {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(raw, prefix))
	if err != nil {
		return false
	}
	login, password, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return false
	}
	// Both compared in constant time: the endpoint is public.
	okLogin := subtle.ConstantTimeCompare([]byte(login), []byte(s.Uzum.Login)) == 1
	okPass := subtle.ConstantTimeCompare([]byte(password), []byte(s.Uzum.Password)) == 1
	return okLogin && okPass
}

// UzumCheck answers whether the bill named in params exists and can be paid.
func (h *Handler) UzumCheck(w http.ResponseWriter, r *http.Request) {
	req, s := h.uzumGuard(w, r)
	if req == nil {
		return
	}
	order, err := h.uzumOrder(r, s, req)
	if err != nil {
		uzumFail(w, req, uzumErrCheckData)
		return
	}
	uzumWrite(w, map[string]any{
		"serviceId": req.ServiceID,
		"timestamp": time.Now().UnixMilli(),
		"status":    uzumStatusOK,
		"data": map[string]any{
			"account": map[string]any{"value": order.Number},
		},
	})
}

// UzumCreate reserves the payment.
func (h *Handler) UzumCreate(w http.ResponseWriter, r *http.Request) {
	req, s := h.uzumGuard(w, r)
	if req == nil {
		return
	}
	if strings.TrimSpace(req.TransID) == "" {
		uzumFail(w, req, uzumErrNotEnough)
		return
	}
	// A repeat of a create we answered already: the bank is retrying, so it
	// gets the same answer rather than a duplicate-transaction error.
	if existing, err := h.findTxn(r.Context(), models.ProviderUzum, req.TransID); err == nil {
		if existing.State != models.TxnCreated {
			uzumFail(w, req, uzumErrAlreadyDone)
			return
		}
		uzumWrite(w, map[string]any{
			"serviceId": req.ServiceID,
			"timestamp": time.Now().UnixMilli(),
			"status":    uzumStatusCreated,
			"transTime": existing.CreateTimeMs,
			"transId":   existing.ProviderTxnID,
			"amount":    int64(existing.Amount) * 100,
		})
		return
	}

	order, err := h.uzumOrder(r, s, req)
	if err != nil {
		uzumFail(w, req, uzumErrCheckData)
		return
	}
	// Tiyin, like Payme.
	if req.Amount != int64(order.Total)*100 {
		uzumFail(w, req, uzumErrCheckData)
		return
	}
	txn, err := h.createTxn(r.Context(), models.ProviderUzum, req.TransID, order, 0)
	if err != nil {
		uzumFail(w, req, uzumErrCheckData)
		return
	}
	uzumWrite(w, map[string]any{
		"serviceId": req.ServiceID,
		"timestamp": time.Now().UnixMilli(),
		"status":    uzumStatusCreated,
		"transTime": txn.CreateTimeMs,
		"transId":   txn.ProviderTxnID,
		"amount":    int64(txn.Amount) * 100,
	})
}

// UzumConfirm settles it — this is the call that means the money moved.
func (h *Handler) UzumConfirm(w http.ResponseWriter, r *http.Request) {
	req, _ := h.uzumGuard(w, r)
	if req == nil {
		return
	}
	txn, err := h.findTxn(r.Context(), models.ProviderUzum, req.TransID)
	if err != nil {
		uzumFail(w, req, uzumErrNotFound)
		return
	}
	switch txn.State {
	case models.TxnPerformed:
		// Retry of a confirm that already worked: answer as before.
		uzumWrite(w, map[string]any{
			"serviceId":   req.ServiceID,
			"transId":     txn.ProviderTxnID,
			"status":      uzumStatusConfirmed,
			"confirmTime": txn.PerformTimeMs,
		})
		return
	case models.TxnCancelledSetup, models.TxnCancelledPaid:
		uzumFail(w, req, uzumErrCancelled)
		return
	}
	if err := h.performTxn(r.Context(), txn); err != nil {
		uzumFail(w, req, uzumErrAlreadyDone)
		return
	}
	h.afterPaid(r, txn)
	uzumWrite(w, map[string]any{
		"serviceId":   req.ServiceID,
		"transId":     txn.ProviderTxnID,
		"status":      uzumStatusConfirmed,
		"confirmTime": txn.PerformTimeMs,
	})
}

// UzumReverse unwinds a payment, settled or not.
func (h *Handler) UzumReverse(w http.ResponseWriter, r *http.Request) {
	req, _ := h.uzumGuard(w, r)
	if req == nil {
		return
	}
	txn, err := h.findTxn(r.Context(), models.ProviderUzum, req.TransID)
	if err != nil {
		uzumFail(w, req, uzumErrNotFound)
		return
	}
	if err := h.cancelTxn(r.Context(), txn, 0); err != nil {
		uzumFail(w, req, uzumErrAlreadyDone)
		return
	}
	uzumWrite(w, map[string]any{
		"serviceId":   req.ServiceID,
		"transId":     txn.ProviderTxnID,
		"status":      uzumStatusReversed,
		"reverseTime": txn.CancelTimeMs,
		"amount":      int64(txn.Amount) * 100,
	})
}

// UzumStatus reports where a transaction stands.
func (h *Handler) UzumStatus(w http.ResponseWriter, r *http.Request) {
	req, _ := h.uzumGuard(w, r)
	if req == nil {
		return
	}
	txn, err := h.findTxn(r.Context(), models.ProviderUzum, req.TransID)
	if err != nil {
		uzumFail(w, req, uzumErrNotFound)
		return
	}
	status := uzumStatusCreated
	switch txn.State {
	case models.TxnPerformed:
		status = uzumStatusConfirmed
	case models.TxnCancelledSetup, models.TxnCancelledPaid:
		status = uzumStatusReversed
	}
	uzumWrite(w, map[string]any{
		"serviceId": req.ServiceID,
		"transId":   txn.ProviderTxnID,
		"status":    status,
		"amount":    int64(txn.Amount) * 100,
	})
}

// uzumOrder resolves the bill named in params and checks it is payable.
func (h *Handler) uzumOrder(
	r *http.Request, s *models.PaymentSettings, req *uzumRequest,
) (*models.Order, error) {
	number := uzumString(req.Params[uzumAccountField(s)])
	if number == "" {
		return nil, fmt.Errorf("no account")
	}
	order, err := h.orderByNumber(r.Context(), number)
	if err != nil {
		return nil, err
	}
	if order.PaymentMethod != models.ProviderUzum {
		return nil, fmt.Errorf("order is not paid by uzum")
	}
	if err := payable(order); err != nil {
		return nil, err
	}
	return order, nil
}

// uzumString accepts either a string or a number: cabinets differ on whether
// they quote the service id and the order number.
func uzumString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return fmt.Sprintf("%.0f", t)
	case json.Number:
		return t.String()
	}
	return ""
}

func uzumWrite(w http.ResponseWriter, body map[string]any) {
	writeJSON(w, body)
}

func uzumFail(w http.ResponseWriter, req *uzumRequest, code int) {
	out := map[string]any{
		"timestamp": time.Now().UnixMilli(),
		"status":    uzumStatusFailed,
		"errorCode": code,
	}
	if req != nil {
		out["serviceId"] = req.ServiceID
		if req.TransID != "" {
			out["transId"] = req.TransID
		}
	}
	writeJSON(w, out)
}

// uzumCheckoutURL is the hosted checkout. The bank's page is addressed by
// service id and carries the bill in the same parameter name the callbacks use,
// so one setting drives both halves.
func uzumCheckoutURL(s *models.PaymentSettings, order *models.Order, returnTo string) string {
	q := url.Values{}
	q.Set("serviceId", s.Uzum.ServiceID)
	q.Set("amount", fmt.Sprintf("%d", int64(order.Total)*100))
	q.Set(uzumAccountField(s), order.Number)
	if back := paymeReturnURL(returnTo, order.Number); back != "" {
		q.Set("redirectUrl", back)
	}
	return "https://www.apelsin.uz/open-service?" + q.Encode()
}
