package instore

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// captured is one request the adapter made, as the bank would have seen it.
type captured struct {
	Method string
	Path   string
	Auth   string
	Body   map[string]any
}

// bank stands in for the provider: it records what arrived and answers with
// whatever the test wants.
func bank(t *testing.T, reply string, got *captured) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			got.Method, got.Path = r.Method, r.URL.Path
			got.Auth = r.Header.Get("Auth")
			if got.Auth == "" {
				got.Auth = r.Header.Get("Authorization")
			}
			raw, _ := io.ReadAll(r.Body)
			if len(raw) > 0 {
				_ = json.Unmarshal(raw, &got.Body)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, reply)
		}))
	t.Cleanup(srv.Close)
	return srv
}

// ⚠️ **The most expensive mistake available in this package: the unit.**
// CLICK Pass takes so'm and Uzum FastPay takes tiyin, and the two adapters sit
// next to each other. Getting it backwards charges a guest a hundred times the
// bill or a hundredth of it — and neither one fails, neither one logs, and the
// only place it shows up is the guest's card statement or the restaurant's
// month end. So the conversion is pinned in both directions.
func TestTheAmountIsInEachProvidersOwnUnit(t *testing.T) {
	var click captured
	srv := bank(t, `{"error_code":0,"payment_id":7,"payment_status":2}`, &click)
	c := &clickPass{
		cfg:  Config{ServiceID: "12345", UserID: "9", SecretKey: "s"},
		http: srv.Client(), now: time.Now,
	}
	// The host is not overridable on the real adapter (there is no sandbox), so
	// the test drives call() through a client whose transport goes to httptest.
	c.http = redirect(srv.URL, srv.Client())
	if _, err := c.Charge(context.Background(), Charge{
		Amount: 45_000, OTPData: strings.Repeat("1", 43), OrderID: "A-12", TxnID: "t1",
	}); err != nil {
		t.Fatalf("charge: %v", err)
	}
	if got := click.Body["amount"]; got != float64(45_000) {
		t.Fatalf("CLICK Pass was sent %v, not 45000 so'm", got)
	}

	var uzum captured
	usrv := bank(t, `{"error_code":0,"payment_id":"p1","payment_status":"SUCCESS"}`, &uzum)
	u := &uzumFastPay{
		cfg:  Config{ServiceID: "1", UserID: "8461", SecretKey: "s", BaseURL: usrv.URL},
		http: usrv.Client(), now: time.Now,
	}
	if _, err := u.Charge(context.Background(), Charge{
		Amount: 45_000, OTPData: strings.Repeat("1", 43), OrderID: "A-12", TxnID: "t1",
	}); err != nil {
		t.Fatalf("charge: %v", err)
	}
	if got := uzum.Body["amount"]; got != float64(4_500_000) {
		t.Fatalf("Uzum FastPay was sent %v, not 4500000 tiyin", got)
	}
}

// ⚠️ **Uzum rejects the header on a regex, and a 401 reads exactly like a wrong
// secret key.** So the shape is pinned rather than trusted: three colon-
// separated fields, the middle one exactly forty hex characters, the last one
// milliseconds. A helper that prefixed "Bearer" — the reflex — turns every call
// into an authentication failure nobody would look for here.
func TestUzumAuthHeaderMatchesTheBanksRegex(t *testing.T) {
	var got captured
	srv := bank(t, `{"error_code":0}`, &got)
	u := &uzumFastPay{
		cfg:  Config{ServiceID: "1", UserID: "8461", SecretKey: "s", BaseURL: srv.URL},
		http: srv.Client(), now: time.Now,
	}
	_, _ = u.Status(context.Background(), "p1", "A-1")

	if !regexp.MustCompile(`^\d*:[0-9a-f]{40}:\d*$`).MatchString(got.Auth) {
		t.Fatalf("Uzum would answer 401 to %q", got.Auth)
	}
	ms, err := strconv.ParseInt(got.Auth[strings.LastIndex(got.Auth, ":")+1:], 10, 64)
	if err != nil {
		t.Fatalf("timestamp: %v", err)
	}
	// ⚠️ Milliseconds, not seconds. A seconds value is a valid-looking ten
	// digits, signs correctly, and comes back as **403** — "more than 50
	// seconds between the header and processing" — which reads like a slow
	// network rather than a unit mistake.
	if time.Since(time.UnixMilli(ms)) > time.Minute {
		t.Fatalf("timestamp %d is not milliseconds", ms)
	}
}

