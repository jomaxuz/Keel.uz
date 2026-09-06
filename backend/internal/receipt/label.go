package receipt

// ---- The shop's own label ----
//
// ⚠️ **A label is not one thing, and that is why there are six of them.** The
// same button prints a price tag clipped to a shelf edge and read from two
// metres, a sticker wrapped round a packet of nuts that only a scanner ever
// looks at, and a promotion card that has to say what the price used to be.
// One layout serving all of those is a layout that is wrong for two of them:
// the shelf tag wastes half a roll on a 30 mm sticker, and the sticker's small
// print is unreadable across an aisle. So the shop picks, once, and the choice
// is a fact about that shop's shelves rather than about our taste.
//
// ⚠️ **Laid out here, beside the receipts, and not in the handler.** The label
// is printed by the same character grid, in the same fonts, onto the same kind
// of roll — and the panel's preview has to be drawn by this code rather than by
// a second implementation in the browser. That rule has already been paid for
// once: two layout engines drift, and the drift is found by a shop whose paper
// does not look like the design they chose.

import (
	"strings"

	"restaurant-backend/internal/escpos"
)

// LabelStyle is which design comes out.
type LabelStyle string

const (
	// LabelShelf — name, shop, price. What a shelf edge in a small shop wants,
	// and what every label printed before this setting existed looked like.
	LabelShelf LabelStyle = "shelf"
	// LabelPrice — the price and almost nothing else, as large as the head can
	// draw it. ⚠️ **The one design with no bars**: it is a price tag for a
	// shelf, not a sticker for a packet, and the bars would take the room the
	// price is there to fill.
	LabelPrice LabelStyle = "price"
	// LabelSticker — a name and the bars. For goods whose price is on the shelf
	// and not on the packet; the shortest piece of paper of the six.
	LabelSticker LabelStyle = "sticker"
	// LabelCompact — name and price on one line, then the bars. For a shop
	// counting metres of roll.
	LabelCompact LabelStyle = "compact"
	// LabelSale — what it cost and what it costs now.
	LabelSale LabelStyle = "sale"
	// LabelFull — everything: the shop, the name, the price, the date it was
	// printed. For a shop that wants the shelf to say when it was last checked.
	LabelFull LabelStyle = "full"
)

// LabelStyles is every design, in the order the panel offers them.
//
// ⚠️ The order is "commonest first", not alphabetical: the shop scrolling this
// list is choosing once, in a hurry, on the day they plug the printer in.
var LabelStyles = []LabelStyle{
	LabelShelf, LabelPrice, LabelSticker, LabelCompact, LabelSale, LabelFull,
}

// ValidLabelStyle reports whether a stored or submitted name is one of ours.
//
// ⚠️ Unknown falls back to the shelf label rather than to nothing — see
// Defaults. A design that printed a blank sticker because a name changed would
// be discovered by a shop holding an empty roll.
func ValidLabelStyle(s string) bool {
	for _, k := range LabelStyles {
		if LabelStyle(s) == k {
			return true
		}
	}
	return false
}

// Bars reports whether the barcode is printed under this design.
func (s LabelStyle) Bars() bool { return s != LabelPrice }

// LabelTemplate is one shop's choice of label.
//
// ⚠️ **Its own struct rather than a `Template` with a style field.** The three
// receipts are laid out by a renderer that knows about lines, totals and
// change; none of that exists on a sticker, and a shared struct would offer a
// shop a footer, a logo and a "show the cashier" switch that do nothing. The
// same reason the receipts are three templates and not one.
type LabelTemplate struct {
	// Which design. Empty is the shelf label — every label printed before this
	// setting existed was that, so an install that never opens the chooser sees
	// no change.
	Style string `bson:"style,omitempty" json:"style"`

	// Paper width in millimetres: 58 or 80.
	//
	// ⚠️ **58 by default here, where the receipts default to 80.** A label roll
	// is the narrow one; a design laid out for 80 mm and printed on 58 loses its
	// right-hand end, which on a price tag is the price.
	WidthMM int `bson:"widthMm,omitempty" json:"widthMm"`

	// Blank lines fed before the cut, so the tear-off is below the bars rather
	// than through them. ⚠️ A barcode cut in half scans as nothing.
	FeedLines int `bson:"feedLines,omitempty" json:"feedLines"`

	// Which language the few words on it are printed in ("uz", "ru", "en").
	Lang string `bson:"lang,omitempty" json:"lang,omitempty"`

	// Which optional lines appear: "shop", "unit", "date".
	//
	// ⚠️ **Missing means shown**, the zero-value rule this codebase follows
	// everywhere: a shop that has never opened the chooser has no entries here,
	// and reading that as "off" would silently strip the shop's name off every
	// label it was printing fine.
	Fields map[string]bool `bson:"fields,omitempty" json:"fields"`
}

