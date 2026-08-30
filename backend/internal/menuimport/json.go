package menuimport

import (
	"context"
	"encoding/json"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// Reading a menu that is not in the HTML at all.
//
// ⚠️ **Most restaurant sites built this decade render their menu in the
// browser.** The page that arrives is an empty shell — an Angular, React or Vue
// bundle and a `<div id="app">` — and the dishes come afterwards from the
// site's own JSON. Neither the schema.org reader nor the assistant can do
// anything with that: there is nothing on the page to read. That is exactly
// what "sahifa bo'sh" meant.
//
// The dishes are still published, exactly, as numbers rather than as sentences.
// Reading them is **free, exact and needs no model** — and reading a price out
// of a JSON field is strictly better than asking anything to read it out of a
// paragraph.

// dishJSON is the shape hunted for: an object with a name and a price.
//
// ⚠️ **Matched by shape, not by schema.** There is no standard here — one site
// answers iiko's web-menu format, another its own — and hard-coding each one is
// a parser that works for the sites we happened to test. What every menu API
// agrees on is that a dish has a name and costs something, and that is enough
// to find them.

// catalog is what the rest of the document knows that one dish does not.
//
// ⚠️ **Both of its fields exist because a menu API states things by
// reference.** A product says `"categories":["b8713c5f-…"]` and `"image":
// "553fb012-…"` — an id and an id, neither of which means anything on its own.
// Nested menus (iiko's, and every schema.org page) need none of this, which is
// why it is optional and nil-safe throughout: the reader that worked yesterday
// keeps working with no catalog at all.
type catalog struct {
	// Category id → its name, gathered from wherever the document keeps them.
	names map[string]string
	// What this site prefixes an image id with. Empty when it never said.
	imageBase string
}

func (c *catalog) name(id string) string {
	if c == nil {
		return ""
	}
	return c.names[id]
}

// image turns a bare id into an address, or returns "" if it cannot.
func (c *catalog) image(id string) string {
	if c == nil || c.imageBase == "" || !looksLikeID(id) {
		return ""
	}
	return c.imageBase + id
}

// looksLikeID keeps this from turning a description into a URL.
//
// ⚠️ A uuid specifically, not "any short string". `"image": "burger.jpg"` is a
// filename we do not know the folder of, and `"image": "1"` is an id into a
// table we cannot see; prefixing either produces a broken image on every card,
// which is worse than no image and much harder to explain.
var uuidOnly = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func looksLikeID(s string) bool { return uuidOnly.MatchString(strings.TrimSpace(s)) }

// FromJSON walks any JSON document and pulls out everything dish-shaped.
func FromJSON(raw []byte) []Dish {
	var doc any
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	// ⚠️ The index is built even here, where most documents are nested and do
	// not need it: a site API that answers categories and products as two
	// sibling lists is exactly as common as one that nests them, and the pass
	// costs nothing on a document that has no ids to gather.
	src := &catalog{names: map[string]string{}}
	indexNames(doc, src)
	var out []Dish
	walkJSON(doc, "", src, &out)
	return dedupe(out)
}

// indexNames gathers "this id is called that" from anywhere in a document.
//
// ⚠️ **Anything with an id and a name that is not itself a dish**, which is
// deliberately loose: these payloads label categories, brands, branches and
// tags in the same shape, and a wrong entry can only ever be reached by a dish
// that named that exact id. A missed entry, by contrast, loses the section for
// every dish under it.
func indexNames(node any, src *catalog) {
	switch v := node.(type) {
	case []any:
		for _, item := range v {
			indexNames(item, src)
		}
	case map[string]any:
		id := jsonString(v, "id", "uuid", "_id")
		name := jsonString(v, "name", "title", "categoryName")
		if id != "" && name != "" && !isDish(v) {
			src.names[id] = clean(name)
		}
		for _, child := range v {
			indexNames(child, src)
		}
	}
}

// walkJSON descends, carrying the nearest enclosing category name.
func walkJSON(node any, section string, src *catalog, out *[]Dish) {
	switch v := node.(type) {
	case []any:
		for _, item := range v {
			walkJSON(item, section, src, out)
		}
	case map[string]any:
		// ⚠️ A container's own name becomes the section for everything below
		// it — but only if the container is not itself a dish, or every dish
		// would be filed under its own name.
		next := section
		if name := jsonString(v, "name", "title", "categoryName"); name != "" &&
			!isDish(v) && hasChildren(v) {
			next = name
		}
		if isDish(v) {
			if d, ok := dishFromJSON(v, section, src); ok {
				*out = append(*out, d)
			}
			// ⚠️ Still descended into: a dish carries its sizes and modifier
			// groups, and a modifier that is itself sold is a dish. Stopping
			// here loses the "add a portion of chips" half of some menus.
		}
		for key, child := range v {
			// ⚠️ **Some branches never hold this menu's own dishes**, and
			// walking them imports things nobody put on the menu. An upsell
			// list (`relatedProducts`) carries whole dishes with their own
			// prices — they belong to somebody else's section and appear there
			// anyway, so importing them here is a duplicate with the wrong
			// category. A modifier group carries the sizes and extras, which
			// are a dish's *options* in this product, not dishes: importing
			// them puts "Katta" on the menu at 5 000 so'm.
			//
			// This was caught by a test rather than in review: `findPrice`
			// already refused to look inside these, so the guard read as
			// complete, and the walk went in through the back door.
			if skipBranch[key] {
				continue
			}
			walkJSON(child, next, src, out)
		}
	}
}

// skipBranch names the keys the walk does not follow. See walkJSON.
var skipBranch = map[string]bool{
	"relatedProducts": true, "related": true, "recommendations": true,
	"recommended": true, "similar": true, "upsell": true, "suggestions": true,
	"modifiers": true, "itemModifierGroups": true, "modifierGroups": true,
	"options": true, "allergens": true, "labels": true, "nutritions": true,
	"tags": true,
}

func hasChildren(m map[string]any) bool {
	for _, v := range m {
		switch v.(type) {
		case []any, map[string]any:
			return true
		}
	}
	return false
}

// isDish decides whether this object is something sold.
//
// ⚠️ **A name and a findable price, and nothing looser.** Loosening it to "has
// a name" imports every category, every modifier group and every city in the
// branch list. The price may be nested — `itemSizes[0].prices[0].price` on an
// iiko web menu — which is why it is looked for rather than read.
func isDish(m map[string]any) bool {
	if jsonString(m, "name", "title", "productName", "itemName") == "" {
		return false
	}
	// A category is a thing with a list of items in it; it is not a dish even
	// when a stray price sits beside it.
	for _, key := range []string{"items", "products", "children", "dishes"} {
		if arr, ok := m[key].([]any); ok && len(arr) > 0 {
			return false
		}
	}
	return findPrice(m, 0) > 0
}

// findPrice looks for a price on this object or inside the containers a price
// is kept in.
//
// ⚠️ **The named keys are the guard, not the depth.** Descending into every key
// would find the price of a *related product* or of a modifier and file it
// against the wrong dish — a mistake that reads as perfectly ordinary in a
// list. So only the containers a dish keeps its own price in are followed, and
// `relatedProducts` is deliberately not among them.
//
// ⚠️ The depth is a recursion guard and nothing more. It was **three**, which
// was wrong: an iiko web menu nests the price at
// `itemSizes[] → prices[] → price`, five levels down — so this reader found
// nothing at all on the most common site builder in the country and fell
// through to the assistant. The bug this file exists to fix, reintroduced by a
// number that looked cautious.
func findPrice(node any, depth int) int {
	if depth > 6 {
		return 0
	}
	switch v := node.(type) {
	case map[string]any:
		for _, key := range []string{
			"price", "currentPrice", "cost", "amount", "value",
			// ⚠️ **`out_price` is what a dish costs on Delever**, which runs a
			// large share of the delivery sites in the country. Without it the
			// reader found a hundred and nineteen named things with no price
			// on them, decided none was a dish, and reported an empty page.
			"out_price", "outPrice", "sale_price", "salePrice",
			"base_price", "basePrice", "new_price", "newPrice",
		} {
			if p := toPrice(v[key]); p > 0 {
				return p
			}
		}
		// The nested containers a price hides in, named rather than walked
		// wholesale: walking every key would reach `relatedProducts`.
		for _, key := range []string{
			"itemSizes", "sizes", "prices", "price", "variants", "modifiers",
		} {
			if p := findPrice(v[key], depth+1); p > 0 {
				return p
			}
		}
	case []any:
		for _, item := range v {
			if p := findPrice(item, depth+1); p > 0 {
				return p
			}
		}
	}
	return 0
}

func dishFromJSON(m map[string]any, section string, src *catalog) (Dish, bool) {
	name := clean(jsonString(m, "name", "title", "productName", "itemName"))
	if name == "" {
		return Dish{}, false
	}
	price := findPrice(m, 0)
	if price <= 0 {
		return Dish{}, false
	}
	image := findImage(m, 0)
	if image == "" {
		// ⚠️ The photograph as an **id**, resolved against the shape this site
		// serves images at. See catalog.image: an id we cannot expand stays
		// empty rather than becoming a broken link on every card.
		image = src.image(jsonString(m, "image", "imageId", "photo", "picture"))
	}
	return Dish{
		Name:        name,
		Description: clean(jsonString(m, "description", "text", "composition")),
		Price:       price,
		ImageURL:    image,
		Category:    clean(sectionOf(m, section, src)),
	}, true
}

// sectionOf answers "which part of the menu is this dish in".
//
// ⚠️ **The enclosing container first, the reference second.** A nested menu
// states it by position — the dish is *inside* "Bar" — and that is both more
// reliable and the only thing available on a schema.org page. A flat payload
// states it by id instead, and has to be looked up. Preferring the id would
// break the nested case for a dish that also happens to carry a stray
// `categoryId`.
func sectionOf(m map[string]any, section string, src *catalog) string {
	if strings.TrimSpace(section) != "" {
		return section
	}
	for _, key := range []string{
		"categoryId", "category_id", "categoryID", "categories", "category",
	} {
		switch v := m[key].(type) {
		case string:
			if name := src.name(v); name != "" {
				return name
			}
			// A category stated by name rather than by id. Taken as it is —
			// this is the shape a hand-written menu JSON uses.
			if !looksLikeID(v) && strings.TrimSpace(v) != "" {
				return v
			}
		case []any:
			// ⚠️ **The first one that resolves, and only the first.** A dish
			// may be filed under several categories; our menu gives it one, and
			// importing it once per category would be the same dish three times
			// on the owner's review screen.
			for _, item := range v {
				if id, ok := item.(string); ok {
					if name := src.name(id); name != "" {
						return name
					}
				}
			}
		case map[string]any:
			if name := jsonString(v, "name", "title"); name != "" {
				return name
			}
		}
	}
	return ""
}

// findImage pulls out the photograph, preferring the original over a thumbnail.
//
// ⚠️ **The original, not the first URL found.** These APIs publish a `src` and
// then a row of resized variants keyed by their dimensions (`44x44x100.webp`),
// and the first key in a Go map is whichever the runtime felt like — so taking
// "the first" imports a 44-pixel thumbnail for every dish, at random.
func findImage(node any, depth int) string {
	if depth > 6 {
		return ""
	}
	switch v := node.(type) {
	case string:
		if looksLikeImageURL(v) {
			return v
		}
	case map[string]any:
		for _, key := range []string{"src", "url", "original", "contentUrl", "large"} {
			if s, ok := v[key].(string); ok && looksLikeImageURL(s) {
				return s
			}
		}
		for _, key := range []string{
			"image", "images", "buttonImage", "photo", "photos", "picture",
			"itemSizes", "sizes",
		} {
			if s := findImage(v[key], depth+1); s != "" {
				return s
			}
		}
	case []any:
		for _, item := range v {
			if s := findImage(item, depth+1); s != "" {
				return s
			}
		}
	}
	return ""
}

var imageExt = regexp.MustCompile(`(?i)\.(jpe?g|png|webp|gif)(\?|$)`)

func looksLikeImageURL(s string) bool {
	return strings.HasPrefix(s, "http") && imageExt.MatchString(s)
}

// jsonString reads the first of these keys that holds usable text.
//
// ⚠️ **A name may be an object, and on an Uzbek platform it usually is.**
// `"title": {"uz": "Kuksi", "ru": "Кукcи", "en": "Kuksi"}` is how every
// multilingual menu in this market states a name — and read as a string it is
// simply absent, which made a hundred and nineteen dishes invisible: no name,
// therefore not a dish, therefore "this page publishes nothing".
func jsonString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		switch v := m[k].(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return v
			}
		case map[string]any:
			if s := localized(v); s != "" {
				return s
			}
		}
	}
	return ""
}

