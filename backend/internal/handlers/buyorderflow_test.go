package handlers

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

// ⚠️ **Empty means the market, and the whole catalogue of every existing
// install is empty.** Reading the zero value the other way would route every
// ingredient in the product to a storekeeper who has never opened the screen —
// silently, on the deploy that adds the field, and with every request looking
// answered-in-progress from every panel.
func TestAnIngredientWithNoSourceGoesToTheMarket(t *testing.T) {
	if got := (models.Ingredient{}).From(); got != models.SourceMarket {
		t.Fatalf("an unset source reads as %q — every old catalogue would reroute", got)
	}
	if got := (models.Ingredient{Source: models.SourceStore}).From(); got != models.SourceStore {
		t.Fatalf("a store ingredient reads as %q", got)
	}
	// Anything the catalogue could not have meant is the market too: a third
	// destination has no queue on any phone, and a request sent there would sit
	// unanswered while looking exactly like one somebody was dealing with.
	if got := (models.Ingredient{Source: "warehouse"}).From(); got != models.SourceMarket {
		t.Fatalf("an unknown source reads as %q rather than as the market", got)
	}
}

// ⚠️ **A row nobody retyped keeps what it was told.** Accepting a list without
// touching anything means "this is right" — the ordinary case — and a default of
// zero would empty the whole list of whoever was quickest to agree with it.
func TestAcceptingWithoutRetypingKeepsWhatWasSent(t *testing.T) {
	line := models.ShoppingLine{GotQty: 5}
	if line.Took() != 5 {
		t.Fatalf("an untouched line arrives as %v, not as what was sent", line.Took())
	}
	if line.Short() != 0 {
		t.Fatal("a line nobody counted reports a shortfall")
	}
	four := 4.0
	line.TookQty = &four
	if line.Took() != 4 || line.Short() != 1 {
		t.Fatalf("a counted line reads %v with %v short", line.Took(), line.Short())
	}
	// ⚠️ Zero is a real answer, distinct from "nobody counted": it says the bag
	// arrived empty, which is the sentence somebody has to be asked about.
	none := 0.0
	line.TookQty = &none
	if line.Took() != 0 || line.Short() != 5 {
		t.Fatal("counting zero is read as not having counted")
	}
}

// ⚠️ **The tick on the employee's card only ever adds.** A role that grants
// `buyorder` must not be withdrawn by leaving the box unticked — one switch that
// sometimes grants and sometimes revokes has to be read against a second
// document, and that reading gets done wrong on the day somebody is in a hurry.
func TestTheShoppingTickGrantsAndNeverRevokes(t *testing.T) {
	byRole := models.Staff{
		IsActive: true, RoleApplied: true, Perms: []string{models.PermBuyOrder},
	}
	if !byRole.Can(models.PermBuyOrder) {
		t.Fatal("the untickled box took away what the role granted")
	}
	byPerson := models.Staff{IsActive: true, RoleApplied: true, CanBuyOrder: true}
	if !byPerson.Can(models.PermBuyOrder) {
		t.Fatal("a barman ticked by hand cannot write a list")
	}
	// ⚠️ And it grants exactly one thing. A tick that also opened the till would
	// be the reason nobody ever ticks it.
	for _, perm := range []string{
		models.PermBuy, models.PermStock, models.PermStockIssue,
		models.PermCashier, models.PermVoid,
	} {
		if byPerson.Can(perm) {
			t.Fatalf("the shopping tick also granted %q", perm)
		}
	}
	// A dismissed account holds nothing, tick or no tick — the gap the kitchen
	// screen once had.
	gone := models.Staff{IsActive: false, CanBuyOrder: true}
	if gone.Can(models.PermBuyOrder) {
		t.Fatal("somebody dismissed this morning still writes lists tonight")
	}
}

