package models

import "testing"

// The ladder moved from a flat 1 000 to 800/560/400/300, and every tenant
// created before that still holds the old figure in `pricePerOrder`. The
// invoice never read it — `PriceForOrders` uses the ladder — but the nightly
// estimate did, so the console reported takings a quarter above what those same
// customers were billed.
//
// Sealed here rather than in the aggregator because the defect is a precedence
// rule, not a query: the stored mirror must lose to the live ladder, and the
// two callers must not be able to answer it differently.
func TestEntryRatePrefersLadderOverStoredPrice(t *testing.T) {
	platform := []PriceTier{
		{UpTo: 3000, Price: 800}, {UpTo: 15000, Price: 560},
		{UpTo: 50000, Price: 400}, {UpTo: 0, Price: 300},
	}

	stale := Tenant{PricePerOrder: 1000}
	if got := stale.EntryRate(platform); got != 800 {
		t.Errorf("stale mirror won: EntryRate = %d, want 800", got)
	}

	// A customer who negotiated their own ladder keeps it: the platform's
	// entry band must not quietly reprice a signed rate.
	own := Tenant{
		PricePerOrder: 1000,
		PriceTiers:    []PriceTier{{UpTo: 0, Price: 300}},
	}
	if got := own.EntryRate(platform); got != 300 {
		t.Errorf("negotiated ladder ignored: EntryRate = %d, want 300", got)
	}

	// ⚠️ With no ladder anywhere the stored flat rate is all there is, and it
	// is still correct — this is the pre-ladder world, and the fallback is what
	// keeps an install that predates tiers billing at the price it agreed.
	if got := stale.EntryRate(nil); got != 1000 {
		t.Errorf("flat fallback lost: EntryRate = %d, want 1000", got)
	}
}

// The estimate and the invoice are allowed to differ — the invoice recomputes
// from the period total — but they must differ only in *which band*, never in
// which table they read. A tenant with its own ladder must not be estimated at
// the platform's entry price.
func TestLadderOrDefaultPrecedence(t *testing.T) {
	platform := []PriceTier{{UpTo: 0, Price: 800}}
	own := []PriceTier{{UpTo: 0, Price: 300}}

	if got := (Tenant{}).LadderOrDefault(platform); len(got) != 1 || got[0].Price != 800 {
		t.Errorf("empty tenant did not fall back to the platform ladder: %v", got)
	}
	if got := (Tenant{PriceTiers: own}).LadderOrDefault(platform); got[0].Price != 300 {
		t.Errorf("tenant ladder overridden by platform: %v", got)
	}
}
