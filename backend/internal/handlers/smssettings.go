package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/sms"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The SMS gateway, chosen and paid for by the restaurant itself.
//
// Everything here exists because the gateway is a per-restaurant setting rather
// than a deploy-time one. Each restaurant signs its own contract with Eskiz,
// Play Mobile, getsms.uz or OneSignal, gets its own moderated sender name, and
// types the credentials into the panel. The environment variables survive only
// as the fallback for an install whose owner has never opened this page.

// smsSettings loads the gateway credentials. A missing document is not an
// error: an install that has never opened the settings page falls back to the
// environment, and then to demo.
func (h *Handler) smsSettings(ctx context.Context) *models.SMSSettings {
	var s models.SMSSettings
	if err := h.Store.SMSSettings.FindOne(ctx, bson.M{}).Decode(&s); err != nil {
		return &models.SMSSettings{}
	}
	return &s
}

// smsConfig turns the stored settings into what the sms package builds from,
// falling back to the environment while the panel has never been used.
func (h *Handler) smsConfig(s *models.SMSSettings) sms.Config {
	if strings.TrimSpace(s.Provider) == "" {
		return sms.Config{
			Provider:           h.Cfg.SMSProvider,
			From:               h.Cfg.SMSFrom,
			EskizEmail:         h.Cfg.EskizEmail,
			EskizPassword:      h.Cfg.EskizPassword,
			EskizBaseURL:       h.Cfg.EskizBaseURL,
			PlayMobileURL:      h.Cfg.PlayMobileURL,
			PlayMobileLogin:    h.Cfg.PlayMobileLogin,
			PlayMobilePassword: h.Cfg.PlayMobilePassword,
		}
	}
	return sms.Config{
		Provider:           s.Provider,
		From:               s.From,
		EskizEmail:         s.Eskiz.Email,
		EskizPassword:      s.Eskiz.Password,
		EskizBaseURL:       s.Eskiz.BaseURL,
		PlayMobileURL:      s.PlayMobile.URL,
		PlayMobileLogin:    s.PlayMobile.Login,
		PlayMobilePassword: s.PlayMobile.Password,
		GetSMSURL:          s.GetSMS.URL,
		GetSMSLogin:        s.GetSMS.Login,
		GetSMSPassword:     s.GetSMS.Password,
		GetSMSNickname:     s.GetSMS.Nickname,
		OneSignalAppID:     s.OneSignal.AppID,
		OneSignalAPIKey:    s.OneSignal.APIKey,
		OneSignalFrom:      s.OneSignal.From,
		OneSignalBaseURL:   s.OneSignal.BaseURL,
	}
}

// sender is the gateway to send through right now.
//
// **Cached, and deliberately so.** The Eskiz sender holds a bearer token it
// re-uses for ~30 days; rebuilding it per request would log in again for every
// single code, which their documentation warns against and which is the same
// trap onlinePBX's three-day key sets. The cache key is the settings document's
// own `updatedAt`, so saving the page swaps the gateway on the next request
// without a restart — and without a stale token surviving a password change.
func (h *Handler) sender(ctx context.Context) sms.Sender {
	s := h.smsSettings(ctx)
	key := fmt.Sprintf("%s|%d", s.Provider, s.UpdatedAt.UnixNano())

	h.smsMu.Lock()
	defer h.smsMu.Unlock()
	if h.smsCached != nil && h.smsKey == key {
		return h.smsCached
	}
	h.smsCached = sms.New(h.smsConfig(s))
	h.smsKey = key
	return h.smsCached
}

