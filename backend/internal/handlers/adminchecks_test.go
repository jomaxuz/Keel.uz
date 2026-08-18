package handlers

import (
	"strings"
	"testing"
	"time"

	"restaurant-backend/internal/models"
)

// ⚠️ **The reports must keep counting till sales**, and the only thing making
// that true is an absence: `ordersInRange` filters on the period and the scope
// and nothing else, so a dining-room sale is in the sales report, the channel
// report and the financial report for the same reason a delivery is.
//
// This is worth a test precisely because it is an absence. The orders board
// grew its `check: {$exists: false}` line for a good reason, and the obvious
// next move — for anybody working on either screen — is to "make the reports
// consistent" by copying it. That would take the dining room out of the
// restaurant's own revenue: the totals would drop by whatever the room takes,
// the page would look entirely normal, and the report is the number an owner
// carries to the bank.
func TestReportsStillCountTillSales(t *testing.T) {
	src := readSource(t, "salesreport.go")
	fn := between(t, src, "func (h *Handler) ordersInRange", "\n}\n")

	if strings.Contains(fn, `"check"`) {
		t.Fatal("the reports now filter on `check` — till sales dropped out of the revenue")
	}
}

// The sales list is the other half of that: a check is on the orders board or
// on this page, and never on neither. Both screens split on the same field,
// from opposite sides.
func TestSalesListTakesExactlyWhatTheBoardLeaves(t *testing.T) {
	src := readSource(t, "adminchecks.go")
	fn := between(t, src, "func (h *Handler) AdminListChecks", "\n}\n")

	if !strings.Contains(fn, `filter["check"] = bson.M{"$exists": true}`) {
		t.Fatal("the sales list no longer selects till checks")
	}
	// The period has to be cut on the same field the reports use, or the list
	// and the report disagree about which month a sale belongs to.
	if !strings.Contains(fn, `filter["createdAt"] = rng`) {
		t.Fatal("the sales list cuts its period on a different field from the reports")
	}
	if !strings.Contains(fn, "h.orderScope(r)") {
		t.Fatal("the sales list is not scoped — a manager could read another branch")
	}
}

func closedRow(total, guests int, method, table string) checkRow {
	return checkRow{Total: total, Guests: guests, PaymentMethod: method, Table: table}
}

func TestTotalsCountTheWholeSetNotThePage(t *testing.T) {
	rows := []checkRow{
		closedRow(100000, 4, "cash", "7"),
		closedRow(60000, 2, "card", "7"),
		closedRow(40000, 0, "cash", ""), // counter, nobody counted heads
		{Total: 999000, Open: true, Table: "3", Guests: 6},
	}
	got := totalsOf(rows)

	if got.Checks != 4 || got.Open != 1 {
		t.Fatalf("checks=%d open=%d", got.Checks, got.Open)
	}
	// ⚠️ An open table has taken no money. Counting its running tab as sales
	// is how a screen reports an evening before it happened.
	if got.Sales != 200000 {
		t.Fatalf("sales=%d, want 200000 (the open table must not be in it)", got.Sales)
	}
	if got.Cash != 140000 || got.Card != 60000 {
		t.Fatalf("cash=%d card=%d", got.Cash, got.Card)
	}
	if got.Hall != 3 || got.Counter != 1 {
		t.Fatalf("hall=%d counter=%d", got.Hall, got.Counter)
	}
	// Average over the three closed checks, not all four.
	if got.AvgCheck != 200000/3 {
		t.Fatalf("avgCheck=%d", got.AvgCheck)
	}
	// ⚠️ Per guest over the closed checks that named a number of guests: the
	// 40 000 counter sale contributes its money and no heads, which is the
	// honest reading of "nobody was sitting there".
	if got.AvgGuest != 200000/6 {
		t.Fatalf("avgGuest=%d", got.AvgGuest)
	}
}

