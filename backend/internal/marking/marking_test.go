package marking

import "testing"

// A real-shaped code: GTIN behind "01", serial behind "21", then the
// separator and the rest of the fields.
const good = "0104607034170203215Fw2R\x1d93dGVz"

func TestAScannerIsAKeyboardAndItsHabitsAreNormalised(t *testing.T) {
	// ⚠️ Three ways the same separator arrives, depending on the pistol's
	// firmware. A code refused because of the scanner's configuration is a code
	// the cashier cannot fix.
	for _, raw := range []string{
		"0104607034170203215Fw2R\x1d93dGVz",
		"0104607034170203215Fw2R\\x1d93dGVz",
		"0104607034170203215Fw2Rè93dGVz",
		"  0104607034170203215Fw2R\x1d93dGVz\n",
	} {
		if got := Normalize(raw); got != good {
			t.Fatalf("Normalize(%q) = %q", raw, got)
		}
	}
}

func TestATrailingSeparatorIsNotPartOfTheCode(t *testing.T) {
	// ⚠️ Kept, it makes two scans of one bottle look like two codes — and the
	// duplicate check is the only thing standing between that and a receipt
	// that withdraws one bottle while two are handed over.
	if got := Normalize(good + "\x1d"); got != good {
		t.Fatalf("got %q", got)
	}
}

func TestWhatCannotBeAMarkingCode(t *testing.T) {
	cases := map[string]string{
		"empty":            "",
		"an EAN-13 barcode": "4607034170203",
		"a half-read code":  "010460703417020",
		"a URL somebody scanned": "https://asl-belgisi.uz/check/12345678901234567890",
		"a control character from a scanner in the wrong mode": "01046070341702032\x0715Fw2R93dGVz",
	}
	for name, code := range cases {
		if err := Check(Normalize(code)); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
}

func TestARealCodeIsAccepted(t *testing.T) {
	if err := Check(good); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestTheSameBottleScannedTwiceIsRefused(t *testing.T) {
	// The ordinary way it happens: the pistol beeps, nobody is sure it took, it
	// is scanned again. The register accepts that receipt, so here is the only
	// place it can be caught.
	err := CheckAll([]Line{
		{Marked: true, Code: good, Qty: 1},
		{Marked: true, Code: good, Qty: 1},
	})
	if err != ErrDuplicate {
		t.Fatalf("err = %v, want ErrDuplicate", err)
	}
}

func TestOneCodeCannotCoverTwoBottles(t *testing.T) {
	err := CheckAll([]Line{{Marked: true, Code: good, Qty: 2}})
	if err == nil {
		t.Fatal("a quantity of two behind one code was accepted")
	}
}

func TestAMarkedLineWithNoCodeIsRefused(t *testing.T) {
	if err := CheckAll([]Line{{Marked: true, Qty: 1}}); err != ErrEmpty {
		t.Fatalf("err = %v, want ErrEmpty", err)
	}
}

func TestAnUnmarkedLineIsNotAsked(t *testing.T) {
	// ⚠️ Every dish on every menu written before this existed is unmarked, and
	// a plate of osh has no code to give.
	if err := CheckAll([]Line{{Qty: 3}, {Qty: 1}}); err != nil {
		t.Fatalf("err = %v", err)
	}
}

func TestTwoDifferentBottlesAreFine(t *testing.T) {
	other := "0104607034170203215Zz9Q\x1d93dGVz"
	err := CheckAll([]Line{
		{Marked: true, Code: good, Qty: 1},
		{Marked: true, Code: other, Qty: 1},
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
}
