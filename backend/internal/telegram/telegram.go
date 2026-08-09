// Package telegram verifies what Telegram says about a person, and talks to a
// restaurant's own bot.
//
// The whole point of this package is one function: `Verify`. Inside a Telegram
// mini app the browser hands us a string that claims "this is user 12345, named
// Ali". Believing it would mean anybody can be anybody — the string is entirely
// under the client's control — so it comes signed with the bot's own token, and
// **the signature is the login**.
//
// ⚠️ This is the same shape as the demo-SMS hole that shipped once: a request
// that succeeds, a response that looks right, and an identity nobody proved. The
// difference is that here the proof exists and is cheap; it just has to actually
// be checked, on the server, every time.
package telegram

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// How old a signed payload may be.
//
// Telegram's `auth_date` is what makes a captured `initData` string stop working:
// without an age check the string is a **permanent** password for that account,
// and it sits in the browser where any extension or shared screenshot can take
// it. A day is long enough that a mini app left open over lunch keeps working,
// and short enough that a leaked string is worthless by tomorrow.
const MaxAuthAge = 24 * time.Hour

var (
	// ErrNoToken means the restaurant has not connected a bot. Not a failure of
	// this package: the site simply has no Telegram login yet.
	ErrNoToken = errors.New("telegram: bot tokeni sozlanmagan")
	// ErrBadSignature is the one that matters. It is returned for a tampered
	// payload, a payload signed by a *different* bot, and a missing hash alike —
	// the caller has no business telling those apart, and neither does an
	// attacker.
	ErrBadSignature = errors.New("telegram: imzo to'g'ri kelmadi")
	ErrStale        = errors.New("telegram: ma'lumot eskirgan")
)

// Verify checks a Telegram-signed payload and returns its fields.
//
// The algorithm is Telegram's, and both halves matter:
//
//  1. the signing key is `HMAC_SHA256("WebAppData", botToken)` — not the token
//     itself, which is what makes a key stolen from one context useless in
//     another;
//  2. the signed string is every field except `hash`, as `k=v`, **sorted by
//     key**, joined with newlines. Sorting is not cosmetic: Telegram signs that
//     exact string, so any other order produces a different digest and every
//     login fails.
//
// Comparison is constant-time. The endpoint is public and a byte-wise compare
// leaks how much of a guess was right.
func Verify(payload, botToken string) (map[string]string, error) {
	if strings.TrimSpace(botToken) == "" {
		return nil, ErrNoToken
	}
	values, err := url.ParseQuery(payload)
	if err != nil {
		return nil, ErrBadSignature
	}
	got := values.Get("hash")
	if got == "" {
		return nil, ErrBadSignature
	}

	fields := make(map[string]string, len(values))
	pairs := make([]string, 0, len(values))
	for k, v := range values {
		if k == "hash" || len(v) == 0 {
			continue
		}
		fields[k] = v[0]
		pairs = append(pairs, k+"="+v[0])
	}
	sort.Strings(pairs)

	secret := hmacSHA256([]byte("WebAppData"), []byte(botToken))
	want := hmacSHA256(secret, []byte(strings.Join(pairs, "\n")))
	if subtle.ConstantTimeCompare([]byte(hex.EncodeToString(want)), []byte(got)) != 1 {
		return nil, ErrBadSignature
	}
	return fields, nil
}

// CheckFresh rejects a payload that is too old to be a login.
//
// Separate from `Verify` because they fail for different reasons and a caller
// may want to say so differently: a bad signature is somebody trying, a stale
// one is usually a mini app that sat open for a day and needs reopening.
func CheckFresh(fields map[string]string, now time.Time, maxAge time.Duration) error {
	raw := fields["auth_date"]
	if raw == "" {
		// A payload with no date cannot be aged out, which makes it a permanent
		// password. Refused rather than trusted.
		return ErrStale
	}
	sec, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return ErrStale
	}
	at := time.Unix(sec, 0)
	if now.Sub(at) > maxAge {
		return ErrStale
	}
	// A little clock skew in the future is normal; a lot is a forged date on a
	// payload somebody hopes will still work next year.
	if at.Sub(now) > 5*time.Minute {
		return ErrStale
	}
	return nil
}