// Defaults fills in what a stored document does not have.
//
// ⚠️ **The zero value has to print something.** A branch whose receipt document
// was written before labels existed decodes into a zero template, and a
// zero-width sticker with no design is indistinguishable, from the shop floor,
// from a broken printer.
func (t LabelTemplate) Defaults() LabelTemplate {
	if !ValidLabelStyle(t.Style) {
		t.Style = string(LabelShelf)
	}
	if t.WidthMM != 80 {
		t.WidthMM = 58
	}
	if t.FeedLines < 0 || t.FeedLines > 10 {
		t.FeedLines = 2
	}
	return t
}

// Shows reports whether an optional line is switched on.
func (t LabelTemplate) Shows(field string) bool {
	if t.Fields == nil {
		return true
	}
	v, ok := t.Fields[field]
	return !ok || v
}

// Bars reports whether this template's design prints a barcode.
func (t LabelTemplate) Bars() bool {
	return LabelStyle(t.Defaults().Style).Bars()
}

// LabelData is one sticker's facts.
//
// ⚠️ Words, not ids: the renderer is handed the unit as the shop says it ("kg")
// rather than the classifier code it is stored as, because the code means
// nothing on paper and the translation of it belongs where the catalogue is.
type LabelData struct {
	Name  string
	Shop  string
	Price int
	// What it cost before. ⚠️ Zero means "not on sale", and the sale design
	// falls back to the plain one when it is — see RenderLabel.
	OldPrice int
	Unit     string
	Barcode  string
	Currency string
	// The day it was printed, already formatted. ⚠️ Formatted by the caller
	// because the shop's day is not the server's: the same trap the reports
	// pay for, one roll of paper along.
	Date string
}

// RenderLabel lays out one sticker, ready for escpos.Encode.
func RenderLabel(t LabelTemplate, d LabelData) []string {
	t = t.Defaults()
	w := wordsForLabel(t.Lang)
	b := &block{w: WidthFor(t.WidthMM)}

	style := LabelStyle(t.Style)
	// ⚠️ **A sale label with nothing on sale is a lie on a shelf**, and it is
	// the one mistake here that a customer could hold us to: the price is what
	// it always was, and the paper says otherwise. So the design steps aside
	// for the plain one rather than printing a banner over an unchanged price.
	if style == LabelSale && d.OldPrice <= d.Price {
		style = LabelShelf
	}

	switch style {
	case LabelPrice:
		labelPriceTag(b, t, d, w)
	case LabelSticker:
		labelSticker(b, d)
	case LabelCompact:
		labelCompact(b, t, d, w)
	case LabelSale:
		labelSale(b, t, d, w)
	case LabelFull:
		labelFull(b, t, d, w)
	default:
		labelShelf(b, t, d, w)
	}
	return b.lines
}

// priceLine is the amount with the unit beside it.
//
// ⚠️ **The unit is not decoration.** "18 500" on a shelf of loose goods does not
// say whether that is a kilo or a packet, and the difference is what the
// customer pays.
func priceLine(t LabelTemplate, d LabelData, w labelWords) string {
	out := money(d.Price, currencyOf(d, w))
	if d.Unit != "" && t.Shows("unit") {
		out += " / " + d.Unit
	}
	return out
}

func currencyOf(d LabelData, w labelWords) string {
	if d.Currency != "" {
		return d.Currency
	}
	return w.Currency
}

// labelShelf is the default: name, shop, price, bars.
func labelShelf(b *block, t LabelTemplate, d LabelData, w labelWords) {
	b.wrap(escpos.MarkBold + d.Name)
	if d.Shop != "" && t.Shows("shop") {
		b.raw(d.Shop)
	}
	bigPrice(b, priceLine(t, d, w))
}

// labelPriceTag is the price and almost nothing else.
//
// ⚠️ **Centred against half the paper on the big line.** A double-width line
// prints two columns per character, so padding it to the middle of 32 columns
// puts it in the middle of 64 — which is to say off the right-hand edge, with
// the last digits of the price missing. That is the whole bug this helper
// exists to not have.
func labelPriceTag(b *block, t LabelTemplate, d LabelData, w labelWords) {
	b.center(escpos.MarkBold + d.Name)
	centerBig(b, escpos.MarkBoldBig+money(d.Price, currencyOf(d, w)))
	if d.Unit != "" && t.Shows("unit") {
		b.center("1 " + d.Unit)
	}
	if d.Shop != "" && t.Shows("shop") {
		b.rule()
		b.center(d.Shop)
	}
}

