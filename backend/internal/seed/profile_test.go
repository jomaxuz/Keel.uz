package seed

import (
	"strings"
	"testing"

	"restaurant-backend/internal/config"
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
func TestTheGivenNameWinsOverTheTemplate(t *testing.T) {
	name, _ := firstProfile(&config.Config{
		BrandName: "Dorixona №7", BusinessType: "pharmacy",
	})
	if name != "Dorixona №7" {
		t.Errorf("the console's name was overwritten with %q", name)
	}
}

// ⚠️ **The demo menu is a restaurant's and it is not given to shops.** The
// sample exists so a new customer's site is not empty on handover — an empty
// grid reads as "this does not work". Forty-eight dishes in a pharmacy read as
// somebody else's shop, and the owner's first task becomes deleting them.
func TestTheDemoMenuIsSkippedForShops(t *testing.T) {
	src := source(t, "menu.go")
	fn := between(t, src, "func ensureMenu(", "\n}\n")
	if !strings.Contains(fn, "SellsGoods()") {
		t.Error("the demo menu is written into a shop's catalogue")
	}
	// Before the count, so nothing is read or extracted for a tenant that is
	// not getting a menu.
	if strings.Index(fn, "SellsGoods()") > strings.Index(fn, "CountDocuments") {
		t.Error("the shop is checked after the catalogue is read")
	}
}
