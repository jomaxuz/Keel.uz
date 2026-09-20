package handlers

// ---- The Meta app: ours, and the one thing a tenant cannot hold ----
//
// Connecting a restaurant's ad account is an OAuth exchange, and the half of it
// that must stay secret is the app secret. ⚠️ **It lives here for the same
// reason the AI key does**: a value copied into every tenant container is a
// value that has to be rotated in every tenant container, and a container is
// the customer's own machine. So the tenant server does the talking to Meta
// with the *restaurant's* token, and comes here exactly twice — once to learn
// which app to open the login dialog for, once to turn the returned code into
// that token.
//
// ⚠️ **What comes back is the restaurant's credential, and we do not keep it.**
// The token is minted against their business portfolio and their ad account; it
// is written to their own database and never to ours. A platform-side copy
// would be a list of keys to a hundred advertising accounts sitting in one
// place, for no feature that needs it.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"keel-control/internal/billing"
	"keel-control/internal/httpx"
)

// metaVersion is the Graph version this exchange goes to.
//
// ⚠️ **Must match the tenant's `meta.Version`.** A code minted against one
// version and exchanged on another works today and is exactly the sort of
// mismatch that surfaces as an unexplained failure the week Meta retires one of
// them.
const metaVersion = "v26.0"

// AdsApp tells a tenant which app to open Meta's login dialog for.
//
// ⚠️ **No secret in the answer, and there is no field for one.** What a browser
// needs is the app id and the configuration id, both of which appear in the
// dialog's own URL; the secret is used two calls later, here, and never leaves.
func (h *Handler) AdsApp(w http.ResponseWriter, r *http.Request) {
	t, err := h.tenantFromLink(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	// The same gate as the plan: connecting an ad account is the add-on, not a
	// free door beside it.
	_, addons, _ := h.grantOf(r.Context(), t)
	if !billing.AdsEntitled(addons) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"entitled": false, "monthly": billing.AdsMonthly,
		})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"entitled":   true,
		"configured": h.metaConfigured(),
		"appId":      strings.TrimSpace(h.Cfg.MetaAppID),
		"configId":   strings.TrimSpace(h.Cfg.MetaConfigID),
		"version":    metaVersion,
	})
}

// metaConfigured is whether this platform has an app at all.
//
// ⚠️ **Answered as a fact rather than an error.** Until Meta has reviewed the
// app there is nothing to connect to, and the restaurant's screen should say so
// plainly instead of drawing a failed button — the same shape the advertising
// section uses for every other absence.
func (h *Handler) metaConfigured() bool {
	return strings.TrimSpace(h.Cfg.MetaAppID) != "" &&
		strings.TrimSpace(h.Cfg.MetaAppSecret) != "" &&
		strings.TrimSpace(h.Cfg.MetaConfigID) != ""
}

type adsTokenRequest struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirectUri"`
}

// AdsToken exchanges the code Meta handed the browser for a token.
//
// ⚠️ **Server side, always.** Meta's own documentation is explicit that the
// code-for-token call carries the app secret and must never be made from a
// page; `response_type=code` exists precisely so the browser never sees a
// credential.
func (h *Handler) AdsToken(w http.ResponseWriter, r *http.Request) {
	t, err := h.tenantFromLink(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	_, addons, _ := h.grantOf(r.Context(), t)
	if !billing.AdsEntitled(addons) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"entitled": false, "monthly": billing.AdsMonthly,
		})
		return
	}
	if !h.metaConfigured() {
		httpx.JSON(w, http.StatusOK, map[string]any{"off": true})
		return
	}
	var req adsTokenRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req) != nil ||
		strings.TrimSpace(req.Code) == "" {
		httpx.Error(w, http.StatusBadRequest, "code required")
		return
	}

	tok, err := h.exchangeMetaCode(r.Context(), req.Code, req.RedirectURI)
	if err != nil {
		// Meta's sentence, not ours: "This authorization code has expired" and
		// "redirect_uri isn't an absolute URI" are two different mistakes made
		// by two different people, and one paraphrase would hide both.
		httpx.JSON(w, http.StatusOK, map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"token":     tok.AccessToken,
		"expiresIn": tok.ExpiresIn,
		"type":      tok.TokenType,
	})
}

type metaToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	// ⚠️ **Zero is the good answer.** A business integration system user token
	// has no expiry, and the sixty-day number is what a *user* token comes back
	// with — which is automation that stops working two months later while the
	// campaigns keep spending. The tenant records this and the panel says which
	// kind it got.
	ExpiresIn int `json:"expires_in"`
}

func (h *Handler) exchangeMetaCode(ctx context.Context, code, redirect string) (metaToken, error) {
	q := url.Values{
		"client_id":     {strings.TrimSpace(h.Cfg.MetaAppID)},
		"client_secret": {strings.TrimSpace(h.Cfg.MetaAppSecret)},
		"code":          {strings.TrimSpace(code)},
	}
	if r := strings.TrimSpace(redirect); r != "" {
		// Meta compares this with the one the dialog was opened with, byte for
		// byte — a trailing slash is a different URI.
		q.Set("redirect_uri", r)
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://graph.facebook.com/"+metaVersion+"/oauth/access_token?"+q.Encode(), nil)
	if err != nil {
		return metaToken{}, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return metaToken{}, err
	}
	defer func() { _ = res.Body.Close() }()

	var body struct {
		metaToken
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(nil, res.Body, 1<<20)).Decode(&body)
	if body.AccessToken == "" {
		msg := strings.TrimSpace(body.Error.Message)
		if msg == "" {
			msg = "Meta tokenni bermadi: " + res.Status
		}
		return metaToken{}, errMeta(msg)
	}
	return body.metaToken, nil
}

type errMeta string

func (e errMeta) Error() string { return string(e) }
