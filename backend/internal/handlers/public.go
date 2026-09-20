package handlers

import (
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetRestaurant returns the company profile with a computed isOpenNow flag.
//
// Hours, address and phones belong to a branch now, so the answer is given for
// one: the branch asked for by ?branchId=, otherwise the brand's first active
// one. The company document still carries the currency and the socials, and —
// on a single-branch install — reads exactly as it always did.
func (h *Handler) GetRestaurant(w http.ResponseWriter, r *http.Request) {
	var rest models.Restaurant
	err := h.Store.Restaurant.FindOne(r.Context(), bson.M{}).Decode(&rest)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "restaurant not configured")
		return
	}
	// The plan's slices are arrays even when nothing has been drawn — see
	// bookingSlices. This is the company document's copy; the branch's own is
	// normalised where it is layered on below and in AdminListBranches.
	rest.Booking = bookingSlices(rest.Booking)
	resp := map[string]any{"restaurant": rest}

	// ?raw=1 skips the merge. The admin settings page needs the company document
	// as stored, because it edits company fields and branch fields separately —
	// handed the merged view, it would save a branch's hours onto the company.
	raw := r.URL.Query().Get("raw") == "1"

	brand, _ := h.publicBrand(r)
	if branch, err := h.bookingBranch(r, r.URL.Query().Get("branchId")); err == nil && !raw {
		// The site reads the profile through the branch it is being served from.
		rest.Address = branch.Address
		rest.Phones = branch.Phones
		rest.WorkingHours = branch.WorkingHours
		rest.Delivery = branch.Delivery
		rest.Booking = bookingSlices(branch.Booking)
		// Normalised on the way out, so the checkout's slot picker reads real
		// numbers rather than having to know the defaults itself — a second copy
		// of those would be a second answer to "how far ahead can I order?".
		rest.Preorder = preorderSettings(branch.Preorder)
		resp["restaurant"] = rest
		resp["branch"] = branch
	}
	if brand != nil {
		if !raw {
			// The brand is the face the guest sees: its name, its logo, its copy
			// and its colours. Overlaid here rather than in every page, so a
			// single-brand install — whose brand carries the same values the
			// company does — renders byte-for-byte what it always did.
			applyBrand(&rest, brand)
			resp["restaurant"] = rest
		}
		resp["brand"] = brand
	}
	// The layout, when one has been drawn for this brand.
	//
	// Folded into this response rather than given its own endpoint: every page
	// already calls it, and the render tier is the platform's bottleneck — a
	// second round trip per page would be paid by every visitor to buy nothing.
	// Absent when no design exists, which is how the site knows to render the
	// template it always did.
	if brand != nil {
		d := h.publishedDesign(r, brand.ID)
		// ⚠️ **The draft, but only for somebody holding a live preview token.**
		//
		// The console's editor needs to show the real site with the unpublished
		// design applied — a schematic preview cannot answer "does this look like
		// the picture the customer sent us", which is the whole job. But a draft
		// visible to a visitor would mean a half-drawn page served to a
		// restaurant's guests, so the draft is behind a token that is short-lived,
		// created by the console, and stored in this tenant's own database (see
		// designPreview). No shared secret, and nothing new the tenant container
		// can reach.
		if preview := h.previewDesign(r, brand.ID); preview != nil {
			d = preview
		}
		if d != nil && !raw {
			resp["design"] = d
			// ⚠️ **The palette the design was drawn with, applied.**
			//
			// It was stored and read by nobody. The console has written
			// `page_design.theme` since the constructor shipped; the site took
			// its colours from the brand, so a design drawn around a yellow
			// panel arrived on an orange site and looked like the drawing was
			// ignored.
			//
			// The half that made it worse: publishing a design sets
			// `designLocked`, which switches the owner's own theme editor off —
			// so a tenant with a drawn design had **nobody** who could change
			// its palette. The console could not, because nothing read what it
			// wrote; the owner could not, because the lock is the whole point.
			//
			// ⚠️ **Field by field, not the whole struct.** A design that only
			// rearranges bands carries an empty theme, and overwriting with it
			// would repaint a restaurant that spent an afternoon choosing its
			// accent — the same rule `applyBrand` follows one function down,
			// and the reason it is written out rather than assigned.
			if d.Theme != nil && !raw {
				rest.Theme = mergeTheme(rest.Theme, *d.Theme)
				resp["restaurant"] = rest
			}
		}
		// ⚠️ Answered even on `?raw=1`, which is what the settings page asks for.
		// The page has to know whether to lock its theme editor, and locking is
		// the whole point: a design somebody paid for must not be undone by an
		// owner nudging the accent colour. Same reasoning as `hideWatermark` —
		// what the business model rests on does not live behind the customer's
		// own switch.
		resp["designLocked"] = d != nil
		// ⚠️ The banner strip, in the same response as everything else the page needs.
		// A second public endpoint would be a second round trip per visit, paid by every
		// guest — the render tier is the platform's bottleneck. Same reasoning as the
		// layout above.
		if !raw {
			resp["banners"] = h.liveBanners(r.Context(), brand.ID)
		}
	}

	// The guests' own words, when the restaurant has switched them on.
	//
	// Folded into this response rather than given an endpoint of its own, for
	// the reason the banners and the layout are: every page already calls this,
	// and a second round trip per visit is paid by every guest to buy nothing.
	// Absent — not empty — when the feature is off, so the section draws
	// nothing rather than a heading over a blank.
	if !raw {
		var branchID any
		if b, ok := resp["branch"].(*models.Branch); ok {
			branchID = b.ID
		}
		if rv := h.publicReviewsFor(r.Context(), &rest, branchID); rv != nil {
			resp["reviews"] = rv
		}
	}
	resp["isOpenNow"] = isOpenNow(rest.WorkingHours, time.Now())
	httpx.JSON(w, http.StatusOK, resp)
}