// smsUsable reports whether a one-time code can actually reach somebody, and
// says why not when it cannot.
//
// ⚠️ **This is the hole that shipped.** With no gateway configured the server
// fell back to the demo sender, and the demo sender's whole purpose is to hand
// the code back in the API response so a developer can finish the flow without
// a paid account. On a real tenant that meant a freshly created restaurant's
// site let **anybody log in as anybody**: ask for a code for a stranger's
// number, read it out of the JSON, and you are them — orders, addresses,
// history and all.
//
// The rule now: a code is only ever handed back when the deployment has
// deliberately asked for it (`SMS_DEMO_EXPOSE_CODE=1`, set in a developer's
// own .env and never on a hosted tenant). Everywhere else, no gateway means
// **no login at all** — which is the correct failure. "Nobody can sign in
// until you configure SMS" is a support call; "anybody can sign in as anybody"
// is not recoverable.
func (h *Handler) smsUsable(r *http.Request) (expose bool, err error) {
	return h.smsUsableFor(r, "")
}

// smsUsableFor is the same question for one specific number.
//
// The phone matters because of the test allowlist: demo mode may hand the code
// back for a number the owner typed into the settings page, and must refuse for
// every other. Callers with no particular number (the settings page's own test
// button) pass "".
func (h *Handler) smsUsableFor(r *http.Request, phone string) (expose bool, err error) {
	ctx := r.Context()
	listed := false
	if phone != "" {
		for _, p := range h.smsSettings(ctx).TestPhones {
			if p == phone {
				listed = true
				break
			}
		}
	}
	return exposeDemoCode(h.sender(ctx).Demo(), h.Cfg.SMSDemoExposeCode, listed)
}

// exposeDemoCode is the decision itself, kept free of the request and the
// database so the rule can be tested directly — it is a security boundary, and
// one that failed silently once already.
func exposeDemoCode(demo, allowed, testPhone bool) (expose bool, err error) {
	// A working gateway sends the code by SMS and never returns it, whatever
	// the flag says. The flag only ever loosens the demo path.
	if !demo {
		return false, nil
	}
	if allowed {
		return true, nil
	}
	// The owner's own number, typed into the settings page. This is the whole
	// legitimate use of demo mode on a live install — "I want to see the login
	// work before I have a gateway contract" — and it is safe for exactly one
	// reason: a stranger's number is not on the list, so the answer for them is
	// still the refusal below. Nobody can sign in as anybody by picking a
	// different number.
	if testPhone {
		return true, nil
	}
	// Refused rather than silently accepted: a guest told nothing waits for a
	// message that will never arrive, and the restaurant hears "your site is
	// broken" instead of "switch SMS on".
	return false, errSMSNotConfigured
}

// errSMSNotConfigured is shown to the guest, so it says what to do rather than
// what went wrong internally.
var errSMSNotConfigured = errors.New(
	"SMS xizmati hali sozlanmagan — restoran bilan bog'laning")

