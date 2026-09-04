package handlers

// ---- The costs nothing else was recording ----
//
// ⚠️ **This is the line that made the financial report optimistic.** Deliveries
// were counted and wages were counted; rent, electricity, gas, tax, repairs and
// the couriers' own pay were not — so "in − out" read better than the month had
// been, by roughly what the building costs, every month, consistently. A number
// that is wrong in the same direction every time is one a restaurant learns to
// trust.
//
// ⚠️ **Only what has no document of its own.** A delivery is a `purchase`, a
// wage is a `staff_payment`, and both already have their own line in the report.
// Entering either here as well would count it twice — and a double-counted cost
// is indistinguishable from a real one.

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// AdminExpenses lists what has been spent, newest first.
func (h *Handler) AdminExpenses(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	from, to, err := parseRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := scopeFilter(scope)
	if rng := timeRange(from, to); len(rng) > 0 {
		filter["at"] = rng
	}
	cur, err := h.Store.Expenses.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(300))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.Expense{}
	_ = cur.All(r.Context(), &rows)

	total := 0
	for _, e := range rows {
		total += e.Amount
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"expenses": rows, "total": total})
}

// AdminCreateExpense records one cost.
func (h *Handler) AdminCreateExpense(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		At       string `json:"at"`
		Category string `json:"category"`
		Amount   int    `json:"amount"`
		Note     string `json:"note"`
		Method   string `json:"method"`
		// Whether the notes came out of the safe.
		//
		// ⚠️ **Its own question, and not implied by "cash".** Cash can come out
		// of a till drawer or somebody's pocket just as easily as out of the
		// safe, and a balance that assumed otherwise would be a confident figure
		// about a box nobody opened.
		FromSafe bool `json:"fromSafe"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Amount <= 0 {
		httpx.Error(w, http.StatusBadRequest, "summani yozing")
		return
	}
	category := clampText(strings.TrimSpace(req.Category), 60)
	if category == "" {
		// ⚠️ Required, the same rule a write-off's reason and a till entry's
		// follow: an outgoing with no description is the row that becomes an
		// argument a month later, and "boshqa" is not a category anybody can
		// act on.
		httpx.Error(w, http.StatusBadRequest, "sababini yozing")
		return
	}
	branch := h.scopeBranch(r, scope)
	if err := h.requireBranchAccess(r, branch); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}

	now := time.Now()
	at := now
	if day, err := parseDay(req.At); err == nil {
		at = day
	}
	// ⚠️ Not into the future: a cost dated forward lands in a month nobody has
	// read yet and quietly changes it when they do. The delivery form draws the
	// same line.
	if at.After(now) {
		at = now
	}

	e := models.Expense{
		BranchID: branch, At: at,
		Category: category, Amount: req.Amount,
		Note:      clampText(strings.TrimSpace(req.Note), 200),
		Method:    expenseMethod(req.Method),
		CreatedBy: h.adminName(r), CreatedAt: now,
	}
	res, err := h.Store.Expenses.InsertOne(r.Context(), e)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	e.ID = oidOf(res.InsertedID)

	if req.FromSafe {
		h.recordSafeMovement(r.Context(), models.SafeEntry{
			BranchID: branch, Kind: models.SafeOut, Amount: e.Amount,
			At: e.At, Category: e.Category, Note: e.Note, By: e.CreatedBy,
			RefKind: models.SafeRefExpense, RefID: e.ID,
		})
	}

	h.logAction(r, "expense.create", "expense", e.ID.Hex(), e.Category,
		formatSum(e.Amount))
	httpx.JSON(w, http.StatusCreated, e)
}

// AdminDeleteExpense removes one entered by mistake.
func (h *Handler) AdminDeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var row models.Expense
	if err := h.Store.Expenses.FindOne(r.Context(), bson.M{"_id": id}).Decode(&row); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	if err := h.requireBranchAccess(r, row.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	if _, err := h.Store.Expenses.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ **The safe's row is left alone**, and that is deliberate: the money
	// physically left the box. Deleting the cost says "this was not an expense";
	// it does not say "the notes are back". Same reasoning as a deleted delivery
	// leaving the prices it wrote.
	h.logAction(r, "expense.delete", "expense", id.Hex(), row.Category,
		formatSum(row.Amount))
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func expenseMethod(m string) string {
	switch m {
	case models.PaidTransfer, models.PaidCard:
		return m
	}
	return models.PaidCash
}