// publishedDesign loads a brand's layout, or nil when there is none to render.
//
// ⚠️ **Sanitised on every read, not on write.** The document is written by the
// console, which is a different codebase on a different deploy schedule — and a
// hand-edited document, a half-applied migration and an older console are all
// real. `Sanitize` drops what it does not recognise, so an unknown band cannot
// reach the page.
//
// A draft is never returned: an operator mid-layout must not be showing a
// half-drawn page to the restaurant's guests.
func (h *Handler) publishedDesign(r *http.Request, brandID primitive.ObjectID) *models.PageDesign {
	var d models.PageDesign
	err := h.Store.Designs.FindOne(r.Context(), bson.M{
		"brandId": brandID,
		"status":  models.DesignPublished,
	}).Decode(&d)
	if err != nil {
		return nil
	}
	d.Sanitize()
	if !d.Renderable() {
		// Published but empty after sanitising: fall back to the template rather
		// than serving a blank page.
		return nil
	}
	return &d
}

// applyBrand lays a brand's identity over the company profile. Only non-empty
// fields win: a brand that has never been given its own logo keeps showing the
// company's rather than showing none.
func applyBrand(rest *models.Restaurant, b *models.Brand) {
	if b.Name != "" {
		rest.Name = b.Name
	}
	if b.Description != "" {
		rest.Description = b.Description
	}
	if b.LogoURL != "" {
		rest.LogoURL = b.LogoURL
	}
	if b.CoverURL != "" {
		rest.CoverURL = b.CoverURL
	}
	if !b.Content.IsEmpty() {
		rest.Content = b.Content
	}
	if b.Theme != (models.SiteTheme{}) {
		rest.Theme = b.Theme
	}
}

// mergeTheme lays a drawn design's palette over the one the site already has.
//
// ⚠️ **Only the fields the design actually set.** A layout that rearranges
// bands and chooses no colours carries an empty theme, and assigning that
// wholesale would strip the accent, the corner radius and the font pairing a
// restaurant picked by hand — a change nobody asked for, made by a document
// about something else. Same rule, same reason, as `applyBrand` above.
func mergeTheme(base, over models.SiteTheme) models.SiteTheme {
	str := func(dst *string, v string) {
		if v != "" {
			*dst = v
		}
	}
	str(&base.Brand, over.Brand)
	str(&base.BrandDark, over.BrandDark)
	str(&base.Accent, over.Accent)
	str(&base.ButtonShape, over.ButtonShape)
	str(&base.Font, over.Font)
	str(&base.Background, over.Background)
	str(&base.Shadow, over.Shadow)
	str(&base.ButtonStyle, over.ButtonStyle)
	// ⚠️ Pointers, because 0 is a real answer for both: a radius of 0 is square
	// corners and a scale of 0 would be nonsense, so "unset" has to be
	// distinguishable from "zero" — which is why they are pointers in the model.
	if over.Radius != nil {
		base.Radius = over.Radius
	}
	if over.Scale != nil {
		base.Scale = over.Scale
	}
	return base
}

