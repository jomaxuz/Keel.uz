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
