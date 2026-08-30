package menuimport

import "testing"

// The shape a restaurant site builder actually answers, trimmed from a live
// page. ⚠️ Kept verbatim rather than simplified: the whole reason this reader
// exists is that the price is not where anybody would put it.
const iikoWebMenu = `{"result":{"itemCategories":[
 {"id":"c1","name":"Bar","items":[
   {"itemId":"a1","name":"Flat white","description":"","itemSizes":[
     {"sku":"00367","isDefault":true,
      "prices":[{"organizationId":"o1","price":35000.0,"storeId":175408}],
      "buttonImage":{"src":"https://cdn.example.com/items/flat.JPEG",
                     "44x44x100.webp":"https://cdn.example.com/thumb-44.webp"},
      "relatedProducts":[{"name":"Croissant","price":99000}]}]},
   {"itemId":"a2","name":"Американо","itemSizes":[
     {"prices":[{"price":20000.0}]}]}]},
 {"id":"c2","name":"Основные блюда","items":[
   {"itemId":"b1","name":"Лагман","description":"Домашняя лапша","itemSizes":[
     {"prices":[{"price":48000.0}]}]}]}]},"error":null}`

// ⚠️ **This is the page that came back empty**, and it is the ordinary case
// rather than an edge: most restaurant sites built this decade render the menu
// in the browser, so what arrives is a bundle and an empty div. Neither the
// schema.org reader nor a model can do anything with that — the dishes are not
// on the page. They are published exactly, as numbers, one request away.
func TestASinglePageAppsMenuIsReadFromItsOwnJSON(t *testing.T) {
	got := FromJSON([]byte(iikoWebMenu))
	if len(got) != 3 {
		t.Fatalf("got %d dishes, want 3: %+v", len(got), got)
	}
	by := map[string]Dish{}
	for _, d := range got {
		by[d.Name] = d
	}

	flat, ok := by["Flat white"]
	if !ok {
		t.Fatalf("dishes: %+v", got)
	}
	// ⚠️ The price is at itemSizes[] → prices[] → price: five levels down. A
	// depth limit of three found nothing at all here and fell through to the
	// assistant, which is the bug this reader exists to fix.
	if flat.Price != 35000 {
		t.Fatalf("price = %d, want 35000", flat.Price)
	}
	if flat.Category != "Bar" {
		t.Fatalf("category = %q, want Bar — the enclosing section was lost", flat.Category)
	}
	if by["Лагман"].Category != "Основные блюда" {
		t.Fatalf("category = %q", by["Лагман"].Category)
	}
}

// ⚠️ **A related product is not this dish.** It sits inside the same size
// object with its own name and price; following every key would import it, and
// worse, could attach its price to the dish it hangs off — a mistake that reads
// as perfectly ordinary in a list of ninety.
func TestARelatedProductIsNotImportedAsTheDishesPrice(t *testing.T) {
	got := FromJSON([]byte(iikoWebMenu))
	for _, d := range got {
		if d.Name == "Croissant" {
			t.Fatal("a related product was imported as a dish")
		}
		if d.Price == 99000 {
			t.Fatalf("%q took a related product's price", d.Name)
		}
	}
}

// ⚠️ **The original, not the first URL found.** These APIs publish `src` and
// then a row of resized variants keyed by their dimensions, and the first key
// of a Go map is whichever the runtime felt like — so "the first" imports a
// 44-pixel thumbnail for every dish, at random.
func TestTheFullSizePhotographIsPreferredOverAThumbnail(t *testing.T) {
	for _, d := range FromJSON([]byte(iikoWebMenu)) {
		if d.Name != "Flat white" {
			continue
		}
		if d.ImageURL != "https://cdn.example.com/items/flat.JPEG" {
			t.Fatalf("image = %q — a thumbnail was picked", d.ImageURL)
		}
		return
	}
	t.Fatal("the dish disappeared")
}

// A category has a name and a price never; importing it would put "Bar" on the
// menu at nothing so'm.
func TestACategoryIsNotADish(t *testing.T) {
	for _, d := range FromJSON([]byte(iikoWebMenu)) {
		if d.Name == "Bar" || d.Name == "Основные блюда" {
			t.Fatalf("%q was imported as a dish", d.Name)
		}
	}
}

// A framework's own state blob is the whole menu already inside the document we
// downloaded — no second request needed.
func TestAFrameworkStateBlobIsRead(t *testing.T) {
	page := `<html><body><div id="app"></div>
	<script id="__NEXT_DATA__" type="application/json">` + iikoWebMenu + `</script>
	</body></html>`
	if got := FromInline(page); len(got) != 3 {
		t.Fatalf("got %d dishes from the inline blob", len(got))
	}
}
