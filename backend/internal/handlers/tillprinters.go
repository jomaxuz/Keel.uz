package handlers

// ---- The branch's printers, from the counter's own screen ----
//
// The printer list already existed and had exactly one door: `/admin/receipts`,
// behind a panel login. That is the wrong door for the event that opens it. A
// printer is connected by whoever is standing in the restaurant with the box in
// their hands — an installer, a manager, the owner on a Saturday — and the
// machine they are standing at is the till. Sending them to a panel login on
// another computer is how a kitchen printer stays in its box for a week.
//
// ⚠️ **The same list, not a second one.** These are `receipt_settings.printers`
// — the records the queue reads and the agent prints from — so a printer added
// here works for every till in the branch, prints kitchen tickets, and appears
// in the panel exactly as if it had been added there. A till-local list would be
// the third writer into a question that already has one owner, and the first
// symptom would be a kitchen ticket that prints from one screen and not the
// next.
//
// ⚠️ **The templates are not writable here**, and that is the boundary between
// the two doors. What a receipt *says* — the header, the footer, which fields
// appear, the paper width — is the restaurant's design, approved once from an
// office. Which machine it comes out of is an operational fact that changes
// when a printer dies on a Friday. The till gets the second and never the
// first, so an accidental save from the counter cannot flatten a design nobody
// on this screen can see.
//
// ⚠️ **Behind PermVoid**, the permission the exit button and the settings
// section already use, and for the reason written there: a new permission would
// sit unticked in every restaurant until each one discovered it. The set of
// people is right — every seeded management role holds it, no cashier, waiter,
// barman or host does.

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/receipt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// StaffPrinters is the branch's printer list, for the till's settings.
func (h *Handler) StaffPrinters(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermVoid)
	if !ok {
		return
	}
	// ⚠️ The branch comes from the employee, never from the request — the rule
	// every till endpoint follows. A manager who typed another branch's id
	// would otherwise be reconfiguring a kitchen they have never stood in.
	set := h.receiptSettingsOf(r.Context(), s.BranchID)
	httpx.JSON(w, http.StatusOK, map[string]any{
		// ⚠️ Never nil: a nil slice marshals to `null` and the screen renders
		// `null.length`. The JSON trap this codebase has been bitten by twice.
		"printers": printersJSON(set.Printers),
		// What each printer can be asked to print, so the screen does not carry
		// its own copy of a list the server validates against.
		"kinds": []string{
			string(receipt.Kitchen), string(receipt.Till),
			string(receipt.Customer), string(receipt.Precheck),
		},
	})
}

// printersJSON is the empty-slice rule in a function rather than at each call
// site, because the next handler to return this list will otherwise skip it —
// which is exactly how `downloads` reached a browser as null.
func printersJSON(in []models.Printer) []models.Printer {
	if in == nil {
		return []models.Printer{}
	}
	return in
}

// StaffSavePrinters replaces the branch's printer list.
//
// ⚠️ **A whole-list replace, and it is safe here for a reason worth stating.**
// Everywhere else this codebase insists on partial updates, because a form that
// posts what it did not load silently blanks fields it never showed. This form
// loads and shows the entire list, so what it sends is what somebody looked at.
// The templates — the part it does *not* show — are the ones excluded from the
// `$set` below rather than sent back empty.
func (h *Handler) StaffSavePrinters(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermVoid)
	if !ok {
		return
	}
	var req struct {
		Printers []models.Printer `json:"printers"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// ⚠️ **The same cleaner the panel uses.** It is what refuses an address
	// that would not parse — "tcp://192.168.1.5o:9100" is discovered here
	// rather than at eight in the evening by a ticket that never comes out.
	// A second cleaner for the second door is a second set of rules.
	printers := cleanPrinters(req.Printers)
	if _, err := h.Store.Receipts.UpdateOne(r.Context(),
		bson.M{"branchId": s.BranchID},
		bson.M{"$set": bson.M{
			"branchId":  s.BranchID,
			"printers":  printers,
			"updatedAt": time.Now(),
		}},
		// ⚠️ Upsert, because a branch that has never opened the panel's receipt
		// page has no document at all — which is every branch connecting its
		// first printer, the case this endpoint exists for. The templates are
		// then absent rather than empty, and receiptSettingsOf fills in the
		// defaults exactly as it does for any other unconfigured branch.
		options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActSettingsUpdate, "receipts", s.BranchID.Hex(), s.Name, "")
	set := h.receiptSettingsOf(r.Context(), s.BranchID)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"printers": printersJSON(set.Printers),
	})
}

// StaffTestPrinter sends a sample receipt to one printer.
//
// ⚠️ **The most useful button on the screen.** An address can be typed
// correctly and the printer still be switched off, on another subnet, or shared
// under a different name — and every one of those looks identical from here
// until a real ticket fails, at the worst possible moment. The person who can
// look at the paper is the one pressing this.
//
// ⚠️ **Queued like any other job**, never printed from the server: the server
// is in a data centre and the printer is in a restaurant, so it is the agent on
// the counter that reaches it. A test that took a different path would be a
// test of the wrong thing.
func (h *Handler) StaffTestPrinter(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermVoid)
	if !ok {
		return
	}
	var req struct {
		PrinterID string `json:"printerId"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := h.receiptSettingsOf(r.Context(), s.BranchID)
	var target *models.Printer
	for i := range set.Printers {
		if set.Printers[i].ID == req.PrinterID {
			target = &set.Printers[i]
			break
		}
	}
	if target == nil {
		httpx.Error(w, http.StatusNotFound, "printer topilmadi")
		return
	}
	n := h.queueTestPrint(r, s.BranchID, *target, set)
	// ⚠️ **"Queued" was the whole answer, and it was not one.** The job is the
	// agent's from here, and when the agent cannot reach the printer it records
	// the reason — on a document only the panel reads. So the person standing
	// at the printer, who pressed the button, was told the one thing that is
	// true of both outcomes, and the answer to "why is there no paper" was on
	// another screen in another room.
	//
	// The id of what was just queued goes back with it, so the screen can ask.
	var jobID string
	if n > 0 {
		var job models.PrintJob
		if err := h.Store.PrintJobs.FindOne(r.Context(),
			bson.M{"branchId": s.BranchID, "printerId": target.ID},
			options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}}),
		).Decode(&job); err == nil {
			jobID = job.ID.Hex()
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"queued": n, "jobId": jobID})
}

// StaffTestPrintState says what became of a test job.
//
// ⚠️ **Three answers, and the third is the useful one.** Printed, refused with
// the printer's own words, or *still sitting in the queue* — which means no
// agent took it, and that is a completely different fault from an unreachable
// printer. Without this they look identical from the counter: no paper.
func (h *Handler) StaffTestPrintState(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermVoid)
	if !ok {
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var job models.PrintJob
	// The branch is in the filter: an id alone never selects a document here.
	if err := h.Store.PrintJobs.FindOne(r.Context(),
		bson.M{"_id": id, "branchId": s.BranchID}).Decode(&job); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"done":  job.DoneAt != nil,
		"taken": job.TakenAt != nil,
		"error": job.Error,
	})
}
