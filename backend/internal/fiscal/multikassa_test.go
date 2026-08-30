package fiscal

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func mkSaleBody(t *testing.T, s Sale) map[string]any {
	t.Helper()
	enc, err := EncoderFor(Multikassa, Creds{RegisterID: "UZ210317263974"})
	if err != nil {
		t.Fatalf("EncoderFor: %v", err)
	}
	req, err := enc.Sale(Build(s))
	if err != nil {
		t.Fatalf("Sale: %v", err)
	}
	if req.Method != "POST" || req.Path != mkOperations {
		t.Fatalf("wrong call: %s %s", req.Method, req.Path)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(req.Body), &out); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	return out
}

// ⚠️ The 100× test. The vendor's PDF and their Postman collection disagree about
// whether the top-level totals are tiyin or so'm, and being wrong here files a
// tax document off by a factor of a hundred. This seals the reading we chose —
// the PDF's — so that changing it is a deliberate act with a failing test
// attached, not something that drifts in during an unrelated edit.
//
// The mixed units are the whole point: receipt_sum in tiyin, items in so'm, in
// the same request body.
func TestMultikassaMoneyUnitsAreMixedOnPurpose(t *testing.T) {
	body := mkSaleBody(t, Sale{
		Lines:      []Line{{Name: "Lag'mon", Price: 45_000, Qty: 2}},
		Cash:       90_000,
		VatPercent: 12,
	})

	if got := body["receipt_sum"]; got != float64(9_000_000) {
		t.Fatalf("receipt_sum = %v, want 9000000 tiyin (90 000 so'm)", got)
	}
	if got := body["receipt_gnk_receivedcash"]; got != float64(9_000_000) {
		t.Fatalf("receivedcash = %v, want 9000000 tiyin", got)
	}

	items := body["items"].([]any)
	line := items[0].(map[string]any)
	if got := line["product_price"]; got != float64(45_000) {
		t.Fatalf("product_price = %v, want 45000 so'm — items are NOT tiyin", got)
	}
	if got := line["total_product_price"]; got != float64(90_000) {
		t.Fatalf("total_product_price = %v, want 90000 so'm", got)
	}
}

