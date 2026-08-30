package posimport

import (
	"testing"

	"restaurant-backend/internal/models"
)

// ⚠️ **This is where an import silently becomes a thousand times wrong.**
func TestUnitsFromEverySystemMapToOurThree(t *testing.T) {
	kg := []string{"кг", "Кг", "КГ", "кг.", "kg", "килограмм", "kilogramm"}
	for _, s := range kg {
		u, small, err := Unit(s)
		if err != nil || u != models.UnitKg || small {
			t.Fatalf("Unit(%q) = %q,%v,%v", s, u, small, err)
		}
	}
	for _, s := range []string{"л", "l", "литр", "litr"} {
		if u, small, err := Unit(s); err != nil || u != models.UnitL || small {
			t.Fatalf("Unit(%q) = %q,%v,%v", s, u, small, err)
		}
	}
	for _, s := range []string{"шт", "pcs", "dona", "порц", "уп", "банка"} {
		if u, _, err := Unit(s); err != nil || u != models.UnitPcs {
			t.Fatalf("Unit(%q) = %q,%v", s, u, err)
		}
	}
}

// ⚠️ **A gram is not a unit, it is a kilo written small**, and saying so is the
// point: no supplier in the country invoices by the gram, so `гр` in a purchase
// column means the exporter wrote the recipe unit where the purchase unit
// belongs — and the price beside it has to be read as per kilo, or the
// ingredient is a thousand times too cheap and every dish with it looks free.
func TestGramsAreFlaggedAsTheSmallUnit(t *testing.T) {
	for _, s := range []string{"г", "гр", "грамм", "g", "gramm"} {
		u, small, err := Unit(s)
		if err != nil || u != models.UnitKg || !small {
			t.Fatalf("Unit(%q) = %q, small=%v, err=%v — want kg and small", s, u, small, err)
		}
	}
	for _, s := range []string{"мл", "ml"} {
		if u, small, err := Unit(s); err != nil || u != models.UnitL || !small {
			t.Fatalf("Unit(%q) = %q, small=%v", s, u, small)
		}
	}
}

// ⚠️ **An unrecognised unit is an error, never a default.** Defaulting to `pcs`
// is the specific disaster: a kilo of beef becomes one piece, so the 180 g of
// it in a dish costs the price of 180 kilos — and every number on screen stays
// plausible.
func TestAnUnknownUnitIsRefusedRatherThanGuessed(t *testing.T) {
	for _, s := range []string{"", "пучок", "ведро", "bunch", "???"} {
		if _, _, err := Unit(s); err == nil {
			t.Fatalf("Unit(%q) was accepted — it would be imported as a guess", s)
		}
	}
}

// A tech card line of `0,18 кг` and one of `180 гр` are the same 180 grams.
// Storing either unconverted is out by a factor of a thousand, in opposite
// directions — one dish looks free and the next looks ruinous.
func TestARecipeLineEndsUpInTheSmallUnitEitherWay(t *testing.T) {
	fromKg, err := RecipeQty(0.18, "кг")
	if err != nil || fromKg != 180 {
		t.Fatalf("0,18 кг = %v, want 180 g", fromKg)
	}
	fromG, err := RecipeQty(180, "гр")
	if err != nil || fromG != 180 {
		t.Fatalf("180 гр = %v, want 180 g", fromG)
	}
	if fromKg != fromG {
		t.Fatal("the same quantity written two ways imported as two quantities")
	}
}
