package push

import (
	"context"
	"strings"
)

// ---- Which transport a token belongs to ----
//
// ⚠️ **One `Send`, two transports, and no caller knows the difference.**
// `notifyStaff`, `notifyCourier` and the owner's alerts all hand over a list of
// messages and delete whatever comes back dead. Teaching each of them which app
// a device is running would be three copies of one rule, and the copy that goes
// stale is the one that stops notifying somebody.
//
// ⚠️ **Split by the token, not by a stored flag.** A column saying "this is an
// FCM device" is a second fact that can disagree with the first — and the
// disagreement is invisible, because the row still reads as registered. A token
// either begins `ExponentPushToken[` or it does not.

// Send delivers messages and reports the tokens that are permanently dead.
//
// The two transports are asked independently: an Expo outage must not stop the
// native phones being told, and an unconfigured Firebase project must not stop
// the four Expo apps that are the whole field today.
func Send(ctx context.Context, msgs []Message, log func(string, ...any)) []string {
	if len(msgs) == 0 {
		return nil
	}
	expo, native := route(msgs)

	dead := []string{}
	if len(expo) > 0 {
		dead = append(dead, sendExpo(ctx, expo, log)...)
	}
	if len(native) > 0 {
		dead = append(dead, sendFCM(ctx, native, log)...)
	}
	return dead
}

// route splits a batch by the shape of its tokens.
func route(msgs []Message) (expo, native []Message) {
	for _, m := range msgs {
		if IsExpoToken(m.To) {
			expo = append(expo, m)
			continue
		}
		native = append(native, m)
	}
	return expo, native
}

// IsFCMToken reports whether a string can be an FCM registration token.
//
// ⚠️ **Deliberately loose.** Google has changed this format more than once and
// documents no grammar for it; a strict pattern would refuse a real phone, and
// the person holding it has no way to tell that from "notifications are off".
// A stored token that turns out to be nonsense costs one failed delivery and is
// then pruned by `UNREGISTERED` — which is the cheap direction to be wrong in.
func IsFCMToken(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 64 || len(s) > 1024 {
		return false
	}
	if strings.ContainsAny(s, " \t\r\n\"'<>") {
		return false
	}
	return !IsExpoToken(s)
}

// IsPushToken reports whether a token is one this server can deliver to.
//
// ⚠️ **Checked at registration rather than at send time**, the rule the Expo
// path already had: an unusable token otherwise sits in the collection forever,
// failing quietly once per kitchen event.
func IsPushToken(s string) bool {
	return IsExpoToken(s) || IsFCMToken(s)
}
