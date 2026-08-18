package receipt

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func sample() Data {
	return Data{
		Title: "Osh Markazi", Address: "Chilonzor 12", Phone: "+998 90 123 45 67",
		Number: "MRC-A1-1745", Table: "7-stol", ClosedAt: "17.08 19:42",
		Server: "Aziz", Cashier: "Dilnoza", Currency: "so'm",
		Lines: []Line{
			{Name: "Lag'mon", Qty: 2, Price: 45000, Sum: 90000, Comment: "piyozsiz"},
			{Name: "Choy", Qty: 1, Price: 5000, Sum: 5000},
		},
		Subtotal: 95000, Total: 95000, Paid: 100000, Change: 5000,
		Method: "Naqd", FiscalSign: "002519286194",
	}
}

func widthOf(lines []string) int {
	w := 0
	for _, l := range lines {
		if n := utf8.RuneCountInString(l); n > w {
			w = n
		}
	}
	return w
}

// ⚠️ **The 58 mm test.** Sending 48 characters to a 58 mm printer cuts the end
// off every line — which on the customer's copy is the totals column, and the
// only person who finds out is the guest holding the receipt.
func TestNothingExceedsThePaper(t *testing.T) {
	for _, mm := range []int{58, 80} {
		tpl := Template{WidthMM: mm, Header: "Xush kelibsiz", Footer: "Rahmat!"}
		for _, kind := range []Kind{Kitchen, Till, Customer} {
			lines := Render(kind, tpl, sample())
			if got := widthOf(lines); got > WidthFor(mm) {
				t.Fatalf("%s at %dmm: a line is %d chars, paper takes %d",
					kind, mm, got, WidthFor(mm))
			}
		}
	}
}

// ⚠️ **Runes, not bytes.** Uzbek written properly carries `oʻ` and `gʻ`, and
// Russian is Cyrillic — both multi-byte. A width computed with len() makes
// every line containing them too short by exactly the number of letters, and
// the receipt's columns drift further right the further down you read.
func TestCyrillicAndApostrophesDoNotShrinkTheLine(t *testing.T) {
	d := sample()
	d.Title = "Ресторан «Осиё»"
	d.Lines = []Line{
		{Name: "Гўшт сомса", Qty: 3, Price: 12000, Sum: 36000},
		{Name: "Qaymoqli choyxona lag'moni", Qty: 1, Price: 48000, Sum: 48000},
	}
	tpl := Template{WidthMM: 58}
	lines := Render(Customer, tpl, d)
	if got := widthOf(lines); got > Width58 {
		t.Fatalf("a Cyrillic line came out %d chars wide, paper takes %d", got, Width58)
	}
	// And the money still landed on the right-hand edge rather than being
	// pushed off it.
	if !hasLineEnding(lines, "48 000 so'm") {
		t.Fatalf("the line total was lost:\n%s", strings.Join(lines, "\n"))
	}
}

func hasLineEnding(lines []string, suffix string) bool {
	for _, l := range lines {
		if strings.HasSuffix(l, suffix) {
			return true
		}
	}
	return false
}

// ⚠️ **The kitchen ticket carries no prices, and no setting can add them.** A
// price tells a cook nothing and lengthens a ticket read under time pressure.
func TestKitchenTicketHasNoMoneyOnIt(t *testing.T) {
	// Every field switched on, to prove none of them is a price.
	tpl := Template{WidthMM: 80, Fields: map[string]bool{
		"comment": true, "time": true, "server": true,
		"cashier": true, "change": true, "address": true, "phone": true,
	}}
	out := strings.Join(Render(Kitchen, tpl, sample()), "\n")
	for _, forbidden := range []string{"45 000", "95 000", "JAMI", "Naqd", "Qaytim"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("the kitchen ticket carries %q:\n%s", forbidden, out)
		}
	}
	// What it must carry: the table, and the comment.
	if !strings.Contains(out, "7-STOL") {
		t.Fatalf("the kitchen ticket does not name the table:\n%s", out)
	}
	if !strings.Contains(out, "piyozsiz") {
		t.Fatalf("the kitchen ticket dropped the dish comment:\n%s", out)
	}
}

// ⚠️ Only the guest's copy carries the fiscal sign — it is the only one anybody
// will check, and printing it at the pass is paper spent on somebody who cannot
// use it.
func TestOnlyTheCustomerCopyCarriesTheFiscalSign(t *testing.T) {
	tpl := Template{WidthMM: 80}
	d := sample()
	if !strings.Contains(strings.Join(Render(Customer, tpl, d), "\n"), d.FiscalSign) {
		t.Fatal("the customer copy lost the fiscal sign")
	}
	for _, kind := range []Kind{Kitchen, Till} {
		if strings.Contains(strings.Join(Render(kind, tpl, d), "\n"), d.FiscalSign) {
			t.Fatalf("%s carries the fiscal sign", kind)
		}
	}
}

