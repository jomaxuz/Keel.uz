package handlers

// ---- Uzum Tezkor: the menu and what is left of it ----
//
// Uzum reads the catalogue once an hour (`composition`) and the stock every
// five minutes (`availability`), per store — one of our branches. See
// docs/vendor/uzum-tezkor-retail.md for the shape.
//
// ⚠️ **The catalogue is a shop's, and a restaurant fills it honestly or not at
// all.** `barcode`, `measure` and `vendorCode` are required. A dish has no
// barcode and no fixed weight, so the fields are filled with what is *true*:
// the dish's own barcode where a shop has one, otherwise its id; the id as the
// article; and an empty measure. An invented "500 g" would be printed under
// the dish on the guest's phone.
//
// ⚠️ **Two endpoints, two jobs.** What a dish *is* changes rarely and is read
// hourly; whether it can be sold right now changes every evening and is read
// every five minutes. So an unavailable or stopped dish stays in the catalogue
// with stock 0 — dropping it from the catalogue would keep it off Uzum for up
// to an hour after the kitchen turned it back on.

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/models"
)

// tezkorOpenStock is the stock of a dish that is on and has no daily limit.
//
// ⚠️ **A kitchen has no count, and Uzum only asks "more than zero?"** Any
// positive number means "orderable"; a large one would read as a warehouse
// figure if Uzum ever shows it, and 1 would read as "last portion".
const tezkorOpenStock = 99

type tezkorImage struct {
	Hash string `json:"hash"`
	URL  string `json:"url"`
}

type tezkorCategory struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	SortOrder int           `json:"sortOrder"`
	Images    []tezkorImage `json:"images,omitempty"`
}

type tezkorBarcode struct {
	Value          string `json:"value"`
	WeightEncoding string `json:"weightEncoding"`
}

type tezkorDescription struct {
	General string `json:"general,omitempty"`
}

type tezkorServiceCodes struct {
	Mxik    string `json:"mxikCodeUz"`
	Package string `json:"packageCodeUz,omitempty"`
}

type tezkorProduct struct {
	ID          string            `json:"id"`
	CategoryID  string            `json:"categoryId"`
	Name        string            `json:"name"`
	Description tezkorDescription `json:"description"`
	Price       float64           `json:"price"`
	OldPrice    *float64          `json:"oldPrice,omitempty"`
	// ⚠️ Never nil: the field is required and `null` is not an empty list.
	Images        []tezkorImage `json:"images"`
	IsCatchWeight bool          `json:"isCatchWeight"`
	// Required, and empty on purpose — see the file comment.
	Measure        struct{}            `json:"measure"`
	Barcode        tezkorBarcode       `json:"barcode"`
	VendorCode     string              `json:"vendorCode"`
	Vat            *int                `json:"vat,omitempty"`
	ServiceCodesUz *tezkorServiceCodes `json:"serviceCodesUz,omitempty"`
	SortOrder      int                 `json:"sortOrder"`
}

type tezkorNomenclature struct {
	Categories []tezkorCategory `json:"categories"`
	Items      []tezkorProduct  `json:"items"`
}

type tezkorStockRow struct {
	ID    string  `json:"id"`
	Stock float64 `json:"stock"`
}

// tezkorSellable reports whether a dish can be ordered through a catalogue that
// has no modifier groups.
//
//   - **A model row** (sizes, colours) is never a thing on the shelf — the same
//     rule `menuLine` and the public menu apply.
//   - ⚠️ **A dish with a required choice** cannot be ordered without one, and
//     Uzum's catalogue has nowhere to ask it. Listed, every order for it would
//     reach the kitchen missing the answer.
//   - **A zero price** is dropped by Uzum anyway; leaving it out keeps the
//     catalogue and what Uzum shows the same list.
func tezkorSellable(it *models.MenuItem) bool {
	if len(it.VariantAxes) > 0 || it.Price <= 0 {
		return false
	}
	for _, o := range it.Options {
		if o.Required {
			return false
		}
	}
	return true
}

