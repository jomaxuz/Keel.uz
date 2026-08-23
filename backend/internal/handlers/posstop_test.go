package handlers

import (
	"testing"

	"restaurant-backend/internal/models"
	"restaurant-backend/internal/pos"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// The till says which of *its* products are stopped; we hold the dishes. The
// translation between them is one map lookup, and every way it can go wrong is
// silent — a dish that stays orderable, or a dish that disappears from a menu
// nobody stopped.
func TestStoppedMenuItems(t *testing.T) {
	lagmon := primitive.NewObjectID()
	somsa := primitive.NewObjectID()
	choy := primitive.NewObjectID()
	unmapped := primitive.NewObjectID()

	products := []pos.Product{
		{ID: "p-lagmon", Name: "Lag'mon", Unavailable: true},
		{ID: "p-somsa", Name: "Somsa"},
		// Stopped, but nothing here points at it: the till sells things we do
		// not, and those must not stop anything.
		{ID: "p-shashlik", Name: "Shashlik", Unavailable: true},
		// A stopped product with no id at all. Left unguarded, the empty string
		// would match every dish whose link was cleared.
		{ID: "", Name: "?", Unavailable: true},
	}
	mappings := []models.POSMapping{
		{MenuItemID: lagmon, POSProductID: "p-lagmon"},
		{MenuItemID: somsa, POSProductID: "p-somsa"},
		// Whitespace comes from ids pasted out of the till's own UI.
		{MenuItemID: choy, POSProductID: " p-shashlik "},
		{MenuItemID: unmapped, POSProductID: ""},
	}

	got := stoppedMenuItems(products, mappings)
	if len(got) != 2 {
		t.Fatalf("expected 2 stopped dishes, got %d (%v)", len(got), got)
	}
	stopped := map[primitive.ObjectID]bool{}
	for _, id := range got {
		stopped[id] = true
	}
	if !stopped[lagmon] {
		t.Error("a dish linked to a stopped product must be stopped")
	}
	if !stopped[choy] {
		t.Error("a padded product id must still match")
	}
	if stopped[somsa] {
		t.Error("a dish linked to a sellable product must stay on sale")
	}
	if stopped[unmapped] {
		t.Error("an unlinked dish can never be stopped by the till")
	}
}

// Nothing stopped is a real answer, and it has to arrive as an empty list rather
// than nil: it is written straight into the branch document, and `null` there
// reads back as "no list" on a screen that renders one.
func TestStoppedMenuItemsEmptyIsNotNil(t *testing.T) {
	got := stoppedMenuItems([]pos.Product{{ID: "a"}}, []models.POSMapping{
		{MenuItemID: primitive.NewObjectID(), POSProductID: "a"},
	})
	if got == nil {
		t.Fatal("expected an empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected nothing stopped, got %v", got)
	}
}

// The two lists are separate writers on purpose (see posstop.go), and the site
// asks one question of both. A dish stopped by either is off sale; only the
// till's half is un-liftable from the counter.
func TestBranchSoldOutCombinesBothLists(t *testing.T) {
	manual := primitive.NewObjectID()
	fromPOS := primitive.NewObjectID()
	fromStock := primitive.NewObjectID()
	other := primitive.NewObjectID()
	b := &models.Branch{
		SoldOut:      []primitive.ObjectID{manual},
		POSSoldOut:   []primitive.ObjectID{fromPOS},
		StockSoldOut: []primitive.ObjectID{fromStock},
	}
	// ⚠️ Three writers now: the person at the counter, the kitchen system, and
	// the arithmetic over the store. Each owns its own list — merged they would
	// undo one another — and the site asks only whether any of them names the
	// dish.
	if !b.IsSoldOut(manual) || !b.IsSoldOut(fromPOS) || !b.IsSoldOut(fromStock) {
		t.Error("every list must take a dish off sale")
	}
	if b.IsSoldOut(other) {
		t.Error("a dish in no list is on sale")
	}
	if b.IsPOSSoldOut(manual) || b.IsPOSSoldOut(fromStock) {
		t.Error("a hand-marked or store-stopped dish is not the till's to hold")
	}
	if !b.IsPOSSoldOut(fromPOS) {
		t.Error("the till's half must be identifiable on its own")
	}
	// ⚠️ And the store's half likewise: a dish stopped because the shelf is
	// empty cannot be put back with the counter's toggle, so the panel has to
	// be able to tell the two apart before it draws the button.
	if b.IsStockSoldOut(manual) || b.IsStockSoldOut(fromPOS) {
		t.Error("only the stock sync's own list is the store's")
	}
	if !b.IsStockSoldOut(fromStock) {
		t.Error("the store's half must be identifiable on its own")
	}
}
