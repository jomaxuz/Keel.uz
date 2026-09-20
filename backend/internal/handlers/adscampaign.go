package handlers

// ---- Creating one campaign: the moment money starts moving ----
//
// ⚠️ **Everything that can be refused is refused before the first call to
// Meta.** There is no transaction across the four objects a campaign is made
// of: the campaign exists the instant it is created, and a failure at the ad
// set leaves an owner with a stray empty campaign in their Ads Manager. So the
// budget, the currency, the radius, the picture and the wording are all
// checked here, and what does get created is written down as it is created —
// and unwound in reverse if the chain breaks halfway.
//
// ⚠️ **The budget the model proposed is not the budget that is sent.** The plan
// writes in so'm because that is what the restaurant counts in; the ad account
// is billed by Meta in its own currency, almost never the som, and in that
// currency's *minor units*. So the owner types the daily budget in the account's
// currency, this server converts it once against the account Meta itself
// reported, and refuses anything below Meta's own floor. A number carried
// straight through from the model would be a campaign spending a hundred times
// what anybody agreed to — on a real card, immediately.
//
// ⚠️ **Created paused, started as a separate step.** Between the last object
// and the switch there is one moment where a mistake is still free, and this is
// the only place in the panel where that moment is worth engineering for.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/meta"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The circle Meta will accept around a point, in kilometres.
//
// ⚠️ **Clamped rather than passed through.** Meta refuses anything outside its
// own range with an error that names the targeting spec and not the radius, and
// a restaurant that delivers 2 km has no business advertising to 80.
const (
	adsMinRadiusKm = 1.0
	adsMaxRadiusKm = 80.0
)

var errAdsNoBranch = errors.New("reklama uchun filialni tanlang")

// adsBranch resolves the one branch a campaign is for.
//
// ⚠️ **One branch, like the stock screens and for the same reason.** A campaign
// is a circle drawn around a kitchen: "the company" has no address, no delivery
// radius and no opening hours, and an advert aimed at the average of three
// districts reaches people who cannot order from any of them.
func (h *Handler) adsBranch(r *http.Request) (models.Branch, error) {
	sc, err := h.adminScope(r)
	if err != nil {
		return models.Branch{}, err
	}
	id := sc.BranchID
	if id.IsZero() {
		only, err := h.onlyBranch(r, sc.BrandID)
		if err != nil || only.IsZero() {
			return models.Branch{}, errAdsNoBranch
		}
		id = only
	}
	var b models.Branch
	if err := h.Store.Branches.FindOne(r.Context(), bson.M{"_id": id}).Decode(&b); err != nil {
		return models.Branch{}, errAdsNoBranch
	}
	return b, nil
}

type adsCreateRequest struct {
	// What is being advertised, and the reason the plan gave for it — kept
	// because Meta stores the campaign and not why anybody made it.
	DishID   string `json:"dishId"`
	DishName string `json:"dishName"`
	Why      string `json:"why"`

	AreaLabel string  `json:"areaLabel"`
	RadiusKm  float64 `json:"radiusKm"`

	// In the **ad account's** currency, as the owner reads it on the screen —
	// 5 means five dollars on a dollar account. Never minor units: a browser
	// doing that conversion is a browser that can get it wrong.
	Daily float64 `json:"daily"`
	// The ceiling the owner sets for this campaign, same units. The rules may
	// never pass it.
	Cap  float64 `json:"cap"`
	Days int     `json:"days"`

	Headline string `json:"headline"`
	Body     string `json:"body"`
	Link     string `json:"link"`

	// Whether to switch it on straight away. Default is to leave it paused.
	Start bool `json:"start"`
}

