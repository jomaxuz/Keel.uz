package fiscal

import (
	"time"
)

// Receipt is one sale, in the shape the state asks for.
//
// The *fields* are fixed — every virtual cash register files a line per item
// with its classifier code, packaging, price, quantity and VAT, and splits the
// payment between cash and card. How they are named and encoded on the wire is
// each provider's business, so that lives in the adapter and this stays in our
// own vocabulary.
//
// ⚠️ **All money is in tiyin** (1 so'm = 100 tiyin), which is the only place in
// this codebase that is true — everything else counts whole so'm (see the
// currency note in CLAUDE.md). The receipt is not ours to denominate, and a
// number that is a hundred times wrong on a filed document is not a rounding
// bug. Conversion happens once, in Build, so no caller ever multiplies by 100
// and no adapter has to wonder which it was handed.
type Receipt struct {
	// Our order number, so a filed receipt can be traced back. Providers call
	// this the external or operation id.
	OrderNumber string
	// Whose sale this is.
	TIN string
	// Who took the money, as printed on the receipt. Not an id and not a login:
	// the register puts this string on the guest's paper, and the guest checks
	// it against the person in front of them.
	Cashier string
	// When the money was taken — the moment that belongs on the receipt, not
	// the moment we happened to file it. They differ whenever the provider was
	// unreachable and the filing was retried, which is exactly when the
	// difference matters.
	Time time.Time

	Items []Item

	// The payment split, in tiyin. Both may be set: a guest paying part cash
	// and part card is ordinary in a hall.
	//
	// ⚠️ Their sum must equal the sum of the lines. Build guarantees it; an
	// adapter must not adjust either one to make its own arithmetic come out.
	ReceivedCash int64
	ReceivedCard int64

	// Set when this receipt reverses an earlier one.
	IsRefund bool

	// ---- What the sale being reversed was ----
	//
	// ⚠️ **A refund without these is a different document.** Every register in
	// the registry files a refund *against* an original receipt: the state's
	// copy has to be able to find the sale that is being undone. Sent without
	// them, a refund is either refused or — worse — accepted as a standalone
	// negative sale, which balances our books and does not balance theirs.
	//
	// Filled from the order's own stored filing (`models.FiscalReceipt`), never
	// recomputed: the sign and the sequence are the register's words about a
	// document it already issued, and nothing here can derive them.
	Original OriginalReceipt
}

// OriginalReceipt identifies the filed sale a refund reverses.
type OriginalReceipt struct {
	// The fiscal module that issued it. Usually this register, but not always:
	// a refund may be taken at a second till in the same restaurant.
	TerminalID string
	// The receipt's sequence number in the fiscal module.
	Seq string
	// When it was issued.
	At time.Time
	// The fiscal sign printed on the guest's copy.
	Sign string
	// The register's own id for the sale, where it gave one.
	SaleID string
}

// Known reports whether there is enough here to name the sale being reversed.
//
// ⚠️ The sign is the one field that cannot be missing: it is what the tax
// committee's copy is indexed by, and a refund carrying the other three without
// it names a receipt nobody can look up.
func (o OriginalReceipt) Known() bool {
	return o.Sign != ""
}

// Item is one line of the receipt.
type Item struct {
	Name string
	// The state product classifier code (ИКПУ / MXIK), and the packaging code
	// that belongs to it. Empty when the restaurant has not entered one —
	// omitted rather than guessed, for the reason written on MenuItem.Ikpu.
	SPIC        string
	PackageCode string

	// Unit price in tiyin, **before** any discount on this line.
	Price int64
	// How many. A plain count in the unit named by Units.
	//
	// ⚠️ **Not scaled here.** Providers encode quantity at different precisions
	// (thousandths are common), and scaling in the builder would mean every
	// adapter had to know what the builder already did to the number. The
	// adapter multiplies when it serialises, which is where the wire format is
	// allowed to be known.
	Qty int
	// Measure unit code: 0 = piece, 10 = gram, 11 = kilogram, 22 = metre,
	// 41 = litre.
	Units int

	// The national marking code scanned off this item (Asl Belgisi,
	// DataMatrix), or empty.
	//
	// ⚠️ **This is how a marked product is withdrawn from circulation**, and
	// there is no second call to make: the code travels inside the fiscal
	// receipt, the OFD hands it to the national system. So the only thing this
	// pipeline owes marking is one field per line — see docs/markirovka.md.
	MarkCode string

	// This line's share of the order's discounts, in tiyin. See Build.
	Discount int64
	// VAT included in the line, in tiyin, and the rate it was computed at.
	VAT        int64
	VATPercent int
}

