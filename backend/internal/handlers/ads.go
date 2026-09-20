package handlers

// ---- The advertising section ----
//
// ⚠️ **The restaurant's money is spent at Meta, never by us.** The ad account is
// theirs, the card on it is theirs, and every so'm of ad spend is billed to them
// directly. What this section sells is the work around it: deciding what to
// advertise from figures this system already holds, writing the campaign, and
// saying afterwards what it brought back in orders rather than in clicks.
//
// ⚠️ **The state is five questions, not one.** "Not bought", "no platform
// behind this install", "no Meta account connected", "connected but nothing
// chosen" and "connected and ready" send an owner to five different places, and
// an earlier version of the assistant collapsed its failures into one empty
// screen that explained none of them. The connect checklist below is the same
// idea drawn out: every step is a fact read from Meta, which is the only kind
// of progress bar worth showing (docs/reklama-reja.md §6).
//
// See docs/reklama-reja.md for the phases and docs/vendor/meta-marketing.md for
// the API as actually read.

import (
	"net/http"
	"strings"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/meta"
	"restaurant-backend/internal/models"
)

// adsStep is one thing that has to be true before an advert can run.
type adsStep struct {
	Key  string `json:"key"`
	Done bool   `json:"done"`
	// What was chosen, when something was — the Page's name rather than its id,
	// because an id on a checklist tells the owner nothing about whether it is
	// the right one.
	Name string `json:"name,omitempty"`
}

// AdminAdsState is what the advertising screen needs before it draws anything.
func (h *Handler) AdminAdsState(w http.ResponseWriter, r *http.Request) {
	// ⚠️ Owner only, like the campaign button next door: this section connects
	// an account that can spend money, and a manager who could connect one
	// could spend from it.
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	ctx := r.Context()

	// ⚠️ **Read from the mirrored grant, never worked out here.** What a
	// restaurant bought is the console's answer (models/subscription.go); a
	// second opinion in this file is the one that disagrees on the day somebody
	// upgrades.
	entitled := h.subscription(ctx).Has(models.ModAds)

	// A self-hosted restaurant with no platform behind it has no assistant and
	// no Meta app to connect through. Answered as a fact rather than an error:
	// the screen then says so instead of drawing a failure.
	linked := h.Cfg.ControlURL != "" && h.Cfg.ControlToken != ""

	s := h.adsSettings(ctx)
	hasToken := strings.TrimSpace(s.Token) != ""

	steps := []adsStep{
		{Key: "token", Done: hasToken},
		{Key: "business", Done: s.BusinessID != "", Name: s.BusinessName},
		{Key: "account", Done: s.AdAccountID != "", Name: s.AdAccountName},
		{Key: "page", Done: s.PageID != "", Name: s.PageName},
		// ⚠️ **The pixel is a step, not a requirement.** Without one a campaign
		// can still run for traffic; what is lost is the answer to "how many
		// orders did it bring", which is the thing this section is sold for. So
		// it is shown as unfinished rather than as a blocker.
		{Key: "pixel", Done: s.PixelID != "", Name: s.PixelName},
	}
	ready := hasToken && s.AdAccountID != "" && s.PageID != ""

	httpx.JSON(w, http.StatusOK, map[string]any{
		"entitled":  entitled,
		"on":        linked,
		"connected": hasToken,
		"ready":     ready,
		"steps":     steps,
		// ⚠️ **The token is not here and there is no field for it.** The same
		// rule as the payment keys: this answer is built for a browser, and one
		// forgotten tag is a credential that can spend money leaving the server.
		"settings": map[string]any{
			"hasToken":       hasToken,
			"businessId":     s.BusinessID,
			"businessName":   s.BusinessName,
			"adAccountId":    s.AdAccountID,
			"adAccountName":  s.AdAccountName,
			"currency":       s.Currency,
			"minDailyBudget": s.MinDailyBudget,
			// ⚠️ **How many of Meta's units make one of that currency, sent
			// rather than known.** The panel shows budgets the way the owner's
			// card will be charged, and a second copy of the currency table in
			// the browser is a copy that drifts — on the one screen where
			// drifting means spending a hundred times the intended amount.
			"unit":        meta.MinorUnits(s.Currency),
			"pageId":      s.PageID,
			"pageName":    s.PageName,
			"instagramId": s.InstagramID,
			"pixelId":     s.PixelID,
			"pixelName":   s.PixelName,
			// ⚠️ The date travels beside the verdict, because a stored "ok"
			// ages the moment the clock passes it and the panel has to be able
			// to say "as of Tuesday" rather than "fine".
			"status":      s.Status,
			"statusNote":  s.StatusNote,
			"lastCheckAt": s.LastCheckAt,
			"connectedAt": s.ConnectedAt,
			"rules":       s.Rules,
		},
	})
}
