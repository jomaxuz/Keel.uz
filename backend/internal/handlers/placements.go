package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- Where each ingredient is kept, in one branch ----
//
// See models/placement.go for why this is a row of its own rather than a field
// on the ingredient.

// errPickBranch is what every stock screen answers when the lens spans more
// than one building.
//
// ⚠️ **Stock is a per-branch fact and there is no useful sum of it.** "The
// company holds nine kilos of beef" is spread over three fridges in three
// districts and answers no question anybody asks: it cannot be counted, cannot
// be ordered against, and cannot be cooked from. Before this, the arithmetic
// happily produced that number anyway, filed against whichever branch's store
// the ingredient happened to name. Refusing is the honest answer, and a
// single-branch restaurant never sees it — the branch is resolved for them.
var errPickBranch = errors.New("omborni ko'rish uchun filialni tanlang")

// stockBranch resolves the one branch a stock screen is about.
//
// ⚠️ Not `orderScope`: that one is built to span branches, which is right for
// orders (a company-wide list of tonight's tickets is useful) and wrong for
// shelves. Here the ambiguity has to become an error rather than a silent
// merge.
func (h *Handler) stockBranch(r *http.Request) (bson.M, primitive.ObjectID, primitive.ObjectID, error) {
	sc, err := h.adminScope(r)
	if err != nil {
		return nil, primitive.NilObjectID, primitive.NilObjectID, err
	}
	branch := sc.BranchID
	if branch.IsZero() {
		// One branch means no choice to make, which is the whole point of the
		// single-branch install seeing none of this.
		b, err := h.onlyBranch(r, sc.BrandID)
		if err != nil {
			return nil, primitive.NilObjectID, primitive.NilObjectID, err
		}
		branch = b
	}
	brand := sc.BrandID
	if brand.IsZero() {
		var row models.Branch
		if err := h.Store.Branches.FindOne(r.Context(),
			bson.M{"_id": branch}).Decode(&row); err == nil {
			brand = row.BrandID
		}
	}
	return bson.M{"branchId": branch}, branch, brand, nil
}

// onlyBranch is the install's single branch, or an error asking for a choice.
func (h *Handler) onlyBranch(
	r *http.Request, brand primitive.ObjectID,
) (primitive.ObjectID, error) {
	filter := bson.M{}
	if !brand.IsZero() {
		filter["brandId"] = brand
	}
	cur, err := h.Store.Branches.Find(r.Context(), filter)
	if err != nil {
		return primitive.NilObjectID, err
	}
	var rows []models.Branch
	if err := cur.All(r.Context(), &rows); err != nil {
		return primitive.NilObjectID, err
	}
	if len(rows) == 1 {
		return rows[0].ID, nil
	}
	// ⚠️ No branches at all is not an error: a fresh install has an ingredient
	// list before it has a branch, and the undivided store answers for it.
	if len(rows) == 0 {
		return primitive.NilObjectID, nil
	}
	return primitive.NilObjectID, errPickBranch
}

// placementsIn is where this branch keeps each ingredient.
//
// ⚠️ A missing entry is the undivided store, not an error — every install that
// never split its stores, and every ingredient added since.
func (h *Handler) placementsIn(
	ctx context.Context, branch primitive.ObjectID,
) map[primitive.ObjectID]primitive.ObjectID {
	out := map[primitive.ObjectID]primitive.ObjectID{}
	cur, err := h.Store.Placements.Find(ctx, bson.M{"branchId": branch})
	if err != nil {
		return out
	}
	var rows []models.IngredientPlacement
	if err := cur.All(ctx, &rows); err != nil {
		return out
	}
	for _, p := range rows {
		out[p.IngredientID] = p.WarehouseID
	}
	return out
}

// AdminSetPlacement files one ingredient into one of this branch's stores.
//
// ⚠️ **Upserted on (branch, ingredient)**, which the unique index also
// enforces: two rows for one shelf would make `FindOne` pick one of them, and
// the symptom is "the store I chose changed back by itself".
func (h *Handler) AdminSetPlacement(w http.ResponseWriter, r *http.Request) {
	_, branch, brand, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		IngredientID string `json:"ingredientId"`
		WarehouseID  string `json:"warehouseId"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ing, err := objectID(req.IngredientID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "masalliq tanlanmagan")
		return
	}
	// The brand inside the filter: an id alone never selects a document.
	if err := h.Store.Ingredients.FindOne(r.Context(),
		Scope{BrandID: brand}.brandFilter(bson.M{"_id": ing})).Err(); err != nil {
		httpx.Error(w, http.StatusNotFound, "masalliq topilmadi")
		return
	}
	store, err := optionalObjectID(req.WarehouseID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "ombor noto'g'ri")
		return
	}
	// ⚠️ And the store has to be **this branch's**. A warehouse id from another
	// building would file the shelf where nobody can walk to it, and every
	// screen afterwards would look consistent.
	if !store.IsZero() {
		if err := h.Store.Warehouses.FindOne(r.Context(),
			bson.M{"_id": store, "branchId": branch}).Err(); err != nil {
			httpx.Error(w, http.StatusBadRequest, "bu filialda bunday ombor yo'q")
			return
		}
	}
	if branch.IsZero() {
		httpx.Error(w, http.StatusBadRequest, errPickBranch.Error())
		return
	}
	if _, err := h.Store.Placements.UpdateOne(r.Context(),
		bson.M{"branchId": branch, "ingredientId": ing},
		bson.M{"$set": bson.M{
			"warehouseId": store, "updatedAt": time.Now(),
		}, "$setOnInsert": bson.M{"branchId": branch, "ingredientId": ing}},
		options.Update().SetUpsert(true),
	); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, "ingredient.place", "ingredient", ing.Hex(), "", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