// AdminAdsCreateCampaign builds the whole chain at Meta.
func (h *Handler) AdminAdsCreateCampaign(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req adsCreateRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<18)).Decode(&req) != nil {
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
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if s.AdAccountID == "" {
		httpx.Error(w, http.StatusBadRequest, errAdsNoAct.Error())
		return
	}
	if s.PageID == "" {
		httpx.Error(w, http.StatusBadRequest, errAdsNoPage.Error())
		return
	}

	// ---- The kitchen has to be somewhere ----
	//
	// ⚠️ A branch with no point on the map is the silent failure the delivery
	// settings page already warns about: the form looks filled in and the
	// result is an advert aimed at nowhere.
	if branch.Address.Lat == 0 && branch.Address.Lng == 0 {
		httpx.Error(w, http.StatusBadRequest,
			"filial xaritada belgilanmagan — reklama hududi shundan o'lchanadi")
		return
	}

	headline := strings.TrimSpace(req.Headline)
	body := strings.TrimSpace(req.Body)
	if headline == "" || body == "" {
		httpx.Error(w, http.StatusBadRequest, "reklama matni tanlanmagan")
		return
	}
	// Meta's own limit on the headline. Refused rather than truncated: a
	// sentence cut in half is an advert the restaurant did not write.
	if len([]rune(headline)) > 90 {
		httpx.Error(w, http.StatusBadRequest, "sarlavha 90 belgidan oshmasligi kerak")
		return
	}

	// ---- The picture ----
	//
	// ⚠️ **The restaurant's own photograph or nothing.** The dish photographs
	// are already on this server because somebody photographed the food they
	// actually serve; an invented picture is an advert that lies about what
	// arrives at the door.
	dishID, _ := objectID(req.DishID)
	imgName, imgData, err := h.adsDishPhoto(ctx, dishID, req.DishName)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// ---- The money ----
	acc, err := client.Account(ctx, s.AdAccountID)
	if err != nil {
		h.noteMetaError(ctx, err)
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	if !acc.Live() {
		httpx.Error(w, http.StatusBadRequest,
			"bu reklama akkaunti Meta tomonidan faol emas")
		return
	}
	unit := meta.MinorUnits(acc.Currency)
	daily := int(math.Round(req.Daily * float64(unit)))
	capMinor := int(math.Round(req.Cap * float64(unit)))
	if capMinor <= 0 {
		// No ceiling typed means the budget itself is the ceiling — never
		// "unlimited". An absent number must be the safe number.
		capMinor = daily
	}
	if daily <= 0 {
		httpx.Error(w, http.StatusBadRequest, "kunlik byudjet ko'rsatilmagan")
		return
	}
	if daily > capMinor {
		httpx.Error(w, http.StatusBadRequest,
			"kunlik byudjet o'zingiz qo'ygan chegaradan oshib ketdi")
		return
	}
	// ⚠️ **Meta's own floor, read from the account rather than assumed.** It is
	// the one check that catches a currency-unit mistake before the money
	// moves: a budget built on the wrong offset lands below or far above it.
	if acc.MinDailyBudget > 0 && daily < acc.MinDailyBudget {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf(
			"Meta kunlik byudjet uchun eng kami %s %s ni talab qiladi",
			money(acc.MinDailyBudget, unit), acc.Currency))
		return
	}
	if s.Rules.MaxDailyMinor > 0 && capMinor > s.Rules.MaxDailyMinor {
		httpx.Error(w, http.StatusBadRequest,
			"chegara sozlamalardagi umumiy chegaradan oshib ketdi")
		return
	}

	days := req.Days
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}
	ends := time.Now().Add(time.Duration(days) * 24 * time.Hour)

	radius := req.RadiusKm
	if radius <= 0 {
		// The kitchen's own reach when the owner did not move the slider: what
		// it delivers to is the only defensible default.
		radius = float64(branch.Delivery.MaxKm)
	}
	radius = math.Min(math.Max(radius, adsMinRadiusKm), adsMaxRadiusKm)

	link, domain, err := h.adsLink(req.Link)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// ---- What Meta is asked to optimise for ----
	//
	// ⚠️ **A pixel decides the objective, not a dropdown.** With one, the
	// orders this server reports are what Meta learns from and the campaign
	// chases purchases; without one, the honest thing to chase is a visit to
	// the menu — promising conversions Meta cannot see would spend the budget
	// against a target that never arrives.
	objective, goal, billing := "OUTCOME_TRAFFIC", "LINK_CLICKS", "IMPRESSIONS"
	if s.PixelID != "" {
		objective, goal = "OUTCOME_SALES", "OFFSITE_CONVERSIONS"
	}

	name := adsCampaignName(branch.Name, req.DishName)
	row := models.AdCampaign{
		BrandID:    branch.BrandID,
		BranchID:   branch.ID,
		Name:       name,
		Objective:  objective,
		Goal:       goal,
		DishID:     dishID,
		DishName:   strings.TrimSpace(req.DishName),
		Why:        strings.TrimSpace(req.Why),
		AreaLabel:  strings.TrimSpace(req.AreaLabel),
		RadiusKm:   radius,
		Lat:        branch.Address.Lat,
		Lng:        branch.Address.Lng,
		DailyMinor: daily,
		CapMinor:   capMinor,
		Currency:   acc.Currency,
		Days:       days,
		Headline:   headline,
		Body:       body,
		Link:       link,
		Status:     models.AdCampaignDraft,
		CreatedAt:  time.Now(),
		EndsAt:     &ends,
	}
	if claims := middleware.ClaimsFrom(ctx); claims != nil {
		row.CreatedBy = claims.UserID
	}

	// ---- The chain ----
	//
	// ⚠️ **Undone in reverse when a step fails.** Meta has no transaction here,
	// and the alternative is a restaurant's Ads Manager slowly filling with
	// empty campaigns from failed attempts — objects they did not make, cannot
	// explain and will not delete.
	var made []string
	fail := func(err error) {
		for i := len(made) - 1; i >= 0; i-- {
			_ = client.Delete(context.WithoutCancel(ctx), made[i])
		}
		h.noteMetaError(ctx, err)
		if e := meta.AsError(err); e != nil {
			httpx.Error(w, http.StatusBadGateway, e.Human())
			return
		}
		httpx.Error(w, http.StatusBadGateway, err.Error())
	}

	campaignID, err := client.CreateCampaign(ctx, acc.ID, meta.CampaignSpec{
		Name: name, Objective: objective,
	})
	if err != nil {
		fail(err)
		return
	}
	made = append(made, campaignID)
	row.MetaCampaignID = campaignID

	adsetID, err := client.CreateAdSet(ctx, acc.ID, meta.AdSetSpec{
		Name:        name,
		CampaignID:  campaignID,
		DailyBudget: daily,
		Targeting: meta.Targeting{
			Lat: branch.Address.Lat, Lng: branch.Address.Lng, RadiusKm: radius,
		},
		OptimizationGoal: goal,
		BillingEvent:     billing,
		PixelID:          pixelFor(goal, s.PixelID),
		CustomEventType:  eventFor(goal),
		// Meta wants this in its own format, and in the ad account's timezone;
		// an ISO-8601 stamp with an offset is what it documents.
		EndTime: ends.Format(time.RFC3339),
	})
	if err != nil {
		fail(err)
		return
	}
	made = append(made, adsetID)
	row.AdSetID = adsetID

	img, err := client.UploadImage(ctx, acc.ID, imgName, imgData)
	if err != nil {
		fail(err)
		return
	}
	row.ImageHash = img.Hash

	creativeID, err := client.CreateCreative(ctx, acc.ID, meta.CreativeSpec{
		Name:        name,
		PageID:      s.PageID,
		InstagramID: s.InstagramID,
		ImageHash:   img.Hash,
		Link:        link,
		Message:     body,
		Headline:    headline,
		// "Order now" is the truth on a site that takes orders; anything else
		// promises a shop that is not there.
		CallToAction: "ORDER_NOW",
	})
	if err != nil {
		fail(err)
		return
	}
	made = append(made, creativeID)
	row.CreativeID = creativeID

	adID, err := client.CreateAd(ctx, acc.ID, meta.AdSpec{
		Name:             name,
		AdSetID:          adsetID,
		CreativeID:       creativeID,
		ConversionDomain: domain,
	})
	if err != nil {
		fail(err)
		return
	}
	row.AdID = adID
	row.Status = models.AdCampaignPaused

	// ---- On, if the owner asked for it ----
	if req.Start {
		if err := h.adsSwitch(ctx, client, row, true); err != nil {
			// ⚠️ **Not unwound.** The chain is whole and correct; what failed
			// is the switch, and deleting a good campaign because one call
			// timed out would be worse than leaving it paused with the reason
			// on the screen.
			row.Status = models.AdCampaignPaused
			row.ReviewNote = err.Error()
		} else {
			row.Status = models.AdCampaignActive
			now := time.Now()
			row.StartedAt = &now
		}
	}

	res, err := h.Store.AdsCampaigns.InsertOne(ctx, row)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "saqlanmadi")
		return
	}
	if id, ok := res.InsertedID.(primitive.ObjectID); ok {
		row.ID = id
	}
	h.logAction(r, ActAdsCreate, "ads", row.MetaCampaignID, name,
		fmt.Sprintf("%s %s/kun", money(daily, unit), acc.Currency))
	httpx.JSON(w, http.StatusOK, row)
}

