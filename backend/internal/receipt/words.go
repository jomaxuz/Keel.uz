package receipt

// The words on a guest's receipt, in the language the restaurant chose.
//
// ⚠️ **The restaurant's choice, not the screen's, and that is the whole
// difference from `shiftWords`.** An X or Z report is read by exactly one
// person — whoever pressed the button — so it follows their interface. A
// receipt is read by a guest at a table and by a cook at a pass, and neither of
// them is signed in to anything: printing in whatever language the cashier
// happens to have selected would hand an Uzbek room receipts in Russian
// because one person on that shift prefers it.
//
// So it is a setting, per receipt kind. ⚠️ **Per kind, because a kitchen is not
// a dining room**: a restaurant with Russian-speaking cooks and Uzbek guests is
// ordinary in Tashkent, and one language for both would be wrong for one of
// them every time.
//
// ⚠️ **A struct, not a map.** A missing key in a map is a blank label beside a
// number on a document somebody is paying against — worse than the wrong
// language. With a struct the compiler names the gap when a line is added.
type words struct {
	Subtotal, Discount, Total string
	Change, Cashier, Server   string
	Guests, PerGuest          string
	FiscalSign, PrecheckNote  string
	// The service charge line. ⚠️ Missed the same way `Currency` was: it is
	// printed by `totals`, which is shared by all three papers, so it went out
	// in Uzbek on every Russian receipt — under a heading that was correct.
	Service string
	// The prefix the till writes on a discount the cashier keyed in
	// (`Kassa chegirmasi: <sabab>`). ⚠️ The reason itself is the cashier's own
	// words and is never touched; only our label in front of it is.
	TillDiscount string
	// What the guest sees where the table number goes on a preview.
	Table string
	// ⚠️ **The currency is a word too, and it was the one that got missed.**
	// "so'm" is Uzbek. On a Russian receipt it sat at the end of every priced
	// line and under the total — the most repeated word on the paper, in the
	// wrong language, on a receipt whose headings were all correct.
	Currency string
}

// wordsForReceipt picks the language.
//
// ⚠️ **An unknown or empty value is Uzbek**, which is what every receipt
// printed before this setting existed — so a restaurant that never opens the
// dropdown sees no change at all.
func wordsForReceipt(lang string) words {
	switch lang {
	case "ru":
		return words{
			Subtotal: "Подытог", Discount: "Скидка", Total: "ИТОГО",
			Change: "Сдача", Cashier: "Кассир", Server: "Официант",
			Guests: "Гостей", PerGuest: "на гостя (примерно)",
			FiscalSign: "Фискальный признак",
			// ⚠️ Kept as loud as the Uzbek one. This line is the only thing
			// standing between a bill and a guest who believes they have a
			// fiscal receipt.
			PrecheckNote: "СЧЁТ — не фискальный чек",
			Currency:     "сум",
			Service:      "Сервисный сбор",
			TillDiscount: "Скидка кассы",
			Table:        "стол",
		}
	case "en":
		return words{
			Subtotal: "Subtotal", Discount: "Discount", Total: "TOTAL",
			Change: "Change", Cashier: "Cashier", Server: "Server",
			Guests: "Guests", PerGuest: "per guest (approx.)",
			FiscalSign:   "Fiscal sign",
			PrecheckNote: "BILL — not a fiscal receipt",
			Currency:     "so'm",
			Service:      "Service charge",
			TillDiscount: "Till discount",
			Table:        "table",
		}
	}
	return words{
		Subtotal: "Oraliq jami", Discount: "Chegirma", Total: "JAMI",
		Change: "Qaytim", Cashier: "Kassir", Server: "Ofitsiant",
		Guests: "Mehmonlar", PerGuest: "kishiga (taxminan)",
		FiscalSign:   "Fiskal belgi",
		PrecheckNote: PrecheckNote,
		Currency:     "so'm",
		Service:      "Xizmat haqi",
		TillDiscount: "Kassa chegirmasi",
		Table:        "stol",
	}
}

// Langs is what the panel offers. ⚠️ The same three the rest of the product
// speaks; a fourth here would be a language the menu itself cannot be written
// in.
var Langs = []string{"uz", "ru", "en"}
