package handlers

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Payme Merchant API.
//
// Payme does not tell us the guest paid — it asks us, in JSON-RPC, whether it
// may take the money, then tells us it did. Six methods, one endpoint, HTTP
// Basic auth with the login "Paycom" and the merchant key as the password.
//
// Two things about this protocol bite implementations that treat it as a
// webhook:
//
//   - **every method may be called more than once.** Payme retries on any
//     timeout, including ones it caused. PerformTransaction arriving twice is
//     normal traffic, not an attack.
//   - **the timestamps must be echoed back exactly.** CheckTransaction has to
//     return the same create_time it was given, which is why the ledger stores
//     the milliseconds rather than deriving them from a Go time.
//
// Amounts are in tiyin (1 so'm = 100 tiyin) throughout.

// Payme error codes. Stored as constants because several of them are returned
// from more than one method and a typo'd number reads as a different failure.
const (
	paymeErrAuth = -32504
	// Malformed JSON / bad request object / unknown method.
	paymeErrParse   = -32700
	paymeErrRequest = -32600
	paymeErrMethod  = -32601
	// The amount does not match the order.
	paymeErrAmount = -31001
	// No such transaction.
	paymeErrTxnNotFound = -31003
	// The order was delivered; the money cannot be given back automatically.
	paymeErrCannotCancel = -31007
	// The state machine forbids it (performing a cancelled transaction, say).
	paymeErrCannotPerform = -31008
	// The -31050..-31099 band is "the customer typed the account wrong". We use
	// the first of it for "no such order", which is exactly what it is: the
	// order number is the account.
	paymeErrOrderNotFound = -31050
	// The order exists but cannot be paid — cancelled, or already settled.
	paymeErrOrderState = -31051
)

// Payme cancels an unfinished transaction after 12 hours, with reason 4.
const paymeTimeoutMs int64 = 43_200_000

type paymeRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	ID     any             `json:"id"`
}

type paymeError struct {
	Code    int    `json:"code"`
	Message any    `json:"message"`
	Data    string `json:"data,omitempty"`
}

// paymeAccountField is the name of the account parameter configured in the
// Payme cabinet. Ours carries the order number.
func paymeAccountField(s *models.PaymentSettings) string {
	if f := strings.TrimSpace(s.Payme.AccountField); f != "" {
		return f
	}
	return "order_id"
}

// PaymeCallback is the single endpoint Payme talks to.
func (h *Handler) PaymeCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := h.paymentSettings(ctx)

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		paymeFail(w, nil, paymeErrParse, "Cannot read request", "")
		return
	}
	var req paymeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		paymeFail(w, nil, paymeErrParse, "Parse error", "")
		return
	}

	// Auth first, and before anything is looked up: an unauthenticated caller
	// must not be able to learn whether an order number exists by watching
	// which error comes back.
	if !paymeAuthorised(r, s) {
		paymeFail(w, req.ID, paymeErrAuth, "Insufficient privilege", "")
		return
	}
	if !s.Payme.Enabled {
		paymeFail(w, req.ID, paymeErrAuth, "Payme is switched off", "")
		return
	}

	switch req.Method {
	case "CheckPerformTransaction":
		h.paymeCheckPerform(w, r, s, &req)
	case "CreateTransaction":
		h.paymeCreate(w, r, s, &req)
	case "PerformTransaction":
		h.paymePerform(w, r, &req)
	case "CancelTransaction":
		h.paymeCancel(w, r, &req)
	case "CheckTransaction":
		h.paymeCheck(w, r, &req)
	case "GetStatement":
		h.paymeStatement(w, r, &req)
	default:
		paymeFail(w, req.ID, paymeErrMethod, "Method not found", "")
	}
}

// paymeAuthorised checks the Basic credentials. The login is always "Paycom";
// the password is the merchant key for the mode currently selected.
func paymeAuthorised(r *http.Request, s *models.PaymentSettings) bool {
	key := s.PaymeKey()
	if key == "" {
		return false
	}
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
	if !ok || login != "Paycom" {
		return false
	}
	// Constant time: this endpoint is public and a byte-wise compare leaks how
	// much of a guessed key was right.
	return subtle.ConstantTimeCompare([]byte(password), []byte(key)) == 1
}

