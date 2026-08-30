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

// ⚠️ **Splitting the bill is a convenience, and it has to stay off by
// default.** This line has never been printed by any receipt, so the "missing
// means shown" rule the `Fields` map follows would put a number nobody asked
// for onto every bill in the country the day it shipped. Hence its own boolean.
func TestTheBillIsOnlySplitWhenAskedFor(t *testing.T) {
	d := Data{Total: 100_000, Guests: 3, Currency: "UZS"}

	off := Render(Customer, Template{Enabled: true, WidthMM: 80}, d)
	if strings.Contains(strings.Join(off, "\n"), "kishiga") {
		t.Fatal("the bill was split without the setting being switched on")
	}

	on := Render(Customer,
		Template{Enabled: true, WidthMM: 80, SplitPerGuest: true}, d)
	joined := strings.Join(on, "\n")
	if !strings.Contains(joined, "3 kishiga") {
		t.Fatalf("the split line is missing:\n%s", joined)
	}
	// ⚠️ Rounded **up**: 100 000 ÷ 3 is 33 333.33, and three guests each
	// handing over 33 333 leave the restaurant a som short. The rounding goes
	// the way that covers the bill.
	if !strings.Contains(joined, "33 334") {
		t.Errorf("the share was not rounded up to cover the bill:\n%s", joined)
	}
}

// One guest, or none, is not a party. Splitting there prints the total twice.
func TestOneGuestIsNotSplit(t *testing.T) {
	tpl := Template{Enabled: true, WidthMM: 80, SplitPerGuest: true}
	for _, guests := range []int{0, 1} {
		out := Render(Customer, tpl,
			Data{Total: 100_000, Guests: guests, Currency: "UZS"})
		if strings.Contains(strings.Join(out, "\n"), "kishiga") {
			t.Errorf("a bill for %d guest(s) was split", guests)
		}
	}
}

// ⚠️ **The language is the restaurant's, not the screen's**, and the difference
// is who reads the paper. An X report has exactly one reader — whoever pressed
// the button — so it follows their interface. A receipt is read by a guest at a
// table who is signed in to nothing, and by a cook at a pass who is signed in
// to nothing either.
func TestAReceiptPrintsInTheLanguageTheRestaurantChose(t *testing.T) {
	d := Data{
		Number: "1", Table: "7", Total: 90000, Subtotal: 95000,
		Discount: 5000, Change: 10000, Method: "Naqd", Cashier: "Dilnoza",
		Currency: "so'm",
		Lines:    []Line{{Name: "Osh", Qty: 2, Price: 45000, Sum: 90000}},
	}
	tpl := Template{Enabled: true, WidthMM: 80, Fields: map[string]bool{
		"prices": true, "change": true, "cashier": true,
	}}

	uz := strings.Join(Render(Till, tpl, d), "\n")
	if !strings.Contains(uz, "JAMI") || !strings.Contains(uz, "Kassir") {
		t.Fatalf("the default is no longer Uzbek:\n%s", uz)
	}

	tpl.Lang = "ru"
	ru := strings.Join(Render(Till, tpl, d), "\n")
	if !strings.Contains(ru, "ИТОГО") || !strings.Contains(ru, "Кассир") {
		t.Fatalf("russian was asked for and not printed:\n%s", ru)
	}

	// ⚠️ An unknown value is Uzbek rather than a receipt of blank labels: this
	// is a print path, and the alternative to the wrong language is no paper.
	tpl.Lang = "kz"
	if !strings.Contains(strings.Join(Render(Till, tpl, d), "\n"), "JAMI") {
		t.Fatal("an unknown language did not fall back to Uzbek")
	}
}

// ⚠️ **The one line standing between a bill and a guest who thinks they hold a
// fiscal receipt.** It has to be as loud in every language as it is in Uzbek.
func TestTheBillSaysItIsNotAFiscalReceiptInEveryLanguage(t *testing.T) {
	d := Data{Number: "1", Table: "7", Total: 90000, Currency: "so'm",
		Lines: []Line{{Name: "Osh", Qty: 1, Price: 90000, Sum: 90000}}}
	for _, lang := range Langs {
		tpl := Template{Enabled: true, WidthMM: 80, Lang: lang,
			Fields: map[string]bool{"prices": true}}
		out := strings.Join(Render(Precheck, tpl, d), "\n")
		if !strings.Contains(out, wordsForReceipt(lang).PrecheckNote) {
			t.Fatalf("%s: the bill does not say what it is not:\n%s", lang, out)
		}
	}
}