// AdminGetSMS returns the gateway settings **without the passwords** — only
// whether each one is stored. Same rule as the payment keys: a page that
// renders a gateway password puts it in every screenshot from then on.
func (h *Handler) AdminGetSMS(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	s := h.smsSettings(r.Context())
	cfg := h.smsConfig(s)
	active := h.sender(r.Context())

	httpx.JSON(w, http.StatusOK, map[string]any{
		"provider":  firstNonEmptyStr(s.Provider, sms.ProviderDemo),
		"providers": sms.Providers,
		"from":      s.From,
		// Returned in full, unlike every credential on this page: these are the
		// owner's own numbers, not a secret, and the whole point is that they
		// can see which ones are currently allowed to skip SMS.
		"testPhones": append([]string{}, s.TestPhones...),
		// What is actually sending, which is not always what was chosen: a
		// half-filled provider silently falls back to demo, and an owner who
		// cannot see that spends the afternoon looking at the wrong screen.
		"active":  active.Name(),
		"demo":    active.Demo(),
		"missing": cfg.Missing(),
		// True while the credentials still come from the server's environment
		// rather than from this page.
		"fromEnv": strings.TrimSpace(s.Provider) == "",
		// The login-code wording, resolved: the stored one or the built-in.
		"codeTemplate":        smsTemplateOf(s),
		"defaultCodeTemplate": defaultSMSTemplate,
		"codePlaceholder":     smsCodePlaceholder,
		// What one code costs, priced the way the gateway does — the same
		// counter the campaign screen uses. ⚠️ Worth showing because the cliff
		// is invisible: a single Cyrillic letter or a `oʻ` takes the message
		// out of GSM-7 and cuts the limit from 160 characters to 70, so a
		// politely-lengthened template can quietly double every login's cost.
		// The last real guest who could not get a code, and why. Shown in the
		// gateway's own words here — this page is behind an owner login, and
		// the reason is exactly what makes the difference actionable.
		"lastErrorAt": s.LastErrorAt,
		"lastError":   s.LastError,
		"codeParts":   smsParts(smsTestText(smsTemplateOf(s))),
		"codeGsm7":    isGSM7(smsTestText(smsTemplateOf(s))),
		"eskiz": map[string]any{
			"email":       s.Eskiz.Email,
			"baseUrl":     s.Eskiz.BaseURL,
			"hasPassword": s.Eskiz.Password != "",
		},
		"playmobile": map[string]any{
			"url":         s.PlayMobile.URL,
			"login":       s.PlayMobile.Login,
			"hasPassword": s.PlayMobile.Password != "",
		},
		"getsms": map[string]any{
			"url":         s.GetSMS.URL,
			"login":       s.GetSMS.Login,
			"nickname":    s.GetSMS.Nickname,
			"hasPassword": s.GetSMS.Password != "",
		},
		"onesignal": map[string]any{
			"appId":     s.OneSignal.AppID,
			"from":      s.OneSignal.From,
			"baseUrl":   s.OneSignal.BaseURL,
			"hasApiKey": s.OneSignal.APIKey != "",
		},
		"lastTestAt":    s.LastTestAt,
		"lastTestOk":    s.LastTestOk,
		"lastTest":      s.LastTest,
		"lastTestPhone": s.LastTestPhone,
	})
}

