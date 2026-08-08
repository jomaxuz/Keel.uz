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
		rest.Booking = branch.Booking
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
	if brand != nil && !raw {
		if d := h.publishedDesign(r, brand.ID); d != nil {
			resp["design"] = d
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
	if b.Content != (models.SiteContent{}) {
		rest.Content = b.Content
	}
	if b.Theme != (models.SiteTheme{}) {
		rest.Theme = b.Theme
	}
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
	itemCur, err := h.Store.Menu.Find(r.Context(), scope.brandFilter(bson.M{"isAvailable": true}), itemOpts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var items []models.MenuItem
	if err := itemCur.All(r.Context(), &items); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// What has run out at the branch serving this guest. The dish stays on the
	// menu — it is on the menu tomorrow too — but it cannot be ordered today.
	branch, branchErr := h.bookingBranch(r, r.URL.Query().Get("branchId"))
	if branchErr == nil {
		for i := range items {
			items[i].SoldOut = branch.IsSoldOut(items[i].ID)
		}
	} else {
		branch = nil
	}
	// Combos: what they contain, what the same dishes cost separately, and
	// whether the set can be assembled today.
	h.decorateCombos(r.Context(), items, branch)

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
	branch, err := h.bookingBranch(r, r.URL.Query().Get("branchId"))
	if err == nil {
		item.SoldOut = branch.IsSoldOut(item.ID)
	} else {
		branch = nil
	}
	one := []models.MenuItem{item}
	h.decorateCombos(r.Context(), one, branch)
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