// paymeOrder resolves the order named in an account object and checks it can
// take this payment. Returns the Payme error code to answer with.
func (h *Handler) paymeOrder(
	r *http.Request, s *models.PaymentSettings, account map[string]any, amountTiyin int64,
) (*models.Order, int, string) {
	field := paymeAccountField(s)
	number, _ := account[field].(string)
	if number == "" {
		// Some cabinets are configured to send numbers rather than strings.
		if n, ok := account[field].(float64); ok {
			number = fmt.Sprintf("%.0f", n)
		}
	}
	if strings.TrimSpace(number) == "" {
		return nil, paymeErrOrderNotFound, field
	}
	order, err := h.orderByNumber(r.Context(), number)
	if err != nil {
		return nil, paymeErrOrderNotFound, field
	}
	if order.PaymentMethod != models.ProviderPayme {
		// Somebody chose cash and is being billed through Payme: refuse rather
		// than quietly convert the order.
		return nil, paymeErrOrderState, field
	}
	if err := payable(order); err != nil {
		return nil, paymeErrOrderState, field
	}
	if amountTiyin != int64(order.Total)*100 {
		return nil, paymeErrAmount, ""
	}
	return order, 0, ""
}

func (h *Handler) paymeCheckPerform(
	w http.ResponseWriter, r *http.Request, s *models.PaymentSettings, req *paymeRequest,
) {
	var p struct {
		Amount  int64          `json:"amount"`
		Account map[string]any `json:"account"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		paymeFail(w, req.ID, paymeErrRequest, "Invalid params", "")
		return
	}
	if _, code, field := h.paymeOrder(r, s, p.Account, p.Amount); code != 0 {
		paymeFailFor(w, req.ID, code, field)
		return
	}
	paymeOK(w, req.ID, map[string]any{"allow": true})
}

func (h *Handler) paymeCreate(
	w http.ResponseWriter, r *http.Request, s *models.PaymentSettings, req *paymeRequest,
) {
	var p struct {
		ID      string         `json:"id"`
		Time    int64          `json:"time"`
		Amount  int64          `json:"amount"`
		Account map[string]any `json:"account"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		paymeFail(w, req.ID, paymeErrRequest, "Invalid params", "")
		return
	}

	// A repeat of a create we have already answered: give the same answer.
	if existing, err := h.findTxn(r.Context(), models.ProviderPayme, p.ID); err == nil {
		if existing.State != models.TxnCreated {
			paymeFail(w, req.ID, paymeErrCannotPerform, "Transaction is not active", "")
			return
		}
		paymeOK(w, req.ID, map[string]any{
			"create_time": existing.CreateTimeMs,
			"transaction": existing.ID.Hex(),
			"state":       models.TxnCreated,
		})
		return
	}

	order, code, field := h.paymeOrder(r, s, p.Account, p.Amount)
	if code != 0 {
		paymeFailFor(w, req.ID, code, field)
		return
	}
	// One live transaction per order. Without this a guest who opens the Payme
	// page twice ends up with two transactions against one bill, and whichever
	// one is cancelled last decides the order's state.
	if h.paymeHasOtherActive(r, order, p.ID) {
		paymeFail(w, req.ID, paymeErrCannotPerform,
			"Another transaction is already pending for this order", "")
		return
	}

	txn, err := h.createTxn(r.Context(), models.ProviderPayme, p.ID, order, p.Time)
	if err != nil {
		paymeFail(w, req.ID, paymeErrCannotPerform, err.Error(), "")
		return
	}
	paymeOK(w, req.ID, map[string]any{
		"create_time": txn.CreateTimeMs,
		"transaction": txn.ID.Hex(),
		"state":       models.TxnCreated,
	})
}

// paymeHasOtherActive reports whether the order already has a different Payme
// transaction waiting to be performed.
func (h *Handler) paymeHasOtherActive(r *http.Request, order *models.Order, exceptID string) bool {
	n, err := h.Store.Payments.CountDocuments(r.Context(), bson.M{
		"orderId":       order.ID,
		"provider":      models.ProviderPayme,
		"state":         models.TxnCreated,
		"providerTxnId": bson.M{"$ne": exceptID},
	})
	return err == nil && n > 0
}