// pixelFor and eventFor keep the "only when the goal needs it" rule in one
// place. ⚠️ Meta rejects a `promoted_object` on a goal that does not take one.
func pixelFor(goal, pixel string) string {
	if goal == "OFFSITE_CONVERSIONS" {
		return pixel
	}
	return ""
}

func eventFor(goal string) string {
	if goal == "OFFSITE_CONVERSIONS" {
		return "PURCHASE"
	}
	return ""
}

// adsSwitch turns a whole campaign on or off.
//
// ⚠️ **All three objects, not just the campaign.** Meta pauses independently at
// every level, and an ad set left paused under an active campaign is a campaign
// that reports itself as running and spends nothing — which reads as our bug.
func (h *Handler) adsSwitch(
	ctx context.Context, client *meta.Client, row models.AdCampaign, on bool,
) error {
	status := "PAUSED"
	if on {
		status = "ACTIVE"
	}
	for _, id := range []string{row.MetaCampaignID, row.AdSetID, row.AdID} {
		if id == "" {
			continue
		}
		if err := client.SetStatus(ctx, id, status); err != nil {
			if e := meta.AsError(err); e != nil {
				return errors.New(e.Human())
			}
			return err
		}
	}
	return nil
}

// adsCampaignName is what the owner will see in their own Ads Manager.
//
// ⚠️ **Named so it is recognisable there, not here.** The restaurant's account
// will hold campaigns made by us and by them, and a row called "Campaign 3" in
// the middle of their own list is one nobody dares pause.
func adsCampaignName(branch, dish string) string {
	parts := []string{"Keel"}
	if branch = strings.TrimSpace(branch); branch != "" {
		parts = append(parts, branch)
	}
	if dish = strings.TrimSpace(dish); dish != "" {
		parts = append(parts, dish)
	}
	parts = append(parts, time.Now().Format("2006-01-02"))
	return strings.Join(parts, " · ")
}

