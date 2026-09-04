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
	"strings"
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

// ---- Petty cash ----

// ⚠️ **The balance is a subtraction over documents, and this is what it must
// come out as.** A stored running total would be a second copy of an answer the
// ledger already contains, drifting the first time a delivery was deleted — in
// a figure about money, silently.
func TestWhatABuyerStillHoldsIsIssuedLessSpentAndReturned(t *testing.T) {
	h, branch := liveHandler(t)
	ctx := context.Background()
	buyer := primitive.NewObjectID()
	if _, err := h.Store.Staff.InsertOne(ctx, models.Staff{
		ID: buyer, BranchID: branch, Name: "Sanjar", IsActive: true,
	}); err != nil {
		t.Fatal(err)
	}
	give := func(kind string, sum int) {
		t.Helper()
		if _, err := h.Store.Advances.InsertOne(ctx, models.StaffAdvance{
			BranchID: branch, StaffID: buyer, StaffName: "Sanjar",
			Kind: kind, Amount: sum,
		}); err != nil {
			t.Fatal(err)
		}
	}
	give(models.AdvanceOut, 2_000_000)
	give(models.AdvanceBack, 250_000)
	if _, err := h.Store.Purchases.InsertOne(ctx, models.Purchase{
		BranchID: branch, CreatedByID: buyer, Paid: true, Total: 1_500_000,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := h.advanceBalances(ctx, branch, buyer)
	if err != nil || len(got) != 1 {
		t.Fatalf("balances=%+v err=%v", got, err)
	}
	if got[0].Balance != 250_000 {
		t.Fatalf("balance=%d, wanted 250 000 (2 000 000 − 250 000 − 1 500 000)",
			got[0].Balance)
	}
}

// ⚠️ **An invoice on credit never passed through anybody's hands.** Taking it
// off a buyer's balance would show them as having spent cash they are still
// carrying — and the shortfall would be found when somebody counted the notes.
func TestADeliveryOnCreditDoesNotSpendTheBuyersCash(t *testing.T) {
	h, branch := liveHandler(t)
	ctx := context.Background()
	buyer := primitive.NewObjectID()
	if _, err := h.Store.Advances.InsertOne(ctx, models.StaffAdvance{
		BranchID: branch, StaffID: buyer, StaffName: "Sanjar",
		Kind: models.AdvanceOut, Amount: 1_000_000,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Store.Purchases.InsertOne(ctx, models.Purchase{
		BranchID: branch, CreatedByID: buyer, Paid: false, Total: 400_000,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := h.advanceBalances(ctx, branch, buyer)
	if err != nil || len(got) != 1 {
		t.Fatalf("balances=%+v err=%v", got, err)
	}
	if got[0].Balance != 1_000_000 {
		t.Fatalf("balance=%d, wanted the full float — an unpaid invoice is not cash spent",
			got[0].Balance)
	}
}

// ⚠️ **Below zero is a real answer, not an error to clamp.** A buyer who ran out
// and paid for the last crate themselves is owed money, and a ledger that
// stopped at zero would be silent about exactly the debt somebody is waiting on.
func TestABuyerWhoSpentTheirOwnMoneyIsOwedIt(t *testing.T) {
	h, branch := liveHandler(t)
	ctx := context.Background()
	buyer := primitive.NewObjectID()
	if _, err := h.Store.Advances.InsertOne(ctx, models.StaffAdvance{
		BranchID: branch, StaffID: buyer, StaffName: "Sanjar",
		Kind: models.AdvanceOut, Amount: 500_000,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Store.Purchases.InsertOne(ctx, models.Purchase{
		BranchID: branch, CreatedByID: buyer, Paid: true, Total: 700_000,
	}); err != nil {
		t.Fatal(err)
	}

	got, _ := h.advanceBalances(ctx, branch, buyer)
	if len(got) != 1 || got[0].Balance != -200_000 {
		t.Fatalf("balances=%+v, wanted −200 000", got)
	}
}

// ⚠️ **A manager entering an invoice from the panel is not holding petty
// cash.** Listing them at a negative balance would be a screen that is wrong
// about a person, in the one report where that is read as an accusation.
func TestSomebodyWhoWasNeverGivenCashIsNotOnTheLedger(t *testing.T) {
	h, branch := liveHandler(t)
	ctx := context.Background()
	if _, err := h.Store.Purchases.InsertOne(ctx, models.Purchase{
		BranchID: branch, CreatedByID: primitive.NewObjectID(),
		Paid: true, Total: 900_000,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := h.advanceBalances(ctx, branch, primitive.NilObjectID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("ledger=%+v, wanted nobody", got)
	}
}

// ⚠️ **A market run is paid at the stall**, and saying so is not a convenience:
// `paid` is what the supplier-debt report reads, and an unpaid run has no
// supplier to owe — so every one of them would sit in that report forever as
// money owed to nobody. It is also what makes the run count against the buyer's
// float.
func TestAMarketRunIsRecordedAsPaid(t *testing.T) {
	src := readSource(t, "staffbuy.go")
	// ⚠️ The write moved into `recordMarketRun` when a finished shopping list
	// became a second door into it. The guard follows the writer: pinned to the
	// handler, it would go on passing against a function that no longer builds
	// the document.
	fn := between(t, src, "func (h *Handler) recordMarketRun", "\n}\n")

	if !strings.Contains(fn, "Paid:   true") {
		t.Fatal("a market run is recorded unpaid — it would show as a debt owed to nobody")
	}
}

// ⚠️ **Two doors into one delivery, and they must stay one document.** A
// free-form market run and a finished shopping list both end as a `purchase`;
// two writers would be two answers to "was it paid", "was it stamped with the
// trip's own time" and "what happens to a name the catalogue does not have" —
// three rules that each took a paragraph to get right.
func TestBothBuyingScreensWriteTheSameDelivery(t *testing.T) {
	buy := readSource(t, "staffbuy.go")
	create := between(t, buy, "func (h *Handler) StaffBuyCreate", "\n}\n")
	if !strings.Contains(create, "h.recordMarketRun(") {
		t.Fatal("the free-form run builds its own purchase again")
	}
	orders := readSource(t, "buyorders.go")
	finish := between(t, orders, "func (h *Handler) StaffFinishBuyOrder", "\n}\n")
	if !strings.Contains(finish, "h.recordMarketRun(") {
		t.Fatal("finishing a shopping list builds its own purchase")
	}
	// ⚠️ And the shelf moves only when the trip is finished. A line that raised
	// stock the moment it was ticked would put food on the shelf while the
	// buyer was still at the market, and an untick would then have to take it
	// off — a correction nothing downstream could tell from a theft.
	mark := between(t, orders, "func (h *Handler) StaffMarkBuyOrderLine", "\n}\n")
	if strings.Contains(mark, "recordMarketRun") || strings.Contains(mark, "Purchases") {
		t.Fatal("ticking a line writes stock — the shelf moves before the trip is over")
	}
}

// ⚠️ **The two halves of the supervision are two permissions.** Held by one
// account, the list stops being a check on the trip and becomes a note the
// buyer wrote to themselves — which is the whole reason it exists.
func TestWritingTheListAndShoppingItAreDifferentPeople(t *testing.T) {
	writer := models.Staff{
		IsActive: true, RoleApplied: true,
		BranchID: primitive.NewObjectID(),
		Perms:    []string{models.PermBuyOrder},
	}
	if !writer.Can(models.PermBuyOrder) || writer.Can(models.PermBuy) {
		t.Fatal("writing a shopping list also grants doing the shopping")
	}
	buyer := models.Staff{
		IsActive: true, RoleApplied: true,
		Perms: []string{models.PermBuy},
	}
	if buyer.Can(models.PermBuyOrder) {
		t.Fatal("the buyer can write their own shopping list")
	}
}

// The roles a restaurant is handed have to make the feature reachable on the
// day it ships: the people who hear "we have run out" are the ones at the
// counter, and the storekeeper is the one who writes the list on purpose.
func TestTheShippedRolesCanWriteAShoppingList(t *testing.T) {
	may := map[string]bool{
		"Ish boshqaruvchi": true, "Menejer": true, "Kassir": true, "Omborchi": true,
	}
	seen := map[string]bool{}
	for _, role := range models.SeedRoles() {
		staff := models.Staff{IsActive: true, RoleApplied: true, Perms: role.Perms}
		got := staff.Can(models.PermBuyOrder)
		if got != may[role.Name] {
			t.Errorf("%s: may write a shopping list = %v, want %v", role.Name, got, may[role.Name])
		}
		if got {
			seen[role.Name] = true
		}
	}
	for name := range may {
		if !seen[name] {
			t.Errorf("shipped role %q can no longer write a shopping list", name)
		}
	}
	// ⚠️ And the buyer still cannot. If this ever flips, the supervision is
	// gone and nothing else in the product would say so.
	for _, role := range models.SeedRoles() {
		if role.Name == "Zakupshik" {
			staff := models.Staff{IsActive: true, RoleApplied: true, Perms: role.Perms}
			if staff.Can(models.PermBuyOrder) {
				t.Fatal("the shipped buyer role writes its own shopping lists")
			}
		}
	}
}
