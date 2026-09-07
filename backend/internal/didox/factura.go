package didox

// The electronic invoice itself: how one is read, and how one is written.
//
// ⚠️ **The receiving side validates the structure strictly, and "strictly"
// here means a field we invent is a document refused whole** — not a warning,
// not a blank line on the paper. The rules, copied from the operator's own
// page (docs/vendor/didox.md):
//
//   - no extra fields at all;
//   - an unused object is `null`, never `{}` with empty strings;
//   - `Count` to six decimals, every other number to two, a dot for the point
//     and no thousands separators;
//   - dates exactly `yyyy-MM-dd`, with no time and no zone.
//
// So the numbers here are **strings formatted once**, at the edge, rather than
// floats left to Go's marshaller: `json.Marshal` writes 1e+07 for ten million,
// which is a valid JSON number and an invalid invoice.

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Factura is the body of an electronic invoice, in the operator's own shape.
//
// ⚠️ Every pointer field exists so it can be **null** rather than empty; see
// the note above. A struct with `omitempty` would drop them entirely, which is
// a third thing — and the validator treats a missing key and a null key
// differently on some of them.
type Factura struct {
	Version         int              `json:"Version"`
	WaybillLocalIds []string         `json:"WaybillLocalIds"`
	HasMarking      bool             `json:"HasMarking"`
	HasRent         bool             `json:"HasRent"`
	FacturaRentDoc  *json.RawMessage `json:"FacturaRentDoc"`
	FacturaType     int              `json:"FacturaType"`
	ProductList     ProductList      `json:"ProductList"`
	FacturaDoc      FacturaDoc       `json:"FacturaDoc"`
	ContractDoc     ContractDoc      `json:"ContractDoc"`
	ContractID      *string          `json:"ContractId"`
	LotID           string           `json:"LotId"`
	OldFacturaDoc   *json.RawMessage `json:"OldFacturaDoc"`
	SellerTin       string           `json:"SellerTin"`
	Seller          Party            `json:"Seller"`
	ItemReleasedDoc *json.RawMessage `json:"ItemReleasedDoc"`
	BuyerTin        string           `json:"BuyerTin"`
	Buyer           *Party           `json:"Buyer"`

	FacturaInvestmentObjectDoc *json.RawMessage `json:"FacturaInvestmentObjectDoc"`
	FacturaEmpowermentDoc      *json.RawMessage `json:"FacturaEmpowermentDoc"`
	ForeignCompany             *json.RawMessage `json:"ForeignCompany"`
}

// FacturaDoc is the invoice's own number and date.
type FacturaDoc struct {
	FacturaNo   string `json:"FacturaNo"`
	FacturaDate string `json:"FacturaDate"`
}

// ContractDoc is the contract the invoice is issued under.
//
// ⚠️ **Required even when there is no written contract**, which is the ordinary
// case for a restaurant buying vegetables. The operator wants a number and a
// date; what goes in is the arrangement the two sides actually have, and the
// panel makes the owner type it rather than inventing one — an invented
// contract number is a claim about a document that does not exist.
type ContractDoc struct {
	ContractNo   string `json:"ContractNo"`
	ContractDate string `json:"ContractDate"`
}

// Party is a seller or a buyer.
type Party struct {
	Name         string `json:"Name"`
	BranchCode   string `json:"BranchCode"`
	BranchName   string `json:"BranchName"`
	VatRegCode   string `json:"VatRegCode"`
	Account      string `json:"Account"`
	BankID       string `json:"BankId"`
	Address      string `json:"Address"`
	Director     string `json:"Director"`
	Accountant   string `json:"Accountant"`
	VatRegStatus int    `json:"VatRegStatus"`
}

// ProductList is the goods and what is true of all of them.
type ProductList struct {
	HasCommittent        bool      `json:"HasCommittent"`
	HasLgota             bool      `json:"HasLgota"`
	Tin                  string    `json:"Tin"`
	HideReportCommittent bool      `json:"HideReportCommittent"`
	HasExcise            bool      `json:"HasExcise"`
	HasVat               bool      `json:"HasVat"`
	Products             []Product `json:"Products"`
}

