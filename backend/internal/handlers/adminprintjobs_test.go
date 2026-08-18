package handlers

import (
	"strings"
	"testing"
)

// ⚠️ A finished job is never re-sent. "It did not come out" is answered by the
// reprint button on the sale, which builds a fresh document and says so;
// quietly re-queueing a job that already printed gives the guest two receipts
// and the kitchen two tickets — and the kitchen acts on both.
func TestRetryOnlyTouchesAJobThatNeverPrinted(t *testing.T) {
	src := readSource(t, "adminprintjobs.go")
	fn := between(t, src, "func (h *Handler) AdminRetryPrintJob", "\n}\n")

	if !strings.Contains(fn, `filter["doneAt"] = bson.M{"$exists": false}`) {
		t.Fatal("a job that already printed can be sent again")
	}
	// The branch stays in the filter, as everywhere else: an id is easy to
	// come by and a manager must not reach another kitchen's queue.
	if !strings.Contains(fn, "h.orderScope(r)") {
		t.Fatal("the retry is not scoped to the manager's branch")
	}
	// ⚠️ The stored bytes are re-offered, not rebuilt: a receipt regenerated
	// after a price changed is a different document from the one the guest was
	// charged for.
	if strings.Contains(fn, "receipt.Render") || strings.Contains(fn, "queueReceipt") {
		t.Fatal("the retry is rebuilding the document instead of re-sending it")
	}
}

// The list is bounded in time. The print queue is the one collection that grows
// with traffic rather than with the business, and a screen that reads all of it
// gets slower the better the restaurant does.
func TestThePrintQueueScreenReadsABoundedWindow(t *testing.T) {
	src := readSource(t, "adminprintjobs.go")
	fn := between(t, src, "func (h *Handler) AdminPrintJobs", "\n}\n")

	if !strings.Contains(fn, `filter["createdAt"]`) {
		t.Fatal("the print queue screen reads the whole collection")
	}
	if !strings.Contains(fn, "SetLimit(200)") {
		t.Fatal("the print queue screen has no row limit")
	}
}
