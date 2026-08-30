package fiscal

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Multikassa (sold as Rahmat POS) — the first adapter, and the one that shaped
// the Encoder/Client split above.
//
// # What it actually is
//
// Not a cloud service. A program on the cash register machine inside the
// restaurant, serving one JSON endpoint on the local network:
//
//	POST http://<the register's LAN address>/api/v1/operations
//
// There is **no authentication of any kind** — no token, no login, no signed
// body. That is not an oversight to work around; it is the security model. The
// endpoint is protected by being unroutable from outside the building, which is
// also precisely why our server cannot use it. See Info.Local.
//
// One endpoint carries every operation, chosen by module_operation_type:
// 1 open shift · 2 close shift (Z-report) · 3 sale · 4 refund · 7 X-report ·
// 8 advance · 10 credit.
//
// # ⚠️ The two sources disagree, and one of the disagreements is 100×
//
// We have the vendor's integrator PDF (April 2026) and their public Postman
// collection, and they do not say the same thing:
//
//   - **Money.** The PDF states receipt_sum and the two received_* fields are in
//     **tiyin**, while everything inside items[] is in **so'm**. The Postman
//     example is arithmetically self-consistent in a *single* unit (70000 +
//     70000 + 150000, less 40% of the first, equals its receipt_sum of 262000).
//     Their own responses are mixed the same way — the zReport endpoint answers
//     in tiyin integers while close-shift answers in formatted so'm — which is
//     what makes the PDF's claim credible rather than a typo.
//
//   - **Naming.** The PDF's item table has `ikpu` and `packageCode`; the Postman
//     example sends `classifier_class_code` and `product_package`. The PDF lists
//     all of them.
//
//   - **Discount.** The PDF has `product_discount`, an absolute amount. The
//     Postman example has `discount_percent`.
//
// The naming one is settled cheaply and safely: ИКПУ and the classifier code
// are the **same number**, so both spellings are sent carrying the same value
// and no reading of the documentation can be contradicted. An unknown field is
// ignored by a JSON decoder; a missing one is a rejected receipt.
//
// The discount one is not settled that way — sending both risks the amount
// being taken off twice — so the PDF wins and only `product_discount` goes out.
//
// ⚠️ **The money one cannot be settled from here at all**, and it is the
// dangerous one: being wrong by a factor of a hundred on a filed tax document
// is not a rounding bug. The PDF is followed, the conversion happens in exactly
// one place (tiyinToSum), and it is sealed by a test — so when the vendor
// confirms it on a real terminal, the fix is one function and the test says
// what changed.

// mkOperation values, from the PDF's table.
const (
	mkOpenShift  = "1"
	mkCloseShift = "2"
	mkSale       = "3"
	mkRefund     = "4"
)

const (
	mkOperations = "/api/v1/operations"
	mkInfo       = "/api/v1/info"
)

type multikassa struct {
	// The fiscal module's terminal id, as printed in the provider's cabinet.
	// Optional: a register serving one shop knows its own id, and the PDF's
	// examples omit it as often as they include it.
	terminal string
}

func newMultikassa(c Creds) (Encoder, error) {
	return &multikassa{terminal: strings.TrimSpace(c.RegisterID)}, nil
}

// mkSum is an amount in tiyin that serialises as so'm.
//
// ⚠️ **Formatted as a decimal literal rather than converted to float64.** The
// discount spread in Build is exact to the tiyin, so a line's share can be
// 3333 tiyin — and 33.33 has no exact float64 representation, which is how a
// receipt ends up carrying 33.329999999999998 to a tax authority. Writing the
// digits directly keeps the exactness the builder worked for.
type mkSum int64

func (s mkSum) MarshalJSON() ([]byte, error) {
	return []byte(tiyinToSum(int64(s))), nil
}

// tiyinToSum renders tiyin as a so'm decimal, always with two places.
//
// The single place the unit boundary is crossed — see the money warning above.
func tiyinToSum(t int64) string {
	neg := t < 0
	if neg {
		t = -t
	}
	out := fmt.Sprintf("%d.%02d", t/100, t%100)
	if neg {
		return "-" + out
	}
	return out
}

