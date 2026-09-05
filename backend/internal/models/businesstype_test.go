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
