package handlers

import (
	"strings"
	"testing"
	"time"
)

// ⚠️ **A box expiring this evening must read the same at nine in the morning
// and at ten at night.** Subtracting timestamps says "0 days left" and then
// "-1" on one working day, which is two different facts about one box — and the
// second one sends somebody to a shelf about something that is still fine.
func TestTheCountdownIsCountedInDaysNotHours(t *testing.T) {
	loc := time.Local
	morning := time.Date(2026, 9, 6, 9, 0, 0, 0, loc)
	night := time.Date(2026, 9, 6, 22, 0, 0, 0, loc)
	expires := time.Date(2026, 9, 6, 18, 0, 0, 0, loc)

	if got := daysBetween(morning, expires); got != 0 {
		t.Errorf("morning: %d days left, want 0", got)
	}
	// ⚠️ Still today, even though the hour has passed.
	if got := daysBetween(night, expires); got != 0 {
		t.Errorf("night: %d days left, want 0 — the box expires today either way", got)
	}
	if got := daysBetween(morning, expires.AddDate(0, 0, 30)); got != 30 {
		t.Errorf("thirty days ahead read as %d", got)
	}
	// Already past reads negative, which is what turns the row red.
	if got := daysBetween(morning, expires.AddDate(0, 0, -3)); got != -3 {
		t.Errorf("three days ago read as %d", got)
	}
}

// ⚠️ **This screen may never claim what is on the shelf.** Consumption keys on
// the ingredient rather than on the box — the rule the whole stockroom is built
// on, and the one thing the shop work was told not to change. A quantity here
// is what a delivery recorded; presenting it as a balance would be inventing a
// number that is right most of the time, and a pharmacist who finds one stock
// figure wrong stops believing every other figure on the screen.
func TestTheExpiryScreenReadsDeliveriesAndSubtractsNothing(t *testing.T) {
	src := readSource(t, "expiry.go")
	// ⚠️ **Calls, not words.** The first version of this matched the file text
	// and failed on the comment explaining why the file does none of this —
	// a guard that fires on its own documentation is a guard somebody deletes.
	for _, bad := range []string{
		"h.Store.StockMovements", "h.Store.Writeoffs", "h.balances(",
	} {
		if strings.Contains(src, bad) {
			t.Errorf("expiry.go calls %s — it would start answering "+
				"\"what is left\", which nothing here can know", bad)
		}
	}
	fn := between(t, src, "func (h *Handler) AdminExpiring", "\n}\n")
	// ⚠️ A delivery of forty lines matches the query because *one* of them
	// expires soon; without the per-line check the screen would list the other
	// thirty-nine as expiring today.
	if !strings.Contains(fn, "l.ExpiresAt == nil || l.ExpiresAt.After(until)") {
		t.Fatal("every line of a matching delivery would be listed, expiring or not")
	}
	// ⚠️ And a delivery with no date is "we did not record it", never "it never
	// expires" — it must not appear on a screen read as a to-do list.
	if !strings.Contains(fn, `"$ne": nil`) {
		t.Error("deliveries with no expiry date would be listed")
	}
}
