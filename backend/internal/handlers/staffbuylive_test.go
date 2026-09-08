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
	// ⚠️ **And it is written when the list is accepted, not when it is
	// shipped.** What reaches the shelf is what somebody at the restaurant
	// counted; a purchase written at the market would put food on a shelf while
	// it was still in a bag on a bus. See handlers/buyorderflow.go.
	flow := readSource(t, "buyorderflow.go")
	accept := between(t, flow, "func (h *Handler) StaffAcceptBuyOrder", "\n}\n")
	if !strings.Contains(accept, "h.recordMarketRun(") {
		t.Fatal("accepting a shopping list builds its own purchase")
	}
	ship := between(t, flow, "func (h *Handler) StaffShipBuyOrder", "\n}\n")
	if strings.Contains(ship, "recordMarketRun") || strings.Contains(ship, "Purchases") {
		t.Fatal("shipping writes the delivery — nobody has counted it yet")
	}

	orders := readSource(t, "buyorders.go")
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

// ---- How the market sells it ----

// ⚠️ **The silent one.** A market sells mint in bunches; the store counts kilos.
// "5" typed into a field measured in kilos puts five kilos on the shelf instead
// of a quarter of one — nothing errors, the figure is twenty times too high, the
// stop list never fires, and the gap surfaces a month later at a count.
func TestAPackIsConvertedIntoWhatTheStoreCounts(t *testing.T) {
	h, _ := liveHandler(t)
	ctx := context.Background()
	brand := primitive.NewObjectID()
	mint := primitive.NewObjectID()
	if _, err := h.Store.Ingredients.InsertOne(ctx, models.Ingredient{
		ID: mint, BrandID: brand, Name: "Myata", Unit: models.UnitKg,
		// One bunch is fifty grams.
		PackName: "bog'lam", PackQty: 0.05,
	}); err != nil {
		t.Fatal(err)
	}
	staff := models.Staff{ID: primitive.NewObjectID(), BranchID: primitive.NewObjectID()}

	lines, _, err := h.buyLines(ctx, staff, brand, []buyRequestLine{
		{IngredientID: mint.Hex(), Qty: 5, Price: 3000, Pack: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 {
		t.Fatalf("%d lines, wanted 1", len(lines))
	}
	if diff := lines[0].Qty - 0.25; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("qty=%v kg, wanted 0.25 — five bunches, not five kilos", lines[0].Qty)
	}
	// ⚠️ Divided, not multiplied. A 3 000 so'm bunch weighing 0.05 kg is 60 000
	// a kilo; getting this backwards costs the opposite of what the quantity's
	// mistake does and looks just as ordinary afterwards.
	if lines[0].Price != 60000 {
		t.Fatalf("price=%d, wanted 60000 per kg", lines[0].Price)
	}
}

// Without the flag nothing is converted — which is every ingredient bought in
// the unit it is kept in, and every line recorded before this existed.
func TestWithoutTheFlagTheFigureIsTakenAsItIs(t *testing.T) {
	h, _ := liveHandler(t)
	ctx := context.Background()
	brand := primitive.NewObjectID()
	mint := primitive.NewObjectID()
	if _, err := h.Store.Ingredients.InsertOne(ctx, models.Ingredient{
		ID: mint, BrandID: brand, Name: "Myata", Unit: models.UnitKg,
		PackName: "bog'lam", PackQty: 0.05,
	}); err != nil {
		t.Fatal(err)
	}
	staff := models.Staff{ID: primitive.NewObjectID(), BranchID: primitive.NewObjectID()}

	lines, _, err := h.buyLines(ctx, staff, brand, []buyRequestLine{
		{IngredientID: mint.Hex(), Qty: 2, Price: 60000},
	})
	if err != nil {
		t.Fatal(err)
	}
	if lines[0].Qty != 2 || lines[0].Price != 60000 {
		t.Fatalf("qty=%v price=%d — an unflagged line was converted",
			lines[0].Qty, lines[0].Price)
	}
}

// ⚠️ **Both halves or neither.** A name with no size converts nothing, and a
// size with no name would multiply somebody's quantity by a factor nobody can
// see on screen — the failure the field exists to prevent, arriving through the
// form that configures it.
func TestHalfAPackagingIsNoPackaging(t *testing.T) {
	if (models.Ingredient{PackName: "qop"}).HasPack() {
		t.Error("a packaging name with no size counts as packaging")
	}
	if (models.Ingredient{PackQty: 50}).HasPack() {
		t.Error("a size with no name counts as packaging")
	}
	if !(models.Ingredient{PackName: "qop", PackQty: 50}).HasPack() {
		t.Error("a complete packaging does not count")
	}
}

// ⚠️ The conversion belongs to the server, not the phone: the factor is a fact
// about the ingredient and the result lands on a shelf. A screen sending kilos
// it worked out itself would be a second implementation of a figure nothing
// downstream can check.
func TestThePhoneSendsWhatWasTappedAndTheServerConverts(t *testing.T) {
	src := readSource(t, "staffbuy.go")
	fn := between(t, src, "func (h *Handler) buyLines", "\n}\n")

	if !strings.Contains(fn, "ing.PackQty") {
		t.Fatal("the conversion left the server — the phone is doing arithmetic that lands on a shelf")
	}
}

// ---- The safe ----

// ⚠️ **A duplicate in a balance is the worst kind of wrong**: plausible, and
// invisible to everything downstream. The automatic entries are written by
// handlers that can be retried, so the guarantee is the index rather than the
// caller being careful.
func TestOneSafeMovementPerThingThatCausedIt(t *testing.T) {
	h, branch := liveHandler(t)
	ctx := context.Background()
	if _, err := h.Store.SafeEntries.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "refKind", Value: 1}, {Key: "refId", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}); err != nil {
		t.Fatal(err)
	}
	ref := primitive.NewObjectID()
	row := models.SafeEntry{
		BranchID: branch, Kind: models.SafeOut, Amount: 2_000_000,
		RefKind: models.SafeRefAdvance, RefID: ref,
	}
	h.recordSafeMovement(ctx, row)
	h.recordSafeMovement(ctx, row)
	h.recordSafeMovement(ctx, row)

	got, err := h.safeBalance(ctx, bson.M{"branchId": branch})
	if err != nil {
		t.Fatal(err)
	}
	if got.Out != 2_000_000 || got.Balance != -2_000_000 {
		t.Fatalf("out=%d balance=%d — the same hand-over was recorded more than once",
			got.Out, got.Balance)
	}
}

