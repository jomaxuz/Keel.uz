package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- The print queue, from the panel ----
//
// ⚠️ **A failed print is the quietest failure in the whole system.** Everything
// else that goes wrong is visible to somebody: an unfiled receipt raises an
// alert, an order the till never sent shows a red badge, a card payment that
// fails leaves the guest standing there. A kitchen ticket that never printed
// leaves no trace at all — the order is on the screen, the sale is in the
// reports, and the only symptom is a plate nobody made, twenty minutes later,
// noticed by a guest.
//
// The queue already recorded every attempt and the printer's own words. Nobody
// could read them.
//
// ⚠️ **Failures first, then the rest.** A list sorted by time answers "what did
// we print today", which nobody asks; the question this screen exists for is
// "what did **not** print", and on a busy evening the answer is three rows
// among four hundred.

type printJobRow struct {
	ID          string     `json:"id"`
	Kind        string     `json:"kind"`
	PrinterName string     `json:"printerName,omitempty"`
	Target      string     `json:"target"`
	Number      string     `json:"number,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	DoneAt      *time.Time `json:"doneAt,omitempty"`
	Tries       int        `json:"tries,omitempty"`
	// The printer's own words. They usually name something fixable in
	// seconds — no paper, wrong name, unplugged — and a summary would turn an
	// instruction into a category.
	Error string `json:"error,omitempty"`
	// Given up on: tried MaxPrintTries times and still owed. The only rows
	// anybody has to act on.
	Failed bool `json:"failed,omitempty"`
	// Handed to the agent and not yet answered for.
	Working bool `json:"working,omitempty"`
}

// AdminPrintJobs lists what the printers were asked to do.
func (h *Handler) AdminPrintJobs(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := bson.M{}
	for k, v := range scope {
		filter[k] = v
	}
	// ⚠️ Bounded to the last day by default. The queue is the one collection
	// that grows with **traffic** rather than with the business — a busy
	// kitchen prints thousands a week — and a screen that reads all of it is a
	// screen that gets slower the better the restaurant does.
	since := time.Now().Add(-24 * time.Hour)
	if h, err := strconv.Atoi(r.URL.Query().Get("hours")); err == nil && h > 0 && h <= 24*14 {
		since = time.Now().Add(-time.Duration(h) * time.Hour)
	}
	filter["createdAt"] = bson.M{"$gte": since}
	if r.URL.Query().Get("failed") == "1" {
		filter["doneAt"] = bson.M{"$exists": false}
		filter["tries"] = bson.M{"$gte": models.MaxPrintTries}
	}

	cur, err := h.Store.PrintJobs.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var jobs []models.PrintJob
	if err := cur.All(r.Context(), &jobs); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	rows := make([]printJobRow, 0, len(jobs))
	failed := 0
	for _, j := range jobs {
		row := printJobRow{
			ID:          j.ID.Hex(),
			Kind:        j.Kind,
			PrinterName: j.PrinterName,
			Target:      j.Target,
			Number:      j.Number,
			CreatedAt:   j.CreatedAt.In(time.Local),
			Tries:       j.Tries,
			Error:       j.Error,
		}
		if j.DoneAt != nil {
			at := j.DoneAt.In(time.Local)
			row.DoneAt = &at
		} else if j.Tries >= models.MaxPrintTries {
			row.Failed = true
			failed++
		} else if j.TakenAt != nil {
			row.Working = true
		}
		rows = append(rows, row)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"jobs": rows, "failed": failed})
}

// AdminRetryPrintJob puts a given-up job back in the queue.
//
// ⚠️ **The bytes are the ones that were built when the sale happened**, not
// rebuilt now: a receipt reprinted after a menu price changed would be a
// different document from the one the guest was charged for. The retry is the
// same paper, offered again.
func (h *Handler) AdminRetryPrintJob(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := bson.M{"_id": id}
	for k, v := range scope {
		filter[k] = v
	}
	// ⚠️ A job that already printed is not retried. "It did not come out" is
	// answered by the reprint button on the sale itself, which builds a fresh
	// document and says so; silently re-sending a finished job would give the
	// guest two receipts and the kitchen two tickets.
	filter["doneAt"] = bson.M{"$exists": false}
	res, err := h.Store.PrintJobs.UpdateOne(r.Context(), filter, bson.M{
		"$set":   bson.M{"tries": 0, "error": ""},
		"$unset": bson.M{"takenAt": ""},
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.MatchedCount == 0 {
		httpx.Error(w, http.StatusNotFound, "chop etish topshirig'i topilmadi")
		return
	}
	h.logAction(r, "print.retry", "print", id.Hex(), "", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
