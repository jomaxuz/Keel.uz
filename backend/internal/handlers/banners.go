package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The banner strip: read by the site, edited by the restaurant.
//
// ⚠️ **Folded into `GET /restaurant` rather than given its own public endpoint.** Every
// page already makes that call, and the render tier is the platform's bottleneck — a
// second round trip per visit would be paid by every guest to buy nothing. The same
// reasoning the page layout follows.

// liveBanners is what the site should show right now, in order.
func (h *Handler) liveBanners(ctx context.Context, brandID primitive.ObjectID) []models.Banner {
	return h.bannersFor(ctx, brandID, models.BannerOnSite)
}

// tillBanners is what a locked monoblock should be showing, in order.
//
// ⚠️ **Read on the lock screen, which is the one screen with nobody signed in.**
// So it is answered from the device's own token and says nothing a passer-by
// could not already see by looking at the monoblock — pictures the restaurant
// chose to put on a screen in its own dining room.
func (h *Handler) tillBanners(ctx context.Context, brandID primitive.ObjectID) []models.Banner {
	return h.bannersFor(ctx, brandID, models.BannerOnTill)
}

// bannersFor is the shared read: one brand, one placement, live right now.
func (h *Handler) bannersFor(
	ctx context.Context, brandID primitive.ObjectID, placement string,
) []models.Banner {
	filter := bson.M{"isActive": true}
	if !brandID.IsZero() {
		// A banner with no brand belongs to all of them: a single-brand restaurant never
		// sets the field, and its banners must not vanish the day a second brand appears.
		filter["$or"] = []bson.M{
			{"brandId": brandID},
			{"brandId": bson.M{"$exists": false}},
			{"brandId": primitive.NilObjectID},
		}
	}
	cur, err := h.Store.Banners.Find(ctx, filter, options.Find().SetSort(bson.D{
		{Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: 1},
	}))
	if err != nil {
		return []models.Banner{}
	}
	var rows []models.Banner
	if err := cur.All(ctx, &rows); err != nil {
		return []models.Banner{}
	}
	now := time.Now()
	// ⚠️ The dates are checked here rather than by a job that switches banners off: a
	// discount that ended at midnight has to stop being advertised at midnight, not at
	// the next time something happens to run.
	//
	// ⚠️ **The placement is filtered here rather than in the query**, so a banner
	// saved before the field existed — every banner in every install today —
	// still counts as a site banner. A Mongo filter on `placement: "site"` would
	// simply not match a document that has no such field, and the home page of
	// every restaurant would empty itself on deploy.
	wantTill := placement == models.BannerOnTill
	out := make([]models.Banner, 0, len(rows))
	for _, b := range rows {
		if b.OnTill() != wantTill {
			continue
		}
		if b.Live(now) {
			out = append(out, b)
		}
	}
	return out
}

// ---- Admin ----