// money renders minor units as the owner reads them.
func money(minor, unit int) string {
	if unit <= 1 {
		return fmt.Sprintf("%d", minor)
	}
	return fmt.Sprintf("%.2f", float64(minor)/float64(unit))
}

// adsLink is where the advert sends a reader, and the domain Meta needs beside
// it.
//
// ⚠️ **`conversion_domain` is a host, never a URL.** Meta requires it on any
// campaign that shares data with a pixel and rejects a full address with a
// message about something else entirely.
func (h *Handler) adsLink(given string) (string, string, error) {
	raw := strings.TrimSpace(given)
	if raw == "" && len(h.Cfg.CORSOrigins) > 0 {
		raw = strings.TrimSuffix(strings.TrimSpace(h.Cfg.CORSOrigins[0]), "/") + "/menu"
	}
	if raw == "" {
		return "", "", errors.New("reklama havolasi ko'rsatilmagan")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", "", errors.New("reklama havolasi noto'g'ri")
	}
	return u.String(), u.Hostname(), nil
}

// adsDishPhoto finds the photograph of the dish being advertised.
func (h *Handler) adsDishPhoto(
	ctx context.Context, id primitive.ObjectID, name string,
) (string, []byte, error) {
	filter := bson.M{}
	switch {
	case !id.IsZero():
		filter["_id"] = id
	case strings.TrimSpace(name) != "":
		filter["name"] = strings.TrimSpace(name)
	default:
		return "", nil, errors.New("qaysi taom reklama qilinishi tanlanmagan")
	}
	var item models.MenuItem
	if err := h.Store.Menu.FindOne(ctx, filter).Decode(&item); err != nil {
		return "", nil, errors.New("taom topilmadi")
	}
	src := strings.TrimSpace(item.ImageURL)
	if src == "" && len(item.Images) > 0 {
		src = strings.TrimSpace(item.Images[0])
	}
	if src == "" {
		// ⚠️ Said plainly instead of generating something: the screen tells the
		// owner to photograph the dish, which is a thing they can do in a
		// minute and which makes the advert true.
		return "", nil, errors.New("bu taomning fotosi yo'q — avval rasm yuklang")
	}
	// Whatever shape the field is in — `/uploads/x.jpg`, a bare name or a full
	// URL — what is on this disk is the last path element.
	file := path.Base(strings.TrimSuffix(src, "/"))
	if q := strings.IndexByte(file, '?'); q >= 0 {
		file = file[:q]
	}
	if file == "" || file == "." || file == "/" {
		return "", nil, errors.New("bu taomning fotosi topilmadi")
	}
	// ⚠️ `OpenRoot`, like the upload server next door: the name comes out of a
	// document, and a document is something somebody could have written a
	// `../../etc` into.
	root, err := os.OpenRoot(h.Cfg.UploadDir)
	if err != nil {
		return "", nil, errors.New("bu taomning fotosi topilmadi")
	}
	defer func() { _ = root.Close() }()
	f, err := root.Open(file)
	if err != nil {
		return "", nil, errors.New("bu taomning fotosi topilmadi")
	}
	defer func() { _ = f.Close() }()
	data := make([]byte, 0, 1<<20)
	buf := make([]byte, 32<<10)
	for {
		n, err := f.Read(buf)
		data = append(data, buf[:n]...)
		if err != nil || len(data) > 8<<20 {
			break
		}
	}
	if len(data) == 0 {
		return "", nil, errors.New("bu taomning fotosi topilmadi")
	}
	// ⚠️ **The name must carry an extension** or Meta rejects the upload with a
	// message that says nothing about the name (docs/vendor/meta-marketing.md
	// §4.4). A stored file without one gets `.jpg`, which is what it is.
	if path.Ext(file) == "" {
		file += ".jpg"
	}
	return file, data, nil
}

