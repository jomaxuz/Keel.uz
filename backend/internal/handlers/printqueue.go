package handlers

import (
	"context"
	"log"
	"time"

	"restaurant-backend/internal/escpos"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/receipt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Queueing receipts for the printers in one restaurant.
//
// ⚠️ **The server builds the bytes; the agent carries them.** Layout, code page
// and every business rule stay here, where they are tested and where somebody
// reads the log. The program on the restaurant's PC opens a socket or a file
// handle and writes — the same division the fiscal agent is built on.

// queueReceipt renders one kind of receipt and puts it on every printer that
// asked for it. Returns how many printers took it.
//
// ⚠️ **Never fails the thing that triggered it.** A kitchen ticket is queued as
// a side effect of sending an order; a printer that is unplugged must not be the
// reason the order does not reach the kitchen screen. Errors are recorded on the
// job, where the panel can show them.
func (h *Handler) queueReceipt(
	ctx context.Context,
	branchID primitive.ObjectID,
	kind receipt.Kind,
	tpl receipt.Template,
	data receipt.Data,
	o *models.Order,
) int {
	return h.queueReceiptTo(ctx, branchID,
		h.receiptSettingsOf(ctx, branchID), kind, tpl, data, o)
}

// queueReceiptTo is the same against a given set of printers — the test-print
// button aims at exactly one of them.
func (h *Handler) queueReceiptTo(
	ctx context.Context,
	branchID primitive.ObjectID,
	set models.ReceiptSettings,
	kind receipt.Kind,
	tpl receipt.Template,
	data receipt.Data,
	orders ...*models.Order,
) int {
	var o *models.Order
	if len(orders) > 0 {
		o = orders[0]
	}
	if len(set.Printers) == 0 {
		return 0
	}
	// ⚠️ **Rendered per printer, not once for all of them.** A kitchen ticket
	// used to be laid out once and handed to every printer that took its kind
	// — which is right while a restaurant has one. The moment it has a bar and
	// a kitchen, each roll has to carry only its own dishes, and that is a
	// different sheet of paper per printer rather than a copy of one.
	//
	// The other three kinds are not routed: a bill is the whole bill, and a
	// guest's copy split across two printers is two halves of a receipt.
	// ⚠️ Before anything is laid out, and on a copy the caller does not see:
	// the same `Data` is used for every printer on this ticket, and translating
	// it in place per printer would translate it twice.
	h.translateLines(ctx, &data, tpl.Lang)
	cats := h.categoryOf(ctx, data)
	// ⚠️ **A dish no printer claims goes to all of them, and this is the most
	// important line in the file.**
	//
	// Routing was added so a bar and a kitchen each get their own roll. The
	// version that shipped let a dish fall off the end: if its section was
	// ticked on no printer, it printed nowhere — silently, with no error
	// anywhere, and the first symptom is a guest waiting for food nobody was
	// ever told to cook. It happens the moment a restaurant adds a category
	// after configuring its printers, which is to say on an ordinary Tuesday.
	//
	// So an unclaimed line is broadcast rather than dropped. A duplicate ticket
	// is a barman asking why he has a lag'mon slip; a missing one is a table
	// waiting forty minutes. Only one of those is recoverable by the people in
	// the room.
	orphans := orphanLines(kind, data, set.Printers, cats)
	if len(orphans) > 0 {
		log.Printf("print: %d line(s) belong to no printer — sent to every kitchen printer", len(orphans))
	}
	queued := 0
	now := time.Now()
	for _, p := range set.Printers {
		if !p.Prints(string(kind)) {
			continue
		}
		lines, ok := linesFor(kind, tpl, data, p, cats, orphans)
		// ⚠️ **No lines means no ticket, not an empty one.** A bar printer that
		// took nothing from this order must stay silent; a header and a rule
		// with nothing between them is a slip a barman has to walk over and
		// read before discovering it was not for them.
		if !ok {
			continue
		}
		// ⚠️ **The logo is the template's decision and the kitchen never gets
		// one**, however the box is ticked: every dot is time at the pass and
		// paper off the roll, and a cook does not need to be told which
		// restaurant they work in.
		var banner []byte
		if tpl.Logo && kind != receipt.Kitchen {
			banner = h.logoRaster(ctx, tpl.WidthMM)
		}
		payload := escpos.Encode(lines, escpos.Options{
			Charset:   charsetOf(p),
			Banner:    banner,
			FeedLines: tpl.FeedLines,
			Cut:       p.Cut,
			FullCut:   p.FullCut,
			// ⚠️ The drawer opens for money, never for a kitchen ticket or a
			// bill: a drawer that springs open every time a starter is fired is
			// a drawer somebody props shut with a fork.
			OpenDrawer: p.Drawer && kind == receipt.Till,
		})
		// ⚠️ The QR goes after the text and only on the guest's copy — it is
		// their right to check the sale, and it is the one thing the characters
		// cannot carry.
		if kind == receipt.Customer && data.QRText != "" {
			payload = append(payload[:len(payload)], escpos.EncodeQR(data.QRText)...)
		}
		copies := p.Copies
		if copies < 1 {
			copies = 1
		}
		for i := 0; i < copies; i++ {
			job := models.PrintJob{
				BranchID:    branchID,
				PrinterID:   p.ID,
				PrinterName: p.Name,
				Target:      p.Target,
				Kind:        string(kind),
				Payload:     payload,
				CreatedAt:   now,
			}
			if o != nil {
				job.OrderID = o.ID
				job.Number = o.Number
			}
			if _, err := h.Store.PrintJobs.InsertOne(ctx, job); err == nil {
				queued++
			}
		}
	}
	return queued
}

// linesFor lays this receipt out for one printer, or reports that this printer
// has nothing to print.
func linesFor(
	kind receipt.Kind, tpl receipt.Template, d receipt.Data,
	p models.Printer, cats map[string]string, orphans map[string]bool,
) ([]string, bool) {
	// ⚠️ Only the kitchen ticket is routed. The money receipts are documents
	// about a whole sale, and half of one is not a receipt.
	if kind != receipt.Kitchen {
		return receipt.Render(kind, tpl, d), true
	}
	mine := make([]receipt.Line, 0, len(d.Lines))
	for _, l := range d.Lines {
		// A line this printer claims, or one nobody claimed — see the note at
		// the call site for why the second half exists.
		if p.Takes(l.MenuItemID, cats[l.MenuItemID]) || orphans[l.MenuItemID] {
			mine = append(mine, l)
		}
	}
	if len(mine) == 0 {
		return nil, false
	}
	d.Lines = mine
	return receipt.Render(kind, tpl, d), true
}

// orphanLines is every dish on this ticket that no printer would take.
//
// ⚠️ **Computed across all of them before any is rendered**, because "nobody
// claimed this" is not a fact any single printer can know. Doing it per printer
// would have each one ask "is this mine?" and none ask "is it anybody's?" —
// which is exactly how the dish disappeared.
func orphanLines(
	kind receipt.Kind, d receipt.Data, printers []models.Printer,
	cats map[string]string,
) map[string]bool {
	if kind != receipt.Kitchen {
		return nil
	}
	claimed := map[string]bool{}
	for _, p := range printers {
		if !p.Prints(string(kind)) {
			continue
		}
		for _, l := range d.Lines {
			if p.Takes(l.MenuItemID, cats[l.MenuItemID]) {
				claimed[l.MenuItemID] = true
			}
		}
	}
	out := map[string]bool{}
	for _, l := range d.Lines {
		if !claimed[l.MenuItemID] {
			out[l.MenuItemID] = true
		}
	}
	return out
}

// translateLines puts the dish names into the language the receipt is printed in.
//
// ⚠️ **The name on the order is the one the guest ordered under, and it is
// frozen there in Uzbek.** A restaurant that set its receipts to Russian got
// Russian headings around a list of Uzbek dishes — which reads as a half-built
// feature, and is: the translations have been in the menu all along
// (`nameRu` / `nameEn`), and nothing on the printing path ever looked at them.
//
// ⚠️ **Read now rather than frozen, the same rule the category follows.** What
// a dish is called in Russian is a fact about the product, not about the sale.
// A dish renamed since is printed under its current name, which is the name the
// kitchen and the guest will both recognise today.
//
// ⚠️ **An untranslated dish keeps its Uzbek name and is not blanked.** Half a
// menu is usually translated and the other half is not, and a receipt with
// empty lines where the untranslated dishes were is worse in every way than one
// with two languages on it.
func (h *Handler) translateLines(ctx context.Context, d *receipt.Data, lang string) {
	if lang == "" || lang == "uz" || len(d.Lines) == 0 {
		return
	}
	ids := make([]primitive.ObjectID, 0, len(d.Lines))
	for _, l := range d.Lines {
		if id, err := primitive.ObjectIDFromHex(l.MenuItemID); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}
	cur, err := h.Store.Menu.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		// A database blink prints the frozen names, which is where every
		// receipt was before this existed.
		return
	}
	defer cur.Close(ctx)
	names := map[string]string{}
	for cur.Next(ctx) {
		var m models.MenuItem
		if cur.Decode(&m) != nil {
			continue
		}
		if n := localName(lang, m.NameRu, m.NameEn); n != "" {
			names[m.ID.Hex()] = n
		}
	}
	for i := range d.Lines {
		if n, ok := names[d.Lines[i].MenuItemID]; ok {
			d.Lines[i].Name = n
		}
	}
}

