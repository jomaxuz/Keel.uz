package fiscal

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func eposEnc(t *testing.T) Encoder {
	t.Helper()
	enc, err := newEPOS(Creds{Token: "tok-123"})
	if err != nil {
		t.Fatalf("newEPOS: %v", err)
	}
	return enc
}

func eposBody(t *testing.T, r Request) eposRequest {
	t.Helper()
	var got eposRequest
	if err := json.Unmarshal([]byte(r.Body), &got); err != nil {
		t.Fatalf("body is not the shape we sent: %v\n%s", err, r.Body)
	}
	return got
}

// ⚠️ Both scalings in one test, because both produce a receipt that adds up and
// is wrong. Money is tiyin — a hundred times out and nothing looks odd — and
// quantity is thousandths, under a field called `amount` that reads like a sum
// of money. Two portions sent as `2` is a filing for two thousandths of a dish
// at a total the register will still print.
func TestEPOSScalesMoneyAndQuantityTheWayTheAppExpects(t *testing.T) {
	r := Build(Sale{
		Lines:      []Line{{Name: "Lag'mon", Price: 42000, Qty: 2, Units: 0}},
		Cash:       84000,
		VatPercent: 12,
	})
	req, err := eposEnc(t).Sale(r)
	if err != nil {
		t.Fatalf("Sale: %v", err)
	}
	got := eposBody(t, req)

	if len(got.Items) != 1 {
		t.Fatalf("items=%d, want 1", len(got.Items))
	}
	it := got.Items[0]
	if it.Price != 4200000 {
		t.Fatalf("price=%d, want 4200000 tiyin (42 000 so'm)", it.Price)
	}
	if it.Amount != 2000 {
		t.Fatalf("amount=%d — two portions must be 2000, not the count", it.Amount)
	}
	if got.Received.Cash != 8400000 {
		t.Fatalf("cash=%d, want 8400000 tiyin", got.Received.Cash)
	}
	// The VAT inside the price, not added on top: 84 000 × 12/112.
	if it.VatPercent != 12 || it.Vat != 900000 {
		t.Fatalf("vatPercent=%d vat=%d, want 12 and 900000", it.VatPercent, it.Vat)
	}
	if got.Type != eposTypeSale || got.RefundInfo != nil {
		t.Fatalf("a sale is being filed as %q with refundInfo=%v", got.Type, got.RefundInfo)
	}
}

// The token is what the app checks; S-001 refuses without it, and it is the one
// header the caller cannot supply — the request is built here and carried by a
// browser that must not be handed the credential separately.
func TestEPOSCarriesItsKeyOnEveryCall(t *testing.T) {
	enc := eposEnc(t)
	sale, _ := enc.Sale(Build(Sale{Lines: []Line{{Name: "Choy", Price: 5000, Qty: 1}}, Cash: 5000}))
	hello, _ := enc.Hello()
	open, _ := enc.(ShiftOpener).OpenShift("", time.Time{})

	for name, req := range map[string]Request{"sale": sale, "hello": hello, "open": open} {
		if req.Headers["X-API-Key"] != "tok-123" {
			t.Fatalf("%s went without the API key: %v", name, req.Headers)
		}
	}
	// ⚠️ S-003 refuses a request with no body, and both shift calls take no
	// parameters — exactly the shape somebody would send empty.
	if open.Body == "" {
		t.Fatal("the shift call went with no body at all")
	}
}

// ⚠️ E-007. A reversal that does not name its sale is a standalone negative
// receipt: our books balance and the state's copy holds a return against
// nothing. Refused here rather than at the register, because the register
// refuses while a guest is waiting for their money.
func TestEPOSRefundNamesTheSaleItReverses(t *testing.T) {
	sold := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	r := Build(Sale{
		Lines:    []Line{{Name: "Non", Price: 3500, Qty: 1}},
		Cash:     3500,
		IsRefund: true,
		Original: OriginalReceipt{
			TerminalID: "LG230110007836",
			Seq:        "44",
			At:         sold,
			Sign:       "ABC123DEF456",
		},
	})
	req, err := eposEnc(t).Sale(r)
	if err != nil {
		t.Fatalf("Sale: %v", err)
	}
	got := eposBody(t, req)

	if got.Type != eposTypeRefund {
		t.Fatalf("type=%q, want %q", got.Type, eposTypeRefund)
	}
	if got.RefundInfo == nil {
		t.Fatal("the reversal carries no refundInfo — E-007")
	}
	if got.RefundInfo.FiscalSign != "ABC123DEF456" {
		t.Fatalf("fiscalSign=%q", got.RefundInfo.FiscalSign)
	}
	// ⚠️ A number here and a string in our record, because our record holds
	// whatever any provider calls a receipt — REGOS answers with a UUID.
	if got.RefundInfo.ReceiptSeq != 44 {
		t.Fatalf("receiptSeq=%d, want 44", got.RefundInfo.ReceiptSeq)
	}
	// ⚠️ `YYYYMMDDTHHmmss`, with the literal T — Multikassa spells the same
	// instant with no separator at all.
	if got.RefundInfo.DateTime != "20260405T100000" {
		t.Fatalf("dateTime=%q, want 20260405T100000", got.RefundInfo.DateTime)
	}
}

