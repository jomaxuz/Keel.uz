// Package sms sends transactional SMS (login codes) through an Uzbek gateway.
//
// Providers:
//
//	demo       — nothing is sent; the code is logged and returned by the API so
//	             the flow can be exercised without a paid account. DEFAULT.
//	eskiz      — https://notify.eskiz.uz  (email+password → bearer token)
//	playmobile — https://playmobile.uz broker-api (basic auth)
//	getsms     — https://getsms.uz gateway (login+password in the body)
//	onesignal  — https://onesignal.com (SMS channel, rides on their Twilio)
//
// Every real provider requires a signed contract and a moderated sender name /
// message template in Uzbekistan, so `demo` stays the default for development.
//
// **Which one is in use is a per-restaurant setting, not a deploy-time one.**
// Each restaurant signs its own contract with its own gateway and types its own
// credentials into the panel; the environment variables remain only as the
// fallback for a fresh install that has never opened that page. A shared
// platform account would put every restaurant's login codes — and their cost —
// on one contract, and one restaurant's moderation problem would silence
// everyone else's.
package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Sender delivers a text message to a phone number in 998XXXXXXXXX form.
type Sender interface {
	Send(ctx context.Context, phone, text string) error
	// Demo reports whether messages are faked (the API may then reveal the
	// code to the caller — never do this with a real provider).
	Demo() bool
	Name() string
}

// Provider ids. These are stored in the database and shown in the panel, so
// they are part of the contract with the frontend — never renamed.
const (
	ProviderDemo       = "demo"
	ProviderEskiz      = "eskiz"
	ProviderPlayMobile = "playmobile"
	ProviderGetSMS     = "getsms"
	ProviderOneSignal  = "onesignal"
)

// Providers lists every id the panel may offer, in the order it shows them.
var Providers = []string{
	ProviderEskiz, ProviderPlayMobile, ProviderGetSMS, ProviderOneSignal, ProviderDemo,
}

// Default endpoints. Kept here rather than in the panel so that a restaurant
// only ever types what its own contract gave it.
const (
	EskizDefaultURL      = "https://notify.eskiz.uz"
	PlayMobileDefaultURL = "http://91.204.239.44/broker-api/send"
	GetSMSDefaultURL     = "http://185.8.212.184/smsgateway/"
	OneSignalDefaultURL  = "https://api.onesignal.com"
)

// Config holds one restaurant's provider credentials, whether they came from
// the database (the normal case) or from the environment (a fresh install).
type Config struct {
	Provider string // demo | eskiz | playmobile | getsms | onesignal
	From     string // sender name / originator, as moderated by the operator
	// Eskiz
	EskizEmail    string
	EskizPassword string
	EskizBaseURL  string
	// Play Mobile
	PlayMobileURL      string
	PlayMobileLogin    string
	PlayMobilePassword string
	// getsms.uz
	GetSMSURL      string
	GetSMSLogin    string
	GetSMSPassword string
	GetSMSNickname string // the sender name registered in their system
	// OneSignal
	OneSignalAppID   string
	OneSignalAPIKey  string
	OneSignalFrom    string // Messaging Service id or an E.164 number
	OneSignalBaseURL string
}

// Missing reports which credential a chosen provider still needs, or "" when it
// is ready to send.
//
// Returned as a reason rather than a bool because the panel shows it: an owner
// who ticked "Eskiz" and saved is otherwise told only that SMS does not work,
// which is exactly what they already knew.
func (c Config) Missing() string {
	switch normalizeProvider(c.Provider) {
	case ProviderEskiz:
		if c.EskizEmail == "" || c.EskizPassword == "" {
			return "eskiz: email va parol"
		}
	case ProviderPlayMobile:
		if c.PlayMobileLogin == "" || c.PlayMobilePassword == "" {
			return "playmobile: login va parol"
		}
	case ProviderGetSMS:
		if c.GetSMSLogin == "" || c.GetSMSPassword == "" {
			return "getsms: login va parol"
		}
	case ProviderOneSignal:
		if c.OneSignalAppID == "" || c.OneSignalAPIKey == "" {
			return "onesignal: App ID va REST API kaliti"
		}
	}
	return ""
}

func normalizeProvider(p string) string {
	return strings.ToLower(strings.TrimSpace(p))
}

func httpClient() *http.Client { return &http.Client{Timeout: 15 * time.Second} }

func orDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