func (h *Handler) paymePerform(w http.ResponseWriter, r *http.Request, req *paymeRequest) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		paymeFail(w, req.ID, paymeErrRequest, "Invalid params", "")
		return
	}
	txn, err := h.findTxn(r.Context(), models.ProviderPayme, p.ID)
	if err != nil {
		paymeFail(w, req.ID, paymeErrTxnNotFound, "Transaction not found", "")
		return
	}
	// Already performed: answer with the stored result rather than an error.
	// Payme retries, and an error here would have it cancel a paid order.
	if txn.State == models.TxnPerformed {
		paymeOK(w, req.ID, map[string]any{
			"transaction":  txn.ID.Hex(),
			"perform_time": txn.PerformTimeMs,
			"state":        models.TxnPerformed,
		})
		return
	}
	if txn.State != models.TxnCreated {
		paymeFail(w, req.ID, paymeErrCannotPerform, "Transaction is cancelled", "")
		return
	}
	// Payme's own 12-hour rule: past it, the transaction is dead and must be
	// cancelled rather than performed.
	if time.Now().UnixMilli()-txn.CreateTimeMs > paymeTimeoutMs {
		_ = h.cancelTxn(r.Context(), txn, 4)
		paymeFail(w, req.ID, paymeErrCannotPerform, "Transaction timed out", "")
		return
	}
	if err := h.performTxn(r.Context(), txn); err != nil {
		paymeFail(w, req.ID, paymeErrCannotPerform, err.Error(), "")
		return
	}
	h.afterPaid(r, txn)
	paymeOK(w, req.ID, map[string]any{
		"transaction":  txn.ID.Hex(),
		"perform_time": txn.PerformTimeMs,
		"state":        models.TxnPerformed,
	})
}

func (h *Handler) paymeCancel(w http.ResponseWriter, r *http.Request, req *paymeRequest) {
	var p struct {
		ID     string `json:"id"`
		Reason int    `json:"reason"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		paymeFail(w, req.ID, paymeErrRequest, "Invalid params", "")
		return
	}
	txn, err := h.findTxn(r.Context(), models.ProviderPayme, p.ID)
	if err != nil {
		paymeFail(w, req.ID, paymeErrTxnNotFound, "Transaction not found", "")
		return
	}
	// A delivered order cannot be unwound automatically: the food is eaten.
	// -31007 is exactly this case, and it is the answer Payme expects so that
	// the refund goes to a human instead.
	if txn.State == models.TxnPerformed {
		var order models.Order
		if err := h.Store.Orders.FindOne(r.Context(),
			bson.M{"_id": txn.OrderID}).Decode(&order); err == nil {
			if order.Status == models.StatusDelivered {
				paymeFail(w, req.ID, paymeErrCannotCancel, "Order already delivered", "")
				return
			}
		}
	}
	if err := h.cancelTxn(r.Context(), txn, p.Reason); err != nil {
		paymeFail(w, req.ID, paymeErrCannotPerform, err.Error(), "")
		return
	}
	paymeOK(w, req.ID, map[string]any{
		"transaction": txn.ID.Hex(),
		"cancel_time": txn.CancelTimeMs,
		"state":       txn.State,
	})
}

func (h *Handler) paymeCheck(w http.ResponseWriter, r *http.Request, req *paymeRequest) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		paymeFail(w, req.ID, paymeErrRequest, "Invalid params", "")
		return
	}
	txn, err := h.findTxn(r.Context(), models.ProviderPayme, p.ID)
	if err != nil {
		paymeFail(w, req.ID, paymeErrTxnNotFound, "Transaction not found", "")
		return
	}
	paymeOK(w, req.ID, map[string]any{
		"create_time":  txn.CreateTimeMs,
		"perform_time": txn.PerformTimeMs,
		"cancel_time":  txn.CancelTimeMs,
		"transaction":  txn.ID.Hex(),
		"state":        txn.State,
		"reason":       paymeReason(txn),
	})
}

// paymeReason must be null rather than 0 when nothing went wrong — Payme reads
// 0 as a reason code of its own.
func paymeReason(txn *models.Payment) any {
	if txn.Reason == 0 {
		return nil
	}
	return txn.Reason
}

func (h *Handler) paymeStatement(w http.ResponseWriter, r *http.Request, req *paymeRequest) {
	var p struct {
		From int64 `json:"from"`
		To   int64 `json:"to"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		paymeFail(w, req.ID, paymeErrRequest, "Invalid params", "")
		return
	}
	cur, err := h.Store.Payments.Find(r.Context(), bson.M{
		"provider":     models.ProviderPayme,
		"createTimeMs": bson.M{"$gte": p.From, "$lte": p.To},
	}, options.Find().SetSort(bson.D{{Key: "createTimeMs", Value: 1}}))
	if err != nil {
		paymeFail(w, req.ID, paymeErrCannotPerform, err.Error(), "")
		return
	}
	var rows []models.Payment
	_ = cur.All(r.Context(), &rows)

	s := h.paymentSettings(r.Context())
	field := paymeAccountField(s)
	out := make([]map[string]any, 0, len(rows))
	for i := range rows {
		t := &rows[i]
		out = append(out, map[string]any{
			"id":           t.ProviderTxnID,
			"time":         t.CreateTimeMs,
			"amount":       int64(t.Amount) * 100,
			"account":      map[string]any{field: t.OrderNumber},
			"create_time":  t.CreateTimeMs,
			"perform_time": t.PerformTimeMs,
			"cancel_time":  t.CancelTimeMs,
			"transaction":  t.ID.Hex(),
			"state":        t.State,
			"reason":       paymeReason(t),
		})
	}
	paymeOK(w, req.ID, map[string]any{"transactions": out})
}