// localName picks a translation, or nothing when there is none.
func localName(lang, ru, en string) string {
	switch lang {
	case "ru":
		return ru
	case "en":
		return en
	}
	return ""
}

// categoryOf looks up which section of the menu each dish belongs to.
//
// ⚠️ **Read now rather than frozen on the order.** A dish moved from the
// kitchen's section to the bar's should print where it is made today, not where
// it was when the table sat down — the same rule the ИКПУ follows, and for the
// same reason: this is a fact about the product, not about the sale.
func (h *Handler) categoryOf(
	ctx context.Context, d receipt.Data,
) map[string]string {
	out := map[string]string{}
	ids := make([]primitive.ObjectID, 0, len(d.Lines))
	for _, l := range d.Lines {
		if id, err := primitive.ObjectIDFromHex(l.MenuItemID); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return out
	}
	cur, err := h.Store.Menu.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		// ⚠️ An empty map routes everything to every printer that takes all
		// categories, which is where a restaurant was before this existed. A
		// database blink must not silently stop the kitchen printing.
		return out
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var m models.MenuItem
		if err := cur.Decode(&m); err == nil {
			out[m.ID.Hex()] = m.CategoryID.Hex()
		}
	}
	return out
}

// charsetOf is which code page this printer is set to.
//
// ⚠️ Latin when nothing is chosen, because the receipts ship in Uzbek Latin —
// and because a wrong page is not a formatting problem, it is a page of
// question marks the restaurant cannot read.
func charsetOf(p models.Printer) escpos.Charset {
	if p.Charset == string(escpos.Cyrillic) {
		return escpos.Cyrillic
	}
	return escpos.Latin
}