// New builds the configured Sender, falling back to the demo one when the
// credentials for a real provider are missing.
//
// Falling back rather than failing is deliberate: a half-configured gateway
// must not take the site's login down with it. The panel says loudly that demo
// is in use, and `Demo()` keeps the real code visible so somebody can still get
// in and fix the settings.
func New(cfg Config) Sender {
	provider := normalizeProvider(cfg.Provider)
	if missing := cfg.Missing(); missing != "" {
		log.Printf("sms: %s not configured (%s) — falling back to demo provider", provider, missing)
		return &DemoSender{}
	}
	switch provider {
	case ProviderEskiz:
		return &EskizSender{
			BaseURL:  strings.TrimRight(orDefault(cfg.EskizBaseURL, EskizDefaultURL), "/"),
			Email:    cfg.EskizEmail,
			Password: cfg.EskizPassword,
			From:     cfg.From,
			client:   httpClient(),
		}
	case ProviderPlayMobile:
		return &PlayMobileSender{
			URL:      orDefault(cfg.PlayMobileURL, PlayMobileDefaultURL),
			Login:    cfg.PlayMobileLogin,
			Password: cfg.PlayMobilePassword,
			From:     cfg.From,
			client:   httpClient(),
		}
	case ProviderGetSMS:
		return &GetSMSSender{
			URL:      orDefault(cfg.GetSMSURL, GetSMSDefaultURL),
			Login:    cfg.GetSMSLogin,
			Password: cfg.GetSMSPassword,
			Nickname: orDefault(cfg.GetSMSNickname, cfg.From),
			client:   httpClient(),
		}
	case ProviderOneSignal:
		return &OneSignalSender{
			BaseURL: strings.TrimRight(orDefault(cfg.OneSignalBaseURL, OneSignalDefaultURL), "/"),
			AppID:   cfg.OneSignalAppID,
			APIKey:  cfg.OneSignalAPIKey,
			From:    orDefault(cfg.OneSignalFrom, cfg.From),
			client:  httpClient(),
		}
	default:
		return &DemoSender{}
	}
}

// ---- demo ----

// DemoSender logs the message instead of sending it.
type DemoSender struct{}

func (d *DemoSender) Send(_ context.Context, phone, text string) error {
	log.Printf("sms[demo]: to %s: %s", phone, text)
	return nil
}
func (d *DemoSender) Demo() bool   { return true }
func (d *DemoSender) Name() string { return "demo" }

// ---- eskiz.uz ----

// EskizSender talks to https://notify.eskiz.uz. The login endpoint returns a
// bearer token valid for ~30 days; it is cached and re-fetched on 401.
type EskizSender struct {
	BaseURL  string
	Email    string
	Password string
	From     string

	client *http.Client
	mu     sync.Mutex
	token  string
}

func (e *EskizSender) Demo() bool   { return false }
func (e *EskizSender) Name() string { return "eskiz" }

func (e *EskizSender) Send(ctx context.Context, phone, text string) error {
	if err := e.send(ctx, phone, text, false); err != nil {
		// One retry with a fresh token: the cached one may have expired.
		if strings.Contains(err.Error(), "401") {
			return e.send(ctx, phone, text, true)
		}
		return err
	}
	return nil
}

