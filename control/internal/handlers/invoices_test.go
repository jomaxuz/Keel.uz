package handlers

import (
	"testing"
	"time"

	"keel-control/internal/models"
)

// The ledger's arithmetic, pinned where it can be wrong in one month of the
// year and right in the other eleven.

// An invoice bills the period that has *finished*. Getting this off by one
// window either charges a restaurant for orders it has not taken yet, or bills
// the same month twice.
func TestPreviousPeriodIsTheWindowThatClosed(t *testing.T) {
	sub := time.Date(2026, 3, 17, 0, 0, 0, 0, time.Local)
	tenant := models.Tenant{
		Status: models.StatusActive, SubscribedAt: &sub,
		CreatedAt: sub,
	}
	// Mid-cycle: the current window is 17 Aug → 17 Sep, so the closed one is
	// 17 Jul → 17 Aug.
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.Local)
	from, to := previousPeriod(tenant, now)
	if from != "2026-07-17" || to != "2026-08-17" {
		t.Fatalf("previousPeriod = %s → %s, want 2026-07-17 → 2026-08-17", from, to)
	}
	// The boundary is shared with the current window, half-open: no day is
	// counted on two invoices and none is skipped.
	if cur := tenantPeriod(tenant, now); cur.From != to {
		t.Fatalf("gap between invoices: previous ends %s, current starts %s", to, cur.From)
	}
}

// A customer anchored on the 31st must not walk backwards through short
// months — the clamp is measured from the anchor, never from the last result.
func TestPreviousPeriodKeepsTheDayOfMonthClamp(t *testing.T) {
	sub := time.Date(2026, 1, 31, 0, 0, 0, 0, time.Local)
	tenant := models.Tenant{
		Status: models.StatusActive, SubscribedAt: &sub, CreatedAt: sub,
	}
	// Current window starts 31 Mar; the one before it started 28 Feb.
	now := time.Date(2026, 4, 10, 0, 0, 0, 0, time.Local)
	from, to := previousPeriod(tenant, now)
	if from != "2026-02-28" || to != "2026-03-31" {
		t.Fatalf("previousPeriod = %s → %s, want 2026-02-28 → 2026-03-31", from, to)
	}
}

// Money arrives in pieces, and only the pieces answer "what did we receive".
func TestCollectedAndOutstanding(t *testing.T) {
	inv := models.Invoice{Amount: 300_000}
	if inv.Outstanding() != 300_000 {
		t.Fatalf("a fresh invoice owes its full amount, got %d", inv.Outstanding())
	}
	inv.Paid = []models.InvoicePayment{
		{Amount: 100_000, Method: models.PayCash, ReceivedBy: "jo"},
		{Amount: 50_000, Method: models.PayCash, ReceivedBy: "jo"},
	}
	if inv.Collected() != 150_000 {
		t.Fatalf("Collected = %d, want 150000", inv.Collected())
	}
	if inv.Outstanding() != 150_000 {
		t.Fatalf("Outstanding = %d, want 150000", inv.Outstanding())
	}
	// Overpaying is a conversation, not a negative debt on a list of what
	// customers owe — otherwise one generous payment cancels somebody else's
	// arrears in the total.
	inv.Paid = append(inv.Paid, models.InvoicePayment{Amount: 500_000})
	if inv.Outstanding() != 0 {
		t.Fatalf("Outstanding = %d after overpayment, want 0", inv.Outstanding())
	}
	// A voided invoice owes nothing regardless of what was recorded against it.
	inv.Status = models.InvoiceVoid
	inv.Paid = nil
	if inv.Outstanding() != 0 {
		t.Fatalf("a voided invoice must owe nothing, got %d", inv.Outstanding())
	}
}
