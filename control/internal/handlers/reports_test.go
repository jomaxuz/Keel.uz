package handlers

import "testing"

// ⚠️ **The whole screen depends on this and nothing else does.** Fingerprinting
// too narrowly splits one bug across four hundred rows, none of which looks
// important enough to open; too broadly merges two bugs into a row somebody
// opens and immediately sees two different stacks in. The costs are not
// symmetric, so the rule leans towards merging.
func TestOneFaultIsOneRowHoweverManyIdsAreInIt(t *testing.T) {
	same := [][2]string{
		{"order 6f3a91bc not found", "/admin/orders/6f3a91bc"},
		{"order 91bc6f3a not found", "/admin/orders/91bc6f3a"},
		{"Order 42 not found", "/admin/orders/42"},
	}
	first := Fingerprint(same[0][0], same[0][1])
	for _, c := range same[1:] {
		if got := Fingerprint(c[0], c[1]); got != first {
			t.Fatalf("%q on %q grouped separately — one bug is becoming a list",
				c[0], c[1])
		}
	}
}

// The other half: things that really are different faults must stay apart, or
// the list says one thing is broken when three are.
func TestDifferentFaultsStayApart(t *testing.T) {
	base := Fingerprint("order not found", "/admin/orders")
	for _, c := range [][2]string{
		{"order already paid", "/admin/orders"},
		{"order not found", "/kassa"},
	} {
		if Fingerprint(c[0], c[1]) == base {
			t.Fatalf("%q on %q merged into an unrelated fault", c[0], c[1])
		}
	}
}

// ⚠️ A quoted value is the same kind of noise as an id: the register refused
// "Lag'mon" and refused "Somsa" is one bug about refusing dishes.
func TestQuotedValuesDoNotSplitAFault(t *testing.T) {
	a := Fingerprint(`register refused "Lag'mon"`, "/kassa")
	b := Fingerprint(`register refused "Somsa"`, "/kassa")
	if a != b {
		t.Fatal("a quoted dish name is splitting one fault into one row per dish")
	}
}

// ⚠️ A stack sliced through a UTF-8 sequence renders as a replacement character
// in the console and reads as corrupted data rather than as a truncation —
// which sends somebody looking for an encoding bug that does not exist.
func TestClipDoesNotCutThroughACharacter(t *testing.T) {
	// "Chek yopilmadi" with a multi-byte apostrophe near the cut.
	s := "xato: to‘lov qabul qilinmadi"
	for n := 1; n < len(s); n++ {
		got := clip(s, n)
		if len(got) > 0 && !isValidUTF8(got) {
			t.Fatalf("clip(%d) produced invalid UTF-8: %q", n, got)
		}
	}
}

func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == '�' {
			return false
		}
	}
	return true
}
