package models

// ---- Advertising: what we connected, what we created, what it cost ----
//
// ⚠️ **Three records because they answer three questions that age differently.**
// The connection is a fact about today (is the token still good?), a campaign
// is a decision that was made once and never changes, and a day's figures are
// history Meta itself rewrites. Folded into one document, re-reading yesterday's
// spend would overwrite the reason a dish was chosen.

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Connection ----

// AdsSettings is the restaurant's own Meta assets, and our key to them.
//
// ⚠️ **Singleton, and outside `restaurant`.** The profile document is returned
// whole to every visitor of the public site (CLAUDE.md §4); a token that can
// spend money from somebody's card has no business travelling with the opening
// hours.
type AdsSettings struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"-"`

	// The business integration system user token. ⚠️ **`json:"-"`, and there is
	// no endpoint that returns it** — the settings answer carries `hasToken`
	// and nothing else, exactly like the payment keys next door.
	Token string `bson:"token,omitempty" json:"-"`

	BusinessID   string `bson:"businessId,omitempty" json:"businessId,omitempty"`
	BusinessName string `bson:"businessName,omitempty" json:"businessName,omitempty"`

	// `act_123…`, as every Graph path wants it.
	AdAccountID   string `bson:"adAccountId,omitempty" json:"adAccountId,omitempty"`
	AdAccountName string `bson:"adAccountName,omitempty" json:"adAccountName,omitempty"`
	// What Meta bills this account in, as Meta reported it — **not** assumed to
	// be the som. Every budget on the screen is shown in this currency because
	// that is the number that will appear on the owner's card statement.
	Currency string `bson:"currency,omitempty" json:"currency,omitempty"`
	// Meta's own floor for a daily budget, in that currency's minor units.
	MinDailyBudget int `bson:"minDailyBudget,omitempty" json:"minDailyBudget,omitempty"`

	PageID      string `bson:"pageId,omitempty" json:"pageId,omitempty"`
	PageName    string `bson:"pageName,omitempty" json:"pageName,omitempty"`
	InstagramID string `bson:"instagramId,omitempty" json:"instagramId,omitempty"`
	PixelID     string `bson:"pixelId,omitempty" json:"pixelId,omitempty"`
	PixelName   string `bson:"pixelName,omitempty" json:"pixelName,omitempty"`

	ConnectedAt time.Time `bson:"connectedAt,omitempty" json:"connectedAt,omitempty"`
	ConnectedBy string    `bson:"connectedBy,omitempty" json:"connectedBy,omitempty"`

	// When we last got a straight answer out of Meta, and what it was.
	//
	// ⚠️ **A date, not only a flag.** A stored "ok" ages the moment the clock
	// passes it — the lesson the console's `provisionStatus` taught this
	// platform — and the one failure this section must never hide is a token
	// that stopped working while campaigns carried on spending.
	LastCheckAt time.Time `bson:"lastCheckAt,omitempty" json:"lastCheckAt,omitempty"`
	// "" (never checked) | "ok" | "revoked" | "error".
	Status     string `bson:"status,omitempty" json:"status,omitempty"`
	StatusNote string `bson:"statusNote,omitempty" json:"statusNote,omitempty"`

	Rules AdRules `bson:"rules" json:"rules"`
}

// AdRules is the standing instruction the owner leaves behind.
//
// ⚠️ **Every number here is a ceiling, never a target.** The rules can stop a
// campaign and can move a budget downwards; the only upward move is inside
// `MaxDailyMinor`, which the owner typed. A machine that could raise a budget
// on its own judgement is a machine spending somebody else's money — and the
// first bad week would be ours rather than a decision they made.
type AdRules struct {
	On bool `bson:"on" json:"on"`
	// The ceiling for any single campaign's daily budget, in the ad account
	// currency's minor units. 0 means the rules may never raise anything.
	MaxDailyMinor int `bson:"maxDailyMinor,omitempty" json:"maxDailyMinor,omitempty"`
	// Stop a campaign that has spent this much (minor units) without a single
	// attributed order.
	NoResultMinor int `bson:"noResultMinor,omitempty" json:"noResultMinor,omitempty"`
	// Stop it when an order costs more than this (minor units) over the window.
	MaxCostMinor int `bson:"maxCostMinor,omitempty" json:"maxCostMinor,omitempty"`
	// Whether the budget may be moved at all, within the ceiling above.
	Tune bool `bson:"tune,omitempty" json:"tune,omitempty"`
}

// ---- One campaign ----

// AdCampaignStatus is ours, not Meta's.
//
// ⚠️ **Kept apart from `effective_status` on purpose.** Meta has a dozen
// states — `PENDING_REVIEW`, `WITH_ISSUES`, `CAMPAIGN_PAUSED`,
// `ADSET_PAUSED` — and an owner asking "is my advert running?" is not asking
// which object in the chain is paused. Ours answers their question; Meta's is
// carried beside it, untranslated, for when the answer is "no, and here is why".
const (
	AdCampaignDraft   = "draft"
	AdCampaignActive  = "active"
	AdCampaignPaused  = "paused"
	AdCampaignStopped = "stopped"
	AdCampaignFailed  = "failed"
)

// AdDecision is one thing the rules did, and why.
//
// ⚠️ **Written whether or not anybody reads it.** An automatic pause that the
// owner discovers as "my advert stopped" with no explanation costs more trust
// than the money it saved.
type AdDecision struct {
	At     time.Time `bson:"at" json:"at"`
	Action string    `bson:"action" json:"action"`
	Why    string    `bson:"why" json:"why"`
	From   int       `bson:"from,omitempty" json:"from,omitempty"`
	To     int       `bson:"to,omitempty" json:"to,omitempty"`
}

// AdCampaign is our record of a decision, not Meta's copy of an object.
//
// ⚠️ **Meta already holds the campaign; what it does not hold is why.** Which
// dish, on what evidence, which of three proposals the owner took, what ceiling
// they set. That is the half this section is sold for, and the half that would
// be gone the moment somebody archived the campaign in Ads Manager.
type AdCampaign struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BrandID  primitive.ObjectID `bson:"brandId,omitempty" json:"brandId,omitempty"`
	BranchID primitive.ObjectID `bson:"branchId,omitempty" json:"branchId,omitempty"`

	MetaCampaignID string `bson:"metaCampaignId,omitempty" json:"metaCampaignId,omitempty"`
	AdSetID        string `bson:"adsetId,omitempty" json:"adsetId,omitempty"`
	AdID           string `bson:"adId,omitempty" json:"adId,omitempty"`
	CreativeID     string `bson:"creativeId,omitempty" json:"creativeId,omitempty"`
	ImageHash      string `bson:"imageHash,omitempty" json:"imageHash,omitempty"`

	Name      string `bson:"name" json:"name"`
	Objective string `bson:"objective,omitempty" json:"objective,omitempty"`
	Goal      string `bson:"goal,omitempty" json:"goal,omitempty"`

	// What is being advertised, and the reason the plan gave for it.
	DishID   primitive.ObjectID `bson:"dishId,omitempty" json:"dishId,omitempty"`
	DishName string             `bson:"dishName,omitempty" json:"dishName,omitempty"`
	Why      string             `bson:"why,omitempty" json:"why,omitempty"`

	AreaLabel string  `bson:"areaLabel,omitempty" json:"areaLabel,omitempty"`
	RadiusKm  float64 `bson:"radiusKm,omitempty" json:"radiusKm,omitempty"`
	Lat       float64 `bson:"lat,omitempty" json:"lat,omitempty"`
	Lng       float64 `bson:"lng,omitempty" json:"lng,omitempty"`

	// The budget as Meta counts it, and the currency it is counted in.
	//
	// ⚠️ **Minor units, and the currency beside it in the same document.** A
	// number without its currency is the bug this section cannot afford: the
	// plan proposes in so'm and the account almost always bills in dollars.
	DailyMinor int    `bson:"dailyMinor" json:"dailyMinor"`
	Currency   string `bson:"currency,omitempty" json:"currency,omitempty"`
	// The ceiling the owner set for this campaign, in the same units. The rules
	// may never pass it and neither may the owner's own later edit.
	CapMinor int `bson:"capMinor,omitempty" json:"capMinor,omitempty"`
	Days     int `bson:"days,omitempty" json:"days,omitempty"`

	Headline string `bson:"headline,omitempty" json:"headline,omitempty"`
	Body     string `bson:"body,omitempty" json:"body,omitempty"`
	Link     string `bson:"link,omitempty" json:"link,omitempty"`

	Status string `bson:"status" json:"status"`
	// Meta's own word for the advert, as last read, plus the reason when it was
	// refused — in a sentence a person can act on.
	MetaStatus string `bson:"metaStatus,omitempty" json:"metaStatus,omitempty"`
	ReviewNote string `bson:"reviewNote,omitempty" json:"reviewNote,omitempty"`

	CreatedAt time.Time  `bson:"createdAt" json:"createdAt"`
	CreatedBy string     `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	StartedAt *time.Time `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	EndsAt    *time.Time `bson:"endsAt,omitempty" json:"endsAt,omitempty"`
	StoppedAt *time.Time `bson:"stoppedAt,omitempty" json:"stoppedAt,omitempty"`

	// Running totals, refreshed by the daily sync.
	//
	// ⚠️ **Denormalised deliberately.** The list screen shows a dozen campaigns
	// and each total is a grouping over `ad_daily`; a dozen aggregations per
	// page view is how a screen that is opened every morning becomes the slow
	// one. The days remain the source — this is a cache of them, rewritten
	// whole on every sync rather than added to.
	Spend      float64   `bson:"spend,omitempty" json:"spend"`
	Purchases  int       `bson:"purchases,omitempty" json:"purchases"`
	Revenue    float64   `bson:"revenue,omitempty" json:"revenue"`
	LastSyncAt time.Time `bson:"lastSyncAt,omitempty" json:"lastSyncAt,omitempty"`

	Decisions []AdDecision `bson:"decisions,omitempty" json:"decisions,omitempty"`
}

// ---- One day of one campaign ----

// AdDaily is what Meta said about one campaign on one day.
//
// ⚠️ **History, not a cache.** Meta rewrites insights as attribution windows
// close, so "what did we see that morning" and "what does Meta say now" are two
// different questions — and the rules that paused a campaign were acting on the
// first. Unique on (campaign, day) so a re-sync corrects a row instead of
// laying a second one beside it.
type AdDaily struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	CampaignID     primitive.ObjectID `bson:"campaignId" json:"campaignId"`
	MetaCampaignID string             `bson:"metaCampaignId" json:"-"`
	Day            string             `bson:"day" json:"day"`

	Spend       float64 `bson:"spend" json:"spend"`
	Impressions int     `bson:"impressions" json:"impressions"`
	Clicks      int     `bson:"clicks" json:"clicks"`
	Purchases   int     `bson:"purchases" json:"purchases"`
	Revenue     float64 `bson:"revenue" json:"revenue"`

	Currency string    `bson:"currency,omitempty" json:"currency,omitempty"`
	TakenAt  time.Time `bson:"takenAt" json:"takenAt"`
}
