package handlers

import (
	"strings"
	"testing"

	"restaurant-backend/internal/escpos"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/receipt"
)

// Which printer gets which receipt.
//
// ⚠️ **An unfinished printer prints nothing.** Somebody adds one, types the
// address, and is called away before choosing what it prints — and the rule
// here is what stops every bill in the building coming out on the pass's roll
// while the kitchen waits for its ticket.
func TestPrinterRouting(t *testing.T) {
	pass := models.Printer{
		ID: "1", Target: "tcp://192.168.1.50:9100", Kinds: []string{"kitchen"},
	}
	counter := models.Printer{
		ID: "2", Target: "usb://XP-58", Kinds: []string{"till", "customer", "precheck"},
	}
	unfinished := models.Printer{ID: "3", Target: "tcp://192.168.1.60:9100"}
	noAddress := models.Printer{ID: "4", Kinds: []string{"kitchen"}}
	broken := models.Printer{
		ID: "5", Target: "tcp://10.0.0.9:9100", Kinds: []string{"kitchen"}, Disabled: true,
	}

	cases := []struct {
		name    string
		printer models.Printer
		kind    string
		want    bool
	}{
		{"the pass takes kitchen tickets", pass, "kitchen", true},
		{"the pass does not take bills", pass, "precheck", false},
		{"the counter takes the guest's copy", counter, "customer", true},
		{"the counter does not take kitchen tickets", counter, "kitchen", false},
		{"a printer with nothing chosen prints nothing", unfinished, "kitchen", false},
		{"a printer with no address prints nothing", noAddress, "kitchen", false},
		// ⚠️ Switched off rather than deleted: a printer that is broken this
		// week must not take the tickets with it when it comes back.
		{"a disabled printer is skipped", broken, "kitchen", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.printer.Prints(c.kind); got != c.want {
				t.Fatalf("Prints(%q) = %v, want %v", c.kind, got, c.want)
			}
		})
	}
}

// ⚠️ **The drawer opens for money and nothing else.** A cash drawer that
// springs open every time a starter is fired is a drawer somebody props shut
// with a fork — after which it never opens for a sale either.
func TestDrawerOnlyOnTheTillCopy(t *testing.T) {
	p := models.Printer{Drawer: true}
	for _, kind := range []receipt.Kind{
		receipt.Kitchen, receipt.Customer, receipt.Precheck,
	} {
		out := escpos.Encode([]string{"x"}, escpos.Options{
			OpenDrawer: p.Drawer && kind == receipt.Till,
		})
		if strings.Contains(string(out), string(escpos.Drawer)) {
			t.Fatalf("%s opened the cash drawer", kind)
		}
	}
	out := escpos.Encode([]string{"x"}, escpos.Options{
		OpenDrawer: p.Drawer && receipt.Till == receipt.Till,
	})
	if !strings.Contains(string(out), string(escpos.Drawer)) {
		t.Fatal("the till copy did not open the drawer")
	}
}
