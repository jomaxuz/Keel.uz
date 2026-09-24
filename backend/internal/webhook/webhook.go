// Package webhook is the wire half of outgoing webhooks: how a delivery is
// signed, which addresses it may go to, and how long to wait before trying
// again. The queue and the events live in handlers/webhooks.go.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"restaurant-backend/internal/netguard"
)

// Header names. ⚠️ Part of the published contract: a receiver's code reads
// them by name, so renaming one breaks every integration silently.
const (
	HeaderSignature = "Keel-Signature"
	HeaderEvent     = "Keel-Event"
	HeaderEventID   = "Keel-Event-Id"
	HeaderDelivery  = "Keel-Delivery-Id"
)

// Timeout is how long a receiver has to answer. Ten seconds is generous for
// "write it down and say 200"; a receiver that does the real work before
// answering will time out on a busy evening, and the docs say so.
const Timeout = 10 * time.Second

// Sign returns the Keel-Signature header value for body sent at ts:
// `t=<unix seconds>,v1=<hex HMAC-SHA256(secret, "<t>.<body>")>`.
//
// ⚠️ **The timestamp is inside the signed string**, not beside it. A signature
// over the body alone lets anybody who once saw a delivery replay it forever;
// with the time signed, the receiver can refuse anything older than a few
// minutes and the replay has nothing left to forge.
func Sign(secret string, ts int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	return "t=" + strconv.FormatInt(ts, 10) + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

// Verify is Sign's other half — the receiver's check, kept here so the tests
// (and the docs' example) exercise exactly what a receiver has to write.
func Verify(secret, header string, body []byte, now time.Time, tolerance time.Duration) bool {
	var ts int64
	var sig string
	for _, part := range strings.Split(header, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return false
			}
			ts = n
		case "v1":
			sig = v
		}
	}
	if ts == 0 || sig == "" {
		return false
	}
	if d := now.Sub(time.Unix(ts, 0)); d > tolerance || d < -tolerance {
		return false
	}
	want := Sign(secret, ts, body)
	_, wantSig, _ := strings.Cut(want, ",v1=")
	return hmac.Equal([]byte(sig), []byte(wantSig))
}

// CheckURL refuses an address a webhook may not be registered for.
//
// ⚠️ **HTTPS only.** The body carries the guest's name and phone number, and
// the signature proves who sent it, not who else read it on the way.
func CheckURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return nil, errors.New("manzil noto'g'ri")
	}
	if u.Scheme != "https" {
		return nil, errors.New("manzil https:// bilan boshlanishi kerak")
	}
	if u.User != nil {
		// A password in the URL would be stored and shown back in the panel in
		// clear; the signature is the authentication.
		return nil, errors.New("manzil ichida login va parol bo'lmasligi kerak")
	}
	if err := netguard.PublicHost(u.Hostname()); err != nil {
		return nil, err
	}
	return u, nil
}

// Client is the HTTP client deliveries go out on.
//
//   - Every connection is checked at dial time (netguard.Control), so a name
//     that resolves somewhere private — now, or after the owner's DNS changes —
//     is refused on the attempt that would have reached it.
//   - ⚠️ **Redirects are not followed.** A 302 to http://mongo:27017 is the
//     classic way round an address check made once, on the first URL; and a
//     receiver that moved should be re-registered, not chased.
//   - No proxy from the environment: a delivery must leave from this server.
var Client = &http.Client{
	Timeout: Timeout,
	Transport: &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
			Control: netguard.Control,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: Timeout,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
	},
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// Request is one delivery attempt.
type Request struct {
	URL        string
	Secret     string
	Event      string
	EventID    string
	DeliveryID string
	Body       []byte
}

// Result is what came back. ⚠️ **The response body is never kept**: it is
// whatever the receiver chose to print, and a receiver that is really an
// internal service we should not have reached would be printing its secrets
// into our database and the panel.
type Result struct {
	StatusCode int
	Err        string
	Took       time.Duration
}

// OK reports a delivery the receiver accepted — any 2xx.
func (r Result) OK() bool { return r.Err == "" && r.StatusCode >= 200 && r.StatusCode < 300 }

// Send makes one attempt.
func Send(ctx context.Context, c *http.Client, req Request, now time.Time) Result {
	start := time.Now()
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		return Result{Err: err.Error()}
	}
	hr.Header.Set("Content-Type", "application/json")
	hr.Header.Set("User-Agent", "Keel-Webhooks/1")
	hr.Header.Set(HeaderEvent, req.Event)
	hr.Header.Set(HeaderEventID, req.EventID)
	hr.Header.Set(HeaderDelivery, req.DeliveryID)
	hr.Header.Set(HeaderSignature, Sign(req.Secret, now.Unix(), req.Body))
	resp, err := c.Do(hr)
	took := time.Since(start)
	if err != nil {
		return Result{Err: describe(err), Took: took}
	}
	// Drained (a little) so the connection can be reused, then dropped.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
	_ = resp.Body.Close()
	res := Result{StatusCode: resp.StatusCode, Took: took}
	if !res.OK() {
		res.Err = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return res
}

// describe shortens a transport error to what the owner can act on. The raw
// Go error names our own resolver and dial path, which says nothing to them.
func describe(err error) string {
	switch {
	case errors.Is(err, netguard.ErrBlocked):
		return "blocked: address is not on the public internet"
	case errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "Client.Timeout"):
		return "timeout: no answer within " + Timeout.String()
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return "dns: " + dns.Name + " not found"
	}
	msg := err.Error()
	if len(msg) > 200 {
		msg = msg[:200]
	}
	return msg
}

// Backoff is how long to wait after the n-th failed attempt (n from 1), and
// false once there are no attempts left.
//
// About two days end to end: long enough to ride out a receiver that is down
// over a weekend night, short enough that an order is not announced after the
// guest has eaten it and forgotten.
func Backoff(n int) (time.Duration, bool) {
	if n < 1 || n > len(schedule) {
		return 0, false
	}
	return schedule[n-1], true
}

var schedule = []time.Duration{
	time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	6 * time.Hour,
	12 * time.Hour,
	24 * time.Hour,
}

// MaxAttempts is the first attempt plus one per step of the schedule.
var MaxAttempts = len(schedule) + 1
