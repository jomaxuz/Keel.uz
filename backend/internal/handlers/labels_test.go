package handlers

import (
	"strings"
	"testing"
	"time"

	"restaurant-backend/internal/models"
)

// ⚠️ **The list a price change makes wrong.** Nothing on any screen used to say
// which shelves had gone stale, so the answer was a shop reprinting everything
// or reprinting nothing — and a shelf that says the old price is the law's
// business rather than a matter of tidiness.
func TestAShelfIsStaleForOneOfThreeReasons(t *testing.T) {
	printed := time.Now().Add(-24 * time.Hour)

	for name, tc := range map[string]struct {
		item models.MenuItem
		want string
	}{
		// The one that stops a sale rather than misdescribing it: no code, no
		// scan, no ring-up. Named first for that reason.
		"never had a barcode": {
			models.MenuItem{Price: 18500},
			"noBarcode",
		},
		"has a code, never printed": {
			models.MenuItem{Price: 18500, Barcode: "2100000000012"},
			"never",
		},
		"printed, then repriced": {
			models.MenuItem{
				Price: 19500, Barcode: "2100000000012",
				LabelAt: &printed, LabelPrice: 18500,
			},
			"price",
		},
		"printed at the price it still sells for": {
			models.MenuItem{
				Price: 18500, Barcode: "2100000000012",
				LabelAt: &printed, LabelPrice: 18500,
			},
			"",
		},
	} {
		if got := staleReason(tc.item); got != tc.want {
			t.Errorf("%s: %q, want %q", name, got, tc.want)
		}
	}
}

// ⚠️ **A price back down to what was printed is not stale.** A shop that ran a
// weekend promotion and put the old price back has the right sticker on the
// shelf, and a list that told them to reprint it would be a list they stop
// reading.
func TestAPriceThatCameBackIsNotStale(t *testing.T) {
	printed := time.Now().Add(-time.Hour)
	it := models.MenuItem{
		Price: 18500, Barcode: "2100000000012",
		LabelAt: &printed, LabelPrice: 18500,
	}
	if got := staleReason(it); got != "" {
		t.Errorf("reason %q, want none", got)
	}
}

// ⚠️ **The unit beside the price, because "18 500" means nothing on a shelf of
// loose goods** until it says whether that is a kilo or a packet. The codes are
// the state classifier's, which is what the field holds — a word invented here
// would disagree with the receipt.
func TestTheUnitIsTheClassifiersAndNotAGuess(t *testing.T) {
	if got := unitWord(models.MenuItem{UnitCode: 10}); got != "kg" {
		t.Errorf("gram code gave %q", got)
	}
	if got := unitWord(models.MenuItem{UnitCode: 112}); got != "l" {
		t.Errorf("litre code gave %q", got)
	}
	// A piece is sold by the piece: nothing to say, and a "/ dona" on every
	// sticker in the shop is noise on the one line that has to be read fast.
	if got := unitWord(models.MenuItem{UnitCode: 0}); got != "" {
		t.Errorf("piece gave %q", got)
	}
}

// ⚠️ **The price is the largest thing on a shelf label and the name is second.**
// It is read from a metre away by somebody deciding whether to pick the packet
// up; the sticker on the packet is read by a scanner, which needs none of the
// words. One layout serves both, so it is written for the harder reader.
func TestTheLabelLeadsWithTheNameAndShoutsThePrice(t *testing.T) {
	lines := labelLines(
		models.MenuItem{Name: "Guruch Lazer", Price: 18500, UnitCode: 10},
		models.Branch{Name: "Chilonzor"},
	)
	if len(lines) < 2 {
		t.Fatalf("a label of %d lines", len(lines))
	}
	if !strings.HasPrefix(lines[0], "!") || !strings.Contains(lines[0], "Guruch Lazer") {
		t.Errorf("first line %q is not the name, emphasised", lines[0])
	}
	last := lines[len(lines)-1]
	if !strings.HasPrefix(last, "!!") {
		t.Errorf("the price line %q is not the big one", last)
	}
	if !strings.Contains(last, "kg") {
		t.Errorf("the price line %q does not say what the price is per", last)
	}
}

// ⚠️ **The cut goes at the end and only once.** `Encode` writes it with the
// text, so a payload built the obvious way would cut the paper before the
// barcode — a sticker with no code on it, and a stray code on the next one.
func TestTheBarcodeIsPrintedBeforeThePaperIsCut(t *testing.T) {
	p := models.Printer{Cut: true}
	bars := []byte{0x1D, 0x6B, 67, 12}
	out := labelPayload([]string{"Guruch"}, bars, p)

	atBars := indexOf(out, bars)
	atCut := indexOf(out, []byte{0x1D, 0x56})
	if atBars < 0 {
		t.Fatal("the barcode is not in the payload")
	}
	if atCut < 0 {
		t.Fatal("a cutting printer got no cut")
	}
	if atBars > atCut {
		t.Error("the paper is cut before the barcode is printed")
	}
}

func indexOf(hay, needle []byte) int {
	return strings.Index(string(hay), string(needle))
}