type mkItem struct {
	Name string `json:"product_name"`
	// Unit price, in so'm.
	Price mkSum `json:"product_price"`
	// The line's total after its share of the discount, in so'm. Sent because
	// the PDF asks for it; the register is not left to re-derive a number the
	// builder already computed exactly.
	Total    mkSum   `json:"total_product_price"`
	Discount mkSum   `json:"product_discount"`
	Count    float64 `json:"count"`

	VATPercent int  `json:"product_vat_percent"`
	WithoutVAT bool `json:"product_without_vat"`

	// The classifier code under both names it goes by. Same value, so the two
	// spellings cannot disagree — see the naming note above.
	IKPU       string `json:"ikpu,omitempty"`
	Classifier string `json:"classifier_class_code,omitempty"`
	// Likewise for packaging.
	PackageCode string `json:"packageCode,omitempty"`
	Package     string `json:"product_package,omitempty"`
	PackageName string `json:"product_package_name,omitempty"`

	// The marking code, in the field the tax committee's own receipt format
	// calls `label` — the OFD forwards it to the national system from there
	// (docs/markirovka.md).
	//
	// ⚠️ **`omitempty`, for the reason the ИКПУ is omitted when empty**: an
	// unmarked dish must send no field at all, not an empty string. A register
	// that reads "" as "this product is marked and has no code" refuses the
	// receipt, and the refusal happens in front of a guest.
	//
	// ⚠️ **The name is taken from the state format and has not been confirmed
	// against this provider's own documentation.** It is spelled once, here, so
	// there is one line to correct rather than a search — and nothing else in
	// the pipeline has to change if it turns out to be spelled differently.
	Label string `json:"label,omitempty"`
}

type mkRequest struct {
	Type string `json:"module_operation_type"`

	// ⚠️ These three are in **tiyin**, unlike everything in Items.
	Sum          int64 `json:"receipt_sum"`
	ReceivedCash int64 `json:"receipt_gnk_receivedcash"`
	ReceivedCard int64 `json:"receipt_gnk_receivedcard"`

	Cashier  string `json:"receipt_cashier_name,omitempty"`
	Time     string `json:"receipt_gnk_time,omitempty"`
	Terminal string `json:"receipt_gnk_terminalid,omitempty"`
	Module   string `json:"module_gnk_terminalid,omitempty"`

	// ⚠️ **false, deliberately.** The register prints its own paper roll when
	// asked; our till screen is a tablet with no printer attached to the fiscal
	// module, and a filing that blocks on a print dialogue at somebody else's
	// machine is a filing that hangs with a guest at the counter. The guest's
	// copy is the QR we get back.
	ForceToPrint bool `json:"force_to_print"`
	PayFromCard  bool `json:"pay_from_card"`

	// ---- Refund only ----
	//
	// ⚠️ **Without these a refund is not a refund.** The vendor's integrator PDF
	// spells it out: type 4 additionally carries `receipt_sale_id` and a
	// `RefundInfo` block, and the block is what gets proxied on to the fiscal
	// drive. Sent without them the register either refuses the operation or
	// files a standalone negative sale — which balances our books and leaves
	// the state's copy with a refund against nothing.
	//
	// The Postman collection sends the same three facts as flat
	// `receipt_gnk_*` fields. Both spellings go out, carrying identical values,
	// for the reason `ikpu`/`classifier_class_code` do: a decoder ignores a
	// field it does not know, and there is no reading of the two documents in
	// which one of these is wrong.
	SaleID     string        `json:"receipt_sale_id,omitempty"`
	RefundInfo *mkRefundInfo `json:"RefundInfo,omitempty"`
	RefundSeq  string        `json:"receipt_gnk_receiptseq,omitempty"`
	RefundSign string        `json:"receipt_gnk_fiscalsign,omitempty"`

	Items []mkItem `json:"items"`
}