// ---- Responses ----

func paymeOK(w http.ResponseWriter, id any, result map[string]any) {
	writeJSON(w, map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

// paymeFail writes an error. Payme's own client shows `message` to the guest,
// so it carries all three languages the platform expects.
func paymeFail(w http.ResponseWriter, id any, code int, note, data string) {
	writeJSON(w, map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": paymeError{
			Code:    code,
			Message: paymeMessage(code, note),
			Data:    data,
		},
	})
}

// paymeFailFor is the account/amount variant: the -3105x band must name the
// field the guest got wrong, so Payme can point at it.
func paymeFailFor(w http.ResponseWriter, id any, code int, field string) {
	switch code {
	case paymeErrAmount:
		paymeFail(w, id, code, "", "")
	default:
		paymeFail(w, id, code, "", field)
	}
}

// paymeMessage is the localised text Payme shows the customer.
func paymeMessage(code int, note string) map[string]string {
	uz, ru, en := "Xatolik", "Ошибка", "Error"
	switch code {
	case paymeErrAmount:
		uz, ru, en = "Noto'g'ri summa", "Неверная сумма", "Incorrect amount"
	case paymeErrOrderNotFound:
		uz, ru, en = "Buyurtma topilmadi", "Заказ не найден", "Order not found"
	case paymeErrOrderState:
		uz = "Buyurtma to'lovga tayyor emas"
		ru = "Заказ не может быть оплачен"
		en = "Order cannot be paid"
	case paymeErrTxnNotFound:
		uz, ru, en = "Tranzaksiya topilmadi", "Транзакция не найдена", "Transaction not found"
	case paymeErrCannotCancel:
		uz = "Buyurtma yetkazilgan, bekor qilib bo'lmaydi"
		ru = "Заказ доставлен, отмена невозможна"
		en = "Order delivered, cannot cancel"
	case paymeErrAuth:
		uz, ru, en = "Ruxsat yo'q", "Недостаточно прав", "Insufficient privilege"
	case paymeErrCannotPerform:
		uz, ru, en = "Amalni bajarib bo'lmadi", "Невозможно выполнить операцию", "Unable to perform"
	}
	if note != "" {
		en = note
	}
	return map[string]string{"uz": uz, "ru": ru, "en": en}
}

// ---- The checkout link ----

// paymeCheckoutURL builds the hosted checkout the guest is sent to. Payme takes
// its parameters as a base64 of a semicolon-separated list.
func paymeCheckoutURL(s *models.PaymentSettings, order *models.Order, returnTo string) string {
	parts := []string{
		"m=" + s.Payme.MerchantID,
		fmt.Sprintf("ac.%s=%s", paymeAccountField(s), order.Number),
		fmt.Sprintf("a=%d", int64(order.Total)*100),
	}
	// Back to the order's own tracking page, like the other two providers: it
	// already says whether the money arrived, which is the only question the
	// guest has at that moment.
	if back := paymeReturnURL(returnTo, order.Number); back != "" {
		parts = append(parts, "c="+back)
	}
	host := "https://checkout.paycom.uz/"
	if s.Payme.TestMode {
		host = "https://test.paycom.uz/"
	}
	return host + base64.StdEncoding.EncodeToString([]byte(strings.Join(parts, ";")))
}

// writeJSON is the raw writer the callbacks use: providers want HTTP 200 with
// the error in the body, not an HTTP error status.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}

// paymeReturnURL is the default place to send the guest back to: the order's
// own tracking page, which already says whether the money arrived.
func paymeReturnURL(base, number string) string {
	if base == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/order/" + url.PathEscape(number)
}