// nextPrintJob hands the agent the oldest thing waiting for a printer.
//
// ⚠️ **Taken, not deleted.** The agent may die between being handed a job and
// printing it — a Windows update, a power cut, somebody closing the window —
// and a job that was deleted on hand-out is a ticket the kitchen never sees and
// nobody can explain. It is marked taken, and released again by the timeout
// below.
func (h *Handler) nextPrintJob(
	ctx context.Context, branchID primitive.ObjectID,
) (*models.PrintJob, error) {
	stale := time.Now().Add(-2 * time.Minute)
	filter := bson.M{
		"branchId": branchID,
		"doneAt":   bson.M{"$exists": false},
		// ⚠️ **The missing field is spelled out, because it is the ordinary
		// case for every job already in a queue.** `Tries` used to be written
		// with `omitempty`, so jobs created before that was fixed carry no
		// `tries` at all — and `$lt` against a number does not match a missing
		// field. Without this branch those jobs stay invisible forever, which
		// is a restaurant's whole backlog of unprinted tickets.
		"$and": []bson.M{{"$or": []bson.M{
			{"tries": bson.M{"$exists": false}},
			{"tries": bson.M{"$lt": models.MaxPrintTries}},
		}}},
		"$or": []bson.M{
			{"takenAt": bson.M{"$exists": false}},
			// ⚠️ Released after two minutes: long enough that a printer chewing
			// through a long ticket is not handed to a second agent, short
			// enough that a crash does not cost the kitchen its order.
			{"takenAt": bson.M{"$lt": stale}},
		},
	}
	now := time.Now()
	var job models.PrintJob
	err := h.Store.PrintJobs.FindOneAndUpdate(ctx, filter,
		bson.M{"$set": bson.M{"takenAt": now}, "$inc": bson.M{"tries": 1}},
		options.FindOneAndUpdate().
			// Oldest first: a queue that printed the newest ticket first would
			// serve the table that just ordered before the one still waiting.
			SetSort(bson.D{{Key: "createdAt", Value: 1}}).
			SetReturnDocument(options.After),
	).Decode(&job)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// finishPrintJob records what the printer said.
func (h *Handler) finishPrintJob(
	ctx context.Context, id primitive.ObjectID, printErr string,
) error {
	set := bson.M{"error": printErr}
	if printErr == "" {
		set["doneAt"] = time.Now()
	}
	_, err := h.Store.PrintJobs.UpdateByID(ctx, id, bson.M{"$set": set})
	return err
}

// queueSaleReceipts prints what a paid check owes: the cashier's copy and the
// guest's.
//
// ⚠️ **Once per sale.** It is called from two places — the close, for a
// restaurant with no register, and the filing, for one with — and a retry after
// a refused filing calls the second again. Two slips for one meal is a guest
// asking which one is real, so the check carries the moment its receipt was
// queued and the second call does nothing.
//
// ⚠️ **The guest's copy is printed even when the register refused.** The sale
// happened and the person is standing there; a paper without a fiscal sign is
// what a restaurant with no register prints all day. The tax side is already
// somebody's problem — the unfiled alert names it — and it is not the guest's.
func (h *Handler) queueSaleReceipts(ctx context.Context, o *models.Order) {
	if !shouldQueueSaleReceipts(o) {
		return
	}
	set := h.receiptSettingsOf(ctx, o.BranchID)
	data := h.checkReceiptOf(ctx, o)

	// The cashier's copy first: it is the one that opens the drawer, and the
	// drawer is wanted while the money is still in the cashier's hand.
	h.queueReceiptTo(ctx, o.BranchID, set, receipt.Till, set.Till, data, o)
	h.queueReceiptTo(ctx, o.BranchID, set, receipt.Customer, set.Customer, data, o)

	now := time.Now()
	o.Check.ReceiptAt = &now
	_, _ = h.Store.Orders.UpdateOne(ctx, bson.M{"_id": o.ID},
		bson.M{"$set": bson.M{"check.receiptAt": now}})
}

// shouldQueueSaleReceipts is the once-per-sale rule, split out so it can be
// sealed in a test: a closed check that has not printed yet, and nothing else.
func shouldQueueSaleReceipts(o *models.Order) bool {
	return o != nil && o.Check != nil &&
		o.Check.ClosedAt != nil && o.Check.ReceiptAt == nil
}

// noAgentHere reports that nothing at this branch would collect a print job.
//
// ⚠️ **A queue nobody is reading is not a queue, it is a drawer.** The till app
// on the monoblock is what carries paper to a printer; with it switched off,
// a job queued from the panel or the phone sits until somebody turns the till
// on — which may be tomorrow, or never if it was a test. The person who pressed
// the button was told "added to the queue", which is true and is not the answer
// to what they asked, and they went to look at a printer that was never going
// to produce anything.
//
// That exact sequence cost a live restaurant an evening of diagnosis. The
// heartbeat to answer it has existed all along (`agentUsable`, already tested);
// nothing on the printing path asked.
//
// ⚠️ **Only ever used to say no, never to say yes.** An agent seen a minute ago
// may have been unplugged since, so a job still has to survive not being
// collected. This turns "silently never" into "told immediately", which is the
// whole of the improvement.
func (h *Handler) noAgentHere(ctx context.Context, branchID primitive.ObjectID) bool {
	var s models.FiscalSettings
	if err := h.Store.FiscalSettings.FindOne(ctx,
		bson.M{"branchId": branchID}).Decode(&s); err != nil {
		// ⚠️ A branch that has never run a till has no settings document at
		// all — and no agent either. Reading a missing document as "probably
		// fine" is what let the first version queue into nothing.
		return true
	}
	return !agentUsable(&s)
}

// errTillOff is what the panel shows instead of "added to the queue".
//
// ⚠️ It names the machine and the fix, because the person reading it is
// standing somewhere else in the building and the useful next action is "go and
// switch the till on", not "try again".
const errTillOff = "Kassa yoqilmagan — chek chiqarish uchun monoblokdagi " +
	"Keel kassa dasturini oching. Chek navbatga qo'yilmadi."