// ---- The list, and changing one ----

// AdminAdsCampaigns lists what this restaurant has run.
func (h *Handler) AdminAdsCampaigns(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	sc, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := bson.M{}
	if !sc.BrandID.IsZero() {
		filter["brandId"] = sc.BrandID
	}
	if !sc.BranchID.IsZero() {
		filter["branchId"] = sc.BranchID
	}
	cur, err := h.Store.AdsCampaigns.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "o'qib bo'lmadi")
		return
	}
	var rows []models.AdCampaign
	if err := cur.All(r.Context(), &rows); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "o'qib bo'lmadi")
		return
	}
	s := h.adsSettings(r.Context())
	httpx.JSON(w, http.StatusOK, map[string]any{
		// ⚠️ Empty slice, never nil — `null.map` is the crash this codebase has
		// already shipped twice.
		"campaigns": nonNil(rows),
		"currency":  s.Currency,
		"unit":      meta.MinorUnits(s.Currency),
	})
}

type adsUpdateRequest struct {
	// "start" | "pause" | "stop" | "budget".
	Action string  `json:"action"`
	Daily  float64 `json:"daily"`
}

// AdminAdsUpdateCampaign starts, pauses, stops or re-budgets one campaign.
func (h *Handler) AdminAdsUpdateCampaign(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var req adsUpdateRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req) != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	ctx := r.Context()
	var row models.AdCampaign
	if err := h.Store.AdsCampaigns.FindOne(ctx, bson.M{"_id": id}).Decode(&row); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	client, s, err := h.adsClient(ctx)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	set := bson.M{}
	switch req.Action {
	case "start":
		if err := h.adsSwitch(ctx, client, row, true); err != nil {
			httpx.Error(w, http.StatusBadGateway, err.Error())
			return
		}
		now := time.Now()
		set["status"] = models.AdCampaignActive
		if row.StartedAt == nil {
			set["startedAt"] = now
		}
	case "pause":
		if err := h.adsSwitch(ctx, client, row, false); err != nil {
			httpx.Error(w, http.StatusBadGateway, err.Error())
			return
		}
		set["status"] = models.AdCampaignPaused
	case "stop":
		// ⚠️ Paused at Meta rather than deleted. A deleted campaign takes its
		// own history with it, and the question "what did we spend on osh last
		// month" is asked after the campaign is over, never during it.
		if err := h.adsSwitch(ctx, client, row, false); err != nil {
			httpx.Error(w, http.StatusBadGateway, err.Error())
			return
		}
		now := time.Now()
		set["status"] = models.AdCampaignStopped
		set["stoppedAt"] = now
	case "budget":
		unit := meta.MinorUnits(row.Currency)
		next := int(math.Round(req.Daily * float64(unit)))
		if next <= 0 {
			httpx.Error(w, http.StatusBadRequest, "kunlik byudjet ko'rsatilmagan")
			return
		}
		// ⚠️ **The owner's own ceiling binds the owner too.** It was set as an
		// answer to "how much am I willing to lose in a day", and a screen that
		// let it be exceeded by typing a bigger number in the next field would
		// make it decoration — including for the rules, which read the same
		// field.
		if row.CapMinor > 0 && next > row.CapMinor {
			httpx.Error(w, http.StatusBadRequest,
				"kunlik byudjet o'zingiz qo'ygan chegaradan oshib ketdi")
			return
		}
		if s.MinDailyBudget > 0 && next < s.MinDailyBudget {
			httpx.Error(w, http.StatusBadRequest, fmt.Sprintf(
				"Meta kunlik byudjet uchun eng kami %s %s ni talab qiladi",
				money(s.MinDailyBudget, unit), row.Currency))
			return
		}
		if err := client.SetDailyBudget(ctx, row.AdSetID, next); err != nil {
			h.noteMetaError(ctx, err)
			if e := meta.AsError(err); e != nil {
				httpx.Error(w, http.StatusBadGateway, e.Human())
				return
			}
			httpx.Error(w, http.StatusBadGateway, err.Error())
			return
		}
		set["dailyMinor"] = next
	default:
		httpx.Error(w, http.StatusBadRequest, "noma'lum amal")
		return
	}

	if _, err := h.Store.AdsCampaigns.UpdateByID(ctx, id, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "saqlanmadi")
		return
	}
	h.logAction(r, ActAdsUpdate, "ads", row.MetaCampaignID, row.Name, req.Action)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
