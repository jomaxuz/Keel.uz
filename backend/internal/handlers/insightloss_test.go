package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **A briefing card must never name one employee on the strength of a
// comparison with themselves.**
//
// This is the single most damaging thing the assistant could produce: an owner
// reading, over breakfast, that a named person voids more than everybody else —
// when "everybody else" is nobody. The guard is a count, not a judgement call,
// so it cannot be softened by a later edit that "handles the single-staff case
// gracefully".
func TestNoOutlierCardWithoutColleagues(t *testing.T) {
	src := readLossSource(t, "insight.go")
	if !strings.Contains(src, "if len(people) < 2 {") {
		t.Fatal("one person can now be reported as an outlier against themselves")
	}
}

// ⚠️ **A restaurant that barely voids has no outlier, whatever the ratios say.**
// Three voids against one is triple the rate and is three voids. Raising that as
// a finding is how a briefing teaches its reader that findings are noise — and
// once that is learned it applies to the real ones too.
func TestSmallNumbersAreNotAFinding(t *testing.T) {
	src := readLossSource(t, "insight.go")
	if !strings.Contains(src, "houseChecks < 200 || houseVoids < 20") {
		t.Fatal("the floor under the outlier comparison is gone")
	}
	if !strings.Contains(src, "p.checks < 50") {
		t.Fatal("somebody who closed four checks can be compared to somebody who closed four hundred")
	}
}

// ⚠️ **Only shortfalls are summed.** A surplus is usually a delivery booked
// twice — a different problem — and netting the two makes a month of both look
// like a quiet month, which is the one reading that stops anybody looking.
func TestSurplusesDoNotCancelShortfalls(t *testing.T) {
	src := readLossSource(t, "insight.go")
	if !strings.Contains(src, "if s.Value < 0 {") {
		t.Fatal("surpluses are cancelling against shortfalls")
	}
}

// ⚠️ **The privacy rule is about guests, and staff attribution is the point.**
// A colleague's name travels with the outlier fact because the card is useless
// without it. This asserts the distinction is still the one being drawn — that
// no guest-derived field crept into a fact's numbers.
func TestNoGuestIdentityInTheFacts(t *testing.T) {
	src := readLossSource(t, "insight.go")
	for _, forbidden := range []string{
		`"phone"`, `"customerName"`, `"firstName"`, `"lastName"`, `"address"`,
	} {
		if strings.Contains(src, forbidden) {
			t.Fatalf("%s reached a fact — guests never leave the restaurant", forbidden)
		}
	}
}