// mkRefundInfo names the sale being reversed.
//
// ⚠️ The field names are capitalised exactly as the PDF prints them —
// `TerminalID`, `ReceiptSeq`, `DateTime`, `FiscalSign`. Everything else in this
// API is snake_case, which is precisely why this block is easy to get wrong and
// why it is written out rather than derived from a struct tag convention.
type mkRefundInfo struct {
	TerminalID string `json:"TerminalID,omitempty"`
	ReceiptSeq string `json:"ReceiptSeq,omitempty"`
	// ⚠️ `YYYYMMDDHHMMSS`, not the `2006-01-02 15:04:05` every other timestamp
	// in this API uses. The PDF says so and the Postman example confirms it
	// ("20241106183614").
	DateTime      string `json:"DateTime,omitempty"`
	FiscalSign    string `json:"FiscalSign,omitempty"`
	ReceiptSaleID string `json:"ReceiptSaleId,omitempty"`
}

// Sale builds the filing request.
func (m *multikassa) Sale(r Receipt) (Request, error) {
	if len(r.Items) == 0 {
		// A receipt with no goods is not a document the state has a shape for,
		// and the register would reject it after the guest had already paid.
		return Request{}, errors.New("fiskal chek bo'sh")
	}

	body := mkRequest{
		Type:         mkSale,
		ReceivedCash: r.ReceivedCash,
		ReceivedCard: r.ReceivedCard,
		Cashier:      r.Cashier,
		Time:         r.Time.Format("2006-01-02 15:04:05"),
		Terminal:     m.terminal,
		Module:       m.terminal,
		PayFromCard:  r.ReceivedCard > 0 && r.ReceivedCash == 0,
		Items:        make([]mkItem, 0, len(r.Items)),
	}
	if r.IsRefund {
		body.Type = mkRefund
		o := r.Original
		if !o.Known() {
			// ⚠️ Refused here rather than filed as something else. A refund the
			// register accepts as a standalone negative sale is worse than one
			// it rejects: the rejection is visible on the till screen with a
			// guest still standing there, and the mis-filing is visible to
			// nobody until an inspection.
			return Request{}, errors.New(
				"qaytarish uchun asl chekning fiskal belgisi kerak")
		}
		terminal := o.TerminalID
		if terminal == "" {
			terminal = m.terminal
		}
		at := o.At.Format("20060102150405")
		body.SaleID = o.SaleID
		body.RefundSeq = o.Seq
		body.RefundSign = o.Sign
		body.RefundInfo = &mkRefundInfo{
			TerminalID: terminal, ReceiptSeq: o.Seq,
			DateTime: at, FiscalSign: o.Sign, ReceiptSaleID: o.SaleID,
		}
	}
	for _, it := range r.Items {
		body.Sum += it.Total()
		body.Items = append(body.Items, mkItem{
			Name:     it.Name,
			Price:    mkSum(it.Price),
			Total:    mkSum(it.Total()),
			Discount: mkSum(it.Discount),
			Count:    float64(it.Qty),
			// ⚠️ A zero rate has to be *declared*, not merely left at zero:
			// "exempt" and "nobody filled the rate in" produce the same number
			// and mean opposite things to an inspector. See VatPercent.
			VATPercent:  it.VATPercent,
			WithoutVAT:  it.VATPercent == 0,
			Label:       it.MarkCode,
			IKPU:        it.SPIC,
			Classifier:  it.SPIC,
			PackageCode: it.PackageCode,
			Package:     it.PackageCode,
			PackageName: mkUnitName(it.Units),
		})
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return Request{}, err
	}
	return Request{
		Method:  "POST",
		Path:    mkOperations,
		Headers: map[string]string{"Content-Type": "application/json;charset=UTF-8"},
		Body:    string(raw),
	}, nil
}

