package receipt

// The shift report's words, in the language it is being printed in.
//
// ⚠️ **Only this report, and that is a decision rather than a stopping point.**
// A guest's receipt is read by a guest, and it is printed in the restaurant's
// own language whatever the panel happens to be set to — a Russian-speaking
// cashier serving an Uzbek room must not hand out receipts nobody at the table
// reads. The X and Z are the opposite: they are read by exactly one person, the
// one who pressed the button, and printing them in a language that person does
// not read makes the document useless to its only reader. So the language
// follows the screen here and nowhere else.
//
// ⚠️ **A struct, not a map of strings.** A missing key in a map is an empty
// label on a printed financial document — a blank line beside a number, which
// is worse than the wrong language. With a struct the compiler names the gap
// when a line is added.
type shiftWords struct {
	XTitle, ZTitle                     string
	OpenedAt, OpenedBy                 string
	ClosedAt, ClosedBy, PrintedAt      string
	Checks, Guests                     string
	Sales, Cash, Card, Transfer        string
	Debt, Service, Discount            string
	Refunded, Cancelled                string
	OpeningFloat, CounterCash, DebtOf  string
	Settlements, ManualIn, ManualOut   string
	Expected, Counted, Variance        string
}

// wordsFor picks the language.
//
// ⚠️ **An unknown value falls back to Uzbek rather than failing.** This is a
// print path: the alternative to a report in the wrong language is no report,
// and the person who pressed the button is standing at a drawer they cannot
// close until they have one.
func wordsFor(lang string) shiftWords {
	switch lang {
	case "ru":
		return shiftWords{
			XTitle: "X-ОТЧЁТ", ZTitle: "Z-ОТЧЁТ",
			OpenedAt: "Открыта", OpenedBy: "Открыл",
			ClosedAt: "Закрыта", ClosedBy: "Закрыл", PrintedAt: "Напечатано",
			Checks: "Чеки", Guests: "Гости",
			Sales: "Продажи", Cash: "  Наличные", Card: "  Карта",
			Transfer: "  Перевод",
			Debt:     "В долг", Service: "Сервисный сбор", Discount: "Скидка",
			Refunded: "Возвращено", Cancelled: "Отменено",
			OpeningFloat: "Остаток на начало", CounterCash: "Наличные продажи",
			DebtOf:      "  из них долг возвращён",
			Settlements: "От курьеров", ManualIn: "Приход", ManualOut: "Расход",
			Expected: "Должно быть в кассе", Counted: "Посчитано",
			Variance: "Разница",
		}
	case "en":
		return shiftWords{
			XTitle: "X-REPORT", ZTitle: "Z-REPORT",
			OpenedAt: "Opened", OpenedBy: "Opened by",
			ClosedAt: "Closed", ClosedBy: "Closed by", PrintedAt: "Printed",
			Checks: "Checks", Guests: "Guests",
			Sales: "Sales", Cash: "  Cash", Card: "  Card",
			Transfer: "  Transfer",
			Debt:     "On credit", Service: "Service charge", Discount: "Discount",
			Refunded: "Refunded", Cancelled: "Cancelled",
			OpeningFloat: "Opening float", CounterCash: "Cash sales",
			DebtOf:      "  of which debt repaid",
			Settlements: "From couriers", ManualIn: "Cash in", ManualOut: "Cash out",
			Expected: "Expected in drawer", Counted: "Counted",
			Variance: "Difference",
		}
	default:
		return shiftWords{
			XTitle: "X-HISOBOT", ZTitle: "Z-HISOBOT",
			OpenedAt: "Ochilgan", OpenedBy: "Ochdi",
			ClosedAt: "Yopilgan", ClosedBy: "Yopdi", PrintedAt: "Chop etildi",
			Checks: "Cheklar", Guests: "Mehmonlar",
			Sales: "Sotuv", Cash: "  Naqd", Card: "  Karta",
			Transfer: "  O'tkazma",
			Debt:     "Qarzga", Service: "Xizmat haqi", Discount: "Chegirma",
			Refunded: "Qaytarilgan", Cancelled: "Bekor qilingan",
			OpeningFloat: "Kassa qoldig'i", CounterCash: "Naqd sotuv",
			DebtOf:      "  shundan qarz qaytdi",
			Settlements: "Kuryerlardan", ManualIn: "Kirim", ManualOut: "Chiqim",
			Expected: "Kassada bo'lishi kerak", Counted: "Sanaldi",
			Variance: "Farq",
		}
	}
}
