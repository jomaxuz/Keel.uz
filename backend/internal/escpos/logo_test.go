package escpos

import "testing"

// ⚠️ **The logo prints at a fixed, small share of the paper.** At the head's
// full width it is the biggest thing on the receipt — the guest is handed a
// poster with their total underneath — and it would change size between 58 mm
// and 80 mm paper, so the same restaurant's receipts would not look like each
// other. It is also the one element on the paper nobody reads closely.
func TestTheLogoIsNarrowerThanTheHead(t *testing.T) {
	for _, mm := range []int{58, 80} {
		logo, head := LogoDotsFor(mm), DotsFor(mm)
		if logo >= head {
			t.Fatalf("%dmm: logo %d dots against a %d-dot head", mm, logo, head)
		}
		// ⚠️ Multiples of eight: the raster command packs eight dots to a byte,
		// and a width that is not shifts every row after the first.
		if logo%8 != 0 {
			t.Fatalf("%dmm: %d dots is not a multiple of eight", mm, logo)
		}
	}
}