// ⚠️ **A missing field means shown.** Every template stored before a field
// existed has no entry for it, and reading that as "off" would silently drop a
// line from receipts that were printing it fine.
func TestUnknownFieldDefaultsToShown(t *testing.T) {
	if !(Template{}).Shows("comment") {
		t.Fatal("a template with no field map hides everything")
	}
	if !(Template{Fields: map[string]bool{"time": false}}).Shows("comment") {
		t.Fatal("switching one field off hid an unrelated one")
	}
	if (Template{Fields: map[string]bool{"time": false}}).Shows("time") {
		t.Fatal("switching a field off did nothing")
	}
}

// ⚠️ A long dish name is **wrapped, not cut**: "Lag'mon (achchiq, katta
// porsiya)" truncated at 32 characters is a different dish, and the kitchen
// cooks what the ticket says.
func TestLongNamesWrapRatherThanTruncate(t *testing.T) {
	d := sample()
	long := "Qaymoqli achchiq lag'mon katta porsiya qo'shimcha go'sht bilan"
	d.Lines = []Line{{Name: long, Qty: 1, Price: 60000, Sum: 60000}}
	out := strings.Join(Render(Kitchen, Template{WidthMM: 58}, d), "\n")
	for _, word := range strings.Fields(long) {
		if !strings.Contains(out, word) {
			t.Fatalf("the word %q was lost from the ticket:\n%s", word, out)
		}
	}
}

// ⚠️ The right-hand column wins a collision, because it is the money: a label
// that eats into a total produces a wrong number rather than a short word.
func TestTheAmountIsNeverCut(t *testing.T) {
	b := &block{w: 20}
	b.line("Juda uzun izohli qator nomi", "1 234 567")
	got := b.lines[0]
	if !strings.HasSuffix(got, "1 234 567") {
		t.Fatalf("the amount was cut: %q", got)
	}
	if utf8.RuneCountInString(got) != 20 {
		t.Fatalf("line is %d chars, want 20: %q", utf8.RuneCountInString(got), got)
	}
}

// ⚠️ Grouped by hand, never by locale: the same restaurant must not print
// commas on one receipt and spaces on another because the panel happened to be
// in a different language when the template was saved.
func TestMoneyIsGroupedTheSameWayEveryTime(t *testing.T) {
	cases := map[int]string{
		0: "0", 999: "999", 1000: "1 000", 95000: "95 000", 1234567: "1 234 567",
	}
	for n, want := range cases {
		if got := money(n, ""); got != want {
			t.Fatalf("money(%d) = %q, want %q", n, got, want)
		}
	}
	if got := money(95000, "so'm"); got != "95 000 so'm" {
		t.Fatalf("currency lost: %q", got)
	}
}

// The bill handed to a table before it pays.
//
// ⚠️ **The danger is that it looks like the receipt.** Same dishes, same total,
// same paper — and a guest handed a document that reads like a fiscal receipt
// has been told the sale is registered when it is not. These two rules are what
// keep the two apart on paper.
func TestPrecheckCannotBeMistakenForTheReceipt(t *testing.T) {
	d := sample()
	lines := Render(Precheck, Template{WidthMM: 80}, d)

	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, PrecheckNote) {
		t.Fatalf("the bill does not say what it is:\n%s", joined)
	}
	// ⚠️ Even when the order already carries one — a check can be re-billed
	// after a failed payment, and the sign belongs to the sale, not the paper.
	if strings.Contains(joined, d.FiscalSign) {
		t.Fatalf("a fiscal sign was printed on a bill:\n%s", joined)
	}
	// Nothing has been paid, so there is no change to give.
	if strings.Contains(joined, "Qaytim") {
		t.Fatalf("change was printed on an unpaid bill:\n%s", joined)
	}
	// It is still a bill: the dishes and the total have to be on it.
	if !strings.Contains(joined, d.Lines[0].Name[:6]) {
		t.Fatalf("the dishes are missing:\n%s", joined)
	}
}

// The paper limit holds for the new kind too — the reason this rule is a test
// rather than a habit.
func TestPrecheckFitsTheNarrowPaper(t *testing.T) {
	lines := Render(Precheck, Template{WidthMM: 58}, sample())
	if w := widthOf(lines); w > Width58 {
		t.Fatalf("a line is %d characters on 32-character paper", w)
	}
}
