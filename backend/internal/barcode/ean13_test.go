package barcode

import (
	"strings"
	"testing"

	"restaurant-backend/internal/models"
)

// ⚠️ **A check digit is not an opinion.** A scanner verifies it before it
// reports anything, so a wrong one is a label that does not beep — and the shop
// blames the printer, the scanner or the app, in that order, for a week.
func TestCheckDigitAgreesWithRealBarcodes(t *testing.T) {
	// Codes off actual packets, and the two GS1 publishes as worked examples:
	// a book, a Coca-Cola, a Faber-Castell pencil.
	for code, want := range map[string]byte{
		"9780306406157": '7',
		"5449000000996": '6',
		"4006381333931": '1',
	} {
		got, err := EAN13Check(code[:12])
		if err != nil {
			t.Fatalf("%s: %v", code, err)
		}
		if got != want {
			t.Errorf("%s: check digit %c, want %c", code, got, want)
		}
		if !Valid(code) {
			t.Errorf("%s should be valid", code)
		}
	}
}

func TestOneWrongDigitIsNotAValidCode(t *testing.T) {
	if Valid("9780306406158") {
		t.Error("a wrong check digit passed")
	}
	if Valid("978030640615") {
		t.Error("twelve digits is not an EAN-13")
	}
	if Valid("97803064061a7") {
		t.Error("a letter is not a digit")
	}
}

// ⚠️ **The whole reason this package exists in the 2x range.** GS1 reserves it
// for in-store items, so no manufacturer will ever ship a packet whose code
// collides with one printed here.
func TestAnInternalCodeStaysInTheReservedRange(t *testing.T) {
	code, err := Allocate(1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(code, InternalPrefix) {
		t.Errorf("%s is outside the in-store range", code)
	}
	if !Valid(code) {
		t.Errorf("%s is not a valid EAN-13", code)
	}
	if len(code) != 13 {
		t.Errorf("%s is %d digits", code, len(code))
	}
}

func TestEverySequenceGetsItsOwnCode(t *testing.T) {
	seen := map[string]bool{}
	for i := int64(0); i < 500; i++ {
		code, err := Allocate(i, nil)
		if err != nil {
			t.Fatal(err)
		}
		if seen[code] {
			t.Fatalf("%s was handed out twice", code)
		}
		seen[code] = true
	}
}

// ⚠️ **The failure this is written to prevent.** A scale label carries the weight
// inside the barcode, and the branch setting that reads them defaults to the bare
// prefix "2" — the whole in-store range. A code printed here that the shop's own
// till decodes as a weight does not fail: it beeps, shows a product, and charges
// for a quantity nobody weighed. Nobody notices until a stocktake.
func TestACodeTheTillWouldReadAsAWeightIsNeverHandedOut(t *testing.T) {
	// The default arrangement, which claims every code beginning with 2.
	scale := models.ScaleLabel{Enabled: true}.Defaults()
	reads := func(code string) bool {
		_, ok := scale.Read(code)
		return ok
	}

	for i := int64(0); i < 200; i++ {
		code, err := Allocate(i, reads)
		if err != nil {
			// With a bare "2" prefix and this length, every candidate collides —
			// and saying so is the honest outcome. What must never happen is a
			// code coming back that the till misreads.
			if !strings.Contains(err.Error(), "scale label") {
				t.Fatalf("seq %d: %v", i, err)
			}
			continue
		}
		if reads(code) {
			t.Fatalf("%s was handed out and reads as a scale label", code)
		}
	}
}

// ⚠️ With the scales configured to a specific prefix — which is what a shop that
// actually weighs things ends up with — the internal range is free and every
// code comes back.
func TestAScaleOnItsOwnPrefixLeavesTheRangeFree(t *testing.T) {
	scale := models.ScaleLabel{Enabled: true, Prefix: "22"}.Defaults()
	reads := func(code string) bool {
		_, ok := scale.Read(code)
		return ok
	}
	for i := int64(0); i < 200; i++ {
		code, err := Allocate(i, reads)
		if err != nil {
			t.Fatalf("seq %d: %v", i, err)
		}
		if reads(code) {
			t.Fatalf("%s reads as a scale label", code)
		}
	}
}

// ⚠️ Refused rather than truncated: a shortened code is two products sharing one
// barcode, and the till charges for whichever it met first.
func TestRunningOutIsAnErrorRatherThanAWrappedCode(t *testing.T) {
	if _, err := Allocate(pow10(10), nil); err == nil {
		t.Error("a sequence past the end was accepted")
	}
	if _, err := Allocate(-1, nil); err == nil {
		t.Error("a negative sequence was accepted")
	}
}