// CLICK counts seconds, in a header with a different name. Same failure shape,
// opposite unit — pinned for the same reason.
func TestClickAuthHeaderIsSecondsAndNamedAuth(t *testing.T) {
	var got captured
	srv := bank(t, `{"error_code":0,"payment_status":2}`, &got)
	c := &clickPass{
		cfg:  Config{ServiceID: "1", UserID: "9", SecretKey: "s"},
		http: redirect(srv.URL, srv.Client()), now: time.Now,
	}
	_, _ = c.Status(context.Background(), "7", "A-1")

	if got.Auth == "" {
		t.Fatal("CLICK Pass sent no Auth header")
	}
	parts := strings.Split(got.Auth, ":")
	if len(parts) != 3 || len(parts[1]) != 40 {
		t.Fatalf("Auth is not merchant_user_id:digest:timestamp — %q", got.Auth)
	}
	sec, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || time.Since(time.Unix(sec, 0)) > time.Minute {
		t.Fatalf("timestamp %q is not UNIX seconds", parts[2])
	}
}

// ⚠️ **`payment_status` 0 and 1 are not paid.** They mean the processing centre
// is still deciding, and a till that reads either as success hands food over on
// a card that may yet decline. Only 2 is money.
func TestClickOnlyStatusTwoIsMoney(t *testing.T) {
	for status, want := range map[int]string{
		0: StatusPending, 1: StatusPending, 2: StatusPaid, -5: StatusFailed,
	} {
		got := clickStatusOf(clickReply{PaymentStatus: status})
		if got != want {
			t.Fatalf("payment_status %d read as %q, want %q", status, got, want)
		}
	}
	// And an error code overrides a status that looks fine.
	if clickStatusOf(clickReply{ErrorCode: -5, PaymentStatus: 2}) != StatusFailed {
		t.Fatal("a refused payment was read as paid because the status said 2")
	}
}

// ⚠️ **Uzum answers 200 even when it refuses**, so the success test is
// `error_code == 0` and never the HTTP status. Reading the status first would
// mark every decline as paid.
func TestUzumRefusalInsideATwoHundred(t *testing.T) {
	var got captured
	srv := bank(t, `{"error_code":400,"error_message":"apelsin.pay.user.otp.data.expired"}`, &got)
	u := &uzumFastPay{
		cfg:  Config{ServiceID: "1", UserID: "8", SecretKey: "s", BaseURL: srv.URL},
		http: srv.Client(), now: time.Now,
	}
	res, err := u.Charge(context.Background(), Charge{
		Amount: 1000, OTPData: strings.Repeat("1", 43), OrderID: "A-1", TxnID: "t",
	})
	if res.Status != StatusFailed {
		t.Fatalf("a refused charge read as %q", res.Status)
	}
	// And the cashier is told what to do, not what the bank called it.
	if err == nil || !strings.Contains(err.Error(), "yangisini") {
		t.Fatalf("an expired QR was reported as %v", err)
	}
}

// ⚠️ **CLICK's confirm mode is a thirty-second fuse.** When the service is
// configured for it, an unconfirmed payment is reversed by the bank — so a
// charge that came back with confirm_mode set and was never confirmed is a
// cashier told "paid" over money that is about to go back.
func TestClickConfirmModeIsConfirmedBeforeAnythingElse(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			paths = append(paths, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w,
				`{"error_code":0,"payment_id":7,"payment_status":2,"confirm_mode":1}`)
		}))
	defer srv.Close()
	c := &clickPass{
		cfg:  Config{ServiceID: "1", UserID: "9", SecretKey: "s"},
		http: redirect(srv.URL, srv.Client()), now: time.Now,
	}
	res, err := c.Charge(context.Background(), Charge{
		Amount: 1000, OTPData: "x", OrderID: "A-1", TxnID: "t",
	})
	if err != nil {
		t.Fatalf("charge: %v", err)
	}
	if res.Status != StatusPaid {
		t.Fatalf("a confirmed payment read as %q", res.Status)
	}
	if len(paths) != 2 || !strings.HasSuffix(paths[1], "/click_pass/confirm") {
		t.Fatalf("confirm was not sent: %v", paths)
	}
}

