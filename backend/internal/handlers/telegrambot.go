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

	// A language button under the greeting. Handled first because it is the more
	// specific case and because Telegram delivers it as a different update type
	// entirely — a handler that only looks at `message` leaves the buttons doing
	// nothing, which is exactly how the bot looked before it could answer at all.
	if cb := up.CallbackQuery; cb != nil {
		raw, isLang := strings.CutPrefix(strings.TrimSpace(cb.Data), "lang:")
		// `lang:ru|<tableId>` — the table the guest scanned, carried through the
		// language step because this webhook call ends in a moment and the tap may
		// come minutes later.
		lang, table, _ := strings.Cut(raw, "|")
		if !isHexID(table) {
			table = ""
		}
		chatID := int64(0)
		if cb.Message != nil && cb.Message.Chat != nil {
			chatID = cb.Message.Chat.ID
		}
		if !isLang || chatID == 0 {
			ok()
			return
		}
		// The feedback button from a campaign. Handled before the language buttons
		// because it is the more specific prefix.
		if strings.HasPrefix(strings.TrimSpace(cb.Data), "fb:") {
			var who int64
			if cb.From != nil {
				who = cb.From.ID
			}
			langCode := ""
			if cb.From != nil {
				langCode = cb.From.LanguageCode
			}
			h.askForFeedback(r.Context(), s, cb.ID, chatID, who, langCode)
			ok()
			return
		}

		picked, valid := langAllowed(lang)
		if !valid {
			// The value came off a button we drew, so this is not a guest error —
			// it is a stale message from a previous version of the bot. Uzbek is
			// the honest answer, and it is what the greeting was written in.
			picked = "uz"
		}
		// Remembered on the account **if there is one**. A guest can press Start
		// having never opened the app, and inventing an account for them here
		// would create a customer with no phone number who never asked for one.
		// The choice is not lost either way: it rides in the link below, and the
		// app stores it the moment they are signed in.
		if cb.From != nil {
			_, _ = h.Store.Users.UpdateOne(r.Context(),
				bson.M{"telegramId": cb.From.ID},
				bson.M{"$set": bson.M{"lang": picked, "updatedAt": time.Now()}})
		}
		text, label := botMenuPrompt(picked, h.restaurantName(r.Context()))
		url := h.miniAppURL(picked, table)
		// ⚠️ Whether this chat is anybody we know.
		//
		// A Start press gives an id and a name, never a number — so a guest who has
		// only ever used the chat is invisible to every campaign, however often they
		// order. That is the "0 recipients, 1 has not opened the bot" the panel was
		// reporting: the account and the chat were never joined up. Asked once, right
		// after the language, where it costs one tap.
		needsPhone := true
		if cb.From != nil {
			needsPhone = h.Store.Users.FindOne(r.Context(),
				bson.M{"telegramId": cb.From.ID}).Err() != nil
		}
		botToken, cbID := s.BotToken, cb.ID
		go func() {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 20*time.Second)
			defer cancel()
			// First the spinner, then the answer: an unanswered callback leaves
			// the button loading for as long as Telegram waits, which reads as a
			// frozen bot even when the reply lands right behind it.
			_ = telegram.AnswerCallback(ctx, botToken, cbID, "")
			kind, err := telegram.SendMenu(ctx, botToken, chatID, text,
				telegram.WebAppButton{Label: label, URL: url})
			if err != nil {
				log.Printf("telegram menu to %d: %v", chatID, err)
				return
			}
			log.Printf("telegram menu to %d: %s button, lang=%s", chatID, kind, picked)
			if needsPhone {
				ask, btn := botPhonePrompt(picked)
				if err := telegram.AskPhone(ctx, botToken, chatID, ask, btn); err != nil {
					// Not worth surfacing: the guest can still order from the app,
					// where the number is asked for anyway.
					log.Printf("telegram phone ask to %d: %v", chatID, err)
				}
			}
		}()
		ok()
		return
	}

	msg := up.Message
	if msg == nil || msg.Chat == nil {
		ok()
		return
	}

	// The number, shared with one tap. Handled first because it is the message that
	// turns a chat into a customer we can actually reach.
	if msg.Contact != nil && msg.From != nil {
		h.linkPhone(r.Context(), s, msg.Chat.ID, msg.From.ID,
			msg.Contact.PhoneNumber, msg.Contact.UserID, msg.From.LanguageCode)
		ok()
		return
	}

	// An answer to the bot's own question, if it asked one. Checked before the
	// greeting: replying to a question and being greeted for it reads as the reply
	// having been ignored.
	if msg.From != nil && h.takeFeedback(r.Context(), s, msg.Chat.ID, msg.From.ID,
		msg.Text, msg.From.LanguageCode) {
		ok()
		return
	}

	// ⚠️ Repaired here, on a message, because a message is the one update type
	// every registration delivers. A bot registered by an older build was never
	// asked for `callback_query`, so its language buttons did nothing at all —
	// and no state of ours could show that, because the stale thing was held by
	// Telegram. Pressing Start fixes it.
	h.ensureWebhookCurrent(r.Context(), s)

	// ⚠️ **Uzbek, not Telegram's guess.** The greeting is the base language and
	// the choice is the guest's own, one tap below it — which is the same rule the
	// mini app's first screen follows. Reading the language off the phone would be
	// right often enough to look correct and wrong for exactly the guests who
	// notice: the Uzbek speaker on an English phone, the Russian speaker whose
	// Telegram was set up by somebody else.
	text, buttons := botGreeting(h.restaurantName(r.Context()))
	// ⚠️ The table from a `/start t_<id>` deep link has to survive the language
	// step, so it is carried on the greeting's buttons rather than held here: this
	// request ends in a moment, and the guest may tap a language minutes later.
	table, branch := parseStartPayload(startPayload(msg.Text))
	for i := range buttons {
		buttons[i].Data = appendPayload(buttons[i].Data, table, branch)
	}

	// Detached: Telegram is holding this request open, and a redelivery of the
	// same Start would send the guest two greetings.
	botToken, chatID := s.BotToken, msg.Chat.ID
	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 20*time.Second)
		defer cancel()
		if err := telegram.SendLangChoice(ctx, botToken, chatID, text, buttons); err != nil {
			// Logged, never surfaced: the commonest cause is a guest who blocked
			// the bot, which the restaurant can neither see nor fix.
			log.Printf("telegram greeting to %d: %v", chatID, err)
		}
	}()

	ok()
}

