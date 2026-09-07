// Package didox talks to Uzbekistan's electronic document operator.
//
// What it is for, in one sentence: a delivery that arrives with an electronic
// invoice already filed at the tax committee should not be retyped into the
// panel by hand.
//
// ⚠️ **This package reads and drafts. It never signs.** Every signature in
// Didox is a PKCS#7 made by an E-IMZO key on the person's own computer, with a
// timestamp attached by the operator; the server holds no key and must not.
// So there is no `Sign` here — not because it was hard, but because the only
// way to write it would be to invent a signature, and an unsigned document that
// the panel calls signed is missing from a tax return that the owner believes
// is complete. The panel says whose job it is and links to didox.uz.
//
// ⚠️ **Two headers, two owners.** `Partner-Authorization` is Keel's token,
// issued once for the platform; `user-key` is the customer's own session,
// which lives 360 minutes and is obtained with their own password or key. They
// are separate fields in the settings for the same reason the map keys and the
// payment keys are separate: one of them being wrong should not look like the
// other one being wrong.
//
// Read copy of the operator's documentation: docs/vendor/didox.md.
package didox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// The two sets of rails Didox publishes.
const (
	ProdURL    = "https://api-partners.didox.uz"
	SandboxURL = "https://testapi3.didox.uz"
)

// TokenLife is how long a user token is good for.
//
// ⚠️ **Shorter than the operator's 360 minutes on purpose.** A token that
// expires between our check and their check produces a 401 in the middle of an
// import, which reads to an owner as "the integration is broken". Twenty
// minutes of margin costs one extra login a day.
const TokenLife = 340 * time.Minute

// Client is one company's connection.
type Client struct {
	BaseURL      string
	PartnerToken string
	// The customer's session. Empty on the login call itself.
	UserToken string
	HTTP      *http.Client
}

// New builds a client for the given rails.
func New(sandbox bool, partnerToken, userToken string) *Client {
	base := ProdURL
	if sandbox {
		base = SandboxURL
	}
	return &Client{
		BaseURL:      base,
		PartnerToken: partnerToken,
		UserToken:    userToken,
		// ⚠️ A timeout, because this runs inside a request an owner is waiting
		// on: an operator having a slow morning must not hold a panel screen
		// open until the browser gives up with nothing to show.
		HTTP: &http.Client{Timeout: 30 * time.Second},
	}
}

// Error is what the operator said, kept whole.
//
// ⚠️ **The status code and their own text, together.** "Failed" is not an
// answer an accountant can act on; "422 User not registered" is — it means the
// СТИР typed into the settings has never opened a Didox account, which is a
// thing they can fix in five minutes and we cannot fix at all.
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("didox: %d", e.Status)
	}
	return fmt.Sprintf("didox: %d: %s", e.Status, e.Message)
}

// Unauthorized reports whether the session has to be established again.
//
// ⚠️ 401 **and** 403: the operator answers 403 to an expired user token on some
// endpoints, and retrying a document import forever against a session that
// died an hour ago is the shape of an integration that looks alive and does
// nothing.
func (e *Error) Unauthorized() bool {
	return e.Status == http.StatusUnauthorized || e.Status == http.StatusForbidden
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, "/")+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.PartnerToken != "" {
		req.Header.Set("Partner-Authorization", c.PartnerToken)
	}
	if c.UserToken != "" {
		req.Header.Set("user-key", c.UserToken)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode >= 400 {
		return &Error{Status: res.StatusCode, Message: message(raw)}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("didox: %s: %w", path, err)
	}
	return nil
}

