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

// AdminMenuImportPreview starts reading a page and proposing what is on it.
//
// ⚠️ **A job, like apply, and for the same reason.** Fetching a page is twenty
// seconds at the outside and the assistant is another sixty — against a handler
// the router allows thirty. Whenever the assistant was needed the connection
// was cut and the owner got a gateway error, which reads as "the link is wrong"
// about a link that is fine.
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

	// Read on the request, while there is one: the worker outlives it.
	existing := h.existingDishNames(r.Context(), r)
	url := req.URL

	// ⚠️ The total is the reader count, not a dish count — nothing is known
	// about the page yet. The bar moves once per reader tried, which is what
	// there is to report honestly at this stage.
	readers := menuimport.Readers()
	job := jobs.start("reading", len(readers)+1)

	runImportJob(r.Context(), job, func(ctx context.Context) (map[string]any, error) {
		page, final, err := menuimport.Fetch(ctx, url)

		// ⚠️ **A challenge address is not the restaurant's address.** Yandex
		// answers our server's page request with a redirect to
		// `/showcaptcha?...` — and every reader is then handed *that* as the
		// page's URL. The slug it carries is "showcaptcha", the menu API 404s
		// on it, and the owner is told the site blocks us. The link they typed
		// was fine, and it is the only address here that means anything.
		//
		// The page itself goes too: a challenge page is not this restaurant's
		// page, and reading dishes out of it is not a thing that can succeed.
		walled := err == nil && menuimport.BotWall(page, final)
		if walled {
			page, final = "", url
		}

		var dishes []menuimport.Dish
		usedReader := ""

		// ⚠️ **The aggregator is tried even when the page could not be read at
		// all**, and this is not optimism — it is the case in front of us.
		// Yandex challenges this server's request for the *page* while
		// answering the same server's request to its *menu API* with 200. The
		// menu the owner asked for is available; only the shell around it is
		// not. Locally neither is blocked, which is why this failed on the
		// server and nowhere else.
		if walled || err != nil {
			if found, _ := menuimport.FromAggregator(ctx, page, url); len(found) > 0 {
				dishes, usedReader = found, menuimport.ReaderAggregator
				jobs.update(job.ID, func(j *ImportJob) {
					j.Done = j.Total - 1
					j.Stage = menuimport.ReaderAggregator
				})
			}
		}
		if len(dishes) == 0 && err != nil {
			// The fetcher's own words: they name what happened — a private
			// address, a site that answered 403, a name that does not resolve —
			// and each sends the owner somewhere different.
			return nil, err
		}

		if len(dishes) == 0 {
			for _, reader := range readers {
				jobs.update(job.ID, func(j *ImportJob) {
					j.Done++
					j.Stage = reader.ID
				})
				if found := reader.Read(ctx, page, final); len(found) > 0 {
					dishes, usedReader = found, reader.ID
					break
				}
			}
		}
		if len(dishes) == 0 && walled {
			// ⚠️ **Before the model, not after.** A challenge page is real
			// text, so the model would be called, paid for, and would answer
			// honestly that there are no dishes in "Siz robot emasmisiz?" —
			// and the owner would be told their menu page has no menu on it.
			// Nothing about that names the one thing they can act on.
			return nil, errors.New(
				"bu sayt avtomatik so'rovlarni bloklaydi (robot tekshiruvi), " +
					"shuning uchun sahifani server o'qiy olmaydi — boshqa havola " +
					"yordam bermaydi. Menyuni fayldan import qiling")
		}
		if len(dishes) == 0 {
			jobs.update(job.ID, func(j *ImportJob) {
				j.Done = j.Total - 1
				j.Stage = menuimport.ReaderText
			})
			if site := menuimport.AggregatorName(final); site != "" {
				// ⚠️ The host is known and answered with nothing. That is not
				// "no reader matched" — it is a fact about this restaurant's
				// listing, and the owner is the one person who can check it.
				return nil, errors.New(site +
					" bu restoran uchun menyu qaytarmadi — havolada to'g'ri " +
					"restoran ochilganini tekshiring")
			}
			dishes, err = h.askPlatformForMenu(ctx, menuimport.PageText(page))
			if err != nil {
				return nil, err
			}
			usedReader = menuimport.ReaderText
		}
		jobs.update(job.ID, func(j *ImportJob) { j.Done = j.Total })

		dishes = absoluteImages(dishes, final)
		if len(dishes) > maxImportDishes {
			dishes = dishes[:maxImportDishes]
		}
		if len(dishes) == 0 {
			return nil, errors.New(
				"bu sahifada taomlar topilmadi — menyu sahifasining havolasini bering")
		}

		type row struct {
			menuimport.Dish
			Exists bool `json:"exists"`
		}
		out := make([]row, 0, len(dishes))
		for _, d := range dishes {
			out = append(out, row{
				Dish:   d,
				Exists: existing[menuimport.NormalName(d.Name)],
			})
		}
		return map[string]any{
			"dishes": out, "source": final,
			"reader":      usedReader,
			"readerLabel": menuimport.ReaderLabel(usedReader),
			"guessed":     usedReader == menuimport.ReaderText,
		}, nil
	})

	httpx.JSON(w, http.StatusAccepted, map[string]any{"jobId": job.ID})
}