// User is the person Telegram named, as far as we are willing to record.
//
// ⚠️ **There is no phone number here, and that is not an omission.** Telegram
// never gives one with `initData`: it hands over an id and a display name. An
// order needs a phone, so the mini app has to ask for it separately (see
// `ParseContact`) — and a design that assumed otherwise would fail at the
// checkout of the first real order.
type User struct {
	ID        int64
	FirstName string
	LastName  string
	Username  string
	// Telegram's own UI language, so a mini app can open in it rather than in
	// whatever the last visitor picked.
	Lang string
}

// ParseUser reads the `user` field of a verified payload.
//
// Called only with fields that came back from `Verify`. Passing unverified input
// here would parse an attacker's JSON into a logged-in identity, which is the
// entire failure this package exists to prevent — so it takes the map rather
// than the raw string, and the map can only be produced by a successful verify.
func ParseUser(fields map[string]string) (*User, error) {
	raw := fields["user"]
	if raw == "" {
		return nil, errors.New("telegram: foydalanuvchi ma'lumoti yo'q")
	}
	var u struct {
		ID           int64  `json:"id"`
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		Username     string `json:"username"`
		LanguageCode string `json:"language_code"`
	}
	if err := json.Unmarshal([]byte(raw), &u); err != nil {
		return nil, errors.New("telegram: foydalanuvchi ma'lumoti o'qilmadi")
	}
	if u.ID == 0 {
		return nil, errors.New("telegram: foydalanuvchi id yo'q")
	}
	return &User{
		ID: u.ID, FirstName: u.FirstName, LastName: u.LastName,
		Username: u.Username, Lang: u.LanguageCode,
	}, nil
}

// Contact is a phone number Telegram vouched for.
type Contact struct {
	UserID int64
	Phone  string
}

// ParseContact reads a phone number out of a verified payload.
//
// ⚠️ **The field names here are the one thing in this package that has not been
// checked against a live bot.** The signature scheme is Telegram's documented
// one and is exercised by tests; the shape of the contact response is read
// leniently for that reason — `contact` as JSON, and the flat `phone_number`
// some clients send instead. Both are accepted, neither is assumed.
//
// Whatever the shape, the trust story is unchanged: the payload is verified
// first, so a phone number that arrives here was signed by the restaurant's own
// bot rather than typed by whoever is holding the browser. That makes it
// **stronger** evidence than an SMS code, not weaker.
func ParseContact(fields map[string]string) (*Contact, error) {
	if raw := fields["contact"]; raw != "" {
		var c struct {
			UserID      int64  `json:"user_id"`
			PhoneNumber string `json:"phone_number"`
		}
		if err := json.Unmarshal([]byte(raw), &c); err == nil && c.PhoneNumber != "" {
			return &Contact{UserID: c.UserID, Phone: c.PhoneNumber}, nil
		}
	}
	if phone := fields["phone_number"]; phone != "" {
		id, _ := strconv.ParseInt(fields["user_id"], 10, 64)
		return &Contact{UserID: id, Phone: phone}, nil
	}
	return nil, errors.New("telegram: telefon raqami topilmadi")
}

// ---- Talking to the bot ----

const apiBase = "https://api.telegram.org"

// Me is what `getMe` answers: proof that a token works, and the bot's username.
type Me struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Name     string `json:"first_name"`
}

// GetMe proves the token and returns who it belongs to.
//
// This is the settings page's check button, and it answers the question a filled
// form cannot: a token that looks right and belongs to a deleted bot is
// indistinguishable from a working one until the first guest tries to sign in.
// Returning the **username** matters too — the mini app's deep link is built
// from it, so a page that only said "connected" would leave the operator to find
// it themselves.
func GetMe(ctx context.Context, token string) (*Me, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrNoToken
	}
	var out struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
		Result      Me     `json:"result"`
	}
	if err := call(ctx, token, "getMe", nil, &out); err != nil {
		return nil, err
	}
	if !out.OK {
		return nil, fmt.Errorf("telegram: %s", out.Description)
	}
	return &out.Result, nil
}

