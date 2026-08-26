// Package receipt turns an order into the lines of text a thermal printer puts
// on paper.
//
// # ⚠️ A receipt is a character grid, not a page
//
// Thermal printers do not lay out text — they print a fixed number of monospace
// characters per line and cut. 58 mm paper takes 32 of them, 80 mm takes 48.
// Everything here is arithmetic on that grid, and the single most common way to
// ruin a receipt is to send 48 characters to a 58 mm printer: the ends of every
// line vanish, which on the customer's copy means the totals column.
//
// # ⚠️ Rendered on the server, so the preview cannot lie
//
// The panel's preview and the paper that comes out of the printer are produced
// by **this** function. A preview drawn in the browser would be a second
// implementation of the same layout, and the two would drift — with the
// difference discovered by a restaurant whose receipts do not look like the
// thing they designed. Same rule as the reports: one calculation, two outputs.
//
// # ⚠️ Three receipts, three readers
//
// They are not one template with fields switched off. The kitchen ticket is
// read by a cook at a pass under time pressure and carries **no prices** — a
// price tells them nothing and lengthens the ticket, and every extra line is
// more paper to read. The till copy is for the person with the drawer. The
// guest's copy is the only one that carries the fiscal QR, because it is the
// only one anybody will check.
package receipt

import (
	"restaurant-backend/internal/escpos"

	"strings"
)

// Kind is which receipt is being printed.
type Kind string

const (
	Kitchen  Kind = "kitchen"
	Till     Kind = "till"
	Customer Kind = "customer"
	// The bill handed to a table **before** they pay.
	Precheck Kind = "precheck"
)

// Widths in characters, by paper size.
//
// ⚠️ **A setting, never a guess.** A restaurant with 58 mm paper and a template
// built for 80 mm loses the right-hand end of every line, and the only person
// who finds out is the guest holding the receipt.
const (
	Width58 = 32
	Width80 = 48
)

// WidthFor turns the stored paper size into a character count.
//
// ⚠️ Unknown values fall back to **80 mm**, which is the common one — and the
// safe direction: a template rendered narrow on wide paper wastes space, while
// wide on narrow paper cuts the totals off.
func WidthFor(mm int) int {
	if mm == 58 {
		return Width58
	}
	return Width80
}

