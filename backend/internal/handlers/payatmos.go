package handlers

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ATMOS — the fourth provider, and the first one we have to *call* to get a
// checkout link.
//
// Payme, Click and Uzum all hand us a URL we can build from settings alone.
// ATMOS instead wants an invoice created server-to-server
// (`/checkout/invoice/create`), and answers with a `checkout.atmos.uz` address
// to send the guest to. Everything after that is the familiar shape: the guest
// pays on ATMOS's page, ATMOS calls this server, and only that call moves money
// in our ledger.
//
// ⚠️ **The hosted invoice, deliberately — not ATMOS's `/merchant/pay/*` API.**
// That one takes the card number and expiry directly, which would put the
// guest's PAN on the restaurant's own server and every tenant container in PCI
// DSS scope. See models.AtmosSettings.
//
// Two things about ATMOS's callback are unlike the other three:
//
//   - **It asks permission, it does not report.** "The amount will only be
//     deducted after receiving a successful status from the merchant" — so our
//     answer decides whether the guest is charged at all. A bug that returns
//     failure here does not lose a record; it declines a real payment at the
//     till.
//   - **The signature's hash function is not documented.** The docs give the
//     concatenation and call it "hashing". See atmosSignMatches.
const (
	atmosBaseURL     = "https://apigw.atmos.uz"
	atmosInvoiceTTL  = 30 // minutes an unpaid invoice stays open
	atmosHTTPTimeout = 20 * time.Second
)

// ---- Talking to ATMOS ----

func atmosBase(s *models.AtmosSettings) string {
	if b := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/"); b != "" {
		return b
	}
	return atmosBaseURL
}

// atmosToken is the OAuth2 client-credentials token, cached until it expires.
//
// Cached for the same reason the Eskiz bearer and the onlinePBX key are: the
// token lasts an hour, and minting a new one per checkout would spend a round
// trip — and an ATMOS rate limit — on every guest who reaches the payment step.
// Keyed by the credentials so saving new ones in the panel takes effect without
// a restart, and so a rotated secret can never keep working from cache.
var atmosTokens sync.Map // key -> *atmosToken

type atmosToken struct {
	mu      sync.Mutex
	value   string
	expires time.Time
}

func (h *Handler) atmosAccessToken(ctx context.Context, s *models.AtmosSettings) (string, error) {
	key := atmosBase(s) + "|" + s.ConsumerKey + "|" + s.ConsumerSecret
	v, _ := atmosTokens.LoadOrStore(key, &atmosToken{})
	t := v.(*atmosToken)

	t.mu.Lock()
	defer t.mu.Unlock()
	// A minute of headroom: a token that expires in flight reads as a random
	// authentication failure at the checkout, which is the hardest kind to
	// reproduce.
	if t.value != "" && time.Now().Before(t.expires.Add(-time.Minute)) {
		return t.value, nil
	}

	body := strings.NewReader("grant_type=client_credentials")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, atmosBase(s)+"/token", body)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(s.ConsumerKey, s.ConsumerSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := (&http.Client{Timeout: atmosHTTPTimeout}).Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("atmos token: %s", strings.TrimSpace(string(raw)))
	}
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || out.AccessToken == "" {
		return "", fmt.Errorf("atmos token: unexpected response")
	}
	if out.ExpiresIn <= 0 {
		out.ExpiresIn = 3600
	}
	t.value = out.AccessToken
	t.expires = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)
	return t.value, nil
}

// atmosInvoiceItem is one line on the ATMOS invoice.
//
// ATMOS requires the basket, not just a total: the invoice feeds the fiscal
// receipt (OFD), which itemises what was sold. `Code` is the ИКПУ — the state
// product classifier — which a restaurant gets from its accountant and which
// we do not have for a dish. It is sent when known and omitted otherwise
// rather than filled with a placeholder: a wrong ИКПУ is a wrong fiscal
// receipt, which is worse than an incomplete one.
type atmosInvoiceItem struct {
	ItemsID  string            `json:"items_id"`
	Code     string            `json:"code,omitempty"`
	Name     string            `json:"name"`
	Amount   int64             `json:"amount"`
	Quantity int               `json:"quantity,omitempty"`
	Details  []atmosItemDetail `json:"details"`
}

type atmosItemDetail struct {
	Name   string `json:"name"`
	Values string `json:"values"`
}

