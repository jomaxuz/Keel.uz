package handlers

// ---- What the advertising actually cost, and what came back ----
//
// ⚠️ **A daily snapshot, never a call per page view.** On Meta's Limited access
// tier an ad account gets a few hundred insights calls an hour for everything,
// and a report screen that asked live would spend a restaurant's whole quota on
// somebody refreshing a tab. So the figures are pulled once a day into
// `ad_daily` and every screen reads from there.
//
// ⚠️ **The pulled rows are history, not a cache.** Meta rewrites insights as
// attribution windows close, so "what we saw on Tuesday morning" and "what Meta
// says about Tuesday now" are different facts — and the rules that paused a
// campaign were acting on the first one. The unique index on (campaign, day)
// means a re-sync corrects a day rather than adding a second one to it.
//
// ⚠️ **Spend is in the ad account's currency and is never converted.** That
// account is almost certainly billed in dollars while the restaurant counts in
// so'm, and we do not know the rate the owner's bank used. A converted number
// standing beside real ones is the most believable kind of wrong.

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/meta"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// How often the sync runs, and how far back it re-reads.
//
// ⚠️ **Six hours rather than daily.** The rules engine downstream can only stop
// a campaign as fast as the figures arrive, and "we noticed at midnight that it
// burned through the day" is a saving nobody feels. ⚠️ **Three days back**
// because Meta keeps adjusting a day for about that long — reading only
// yesterday would leave the first day of every campaign permanently wrong.
const (
	adsSyncEvery = 6 * time.Hour
	adsSyncBack  = 3 * 24 * time.Hour
)

