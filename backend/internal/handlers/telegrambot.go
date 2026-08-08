package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/telegram"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The bot answering back.
//
// ⚠️ **This was the missing half, and its absence looked exactly like a broken
// bot.** Everything built before this could only send: order updates went out,
// and nothing listened. A guest who found the bot and pressed Start — the first
// thing anybody does with a bot — got silence. Nothing in the panel showed a
// problem, because from our side there was none.
//
// What arrives here is a webhook delivery, and three rules shape the handler:
//
//   - **The URL is the credential.** Telegram sends no password of its own, so
//     the secret lives in the path (generated, rotatable) and is compared in
//     constant time. Telegram's own `secret_token` header is checked too when
//     one is registered — two independent halves, either of which is enough to
//     reject a stranger.
//   - **Always 200.** A webhook that answers with an error gets retried, and a
//     retry storm over a payload we do not understand helps nobody. A rejected
//     delivery is refused *silently* — a 401 tells a scanner it found something.
//   - **A reply is best effort, never a failure.** Telegram is waiting on this
//     response; the message is sent in the background so a slow send does not
//     turn into a redelivery of the same Start.

// TelegramWebhook receives what a guest wrote to the bot.
//
// Public by necessity — Telegram is the caller — and safe by the secret in the
// path rather than by a session.
func (h *Handler) TelegramWebhook(w http.ResponseWriter, r *http.Request) {
	ok := func() { httpx.JSON(w, http.StatusOK, map[string]any{"ok": true}) }

	s := h.telegramSettings(r.Context())
	if !s.Usable() || s.WebhookToken == "" {
		ok()
		return
	}
	token := chi.URLParam(r, "token")
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.WebhookToken)) != 1 {
		ok()
		return
	}
	// Telegram echoes the secret we registered. Checked only when the header is
	// present: a registration made before this field existed sends none, and
	// refusing those would silence a bot that is working.
	if got := r.Header.Get("X-Telegram-Bot-Api-Secret-Token"); got != "" {
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.WebhookToken)) != 1 {
			ok()
			return
		}
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		ok()
		return
	}
	var up telegram.Update
	if err := json.Unmarshal(body, &up); err != nil {
		ok()
		return
	}

	// ⚠️ Recorded before anything else, and even for an update we ignore. This is
	// the one fact the settings page cannot get any other way: the check button
	// proves *we* can reach Telegram, never that Telegram can reach us. An owner
	// staring at a silent bot needs to know whether the press arrived at all —
	// "nothing has ever arrived" and "it arrives and the reply fails" have
	// completely different next steps.
	_, _ = h.Store.TelegramSettings.UpdateOne(r.Context(), bson.M{},
		bson.M{"$set": bson.M{"lastUpdateAt": time.Now()}},
		options.Update().SetUpsert(true))

	msg := up.Message
	if msg == nil || msg.Chat == nil {
		ok()
		return
	}

	lang := "uz"
	if msg.From != nil {
		// ⚠️ A guess, and the only place in the app where guessing is right: the
		// guest has not chosen yet and cannot be asked — they are in a chat, not
		// in the app. The reply's own button opens the app, where the first
		// screen asks properly and stores the answer (see userlang.go).
		lang = notifyLangCode(msg.From.LanguageCode)
	}
	text, label := botWelcome(lang, h.restaurantName(r.Context()))
	url := h.miniAppURL(strings.TrimSpace(msg.Text))

	// Detached: Telegram is holding this request open, and a redelivery of the
	// same Start would send the guest two greetings.
	token2, chatID := s.BotToken, msg.Chat.ID
	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 20*time.Second)
		defer cancel()
		kind, err := telegram.SendMenu(ctx, token2, chatID, text,
			telegram.WebAppButton{Label: label, URL: url})
		if err != nil {
			// Logged, never surfaced: the commonest cause is a guest who blocked
			// the bot, which the restaurant can neither see nor fix.
			log.Printf("telegram reply to %d: %v", chatID, err)
			return
		}
		// Which button style Telegram accepted. Worth a line: if the fallback is
		// what works, the owner has no Mini App configured in @BotFather, and
		// that is a five-second fix nobody would think to look for.
		log.Printf("telegram reply to %d: sent with %s button", chatID, kind)
	}()

	ok()
}

