package sms

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// What an SMS gateway gets wrong quietly is *refusing while looking like it
// accepted*. getsms.uz reports a bad password, an unmoderated nickname and a
// number outside the contract all inside a 200, and OneSignal answers 200 with
// an `errors` field when it accepted nobody. A sender that returns nil there
// tells the site the login code is on its way, and the guest waits for a
// message that was never sent — the single most expensive failure in this
// package. Both are pinned here, along with the phone format each one wants.

func stub(t *testing.T, body string, captured *map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if captured != nil {
			_ = json.Unmarshal(raw, captured)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// ---- getsms.uz ----

func TestGetSMSSendsDocumentedShape(t *testing.T) {
	var sent map[string]any
	srv := stub(t, `[{"recipient":"998901234567","message_id":1,"request_id":7}]`, &sent)

	s := New(Config{
		Provider: ProviderGetSMS, GetSMSURL: srv.URL,
		GetSMSLogin: "user", GetSMSPassword: "pw", GetSMSNickname: "RESTORAN",
	})
	if s.Name() != ProviderGetSMS {
		t.Fatalf("provider = %s, want %s", s.Name(), ProviderGetSMS)
	}
	if err := s.Send(context.Background(), "998901234567", "Kod: 123456"); err != nil {
		t.Fatalf("send: %v", err)
	}
	// Credentials in the body, sender name as "nickname", messages batched.
	for _, k := range []string{"login", "password", "nickname", "data"} {
		if _, ok := sent[k]; !ok {
			t.Errorf("payload missing %q: %v", k, sent)
		}
	}
	data, ok := sent["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("data = %v, want one message", sent["data"])
	}
	msg := data[0].(map[string]any)
	// No "+": this gateway wants the international form without it.
	if msg["phone"] != "998901234567" {
		t.Errorf("phone = %v, want 998901234567 (no plus)", msg["phone"])
	}
}

// The one that matters: a refusal delivered inside a 200 must be an error.
func TestGetSMSErrorInsideOKIsAFailure(t *testing.T) {
	cases := map[string]string{
		"error_text": `[{"error":1,"error_text":"invalid password","error_no":101}]`,
		"error_only": `[{"error":"1","error_text":""}]`,
		"not_json":   `some error occurred`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			srv := stub(t, body, nil)
			s := New(Config{
				Provider: ProviderGetSMS, GetSMSURL: srv.URL,
				GetSMSLogin: "user", GetSMSPassword: "pw",
			})
			if err := s.Send(context.Background(), "998901234567", "Kod: 1"); err == nil {
				t.Fatal("want an error, got nil — the site would report a code it never sent")
			}
		})
	}
}

func TestGetSMSSuccessIsNotMistakenForFailure(t *testing.T) {
	// error_no: 0 and an absent error field both mean "delivered".
	srv := stub(t, `[{"recipient":"998901234567","error_no":0,"message_id":9}]`, nil)
	s := New(Config{
		Provider: ProviderGetSMS, GetSMSURL: srv.URL,
		GetSMSLogin: "user", GetSMSPassword: "pw",
	})
	if err := s.Send(context.Background(), "998901234567", "Kod: 1"); err != nil {
		t.Fatalf("send: %v", err)
	}
}

// ---- OneSignal ----

func TestOneSignalSendsE164AndKeyAuth(t *testing.T) {
	var sent map[string]any
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &sent)
		io.WriteString(w, `{"id":"abc","recipients":1}`)
	}))
	t.Cleanup(srv.Close)

	s := New(Config{
		Provider: ProviderOneSignal, OneSignalBaseURL: srv.URL,
		OneSignalAppID: "app-uuid", OneSignalAPIKey: "secret", OneSignalFrom: "+15551234567",
	})
	if err := s.Send(context.Background(), "998901234567", "Kod: 123456"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !strings.HasPrefix(auth, "Key ") {
		t.Errorf("Authorization = %q, want the \"Key <api key>\" scheme", auth)
	}
	if sent["target_channel"] != "sms" {
		t.Errorf("target_channel = %v, want sms", sent["target_channel"])
	}
	// The one conversion this package does: everything else stores 998…, and
	// OneSignal is the only gateway that insists on the plus.
	nums, _ := sent["include_phone_numbers"].([]any)
	if len(nums) != 1 || nums[0] != "+998901234567" {
		t.Errorf("include_phone_numbers = %v, want [+998901234567]", sent["include_phone_numbers"])
	}
}

func TestOneSignalErrorsFieldIsAFailure(t *testing.T) {
	srv := stub(t, `{"id":"","errors":["All included players are not subscribed"]}`, nil)
	s := New(Config{
		Provider: ProviderOneSignal, OneSignalBaseURL: srv.URL,
		OneSignalAppID: "app", OneSignalAPIKey: "secret",
	})
	if err := s.Send(context.Background(), "998901234567", "Kod: 1"); err == nil {
		t.Fatal("want an error: OneSignal accepted nobody but answered 200")
	}
}

// ---- choosing a provider ----

// Half-filled credentials must degrade to demo rather than take login down —
// and Demo() must stay true so the code is still visible to whoever fixes it.
func TestIncompleteCredentialsFallBackToDemo(t *testing.T) {
	cases := map[string]Config{
		"eskiz no password":  {Provider: ProviderEskiz, EskizEmail: "a@b.uz"},
		"playmobile no pass": {Provider: ProviderPlayMobile, PlayMobileLogin: "user"},
		"getsms no login":    {Provider: ProviderGetSMS, GetSMSPassword: "pw"},
		"onesignal no key":   {Provider: ProviderOneSignal, OneSignalAppID: "app"},
		"unknown provider":   {Provider: "twilio"},
		"empty":              {},
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			if s := New(cfg); !s.Demo() {
				t.Fatalf("want the demo sender, got %s", s.Name())
			}
		})
	}
}

func TestMissingNamesTheCredentialThatIsAbsent(t *testing.T) {
	if got := (Config{Provider: ProviderOneSignal, OneSignalAppID: "app"}).Missing(); got == "" {
		t.Fatal("want a reason naming the missing API key")
	}
	full := Config{
		Provider: ProviderGetSMS, GetSMSLogin: "user", GetSMSPassword: "pw",
	}
	if got := full.Missing(); got != "" {
		t.Fatalf("Missing() = %q, want empty for complete credentials", got)
	}
}