// ⚠️ **Below zero is shown, not clamped.** A negative safe means the ledger is
// missing something that went in — an owner's own money, a collection nobody
// recorded — and hiding it would leave the one screen that could have said so
// agreeing with a count that cannot be right.
func TestASafeCanReadNegativeAndSaysSo(t *testing.T) {
	h, branch := liveHandler(t)
	ctx := context.Background()
	if _, err := h.Store.SafeEntries.InsertOne(ctx, models.SafeEntry{
		BranchID: branch, Kind: models.SafeOut, Amount: 500_000,
	}); err != nil {
		t.Fatal(err)
	}

	got, _ := h.safeBalance(ctx, bson.M{"branchId": branch})
	if got.Balance != -500_000 {
		t.Fatalf("balance=%d, wanted −500 000", got.Balance)
	}
}

// ⚠️ **A hand-over typed with no reference is still a movement.** Most rows are
// typed by somebody — an owner's deposit, the rent — and a guard that demanded a
// reference would refuse exactly those.
func TestATypedMovementNeedsNoReference(t *testing.T) {
	h, branch := liveHandler(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := h.Store.SafeEntries.InsertOne(ctx, models.SafeEntry{
			BranchID: branch, Kind: models.SafeIn, Amount: 100_000,
		}); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := h.safeBalance(ctx, bson.M{"branchId": branch})
	if got.In != 300_000 {
		t.Fatalf("in=%d, wanted 300 000 — a sparse unique index refused rows with no reference", got.In)
	}
}

// ⚠️ **The safe is a place, not a profit and loss.** Its movements must never
// reach the financial report: cash going from the drawer into the safe is not an
// expense, and money handed to a buyer is not spent until it buys something.
// Counting them would subtract the same money twice.
func TestTheSafeStaysOutOfTheFinancialReport(t *testing.T) {
	src := readSource(t, "finreport.go")
	if strings.Contains(src, "SafeEntries") || strings.Contains(src, "Advances") {
		t.Fatal("the financial report reads a cash location — the same money is now subtracted twice")
	}
}

// ---- Everything that moves money says so ----

// ⚠️ **A collection is not a cost.** Emptying the drawer into the office box
// takes money *out* of the till and puts it *into* the safe, so the mirrored row
// has to be the opposite kind. Copying the kind across instead — the obvious
// reading of "record it in both places" — would make every collection subtract
// from a safe it was filling, and the balance would go negative by twice the
// takings while every individual row looked right.
func TestACollectionFillsTheSafeItLeavesTheDrawerFor(t *testing.T) {
	src := between(t, readSource(t, "cash.go"), "if req.ToSafe {", "RefKind: models.SafeRefCash")
	if !strings.Contains(src, "kind := models.SafeIn") ||
		!strings.Contains(src, "entry.Kind == models.CashIn") ||
		!strings.Contains(src, "kind = models.SafeOut") {
		t.Fatal("the till's mirror no longer inverts the direction — a collection now empties the safe it fills")
	}
}

// ⚠️ **Both screens write the same movement.** The panel and the till share
// addCashEntry precisely so a collection typed at the counter and one typed in
// the office cannot disagree; a till handler that built its own row would drift
// the first time either side changed.
func TestTheTillsCollectionGoesThroughTheSameDoor(t *testing.T) {
	src := readSource(t, "tillcash.go")
	if !strings.Contains(src, "ToSafe: req.ToSafe") {
		t.Fatal("the till drops toSafe — a collection recorded at the counter never reaches the safe")
	}
	if strings.Contains(src, "recordSafeMovement") {
		t.Fatal("the till writes its own safe row instead of sharing addCashEntry's")
	}
}

// ⚠️ **Cash wages leave the box; transferred ones do not.** Payroll asks rather
// than assuming, because a safe that assumed would be short by every wage that
// was actually sent to a card — and short in a way that reads exactly like theft.
func TestACashWageIsTheOnlyKindThatEmptiesTheSafe(t *testing.T) {
	src := readSource(t, "adminstaff.go")
	if !strings.Contains(src, "FromSafe") || !strings.Contains(src, "models.SafeRefSalary") {
		t.Fatal("a wage paid from the safe no longer moves it")
	}
	if !strings.Contains(src, "models.SafeOut") {
		t.Fatal("a wage is being recorded as money going into the safe")
	}
}

// ⚠️ **The costs that made the report optimistic must reach it.** Rent, gas and
// tax have no document of their own, so if the financial report does not read
// the expenses collection, "in − out" is better than the month was — by
// whatever the building costs, every month, in the same direction.
func TestOtherCostsReachTheFinancialReport(t *testing.T) {
	src := readSource(t, "finreport.go")
	if !strings.Contains(src, "h.Store.Expenses") {
		t.Fatal("the financial report ignores other costs — the profit line is optimistic again")
	}
	if !strings.Contains(src, `Kind: "out"`) {
		t.Fatal("other costs are not counted as an outgoing")
	}
}

// ⚠️ **An expense and a cash movement are two facts.** Paying the rent from the
// safe makes the restaurant poorer *and* empties a box; paying it by transfer
// only does the first. So deleting the expense must leave the safe's row where
// it is: the money physically went, and un-typing the cost does not bring the
// notes back.
func TestDeletingACostLeavesTheSafeAlone(t *testing.T) {
	src := between(t, readSource(t, "expenses.go"),
		"func (h *Handler) AdminDeleteExpense", "func expenseMethod")
	if strings.Contains(src, "SafeEntries") || strings.Contains(src, "recordSafeMovement") {
		t.Fatal("deleting a cost now edits the safe — the ledger no longer matches the notes in the box")
	}
}

// ---- The four remaining holes ----

// ⚠️ **Paying a supplier is the commonest way money leaves a safe**, and a
// delivery marked paid used to say only that the supplier was square. Asked
// rather than inferred, because it is settled by transfer or out of the drawer
// just as often.
func TestSettlingAnInvoiceCanEmptyTheSafe(t *testing.T) {
	src := readSource(t, "suppliers.go")
	if !strings.Contains(src, "req.FromSafe") ||
		!strings.Contains(src, "models.SafeRefPurchase") {
		t.Fatal("a supplier paid in cash no longer moves the safe")
	}
	// ⚠️ The endpoint took no body until now, so a screen that still sends none
	// has to keep working: a decode failure must not refuse the payment.
	if !strings.Contains(src, "_ = httpx.Decode(r, &req)") {
		t.Fatal("a body-less pay request is now refused — every older screen just broke")
	}
}

// ⚠️ **A courier's pay and a courier's settlement are opposites.** One is us
// handing them a wage; the other is them handing our collected cash back.
// Sharing a document would credit a courier for money they returned.
func TestCourierPayIsNotACourierSettlement(t *testing.T) {
	src := readSource(t, "courierpay.go")
	if strings.Contains(src, "Settlements") {
		t.Fatal("courier pay writes to the settlement ledger — a returned handover now reads as a wage")
	}
	if !strings.Contains(src, "h.Store.CourierPayments") {
		t.Fatal("courier pay has no ledger of its own")
	}
}

// ⚠️ **Earned is derived, paid is recorded, and neither may be computed from
// the other.** What a courier earned comes from the deliveries and the payout
// rule; what they were handed is a document. Deriving either direction gives a
// figure that moves when a rule is edited, months after the notes were counted.
func TestWhatACourierEarnedIsNeverWhatTheyWerePaid(t *testing.T) {
	// ⚠️ Scoped to the **write**, not the file. Working out what a period owes
	// must read the payout rule — that is what earnings are. What must never
	// touch it is the record of money handed over: a rule edited in December
	// would otherwise rewrite an August payment that was counted out in notes.
	pay := between(t, readSource(t, "courierpay.go"),
		"func (h *Handler) AdminPayCourier", "\n}\n")
	if strings.Contains(pay, "courierEarning(") ||
		strings.Contains(pay, "courierEarnedBetween") {
		t.Fatal("a recorded payment is now derived from the payout rule — history rewrites itself")
	}
}

// ⚠️ **Couriers get their own line, not a share of wages.** They are the cost
// that scales with delivery volume, and an owner asking whether delivery pays
// for itself needs that number apart from the kitchen's.
func TestCourierPayReachesTheReportOnItsOwnLine(t *testing.T) {
	src := readSource(t, "finreport.go")
	if !strings.Contains(src, "h.Store.CourierPayments") {
		t.Fatal("the report still leaves courier pay out — it was the one cost it admitted it could not see")
	}
	if !strings.Contains(src, "Kuryerlarga to'langan") {
		t.Fatal("courier pay was folded into another line")
	}
}

// ⚠️ **A short drawer is a counting problem, not an expense.** Nothing was
// bought and no document exists; subtracting it would quietly turn a miscount
// into a cost, and a surplus into income. It is shown so a pattern is visible —
// one short evening is noise, the same drawer short every Friday is not.
func TestTheDrawerVarianceIsShownAndChangesNoTotal(t *testing.T) {
	src := readSource(t, "finreport.go")
	block := between(t, src, "Kassa farqi", "Spisaniya")
	if strings.Contains(block, "out +=") || strings.Contains(block, "in +=") {
		t.Fatal("the drawer variance now moves the bottom line — a miscount reads as an expense")
	}
	if !strings.Contains(block, `Kind: "info"`) {
		t.Fatal("the variance is no longer marked as information")
	}
}

// ⚠️ **Spoiled stock is already inside the cost of food sold.** Counting it as
// an outgoing as well would subtract the same tomatoes twice. It is shown
// because "4 200 000 in the bin this month" changes what an owner does and
// nothing else in this report says it.
func TestWriteOffsAreShownButNotSubtractedTwice(t *testing.T) {
	src := readSource(t, "finreport.go")
	block := between(t, src, "Spisaniya", "Ekvayring")
	if strings.Contains(block, "out +=") {
		t.Fatal("write-offs are being subtracted on top of the cost of food sold")
	}
}

// ⚠️ **An open shift has no count**, and its zero would read as "the drawer is
// exactly right" — the most reassuring possible way to be wrong about money.
func TestOnlyCountedDrawersEnterTheVariance(t *testing.T) {
	src := between(t, readSource(t, "finreport.go"),
		"func (h *Handler) shiftVariance", "func mergeExists")
	if !strings.Contains(src, `f["closedAt"] = mergeExists`) {
		t.Fatal("shifts that were never counted now contribute a zero variance")
	}
}

// ---- Money somebody else is holding ----

// ⚠️ **The transfer is not income — the sale already was.** An aggregator
// collects the guest's money and sends it on a month later; counting the
// arrival as revenue would book every marketplace sale twice, and a doubled
// revenue figure looks entirely plausible. Only the commission is a cost.
func TestAPayoutArrivingIsNotRevenue(t *testing.T) {
	src := readSource(t, "finreport.go")
	if strings.Contains(src, `"$net"`) || strings.Contains(src, `"$gross"`) {
		t.Fatal("the financial report reads a payout's own totals — every marketplace sale is now counted twice")
	}
	if !strings.Contains(src, `h.Store.Payouts`) || !strings.Contains(src, `"$commission"`) {
		t.Fatal("the commission an aggregator keeps is not counted as a cost")
	}
}

// ⚠️ **Net is stored, never derived from gross − commission.** When the three
// disagree the difference is the most valuable thing on the page: a refund
// clawed back, a penalty, a correction carried over. Computing one of them
// erases exactly the discrepancy this document exists to surface.
func TestAStatementIsRecordedAsItReads(t *testing.T) {
	src := readSource(t, "payouts.go")
	if strings.Contains(src, "Gross - req.Commission") ||
		strings.Contains(src, "req.Gross - req.Commission") {
		t.Fatal("a payout now computes one of its own figures — a short transfer would balance perfectly")
	}
	if !strings.Contains(src, "Net: req.Net") {
		t.Fatal("the amount that actually landed is no longer taken from the statement")
	}
}

// ⚠️ **Cash, transfer and the slate hold nobody's money.** Cash is in the
// drawer, a transfer lands directly, and a debt is owed by a named guest and
// already has its own line. A payout balance for any of them would be a debt
// that can never be cleared.
func TestOnlyRailsThatHoldMoneyGetABalance(t *testing.T) {
	src := between(t, readSource(t, "payouts.go"),
		"func (h *Handler) payoutRails", "\n}\n")
	for _, bad := range []string{"ProviderCash", "MethodDebt", "MethodTransfer"} {
		if strings.Contains(src, bad) {
			t.Fatalf("%s is treated as a rail that owes us money", bad)
		}
	}
}

// ⚠️ **Only paid orders are money a rail owes us.** A pending online payment is
// a guest still finding their phone; counting it would put a debt on somebody
// who was never given anything. A cancelled order likewise.
func TestOnlyPaidSalesCountAsOwedByARail(t *testing.T) {
	src := between(t, readSource(t, "payouts.go"),
		"func (h *Handler) soldThrough", "\n}\n")
	if !strings.Contains(src, "models.PayPaid") ||
		!strings.Contains(src, "models.StatusCancelled") {
		t.Fatal("unpaid or cancelled sales are now counted as money a provider owes")
	}
}

// ⚠️ **The latest period end, not the latest document.** Statements arrive out
// of order often enough — a corrected March lands after April — and taking the
// newest payout's period would reopen sales that are already settled.
func TestSettlementBoundaryIsTheFurthestPeriod(t *testing.T) {
	src := between(t, readSource(t, "payouts.go"),
		"func (h *Handler) payoutBalances", "\n}\n")
	if !strings.Contains(src, "if p.PeriodTo > a.through") {
		t.Fatal("the settled-through boundary follows document order — a late correction reopens settled sales")
	}
}

// ⚠️ **A marketplace order must be closable on the till.** The server accepts
// the two ids unconditionally while the screen only offers them when the
// restaurant has switched them on: a method the server refuses is a cashier
// standing at a counter unable to close a check.
func TestAMarketplaceCheckCanBeClosed(t *testing.T) {
	src := readSource(t, "tillclose.go")
	if !strings.Contains(src, "models.ProviderYandexEats: true") ||
		!strings.Contains(src, "models.ProviderUzumTezkor: true") {
		t.Fatal("a marketplace order can no longer be closed at the till")
	}
}

// ---- Where the money is ----

// ⚠️ **Three kinds of having, never one number.** Cash in a box can be spent
// tonight, money in the bank this week, money an aggregator holds when somebody
// else decides. A single "we have X" would be the most quotable and least true
// figure on the platform — and the one an owner takes to a bank.
func TestTheMoneyPositionKeepsItsThreeTotalsApart(t *testing.T) {
	src := readSource(t, "money.go")
	if strings.Contains(src, "CashTotal + pos.BankTotal") ||
		strings.Contains(src, "Total = pos.CashTotal") {
		t.Fatal("the three kinds of money were added into one — cash, bank and receivable are not the same having")
	}
}

// ⚠️ **We do not know the bank balance and must never pretend to.** Money
// reaches that account from rails we record and from a dozen we do not: an
// owner's own deposit, a loan, a transfer between the company's accounts. A
// derived balance would be wrong by everything we cannot see, and wrong in a
// way that looks exactly like a balance. So it is counted, with a date.
func TestTheBankBalanceIsCountedNotDerived(t *testing.T) {
	src := between(t, readSource(t, "money.go"),
		"func (h *Handler) bankPlaces", "\n}\n")
	if strings.Contains(src, "h.Store.Payouts") || strings.Contains(src, "h.Store.Expenses") {
		t.Fatal("the bank balance is being computed from movements — it is now confidently wrong")
	}
	if !strings.Contains(src, "Counted: true") {
		t.Fatal("the bank figure no longer says it was counted")
	}
}

// ⚠️ **Only open drawers hold cash right now.** A closed shift has been counted
// and handed on; adding it would count last night's takings again this morning.
func TestOnlyOpenDrawersCountAsCashOnHand(t *testing.T) {
	src := between(t, readSource(t, "money.go"),
		"func (h *Handler) drawerCash", "\n}\n")
	if !strings.Contains(src, `filter["closedAt"] = bson.M{"$exists": false}`) {
		t.Fatal("closed shifts are being counted as cash still in the drawer")
	}
}

// ⚠️ **Courier cash is summed over every delivery, not a recent window.** The
// courier card samples the last two hundred orders because it draws a list; a
// figure about money owed cannot sample — dropping old deliveries while keeping
// every handover makes the debt shrink on its own.
func TestCourierCashIsNotSampled(t *testing.T) {
	src := between(t, readSource(t, "money.go"),
		"func (h *Handler) courierCash", "\n}\n")
	if strings.Contains(src, "SetLimit") || strings.Contains(src, "deliveredOrders") {
		t.Fatal("the courier cash figure samples orders — the debt now shrinks by itself")
	}
}

// ---- Inkassatsiya ----

// ⚠️ **A collection is a place, not a cost.** Money going to the bank has not
// been spent; it moved from a box to an account. Counting it as an outgoing
// would subtract the restaurant's own takings from itself.
func TestACollectionIsNotAnExpense(t *testing.T) {
	src := readSource(t, "finreport.go")
	if strings.Contains(src, "h.Store.Collections") {
		t.Fatal("the financial report counts collections — the restaurant now subtracts its own takings")
	}
}

// ⚠️ **Closed shifts only.** An open drawer has not been counted, and folding
// in a figure nobody verified is how a handover comes to be checked against an
// estimate.
func TestACollectionChecksOnlyCountedDrawers(t *testing.T) {
	src := between(t, readSource(t, "collections.go"),
		"func (h *Handler) collectionDraft", "\n}\n")
	if !strings.Contains(src, `closed := bson.M{"$exists": true, "$ne": nil}`) {
		t.Fatal("an uncounted shift now enters the reconciliation")
	}
}

// ⚠️ **The preview and the document come from one function.** Two code paths
// for "what is due" would drift, and the one on screen is the one somebody
// counts against before signing.
func TestTheHandoverIsCheckedAgainstWhatWasShown(t *testing.T) {
	src := readSource(t, "collections.go")
	if strings.Count(src, "h.collectionDraft(r,") < 2 {
		t.Fatal("the preview and the recorded document no longer share their arithmetic")
	}
}

// ⚠️ **A handover happens at one door, with one bag, from one drawer.** "The
// company collected 14 000 000" is a number nobody can hand to anybody — the
// same rule the stock screens follow.
func TestACollectionBelongsToOneBranch(t *testing.T) {
	src := between(t, readSource(t, "collections.go"),
		"func (h *Handler) AdminCreateCollection", "\n}\n")
	if !strings.Contains(src, "branch.IsZero()") {
		t.Fatal("a collection can be recorded across branches — it belongs to a door")
	}
}

// ⚠️ **Zero is written too.** A document that only carried a difference when
// there was one would leave "checked and correct" indistinguishable from
// "nobody checked".
func TestTheHandoverRecordsAZeroDifference(t *testing.T) {
	src := readSource(t, "collections.go")
	if !strings.Contains(src, "c.Diff = c.Amount - c.Counted") {
		t.Fatal("the difference between what left and what was counted is no longer frozen")
	}
}

// ⚠️ **Our own records are evidence, not a substitute for the statement.** For
// Click, Payme, Uzum and ATMOS the provider calls this server to confirm every
// payment, so what they collected is something we watched happen. It fills the
// form; the owner still types what the statement says, because the difference
// between the two is the entire point. Writing our figure into `gross` would
// produce a payout that always reconciles and never catches anything.
func TestOurOwnFigureOnlySuggests(t *testing.T) {
	src := readSource(t, "payouts.go")
	create := between(t, src, "func (h *Handler) AdminCreatePayout", "\n}\n")
	if strings.Contains(create, "AdminPayoutExpected") ||
		strings.Contains(create, "sumField") {
		t.Fatal("a payout now fills its own gross from our records — it will always balance and never catch a short transfer")
	}
	if !strings.Contains(src, `"watched": watched`) {
		t.Fatal("the screen can no longer tell a rail we watched from one we only typed")
	}
}

// ---- Wages paid out of the drawer ----

// ⚠️ **Two facts, not two copies.** A cashier handing a courier his month out
// of the till used to produce one document: a cash entry reading "maosh". The
// drawer was right and payroll went on saying the whole amount was owed — so
// the restaurant either paid twice or spent an evening arguing about a Tuesday
// nobody could remember.
func TestAWagePaidAtTheTillReachesPayroll(t *testing.T) {
	src := readSource(t, "cash.go")
	if !strings.Contains(src, "h.recordWagePayment(r, req, entry)") {
		t.Fatal("a wage paid from the drawer no longer reaches payroll — it will be paid again at the end of the month")
	}
	fn := between(t, src, "func (h *Handler) recordWagePayment", "\n}\n")
	if !strings.Contains(fn, "h.Store.CourierPayments") ||
		!strings.Contains(fn, "h.Store.StaffPayments") {
		t.Fatal("only one kind of person can be paid at the counter")
	}
}

// ⚠️ **The money already left the drawer.** Refusing to record that because the
// payroll write failed would lose the one fact we are certain of, to protect a
// bookkeeping nicety.
func TestABrokenPayrollWriteStillRecordsTheCash(t *testing.T) {
	fn := between(t, readSource(t, "cash.go"),
		"func (h *Handler) recordWagePayment", "\n}\n")
	if strings.Contains(fn, "return entry, http.Status") {
		t.Fatal("a failed payroll write now refuses the cash entry — the drawer's own record is the one thing we know")
	}
}

// ⚠️ **Picked from a list, never typed.** A name written at a counter is
// "Aziz", "aziz", "Азиз" and "Aziz kuryer" inside a week, and none of them can
// be matched to the person payroll still owes.
func TestTheCounterPicksAPersonRatherThanTypingOne(t *testing.T) {
	src := readSource(t, "tillcash.go")
	if !strings.Contains(src, "func (h *Handler) StaffPayees") {
		t.Fatal("the till has no list of people to pay — the name will be typed")
	}
	if !strings.Contains(src, "withActive(branch)") {
		t.Fatal("dismissed people are offered as payees")
	}
}

// ---- Couriers on the payroll ----

// ⚠️ **A courier is paid once a month like everybody else.** They were the only
// paid people with no pay period, so the screens could say what one had ever
// earned — a figure that only grows — and never what was owed now.
func TestCouriersHaveAPayPeriodLikeEverybodyElse(t *testing.T) {
	src := readSource(t, "courierpay.go")
	if !strings.Contains(src, "payPeriodBounds(c.PayPeriod, now)") {
		t.Fatal("a courier's pay has no window — 'what do we owe him' has no answer")
	}
	if !strings.Contains(src, "Earned: earned, Paid: paid, Due: earned - paid") {
		t.Fatal("a courier row no longer answers what is still owed")
	}
}

// ⚠️ **A salaried courier earns nothing per delivery.** Paying the fee as well
// would pay them twice, and the doubled figure would look exactly like a busy
// month.
func TestASalariedCourierIsNotPaidPerDeliveryToo(t *testing.T) {
	fn := between(t, readSource(t, "courierstats.go"),
		"func courierEarning", "\n}\n")
	if !strings.Contains(fn, "case models.PayoutMonthly:") {
		t.Fatal("a monthly courier is also earning a delivery fee — they are paid twice")
	}
}

// ⚠️ **`courierId`, never `staffId`.** The pay button posts to a different
// endpoint for each, and one id field with a kind flag would let a row whose
// flag was wrong pay a cook through the courier endpoint — silently, because
// both are valid ObjectIDs.
func TestAPayrollRowSaysWhichKindOfPersonItIs(t *testing.T) {
	src := readSource(t, "adminstaff.go")
	if !strings.Contains(src, `CourierID  string                `+"`json:\"courierId,omitempty\"`") {
		t.Fatal("a payroll row no longer distinguishes a courier from an employee")
	}
}
