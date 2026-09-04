package handlers

// ---- The market run, against a real database ----
//
// ⚠️ **The two failures here are both silent and both cost money.** A retry
// that lands twice raises the shelf twice and pays the invoice twice; a new
// ingredient created on every run fills the catalogue with rows no tech card
// points at. Neither shows an error, and both are found weeks later by whoever
// is holding the clipboard. Only running the thing against a database answers
// whether the guards hold.

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/models"
)

// ⚠️ **The index is the guarantee, not the code that reads it.** The handler's
// look-up closes the common case; this closes the one where two sends cross and
// both miss each other's write. Without it the second insert simply succeeds.
func TestAMarketRunCannotBeRecordedTwice(t *testing.T) {
	h, branch := liveHandler(t)
	ctx := context.Background()
	if _, err := h.Store.Purchases.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "clientId", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}); err != nil {
		t.Fatal(err)
	}

	run := models.Purchase{
		ClientID: "run-1", BranchID: branch,
		Lines: []models.PurchaseLine{{IngredientID: primitive.NewObjectID(), Qty: 3, Price: 9000}},
	}
	if _, err := h.Store.Purchases.InsertOne(ctx, run); err != nil {
		t.Fatal(err)
	}
	_, err := h.Store.Purchases.InsertOne(ctx, run)
	if !mongo.IsDuplicateKeyError(err) {
		t.Fatalf("a retried market run was written twice: %v", err)
	}
}

// ⚠️ **"Pomidor" where "pomidor" exists is the commonest way this goes wrong**,
// and it is not a genuinely new product — it is a second row every tech card
// ignores, on a shelf that then reads half empty forever.
func TestSomethingTheCatalogueAlreadyHasIsNotInventedAgain(t *testing.T) {
	h, _ := liveHandler(t)
	ctx := context.Background()
	brand := primitive.NewObjectID()
	existing := primitive.NewObjectID()
	if _, err := h.Store.Ingredients.InsertOne(ctx, models.Ingredient{
		ID: existing, BrandID: brand, Name: "pomidor", Unit: models.UnitKg,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := h.newBoughtIngredient(ctx, brand, "Pomidor", 12000)
	if err != nil {
		t.Fatal(err)
	}
	if got != existing {
		t.Fatal("a second row was created for an ingredient that already exists")
	}
	n, err := h.Store.Ingredients.CountDocuments(ctx, bson.M{"brandId": brand})
	if err != nil || n != 1 {
		t.Fatalf("%d ingredients in the catalogue, wanted 1", n)
	}
}

// ⚠️ **It is created, and it is marked.** Refusing would mean the buyer cannot
// record half a market run, so they stop recording any of it — the lesson this
// codebase paid for with the supplier field and the void reason. Creating it
// silently would leave a row with no unit, no minimum and no card that looks
// exactly like a finished one.
func TestSomethingNewIsCreatedAndFlaggedForSomebodyToFinish(t *testing.T) {
	h, _ := liveHandler(t)
	ctx := context.Background()
	brand := primitive.NewObjectID()

	id, err := h.newBoughtIngredient(ctx, brand, "Rayhon", 5000)
	if err != nil {
		t.Fatal(err)
	}
	var got models.Ingredient
	if err := h.Store.Ingredients.FindOne(ctx, bson.M{"_id": id}).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if !got.NeedsCare {
		t.Fatal("an ingredient invented at a market looks finished")
	}
	// ⚠️ Pieces rather than a guessed kilo: a card written against "kg" for
	// something sold by the bunch is wrong by three orders of magnitude and
	// reads as deliberate.
	if got.Unit != models.UnitPcs {
		t.Fatalf("unit=%q, wanted pieces until somebody chooses", got.Unit)
	}
	if got.Price != 5000 {
		t.Fatalf("price=%d, wanted what was paid for it", got.Price)
	}
}

// The permission is the whole gate: this account never sees the panel, so if
// the check is wrong there is no second door that would have caught it.
func TestOnlyABuyerRecordsAMarketRun(t *testing.T) {
	buyer := models.Staff{IsActive: true, RoleApplied: true, Perms: []string{models.PermBuy}}
	if msg := buyDenial(buyer); msg != "" {
		t.Fatalf("a buyer was refused: %s", msg)
	}
	// ⚠️ A storekeeper counts shelves and is not a buyer. The two permissions
	// are near opposites — one writes the baseline a shortfall is measured
	// from, the other writes the prices every dish is costed at.
	keeper := models.Staff{IsActive: true, RoleApplied: true, Perms: []string{models.PermStock}}
	if buyDenial(keeper) == "" {
		t.Fatal("counting the store also grants buying for it")
	}
	gone := models.Staff{IsActive: false, RoleApplied: true, Perms: []string{models.PermBuy}}
	if buyDenial(gone) == "" {
		t.Fatal("a dismissed buyer can still record deliveries")
	}
}
