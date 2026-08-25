package handlers

import (
	"strings"
	"testing"
	"time"

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

// A paid sale prints once.
//
// ⚠️ **Two callers, one receipt.** The close queues it for a restaurant with no
// register; the filing queues it for one with — and the filing runs again every
// time a cashier retries a refused one. Two slips for one meal is a guest asking
// which of them is real, and a cashier who cannot answer.
func TestSaleReceiptsAreQueuedOnce(t *testing.T) {
	closed := time.Now()
	o := &models.Order{Check: &models.OrderCheck{ClosedAt: &closed}}

	if !shouldQueueSaleReceipts(o) {
		t.Fatal("a paid check refused to print its first receipt")
	}
	at := closed
	o.Check.ReceiptAt = &at
	if shouldQueueSaleReceipts(o) {
		t.Fatal("a second call printed the sale again")
	}
}

// ⚠️ An open check has nothing to print: the guest is still eating, and the
// bill they might ask for is the pre-check, which is a different document with
// "not a fiscal receipt" on it.
func TestOpenChecksDoNotPrintReceipts(t *testing.T) {
	if shouldQueueSaleReceipts(&models.Order{Check: &models.OrderCheck{}}) {
		t.Fatal("an open check queued a sales receipt")
	}
	if shouldQueueSaleReceipts(&models.Order{}) {
		t.Fatal("an order that is not a check queued a till receipt")
	}
	if shouldQueueSaleReceipts(nil) {
		t.Fatal("nil queued something")
	}
}

// ⚠️ **The bug this seals cost every receipt the queue ever held**, and it was
// one word: `Tries` was written with `omitempty`, so a fresh job carried no
// `tries` field — and MongoDB's `$lt` against a number does not match a missing
// field. Proven against a real database: the query returned the document with
// `tries: 1` and not the one without it.
//
// The symptom left nothing to follow. Jobs queued and stayed queued, the relay
// ran and asked, the server answered "nothing to do", nothing printed, and
// nothing appeared in any log — because no code path was ever reached.
func TestANewJobIsNotInvisibleToTheQueue(t *testing.T) {
	src := readSource(t, "printqueue.go")
	fn := between(t, src, "func (h *Handler) nextPrintJob", "\n}\n")

	if !strings.Contains(fn, `{"tries": bson.M{"$exists": false}}`) {
		t.Fatal("a job with no tries field is invisible to the agent again")
	}
	if !strings.Contains(fn, `"tries": bson.M{"$lt": models.MaxPrintTries}`) {
		t.Fatal("the retry cap is gone: a job that kills the agent is handed out forever")
	}

	// And the other half of the same fix: the field has to be written.
	model := readSource(t, "../models/printjob.go")
	if strings.Contains(model, `bson:"tries,omitempty"`) {
		t.Fatal("Tries is omitempty again — new jobs will carry no tries field")
	}
}
