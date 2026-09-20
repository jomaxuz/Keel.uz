package meta

// ---- Meta Graph / Marketing API: the client, and what its failures mean ----
//
// ⚠️ **We work inside the restaurant's own ad account.** The account is theirs,
// the card on it is theirs, and every so'm of ad spend is billed to them by
// Meta directly. This package is a third-party tool holding a token somebody
// handed us, which decides almost everything below: the token can be revoked
// without warning, the quota is counted against *their* account, and a mistake
// here spends a real restaurant's money.
//
// ⚠️ **One version, one constant.** Meta retires a Graph version roughly every
// two years and v26.0's breaking changes reach every older version anyway
// (docs/vendor/meta-marketing.md §7), so pinning per call site would buy
// nothing and cost a day of grep the first time it moves.
//
// ⚠️ **Errors are translated where the code is known and quoted where it is
// not.** "rejected", "(#17) User request limit reached" and code 190 send an
// owner to three different places — top up nothing, wait an hour, reconnect the
// account — and a single "Meta xatosi" sentence would send them to none of
// them. What we cannot recognise travels verbatim: Meta's own sentence is worth
// more than our paraphrase of it.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Version is the Graph API version every call in this package goes to.
const Version = "v26.0"

const base = "https://graph.facebook.com/"

// Client talks to Meta as one restaurant.
//
// ⚠️ **One client per restaurant, never one per process.** The quota Meta
// counts is per ad account (docs/vendor/meta-marketing.md §2); a shared client
// would make one busy restaurant's sync the reason another cannot read its own
// figures.
type Client struct {
	Token string
	HTTP  *http.Client
}

