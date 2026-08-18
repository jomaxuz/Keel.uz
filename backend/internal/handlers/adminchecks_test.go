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