// SendMessage tells a guest what happened to their order.
//
// Here because it is the reason a restaurant wants a bot at all: an order update
// through Telegram costs nothing, and the same message as an SMS costs money
// every time. Kept in this package rather than in a handler so the token never
// travels further than it has to.
func SendMessage(ctx context.Context, token string, chatID int64, text string) error {
	var out struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	body := map[string]any{
		"chat_id": chatID,
		"text":    text,
		// Plain text: a restaurant's dish name can contain any character, and an
		// unescaped underscore in Markdown mode makes Telegram reject the whole
		// message. A notification that silently does not arrive is worse than
		// one without bold text.
		"disable_notification": false,
	}
	if err := call(ctx, token, "sendMessage", body, &out); err != nil {
		return err
	}
	if !out.OK {
		return fmt.Errorf("telegram: %s", out.Description)
	}
	return nil
}

func call(ctx context.Context, token, method string, body any, out any) error {
	var reader *strings.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = strings.NewReader(string(raw))
	} else {
		reader = strings.NewReader("")
	}
	url := fmt.Sprintf("%s/bot%s/%s", apiBase, token, method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	// Decoded whatever the status: Telegram puts the useful sentence in the body
	// even on a 4xx, and "401 Unauthorized" tells an operator far less than
	// "Unauthorized: bot token is invalid".
	return json.NewDecoder(res.Body).Decode(out)
}

func hmacSHA256(key, data []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(data)
	return m.Sum(nil)
}

// Sign produces a payload the way Telegram would. Test-only helper, exported so
// the handler tests can build a valid login without a live bot.
func Sign(fields map[string]string, botToken string) string {
	pairs := make([]string, 0, len(fields))
	for k, v := range fields {
		pairs = append(pairs, k+"="+v)
	}
	sort.Strings(pairs)
	secret := hmacSHA256([]byte("WebAppData"), []byte(botToken))
	sum := hmacSHA256(secret, []byte(strings.Join(pairs, "\n")))

	q := url.Values{}
	for k, v := range fields {
		q.Set(k, v)
	}
	q.Set("hash", hex.EncodeToString(sum))
	return q.Encode()
}

// ---- The bot's incoming side ----
//
// Until this existed the bot could only **talk**: it sent order updates and
// nothing listened. A guest who found the bot and pressed Start got silence,
// which is not a missing feature — it reads as a broken restaurant, and it is
// the first thing anybody does with a bot.
//
// Webhook rather than long polling, and the reason is the shape of this product:
// one container per restaurant. Polling would mean every tenant holding an open
// request to Telegram forever, awake and costing memory whether or not that
// restaurant has a single guest. A webhook costs nothing until somebody writes.

// WebhookUpdates is what Telegram is asked to deliver.
//
// ⚠️ **Exported and tested**, because leaving a type out of this list is a bug
// with no symptom on our side: the feature that needs it simply does nothing, the
// panel shows a healthy bot, and nothing is logged. Adding to this list means
// bumping handlers.TelegramWebhookVersion, which re-registers existing bots —
// otherwise the new type is asked for only by installs that happen to re-check.
var WebhookUpdates = []string{"message", "callback_query"}

// SetWebhook points the bot at us.
//
// `secret` is Telegram's own `secret_token`: it comes back on every delivery in
// the `X-Telegram-Bot-Api-Secret-Token` header, which is what lets the handler
// tell a real delivery from anybody who guessed the URL. Passed here rather than
// derived so the caller keeps both halves — the URL and the secret — in one place.
//
// `allowed` narrows what Telegram sends: only messages. Asking for everything
// means paying for edited-message and reaction deliveries this bot ignores.
func SetWebhook(ctx context.Context, token, url, secret string) error {
	var out struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	body := map[string]any{
		"url":          url,
		"secret_token": secret,
		// Callbacks too: the greeting's language buttons are the first thing a
		// guest touches, and an update type Telegram was never asked for is an
		// update type it never delivers — the buttons would simply do nothing.
		"allowed_updates": WebhookUpdates,
		// ⚠️ Deliberately true. A restaurant that re-saves its token gets a fresh
		// registration, and a backlog of updates from before that point is a
		// backlog of guests who have long since given up waiting for a reply.
		"drop_pending_updates": true,
	}
	if err := call(ctx, token, "setWebhook", body, &out); err != nil {
		return err
	}
	if !out.OK {
		return fmt.Errorf("telegram: %s", out.Description)
	}
	return nil
}

