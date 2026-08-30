package instore

// ---- The bank terminal on the counter ----
//
// ⚠️ **This file deliberately contains no driver, and that is the finding
// rather than a gap.** The ask was ordinary and correct: the cashier should not
// retype the total into the bank terminal, the till should push the amount to
// it and learn whether the card went through. Every part of that is possible —
// the terminals do it for other cash-register software every day.
//
// What does not exist is a **published protocol**. Checked 2026-08-30:
//
//	humocard.uz    Smart PIN Pad, sold specifically "for integration with cash
//	               register systems" — and the page's integration instructions
//	               are an email address (info@nmpc.uz).
//	rhmt.uz        Rahmat POS. "Open API" in the marketing copy; the developer
//	               page says to contact the partnership department for the
//	               technical documentation.
//	uzkassa.uz     Same shape, same answer.
//
// So the honest position is the one internal/fiscal already takes for the
// providers whose API nobody publishes: the names are listed, the credentials
// can be typed in ahead of a contract, and **enabling is refused with a
// reason**. The alternative — inventing an endpoint from the shape of the
// others — produces code that compiles, reviews cleanly, and leaves a
// restaurant believing its terminal is driven when the cashier is still typing
// the amount by hand. That failure is invisible from every screen we own.
//
// ⚠️ **Meanwhile the ask is answered another way, and better.** CLICK Pass and
// Uzum FastPay remove the retyping *without* a terminal at all: the cashier
// scans the code on the guest's phone and the card is charged from our screen,
// with the amount taken from the check. For a guest who wants to tap a physical
// card the terminal stays what it has always been — its own device, its own
// receipt, recorded here as `card` — and that is a truthful record rather than
// a green tick over a payment we cannot see.
//
// When a protocol document does arrive, the work is one file: implement
// Charger, drop the id into the list below, flip Ready. Charge pushes the
// amount and blocks until the terminal answers, Status re-asks after a timeout,
// Reverse voids. The interface was written with this in mind — Confirm and
// Fiscal are no-ops for a terminal, as they already are for one provider each.

func terminals() []Info {
	// Every one of these is Ready:false. The list is short and only holds
	// vendors an owner in Uzbekistan is actually likely to have been sold.
	needs := []string{NeedServiceID, NeedUserID, NeedSecretKey, NeedBaseURL}
	return []Info{
		{"humo_pinpad", "HUMO Smart PIN Pad", KindTerminal, false,
			"⚠️ Protokoli ochiq emas — NMPC bilan shartnoma orqali beriladi (info@nmpc.uz). Hujjat kelsa ulanadi.",
			needs},
		{"rahmat_terminal", "Rahmat POS terminali", KindTerminal, false,
			"⚠️ «Ochiq API» deyilgan, lekin hujjat nashr qilinmagan — Rahmat hamkorlik bo'limi orqali beriladi.",
			needs},
		{"uzkassa_terminal", "UZKASSA / Smart Business terminali", KindTerminal, false,
			"⚠️ Hujjati shartnoma bilan beriladi. Hozircha karta terminaldan olinadi va kassada «Karta» deb yoziladi.",
			needs},
	}
}