// tezkorCatalog builds the catalogue from the menu. `image` resolves a stored
// image URL to Uzum's `{hash, url}`, or reports that it cannot.
//
// ⚠️ **A dish whose category is not in the list is left out**, and so is a
// category with no dish in it. Uzum validates `categoryId` against the
// categories it was sent; a switched-off category's dishes would fail that
// check, and an empty category is a heading with nothing under it.
func tezkorCatalog(
	cats []models.Category,
	items []models.MenuItem,
	image func(string) (tezkorImage, bool),
) tezkorNomenclature {
	active := map[primitive.ObjectID]models.Category{}
	for _, c := range cats {
		if c.IsActive {
			active[c.ID] = c
		}
	}
	out := tezkorNomenclature{Categories: []tezkorCategory{}, Items: []tezkorProduct{}}
	used := map[primitive.ObjectID]bool{}
	for i := range items {
		it := &items[i]
		if !tezkorSellable(it) {
			continue
		}
		if _, ok := active[it.CategoryID]; !ok {
			continue
		}
		used[it.CategoryID] = true
		p := tezkorProduct{
			ID:          it.ID.Hex(),
			CategoryID:  it.CategoryID.Hex(),
			Name:        it.Name,
			Description: tezkorDescription{General: strings.TrimSpace(it.Description)},
			Price:       float64(it.Price),
			Images:      []tezkorImage{},
			Barcode:     tezkorBarcode{Value: it.ID.Hex(), WeightEncoding: "none"},
			VendorCode:  it.ID.Hex(),
			Vat:         it.VatPercent,
			SortOrder:   it.SortOrder,
		}
		if b := strings.TrimSpace(it.Barcode); b != "" {
			p.Barcode.Value = b
		}
		// "Old price" is a promotion only when it is higher; anything else is a
		// stale field, and Uzum would draw it crossed out beside the real one.
		if it.OldPrice != nil && *it.OldPrice > it.Price {
			old := float64(*it.OldPrice)
			p.OldPrice = &old
		}
		// ⚠️ **Only a valid ИКПУ, and none rather than a guess** — the same rule
		// the fiscal receipt follows (DECISIONS → ИКПУ).
		if code := strings.TrimSpace(it.Ikpu); code != "" {
			p.ServiceCodesUz = &tezkorServiceCodes{Mxik: code, Package: strings.TrimSpace(it.PackageCode)}
		}
		seen := map[string]bool{}
		for _, u := range append([]string{it.ImageURL}, it.Images...) {
			if u == "" || seen[u] {
				continue
			}
			seen[u] = true
			if img, ok := image(u); ok {
				p.Images = append(p.Images, img)
			}
		}
		out.Items = append(out.Items, p)
	}
	for _, c := range cats {
		if !c.IsActive || !used[c.ID] {
			continue
		}
		tc := tezkorCategory{ID: c.ID.Hex(), Name: c.Name, SortOrder: c.SortOrder}
		if c.ImageURL != "" {
			if img, ok := image(c.ImageURL); ok {
				tc.Images = []tezkorImage{img}
			}
		}
		out.Categories = append(out.Categories, tc)
	}
	sort.SliceStable(out.Categories, func(i, j int) bool {
		return out.Categories[i].SortOrder < out.Categories[j].SortOrder
	})
	return out
}

// tezkorStock is how many of a dish Uzum may sell right now.
//
// ⚠️ **Every reason we already refuse a sale, and no new one**: switched off in
// the menu, on any of the stop lists (`Branch.IsSoldOut` asks all of them and
// the daily limit), or a combo with a member that ran out — the same checks the
// website's basket makes, so Uzum cannot sell what our own site would refuse.
// A daily limit gives the real number left.
func tezkorStock(
	it *models.MenuItem,
	branch *models.Branch,
	sold map[primitive.ObjectID]int,
) float64 {
	if !it.IsAvailable || branch.IsSoldOut(it.ID) {
		return 0
	}
	for _, m := range it.ComboItems {
		if branch.IsSoldOut(m.MenuItemID) {
			return 0
		}
	}
	if limit := branch.LimitFor(it.ID); limit > 0 {
		left := limit - sold[it.ID]
		if left < 0 {
			left = 0
		}
		return float64(left)
	}
	return tezkorOpenStock
}

// tezkorUploadName turns a stored image URL into a path under the upload
// directory, or reports that it is not one of our files.
//
// ⚠️ **Both shapes are stored**: `/uploads/seed/osh.jpg` from the seed and
// `https://…/uploads/abc.jpg` from the upload handler. And the path is cleaned
// exactly as ServeUploads cleans it — the URL is data from the database, but a
// catalogue that hashed `../../etc/passwd` because a row said so is still a
// catalogue reading outside its directory.
func tezkorUploadName(u string) (string, bool) {
	i := strings.Index(u, "/uploads/")
	if i < 0 {
		return "", false
	}
	name := u[i+len("/uploads/"):]
	if j := strings.IndexAny(name, "?#"); j >= 0 {
		name = name[:j]
	}
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	if name == "" || strings.HasPrefix(name, thumbDir) {
		return "", false
	}
	return name, true
}

// tezkorImageHashes remembers a file's SHA-1 by its name, size and time.
//
// ⚠️ **Hashed from the bytes, as the spec asks** — Uzum reloads the picture when
// the hash changes, so a hash of the URL would never notice a photo replaced
// under the same name. Cached because the catalogue is read hourly per branch
// and the photographs do not change between reads.
var tezkorImageHashes sync.Map // name → tezkorHashEntry

type tezkorHashEntry struct {
	size int64
	mod  time.Time
	hash string
}

