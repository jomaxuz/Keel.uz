package fiscal

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// REGOS: VCR — the second adapter, and a much better-documented one.
//
// # What it is
//
// A virtual cash register with a **published API** (docs.regos.uz), which
// already makes it different from every other provider on the list. Like
// Multikassa it is a program with a printer attached — the documentation says
// every method except the Sys.* ones needs one — so it is treated as local and
// filed through the till screen or the relay. Unlike Multikassa it also offers
// a hosted sandbox at `vcr-test.regos.uz`, which means this adapter can be
// tested before a single restaurant signs anything.
//
// # The wire format
//
// **JSON-RPC 2.0**, one endpoint, method in the body. Authentication is
// `Base64(login:password)` in an `auth` **field of the payload** rather than a
// header — unusual, and the reason Creds carries a login and password here
// where Multikassa carries only an address.
//
// ⚠️ **`ok`, not the HTTP status.** The envelope is
// `{"id":1,"ok":true,"result":{…}}`, and a refusal comes back as `ok:false`
// with the reason in `result`. Reading the status code would call every
// business refusal a success.
//
// # ⚠️ Three different scalings in one request, and none of them is obvious
//
//  1. **Money is tiyin.** `amount: 900000` is 9 000 so'm.
//  2. **Quantity is thousandths.** `quantity: 1000` is *one* unit. A whole
//     number sent as-is would file a receipt for a thousandth of a portion —
//     and the total would still look right, because the total is sent
//     separately.
//  3. **VAT is percent × 100.** `vat_value: 1200` is 12%. Sending 12 files a
//     receipt at 0.12% tax.
//
// Each of these produces a document that passes every eyeball check and is
// wrong, which is why all three are converted in one place and sealed by tests.
const (
	regosSaleMethod    = "Receipt.Sale"
	regosZOpenMethod   = "ZReport.Open"
	regosZCloseMethod  = "ZReport.Close"
	regosSysInfoMethod = "Sys.GetInfo"

	// The single JSON-RPC endpoint. Everything is a POST to here.
	regosPath = "/"
)

// regosSandbox is the hosted test register.
//
// ⚠️ Used only when nothing is configured, and it is a **sandbox**: a
// restaurant whose base URL is empty files into a test register rather than
// failing, which sounds worse than it is — the alternative is a confusing
// transport error, and a receipt that lands in a sandbox is at least traceable
// to a setting nobody filled in. The panel refuses to *enable* the provider
// without an address anyway (fiscalEnableRefusal), so this is only reachable
// while somebody is experimenting.
const regosSandbox = "http://vcr-test.regos.uz"

type regos struct {
	auth string
	// Passed through to the receipt so the paper names the person, not the
	// integration.
	posID string
}

func newRegos(c Creds) (Encoder, error) {
	login := strings.TrimSpace(c.Login)
	pass := c.Password
	if login == "" || pass == "" {
		return nil, ErrNotConfigured
	}
	return &regos{
		auth:  base64.StdEncoding.EncodeToString([]byte(login + ":" + pass)),
		posID: strings.TrimSpace(c.RegisterID),
	}, nil
}

// ---- The envelope ----

type regosRequest struct {
	ID      int    `json:"id"`
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Auth    string `json:"auth"`
	Params  any    `json:"params,omitempty"`
}

type regosResponse struct {
	ID int `json:"id"`
	// ⚠️ The verdict. Not the HTTP status — a refusal arrives as 200 with
	// ok:false, and reading the status would call it a success.
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
}

type regosSaleResult struct {
	ID         string `json:"Id"`
	Amount     int64  `json:"Amount"`
	QRCodeURL  string `json:"QRCodeURL"`
	TerminalID string `json:"TerminalID"`
	ReceiptNo  string `json:"ReceiptNo"`
	DateTime   string `json:"DateTime"`
	FiscalSign string `json:"FiscalSign"`
}

// ---- Positions ----

