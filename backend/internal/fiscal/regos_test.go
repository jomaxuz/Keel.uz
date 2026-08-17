package fiscal

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func regosBody(t *testing.T, s Sale) map[string]any {
	t.Helper()
	enc, err := EncoderFor(Regos, Creds{Login: "kassa", Password: "kas123456", RegisterID: "1"})
	if err != nil {
		t.Fatalf("EncoderFor: %v", err)
	}
	req, err := enc.Sale(Build(s))
	if err != nil {
		t.Fatalf("Sale: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(req.Body), &out); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	return out
}

func regosParams(t *testing.T, s Sale) map[string]any {
	t.Helper()
	return regosBody(t, s)["params"].(map[string]any)
}

// ⚠️ **Three different scalings in one request, and none of them is obvious.**
// Each produces a document that passes every eyeball check and is wrong:
//
//   - money in tiyin (900000 = 9 000 so'm)
//   - quantity in **thousandths** (1000 = one portion)
//   - VAT as **percent × 100** (1200 = 12%)
//
// Sending a whole number as a quantity files a receipt for a thousandth of a
// portion, and the total still looks right because it is sent separately.
func TestRegosScalesQuantityAndVAT(t *testing.T) {
	p := regosParams(t, Sale{
		Lines:      []Line{{Name: "Coca-cola", Price: 9_000, Qty: 2}},
		Cash:       18_000,
		VatPercent: 12,
	})
	pos := p["positions"].([]any)[0].(map[string]any)

	if got := pos["amount"]; got != float64(1_800_000) {
		t.Fatalf("amount = %v, want 1800000 tiyin (18 000 so'm)", got)
	}
	if got := pos["quantity"]; got != float64(2000) {
		t.Fatalf("quantity = %v, want 2000 — one portion is 1000", got)
	}
	if got := pos["vat_value"]; got != float64(1200) {
		t.Fatalf("vat_value = %v, want 1200 — 12%% is 1200, not 12", got)
	}
}

// ⚠️ Both halves of a split payment are sent. A guest paying part cash and part
// card is ordinary in a hall, and collapsing it to one line would file a
// receipt that disagrees with the drawer.
func TestRegosSendsBothPaymentHalves(t *testing.T) {
	p := regosParams(t, Sale{
		Lines:      []Line{{Name: "Osh", Price: 50_000, Qty: 1}},
		Cash:       20_000,
		Card:       30_000,
		VatPercent: 12,
	})
	pays := p["payments"].([]any)
	if len(pays) != 2 {
		t.Fatalf("got %d payments, want 2: %v", len(pays), pays)
	}
	byType := map[float64]float64{}
	for _, x := range pays {
		m := x.(map[string]any)
		byType[m["type"].(float64)] = m["value"].(float64)
	}
	if byType[1] != 2_000_000 || byType[2] != 3_000_000 {
		t.Fatalf("cash/card = %v/%v tiyin, want 2000000/3000000", byType[1], byType[2])
	}
}

// ⚠️ Our order number goes in `code`, which REGOS uses for its duplicate check.
// That is the point: a filing retried after a lost reply is refused by the
// register itself rather than filed twice.
func TestRegosSendsTheOrderNumberAsTheDuplicateKey(t *testing.T) {
	p := regosParams(t, Sale{
		OrderNumber: "MRC-A1-1745",
		Lines:       []Line{{Name: "Choy", Price: 5_000, Qty: 1}},
		Cash:        5_000,
	})
	if p["code"] != "MRC-A1-1745" {
		t.Fatalf("code = %v, want the order number", p["code"])
	}
}

// Auth is Base64(login:password) in the **body**, not a header — unusual, and
// the reason this provider's drawer carries a login where Multikassa's carries
// only an address.
func TestRegosAuthIsBase64InTheBody(t *testing.T) {
	b := regosBody(t, Sale{
		Lines: []Line{{Name: "Choy", Price: 5_000, Qty: 1}}, Cash: 5_000,
	})
	want := base64.StdEncoding.EncodeToString([]byte("kassa:kas123456"))
	if b["auth"] != want {
		t.Fatalf("auth = %v, want %q", b["auth"], want)
	}
	if b["jsonrpc"] != "2.0" || b["method"] != "Receipt.Sale" {
		t.Fatalf("not a JSON-RPC sale: %v", b)
	}
}

// An absent classifier code is absent, never an empty string: a blank ИКПУ on a
// filed receipt is a claim about a product.
func TestRegosOmitsAnUnknownClassifier(t *testing.T) {
	p := regosParams(t, Sale{
		Lines: []Line{{Name: "Choy", Price: 5_000, Qty: 1}}, Cash: 5_000,
	})
	pos := p["positions"].([]any)[0].(map[string]any)
	if _, ok := pos["icps"]; ok {
		t.Fatalf("an empty ИКПУ was sent: %v", pos)
	}
	if _, ok := pos["package_code"]; ok {
		t.Fatalf("an empty package code was sent: %v", pos)
	}
}

func regosParse(t *testing.T, body string) (Result, error) {
	t.Helper()
	enc, _ := EncoderFor(Regos, Creds{Login: "a", Password: "b"})
	return enc.Parse(200, []byte(body))
}

// ⚠️ **`ok` decides, not the HTTP status.** A refusal arrives inside a 200, and
// reading the status would call every business refusal a success.
func TestRegosReadsTheOkFlagNotTheStatus(t *testing.T) {
	_, err := regosParse(t, `{"id":1,"ok":false,"result":"Смена не открыта","jsonrpc":"2.0"}`)
	if err == nil {
		t.Fatal("a refusal inside a 200 was read as a success")
	}
	if !strings.Contains(err.Error(), "Смена") {
		t.Fatalf("the register's own words were lost: %v", err)
	}
}

// A filing counts only if a fiscal sign came back — the thing the whole
// operation exists to obtain.
func TestRegosNeedsAFiscalSign(t *testing.T) {
	if _, err := regosParse(t, `{"id":1,"ok":true,"result":{}}`); err == nil {
		t.Fatal("a reply with no fiscal sign was accepted as filed")
	}

	res, err := regosParse(t, `{"id":1,"ok":true,"result":{
		"Id":"59683513-a845-5ce3-bf75-4f35f7a7d846",
		"Amount":900000,
		"QRCodeURL":"https://ofd.soliq.uz/check?t=VG369473163905&r=583",
		"TerminalID":"VG369473163905","ReceiptNo":"759",
		"FiscalSign":"848035748636"},"jsonrpc":"2.0"}`)
	if err != nil {
		t.Fatalf("a real filing was rejected: %v", err)
	}
	if res.FiscalSign != "848035748636" {
		t.Fatalf("fiscal sign = %q", res.FiscalSign)
	}
	if !strings.HasPrefix(res.QRText, "https://ofd.soliq.uz/check?") {
		t.Fatalf("the guest's QR was lost: %q", res.QRText)
	}
}

// Non-JSON is a failure, never a refusal shown as one — the paytest lesson.
func TestRegosRejectsNonJSON(t *testing.T) {
	enc, _ := EncoderFor(Regos, Creds{Login: "a", Password: "b"})
	if _, err := enc.Parse(502, []byte("<html>Bad Gateway</html>")); err == nil {
		t.Fatal("an HTML error page was accepted")
	}
}

// ⚠️ The connection check uses Sys.GetInfo, one of the three methods the
// documentation says works **without a printer** — which is what separates
// "cannot reach the register" from "the register is there and its printer is
// not".
func TestRegosHelloWorksWithoutAPrinter(t *testing.T) {
	enc, _ := EncoderFor(Regos, Creds{Login: "a", Password: "b"})
	req, err := enc.Hello()
	if err != nil {
		t.Fatalf("Hello: %v", err)
	}
	if !strings.Contains(req.Body, "Sys.GetInfo") {
		t.Fatalf("the connection check needs a printer: %s", req.Body)
	}
}

// Opening and closing the register's day, through the same interfaces the till
// already drives for Multikassa.
func TestRegosOpensAndClosesTheDay(t *testing.T) {
	enc, _ := EncoderFor(Regos, Creds{Login: "a", Password: "b"})
	opener, ok := enc.(ShiftOpener)
	if !ok {
		t.Fatal("REGOS cannot open a shift")
	}
	closer, ok := enc.(ShiftCloser)
	if !ok {
		t.Fatal("REGOS cannot close a shift")
	}
	open, _ := opener.OpenShift("Aziz", timeZero())
	closed, _ := closer.CloseShift("Aziz", timeZero())
	if !strings.Contains(open.Body, "ZReport.Open") {
		t.Fatalf("open: %s", open.Body)
	}
	if !strings.Contains(closed.Body, "ZReport.Close") {
		t.Fatalf("close: %s", closed.Body)
	}
}


func timeZero() time.Time { return time.Time{} }
