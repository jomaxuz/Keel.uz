package handlers

import (
	"testing"

	"restaurant-backend/internal/models"
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
	_, in, out, pending := financeLines(orders)

	if in != 150_000 {
		t.Errorf("takings = %d, want 150 000", in)
	}
	if pending != 80_000 {
		t.Errorf("pending = %d, want 80 000", pending)
	}
	// ⚠️ A cancelled order that was never paid contributes nothing anywhere.
	// The 999 000 above must not appear in any of the three figures.
	if out != 30_000 {
		t.Errorf("out = %d, want only the 30 000 refund", out)
	}
}

// ⚠️ Discounts and points are not outgoings: no money left the till, it never
// arrived. The takings line is already net of them, so subtracting them again
// would report the campaigns twice.
func TestDiscountsAreNotCosts(t *testing.T) {
	o := paid(70_000, models.ProviderCash, "delivery")
	o.DiscountTotal = 20_000
	o.PointsSpent = 10_000

	lines, in, out, _ := financeLines([]models.Order{o})
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
	lines, in, _, _ := financeLines([]models.Order{
		paid(100_000, models.ProviderCash, "delivery"),
		paid(60_000, models.ProviderCash, "pickup"),
		paid(40_000, models.ProviderClick, "dinein"),
	})
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
	note := financeNote()
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
