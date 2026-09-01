package handlers

import (
	"context"
	"errors"
	"log"
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
		// ⚠️ Group ids are **not** secrets and go back as they are — unlike the
		// token above. A chat id is useless without the bot being a member,
		// and hiding it would mean an owner could never check what they typed.
		"alertChatId":    s.AlertChatID,
		"feedbackChatId": s.FeedbackChatID,
		"notifyLang":     s.NotifyLang,
		// The list the panel draws, so the screen carries no copy of a set the
		// server validates against.
		"notifyLangs": NotifyLangs,
		// ⚠️ **The most useful line on this page**, and the reason it is separate
		// from the check above: the check proves *we* can reach Telegram, and
		// cannot show whether Telegram can reach **us**. A token can be perfect
		// while the bot stays silent, and this is the only field that tells the
		// two apart — "nothing has ever arrived" and "it arrives and the reply
		// fails" have completely different next steps. Same role `lastEventAt`
		// plays for onlinePBX.
		"webhookAt":    s.WebhookAt,
		"lastUpdateAt": s.LastUpdateAt,
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
	// Where this restaurant's own notifications go. ⚠️ Zero is a decision
	// ("stop sending"), not an omission — see the note in the update.
	AlertChatID    int64  `json:"alertChatId"`
	FeedbackChatID int64  `json:"feedbackChatId"`
	NotifyLang     string `json:"notifyLang"`
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
		// ⚠️ **Written as sent, including zero.** These are the opposite of the
		// token above: an owner who clears the box means "stop sending there",
		// and keeping the old value would leave a group being posted to that
		// nobody can switch off from this page. The token is kept on empty
		// because it cannot be shown; these can, so an empty box is a decision.
		"alertChatId":    req.AlertChatID,
		"feedbackChatId": req.FeedbackChatID,
		// ⚠️ Validated against the list rather than stored as sent: an
		// unrecognised value falls back to Uzbek at send time anyway, so
		// storing one would leave the dropdown showing a language the messages
		// are not written in.
		"notifyLang": cleanNotifyLang(req.NotifyLang),
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
	// Held in a variable as well as in the update: the same sentence is stored
	// as written and answered translated, and reading it back out of a bson.M
	// would hand the writer an `any`.
	lastCheck := ""
	if err != nil {
		lastCheck = err.Error()
	} else {
		// The name, not just "ok": "@osh_markazi_bot · Osh Markazi" is the
		// difference between configured and configured correctly.
		lastCheck = "@" + me.Username + " · " + me.Name
		set["botUsername"] = me.Username

		// ⚠️ **The bot is also pointed back at us here**, and it has to happen on
		// this button rather than in a separate step. Until it does the bot can
		// only talk: order updates go out and a guest who presses Start gets
		// silence — which is not read as "one feature is missing", it is read as
		// "this restaurant's bot is broken". Nothing in the panel would show it,
		// because from our side nothing failed.
		//
		// Registered even when it was registered before: the address contains the
		// site's domain, so an owner who connected their own domain after pasting
		// the token has a webhook pointing at the old one. Re-registering is free;
		// noticing that it is stale is not.
		if webhookErr := h.registerTelegramWebhook(r.Context(), s.BotToken, set); webhookErr != nil {
			// Not a failed check: the token is proven, and login and order
			// messages already work. Only the incoming half is missing, and
			// saying so precisely is the difference between one five-minute fix
			// and an owner re-pasting a token that was never the problem.
			lastCheck = "@" + me.Username + " · " + me.Name +
				" — lekin bot javob bera olmaydi: " + webhookErr.Error()
		}
	}
	set["lastCheck"] = lastCheck
	_, _ = h.Store.TelegramSettings.UpdateOne(r.Context(), bson.M{},
		bson.M{"$set": set}, options.Update().SetUpsert(true))

	// Never an HTTP error: a bad connection is an answer this page exists to
	// give, not an exception.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok":      err == nil,
		"message": httpx.T(w, lastCheck),
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

// registerTelegramWebhook points the bot's incoming updates at this site.
//
// The secret in the address is generated once and reused: rotating it on every
// check would leave Telegram holding the previous one for as long as the
// registration takes to land, and a rotation nobody asked for is a bot that
// stops answering for no visible reason. It is rotatable on purpose (a new token
// is generated when there is none), which is the recovery path if an address ever
// leaks.
//
// ⚠️ The URL is built from `PUBLIC_BASE_URL` — the site's own primary domain,
// which the control plane rewrites when an owner connects their own. It is
// deliberately not taken from the incoming request: the panel may be open on an
// IP, a tunnel or a preview host, and Telegram would then be told to deliver
// updates somewhere that stops existing tomorrow.
// TelegramWebhookVersion is the shape of the registration this code needs.
//
// ⚠️ **Bump this whenever `telegram.SetWebhook`'s arguments change** — a new
// update type, a different path, anything. The number is compared against what
// was stored when the bot was last registered, and a mismatch re-registers. See
// models.TelegramSettings.WebhookVersion for the failure that produced it.
//
//	1 — messages only
//	2 — messages + callback_query (the greeting's language buttons)
const TelegramWebhookVersion = 2

func (h *Handler) registerTelegramWebhook(ctx context.Context, botToken string,
	set bson.M) error {
	s := h.telegramSettings(ctx)
	secret := strings.TrimSpace(s.WebhookToken)
	if secret == "" {
		secret = randomToken()
		set["webhookToken"] = secret
	}
	base := strings.TrimRight(h.Cfg.PublicBaseURL, "/")
	if !strings.HasPrefix(base, "https://") {
		// Telegram refuses plain HTTP, and it says so in a sentence an owner
		// cannot act on. Answered here instead: on a local install there is
		// nothing to fix, the bot simply cannot be reached from the internet.
		return errors.New("webhook uchun HTTPS manzil kerak (hozir: " + base + ")")
	}
	url := base + "/api/v1/telegram/" + secret
	if err := telegram.SetWebhook(ctx, botToken, url, secret); err != nil {
		return err
	}
	set["webhookAt"] = time.Now()
	set["webhookVersion"] = TelegramWebhookVersion
	return nil
}

// ensureWebhookCurrent re-registers a bot whose registration predates this code.
//
// ⚠️ Called from the webhook handler itself, on an update we are already
// handling — which is the one moment we know Telegram *can* reach us. That is
// deliberate: the registration this repairs is exactly the one that stops some
// update types arriving, so waiting for the missing type would wait for ever.
// Messages still arrive under every version, so a guest pressing Start is what
// fixes the buttons.
//
// Silent and best-effort. The guest is mid-conversation, and a failed
// re-registration means the bot keeps working exactly as it did a second ago.
func (h *Handler) ensureWebhookCurrent(ctx context.Context, s *models.TelegramSettings) {
	if s == nil || !s.Usable() || s.WebhookVersion >= TelegramWebhookVersion {
		return
	}
	set := bson.M{}
	if err := h.registerTelegramWebhook(ctx, s.BotToken, set); err != nil {
		log.Printf("telegram: could not refresh the webhook registration: %v", err)
		return
	}
	if _, err := h.Store.TelegramSettings.UpdateOne(ctx, bson.M{},
		bson.M{"$set": set}, options.Update().SetUpsert(true)); err != nil {
		log.Printf("telegram: webhook refreshed but not recorded: %v", err)
		return
	}
	log.Printf("telegram: webhook re-registered for update shape v%d",
		TelegramWebhookVersion)
}

// AdminTestNotifyChat sends one message to a group and reports what happened.
//
// ⚠️ **The most useful button on this page, and it was missing.** A chat id can
// be typed perfectly and the message still not arrive — the bot is not in the
// group, it was added but not made an administrator, the id belongs to a
// different chat, the group was upgraded to a supergroup and its id changed.
// Every one of those looks identical from the panel: a saved setting and
// silence. The same argument as the printer test button, and the same person is
// pressing it — somebody who can look at the result immediately.
//
// ⚠️ **Telegram's own sentence is passed through.** "bot is not a member of the
// chat" and "chat not found" have completely different fixes, and replacing
// both with "yuborilmadi" is how an evening gets spent guessing.
func (h *Handler) AdminTestNotifyChat(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req struct {
		// "alerts" or "feedback": which of the two ids to try.
		Which string `json:"which"`
		// ⚠️ Taken from the request rather than from the database, so the
		// button works **before** saving. An owner pasting an id wants to know
		// it is right, not to save a wrong one and then find out.
		ChatID int64 `json:"chatId"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	s := h.telegramSettings(r.Context())
	if s == nil || s.BotToken == "" {
		httpx.Error(w, http.StatusBadRequest,
			"Avval bot tokenini kiriting va saqlang.")
		return
	}
	if !s.Enabled {
		// ⚠️ Named separately, because a disabled bot is the one cause the
		// owner can see on this very screen and would never suspect: everything
		// is filled in and the master switch is off.
		httpx.Error(w, http.StatusBadRequest,
			"Telegram bot o'chirilgan — yuqoridagi tugmani yoqing.")
		return
	}
	chat := req.ChatID
	if chat == 0 {
		if req.Which == "feedback" {
			chat = s.FeedbackChatID
		} else {
			chat = s.AlertChatID
		}
	}
	if chat == 0 {
		httpx.Error(w, http.StatusBadRequest, "Guruh yoki kanal ID si kiritilmagan.")
		return
	}

	w2 := notifyWordsFor(s.NotifyLang)
	text := testNotifyText(req.Which, w2, h.restaurantName(r.Context()))
	if err := h.sendNotify(r.Context(), s.BotToken, chat, chatField(req.Which), text); err != nil {
		// ⚠️ 200 with the reason rather than a 5xx: the request worked, the
		// send did not, and the panel needs to print Telegram's words rather
		// than a status code.
		httpx.JSON(w, http.StatusOK, map[string]any{
			"ok": false,
			// The commonest cause, said before the raw error — Telegram's
			// wording is accurate and means nothing to a restaurant owner.
			"hint":  httpx.T(w, "Botni shu guruhga qo'shdingizmi va admin qildingizmi?"),
			"error": httpx.T(w, err.Error()),
		})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// testNotifyText is what lands in the group.
//
// ⚠️ **It says what this group is for**, because the people who read it later
// are not the person who pressed the button. A group whose first message is
// "test" tells the accountant added next week nothing at all.
func testNotifyText(which string, w notifyWords, restaurant string) string {
	head := "⚠️ " + w.Unknown
	body := w.NotAnAccusation
	if which == "feedback" {
		head = "★ " + w.FeedbackFrom
		body = ""
	}
	out := head
	if restaurant != "" {
		out += " · " + restaurant
	}
	if body != "" {
		out += "\n\n" + body
	}
	return out
}

// sendNotify posts to one of the two groups, following a supergroup upgrade.
//
// ⚠️ **A group becoming a supergroup is not a mistake anybody made.** Adding a
// bot, or giving somebody administrator rights, upgrades a small group on its
// own — so a restaurant that set this up correctly on Monday finds it silent on
// Tuesday, having changed nothing. The old id is dead permanently and the new
// one appears in exactly one place: the body of the response that just refused
// the message.
//
// So it is taken and stored, and the message is sent again. Making the owner
// re-copy an id they never saw change would be asking them to fix our
// bookkeeping — and they would have to notice it was broken first, which is the
// hard part.
func (h *Handler) sendNotify(
	ctx context.Context, token string, chatID int64, field, text string,
) error {
	err := telegram.SendMessage(ctx, token, chatID, text)
	var moved *telegram.MigratedError
	if !errors.As(err, &moved) {
		return err
	}
	// ⚠️ Written before the retry: if the second attempt also fails, the
	// restaurant is still better off with the id that at least exists. The
	// alternative is storing it only on success and re-learning it every time.
	if _, uerr := h.Store.TelegramSettings.UpdateOne(ctx, bson.M{},
		bson.M{"$set": bson.M{field: moved.NewChatID}},
		options.Update().SetUpsert(true)); uerr != nil {
		return err
	}
	log.Printf("telegram: %s became a supergroup, id %d -> %d",
		field, chatID, moved.NewChatID)
	return telegram.SendMessage(ctx, token, moved.NewChatID, text)
}

// chatField is which stored id a send belongs to, so a supergroup upgrade
// updates the right one.
func chatField(which string) string {
	if which == "feedback" {
		return "feedbackChatId"
	}
	return "alertChatId"
}
