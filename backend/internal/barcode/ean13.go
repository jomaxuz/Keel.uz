// Package barcode makes the codes a shop prints for itself.
//
// ⚠️ **Only for goods that arrived without one.** A packet with a barcode on it
// already has an identity the whole world agrees about; printing our own over it
// would mean the same tin scans as two different products depending on which
// sticker the cashier's beam caught. This is for what a shop repacks, weighs out,
// bakes itself, or buys from a market stall — the things that have no code and
// cannot be sold across a counter without one.
//
// ⚠️ **GS1 reserves the 2x prefixes for in-store items**, which is what makes a
// code invented here safe: no manufacturer will ever ship a product that begins
// with one, so an internal code cannot collide with a real EAN. That is the only
// guarantee there is, and it is the reason the range is not a matter of taste.
package barcode

import (
	"fmt"
	"strings"
)

// InternalPrefix is where a shop's own codes live.
//
// ⚠️ **"21", not "2".** A scale prints its labels in the same reserved range,
// and the branch setting that reads them defaults to the bare prefix "2" — which
// claims the whole of it. A code that a shop's own till reads back as a weight is
// the failure `models.ScaleLabel` is written about: the counter beeps, shows a
// product, prints a receipt, and charges for a quantity nobody weighed. Two
// digits leave room for both, and `Allocate` checks the outcome rather than
// trusting this constant.
const InternalPrefix = "21"

// EAN13Check is the thirteenth digit: what makes a string of numbers a barcode.
//
// ⚠️ **Computed, never chosen.** A scanner verifies this before it reports
// anything, so a printed code with the wrong one is a label that simply does not
// beep — and the shop concludes the printer, or the scanner, or the app is
// broken. The rule is GS1's: digits alternate weight 1 and 3 from the left, the
// check digit is what rounds the total up to a multiple of ten.
func EAN13Check(first12 string) (byte, error) {
	if len(first12) != 12 {
		return 0, fmt.Errorf("ean-13 needs 12 digits, got %d", len(first12))
	}
	sum := 0
	for i, r := range first12 {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("ean-13 takes digits only")
		}
		n := int(r - '0')
		// The leftmost digit has weight 1, and every second one after it 3.
		if i%2 == 1 {
			n *= 3
		}
		sum += n
	}
	return byte('0' + (10-sum%10)%10), nil
}

// Valid reports whether this is a well-formed EAN-13.
func Valid(code string) bool {
	code = strings.TrimSpace(code)
	if len(code) != 13 {
		return false
	}
	want, err := EAN13Check(code[:12])
	return err == nil && code[12] == want
}

// Allocate builds one internal code from a number nobody else is using.
//
// ⚠️ **`readsAsScale` is passed in rather than assumed away.** Whatever prefix a
// branch has configured for its scales, the code this returns has to survive
// being scanned in *that* shop — so the candidate is offered to the very reader
// that will meet it, and rejected if it decodes as a weight. A rule about which
// prefixes are safe would be right today and wrong the first time somebody
// changes a scale setting, which is the direction that costs money silently.
//
// ⚠️ **The sequence is the caller's**, because uniqueness is a fact about a
// catalogue and this package has no database. `seq` is expected to be a counter
// per brand; anything above ten digits is refused rather than truncated — a
// silently shortened code is two products sharing one barcode.
func Allocate(seq int64, readsAsScale func(code string) bool) (string, error) {
	if seq < 0 {
		return "", fmt.Errorf("sequence must not be negative")
	}
	// prefix + body = 12 digits, then the check digit.
	body := 12 - len(InternalPrefix)
	// ⚠️ Room is checked before the shop runs out rather than after: a catalogue
	// that overflowed would start reusing codes, and two products with one
	// barcode is a till that charges for whichever it saw first.
	if seq >= pow10(body) {
		return "", fmt.Errorf("no internal barcodes left")
	}

	// ⚠️ **Tried in order until one is not a scale label**, rather than computed
	// once. Only a handful of candidates can ever collide — the check digit does
	// not repeat often — and stopping at the first clean one keeps the codes
	// dense and predictable.
	for attempt := int64(0); attempt < 1000 && seq+attempt < pow10(body); attempt++ {
		first12 := fmt.Sprintf("%s%0*d", InternalPrefix, body, seq+attempt)
		check, err := EAN13Check(first12)
		if err != nil {
			return "", err
		}
		code := first12 + string(check)
		if readsAsScale != nil && readsAsScale(code) {
			continue
		}
		return code, nil
	}
	return "", fmt.Errorf("every candidate reads as a scale label")
}

func pow10(n int) int64 {
	out := int64(1)
	for range n {
		out *= 10
	}
	return out
}
