package handlers

// ---- Importing a menu from a link ----
//
// Typing a menu in is the longest job in setting a restaurant up: a hundred
// dishes, each with a name, a price, a description and a photograph. Almost
// every restaurant already has all of it somewhere — an old site, an
// aggregator's listing, a delivery service's page — and retyping it is why a
// signed customer takes three weeks to go live.
//
// ⚠️ **Two steps, never one.** Preview reads the page and proposes; apply
// writes what the owner ticked. An importer that wrote a hundred and twenty
// dishes into a live menu on one press would be a mistake nobody can undo by
// hand, and mistakes are guaranteed — the input is somebody else's page.

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/images"
	"restaurant-backend/internal/menuimport"
	"restaurant-backend/internal/models"
)

// How many dishes one import may propose.
//
// ⚠️ A ceiling rather than a page limit: a badly parsed page can produce every
// word on it as a dish, and a review screen with nine hundred rows is one the
// owner closes.
const maxImportDishes = 300

// AdminMenuImportPreview reads a page and proposes what is on it.
//
// ⚠️ **Owner only.** This reaches out to the internet from the restaurant's
// server and can propose rewriting the whole menu; it is not a shift manager's
// button.
func (h *Handler) AdminMenuImportPreview(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req struct {
		URL string `json:"url"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	page, final, err := menuimport.Fetch(r.Context(), req.URL)
	if err != nil {
		// The fetcher's own words: they name what happened — a private address,
		// a site that answered 403, a name that does not resolve — and each
		// sends the owner somewhere different.
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// ⚠️ **Four readers, and the assistant is the last of them.**
	//
	// Every step above it reads numbers the site published — exactly, free, and
	// with no model involved. Asking anything to read a price out of a sentence
	// when the same price is sitting in a JSON field is strictly worse and
	// costs money to be worse. The order is by how certain the answer is:
	//
	//  1. schema.org JSON-LD in the page. What puts an aggregator in Google's
	//     results, so aggregators almost always have it.
	//  2. The framework's own state blob (`__NEXT_DATA__`, `__NUXT__`). The
	//     whole menu, already inside the document we downloaded.
	//  3. The site's own menu API. ⚠️ **This is the one that was missing**, and
	//     it is the ordinary case rather than an edge: most restaurant sites
	//     built this decade render in the browser, so the page that arrives is
	//     an empty shell and the dishes come afterwards from JSON. Neither the
	//     schema reader nor a model can do anything with an empty shell —
	//     which is precisely what "sahifa bo'sh" was.
	//  4. The assistant, on the page's text. For a photograph of a menu turned
	//     into a web page, and nothing else.
	dishes := menuimport.FromStructured(page)
	if len(dishes) == 0 {
		dishes = menuimport.FromInline(page)
	}
	if len(dishes) == 0 {
		dishes = menuimport.FromSiteAPI(r.Context(), final)
	}
	guessed := false
	if len(dishes) == 0 {
		dishes, err = h.askPlatformForMenu(r.Context(), menuimport.PageText(page))
		if err != nil {
			httpx.Error(w, http.StatusBadGateway, err.Error())
			return
		}
		guessed = true
	}

	dishes = absoluteImages(dishes, final)
	if len(dishes) > maxImportDishes {
		dishes = dishes[:maxImportDishes]
	}
	if len(dishes) == 0 {
		httpx.Error(w, http.StatusUnprocessableEntity,
			"bu sahifada taomlar topilmadi — menyu sahifasining havolasini bering")
		return
	}

	// What is already on the menu, so the panel can tick the new ones and leave
	// the duplicates alone. ⚠️ Decided here rather than in the browser: the
	// browser has the menu it loaded, which may be a week old in an open tab.
	existing := h.existingDishNames(r.Context(), r)
	type row struct {
		menuimport.Dish
		Exists bool `json:"exists"`
	}
	out := make([]row, 0, len(dishes))
	for _, d := range dishes {
		out = append(out, row{Dish: d, Exists: existing[menuimport.NormalName(d.Name)]})
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"dishes": out, "source": final, "guessed": guessed,
	})
}

// askPlatformForMenu is the fallback for pages that publish nothing structured.
func (h *Handler) askPlatformForMenu(
	ctx context.Context, text string,
) ([]menuimport.Dish, error) {
	if strings.TrimSpace(text) == "" {
		// ⚠️ Says what to do, not what happened. By this point three exact
		// readers have found nothing and the page really is a shell — telling
		// the owner "the page is empty" sends them to check a link that is
		// perfectly correct.
		return nil, errors.New(
			"bu sahifadagi menyu brauzerda chiziladi — menyu ochiq turgan " +
				"sahifaning havolasini bering, yoki taomlarni fayldan import qiling")
	}
	res, err := h.callControlPath(ctx, "/internal/menu-extract",
		map[string]any{"text": text})
	if err != nil {
		if errors.Is(err, ErrNotLinked) {
			// ⚠️ The honest sentence, not a transport error. A self-hosted
			// install has no platform behind it, and the answer is "this page
			// has nothing to read automatically", not "something failed".
			return nil, errors.New(
				"bu sahifada tayyor ma'lumot yo'q, avtomatik o'qish esa yoqilmagan")
		}
		// ⚠️ **A model's own words are not the owner's problem.** Quota
		// messages, billing pages and rate-limit URLs from two providers
		// arrived on this screen verbatim — an owner reading "your credit
		// balance is too low" about somebody else's account has been told
		// something true, useless, and alarming. The assistant is the last of
		// four readers here; when it is unavailable the answer is what to do
		// next.
		return nil, errors.New(
			"avtomatik o'qish hozir ishlamayapti. Menyu sahifasining " +
				"boshqa havolasini sinab ko'ring yoki taomlarni fayldan import qiling")
	}
	if off, _ := res["off"].(bool); off {
		return nil, errors.New("avtomatik o'qish hozircha yoqilmagan")
	}
	if capped, _ := res["capped"].(bool); capped {
		return nil, errors.New("bugungi AI limiti tugadi — ertaga qayta urinib ko'ring")
	}
	raw, _ := res["dishes"].([]any)
	out := make([]menuimport.Dish, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		if strings.TrimSpace(name) == "" {
			continue
		}
		desc, _ := m["description"].(string)
		img, _ := m["imageUrl"].(string)
		cat, _ := m["category"].(string)
		price := 0
		if f, ok := m["price"].(float64); ok {
			price = int(f)
		}
		out = append(out, menuimport.Dish{
			Name: name, Description: desc, Price: price,
			ImageURL: img, Category: cat,
		})
	}
	return out, nil
}

// AdminMenuImportApply writes the dishes the owner ticked.
func (h *Handler) AdminMenuImportApply(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req struct {
		Dishes []menuimport.Dish `json:"dishes"`
		// Whether to fetch the photographs. ⚠️ A choice, because it is the slow
		// half: a hundred images is a hundred requests to somebody else's
		// server, and an owner who only wants the names and prices should not
		// wait for them.
		WithImages bool `json:"withImages"`
		// Whether the dishes go straight onto the site.
		//
		// ⚠️ **Off by default, and it stays off by default.** These prices came
		// off somebody else's page: putting them in front of guests unread
		// sells a dish at whatever that page happened to say, and the argument
		// is at the till with a cashier who has never seen the number. But an
		// owner importing their *own* menu, which is the usual case, then has
		// to open ninety dishes to switch each one on — so the choice is
		// theirs and it is one press.
		Active bool `json:"active"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Dishes) == 0 {
		httpx.Error(w, http.StatusBadRequest, "hech nima tanlanmagan")
		return
	}
	if len(req.Dishes) > maxImportDishes {
		req.Dishes = req.Dishes[:maxImportDishes]
	}

	ctx := r.Context()
	brand := h.importBrand(r)
	created, skipped, images := 0, 0, 0
	now := time.Now()

	// ⚠️ **Read once, compared in memory.** A query per dish is a hundred round
	// trips, and — more to the point — it could not do the comparison this
	// needs: Mongo has no idea that `Lagʻmon` and `Lag'mon` are one dish. See
	// menuimport.NormalName.
	//
	// ⚠️ **The whole brand, not the target category.** The same dish under
	// "Import" and under "Issiq taomlar" is still the same dish, and the
	// category is the field an import is least likely to get right — it comes
	// from somebody else's section headings.
	taken := h.existingDishNames(ctx, r)

	for _, d := range req.Dishes {
		name := strings.TrimSpace(d.Name)
		if name == "" {
			continue
		}
		// ⚠️ **Checked before the category is created**, or a re-import of a
		// page whose every dish is already on the menu leaves a fresh empty
		// section behind each time somebody presses the button.
		//
		// ⚠️ **Checked again here, not only in the preview.** Between the two
		// presses the owner may have imported the same page twice, or added the
		// dish by hand. A duplicate menu is not something anybody deletes a
		// hundred rows of.
		//
		// ⚠️ **And the set is updated as we go**, so one import carrying the
		// same dish twice — a "popular" carousel above the menu it is taken
		// from — inserts it once. The extractor already drops exact repeats;
		// this catches the ones that differ only by price or by apostrophe.
		key := menuimport.NormalName(name)
		if key == "" || taken[key] {
			skipped++
			continue
		}
		taken[key] = true

		catID, err := h.categoryFor(ctx, brand, strings.TrimSpace(d.Category))
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		item := models.MenuItem{
			BrandID:     brand,
			CategoryID:  catID,
			Name:        name,
			Description: strings.TrimSpace(d.Description),
			Price:       d.Price,
			// See the note on the request field: hidden unless the owner said
			// otherwise, because these prices came off somebody else's page.
			IsAvailable: req.Active,
			UpdatedAt:   now,
		}
		if req.WithImages && d.ImageURL != "" {
			if url, err := h.saveRemoteImage(ctx, d.ImageURL); err == nil {
				item.ImageURL = url
				images++
			}
			// A photograph that would not download is not a reason to lose the
			// dish: the name and the price are the part that took an hour.
		}
		if _, err := h.Store.Menu.InsertOne(ctx, item); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		created++
	}

	h.logAction(r, ActMenuCreate, "menu", "", "import",
		"havoladan import: "+strconv.Itoa(created)+" ta taom")

	// ⚠️ **After the menu is written, and detached from the request.** The
	// photographs from earlier runs — of dishes since deleted, of a page
	// imported twice, of a listing replaced by a better one — are on the
	// restaurant's disk referenced by nothing, and every one of them is backed
	// up every night. The owner is not waiting for this and must not be: the
	// import has already succeeded.
	go func() {
		ctx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx), 2*time.Minute)
		defer cancel()
		_, _ = h.sweepImportAssets(ctx)
	}()

	httpx.JSON(w, http.StatusOK, map[string]any{
		"created": created, "skipped": skipped, "images": images,
		// Echoed so the panel says which of the two things just happened rather
		// than printing one sentence and hoping it was the right one.
		"active": req.Active,
	})
}