// ⚠️ **A marker has no width on paper, so it must have none in the layout
// either.** Counting it shifts the text one column — on precisely the lines
// somebody chose to emphasise, which are the ones they will be looking at.
func TestEmphasisDoesNotMoveTheColumns(t *testing.T) {
	d := Data{Number: "1", Table: "7", Total: 90000, Currency: "so'm",
		Lines: []Line{{Name: "Osh", Qty: 1, Price: 90000, Sum: 90000}}}
	tpl := Template{Enabled: true, WidthMM: 80,
		Fields: map[string]bool{"prices": true}}

	plainTotal := totalLine(t, Render(Till, tpl, d))

	tpl.Emphasis = map[string]string{"total": "boldbig"}
	markedTotal := totalLine(t, Render(Till, tpl, d))

	mark, stripped := splitMark(markedTotal)
	if mark == "" {
		t.Fatal("the total was not emphasised")
	}
	if stripped != plainTotal {
		t.Fatalf("the emphasis moved the line:\n plain: %q\nmarked: %q",
			plainTotal, stripped)
	}
}

func totalLine(t *testing.T, lines []string) string {
	t.Helper()
	for _, l := range lines {
		if _, s := splitMark(l); strings.HasPrefix(strings.TrimSpace(s), "JAMI") {
			return l
		}
	}
	t.Fatal("no total line")
	return ""
}

// ⚠️ Every receipt printed before this setting existed must be byte-identical.
func TestNoEmphasisIsTheOldReceipt(t *testing.T) {
	d := Data{Number: "1", Table: "7", Total: 90000, Currency: "so'm",
		Lines: []Line{{Name: "Osh", Qty: 1, Price: 90000, Sum: 90000}}}
	tpl := Template{Enabled: true, WidthMM: 80,
		Fields: map[string]bool{"prices": true}}
	for _, l := range Render(Till, tpl, d) {
		if m, _ := splitMark(l); m != "" {
			t.Fatalf("an unasked-for marker appeared on %q", l)
		}
	}
}

// ⚠️ **The header is documented as the place for an address and a phone
// number, and it was drawn with `center`, which cuts at the paper's width.** A
// restaurant that typed both got the first half of the first one — and the
// panel offered a single-line box, so newlines were not reachable either.
func TestTheHeaderKeepsEveryLineItWasGiven(t *testing.T) {
	d := Data{Number: "1", Total: 1000, Currency: "so'm",
		Lines: []Line{{Name: "Osh", Qty: 1, Price: 1000, Sum: 1000}}}
	tpl := Template{
		Enabled: true, WidthMM: 80,
		Header: "Amir Temur ko'chasi 108-uy, Toshkent\n+998 90 123 45 67",
		Footer: "Rahmat!\nYana kuting",
	}
	out := strings.Join(Render(Customer, tpl, d), "\n")

	for _, want := range []string{"108-uy", "+998 90 123 45 67", "Rahmat!", "Yana kuting"} {
		if !strings.Contains(out, want) {
			t.Fatalf("%q was cut off the receipt:\n%s", want, out)
		}
	}
}

// ⚠️ A line wider than the paper becomes two centred lines rather than half a
// line: this is text somebody designed, and cutting it is the failure it is
// most likely to be reported as.
func TestALongHeaderLineWrapsRatherThanBeingCut(t *testing.T) {
	long := "Restoran nomi juda uzun bo'lsa ham har bir so'zi qog'ozga sig'ishi kerak"
	tpl := Template{Enabled: true, WidthMM: 58, Header: long}
	out := Render(Customer, tpl, Data{Number: "1", Currency: "so'm"})

	joined := strings.Join(out, " ")
	for _, word := range strings.Fields(long) {
		if !strings.Contains(joined, word) {
			t.Fatalf("%q was lost:\n%s", word, strings.Join(out, "\n"))
		}
	}
}

