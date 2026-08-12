package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// The gate must turn away the (limit+1)th request from one IP inside the
// window, and must not touch a different IP — an attacker walking a thousand
// phone numbers from one machine is exactly what it is for, and a shared office
// NAT hitting the wall must not lock out the guest next to them on another one.
func TestRateLimitPerIP(t *testing.T) {
	gate := NewRateLimit(3, time.Minute)
	handler := gate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func(ip string) int {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = ip + ":54321"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	for i := range 3 {
		if got := call("10.0.0.1"); got != http.StatusOK {
			t.Fatalf("request %d was blocked (%d), want 200", i+1, got)
		}
	}
	if got := call("10.0.0.1"); got != http.StatusTooManyRequests {
		t.Fatalf("the 4th request returned %d, want 429", got)
	}
	// A different machine is unaffected.
	if got := call("10.0.0.2"); got != http.StatusOK {
		t.Fatalf("a second IP was blocked (%d) by the first one's spend", got)
	}
}

// The window resets: a limit that never let up would lock out a real user for
// good after one bad minute.
func TestRateLimitWindowResets(t *testing.T) {
	gate := NewRateLimit(1, 20*time.Millisecond)
	handler := gate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))

	call := func() int {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "10.0.0.9:1"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if call() != http.StatusOK {
		t.Fatal("first call blocked")
	}
	if call() != http.StatusTooManyRequests {
		t.Fatal("second call in the window was allowed")
	}
	time.Sleep(30 * time.Millisecond)
	if call() != http.StatusOK {
		t.Fatal("the window did not reset")
	}
}
