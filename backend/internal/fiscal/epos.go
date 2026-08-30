package fiscal

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// E-POS Mobile — the third adapter, and the first one that is a phone.
//
// # What it is
//
// E-POS sell three things (docs.epos.uz). Only one of them files receipts for
// somebody else's software:
//
//   - **E-POS Mobile** — an Android app that turns a phone into a register,
//     with a *local* HTTP API on that phone at `http://localhost:8765`. This is
//     the one implemented here.
//   - **Fiscal Bridge** — their own way of putting an existing till program on
//     the E-POS platform. That is a migration for a restaurant already running
//     somebody else's POS, not an integration point for ours.
//   - **Cashdesk** — a browser workstation, not released.
//
// # Local, and more literally so than the others
//
// Multikassa and REGOS are programs on a PC in the back office; this one is an
// app on a phone, which is the same transport problem in a smaller box. The
// address in the drawer is that phone's address on the restaurant's Wi-Fi
// (`http://192.168.x.x:8765`) — the documentation's `localhost` is written from
// the phone's own point of view, and is the one address that will never work
// for us.
//
// ⚠️ **And the phone has to stay on the same network as the till screen.** A
// register that is also somebody's telephone will leave the building; when it
// does, filing stops and the sales queue as pending. That is visible and
// recoverable, and it is the reason the note in the provider list says so out
// loud rather than leaving an owner to discover it on a Friday evening.
//
// # The wire format
//
// Plain JSON over `POST /receipts`, with the document type in the body:
// `sale`, `refund`, `advance` or `credit`. Authentication is a token in an
// `X-API-Key` header, copied out of the app under Profil → Lokal server.
//
// ⚠️ **`success`, not the HTTP status.** A refusal comes back as
// `{"success": false, "error": {"code": "E-011", "rule": …, "message": …}}`,
// and the codes are grouped by prefix: `E-` business and validation, `F-` the
// fiscal module, `S-` their API server, `T-` the OFD. Only the first two are
// about the receipt.
//
// # ⚠️ Two scalings, both silent
//
//  1. **Money is tiyin.** `price: 800000` is 8 000 so'm.
//  2. **Quantity is thousandths.** `amount: 1000` is *one* portion — the same
//     encoding REGOS uses, under a field name that reads like a sum of money.
//     A whole number sent as-is files a receipt for a thousandth of a dish,
//     and the money on it still adds up.
//
// # ⚠️ A filed receipt with `ofdSent: false` is still a filed receipt
//
// This one is specific to E-POS and it is the trap worth knowing. If their OFD
// servers are unreachable, the app **still issues the receipt** — the guest has
// their paper, the fiscal module has assigned a sign, and the answer carries it
// as usual, with `ofdSent: false` and an `ofdError` beside it. Only the onward
// delivery is outstanding, and the app resends it itself through
// `/receipts/send-unsent`.
//
// Treating that as a failure would be actively harmful: our own retry would
// file the sale a **second time**, and a duplicate fiscal receipt is a sale the
// restaurant is taxed on twice and has to unwind with paperwork. So the sign
// decides, exactly as it does for the other two providers.
const (
	eposReceipts = "/receipts"
	eposStatus   = "/status"
	eposZOpen    = "/z-report/open"
	eposZClose   = "/z-report/close"

	eposTypeSale   = "sale"
	eposTypeRefund = "refund"

	// The register's day is not open. Their F- table, and the reason this
	// provider implements ShiftOpener.
	eposShiftClosed = "F-002"
	// A refund arrived without the sale it reverses. We refuse to build one of
	// those before it leaves here, so this code should never come back — it is
	// listed because it is the thing that would come back if that guard ever
	// broke.
	eposRefundInfoRequired = "E-007"
)

type epos struct {
	token string
}

func newEPOS(c Creds) (Encoder, error) {
	token := strings.TrimSpace(c.Token)
	if token == "" {
		return nil, ErrNotConfigured
	}
	return &epos{token: token}, nil
}