// botGreeting is the first message: hello, and which language do you read?
//
// Two steps rather than one, and the order matters. A greeting that already
// carried the menu button would be a greeting in a language the guest may not
// read, holding the only control that matters — so the menu is offered **after**
// the choice, in the language they just picked.
//
// ⚠️ The options are labelled in their own language and script. That is the only
// text guaranteed readable by the person who has to read it; a translated
// "Choose your language" heading has already picked a winner.
func botGreeting(restaurant string) (text string, buttons []telegram.LangButton) {
	text = restaurant + ": assalomu alaykum! 👋\n" +
		"Tilni tanlang / Выберите язык / Choose language"
	return text, []telegram.LangButton{
		{Label: "O'zbekcha", Data: "lang:uz"},
		{Label: "Русский", Data: "lang:ru"},
		{Label: "English", Data: "lang:en"},
	}
}

// botMenuPrompt is the second message: the way in, in the chosen language.
//
// Short on purpose. Everything a guest needs — the menu, the cart, the checkout,
// their addresses — is in the app already, and a bot that grows into a second
// interface to the restaurant is a second interface to keep in step with the
// first.
func botMenuPrompt(lang, restaurant string) (text, button string) {
	switch lang {
	case "ru":
		return restaurant + ": меню, заказ и доставка — в приложении ниже.",
			"Открыть меню"
	case "en":
		return restaurant + ": the menu, ordering and delivery are in the app below.",
			"Open the menu"
	default:
		return restaurant + ": menyu, buyurtma va yetkazib berish — pastdagi ilovada.",
			"Menyuni ochish"
	}
}

// startPayload is the argument of `/start <payload>`, or "".
func startPayload(text string) string {
	if fields := strings.Fields(strings.TrimSpace(text)); len(fields) > 1 {
		return fields[1]
	}
	return ""
}

// appendPayload carries the scanned table through the language step.
//
// ⚠️ Telegram allows 64 bytes of `callback_data`, which two 24-character ids plus
// labels would overflow — and an overflowing button is rejected with the whole
// message, leaving the bot silent again. So the table rides alone: the branch is
// re-derived from it on the site, exactly as it is for a QR opened in a browser.
func appendPayload(data, table, _ string) string {
	if table == "" {
		return data
	}
	return data + "|" + table
}

