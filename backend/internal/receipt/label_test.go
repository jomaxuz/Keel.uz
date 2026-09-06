package receipt

import (
	"strings"
	"testing"

	"restaurant-backend/internal/escpos"
)

// The six label designs.
//
// ⚠️ **These are rules about paper nobody can see from here.** A sticker is
// printed in a shop, stuck to a packet and read by a scanner; every failure in
// this file is discovered by a queue at a counter, and none of them raises an
// error anywhere. So the awkward parts are pinned rather than trusted to a
// reading: the price fits, the bars are there, and the sale label does not claim
// a reduction that did not happen.

func sampleLabel() LabelData {
	return LabelData{
		Name:     "Guruch Lazer, oq, 1 kg",
		Shop:     "Chilonzor",
		Price:    18500,
		OldPrice: 24000,
		Unit:     "kg",
		Barcode:  "2100000000017",
		Currency: "so'm",
		Date:     "06.09.2026",
	}
}

func joined(lines []string) string { return strings.Join(lines, "\n") }

// ⚠️ **Every design prints the price except the one that is only a code.** A
// label with no price on it is a design; a label that lost its price is a bug,
// and on a shelf the two look identical.
func TestEveryDesignPrintsSomethingUseful(t *testing.T) {
	for _, style := range LabelStyles {
		lines := RenderLabel(LabelTemplate{Style: string(style)}, sampleLabel())
		if len(lines) == 0 {
			t.Fatalf("%s printed nothing", style)
		}
		if !strings.Contains(joined(lines), "Guruch") {
			t.Errorf("%s does not name the product", style)
		}
		if style == LabelSticker {
			continue
		}
		if !strings.Contains(joined(lines), "18 500") {
			t.Errorf("%s does not print the price", style)
		}
	}
}

// ⚠️ **Nothing runs off the paper.** A double-width line prints two columns per
// character, so a price centred against the full width lands with its last
// digits past the edge — and the shop reads a smaller number than it charges.
func TestNoLinePrintsPastTheEdgeOfTheRoll(t *testing.T) {
	for _, mm := range []int{58, 80} {
		w := WidthFor(mm)
		for _, style := range LabelStyles {
			lines := RenderLabel(
				LabelTemplate{Style: string(style), WidthMM: mm}, sampleLabel())
			for _, line := range lines {
				mark, text := splitMark(line)
				limit := w
				if mark == escpos.MarkBig || mark == escpos.MarkBoldBig {
					limit = w / 2
				}
				if width(text) > limit {
					t.Errorf("%s at %dmm: %q is %d wide, paper holds %d",
						style, mm, text, width(text), limit)
				}
			}
		}
	}
}

// ⚠️ **A price is never cut to make it large.** The big face holds half the
// characters, and a truncated amount is a *smaller number* than the till will
// charge — a shelf that lies in the customer's favour, and an argument the shop
// loses. Saffron, meat and imported cheese all reach this width.
func TestALongPriceIsPrintedSmallRatherThanCut(t *testing.T) {
	d := sampleLabel()
	d.Price = 1750000
	d.OldPrice = 2400000
	for _, style := range LabelStyles {
		lines := RenderLabel(LabelTemplate{Style: string(style)}, d)
		if style == LabelSticker {
			continue
		}
		if !strings.Contains(joined(lines), "1 750 000") {
			t.Errorf("%s lost part of the price: %q", style, joined(lines))
		}
	}
}

// ⚠️ **A sale label with nothing on sale is a lie on a shelf**, and the only
// mistake here a customer could hold the shop to.
func TestTheSaleDesignStepsAsideWhenNothingIsReduced(t *testing.T) {
	d := sampleLabel()
	d.OldPrice = 0
	lines := RenderLabel(LabelTemplate{Style: string(LabelSale)}, d)
	if strings.Contains(joined(lines), "AKSIYA") {
		t.Error("a sale banner is printed over an unchanged price")
	}
	// And it is the plain shelf label rather than nothing at all.
	if joined(lines) != joined(RenderLabel(LabelTemplate{Style: string(LabelShelf)}, d)) {
		t.Error("the fallback is not the shelf label")
	}
}

