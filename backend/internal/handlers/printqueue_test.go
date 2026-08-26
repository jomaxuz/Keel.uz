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

// ⚠️ **A kitchen ticket is rendered per printer; the money receipts are not.**
// A bar and a kitchen each need a roll carrying only their own dishes — but a
// bill split across two printers is two halves of a receipt, and a guest's copy
// with half the meal on it is worse than none.
func TestOnlyTheKitchenTicketIsRouted(t *testing.T) {
	src := readSource(t, "printqueue.go")
	fn := between(t, src, "func linesFor", "\n}\n")

	if !strings.Contains(fn, "kind != receipt.Kitchen") {
		t.Fatal("a bill or a guest's copy is being split across printers")
	}
	if !strings.Contains(fn, "p.Takes(") {
		t.Fatal("the kitchen ticket stopped being routed")
	}
	// ⚠️ No lines means no ticket. A bar printer that took nothing must stay
	// silent: a header and a rule with nothing between them is a slip somebody
	// walks over to read before discovering it was not for them.
	if !strings.Contains(fn, "len(mine) == 0") {
		t.Fatal("a printer with nothing to print is being sent an empty ticket")
	}
}

// ⚠️ **Rendered inside the loop, not once above it.** Laying the ticket out
// once and handing the same bytes to every printer is right while a restaurant
// has one printer, and silently wrong the moment it has two.
func TestEachPrinterGetsItsOwnSheet(t *testing.T) {
	src := readSource(t, "printqueue.go")
	fn := between(t, src, "func (h *Handler) queueReceiptTo", "\n}\n")

	if !strings.Contains(fn, "lines, ok := linesFor(") {
		t.Fatal("every printer is being handed the same rendered ticket again")
	}
}

// ⚠️ **The hole the routing shipped with, and the reason a kitchen stopped
// printing.**
//
// A dish whose section is ticked on no printer used to print nowhere — no
// error, no log, no paper. It happens the moment a restaurant adds a category
// after configuring its printers, which is to say on an ordinary Tuesday, and
// the first symptom is a guest waiting for food nobody was told to cook.
func TestADishNobodyClaimsStillReachesTheKitchen(t *testing.T) {
	kitchen := models.Printer{
		ID: "k", Name: "Oshxona", Target: "tcp://10.0.0.1:9100",
		Kinds: []string{"kitchen"}, Categories: []string{"soups"},
	}
	bar := models.Printer{
		ID: "b", Name: "Bar", Target: "tcp://10.0.0.2:9100",
		Kinds: []string{"kitchen"}, Categories: []string{"drinks"},
	}
	d := receipt.Data{
		Number: "1",
		Lines: []receipt.Line{
			{Name: "Sho'rva", Qty: 1, MenuItemID: "m1"},
			{Name: "Yangi taom", Qty: 1, MenuItemID: "m2"}, // a category nobody ticked
		},
	}
	cats := map[string]string{"m1": "soups", "m2": "desserts"}
	printers := []models.Printer{kitchen, bar}
	orphans := orphanLines(receipt.Kitchen, d, printers, cats)

	if !orphans["m2"] {
		t.Fatal("the unclaimed dish was not noticed")
	}
	if orphans["m1"] {
		t.Fatal("a dish the kitchen claims was treated as unclaimed")
	}

	// ⚠️ Broadcast rather than dropped: a duplicate slip is a barman asking why
	// he has a dessert; a missing one is a table waiting forty minutes.
	for _, p := range printers {
		lines, ok := linesFor(receipt.Kitchen, receipt.Template{WidthMM: 80}, d, p, cats, orphans)
		if !ok {
			t.Fatalf("%s printed nothing at all", p.Name)
		}
		if !strings.Contains(strings.Join(lines, "\n"), "Yangi taom") {
			t.Fatalf("%s did not get the unclaimed dish", p.Name)
		}
	}
}