func (h *Handler) tezkorImage(root *os.Root, u string) (tezkorImage, bool) {
	name, ok := tezkorUploadName(u)
	if !ok || root == nil {
		// ⚠️ **Not ours, not listed.** A picture on somebody else's server has
		// no bytes we can hash, and an image without its required hash fails
		// Uzum's validation for the whole dish.
		return tezkorImage{}, false
	}
	info, err := root.Stat(name)
	if err != nil || info.IsDir() {
		return tezkorImage{}, false
	}
	url := strings.TrimRight(h.Cfg.PublicBaseURL, "/") + "/uploads/" + name
	if v, ok := tezkorImageHashes.Load(name); ok {
		e := v.(tezkorHashEntry)
		if e.size == info.Size() && e.mod.Equal(info.ModTime()) {
			return tezkorImage{Hash: e.hash, URL: url}, true
		}
	}
	f, err := root.Open(name)
	if err != nil {
		return tezkorImage{}, false
	}
	defer f.Close()
	sum := sha1.New()
	if _, err := io.Copy(sum, f); err != nil {
		return tezkorImage{}, false
	}
	hash := hex.EncodeToString(sum.Sum(nil))
	tezkorImageHashes.Store(name, tezkorHashEntry{size: info.Size(), mod: info.ModTime(), hash: hash})
	return tezkorImage{Hash: hash, URL: url}, true
}

// tezkorStore is the branch a request is about, or a 404 in Uzum's shape.
func (h *Handler) tezkorStore(w http.ResponseWriter, r *http.Request) (*models.Branch, bool) {
	raw := chi.URLParam(r, "storeId")
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(raw))
	if err != nil {
		tezkorFail(w, http.StatusNotFound, tezkorErr{tezkorCodeNotFound, "store " + raw + " was not found"})
		return nil, false
	}
	branch, err := h.branchByIDCtx(r.Context(), id)
	if err != nil || branch == nil {
		tezkorFail(w, http.StatusNotFound, tezkorErr{tezkorCodeNotFound, "store " + raw + " was not found"})
		return nil, false
	}
	return branch, true
}

// tezkorMenu is the branch's brand menu, sorted the way the site sorts it.
func (h *Handler) tezkorMenu(ctx context.Context, branch *models.Branch) ([]models.Category, []models.MenuItem, error) {
	filter := bson.M{}
	if !branch.BrandID.IsZero() {
		filter["brandId"] = branch.BrandID
	}
	sorted := options.Find().SetSort(bson.D{{Key: "sortOrder", Value: 1}})
	cats := []models.Category{}
	cur, err := h.Store.Categories.Find(ctx, filter, sorted)
	if err != nil {
		return nil, nil, err
	}
	if err := cur.All(ctx, &cats); err != nil {
		return nil, nil, err
	}
	items := []models.MenuItem{}
	cur, err = h.Store.Menu.Find(ctx, filter, sorted)
	if err != nil {
		return nil, nil, err
	}
	if err := cur.All(ctx, &items); err != nil {
		return nil, nil, err
	}
	return cats, items, nil
}

// tezkorWrite answers with the content type the spec names as current.
func tezkorWrite(w http.ResponseWriter, contentType string, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		tezkorFail(w, http.StatusInternalServerError, tezkorErr{tezkorCodeInternal, "response could not be built"})
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// TezkorComposition is the catalogue Uzum reads once an hour.
func (h *Handler) TezkorComposition(w http.ResponseWriter, r *http.Request) {
	branch, ok := h.tezkorStore(w, r)
	if !ok {
		return
	}
	cats, items, err := h.tezkorMenu(r.Context(), branch)
	if err != nil {
		tezkorFail(w, http.StatusInternalServerError, tezkorErr{tezkorCodeInternal, "menu could not be read"})
		return
	}
	root, err := os.OpenRoot(h.Cfg.UploadDir)
	if err != nil {
		root = nil
	} else {
		defer root.Close()
	}
	out := tezkorCatalog(cats, items, func(u string) (tezkorImage, bool) {
		return h.tezkorImage(root, u)
	})
	tezkorWrite(w, "application/vnd.eda.picker.nomenclature.v1+json", out)
}

// TezkorAvailability is the stock Uzum reads every five minutes.
//
// ⚠️ **The same dishes as the catalogue, every one of them** — a dish missing
// from this list is unavailable to Uzum, so leaving out a stopped one would be
// the same as zero, and leaving out a live one would hide it.
func (h *Handler) TezkorAvailability(w http.ResponseWriter, r *http.Request) {
	branch, ok := h.tezkorStore(w, r)
	if !ok {
		return
	}
	cats, items, err := h.tezkorMenu(r.Context(), branch)
	if err != nil {
		tezkorFail(w, http.StatusInternalServerError, tezkorErr{tezkorCodeInternal, "menu could not be read"})
		return
	}
	// The sales count costs an aggregation, and almost no branch has a limit.
	sold := map[primitive.ObjectID]int{}
	if len(branch.DailyLimits) > 0 {
		if s, err := h.soldToday(r.Context(), branch.ID); err == nil {
			sold = s
		}
	}
	catalog := tezkorCatalog(cats, items, func(string) (tezkorImage, bool) { return tezkorImage{}, false })
	listed := map[string]bool{}
	for _, p := range catalog.Items {
		listed[p.ID] = true
	}
	rows := []tezkorStockRow{}
	for i := range items {
		it := &items[i]
		if !listed[it.ID.Hex()] {
			continue
		}
		rows = append(rows, tezkorStockRow{ID: it.ID.Hex(), Stock: tezkorStock(it, branch, sold)})
	}
	tezkorWrite(w, "application/vnd.eda.picker.availability.v1+json", map[string]any{"items": rows})
}
