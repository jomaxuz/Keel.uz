package menuimport

import (
	"encoding/json"
	"strings"
	"testing"
)

// ---- The page that came back empty, and why ----
//
// ⚠️ **A live import of yamato.delever.uz returned nothing, and the page has a
// hundred and nineteen dishes on it.** Every one of them is in the HTML we
// downloaded. Four separate things had to be wrong at once for that to look
// like an empty page, and each of them is ordinary:
//
//	1. the payload is Next.js **App Router**, so there is no `__NEXT_DATA__`;
//	2. the name is `{"uz": …, "ru": …}`, an object, so no reader saw a name;
//	3. the price is `out_price`, which was not a key we looked for;
//	4. the photograph is a bare uuid, not a URL.
//
// Delever runs a large share of the delivery sites in the country, so each of
// these is pinned separately: they fail independently and three of the four
// fail *silently*, by importing fewer dishes rather than by erroring.

// A page in the shape the live one is, trimmed to three dishes and two
// categories. ⚠️ Kept as the real thing — streamed chunks, escaped quotes,
// sibling category list, ids for images — because every one of those details is
// what broke.
func deleverPage() string {
	part := func(s string) string {
		raw, _ := json.Marshal(s)
		return `<script>self.__next_f.push([1,` + string(raw) + `])</script>`
	}
	head := `<!DOCTYPE html><html><head>` +
		`<link rel="icon" href="https://cdn.delever.uz/delever/848fbd30-ad32-4a1c-ad1a-4075527eedd8"/>` +
		`</head><body><div id="__next"></div>`
	// Split mid-object on purpose: the payload arrives as chunks and a reader
	// that looks at them one at a time finds half a dish.
	whole := `4:["$","div",null,{"pb":{"categories":[` +
		`{"id":"c-hot","slug":"goryachie","title":{"uz":"Issiq taomlar","ru":"Горячие блюда"},"order_no":"0","active":true},` +
		`{"id":"c-sushi","slug":"sushi","title":{"uz":"Sushi","ru":"Суши"},"order_no":"1","active":true}],` +
		`"products":[` +
		`{"id":"p1","slug":"kuksi","out_price":55000,"currency":"UZS","type":"simple",` +
		`"categories":["c-hot"],"image":"3003b140-0534-4924-a0a8-d4fbcc383e43",` +
		`"title":{"uz":"Kuksi","ru":"Кукси"},"description":{"uz":"","ru":"Холодный суп"}},` +
		`{"id":"p2","slug":"sushi-burger","out_price":65000,"currency":"UZS","type":"simple",` +
		`"categories":["c-sushi"],"image":"13816b6b-915c-48b0-9ffb-9be922245975",` +
		`"title":{"uz":"Sushi burger","ru":"Суши бургер"},"description":{"uz":"","ru":""}},` +
		`{"id":"p3","slug":"tovuq","out_price":89000,"currency":"UZS","type":"simple",` +
		`"categories":["c-hot"],"image":"553fb012-a626-48b5-b752-8c88079dcaad",` +
		`"title":{"uz":"Teriyaki sousidagi tovuq filesi","ru":"Курица в соусе терияки"},` +
		`"description":{"uz":"","ru":"Куриное бедро, картошка"}}]}}]` + "\n"
	cut := len(whole) / 2
	return head + part(whole[:cut]) + part(whole[cut:]) + `</body></html>`
}

// ⚠️ **The whole menu is in the page and no reader saw it.** `__NEXT_DATA__` is
// the Pages Router; every Next.js site built since streams instead, and the
// streamed payload is not a JSON document — it is React's flight format with
// the JSON sitting inside it.
func TestAnAppRouterPagesMenuIsRead(t *testing.T) {
	got := FromEmbedded(deleverPage())
	if len(got) != 3 {
		t.Fatalf("got %d dishes, want 3: %+v", len(got), got)
	}
}

// ⚠️ **A chunk boundary falls wherever the framework felt like it**, including
// through the middle of a dish. A reader that parsed each `push` on its own
// would find nothing here and would keep finding nothing intermittently — the
// worst shape of bug, because it would work on the page somebody tested.
func TestTheStreamedChunksAreJoinedBeforeReading(t *testing.T) {
	page := deleverPage()
	if strings.Count(page, "__next_f.push") < 2 {
		t.Fatal("the fixture stopped being split — the test proves nothing")
	}
	if len(FromEmbedded(page)) != 3 {
		t.Fatal("a dish was lost across a chunk boundary")
	}
}