// ⚠️ **The large number is the one the customer pays.** An old price printed as
// loudly as the new one is a shelf that says two things.
func TestTheSaleDesignShoutsTheNewPriceOnly(t *testing.T) {
	lines := RenderLabel(LabelTemplate{Style: string(LabelSale)}, sampleLabel())
	for _, line := range lines {
		mark, text := splitMark(line)
		if strings.Contains(text, "24 000") && mark == escpos.MarkBoldBig {
			t.Error("the old price is printed at the size of the new one")
		}
	}
	if !strings.Contains(joined(lines), "24 000") {
		t.Error("the old price is not printed at all")
	}
}

// ⚠️ **The zero value has to print something.** A branch whose settings were
// written before this field existed decodes into an empty template, and a blank
// sticker is indistinguishable from a broken printer.
func TestAnEmptyTemplateIsTheShelfLabelOn58mm(t *testing.T) {
	d := LabelTemplate{}.Defaults()
	if d.Style != string(LabelShelf) {
		t.Errorf("empty style became %q", d.Style)
	}
	if d.WidthMM != 58 {
		t.Errorf("empty width became %d, want the label roll's 58", d.WidthMM)
	}
	if len(RenderLabel(LabelTemplate{}, sampleLabel())) == 0 {
		t.Error("an unset template prints a blank sticker")
	}
}

// ⚠️ **Missing means shown**, the zero-value rule this codebase follows: a shop
// that has never opened the chooser has no entries, and reading that as "off"
// would strip the shop's name off labels it was printing fine.
func TestAnUntouchedTemplateShowsTheOptionalLines(t *testing.T) {
	lines := joined(RenderLabel(LabelTemplate{Style: string(LabelShelf)}, sampleLabel()))
	if !strings.Contains(lines, "Chilonzor") {
		t.Error("the shop's name is missing from an untouched template")
	}
	if !strings.Contains(lines, "/ kg") {
		t.Error("the unit is missing from an untouched template")
	}
	off := LabelTemplate{
		Style:  string(LabelShelf),
		Fields: map[string]bool{"shop": false, "unit": false},
	}
	if strings.Contains(joined(RenderLabel(off, sampleLabel())), "Chilonzor") {
		t.Error("the shop's name is printed after being switched off")
	}
}

// ⚠️ **The price tag is the one design with no bars**, and the panel is told so
// rather than guessing: a shop choosing between six pictures must not be shown a
// barcode on the one design that has none.
func TestOnlyThePriceTagDropsTheBarcode(t *testing.T) {
	for _, style := range LabelStyles {
		want := style != LabelPrice
		if style.Bars() != want {
			t.Errorf("%s: bars=%v, want %v", style, style.Bars(), want)
		}
	}
}

// ⚠️ **The sticker stays one line.** It is chosen for 30 mm labels, and a name
// that wrapped to three lines would push the bars off the sticker — which is the
// one failure that stops a sale at the till.
func TestTheStickerNeverGrowsWithTheName(t *testing.T) {
	d := sampleLabel()
	d.Name = strings.Repeat("juda uzun nom ", 12)
	lines := RenderLabel(LabelTemplate{Style: string(LabelSticker)}, d)
	if len(lines) != 1 {
		t.Errorf("a long name made the sticker %d lines", len(lines))
	}
}

// ⚠️ **An unknown language is Uzbek**, which is what every label printed before
// the setting existed.
func TestTheWordsFollowTheChosenLanguage(t *testing.T) {
	ru := RenderLabel(
		LabelTemplate{Style: string(LabelSale), Lang: "ru"}, sampleLabel())
	if !strings.Contains(joined(ru), "АКЦИЯ") {
		t.Error("the Russian sale banner is not printed")
	}
	uz := RenderLabel(
		LabelTemplate{Style: string(LabelSale), Lang: "klingon"}, sampleLabel())
	if !strings.Contains(joined(uz), "AKSIYA") {
		t.Error("an unknown language did not fall back to Uzbek")
	}
}
