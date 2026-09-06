package handlers

import (
	"strings"
	"testing"
	"time"

	"regexp"

	"restaurant-backend/internal/barcode"
	"restaurant-backend/internal/escpos"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/receipt"
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
// loose goods** until it says whether that is a kilo or a packet.
//
// ⚠️ **The codes are the ones the panel actually writes**, and this test used to
// assert one that it does not. `unitWord` read 112 for a litre — a number from
// no list in this product; the menu form offers 0, 11, 10, 41 and 22. So juice
// sold by the litre printed "18 500 so'm" with nothing after it, and the test
// agreed with the code instead of with the form. The options are pinned against
// the form here for that reason.
func TestTheUnitIsTheClassifiersAndNotAGuess(t *testing.T) {
	// The five the menu form offers, which is the whole set this field ever
	// holds: piece, kilogram, gram, litre, metre.
	for code, want := range map[int]string{
		0: "", 11: "kg", 10: "kg", 41: "l", 22: "m",
	} {
		if got := unitWord(models.MenuItem{UnitCode: code}); got != want {
			t.Errorf("code %d gave %q, want %q", code, got, want)
		}
	}
	// A piece is sold by the piece: nothing to say, and a "/ dona" on every
	// sticker in the shop is noise on the one line that has to be read fast.
	if got := unitWord(models.MenuItem{UnitCode: 0}); got != "" {
		t.Errorf("piece gave %q", got)
	}
	// ⚠️ And the store's own unit agrees with it: the same codes, mapped into
	// the words a purchase is written in. Two vocabularies for one fact is
	// already one too many.
	for code, want := range map[int]string{
		11: models.UnitKg, 10: models.UnitKg, 41: models.UnitL, 0: models.UnitPcs,
	} {
		if got := stockUnitOf(&models.MenuItem{UnitCode: code}); got != want {
			t.Errorf("stock unit for %d gave %q, want %q", code, got, want)
		}
	}
}

// ⚠️ **The price is the largest thing on a shelf label and the name is second.**
// It is read from a metre away by somebody deciding whether to pick the packet
// up; the sticker on the packet is read by a scanner, which needs none of the
// words. Which of the six designs those facts land in is the shop's choice —
// what is checked here is that the facts are complete before they get there.
//
// ⚠️ **This test used to assert the bug.** It required the name to start with
// "!" and the price with "!!", which are not the emphasis markers escpos reads —
// so it passed for a label that printed two stray exclamation marks and no
// emphasis at all. A test written against the code rather than against the paper
// keeps whatever the code happened to do.
func TestTheLabelLeadsWithTheNameAndShoutsThePrice(t *testing.T) {
	price := 24000
	d := labelData(
		models.MenuItem{
			Name: "Guruch Lazer", Price: 18500, UnitCode: 10,
			OldPrice: &price,
		},
		models.Branch{Name: "Chilonzor"}, "so'm", "06.09.2026",
	)
	if d.Name != "Guruch Lazer" || d.Shop != "Chilonzor" {
		t.Errorf("the label does not name the product and the shop: %+v", d)
	}
	if d.Unit != "kg" {
		t.Errorf("unit %q — the price does not say what it is per", d.Unit)
	}
	if d.OldPrice != 24000 {
		t.Errorf("a genuine reduction was dropped: %+v", d)
	}

	lines := receipt.RenderLabel(receipt.LabelTemplate{}, d)
	if !strings.HasPrefix(lines[0], escpos.MarkBold) ||
		!strings.Contains(lines[0], "Guruch Lazer") {
		t.Errorf("first line %q is not the name, emphasised", lines[0])
	}
	last := lines[len(lines)-1]
	if !strings.HasPrefix(last, escpos.MarkBoldBig) {
		t.Errorf("the price line %q is not the big one", last)
	}
	if !strings.Contains(last, "18 500") || !strings.Contains(last, "kg") {
		t.Errorf("the price line %q is not the price per unit", last)
	}
}

// ⚠️ **A sale label is only ever printed over a real reduction.** `oldPrice` is
// a nullable field a panel may have left holding the same number, and a shelf
// that claims a discount which did not happen is a claim we printed ourselves.
func TestAnOldPriceThatIsNotLowerIsNotASale(t *testing.T) {
	same := 18500
	d := labelData(
		models.MenuItem{Name: "Guruch", Price: 18500, OldPrice: &same},
		models.Branch{}, "so'm", "",
	)
	if d.OldPrice != 0 {
		t.Errorf("an unchanged price was carried as a reduction: %d", d.OldPrice)
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

// ---- The design the shop chose ----

// ⚠️ **The emphasis markers are control characters, and they were literal
// exclamation marks.** `labelLines` wrote "!name" and "!!price" — which is not
// what escpos reads, so every label came out at one size with two stray
// punctuation marks on it. Nothing failed; the paper was just wrong, and only a
// shop holding one would have found it.
func TestLabelEmphasisUsesTheMarkersAndNotPunctuation(t *testing.T) {
	src := readSource(t, "labels.go")
	if strings.Contains(src, `"!" + name`) || strings.Contains(src, `"!!"+price`) {
		t.Error("emphasis is written as punctuation the printer prints")
	}
	if !strings.Contains(src, "receipt.RenderLabel") {
		t.Error("the label is laid out somewhere other than the renderer")
	}
}

// ⚠️ **The price tag has no bars, and the queue has to honour that.** Encoding
// them anyway would print a barcode under the one design chosen for not having
// one — and the shop would conclude the chooser does nothing.
func TestTheQueueAsksTheDesignWhetherToPrintBars(t *testing.T) {
	fn := between(t, readSource(t, "labels.go"), "func (h *Handler) queueLabels", "\n}\n")
	if !strings.Contains(fn, "tpl.Bars()") {
		t.Error("the barcode is encoded without asking the design")
	}
}

// ⚠️ **A save from the labels screen must not blank the branch's printers.**
// They live in one document because they are one machine's settings, and a
// whole-document write from the smaller form is how the bigger one is lost.
func TestSavingTheDesignTouchesOnlyTheDesign(t *testing.T) {
	fn := between(t, readSource(t, "labeldesign.go"),
		"func (h *Handler) AdminSaveLabelDesign", "\n}\n")
	if strings.Contains(fn, `"printers"`) || strings.Contains(fn, `"kitchen"`) {
		t.Error("the design save writes the receipts as well")
	}
	if !strings.Contains(fn, `"label": tpl`) {
		t.Error("the design is not written")
	}
}

// ⚠️ **And the receipts form must not blank the design**, which is the same trap
// from the other side.
func TestSavingTheReceiptsLeavesTheDesignAlone(t *testing.T) {
	fn := between(t, readSource(t, "receipts.go"),
		"func (h *Handler) AdminUpdateReceipts", "\n}\n")
	if strings.Contains(fn, `"label"`) {
		t.Error("the receipts form writes the label design too")
	}
}

// ⚠️ **`label` is a printer kind the editor offers**, and the whitelist that
// validates a save was dropping it — so a shop ticked the box, saved, and every
// label run answered "no printer is set to print labels".
func TestALabelPrinterSurvivesBeingSaved(t *testing.T) {
	in := []models.Printer{{
		Name: "Yorliq", Target: "tcp://192.168.1.50:9100",
		Kinds: []string{"label"},
	}}
	out := cleanPrinters(in)
	if len(out) != 1 || len(out[0].Kinds) != 1 || out[0].Kinds[0] != "label" {
		t.Fatalf("the label kind was dropped on save: %+v", out)
	}
}

// ⚠️ **The preview's sample code has to be a real EAN-13.** An invented one
// fails `barcode.Valid`, so the queue would send it to the printer as CODE128
// while the chooser drew it as an EAN — and the screen whose whole job is "what
// comes out of the printer" would be showing something else. This is the second
// time an EAN has been invented in this repository and the second time the
// check digit caught it.
func TestThePreviewSampleIsARealEAN13(t *testing.T) {
	src := readSource(t, "labeldesign.go")
	m := regexp.MustCompile(`Barcode:\s+"(\d+)"`).FindStringSubmatch(src)
	if m == nil {
		t.Fatal("the sample has no barcode — was it renamed?")
	}
	if !barcode.Valid(m[1]) {
		t.Errorf("the sample code %q is not a valid EAN-13", m[1])
	}
}