// Template is one restaurant's design for one of the three receipts.
type Template struct {
	// Whether this receipt is printed at all. A restaurant with no printer at
	// the pass switches the kitchen ticket off rather than losing the till.
	Enabled bool `bson:"enabled" json:"enabled"`
	// Paper width in millimetres: 58 or 80.
	WidthMM int `bson:"widthMm" json:"widthMm"`
	// Free lines above and below the body. The header usually carries an
	// address and a phone number, the footer a thank-you.
	Header string `bson:"header" json:"header"`
	Footer string `bson:"footer" json:"footer"`
	// How many blank lines to feed before the cut, so the tear-off does not
	// take the last line of text with it. Printer-dependent, hence a setting.
	FeedLines int `bson:"feedLines" json:"feedLines"`
	// Which optional lines appear. Named per receipt rather than shared,
	// because "show prices" is meaningless on a kitchen ticket and "show the
	// dish comment" is the whole point of one.
	Fields map[string]bool `bson:"fields" json:"fields"`

	// Print the restaurant's logo above the header.
	//
	// ⚠️ **Per receipt, and off by default.** The guest's copy is the one a logo
	// belongs on; the kitchen ticket never gets one however this is set (see
	// renderKitchen) — every dot is time at the pass and paper on the roll, and
	// a cook does not need to be told which restaurant they work in. Off by
	// default because a logo is only ever an improvement when somebody has
	// looked at how it comes out: flat artwork prints, a photograph smudges.
	Logo bool `bson:"logo,omitempty" json:"logo,omitempty"`

	// Which language this receipt prints in: "uz" (default), "ru" or "en".
	//
	// ⚠️ **Per receipt, because a kitchen is not a dining room.** A restaurant
	// with Russian-speaking cooks and Uzbek guests is ordinary here, and one
	// setting for both would be wrong for one of them every time.
	//
	// ⚠️ **Empty is Uzbek**, which is what every receipt printed before this
	// existed — so a restaurant that never opens the dropdown sees no change.
	// See words.go for why this follows the restaurant rather than whoever is
	// signed in at the screen.
	Lang string `bson:"lang,omitempty" json:"lang,omitempty"`

	// Which lines are printed larger or in bold.
	//
	// ⚠️ **Named lines, not a font picker.** A thermal printer has two built-in
	// faces and a size multiplier — it cannot be given a typeface, and offering
	// one would promise what the hardware does not have. What it can do is make
	// a chosen line twice the size or bold, and the lines worth that are known:
	// the table number a cook reads across a hot pass, the total a guest checks
	// against their money, the dish names.
	//
	// Keys: "table", "total", "items". ⚠️ Empty is every line at normal weight,
	// which is what every receipt printed before this existed.
	Emphasis map[string]string `bson:"emphasis,omitempty" json:"emphasis,omitempty"`

	// Blank lines printed before anything else.
	//
	// ⚠️ **For the printers whose cutter eats the top of the next receipt.**
	// `FeedLines` already solves the bottom — the tear-off taking the last line
	// — and the same machine often takes the first line of the following one,
	// because the paper sits under the head where it was cut. It is a setting
	// for the same reason: only the person holding the paper can see it.
	TopLines int `bson:"topLines,omitempty" json:"topLines,omitempty"`

	// Print what each guest owes if the bill is divided evenly.
	//
	// ⚠️ **Its own field, not one of `Fields`.** That map's rule is "missing
	// means shown", which is right for a line every receipt used to print and
	// wrong for one no receipt has ever printed: switching this on for every
	// restaurant in the country would put a number on every bill that nobody
	// asked for, and on a table of one it is the total printed twice.
	//
	// ⚠️ **A convenience, never a demand.** It is what each guest owes if they
	// split it evenly — the arithmetic a table does out loud with a phone — and
	// the label says so. A receipt that stated a per-person amount as if it were
	// due would be a restaurant deciding how a party settles among itself.
	SplitPerGuest bool `bson:"splitPerGuest,omitempty" json:"splitPerGuest,omitempty"`
}

// mark returns the prefix a line should carry: "" for normal, or one of the
// escpos markers.
//
// ⚠️ **Unknown values are normal weight rather than an error.** This is a print
// path, and a template written by an older panel — or by hand — must produce a
// readable receipt rather than none.
func (t Template) mark(key string) string {
	switch t.Emphasis[key] {
	case "bold":
		return escpos.MarkBold
	case "big":
		return escpos.MarkBig
	case "boldbig":
		return escpos.MarkBoldBig
	}
	return ""
}

// Shows reports whether an optional line is switched on.
//
// ⚠️ **Missing means shown.** Every template stored before a field existed has
// no entry for it, and reading that as "off" would silently drop a line from
// receipts that were printing it fine — the zero-value rule this codebase
// follows everywhere.
func (t Template) Shows(field string) bool {
	if t.Fields == nil {
		return true
	}
	v, ok := t.Fields[field]
	return !ok || v
}

// Data is everything the renderer needs, in our own vocabulary.
//
// ⚠️ Deliberately not an *models.Order. The renderer is arithmetic on strings
// and has no business knowing about Mongo, brands or payment providers — and
// keeping it that way is what lets the preview feed it a made-up sale.
type Data struct {
	// The heading: the restaurant, and where it is.
	Title   string
	Address string
	Phone   string

	Number    string
	OpenedAt  string
	ClosedAt  string
	Table     string
	Guests    int
	Server    string
	Cashier   string
	OrderType string

	Lines []Line

	Subtotal int
	Discount int
	// Named so the receipt can say *which* discount, which is the whole reason
	// a guest stops arguing about the total.
	DiscountName string
	// What the room added for service, and the rate it was charged at. ⚠️ On
	// the paper as its own line: see totals().
	Service        int
	ServicePercent int
	Total          int
	Paid           int
	Change         int
	Method         string

	// The fiscal sign and the QR the guest checks. Customer copy only.
	FiscalSign string
	QRText     string

	Currency string
}

