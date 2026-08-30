package posimport

import "testing"

// ⚠️ **The decimal comma is the most expensive character in this import.**
// Every one of these systems exports from a Russian locale: 180 grams is
// written `0,180`, and ParseFloat fails on it. The tempting recovery — call a
// failed parse zero — imports the tech card with every quantity at nothing,
// which makes every dish cost nothing, which makes the food cost report say the
// kitchen is free. It is arithmetically consistent, sits on a screen full of
// correct-looking dishes, and is discovered at a stocktake.
func TestARussianExportsNumbersAreRead(t *testing.T) {
	cases := map[string]float64{
		"0,180":         0.18,
		"1,5":           1.5,
		"180":           180,
		"1 234,56":      1234.56,
		"1\u00a0234,56": 1234.56, // the non-breaking space a spreadsheet writes
		"1.234,56":      1234.56, // dot thousands, comma decimal
		"1,234.56":      1234.56, // the other way round
		"42000":         42000,
		" 12 ":          12,
	}
	for in, want := range cases {
		got, err := ParseQty(in)
		if err != nil {
			t.Fatalf("ParseQty(%q) failed — a failed parse becomes a zero quantity", in)
		}
		if got != want {
			t.Fatalf("ParseQty(%q) = %v, want %v", in, got, want)
		}
	}
}

// ⚠️ **A lone comma is a decimal comma, not a thousands mark.** `1,5` is one and
// a half kilos on every export these systems produce; reading it as fifteen
// hundred puts a thousand times the meat in the dish.
func TestALoneCommaIsADecimal(t *testing.T) {
	got, err := ParseQty("1,5")
	if err != nil || got != 1.5 {
		t.Fatalf("ParseQty(1,5) = %v, %v — want 1.5", got, err)
	}
}

// ⚠️ **A cell that cannot be read is an error, never a zero.** Silence here is
// the difference between an import somebody corrects and an import somebody
// trusts.
func TestAnUnreadableCellIsRefused(t *testing.T) {
	for _, s := range []string{"", "—", "нет", "abc", "n/a"} {
		if _, err := ParseQty(s); err == nil {
			t.Fatalf("ParseQty(%q) succeeded — it would import as a zero", s)
		}
	}
}
