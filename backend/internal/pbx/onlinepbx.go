// Package pbx talks to the restaurant's phone system.
//
// Only onlinePBX for now, but the shape is deliberately provider-neutral: what
// the call centre needs from a PBX is the same everywhere — tell me a call is
// ringing, tell me what became of it, and dial a number for me.
//
// onlinePBX's HTTP API (https://api2.onlinepbx.ru) authenticates the way S3
// does: an API key from the control panel is exchanged once for a `key_id` and
// `key` pair, and every later request carries them in an `x-pbx-authentication`
// header. **The pair lives three days from its last use**, so it is cached —
// the docs warn that authenticating four or five times a second corrupts
// sessions, because each authentication issues a new key and invalidates the
// last.
//
// Bodies are `application/x-www-form-urlencoded`, not JSON. Answers are JSON
// with a string `status` ("1" for success) and the payload under `data`.
package pbx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client is one onlinePBX account.
type Client struct {
	domain string
	apiKey string
	base   string
	http   *http.Client

	mu       sync.Mutex
	keyID    string
	key      string
	obtained time.Time
}

// Config is what the panel stores.
type Config struct {
	// The account's own subdomain, e.g. "example.onpbx.ru".
	Domain string
	// The API key from the onlinePBX control panel.
	APIKey string
	// Overridable for tests; empty means the live host.
	BaseURL string
}

func New(cfg Config) *Client {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = "https://api2.onlinepbx.ru"
	}
	return &Client{
		domain: strings.TrimSpace(cfg.Domain),
		apiKey: strings.TrimSpace(cfg.APIKey),
		base:   base,
		http:   &http.Client{Timeout: 20 * time.Second},
	}
}

// Configured reports whether there is anything to talk to.
func (c *Client) Configured() bool {
	return c != nil && c.domain != "" && c.apiKey != ""
}

// response is the envelope every endpoint answers with.
type response struct {
	Status    string          `json:"status"`
	Comment   string          `json:"comment"`
	IsNotAuth bool            `json:"isNotAuth"`
	ErrorCode string          `json:"errorCode"`
	Data      json.RawMessage `json:"data"`
}

func (r *response) ok() bool { return r.Status == "1" }

func (r *response) err() error {
	msg := strings.TrimSpace(r.Comment)
	if msg == "" {
		msg = r.ErrorCode
	}
	if msg == "" {
		msg = "onlinePBX so'rovni bajarmadi"
	}
	return fmt.Errorf("onlinePBX: %s", msg)
}

// auth exchanges the API key for a session pair, or returns the cached one.
//
// Re-authenticating is not free: each call issues a *new* key and kills the
// previous one, so a client that authenticated per request would spend its life
// invalidating itself. The pair is refreshed a day before its three-day life
// runs out.
func (c *Client) auth(ctx context.Context, force bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !force && c.keyID != "" && time.Since(c.obtained) < 48*time.Hour {
		return c.keyID + ":" + c.key, nil
	}
	form := url.Values{"auth_key": {c.apiKey}}
	res, err := c.post(ctx, "/auth.json", form, "")
	if err != nil {
		return "", err
	}
	if !res.ok() {
		return "", res.err()
	}
	var data struct {
		Key   string `json:"key"`
		KeyID string `json:"key_id"`
	}
	if err := json.Unmarshal(res.Data, &data); err != nil || data.Key == "" {
		return "", fmt.Errorf("onlinePBX: API kalit qabul qilinmadi")
	}
	c.keyID, c.key, c.obtained = data.KeyID, data.Key, time.Now()
	return c.keyID + ":" + c.key, nil
}

// post sends one form-encoded request.
func (c *Client) post(ctx context.Context, path string, form url.Values, authHeader string) (*response, error) {
	endpoint := fmt.Sprintf("%s/%s%s", c.base, c.domain, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if authHeader != "" {
		req.Header.Set("x-pbx-authentication", authHeader)
	}
	httpRes, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("onlinePBX: ulanib bo'lmadi: %w", err)
	}
	defer httpRes.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(httpRes.Body, 8<<20))
	var out response
	if err := json.Unmarshal(raw, &out); err != nil {
		// A non-JSON body means the request never reached the API at all —
		// almost always a domain that does not exist. Saying "could not read
		// the answer" sends the owner looking at their API key, which is the
		// one thing that is probably fine.
		switch httpRes.StatusCode {
		case http.StatusForbidden, http.StatusUnauthorized:
			return nil, fmt.Errorf(
				"onlinePBX: %s domeni qabul qilmadi (%s) — domen va API kalitni tekshiring",
				c.domain, httpRes.Status)
		case http.StatusNotFound:
			return nil, fmt.Errorf("onlinePBX: %s domeni topilmadi", c.domain)
		}
		return nil, fmt.Errorf("onlinePBX: kutilmagan javob (%s)", httpRes.Status)
	}
	return &out, nil
}