// atmosInvoice builds the basket ATMOS is asked to bill for.
//
// ⚠️ Line amounts are the **per-unit** price in tiyin and the total is the
// order's own total, taken from the order rather than re-summed here. Delivery
// and discounts do not appear as basket lines, so a sum of the lines would not
// equal the total — and the number that must be right is the one the guest is
// charged.
func atmosInvoiceItems(order *models.Order) []atmosInvoiceItem {
	items := make([]atmosInvoiceItem, 0, len(order.Items)+1)
	for i, it := range order.Items {
		items = append(items, atmosInvoiceItem{
			ItemsID:  strconv.Itoa(i + 1),
			Name:     it.Name,
			Amount:   int64(it.Price) * 100,
			Quantity: it.Qty,
			Details:  []atmosItemDetail{},
		})
	}
	// Delivery is a line on the receipt too — a guest comparing the fiscal
	// receipt with the total should not find an unexplained difference.
	if order.DeliveryFee > 0 {
		items = append(items, atmosInvoiceItem{
			ItemsID:  strconv.Itoa(len(items) + 1),
			Name:     "Yetkazib berish",
			Amount:   int64(order.DeliveryFee) * 100,
			Quantity: 1,
			Details:  []atmosItemDetail{},
		})
	}
	return items
}

// atmosCreateInvoice asks ATMOS for a checkout page for this order.
func (h *Handler) atmosCreateInvoice(
	ctx context.Context, s *models.AtmosSettings, order *models.Order, returnTo string,
) (string, error) {
	token, err := h.atmosAccessToken(ctx, s)
	if err != nil {
		return "", err
	}
	storeID, err := strconv.ParseInt(strings.TrimSpace(s.StoreID), 10, 64)
	if err != nil {
		return "", fmt.Errorf("atmos: store id is not a number")
	}

	payload := map[string]any{
		// ⚠️ The order id, not a fresh random value. ATMOS treats `request_id`
		// as the idempotency key, so a guest who presses "pay" twice — or whose
		// first request timed out after it succeeded — gets the same invoice
		// back instead of a second one against the same order.
		"request_id":      order.ID.Hex(),
		"store_id":        storeID,
		"account":         order.Number,
		"amount":          int64(order.Total) * 100, // tiyin
		"expiration_time": atmosInvoiceTTL,
		"success_url":     returnTo,
		"items":           atmosInvoiceItems(order),
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		atmosBase(s)+"/checkout/invoice/create", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	res, err := (&http.Client{Timeout: atmosHTTPTimeout}).Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))

	var out struct {
		URL    string `json:"url"`
		Status struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"status"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("atmos invoice: unexpected response")
	}
	if out.URL == "" {
		// The gateway's own words, which are the only useful ones here: a
		// missing store, a closed contract and a bad amount all arrive as 200.
		return "", fmt.Errorf("atmos invoice: %s %s", out.Status.Code, out.Status.Description)
	}
	return out.URL, nil
}

// ---- The callback ----

// atmosSignMatches checks ATMOS's signature over the callback.
//
// ⚠️ **The documentation gives the formula but never names the hash.** It says
// only that `sign` is "generated by hashing according to the formula:
// store_id+transaction_id+invoice+amount+api_key without separators". So all
// three plausible digests are accepted.
//
// That sounds like a weakening and is not: the string being hashed contains
// `api_key`, so somebody who cannot produce one digest cannot produce any of
// them. What it buys is that the first real payment succeeds instead of failing
// in a way that looks like "ATMOS is broken" — and a declined callback here is
// not a lost record, it is a guest not charged at the till.
//
// Compared in constant time, and the matching digest is logged so this can be
// narrowed to one once a real payment has been seen.
func atmosSignMatches(sign, payload string) (string, bool) {
	sign = strings.ToLower(strings.TrimSpace(sign))
	if sign == "" {
		return "", false
	}
	md := md5.Sum([]byte(payload))
	s1 := sha1.Sum([]byte(payload))
	s256 := sha256.Sum256([]byte(payload))
	for name, sum := range map[string][]byte{
		"md5":    md[:],
		"sha1":   s1[:],
		"sha256": s256[:],
	} {
		want := hex.EncodeToString(sum)
		if subtle.ConstantTimeCompare([]byte(sign), []byte(want)) == 1 {
			return name, true
		}
	}
	return "", false
}