// StartAdsSync keeps the advertising figures fresh.
func (h *Handler) StartAdsSync(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(adsSyncEvery)
		defer ticker.Stop()
		for {
			h.syncAdsOnce(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// syncAdsOnce reads every campaign that is still worth reading.
func (h *Handler) syncAdsOnce(ctx context.Context) {
	client, s, err := h.adsClient(ctx)
	if err != nil {
		// Not connected is the ordinary state of most installs, and it is not
		// a thing to log every six hours.
		return
	}
	// ⚠️ Stopped campaigns are read for a few more days rather than dropped:
	// Meta is still adjusting their last days, and a campaign whose final
	// numbers are wrong is the one an owner judges the whole section by.
	cutoff := time.Now().Add(-adsSyncBack)
	cur, err := h.Store.AdsCampaigns.Find(ctx, bson.M{
		"metaCampaignId": bson.M{"$ne": ""},
		"$or": []bson.M{
			{"status": bson.M{"$in": []string{
				models.AdCampaignActive, models.AdCampaignPaused,
			}}},
			{"stoppedAt": bson.M{"$gte": cutoff}},
		},
	}, options.Find().SetLimit(200))
	if err != nil {
		log.Printf("ads sync: %v", err)
		return
	}
	var rows []models.AdCampaign
	if err := cur.All(ctx, &rows); err != nil {
		log.Printf("ads sync: %v", err)
		return
	}
	for _, row := range rows {
		if err := h.syncCampaign(ctx, client, s, row); err != nil {
			// ⚠️ One campaign's failure stops that campaign, not the sweep: a
			// rate limit hit on the first of twelve would otherwise leave the
			// other eleven unread until the next tick.
			log.Printf("ads sync %s: %v", row.MetaCampaignID, err)
			if e := meta.AsError(err); e != nil && (e.Revoked() || e.RateLimited()) {
				h.noteMetaError(ctx, err)
				return
			}
		}
	}
}

// syncCampaign pulls one campaign's days and rewrites its totals.
func (h *Handler) syncCampaign(
	ctx context.Context, client *meta.Client, s *models.AdsSettings, row models.AdCampaign,
) error {
	since := time.Now().Add(-adsSyncBack)
	if row.CreatedAt.After(since) {
		since = row.CreatedAt
	}
	days, err := client.Insights(ctx, row.MetaCampaignID, since, time.Now())
	if err != nil {
		return err
	}
	now := time.Now()
	for _, d := range days {
		_, err := h.Store.AdsDaily.UpdateOne(ctx,
			bson.M{"metaCampaignId": row.MetaCampaignID, "day": d.Date},
			bson.M{"$set": bson.M{
				"campaignId":     row.ID,
				"metaCampaignId": row.MetaCampaignID,
				"day":            d.Date,
				"spend":          d.Spend,
				"impressions":    d.Impressions,
				"clicks":         d.Clicks,
				"purchases":      d.Purchases,
				"revenue":        d.Revenue,
				"currency":       s.Currency,
				"takenAt":        now,
			}},
			options.Update().SetUpsert(true))
		if err != nil {
			return err
		}
	}

	// ⚠️ **Totals rebuilt from the days, never added to.** Meta revises a day
	// downwards as often as upwards, and a running total that only ever grew
	// would drift away from the rows it claims to summarise — on the one screen
	// whose job is to be trusted about money.
	totals := h.adsTotals(ctx, bson.M{"metaCampaignId": row.MetaCampaignID})
	set := bson.M{
		"spend":      totals.Spend,
		"purchases":  totals.Purchases,
		"revenue":    totals.Revenue,
		"lastSyncAt": now,
	}

	// What Meta made of the advert itself. ⚠️ The rejection reason is fetched
	// with it: "rejected" alone sends the owner to Ads Manager to read the one
	// sentence we could have shown them.
	if row.AdID != "" {
		if rev, err := client.AdReview(ctx, row.AdID); err == nil {
			set["metaStatus"] = rev.EffectiveStatus
			note := ""
			for _, is := range rev.IssuesInfo {
				note = strings.TrimSpace(is.ErrorSummary + " " + is.ErrorMessage)
				break
			}
			set["reviewNote"] = note
		}
	}
	_, err = h.Store.AdsCampaigns.UpdateByID(ctx, row.ID, bson.M{"$set": set})
	return err
}

// adsSums is one window's figures.
type adsSums struct {
	Spend       float64 `json:"spend"`
	Impressions int     `json:"impressions"`
	Clicks      int     `json:"clicks"`
	Purchases   int     `json:"purchases"`
	Revenue     float64 `json:"revenue"`
}

// adsTotals adds up the daily rows that match.
func (h *Handler) adsTotals(ctx context.Context, match bson.M) adsSums {
	cur, err := h.Store.AdsDaily.Aggregate(ctx, []bson.M{
		{"$match": match},
		{"$group": bson.M{
			"_id":         nil,
			"spend":       bson.M{"$sum": "$spend"},
			"impressions": bson.M{"$sum": "$impressions"},
			"clicks":      bson.M{"$sum": "$clicks"},
			"purchases":   bson.M{"$sum": "$purchases"},
			"revenue":     bson.M{"$sum": "$revenue"},
		}},
	})
	if err != nil {
		return adsSums{}
	}
	var rows []adsSums
	if err := cur.All(ctx, &rows); err != nil || len(rows) == 0 {
		return adsSums{}
	}
	return rows[0]
}

// AdminAdsReport is what the money bought, day by day.
//
// ⚠️ **Three numbers from two different systems, each labelled with its
// source.** The spend is Meta's, in Meta's currency; the orders and their value
// are Meta's attribution of the events *this* server reported; and the panel
// puts the restaurant's own takings beside them without ever subtracting one
// from the other. "Profit from advertising" would be a single number made of
// two currencies and somebody else's attribution model — exactly the kind an
// owner would act on and we could not defend.
func (h *Handler) AdminAdsReport(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	ctx := r.Context()
	// Thirty days by default: long enough that a week-long campaign is whole in
	// it, short enough to stay one screen.
	days := 30
	if n, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("days"))); err == nil &&
		n > 0 && n <= 180 {
		days = n
	}
	from := time.Now().In(time.Local).AddDate(0, 0, -days).Format("2006-01-02")

	match := bson.M{"day": bson.M{"$gte": from}}
	cur, err := h.Store.AdsDaily.Find(ctx, match,
		options.Find().SetSort(bson.D{{Key: "day", Value: 1}}).SetLimit(2000))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "o'qib bo'lmadi")
		return
	}
	var rows []models.AdDaily
	if err := cur.All(ctx, &rows); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "o'qib bo'lmadi")
		return
	}
	// One row per day, summed across campaigns — the shape the chart draws.
	byDay := map[string]*adsSums{}
	order := []string{}
	for _, d := range rows {
		s, ok := byDay[d.Day]
		if !ok {
			s = &adsSums{}
			byDay[d.Day] = s
			order = append(order, d.Day)
		}
		s.Spend += d.Spend
		s.Impressions += d.Impressions
		s.Clicks += d.Clicks
		s.Purchases += d.Purchases
		s.Revenue += d.Revenue
	}
	out := make([]map[string]any, 0, len(order))
	for _, day := range order {
		s := byDay[day]
		out = append(out, map[string]any{
			"day": day, "spend": s.Spend, "impressions": s.Impressions,
			"clicks": s.Clicks, "purchases": s.Purchases, "revenue": s.Revenue,
		})
	}

	set := h.adsSettings(ctx)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"days":     out,
		"total":    h.adsTotals(ctx, match),
		"currency": set.Currency,
		// ⚠️ Said in the payload rather than only in the interface copy: every
		// screen that draws this has to be able to name whose attribution it
		// is showing, and a caption written twice is a caption that drifts.
		"attribution": "meta",
		"pixel":       set.PixelID != "",
		"lastSyncAt":  h.adsLastSync(ctx),
	})
}

// adsLastSync is the newest snapshot we hold — the honest "as of".
func (h *Handler) adsLastSync(ctx context.Context) time.Time {
	var row models.AdDaily
	err := h.Store.AdsDaily.FindOne(ctx, bson.M{},
		options.FindOne().SetSort(bson.D{{Key: "takenAt", Value: -1}})).Decode(&row)
	if err != nil {
		return time.Time{}
	}
	return row.TakenAt
}
