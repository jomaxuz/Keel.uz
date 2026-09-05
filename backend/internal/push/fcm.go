package push

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ---- Firebase Cloud Messaging, HTTP v1 ----
//
// ⚠️ **Added beside Expo's relay, never in place of it.** Four of the five
// phone applications are still Expo builds and their tokens only Expo can
// deliver to; the native waiter app holds an FCM registration token, which the
// relay refuses. Replacing one transport with the other would have silenced the
// courier, the team, the owner and the television on the day the waiter app
// shipped — and silently, because a notification nobody receives raises nothing.
//
// ⚠️ **Which transport is decided by the token's own shape, not by a setting.**
// A flag would be a second fact that can disagree with the first, and the
// disagreement is invisible: the row still says "registered". A token either
// begins `ExponentPushToken[` or it does not, and that is not a matter of
// opinion. See `route` in send.go.
//
// ⚠️ **No Firebase SDK.** The whole of this is a signed assertion, one token
// exchange and one POST per message — against a dependency that pulls in
// google-api-go-client, the OAuth libraries and their transitive half. The
// signing is `golang-jwt/jwt/v5`, which this server already uses for its own
// sessions.

var (
	fcmSendURL  = "https://fcm.googleapis.com/v1/projects/%s/messages:send"
	oauthScope  = "https://www.googleapis.com/auth/firebase.messaging"
	httpTimeout = 10 * time.Second
)

// credentials is the part of a Google service-account JSON that matters here.
type credentials struct {
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	TokenURI    string `json:"token_uri"`

	key *rsa.PrivateKey
}

// fcm holds what the process needs to send, and the access token it is reusing.
//
// ⚠️ **Package state rather than a parameter threaded through thirty call
// sites.** There is one Firebase project per deployment for the whole life of
// the process, and `notifyStaff`, `notifyCourier` and the rest already take no
// configuration. Guarded because the access token is refreshed from whichever
// goroutine notices it has expired.
var fcm struct {
	mu      sync.Mutex
	creds   *credentials
	token   string
	expires time.Time
	// ⚠️ Said once, not once per notification. A deployment that has not
	// configured FCM is the ordinary state for every restaurant still on the
	// Expo build, and a line per kitchen event would bury the log it belongs in.
	warned bool
}

// Configure loads the service account this process sends as.
//
// `raw` is either the JSON itself or a path to it — both, because a container
// gets the value through the environment and a VPS gets it as a mounted file,
// and asking which is a question with two right answers and one variable.
//
// ⚠️ **An unconfigured process is not a broken one.** Every install that has
// only Expo builds in the field is correct with this empty, so this reports the
// problem and returns; nothing here is allowed to stop a server from starting.
func Configure(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	blob := []byte(raw)
	if !strings.HasPrefix(raw, "{") {
		b, err := os.ReadFile(raw)
		if err != nil {
			return fmt.Errorf("fcm hisob fayli o'qilmadi: %w", err)
		}
		blob = b
	}
	var c credentials
	if err := json.Unmarshal(blob, &c); err != nil {
		return fmt.Errorf("fcm hisob fayli JSON emas: %w", err)
	}
	if c.ProjectID == "" || c.ClientEmail == "" || c.PrivateKey == "" {
		return errors.New("fcm hisob faylida project_id, client_email yoki private_key yo'q")
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(c.PrivateKey))
	if err != nil {
		return fmt.Errorf("fcm kaliti o'qilmadi: %w", err)
	}
	c.key = key
	if c.TokenURI == "" {
		c.TokenURI = "https://oauth2.googleapis.com/token"
	}

	fcm.mu.Lock()
	defer fcm.mu.Unlock()
	fcm.creds = &c
	fcm.token = ""
	fcm.expires = time.Time{}
	return nil
}

// FCMReady reports whether this process can deliver to a native app.
//
// ⚠️ Read by the registration handlers so a phone is told at sign-in rather
// than left registered against a transport that does not exist. A token stored
// against a server that cannot send is the failure this whole file exists to
// stop being silent.
func FCMReady() bool {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()
	return fcm.creds != nil
}

// accessToken returns a valid bearer, minting one when the last has expired.
//
// ⚠️ **Refreshed a minute early.** A token that expires between the check and
// the request comes back as 401, which is indistinguishable here from a
// misconfigured service account — and the wrong diagnosis for that is expensive
// to reach.
func accessToken(ctx context.Context) (string, string, error) {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()
	if fcm.creds == nil {
		return "", "", errors.New("fcm sozlanmagan")
	}
	c := fcm.creds
	if fcm.token != "" && time.Now().Before(fcm.expires.Add(-time.Minute)) {
		return fcm.token, c.ProjectID, nil
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   c.ClientEmail,
		"scope": oauthScope,
		"aud":   c.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	assertion, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(c.key)
	if err != nil {
		return "", "", err
	}

	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertion},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.TokenURI,
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := (&http.Client{Timeout: httpTimeout}).Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", "", err
	}
	if out.AccessToken == "" {
		return "", "", fmt.Errorf("fcm tokeni olinmadi: %s %s", out.Error, out.Description)
	}
	fcm.token = out.AccessToken
	fcm.expires = now.Add(time.Duration(out.ExpiresIn) * time.Second)
	return fcm.token, c.ProjectID, nil
}