// ⚠️ **The name is an object, and on an Uzbek platform it usually is.** Read as
// a string it is simply absent — so the object has no name, is therefore not a
// dish, and the page "publishes nothing". Uzbek first, then Russian: the import
// fills the menu's primary language.
func TestAMultilingualNameIsRead(t *testing.T) {
	by := byName(t, FromEmbedded(deleverPage()))
	if _, ok := by["Kuksi"]; !ok {
		t.Fatalf("the Uzbek name was not used: %v", names(by))
	}
	// ⚠️ Half these payloads leave `uz` empty and fill `ru` — the live page
	// does exactly that for every description. Falling back matters more than
	// being consistent about the language.
	if got := by["Kuksi"].Description; got != "Холодный суп" {
		t.Fatalf("description = %q — the Russian fallback was not taken", got)
	}
}

// ⚠️ **`out_price` is what a dish costs on this platform.** Without it the
// reader found a hundred and nineteen named things with no price, decided none
// of them was a dish, and reported an empty page — the price key and the name
// shape failing together is why nothing at all came back.
func TestTheOutPriceKeyIsAPrice(t *testing.T) {
	by := byName(t, FromEmbedded(deleverPage()))
	if by["Kuksi"].Price != 55000 {
		t.Fatalf("price = %d, want 55000", by["Kuksi"].Price)
	}
}

// ⚠️ **The category is stated by id, in a sibling list.** The walker's
// "nearest enclosing container" rule is right for a nested menu and finds
// nothing here — every dish would import with no section, leaving the owner to
// sort a hundred and nineteen of them by hand.
func TestACategoryNamedByIdIsResolved(t *testing.T) {
	by := byName(t, FromEmbedded(deleverPage()))
	if got := by["Kuksi"].Category; got != "Issiq taomlar" {
		t.Fatalf("category = %q, want Issiq taomlar", got)
	}
	if got := by["Sushi burger"].Category; got != "Sushi" {
		t.Fatalf("category = %q, want Sushi", got)
	}
}

// ⚠️ **`"image": "3003b140-…"` is an id, not a photograph**, and no amount of
// URL-shaped checking makes it one. The site states the shape itself, in its
// own favicon, so it is read off the page rather than guessed.
func TestAnImageIdIsResolvedAgainstTheSitesOwnCdn(t *testing.T) {
	by := byName(t, FromEmbedded(deleverPage()))
	want := "https://cdn.delever.uz/delever/3003b140-0534-4924-a0a8-d4fbcc383e43"
	if got := by["Kuksi"].ImageURL; got != want {
		t.Fatalf("image = %q, want %q", got, want)
	}
}

// ⚠️ **An id we cannot expand stays empty.** A page that never shows the shape
// leaves us guessing, and a guess is a broken image on every card — worse than
// no image and much harder for an owner to explain.
func TestAnImageIdWithNoKnownBaseIsLeftAlone(t *testing.T) {
	page := strings.Replace(deleverPage(),
		`<link rel="icon" href="https://cdn.delever.uz/delever/848fbd30-ad32-4a1c-ad1a-4075527eedd8"/>`,
		"", 1)
	for _, d := range FromEmbedded(page) {
		if d.ImageURL != "" {
			t.Fatalf("%q got the invented address %q", d.Name, d.ImageURL)
		}
	}
}

// ⚠️ **The sweep must not return the same dish once per nesting level.** The
// payload is one big value with dishes several layers down; resuming the scan
// inside a value it had already parsed would yield each dish a dozen times and
// re-parse the page once per dish.
func TestTheSweepDoesNotReturnNestedValuesTwice(t *testing.T) {
	got := FromEmbedded(deleverPage())
	seen := map[string]int{}
	for _, d := range got {
		seen[d.Name]++
	}
	for name, n := range seen {
		if n != 1 {
			t.Fatalf("%q imported %d times", name, n)
		}
	}
}

// A page with no embedded JSON is not an error — it is the ordinary answer for
// most of the sites this reader is tried on.
func TestAPlainPageYieldsNothing(t *testing.T) {
	if got := FromEmbedded(`<html><body><h1>Menyu</h1><p>Lag'mon 32 000</p></body></html>`); got != nil {
		t.Fatalf("got %+v from a page with no JSON in it", got)
	}
}

func byName(t *testing.T, dishes []Dish) map[string]Dish {
	t.Helper()
	out := map[string]Dish{}
	for _, d := range dishes {
		out[d.Name] = d
	}
	return out
}

func names(by map[string]Dish) []string {
	out := make([]string, 0, len(by))
	for n := range by {
		out = append(out, n)
	}
	return out
}
