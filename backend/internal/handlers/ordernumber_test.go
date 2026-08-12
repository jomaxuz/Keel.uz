package handlers

import (
	"regexp"
	"strings"
	"testing"
)

var orderNumRe = regexp.MustCompile(`^[A-Z2-9]{4}-[A-Z2-9]{4}$`)

// The order number is the key to a public tracking page that hands back a
// stranger's address and the courier's phone, so it has to look random and be
// random. Shape first: eight unambiguous characters, one dash, no I/O/0/1 that
// a guest could misread off a receipt.
func TestOrderNumberShape(t *testing.T) {
	for range 200 {
		n := orderNumber()
		if !orderNumRe.MatchString(n) {
			t.Fatalf("order number %q does not match the expected shape", n)
		}
		if strings.ContainsAny(n, "IO01") {
			t.Fatalf("order number %q contains an ambiguous character", n)
		}
	}
}

// And no collisions across a run of realistic size — a weak generator repeats,
// and two orders sharing a number means one guest tracking the other's.
func TestOrderNumberNoCollisions(t *testing.T) {
	seen := make(map[string]bool, 5000)
	for range 5000 {
		n := orderNumber()
		if seen[n] {
			t.Fatalf("order number %q repeated within 5000 draws", n)
		}
		seen[n] = true
	}
}