// ⚠️ **Shipping is not a delivery, and accepting is.** What reaches a shelf is
// what somebody at the restaurant counted; a purchase written when the buyer
// leaves the market would put food on a shelf while it is still in a bag on a
// bus, and every later correction would be indistinguishable from a theft.
func TestTheShelfMovesWhenSomebodySignsForIt(t *testing.T) {
	flow := readSource(t, "buyorderflow.go")
	accept := between(t, flow, "func (h *Handler) StaffAcceptBuyOrder", "\n}\n")
	if !strings.Contains(accept, "l.Took()") {
		t.Fatal("the delivery is written from what the buyer claimed, not from what was counted")
	}
	if !strings.Contains(accept, "models.ShoppingDone") {
		t.Fatal("accepting does not close the list")
	}
}

// ⚠️ **A store issue never writes a price.** The goods were bought once; a
// second price here enters the history as a purchase and recosts every dish the
// ingredient goes into — the loudest possible way for an errand between two
// rooms to be wrong.
func TestPickingFromTheStoreWritesNoPrice(t *testing.T) {
	mark := between(t, readSource(t, "buyorders.go"),
		"func (h *Handler) StaffMarkBuyOrderLine", "\n}\n")
	if !strings.Contains(mark, "order.FromStore()") {
		t.Fatal("a storekeeper's tick is priced like a market run")
	}
}

// ⚠️ **A branch never supplies itself.** Read literally that builds a dispatch
// whose two ends are the same shelf — the same kilo subtracted and added, and a
// slip nobody can accept.
func TestABranchNeverSuppliesItself(t *testing.T) {
	src := readSource(t, "buyorders.go")
	fn := between(t, src, "func (h *Handler) supplyBranchFor", "\n}\n")
	if !strings.Contains(fn, "b.SupplyBranchID == branch") {
		t.Fatal("a branch pointed at itself would ship a van to its own shelf")
	}
}

// ⚠️ **Only the market half is subtracted from the next suggestion.** Something
// a buyer is carrying home has been asked for and is not on the shelf yet; a
// request answered from the store room is already *in* the balance the
// suggestion was computed from, and counting it twice quietly stops reordering
// the things a restaurant moves between its own rooms most often.
func TestOnlyTheMarketHalfDiscountsTheNextOrder(t *testing.T) {
	fn := between(t, readSource(t, "orderplan.go"), "func (h *Handler) requestedQty", "\n}\n")
	if !strings.Contains(fn, "models.SourceStore") {
		t.Fatal("a store request is subtracted from the shopping list as well")
	}
	if !strings.Contains(fn, "models.ShoppingShipped") {
		t.Fatal("goods somebody is carrying home are suggested again")
	}
}

// ⚠️ **The list splits itself, and the person writing it never chooses.**
// Sorting a request into "market" and "store" is knowledge about the store's
// contents, which is exactly what a barman's job does not involve — and the
// sorting would be wrong on the mornings it mattered.
func TestOneRequestBecomesOneListPerPlace(t *testing.T) {
	create := between(t, readSource(t, "buyorders.go"),
		"func (h *Handler) StaffCreateBuyOrder", "\n}\n")
	if !strings.Contains(create, "in.From()") {
		t.Fatal("the split asks the request where a line goes rather than the catalogue")
	}
	if !strings.Contains(create, "GroupID: group") {
		t.Fatal("the two halves are not tied back to the one request")
	}
}

// ⚠️ **A name the catalogue does not have goes to the market**, because it is
// something nobody has ever put on a shelf here. Routed to a storekeeper it
// would sit unanswered while looking, on every screen, exactly like a request
// somebody was dealing with.
func TestAnUnknownNameIsSomethingToBuy(t *testing.T) {
	line := models.ShoppingLine{Name: "kinza", IngredientID: primitive.NilObjectID}
	if !line.IngredientID.IsZero() {
		t.Fatal("the fixture is wrong")
	}
	create := between(t, readSource(t, "buyorders.go"),
		"func (h *Handler) StaffCreateBuyOrder", "\n}\n")
	if !strings.Contains(create, "source := models.SourceMarket") {
		t.Fatal("a line the catalogue does not know defaults somewhere else")
	}
}
