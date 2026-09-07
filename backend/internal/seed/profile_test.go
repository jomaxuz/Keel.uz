package seed

import (
	"strings"
	"testing"

	"restaurant-backend/internal/config"
	"restaurant-backend/internal/models"
)

// What a brand-new install calls itself.
//
// ⚠️ **"My Restaurant" was not a placeholder anybody saw and dismissed.** The
// brand and the branch are both created from this profile
// (repository.EnsureBrandAndBranch), so the installer's word travelled to the
// website's header, the receipt's first line, the Telegram messages and the
// till's paper — and it said "restaurant" to shops. It is the sort of fault that
// nothing reports: every screen works, and the name is simply wrong.

func TestTheConsolesNameIsWhatTheInstallStartsWith(t *testing.T) {
	name, _ := firstProfile(&config.Config{BrandName: "B5 Somsa"})
	if name != "B5 Somsa" {
		t.Errorf("named itself %q instead of what the console was told", name)
	}
}

// ⚠️ **The fallback is the kind of thing it is, not a restaurant.** A tenant
// provisioned by hand or by an older console still has to start as something,
// and "Do'kon" is wrong for nobody who runs a shop.
func TestAnUnnamedShopIsNotCalledARestaurant(t *testing.T) {
	for _, biz := range []string{"grocery", "pharmacy", "clothing", "flowers"} {
		name, desc := firstProfile(&config.Config{BusinessType: biz})
		if strings.Contains(strings.ToLower(name), "restoran") {
			t.Errorf("%s starts life as %q", biz, name)
		}
		// The description is printed on the shop's own home page until it is
		// rewritten, and "milliy va zamonaviy taomlar" over a pharmacy is the
		// same mistake as the name.
		if strings.Contains(desc, "taomlar") {
			t.Errorf("%s is described as a kitchen: %q", biz, desc)
		}
		if desc == "" {
			t.Errorf("%s has nothing on its home page", biz)
		}
	}
}

func TestAnUnnamedRestaurantStillReadsAsOne(t *testing.T) {
	name, desc := firstProfile(&config.Config{})
	if name != "Restoran" {
		t.Errorf("an unnamed restaurant is called %q", name)
	}
	if !strings.Contains(desc, "taomlar") {
		t.Errorf("a restaurant is not described as one: %q", desc)
	}
}

// ⚠️ **A name is never invented over one the console gave.** An operator who
// typed "Dorixona №7" gets that, business type or not.
// ⚠️ **Each kind of shop is described as that kind of shop.** "Kundalik
// mahsulotlar" over a boutique is the smaller version of the same mistake, and
// this sentence is the first one a customer reads on the site.
func TestEachShopIsDescribedAsItself(t *testing.T) {
	seen := map[string]string{}
	for _, biz := range models.BusinessTypes {
		if !biz.SellsGoods() {
			continue
		}
		_, desc := firstProfile(&config.Config{BusinessType: string(biz)})
		if prev, dup := seen[desc]; dup {
			t.Errorf("%s and %s share one description", biz, prev)
		}
		seen[desc] = string(biz)
	}
}

func TestTheGivenNameWinsOverTheTemplate(t *testing.T) {
	name, _ := firstProfile(&config.Config{
		BrandName: "Dorixona №7", BusinessType: "pharmacy",
	})
	if name != "Dorixona №7" {
		t.Errorf("the console's name was overwritten with %q", name)
	}
}

// ⚠️ **The restaurant's demo menu is never written into a shop.** Forty-eight
// dishes in a pharmacy read as somebody else's shop, and the owner's first task
// becomes deleting them one at a time.
func TestAShopGetsItsOwnCatalogueAndNotTheMenu(t *testing.T) {
	src := source(t, "menu.go")
	fn := between(t, src, "func ensureMenu(", "\n}\n")
	if !strings.Contains(fn, "SellsGoods()") {
		t.Fatal("the demo menu is written into a shop's catalogue")
	}
	shop := strings.Index(fn, "writeShopCatalogue")
	menu := strings.Index(fn, "WriteMenu(")
	if shop < 0 || menu < 0 || shop > menu {
		t.Error("a shop does not get its own catalogue before the menu is reached")
	}
	// ⚠️ And the menu is not written on the way past: the shop branch returns.
	if !strings.Contains(fn[shop:menu], "return") {
		t.Error("a shop is given the restaurant menu as well as its own")
	}
}

