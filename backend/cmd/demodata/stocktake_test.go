package main

import (
	"os"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

// ⚠️ **A demo that disagrees with the product is a demo of a different
// product**, and this one did: the count it wrote put the worth of the *shelf*
// into a field that means "what the count was out by" (models/stocktake.go).
// Every line therefore read as a surplus the size of the whole store — so the
// shortfall queue found nothing to show on the one tenant it is demonstrated
// from, and the count's own total claimed the kitchen had gained a store room.
//
// Pinned as a source guard because the thing that went wrong is one expression,
// there is no database in this test, and the same expression is what the save
// path computes (handlers/stocktake.go: the difference, valued, signed).
func TestTheDemoCountIsOutByTheDifferenceNotByTheShelf(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	i := strings.Index(body, "countAt := w.now.AddDate(0, 0, -14)")
	if i < 0 {
		t.Fatal("the demo stocktake moved — find it and re-point this guard")
	}
	j := strings.Index(body[i:], "w.store.Stocktakes.InsertOne")
	if j < 0 {
		t.Fatal("the demo stocktake is no longer inserted")
	}
	fn := body[i : i+j]

	if strings.Contains(fn, "Value: int(q * float64(in.Price))") {
		t.Fatal("a line's value is the counted stock again — every row reads as a surplus")
	}
	if !strings.Contains(fn, "lineValue := int(diff * float64(in.Price))") {
		t.Fatal("a line's value is not the difference, valued")
	}
	// ⚠️ And the count's total is the sum of those, which is what makes a
	// shortfall a shortfall: `Stocktake.Value` is what the whole count was out
	// by, and the alert that wakes an owner at night reads it.
	if !strings.Contains(fn, "value += lineValue") {
		t.Fatal("the count's total is not the sum of its line differences")
	}
	// ⚠️ Not every line: two per cent off on all two hundred ingredients is not
	// a count anybody has taken, and it fills the shortfall queue with two
	// hundred identical rows — which is the screen that queue replaces.
	if !strings.Contains(fn, "rng.Intn(100)") {
		t.Fatal("every line disagrees by the same amount again")
	}
}

// ⚠️ **A store where every shelf is full has an empty shopping list**, and this
// tool exists because an empty screen is not a screenshot of a working product.
// The generator used to buy a week and a bit for everything on every delivery,
// which is a kitchen that never needs to ring anybody — so the one screen that
// turns the whole stock module into an action had nothing on it, on the tenant
// the product is demonstrated from.
func TestTheDemoStoreIsShortOfSomething(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	i := strings.Index(body, "var purchases []models.Purchase")
	j := strings.Index(body[i:], "insertMany(ctx, w.store.Purchases")
	if i < 0 || j < 0 {
		t.Fatal("the demo deliveries moved — find them and re-point this guard")
	}
	fn := body[i : i+j]
	// A deliberate subset, on the most recent delivery only: earlier weeks stay
	// full, so the history and the supplier report still read normally.
	if !strings.Contains(fn, "week == 0 && i%5 == 0") {
		t.Fatal("every line is bought in full again — the shopping list will be empty")
	}
	// ⚠️ And short, never negative: a demo store that owes itself food reads as
	// a broken product rather than as a busy one.
	if !strings.Contains(fn, "share = 0.2 + rng.Float64()*0.2") {
		t.Fatal("the under-bought share is gone or is no longer a fraction of a week")
	}
}

// ⚠️ **A card is written in grams and this file counts in kilos**, and storing
// one as the other was silent and total: 320 g of rice went in as 0.32 g, so
// consumption rounded to zero, no shelf ever fell, and the shopping list on
// every demo tenant was permanently empty — the exact screen this tool exists
// to fill. The food cost read as a fraction of a per cent on the one page a
// restaurant checks the money on, and `dona` lines were correct (PerUnit is 1),
// which is why looking at one card did not show it.
func TestTheDemoCardIsStoredInRecipeUnits(t *testing.T) {
	ings := []models.Ingredient{
		{ID: primitive.NewObjectID(), Unit: "kg"},
		{ID: primitive.NewObjectID(), Unit: "l"},
		{ID: primitive.NewObjectID(), Unit: "dona"},
	}
	got := recipeUnits([]models.RecipeLine{
		{IngredientID: ings[0].ID, Qty: 0.32},  // 320 g
		{IngredientID: ings[1].ID, Qty: 0.025}, // 25 ml
		{IngredientID: ings[2].ID, Qty: 2},     // two of them
	}, ings)
	want := []float64{320, 25, 2}
	for i, w := range want {
		if got[i].Qty != w {
			t.Errorf("line %d = %v, want %v", i, got[i].Qty, w)
		}
	}
	// ⚠️ And whole units: a card asking for 316.8 g reads as generated, and the
	// third decimal of a gram is not a portion anybody weighs.
	odd := recipeUnits([]models.RecipeLine{
		{IngredientID: ings[0].ID, Qty: 0.3168},
	}, ings)
	if odd[0].Qty != 317 {
		t.Errorf("rounded to %v, want 317", odd[0].Qty)
	}
	// An ingredient this run does not know about must not silently become a
	// portion of nothing.
	unknown := recipeUnits([]models.RecipeLine{
		{IngredientID: primitive.NewObjectID(), Qty: 0.5},
	}, ings)
	if unknown[0].Qty == 0 {
		t.Error("an unknown unit collapsed the line to zero")
	}
}
