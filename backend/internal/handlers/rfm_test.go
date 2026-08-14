package handlers

import (
	"fmt"
	"testing"
	"time"
)

// A base of `n` customers whose last order was `i` days ago, who ordered `i`
// times and spent `i * 10 000`. Enough shape to test the scoring rules without
// pretending to be a real restaurant.
func ladderFacts(n int, now time.Time) map[string]customerFacts {
	out := map[string]customerFacts{}
	for i := 1; i <= n; i++ {
		last := now.AddDate(0, 0, -i)
		out[fmt.Sprintf("u%d", i)] = customerFacts{
			OrdersCount: i, OrdersTotal: i * 10_000, LastOrder: &last,
		}
	}
	return out
}

// ⚠️ Recency is inverted, and getting it wrong is invisible.
//
// The axis is measured in **days since the last order**, where small is good.
// Scoring it like frequency and money would put the customers who left longest
// ago at the top of the screen labelled "champions" — and every number on the
// page would still look plausible, so nobody would check.
func TestRecencyIsInverted(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.Local)
	scale := newRFMScale(ladderFacts(20, now), now)
	if scale == nil {
		t.Fatal("20 customers should be enough to rank")
	}

	yesterday := now.AddDate(0, 0, -1)
	longAgo := now.AddDate(0, 0, -400)

	recent, _ := scoreFor(customerFacts{OrdersCount: 5, OrdersTotal: 50_000, LastOrder: &yesterday}, scale, now)
	gone, _ := scoreFor(customerFacts{OrdersCount: 5, OrdersTotal: 50_000, LastOrder: &longAgo}, scale, now)

	if recent.R <= gone.R {
		t.Fatalf("R: recent=%d, long gone=%d — recency must score higher for the recent one",
			recent.R, gone.R)
	}
	if recent.R != 5 {
		t.Fatalf("a customer who ordered yesterday scored R=%d, want 5", recent.R)
	}
	if gone.R != 1 {
		t.Fatalf("a customer gone for a year scored R=%d, want 1", gone.R)
	}
	// The other two axes are the right way up: more is better.
	if recent.F != gone.F || recent.M != gone.M {
		t.Fatal("frequency and money should be identical for these two")
	}
}

// ⚠️ Identical customers must score identically.
//
// Half a restaurant's base has ordered exactly once. Splitting by position in
// the sorted list hands two people with one order each different frequency
// scores — and therefore different segments — which is indefensible the moment
// the owner sorts the list, and takes the credibility of the whole screen with
// it.
func TestTiesScoreTheSame(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.Local)
	// Fifteen customers, twelve of whom ordered exactly once.
	facts := map[string]customerFacts{}
	for i := 1; i <= 12; i++ {
		last := now.AddDate(0, 0, -3)
		facts[fmt.Sprintf("one%d", i)] = customerFacts{
			OrdersCount: 1, OrdersTotal: 40_000, LastOrder: &last,
		}
	}
	for i := 1; i <= 3; i++ {
		last := now.AddDate(0, 0, -1)
		facts[fmt.Sprintf("many%d", i)] = customerFacts{
			OrdersCount: 20, OrdersTotal: 900_000, LastOrder: &last,
		}
	}
	scale := newRFMScale(facts, now)
	if scale == nil {
		t.Fatal("15 customers should be enough to rank")
	}

	seen := map[int]bool{}
	for id, f := range facts {
		s, ok := scoreFor(f, scale, now)
		if !ok {
			t.Fatalf("%s could not be scored", id)
		}
		if f.OrdersCount == 1 {
			seen[s.F] = true
		}
	}
	if len(seen) != 1 {
		t.Fatalf("customers with one order each got frequency scores %v — identical inputs must score identically", seen)
	}
}

// ⚠️ `atRisk` is tested before `loyal`. A high frequency score with a low
// recency one is a regular who has stopped coming — the single most valuable
// thing this grid can surface. Testing `loyal` first files them as healthy and
// the alert never appears.
func TestAtRiskBeatsLoyal(t *testing.T) {
	// Ordered a lot (F=5), not for a long time (R=1).
	if got := rfmCell(rfmScores{R: 1, F: 5, M: 5}); got != RFMAtRisk {
		t.Fatalf("cell %q for a frequent customer who has gone quiet, want %q", got, RFMAtRisk)
	}
	// Ordered a lot and recently: that one is a champion.
	if got := rfmCell(rfmScores{R: 5, F: 5, M: 5}); got != RFMChampions {
		t.Fatalf("cell %q, want %q", got, RFMChampions)
	}
	// Frequent, middling recency: still loyal, not yet at risk.
	if got := rfmCell(rfmScores{R: 3, F: 5, M: 3}); got != RFMLoyal {
		t.Fatalf("cell %q, want %q", got, RFMLoyal)
	}
}

