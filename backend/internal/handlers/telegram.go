package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/auth"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/telegram"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The restaurant's Telegram bot: its settings, and signing in through it.
//
// Two things happen in this file, and the second is the interesting one.
//
// **Settings** follow the same shape as the SMS gateway and the payment
// providers: owner only, the token never returned to a browser, an empty field
// meaning "keep the stored one", and a check button that proves the credential
// rather than the form.
//
// **Signing in** replaces the SMS code entirely inside the mini app. Telegram
// already knows who the person is and says so in a payload signed with the
// restaurant's own bot token — so a guest opening the mini app is logged in
// before they have touched anything. That removes the two things that lose
// orders: waiting for a code, and paying for the message that carries it.
//
// ⚠️ **What Telegram does not give us is a phone number**, and an order needs
// one. So a Telegram login can succeed and still leave the guest unable to
// receive food; `needsPhone` says so, and the mini app asks once.

func (h *Handler) telegramSettings(ctx context.Context) *models.TelegramSettings {
	var s models.TelegramSettings
	if err := h.Store.TelegramSettings.FindOne(ctx, bson.M{}).Decode(&s); err != nil {
		return &models.TelegramSettings{}
	}
	return &s
}

// ---- Settings (owner only) ----

// AdminGetTelegram returns the bot settings without the token.
func (h *Handler) AdminGetTelegram(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	s := h.telegramSettings(r.Context())
	httpx.JSON(w, http.StatusOK, map[string]any{
		"enabled":     s.Enabled,
		"botUsername": s.BotUsername,
		// The panel learns that a token exists, never what it is.
		"hasToken":    s.BotToken != "",
		"lastCheckAt": s.LastCheckAt,
		"lastCheckOk": s.LastCheckOk,
		"lastCheck":   s.LastCheck,
		// Built here rather than in the browser: the deep link is the one thing
		// the operator has to hand to the restaurant, and a page that made them
		// assemble it from a username is a page that produces typos.
		"miniAppUrl": miniAppURL(s.BotUsername),
	})
}

type telegramSettingsRequest struct {
	Enabled bool `json:"enabled"`
	// Empty means "keep the stored token".
	BotToken string `json:"botToken"`
}

// AdminUpdateTelegram saves the bot settings.
func (h *Handler) AdminUpdateTelegram(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req telegramSettingsRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	current := h.telegramSettings(r.Context())

	// ⚠️ An empty token means "keep what is stored", never "erase it".
	//
	// The form cannot show a token it never received, so an owner who only meant
	// to flip the switch sends an empty field — and treating that as a deletion
	// would silently disconnect the bot. Exactly the trap that shipped once with
	// the payment keys and once with `kioskSecret`.
	set := bson.M{
		"enabled":   req.Enabled,
		"botToken":  keepSecret(req.BotToken, current.BotToken),
		"updatedAt": time.Now(),
	}
	// A new token belongs to a different bot, so the username the check button
	// filled in is no longer true. Cleared rather than left behind: a deep link
	// pointing at the previous bot is worse than no deep link.
	if strings.TrimSpace(req.BotToken) != "" &&
		strings.TrimSpace(req.BotToken) != current.BotToken {
		set["botUsername"] = ""
		set["lastCheckOk"] = false
		set["lastCheck"] = ""
	}
	if _, err := h.Store.TelegramSettings.UpdateOne(r.Context(), bson.M{},
		bson.M{"$set": set}, options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActTelegramSettings, "settings", "telegram", "Telegram bot",
		boolWord(req.Enabled))
	h.AdminGetTelegram(w, r)
}