func TestTotalsSayNothingRatherThanGuess(t *testing.T) {
	// Every check open: no sales, and no average invented from a division by
	// zero or by the open tables.
	got := totalsOf([]checkRow{{Open: true}, {Open: true}})
	if got.AvgCheck != 0 || got.AvgGuest != 0 || got.Sales != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestVoidedFoodIsNotSold(t *testing.T) {
	now := time.Now()
	o := &models.Order{
		Number:      "MRC-1",
		TableNumber: "7",
		CreatedAt:   now,
		Items: []models.OrderItem{
			{Name: "Lag'mon", Qty: 2, Price: 32000},
			{Name: "Salat", Qty: 1, Price: 18000,
				Void: &models.CheckLineVoid{At: now, Reason: "mehmon voz kechdi"}},
		},
		Check: &models.OrderCheck{OpenedAt: now, ServerName: "Aziz", Guests: 2},
	}
	row := checkRowOf(o)

	// Two live portions; the voided starter is on the check and not in the
	// count — that distinction is the whole point of a void.
	if row.Items != 2 {
		t.Fatalf("items=%d, want 2", row.Items)
	}
	if row.Table != "7" || row.Server != "Aziz" {
		t.Fatalf("%+v", row)
	}
}

// ⚠️ Opening one sale by id is where the branch lens is easiest to lose: the
// list is scoped, so it feels done. It is not — an id from a list a manager
// *can* see is not the only id they can type, and the detail view carries the
// takings, the voids and the guest count of whatever it is given.
func TestOpeningOneSaleKeepsTheBranchInTheFilter(t *testing.T) {
	src := readSource(t, "adminchecks.go")
	fn := between(t, src, "func (h *Handler) AdminGetCheck", "\n}\n")

	if !strings.Contains(fn, "h.scopedOrderFilter(r, id)") {
		t.Fatal("the detail view selects by id alone — another branch's sale is one paste away")
	}
	// A delivery is not a till sale, and neither is out of scope: both are the
	// same 404, so no id tells the caller which kind of document exists.
	if !strings.Contains(fn, `filter["check"] = bson.M{"$exists": true}`) {
		t.Fatal("the detail view will open any order, not only a till check")
	}
	if !strings.Contains(fn, "StatusNotFound") {
		t.Fatal("out of scope must be a 404")
	}
}

// ⚠️ The voided line is the most important row on the screen: a void that
// leaves no trace is the oldest way to take money out of a restaurant. It is on
// the bill's face and out of its total, and nothing about that is optional.
func TestVoidedLineIsShownAndCountsNothing(t *testing.T) {
	src := readSource(t, "adminchecks.go")
	fn := between(t, src, "func (h *Handler) AdminGetCheck", "\n}\n")

	if strings.Contains(fn, "it.Live()") || strings.Contains(fn, "continue") {
		t.Fatal("voided lines are being dropped from the detail view")
	}
	if !strings.Contains(fn, "line.Sum = it.Price * it.Qty") {
		t.Fatal("the line sum is no longer written only on the live branch")
	}
}

// ⚠️ A table that asked for two bills had one dinner. Counting both would show
// a busier night than the room had — and, worse, would drag the average check
// towards half of what a table actually spends, which is the number an owner
// prices a menu against.
func TestASplitTableIsOneVisitWithAllOfItsMoney(t *testing.T) {
	whole := closedRow(200000, 4, "cash", "7")
	half := closedRow(120000, 0, "card", "7")
	half.Split = true

	got := totalsOf([]checkRow{whole, half})

	if got.Checks != 1 || got.Splits != 1 {
		t.Fatalf("checks=%d splits=%d, want 1 and 1", got.Checks, got.Splits)
	}
	// ⚠️ Every som is still counted: the guest paid it, and the split was a
	// piece of paper, not a discount.
	if got.Sales != 320000 {
		t.Fatalf("sales=%d, want 320000", got.Sales)
	}
	if got.Cash != 200000 || got.Card != 120000 {
		t.Fatalf("cash=%d card=%d", got.Cash, got.Card)
	}
	if got.AvgCheck != 320000 {
		t.Fatalf("avgCheck=%d, want the table's whole dinner", got.AvgCheck)
	}
}

// ⚠️ The half of a refund nobody sees. A delivery refund drops out of the
// takings by itself, because the money moves `paymentStatus` off `paid`. A
// dining-room sale is `delivered` the moment it is closed — so before this,
// a refunded table went on counting as revenue with nothing disagreeing: the
// guest had the cash back, the drawer was short by it, and the dashboard was
// not.
func TestRefundedMoneyIsNotTakings(t *testing.T) {
	table := models.Order{
		Status:        models.StatusDelivered,
		PaymentStatus: models.PayRefunded,
	}
	if received(table) {
		t.Fatal("a refunded dine-in sale is still being counted as revenue")
	}
	// The ordinary cases are untouched.
	if !received(models.Order{Status: models.StatusDelivered, PaymentStatus: "unpaid"}) {
		t.Fatal("cash collected on delivery stopped counting")
	}
	if !received(models.Order{Status: models.StatusConfirmed, PaymentStatus: models.PayPaid}) {
		t.Fatal("a card payment the bank confirmed stopped counting")
	}
}

func TestRefundLeavesTheSaleAndTakesTheMoney(t *testing.T) {
	sold := closedRow(200000, 2, "cash", "7")
	back := closedRow(90000, 2, "cash", "3")
	back.Refunded = true

	got := totalsOf([]checkRow{sold, back})

	// ⚠️ Both are still checks: the second table ate. Hiding it would make the
	// evening's covers disagree with the room.
	if got.Checks != 2 {
		t.Fatalf("checks=%d, want 2", got.Checks)
	}
	if got.Sales != 200000 || got.Cash != 200000 {
		t.Fatalf("sales=%d cash=%d — refunded money is being counted", got.Sales, got.Cash)
	}
	// Its own line: "sales are down" and "we refunded a table" are different
	// evenings, and only one is about the food.
	if got.Refunded != 90000 || got.RefundedN != 1 {
		t.Fatalf("refunded=%d n=%d", got.Refunded, got.RefundedN)
	}
	if got.AvgCheck != 200000 {
		t.Fatalf("avgCheck=%d — the refunded table must not drag it down", got.AvgCheck)
	}
}

// ⚠️ The drawer correction is the part that is easy to get exactly backwards,
// and both directions are wrong in a way somebody has to count cash to find.
func TestDrawerIsCorrectedOnlyForAnEarlierShift(t *testing.T) {
	src := readSource(t, "adminchecks.go")
	fn := between(t, src, "func (h *Handler) correctDrawer", "\n}\n")

	// Same shift: the expected figure is built from cash sales paid inside it,
	// so the sale leaving `paid` already removes the money. An entry as well
	// subtracts it twice.
	if !strings.Contains(fn, "o.PaidAt.Before(shift.OpenedAt)") {
		t.Fatal("a refund of this shift's own sale is being subtracted twice")
	}
	// A card refund goes back the way it came; nothing leaves the drawer.
	if !strings.Contains(fn, "models.ProviderCash") {
		t.Fatal("card refunds are being taken out of the cash drawer")
	}
	if !strings.Contains(fn, "models.CashOut") {
		t.Fatal("the correction is no longer money leaving the drawer")
	}
}

// ⚠️ A cancelled check is not a quiet zero — the total is still on the
// document, and counting it would book a sale nobody paid for. It is not a
// cover either: a table that walked out did not eat.
func TestACancelledCheckIsCountedAsNothing(t *testing.T) {
	sold := closedRow(150000, 2, "cash", "7")
	gone := closedRow(90000, 4, "cash", "3")
	gone.Cancelled = true

	got := totalsOf([]checkRow{sold, gone})

	if got.Checks != 1 || got.Cancelled != 1 {
		t.Fatalf("checks=%d cancelled=%d", got.Checks, got.Cancelled)
	}
	if got.Sales != 150000 || got.Guests != 2 {
		t.Fatalf("sales=%d guests=%d", got.Sales, got.Guests)
	}
}
