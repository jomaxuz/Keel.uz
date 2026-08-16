package handlers

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

// ⚠️ A guest must never be shown the gateway's own words.
//
// The public code endpoint used to return them: on an unmoderated Eskiz account
// somebody typing their number into the login form received
// `SMS yuborilmadi: eskiz send: 400 {"message":"Для теста можно…","id":"6bf8…"}`.
// Wrong twice over — unreadable to the person it is shown to, and it publishes
// the restaurant's gateway state to anybody who can reach a login form.
func TestGatewayWordsNeverReachTheGuest(t *testing.T) {
	raw := `eskiz send: 400 {"id":"6bf81cd4","message":"Для теста можно ` +
		`использовать только один из этих текстов","status":"error"}`

	w := httptest.NewRecorder()
	smsRequestFailed(w, smsSendError{err: errors.New(raw)})

	body := w.Body.String()
	for _, leaked := range []string{"eskiz", "Для теста", "6bf81cd4", "400"} {
		if strings.Contains(body, leaked) {
			t.Fatalf("gateway detail %q leaked to the guest: %s", leaked, body)
		}
	}
	// And it still tells the person what to do, which is the only thing that
	// makes a refusal better than silence.
	if !strings.Contains(body, "restoran bilan bog'laning") {
		t.Fatalf("the guest needs a next step, got: %s", body)
	}
	// ⚠️ 503, not 429. Everything from issueCode used to answer "too many
	// requests" — including a gateway that refused on the very first try,
	// which tells the guest to wait for something that will never change.
	if w.Code != 503 {
		t.Fatalf("a gateway refusal is 503, got %d", w.Code)
	}
}

// The cooldown is the caller's own doing, so it is explained in full: it says
// how many seconds are left, and that is the helpful answer.
func TestCooldownIsStillExplained(t *testing.T) {
	w := httptest.NewRecorder()
	smsRequestFailed(w, errors.New("kod yaqinda yuborilgan, 42 soniyadan keyin qayta urining"))

	if w.Code != 429 {
		t.Fatalf("the cooldown is 429, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "42") {
		t.Fatalf("the wait must survive: %s", w.Body.String())
	}
}

// Wrapping must survive: recordSMSFailure returns the typed error, and anything
// that wraps it further has to keep answering "was this the gateway".
func TestSendErrorUnwraps(t *testing.T) {
	inner := errors.New("connection refused")
	wrapped := smsSendError{err: inner}
	if !errors.Is(wrapped, inner) {
		t.Fatal("the cause must stay reachable for the log")
	}
	var se smsSendError
	if !errors.As(error(wrapped), &se) {
		t.Fatal("the marker must survive errors.As — smsRequestFailed depends on it")
	}
}