// AdminPingTelegram proves the token and learns the bot's username.
//
// This is the page's main control, for the same reason the SMS test button is:
// a token that looks right and belongs to a deleted bot is indistinguishable
// from a working one until the first guest tries to sign in — and then the
// restaurant reads it as "your site is broken".
func (h *Handler) AdminPingTelegram(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	s := h.telegramSettings(r.Context())
	me, err := telegram.GetMe(r.Context(), s.BotToken)

	set := bson.M{"lastCheckAt": time.Now(), "lastCheckOk": err == nil}
	if err != nil {
		set["lastCheck"] = err.Error()
	} else {
		// The name, not just "ok": "@osh_markazi_bot · Osh Markazi" is the
		// difference between configured and configured correctly.
		set["lastCheck"] = "@" + me.Username + " · " + me.Name
		set["botUsername"] = me.Username
	}
	_, _ = h.Store.TelegramSettings.UpdateOne(r.Context(), bson.M{},
		bson.M{"$set": set}, options.Update().SetUpsert(true))

	// Never an HTTP error: a bad connection is an answer this page exists to
	// give, not an exception.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok":      err == nil,
		"message": set["lastCheck"],
	})
}

// ---- Signing in ----

type telegramAuthRequest struct {
	// The raw `Telegram.WebApp.initData` string, exactly as the client received
	// it. Passed through untouched: it is a signed string, and re-encoding it
	// changes the bytes the signature covers.
	InitData string `json:"initData"`
}

// TelegramLogin turns a signed Telegram payload into one of our sessions.
//
// Public, because it is a login. Everything that makes it safe is inside
// `telegram.Verify`: the payload is signed with this restaurant's own bot token,
// so a string minted by another bot — or edited in the browser — cannot name a
// different person.
func (h *Handler) TelegramLogin(w http.ResponseWriter, r *http.Request) {
	var req telegramAuthRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	s := h.telegramSettings(r.Context())
	if !s.Usable() {
		// The honest failure: with no bot connected there is no Telegram login.
		// Said plainly so the guest is not left waiting for something that will
		// never happen — the same rule the SMS path follows.
		httpx.Error(w, http.StatusServiceUnavailable,
			"Telegram orqali kirish hali sozlanmagan")
		return
	}

	fields, err := telegram.Verify(req.InitData, s.BotToken)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	if err := telegram.CheckFresh(fields, time.Now(), telegram.MaxAuthAge); err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	tgUser, err := telegram.ParseUser(fields)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	now := time.Now()
	// Keyed on the Telegram id, which is the only thing Telegram guarantees is
	// stable — a username can be changed or given away, and matching on one would
	// hand a stranger somebody else's order history.
	setOnInsert := bson.M{
		"createdAt":    now,
		"authProvider": "telegram",
		"addresses":    []models.UserAddress{},
	}
	if tgUser.FirstName != "" {
		setOnInsert["firstName"] = tgUser.FirstName
	}
	if tgUser.LastName != "" {
		setOnInsert["lastName"] = tgUser.LastName
	}
	_, err = h.Store.Users.UpdateOne(r.Context(),
		bson.M{"telegramId": tgUser.ID},
		bson.M{
			"$set": bson.M{
				"telegramId":       tgUser.ID,
				"telegramUsername": tgUser.Username,
				"telegramLang":     tgUser.Lang,
				"updatedAt":        now,
			},
			"$setOnInsert": setOnInsert,
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	var user models.User
	if err := h.Store.Users.FindOne(r.Context(),
		bson.M{"telegramId": tgUser.ID}).Decode(&user); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.grantWelcomePoints(r.Context(), &user)
	_ = h.Store.Users.FindOne(r.Context(), bson.M{"_id": user.ID}).Decode(&user)

	token, err := auth.Generate(h.Cfg.JWTSecret, user.ID.Hex(), "user")
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  user,
		// ⚠️ The field the checkout turns on. A Telegram login is complete and
		// still cannot receive an order until a phone number exists, and a mini
		// app that discovered this at the "confirm order" step would lose the
		// order it had already won.
		"needsPhone": strings.TrimSpace(user.Phone) == "",
		// ⚠️ Asked **before** anything else, and only once per account.
		//
		// A guest arriving from a QR code has no address bar to carry `/ru/` and
		// no cookie yet, so without this screen the mini app opens in Uzbek for
		// everybody and the switch is something they have to go looking for while
		// reading a menu they cannot read. Telegram's own UI language is not a
		// substitute — it is a guess, and it is wrong for exactly the guests who
		// would notice.
		//
		// Once stored it also decides what the bot writes later (see notifyLang),
		// which is the half a cookie could never cover.
		"needsLang": func() bool { _, ok := langAllowed(user.Lang); return !ok }(),
	})
}

