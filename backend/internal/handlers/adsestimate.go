package handlers

// ---- What Meta thinks a budget will buy, before it is spent ----
//
// ⚠️ **Meta's forecast, not ours, and the screen never drops the attribution.**
// The planner is forbidden from predicting results, and that rule stands: a
// number we invented and an owner acted on is the one failure this section
// cannot survive. This is a different thing — the same estimate Ads Manager
// shows them, asked of the system that will actually deliver the adverts.
//
// ⚠️ **Asked at the moment the budget is typed, not stored.** An estimate is
// about an audience that changes daily; a cached one would be a stale promise
// sitting next to a live budget field.

import (
	"encoding/json"
	"math"
	"net/http"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/meta"
)

type adsEstimateRequest struct {
	RadiusKm float64 `json:"radiusKm"`
	// In the ad account's currency, as the owner typed it.
	Daily float64 `json:"daily"`
}

// AdminAdsEstimate answers "what does this buy".
func (h *Handler) AdminAdsEstimate(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req adsEstimateRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req) != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	ctx := r.Context()
	branch, err := h.adsBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	client, s, err := h.adsClient(ctx)
	if err != nil || s.AdAccountID == "" {
		httpx.Error(w, http.StatusBadRequest, errAdsNoAct.Error())
		return
	}
	if branch.Address.Lat == 0 && branch.Address.Lng == 0 {
		httpx.Error(w, http.StatusBadRequest,
			"filial xaritada belgilanmagan — reklama hududi shundan o'lchanadi")
		return
	}

	radius := req.RadiusKm
	if radius <= 0 {
		radius = float64(branch.Delivery.MaxKm)
	}
	radius = math.Min(math.Max(radius, adsMinRadiusKm), adsMaxRadiusKm)

	// The same pair the campaign would be created with, so the estimate is for
	// the campaign that will actually run rather than for a simpler one.
	goal := "LINK_CLICKS"
	if s.PixelID != "" {
		goal = "OFFSITE_CONVERSIONS"
	}
	est, err := client.DeliveryEstimate(ctx, s.AdAccountID, goal, meta.Targeting{
		Lat: branch.Address.Lat, Lng: branch.Address.Lng, RadiusKm: radius,
	}, pixelFor(goal, s.PixelID), eventFor(goal))
	if err != nil {
		h.noteMetaError(ctx, err)
		if e := meta.AsError(err); e != nil {
			httpx.Error(w, http.StatusBadGateway, e.Human())
			return
		}
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}

	unit := meta.MinorUnits(s.Currency)
	out := map[string]any{
		"ready":    est.Ready,
		"audience": est.MAU,
		"daily":    est.DAU,
		"currency": s.Currency,
		"radiusKm": radius,
		"goal":     goal,
	}
	// ⚠️ **Only when Meta answered for a budget.** `estimate_ready: false` is
	// the ordinary state of a fresh account and of an audience Meta thinks too
	// small; drawing its zeroes as a forecast would be predicting nothing and
	// calling it a prediction.
	if p, ok := est.At(int(math.Round(req.Daily * float64(unit)))); ok {
		out["reach"] = int64(p.Reach)
		out["impressions"] = int64(p.Impressions)
		// The outcome Meta optimises for — clicks, or orders when a pixel is
		// reporting them. Named by `goal` above so the panel can say which.
		out["results"] = int64(p.Actions)
		out["atSpend"] = p.Spend / float64(unit)
	}
	httpx.JSON(w, http.StatusOK, out)
}
