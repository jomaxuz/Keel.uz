// Package push sends notifications to the staff applications.
//
// ⚠️ **Expo's service, not Firebase directly, and that is a deliberate trade.**
// A token issued by the app's own build is delivered by Expo to whichever
// transport the device uses — FCM on Android, APNs on iOS — with no credentials
// on our side and no per-platform branch here. What it costs is a dependency on
// somebody else's relay staying up; what it buys is that a restaurant's
// notifications do not stop the day an Apple certificate quietly expires,
// because there is no certificate here to expire.
//
// ⚠️ **A failed notification is never a failed sale.** Everything here is called
// after the thing it announces has already happened and been written down. It
// logs and returns; it must never be the reason an order does not save.
package push

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// endpoint is Expo's push API. A variable so a test can point it at itself.
var endpoint = "https://exp.host/--/api/v2/push/send"

// Message is one notification, in Expo's shape.
type Message struct {
	To    string         `json:"to"`
	Title string         `json:"title,omitempty"`
	Body  string         `json:"body,omitempty"`
	Sound string         `json:"sound,omitempty"`
	Data  map[string]any `json:"data,omitempty"`
	// ⚠️ Android needs a channel or the notification arrives silent and
	// unranked. The app creates one with this id at startup; the two spellings
	// have to agree, and this is one of them.
	ChannelID string `json:"channelId,omitempty"`
	// ⚠️ **"high" for the kitchen, and not by reflex.** A dish going cold at the
	// pass is worth waking a screen for; anything that can wait until somebody
	// next looks at their phone must not use this, or the priority stops
	// meaning anything.
	Priority string `json:"priority,omitempty"`
}

// KitchenChannel is the Android channel the app registers for kitchen news.
const KitchenChannel = "kitchen"

// IsExpoToken reports whether a string can be an Expo push token.
//
// ⚠️ Checked before storing rather than before sending: an unusable token
// otherwise sits in the collection forever, failing quietly once per event.
func IsExpoToken(s string) bool {
	s = strings.TrimSpace(s)
	return (strings.HasPrefix(s, "ExponentPushToken[") ||
		strings.HasPrefix(s, "ExpoPushToken[")) &&
		strings.HasSuffix(s, "]") && len(s) < 200
}

// Send delivers messages, in batches, and reports the tokens the service
// rejected as permanently invalid.
//
// ⚠️ **The rejected tokens are the useful half of the answer.** A phone that was
// reinstalled or had the app removed keeps a row in the database, and without
// pruning it every kitchen event pays for a delivery nobody receives. The
// caller deletes what comes back.
func Send(ctx context.Context, msgs []Message, log func(string, ...any)) []string {
	dead := []string{}
	if len(msgs) == 0 {
		return dead
	}
	client := &http.Client{Timeout: 10 * time.Second}
	// Expo takes up to 100 per request; the batch is smaller because a
	// restaurant has tens of staff, not thousands, and a smaller body fails
	// faster on a bad connection.
	const batch = 50
	for start := 0; start < len(msgs); start += batch {
		end := start + batch
		if end > len(msgs) {
			end = len(msgs)
		}
		body, err := json.Marshal(msgs[start:end])
		if err != nil {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint,
			bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			// The relay is unreachable. ⚠️ Nothing is retried: the event this
			// announces is minutes old, and a notification that arrives late is
			// worse than one that does not arrive — a waiter walks to the pass
			// for a dish collected twenty minutes ago.
			log("push yuborilmadi: %v", err)
			continue
		}
		var parsed struct {
			Data []struct {
				Status  string `json:"status"`
				Message string `json:"message"`
				Details struct {
					Error string `json:"error"`
				} `json:"details"`
			} `json:"data"`
		}
		_ = json.NewDecoder(res.Body).Decode(&parsed)
		res.Body.Close()
		for i, d := range parsed.Data {
			if d.Status == "ok" {
				continue
			}
			// ⚠️ Only "DeviceNotRegistered" is permanent. A rate limit or a
			// message-too-big is our problem to fix, and deleting the token
			// over it would silently unsubscribe a working phone.
			if d.Details.Error == "DeviceNotRegistered" && start+i < len(msgs) {
				dead = append(dead, msgs[start+i].To)
				continue
			}
			log("push rad etildi: %s (%s)", d.Message, d.Details.Error)
		}
	}
	return dead
}