// botWelcome is what a guest reads when they press Start.
//
// Deliberately short and in three languages, with the button label beside its
// own text: this message exists to get the guest into the app, where the menu,
// the language screen and the cart already live. A bot that tries to be a second
// interface to the restaurant is a second interface to keep in step with the
// first.
func botWelcome(lang, restaurant string) (text, button string) {
	switch lang {
	case "ru":
		return restaurant + ": здравствуйте! Меню, заказ и доставка — в приложении ниже.",
			"Открыть меню"
	case "en":
		return restaurant + ": welcome! The menu, ordering and delivery are in the app below.",
			"Open the menu"
	default:
		return restaurant + ": assalomu alaykum! Menyu, buyurtma va yetkazib berish — pastdagi ilovada.",
			"Menyuni ochish"
	}
}

// notifyLangCode maps Telegram's language code onto the three the app speaks.
//
// A thin wrapper over normalizeLang so the call site above reads as what it is —
// a guess about a phone — rather than as a stored preference.
func notifyLangCode(code string) string { return normalizeLang(code) }

// miniAppURL is where the button points.
//
// ⚠️ `/start <payload>` is the **chat** deep link, and it is a different
// parameter from the mini app's `startapp` — a table QR that goes through the bot
// arrives here as a Start payload, not as a query string. Translated into the
// site's own `?table=` so the table logic, the brand cookie and the dine-in
// order type keep working unchanged, exactly as the mini app does with
// `startapp`.
func (h *Handler) miniAppURL(command string) string {
	base := strings.TrimRight(h.Cfg.PublicBaseURL, "/") + "/menu"
	payload := ""
	if fields := strings.Fields(command); len(fields) > 1 {
		payload = fields[1]
	}
	if payload == "" {
		return base
	}
	table, branch := parseStartPayload(payload)
	if table == "" {
		return base
	}
	url := base + "?table=" + table
	if branch != "" {
		url += "&branch=" + branch
	}
	return url
}

// parseStartPayload reads `t_<id>-b_<id>`, the same encoding the mini app uses.
//
// ⚠️ Validated as a hex id rather than trusted: this value comes from a link
// anybody can type into a chat, and it ends up in a URL we hand back to a guest.
func parseStartPayload(raw string) (table, branch string) {
	for _, part := range strings.Split(raw, "-") {
		key, value, found := strings.Cut(part, "_")
		if !found || !isHexID(value) {
			continue
		}
		switch key {
		case "t":
			table = value
		case "b":
			branch = value
		}
	}
	return table, branch
}

func isHexID(v string) bool {
	if len(v) != 24 {
		return false
	}
	for _, c := range v {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

// restaurantName is the name a guest is greeted and notified with.
//
// ⚠️ **Brand first, company second**, and this is the third place that rule has
// had to be learned. Once a tenant has a brand — every one of them does — the
// settings page saves the name onto the **brand** document and stops sending it on
// the company payload, so `restaurant.name` keeps the seeded "My Restaurant" for
// ever. It shipped once as a `my-restaurant.zip` download for an owner whose site
// said "Osh Markazi" everywhere, and once as the showcase strip on keel.uz.
//
// It was live here too: `notifyOrderStatus` read `restaurant.name` on its own,
// so every branded tenant's order messages were signed **"Restoran"**. Nothing
// fails, nobody sees a stack trace — the guest simply gets an anonymous message
// about their own money from a restaurant that has a name.
//
// Takes a `ctx` rather than a request on purpose: the two callers are a webhook
// from Telegram and a background goroutine, and neither has a brand cookie to
// read. The primary brand is the right answer for both.
func (h *Handler) restaurantName(ctx context.Context) string {
	var rest models.Restaurant
	_ = h.Store.Restaurant.FindOne(ctx, bson.M{}).Decode(&rest)

	opts := options.FindOne().SetSort(bson.D{
		{Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: 1},
	})
	var brand models.Brand
	if err := h.Store.Brands.FindOne(ctx, bson.M{"isActive": true}, opts).
		Decode(&brand); err == nil {
		applyBrand(&rest, &brand)
	}
	name := strings.TrimSpace(rest.Name)
	if name == "" || name == seedRestaurantName {
		return "Restoran"
	}
	return name
}
