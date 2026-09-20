package meta

// ---- Building one campaign: four objects that must be made in order ----
//
// `campaign` → `ad set` → `ad creative` (+ `ad image`) → `ad`. Meta has no
// transaction across them: the campaign exists the moment it is created, and if
// the ad set fails afterwards the restaurant is left with an empty campaign in
// their Ads Manager. So everything that can be checked is checked *before* the
// first call, and what is created is written down as it is created — see
// handlers/adscampaign.go, which rolls the half-built chain back.
//
// ⚠️ **Nothing here decides anything.** No budget is chosen, no radius is
// clamped, no objective is picked in this file: it is the shape of the API and
// nothing else. The decisions, and the ceilings the owner set on them, live in
// the handler — one place, with the tests.
//
// ⚠️ **Created paused, always.** An advert that starts running the instant a
// button is pressed leaves no moment in which a mistake is still free. The
// handler switches it on as its own step, after the chain is whole.

import (
	"context"
	"math"
	"net/url"
	"strconv"
)

// Created is the id every create call answers with.
type Created struct {
	ID string `json:"id"`
}

// CampaignSpec is one campaign.
type CampaignSpec struct {
	Name string
	// `OUTCOME_SALES` when a pixel is reporting orders, `OUTCOME_TRAFFIC`
	// otherwise. The old objective names still exist and are being replaced;
	// we only ever send the new ones.
	Objective string
}

// CreateCampaign makes the outer object.
//
// ⚠️ **`special_ad_categories` is required and cannot be left out**, even
// though a restaurant is in none of them. Meta rejects the create without the
// field rather than defaulting it, and the error names something else.
//
// ⚠️ **No budget here.** Meta refuses a budget on the campaign *and* the ad set
// at once, and ours lives on the ad set, where the targeting it pays for is.
func (c *Client) CreateCampaign(ctx context.Context, act string, s CampaignSpec) (string, error) {
	form := url.Values{
		"name":                  {s.Name},
		"objective":             {s.Objective},
		"status":                {"PAUSED"},
		"buying_type":           {"AUCTION"},
		"special_ad_categories": {JSONField([]string{"NONE"})},
	}
	var out Created
	err := c.Post(ctx, act+"/campaigns", form, &out)
	return out.ID, err
}

// Targeting is who sees the advert.
//
// ⚠️ **A circle on the map, and nothing else about the person.** In 2026 Meta
// picks the audience itself (`advantage_audience`), which is the same button a
// $500-a-month targetolog presses; what we bring is the kitchen's own reach —
// the address it cooks at and how far its van goes.
type Targeting struct {
	Lat, Lng float64
	RadiusKm float64
	AgeMin   int
	AgeMax   int
}

func (t Targeting) spec() map[string]any {
	loc := map[string]any{
		"latitude":      t.Lat,
		"longitude":     t.Lng,
		"radius":        t.RadiusKm,
		"distance_unit": "kilometer",
	}
	tg := map[string]any{
		// ⚠️ **A custom location alone, never a country beside it.** Meta
		// rejects a country given together with a place inside it as an
		// "overlap", and the message does not say which two fields overlapped.
		"geo_locations": map[string]any{
			"custom_locations": []any{loc},
		},
		"publisher_platforms": []string{"facebook", "instagram"},
		// The audience flag itself: Meta's own AI widens from the circle.
		"targeting_automation": map[string]any{"advantage_audience": 1},
	}
	if t.AgeMin > 0 {
		tg["age_min"] = t.AgeMin
	}
	if t.AgeMax > 0 {
		tg["age_max"] = t.AgeMax
	}
	return tg
}

// AdSetSpec is the budget, the audience and what Meta optimises towards.
type AdSetSpec struct {
	Name       string
	CampaignID string
	// ⚠️ **In the account currency's minor units** — see MinorUnits. The handler
	// converts once, against the account Meta itself reported, and clamps to
	// the ceiling the owner set.
	DailyBudget int
	Targeting   Targeting
	// `OFFSITE_CONVERSIONS` with a pixel, `LINK_CLICKS` without one.
	OptimizationGoal string
	BillingEvent     string
	// Set only for OFFSITE_CONVERSIONS, where Meta requires it.
	PixelID         string
	CustomEventType string
	// When the campaign stops paying. ⚠️ **Always set**: an ad set with no end
	// spends until somebody remembers it, and the owner chose a number of days.
	EndTime string
}

