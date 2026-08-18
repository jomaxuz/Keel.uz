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
	// Checks voided before payment. Shown because a shift with twenty of them
	// is a story, and no other line on this paper would carry it.
	Cancelled int

	// The drawer.
	OpeningFloat int
	CounterCash  int
	Settlements  int
	ManualIn     int
	ManualOut    int
	Expected     int
	// Zero until the drawer has been counted, which only happens on a Z.
	Counted      int
	Variance     int
	VarianceNote string

	Currency string
}

// RenderShift lays out an X or Z report.
func RenderShift(t Template, d ShiftData) []string {
	w := WidthFor(t.WidthMM)
	b := &block{w: w}

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
	b.center(d.Kind + "-HISOBOT")
	b.rule()

	b.line("Ochilgan", d.OpenedAt)
	if d.OpenedBy != "" {
		b.line("Ochdi", d.OpenedBy)
	}
	if d.ClosedAt != "" {
		b.line("Yopilgan", d.ClosedAt)
		if d.ClosedBy != "" {
			b.line("Yopdi", d.ClosedBy)
		}
	}
	if d.PrintedAt != "" {
		b.line("Chop etildi", d.PrintedAt)
	}
	b.rule()

	b.line("Cheklar", strconv.Itoa(d.Checks))
	if d.Guests > 0 {
		b.line("Mehmonlar", strconv.Itoa(d.Guests))
	}
	b.line("Sotuv", money(d.Sales, d.Currency))
	b.line("  Naqd", money(d.Cash, d.Currency))
	b.line("  Karta", money(d.Card, d.Currency))
	if d.Transfer > 0 {
		b.line("  O'tkazma", money(d.Transfer, d.Currency))
	}
	if d.Service > 0 {
		b.line("Xizmat haqi", money(d.Service, d.Currency))
	}
	if d.Discount > 0 {
		b.line("Chegirma", money(d.Discount, d.Currency))
	}
	// ⚠️ Refunds are printed even when zero is the answer people expect,
	// because the line's absence is indistinguishable from a shift where
	// nothing was handed back — and that is exactly the number somebody
	// reconciling a drawer is looking for.
	b.line("Qaytarilgan", money(d.Refunded, d.Currency))
	if d.Cancelled > 0 {
		b.line("Bekor qilingan", strconv.Itoa(d.Cancelled))
	}
	b.rule()

	b.line("Kassa qoldig'i", money(d.OpeningFloat, d.Currency))
	b.line("Naqd sotuv", money(d.CounterCash, d.Currency))
	if d.Settlements > 0 {
		b.line("Kuryerlardan", money(d.Settlements, d.Currency))
	}
	if d.ManualIn > 0 {
		b.line("Kirim", money(d.ManualIn, d.Currency))
	}
	if d.ManualOut > 0 {
		b.line("Chiqim", money(d.ManualOut, d.Currency))
	}
	b.line("Kassada bo'lishi kerak", money(d.Expected, d.Currency))

	// ⚠️ Only on a Z, and only once counted. An X printing "sanaldi: 0" says
	// the drawer was counted and found empty.
	if d.ClosedAt != "" {
		b.line("Sanaldi", money(d.Counted, d.Currency))
		b.line("Farq", money(d.Variance, d.Currency))
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
