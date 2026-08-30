package menuimport

import (
	"context"
	"encoding/json"
	"net/url"
	"regexp"
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

// FromJSON walks any JSON document and pulls out everything dish-shaped.
func FromJSON(raw []byte) []Dish {
	var doc any
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	var out []Dish
	walkJSON(doc, "", &out)
	return dedupe(out)
}

// walkJSON descends, carrying the nearest enclosing category name.
func walkJSON(node any, section string, out *[]Dish) {
	switch v := node.(type) {
	case []any:
		for _, item := range v {
			walkJSON(item, section, out)
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
			if d, ok := dishFromJSON(v, section); ok {
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
			walkJSON(child, next, out)
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
		for _, key := range []string{"price", "currentPrice", "cost", "amount", "value"} {
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

func dishFromJSON(m map[string]any, section string) (Dish, bool) {
	name := clean(jsonString(m, "name", "title", "productName", "itemName"))
	if name == "" {
		return Dish{}, false
	}
	price := findPrice(m, 0)
	if price <= 0 {
		return Dish{}, false
	}
	return Dish{
		Name:        name,
		Description: clean(jsonString(m, "description", "text", "composition")),
		Price:       price,
		ImageURL:    findImage(m, 0),
		Category:    clean(section),
	}, true
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

func jsonString(m map[string]any, keys ...string) string {
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