// labelSticker is a name and the bars, and nothing else fits.
//
// ⚠️ **One line, truncated rather than wrapped.** This design is chosen for
// 30 mm stickers, where a name that wrapped to three lines would push the bars
// off the label — and a sticker with no bars is the one failure that stops a
// sale at the till.
func labelSticker(b *block, d LabelData) {
	b.raw(d.Name)
}

// labelCompact puts the name and the price on one line.
func labelCompact(b *block, t LabelTemplate, d LabelData, w labelWords) {
	b.lineMarked(escpos.MarkBold, d.Name, priceLine(t, d, w))
}

// labelSale says what it cost and what it costs now.
//
// ⚠️ **The old price is printed above the new one and in plain weight**, so the
// large number on the label is always the one the customer pays. A struck-out
// price is not available — a thermal head has no strike-through — and the word
// in front of it is what does that job instead.
func labelSale(b *block, t LabelTemplate, d LabelData, w labelWords) {
	// ⚠️ centerBig, not center: the banner is double-width, and centring it
	// against the full paper puts it past the right-hand edge. The test caught
	// this one — which is the point of testing a thing nobody here can see.
	centerBig(b, escpos.MarkBoldBig+w.Sale)
	b.wrap(escpos.MarkBold + d.Name)
	b.raw(w.Was + ": " + money(d.OldPrice, currencyOf(d, w)))
	bigPrice(b, priceLine(t, d, w))
}

// labelFull is the shelf label with the paperwork on it.
func labelFull(b *block, t LabelTemplate, d LabelData, w labelWords) {
	if d.Shop != "" && t.Shows("shop") {
		b.raw(d.Shop)
		b.rule()
	}
	b.wrap(escpos.MarkBold + d.Name)
	bigPrice(b, priceLine(t, d, w))
	if d.Date != "" && t.Shows("date") {
		b.line(w.Printed, d.Date)
	}
}

// bigPrice writes an amount as large as it will actually fit.
//
// ⚠️ **Never truncated to make it fit.** A double-width line holds half the
// characters of a plain one, and a price cut to that width is a *smaller
// number* than the till will charge — a shelf that lies in the customer's
// favour, and an argument at the counter that the shop loses. A long price is
// printed at ordinary size instead: less shouty, still true. Loose goods at a
// million and a half a kilo are ordinary in a shop that sells saffron or meat.
func bigPrice(b *block, s string) {
	if width(s) <= b.w/2 {
		b.raw(escpos.MarkBoldBig + s)
		return
	}
	b.raw(escpos.MarkBold + s)
}

// centerBig centres a double-width line.
//
// ⚠️ Half the columns, because every character it contains takes two. See
// labelPriceTag for what the ordinary `center` does to a price.
func centerBig(b *block, s string) {
	mark, text := splitMark(s)
	half := b.w / 2
	// ⚠️ Same rule as bigPrice: too long for the large face is printed at the
	// ordinary one rather than cut. See there for what a cut price costs.
	if width(text) > half {
		if mark == escpos.MarkBoldBig {
			mark = escpos.MarkBold
		} else {
			mark = ""
		}
		b.center(mark + text)
		return
	}
	pad := (half - width(text)) / 2
	b.lines = append(b.lines, mark+strings.Repeat(" ", pad)+text)
}

// labelWords is the handful of words a sticker carries.
//
// ⚠️ **A struct rather than a map**, for the reason the receipt's words are: a
// missing key would be a blank space in front of a price on a shelf, which is
// worse than the wrong language.
type labelWords struct {
	Sale     string
	Was      string
	Printed  string
	Currency string
}

// wordsForLabel picks the language. ⚠️ Unknown or empty is Uzbek — what every
// label printed before this setting existed.
func wordsForLabel(lang string) labelWords {
	switch lang {
	case "ru":
		return labelWords{Sale: "АКЦИЯ", Was: "Старая цена", Printed: "Напечатано", Currency: "сум"}
	case "en":
		return labelWords{Sale: "SALE", Was: "Was", Printed: "Printed", Currency: "so'm"}
	}
	return labelWords{Sale: "AKSIYA", Was: "Eski narx", Printed: "Bosilgan", Currency: "so'm"}
}