// ⚠️ **A provider with no adapter must fail loudly.** The quiet version — a nil
// driver, a no-op charge — leaves a restaurant believing it is taking cards.
func TestAProviderWithNoAdapterRefuses(t *testing.T) {
	for _, p := range Providers() {
		_, err := New(Config{
			Provider: p.ID, ServiceID: "1", UserID: "1", SecretKey: "s",
		})
		if p.Ready && err != nil {
			t.Fatalf("%s is listed as ready but cannot be built: %v", p.ID, err)
		}
		if !p.Ready && err == nil {
			t.Fatalf("%s has no adapter but built one anyway", p.ID)
		}
	}
}

// ⚠️ **Every listed provider names the boxes it needs**, and the ready ones can
// be built from exactly those. The fiscal settings shipped a provider whose
// credential field was missing from the panel: it could be selected, saved and
// enabled, and simply never authenticated — with no error anywhere. Same test,
// same reason.
func TestEveryProviderAsksForWhatItNeeds(t *testing.T) {
	for _, p := range Providers() {
		if len(p.Needs) == 0 {
			t.Fatalf("%s asks for no credentials at all", p.ID)
		}
		if p.Name == "" || p.Note == "" {
			t.Fatalf("%s has no name or no explanation on the settings page", p.ID)
		}
		if p.Kind != KindScan && p.Kind != KindTerminal {
			t.Fatalf("%s has no kind, so no screen knows how to offer it", p.ID)
		}
	}
}

// redirect points an adapter with a hard-coded host at the test server.
//
// ⚠️ CLICK Pass has no configurable base URL on purpose — no sandbox is
// published, and a box whose only correct value is the default eventually holds
// something else. So the test rewrites the destination in the transport rather
// than adding a field to production code for the benefit of a test.
func redirect(to string, base *http.Client) *http.Client {
	return &http.Client{Transport: rewriteTo{to, base.Transport}}
}

type rewriteTo struct {
	host string
	next http.RoundTripper
}

func (r rewriteTo) RoundTrip(req *http.Request) (*http.Response, error) {
	u := *req.URL
	target, err := http.NewRequest(req.Method, r.host+u.Path, req.Body)
	if err != nil {
		return nil, err
	}
	target.Header = req.Header
	next := r.next
	if next == nil {
		next = http.DefaultTransport
	}
	return next.RoundTrip(target)
}

// ⚠️ **A product's barcode is not a payment code, and the till should say so
// before the bank does.** FastPay documents a floor of forty characters; a
// cashier who scans the wrong thing — a barcode, a loyalty card, a CLICK code
// into the Uzum field — otherwise waits for a round trip to be told "wrong
// prefix" in a machine word. Checked against the live error table
// (developer.uzumbank.uz/fastpay, re-read 2026-09-07).
func TestAShortCodeIsRefusedWithoutAskingTheBank(t *testing.T) {
	charger, err := New(Config{
		Provider: UzumFastPay, ServiceID: "1", UserID: "8461",
		SecretKey: "secret", Cashbox: "kassa-1",
		// ⚠️ A host nothing may reach: the point of the guard is that a short
		// code never leaves the building, so a call would fail the test by
		// failing to connect rather than by reaching Uzum.
		BaseURL: "http://127.0.0.1:1",
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := charger.Charge(context.Background(), Charge{
		Amount: 10000, OTPData: "4780123456789", OrderID: "1", TxnID: "t",
	})
	if err == nil {
		t.Fatal("a thirteen-digit barcode was accepted as a payment code")
	}
	if res.Status != StatusFailed {
		t.Errorf("status %q, want failed", res.Status)
	}
	// And the sentence is the cashier's next action, not the bank's word.
	if !strings.Contains(err.Error(), "QR") {
		t.Errorf("message %q does not tell the cashier what to do", err)
	}
}