// ---- The document ----

type eposItem struct {
	Name string `json:"name"`
	// The state classifier and its packaging, omitted when the restaurant has
	// not entered one — see MenuItem.Ikpu. E-013 and E-014 refuse a code that
	// is not in Tasnif, which is the good failure: it is refused here rather
	// than accepted onto a state document.
	SPIC        string `json:"spic,omitempty"`
	PackageCode string `json:"packageCode,omitempty"`
	// ⚠️ **The one field whose code space is not settled.** The documentation
	// calls it OKEI; the worked example shows `1372873`, which is neither an
	// OKEI code nor the small code the state receipt format uses (0 piece,
	// 10 gram, 11 kilogram, 41 litre) and which is what our Item carries. No
	// validation rule is attached to it, so a mismatch does not refuse the
	// receipt — it prints the wrong unit name beside a correct price. Our value
	// is passed through unchanged, and correcting it is one line here.
	Units int `json:"units"`
	// Unit price in tiyin, before this line's discount.
	Price int64 `json:"price"`
	// ⚠️ Quantity in **thousandths**: 1000 is one. See the note at the top.
	Amount     int64 `json:"amount"`
	VatPercent int   `json:"vatPercent"`
	// VAT contained in this line, in tiyin.
	//
	// ⚠️ **The line's, not one unit's**, and the documentation is genuinely
	// ambiguous here: rule E-016 spells the check as
	// `price × vatPercent / (100 + vatPercent) ±100`, which is written against
	// the *unit* price, while rule E-010 requires the receipt's `totalVAT` to
	// equal the sum of these — which only holds if each is the whole line's.
	// The worked example has a quantity of one and so cannot separate them.
	//
	// The line's VAT is sent because it is the one reading that produces a
	// correct tax document, and because a wrong choice here fails loudly and
	// immediately: E-016 is refused in front of the cashier on the first
	// receipt with two of anything on it, not discovered at an inspection.
	Vat      int64 `json:"vat"`
	Discount int64 `json:"discount,omitempty"`
	// The marking code (Asl Belgisi) scanned off this item. E-004 refuses a
	// markable product sent without one, which is the whole point of carrying
	// it: the code travels inside the receipt and the OFD withdraws the bottle
	// from circulation.
	Label string `json:"label,omitempty"`
}

type eposReceived struct {
	Cash int64 `json:"cash"`
	Card int64 `json:"card"`
}

// eposRefundInfo names the sale a reversal undoes.
//
// ⚠️ **Mandatory** — E-007 refuses a refund without it. Which is the correct
// behaviour and the one every register should have: a reversal that does not
// name its sale is a standalone negative receipt, and the state's copy is then
// left holding a return against nothing.
type eposRefundInfo struct {
	TerminalID string `json:"terminalId"`
	ReceiptSeq int64  `json:"receiptSeq"`
	// ⚠️ `YYYYMMDDTHHmmss` — with a literal `T`, and unlike Multikassa's
	// `RefundInfo`, which spells the same instant with no separator at all.
	// Two providers, two spellings of one timestamp, both undocumented anywhere
	// near the other fields.
	DateTime   string `json:"dateTime"`
	FiscalSign string `json:"fiscalSign"`
}

type eposRequest struct {
	Type       string          `json:"type"`
	Items      []eposItem      `json:"items"`
	Received   eposReceived    `json:"received"`
	RefundInfo *eposRefundInfo `json:"refundInfo,omitempty"`
}