// And the ordinary case is untouched: with every dish claimed, each roll still
// carries only its own.
func TestClaimedDishesAreStillSplit(t *testing.T) {
	kitchen := models.Printer{
		ID: "k", Target: "tcp://10.0.0.1:9100",
		Kinds: []string{"kitchen"}, Categories: []string{"soups"},
	}
	bar := models.Printer{
		ID: "b", Target: "tcp://10.0.0.2:9100",
		Kinds: []string{"kitchen"}, Categories: []string{"drinks"},
	}
	d := receipt.Data{Number: "1", Lines: []receipt.Line{
		{Name: "Sho'rva", Qty: 1, MenuItemID: "m1"},
		{Name: "Choy", Qty: 1, MenuItemID: "m2"},
	}}
	cats := map[string]string{"m1": "soups", "m2": "drinks"}
	orphans := orphanLines(receipt.Kitchen, d, []models.Printer{kitchen, bar}, cats)
	if len(orphans) != 0 {
		t.Fatalf("nothing was unclaimed, but %v was", orphans)
	}
	k, _ := linesFor(receipt.Kitchen, receipt.Template{WidthMM: 80}, d, kitchen, cats, orphans)
	if strings.Contains(strings.Join(k, "\n"), "Choy") {
		t.Fatal("the kitchen got the bar's drink")
	}
}

// ⚠️ **The burst a restaurant reported: nothing printed all afternoon, then
// the whole afternoon printed at once.**
//
// The queue had no lifetime, so a job waited for an agent forever. A till
// switched on in the evening drained everything — and a kitchen ticket for an
// order served five hours ago is not a late ticket, it is an instruction to
// cook it again, arriving at the pass looking exactly like a new one.
func TestAStaleJobIsNeverHandedOut(t *testing.T) {
	src := between(t, readLossSource(t, "printqueue.go"),
		"func (h *Handler) nextPrintJob", "\n}\n")
	if !strings.Contains(src, `"createdAt": bson.M{"$gte": now.Add(-models.MaxPrintAge)}`) {
		t.Fatal("the queue has no lifetime again — a whole shift can print at once")
	}
}

// ⚠️ **Dropped is not deleted.** A ticket that never printed is a thing that
// happened to a restaurant — possibly an order the kitchen never saw — and the
// one outcome worse than a stack of dead paper is no paper and no record of
// why. A job the filter quietly steps over also stays in the collection
// forever, `doneAt` absent, looking to every future reader like something still
// waiting.
func TestAnExpiredJobIsMarkedRatherThanForgotten(t *testing.T) {
	src := readLossSource(t, "printqueue.go")
	if !strings.Contains(src, "func (h *Handler) expirePrintJobs") {
		t.Fatal("expired jobs are silently skipped and left looking pending")
	}
	if !strings.Contains(src, `"error": "kassa o'chiq edi`) {
		t.Fatal("an expired job carries no reason")
	}
	// ⚠️ Only jobs that have no reason yet: a real printer error must not be
	// overwritten with "the till was off", which would send somebody to check
	// the wrong thing.
	if !strings.Contains(src, `"error":     bson.M{"$in": []any{nil, ""}}`) {
		t.Fatal("a genuine printer failure would be relabelled as a stale job")
	}
	// ⚠️ **And given up on, which is a second and independent stop.** Writing a
	// reason does not make a job ineligible — the hand-out filter decides that,
	// and it would still offer this one if the age clause were loosened or
	// moved. The failure these prevent is a restaurant's whole afternoon coming
	// out of a printer at once, which is not worth leaving to one line.
	if !strings.Contains(src, `bson.M{"$set": bson.M{"tries": models.MaxPrintTries}}`) {
		t.Fatal("an expired job is still eligible if the age filter ever moves")
	}
}

// ⚠️ Run on the agent's own poll: the one moment we know a till is alive and
// which branch it belongs to — and exactly the moment the backlog would
// otherwise be drained onto the paper.
func TestExpiryRunsWhenTheTillWakesUp(t *testing.T) {
	src := readLossSource(t, "fiscalagent.go")
	seen := strings.Index(src, "h.markAgentSeen(r.Context(), set)")
	expire := strings.Index(src, "h.expirePrintJobs(r.Context(), set.BranchID)")
	next := strings.Index(src, "h.nextPrintJob(r.Context(), set.BranchID)")
	if seen < 0 || expire < 0 || next < 0 {
		t.Fatal("the agent no longer expires the backlog before draining it")
	}
	if expire > next {
		t.Fatal("the backlog is handed out before it is expired — the burst is back")
	}
}