type telegramPhoneRequest struct {
	// The signed response from `requestContact`, in the same form as initData.
	Contact string `json:"contact"`
}

// TelegramPhone records a phone number Telegram vouched for.
//
// ⚠️ **This is stronger evidence than an SMS code, not weaker.** An SMS proves
// somebody held the phone for thirty seconds; this is Telegram stating the number
// on the account, signed with the restaurant's bot token. So it is accepted in
// place of the SMS flow rather than in addition to it — and no code is sent,
// which is also one fewer paid message.
//
// The number is only ever written to the account the caller is already signed in
// as. Without that, a valid contact payload would be a way to attach your phone
// to somebody else's history.
func (h *Handler) TelegramPhone(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "avtorizatsiya kerak")
		return
	}
	uid, err := objectID(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var user models.User
	if err := h.Store.Users.FindOne(r.Context(), bson.M{"_id": uid}).Decode(&user); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "foydalanuvchi topilmadi")
		return
	}
	var req telegramPhoneRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	s := h.telegramSettings(r.Context())
	if !s.Usable() {
		httpx.Error(w, http.StatusServiceUnavailable,
			"Telegram orqali kirish hali sozlanmagan")
		return
	}

	fields, err := telegram.Verify(req.Contact, s.BotToken)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	if err := telegram.CheckFresh(fields, time.Now(), telegram.MaxAuthAge); err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	contact, err := telegram.ParseContact(fields)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// The signed payload names a Telegram user; it has to be the one holding this
	// session. Otherwise a payload obtained anywhere could be replayed to write a
	// stranger's number onto this account — or this number onto theirs.
	if contact.UserID != 0 && user.TelegramID != 0 && contact.UserID != user.TelegramID {
		httpx.Error(w, http.StatusForbidden, "bu raqam boshqa hisobga tegishli")
		return
	}

	phone, ok := normalizePhone(contact.Phone)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "telefon raqami noto'g'ri")
		return
	}
	// Somebody else may already hold this number — the same person, having signed
	// in by SMS first. Refused rather than merged: joining two accounts silently
	// moves orders, points and addresses between them, and the right place to
	// decide that is not inside a login.
	taken, err := h.Store.Users.CountDocuments(r.Context(), bson.M{
		"phone": phone, "_id": bson.M{"$ne": user.ID},
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if taken > 0 {
		httpx.Error(w, http.StatusConflict,
			"bu raqam boshqa hisobda band — saytga shu raqam bilan kiring")
		return
	}

	if _, err := h.Store.Users.UpdateByID(r.Context(), user.ID, bson.M{
		"$set": bson.M{"phone": phone, "updatedAt": time.Now()},
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var out models.User
	_ = h.Store.Users.FindOne(r.Context(), bson.M{"_id": user.ID}).Decode(&out)
	httpx.JSON(w, http.StatusOK, map[string]any{"user": out})
}

// miniAppURL is where the restaurant sends its guests.
//
// Empty until the check button has run: a link built from a username nobody
// verified is a link that opens somebody else's bot.
func miniAppURL(username string) string {
	u := strings.TrimPrefix(strings.TrimSpace(username), "@")
	if u == "" {
		return ""
	}
	return "https://t.me/" + u + "/app"
}

func boolWord(v bool) string {
	if v {
		return "yoqildi"
	}
	return "o'chirildi"
}