// OpenShift builds the request that opens the register's day.
//
// ⚠️ **Its own operation, and not something we do speculatively.** The register
// refuses a sale into a closed shift with a named error (#2D), which is a
// better trigger than any guess we could make about whether a shift is open:
// asking first would add a round trip to every receipt, and opening one "just
// in case" files a shift-opening document at the tax committee for a register
// that already had one.
//
// So this is called exactly when the register has just said it is needed. See
// needsShift.
func (m *multikassa) OpenShift(cashier string, at time.Time) (Request, error) {
	body := mkRequest{
		Type:     mkOpenShift,
		Cashier:  cashier,
		Time:     at.Format("2006-01-02 15:04:05"),
		Terminal: m.terminal,
		Module:   m.terminal,
		// ⚠️ Not nil. Go marshals a nil slice as `null`, and a register reading
		// `"items": null` where it expects a list is the JSON trap from
		// CLAUDE.md arriving at somebody else's parser, where we cannot see it
		// fail.
		Items: []mkItem{},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return Request{}, err
	}
	return Request{
		Method:  "POST",
		Path:    mkOperations,
		Headers: map[string]string{"Content-Type": "application/json;charset=UTF-8"},
		Body:    string(raw),
	}, nil
}

// CloseShift builds the Z-report — the register's end of day.
//
// ⚠️ **The Z-report is a tax document, not a button.** It totals everything the
// register filed since the day was opened and hands that total to the state; it
// cannot be undone, and a receipt filed afterwards belongs to the next day. So
// this is never called speculatively, and the handler refuses it while any sale
// is still waiting to be registered — see the guard in fiscalday.go.
func (m *multikassa) CloseShift(cashier string, at time.Time) (Request, error) {
	body := mkRequest{
		Type:     mkCloseShift,
		Cashier:  cashier,
		Time:     at.Format("2006-01-02 15:04:05"),
		Terminal: m.terminal,
		Module:   m.terminal,
		// Same reason as OpenShift: a nil slice marshals to `null`, which fails
		// inside somebody else's parser where we cannot watch it.
		Items: []mkItem{},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return Request{}, err
	}
	return Request{
		Method:  "POST",
		Path:    mkOperations,
		Headers: map[string]string{"Content-Type": "application/json;charset=UTF-8"},
		Body:    string(raw),
	}, nil
}

// ShiftCloser is implemented by providers whose register ends its day with a
// report we have to ask for. Separate from ShiftOpener because a register that
// opens its own day may still need telling when it is over, and the reverse.
type ShiftCloser interface {
	CloseShift(cashier string, at time.Time) (Request, error)
}

// ZReport is what the register totalled for the day, in whole so'm.
//
// Worth carrying rather than discarding because it is an **independent second
// count** of the same day's takings: ours comes from the orders we recorded,
// this comes from the machine that filed them. When a drawer is short, the
// first useful question is which of the two the cash agrees with — and without
// this number that question cannot be asked at all.
type ZReport struct {
	// The Z-report's sequence number, which is what an inspector asks for.
	Number string
	// Sales, split the way the register splits them.
	SaleCash  int
	SaleCard  int
	SaleTotal int
	SaleCount int
	// Refunds, kept separate: a day with heavy refunds and a matching drawer is
	// a different story from a quiet one, and netting them hides it.
	RefundTotal int
	OpenedAt    string
}

type mkZResult struct {
	TotalSaleCash  string `json:"totalSaleCash"`
	TotalSaleCard  string `json:"totalSaleCard"`
	TotalSale      string `json:"totalSale"`
	TotalSaleCount string `json:"totalSaleCount"`
	TotalRefund    string `json:"totalRefund"`
	ZCount         string `json:"zCount"`
	OpenTime       string `json:"openTime"`
}

// ParseZReport reads the day's totals out of a close-shift reply.
//
// ⚠️ **A third money format from the same vendor**, and the reason this has its
// own parser rather than reusing anything above. The close-shift reply answers
// in **formatted so'm** — `"6,651,020.00"` — while their zReport endpoint
// answers the same figures as **bare tiyin integers** (`"665102000"`), and the
// sale request takes tiyin at the top level and so'm in the lines. Reading this
// one with either of the other two rules is wrong by a factor of a hundred or
// by a thousand, in a number an owner will compare against real cash.
func ParseZReport(body []byte) (ZReport, bool) {
	var env mkResponse
	if json.Unmarshal(body, &env) != nil {
		return ZReport{}, false
	}
	var data struct {
		Result *mkZResult `json:"result"`
	}
	if len(env.Data) == 0 || json.Unmarshal(env.Data, &data) != nil || data.Result == nil {
		return ZReport{}, false
	}
	r := data.Result
	return ZReport{
		Number:      r.ZCount,
		SaleCash:    mkMoney(r.TotalSaleCash),
		SaleCard:    mkMoney(r.TotalSaleCard),
		SaleTotal:   mkMoney(r.TotalSale),
		SaleCount:   mkMoney(r.TotalSaleCount),
		RefundTotal: mkMoney(r.TotalRefund),
		OpenedAt:    r.OpenTime,
	}, true
}