// miniAppURL is where the button points.
//
// ⚠️ `/start <payload>` is the **chat** deep link, and it is a different
// parameter from the mini app's `startapp` — a table QR that goes through the bot
// arrives here as a Start payload, not as a query string. Translated into the
// site's own `?table=` so the table logic, the brand cookie and the dine-in
// order type keep working unchanged, exactly as the mini app does with
// `startapp`.
func (h *Handler) miniAppURL(lang, table string) string {
	base := strings.TrimRight(h.Cfg.PublicBaseURL, "/")
	// ⚠️ The language goes in the **path**, because that is how the site carries
	// it: `/ru/menu`. A query parameter would be read by nothing.
	if lang == "ru" || lang == "en" {
		base += "/" + lang
	}
	// `lc=1` says the language was already chosen, in the chat. Without it the
	// app's own first screen would ask again one tap later — the same question,
	// which reads as the first answer having been ignored.
	url := base + "/menu?lc=1"
	if table != "" {
		url += "&table=" + table
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

// The feedback loop behind a campaign's second button.
//
// ⚠️ **One question, one flag.** The guest presses "tell us what you think", the bot
// asks, and the next thing they type is stored. That is deliberately not a
// conversation state machine: there is exactly one question the bot ever asks, and
// modelling it as anything larger would be inventing states nobody reaches. The flag
// is cleared as soon as an answer arrives, so a forgotten one cannot turn next
// week's "salom" into a review.
//
// Stored as ordinary feedback with **no order attached** — it is an opinion about the
// restaurant rather than about a delivery, and the panel already shows unanswered
// ones first. A complaint that arrived this way is as answerable as any other.
func (h *Handler) askForFeedback(ctx context.Context, s *models.TelegramSettings,
	callbackID string, chatID, telegramID int64, langCode string) {
	lang := normalizeLang(langCode)
	if telegramID != 0 {
		// Best effort: a guest we cannot identify still gets the prompt, and their
		// answer is simply not attributed. Refusing to ask would be worse.
		_, _ = h.Store.Users.UpdateOne(ctx, bson.M{"telegramId": telegramID},
			bson.M{"$set": bson.M{"awaitingFeedback": true, "updatedAt": time.Now()}})
	}
	prompt := map[string]string{
		"ru": "Напишите, что вы думаете — одним сообщением. Мы прочитаем.",
		"en": "Write what you think, in one message. We read every one.",
	}[lang]
	if prompt == "" {
		prompt = "Fikringizni bitta xabarda yozing — biz o'qiymiz."
	}
	token, id := s.BotToken, callbackID
	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
		defer cancel()
		_ = telegram.AnswerCallback(ctx, token, id, "")
		if err := telegram.SendMessage(ctx, token, chatID, prompt); err != nil {
			log.Printf("telegram feedback prompt to %d: %v", chatID, err)
		}
	}()
}

// takeFeedback stores a typed message as feedback when the bot asked for one.
//
// Returns true when the message was consumed, so the greeting is not sent as well:
// answering a question and being greeted for it reads as the answer being ignored.
func (h *Handler) takeFeedback(ctx context.Context, s *models.TelegramSettings,
	chatID, telegramID int64, text, langCode string) bool {
	text = strings.TrimSpace(text)
	if telegramID == 0 || text == "" || strings.HasPrefix(text, "/") {
		return false
	}
	var user models.User
	if err := h.Store.Users.FindOne(ctx,
		bson.M{"telegramId": telegramID, "awaitingFeedback": true}).Decode(&user); err != nil {
		return false
	}
	if len([]rune(text)) > 1000 {
		text = string([]rune(text)[:1000])
	}
	// Cleared first: a failed insert must not leave the guest in a state where
	// everything they type for the next week becomes a review.
	_, _ = h.Store.Users.UpdateByID(ctx, user.ID,
		bson.M{"$set": bson.M{"awaitingFeedback": false, "updatedAt": time.Now()}})

	fb := models.Feedback{
		UserID: user.ID,
		Customer: models.OrderCustomer{
			Name:  strings.TrimSpace(user.FirstName + " " + user.LastName),
			Phone: user.Phone,
		},
		// ⚠️ No rating. A star nobody chose is a made-up number, and the panel's
		// complaint rule reads the rating — so inventing a low one would file every
		// kind word as a complaint. Zero means "they wrote to us".
		Comment:   text,
		CreatedAt: time.Now(),
	}
	if _, err := h.Store.Feedback.InsertOne(ctx, fb); err != nil {
		log.Printf("telegram feedback from %d: %v", telegramID, err)
		return true // consumed either way: asking again would be worse
	}

	thanks := map[string]string{
		"ru": "Спасибо! Мы прочитали.",
		"en": "Thank you — we have read it.",
	}[normalizeLang(langCode)]
	if thanks == "" {
		thanks = "Rahmat! Fikringizni oldik."
	}
	token := s.BotToken
	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
		defer cancel()
		_ = telegram.SendMessage(ctx, token, chatID, thanks)
	}()
	return true
}