// Every possible score lands in exactly one cell. Unlike the rule segments,
// these are a partition: a grid position a customer could occupy twice is not
// a position, and the counts would add up to more than the customer base.
func TestEveryScoreHasExactlyOneCell(t *testing.T) {
	known := map[string]bool{}
	for _, cell := range RFMCells {
		known[cell] = true
	}
	for r := 1; r <= 5; r++ {
		for f := 1; f <= 5; f++ {
			for m := 1; m <= 5; m++ {
				cell := rfmCell(rfmScores{R: r, F: f, M: m})
				if !known[cell] {
					t.Fatalf("R%dF%dM%d produced %q, which is not in RFMCells", r, f, m, cell)
				}
			}
		}
	}
}

// ⚠️ Too small a base means no RFM at all, not everybody scoring 1.
//
// Calling the fourth customer of a brand-new restaurant a "champion" is a label
// about the restaurant's age. The same reasoning — and the same threshold — as
// `vipFloor`.
func TestSmallBaseHasNoRanking(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.Local)
	if scale := newRFMScale(ladderFacts(rfmMinBase-1, now), now); scale != nil {
		t.Fatalf("ranked a base of %d customers; the minimum is %d", rfmMinBase-1, rfmMinBase)
	}
	if scale := newRFMScale(ladderFacts(rfmMinBase, now), now); scale == nil {
		t.Fatalf("refused to rank a base of exactly %d", rfmMinBase)
	}
	// And with no scale, a customer simply has no cell — they are not "lost".
	last := now.AddDate(0, 0, -1)
	if got := cellFor(customerFacts{OrdersCount: 3, LastOrder: &last}, nil, now); got != "" {
		t.Fatalf("cell %q without a scale, want empty", got)
	}
}

// Someone with an account and no orders has no position on any axis. Scoring
// them 1/1/1 would put them in "lost", which says they left — they never
// arrived. The rule segments already have `noOrders` for exactly this person.
func TestAccountsWithoutOrdersAreNotRanked(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.Local)
	scale := newRFMScale(ladderFacts(20, now), now)
	if _, ok := scoreFor(customerFacts{}, scale, now); ok {
		t.Fatal("scored a customer who has never ordered")
	}
	// And they do not move the cut points either.
	withEmpties := ladderFacts(20, now)
	for i := range 50 {
		withEmpties[fmt.Sprintf("empty%d", i)] = customerFacts{}
	}
	other := newRFMScale(withEmpties, now)
	if other.Base != scale.Base {
		t.Fatalf("base %d with empty accounts, %d without — they must not count",
			other.Base, scale.Base)
	}
	if other.Frequency != scale.Frequency {
		t.Fatalf("cut points moved from %v to %v because of accounts with no orders",
			scale.Frequency, other.Frequency)
	}
}

// ⚠️ RFM cells are namespaced when used as a campaign audience.
//
// The rule segments already contain `lost`, and it means something different
// there — a fixed 180-day line — from what it means here, which is "bottom
// fifth by recency". Sharing the id would let a campaign aimed at one go
// silently to the other, and the two overlap enough that the count would look
// right.
func TestRFMAudienceIDsAreNamespaced(t *testing.T) {
	if rfmSegmentID(RFMLost) == SegLost {
		t.Fatal("an RFM cell collides with a rule segment id")
	}
	for _, id := range RFMSegmentIDs() {
		if len(id) < len(rfmPrefix) || id[:len(rfmPrefix)] != rfmPrefix {
			t.Fatalf("audience id %q is missing the %q prefix", id, rfmPrefix)
		}
	}
	// And a customer carries both kinds at once: the rules describe them, the
	// cell ranks them.
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.Local)
	last := now.AddDate(0, 0, -1)
	segs := segmentsFor(
		customerFacts{OrdersCount: 5, OrdersTotal: 500_000, LastOrder: &last},
		100_000, now, RFMChampions,
	)
	var hasRule, hasCell bool
	for _, s := range segs {
		if s == SegRegular {
			hasRule = true
		}
		if s == rfmSegmentID(RFMChampions) {
			hasCell = true
		}
	}
	if !hasRule || !hasCell {
		t.Fatalf("segments %v — a customer must carry both the rule labels and their RFM cell", segs)
	}
}