// CreateAdSet makes the object that actually spends money.
func (c *Client) CreateAdSet(ctx context.Context, act string, s AdSetSpec) (string, error) {
	form := url.Values{
		"name":              {s.Name},
		"campaign_id":       {s.CampaignID},
		"daily_budget":      {strconv.Itoa(s.DailyBudget)},
		"billing_event":     {s.BillingEvent},
		"optimization_goal": {s.OptimizationGoal},
		"targeting":         {JSONField(s.Targeting.spec())},
		"status":            {"PAUSED"},
	}
	if s.EndTime != "" {
		form.Set("end_time", s.EndTime)
	}
	if s.PixelID != "" && s.CustomEventType != "" {
		form.Set("promoted_object", JSONField(map[string]any{
			"pixel_id":          s.PixelID,
			"custom_event_type": s.CustomEventType,
		}))
	}
	var out Created
	err := c.Post(ctx, act+"/adsets", form, &out)
	return out.ID, err
}

// Image is what an upload answers with.
//
// ⚠️ **Only the hash is usable.** The `url` Meta returns is temporary and a
// creative built from it breaks days later, with the advert still running and
// the picture gone.
type Image struct {
	Hash string `json:"hash"`
	URL  string `json:"url"`
}

// UploadImage puts one of the restaurant's own dish photographs on the account.
//
// ⚠️ **Their photograph, never a generated one.** The pictures are already in
// `uploads/` because somebody photographed the food they actually serve; an
// invented image of a dish is an advert that lies about what arrives.
func (c *Client) UploadImage(ctx context.Context, act, filename string, data []byte) (Image, error) {
	var out struct {
		Images map[string]Image `json:"images"`
	}
	if err := c.PostFile(ctx, act+"/adimages", "source", filename, data, nil, &out); err != nil {
		return Image{}, err
	}
	// Meta keys the answer by the file name it was given, which is why the
	// name matters twice: once for the extension, once to find the result.
	for _, img := range out.Images {
		return img, nil
	}
	return Image{}, &Error{Message: "Meta rasmni qabul qilmadi"}
}

// CreativeSpec is the advert as a reader sees it.
type CreativeSpec struct {
	Name        string
	PageID      string
	InstagramID string
	ImageHash   string
	Link        string
	Message     string
	Headline    string
	Description string
	// `ORDER_NOW` for delivery, `LEARN_MORE` otherwise.
	CallToAction string
}

// CreateCreative makes the advert's body.
//
// ⚠️ **A creative cannot be edited after it is made** (docs/vendor/meta-marketing.md
// §4.5). Changing a word means a new creative and a new ad, which is why the
// owner picks the wording *before* anything is created rather than after seeing
// it live — this is Meta's rule, not a limitation of our screen.
func (c *Client) CreateCreative(ctx context.Context, act string, s CreativeSpec) (string, error) {
	link := map[string]any{
		"image_hash": s.ImageHash,
		"link":       s.Link,
		"message":    s.Message,
		"name":       s.Headline,
	}
	if s.Description != "" {
		link["description"] = s.Description
	}
	if s.CallToAction != "" {
		link["call_to_action"] = map[string]any{
			"type":  s.CallToAction,
			"value": map[string]any{"link": s.Link},
		}
	}
	story := map[string]any{
		"page_id":   s.PageID,
		"link_data": link,
	}
	if s.InstagramID != "" {
		story["instagram_user_id"] = s.InstagramID
	}
	form := url.Values{
		"name":              {s.Name},
		"object_story_spec": {JSONField(story)},
	}
	var out Created
	err := c.Post(ctx, act+"/adcreatives", form, &out)
	return out.ID, err
}

// AdSpec ties a creative to an ad set.
type AdSpec struct {
	Name       string
	AdSetID    string
	CreativeID string
	// ⚠️ Required by Meta whenever the campaign shares data with a pixel, and
	// it is a domain — `keel.uz`, never `https://keel.uz/menu`.
	ConversionDomain string
}

// CreateAd makes the last object. It arrives `PENDING_REVIEW`: Meta reads every
// advert before it runs, and nothing we do here shortens that.
func (c *Client) CreateAd(ctx context.Context, act string, s AdSpec) (string, error) {
	form := url.Values{
		"name":     {s.Name},
		"adset_id": {s.AdSetID},
		"creative": {JSONField(map[string]any{"creative_id": s.CreativeID})},
		"status":   {"PAUSED"},
	}
	if s.ConversionDomain != "" {
		form.Set("conversion_domain", s.ConversionDomain)
	}
	var out Created
	err := c.Post(ctx, act+"/ads", form, &out)
	return out.ID, err
}

// SetStatus switches one object on or off. `ACTIVE` | `PAUSED` | `ARCHIVED`.
func (c *Client) SetStatus(ctx context.Context, id, status string) error {
	return c.Post(ctx, id, url.Values{"status": {status}}, nil)
}

