package didox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// ⚠️ **Two headers, two owners, and both go on every request.** The partner
// token is Keel's, the user key is the customer's session. A request missing
// either is answered 401 by the operator — and the two failures look identical
// from here, which is why they are separate fields in the settings and why this
// test names both.
func TestBothKeysTravelWithEveryRequest(t *testing.T) {
	var gotPartner, gotUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPartner = r.Header.Get("Partner-Authorization")
		gotUser = r.Header.Get("user-key")
		_, _ = w.Write([]byte(`{"data":[],"total":0}`))
	}))
	defer srv.Close()

	c := New(true, "partner-token", "user-token")
	c.BaseURL = srv.URL
	if _, _, err := c.List(context.Background(), ListFilter{}); err != nil {
		t.Fatal(err)
	}
	if gotPartner != "partner-token" || gotUser != "user-token" {
		t.Fatalf("headers = %q / %q", gotPartner, gotUser)
	}
}

// ⚠️ **`page` and `limit` are required by the operator**, and `owner` is
// written from the caller's meaning rather than from theirs: `owner=1` is
// *ours*, so the post arriving is `owner=0`. Reading it the natural way round
// would show an owner their own outgoing invoices under the heading "kelgan
// hujjatlar" — every number real, every one of them from the wrong direction.
func TestTheInboxAsksForTheIncomingPost(t *testing.T) {
	var q url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q = r.URL.Query()
		_, _ = w.Write([]byte(`{"data":[],"total":0}`))
	}))
	defer srv.Close()
	c := New(true, "p", "u")
	c.BaseURL = srv.URL

	if _, _, err := c.List(context.Background(), ListFilter{
		From: "2026-08-01", To: "2026-09-01", Types: []string{"002", "008"},
	}); err != nil {
		t.Fatal(err)
	}
	if q.Get("owner") != "0" {
		t.Errorf("owner = %q, want 0 (the post arriving)", q.Get("owner"))
	}
	if q.Get("page") == "" || q.Get("limit") == "" {
		t.Error("page and limit are required and were not sent")
	}
	if q.Get("doctype") != "002,008" {
		t.Errorf("doctype = %q", q.Get("doctype"))
	}
	if q.Get("docDateFromCreated") != "2026-08-01" {
		t.Errorf("the window was not sent by the document's own date: %v", q)
	}

	// And the outgoing side asks for the other direction.
	if _, _, err := c.List(context.Background(), ListFilter{Outgoing: true}); err != nil {
		t.Fatal(err)
	}
	if q.Get("owner") != "1" {
		t.Errorf("outgoing owner = %q, want 1", q.Get("owner"))
	}
}

// ⚠️ **The operator's own sentence is what reaches the screen.** "Ulanmadi"
// sends an owner to us; "User not registered" sends them to their accountant,
// which is where the fix is. They answer in three shapes and the interesting
// messages arrive in the ones a single-shape parser drops.
func TestTheOperatorsOwnWordsSurvive(t *testing.T) {
	for _, c := range []struct {
		body string
		want string
	}{
		{`{"message":"Пользователь заблокирован"}`, "Пользователь заблокирован"},
		{`{"error":"User not registered"}`, "User not registered"},
		{`"Unauthorized. Invalid signature"`, "Unauthorized. Invalid signature"},
		{`plain text failure`, "plain text failure"},
	} {
		if got := message([]byte(c.body)); got != c.want {
			t.Errorf("message(%s) = %q, want %q", c.body, got, c.want)
		}
	}
}

// ⚠️ **401 and 403 both mean "log in again".** The operator answers 403 to an
// expired session on some endpoints; treating only 401 as expiry leaves an
// integration that looks alive and imports nothing until somebody re-saves the
// settings.
func TestAnExpiredSessionIsRecognisedByBothCodes(t *testing.T) {
	for _, code := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		e := &Error{Status: code}
		if !e.Unauthorized() {
			t.Errorf("%d was not read as an expired session", code)
		}
	}
	if (&Error{Status: 422}).Unauthorized() {
		t.Error("a 422 was read as an expired session — that one is the customer's own data")
	}
}

// The token arrives at the top level on some builds and under `data` on
// others; a client that knows one shape logs in successfully and then behaves
// as if it had not.
func TestTheTokenIsFoundInEitherShape(t *testing.T) {
	for _, body := range []string{
		`{"token":"abc-123"}`,
		`{"data":{"token":"abc-123"}}`,
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var in map[string]string
			_ = json.NewDecoder(r.Body).Decode(&in)
			if in["password"] != "hunter22" {
				t.Errorf("the password was not sent: %v", in)
			}
			_, _ = w.Write([]byte(body))
		}))
		c := New(true, "p", "")
		c.BaseURL = srv.URL
		got, err := c.LoginByPassword(context.Background(), "302936161", "hunter22")
		srv.Close()
		if err != nil {
			t.Fatal(err)
		}
		if got != "abc-123" {
			t.Errorf("token = %q from %s", got, body)
		}
	}
	// A 200 with no token is an error rather than an empty session: carrying on
	// would send unauthenticated requests that fail one screen later.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := New(true, "p", "")
	c.BaseURL = srv.URL
	if _, err := c.LoginByPassword(context.Background(), "1", "2"); err == nil {
		t.Error("a login with no token in the answer was accepted")
	}
}

// A document's lines come back parsed, and the numbers arrive as the operator
// sends them: strings for money, a number for the count.
func TestADocumentComesBackWithItsLines(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("owner"); got != "0" {
			t.Errorf("a document was asked for as %q", got)
		}
		_, _ = w.Write([]byte(`{"data":{"json":{
			"ProductList":{"Products":[{"OrdNo":1,"Name":"Un","Count":25,
			 "Summa":"5200.00","DeliverySum":"130000.00",
			 "DeliverySumWithVat":"145600.00","PackageName":"kg"}]},
			"FacturaDoc":{"FacturaNo":"SF-9","FacturaDate":"2026-09-05"}}}}`))
	}))
	defer srv.Close()
	c := New(true, "p", "u")
	c.BaseURL = srv.URL
	doc, err := c.Get(context.Background(), "ABC", false)
	if err != nil {
		t.Fatal(err)
	}
	f, err := ParseFactura(doc.JSON)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.ProductList.Products) != 1 || f.FacturaDoc.FacturaNo != "SF-9" {
		t.Fatalf("document did not survive the round trip: %+v", f)
	}
	if got := ParseMoney(f.ProductList.Products[0].DeliverySumWithVat); got != 145600 {
		t.Errorf("line total = %v", got)
	}
}