// Line is one item.
type Line struct {
	Name string
	Qty  int
	// Per unit, in whole so'm. Ignored on the kitchen ticket.
	Price int
	Sum   int
	// "piyozsiz" — the reason the kitchen ticket exists in this shape.
	Comment string
	Options string

	// Which dish and which section of the menu this is.
	//
	// ⚠️ **Carried for routing, never printed.** A restaurant with a bar and a
	// kitchen needs the drinks on one roll and the food on another, and the
	// only thing that can decide is the dish's own category — the name on the
	// line cannot, and asking a waiter to choose per order would be a question
	// at every table. Neither of these ever appears on paper.
	MenuItemID string
	CategoryID string
}

// Render lays a receipt out as lines of text.
func Render(kind Kind, t Template, d Data) []string {
	w := WidthFor(t.WidthMM)
	b := &block{w: w}

	// ⚠️ **The currency follows the receipt's language unless the restaurant
	// named one.** "so'm" printed under a Russian total was the most repeated
	// wrong word on the paper. A place that prices in dollars still gets
	// dollars in every language — a restaurant's own setting is a fact about
	// its prices, not about who is reading them.
	if d.Currency == "" || d.Currency == "so'm" {
		d.Currency = wordsForReceipt(t.Lang).Currency
	}

	// ⚠️ Bounded rather than trusted: this is a number typed into a box, and a
	// hundred blank lines is a roll of paper on the floor.
	for i := 0; i < t.TopLines && i < 6; i++ {
		b.raw("")
	}

	switch kind {
	case Kitchen:
		renderKitchen(b, t, d)
	case Till:
		renderTill(b, t, d)
	case Precheck:
		renderPrecheck(b, t, d)
	default:
		renderCustomer(b, t, d)
	}

	for range t.FeedLines {
		b.raw("")
	}
	return b.lines
}

// ---- The three ----

// renderKitchen is read at a pass, standing up, under time pressure.
//
// ⚠️ **No prices, and that is not an option anybody can switch on.** A price
// tells a cook nothing, and every character on this ticket is one more to read
// while three others are waiting. The table number is the largest thing on it
// because it is the only thing that has to be right.
func renderKitchen(b *block, t Template, d Data) {
	// ⚠️ **The header prints here too, and its absence was not a decision.**
	// The logo is deliberately kept off a kitchen ticket — every dot is time at
	// the pass — but the header is text the restaurant wrote, and a station
	// name ("ISSIQ SEX") or a shift note is exactly what it is for. It was
	// simply never drawn: the footer was, so an owner filling both in got half
	// of what they typed and no reason for the other half.
	if t.Header != "" {
		b.wrap(t.Header)
		b.rule()
	}
	if d.Table != "" {
		b.center(t.mark("table") + strings.ToUpper(d.Table))
	} else {
		b.center(t.mark("table") + strings.ToUpper(d.OrderType))
	}
	b.center("#" + d.Number)
	b.rule()

	for _, l := range d.Lines {
		// Quantity first: it is what a cook counts, and a leading number is
		// found faster than one at the end of a name.
		b.wrap(t.mark("items") + itoa(l.Qty) + " x " + l.Name)
		if l.Options != "" {
			b.wrap("  " + l.Options)
		}
		// ⚠️ The comment is indented and never truncated. "piyozsiz" is the one
		// line on this ticket that a guest will notice if it is missed, and a
		// cut-off comment reads as no comment at all.
		if l.Comment != "" && t.Shows("comment") {
			b.wrap("  * " + l.Comment)
		}
	}

	b.rule()
	if t.Shows("time") {
		b.line(d.OpenedAt, "")
	}
	if d.Server != "" && t.Shows("server") {
		b.wrap(d.Server)
	}
	multiline(b, t.Footer)
}