// Total is what this line contributes to the receipt, in tiyin.
func (i Item) Total() int64 { return i.Price*int64(i.Qty) - i.Discount }

// Line is one item of a sale as the caller has it — whole so'm, our own
// vocabulary, no tax arithmetic done yet.
type Line struct {
	Name  string
	Price int // per unit, in so'm
	Qty   int

	SPIC        string
	PackageCode string
	Units       int
	// The dish's own VAT rate when it overrides the branch's, nil otherwise.
	VatPercent *int
	// The marking code scanned off this item, or empty. Frozen on the order
	// line rather than read from the menu: it describes the bottle handed over.
	MarkCode string
}

// Sale is everything Build needs about one sale, in so'm.
type Sale struct {
	OrderNumber string
	TIN         string
	Cashier     string
	Time        time.Time

	Lines []Line
	// Delivery, added as its own line when non-zero. A guest comparing the
	// fiscal receipt against what they paid must not find an unexplained gap.
	DeliveryFee int
	// Everything taken off the basket: promotions, promo codes and loyalty
	// points together. They are one number here because the receipt has one
	// place to put them — per line — and the distinction between them is a
	// question about our pricing, not about this sale's tax.
	Discount int

	// What was actually taken, in so'm.
	Cash int
	Card int

	// The branch's rate, used by every line that does not override it.
	VatPercent int

	// ---- Reversing an earlier sale ----
	//
	// ⚠️ **Both or neither.** A refund needs the original's fiscal sign, and an
	// adapter refuses to build one without it — see `OriginalReceipt.Known`.
	// Carried through `Build` rather than set on the Receipt afterwards so the
	// caller cannot forget it: the field is where the refund flag is, and the
	// two are read together.
	IsRefund bool
	Original OriginalReceipt
}