// ⚠️ An order-level discount is spread across the lines to the tiyin, so a
// line's share is routinely not a whole so'm. Serialised through float64 that
// becomes 33.329999999999998 on a filed document; the decimal has to survive
// the trip intact.
func TestMultikassaKeepsFractionalSoM(t *testing.T) {
	enc, _ := EncoderFor(Multikassa, Creds{})
	// Three equal lines and a discount that does not divide by three.
	req, err := enc.Sale(Build(Sale{
		Lines: []Line{
			{Name: "A", Price: 10_000, Qty: 1},
			{Name: "B", Price: 10_000, Qty: 1},
			{Name: "C", Price: 10_000, Qty: 1},
		},
		Discount:   1_000,
		Cash:       29_000,
		VatPercent: 12,
	}))
	if err != nil {
		t.Fatalf("Sale: %v", err)
	}
	if strings.Contains(req.Body, "9999999") || strings.Contains(req.Body, "0000001") {
		t.Fatalf("float noise reached the wire:\n%s", req.Body)
	}
	// And the lines must still add up to what was taken. A receipt whose goods
	// do not sum to the money is the failure the spread exists to prevent.
	var body struct {
		Sum   int64 `json:"receipt_sum"`
		Items []struct {
			Total json.Number `json:"total_product_price"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var lines int64
	for _, it := range body.Items {
		f, _ := it.Total.Float64()
		lines += int64(f*100 + 0.5)
	}
	if lines != body.Sum {
		t.Fatalf("lines total %d tiyin, receipt_sum %d tiyin", lines, body.Sum)
	}
	if body.Sum != 2_900_000 {
		t.Fatalf("receipt_sum = %d, want 2900000 tiyin", body.Sum)
	}
}

// ⚠️ ИКПУ goes out under both names the two documents use. They are the same
// number, so this cannot be wrong under either reading — while sending only one
// of them is a rejected receipt under the other.
func TestMultikassaSendsClassifierUnderBothNames(t *testing.T) {
	body := mkSaleBody(t, Sale{
		Lines: []Line{{
			Name: "Osh", Price: 40_000, Qty: 1,
			SPIC: "01806001001129002", PackageCode: "1302374",
		}},
		Cash:       40_000,
		VatPercent: 12,
	})
	line := body["items"].([]any)[0].(map[string]any)
	if line["ikpu"] != "01806001001129002" || line["classifier_class_code"] != "01806001001129002" {
		t.Fatalf("classifier code not sent under both names: %v", line)
	}
	if line["packageCode"] != "1302374" || line["product_package"] != "1302374" {
		t.Fatalf("package code not sent under both names: %v", line)
	}

	// ⚠️ And an absent code is absent, never an empty string: a blank ИКПУ on a
	// filed receipt is a claim about a product, and the reason MenuItem.Ikpu is
	// allowed to be empty in the first place.
	plain := mkSaleBody(t, Sale{
		Lines: []Line{{Name: "Choy", Price: 5_000, Qty: 1}},
		Cash:  5_000,
	})
	bare := plain["items"].([]any)[0].(map[string]any)
	if _, ok := bare["ikpu"]; ok {
		t.Fatalf("empty ИКПУ was sent as a field: %v", bare)
	}
	if _, ok := bare["classifier_class_code"]; ok {
		t.Fatalf("empty classifier code was sent as a field: %v", bare)
	}
}

// ⚠️ Only the absolute discount is sent. The Postman example's discount_percent
// says the same thing a second way, and a register that honoured both would take
// the money off twice — which looks like a pricing bug in our system.
func TestMultikassaSendsDiscountOnlyOnce(t *testing.T) {
	body := mkSaleBody(t, Sale{
		Lines:      []Line{{Name: "Osh", Price: 50_000, Qty: 1}},
		Discount:   10_000,
		Cash:       40_000,
		VatPercent: 12,
	})
	line := body["items"].([]any)[0].(map[string]any)
	if _, ok := line["discount_percent"]; ok {
		t.Fatalf("discount sent twice: %v", line)
	}
	if line["product_discount"] != float64(10_000) {
		t.Fatalf("product_discount = %v, want 10000 so'm", line["product_discount"])
	}
	if body["receipt_sum"] != float64(4_000_000) {
		t.Fatalf("receipt_sum = %v, want 4000000 tiyin", body["receipt_sum"])
	}
}

// A zero rate is declared as exempt rather than left to be read as unfilled.
func TestMultikassaDeclaresExemptLines(t *testing.T) {
	body := mkSaleBody(t, Sale{
		Lines:      []Line{{Name: "Non", Price: 3_000, Qty: 1}},
		Cash:       3_000,
		VatPercent: 0,
	})
	line := body["items"].([]any)[0].(map[string]any)
	if line["product_without_vat"] != true {
		t.Fatalf("a 0%% line must be flagged exempt: %v", line)
	}
}

func mkParse(t *testing.T, status int, body string) (Result, error) {
	t.Helper()
	enc, err := EncoderFor(Multikassa, Creds{})
	if err != nil {
		t.Fatalf("EncoderFor: %v", err)
	}
	return enc.Parse(status, []byte(body))
}

// ⚠️ The register answers business refusals with HTTP 500 and a readable body.
// Reading the status instead of the body turns "open the shift" — a ten-second
// fix at the counter — into "the network is down", which is somebody else's
// problem and the wrong one.
func TestMultikassaReadsRefusalsOutOfA500(t *testing.T) {
	_, err := mkParse(t, 500,
		`{"code":500,"data":{"message":"#2D - Z-отчет не был открыт"},"success":false}`)
	if err == nil {
		t.Fatal("a refusal was read as a success")
	}
	if !strings.Contains(err.Error(), "#2D") {
		t.Fatalf("the register's own words were lost: %v", err)
	}
}

// ⚠️ A filing counts only if a fiscal sign came back. A cheerful envelope
// without one is a receipt that was not filed, and recording it as filed is the
// one outcome that cannot be detected later.
func TestMultikassaNeedsAFiscalSign(t *testing.T) {
	if _, err := mkParse(t, 200, `{"code":200,"success":true,"data":{}}`); err == nil {
		t.Fatal("a response with no fiscal sign was accepted as filed")
	}

	res, err := mkParse(t, 200, `{"code":200,"data":{
		"receipt_gnk_fiscalsign":"002519286194",
		"receipt_gnk_qrcodeurl":"https://ofd.soliq.uz/check?t=UZ21&r=538&c=20241104181718&s=002519286194",
		"receipt_gnk_receiptseq":"538"}}`)
	if err != nil {
		t.Fatalf("a real filing was rejected: %v", err)
	}
	if res.FiscalSign != "002519286194" || res.ReceiptID != "538" {
		t.Fatalf("bad result: %+v", res)
	}
	if !strings.HasPrefix(res.QRText, "https://ofd.soliq.uz/check?") {
		t.Fatalf("the guest's QR was not carried through: %q", res.QRText)
	}
}

// The receipt number arrives quoted in every example we have and bare in their
// other endpoints. Both are the same number.
func TestMultikassaReadsBareReceiptNumbers(t *testing.T) {
	res, err := mkParse(t, 200,
		`{"code":200,"data":{"receipt_gnk_fiscalsign":"x","receipt_gnk_receiptseq":538}}`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if res.ReceiptID != "538" {
		t.Fatalf("receipt number = %q, want 538", res.ReceiptID)
	}
}

// ⚠️ A non-JSON reply is a failure, never a refusal to be shown as one. The
// paytest tool was fooled by exactly this once: a "404 page not found" parsed as
// a structured rejection made every negative test pass while nothing worked.
func TestMultikassaRejectsNonJSON(t *testing.T) {
	_, err := mkParse(t, 404, "404 page not found")
	if err == nil {
		t.Fatal("an HTML/text error page was accepted")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("the status was not reported: %v", err)
	}
}

// The connection check says what it reached, not merely that it reached
// something — one till looks exactly like another from a status code.
func TestMultikassaDescribesWhatItReached(t *testing.T) {
	enc, _ := EncoderFor(Multikassa, Creds{})
	req, err := enc.Hello()
	if err != nil {
		t.Fatalf("Hello: %v", err)
	}
	if req.Method != "GET" || req.Path != mkInfo {
		t.Fatalf("wrong hello call: %s %s", req.Method, req.Path)
	}
	got := Describe([]byte(`{"code":200,"data":{"result":{
		"terminalId":"VG298430009967","appletVersion":"0322",
		"currentRecCount":"0","currentRecMaxCount":"192"}}}`))
	for _, want := range []string{"VG298430009967", "0322", "0/192"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Describe() = %q, missing %q", got, want)
		}
	}
}

// The receipt carries the moment the money was taken, not the moment we managed
// to file it. They differ exactly when the register was unreachable and the
// filing was retried — which is when the difference matters.
func TestMultikassaFilesTheTimeOfTheSale(t *testing.T) {
	when := time.Date(2026, 8, 17, 19, 30, 5, 0, time.UTC)
	body := mkSaleBody(t, Sale{
		Time:  when,
		Lines: []Line{{Name: "Osh", Price: 40_000, Qty: 1}},
		Cash:  40_000,
	})
	if body["receipt_gnk_time"] != "2026-08-17 19:30:05" {
		t.Fatalf("receipt time = %v", body["receipt_gnk_time"])
	}
}

// ⚠️ The morning case. The first sale of every day meets a register whose day
// has not been started, and the refusal is entirely mechanical to fix — so it
// is recognised and answered rather than handed to a cashier as an error.
//
// Matched on the **code**, never the message: the prose arrives in Russian and
// is the vendor's to reword at any release. A match on the words would pass
// here and quietly stop working after an update, with the symptom being that
// every first sale of the day fails until somebody opens the shift by hand.
func TestMultikassaRecognisesAClosedShift(t *testing.T) {
	_, err := mkParse(t, 500,
		`{"code":500,"data":{"message":"#2D - Z-отчет не был открыт"},"success":false}`)
	if !NeedsShift(Multikassa, err) {
		t.Fatalf("a closed shift was not recognised: %v", err)
	}

	// Any other refusal is a real failure and must not be answered by opening a
	// shift — that would file a shift-opening document at the tax committee in
	// response to an unrelated problem.
	_, other := mkParse(t, 500,
		`{"code":500,"data":{"message":"#2B - Не проведено ни одной операции"},"success":false}`)
	if NeedsShift(Multikassa, other) {
		t.Fatal("an unrelated refusal was treated as a closed shift")
	}
	if NeedsShift(FirstOFD, err) {
		t.Fatal("another provider's errors were read with Multikassa's vocabulary")
	}
	if NeedsShift(Multikassa, nil) {
		t.Fatal("a success was treated as a closed shift")
	}
}

// Opening the day is its own operation and carries no goods.
//
// ⚠️ `items` must be `[]` and never `null`: Go marshals a nil slice as null, and
// a register reading null where it expects a list fails inside somebody else's
// parser, where we cannot see it. The same JSON trap that has bitten this
// codebase twice, arriving at a third party.
func TestMultikassaOpenShiftSendsAnEmptyBasket(t *testing.T) {
	enc, _ := EncoderFor(Multikassa, Creds{RegisterID: "UZ21"})
	opener, ok := enc.(ShiftOpener)
	if !ok {
		t.Fatal("the Multikassa adapter cannot open a shift")
	}
	req, err := opener.OpenShift("Aziz", time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("OpenShift: %v", err)
	}
	if strings.Contains(req.Body, `"items":null`) {
		t.Fatalf("nil slice reached the wire:\n%s", req.Body)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body["module_operation_type"] != "1" {
		t.Fatalf("wrong operation: %v", body["module_operation_type"])
	}
	if body["receipt_cashier_name"] != "Aziz" {
		t.Fatalf("the shift is not attributed to anybody: %v", body)
	}
	if items, ok := body["items"].([]any); !ok || len(items) != 0 {
		t.Fatalf("items = %v, want an empty list", body["items"])
	}
}

// ⚠️ A third money format from one vendor, and the one an owner checks against
// real banknotes.
//
// The close-shift reply answers in formatted so'm ("6,651,020.00"); their
// zReport endpoint answers the same figures as bare tiyin ("665102000"); the
// sale request takes tiyin at the top and so'm in the lines. Reading this reply
// with either of the other rules is wrong by 100× or 1000× in a number that
// lands beside a cash count.
func TestMultikassaReadsZReportTotalsAsSoM(t *testing.T) {
	z, ok := ParseZReport([]byte(`{"code":200,"success":true,"data":{"result":{
		"totalSaleCard":"4,036,460.00",
		"totalSaleCash":"6,651,020.00",
		"totalSale":"10,687,480.00",
		"totalSaleCount":"14.00",
		"totalRefund":"1,665,000.00",
		"zCount":"2",
		"openTime":"2024-10-30 10:10:58"}}}`))
	if !ok {
		t.Fatal("a real Z-report was not recognised")
	}
	if z.SaleCash != 6_651_020 {
		t.Fatalf("cash = %d, want 6651020 so'm — not tiyin and not 6", z.SaleCash)
	}
	if z.SaleCard != 4_036_460 || z.SaleTotal != 10_687_480 {
		t.Fatalf("bad totals: %+v", z)
	}
	// ⚠️ Refunds stay their own number. Netting them into the total hides the
	// difference between a quiet day and a day of heavy refunds that happens to
	// balance — and those two want different questions asked.
	if z.RefundTotal != 1_665_000 {
		t.Fatalf("refunds = %d, want 1665000", z.RefundTotal)
	}
	if z.SaleCount != 14 || z.Number != "2" {
		t.Fatalf("bad counts: %+v", z)
	}
}

// Anything that is not a Z-report is not one. Guessing here would write
// zeroes onto a cash shift and make a real day look like an empty one.
func TestMultikassaRejectsNonZReports(t *testing.T) {
	for _, body := range []string{
		`not json`,
		`{"code":500,"success":false,"data":{"message":"#2B - ..."}}`,
		`{"code":200,"data":{"receipt_gnk_fiscalsign":"x"}}`,
	} {
		if _, ok := ParseZReport([]byte(body)); ok {
			t.Fatalf("accepted %q as a Z-report", body)
		}
	}
}

// Closing the day is its own operation, carries no goods, and — like opening —
// must not send `null` for its item list.
func TestMultikassaCloseShiftIsAZReport(t *testing.T) {
	enc, _ := EncoderFor(Multikassa, Creds{RegisterID: "UZ21"})
	closer, ok := enc.(ShiftCloser)
	if !ok {
		t.Fatal("the Multikassa adapter cannot close a shift")
	}
	req, err := closer.CloseShift("Aziz", time.Now())
	if err != nil {
		t.Fatalf("CloseShift: %v", err)
	}
	if strings.Contains(req.Body, `"items":null`) {
		t.Fatalf("nil slice reached the wire:\n%s", req.Body)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body["module_operation_type"] != "2" {
		t.Fatalf("wrong operation: %v", body["module_operation_type"])
	}
}

// ⚠️ **A refund with no original is a different document.**
//
// Every register in the registry files a refund *against* a sale: the state's
// copy has to be able to find what is being undone. The adapter sent
// `module_operation_type: 4` and the lines, and nothing else — so the register
// either refuses it, or accepts it as a standalone negative sale, which
// balances our books and leaves theirs with a refund against nothing. The
// vendor's PDF is explicit: type 4 additionally carries `receipt_sale_id` and a
// `RefundInfo` block.
func TestARefundNamesTheSaleItReverses(t *testing.T) {
	enc, err := newMultikassa(Creds{RegisterID: "VK240813144005"})
	if err != nil {
		t.Fatal(err)
	}
	r := Receipt{
		Cashier: "Kassir", IsRefund: true,
		Time:         time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC),
		ReceivedCash: 4_500_000,
		Items:        []Item{{Name: "Osh", Price: 4_500_000, Qty: 1, VATPercent: 12}},
		Original: OriginalReceipt{
			TerminalID: "VG298430009967", Seq: "23",
			At:   time.Date(2024, 11, 6, 18, 36, 14, 0, time.UTC),
			Sign: "535867058263", SaleID: "sale-1",
		},
	}
	req, err := enc.Sale(r)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		t.Fatal(err)
	}
	info, ok := body["RefundInfo"].(map[string]any)
	if !ok {
		t.Fatalf("no RefundInfo block:\n%s", req.Body)
	}
	// ⚠️ The block's own capitalisation, which is not this API's convention
	// anywhere else — which is exactly why it is easy to get wrong.
	for field, want := range map[string]string{
		"TerminalID": "VG298430009967",
		"ReceiptSeq": "23",
		"FiscalSign": "535867058263",
		// ⚠️ `YYYYMMDDHHMMSS`, not the format every other timestamp uses here.
		"DateTime": "20241106183614",
	} {
		if got, _ := info[field].(string); got != want {
			t.Errorf("RefundInfo.%s = %q, want %q", field, got, want)
		}
	}
	if body["receipt_sale_id"] != "sale-1" {
		t.Errorf("receipt_sale_id = %v", body["receipt_sale_id"])
	}
	// The Postman collection's flat spelling of the same three facts.
	if body["receipt_gnk_fiscalsign"] != "535867058263" {
		t.Errorf("receipt_gnk_fiscalsign missing: %v", body["receipt_gnk_fiscalsign"])
	}
}

// ⚠️ Refused rather than filed as something else. A rejection is visible on the
// till screen with the guest still there; a mis-filing is visible to nobody
// until an inspection.
func TestARefundWithNoFiscalSignIsRefused(t *testing.T) {
	enc, _ := newMultikassa(Creds{RegisterID: "VK1"})
	_, err := enc.Sale(Receipt{
		IsRefund: true,
		Items:    []Item{{Name: "Osh", Price: 1000, Qty: 1}},
	})
	if err == nil {
		t.Fatal("a refund with no original was built anyway")
	}
}

// And a sale carries none of it: a `RefundInfo` on a sale is a field the
// register was not expecting on that operation.
func TestASaleCarriesNoRefundBlock(t *testing.T) {
	enc, _ := newMultikassa(Creds{RegisterID: "VK1"})
	req, err := enc.Sale(Receipt{
		Cashier: "K", ReceivedCash: 100_000,
		Items: []Item{{Name: "Osh", Price: 100_000, Qty: 1, VATPercent: 12}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"RefundInfo", "receipt_sale_id", "receipt_gnk_fiscalsign"} {
		if strings.Contains(req.Body, field) {
			t.Errorf("a sale carries %q:\n%s", field, req.Body)
		}
	}
}
