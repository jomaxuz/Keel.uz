package handlers

// ---- Connecting the restaurant's own Meta ad account ----
//
// ⚠️ **Five facts, not one switch.** A working connection is a business
// portfolio, an ad account inside it, a Page the advert is published as, the
// Instagram account beside that Page, and a pixel the orders are reported to.
// Meta holds all five, the owner can rarely name any of them, and a campaign
// created with one of them missing fails at a different step each time. So the
// screen lists what the token can actually see and the owner points at it —
// and the state endpoint says which of the five is still missing rather than
// "not connected".
//
// ⚠️ **The token is the restaurant's and lives only in their database.** The
// app secret is the platform's and lives only in the console (control's
// adsmeta.go); this server holds the key to one ad account and talks to Meta
// with it directly, because the quota Meta counts is that account's own.
//
// ⚠️ **A revoked token does not stop the adverts.** The owner removing our
// access at Meta leaves every campaign running on their card and leaves us
// blind — so `status` is re-read against Meta rather than remembered, and the
// panel is told the difference between "connected" and "we last had an answer
// on Tuesday".

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/meta"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	errAdsNoToken = errors.New("Meta akkaunti ulanmagan")
	errAdsNoAct   = errors.New("reklama akkaunti tanlanmagan")
	errAdsNoPage  = errors.New("Facebook sahifasi tanlanmagan")
)

// adsSettings loads the connection. A missing document is an install whose
// owner has never opened the page, which is not an error.
func (h *Handler) adsSettings(ctx context.Context) *models.AdsSettings {
	var s models.AdsSettings
	if err := h.Store.AdsSettings.FindOne(ctx, bson.M{}).Decode(&s); err != nil {
		return &models.AdsSettings{}
	}
	return &s
}

// saveAds writes part of the connection.
//
// ⚠️ **`$set` of named fields, never the whole document.** The same rule the
// restaurant profile follows: a full replace made by a screen that edits one
// field is how the token, or the pixel, quietly disappears — and the failure
// only shows up the next time a campaign is created.
func (h *Handler) saveAds(ctx context.Context, set bson.M) error {
	_, err := h.Store.AdsSettings.UpdateOne(ctx, bson.M{},
		bson.M{"$set": set}, options.Update().SetUpsert(true))
	return err
}

// adsClient builds a Meta client for this restaurant.
func (h *Handler) adsClient(ctx context.Context) (*meta.Client, *models.AdsSettings, error) {
	s := h.adsSettings(ctx)
	if strings.TrimSpace(s.Token) == "" {
		return nil, s, errAdsNoToken
	}
	return meta.New(s.Token), s, nil
}

// noteMetaError records what Meta last said, so the panel can show a
// connection that has died without waiting for somebody to press a button.
//
// ⚠️ **Only a revoked token changes the status.** A rate limit or a five-second
// outage is not a broken connection, and marking one as such would put a
// "reconnect your account" banner in front of an owner whose account is fine —
// which is how a working feature teaches people to ignore its warnings.
func (h *Handler) noteMetaError(ctx context.Context, err error) {
	e := meta.AsError(err)
	if e == nil {
		return
	}
	set := bson.M{"lastCheckAt": time.Now(), "statusNote": e.Human()}
	if e.Revoked() {
		set["status"] = "revoked"
	}
	_ = h.saveAds(ctx, set)
}

// ---- The door: which app, and the code it hands back ----