func TestEPOSRefundWithNoFiscalSignIsRefused(t *testing.T) {
	r := Build(Sale{
		Lines:    []Line{{Name: "Non", Price: 3500, Qty: 1}},
		Cash:     3500,
		IsRefund: true,
		Original: OriginalReceipt{TerminalID: "LG23", Seq: "44"},
	})
	if _, err := eposEnc(t).Sale(r); err == nil {
		t.Fatal("a reversal was built against a sale it cannot name")
	}
}

// ⚠️ **The trap that is specific to this provider.** With their OFD down the
// app still issues the receipt: the guest has their paper and the fiscal module
// has signed it. Reading `ofdSent: false` as a failure would send our own retry
// to file the sale a second time, and a duplicate fiscal receipt is a sale the
// restaurant is taxed on twice.
func TestEPOSReceiptIsFiledEvenWhenTheOFDIsDown(t *testing.T) {
	body := []byte(`{"success":true,"receipt":{"terminalId":"LG23","receiptSeq":44,
		"fiscalSign":"ABC123","qrUrl":"https://ofd.soliq.uz/check?x=1"},
		"ofdSent":false,"ofdError":"All OFD servers failed"}`)

	res, err := eposEnc(t).Parse(200, body)
	if err != nil {
		t.Fatalf("a signed receipt was reported as unfiled: %v", err)
	}
	if res.FiscalSign != "ABC123" || res.ReceiptID != "44" {
		t.Fatalf("result=%+v", res)
	}
	if res.QRText == "" {
		t.Fatal("the guest's QR was dropped")
	}
}

// ⚠️ `success` decides, not the status code, and the code travels with the
// message: the message is Russian prose the vendor may reword, the code is what
// their own error table — and any support conversation — is written in.
func TestEPOSRefusalIsReadFromTheBodyNotTheStatus(t *testing.T) {
	body := []byte(`{"success":false,"error":{"code":"E-013","rule":"ITEM_SPIC_NOT_IN_TASNIF",
		"message":"Код ИКПУ не найден в справочнике Tasnif"}}`)

	if _, err := eposEnc(t).Parse(200, body); err == nil {
		t.Fatal("a refusal inside a 200 was read as a success")
	} else if !strings.Contains(err.Error(), "E-013") ||
		!strings.Contains(err.Error(), "Tasnif") {
		t.Fatalf("the refusal lost its code or its words: %v", err)
	}
}

// ⚠️ Dispatched by provider. Two registers spell "the day is not open" as `#2D`
// and `F-002`; matching either string against the other's replies turns an
// unrelated refusal into an automatic shift opening, which files a document
// with the tax committee for a day that may already be open.
func TestEPOSShiftRefusalIsRecognisedAndNotBorrowed(t *testing.T) {
	closed := []byte(`{"success":false,"error":{"code":"F-002",
		"message":"Z-отчёт не открыт. Откройте смену перед продажей"}}`)
	_, err := eposEnc(t).Parse(200, closed)

	if !NeedsShift(EPOS, err) {
		t.Fatalf("F-002 was not recognised as a closed shift: %v", err)
	}
	if NeedsShift(Multikassa, err) {
		t.Fatal("E-POS's F-002 is being read by the Multikassa adapter")
	}
	other := []byte(`{"success":false,"error":{"code":"E-006","message":"Список товаров пуст"}}`)
	_, err = eposEnc(t).Parse(200, other)
	if NeedsShift(EPOS, err) {
		t.Fatalf("an unrelated refusal is opening the register's day: %v", err)
	}
}

// ⚠️ A reachable register that will refuse every sale must not answer
// "connected". Both states are things the owner fixes on the phone, and neither
// is visible from anywhere else.
func TestEPOSConnectionCheckNamesWhatIsWrong(t *testing.T) {
	ok := DescribeEPOS([]byte(`{"app":"EPOS","version":"1.0.2",
		"fmInitialized":true,"activated":true,"subscriptionState":"active"}`))
	if !strings.Contains(ok, "EPOS") || !strings.Contains(ok, "1.0.2") {
		t.Fatalf("describe=%q", ok)
	}
	if strings.Contains(ok, "faollashtirilmagan") {
		t.Fatalf("a healthy register is being reported as broken: %q", ok)
	}

	bad := DescribeEPOS([]byte(`{"app":"EPOS","version":"1.0.2",
		"fmInitialized":false,"activated":false,"subscriptionState":"expired"}`))
	for _, want := range []string{"faollashtirilmagan", "fiskal modul", "obuna"} {
		if !strings.Contains(bad, want) {
			t.Fatalf("describe=%q, missing %q", bad, want)
		}
	}
}
