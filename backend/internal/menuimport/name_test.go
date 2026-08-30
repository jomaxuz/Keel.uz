package menuimport

import "testing"

// ⚠️ **The apostrophe is the whole problem.** Uzbek writes Lag'mon, and every
// source spells that mark differently — the typographic ʻ an aggregator's CMS
// produces, the ʼ a phone keyboard produces, a backtick, a plain quote. Four
// strings, one dish. Comparing them literally imports a second Lag'mon that
// reads as identical in the list and is a separate row in every report from
// then on.
func TestOneDishHoweverItsApostropheIsWritten(t *testing.T) {
	want := NormalName("Lag'mon")
	for _, spelling := range []string{
		"Lagʻmon", "Lagʼmon", "Lag‘mon", "Lag’mon", "Lag`mon", "Lag´mon",
		"LAG'MON", "  lag'mon  ", "Lag'mon",
	} {
		if got := NormalName(spelling); got != want {
			t.Fatalf("%q normalised to %q, want %q", spelling, got, want)
		}
	}
}

// ⚠️ **And it goes no further than that.** "Lag'mon" and "Lag'mon qovurma" are
// two dishes on every menu in the country; a rule that merged them would
// silently refuse to import the second, which is a missing dish nobody can see
// is missing.
func TestDifferentDishesStayDifferent(t *testing.T) {
	pairs := [][2]string{
		{"Lag'mon", "Lag'mon qovurma"},
		{"Osh", "Osh palov"},
		{"Somsa", "Somsa go'shtli"},
		{"Choy", "Ko'k choy"},
	}
	for _, p := range pairs {
		if NormalName(p[0]) == NormalName(p[1]) {
			t.Fatalf("%q and %q were treated as one dish", p[0], p[1])
		}
	}
}

// Doubled spaces from a copy-paste are not a different dish.
func TestSpacingIsCollapsedNotRemoved(t *testing.T) {
	if NormalName("Osh  palov") != NormalName("Osh palov") {
		t.Fatal("a doubled space made a second dish")
	}
	// ⚠️ Removed rather than collapsed would make these one, and they are not
	// obviously the same thing — guessing costs a dish.
	if NormalName("Oshpalov") == NormalName("Osh palov") {
		t.Fatal("spaces are being removed, not collapsed")
	}
}