// message pulls the operator's own sentence out of whatever shape they sent.
//
// ⚠️ **Three shapes, because they use three.** A JSON object with `message`, a
// bare JSON string, and plain text — and the one that gets dropped by a parser
// that only knows one shape is usually the interesting one ("Пользователь
// заблокирован"). Anything unrecognised is passed through as text rather than
// swallowed.
func message(raw []byte) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return ""
	}
	var obj struct {
		Message string `json:"message"`
		Error   string `json:"error"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		for _, s := range []string{obj.Message, obj.Error, obj.Reason} {
			if s != "" {
				return s
			}
		}
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil && s != "" {
		return s
	}
	if len(trimmed) > 300 {
		trimmed = trimmed[:300]
	}
	return trimmed
}

// ---- Session ----

// LoginByPassword exchanges the company's own password for a user token.
//
// ⚠️ **The password path rather than the key path, and that is the whole reason
// this integration is usable.** Signing needs E-IMZO on somebody's desk;
// *reading* the inbox does not, and a restaurant's storekeeper does not have
// the accountant's key in the store room. With a password the panel can show
// what has arrived; the signature stays where it belongs.
func (c *Client) LoginByPassword(ctx context.Context, tin, password string) (string, error) {
	var out struct {
		Token string `json:"token"`
		Data  struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	path := "/v1/auth/" + url.PathEscape(tin) + "/password/ru"
	if err := c.do(ctx, http.MethodPost, path, map[string]string{"password": password}, &out); err != nil {
		return "", err
	}
	// ⚠️ Both shapes accepted: the operator has answered with the token at the
	// top level and nested under `data` at different times, and a client that
	// knows one of them logs in successfully and then behaves as if it had not.
	if out.Token != "" {
		return out.Token, nil
	}
	if out.Data.Token != "" {
		return out.Data.Token, nil
	}
	return "", &Error{Status: http.StatusOK, Message: "token qaytmadi"}
}

// ---- Documents ----

// ListFilter is what the inbox asks for.
type ListFilter struct {
	// Incoming (false) or outgoing (true). ⚠️ Named for what it means rather
	// than for the operator's `owner=1/0`, which reads backwards at every call
	// site: `owner` is *us*, so `owner=0` is the post arriving.
	Outgoing bool
	Types    []string
	Statuses []int
	// Both inclusive, "yyyy-mm-dd", by the document's own date.
	From string
	To   string
	Page int
	// Capped at 100 by the operator.
	Limit int
}

// Row is one line of the operator's list.
//
// The field names are theirs; the tags are what makes this readable at the call
// site. ⚠️ Numbers arrive as numbers *and* as strings depending on the field,
// so the ones that have both are read through json.Number.
type Row struct {
	DocID      string      `json:"doc_id"`
	Number     string      `json:"name"`
	Date       string      `json:"doc_date"`
	Status     int         `json:"doc_status"`
	DocType    string      `json:"doctype"`
	Owner      int         `json:"owner"`
	PartnerTIN string      `json:"partnerTin"`
	Partner    string      `json:"partnerCompany"`
	Total      json.Number `json:"total_sum"`
	VatTotal   json.Number `json:"total_vat_sum"`
	TotalWith  json.Number `json:"total_delivery_sum_with_vat"`
	Delivery   json.Number `json:"total_delivery_sum"`
	HasVat     bool        `json:"has_vat"`
	HasMarks   int         `json:"has_marks"`
	Contract   string      `json:"contract_number"`
	ContractAt string      `json:"contract_date"`
	Created    string      `json:"created"`
	Updated    string      `json:"updated_date"`
}

// List returns one page of documents.
func (c *Client) List(ctx context.Context, f ListFilter) ([]Row, int, error) {
	q := url.Values{}
	page, limit := f.Page, f.Limit
	if page < 1 {
		page = 1
	}
	// ⚠️ `page` and `limit` are **required** by the operator; a request without
	// them is refused, not defaulted.
	if limit < 1 || limit > 100 {
		limit = 50
	}
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))
	if f.Outgoing {
		q.Set("owner", "1")
	} else {
		q.Set("owner", "0")
	}
	if len(f.Types) > 0 {
		q.Set("doctype", strings.Join(f.Types, ","))
	}
	if len(f.Statuses) > 0 {
		parts := make([]string, 0, len(f.Statuses))
		for _, s := range f.Statuses {
			parts = append(parts, strconv.Itoa(s))
		}
		q.Set("status", strings.Join(parts, ","))
	}
	if f.From != "" {
		q.Set("docDateFromCreated", f.From)
	}
	if f.To != "" {
		q.Set("docDateToCreated", f.To)
	}
	var out struct {
		Data  []Row `json:"data"`
		Total int   `json:"total"`
	}
	if err := c.do(ctx, http.MethodGet, "/v2/documents?"+q.Encode(), nil, &out); err != nil {
		return nil, 0, err
	}
	return out.Data, out.Total, nil
}

// Document is the full document: the operator's metadata and its own body.
type Document struct {
	// The body, exactly as it was signed. ⚠️ Kept raw as well as parsed: a
	// correction, a rent invoice or a marked-goods line carries fields this
	// build has never heard of, and the ones we drop are the ones the next
	// version needs.
	JSON json.RawMessage `json:"json"`
	Meta Row             `json:"document"`
}

// Get fetches one document with its lines.
func (c *Client) Get(ctx context.Context, id string, outgoing bool) (*Document, error) {
	owner := "0"
	if outgoing {
		owner = "1"
	}
	var out struct {
		Data Document `json:"data"`
	}
	path := "/v1/documents/" + url.PathEscape(id) + "?owner=" + owner
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// Create drafts a document of the given type and returns the operator's id.
//
// ⚠️ **A draft, and it says so.** The document is not sent and not filed until
// somebody signs it with their key at didox.uz; every screen that offers this
// has to say that, or an owner will believe an invoice went out this morning.
func (c *Client) Create(ctx context.Context, docType string, body any) (string, error) {
	var out struct {
		Data struct {
			ID     string `json:"id"`
			DocID  string `json:"doc_id"`
			Number string `json:"name"`
		} `json:"data"`
		ID    string `json:"id"`
		DocID string `json:"doc_id"`
	}
	path := "/v1/documents/" + url.PathEscape(docType) + "/create/ru"
	if err := c.do(ctx, http.MethodPost, path, body, &out); err != nil {
		return "", err
	}
	for _, id := range []string{out.Data.DocID, out.Data.ID, out.DocID, out.ID} {
		if id != "" {
			return id, nil
		}
	}
	return "", nil
}

// HTML is the operator's own printable form of a document.
//
// ⚠️ **Theirs rather than ours, and that is deliberate.** The paper an
// accountant files has to be the one the counterparty and the tax committee
// see; a form we drew ourselves would differ in some detail nobody noticed
// until it was refused at a desk.
func (c *Client) HTML(ctx context.Context, id, locale string) ([]byte, error) {
	if locale != "uz" {
		locale = "ru"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		strings.TrimRight(c.BaseURL, "/")+"/v1/documents/view/"+url.PathEscape(id)+"/html/"+locale, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Partner-Authorization", c.PartnerToken)
	req.Header.Set("user-key", c.UserToken)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode >= 400 {
		return nil, &Error{Status: res.StatusCode, Message: message(raw)}
	}
	return raw, nil
}