// Sale builds the filing request, for a sale or a reversal.
func (e *epos) Sale(r Receipt) (Request, error) {
	if len(r.Items) == 0 {
		return Request{}, errors.New("fiskal chek bo'sh")
	}

	body := eposRequest{
		Type:  eposTypeSale,
		Items: make([]eposItem, 0, len(r.Items)),
		Received: eposReceived{
			Cash: r.ReceivedCash,
			Card: r.ReceivedCard,
		},
	}

	if r.IsRefund {
		// ⚠️ Refused here rather than at the register. The register's refusal
		// arrives while a guest is waiting for their money; ours arrives in the
		// unfiled list, where somebody can look at why the sale has no sign.
		if !r.Original.Known() {
			return Request{}, errors.New(
				"qaytarish uchun asl chekning fiskal belgisi yo'q")
		}
		body.Type = eposTypeRefund
		body.RefundInfo = &eposRefundInfo{
			TerminalID: r.Original.TerminalID,
			ReceiptSeq: eposSeq(r.Original.Seq),
			DateTime:   r.Original.At.Format("20060102T150405"),
			FiscalSign: r.Original.Sign,
		}
	}

	for _, it := range r.Items {
		body.Items = append(body.Items, eposItem{
			Name:        it.Name,
			SPIC:        it.SPIC,
			PackageCode: it.PackageCode,
			Units:       it.Units,
			Price:       it.Price,
			// One portion is 1000. See the scaling note at the top.
			Amount:     int64(it.Qty) * 1000,
			VatPercent: it.VATPercent,
			Vat:        it.VAT,
			Discount:   it.Discount,
			Label:      it.MarkCode,
		})
	}

	return e.post(eposReceipts, body)
}

// Hello asks the phone to describe itself.
//
// ⚠️ `/status` rather than `/business` or `/branch`, which also answer: those
// return the taxpayer's registration details, which is a lot of somebody's data
// to pull across for a question that is only "is the register there and is its
// fiscal module up". `/status` answers exactly that — and it is also the one
// endpoint that does not carry the subscription headers, so it keeps answering
// when the subscription has lapsed and that is what somebody needs to see.
func (e *epos) Hello() (Request, error) {
	return Request{Method: "GET", Path: eposStatus, Headers: e.headers()}, nil
}

// OpenShift starts the register's day.
//
// ⚠️ No cashier and no time: this register takes neither. The cashier is
// whoever is signed into the app, which is a fact about the phone rather than
// about our staff record — so the name on the paper is theirs, not ours, and
// nothing here can change that.
func (e *epos) OpenShift(_ string, _ time.Time) (Request, error) {
	return e.post(eposZOpen, nil)
}

// CloseShift files the Z-report, which the app sends to the OFD itself.
func (e *epos) CloseShift(_ string, _ time.Time) (Request, error) {
	return e.post(eposZClose, nil)
}

func (e *epos) post(path string, body any) (Request, error) {
	req := Request{Method: "POST", Path: path, Headers: e.headers()}
	if body == nil {
		// ⚠️ `{}` rather than nothing: S-003 refuses a request with no body,
		// and both shift calls take no parameters — which is exactly the shape
		// somebody would send empty.
		req.Body = "{}"
		return req, nil
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return Request{}, err
	}
	req.Body = string(raw)
	return req, nil
}

func (e *epos) headers() map[string]string {
	return map[string]string{
		"Content-Type": "application/json;charset=utf-8",
		"X-API-Key":    e.token,
	}
}

// ---- The answer ----

type eposReceiptOut struct {
	TerminalID string `json:"terminalId"`
	ReceiptSeq int64  `json:"receiptSeq"`
	FiscalSign string `json:"fiscalSign"`
	QRUrl      string `json:"qrUrl"`
}

type eposResponse struct {
	Success bool            `json:"success"`
	Receipt *eposReceiptOut `json:"receipt"`
	// Whether the receipt reached the OFD. ⚠️ Not part of the verdict — see the
	// note at the top of the file.
	OfdSent  bool   `json:"ofdSent"`
	OfdError string `json:"ofdError"`
	Error    *struct {
		Code    string `json:"code"`
		Rule    string `json:"rule"`
		Message string `json:"message"`
	} `json:"error"`
	// Present on a Z-report close.
	ZReport *struct {
		Number    int64  `json:"number"`
		CloseTime string `json:"closeTime"`
	} `json:"zReport"`
}