// call runs an authenticated request, re-authenticating once if the session
// expired. `isNotAuth` is onlinePBX's own way of saying "your key is stale" —
// the docs are explicit that this, and only this, is when to ask for a new one.
func (c *Client) call(ctx context.Context, path string, form url.Values) (*response, error) {
	header, err := c.auth(ctx, false)
	if err != nil {
		return nil, err
	}
	res, err := c.post(ctx, path, form, header)
	if err != nil {
		return nil, err
	}
	if res.IsNotAuth {
		if header, err = c.auth(ctx, true); err != nil {
			return nil, err
		}
		if res, err = c.post(ctx, path, form, header); err != nil {
			return nil, err
		}
	}
	if !res.ok() {
		return nil, res.err()
	}
	return res, nil
}

// Ping proves the credentials work. The balance endpoint is the cheapest
// authenticated read onlinePBX offers, and its answer is worth showing.
func (c *Client) Ping(ctx context.Context) (string, error) {
	if !c.Configured() {
		return "", fmt.Errorf("onlinePBX: domen yoki API kalit kiritilmagan")
	}
	res, err := c.call(ctx, "/balance/get.json", url.Values{})
	if err != nil {
		return "", err
	}
	var data struct {
		Balance any `json:"balance"`
	}
	_ = json.Unmarshal(res.Data, &data)
	if data.Balance != nil {
		return fmt.Sprintf("%s · balans %v", c.domain, data.Balance), nil
	}
	return c.domain, nil
}

// Call dials `to` from the operator's extension `from`.
//
// The order matters and is not the obvious one: onlinePBX rings the **first**
// number first, so `from` is the operator's own handset. They pick up, and only
// then does the customer's phone ring. Reversing them would make the customer
// wait while the operator finds their headset.
func (c *Client) Call(ctx context.Context, from, to string) (string, error) {
	if !c.Configured() {
		return "", fmt.Errorf("onlinePBX: sozlanmagan")
	}
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	if from == "" {
		return "", fmt.Errorf("onlinePBX: operatorning ichki raqami ko'rsatilmagan")
	}
	if to == "" {
		return "", fmt.Errorf("onlinePBX: qaysi raqamga qo'ng'iroq qilish kerak?")
	}
	res, err := c.call(ctx, "/call/now.json", url.Values{
		"from": {from}, "to": {to},
	})
	if err != nil {
		return "", err
	}
	var data struct {
		UUID string `json:"uuid"`
	}
	_ = json.Unmarshal(res.Data, &data)
	return data.UUID, nil
}

// HistoryEntry is one call as onlinePBX recorded it.
type HistoryEntry struct {
	UUID              string `json:"uuid"`
	CallerIDNumber    string `json:"caller_id_number"`
	DestinationNumber string `json:"destination_number"`
	StartStamp        int64  `json:"start_stamp"`
	EndStamp          int64  `json:"end_stamp"`
	Duration          int    `json:"duration"`
	UserTalkTime      int    `json:"user_talk_time"`
	HangupCause       string `json:"hangup_cause"`
	// "inbound" | "outbound" | "local"
	AccountCode string `json:"accountcode"`
}

// History looks a call up by its uuid — used to fill in what a webhook did not
// carry, and to recover a call whose webhook never arrived.
func (c *Client) History(ctx context.Context, uuid string) (*HistoryEntry, error) {
	res, err := c.call(ctx, "/mongo_history/search.json", url.Values{"uuid": {uuid}})
	if err != nil {
		return nil, err
	}
	var rows []HistoryEntry
	// The numeric fields come back as numbers or strings depending on the row,
	// so a failed decode is not fatal — the caller treats a nil entry as "not
	// found" rather than as an error worth showing anyone.
	if err := json.Unmarshal(res.Data, &rows); err != nil || len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

// RecordingURL asks for a temporary link to the call's audio.
//
// Not stored as a permanent link: onlinePBX hands out signed download URLs, and
// one saved in our database would quietly stop working. The panel asks for a
// fresh one at the moment somebody presses play.
func (c *Client) RecordingURL(ctx context.Context, uuid string) (string, error) {
	res, err := c.call(ctx, "/mongo_history/search.json", url.Values{
		"uuid": {uuid}, "download": {"1"},
	})
	if err != nil {
		return "", err
	}
	var link string
	if err := json.Unmarshal(res.Data, &link); err != nil {
		return "", fmt.Errorf("onlinePBX: yozuv havolasi qaytmadi")
	}
	return link, nil
}
