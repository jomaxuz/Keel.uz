package handlers

import "testing"

// The SMS part count is the number an owner budgets a campaign from, and the
// boundary that decides it is invisible on screen: Uzbek written properly (oʻ,
// gʻ) and anything Cyrillic falls outside GSM-7, so one message is 70
// characters rather than 160. Getting this wrong understates a campaign's cost
// by more than half, and it is only discovered on the invoice.
func TestSMSPartsCountsAlphabetNotJustLength(t *testing.T) {
	cases := []struct {
		name string
		text string
		want int
	}{
		{"bo'sh", "", 0},
		{"qisqa lotin", "Salom", 1},
		{"160 lotin — hali bitta", string(make([]byte, 0)) + repeat("a", 160), 1},
		{"161 lotin — ikkita", repeat("a", 161), 2},
		// The trap: the same length in Cyrillic is three messages, not one.
		{"70 kirill — bitta", repeat("я", 70), 1},
		{"71 kirill — ikkita", repeat("я", 71), 2},
		{"160 kirill — uchta", repeat("я", 160), 3},
		// The apostrophe Uzbek actually uses is not the ASCII one, and it is
		// enough on its own to take the whole message out of GSM-7.
		{"o‘zbek tutuq belgisi", "Bugun oʻsha taom", 1},
	}
	for _, c := range cases {
		if got := smsParts(c.text); got != c.want {
			t.Errorf("%s: smsParts = %d, kutilgan %d", c.name, got, c.want)
		}
	}
}

func TestGSM7RejectsWhatGatewaysReject(t *testing.T) {
	for _, s := range []string{"Salom", "15% chegirma", "Tel: +998901234567"} {
		if !isGSM7(s) {
			t.Errorf("isGSM7(%q) = false — narx ortiqcha hisoblanadi", s)
		}
	}
	for _, s := range []string{"Привет", "oʻsha", "gʻalati", "🍕"} {
		if isGSM7(s) {
			t.Errorf("isGSM7(%q) = true — narx kam hisoblanadi", s)
		}
	}
}

func repeat(s string, n int) string {
	out := make([]rune, 0, n)
	r := []rune(s)[0]
	for i := 0; i < n; i++ {
		out = append(out, r)
	}
	return string(out)
}