// Parse reads a reply.
//
// ⚠️ **`success` decides, not the status code**, and a filing counts only if a
// fiscal sign came back — the same two rules as the other two adapters, for the
// same reasons. The third rule is this provider's own: `ofdSent: false` is
// **not** a failure. The receipt exists and is signed; only its onward delivery
// is outstanding, and the app retries that itself.
func (e *epos) Parse(status int, body []byte) (Result, error) {
	var env eposResponse
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

	if !env.Success {
		return Result{}, errors.New(eposError(env))
	}

	if env.Receipt != nil && env.Receipt.FiscalSign != "" {
		return Result{
			// This register has no id for the sale beyond its sequence number
			// in the fiscal module, which is also what a reversal names it by.
			ReceiptID:  strconv.FormatInt(env.Receipt.ReceiptSeq, 10),
			FiscalSign: env.Receipt.FiscalSign,
			QRText:     env.Receipt.QRUrl,
		}, nil
	}

	// A shift operation: `{"success": true}`, with no receipt and no sign, and
	// the caller is not looking for one.
	if env.Receipt == nil {
		return Result{}, nil
	}
	return Result{}, errors.New("kassa fiskal belgi qaytarmadi")
}

// eposError turns a refusal into one readable sentence.
//
// ⚠️ **The code travels with the message.** The message arrives in Russian and
// is the vendor's to reword at any release; the code beside it is what their
// own error table is written in, and it is what a support conversation will be
// about.
func eposError(env eposResponse) string {
	if env.Error == nil {
		return "kassa amalni rad etdi"
	}
	msg := strings.TrimSpace(env.Error.Message)
	if msg == "" {
		msg = strings.TrimSpace(env.Error.Rule)
	}
	code := strings.TrimSpace(env.Error.Code)
	switch {
	case msg == "" && code == "":
		return "kassa amalni rad etdi"
	case msg == "":
		return code
	case code == "":
		return msg
	}
	return code + ": " + msg
}

// eposNeedsShift reports whether a refusal means "the day has not been opened".
//
// ⚠️ Matched on the code rather than the message text, for the reason spelled
// out on Multikassa's `needsShift`: the prose is translated and reworded, the
// code is not.
func eposNeedsShift(err error) bool {
	return err != nil && strings.Contains(err.Error(), eposShiftClosed)
}

// eposSeq turns a stored receipt number into the integer this API wants.
//
// ⚠️ It is a **number** here and a string in our record, because our record has
// to hold whatever any provider calls a receipt — REGOS answers with a UUID.
// An unparseable value becomes 0, which the register refuses as a receipt that
// does not exist; that is the correct outcome, and better than a reversal
// pointed at receipt number one.
func eposSeq(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// DescribeEPOS turns a /status reply into the connection check's line.
func DescribeEPOS(body []byte) string {
	var info struct {
		App           string `json:"app"`
		Version       string `json:"version"`
		FMInitialized bool   `json:"fmInitialized"`
		Activated     bool   `json:"activated"`
		Subscription  string `json:"subscriptionState"`
	}
	if json.Unmarshal(body, &info) != nil || info.App == "" {
		return ""
	}
	parts := []string{info.App}
	if info.Version != "" {
		parts = append(parts, "v"+info.Version)
	}
	// ⚠️ The two things that make a reachable register useless are said out
	// loud. "Connected" beside a register that will refuse every sale is the
	// answer that sends somebody looking in the wrong place — the same lesson
	// the POS package's Ping learned.
	if !info.Activated {
		parts = append(parts, "kassa faollashtirilmagan")
	}
	if !info.FMInitialized {
		parts = append(parts, "fiskal modul ishga tushmagan")
	}
	if info.Subscription == "expired" {
		parts = append(parts, "obuna tugagan")
	}
	return strings.Join(parts, " · ")
}
