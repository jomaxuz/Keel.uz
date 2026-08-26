package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **The journal entry has to name the ingredient and both numbers**, because
// "menu.update · Lag'mon" is what made this invisible in the first place: true,
// filed, and answering nothing. A person opening the log a month later needs to
// know which ingredient and from what to what — that is the whole difference
// between a record and a receipt for having kept one.
func TestTheJournalSaysWhatActuallyChanged(t *testing.T) {
	got := describeRecipeDiff([]recipeChange{
		{Name: "Mol go'shti", Unit: "g", From: 150, To: 200},
	})
	for _, want := range []string{"Mol go'shti", "150", "200", "g"} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q missing from %q", want, got)
		}
	}
}

// An ingredient arriving and an ingredient leaving read differently from a norm
// that moved, and the log has to say which happened.
func TestAddedAndRemovedLinesReadDifferently(t *testing.T) {
	added := describeRecipeDiff([]recipeChange{{Name: "Zira", Unit: "g", From: 0, To: 5}})
	if !strings.HasPrefix(added, "+") {
		t.Fatalf("a new ingredient does not read as one: %q", added)
	}
	gone := describeRecipeDiff([]recipeChange{{Name: "Zira", Unit: "g", From: 5, To: 0}})
	if !strings.Contains(gone, "5") || strings.HasPrefix(gone, "+") {
		t.Fatalf("a removed ingredient does not read as one: %q", gone)
	}
}

// ⚠️ **Bounded, because a journal row is scanned, not studied.** An edit that
// rewrote a forty-line card must not put forty lines into a table.
func TestALongDiffIsTrimmedAndSaysSo(t *testing.T) {
	var many []recipeChange
	for i := 0; i < 30; i++ {
		many = append(many, recipeChange{Name: string(rune('a' + i)), From: 1, To: 2})
	}
	got := describeRecipeDiff(many)
	if strings.Count(got, ",") > 10 {
		t.Fatalf("the whole card went into one journal row: %q", got)
	}
	if !strings.Contains(got, "+22") {
		t.Fatalf("the trimmed remainder is not counted: %q", got)
	}
}

// ⚠️ **Proportion, never absolute quantity.** 50 of anything is enormous in
// saffron and nothing in flour; a threshold in grams would alert on every bread
// recipe and never on the one that matters.
func TestIncreasesAreMeasuredInProportion(t *testing.T) {
	_, pct, ok := biggestIncrease([]recipeChange{
		{Name: "Un", Unit: "g", From: 1000, To: 1050}, // +5%, +50g
		{Name: "Za'faron", Unit: "g", From: 1, To: 2}, // +100%, +1g
	})
	if !ok {
		t.Fatal("no increase found")
	}
	if pct != 100 {
		t.Fatalf("the largest *proportional* rise was not chosen: %d%%", pct)
	}
}

// ⚠️ **A norm going down raises nothing.** It is a kitchen being told to use
// less, which turns up as a surplus at the next count and embarrasses nobody.
func TestReductionsRaiseNothing(t *testing.T) {
	if _, _, ok := biggestIncrease([]recipeChange{
		{Name: "Go'sht", From: 200, To: 150},
	}); ok {
		t.Fatal("using less raised an alert")
	}
}

// ⚠️ **An ingredient added to a card is not an increase.** It is a new line,
// and its arrival has ordinary causes — a garnish, a sauce, a dish reworked.
// Treating a zero baseline as a rise makes the percentage infinite and every
// new ingredient an alert.
func TestANewIngredientIsNotAnIncrease(t *testing.T) {
	if _, _, ok := biggestIncrease([]recipeChange{
		{Name: "Zira", From: 0, To: 5},
	}); ok {
		t.Fatal("adding an ingredient raised an increase alert")
	}
}

// ⚠️ **The threshold is high on purpose.** A chef who finds the card says 180g
// while the ladle holds 200g is doing the right thing by fixing it, and nobody
// should be messaged about that 11%.
func TestSmallCorrectionsStayQuiet(t *testing.T) {
	_, pct, ok := biggestIncrease([]recipeChange{
		{Name: "Sho'rva", From: 180, To: 200},
	})
	if !ok {
		t.Fatal("no increase computed")
	}
	if pct >= RecipeIncreaseAlertPct {
		t.Fatalf("an ordinary correction of %d%% would send a message", pct)
	}
}

// ⚠️ **Read before the replace**, because a whole-document write leaves nothing
// to compare against afterwards.
func TestTheDiffIsTakenBeforeTheWrite(t *testing.T) {
	src := readLossSource(t, "admin.go")
	diff := strings.Index(src, "changes := h.recipeDiff(")
	write := strings.Index(src, "h.Store.Menu.ReplaceOne(")
	if diff < 0 || write < 0 {
		t.Fatal("the recipe diff or the write is gone")
	}
	if diff > write {
		t.Fatal("the previous card is read after it has been overwritten")
	}
}

// Two decimal grams are the same norm, and floating point must not turn an
// unchanged card into a journal entry.
func TestUnchangedNormsAreNotChanges(t *testing.T) {
	if !diffTiny(0.1+0.2, 0.3) {
		t.Fatal("floating point noise reads as an edit")
	}
	if diffTiny(150, 200) {
		t.Fatal("a real change was swallowed as noise")
	}
}