// categoryFor finds or creates the category a dish said it was in.
//
// ⚠️ **Created rather than dropped.** A menu whose sections are lost is a flat
// list of ninety dishes, which is a worse starting point than no import at all
// — the owner would have to sort them by hand, which is most of the work they
// were saved.
func (h *Handler) categoryFor(
	ctx context.Context, brand primitive.ObjectID, name string,
) (primitive.ObjectID, error) {
	if name == "" {
		name = "Import"
	}
	filter := bson.M{"name": bson.M{"$regex": "^" + regexp.QuoteMeta(name) + "$", "$options": "i"}}
	if !brand.IsZero() {
		filter["brandId"] = brand
	}
	var found models.Category
	if h.Store.Categories.FindOne(ctx, filter).Decode(&found) == nil {
		return found.ID, nil
	}
	cat := models.Category{
		BrandID:  brand,
		Name:     name,
		IsActive: true,
	}
	res, err := h.Store.Categories.InsertOne(ctx, cat)
	if err != nil {
		return primitive.NilObjectID, err
	}
	id, _ := res.InsertedID.(primitive.ObjectID)
	return id, nil
}

// saveRemoteImage downloads one photograph onto this restaurant's own disk.
//
// ⚠️ **Copied, never linked.** Keeping the source URL would mean every dish
// photograph on the site is served by a competitor's CDN: it breaks the day
// they change their paths, it sends them a referrer header for every guest who
// opens the menu, and it is their bandwidth being spent.
func (h *Handler) saveRemoteImage(ctx context.Context, src string) (string, error) {
	data, ext, err := menuimport.FetchImage(ctx, src)
	if err != nil {
		return "", err
	}
	// ⚠️ Shrunk exactly as an uploaded photograph is. An aggregator serves
	// 2000px hero images; a hundred of them is half a gigabyte on the
	// restaurant's disk, backed up every night, for pictures the site never
	// displays above 1600.
	if fitted, _, err := images.Fit(bytes.NewReader(data), 1600); err == nil {
		data = fitted
	}
	if err := os.MkdirAll(h.Cfg.UploadDir, 0o755); err != nil {
		return "", err
	}
	name := randomName() + ext
	if err := os.WriteFile(filepath.Join(h.Cfg.UploadDir, name), data, 0o644); err != nil {
		return "", err
	}
	// ⚠️ **Recorded before it is used, not after.** The record is what makes
	// this file sweepable later; writing it only on success of everything that
	// follows would leave a file on disk that nothing can ever clear up — an
	// orphan the sweeper is forbidden to touch, because it only touches what it
	// knows it wrote.
	h.rememberImportAsset(ctx, name)
	return strings.TrimRight(h.Cfg.PublicBaseURL, "/") + "/uploads/" + name, nil
}