// SetDailyBudget moves an ad set's daily budget, in minor units.
func (c *Client) SetDailyBudget(ctx context.Context, adsetID string, minor int) error {
	return c.Post(ctx, adsetID,
		url.Values{"daily_budget": {strconv.Itoa(minor)}}, nil)
}

// Delete removes an object we created — used only to undo a half-built chain.
func (c *Client) Delete(ctx context.Context, id string) error {
	return c.do(ctx, "DELETE", id, nil, nil, "", nil)
}

// Review is what Meta made of an advert.
type Review struct {
	ID              string `json:"id"`
	EffectiveStatus string `json:"effective_status"`
	// Present when the advert was rejected, and the only place the reason is.
	IssuesInfo []struct {
		ErrorCode    int    `json:"error_code"`
		ErrorSummary string `json:"error_summary"`
		ErrorMessage string `json:"error_message"`
	} `json:"issues_info"`
}

// AdReview reads an advert's standing.
//
// ⚠️ **"rejected" is not an answer an owner can act on.** The reason is in
// `issues_info` and nowhere else; a screen that showed only the status would
// send somebody to Ads Manager to find out what we already had.
func (c *Client) AdReview(ctx context.Context, adID string) (Review, error) {
	var out Review
	err := c.Get(ctx, adID,
		url.Values{"fields": {"id,effective_status,issues_info"}}, &out)
	return out, err
}

// ---- What Meta thinks this money will buy ----

// Estimate is Meta's own forecast for a targeting and a budget.
//
// ⚠️ **Meta's number, never ours, and the screen says so.** The planner is
// forbidden from predicting results for a reason — an owner who was promised a
// figure measures us against it. This is different in kind: it is the same
// estimate Ads Manager shows them, fetched from the system that will actually
// deliver the adverts, and it is labelled with whose it is.
type Estimate struct {
	// How many people the targeting can reach at all. ⚠️ Monthly and daily
	// active, as Meta counts them — not "customers".
	DAU int64 `json:"estimate_dau"`
	MAU int64 `json:"estimate_mau"`
	// ⚠️ **False means the numbers are not usable yet**, which happens on a
	// fresh account or an audience Meta considers too small. Showing the zeroes
	// as a forecast would be inventing a prediction of nothing.
	Ready bool `json:"estimate_ready"`
	// Spend against outcome, in the ad account's currency's minor units.
	Curve []EstimatePoint `json:"daily_outcomes_curve"`
}

// EstimatePoint is one budget on Meta's curve.
type EstimatePoint struct {
	Spend       float64 `json:"spend"`
	Reach       float64 `json:"reach"`
	Impressions float64 `json:"impressions"`
	Actions     float64 `json:"actions"`
}

// At is the point on the curve closest to a daily budget, in minor units.
//
// ⚠️ **The nearest point, never an interpolation.** Meta returns the budgets it
// is willing to answer for; a number invented between two of them would be ours
// while looking like theirs.
func (e Estimate) At(minor int) (EstimatePoint, bool) {
	if !e.Ready || len(e.Curve) == 0 {
		return EstimatePoint{}, false
	}
	want := float64(minor)
	best, bestGap := e.Curve[0], math.Abs(e.Curve[0].Spend-want)
	for _, p := range e.Curve[1:] {
		if gap := math.Abs(p.Spend - want); gap < bestGap {
			best, bestGap = p, gap
		}
	}
	return best, true
}

// DeliveryEstimate asks Meta what a targeting is worth.
func (c *Client) DeliveryEstimate(
	ctx context.Context, act, goal string, t Targeting, pixelID, event string,
) (Estimate, error) {
	params := url.Values{
		"optimization_goal": {goal},
		"targeting_spec":    {JSONField(t.spec())},
	}
	if pixelID != "" && event != "" {
		params.Set("promoted_object", JSONField(map[string]any{
			"pixel_id":          pixelID,
			"custom_event_type": event,
		}))
	}
	var out listOf[Estimate]
	if err := c.Get(ctx, act+"/delivery_estimate", params, &out); err != nil {
		return Estimate{}, err
	}
	if len(out.Data) == 0 {
		return Estimate{}, nil
	}
	return out.Data[0], nil
}

// ---- Boosting a post the restaurant already published ----
//
// ⚠️ **A different creative, not a different picture.** Everything above builds
// an advert out of parts: a photograph, a headline, a body, a link. This builds
// one out of something the restaurant already posted and that its followers
// already reacted to — Meta keeps the likes and comments on the promoted post,
// which is the whole reason an owner asks for it. The two cannot be mixed:
// a creative is either assembled or borrowed.