// Update is the slice of Telegram's update object this bot reads.
//
// Only messages, and only what a reply needs: who wrote, in which chat, what
// they said. Everything else Telegram sends is ignored rather than parsed — a
// bot that tries to understand every update type is a bot with a parser to keep
// in step with Telegram's release notes.
type Update struct {
	// A button press under a message the bot sent. Carried separately by
	// Telegram from a typed message, and it has to be answered twice: once to
	// Telegram (so the button stops spinning) and once to the guest (with what
	// they asked for).
	CallbackQuery *struct {
		ID   string `json:"id"`
		Data string `json:"data"`
		From *struct {
			ID           int64  `json:"id"`
			LanguageCode string `json:"language_code"`
		} `json:"from"`
		Message *struct {
			// Needed to take the buttons away once they have been used — see
			// ClearButtons.
			MessageID int64 `json:"message_id"`
			Chat      *struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message"`
	} `json:"callback_query"`
	Message *struct {
		Chat *struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		From *struct {
			ID           int64  `json:"id"`
			FirstName    string `json:"first_name"`
			LanguageCode string `json:"language_code"`
		} `json:"from"`
		Text string `json:"text"`
		// ⚠️ The one thing a Start press cannot tell us: who this is.
		//
		// Telegram gives an id and a name, never a number, so a guest who only ever
		// pressed Start is a chat we cannot match to an account — and therefore
		// cannot include in a campaign, however loyal they are. This arrives when
		// they tap the "share my number" button; `user_id` is Telegram's own
		// assertion that the number belongs to this account.
		Contact *struct {
			PhoneNumber string `json:"phone_number"`
			UserID      int64  `json:"user_id"`
			FirstName   string `json:"first_name"`
		} `json:"contact"`
	} `json:"message"`
}

// WebAppButton is a button that opens the mini app from inside a chat.
type WebAppButton struct {
	Label string
	URL   string
}

// SendMenu replies with a message and one button that opens the mini app.
//
// ⚠️ Two button types are attempted, and this is not indecision.
//
// A `web_app` button opens the mini app inside Telegram with no setup by the
// restaurant, which is what we want: the owner has pasted a token and expects the
// bot to work. But whether Telegram accepts one depends on the bot having a Mini
// App configured in @BotFather, and that is a step the owner may not have done —
// in which case the **whole message** is rejected and the bot is silent again,
// for a reason the owner cannot see.
//
// So a plain `url` button to the same page is the fallback: it opens the site in
// Telegram's browser rather than as a mini app, which is a smaller thing than a
// bot that does not answer. Which one was used is returned so it can be logged —
// the same approach as the ATMOS signature, where the documentation did not say
// and guessing wrong looked like an outage.
func SendMenu(ctx context.Context, token string, chatID int64, text string,
	btn WebAppButton) (string, error) {
	kinds := []struct {
		name string
		key  string
	}{{"web_app", "web_app"}, {"url", "url"}}
	var lastErr error
	for _, k := range kinds {
		var value any = btn.URL
		if k.key == "web_app" {
			value = map[string]any{"url": btn.URL}
		}
		body := map[string]any{
			"chat_id": chatID,
			"text":    text,
			"reply_markup": map[string]any{
				"inline_keyboard": [][]map[string]any{{
					{"text": btn.Label, k.key: value},
				}},
			},
		}
		var out struct {
			OK          bool   `json:"ok"`
			Description string `json:"description"`
		}
		if err := call(ctx, token, "sendMessage", body, &out); err != nil {
			return "", err // a transport failure is not a button problem
		}
		if out.OK {
			return k.name, nil
		}
		lastErr = fmt.Errorf("telegram: %s", out.Description)
	}
	return "", lastErr
}

// LangButton is one option in the greeting's language row.
type LangButton struct {
	Label string
	Data  string
}

// SendLangChoice greets the guest and asks which language they read.
//
// ⚠️ The greeting itself is in Uzbek, deliberately, and the options name
// themselves — the same rule the mini app's first screen follows. A "choose your
// language" sentence written in one language has already made the choice for the
// guest who cannot read it, so the sentence is short, the base language is the
// default, and the answer is one tap away in the language it is written in.
//
// Callback buttons rather than a `web_app` button here: this message must work
// before anything about the restaurant's @BotFather setup is known, and a
// callback button is the one kind Telegram always accepts.
func SendLangChoice(ctx context.Context, token string, chatID int64,
	text string, buttons []LangButton) error {
	row := make([]map[string]any, 0, len(buttons))
	for _, b := range buttons {
		row = append(row, map[string]any{"text": b.Label, "callback_data": b.Data})
	}
	body := map[string]any{
		"chat_id": chatID,
		"text":    text,
		// One button per row: three languages side by side truncate to "O'z…",
		// "Рус…", "Eng…" on a narrow phone, which is the one screen where the
		// label is the whole message.
		"reply_markup": map[string]any{"inline_keyboard": rows(row)},
	}
	var out struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := call(ctx, token, "sendMessage", body, &out); err != nil {
		return err
	}
	if !out.OK {
		return fmt.Errorf("telegram: %s", out.Description)
	}
	return nil
}

func rows(buttons []map[string]any) [][]map[string]any {
	out := make([][]map[string]any, 0, len(buttons))
	for _, b := range buttons {
		out = append(out, []map[string]any{b})
	}
	return out
}

// AnswerCallback tells Telegram the button press was handled.
//
// ⚠️ Not optional and not cosmetic: an unanswered callback leaves a loading
// spinner on the button for as long as Telegram waits, which reads as a frozen
// bot even when the reply arrives right behind it. Failures are ignored by the
// caller — the guest's actual answer matters more than the spinner.
func AnswerCallback(ctx context.Context, token, id, text string) error {
	var out struct {
		OK bool `json:"ok"`
	}
	return call(ctx, token, "answerCallbackQuery", map[string]any{
		"callback_query_id": id,
		"text":              text,
	}, &out)
}

// SendCampaign is one marketing message, with an optional photograph and the two
// buttons every campaign carries.
//
// ⚠️ **The buttons are the point of sending it through Telegram at all.** An SMS
// ends in the guest's inbox and whatever they do next starts from scratch; a bot
// message can end in the menu, one tap away, and can ask what they thought without
// them typing an address. So: "Menyu" opens the mini app, and "Fikr bildirish" is a
// callback the bot answers — see handlers/telegrambot.go.
//
// Photo and text are one call, not two (`sendPhoto` with a caption): two calls
// arrive as two notifications, and the second one is the advert without the picture
// it was written around. ⚠️ A caption is capped at 1024 characters by Telegram, and
// a message that exceeds it is **rejected whole** — so a long text falls back to a
// plain message rather than being silently truncated.
func SendCampaign(ctx context.Context, token string, chatID int64,
	text, photoURL string, buttons []MessageButton) error {
	rows := make([][]map[string]any, 0, len(buttons))
	for _, b := range buttons {
		btn := map[string]any{"text": b.Label}
		switch {
		case b.Callback != "":
			btn["callback_data"] = b.Callback
		case b.WebApp != "":
			btn["web_app"] = map[string]any{"url": b.WebApp}
		default:
			btn["url"] = b.URL
		}
		rows = append(rows, []map[string]any{btn})
	}
	markup := map[string]any{"inline_keyboard": rows}

	method := "sendMessage"
	body := map[string]any{"chat_id": chatID, "text": text, "reply_markup": markup}
	if photoURL != "" && len([]rune(text)) <= 1024 {
		method = "sendPhoto"
		body = map[string]any{
			"chat_id": chatID, "photo": photoURL, "caption": text, "reply_markup": markup,
		}
	}

	var out struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := call(ctx, token, method, body, &out); err != nil {
		return err
	}
	if !out.OK {
		// ⚠️ A web_app button is refused when the bot has no Mini App configured,
		// and it takes the whole message with it. Retried once without it: a
		// campaign that reached nobody because of a button is worse than one whose
		// button opens a browser.
		if method == "sendPhoto" || hasWebApp(buttons) {
			return retryPlain(ctx, token, chatID, text, photoURL, buttons, out.Description)
		}
		return fmt.Errorf("telegram: %s", out.Description)
	}
	return nil
}

// MessageButton is one inline button. Exactly one of Callback, WebApp or URL.
type MessageButton struct {
	Label    string
	Callback string
	WebApp   string
	URL      string
}

func hasWebApp(buttons []MessageButton) bool {
	for _, b := range buttons {
		if b.WebApp != "" {
			return true
		}
	}
	return false
}

// retryPlain sends the same message with the photograph dropped and any web_app
// button turned into an ordinary link.
func retryPlain(ctx context.Context, token string, chatID int64,
	text, photoURL string, buttons []MessageButton, firstErr string) error {
	plain := make([]MessageButton, 0, len(buttons))
	for _, b := range buttons {
		if b.WebApp != "" {
			b.URL, b.WebApp = b.WebApp, ""
		}
		plain = append(plain, b)
	}
	rows := make([][]map[string]any, 0, len(plain))
	for _, b := range plain {
		btn := map[string]any{"text": b.Label}
		if b.Callback != "" {
			btn["callback_data"] = b.Callback
		} else {
			btn["url"] = b.URL
		}
		rows = append(rows, []map[string]any{btn})
	}
	var out struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	body := map[string]any{
		"chat_id": chatID, "text": text,
		"reply_markup": map[string]any{"inline_keyboard": rows},
	}
	if err := call(ctx, token, "sendMessage", body, &out); err != nil {
		return err
	}
	if !out.OK {
		// Both failures reported: the first one is the interesting one, and the
		// second explains why the fallback did not save it.
		return fmt.Errorf("telegram: %s (fallback: %s)", firstErr, out.Description)
	}
	_ = photoURL
	return nil
}

// AskPhone sends a message with a one-tap button that shares the guest's number.
//
// ⚠️ A **reply keyboard**, not an inline one: `request_contact` exists only there.
// It is one tap and Telegram fills the number in — no typing, no SMS, and no code
// to wait for. `one_time_keyboard` so the keyboard disappears after the tap rather
// than sitting under every later message.
func AskPhone(ctx context.Context, token string, chatID int64, text, label string) error {
	var out struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	body := map[string]any{
		"chat_id": chatID,
		"text":    text,
		"reply_markup": map[string]any{
			"keyboard": [][]map[string]any{{
				{"text": label, "request_contact": true},
			}},
			"resize_keyboard":   true,
			"one_time_keyboard": true,
		},
	}
	if err := call(ctx, token, "sendMessage", body, &out); err != nil {
		return err
	}
	if !out.OK {
		return fmt.Errorf("telegram: %s", out.Description)
	}
	return nil
}

// SendClosing is a message that also takes the reply keyboard away.
//
// ⚠️ `one_time_keyboard` **hides** the keyboard on some clients and leaves it on
// others, so the "share my number" button stayed on screen after the number had been
// shared — an offer to do again what was already done. Only `remove_keyboard` actually
// removes it, and it has to ride on the next message.
func SendClosing(ctx context.Context, token string, chatID int64, text string) error {
	var out struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	body := map[string]any{
		"chat_id":      chatID,
		"text":         text,
		"reply_markup": map[string]any{"remove_keyboard": true},
	}
	if err := call(ctx, token, "sendMessage", body, &out); err != nil {
		return err
	}
	if !out.OK {
		return fmt.Errorf("telegram: %s", out.Description)
	}
	return nil
}

// ClearButtons removes the inline keyboard from a message that has been answered.
//
// ⚠️ **A used button has to stop being a button.** Telegram leaves an inline keyboard on
// screen for ever, so a row of five stars invites a guest to tap all five — and then tap
// them again tomorrow. The server refuses the repeats either way, but a control that
// still looks live and does nothing reads as broken, and one that still looks live and
// *works* is a spam form.
//
// Failures are ignored by callers: the rating is already stored, and a keyboard that
// could not be edited is a cosmetic problem next to that.
func ClearButtons(ctx context.Context, token string, chatID, messageID int64) error {
	var out struct {
		OK bool `json:"ok"`
	}
	return call(ctx, token, "editMessageReplyMarkup", map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		// An empty keyboard rather than omitting the field: omitting it leaves the
		// existing one in place, which is the bug this function exists to fix.
		"reply_markup": map[string]any{"inline_keyboard": [][]map[string]any{}},
	}, &out)
}
