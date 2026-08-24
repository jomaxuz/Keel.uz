package receipt

import "strconv"

// The shift report — X and Z.
//
// ⚠️ **Two reports, one layout, and the difference is not cosmetic.** X is read
// mid-shift and changes nothing: "how are we doing, is the drawer roughly
// right, how much cash should I be able to see". Z is the end of the day — it
// closes the shift, freezes the expected figure and is the paper the owner
// keeps. Printing the same numbers under one heading would let somebody hand
// over an X and call it the day's close, which is the one document in a
// restaurant that has to be unambiguous.
//
// ⚠️ **No dish list.** A shift report is money, not food: what sold is the
// sales report's question and it answers it better, across any period. Two
// hundred lines of ticket paper at 2am is a report nobody reads twice.
//
// ⚠️ **The variance is on the paper, with its reason.** A Z report that shows
// only what was counted is a receipt for whatever the drawer happened to hold —
// the shortfall it exists to record has been overwritten by the person who
// might have caused it (the same rule the closing screen follows).

// ShiftData is one cash shift, as the paper shows it.
type ShiftData struct {
	// Which language to print the labels in: "uz" (default), "ru", "en".
	//
	// ⚠️ Sent by the screen rather than read from a setting: the till is a
	// shared machine and the language is whoever unlocked it, not a property of
	// the restaurant. An unknown or empty value prints Uzbek — the alternative
	// to the wrong language is no report, and the drawer cannot be closed
	// without one.
	Lang string

	Title   string
	Address string
	Phone   string

	// "X" or "Z". On the paper, and in large letters: see above.
	Kind   string
	Branch string

	OpenedAt string
	OpenedBy string
	// Empty on an X — the shift is still open, and printing a closing time
	// that has not happened is the whole confusion this avoids.
	ClosedAt  string
	ClosedBy  string
	PrintedAt string

	// What was sold, by how it was paid.
	Checks   int
	Guests   int
	Sales    int
	Cash     int
	Card     int
	Transfer int
	Discount int
	Service  int
	Refunded int
	// Taken away on the slate this shift. Not in Sales — nobody paid.
	Debt int
	// Checks voided before payment. Shown because a shift with twenty of them
	// is a story, and no other line on this paper would carry it.
	Cancelled int

	// The drawer.
	OpeningFloat int
	CounterCash  int
	// Of the cash sales above, what was owed from an earlier shift. Printed
	// only when it happened, and only to explain a drawer that holds more than
	// the shift sold.
	DebtPaid    int
	Settlements int
	ManualIn    int
	ManualOut   int
	Expected    int
	// Zero until the drawer has been counted, which only happens on a Z.
	Counted      int
	Variance     int
	VarianceNote string

	Currency string
}

// RenderShift lays out an X or Z report.
func RenderShift(t Template, d ShiftData) []string {
	width := WidthFor(t.WidthMM)
	b := &block{w: width}
	// ⚠️ **The words follow the screen that asked, and only on this report.**
	// A guest's receipt stays in the restaurant's own language whatever the
	// panel is set to; this one is read by exactly one person — whoever pressed
	// the button — so printing it in a language they cannot read makes it
	// useless to its only reader. See shiftwords.go.
	w := wordsFor(d.Lang)

	if t.Header != "" {
		b.center(t.Header)
	}
	if d.Title != "" {
		b.center(d.Title)
	}
	// ⚠️ Not when they are the same word. A single-branch restaurant names its
	// branch after itself, and the report opened with its own name twice —
	// which reads as a rendering fault on the one document an owner keeps.
	if d.Branch != "" && d.Branch != d.Title {
		b.center(d.Branch)
	}
	b.raw("")
	// ⚠️ The kind is the first thing on the paper and it stands alone. Somebody
	// sorting a pile of these at the end of a week is reading one word.
	title := w.ZTitle
	if d.Kind == "X" {
		title = w.XTitle
	}
	b.center(title)
	b.rule()

	b.line(w.OpenedAt, d.OpenedAt)
	if d.OpenedBy != "" {
		b.line(w.OpenedBy, d.OpenedBy)
	}
	if d.ClosedAt != "" {
		b.line(w.ClosedAt, d.ClosedAt)
		if d.ClosedBy != "" {
			b.line(w.ClosedBy, d.ClosedBy)
		}
	}
	if d.PrintedAt != "" {
		b.line(w.PrintedAt, d.PrintedAt)
	}
	b.rule()

	b.line(w.Checks, strconv.Itoa(d.Checks))
	if d.Guests > 0 {
		b.line(w.Guests, strconv.Itoa(d.Guests))
	}
	b.line(w.Sales, money(d.Sales, d.Currency))
	b.line(w.Cash, money(d.Cash, d.Currency))
	b.line(w.Card, money(d.Card, d.Currency))
	if d.Transfer > 0 {
		b.line(w.Transfer, money(d.Transfer, d.Currency))
	}
	// ⚠️ Under the payment lines but outside the sales total, because that is
	// what it is: food that left without money. Printed only when it happened —
	// a permanent zero here would read as one more way of paying.
	if d.Debt > 0 {
		b.line(w.Debt, money(d.Debt, d.Currency))
	}
	if d.Service > 0 {
		b.line(w.Service, money(d.Service, d.Currency))
	}
	if d.Discount > 0 {
		b.line(w.Discount, money(d.Discount, d.Currency))
	}
	// ⚠️ Refunds are printed even when zero is the answer people expect,
	// because the line's absence is indistinguishable from a shift where
	// nothing was handed back — and that is exactly the number somebody
	// reconciling a drawer is looking for.
	b.line(w.Refunded, money(d.Refunded, d.Currency))
	if d.Cancelled > 0 {
		b.line(w.Cancelled, strconv.Itoa(d.Cancelled))
	}
	b.rule()

	b.line(w.OpeningFloat, money(d.OpeningFloat, d.Currency))
	b.line(w.CounterCash, money(d.CounterCash, d.Currency))
	if d.DebtPaid > 0 {
		b.line(w.DebtOf, money(d.DebtPaid, d.Currency))
	}
	if d.Settlements > 0 {
		b.line(w.Settlements, money(d.Settlements, d.Currency))
	}
	if d.ManualIn > 0 {
		b.line(w.ManualIn, money(d.ManualIn, d.Currency))
	}
	if d.ManualOut > 0 {
		b.line(w.ManualOut, money(d.ManualOut, d.Currency))
	}
	b.line(w.Expected, money(d.Expected, d.Currency))

	// ⚠️ Only on a Z, and only once counted. An X printing "sanaldi: 0" says
	// the drawer was counted and found empty.
	if d.ClosedAt != "" {
		b.line(w.Counted, money(d.Counted, d.Currency))
		b.line(w.Variance, money(d.Variance, d.Currency))
		if d.VarianceNote != "" {
			b.wrap(d.VarianceNote)
		}
	}

	if t.Footer != "" {
		b.rule()
		b.center(t.Footer)
	}
	for range t.FeedLines {
		b.raw("")
	}
	return b.lines
}