// GetCategories returns active categories of one brand, sorted by sortOrder.
func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.publicScope(r)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	opts := options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}})
	cur, err := h.Store.Categories.Find(r.Context(), scope.brandFilter(bson.M{"isActive": true}), opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var cats []models.Category
	if err := cur.All(r.Context(), &cats); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cats == nil {
		cats = []models.Category{}
	}
	httpx.JSON(w, http.StatusOK, cats)
}

// GetMenu returns one brand's available menu items grouped by category.
func (h *Handler) GetMenu(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.publicScope(r)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	catOpts := options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}})
	catCur, err := h.Store.Categories.Find(r.Context(), scope.brandFilter(bson.M{"isActive": true}), catOpts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var cats []models.Category
	if err := catCur.All(r.Context(), &cats); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	itemOpts := options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}})
	// ⚠️ **A model row is not on this list, because it cannot be sold.** A shirt
	// that comes in five sizes has a row of its own to hang the photograph, the
	// name and the category on; what is counted, scanned and carried out of the
	// shop is always a size, and `menuLine` refuses the model outright. Left in
	// the catalogue it would be a card on the website and a tile on the till
	// that answers every tap with a refusal — which is worse than an absent
	// tile, because somebody tries it a second time.
	//
	// ⚠️ Refused **here**, in the one query the website, the till and the floor
	// screen all read the menu from, rather than in each of them.
	menuFilter := scope.brandFilter(bson.M{"isAvailable": true})
	menuFilter["variantAxes"] = bson.M{"$in": []any{nil, []any{}}}
	itemCur, err := h.Store.Menu.Find(r.Context(), menuFilter, itemOpts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var items []models.MenuItem
	if err := itemCur.All(r.Context(), &items); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// What has run out for this guest. The dish stays on the menu — it is on
	// the menu tomorrow too — but it cannot be ordered today.
	//
	// ⚠️ Which kitchen's list that is depends on whether one has been chosen
	// yet; on a multi-branch install being browsed without an address it is
	// "stopped everywhere". See soldoutlens.go — the old code used whichever
	// branch sorted first, which hid dishes the company could deliver and
	// offered dishes the guest's own branch had run out of.
	soldOut, _ := h.publicSoldOut(r, scope.BrandID)
	if soldOut != nil {
		for i := range items {
			items[i].SoldOut = soldOut(items[i].ID)
		}
	}
	// Combos: what they contain, what the same dishes cost separately, and
	// whether the set can be assembled today.
	h.decorateCombos(r.Context(), items, soldOut)

	type group struct {
		Category models.Category   `json:"category"`
		Items    []models.MenuItem `json:"items"`
	}
	groups := make([]group, 0, len(cats))
	for _, c := range cats {
		g := group{Category: c, Items: []models.MenuItem{}}
		for _, it := range items {
			if it.CategoryID == c.ID {
				g.Items = append(g.Items, it)
			}
		}
		groups = append(groups, g)
	}
	httpx.JSON(w, http.StatusOK, groups)
}

// GetMenuItem returns a single menu item by id.
func (h *Handler) GetMenuItem(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var item models.MenuItem
	if err := h.Store.Menu.FindOne(r.Context(), bson.M{"_id": id}).Decode(&item); err != nil {
		httpx.Error(w, http.StatusNotFound, "not found")
		return
	}
	// The same lens the list uses, so a dish cannot read "available" on the
	// menu and "sold out" on its own page.
	var brandID primitive.ObjectID
	if brand, err := h.publicBrand(r); err == nil && brand != nil {
		brandID = brand.ID
	}
	soldOut, _ := h.publicSoldOut(r, brandID)
	if soldOut != nil {
		item.SoldOut = soldOut(item.ID)
	}
	one := []models.MenuItem{item}
	h.decorateCombos(r.Context(), one, soldOut)
	httpx.JSON(w, http.StatusOK, one[0])
}

// isOpenNow checks the working hours for the current weekday/time.
func isOpenNow(hours []models.WorkingHour, now time.Time) bool {
	day := int(now.Weekday())
	cur := now.Format("15:04")
	for _, h := range hours {
		if h.Day != day || h.IsClosed {
			continue
		}
		if h.Close < h.Open { // overnight (e.g. 18:00 - 02:00)
			return cur >= h.Open || cur <= h.Close
		}
		return cur >= h.Open && cur <= h.Close
	}
	return false
}
