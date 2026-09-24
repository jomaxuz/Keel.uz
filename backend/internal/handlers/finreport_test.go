package handlers

import (
	"strings"
	"testing"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func paid(total int, method, typ string) models.Order {
	return models.Order{
		Total: total, PaymentMethod: method, Type: typ,
		Status: models.StatusDelivered, PaymentStatus: models.PayPaid,
	}
}

// The three questions the report keeps apart, and why they must stay apart.
func TestFinanceSeparatesInPendingAndRefunded(t *testing.T) {
	orders := []models.Order{
		paid(100_000, models.ProviderCash, "delivery"),
		paid(50_000, models.ProviderPayme, "pickup"),
		// Placed, cooking, nobody has paid: real, and not takings.
		{Total: 80_000, Status: models.StatusPreparing, PaymentStatus: models.PayUnpaid},
		// Cancelled: neither takings nor pending.
		{Total: 999_000, Status: models.StatusCancelled},
		// Money that came in and went back out.
		{Total: 30_000, Status: models.StatusCancelled, PaymentStatus: models.PayRefunded},
	}
	_, in, out, pending, _ := financeLines(orders, "uz", nil)

	if in != 150_000 {
		t.Errorf("takings = %d, want 150 000", in)
	}
	if pending != 80_000 {
		t.Errorf("pending = %d, want 80 000", pending)
	}
	// ⚠️ A cancelled order that was never paid contributes nothing anywhere.
	// The 999 000 above must not appear in any of the three figures.
	//
	// ⚠️ **Nor does the refund, and this test used to say the opposite** (out =
	// 30 000). The refunded sale is not in the takings, so subtracting its
	// refund took it off twice — "in − out" read 120 000 with 150 000 in the
	// drawer. It is shown as its own info line instead.
	if out != 0 {
		t.Errorf("out = %d, want 0 — the refund is already outside the takings", out)
	}
}

// ⚠️ Discounts and points are not outgoings: no money left the till, it never
// arrived. The takings line is already net of them, so subtracting them again
// would report the campaigns twice.
func TestDiscountsAreNotCosts(t *testing.T) {
	o := paid(70_000, models.ProviderCash, "delivery")
	o.DiscountTotal = 20_000
	o.PointsSpent = 10_000

	lines, in, out, _, _ := financeLines([]models.Order{o}, "uz", nil)
	if in != 70_000 {
		t.Errorf("takings = %d, want the 70 000 actually charged", in)
	}
	if out != 0 {
		t.Errorf("out = %d, want 0 — nothing was handed out", out)
	}
	// They are still reported, because an owner wants to know what the
	// campaigns gave away — as information, not as a cost.
	var sawDiscount, sawPoints bool
	for _, l := range lines {
		if l.Kind != "info" {
			continue
		}
		if l.Amount == 20_000 {
			sawDiscount = true
		}
		if l.Amount == 10_000 {
			sawPoints = true
		}
	}
	if !sawDiscount || !sawPoints {
		t.Error("discounts and points must still be shown, as information")
	}
}

// The channel and payment splits are parts of the takings, not extra income —
// so they are marked as sub-lines and must add back up to it.
func TestChannelSplitAddsUpToTakings(t *testing.T) {
	lines, in, _, _, _ := financeLines([]models.Order{
		paid(100_000, models.ProviderCash, "delivery"),
		paid(60_000, models.ProviderCash, "pickup"),
		paid(40_000, models.ProviderClick, "dinein"),
	}, "uz", nil)
	if in != 200_000 {
		t.Fatalf("takings = %d", in)
	}
	channels := 0
	for _, l := range lines {
		if l.Sub && (l.Label == "— yetkazib berish" || l.Label == "— olib ketish" || l.Label == "— stolda") {
			channels += l.Amount
		}
	}
	if channels != in {
		t.Errorf("channels add to %d but takings are %d", channels, in)
	}
	// Every split line must be marked Sub, or the panel would render the parts
	// and the whole as peers that can be added together.
	for _, l := range lines {
		if l.Label[:1] == "—" && !l.Sub {
			t.Errorf("%q is a part but not marked as one", l.Label)
		}
	}
}

// The one sentence that keeps this report from being read as profit.
func TestFinanceNoteRefusesToClaimProfit(t *testing.T) {
	note := financeNote("uz", false)
	if len(note) == 0 {
		t.Fatal("no note")
	}
	// The word must be there, negated: this figure will end up in front of a
	// bank or a tax inspector, and "kirim − chiqim" looks exactly like profit.
	if !contains(note, "foyda") {
		t.Error("the note must say what this is not")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ⚠️ **The cost of food is information, never an outgoing.** This report is
// cash movement: the ingredients were paid for when they were bought, and if
// that purchase was recorded it is already in the outgoings. Subtracting the
// cost of goods here as well would count the same money twice and leave
// "in − out" meaning nothing.
func TestFoodCostDoesNotMoveTheCashTotals(t *testing.T) {
	dish := primitive.NewObjectID()
	o := models.Order{
		Status:        models.StatusDelivered,
		PaymentStatus: models.PayPaid,
		Total:         100000,
		Items: []models.OrderItem{
			{MenuItemID: dish, Qty: 2, Price: 50000},
		},
	}
	lines, in, out, _, covered := financeLines(
		[]models.Order{o}, "uz", fixedCosts(map[primitive.ObjectID]int{dish: 20000}))

	if in != 100000 || out != 0 {
		t.Fatalf("in=%d out=%d — the food cost moved the cash figures", in, out)
	}
	var cogs, margin *finLine
	for i := range lines {
		switch lines[i].Label {
		case "Sotilgan taomlar tannarxi":
			cogs = &lines[i]
		case "Yalpi foyda (taxminiy)":
			margin = &lines[i]
		}
	}
	if cogs == nil || cogs.Amount != 40000 || cogs.Kind != "info" {
		t.Fatalf("cogs line: %+v", cogs)
	}
	if margin == nil || margin.Amount != 60000 || margin.Kind != "info" {
		t.Fatalf("margin line: %+v", margin)
	}
	if covered != 100000 {
		t.Fatalf("covered=%d", covered)
	}
}

// ⚠️ **The margin is computed against the costed part of the takings only.** A
// menu where a third of the dishes have a cost would otherwise show the other
// two thirds as pure profit — the most flattering wrong answer available, on
// the report somebody eventually puts in a bank application.
func TestMarginNeverTreatsUncostedDishesAsFreeMoney(t *testing.T) {
	priced := primitive.NewObjectID()
	unpriced := primitive.NewObjectID()
	o := models.Order{
		Status:        models.StatusDelivered,
		PaymentStatus: models.PayPaid,
		Total:         100000,
		Items: []models.OrderItem{
			{MenuItemID: priced, Qty: 1, Price: 40000},
			{MenuItemID: unpriced, Qty: 1, Price: 60000},
		},
	}
	lines, in, _, _, covered := financeLines(
		[]models.Order{o}, "uz", fixedCosts(map[primitive.ObjectID]int{priced: 15000}))

	var margin int
	for _, l := range lines {
		if l.Label == "Yalpi foyda (taxminiy)" {
			margin = l.Amount
		}
	}
	// 40 000 − 15 000, and nothing at all from the dish nobody priced.
	if margin != 25000 {
		t.Fatalf("margin=%d, want 25000", margin)
	}
	if covered != 40000 {
		t.Fatalf("covered=%d, want 40000", covered)
	}
	// And the report says so, in a sentence, rather than leaving the reader to
	// work out that it describes 40% of the evening.
	note := financeCostNote("uz", covered, in)
	if note == "" || !strings.Contains(note, "40%") {
		t.Fatalf("note=%q", note)
	}
}

// ⚠️ The warning stays once dishes are priced; only its reason changes. Rent,
// tax and most wages are still not in this system, so "in − out" is still not
// profit — but telling an owner who has just spent an evening pricing the menu
// that "the system holds no dish cost" is the panel failing to notice.
func TestTheWarningStopsClaimingThereAreNoCosts(t *testing.T) {
	none := financeNote("uz", false)
	some := financeNote("uz", true)

	if !strings.Contains(none, "tannarxi yo'q") {
		t.Fatal("the uncosted note no longer explains why this is not profit")
	}
	if strings.Contains(some, "tannarxi yo'q") {
		t.Fatal("the report still says there are no costs after they were entered")
	}
	// Both still refuse to be read as a profit report.
	for _, n := range []string{none, some} {
		if !strings.Contains(n, "foyda hisoboti EMAS") {
			t.Fatalf("a version of the note dropped the warning: %q", n)
		}
	}
}

// A refunded sale is out of the takings, so it must not also be an outgoing:
// the drawer holds the 100, not 60.
func TestFinanceRefundIsNotSubtractedTwice(t *testing.T) {
	sold := models.Order{Status: models.StatusDelivered, PaymentStatus: models.PayPaid, Total: 100, PaymentMethod: "cash", Type: "dinein"}
	refunded := models.Order{Status: models.StatusDelivered, PaymentStatus: models.PayRefunded, Total: 40, PaymentMethod: "cash", Type: "dinein"}
	_, in, out, _, _ := financeLines([]models.Order{sold, refunded}, "uz", nil)
	if in-out != 100 {
		t.Fatalf("in=%d out=%d: net %d, the drawer holds 100", in, out, in-out)
	}
}
