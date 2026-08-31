package menuimport

import (
	"net/url"
	"testing"
)

// ⚠️ **This is the bug that made a working link look broken.** The path segment
// and the `placeSlug` parameter are different words for the same restaurant,
// and the API answers only to the second: `sam_plov` returns 47 dishes,
// `sam_plov_restaurant` returns 404. Reading the path first gives an empty menu
// for a link that is perfectly good, and the owner is told their page has
// nothing on it while looking at a screen full of dishes.
func TestPlaceSlugWinsOverThePath(t *testing.T) {
	u, _ := url.Parse(
		"https://eats.yandex.com/en-uz/tashkent/r/sam_plov_restaurant?placeSlug=sam_plov")
	if got := yandexSlug(u); got != "sam_plov" {
		t.Fatalf("slug %q, want sam_plov", got)
	}
}

// Without the parameter the path is all there is, and `/r/<slug>` is where it
// sits — not the last segment, which is a city on some of these addresses.
func TestSlugFallsBackToThePathAfterR(t *testing.T) {
	u, _ := url.Parse("https://eda.yandex.ru/moscow/r/some_place")
	if got := yandexSlug(u); got != "some_place" {
		t.Fatalf("slug %q, want some_place", got)
	}
}

func TestOnlyYandexEatsHostsMatch(t *testing.T) {
	yes := []string{
		"https://eats.yandex.com/en-uz/tashkent/r/x",
		"https://eda.yandex.ru/moscow/r/x",
	}
	no := []string{
		// ⚠️ Other Yandex properties must not match: a reader that fires on
		// every yandex.* host spends a request per import on pages it cannot
		// read, and returns nothing on all of them.
		"https://market.yandex.ru/x",
		"https://yandex.uz/maps/x",
		"https://example.com/eats.yandex.com/r/x",
	}
	for _, raw := range yes {
		if AggregatorName(raw) != "Yandex Eats" {
			t.Errorf("%s tanilmadi", raw)
		}
	}
	for _, raw := range no {
		if n := AggregatorName(raw); n != "" {
			t.Errorf("%s noto'g'ri %q deb tanildi", raw, n)
		}
	}
}

// ⚠️ The picture is a template, not a URL. Stored unfilled it is a broken image
// on every dish, and it breaks after the import, inside the menu.
func TestThePictureTemplateIsFilledIn(t *testing.T) {
	got := yandexImage("/images/207/abc-{w}x{h}.jpeg", "https://eats.yandex.com")
	want := "https://eats.yandex.com/images/207/abc-600x600.jpeg"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if yandexImage("", "https://eats.yandex.com") != "" {
		t.Fatal("bo'sh rasm manzilga aylandi")
	}
}

func TestTheMenuDocumentIsRead(t *testing.T) {
	body := `{"payload":{"categories":[
	  {"name":"Sho'rvalar","items":[
	    {"name":"Mastava","description":"","price":50000,
	     "picture":{"uri":"/images/1/a-{w}x{h}.jpeg"}},
	    {"name":"  ","price":1000}
	  ]},
	  {"name":"Ichimliklar","items":[{"name":"Cola","price":13000}]}
	]}}`
	got := yandexDishes(body, "https://eats.yandex.com")
	if len(got) != 2 {
		t.Fatalf("%d ta taom, 2 kutilgan: %+v", len(got), got)
	}
	if got[0].Name != "Mastava" || got[0].Price != 50000 ||
		got[0].Category != "Sho'rvalar" {
		t.Fatalf("birinchi taom: %+v", got[0])
	}
	if got[0].ImageURL != "https://eats.yandex.com/images/1/a-600x600.jpeg" {
		t.Fatalf("rasm: %q", got[0].ImageURL)
	}
	// ⚠️ A nameless row is dropped rather than imported blank: a menu item
	// with no name is a line the owner cannot identify to delete.
	if got[1].Name != "Cola" {
		t.Fatalf("nomsiz qator tushib qolmadi: %+v", got)
	}
}

// ⚠️ Nothing rather than an empty list, so the handler's "no dishes" branch
// runs and the owner gets a sentence instead of a screen with zero rows.
func TestAnEmptyDocumentReadsAsNothing(t *testing.T) {
	if d := yandexDishes(`{"payload":{"categories":[]}}`, "x"); d != nil {
		t.Fatalf("bo'sh hujjat %d ta taom berdi", len(d))
	}
	if d := yandexDishes("not json", "x"); d != nil {
		t.Fatal("JSON bo'lmagan javob taom berdi")
	}
}

