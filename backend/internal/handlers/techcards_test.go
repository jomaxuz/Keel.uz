package handlers

import (
	"context"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

// ⚠️ **This is the trap the move created, and it is silent.** The dish form no
// longer edits the card, so it no longer posts one — and `UpdateMenuItem`
// replaces the whole document. Read as a plain slice, changing a dish's price
// would delete its recipe: no error, no warning, and a menu that has quietly
// stopped being costed looks exactly like a menu nobody has costed yet.
func TestASaveWithoutTheFieldKeepsTheStoredCard(t *testing.T) {
	io := readSource(t, "menucost.go")
	if !strings.Contains(io, "Recipe     *[]models.RecipeLine `json:\"recipe\"`") {
		t.Fatal("the card is not a pointer: a save that omits it cannot be told from one that clears it")
	}
	for _, fn := range []string{"func (h *Handler) UpdateMenuItem", "func (h *Handler) CreateMenuItem"} {
		body := between(t, readSource(t, "admin.go"), fn, "\n}\n")
		if !strings.Contains(body, "h.keepRecipe(") {
			t.Fatalf("%s can wipe a dish's tech card", fn)
		}
	}
}

// An empty array is still an answer: the card screen is allowed to clear a
// card, and only a body that never mentions the field keeps what is stored.
func TestClearingACardIsDifferentFromNotSendingOne(t *testing.T) {
	h := &Handler{}
	id := primitive.NewObjectID()
	ing := primitive.NewObjectID()

	empty := []models.RecipeLine{}
	if got := h.keepRecipe(context.Background(), id, &empty); len(got) != 0 {
		t.Fatalf("an empty card was not stored as empty: %v", got)
	}

	// And a card that is sent is cleaned on the way in — a zero-gram line is
	// always a half-finished row, never a decision.
	sent := []models.RecipeLine{
		{IngredientID: ing, Qty: 180},
		{IngredientID: primitive.NewObjectID(), Qty: 0},
	}
	got := h.keepRecipe(context.Background(), id, &sent)
	if len(got) != 1 || got[0].IngredientID != ing {
		t.Fatalf("the card was not normalised on the way in: %v", got)
	}
}

// ⚠️ A card is the one document here that can be edited to make a theft
// arithmetically invisible, so reaching one by id across a brand boundary is
// the exact shape that must not work — and the previous norms have to be read
// **before** the write, because afterwards there is nothing to compare to.
func TestTheCardSaveIsScopedAndDiffedBeforeItWrites(t *testing.T) {
	fn := between(t, readSource(t, "techcards.go"),
		"func (h *Handler) AdminSaveDishCard", "\n}\n")

	if !strings.Contains(fn, "scope.brandFilter(bson.M{\"_id\": id})") {
		t.Fatal("a dish is selected by _id alone: another brand's card is editable by id")
	}
	diff := strings.Index(fn, "h.recipeDiff(")
	write := strings.Index(fn, "UpdateOne(")
	if diff < 0 || write < 0 || diff > write {
		t.Fatal("the card is written before the change is read: the edit leaves no trace")
	}
	if !strings.Contains(fn, "h.alertOnRecipeIncrease(") {
		t.Fatal("a norm going up on this screen raises nothing")
	}
	// ⚠️ One field, not a whole dish: the card screen holds a copy of the dish
	// it read minutes ago, and posting that back would revert a price somebody
	// changed on the menu screen in between.
	if strings.Contains(fn, "ReplaceOne(") {
		t.Fatal("saving a card rewrites the whole dish")
	}
}

// ⚠️ **Saving an ingredient replaces the whole document, and the form does not
// send the brand.** Without a guard, an edit as small as fixing a typo in a
// name moved the row out of its brand: the ingredient stayed in the database
// and the dish cards kept pointing at it by id, but it stopped appearing on
// the list somebody was about to count.
func TestEditingAnIngredientKeepsItsBrand(t *testing.T) {
	fn := between(t, readSource(t, "ingredients.go"),
		"func (h *Handler) AdminSaveIngredient", "\n}\n")
	if !strings.Contains(fn, "h.keepBrandID(r, h.Store.Ingredients") {
		t.Fatal("an ingredient edit can drop the brand it belongs to")
	}
}