// localized picks one language out of a translated field.
//
// ⚠️ **Uzbek first, then Russian, then English — the panel's own order.** The
// import fills the menu's primary language, and a Tashkent restaurant's menu
// is in Uzbek; falling to Russian matters because half these payloads leave
// `uz` empty and fill `ru` (the live page this was written for does exactly
// that for every description). Anything else non-empty is better than nothing:
// a name in the wrong language can be edited, and a missing one deletes the
// dish from the import.
func localized(m map[string]any) string {
	for _, lang := range []string{"uz", "ru", "en", "uz_latn", "uzLatn", "oz"} {
		if s, ok := m[lang].(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	// ⚠️ Sorted, not "whatever the map yields first". Go randomises map order,
	// so an unsorted fallback would import the same page in a different
	// language on every run — and look like a bug in the site.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// ---- Where the JSON lives ----

// inlineJSON pulls the state blobs a framework leaves in the page.
//
// ⚠️ Next.js and Nuxt both ship the page's data inside the HTML so the browser
// does not have to fetch it again — which means the whole menu is already in
// the document we downloaded, and no second request is needed at all.
var inlineBlobs = []*regexp.Regexp{
	regexp.MustCompile(`(?s)<script[^>]+id="__NEXT_DATA__"[^>]*>(.*?)</script>`),
	regexp.MustCompile(`(?s)window\.__NUXT__\s*=\s*(\{.*?\});?\s*</script>`),
	regexp.MustCompile(`(?s)window\.__INITIAL_STATE__\s*=\s*(\{.*?\});?\s*</script>`),
}

// FromInline reads a menu out of a framework's own state blob.
func FromInline(page string) []Dish {
	for _, re := range inlineBlobs {
		m := re.FindStringSubmatch(page)
		if len(m) < 2 {
			continue
		}
		if out := FromJSON([]byte(m[1])); len(out) > 0 {
			return out
		}
	}
	return nil
}

// menuAPIPaths are the addresses a site's own menu JSON is served from.
//
// ⚠️ **Tried, not guessed at forever.** Each is one request against the site
// the owner asked us to read, and a 404 costs nothing. `/api/v1/menu` is the
// iiko web-menu format that most restaurant site builders in the country are
// built on — myresto.online among them — so it is first.
var menuAPIPaths = []string{
	"/api/v1/menu",
	"/api/menu",
	"/api/v1/products",
	"/api/products",
	"/api/catalog",
	"/api/v1/catalog",
}

// FromSiteAPI asks the site itself for its menu.
//
// ⚠️ **Same address rules as the page.** These are URLs derived from one the
// owner typed, and derived is not safer than typed — `checkPublic` runs on
// every one of them, inside Fetch.
func FromSiteAPI(ctx context.Context, pageURL string) []Dish {
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil
	}
	// The site root, not the page: `/ru/menu` is a route in a single-page app
	// and its API lives at the top.
	base.Path, base.RawQuery, base.Fragment = "", "", ""

	for _, p := range menuAPIPaths {
		body, _, err := Fetch(ctx, base.String()+p)
		if err != nil || len(body) < 64 {
			continue
		}
		if out := FromJSON([]byte(body)); len(out) > 0 {
			return out
		}
	}
	return nil
}