// botPhonePrompt asks for the number, and says what it is for.
//
// ⚠️ "So we can tell you about your order" is the honest reason and the one that
// gets a tap. A bare "send your number" from a restaurant's bot reads as a data
// grab, and the guests who refuse are the ones who would have ordered.
func botPhonePrompt(lang string) (text, button string) {
	switch lang {
	case "ru":
		return "Оставьте номер — чтобы мы могли сообщать о заказе и акциях. Один тап, вводить ничего не нужно.",
			"📱 Отправить номер"
	case "en":
		return "Share your number so we can tell you about your order and our offers. One tap — nothing to type.",
			"📱 Share my number"
	default:
		return "Raqamingizni qoldiring — buyurtma va aksiyalar haqida xabar berib turamiz. Bir teginish, hech nima yozish kerak emas.",
			"📱 Raqamni yuborish"
	}
}

// linkPhone joins a chat to an account using the number Telegram vouched for.
//
// ⚠️ **Telegram's assertion is stronger evidence than an SMS code**, and this is the
// same reasoning the mini app's phone step follows: an SMS proves somebody held a
// handset for thirty seconds, this is Telegram stating the number on the account.
// So it is accepted without a code — and one paid message is avoided.
//
// Three cases, and the middle one is the one that matters:
//   - an account with that number exists → the chat id is written onto it, and every
//     campaign can now reach them;
//   - it does not → an account is created, because a guest who has ordered by phone
//     before may have none, and refusing would leave them unreachable for ever;
//   - the number already belongs to another Telegram account → refused, because
//     merging two accounts silently moves somebody's orders, points and addresses.
func (h *Handler) linkPhone(ctx context.Context, s *models.TelegramSettings,
	chatID, telegramID int64, phone string, contactUserID int64, langCode string) {
	lang := normalizeLang(langCode)
	say := func(msg string) {
		token := s.BotToken
		go func() {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
			defer cancel()
			if err := telegram.SendMessage(ctx, token, chatID, msg); err != nil {
				log.Printf("telegram link reply to %d: %v", chatID, err)
			}
		}()
	}

	// ⚠️ Somebody else's contact card, forwarded. Telegram sets `user_id` to the
	// account the number belongs to, so a mismatch means this is not their number —
	// and accepting it would let anybody claim a stranger's account.
	if contactUserID != 0 && telegramID != 0 && contactUserID != telegramID {
		say(map[string]string{
			"ru": "Это чужой номер. Отправьте, пожалуйста, свой.",
			"en": "That is somebody else's number. Please send your own.",
		}[lang] + uzOr(lang, "Bu boshqa odamning raqami. O'zingizning raqamingizni yuboring."))
		return
	}

	normalized, ok := normalizePhone(phone)
	if !ok {
		say(map[string]string{
			"ru": "Не удалось разобрать номер.",
			"en": "That number could not be read.",
		}[lang] + uzOr(lang, "Raqamni o'qib bo'lmadi."))
		return
	}

	var byPhone models.User
	err := h.Store.Users.FindOne(ctx, bson.M{"phone": normalized}).Decode(&byPhone)
	switch {
	case err == nil:
		if byPhone.TelegramID != 0 && byPhone.TelegramID != telegramID {
			say(map[string]string{
				"ru": "Этот номер уже привязан к другому аккаунту Telegram.",
				"en": "That number is already linked to another Telegram account.",
			}[lang] + uzOr(lang, "Bu raqam boshqa Telegram hisobiga bog'langan."))
			return
		}
		_, _ = h.Store.Users.UpdateByID(ctx, byPhone.ID, bson.M{"$set": bson.M{
			"telegramId":   telegramID,
			"telegramLang": langCode,
			"updatedAt":    time.Now(),
		}})
	default:
		now := time.Now()
		_, insErr := h.Store.Users.InsertOne(ctx, models.User{
			Phone:      normalized,
			TelegramID: telegramID,
			// The door they came through, recorded rather than guessed at: this
			// account passed no SMS check, and `authProvider` is where that is
			// written down everywhere else in this app.
			AuthProvider: "telegram",
			TelegramLang: langCode,
			Addresses:    []models.UserAddress{},
			CreatedAt:    now,
			UpdatedAt:    now,
		})
		if insErr != nil {
			log.Printf("telegram link create %s: %v", normalized, insErr)
			return
		}
	}

	say(map[string]string{
		"ru": "Спасибо! Теперь мы можем сообщать вам о заказе.",
		"en": "Thank you. We can tell you about your order now.",
	}[lang] + uzOr(lang, "Rahmat! Endi buyurtma haqida xabar berib turamiz."))
}

// uzOr is the Uzbek text when no other language matched — the base language, and
// the reason the maps above carry only ru and en.
func uzOr(lang, uz string) string {
	if lang == "ru" || lang == "en" {
		return ""
	}
	return uz
}