// ⚠️ Bounded, because it is a number typed into a box and a hundred blank lines
// is a roll of paper on the floor.
func TestTheTopMarginIsBounded(t *testing.T) {
	tpl := Template{Enabled: true, WidthMM: 80, TopLines: 500}
	out := Render(Customer, tpl, Data{Number: "1", Currency: "so'm"})
	blank := 0
	for _, l := range out {
		if strings.TrimSpace(l) != "" {
			break
		}
		blank++
	}
	if blank > 6 {
		t.Fatalf("%d blank lines at the top", blank)
	}
}

// ⚠️ **The most repeated word on the paper was in the wrong language.** A
// Russian receipt had Russian headings and "so'm" at the end of every priced
// line and under the total — which reads as a half-finished translation, and
// was one.
func TestTheCurrencyFollowsTheReceiptLanguage(t *testing.T) {
	d := Data{Number: "1", Total: 92000,
		Lines: []Line{{Name: "Osh", Qty: 1, Price: 92000, Sum: 92000}}}

	ru := strings.Join(Render(Customer, Template{Enabled: true, WidthMM: 80, Lang: "ru"}, d), "\n")
	if !strings.Contains(ru, "сум") {
		t.Fatalf("a Russian receipt priced in Uzbek:\n%s", ru)
	}
	if strings.Contains(ru, "so'm") {
		t.Fatalf("the Uzbek word survived on a Russian receipt:\n%s", ru)
	}

	// ⚠️ And a restaurant that named its own currency keeps it in every
	// language: pricing in dollars is a fact about the prices, not about who is
	// reading them.
	d.Currency = "USD"
	out := strings.Join(Render(Customer, Template{Enabled: true, WidthMM: 80, Lang: "ru"}, d), "\n")
	if !strings.Contains(out, "USD") {
		t.Fatalf("the restaurant's own currency was overwritten:\n%s", out)
	}
}

// ⚠️ **The guest's copy printed "Ofitsiant" on a Russian receipt.** The bill
// renderer had used the translated word for a while; the receipt — the paper
// the guest keeps — still had the literal, under a header that was correct and
// beside a total that was correct, which is why nobody spotted it.
func TestTheReceiptPrintsEveryLabelInItsOwnLanguage(t *testing.T) {
	d := Data{
		Number: "A-1", Table: "стол 7", Server: "Дилноза",
		Lines:    []Line{{Name: "Лагман", Qty: 1, Price: 45000, Sum: 45000}},
		Subtotal: 45000, Discount: 2000, DiscountName: "Kassa chegirmasi: menejer",
		Service: 3000, ServicePercent: 10,
		Total: 46000, Currency: "so'm",
	}
	out := strings.Join(Render(Customer, Template{Enabled: true, Lang: "ru"}, d), "\n")
	for _, uzbek := range []string{"Ofitsiant", "Xizmat haqi", "Kassa chegirmasi"} {
		if strings.Contains(out, uzbek) {
			t.Errorf("a Russian receipt still says %q:\n%s", uzbek, out)
		}
	}
	for _, russian := range []string{"Официант", "Сервисный сбор", "Скидка кассы"} {
		if !strings.Contains(out, russian) {
			t.Errorf("a Russian receipt does not say %q:\n%s", russian, out)
		}
	}
	// ⚠️ The cashier's own reason survives untranslated: those are their words
	// about this sale, not a label this code chose.
	if !strings.Contains(out, "menejer") {
		t.Errorf("the cashier's reason was rewritten:\n%s", out)
	}
}

// A promotion the restaurant named itself is never touched.
func TestARestaurantsOwnDiscountNameIsLeftAlone(t *testing.T) {
	d := Data{
		Number: "A-1", Lines: []Line{{Name: "Osh", Qty: 1, Price: 45000, Sum: 45000}},
		Subtotal: 45000, Discount: 5000, DiscountName: "Tug'ilgan kun 10%",
		Total: 40000, Currency: "so'm",
	}
	out := strings.Join(Render(Customer, Template{Enabled: true, Lang: "ru"}, d), "\n")
	if !strings.Contains(out, "Tug'ilgan kun 10%") {
		t.Errorf("the restaurant's own discount name was translated:\n%s", out)
	}
}