func (h *Handler) AdminListBanners(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := bson.M{}
	if !scope.BrandID.IsZero() {
		filter["$or"] = []bson.M{
			{"brandId": scope.BrandID},
			{"brandId": bson.M{"$exists": false}},
			{"brandId": primitive.NilObjectID},
		}
	}
	cur, err := h.Store.Banners.Find(r.Context(), filter, options.Find().SetSort(bson.D{
		{Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: 1},
	}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	all := []models.Banner{}
	_ = cur.All(r.Context(), &all)
	// ⚠️ **The panel asks for one placement at a time**, because the two are
	// edited in two different sections and a list that mixed them would let an
	// owner reorder the lock screen from the site's editor without noticing.
	// Filtered in Go for the same reason the live read is — a banner saved
	// before this field existed has no `placement` at all, and a Mongo filter
	// would drop every one of them from the site's list.
	wantTill := bannerPlacement(r.URL.Query().Get("placement")) == models.BannerOnTill
	rows := []models.Banner{}
	for _, b := range all {
		if b.OnTill() == wantTill {
			rows = append(rows, b)
		}
	}
	httpx.JSON(w, http.StatusOK, rows)
}

// bannerPlacement reads a placement off the wire.
//
// ⚠️ Anything unrecognised is the site — the zero value, and the direction in
// which a typo shows a banner in the wrong place rather than hiding one nobody
// can then find.
func bannerPlacement(v string) string {
	if strings.TrimSpace(v) == models.BannerOnTill {
		return models.BannerOnTill
	}
	return models.BannerOnSite
}

type bannerRequest struct {
	// "" / "site" / "till" — see models.Banner.Placement.
	Placement string               `json:"placement"`
	ImageURL  string               `json:"imageUrl"`
	Link      string               `json:"link"`
	Title     models.LocalizedText `json:"title"`
	SortOrder int                  `json:"sortOrder"`
	IsActive  *bool                `json:"isActive"`
	StartsAt  *time.Time           `json:"startsAt"`
	EndsAt    *time.Time           `json:"endsAt"`
}

// bannerLinks is where a banner may lead.
//
// ⚠️ An allowlist, plus a dish page. A banner is the most-clicked thing on the home page,
// and "any address" there is a way to send a restaurant's own guests to somebody else's
// site — by accident when a link is pasted wrong, or on purpose by whoever gets the panel
// password next.
func bannerLink(v string) string {
	v = strings.TrimSpace(v)
	switch v {
	case "", "/", "/menu", "/cart", "/bron", "/about", "/filiallar", "/profile":
		return v
	}
	// A single dish: `/menu/<id>`. Checked as an id rather than by pattern, so a path
	// that merely looks right cannot get through.
	if rest, ok := strings.CutPrefix(v, "/menu/"); ok {
		if _, err := objectID(rest); err == nil {
			return v
		}
	}
	return ""
}

func (h *Handler) AdminCreateBanner(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req bannerRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	image := sanitizeCampaignImage(req.ImageURL)
	if image == "" {
		httpx.Error(w, http.StatusBadRequest, "banner rasmi kerak")
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	placement := bannerPlacement(req.Placement)
	b := models.Banner{
		BrandID:   scope.BrandID,
		Placement: placement,
		ImageURL:  image,
		// ⚠️ A till banner never carries a link, whatever was sent. It is shown
		// on a **locked** screen: the only thing a tap there may do is bring up
		// the keypad, and a picture that navigated somewhere would be a way past
		// the lock — small, but a lock with a small way past it is not one.
		Link:      tillOrLink(placement, req.Link),
		Title:     req.Title,
		SortOrder: req.SortOrder,
		IsActive:  active,
		StartsAt:  req.StartsAt,
		EndsAt:    req.EndsAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	res, err := h.Store.Banners.InsertOne(r.Context(), b)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	b.ID, _ = res.InsertedID.(primitive.ObjectID)
	h.logAction(r, ActBannerSave, "banner", b.ID.Hex(), "banner", "")
	httpx.JSON(w, http.StatusOK, b)
}

func (h *Handler) AdminUpdateBanner(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	var req bannerRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := bson.M{"updatedAt": time.Now(), "sortOrder": req.SortOrder, "title": req.Title}
	// ⚠️ An empty image means "keep the one that is there", not "clear it": the form
	// cannot show a file, and a banner with no picture is a banner that renders nothing.
	if img := sanitizeCampaignImage(req.ImageURL); img != "" {
		set["imageUrl"] = img
	}
	set["placement"] = bannerPlacement(req.Placement)
	set["link"] = tillOrLink(bannerPlacement(req.Placement), req.Link)
	if req.IsActive != nil {
		set["isActive"] = *req.IsActive
	}
	set["startsAt"] = req.StartsAt
	set["endsAt"] = req.EndsAt
	if _, err := h.Store.Banners.UpdateByID(r.Context(), id, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActBannerSave, "banner", id.Hex(), "banner", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) AdminDeleteBanner(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	if _, err := h.Store.Banners.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActBannerDelete, "banner", id.Hex(), "banner", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// tillOrLink drops the link on a lock-screen banner — see AdminCreateBanner.
func tillOrLink(placement, link string) string {
	if placement == models.BannerOnTill {
		return ""
	}
	return bannerLink(link)
}