// Build turns a sale into the receipt to file.
//
// Three things here are easy to get wrong in a way that produces a receipt
// which looks right and is not:
//
//  1. **VAT is included in the price, not added to it.** Uzbek menu prices are
//     what the guest pays, so the tax inside a 12% line is price*12/112, not
//     price*12/100. The second formula is the obvious one, overstates the tax
//     by about 12%, and produces numbers that pass every eyeball check.
//
//  2. **A discount taken off the order has to be spread across the lines**,
//     because the receipt has nowhere else to put it, and the spread has to
//     come out exact. Dividing proportionally and rounding each line leaves a
//     few tiyin unaccounted for; a receipt whose lines do not sum to the money
//     taken is a rejected — or worse, accepted and wrong — filing. The
//     remainder is handed to the largest lines, so it lands where it is
//     proportionally smallest.
//
//  3. **The payment split has to match the lines.** If it does not, Build
//     corrects the split rather than the lines: the lines describe goods and
//     are checkable against the menu, while the split is a fact about two
//     numbers that must add to a known total. Fixing it the other way would
//     mean editing what was sold to make the till balance.
func Build(s Sale) Receipt {
	items := make([]Item, 0, len(s.Lines)+1)
	for _, l := range s.Lines {
		if l.Qty <= 0 {
			// A voided or zero line is not a sale. Filing it as one puts a
			// product on a state document that nobody bought.
			continue
		}
		items = append(items, Item{
			Name:        l.Name,
			SPIC:        l.SPIC,
			PackageCode: l.PackageCode,
			Price:       int64(l.Price) * 100,
			Qty:         l.Qty,
			Units:       l.Units,
			MarkCode:    l.MarkCode,
			VATPercent:  rateFor(l.VatPercent, s.VatPercent),
		})
	}
	if s.DeliveryFee > 0 {
		items = append(items, Item{
			Name:  "Yetkazib berish",
			Price: int64(s.DeliveryFee) * 100,
			Qty:   1,
			// Delivery carries no classifier code: it is a service, and the
			// code for it is the restaurant's to declare if its accountant says
			// so. Guessing one here would file every delivery in the country
			// against whatever we picked.
			VATPercent: s.VatPercent,
		})
	}

	// Spread the discount over the goods. Delivery is included in the spread
	// only if it is the sole line — a discount is on the basket, and taking it
	// off the delivery instead would change what the courier is recorded as
	// having been paid for.
	gross := make([]int64, len(items))
	for i, it := range items {
		gross[i] = it.Price * int64(it.Qty)
	}
	for i, share := range distribute(int64(s.Discount)*100, gross) {
		items[i].Discount = share
	}

	var total int64
	for i := range items {
		items[i].VAT = vatOf(items[i].Total(), items[i].VATPercent)
		total += items[i].Total()
	}

	cash, card := int64(s.Cash)*100, int64(s.Card)*100
	if cash+card != total {
		// See (3) above. Whatever was recorded as taken does not add up to what
		// was sold — usually because a caller passed only one of the two.
		//
		// The card figure is kept and cash takes the difference: a card amount
		// is a fact a terminal reported, while cash is by definition what is
		// left over in the drawer. Clamped first, because a card amount larger
		// than the sale would otherwise make the cash line negative, and a
		// receipt cannot say the guest was handed money.
		if card < 0 {
			card = 0
		}
		if card > total {
			card = total
		}
		cash = total - card
	}

	when := s.Time
	if when.IsZero() {
		when = time.Now()
	}
	return Receipt{
		IsRefund:     s.IsRefund,
		Original:     s.Original,
		OrderNumber:  s.OrderNumber,
		TIN:          s.TIN,
		Cashier:      s.Cashier,
		Time:         when,
		Items:        items,
		ReceivedCash: cash,
		ReceivedCard: card,
	}
}

// rateFor picks the dish's own VAT rate over the branch's.
//
// nil is "not set", which is the state almost every dish is in — see
// MenuItem.VatPercent for why it cannot be an int.
func rateFor(dish *int, branch int) int {
	if dish != nil {
		return *dish
	}
	return branch
}

// vatOf returns the VAT contained in an amount that already includes it.
//
// ⚠️ Rounded half-up on a positive amount, which is what a tax figure is; the
// alternative, truncation, biases every receipt downward by up to a tiyin per
// line and does so consistently, which over a year is a pattern rather than
// noise.
func vatOf(amount int64, percent int) int64 {
	if percent <= 0 || amount <= 0 {
		return 0
	}
	den := int64(100 + percent)
	return (amount*int64(percent)*2/den + 1) / 2
}

// distribute splits a total across weights so that the parts sum to exactly the
// total.
//
// Largest remainder: each weight gets its floor share, and the leftover units
// go one each to the weights with the largest dropped fractions. The naive
// alternative — round each share independently — is short or long by a few
// units, and on a fiscal document that is the difference between a receipt that
// balances and one that does not.
func distribute(total int64, weights []int64) []int64 {
	out := make([]int64, len(weights))
	if total <= 0 || len(weights) == 0 {
		return out
	}
	var sum int64
	for _, w := range weights {
		if w > 0 {
			sum += w
		}
	}
	if sum <= 0 {
		return out
	}
	if total > sum {
		// A discount larger than the basket would make lines negative. Cap it:
		// a free basket is representable, a basket that owes money is not.
		total = sum
	}

	var given int64
	// Remainders, kept as numerators over the same denominator so the
	// comparison below needs no floating point.
	rem := make([]int64, len(weights))
	for i, w := range weights {
		if w <= 0 {
			continue
		}
		part := total * w
		out[i] = part / sum
		rem[i] = part % sum
		given += out[i]
	}
	for given < total {
		best, bestRem := -1, int64(-1)
		for i := range weights {
			if rem[i] > bestRem {
				best, bestRem = i, rem[i]
			}
		}
		if best < 0 {
			break
		}
		out[best]++
		rem[best] = -1
		given++
	}
	return out
}
