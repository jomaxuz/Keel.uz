package didox

import (
	"encoding/json"
	"strings"
	"testing"
)

// ⚠️ **Ten million so'm is where Go's default float formatting switches to
// scientific notation**, and ten million is an ordinary week's delivery. The
// receiving side validates the structure strictly: `1e+07` is valid JSON, an
// invalid invoice, and a refusal whose message names a field that looks
// perfectly normal on our own screen.
func TestMoneyIsNeverWrittenInScientificNotation(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want string
	}{
		{10_000_000, "10000000.00"},
		{100_000_000, "100000000.00"},
		{0, "0.00"},
		{1234.5, "1234.50"},
		{0.005, "0.01"},
	} {
		if got := Money(c.in); got != c.want {
			t.Errorf("Money(%v) = %q, want %q", c.in, got, c.want)
		}
	}
	// And the whole document, marshalled, carries no exponent anywhere.
	raw, err := json.Marshal(Factura{
		ProductList: ProductList{Products: []Product{{
			Summa:              Money(12_000_000),
			DeliverySum:        Money(120_000_000),
			DeliverySumWithVat: Money(134_400_000),
			Count:              Count(10),
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "e+") {
		t.Fatalf("an exponent reached the invoice: %s", raw)
	}
}

// ⚠️ **Six decimals for a quantity, and no trailing noise.** 0.1 kilo written
// as `0.100000` is accepted, but a count that arrived as 1 must not go back as
// `1.000000` — the operator's own examples are plain, and a needlessly long
// number is the kind of difference that gets blamed for a refusal nobody can
// reproduce.
func TestCountKeepsWhatWasMeasuredAndNothingMore(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want string
	}{
		{1, "1"},
		{2.5, "2.5"},
		{0.125, "0.125"},
		{0, "0"},
		// Beyond six decimals the operator's own limit applies.
		{1.23456789, "1.234568"},
	} {
		if got := string(Count(c.in)); got != c.want {
			t.Errorf("Count(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ⚠️ **The unused objects are `null`, never `{}`.** The operator's page says so
// in as many words, and the difference is a document the counterparty's system
// refuses whole rather than a blank line on the paper.
func TestUnusedObjectsAreNullRatherThanEmpty(t *testing.T) {
	raw, err := json.Marshal(Factura{Version: 1, WaybillLocalIds: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{
		"FacturaRentDoc", "OldFacturaDoc", "ItemReleasedDoc",
		"FacturaInvestmentObjectDoc", "FacturaEmpowermentDoc", "ForeignCompany",
	} {
		if !strings.Contains(string(raw), `"`+field+`":null`) {
			t.Errorf("%s is not null: %s", field, raw)
		}
	}
	// ⚠️ And the list of waybills is `[]`, not `null`: Go's nil slice writes
	// `null`, which is the trap CLAUDE.md §10 describes — here it would be a
	// field the validator reads as "a waybill I cannot see".
	if !strings.Contains(string(raw), `"WaybillLocalIds":[]`) {
		t.Errorf("an empty waybill list became null: %s", raw)
	}
}

// ⚠️ **A comma is accepted coming in and never written going out.** Some
// senders' systems localise the decimal point; dropping their invoice over it
// would be dropping a delivery that physically arrived.
func TestAmountsAreReadHoweverTheyWereWritten(t *testing.T) {
	for _, c := range []struct {
		in   string
		want float64
	}{
		{"10000.00", 10000},
		{"10000,50", 10000.5},
		{"1 200.00", 1200},
		{"", 0},
		{"nonsense", 0},
	} {
		if got := ParseMoney(c.in); got != c.want {
			t.Errorf("ParseMoney(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// An incoming invoice is read for what we understand and never refused for
// carrying more — it describes a delivery that has already happened.
func TestAnIncomingInvoiceIsReadRatherThanValidated(t *testing.T) {
	raw := json.RawMessage(`{
		"Version": 1,
		"SomethingWeHaveNeverSeen": {"a": 1},
		"ProductList": {"HasVat": true, "Products": [
			{"OrdNo": 1, "Name": "Go'sht", "CatalogCode": "01012001",
			 "Count": 12.5, "Summa": "95000.00", "DeliverySum": "1187500.00",
			 "VatRate": "12", "VatSum": "142500.00",
			 "DeliverySumWithVat": "1330000.00", "PackageName": "kg"}]},
		"FacturaDoc": {"FacturaNo": "SF-77", "FacturaDate": "2026-09-01"}
	}`)
	f, err := ParseFactura(raw)
	if err != nil {
		t.Fatalf("an invoice with an unknown field was refused: %v", err)
	}
	if len(f.ProductList.Products) != 1 {
		t.Fatalf("lines lost: %+v", f.ProductList)
	}
	p := f.ProductList.Products[0]
	if got := Num(p.Count); got != 12.5 {
		t.Errorf("count = %v, want 12.5", got)
	}
	if got := ParseMoney(p.DeliverySumWithVat); got != 1_330_000 {
		t.Errorf("line total = %v", got)
	}
	if f.FacturaDoc.FacturaNo != "SF-77" {
		t.Errorf("invoice number = %q", f.FacturaDoc.FacturaNo)
	}
	// An empty body is an error rather than an empty invoice: "a delivery with
	// no lines" and "we could not read the delivery" are different answers.
	if _, err := ParseFactura(nil); err == nil {
		t.Error("an empty body was read as a document")
	}
}
