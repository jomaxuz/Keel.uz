package models

import "testing"

// ⚠️ **What is worth sealing is the empty value.** Every brand written before
// this field existed has none, and reading that as anything but a restaurant
// would re-tailor twelve live customers on the day it shipped — the same class
// of fault as an empty `mapProvider` switching everybody's map off.

func TestTheEmptyValueIsARestaurant(t *testing.T) {
	var zero BusinessType
	if zero != BizRestaurant {
		t.Fatal("the zero value is not a restaurant")
	}
	if !zero.HasTables() || !zero.HasKitchen() || zero.SellsGoods() {
		t.Fatal("an existing brand would be re-tailored by this field appearing")
	}
	if got := zero.Defaults(); !got.DineIn || !got.Booking || !got.Delivery {
		t.Fatalf("defaults for an existing brand changed: %+v", got)
	}
}

func TestAnUnknownTypeDegradesRatherThanBreaks(t *testing.T) {
	// Written by a newer build, read by an older one. It loses the tailoring,
	// never the catalogue.
	odd := BusinessType("bakery-with-a-cinema")
	if odd.Valid() {
		t.Fatal("an unknown type was accepted as known")
	}
	if odd.SellsGoods() || !odd.HasTables() {
		t.Fatal("an unknown type did not fall back to the restaurant's behaviour")
	}
}

// ⚠️ Fast food is the case that proves scanning and selling-goods are two
// questions: it cooks, so it is not a shop — and it has no tables, so its till
// must not open on a floor plan.
func TestFastFoodCooksButHasNoFloor(t *testing.T) {
	if BizFastFood.SellsGoods() {
		t.Fatal("fast food was treated as a shop")
	}
	if !BizFastFood.HasKitchen() {
		t.Fatal("fast food lost its kitchen screen")
	}
	if BizFastFood.HasTables() {
		t.Fatal("a counter was given a floor plan")
	}
}

func TestEveryShopSellsWhatItBought(t *testing.T) {
	for _, b := range []BusinessType{BizGrocery, BizClothing, BizFlowers, BizPharmacy} {
		if !b.SellsGoods() || !b.ScansToSell() {
			t.Fatalf("%q does not sell goods", b)
		}
		if b.HasKitchen() || b.HasTables() {
			t.Fatalf("%q was given a kitchen or a floor plan by default", b)
		}
	}
}

func TestTheConsoleOffersTheCommonAnswerFirst(t *testing.T) {
	if BusinessTypes[0] != BizRestaurant {
		t.Fatal("the list does not open on the common answer")
	}
	seen := map[BusinessType]bool{}
	for _, b := range BusinessTypes {
		if seen[b] {
			t.Fatalf("%q listed twice", b)
		}
		seen[b] = true
		if !b.Valid() {
			t.Fatalf("%q is offered but not valid", b)
		}
	}
}

// ⚠️ **A bouquet is the most literal technical card in the product**, and the
// first version of the shop work hid the cards from florists because it asked
// "does this have a kitchen?" — a question a flower shop answers no to while
// composing everything it sells. The two predicates are separate so that
// mistake cannot be made again by reading one as the other.
func TestAFloristComposesWithoutAKitchen(t *testing.T) {
	if BizFlowers.HasKitchen() {
		t.Error("a flower shop has no kitchen")
	}
	if !BizFlowers.Composes() {
		t.Error("a flower shop composes bouquets — it needs technical cards")
	}
	for _, b := range []BusinessType{BizGrocery, BizClothing, BizPharmacy} {
		if b.Composes() {
			t.Errorf("%q composes nothing: the packet sold is the packet delivered", b)
		}
	}
	for _, b := range []BusinessType{BizRestaurant, BizFastFood} {
		if !b.Composes() {
			t.Errorf("%q turns inputs into outputs", b)
		}
	}
}

// ⚠️ **Half of what a florist sells is carried to somebody else's address**, and
// on the eighth of March nearly all of it. A shop set up the week before with
// delivery switched off would discover that on the busiest morning of its year.
func TestAFloristStartsWithDeliveryOn(t *testing.T) {
	if !BizFlowers.Defaults().Delivery {
		t.Error("a flower shop must start with delivery on")
	}
	// And the rest of the shops keep the counter default they were given.
	for _, b := range []BusinessType{BizGrocery, BizClothing, BizPharmacy} {
		if b.Defaults().Delivery {
			t.Errorf("%q now starts with delivery on — was that meant?", b)
		}
		if !b.Defaults().Pickup {
			t.Errorf("%q cannot hand goods over the counter", b)
		}
	}
}

// ⚠️ **A maker is not a shop, and the ladder is not the catalogue.** A bakery,
// a coffee house and a pastry shop are sold at a shop's price (the console's
// question, answered in control/internal/billing) while composing everything
// they sell (this file's question). Reading the cheap price as "sells the thing
// it bought" would take the technical cards, the batches and the cost per loaf
// away from the three businesses whose whole counter is a recipe.
func TestAMakerComposesWithoutBeingAShop(t *testing.T) {
	for _, b := range []BusinessType{BizBakery, BizCoffee, BizPastry} {
		if b.SellsGoods() {
			t.Errorf("%q was read as a shop: its counter is what it made", b)
		}
		if !b.Composes() {
			t.Errorf("%q lost its technical cards", b)
		}
		if b.HasTables() {
			t.Errorf("%q was given a floor plan", b)
		}
		if !b.Defaults().Pickup {
			t.Errorf("%q cannot hand anything over its own counter", b)
		}
	}
}

// ⚠️ **Baked in batches before the doors open is not cooked to order.** A
// kitchen screen in a bakery is a screen that stays empty all day; the document
// those two need is a production batch, and it is `Composes` that gives it to
// them. A coffee house is the opposite case and proves the two are different
// questions: nothing is cooked and every cup is still made after somebody asks.
func TestBatchBakersHaveNoTicketScreen(t *testing.T) {
	for _, b := range []BusinessType{BizBakery, BizPastry} {
		if b.HasKitchen() {
			t.Errorf("%q was given a ticket screen for work done before opening", b)
		}
	}
	if !BizCoffee.HasKitchen() {
		t.Error("a coffee house makes every cup to order")
	}
}

// The shops added beside the first four sell the packet that was delivered.
func TestTheNewerShopsStillSellWhatTheyBought(t *testing.T) {
	for _, b := range []BusinessType{BizButcher, BizCosmetics, BizHardware} {
		if !b.SellsGoods() || !b.ScansToSell() {
			t.Errorf("%q does not sell goods", b)
		}
		if b.Composes() {
			t.Errorf("%q composes nothing: the packet sold is the packet delivered", b)
		}
		if b.HasKitchen() || b.HasTables() {
			t.Errorf("%q was given a kitchen or a floor plan by default", b)
		}
	}
}
