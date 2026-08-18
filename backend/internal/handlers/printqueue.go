package handlers

import (
	"context"
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
	lines := receipt.Render(kind, tpl, data)
	queued := 0
	now := time.Now()
	for _, p := range set.Printers {
		if !p.Prints(string(kind)) {
			continue
		}
		payload := escpos.Encode(lines, escpos.Options{
			Charset:   charsetOf(p),
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
		"tries":    bson.M{"$lt": models.MaxPrintTries},
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