type smsSettingsRequest struct {
	Provider string `json:"provider"`
	From     string `json:"from"`
	// Numbers allowed to see a demo code in the API response. See
	// models.SMSSettings.TestPhones.
	TestPhones []string `json:"testPhones"`
	// The login-code wording, with {code} for the digits. Empty keeps the
	// built-in Uzbek text.
	CodeTemplate string `json:"codeTemplate"`
	Eskiz        struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		BaseURL  string `json:"baseUrl"`
	} `json:"eskiz"`
	PlayMobile struct {
		URL      string `json:"url"`
		Login    string `json:"login"`
		Password string `json:"password"`
	} `json:"playmobile"`
	GetSMS struct {
		URL      string `json:"url"`
		Login    string `json:"login"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
	} `json:"getsms"`
	OneSignal struct {
		AppID   string `json:"appId"`
		APIKey  string `json:"apiKey"`
		From    string `json:"from"`
		BaseURL string `json:"baseUrl"`
	} `json:"onesignal"`
}

// AdminUpdateSMS saves the gateway credentials.
//
// **An empty password means "keep the stored one"**, never "erase it" — the
// same rule as the payment keys and the kiosk secret. The form cannot show the
// password it is editing, so an owner correcting a typo in their login would
// otherwise submit a blank password field and silently switch SMS off. Nobody
// notices until a guest cannot log in.
func (h *Handler) AdminUpdateSMS(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req smsSettingsRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if !validSMSProvider(provider) {
		httpx.Error(w, http.StatusBadRequest, "noma'lum SMS provayderi: "+provider)
		return
	}
	current := h.smsSettings(r.Context())

	// ⚠️ **A template with no placeholder would send every guest a message with
	// no code in it.** Nothing errors — the gateway accepts it, the SMS
	// arrives, and the guest simply cannot log in. Refused here by name so the
	// mistake is caught while the owner is looking at the field.
	template := strings.TrimSpace(req.CodeTemplate)
	if template != "" && !strings.Contains(template, smsCodePlaceholder) {
		httpx.Error(w, http.StatusBadRequest,
			"matn ichida "+smsCodePlaceholder+" bo'lishi shart — kod o'sha yerga qo'yiladi")
		return
	}
	if template == defaultSMSTemplate {
		// Stored empty when it matches the built-in, so an install that never
		// changed anything keeps following the default if it is ever reworded.
		template = ""
	}
	// ⚠️ **Rewording invalidates the gateway's moderation.** Eskiz and Play
	// Mobile approve an exact string; the edited one is a new string and is
	// refused until approved again. Leaving the old green tick standing would
	// say "checked" about a message the gateway has never seen — and the owner
	// would find out when a guest could not sign in.
	retest := smsTemplateOf(current) != smsTemplateOf(&models.SMSSettings{CodeTemplate: template})

	// Normalised on the way in, and silently dropped when unparseable: a list
	// that only matches numbers written one particular way is a list that fails
	// to match and gives no reason. Capped, because this is a testing aid — a
	// long list of numbers that may sign in without SMS is not one.
	testPhones := []string{}
	seen := map[string]bool{}
	for _, raw := range req.TestPhones {
		phone, ok := normalizePhone(raw)
		if !ok || seen[phone] || len(testPhones) >= 5 {
			continue
		}
		seen[phone] = true
		testPhones = append(testPhones, phone)
	}

	set := bson.M{
		"provider":   provider,
		"from":       strings.TrimSpace(req.From),
		"testPhones": testPhones,
		"eskiz": models.EskizSMS{
			Email:    strings.TrimSpace(req.Eskiz.Email),
			Password: keepSecret(req.Eskiz.Password, current.Eskiz.Password),
			BaseURL:  strings.TrimSpace(req.Eskiz.BaseURL),
		},
		"playmobile": models.PlayMobileSMS{
			URL:      strings.TrimSpace(req.PlayMobile.URL),
			Login:    strings.TrimSpace(req.PlayMobile.Login),
			Password: keepSecret(req.PlayMobile.Password, current.PlayMobile.Password),
		},
		"getsms": models.GetSMS{
			URL:      strings.TrimSpace(req.GetSMS.URL),
			Login:    strings.TrimSpace(req.GetSMS.Login),
			Password: keepSecret(req.GetSMS.Password, current.GetSMS.Password),
			Nickname: strings.TrimSpace(req.GetSMS.Nickname),
		},
		"onesignal": models.OneSignalSMS{
			AppID:   strings.TrimSpace(req.OneSignal.AppID),
			APIKey:  keepSecret(req.OneSignal.APIKey, current.OneSignal.APIKey),
			From:    strings.TrimSpace(req.OneSignal.From),
			BaseURL: strings.TrimSpace(req.OneSignal.BaseURL),
		},
		"codeTemplate": template,
		"updatedAt":    time.Now(),
	}
	if retest {
		set["lastTestOk"] = false
		set["lastTest"] = "SMS matni o'zgartirildi — shlyuzda qayta moderatsiyadan " +
			"o'tkazing va sinovni takrorlang."
	}
	if _, err := h.Store.SMSSettings.UpdateOne(r.Context(), bson.M{},
		bson.M{"$set": set}, options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActSMSSettings, "settings", "sms", "SMS provayderi", provider)
	h.AdminGetSMS(w, r)
}

func validSMSProvider(p string) bool {
	return slices.Contains(sms.Providers, p)
}

type smsTestRequest struct {
	Phone string `json:"phone"`
	// Send the gateway's own fixed probe text instead of the real template.
	// Only Eskiz has one; for everyone else this is ignored.
	Probe bool `json:"probe"`
}

// AdminTestSMS sends one real message and reports what the gateway said.
//
// This is the whole point of the page. Credentials that look right and a
// contract that is signed still leave two things invisible: whether the sender
// name was actually moderated, and whether the account has any money on it.
// Both fail at the same moment — the first guest trying to log in — and the
// restaurant reads that as "the site is broken".
//
// The message goes to the owner's own number by default, so testing costs one
// SMS and bothers nobody.
func (h *Handler) AdminTestSMS(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req smsTestRequest
	_ = httpx.Decode(r, &req)

	phone := strings.TrimSpace(req.Phone)
	if phone == "" {
		if admin, err := h.adminUser(r); err == nil {
			phone = admin.Phone
		}
	}
	normalized, ok := normalizePhone(phone)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "telefon raqami noto'g'ri")
		return
	}

	s := h.smsSettings(r.Context())
	cfg := h.smsConfig(s)
	sender := h.sender(r.Context())

	now := time.Now()
	var ok2 bool
	var probed bool
	var msg string
	switch {
	case sender.Demo():
		// Not a failure worth recording as one, but never a pass either: demo
		// sends nothing at all, and reporting success here would be the single
		// most misleading thing this page could do.
		msg = "demo rejim — hech qanday SMS yuborilmadi"
		if m := cfg.Missing(); m != "" {
			msg += " (" + m + ")"
		}
	default:
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		// The probe is only ever Eskiz's: no other gateway here publishes a
		// fixed text, and silently substituting one would make the button mean
		// something different per provider.
		probe := req.Probe && sender.Name() == sms.ProviderEskiz
		text := smsTestText(smsTemplateOf(s))
		if probe {
			text = eskizProbeText
		}
		err := sender.Send(ctx, normalized, text)
		ok2 = err == nil
		switch {
		case err != nil:
			msg = err.Error()
			// Translate the one refusal every new Eskiz account hits, into
			// what it means and what to do about it. The raw body names three
			// Russian strings and no next step, which reads as a broken
			// integration rather than an unfinished signup.
			if eskizNeedsModeration(sender.Name(), msg) {
				msg = "Eskiz hisobingizda SMS matni hali moderatsiyadan " +
					"o'tmagan, shuning uchun faqat Eskiz'ning o'z sinov matnini " +
					"yuborish mumkin. Eskiz kabinetida quyidagi matnni " +
					"moderatsiyaga bering — sayt aynan shuni yuboradi: «" +
					smsTestText(smsTemplateOf(s)) + "» (kod har safar boshqacha bo'ladi). " +
					"Tasdiqlanguncha ulanishni «Eskiz sinov matni bilan» " +
					"tugmasi orqali tekshirishingiz mumkin."
			}
		case probe:
			// ⚠️ Never reported as a plain pass. It proves the credentials
			// reach Eskiz and nothing else — a guest logging in would still
			// get nothing until the template is approved.
			msg = "Eskiz'ning sinov matni yuborildi — email va parol to'g'ri. " +
				"Bu haqiqiy kod xabari yetib borishini ISBOTLAMAYDI: buning " +
				"uchun matn moderatsiyadan o'tishi kerak."
		default:
			msg = "yuborildi: " + sender.Name()
		}
		probed = probe
	}

	// ⚠️ **A probe never records as a passing test.** The stored flag is what
	// the page shows days later, next to "last checked", and the question it
	// answers is "will a guest's login code arrive" — which the probe did not
	// ask. Recording it as a pass would leave a green tick standing over an
	// account whose real messages are still refused.
	_, _ = h.Store.SMSSettings.UpdateOne(r.Context(), bson.M{},
		bson.M{"$set": bson.M{
			"lastTestAt":    now,
			"lastTestOk":    ok2 && !probed,
			"lastTest":      clampText(msg, 300),
			"lastTestPhone": normalized,
		}}, options.Update().SetUpsert(true))

	// ⚠️ A passing test clears the standing failure. It is the same question
	// asked later and answered yes, and leaving the old red line up would have
	// the page contradict itself — with the stale half being the alarming one.
	if ok2 && !probed {
		_, _ = h.Store.SMSSettings.UpdateOne(r.Context(), bson.M{},
			bson.M{"$set": bson.M{"lastError": "", "lastErrorAt": time.Time{}}})
	}

	h.logAction(r, ActSMSTest, "settings", "sms", "SMS sinovi",
		fmt.Sprintf("%s → %s", sender.Name(), normalized))

	httpx.JSON(w, http.StatusOK, map[string]any{
		// `ok` is whether the send itself succeeded; `probe` says which
		// question was asked, so the panel can show a delivered probe as
		// information rather than as proof.
		"ok": ok2, "probe": probed, "message": httpx.T(w, msg),
		"phone": normalized, "provider": sender.Name(),
		// The exact wording to submit for moderation. Shown here because the
		// owner is standing in front of this page when they find out they need
		// it, and retyping it by hand is how a template gets approved that is
		// not the one the site sends.
		"template": smsTestText(smsTemplateOf(s)),
	})
}

// smsCodePlaceholder is where the digits go in a template.
const smsCodePlaceholder = "{code}"

// defaultSMSTemplate is the built-in wording, used whenever the setting is
// empty — which is every install that has never opened the page.
//
// ⚠️ **Latin Uzbek, and both halves of that matter.** Uzbek is understood
// across the country including the regions, where Russian thins out; and plain
// Latin stays inside GSM-7, which is a 160-character message rather than the 70
// a single Cyrillic or `oʻ`-carrying character drops it to. At 48 characters
// this specific text costs one part either way — the difference is headroom, and
// an owner adding their restaurant's name is the ordinary way that headroom
// gets spent.
const defaultSMSTemplate = "Tasdiqlash kodi: " + smsCodePlaceholder +
	". Uni hech kimga bermang."

// smsCodeText renders one login-code message.
//
// ⚠️ **One function, because gateways moderate the template, not the account.**
// Eskiz and Play Mobile approve an exact wording and refuse anything else, so
// two separately-written strings mean two separate moderation requests — and
// the owner only ever submits the one they were shown. This used to be exactly
// that: the test sent "Test: SMS sozlamalari tekshirilmoqda. Kod: 000000" while
// logins sent "Tasdiqlash kodi: …", i.e. the test could pass on a moderated
// account whose real messages were still being rejected. That is the one
// outcome this page exists to prevent.
func smsCodeText(template, code string) string {
	t := strings.TrimSpace(template)
	// ⚠️ A template without the placeholder would send every guest a message
	// with no code in it — a failure with no error anywhere, on the one screen
	// nobody is watching. AdminUpdateSMS refuses to store such a template, and
	// this is the second guard: a document written before the check existed, or
	// by hand, must not be able to break every login.
	if t == "" || !strings.Contains(t, smsCodePlaceholder) {
		t = defaultSMSTemplate
	}
	return strings.ReplaceAll(t, smsCodePlaceholder, code)
}

// smsTemplateOf is the wording this install actually sends.
func smsTemplateOf(s *models.SMSSettings) string {
	if s == nil {
		return defaultSMSTemplate
	}
	t := strings.TrimSpace(s.CodeTemplate)
	if t == "" || !strings.Contains(t, smsCodePlaceholder) {
		return defaultSMSTemplate
	}
	return t
}

// smsTestText is the real template with a dummy code — the same string a guest
// logging in would receive, so a pass here means logins work.
func smsTestText(template string) string { return smsCodeText(template, "000000") }

// eskizProbeText is the fixed wording Eskiz accepts on an account whose
// template has not been moderated yet.
//
// ⚠️ **A different question, and labelled as one.** This proves the email,
// password and network path reach Eskiz; it proves nothing about whether a
// login code will arrive, because that depends on the template above being
// approved. Sending it as the ordinary test would turn a page whose entire
// purpose is catching an unmoderated sender into a green tick that certifies
// the opposite.
const eskizProbeText = "Bu Eskiz dan test"

// eskizNeedsModeration recognises Eskiz refusing an unmoderated template.
//
// Matched on the sentinel wording rather than a status code: the refusal
// arrives as a 400 like any other, and the body is in Russian. Without this the
// owner reads a raw JSON blob naming three Russian strings and concludes the
// integration is broken — when in fact the credentials are correct and the only
// thing missing is a moderation request they have not been told to make.
func eskizNeedsModeration(provider, msg string) bool {
	return provider == sms.ProviderEskiz &&
		strings.Contains(msg, "Для теста можно использовать только один из этих")
}
