package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSignVerifiesAndRefusesTampering(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	body := []byte(`{"id":"evt_1"}`)
	h := Sign("whsec_x", now.Unix(), body)
	if !strings.HasPrefix(h, "t=1700000000,v1=") {
		t.Fatalf("header shape: %s", h)
	}
	if !Verify("whsec_x", h, body, now, 5*time.Minute) {
		t.Fatal("own signature did not verify")
	}
	if Verify("whsec_y", h, body, now, 5*time.Minute) {
		t.Fatal("wrong secret verified")
	}
	if Verify("whsec_x", h, []byte(`{"id":"evt_2"}`), now, 5*time.Minute) {
		t.Fatal("changed body verified")
	}
	// ⚠️ The replay the timestamp exists to stop.
	if Verify("whsec_x", h, body, now.Add(time.Hour), 5*time.Minute) {
		t.Fatal("an hour-old delivery verified")
	}
}

// A vector computed outside Go (Python's hmac): a receiver in any language
// must arrive at the same string, and the docs quote this one.
func TestSignKnownVector(t *testing.T) {
	got := Sign("secret", 1, []byte("{}"))
	want := "t=1,v1=1122767b193110cfec322b6f199b599edbf608ed087f2d27afb0b97d99523908"
	if got != want {
		t.Fatalf("got %s", got)
	}
}

func TestCheckURLRefusesWhatIsBesideUs(t *testing.T) {
	bad := []string{
		"http://example.com/hook", // not https
		"https://localhost/hook",
		"https://mongo:27017/",
		"https://keel-control/x",
		"https://127.0.0.1/x",
		"https://10.0.0.5/x",
		"https://169.254.169.254/latest/meta-data",
		"https://[::1]/x",
		"https://user:pass@example.com/x",
		"ftp://example.com/x",
		"not a url",
	}
	for _, u := range bad {
		if _, err := CheckURL(u); err == nil {
			t.Errorf("%s was accepted", u)
		}
	}
	for _, u := range []string{"https://example.com/hook", "https://api.partner.uz:8443/keel?x=1"} {
		if _, err := CheckURL(u); err != nil {
			t.Errorf("%s refused: %v", u, err)
		}
	}
}

// ⚠️ The dial-time check is what stops a name that *resolves* somewhere
// private — the test server here is on 127.0.0.1, which is exactly that.
func TestClientRefusesPrivateAddressAtDial(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("the request reached a loopback server")
	}))
	defer srv.Close()
	res := Send(context.Background(), Client, Request{URL: srv.URL, Secret: "s", Body: []byte("{}")}, time.Now())
	if res.OK() || !strings.HasPrefix(res.Err, "blocked") {
		t.Fatalf("got %+v", res)
	}
}

func TestSendSignsAndReportsStatus(t *testing.T) {
	var gotSig, gotEvent string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get(HeaderSignature)
		gotEvent = r.Header.Get(HeaderEvent)
		buf := make([]byte, 100)
		n, _ := r.Body.Read(buf)
		gotBody = buf[:n]
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()
	now := time.Now()
	// The ordinary client for a loopback test server; the guard is tested above.
	res := Send(context.Background(), srv.Client(), Request{
		URL: srv.URL, Secret: "s", Event: "order.created", Body: []byte(`{"a":1}`),
	}, now)
	if !res.OK() || res.StatusCode != http.StatusAccepted {
		t.Fatalf("got %+v", res)
	}
	if gotEvent != "order.created" || !Verify("s", gotSig, gotBody, now, time.Minute) {
		t.Fatalf("event %q sig %q body %s", gotEvent, gotSig, gotBody)
	}
}

func TestNonSuccessIsAFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "http://mongo:27017/")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()
	c := srv.Client()
	c.CheckRedirect = Client.CheckRedirect
	res := Send(context.Background(), c, Request{URL: srv.URL, Secret: "s", Body: []byte("{}")}, time.Now())
	if res.OK() || res.StatusCode != http.StatusFound {
		t.Fatalf("a redirect was followed or counted as success: %+v", res)
	}
}

func TestBackoffEnds(t *testing.T) {
	if d, ok := Backoff(1); !ok || d != time.Minute {
		t.Fatalf("first retry: %v %v", d, ok)
	}
	if _, ok := Backoff(MaxAttempts); ok {
		t.Fatal("retries never end")
	}
}