type regosPosition struct {
	Name string `json:"name,omitempty"`
	// ⚠️ Line total in **tiyin**, not the unit price.
	Amount int64 `json:"amount"`
	// ⚠️ **Thousandths.** 1000 is one portion. See the note at the top.
	Quantity int64 `json:"quantity"`
	// ⚠️ **Percent × 100.** 1200 is 12%.
	VATValue int64 `json:"vat_value"`
	Discount int64 `json:"discount,omitempty"`

	// The state classifier and its packaging. Omitted when the restaurant has
	// not entered one — see MenuItem.Ikpu for why a placeholder is worse.
	ICPS        string `json:"icps,omitempty"`
	PackageCode string `json:"package_code,omitempty"`
	Barcode     string `json:"barcode,omitempty"`
	OwnerType   string `json:"owner_type,omitempty"`
}

// Payment types, from the documentation.
const (
	regosPayCash = 1
	regosPayCard = 2
)

type regosPayment struct {
	Type int `json:"type"`
	// In tiyin. Omitted for a payment created through Payment.Create, which
	// carries its own id instead — not a flow we use.
	Value int64 `json:"value,omitempty"`
}

type regosSaleParams struct {
	CashierName string          `json:"cashier_name,omitempty"`
	Code        string          `json:"code,omitempty"`
	POSID       string          `json:"pos_id,omitempty"`
	SessionCode string          `json:"session_code,omitempty"`
	Positions   []regosPosition `json:"positions"`
	Payments    []regosPayment  `json:"payments"`
}

// Sale builds the filing request.
func (g *regos) Sale(r Receipt) (Request, error) {
	if len(r.Items) == 0 {
		return Request{}, errors.New("fiskal chek bo'sh")
	}

	params := regosSaleParams{
		CashierName: r.Cashier,
		// ⚠️ Our order number goes in `code`, which REGOS uses for its
		// duplicate check when the setting is on. That is exactly what we want:
		// a retried filing after a lost reply is refused rather than filed
		// twice, by the register itself.
		Code:      r.OrderNumber,
		POSID:     g.posID,
		Positions: make([]regosPosition, 0, len(r.Items)),
	}

	for _, it := range r.Items {
		params.Positions = append(params.Positions, regosPosition{
			Name:   it.Name,
			Amount: it.Total(),
			// One portion is 1000. See the scaling note at the top of the file.
			Quantity:    int64(it.Qty) * 1000,
			VATValue:    int64(it.VATPercent) * 100,
			Discount:    it.Discount,
			ICPS:        it.SPIC,
			PackageCode: it.PackageCode,
			// Goods the restaurant bought and resells, which is what a dish is.
			OwnerType: "BuyingAndSelling",
		})
	}

	// ⚠️ Both halves are sent when both were taken. A guest paying part cash
	// and part card is ordinary in a hall, and collapsing it to one line would
	// file a receipt that disagrees with the drawer.
	if r.ReceivedCash > 0 {
		params.Payments = append(params.Payments,
			regosPayment{Type: regosPayCash, Value: r.ReceivedCash})
	}
	if r.ReceivedCard > 0 {
		params.Payments = append(params.Payments,
			regosPayment{Type: regosPayCard, Value: r.ReceivedCard})
	}
	if len(params.Payments) == 0 {
		// A zero-total sale still needs a payment line, or the register has
		// nothing to balance the positions against.
		params.Payments = append(params.Payments,
			regosPayment{Type: regosPayCash, Value: 0})
	}

	return g.call(regosSaleMethod, params)
}

// Hello asks the register to describe itself.
//
// ⚠️ Sys.GetInfo is one of the three methods the documentation says works
// **without a printer attached** — which makes it the right connection check:
// it separates "cannot reach the register" from "the register is there and its
// printer is not".
func (g *regos) Hello() (Request, error) {
	return g.call(regosSysInfoMethod, nil)
}