// ⚠️ **Every sample product goes through the panel's own stock code.** A row
// written straight into Mongo would be a product that can be sold and cannot be
// counted — wrong in silence, from the first day, on the demonstration data.
func TestEverySampleProductGetsItsStockRow(t *testing.T) {
	fn := between(t, source(t, "shop.go"), "func writeShopCatalogue(", "\n\tlog.Printf")
	if !strings.Contains(fn, "repository.SyncProductStock") {
		t.Error("sample products are written without their stock rows")
	}
	if strings.Contains(fn, "InsertMany") {
		t.Error("a bulk insert cannot write back the stock id it needs")
	}
	if !strings.Contains(fn, "SellsItself: true") {
		t.Error("a sample product is not marked as its own stock row")
	}
}

// ⚠️ **Every kind of shop we sell to has a catalogue.** A pharmacy with none
// would get the empty grid this whole file exists to avoid — and the one that
// would be missed is whichever type was added last.
func TestEveryShopTypeHasASample(t *testing.T) {
	for _, biz := range models.BusinessTypes {
		if !biz.SellsGoods() {
			continue
		}
		cats, ok := shopCatalogues[biz]
		if !ok || len(cats) == 0 {
			t.Errorf("%s starts with an empty catalogue", biz)
			continue
		}
		for _, c := range cats {
			if len(c.Items) == 0 {
				t.Errorf("%s: category %q is empty", biz, c.Name)
			}
			for _, it := range c.Items {
				// ⚠️ A price of zero would make the till, the receipt and the
				// shelf label all look broken on the demonstration data.
				if it.Price <= 0 {
					t.Errorf("%s: %q has no price", biz, it.Name)
				}
				if it.NameRu == "" || it.NameEn == "" {
					t.Errorf("%s: %q is not named in all three languages", biz, it.Name)
				}
				// The five codes the menu form writes, and nothing else: a
				// number from another list is a unit the shelf label cannot
				// name and a purchase cannot be divided by.
				switch it.Unit {
				case 0, 10, 11, 41, 22:
				default:
					t.Errorf("%s: %q has unit code %d", biz, it.Name, it.Unit)
				}
			}
		}
	}
}

// ⚠️ **A counter that cooks gets neither the restaurant's menu nor a shop's
// shelf.** Forty-eight dishes in a bakery is the pharmacy's problem with bread
// in it; a shop's catalogue is worse than wrong, because every loaf would be
// written as its own stock row and the first technical card the baker writes
// would point a product at itself. The one that would be missed is whichever
// maker was added last.
func TestEveryMakerTypeHasItsOwnShortMenu(t *testing.T) {
	for _, biz := range models.BusinessTypes {
		if biz.SellsGoods() || !biz.Composes() || biz.HasTables() {
			continue // shops have their own test; a restaurant has the demo menu
		}
		if biz == models.BizFastFood {
			continue // a fast food is a restaurant's menu, shorter
		}
		cats, ok := makerCatalogues[biz]
		if !ok || len(cats) == 0 {
			t.Errorf("%s starts with an empty catalogue", biz)
			continue
		}
		for _, c := range cats {
			if len(c.Items) == 0 {
				t.Errorf("%s: category %q is empty", biz, c.Name)
			}
			for _, it := range c.Items {
				if it.Price <= 0 {
					t.Errorf("%s: %q has no price", biz, it.Name)
				}
				if it.NameRu == "" || it.NameEn == "" {
					t.Errorf("%s: %q is not named in all three languages", biz, it.Name)
				}
			}
		}
	}
}

// ⚠️ **What a bakery sells is not its own stock row.** The loaf is costed
// through a card over flour and yeast; marking it `SellsItself` — the shop
// path — would make that card point the product at itself, and the cost report
// cannot answer a loop.
func TestAMakersSampleIsNotWrittenAsShopStock(t *testing.T) {
	fn := between(t, source(t, "shop.go"), "func writeMakerCatalogue(", "\n\tlog.Printf")
	if strings.Contains(fn, "SellsItself") {
		t.Error("a maker's sample is written as goods that sell themselves")
	}
	if strings.Contains(fn, "SyncProductStock") {
		t.Error("a maker's menu item was given a stock row of its own")
	}
	// And the branch is taken before the restaurant's demo menu is reached.
	menu := between(t, source(t, "menu.go"), "func ensureMenu(", "\n}\n")
	maker := strings.Index(menu, "writeMakerCatalogue")
	demo := strings.Index(menu, "WriteMenu(")
	if maker < 0 || demo < 0 || maker > demo {
		t.Error("a maker is given the restaurant's demo menu")
	}
	if !strings.Contains(menu[maker:demo], "return") {
		t.Error("a maker is given the restaurant menu as well as its own")
	}
}
