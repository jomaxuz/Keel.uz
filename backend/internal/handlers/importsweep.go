package handlers

// ---- Clearing up after an import ----
//
// An import downloads a photograph per dish onto the restaurant's own disk.
// Those files outlive the dishes: re-import the same page after editing the
// menu, delete half the dishes, import a second aggregator's listing — and
// every picture from every earlier run is still there, backed up every night,
// referenced by nothing.
//
// # ⚠️ The sweeper only ever considers files it downloaded itself
//
// This is the whole safety design, and it is not a detail. A sweeper that
// scanned `uploads/` and deleted whatever it could not find a reference to
// would eventually delete the restaurant's logo — because "could not find a
// reference" is a claim about how well we enumerated the places a URL can hide,
// and images hide in a page design's free-form settings map and inside custom
// CSS as `url(/uploads/…)`. One missed hiding place is a live photograph gone
// with no way back.
//
// So every file the importer writes is recorded, and the candidate set is that
// record and nothing else. The owner's own uploads are not in it and cannot be
// deleted by this code however wrong the reference check turns out to be.
//
// The reference check is then still made against every collection that could
// mention the name — belt as well as braces, because an owner can perfectly
// well pick an imported photograph as a branch cover.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// importAsset is one file the importer downloaded.
type importAsset struct {
	File string    `bson:"file"`
	At   time.Time `bson:"at"`
}

// ⚠️ **A file written a moment ago is not garbage, it is in flight.** Two
// imports can run in two tabs; one downloads a photograph and is still working
// through its list when the other finishes and sweeps. Without this the second
// deletes the first's picture and the dish is saved pointing at nothing.
const importSweepGrace = 15 * time.Minute

// rememberImportAsset records a downloaded file so it can be cleared up later.
func (h *Handler) rememberImportAsset(ctx context.Context, file string) {
	_, _ = h.Store.ImportAssets.InsertOne(ctx,
		importAsset{File: file, At: time.Now()})
}

// sweepImportAssets deletes downloaded photographs nothing points at any more.
//
// ⚠️ **Errors are swallowed and nothing is reported.** This runs after the
// import has already succeeded; an owner who has just added ninety dishes must
// not be shown a failure about housekeeping. A file that survives is a file
// swept next time.
func (h *Handler) sweepImportAssets(ctx context.Context) (int, error) {
	cutoff := time.Now().Add(-importSweepGrace)
	cur, err := h.Store.ImportAssets.Find(ctx,
		bson.M{"at": bson.M{"$lt": cutoff}})
	if err != nil {
		return 0, err
	}
	var assets []importAsset
	if err := cur.All(ctx, &assets); err != nil {
		return 0, err
	}

	removed := 0
	for _, a := range assets {
		name := filepath.Base(strings.TrimSpace(a.File))
		// ⚠️ `filepath.Base` and nothing else joined to the upload directory. A
		// row saying `../../etc/nginx/nginx.conf` cannot be written through this
		// code today, and a delete built on that assumption is one refactor away
		// from being wrong about it.
		if name == "" || name == "." || name == string(filepath.Separator) {
			continue
		}
		used, err := h.imageIsReferenced(ctx, name)
		if err != nil {
			// Could not answer the question — so do not act on it. A sweep that
			// deletes when the database is unreachable is a sweep that empties
			// the folder on the one day the database is unreachable.
			continue
		}
		if used {
			continue
		}
		path := filepath.Join(h.Cfg.UploadDir, name)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			continue
		}
		// ⚠️ The record goes only once the file is gone, in that order. The
		// other way round loses track of a file that failed to delete, which is
		// how an orphan becomes permanent.
		_, _ = h.Store.ImportAssets.DeleteOne(ctx, bson.M{"file": a.File})
		removed++
	}
	return removed, nil
}

// imageIsReferenced asks whether any document anywhere mentions this filename.
//
// ⚠️ **Matched against the raw document, not against a list of fields.** A
// picture can sit in `menu_item.imageUrl`, in a page design's free-form
// `settings` map, or inside custom CSS as `url(/uploads/x.jpg)` — and a
// field-by-field check would answer "unreferenced" for the last two with
// complete confidence. The filename is a 24-character random string, so a
// substring match over the document has no false positives worth worrying
// about, and a false positive here only means a file survives.
func (h *Handler) imageIsReferenced(ctx context.Context, name string) (bool, error) {
	// The escape is unnecessary for a hex name and present anyway: the day this
	// is called with something else, a regex metacharacter would silently
	// change what "referenced" means.
	rx := bson.M{"$regex": quoteRegex(name)}
	for _, c := range h.imageBearingCollections() {
		if c == nil {
			continue
		}
		found, err := scanForName(ctx, c, rx)
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}

// imageBearingCollections is every place an uploaded picture can end up.
//
// ⚠️ Listed rather than derived, and deliberately generous: a collection left
// out of this list is a photograph deleted while it is on screen. Adding one
// that never holds an image costs a scan of a small collection.
func (h *Handler) imageBearingCollections() []*mongo.Collection {
	s := h.Store
	return []*mongo.Collection{
		s.Menu, s.Categories, s.Restaurant, s.Brands, s.Branches,
		s.Banners, s.Designs, s.DesignPreviews, s.Promotions, s.Vacancies,
	}
}

func scanForName(ctx context.Context, c *mongo.Collection, rx bson.M) (bool, error) {
	// One query per collection, asking Mongo to do the matching over every
	// string field it has — expressed as a regex on the whole document via
	// `$text`-free means: a small collection scanned server-side.
	filter := bson.M{"$or": []bson.M{
		{"imageUrl": rx}, {"images": rx}, {"logoUrl": rx}, {"coverUrl": rx},
		{"image": rx}, {"photo": rx},
	}}
	if n, err := c.CountDocuments(ctx, filter); err != nil {
		return false, err
	} else if n > 0 {
		return true, nil
	}
	// ⚠️ **And then the whole document, because the fields above are not where
	// a design keeps its pictures.** `page_design` holds them inside a
	// free-form settings map and inside custom CSS, under keys this code cannot
	// name. Those collections are a handful of documents each, so reading them
	// and looking at the bytes is both correct and cheap — and being correct is
	// what stops this deleting a background somebody is looking at.
	cur, err := c.Find(ctx, bson.M{})
	if err != nil {
		return false, err
	}
	defer cur.Close(ctx)
	needle, _ := rx["$regex"].(string)
	needle = strings.ReplaceAll(needle, `\`, "")
	for cur.Next(ctx) {
		if strings.Contains(cur.Current.String(), needle) {
			return true, nil
		}
	}
	return false, cur.Err()
}

func quoteRegex(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(`\.+*?()|[]{}^$`, r) {
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
