package handlers

// ---- Where the restaurant's cash is ----
//
// ⚠️ **A place, not a profit and loss.** Nothing here reaches the financial
// report: cash moved from the drawer into the safe is not an expense, and money
// handed to a buyer is not spent until it buys something. Counting a location's
// movements as outgoings is how a report subtracts the same money twice.
//
// ⚠️ **One movement, one row.** The other screens offer to write one when the
// money plainly came from or went into the safe, and the `refKind`/`refId` pair
// is unique — so a retry, or an owner tapping twice, cannot put the same
// hand-over in the ledger again. A duplicate in a balance is the worst kind of
// wrong: plausible, and invisible to everything downstream.

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// safeBalance is what the ledger adds up to.
//
// ⚠️ Aggregated in the database rather than loaded: a restaurant two years in
// has thousands of rows here, and this is read every time the screen opens.
func (h *Handler) safeBalance(ctx context.Context, scope bson.M) (models.SafeBalance, error) {
	var out models.SafeBalance
	cur, err := h.Store.SafeEntries.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: scope}},
		{{Key: "$group", Value: bson.M{
			"_id": "$kind", "sum": bson.M{"$sum": "$amount"},
			"last": bson.M{"$max": "$at"},
		}}},
	})
	if err != nil {
		return out, err
	}
	var rows []struct {
		Kind string    `bson:"_id"`
		Sum  int       `bson:"sum"`
		Last time.Time `bson:"last"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return out, err
	}
	for _, r := range rows {
		if r.Kind == models.SafeOut {
			out.Out += r.Sum
		} else {
			out.In += r.Sum
		}
		if out.LastAt == nil || r.Last.After(*out.LastAt) {
			when := r.Last
			out.LastAt = &when
		}
	}
	// ⚠️ **It can go below zero and that is shown, not clamped.** A negative
	// safe means the ledger is missing something that went in — an owner's own
	// money, a collection nobody recorded — and hiding it would leave the one
	// screen that could have said so agreeing with a count that cannot be right.
	out.Balance = out.In - out.Out
	return out, nil
}

// AdminSafe is the balance and the movements behind it.
func (h *Handler) AdminSafe(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := bson.M{}
	if branch := h.scopeBranch(r, scope); !branch.IsZero() {
		filter["branchId"] = branch
	}
	balance, err := h.safeBalance(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	cur, err := h.Store.SafeEntries.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	entries := []models.SafeEntry{}
	_ = cur.All(r.Context(), &entries)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"balance": balance, "entries": entries,
	})
}

// AdminCreateSafeEntry records money going into or out of the safe.
func (h *Handler) AdminCreateSafeEntry(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		Kind     string `json:"kind"`
		Category string `json:"category"`
		Amount   int    `json:"amount"`
		Note     string `json:"note"`
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
	if branch.IsZero() {
		// ⚠️ A safe belongs to a building. "The company's cash" spread across
		// three branches is a figure nobody can count, which is the same reason
		// the stock screens demand one branch.
		httpx.Error(w, http.StatusBadRequest, "filial tanlanmagan")
		return
	}
	if err := h.requireBranchAccess(r, branch); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}

	entry := models.SafeEntry{
		BranchID: branch,
		Kind:     safeKind(req.Kind),
		Category: clampText(strings.TrimSpace(req.Category), 60),
		Amount:   req.Amount,
		Note:     clampText(strings.TrimSpace(req.Note), 200),
		By:       h.adminName(r),
		At:       time.Now(),
	}
	entry.CreatedAt = entry.At
	res, err := h.Store.SafeEntries.InsertOne(r.Context(), entry)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	entry.ID = oidOf(res.InsertedID)

	// Money moving is written down with a name against it — the rule every
	// other cash action here follows.
	h.logAction(r, "safe.create", "safe", entry.ID.Hex(), entry.Category,
		safeWord(entry.Kind)+": "+formatSum(entry.Amount))

	balance, _ := h.safeBalance(r.Context(), bson.M{"branchId": branch})
	httpx.JSON(w, http.StatusCreated, map[string]any{
		"entry": entry, "balance": balance,
	})
}

// AdminDeleteSafeEntry removes a movement that was written by mistake.
//
// ⚠️ **Deleted rather than reversed, and journalled.** A counter-entry would be
// the tidier accounting answer and the wrong one here: a mistyped amount
// corrected with a second row leaves two figures that both look like real
// movements, and a month later nobody can tell a correction from a second trip
// to the safe. Same call the petty cash ledger makes.
func (h *Handler) AdminDeleteSafeEntry(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var row models.SafeEntry
	if err := h.Store.SafeEntries.FindOne(r.Context(), bson.M{"_id": id}).Decode(&row); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	if err := h.requireBranchAccess(r, row.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	if _, err := h.Store.SafeEntries.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, "safe.delete", "safe", id.Hex(), row.Category,
		safeWord(row.Kind)+": "+formatSum(row.Amount))
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// recordSafeMovement writes the safe's side of something that happened
// elsewhere.
//
// ⚠️ **Idempotent through the unique reference, not through the caller being
// careful.** Every caller is a handler that can be retried, and a duplicate row
// here is a wrong balance that looks exactly like a right one. A second attempt
// for the same document is a no-op rather than an error, because from the
// caller's point of view it succeeded.
//
// ⚠️ **It never fails the thing that triggered it.** An owner must not be
// unable to hand somebody their float because a second write did not land — the
// same rule the alert bell and the print queue follow. The row can be added by
// hand afterwards; the hand-over cannot be un-done.
func (h *Handler) recordSafeMovement(ctx context.Context, e models.SafeEntry) {
	if e.Amount <= 0 || e.RefKind == "" || e.RefID.IsZero() {
		return
	}
	e.Kind = safeKind(e.Kind)
	if e.At.IsZero() {
		e.At = time.Now()
	}
	e.CreatedAt = time.Now()
	if _, err := h.Store.SafeEntries.InsertOne(ctx, e); err != nil &&
		!mongo.IsDuplicateKeyError(err) {
		// Logged rather than returned: see the note above.
		log.Printf("safe movement: %v", err)
	}
}

func safeKind(kind string) string {
	if kind == models.SafeIn {
		return models.SafeIn
	}
	return models.SafeOut
}

func safeWord(kind string) string {
	if kind == models.SafeIn {
		return "seyfga kirim"
	}
	return "seyfdan chiqim"
}