// existingDishNames is what the menu already has, lowercased.
func (h *Handler) existingDishNames(ctx context.Context, r *http.Request) map[string]bool {
	out := map[string]bool{}
	filter := bson.M{}
	if brand := h.importBrand(r); !brand.IsZero() {
		filter["brandId"] = brand
	}
	cur, err := h.Store.Menu.Find(ctx, filter)
	if err != nil {
		return out
	}
	var items []models.MenuItem
	if cur.All(ctx, &items) != nil {
		return out
	}
	for _, m := range items {
		out[menuimport.NormalName(m.Name)] = true
	}
	return out
}

// absoluteImages turns a page's relative image paths into addresses.
//
// ⚠️ Done on the server, against the URL that was actually fetched after
// redirects — a page reached through a shortener resolves its images against
// where it landed, not against what was typed.
func absoluteImages(dishes []menuimport.Dish, base string) []menuimport.Dish {
	b, err := url.Parse(base)
	if err != nil {
		return dishes
	}
	for i := range dishes {
		if dishes[i].ImageURL == "" {
			continue
		}
		if ref, err := b.Parse(dishes[i].ImageURL); err == nil {
			dishes[i].ImageURL = ref.String()
		}
	}
	return dishes
}

// importBrand is whose menu is being written to, under the panel's branch lens.
//
// ⚠️ Read from the request rather than left empty: a chain importing a menu
// while looking at one brand means that brand's menu, and dishes written with
// no brand appear under every one of them.
func (h *Handler) importBrand(r *http.Request) primitive.ObjectID {
	scope, err := h.adminScope(r)
	if err != nil {
		return primitive.NilObjectID
	}
	return h.scopeBrand(r, scope)
}
