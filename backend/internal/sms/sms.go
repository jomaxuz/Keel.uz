// Package sms sends transactional SMS (login codes) through an Uzbek gateway.
//
// Providers (set SMS_PROVIDER):
//
//	demo       — nothing is sent; the code is logged and returned by the API so
//	             the flow can be exercised without a paid account. DEFAULT.
//	eskiz      — https://notify.eskiz.uz  (email+password → bearer token)
//	playmobile — https://playmobile.uz broker-api (basic auth)
//
// Both real providers require a signed contract and a moderated sender name /
// message template in Uzbekistan, so `demo` stays the default for development.
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

// Config holds provider credentials read from the environment.
type Config struct {
	Provider string // demo | eskiz | playmobile
	From     string // sender name / originator
	// Eskiz
	EskizEmail    string
	EskizPassword string
	EskizBaseURL  string
	// Play Mobile
	PlayMobileURL      string
	PlayMobileLogin    string
	PlayMobilePassword string
}

// New builds the configured Sender, falling back to the demo one when the
// credentials for a real provider are missing.
func New(cfg Config) Sender {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "eskiz":
		if cfg.EskizEmail == "" || cfg.EskizPassword == "" {
			log.Print("sms: ESKIZ_EMAIL/ESKIZ_PASSWORD missing — falling back to demo provider")
			return &DemoSender{}
		}
		base := cfg.EskizBaseURL
		if base == "" {
			base = "https://notify.eskiz.uz"
		}
		return &EskizSender{
			BaseURL:  strings.TrimRight(base, "/"),
			Email:    cfg.EskizEmail,
			Password: cfg.EskizPassword,
			From:     cfg.From,
			client:   &http.Client{Timeout: 15 * time.Second},
		}
	case "playmobile":
		if cfg.PlayMobileLogin == "" || cfg.PlayMobilePassword == "" {
			log.Print("sms: PLAYMOBILE_LOGIN/PLAYMOBILE_PASSWORD missing — falling back to demo provider")
			return &DemoSender{}
		}
		url := cfg.PlayMobileURL
		if url == "" {
			url = "http://91.204.239.44/broker-api/send"
		}
		return &PlayMobileSender{
			URL:      url,
			Login:    cfg.PlayMobileLogin,
			Password: cfg.PlayMobilePassword,
			From:     cfg.From,
			client:   &http.Client{Timeout: 15 * time.Second},
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