// OpenShift starts the register's day.
func (g *regos) OpenShift(cashier string, _ time.Time) (Request, error) {
	return g.call(regosZOpenMethod, map[string]any{"cashier_name": cashier})
}

// CloseShift files the Z-report.
func (g *regos) CloseShift(cashier string, _ time.Time) (Request, error) {
	return g.call(regosZCloseMethod, map[string]any{"cashier_name": cashier})
}

func (g *regos) call(method string, params any) (Request, error) {
	body, err := json.Marshal(regosRequest{
		ID: 1, JSONRPC: "2.0", Method: method, Auth: g.auth, Params: params,
	})
	if err != nil {
		return Request{}, err
	}
	return Request{
		Method:  "POST",
		Path:    regosPath,
		Headers: map[string]string{"Content-Type": "application/json;charset=utf-8"},
		Body:    string(body),
	}, nil
}

// Parse reads a reply.
//
// ⚠️ **`ok` decides, not the status code**, and a filing counts only if a
// fiscal sign came back — the same two rules as Multikassa, for the same
// reasons: business refusals arrive inside a 200, and a cheerful envelope with
// no sign in it is a receipt that was not filed.
func (g *regos) Parse(status int, body []byte) (Result, error) {
	var env regosResponse
	if err := json.Unmarshal(body, &env); err != nil {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 120 {
			snippet = snippet[:120] + "…"
		}
		if snippet == "" {
			return Result{}, fmt.Errorf("kassa javob bermadi (HTTP %d)", status)
		}
		return Result{}, fmt.Errorf("kassadan tushunarsiz javob (HTTP %d): %s", status, snippet)
	}

	if !env.OK {
		// ⚠️ The register's own words, whatever shape they came in: `result`
		// holds a description on failure, and it may be a string or an object.
		return Result{}, errors.New(regosError(env.Result))
	}

	var res regosSaleResult
	if len(env.Result) > 0 {
		_ = json.Unmarshal(env.Result, &res)
	}
	if res.FiscalSign != "" {
		return Result{
			ReceiptID:  res.ID,
			FiscalSign: res.FiscalSign,
			QRText:     res.QRCodeURL,
		}, nil
	}
	// A shift operation or Sys.GetInfo: no sign is expected and the caller
	// reads the description instead.
	if res.TerminalID != "" {
		return Result{ReceiptID: res.TerminalID}, nil
	}
	return Result{}, errors.New("kassa fiskal belgi qaytarmadi")
}

// regosError turns whatever came back into one readable sentence.
func regosError(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return "kassa amalni rad etdi"
	}
	// A bare string is the common case.
	var msg string
	if json.Unmarshal(raw, &msg) == nil && msg != "" {
		return msg
	}
	// Otherwise an object; the useful fields it might carry.
	var obj struct {
		Message string `json:"message"`
		Error   string `json:"error"`
		Text    string `json:"text"`
		Code    any    `json:"code"`
	}
	if json.Unmarshal(raw, &obj) == nil {
		for _, v := range []string{obj.Message, obj.Error, obj.Text} {
			if v != "" {
				if obj.Code != nil {
					return fmt.Sprintf("%v: %s", obj.Code, v)
				}
				return v
			}
		}
	}
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

// DescribeRegos turns a Sys.GetInfo reply into the connection check's line.
func DescribeRegos(body []byte) string {
	var env regosResponse
	if json.Unmarshal(body, &env) != nil || !env.OK {
		return ""
	}
	var info struct {
		TerminalID string `json:"TerminalID"`
		Version    string `json:"Version"`
		INN        string `json:"INN"`
		Name       string `json:"Name"`
	}
	if len(env.Result) == 0 || json.Unmarshal(env.Result, &info) != nil {
		return ""
	}
	parts := []string{}
	if info.Name != "" {
		parts = append(parts, info.Name)
	}
	if info.TerminalID != "" {
		parts = append(parts, "kassa "+info.TerminalID)
	}
	if info.Version != "" {
		parts = append(parts, "v"+info.Version)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " · ")
}
