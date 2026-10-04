package push

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// ⚠️ **What is worth sealing here is the same pair the Expo test seals, plus
// the routing.** A token deleted over the wrong error unsubscribes a working
// phone and nobody reports it, because the app looks fine; and a message posted
// to the wrong transport is accepted politely and delivered nowhere.

func serviceAccount(t *testing.T, tokenURI string) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	blob, _ := json.Marshal(map[string]string{
		"type":         "service_account",
		"project_id":   "keel-test",
		"client_email": "push@keel-test.iam.gserviceaccount.com",
		"private_key":  string(pemKey),
		"token_uri":    tokenURI,
	})
	return string(blob)
}

// oauthStub answers the token exchange, so the tests never leave the machine.
func oauthStub(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "stub-bearer", "expires_in": 3600,
		})
	}))
}

func reset() {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()
	fcm.creds, fcm.token, fcm.warned = nil, "", false
}

func TestOnlyAnUnregisteredTokenIsPrunedFromFCM(t *testing.T) {
	defer reset()
	oauth := oauthStub(t)
	defer oauth.Close()

	send := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Message struct {
				Token string `json:"token"`
			} `json:"message"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		switch in.Message.Token {
		case deadToken:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{
				"status": "NOT_FOUND", "message": "gone",
				"details": []map[string]any{{"errorCode": "UNREGISTERED"}},
			}})
		case badPayloadToken:
			// ⚠️ The trap: INVALID_ARGUMENT is what a malformed *message*
			// answers too. Pruning on it lets one bad release delete every
			// phone in the restaurant.
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{
				"status": "INVALID_ARGUMENT", "message": "bad",
				"details": []map[string]any{{"errorCode": "INVALID_ARGUMENT"}},
			}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "ok"})
		}
	}))
	defer send.Close()

	if err := Configure(serviceAccount(t, oauth.URL)); err != nil {
		t.Fatal(err)
	}
	fcmSendURL = send.URL + "/v1/projects/%s/messages:send"
	defer func() { fcmSendURL = "https://fcm.googleapis.com/v1/projects/%s/messages:send" }()

	dead := sendFCM(context.Background(), []Message{
		{To: liveToken}, {To: deadToken}, {To: badPayloadToken},
	}, func(string, ...any) {})

	if len(dead) != 1 || dead[0] != deadToken {
		t.Fatalf("dead = %v, want only the unregistered device", dead)
	}
}

func TestAnUnconfiguredProjectPrunesNothingAndWarnsOnce(t *testing.T) {
	defer reset()
	reset()
	var lines int
	var mu sync.Mutex
	log := func(string, ...any) { mu.Lock(); lines++; mu.Unlock() }

	for i := 0; i < 3; i++ {
		if got := sendFCM(context.Background(), []Message{{To: liveToken}}, log); len(got) != 0 {
			t.Fatalf("dead = %v, want nothing pruned when we are the problem", got)
		}
	}
	// ⚠️ A line per kitchen event would bury the log it belongs in, and an
	// install with only Expo builds is in this state permanently.
	if lines != 1 {
		t.Fatalf("logged %d times, want exactly one warning", lines)
	}
}

// ⚠️ **The routing is the thing that is invisible when it is wrong.** Expo
// answers a POST carrying an FCM token with a cheerful "ok" and delivers
// nothing at all, which is a working server, a registered phone, and silence.
func TestEachTokenGoesToItsOwnTransport(t *testing.T) {
	expo, native := route([]Message{
		{To: "ExponentPushToken[aaaaaaaaaaaaaaaaaaaa]"},
		{To: liveToken},
		{To: "ExpoPushToken[bbbbbbbbbbbbbbbbbbbb]"},
	})
	if len(expo) != 2 {
		t.Fatalf("expo = %v", expo)
	}
	if len(native) != 1 || native[0].To != liveToken {
		t.Fatalf("native = %v", native)
	}
}

func TestAnExpoTokenIsNeverMistakenForAnFCMOne(t *testing.T) {
	// Long enough to pass the length floor, and still Expo's.
	long := "ExponentPushToken[" + strings.Repeat("x", 80) + "]"
	if IsFCMToken(long) {
		t.Fatal("an Expo token was routed to Firebase")
	}
	if !IsPushToken(long) || !IsPushToken(liveToken) {
		t.Fatal("a usable token was refused at registration")
	}
	for _, bad := range []string{"", "short", "has space " + liveToken, "<script>"} {
		if IsPushToken(bad) {
			t.Fatalf("%q was accepted", bad)
		}
	}
}

// ⚠️ Both blocks, because they answer two different states of the phone: the
// data half is what a foregrounded app draws, and the notification half is what
// survives a battery saver when the app is not in front.
func TestTheEnvelopeCarriesBothHalvesAndOnlyStrings(t *testing.T) {
	env := fcmEnvelope(Message{
		To: liveToken, Title: "Tayyor", Body: "Lag'mon",
		ChannelID: KitchenChannel, Data: map[string]any{"checkId": "abc", "n": 3},
	})
	if _, ok := env["notification"]; !ok {
		t.Fatal("no notification block: a backgrounded phone shows nothing")
	}
	data, _ := env["data"].(map[string]string)
	if data["checkId"] != "abc" || data["n"] != "3" {
		t.Fatalf("data = %v, want every value a string", data)
	}
	if data["title"] != "Tayyor" {
		t.Fatal("a foregrounded app has no words to draw")
	}
	if data["channel"] != KitchenChannel {
		t.Fatal("no channel in data: Keel cannot tell whose notification a tap was")
	}
	android, _ := env["android"].(map[string]any)
	note, _ := android["notification"].(map[string]any)
	if note["channel_id"] != KitchenChannel {
		t.Fatal("wrong channel: the message arrives silent and unranked")
	}
}

const (
	liveToken       = "fZ1x_live:APA91bF" + "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	deadToken       = "fZ1x_dead:APA91bF" + "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	badPayloadToken = "fZ1x_bad_:APA91bF" + "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
)
