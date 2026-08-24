package handlers

import (
	"os"
	"strings"
	"testing"
)

// Nothing that prints may read `restaurant.name` on its own.
//
// ⚠️ **This trap has now been paid for four times** — the export filename, the
// keel.uz partner strip, the bot's messages, and now every receipt and every
// Z-report a restaurant files. The mechanism is always the same: the settings
// page writes the name to the *brand*, so the company document keeps the
// installer's "My Restaurant" forever, and nothing anywhere looks broken. It is
// only visible on paper, in a guest's hand.
//
// A source check rather than a behaviour test, and deliberately: the defect is
// not that one function returns the wrong string, it is that the wrong field is
// within reach of any new print path. This fails on the next one.
func TestNothingPrintsTheCompanyNameDirectly(t *testing.T) {
	for _, file := range []string{
		"receipts.go", "tillprint.go", "tillshiftreport.go",
	} {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		if strings.Contains(string(raw), "d.Title = rest.Name") {
			t.Errorf("%s prints restaurant.name directly — a restaurant that has "+
				"named its brand will print \"My Restaurant\" on paper. Use "+
				"h.receiptTitle(ctx).", file)
		}
	}
}
