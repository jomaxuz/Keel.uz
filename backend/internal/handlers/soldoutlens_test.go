package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **A branch that cannot be resolved must not switch the stop list off.**
// It used to: an unknown id — a stale `branch` cookie, a branch deleted or
// moved to another brand, a mistyped link — produced a nil lens, and a nil lens
// means "nothing is sold out anywhere". The site then offered every stopped
// dish with an add button, a guest ordered one, and the kitchen found out at
// the pass.
//
// Failing open is the wrong direction for this question: hiding a dish one
// kitchen could have cooked costs an order, and ignoring the list sells food
// nobody has.
func TestAnUnknownBranchFallsBackRatherThanClearingTheStopList(t *testing.T) {
	src := readSource(t, "soldoutlens.go")
	fn := between(t, src, "func (h *Handler) publicSoldOut", "\n}\n")

	if strings.Contains(fn, "if err != nil {\n\t\treturn nil, nil\n\t}") {
		t.Fatal("an unresolvable branch still turns the whole stop list off")
	}
	if !strings.Contains(fn, "raw = \"\"") || !strings.Contains(fn, "branch = nil") {
		t.Fatal("an unresolvable branch is no longer treated as 'none named'")
	}
	// ⚠️ And the single-branch path must not read a nil branch: that would be
	// the same failure wearing a different hat.
	if !strings.Contains(fn, "if branch != nil {") {
		t.Fatal("the one-branch path can dereference a branch that was never found")
	}
}
