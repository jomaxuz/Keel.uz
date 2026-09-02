// Command importmenu writes a restaurant's real menu — categories, dishes,
// their size and flavour options, and the photographs — from one JSON file.
//
//	docker compose exec backend /app/importmenu \
//	    -db t_sanseb -spec /app/uploads/import/menu.json
//
// ⚠️ **The photographs travel with the file rather than being embedded**, and
// the folder they sit in is the tenant's own uploads volume: that directory is
// already mounted into the container, so a menu can be handed over with `scp`
// and imported without adding a mount, rebuilding an image or opening a port.
// `-images` defaults to the spec's own folder for exactly that reason.
//
// ⚠️ **Photographs are converted like any upload** (internal/images): the file
// that lands on disk is WebP, named after its contents. A menu imported here
// and a menu typed into the panel produce the same bytes — an importer that
// wrote the originals would leave one restaurant's photographs three times the
// size of everybody else's, and nobody would ever look for the reason.
//
// ⚠️ **Nothing is deleted.** Re-running adds; a category with the same name is
// reused rather than duplicated, and a dish with the same name inside it is
// updated in place. That makes a half-finished import safe to repeat, which is
// the state an import is usually in when something goes wrong.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"restaurant-backend/internal/config"
	"restaurant-backend/internal/db"
	"restaurant-backend/internal/images"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// spec is the file an import is written as.
//
// ⚠️ **Russian names go in `nameRu` *and* in `name`.** The base name is what
// every language falls back to, so a menu written only in Russian shows Russian
// everywhere — which is the honest outcome, and better than a Uzbek column full
// of Russian words pretending to be a translation.
type spec struct {
	Categories []specCategory `json:"categories"`
}

type specCategory struct {
	Name  string     `json:"name"`
	Items []specItem `json:"items"`
}

type specItem struct {
	Name string `json:"name"`
	// The price of the cheapest option, which is what the menu card shows.
	Price int `json:"price"`
	// The photograph, as a filename inside the images folder.
	Image       string       `json:"image"`
	Description string       `json:"description"`
	Options     []specOption `json:"options"`
}

type specOption struct {
	Name string `json:"name"`
	// ⚠️ Required by default: a size or a flavour is not a topping. A cheesecake
	// sold by the slice and by the whole tin has to be answered before it can
	// be put in a basket, or the till prints "one cheesecake" and the kitchen
	// guesses.
	Optional bool         `json:"optional"`
	Multiple bool         `json:"multiple"`
	Choices  []specChoice `json:"choices"`
}

type specChoice struct {
	Name string `json:"name"`
	// Added to the dish price. The first choice is normally 0 — see Price.
	Delta int `json:"delta"`
}

func main() {
	dbName := flag.String("db", "", "database name (default: MONGO_DB)")
	specPath := flag.String("spec", "", "path to the menu JSON")
	imagesDir := flag.String("images", "", "folder with the photographs (default: the spec's folder)")
	dry := flag.Bool("dry", false, "read and report, write nothing")
	flag.Parse()

	if *specPath == "" {
		log.Fatal("-spec is required")
	}
	raw, err := os.ReadFile(*specPath)
	if err != nil {
		log.Fatalf("spec: %v", err)
	}
	var s spec
	if err := json.Unmarshal(raw, &s); err != nil {
		log.Fatalf("spec: %v", err)
	}
	dir := *imagesDir
	if dir == "" {
		dir = filepath.Dir(*specPath)
	}

	cfg := config.Load()
	if *dbName != "" {
		cfg.MongoDB = *dbName
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	database, err := db.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("mongo: %v", err)
	}
	store := repository.New(database)

	// ⚠️ **The brand, because a dish without one is invisible.** Every menu
	// screen filters by the brand it is showing, so an item written with an
	// empty brandId is in the database and on nobody's screen — the failure is
	// silent and looks like the import having done nothing at all.
	brandID, brandName, err := onlyBrand(ctx, store)
	if err != nil {
		log.Fatalf("brand: %v", err)
	}

	if err := images.Available(); err != nil {
		log.Printf("⚠ webp encoder unavailable (%v) — photographs are stored as they came", err)
	}

	fmt.Printf("database %q, brand %q, %d categories\n", cfg.MongoDB, brandName, len(s.Categories))
	if *dry {
		for _, c := range s.Categories {
			fmt.Printf("  %s (%d)\n", c.Name, len(c.Items))
			for _, it := range c.Items {
				fmt.Printf("    %-34s %9d  %s\n", it.Name, it.Price, it.Image)
			}
		}
		fmt.Println("dry run: nothing written")
		return
	}

	added, updated := 0, 0
	for ci, c := range s.Categories {
		catID, err := upsertCategory(ctx, store, brandID, c.Name, ci)
		if err != nil {
			log.Fatalf("category %q: %v", c.Name, err)
		}
		for ii, it := range c.Items {
			url, err := storeImage(cfg.UploadDir, dir, it.Image)
			if err != nil {
				// ⚠️ Not fatal: a missing photograph is a dish without a
				// picture, and stopping the whole import over one file would
				// leave the menu half written.
				log.Printf("  %s: %v", it.Name, err)
			}
			isNew, err := upsertItem(ctx, store, brandID, catID, it, url, ii)
			if err != nil {
				log.Fatalf("item %q: %v", it.Name, err)
			}
			if isNew {
				added++
			} else {
				updated++
			}
		}
		fmt.Printf("  %-28s %d ta\n", c.Name, len(c.Items))
	}
	fmt.Printf("done: %d added, %d updated\n", added, updated)
}

