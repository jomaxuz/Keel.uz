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
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"keel-control/internal/billing"
	"keel-control/internal/httpx"
	"keel-control/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

// metaVersion is the Graph version this exchange goes to.
//
// ⚠️ **Must match the tenant's `meta.Version`.** A code minted against one
// version and exchanged on another works today and is exactly the sort of
// mismatch that surfaces as an unexplained failure the week Meta retires one of
// them.
const metaVersion = "v26.0"

// AdsApp tells a tenant which app to open Meta's login dialog for, and books
// the one-time state the returning browser will carry.
//
// ⚠️ **No secret in the answer, and there is no field for one.** What a browser
// needs is the app id and the configuration id, both of which appear in the
// dialog's own URL; the secret is used two calls later, here, and never leaves.
//
// ⚠️ **The redirect address is ours, not the restaurant's.** Meta only returns
// to URIs whitelisted in the app, and every customer has their own domain —
// whitelisting each one would mean editing Meta's settings per customer, and
// the one nobody remembered to add is a connect button that fails with a
// message about redirect URIs. So the dialog always comes back to this service
// and `AdsRedirect` forwards the code to whoever started it.
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
	if !h.metaConfigured() {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"entitled": true, "configured": false,
		})
		return
	}

	var req struct {
		ReturnTo string `json:"returnTo"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req)

	// ⚠️ **Checked against this tenant's own hostnames, never trusted.** The
	// caller is a customer's container, and the value decides where a browser
	// holding an authorization code is sent next. A container that had been
	// tampered with could otherwise name somebody else's address and be handed
	// a key to this restaurant's ad account.
	back, err := h.adsReturnTo(t, req.ReturnTo)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	state, err := h.newAdsState(r.Context(), t.Slug, back)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "state saqlanmadi")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"entitled":    true,
		"configured":  true,
		"appId":       strings.TrimSpace(h.Cfg.MetaAppID),
		"configId":    strings.TrimSpace(h.Cfg.MetaConfigID),
		"version":     metaVersion,
		"redirectUri": strings.TrimSpace(h.Cfg.MetaRedirectURI),
		"state":       state,
	})
}

// adsReturnTo is where this restaurant's browser may be sent back to.
func (h *Handler) adsReturnTo(t *models.Tenant, given string) (string, error) {
	raw := strings.TrimSpace(given)
	if raw == "" {
		return "", errMeta("qaytish manzili ko'rsatilmagan")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errMeta("qaytish manzili noto'g'ri")
	}
	host := strings.ToLower(u.Hostname())
	for _, d := range t.Domains {
		if strings.ToLower(strings.TrimSpace(d)) == host {
			// Only the path travels on: a query or a fragment from the caller
			// would be appended to an address we are about to put a code on.
			return u.Scheme + "://" + u.Host + u.Path, nil
		}
	}
	return "", errMeta("qaytish manzili bu restoranning domeni emas")
}

// newAdsState books one pending login.
func (h *Handler) newAdsState(ctx context.Context, slug, back string) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	state := hex.EncodeToString(buf)
	_, err := h.Store.AdsStates.InsertOne(ctx, bson.M{
		"state": state, "slug": slug, "returnTo": back, "at": time.Now(),
	})
	return state, err
}

// AdsRedirect is where Meta sends the browser, for every restaurant.
//
// ⚠️ **It forwards and does nothing else.** The code is exchanged by the
// tenant's own server through `AdsToken`, because that is where the token has
// to end up; this endpoint only knows which restaurant asked, and it knows it
// from a row we wrote rather than from anything in the request.
func (h *Handler) AdsRedirect(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	state := strings.TrimSpace(q.Get("state"))
	var row struct {
		ReturnTo string `bson:"returnTo"`
	}
	err := h.Store.AdsStates.FindOneAndDelete(r.Context(),
		bson.M{"state": state}).Decode(&row)
	if err != nil || row.ReturnTo == "" {
		// ⚠️ **Not redirected anywhere.** An unknown state is either an expired
		// dialog or somebody trying the address by hand, and the one thing this
		// endpoint must never do is forward a code to an address it cannot
		// account for.
		http.Error(w, "Bu havola eskirgan. Panelda «Meta bilan ulash» ni qayta bosing.",
			http.StatusBadRequest)
		return
	}
	back, err := url.Parse(row.ReturnTo)
	if err != nil {
		http.Error(w, "qaytish manzili o'qilmadi", http.StatusInternalServerError)
		return
	}
	out := url.Values{}
	// Meta sends either a code or a refusal. Both travel on: the panel says
	// "you cancelled" rather than sitting on a spinner.
	for _, k := range []string{"code", "error", "error_reason", "error_description"} {
		if v := strings.TrimSpace(q.Get(k)); v != "" {
			out.Set(k, v)
		}
	}
	back.RawQuery = out.Encode()
	http.Redirect(w, r, back.String(), http.StatusFound)
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
	Code string `json:"code"`
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

	// ⚠️ **Our own redirect address, never the caller's.** Meta compares the
	// value byte for byte with the one the dialog was opened with, and the
	// dialog was opened with ours — a tenant that sent a different one would
	// get "redirect_uri isn't the same" and no way to tell why.
	tok, err := h.exchangeMetaCode(r.Context(), req.Code,
		strings.TrimSpace(h.Cfg.MetaRedirectURI))
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