// How many messages are in flight at once.
//
// ⚠️ **FCM v1 has no batch endpoint** — the one that existed was withdrawn — so
// a table of eight dishes is eight requests. Eight at a time, because the fan-out
// here is a restaurant's staff rather than a mailing list, and an unbounded
// goroutine per device turns a busy service into a burst the far end rate-limits.
const fcmParallel = 8

// sendFCM delivers to native apps and reports the tokens Google called dead.
func sendFCM(ctx context.Context, msgs []Message, log func(string, ...any)) []string {
	if len(msgs) == 0 {
		return nil
	}
	bearer, project, err := accessToken(ctx)
	if err != nil {
		fcm.mu.Lock()
		warned := fcm.warned
		fcm.warned = true
		fcm.mu.Unlock()
		if !warned {
			log("fcm yuborilmadi: %v", err)
		}
		// ⚠️ Nothing pruned. The tokens are fine; we are not.
		return nil
	}

	url := fmt.Sprintf(fcmSendURL, project)
	client := &http.Client{Timeout: httpTimeout}

	var (
		mu   sync.Mutex
		dead []string
		wg   sync.WaitGroup
	)
	gate := make(chan struct{}, fcmParallel)

	for i := range msgs {
		m := msgs[i]
		wg.Add(1)
		gate <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-gate }()
			permanent, err := postOne(ctx, client, url, bearer, m)
			if err != nil {
				log("fcm rad etildi: %v", err)
			}
			if permanent {
				mu.Lock()
				dead = append(dead, m.To)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return dead
}

// postOne sends one message. It returns whether the token is permanently dead.
func postOne(
	ctx context.Context, client *http.Client, url, bearer string, m Message,
) (bool, error) {
	body, err := json.Marshal(map[string]any{"message": fcmEnvelope(m)})
	if err != nil {
		return false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearer)

	res, err := client.Do(req)
	if err != nil {
		// Unreachable. ⚠️ Not retried, for the reason the Expo path gives: the
		// event is minutes old, and a notification that arrives late sends a
		// waiter to the pass for a dish that was collected.
		return false, err
	}
	defer res.Body.Close()
	if res.StatusCode < 300 {
		return false, nil
	}

	var out struct {
		Error struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Details []struct {
				ErrorCode string `json:"errorCode"`
			} `json:"details"`
		} `json:"error"`
	}
	_ = json.NewDecoder(res.Body).Decode(&out)

	code := ""
	for _, d := range out.Error.Details {
		if d.ErrorCode != "" {
			code = d.ErrorCode
			break
		}
	}
	// ⚠️ **Only two codes are permanent, and `INVALID_ARGUMENT` is not one of
	// them.** It is what a malformed *payload* answers as well as a malformed
	// token — so treating it as fatal would let one bad message delete every
	// phone in the restaurant, on an evening nobody would connect to a release.
	// Same rule as the Expo path, where only `DeviceNotRegistered` counts.
	permanent := code == "UNREGISTERED" || code == "SENDER_ID_MISMATCH" ||
		(code == "" && res.StatusCode == http.StatusNotFound)
	return permanent, fmt.Errorf("%s %s (%s)", out.Error.Status, out.Error.Message, code)
}

// fcmEnvelope turns one Message into FCM's shape.
//
// ⚠️ **Both a `notification` block and a `data` one, and that pairing is the
// decision.** Data alone is what a phone with the app open handles best — our own
// service draws it — but a data-only message is throttled or dropped outright by
// the battery savers the phones sold here ship with, which is exactly the
// condition a waiter's phone is in mid-shift. With the notification block the
// system draws it when the app is not in front, and Firebase copies the data keys
// into the tap intent — which is how `MainActivity` still receives `checkId` and
// opens the table rather than the room.
func fcmEnvelope(m Message) map[string]any {
	// ⚠️ Every data value must be a string: FCM refuses the message outright
	// otherwise, and it refuses the whole batch of one — a number here is a
	// notification nobody gets.
	data := map[string]string{}
	for k, v := range m.Data {
		data[k] = fmt.Sprint(v)
	}
	// Carried in `data` as well so a foregrounded app draws the same words the
	// system tray would have.
	if m.Title != "" {
		data["title"] = m.Title
	}
	if m.Body != "" {
		data["body"] = m.Body
	}

	android := map[string]any{
		// ⚠️ "high" for the kitchen, and this is the case it is for: a dish at
		// the pass is going cold while the phone decides whether to wake up.
		"priority": "HIGH",
		"notification": map[string]any{
			// ⚠️ The channel the phone actually created. A message sent to a
			// channel that was never created arrives silent and unranked, which
			// looks exactly like a notification nobody sent.
			"channel_id": m.ChannelID,
			"sound":      "default",
		},
	}

	env := map[string]any{
		"token":   m.To,
		"data":    data,
		"android": android,
	}
	if m.Title != "" || m.Body != "" {
		env["notification"] = map[string]any{"title": m.Title, "body": m.Body}
	}
	return env
}