// IGPost is one thing the restaurant published on Instagram.
type IGPost struct {
	ID        string `json:"id"`
	MediaType string `json:"media_type"`
	MediaURL  string `json:"media_url"`
	Thumbnail string `json:"thumbnail_url"`
	Caption   string `json:"caption"`
	Permalink string `json:"permalink"`
	Timestamp string `json:"timestamp"`
	// ⚠️ **Meta decides what may be boosted, not us.** A post with music, with
	// somebody else's material, or simply too old is refused at creation with
	// a message about the media id — long after the owner chose it. This field
	// is the same answer, before the choice.
	Boost struct {
		Eligible bool     `json:"eligible_to_boost"`
		Reasons  []string `json:"eligibility_reasons"`
	} `json:"boost_eligibility_info"`
}

// InstagramPosts lists what the connected Instagram account has published.
func (c *Client) InstagramPosts(ctx context.Context, igUserID string, limit int) ([]IGPost, error) {
	if limit <= 0 || limit > 50 {
		limit = 24
	}
	var out listOf[IGPost]
	err := c.Get(ctx, igUserID+"/media", url.Values{
		"fields": {"id,media_type,media_url,thumbnail_url,caption," +
			"permalink,timestamp,boost_eligibility_info"},
		"limit": {strconv.Itoa(limit)},
	}, &out)
	return out.Data, err
}

// ConnectedInstagram is the Instagram account an ad account may advertise as.
//
// ⚠️ **Asked of the ad account, not only of the Page.** The Page's
// `instagram_business_account` is the usual answer and sometimes empty while
// the account is perfectly reachable from the ad account — two edges for one
// fact, and an install that read only the first showed an owner no posts at
// all.
func (c *Client) ConnectedInstagram(ctx context.Context, act string) (string, error) {
	var out listOf[struct {
		ID string `json:"id"`
	}]
	if err := c.Get(ctx, act+"/connected_instagram_accounts",
		url.Values{"fields": {"id"}}, &out); err != nil {
		return "", err
	}
	if len(out.Data) == 0 {
		return "", nil
	}
	return out.Data[0].ID, nil
}

// CreateCreativeFromPost makes an advert out of an existing Instagram post.
//
// ⚠️ **Three ids and no content.** The Page owns the advert, the Instagram
// account published it, and the media id says which post. Nothing about the
// wording or the picture travels — they are the post's, and changing either
// would make it a different post.
func (c *Client) CreateCreativeFromPost(
	ctx context.Context, act, pageID, igUserID, mediaID, name string,
) (string, error) {
	form := url.Values{
		"name":                      {name},
		"object_id":                 {pageID},
		"instagram_user_id":         {igUserID},
		"source_instagram_media_id": {mediaID},
	}
	var outCreated Created
	err := c.Post(ctx, act+"/adcreatives", form, &outCreated)
	return outCreated.ID, err
}

// PagePost is one thing the restaurant published on its Facebook Page.
//
// ⚠️ **The path that works with the permissions an ads integration is actually
// granted.** Listing Instagram media needs `instagram_basic`, which Meta does
// not offer inside a Login for Business configuration built on a system user
// token — the token an ad account needs. A Page's own posts need
// `pages_read_engagement`, which it does offer, and a Page post is delivered on
// Instagram placements too when an Instagram account is attached to the Page.
type PagePost struct {
	// Already `{page-id}_{post-id}`, which is exactly what `object_story_id`
	// wants — no assembling on our side.
	ID          string `json:"id"`
	Message     string `json:"message"`
	FullPicture string `json:"full_picture"`
	Permalink   string `json:"permalink_url"`
	CreatedTime string `json:"created_time"`
}

// PagePosts lists what the Page has published.
//
// ⚠️ **`published_posts`, not `feed`.** The feed carries what other people
// wrote on the Page as well, and an advert built from a stranger's post is
// money spent promoting somebody else's words.
func (c *Client) PagePosts(ctx context.Context, pageID string, limit int) ([]PagePost, error) {
	if limit <= 0 || limit > 50 {
		limit = 24
	}
	var out listOf[PagePost]
	err := c.Get(ctx, pageID+"/published_posts", url.Values{
		"fields": {"id,message,full_picture,permalink_url,created_time"},
		"limit":  {strconv.Itoa(limit)},
	}, &out)
	return out.Data, err
}

// CreateCreativeFromPagePost makes an advert out of a post the Page published.
func (c *Client) CreateCreativeFromPagePost(
	ctx context.Context, act, storyID, name string,
) (string, error) {
	form := url.Values{
		"name":            {name},
		"object_story_id": {storyID},
	}
	var out Created
	err := c.Post(ctx, act+"/adcreatives", form, &out)
	return out.ID, err
}