// ---- Uzum Tezkor ----

// ⚠️ **The single loudest number in this file.** Their catalogue is in tiyin:
// a pita box comes back as 7 500 000, which is 75 000 so'm. Imported as it
// stands it is a seven-and-a-half-million-so'm pita, and nobody catches it
// while skimming a hundred rows — every price is wrong by the same factor, so
// they look consistent with one another. It surfaces at the till, in front of
// a guest.
func TestUzumPricesAreTiyinAndOursAreNot(t *testing.T) {
	if got := uzumPrice(7500000); got != 75000 {
		t.Fatalf("7500000 tiyin -> %d, 75000 kutilgan", got)
	}
	if got := uzumPrice(400000); got != 4000 {
		t.Fatalf("400000 tiyin -> %d, 4000 kutilgan", got)
	}
}

func TestUzumCatalogueIsRead(t *testing.T) {
	body := `{"categories":[{"id":"106","name":"Roll"},{"id":"247","name":"Kombo"}],
	  "products":[
	    {"id":"1","name":"Donar-Pita box","description":"Pita","categories":["247"],
	     "price":{"value":7500000,"currency":"UZS"},
	     "images":[{"default":"https://cdn.uzumtezkor.uz/images/a.jpg"}]},
	    {"id":"2","name":"","price":{"value":100000}},
	    {"id":"3","name":"Roll","categories":["106"],"price":{"value":4000000}}
	  ]}`
	got := uzumDishes(body)
	if len(got) != 2 {
		t.Fatalf("%d ta taom, 2 kutilgan: %+v", len(got), got)
	}
	if got[0].Price != 75000 {
		t.Fatalf("narx %d, 75000 kutilgan", got[0].Price)
	}
	// ⚠️ The product carries a category *id*; without the lookup every dish
	// would be filed under a number the owner has never seen.
	if got[0].Category != "Kombo" || got[1].Category != "Roll" {
		t.Fatalf("kategoriyalar: %q, %q", got[0].Category, got[1].Category)
	}
	if got[0].ImageURL != "https://cdn.uzumtezkor.uz/images/a.jpg" {
		t.Fatalf("rasm: %q", got[0].ImageURL)
	}
}

// ⚠️ The catalogue answers 401 without the anonymous token the site hands every
// visitor inside `__NEXT_DATA__`. Losing it is not an error anywhere — it is a
// menu that reads as empty.
func TestTheGuestTokenIsReadOutOfThePage(t *testing.T) {
	page := `<html><body><script id="__NEXT_DATA__" type="application/json">` +
		`{"props":{"pageProps":{"authorized":false,"accessToken":"ey.abc.123"}}}` +
		`</script></body></html>`
	if got := nextDataAccessToken(page); got != "ey.abc.123" {
		t.Fatalf("token %q", got)
	}
	if nextDataAccessToken("<html>no next data</html>") != "" {
		t.Fatal("yo'q token topildi")
	}
}

// ⚠️ Their API refuses the ordinary browser header `uz,ru;q=0.9,en;q=0.8` with
// 422 "should be one of [ru en uz]". The language comes from the address, so
// the menu arrives in the language the owner was reading.
func TestTheLocaleComesFromTheAddress(t *testing.T) {
	for path, want := range map[string]string{
		"/ru/restaurants/x": "ru",
		"/uz/restaurants/x": "uz",
		"/en/restaurants/x": "en",
		"/restaurants/x":    "uz",
	} {
		u, _ := url.Parse("https://www.uzumtezkor.uz" + path)
		if got := uzumLocale(u); got != want {
			t.Errorf("%s -> %q, %q kutilgan", path, got, want)
		}
	}
}

// ⚠️ uzum.uz and uzumtezkor.uz are two front doors to one service and behave
// nothing alike: the first is behind a captcha and cannot be read at all.
// Matching the wrong one would spend a request and return nothing.
func TestUzumTezkorHostsMatchAndUzumUzDoesNot(t *testing.T) {
	if AggregatorName("https://www.uzumtezkor.uz/ru/restaurants/abc") != "Uzum Tezkor" {
		t.Error("uzumtezkor.uz tanilmadi")
	}
	if n := AggregatorName("https://www.uzum.uz/uz/tezkor"); n != "" {
		t.Errorf("uzum.uz %q deb tanildi", n)
	}
}