// mkMoney reads "6,651,020.00" as whole so'm.
//
// ⚠️ Truncates the tiyin rather than rounding, because every one of these is a
// **total of amounts we already know**, and rounding a total up can make it
// exceed the sum of its parts by a so'm — which reads, to the person checking a
// drawer, as the till being over.
func mkMoney(s string) int {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	s = strings.ReplaceAll(s, " ", "")
	if s == "" {
		return 0
	}
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = s[:i]
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// needsShift reports whether a refusal means "the day has not been opened".
//
// ⚠️ Matched on the register's **code** rather than its message text. The
// message arrives in Russian and is the vendor's to reword at any release; the
// code beside it is the thing their own error table is written in. Matching
// prose would work in testing and quietly stop working after an update, with
// the symptom being that every first sale of the day fails until somebody
// opens the shift by hand.
func needsShift(err error) bool {
	return err != nil && strings.Contains(err.Error(), "#2D")
}

// NeedsShift reports whether an error from a Multikassa filing means the
// register's shift has to be opened first.
//
// Exported so the handler can act on it without knowing the provider's error
// vocabulary — the same separation that keeps wire formats out of handlers.
func NeedsShift(provider string, err error) bool {
	// ⚠️ Dispatched by provider, never by sniffing the message. Two registers
	// spell "the day is not open" as `#2D` and `F-002`; matching either string
	// against the other's replies is how a refusal about something else becomes
	// an automatic shift-opening — which files a document with the tax
	// committee for a day that may already be open.
	switch provider {
	case Multikassa:
		return needsShift(err)
	case EPOS:
		return eposNeedsShift(err)
	}
	return false
}

// ShiftOpener is implemented by providers whose register needs its day opened
// before it will accept a sale.
//
// An interface rather than a method on Encoder because it is not universal:
// several virtual registers manage their own shift, and requiring every future
// adapter to implement an operation it does not have would push empty methods
// into all of them.
type ShiftOpener interface {
	OpenShift(cashier string, at time.Time) (Request, error)
}

// Hello asks the fiscal module to describe itself.
//
// ⚠️ Deliberately /api/v1/info and not /api/v1/contragents, though both answer.
// contragents returns the company — the taxpayer's name, address and bank
// account — which is a lot of somebody's registration data to pull across for a
// question that is only "is the register there and does it have room". info
// answers exactly that: terminal id, applet version, and how many receipts fit
// before the memory has to be cleared.
func (m *multikassa) Hello() (Request, error) {
	return Request{Method: "GET", Path: mkInfo}, nil
}

// mkResponse is the envelope every operation answers in.
//
// ⚠️ `data` is polymorphic — the receipt on success, `{"message": ...}` on
// failure, `{"result": {...}}` for shift operations — so it is held raw and
// read only after the outcome is known.
type mkResponse struct {
	Code    int             `json:"code"`
	Success *bool           `json:"success"`
	Data    json.RawMessage `json:"data"`
}

type mkData struct {
	Message    string `json:"message"`
	FiscalSign string `json:"receipt_gnk_fiscalsign"`
	QRCode     string `json:"receipt_gnk_qrcodeurl"`
	// A string in every example we have, but their own zReport answers with
	// bare integers elsewhere, so it is read leniently.
	ReceiptSeq json.RawMessage `json:"receipt_gnk_receiptseq"`
	Terminal   string          `json:"module_gnk_terminalid"`
	Company    string          `json:"company"`

	Result *struct {
		TerminalID   string `json:"terminalId"`
		AppletVer    string `json:"appletVersion"`
		ZReportCount string `json:"zReportCount"`
		RecCount     string `json:"currentRecCount"`
		RecMaxCount  string `json:"currentRecMaxCount"`
	} `json:"result"`
}

// Parse reads a response from the register.
//
// ⚠️ **The HTTP status is evidence, not the verdict.** This register answers
// business refusals with 500 and a body — "#2D - Z-отчет не был открыт" means
// the shift is not open, which is a thing the cashier fixes in ten seconds, and
// treating it as a transport failure would send them to check the network cable
// instead. So the body is read first and the status only decides what to say
// when there is no readable body at all.
//
// ⚠️ Success is likewise not taken from the `success` flag alone: **a filing
// counted as done only if a fiscal sign came back.** That is the thing the whole
// operation exists to obtain, and a 200 without one is a receipt that was not
// filed no matter how cheerful the envelope is.
func (m *multikassa) Parse(status int, body []byte) (Result, error) {
	var env mkResponse
	if err := json.Unmarshal(body, &env); err != nil {
		// Not JSON at all — a proxy page, a wrong port, or something that is not
		// the register. The paytest tool learned this one the hard way: a
		// "404 page not found" read as a structured refusal made every rejection
		// test pass while nothing worked.
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 120 {
			snippet = snippet[:120] + "…"
		}
		if snippet == "" {
			return Result{}, fmt.Errorf("kassa javob bermadi (HTTP %d)", status)
		}
		return Result{}, fmt.Errorf("kassadan tushunarsiz javob (HTTP %d): %s", status, snippet)
	}

	var d mkData
	if len(env.Data) > 0 {
		_ = json.Unmarshal(env.Data, &d)
	}

	if d.FiscalSign != "" {
		return Result{
			ReceiptID:  mkScalar(d.ReceiptSeq),
			FiscalSign: d.FiscalSign,
			QRText:     d.QRCode,
		}, nil
	}
	if msg := strings.TrimSpace(d.Message); msg != "" {
		return Result{}, errors.New(msg)
	}
	if env.Success != nil && !*env.Success {
		return Result{}, fmt.Errorf("kassa amalni rad etdi (kod %d)", env.Code)
	}
	if d.Result != nil {
		// A shift operation or the hello call. No fiscal sign is expected; the
		// caller reads the description rather than the Result.
		return Result{ReceiptID: d.Result.TerminalID}, nil
	}
	return Result{}, errors.New("kassa fiskal belgi qaytarmadi")
}

// Describe turns a hello response into the line the connection check shows.
//
// ⚠️ Says **what** it reached, following the POS package's rule: a bare
// "connected" cannot tell the restaurant's own register from the one next door,
// and on this screen that is the entire question. The receipt-memory figure is
// included because it is the one thing about a fiscal module that goes wrong
// slowly and then stops the till dead.
func Describe(body []byte) string {
	var env mkResponse
	if json.Unmarshal(body, &env) != nil {
		return ""
	}
	var d mkData
	if len(env.Data) > 0 {
		_ = json.Unmarshal(env.Data, &d)
	}
	if d.Result == nil {
		return ""
	}
	out := "kassa " + d.Result.TerminalID
	if d.Result.AppletVer != "" {
		out += " · v" + d.Result.AppletVer
	}
	if d.Result.RecMaxCount != "" {
		out += " · cheklar " + d.Result.RecCount + "/" + d.Result.RecMaxCount
	}
	return out
}

// mkScalar reads a field the register sends sometimes quoted and sometimes not.
// Same leniency the ATMOS callback needed, and for the same reason: the
// documentation does not say which, and both turn up.
func mkScalar(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return ""
	}
	if unquoted, err := strconv.Unquote(s); err == nil {
		return unquoted
	}
	return s
}

// mkUnitName is the human packaging label beside the code.
//
// The register prints this on the guest's receipt, so it is in the language the
// receipt is read in. Only the units a restaurant actually sells in are named;
// anything else falls back to pieces, which is what a portion is.
func mkUnitName(units int) string {
	switch units {
	case 10:
		return "gramm"
	case 11:
		return "kg"
	case 41:
		return "litr"
	}
	return "dona"
}
