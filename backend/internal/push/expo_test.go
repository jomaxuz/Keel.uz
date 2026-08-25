package push

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ⚠️ **What is worth sealing here is the pruning, not the sending.** A phone
// that was reinstalled keeps a row in the database, and every kitchen event
// then pays for a delivery nobody receives — quietly, forever. And the mirror
// of it: deleting a token over a rate limit would unsubscribe a working phone,
// which nobody would report because the app looks fine.

func TestOnlyAPermanentlyDeadTokenIsReportedBack(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"status": "ok"},
				{"status": "error", "message": "not registered",
					"details": map[string]any{"error": "DeviceNotRegistered"}},
				{"status": "error", "message": "slow down",
					"details": map[string]any{"error": "MessageRateExceeded"}},
			},
		})
	}))
	defer srv.Close()
	endpoint = srv.URL
	defer func() { endpoint = "https://exp.host/--/api/v2/push/send" }()

	dead := Send(context.Background(), []Message{
		{To: "ExponentPushToken[a]"},
		{To: "ExponentPushToken[b]"},
		{To: "ExponentPushToken[c]"},
	}, func(string, ...any) {})

	if len(dead) != 1 || dead[0] != "ExponentPushToken[b]" {
		t.Fatalf("dead = %v, want only the unregistered device", dead)
	}
}

func TestAnUnreachableRelayLosesNothingAndDeletesNothing(t *testing.T) {
	// ⚠️ The event this announces is already written down. Failing to say so
	// must not cost a token, and must not be retried later — a notification
	// that arrives twenty minutes late sends a waiter to the pass for a dish
	// that was collected.
	endpoint = "http://127.0.0.1:1/nowhere"
	defer func() { endpoint = "https://exp.host/--/api/v2/push/send" }()

	dead := Send(context.Background(),
		[]Message{{To: "ExponentPushToken[a]"}}, func(string, ...any) {})
	if len(dead) != 0 {
		t.Fatalf("dead = %v, want nothing pruned over an outage", dead)
	}
}

func TestAnEmptySendIsNotACall(t *testing.T) {
	endpoint = "http://127.0.0.1:1/nowhere"
	defer func() { endpoint = "https://exp.host/--/api/v2/push/send" }()
	if got := Send(context.Background(), nil, func(string, ...any) {}); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestWhatCanBeAnExpoToken(t *testing.T) {
	// Checked before storing: an unusable token otherwise sits in the
	// collection forever, failing once per kitchen event.
	for _, ok := range []string{
		"ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]",
		"ExpoPushToken[yyyyyyyyyyyyyyyyyyyyyy]",
	} {
		if !IsExpoToken(ok) {
			t.Fatalf("%q was refused", ok)
		}
	}
	for _, bad := range []string{
		"",
		"fcm-token-from-somewhere-else",
		"ExponentPushToken[unterminated",
		"<script>",
	} {
		if IsExpoToken(bad) {
			t.Fatalf("%q was accepted", bad)
		}
	}
}