// Product is one line.
type Product struct {
	OrdNo                  int              `json:"OrdNo"`
	LgotaID                *string          `json:"LgotaId"`
	CommittentName         string           `json:"CommittentName"`
	CommittentTin          string           `json:"CommittentTin"`
	CommittentVatRegCode   string           `json:"CommittentVatRegCode"`
	CommittentVatRegStatus *int             `json:"CommittentVatRegStatus"`
	Name                   string           `json:"Name"`
	CatalogCode            string           `json:"CatalogCode"`
	CatalogName            string           `json:"CatalogName"`
	Marks                  *json.RawMessage `json:"Marks"`
	Barcode                string           `json:"Barcode"`
	PackageCode            string           `json:"PackageCode"`
	PackageName            string           `json:"PackageName"`
	Count                  json.Number      `json:"Count"`
	Summa                  string           `json:"Summa"`
	DeliverySum            string           `json:"DeliverySum"`
	VatRate                string           `json:"VatRate"`
	VatSum                 string           `json:"VatSum"`
	ExciseRate             int              `json:"ExciseRate"`
	ExciseSum              int              `json:"ExciseSum"`
	DeliverySumWithVat     string           `json:"DeliverySumWithVat"`
	WithoutVat             bool             `json:"WithoutVat"`
	LgotaType              *int             `json:"LgotaType"`
	LgotaName              *string          `json:"LgotaName"`
	LgotaVatSum            *float64         `json:"LgotaVatSum"`
	WarehouseID            *string          `json:"WarehouseId"`
	Origin                 int              `json:"Origin"`
}

// Money formats a so'm amount the way the validator wants it.
//
// ⚠️ **Two decimals, a dot, no separators, and never scientific notation.**
// Ten million so'm — an ordinary week's delivery — is exactly where Go's
// default float formatting switches to `1e+07`, and the document is refused
// with a message about a field that looks fine on our screen.
func Money(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }

// Count formats a quantity: up to six decimals, and no trailing zeroes beyond
// what was actually measured.
func Count(v float64) json.Number {
	s := strconv.FormatFloat(v, 'f', 6, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimSuffix(s, ".")
	if s == "" || s == "-" {
		s = "0"
	}
	return json.Number(s)
}

// Num reads one of the operator's numbers, whichever shape it arrived in.
//
// ⚠️ **They send the same field as a number and as a string** — `"Summa"` is a
// string, `"Count"` is a number, and a list row's `total_sum` has been both.
// A parser that assumes one of them silently reads zero for the other, and a
// zero here is a delivery that cost nothing.
func Num(v json.Number) float64 {
	f, err := v.Float64()
	if err != nil {
		return 0
	}
	return f
}

// ParseFactura reads an invoice body into the shape our panel works with.
//
// ⚠️ **Tolerant on the way in, strict on the way out.** An incoming document
// was written by somebody else's accounting system: it carries fields we have
// never seen, decimals as strings, and objects where we expect null. Refusing
// it would mean refusing to *show* a delivery that has already legally
// happened — so reading keeps what it understands and ignores the rest, while
// writing (above) obeys the validator to the letter.
func ParseFactura(raw json.RawMessage) (*Factura, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("didox: hujjat tanasi bo'sh")
	}
	var f Factura
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("didox: hujjatni o'qib bo'lmadi: %w", err)
	}
	return &f, nil
}

// ParseMoney reads one of the string amounts on a line.
func ParseMoney(s string) float64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, " ", ""))
	if s == "" {
		return 0
	}
	// ⚠️ A comma is accepted on the way in and never written on the way out:
	// some senders' systems localise the decimal point, and dropping their
	// invoice over a comma would be dropping a delivery that arrived.
	s = strings.ReplaceAll(s, ",", ".")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}