// onlyBrand is the brand this menu belongs to.
//
// ⚠️ Refuses when there is more than one rather than picking: a chain's second
// brand is a different menu, and guessing would put somebody's cheesecakes on
// the wrong company's website.
func onlyBrand(ctx context.Context, store *repository.Store) (id any, name string, err error) {
	cur, err := store.Brands.Find(ctx, bson.M{})
	if err != nil {
		return nil, "", err
	}
	var brands []models.Brand
	if err := cur.All(ctx, &brands); err != nil {
		return nil, "", err
	}
	switch len(brands) {
	case 0:
		return nil, "", fmt.Errorf("this database has no brand")
	case 1:
		return brands[0].ID, brands[0].Name, nil
	default:
		names := make([]string, 0, len(brands))
		for _, b := range brands {
			names = append(names, b.Name)
		}
		return nil, "", fmt.Errorf("more than one brand (%s) — import one menu at a time",
			strings.Join(names, ", "))
	}
}

func upsertCategory(ctx context.Context, store *repository.Store, brandID any, name string, sort int) (any, error) {
	filter := bson.M{"brandId": brandID, "name": name}
	var existing models.Category
	err := store.Categories.FindOne(ctx, filter).Decode(&existing)
	if err == nil {
		return existing.ID, nil
	}
	if err != mongo.ErrNoDocuments {
		return nil, err
	}
	res, err := store.Categories.InsertOne(ctx, bson.M{
		"brandId": brandID,
		"name":    name,
		// The same words in every language — see the note on spec.
		"nameRu":    name,
		"nameEn":    "",
		"slug":      slugify(name),
		"sortOrder": sort,
		"isActive":  true,
		"imageUrl":  "",
	})
	if err != nil {
		return nil, err
	}
	return res.InsertedID, nil
}

func upsertItem(
	ctx context.Context, store *repository.Store,
	brandID, catID any, it specItem, imageURL string, sort int,
) (bool, error) {
	opts := make([]bson.M, 0, len(it.Options))
	for _, o := range it.Options {
		choices := make([]bson.M, 0, len(o.Choices))
		for _, ch := range o.Choices {
			choices = append(choices, bson.M{
				"name": ch.Name, "nameRu": ch.Name, "nameEn": "",
				"priceDelta": ch.Delta,
			})
		}
		opts = append(opts, bson.M{
			"name": o.Name, "nameRu": o.Name, "nameEn": "",
			"required": !o.Optional,
			"multiple": o.Multiple,
			"choices":  choices,
		})
	}

	set := bson.M{
		"brandId":    brandID,
		"categoryId": catID,
		"name":       it.Name,
		"nameRu":     it.Name,
		"nameEn":     "",
		"price":      it.Price,
		"options":    opts,
		"sortOrder":  sort,
	}
	if it.Description != "" {
		set["description"] = it.Description
		set["descriptionRu"] = it.Description
	}
	// ⚠️ Only when there is one: a re-run without the photographs must not
	// erase the pictures a previous run uploaded.
	if imageURL != "" {
		set["imageUrl"] = imageURL
	}

	filter := bson.M{"brandId": brandID, "name": it.Name}
	res, err := store.Menu.UpdateOne(ctx, filter, bson.M{
		"$set": set,
		"$setOnInsert": bson.M{
			"isAvailable": true,
			"images":      []string{},
			"tags":        []string{},
		},
	}, options.Update().SetUpsert(true))
	if err != nil {
		return false, err
	}
	return res.UpsertedCount > 0, nil
}

// storeImage converts one photograph and writes it into the uploads directory,
// returning the URL the menu item points at.
func storeImage(uploadDir, imagesDir, name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", nil
	}
	src, err := os.Open(filepath.Join(imagesDir, name))
	if err != nil {
		return "", fmt.Errorf("photograph %q: %w", name, err)
	}
	defer src.Close()

	// The same 1600 px bound the upload endpoint applies, so an imported menu
	// and a typed one weigh the same.
	data, contentType, err := images.Fit(src, 1600)
	if err != nil {
		// Unsupported (an animation, or not an image): store it untouched, the
		// way the upload endpoint does.
		if _, err := src.Seek(0, 0); err != nil {
			return "", err
		}
		raw, rerr := os.ReadFile(filepath.Join(imagesDir, name))
		if rerr != nil {
			return "", rerr
		}
		data = raw
		contentType = ""
	}

	// ⚠️ **A name derived from the source, not a random one.** Re-running the
	// import then overwrites the same file instead of leaving the previous
	// upload behind as an orphan nobody can identify.
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	out := "import-" + slugify(base) + images.ExtFor(contentType)
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(uploadDir, out), data, 0o644); err != nil {
		return "", err
	}
	return "/uploads/" + out, nil
}

// slugify makes a filename- and URL-safe token out of a name.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	dash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case r >= 'а' && r <= 'я':
			b.WriteRune(translit(r))
			dash = false
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// translit is enough Cyrillic to make a readable slug — this is a filename and
// a URL fragment, not a transliteration anybody reads as text.
//
// ⚠️ **Runes, not bytes, and the first version got this wrong and panicked.**
// `strings.IndexRune` answers in *bytes*, and a Cyrillic letter is two of them
// — so indexing the Latin table with that offset walked off the end on the
// first category name. Found by running the import against a scratch database,
// which is the whole reason this command has a `-dry` flag and a rehearsal.
var translitTable = map[rune]rune{}

func init() {
	from := []rune("абвгдежзийклмнопрстуфхцчшщъыьэюя")
	to := []rune("abvgdejziyklmnoprstufhccssiiieua")
	for i, r := range from {
		translitTable[r] = to[i]
	}
}

func translit(r rune) rune {
	if v, ok := translitTable[r]; ok {
		return v
	}
	return '-'
}
