package handlers

import (
	"testing"
	"time"

	"keel-control/internal/models"
)

// Free terms are the promise an early customer was brought in on, and the one
// billing mistake that costs a customer rather than money is charging — or
// switching off — somebody who was told they would not be.

// `day` is shared with billing_test.go — the same helper, and deliberately not
// a second one: two notions of "a day" in one package is how the tests stop
// agreeing with each other.

// Nil end date means forever. Read as "expired at the zero time", an anchor
// customer promised free service for good would be billed from day one.
func TestFreeWithNoEndDateIsForever(t *testing.T) {
	tn := models.Tenant{Free: true, FreeReason: "birinchi tarmoq"}
	for _, when := range []time.Time{
		day(2026, time.August, 6),
		day(2030, time.January, 1),
	} {
		if !tn.FreeAt(when) {
			t.Fatalf("FreeAt(%s) = false, want free forever", when.Format("2006-01-02"))
		}
		if got := tn.ChargeFor(500_000, when); got != 0 {
			t.Fatalf("ChargeFor = %d, want 0", got)
		}
	}
}

func TestFreeEndsOnItsDate(t *testing.T) {
	until := day(2026, time.September, 1)
	tn := models.Tenant{Free: true, FreeReason: "6 oy", FreeUntil: &until}

	if !tn.FreeAt(day(2026, time.August, 31)) {
		t.Error("still free the day before it ends")
	}
	// Half-open, like every other period in this system: the end date is the
	// first paid day, not the last free one.
	if tn.FreeAt(until) {
		t.Error("the end date itself is already chargeable")
	}
	if got := tn.ChargeFor(500_000, until); got != 500_000 {
		t.Errorf("ChargeFor after the end = %d, want the full amount", got)
	}
}

// The flag, not the price. `pricePerOrder: 0` is indistinguishable from a
// cleared field; this has to be what decides.
func TestFreeBeatsDiscountAndFlagIsWhatDecides(t *testing.T) {
	now := day(2026, time.August, 6)
	free := models.Tenant{Free: true, FreeReason: "x", DiscountPercent: 20}
	if got := free.ChargeFor(300_000, now); got != 0 {
		t.Errorf("a free account with a stale discount charged %d, want 0", got)
	}
	notFree := models.Tenant{Free: false, FreeReason: "x", DiscountPercent: 20}
	if got := notFree.ChargeFor(300_000, now); got != 240_000 {
		t.Errorf("ChargeFor = %d, want 240000", got)
	}
}

func TestDiscountBounds(t *testing.T) {
	now := day(2026, time.August, 6)
	cases := map[int]int{0: 100_000, 50: 50_000, 100: 0, 33: 67_000}
	for pct, want := range cases {
		tn := models.Tenant{DiscountPercent: pct}
		if got := tn.ChargeFor(100_000, now); got != want {
			t.Errorf("%d%% → %d, want %d", pct, got, want)
		}
	}
}

// The sweep and the "call them" list must both leave a free customer alone.
// A twelve-restaurant chain brought in on a promise still carries the trial
// end date from the day its account was opened.
func TestFreeCustomerIsNotChasedForMoney(t *testing.T) {
	now := day(2026, time.August, 6)
	ended := day(2026, time.July, 1)
	tn := models.Tenant{
		Status: models.StatusTrial, TrialEndsAt: &ended,
		Free: true, FreeReason: "anchor",
	}
	if a := tenantAttention(tn, now, "", "running"); a.Kind != "" {
		t.Fatalf("attention = %q, want none for a free customer", a.Kind)
	}
	// Without the flag the same row is overdue — proving the flag is what
	// silenced it rather than the dates being wrong.
	tn.Free = false
	if a := tenantAttention(tn, now, "", "running"); a.Kind != AttentionTrialExpired {
		t.Fatalf("attention = %q, want the trial to read as expired", a.Kind)
	}
}

// A suspended customer is still shown, free or not: somebody switched them off
// by hand and that decision must stay visible.
func TestSuspendedFreeCustomerStaysOnTheList(t *testing.T) {
	now := day(2026, time.August, 6)
	tn := models.Tenant{Status: models.StatusSuspended, Free: true, FreeReason: "x"}
	if a := tenantAttention(tn, now, "", "running"); a.Kind != AttentionUnpaid {
		t.Fatalf("attention = %q, want the suspension to stay visible", a.Kind)
	}
}
