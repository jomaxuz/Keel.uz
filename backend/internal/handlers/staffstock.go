package handlers

import (
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- Counting the store from a phone ----
//
// ⚠️ **The count happened in the store and was typed in the office.** Somebody
// walked the shelves with a clipboard, then carried the paper to a computer and
// entered forty numbers a second time. A number written twice is a number that
// is wrong the second time, and the error lands in the one figure this whole
// module exists to produce — the variance — where it is indistinguishable from
// a shortfall.
//
// ⚠️ **Its own permission** (`PermStock`), because a counting screen is not a
// small thing to hand to a shared tablet: it writes the baseline every later
// shortfall is measured from, so a saved count silently forgives whatever went
// missing before it. Unlike the pass screen, nothing needs grandfathering —
// this screen is new, so refusing by default takes nothing from anybody.
//
// ⚠️ **The branch comes off the employee, never the request** — the KDS rule.
// The brand comes off the branch, so the sheet holds this kitchen's ingredients
// and no other's.

// stockDenial says why this employee may not count, or "" if they may.
//
// Two refusals with different words, because they send the person somewhere
// different: one to their manager, one to whoever switched the account off.
func stockDenial(s models.Staff) string {
	if !s.IsActive {
		return "hisob o'chirilgan — ma'muriyat bilan bog'laning"
	}
	if !s.Can(models.PermStock) {
		return "omborni sanashga ruxsat berilmagan — administratorga murojaat qiling"
	}
	return ""
}

// staffStockScope resolves the employee into the lens their screens read
// through: their own branch, and the brand that branch belongs to.
func (h *Handler) staffStockScope(
	r *http.Request, s models.Staff,
) (bson.M, primitive.ObjectID) {
	scope := bson.M{"branchId": s.BranchID}
	var branch models.Branch
	if err := h.Store.Branches.FindOne(r.Context(),
		bson.M{"_id": s.BranchID}).Decode(&branch); err != nil {
		return scope, primitive.NilObjectID
	}
	return scope, branch.BrandID
}

// staffStockGuard is the three checks every endpoint below starts with.
func (h *Handler) staffStockGuard(
	w http.ResponseWriter, r *http.Request,
) (models.Staff, bool) {
	s, ok := h.staffFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return s, false
	}
	if why := stockDenial(s); why != "" {
		// 403 rather than 404, unlike the branch-scope refusals: this person
		// works here and can see the button is missing, so pretending the
		// screen does not exist would only be confusing.
		httpx.Error(w, http.StatusForbidden, why)
		return s, false
	}
	return s, true
}

// StaffWarehouses lists the stores of this employee's branch.
//
// ⚠️ Needed before the sheet: a count is one room, and the phone has to ask
// which one before it can hand over a list.
func (h *Handler) StaffWarehouses(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffStockGuard(w, r)
	if !ok {
		return
	}
	cur, err := h.Store.Warehouses.Find(r.Context(),
		bson.M{"branchId": s.BranchID, "isActive": true},
		options.Find().SetSort(bson.D{{Key: "sort", Value: 1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.Warehouse{}
	_ = cur.All(r.Context(), &rows)
	httpx.JSON(w, http.StatusOK, map[string]any{"warehouses": rows})
}

// StaffStocktakeSheet is what to count, for this employee's branch.
func (h *Handler) StaffStocktakeSheet(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffStockGuard(w, r)
	if !ok {
		return
	}
	scope, brand := h.staffStockScope(r, s)
	h.stocktakeSheet(w, r, scope, brand, s.BranchID)
}

// StaffSaveStocktake records a count taken on a phone.
//
// ⚠️ **The same function the panel saves through.** The expected figure is
// frozen by the server, the variance is stored, and a difference cannot be
// saved without a sentence — all three have to be one implementation, or the
// count "saves differently on the tablet" and the argument moves from the shelf
// to our two screens.
func (h *Handler) StaffSaveStocktake(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffStockGuard(w, r)
	if !ok {
		return
	}
	var in models.Stocktake
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	scope, brand := h.staffStockScope(r, s)
	// ⚠️ The branch is taken off the employee and the posted one discarded:
	// a phone that could name a branch could count somebody else's store, and
	// a count is the one write that resets a baseline.
	in.BranchID = s.BranchID
	in.At = time.Now()
	h.saveStocktake(w, r, in, scope, brand, s.BranchID, s.Name)
}
