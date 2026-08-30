package menuimport

import "testing"

// ⚠️ **This is where an imported menu quietly becomes free.** Uzbek sites write
// forty-five thousand four different ways, and two of them are a decimal point
// somewhere else in the world. Parsing "45.000" as a float gives 45 — which
// imports as forty-five so'm, a plausible number in a box that nobody reads as
// wrong until a guest orders at it.
func TestForthyFiveThousandIsForthyFiveThousandHoweverItIsWritten(t *testing.T) {
	for _, s := range []any{
		"45000", "45 000", "45,000", "45.000", "45 000 so'm", "45 000 сум",
		"UZS 45000", 45000.0,
	} {
		if got := toPrice(s); got != 45000 {
			t.Fatalf("toPrice(%v) = %d, want 45000", s, got)
		}
	}
	// A currency's minor unit at the end is not part of the figure.
	if got := toPrice("45000.00"); got != 45000 {
		t.Fatalf("toPrice(45000.00) = %d, want 45000", got)
	}
	// And nothing readable is zero, not a guess.
	if got := toPrice("narxi kelishilgan"); got != 0 {
		t.Fatalf("a price that is not there came back as %d", got)
	}
}

const menuPage = `<html><head>
<script type="application/ld+json">
{"@context":"https://schema.org","@type":"Menu","hasMenuSection":[
 {"@type":"MenuSection","name":"Salatlar","hasMenuItem":[
   {"@type":"MenuItem","name":"Achichuk","description":"Pomidor, piyoz",
    "offers":{"@type":"Offer","price":"18 000","priceCurrency":"UZS"},
    "image":"/img/achichuk.jpg"}]},
 {"@type":"MenuSection","name":"Issiq taomlar","hasMenuItem":[
   {"@type":"MenuItem","name":"Lag'mon","offers":{"price":"45.000"}},
   {"@type":"MenuItem","name":"Somsa"}]}]}
</script></head><body>Menyu</body></html>`

func TestASchemaMenuIsReadExactly(t *testing.T) {
	got := FromStructured(menuPage)
	if len(got) != 3 {
		t.Fatalf("got %d dishes, want 3: %+v", len(got), got)
	}

	byName := map[string]Dish{}
	for _, d := range got {
		byName[d.Name] = d
	}
	if d := byName["Achichuk"]; d.Price != 18000 || d.Category != "Salatlar" {
		t.Fatalf("achichuk = %+v", d)
	}
	if d := byName["Lag'mon"]; d.Price != 45000 {
		t.Fatalf("lag'mon price = %d — a dotted thousand separator was read as a decimal", d.Price)
	}
	// ⚠️ **A dish with no price is kept.** Plenty of pages list the name and
	// put the price in an element the JSON-LD does not carry; dropping those
	// loses the names, descriptions and photographs — the parts that take an
	// hour — over the field that takes five seconds.
	if _, ok := byName["Somsa"]; !ok {
		t.Fatal("a dish with no price was dropped")
	}
	// ⚠️ And a section is not a dish. Taking one would put "Salatlar" on the
	// menu at nothing so'm.
	if _, bad := byName["Salatlar"]; bad {
		t.Fatal("a menu section was imported as a dish")
	}
}

// The same dish listed twice — a "popular" carousel above the menu it is taken
// from is the usual reason — is one dish, not two rows to delete by hand.
func TestTheSameDishTwiceIsOneDish(t *testing.T) {
	page := `<script type="application/ld+json">
	[{"@type":"MenuItem","name":"Somsa","offers":{"price":"12000"}},
	 {"@type":"MenuItem","name":"Somsa","offers":{"price":"12000"}}]</script>`
	if got := FromStructured(page); len(got) != 1 {
		t.Fatalf("got %d, want 1: %+v", len(got), got)
	}
}

// ⚠️ A lazy-loaded page keeps the real address in `data-src` and a grey square
// in `src`. Reading `src` imports the same placeholder as every dish's photo.
func TestPageTextKeepsRealImagesAndDropsFurniture(t *testing.T) {
	page := `<body>
	  <img src="/logo.svg" alt="Logo">
	  <div>Lag'mon 45 000</div>
	  <img src="/placeholder.gif" data-src="https://cdn.example.com/lagmon.jpg">
	  <script>var x = 1;</script>
	</body>`
	txt := PageText(page)
	if !contains(txt, "https://cdn.example.com/lagmon.jpg") {
		t.Fatalf("the lazy-loaded photograph was lost:\n%s", txt)
	}
	if contains(txt, "logo.svg") || contains(txt, "placeholder") {
		t.Fatalf("furniture reached the model:\n%s", txt)
	}
	if contains(txt, "var x") {
		t.Fatalf("script source reached the model:\n%s", txt)
	}
	if !contains(txt, "Lag'mon") {
		t.Fatalf("the dish itself was lost:\n%s", txt)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