func (e *EskizSender) send(ctx context.Context, phone, text string, forceLogin bool) error {
	token, err := e.authToken(ctx, forceLogin)
	if err != nil {
		return err
	}
	from := e.From
	if from == "" {
		from = "4546" // Eskiz's default test sender
	}
	body := map[string]string{
		"mobile_phone": phone,
		"message":      text,
		"from":         from,
	}
	buf, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		e.BaseURL+"/api/message/sms/send", bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := e.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("eskiz send: %d %s", res.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

func (e *EskizSender) authToken(ctx context.Context, force bool) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.token != "" && !force {
		return e.token, nil
	}
	buf, _ := json.Marshal(map[string]string{"email": e.Email, "password": e.Password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		e.BaseURL+"/api/auth/login", bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := e.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return "", fmt.Errorf("eskiz login: %d %s", res.StatusCode, strings.TrimSpace(string(msg)))
	}
	var out struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Data.Token == "" {
		return "", fmt.Errorf("eskiz login: empty token")
	}
	e.token = out.Data.Token
	return e.token, nil
}

// ---- playmobile.uz (smsxabar) ----

// PlayMobileSender posts to the broker-api endpoint with HTTP basic auth.
type PlayMobileSender struct {
	URL      string
	Login    string
	Password string
	From     string

	client *http.Client
}

func (p *PlayMobileSender) Demo() bool   { return false }
func (p *PlayMobileSender) Name() string { return "playmobile" }

func (p *PlayMobileSender) Send(ctx context.Context, phone, text string) error {
	payload := map[string]any{
		"messages": []map[string]any{{
			"recipient":  phone,
			"message-id": fmt.Sprintf("code-%d", time.Now().UnixNano()),
			"sms": map[string]any{
				"originator": p.From,
				"content":    map[string]string{"text": text},
			},
		}},
	}
	buf, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.SetBasicAuth(p.Login, p.Password)

	res, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("playmobile send: %d %s", res.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

// ---- getsms.uz ----

// GetSMSSender posts to the getsms.uz gateway. Credentials travel in the body
// rather than in a header, and the payload is a batch even for one message.
type GetSMSSender struct {
	URL      string
	Login    string
	Password string
	Nickname string // the moderated sender name

	client *http.Client
}

func (g *GetSMSSender) Demo() bool   { return false }
func (g *GetSMSSender) Name() string { return ProviderGetSMS }

// getSMSResponse is one element of the array the gateway answers with. It
// reports failures **inside a 200**, so the status code alone proves nothing.
type getSMSResponse struct {
	Error     any    `json:"error"`
	ErrorText string `json:"error_text"`
	ErrorNo   any    `json:"error_no"`
	MessageID any    `json:"message_id"`
	RequestID any    `json:"request_id"`
}

func (g *GetSMSSender) Send(ctx context.Context, phone, text string) error {
	payload := map[string]any{
		"login":    g.Login,
		"password": g.Password,
		"nickname": g.Nickname,
		// Up to 100 per call; we only ever send one login code at a time.
		"data": []map[string]string{{"phone": phone, "text": text}},
	}
	buf, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.URL, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	res, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	if res.StatusCode >= 300 {
		return fmt.Errorf("getsms send: %d %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	// A 200 carrying an error element is the normal way this gateway refuses:
	// a bad password, an unmoderated nickname and a number outside the
	// contract all arrive looking like success.
	var out []getSMSResponse
	if err := json.Unmarshal(body, &out); err != nil {
		// Not the documented shape. Treat an "error" word in the body as a
		// refusal rather than reporting a delivery nobody made.
		if strings.Contains(strings.ToLower(string(body)), "error") {
			return fmt.Errorf("getsms send: %s", strings.TrimSpace(string(body)))
		}
		return nil
	}
	for _, item := range out {
		if failed(item.Error) || item.ErrorText != "" || failed(item.ErrorNo) {
			msg := item.ErrorText
			if msg == "" {
				msg = fmt.Sprintf("%v", item.Error)
			}
			return fmt.Errorf("getsms send: %s", strings.TrimSpace(msg))
		}
	}
	return nil
}

// failed reads the gateway's error field, which is a number in some responses
// and a string in others. Zero and empty mean "no error".
func failed(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		s := strings.TrimSpace(t)
		return s != "" && s != "0"
	}
	return false
}

// ---- onesignal.com ----

// OneSignalSender sends through OneSignal's SMS channel, which rides on their
// Twilio connection rather than on an Uzbek gateway.
//
// Worth knowing before choosing it: delivery to Uzbek networks is international
// A2P traffic, so it costs more and the sender name is whatever Twilio shows,
// not a locally moderated alphanumeric one. It earns its place for a restaurant
// that already runs OneSignal for push.
type OneSignalSender struct {
	BaseURL string
	AppID   string
	APIKey  string
	From    string // Messaging Service id or an E.164 number

	client *http.Client
}

func (o *OneSignalSender) Demo() bool   { return false }
func (o *OneSignalSender) Name() string { return ProviderOneSignal }

func (o *OneSignalSender) Send(ctx context.Context, phone, text string) error {
	body := map[string]any{
		"app_id":                o.AppID,
		"target_channel":        "sms",
		"include_phone_numbers": []string{e164(phone)},
		"contents":              map[string]string{"en": text},
	}
	if o.From != "" {
		body["sms_from"] = o.From
	}
	buf, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		o.BaseURL+"/notifications?c=sms", bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Key "+o.APIKey)

	res, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body2, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	if res.StatusCode >= 300 {
		return fmt.Errorf("onesignal send: %d %s", res.StatusCode, strings.TrimSpace(string(body2)))
	}
	// OneSignal answers 200 with an `errors` field when it accepted nothing —
	// most often "All included players are not subscribed", which for SMS
	// means the number was rejected.
	var out struct {
		ID         string `json:"id"`
		Recipients int    `json:"recipients"`
		Errors     any    `json:"errors"`
	}
	if err := json.Unmarshal(body2, &out); err != nil {
		return nil // accepted, shape unknown
	}
	if out.Errors != nil {
		return fmt.Errorf("onesignal send: %v", out.Errors)
	}
	return nil
}

// e164 turns our internal 998XXXXXXXXX into the +998XXXXXXXXX OneSignal
// requires. Every other gateway here wants it without the plus, which is why
// the conversion lives at this one call site rather than in normalizePhone.
func e164(phone string) string {
	phone = strings.TrimSpace(phone)
	if strings.HasPrefix(phone, "+") {
		return phone
	}
	return "+" + phone
}