// AdminAdsApp tells the browser which Meta app to open the login dialog for.
//
// ⚠️ **Proxied through this server rather than read from the bundle.** A
// `NEXT_PUBLIC_*` value is sealed into the build (CLAUDE.md §10), and this one
// differs per platform and changes when the app is re-reviewed — which would
// mean rebuilding every restaurant's frontend to change a number that is not
// even secret.
func (h *Handler) AdminAdsApp(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	// Where the owner should land when Meta is done with them.
	//
	// ⚠️ **Sent by the browser, checked by the platform.** This panel can be
	// opened on any hostname the restaurant has connected, and this server has
	// no reliable way to know which one is in the address bar. The console
	// compares it against the domains it routes to this tenant, which is the
	// list that decides the answer anyway — so a wrong value is refused there
	// rather than trusted here.
	back := strings.TrimSpace(r.URL.Query().Get("returnTo"))
	if back == "" && len(h.Cfg.CORSOrigins) > 0 {
		back = strings.TrimSuffix(strings.TrimSpace(h.Cfg.CORSOrigins[0]), "/") +
			"/admin/ads"
	}
	out, err := h.callControlPath(r.Context(), "/internal/ads-app", map[string]any{
		"returnTo": back,
	})
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

type adsConnectRequest struct {
	Code string `json:"code"`
}

// AdminAdsConnect turns the code Meta gave the browser into a stored token.
func (h *Handler) AdminAdsConnect(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req adsConnectRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req) != nil ||
		strings.TrimSpace(req.Code) == "" {
		httpx.Error(w, http.StatusBadRequest, "kod kelmadi")
		return
	}
	ctx := r.Context()
	// ⚠️ No redirect address travels with the code: the dialog was opened with
	// the platform's own, and the exchange has to use that exact string. A
	// value from here would be a second copy of it, wrong the first time
	// anybody edits one of the two.
	out, err := h.callControlPath(ctx, "/internal/ads-token", map[string]any{
		"code": strings.TrimSpace(req.Code),
	})
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	if msg, _ := out["error"].(string); msg != "" {
		httpx.Error(w, http.StatusBadRequest, msg)
		return
	}
	token, _ := out["token"].(string)
	if strings.TrimSpace(token) == "" {
		httpx.Error(w, http.StatusBadGateway, "Meta tokenni bermadi")
		return
	}

	// ⚠️ **Tried before it is stored.** A token that cannot read a single
	// business portfolio is a connection that will fail on the next screen
	// instead, with nothing to say why — and the owner has by then closed the
	// dialog that could have been reopened.
	client := meta.New(token)
	// ⚠️ A cheap call that proves the token works. Its *result* is not used:
	// a business integration system user has no "my businesses" to list, and
	// an empty answer here is the normal one — see meta.AdAccount.Business.
	_, err = client.AdAccounts(ctx, "")
	if err != nil {
		if e := meta.AsError(err); e != nil {
			httpx.Error(w, http.StatusBadRequest, e.Human())
			return
		}
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}

	set := bson.M{
		"token":       token,
		"connectedAt": time.Now(),
		"lastCheckAt": time.Now(),
		"status":      "ok",
		"statusNote":  "",
	}
	if claims := middleware.ClaimsFrom(ctx); claims != nil {
		set["connectedBy"] = claims.UserID
	}
	if err := h.saveAds(ctx, set); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "saqlanmadi")
		return
	}
	h.logAction(r, ActAdsConnect, "ads", "", "Meta", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// AdminAdsDisconnect forgets the token.
//
// ⚠️ **The campaigns are not stopped, and the screen says so.** Deleting our
// key does not reach into somebody's ad account; an advert that is running
// keeps running and keeps costing money. Pretending otherwise would be the
// worst possible lie to tell at exactly the moment an owner is trying to make
// spending stop.
func (h *Handler) AdminAdsDisconnect(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	_, err := h.Store.AdsSettings.UpdateOne(r.Context(), bson.M{}, bson.M{
		"$set":   bson.M{"status": "", "statusNote": "", "lastCheckAt": time.Now()},
		"$unset": bson.M{"token": "", "connectedAt": "", "connectedBy": ""},
	}, options.Update().SetUpsert(true))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "saqlanmadi")
		return
	}
	h.logAction(r, ActAdsDisconnect, "ads", "", "Meta", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- What the token can see ----

// AdminAdsAssets lists the portfolios, accounts, Pages and pixels to pick from.
func (h *Handler) AdminAdsAssets(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	ctx := r.Context()
	client, s, err := h.adsClient(ctx)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	accounts, err := client.AdAccounts(ctx, s.BusinessID)
	if err != nil {
		h.noteMetaError(ctx, err)
		if e := meta.AsError(err); e != nil {
			httpx.Error(w, http.StatusBadGateway, e.Human())
			return
		}
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}

	// ⚠️ **A list that could not be fetched says so, instead of arriving
	// empty.** Pages and pixels are asked for with the same token and either
	// can be refused — Meta answers `(#210) A page access token is required`
	// for one of them on a system user token — and an empty dropdown over a
	// working connection is the screen an owner reads as "this is broken".
	// The reason travels beside the list, in a sentence, and the account
	// remains choosable either way.
	pages, pagesErr := client.Pages(ctx)
	var pixels []meta.Pixel
	var pixelsErr error
	if s.AdAccountID != "" {
		pixels, pixelsErr = client.Pixels(ctx, s.AdAccountID)
	}

	_ = h.saveAds(ctx, bson.M{"lastCheckAt": time.Now(), "status": "ok", "statusNote": ""})
	httpx.JSON(w, http.StatusOK, map[string]any{
		// ⚠️ Empty slices rather than nil: a nil slice reaches the browser as
		// `null` and `null.map` is the crash this codebase has shipped twice
		// (CLAUDE.md §10).
		"accounts":  nonNil(accounts),
		"pages":     nonNil(pages),
		"pixels":    nonNil(pixels),
		"pagesNote": metaNote(pagesErr),
		"pixelNote": metaNote(pixelsErr),
	})
}

// metaNote is Meta's refusal in a sentence, or nothing at all.
func metaNote(err error) string {
	if err == nil {
		return ""
	}
	if e := meta.AsError(err); e != nil {
		return e.Human()
	}
	return err.Error()
}

type adsChooseRequest struct {
	BusinessID  string `json:"businessId"`
	AdAccountID string `json:"adAccountId"`
	PageID      string `json:"pageId"`
	InstagramID string `json:"instagramId"`
	PixelID     string `json:"pixelId"`
}

// AdminAdsChoose records which of Meta's objects this restaurant advertises
// from.
//
// ⚠️ **The account's currency and Meta's own budget floor are re-read here,
// from Meta, and stored beside the choice.** Everything about money in this
// section depends on them: Meta takes budgets in the account currency's minor
// units, the account is almost never in so'm, and a budget built on an assumed
// currency is a campaign that spends a hundred times what the owner typed.
func (h *Handler) AdminAdsChoose(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req adsChooseRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req) != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	ctx := r.Context()
	client, _, err := h.adsClient(ctx)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	set := bson.M{}
	if id := strings.TrimSpace(req.BusinessID); id != "" {
		set["businessId"] = id
	}
	if id := strings.TrimSpace(req.AdAccountID); id != "" {
		acc, err := client.Account(ctx, id)
		if err != nil {
			h.noteMetaError(ctx, err)
			httpx.Error(w, http.StatusBadGateway, err.Error())
			return
		}
		if !acc.Live() {
			// Said now rather than at the first campaign: an account Meta has
			// disabled creates objects happily and runs none of them.
			httpx.Error(w, http.StatusBadRequest,
				"bu reklama akkaunti Meta tomonidan faol emas")
			return
		}
		set["adAccountId"] = acc.ID
		set["adAccountName"] = acc.Name
		set["currency"] = acc.Currency
		set["minDailyBudget"] = acc.MinDailyBudget
		// The portfolio comes with the account rather than from a list the
		// owner picks from — see meta.AdAccount.Business.
		if acc.Business.ID != "" {
			set["businessId"] = acc.Business.ID
			set["businessName"] = acc.Business.Name
		}
	}
	if id := strings.TrimSpace(req.PageID); id != "" {
		set["pageId"] = id
		set["instagramId"] = strings.TrimSpace(req.InstagramID)
		if pages, err := client.Pages(ctx); err == nil {
			for _, p := range pages {
				if p.ID == id {
					set["pageName"] = p.Name
					// ⚠️ Taken from Meta rather than from the request: the
					// Instagram account is a property of the Page, and a
					// browser that sent the wrong one would publish the advert
					// under somebody else's profile.
					set["instagramId"] = p.Instagram.ID
				}
			}
		}
	}
	if id := strings.TrimSpace(req.PixelID); id != "" {
		set["pixelId"] = id
		if act, _ := set["adAccountId"].(string); act != "" {
			if pixels, err := client.Pixels(ctx, act); err == nil {
				for _, p := range pixels {
					if p.ID == id {
						set["pixelName"] = p.Name
					}
				}
			}
		}
	}
	if len(set) == 0 {
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	if err := h.saveAds(ctx, set); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "saqlanmadi")
		return
	}
	h.logAction(r, ActAdsChoose, "ads", "", "Meta", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

type adsRulesRequest struct {
	Rules models.AdRules `json:"rules"`
}

// AdminAdsRules stores the standing instruction the owner leaves behind.
func (h *Handler) AdminAdsRules(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req adsRulesRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req) != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	// ⚠️ Negatives clamped rather than rejected: a minus in a ceiling would
	// read as "never allowed" in one comparison and "always allowed" in
	// another, and the two comparisons are in different files.
	rules := req.Rules
	for _, p := range []*int{&rules.MaxDailyMinor, &rules.NoResultMinor, &rules.MaxCostMinor} {
		if *p < 0 {
			*p = 0
		}
	}
	if err := h.saveAds(r.Context(), bson.M{"rules": rules}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "saqlanmadi")
		return
	}
	h.logAction(r, ActAdsRules, "ads", "", "Meta", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