// renderTill is the counter's own copy: the money, and who took it.
func renderTill(b *block, t Template, d Data) {
	w := wordsForReceipt(t.Lang)
	header(b, t, d, false)
	b.line("#"+d.Number, d.Table)
	if t.Shows("time") {
		b.line(d.ClosedAt, "")
	}
	b.rule()
	items(b, d, true)
	b.rule()
	totals(b, t, d)
	if d.Cashier != "" && t.Shows("cashier") {
		b.line(w.Cashier, d.Cashier)
	}
	if strings.TrimSpace(t.Footer) != "" {
		b.rule()
		multiline(b, t.Footer)
	}
}

// renderPrecheck is the bill a table is given before they pay.
//
// ⚠️ **It must not be mistakable for the receipt.** It carries the same dishes
// and the same total, and that is exactly the danger: a guest handed a document
// that looks like a fiscal receipt has been told the sale is registered when it
// is not, and a cashier holding one has no way to tell it apart at the end of
// the evening. So it never carries a fiscal sign — there is none yet — and it
// says on the paper, in the guest's own language, that this is a bill and the
// receipt follows the payment.
//
// ⚠️ **No "paid" and no change.** Nothing has been paid; printing a zero there
// would be answering a question nobody asked with a number that looks like a
// fact.
func renderPrecheck(b *block, t Template, d Data) {
	w := wordsForReceipt(t.Lang)
	header(b, t, d, true)
	b.line("#"+d.Number, d.Table)
	if t.Shows("time") {
		b.line(d.OpenedAt, "")
	}
	if d.Server != "" && t.Shows("server") {
		b.line(w.Server, d.Server)
	}
	if d.Guests > 0 && t.Shows("guests") {
		b.line(w.Guests, itoa(d.Guests))
	}
	b.rule()
	items(b, d, true)
	b.rule()
	// ⚠️ **Cleared here, not trusted to the caller.** Nothing has been paid, and
	// this document is built from the same data the receipt is — a check being
	// re-billed after a refused card still carries what was tendered. The
	// renderer decides what each kind of paper may say; a "change: 5 000" line
	// on a bill is money the guest has not handed over and would be right to
	// expect back.
	d.Paid, d.Change, d.Method = 0, 0, ""
	totals(b, t, d)
	b.raw("")
	// ⚠️ Centred and on its own line rather than folded into the footer the
	// owner edits: the one sentence that keeps this document honest cannot be a
	// setting somebody switches off to save a line of paper.
	b.center(w.PrecheckNote)
	if t.Footer != "" {
		b.raw("")
		b.center(t.Footer)
	}
}

// PrecheckNote is what a bill says instead of a fiscal sign.
//
// Uzbek, and not translated per guest: the paper is read at a table by whoever
// is sitting there, the restaurant hands it over without knowing who that is,
// and a receipt printer has one character set the restaurant has already
// checked. The same reasoning as the receipt designer's other fixed words.
const PrecheckNote = "HISOB — fiskal chek emas"

// renderCustomer is the copy that leaves the building.
//
// ⚠️ **The only one that carries the fiscal QR.** It is the guest's right to
// check the sale, and printing it on the kitchen ticket would be paper spent on
// somebody who cannot use it.
func renderCustomer(b *block, t Template, d Data) {
	w := wordsForReceipt(t.Lang)
	header(b, t, d, true)
	b.line("#"+d.Number, d.Table)
	if t.Shows("time") {
		b.line(d.ClosedAt, "")
	}
	if d.Server != "" && t.Shows("server") {
		b.line("Ofitsiant", d.Server)
	}
	b.rule()
	items(b, d, true)
	b.rule()
	totals(b, t, d)

	if d.FiscalSign != "" {
		b.raw("")
		b.center(w.FiscalSign)
		b.center(d.FiscalSign)
	}
	if t.Footer != "" {
		b.raw("")
		b.center(t.Footer)
	}
}

// ---- Shared pieces ----