// atmosScalar is a callback field that may arrive as a JSON number or as a
// JSON string, kept exactly as it was written.
//
// ⚠️ **The documentation does not say which.** It lists `store_id`,
// `transaction_id` and `amount` with values like "merchant ID" and "amount" and
// never gives their JSON types, while the signature is a concatenation of them
// as text. Declaring them as numbers refuses every callback that quotes them,
// declaring them as strings refuses every callback that does not — and either
// way the failure is total and looks identical from outside: the guest reaches
// the payment page, pays, and is told it failed.
//
// The raw text is what matters anyway, because that is what the signature is
// over: reformatting "100000.00" to "100000" breaks a signature that was good.
type atmosScalar struct{ raw string }

func (a *atmosScalar) UnmarshalJSON(b []byte) error {
	text := string(b)
	if len(text) > 1 && text[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		a.raw = s
		return nil
	}
	a.raw = text
	return nil
}

func (a atmosScalar) String() string { return a.raw }

// atmosSignPayload is the string ATMOS hashes, in its documented order.
func atmosSignPayload(storeID, txnID, invoice, amount, apiKey string) string {
	return storeID + txnID + invoice + amount + apiKey
}

// atmosReply is the answer ATMOS requires. `status: 1` **charges the guest**;
// anything else declines the payment.
func atmosReply(w http.ResponseWriter, ok bool, message string) {
	status := 0
	if ok {
		status = 1
	}
	// Always HTTP 200: ATMOS treats a non-200 as a failure to reach us and
	// retries, and a retry storm helps nobody. The decision lives in the body.
	httpx.JSON(w, http.StatusOK, map[string]any{"status": status, "message": message})
}

// AtmosCallback is ATMOS asking whether to charge the guest for an order.
//
// ⚠️ **This is a permission, not a notification**, and that inverts the usual
// risk. With Payme or Click a mistaken failure means a payment we recorded
// badly; here it means a guest standing at the checkout being told their card
// was declined. So every refusal below is a case where charging would be
// worse: an order that does not exist, one already paid, one cancelled, or an
// amount that disagrees with ours.
func (h *Handler) AtmosCallback(w http.ResponseWriter, r *http.Request) {
	s := h.paymentSettings(r.Context())
	if !s.Configured(models.ProviderAtmos) {
		atmosReply(w, false, "atmos is not configured")
		return
	}

	var req struct {
		StoreID       atmosScalar `json:"store_id"`
		TransactionID atmosScalar `json:"transaction_id"`
		Amount        atmosScalar `json:"amount"`
		Invoice       string      `json:"invoice"`
		Sign          string      `json:"sign"`
	}
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err := json.Unmarshal(raw, &req); err != nil {
		atmosReply(w, false, "bad request")
		return
	}

	// ⚠️ Signed over the values **exactly as they arrived on the wire**, not
	// over reformatted ones. `atmosScalar` keeps "100000" as it was written;
	// decoding into an int and printing it back would turn a float-shaped
	// "100000.00" into "100000" and break a signature that was perfectly good.
	payload := atmosSignPayload(
		req.StoreID.String(), req.TransactionID.String(),
		req.Invoice, req.Amount.String(), s.Atmos.APIKey)
	algo, ok := atmosSignMatches(req.Sign, payload)
	if !ok {
		// Refused without saying which part was wrong.
		atmosReply(w, false, "bad signature")
		return
	}

	order, err := h.orderByNumber(r.Context(), req.Invoice)
	if err != nil {
		atmosReply(w, false, "invoice "+req.Invoice+" is not available in the system")
		return
	}
	if err := payable(order); err != nil {
		atmosReply(w, false, err.Error())
		return
	}

	// The amount is the order's, never the callback's. A callback claiming
	// 1 000 for a 100 000 order is refused rather than recorded.
	amount, err := strconv.ParseInt(req.Amount.String(), 10, 64)
	if err != nil || amount != int64(order.Total)*100 {
		atmosReply(w, false, "amount does not match the order")
		return
	}

	txn, err := h.createTxn(r.Context(), models.ProviderAtmos,
		req.TransactionID.String(), order, 0)
	if err != nil {
		atmosReply(w, false, "internal error")
		return
	}
	if err := h.performTxn(r.Context(), txn); err != nil {
		atmosReply(w, false, err.Error())
		return
	}
	h.afterPaid(r, txn)
	log.Printf("atmos: callback signature verified with %s", algo)
	atmosReply(w, true, "Successfully")
}
