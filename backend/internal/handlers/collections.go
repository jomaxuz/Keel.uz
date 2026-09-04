package handlers

// ---- Inkassatsiya: the day the cash leaves the building ----
//
// ⚠️ **The one cash event with an outside deadline.** The rules for cash
// operations (Правила ведения кассовых операций, ст. 7) require every so'm
// above the limit agreed with the bank to be handed over for crediting; only
// wages may stay behind, and only for three working days (ст. 8). So the
// contents of the safe stop being an owner's private business at a threshold,
// and the system that knows the number is the one that can say so.
//
// ⚠️ **Not a cost.** Money going to the bank has not been spent — it moved from
// a box to an account. Counting a collection as an outgoing would subtract the
// restaurant's own takings from itself, the same mistake the courier settlement
// line exists to avoid.
//
// ⚠️ **The reconciliation is frozen onto the document.** What each shift
// counted, what it was expected to hold, what the safe held — all written at
// the moment of the handover. Recomputed later it would answer differently
// every time an old shift is corrected, and "did what left match what was
// there?" would quietly change its mind months after everybody signed.

import (
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// AdminCollections lists the handovers, newest first, with what is due now.
func (h *Handler) AdminCollections(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	cur, err := h.Store.Collections.Find(r.Context(), scopeFilter(scope),
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(100))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.Collection{}
	_ = cur.All(r.Context(), &rows)

	// What a collection made right now would cover — the same figures the
	// document will freeze, shown before anybody signs.
	//
	// ⚠️ **Computed by the same function that writes them.** Two code paths for
	// "what is due" is one too many: the preview and the document would drift,
	// and the one on screen is the one somebody counts against.
	due := h.collectionDraft(r, scope)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"collections": rows,
		"due":         due,
	})
}

// collectionDraft is the window since the last handover, and what it holds.
func (h *Handler) collectionDraft(r *http.Request, scope bson.M) models.Collection {
	ctx := r.Context()
	draft := models.Collection{At: time.Now()}

	// Where the window starts: the last collection, or the beginning.
	var last models.Collection
	err := h.Store.Collections.FindOne(ctx, scopeFilter(scope),
		options.FindOne().SetSort(bson.D{{Key: "at", Value: -1}})).Decode(&last)
	if err == nil {
		draft.FromAt = last.At
	}

	// The Z reports of the window. ⚠️ **Closed shifts only**: an open drawer
	// has not been counted, and folding in a figure nobody has verified is how
	// a handover comes to be checked against an estimate.
	filter := scopeFilter(scope)
	closed := bson.M{"$exists": true, "$ne": nil}
	if !draft.FromAt.IsZero() {
		closed["$gt"] = draft.FromAt
	}
	filter["closedAt"] = closed
	cur, err := h.Store.CashShifts.Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "closedAt", Value: 1}}).SetLimit(200))
	if err == nil {
		var shifts []models.CashShift
		if err := cur.All(ctx, &shifts); err == nil {
			for i := range shifts {
				s := shifts[i]
				draft.ShiftIDs = append(draft.ShiftIDs, s.ID)
				draft.Counted += s.Counted
				draft.Expected += s.Expected
				draft.Variance += s.Variance
			}
			draft.Shifts = len(shifts)
		}
	}

	if bal, err := h.safeBalance(ctx, scopeFilter(scope)); err == nil {
		draft.SafeBefore = bal.Balance
	}
	return draft
}

// AdminCreateCollection records one handover.
func (h *Handler) AdminCreateCollection(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		Amount  int    `json:"amount"`
		To      string `json:"to"`
		TakenBy string `json:"takenBy"`
		Bag     string `json:"bag"`
		Note    string `json:"note"`
		// Whether the notes came out of the office box. ⚠️ Asked like every
		// other safe movement: a branch that carries the drawer straight to the
		// bank never touches the safe, and a ledger that assumed otherwise
		// would empty a box nobody opened.
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
	branch := h.scopeBranch(r, scope)
	// ⚠️ Scoped to one branch on purpose. A handover happens at a door, with a
	// bag, from one drawer — "the company collected 14 000 000" is a number
	// nobody can hand to anybody. A single-branch restaurant never sees this
	// question: its branch resolves itself, exactly as the stock screens do.
	if branch.IsZero() {
		only, err := h.onlyBranch(r, scope.BrandID)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, errPickBranch.Error())
			return
		}
		branch = only
	}
	if err := h.requireBranchAccess(r, branch); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	one := bson.M{"branchId": branch}

	c := h.collectionDraft(r, one)
	c.BranchID = branch
	c.Amount = req.Amount
	c.To = models.CollectionToBank
	if req.To == models.CollectionToSafe {
		c.To = models.CollectionToSafe
	}
	c.TakenBy = clampText(strings.TrimSpace(req.TakenBy), 80)
	c.Bag = clampText(strings.TrimSpace(req.Bag), 40)
	c.Note = clampText(strings.TrimSpace(req.Note), 200)
	c.By = h.adminName(r)
	c.CreatedAt = time.Now()
	// ⚠️ Written even when it is zero: a document that only carried a
	// difference when there was one would leave "checked and correct"
	// indistinguishable from "nobody checked".
	c.Diff = c.Amount - c.Counted

	res, err := h.Store.Collections.InsertOne(r.Context(), c)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	c.ID = oidOf(res.InsertedID)

	if req.FromSafe {
		h.recordSafeMovement(r.Context(), models.SafeEntry{
			BranchID: branch, Kind: models.SafeOut, Amount: c.Amount,
			At: c.At, Category: "inkassatsiya", Note: c.TakenBy, By: c.By,
			RefKind: models.SafeRefCollection, RefID: c.ID,
		})
	}

	h.logAction(r, "collection.create", "collection", c.ID.Hex(), c.To,
		formatSum(c.Amount))
	httpx.JSON(w, http.StatusCreated, c)
}