// askPlatformForMenu is the fallback for pages that publish nothing structured.
func (h *Handler) askPlatformForMenu(
	ctx context.Context, text string,
) ([]menuimport.Dish, error) {
	if strings.TrimSpace(text) == "" {
		// ⚠️ **Names what was tried, not just what to do next.** By this point
		// three exact readers have found nothing and the page really is a
		// shell. Saying only "the page is empty" sends the owner to check a
		// link that is perfectly correct; listing the four readers tells them
		// the tool did look, and — when this message appears on a page that
		// obviously does have a menu — tells whoever they forward it to that
		// the server is running an older build than the one that reads it.
		names := make([]string, 0, 4)
		for _, r := range menuimport.Readers() {
			names = append(names, r.Label)
		}
		return nil, errors.New(
			"bu sahifada menyu topilmadi. Tekshirildi: " +
				strings.Join(names, ", ") + ". Menyu brauzerda chizilsa, " +
				"menyu ochiq turgan sahifaning havolasini bering yoki " +
				"taomlarni fayldan import qiling")
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

// AdminMenuImportApply starts writing the dishes the owner ticked.
//
// ⚠️ **Returns a job id, not a result, and that is the fix for the 502.**
// Ninety dishes with photographs is ninety requests to somebody else's server;
// the router allows a handler thirty seconds and the edge allows less, so the
// connection was cut while the import was still running — the owner saw a
// gateway error and the menu filled up anyway. Pressing the button again would
// then have imported everything twice.
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

	// ⚠️ Everything the worker needs is read **here**, on the request, while
	// there is still a request to read it from. `brandForWrite` and the branch
	// lens come off the URL and the admin's record; a goroutine holding the
	// `*http.Request` after the handler returns is reading a value the server
	// is free to reuse.
	brand := h.importBrand(r)
	taken := h.existingDishNames(r.Context(), r)
	by := "import"
	if u, err := h.adminUser(r); err == nil {
		by = u.Name
	}

	job := jobs.start("dishes", len(req.Dishes))
	dishes, withImages, active := req.Dishes, req.WithImages, req.Active

	runImportJob(r.Context(), job, func(ctx context.Context) (map[string]any, error) {
		created, skipped, images := 0, 0, 0
		now := time.Now()

		for _, d := range dishes {
			// ⚠️ Counted before the work, not after: a dish skipped as a
			// duplicate still moves the bar, or an import of a page already on
			// the menu would sit at 0% and look hung.
			jobs.update(job.ID, func(j *ImportJob) { j.Done++ })

			name := strings.TrimSpace(d.Name)
			if name == "" {
				continue
			}
			key := menuimport.NormalName(name)
			if key == "" || taken[key] {
				skipped++
				continue
			}
			taken[key] = true

			catID, err := h.categoryFor(ctx, brand, strings.TrimSpace(d.Category))
			if err != nil {
				return nil, err
			}

			item := models.MenuItem{
				BrandID:     brand,
				CategoryID:  catID,
				Name:        name,
				Description: strings.TrimSpace(d.Description),
				Price:       d.Price,
				// See the note on the request field: hidden unless the owner
				// said otherwise, because these prices came off somebody
				// else's page.
				IsAvailable: active,
				UpdatedAt:   now,
			}
			if withImages && d.ImageURL != "" {
				if url, err := h.saveRemoteImage(ctx, d.ImageURL); err == nil {
					item.ImageURL = url
					images++
				}
				// A photograph that would not download is not a reason to lose
				// the dish: the name and the price are the part that took an
				// hour.
			}
			if _, err := h.Store.Menu.InsertOne(ctx, item); err != nil {
				return nil, err
			}
			created++
		}

		// ⚠️ **After the menu is written.** The photographs from earlier runs —
		// of dishes since deleted, of a page imported twice — are on the
		// restaurant's disk referenced by nothing, and every one is backed up
		// nightly.
		_, _ = h.sweepImportAssets(ctx)

		h.logActionAs(ctx, by, ActMenuCreate, "menu", "", "import",
			"havoladan import: "+strconv.Itoa(created)+" ta taom")
		return map[string]any{
			"created": created, "skipped": skipped, "images": images,
			// Echoed so the panel says which of the two things just happened
			// rather than printing one sentence and hoping it was the right one.
			"active": active,
		}, nil
	})

	httpx.JSON(w, http.StatusAccepted, map[string]any{
		"jobId": job.ID, "total": len(dishes),
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