func header(b *block, t Template, d Data, full bool) {
	if d.Title != "" {
		b.center(strings.ToUpper(d.Title))
	}
	if full {
		if d.Address != "" && t.Shows("address") {
			b.center(d.Address)
		}
		if d.Phone != "" && t.Shows("phone") {
			b.center(d.Phone)
		}
	}
	multiline(b, t.Header)
	b.rule()
}

// multiline prints free text the way somebody typed it.
//
// ⚠️ **Centred per line and wrapped, not truncated.** The header is documented
// as the place for an address and a phone number, and it was drawn with
// `center`, which cuts at the paper's width — so a restaurant that typed both
// got the first half of the first one. Newlines are what somebody reaches for
// when one line is not enough, and until now they were printed as a single
// long line and then cut.
func multiline(b *block, s string) {
	if strings.TrimSpace(s) == "" {
		return
	}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimRight(line, " \r")
		if line == "" {
			// A blank line the owner typed is a blank line: it is how a phone
			// number is separated from a thank-you.
			b.raw("")
			continue
		}
		// ⚠️ Wrapped rather than cut, and centred line by line. A long line
		// becomes two centred lines, which is what the text was for.
		for _, part := range splitToWidth(line, b.w) {
			b.center(part)
		}
	}
}

func items(b *block, d Data, prices bool) {
	for _, l := range d.Lines {
		b.wrap(l.Name)
		if l.Options != "" {
			b.wrap("  " + l.Options)
		}
		if prices {
			// ⚠️ "2 × 45 000" on the left and the line total on the right: a
			// guest checking a receipt is verifying the multiplication, and
			// hiding one of the three numbers makes that impossible.
			b.line("  "+itoa(l.Qty)+" × "+money(l.Price, d.Currency),
				money(l.Sum, d.Currency))
		}
	}
}

func totals(b *block, t Template, d Data) {
	w := wordsForReceipt(t.Lang)
	if d.Discount > 0 || d.Service > 0 {
		b.line(w.Subtotal, money(d.Subtotal, d.Currency))
	}
	if d.Discount > 0 {
		name := d.DiscountName
		if name == "" {
			name = w.Discount
		}
		b.line(name, "-"+money(d.Discount, d.Currency))
	}
	// ⚠️ **Its own line, with the rate on it.** A service charge folded into
	// the total is the single most common complaint about restaurant bills
	// anywhere, and the guest is holding the only document that can answer it.
	// Naming the percentage saves them dividing one number by another at a
	// table in bad light.
	if d.Service > 0 {
		label := "Xizmat haqi"
		if d.ServicePercent > 0 {
			label += " " + itoa(d.ServicePercent) + "%"
		}
		b.line(label, money(d.Service, d.Currency))
	}
	b.lineMarked(t.mark("total"), w.Total, money(d.Total, d.Currency))
	// ⚠️ **Under the total, and only when it says something.** Two guests or
	// more, a total worth dividing, and the setting on: at one guest this is
	// the total printed a second time, and a division that came out at zero
	// would be a line reading "1 kishi: 0 so'm" on a bill for nothing.
	//
	// ⚠️ Rounded **up**, and the label says "taxminan". Dividing 100 000 by
	// three gives a number that cannot be paid in cash, and three guests each
	// handing over the rounded-down share leave the restaurant short — so the
	// rounding goes the way that covers the bill, and the word admits it is not
	// exact.
	if t.SplitPerGuest && d.Guests > 1 && d.Total > 0 {
		per := (d.Total + d.Guests - 1) / d.Guests
		b.line(itoa(d.Guests)+" "+w.PerGuest, money(per, d.Currency))
	}
	if d.Method != "" {
		b.line(d.Method, money(d.Paid, d.Currency))
	}
	// ⚠️ Change is printed only when there is any. A "Qaytim: 0" line on every
	// card payment is a line the guest has to read past to find the total.
	if d.Change > 0 && t.Shows("change") {
		b.line(w.Change, money(d.Change, d.Currency))
	}
}