// New builds a client with the timeout every call here shares.
//
// ⚠️ **Thirty seconds, not the default none.** Creating a campaign is four
// round trips, and a Meta edge that stops answering would otherwise hold an
// owner's browser open until the request died of old age — with the campaign
// half built and nothing on the screen saying which half.
func New(token string) *Client {
	return &Client{Token: token, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

// Error is Meta's error, kept whole.
//
// ⚠️ **The subcode is carried, not folded into the code.** `613` alone is "rate
// limited"; `613/1487742` is "this ad account has hit its own hourly ceiling"
// and `613/5044001` is "too many writes per second" — same code, and the answer
// to one is to wait an hour while the answer to the other is to wait a second.
type Error struct {
	Status  int    `json:"-"`
	Code    int    `json:"code"`
	Subcode int    `json:"error_subcode"`
	Type    string `json:"type"`
	Message string `json:"message"`
	// What Meta wrote for a human to read. Often empty, and when it is present
	// it is better than anything we would write: it names the field.
	UserTitle string `json:"error_user_title"`
	UserMsg   string `json:"error_user_msg"`
}

func (e *Error) Error() string { return e.Human() }

// Revoked is the one failure that is not about this request.
//
// ⚠️ **Code 190 means the connection is gone, not that this call failed.** The
// owner removed our access, changed a password, or Meta expired the session —
// and Meta does not stop the campaigns when that happens. We simply go blind
// while their money keeps being spent, which is why this is a separate question
// with its own answer on the screen (docs/reklama-reja.md §8).
func (e *Error) Revoked() bool { return e.Code == 190 || e.Code == 102 }

// RateLimited is "come back later", in all the shapes Meta says it.
//
// The list is the one read from the rate-limiting page, not a guess: 4 and 17
// are the application and user ceilings, 613 is the calls-per-period one, 32 is
// the page-level one, and 80000–80014 are the business-use-case buckets that
// replaced them for Marketing API.
func (e *Error) RateLimited() bool {
	switch e.Code {
	case 4, 17, 32, 613:
		return true
	}
	return e.Code >= 80000 && e.Code <= 80014
}

// Human is the sentence an owner can act on.
//
// ⚠️ **Uzbek, like every other server message here** (CLAUDE.md §5): the
// translation into Russian and English happens in internal/i18n against the
// Uzbek text as the key, so a sentence added here is added there too.
func (e *Error) Human() string {
	switch {
	case e.Revoked():
		return "Meta bilan aloqa uzildi — reklama akkauntini qayta ulang"
	case e.RateLimited():
		return "Meta so'rovlar chegarasiga yetdi — birozdan keyin urinib ko'ring"
	case e.Code == 100 && e.Subcode == 33:
		// The most confusing one Meta has: "unsupported get request" is what it
		// answers when the token may not see the object at all.
		return "Meta bu obyektni ko'rsatmadi — ruxsat yoki akkaunt noto'g'ri"
	case e.Code == 200 || e.Code == 10:
		return "Meta ruxsat bermadi — reklama akkauntida sizning rolingiz yetarli emas"
	case e.Code == 2:
		return "Meta vaqtincha javob bermayapti — birozdan keyin urinib ko'ring"
	}
	if msg := strings.TrimSpace(e.UserMsg); msg != "" {
		return msg
	}
	if msg := strings.TrimSpace(e.Message); msg != "" {
		return msg
	}
	return fmt.Sprintf("Meta xatosi (%d)", e.Status)
}

// AsError digs a *Error out of an error chain, or nil.
func AsError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}

// do performs one call and decodes the body into out.
//
// ⚠️ **The token travels in a header, never in the query string.** Meta accepts
// `?access_token=`, and every proxy, access log and error report on the way
// would then hold a credential that can spend money.
func (c *Client) do(
	ctx context.Context, method, path string, params url.Values,
	body io.Reader, contentType string, out any,
) error {
	if c.Token == "" {
		return &Error{Code: 190, Message: "no token"}
	}
	u := base + Version + "/" + strings.TrimPrefix(path, "/")
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	cl := c.HTTP
	if cl == nil {
		cl = &http.Client{Timeout: 30 * time.Second}
	}
	res, err := cl.Do(req)
	if err != nil {
		return fmt.Errorf("Meta javob bermadi: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	// ⚠️ Capped: an insights reply for a long window is large, and a truncated
	// JSON failing to parse is a better outcome than a container holding a
	// hundred megabytes because somebody asked for a year.
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))

	if res.StatusCode >= 400 {
		var wrap struct {
			Error Error `json:"error"`
		}
		_ = json.Unmarshal(raw, &wrap)
		e := wrap.Error
		e.Status = res.StatusCode
		if e.Code == 0 && e.Message == "" {
			e.Message = strings.TrimSpace(string(raw))
		}
		return &e
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("Meta javobi o'qilmadi: %w", err)
	}
	return nil
}

// Get reads one edge.
func (c *Client) Get(ctx context.Context, path string, params url.Values, out any) error {
	return c.do(ctx, http.MethodGet, path, params, nil, "", out)
}

// Post writes one object.
//
// ⚠️ **Form encoded, not JSON.** The Marketing API takes `application/x-www-
// form-urlencoded` with nested structures as JSON *strings* inside a field
// (`targeting`, `object_story_spec`, `promoted_object`), and a JSON body is
// accepted by some edges and silently mis-read by others. One shape everywhere
// is the shape the documentation is written in.
func (c *Client) Post(ctx context.Context, path string, form url.Values, out any) error {
	return c.do(ctx, http.MethodPost, path, nil,
		strings.NewReader(form.Encode()),
		"application/x-www-form-urlencoded", out)
}

// PostJSON writes a JSON body — the Conversions API's shape, and only its.
func (c *Client) PostJSON(ctx context.Context, path string, body any, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, path, nil,
		bytes.NewReader(raw), "application/json", out)
}

// PostFile uploads one file as multipart.
//
// ⚠️ **The file name must carry an extension** — `sample.jpg`, never `sample`.
// Meta rejects the upload otherwise, and the message it returns says nothing
// about the name (docs/vendor/meta-marketing.md §4.4).
func (c *Client) PostFile(
	ctx context.Context, path, field, filename string, data []byte,
	extra url.Values, out any,
) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, vs := range extra {
		for _, v := range vs {
			_ = mw.WriteField(k, v)
		}
	}
	part, err := mw.CreateFormFile(field, filename)
	if err != nil {
		return err
	}
	if _, err := part.Write(data); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, path, nil, &buf, mw.FormDataContentType(), out)
}

// JSONField renders a nested structure for a form field.
func JSONField(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(raw)
}
