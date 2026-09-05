package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **The matrix multiplies, and a form that multiplies is a form somebody
// empties a shop into.** Four axes of ten values is ten thousand products from
// one press, entered by somebody trying the feature out and undone one row at a
// time. These are the shapes the expansion has to answer for before any of them
// reaches the database.
func TestTheMatrixIsBuiltAndBounded(t *testing.T) {
	names, combos, err := expandAxes([]variantAxis{
		{Name: "O'lcham", Values: []string{"S", "M", "L"}},
		{Name: "Rang", Values: []string{"Qora", "Oq"}},
	})
	if err != nil {
		t.Fatalf("two ordinary axes were refused: %v", err)
	}
	if len(names) != 2 || names[0] != "O'lcham" {
		t.Fatalf("axes came back as %v", names)
	}
	if len(combos) != 6 {
		t.Fatalf("3×2 made %d combinations, want 6", len(combos))
	}
	// ⚠️ **In axis order, every time.** "M / Qora" and "Qora / M" are the same
	// shirt, and a matrix that regenerates in a different order would create
	// every variant a second time — each with its own stock row, splitting the
	// count on the shelf between two rows nobody can tell apart.
	for _, c := range combos {
		if len(c) != 2 {
			t.Fatalf("combination %v is not one value per axis", c)
		}
		if c[0] != "S" && c[0] != "M" && c[0] != "L" {
			t.Errorf("combination %v has the axes the wrong way round", c)
		}
	}

	// An axis with values and no name is a question with answers and no
	// question — the defect `optionProblems` exists for, in a second form.
	if _, _, err := expandAxes([]variantAxis{
		{Values: []string{"S", "M"}},
	}); err == nil {
		t.Error("an unnamed axis was accepted")
	}
	// A named axis with nothing in it cannot be expanded at all.
	if _, _, err := expandAxes([]variantAxis{{Name: "Rang"}}); err == nil {
		t.Error("an axis with no values was accepted")
	}
	// ⚠️ But an entirely blank row is the one the "add axis" button just made,
	// and refusing it would stop somebody saving because they opened a form and
	// changed their mind.
	if _, combos, err := expandAxes([]variantAxis{
		{Name: "O'lcham", Values: []string{"S"}}, {},
	}); err != nil || len(combos) != 1 {
		t.Errorf("a blank row broke the matrix: %v, %v", combos, err)
	}
	// And the ceiling holds.
	big := []variantAxis{}
	for i := 0; i < 4; i++ {
		vals := []string{}
		for j := 0; j < 10; j++ {
			vals = append(vals, string(rune('a'+j)))
		}
		big = append(big, variantAxis{Name: string(rune('A' + i)), Values: vals})
	}
	if _, _, err := expandAxes(big); err == nil {
		t.Error("ten thousand variants were accepted from one press")
	}
}

// ⚠️ **A model is not a thing on the shelf**, and the refusal has to live in
// the one function the till, the website and the call centre all price through.
// Sold directly it would take money for a line no delivery ever stocked and no
// stocktake could find — and print "Ko'ylak" on a receipt with no size on it,
// which is unanswerable when somebody brings it back.
func TestAModelRowCannotBeSold(t *testing.T) {
	src := readSource(t, "orderline.go")
	fn := between(t, src, "func (h *Handler) menuLine(", "\n}\n")
	if !strings.Contains(fn, "len(dbItem.VariantAxes) > 0") {
		t.Fatal("menuLine no longer refuses a model row — a shirt with no size can be rung up")
	}
}

// ⚠️ **Generating never deletes.** A variant dropped from the axes may still be
// on a shelf, in a delivery and on last month's receipts; the sweep that tidies
// it away is the sweep that silently drops a stock row and its history.
func TestGeneratingVariantsDeletesNothing(t *testing.T) {
	src := readSource(t, "menuvariants.go")
	for _, bad := range []string{"DeleteOne", "DeleteMany", "$unset"} {
		if strings.Contains(src, bad) {
			t.Errorf("the generator calls %s — a variant with stock behind it "+
				"must never be removed by pressing a button twice", bad)
		}
	}
	// And it must not mint a second copy of what is already there.
	fn := between(t, src, "func (h *Handler) AdminGenerateVariants", "\n}\n")
	if !strings.Contains(fn, "seen[") {
		t.Error("pressing generate twice would create every variant again")
	}
	// ⚠️ Each variant sells itself, or no delivery ever reaches its shelf.
	if !strings.Contains(fn, "child.SellsItself = true") {
		t.Error("variants no longer own their stock rows")
	}
	// ⚠️ And none of them inherits the model's barcode: it is unique within a
	// brand, so the copy would be refused by the index halfway through the run.
	if !strings.Contains(fn, "child.Barcode = \"\"") {
		t.Error("a variant would inherit the model's barcode")
	}
}

// ⚠️ **The catalogue every screen reads must not carry a row that cannot be
// sold.** The website, the till and the floor screen all take the menu from one
// query; a model left in it is a card and a tile that answer every tap with a
// refusal, which is worse than an absent one because somebody tries again.
func TestTheCatalogueLeavesOutModelRows(t *testing.T) {
	fn := between(t, readSource(t, "public.go"), "func (h *Handler) GetMenu", "\n}\n")
	if !strings.Contains(fn, `menuFilter["variantAxes"]`) {
		t.Fatal("GetMenu no longer excludes model rows — a shirt with no size " +
			"is back on the website and on the till")
	}
}
