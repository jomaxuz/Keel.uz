package handlers

import (
	"context"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- Ingredients, and the cost that follows from them ----
//
// ⚠️ **A typed cost is correct on the day it is typed.** That was the whole
// weakness of the field this replaces: the number was right in March and
// silently wrong from the next delivery, and nothing on any screen said so. A
// tech card moves the fact to where it actually changes — meat goes up once,
// and every dish containing meat is dearer the same afternoon.
//
// ⚠️ **Costing, not stock.** Nothing here knows what is in the fridge. A
// restaurant that believes a stock figure and finds it wrong stops believing
// the panel entirely, so the quantities here are only ever "what leaves the
// store to make this dish" — which is exactly what it costs.

// AdminListIngredients returns the shopping list, cheapest lookup first.
func (h *Handler) AdminListIngredients(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	cur, err := h.Store.Ingredients.Find(r.Context(), scope.brandFilter(bson.M{}),
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var rows []models.Ingredient
	_ = cur.All(r.Context(), &rows)
	if rows == nil {
		// ⚠️ Nil slices arrive as `null`, and the screen maps over this.
		rows = []models.Ingredient{}
	}
	httpx.JSON(w, http.StatusOK, rows)
}

// AdminSaveIngredient creates or updates one.
func (h *Handler) AdminSaveIngredient(w http.ResponseWriter, r *http.Request) {
	var in models.Ingredient
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "nomini yozing")
		return
	}
	// ⚠️ An unknown unit becomes pieces rather than being refused. The three
	// families are a closed list precisely so a recipe can always be costed;
	// a fourth spelling arriving from an old tab must not be stored, because
	// `PerUnit` would then divide by a unit nobody can convert.
	switch in.Unit {
	case models.UnitKg, models.UnitL, models.UnitPcs:
	default:
		in.Unit = models.UnitPcs
	}
	if in.Price < 0 {
		in.Price = 0
	}
	in.Note = clampText(in.Note, 120)
	in.UpdatedAt = time.Now()

	if id, err := objectID(chi.URLParam(r, "id")); err == nil && !id.IsZero() {
		in.ID = id
		if _, err := h.Store.Ingredients.ReplaceOne(r.Context(),
			bson.M{"_id": id}, in); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		h.logAction(r, "ingredient.update", "ingredient", id.Hex(), in.Name, "")
		httpx.JSON(w, http.StatusOK, in)
		return
	}

	if scope, err := h.adminScope(r); err == nil && in.BrandID.IsZero() {
		in.BrandID = h.scopeBrand(r, scope)
	}
	res, err := h.Store.Ingredients.InsertOne(r.Context(), in)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	in.ID = oidOf(res.InsertedID)
	h.logAction(r, "ingredient.create", "ingredient", in.ID.Hex(), in.Name, "")
	httpx.JSON(w, http.StatusCreated, in)
}

// AdminDeleteIngredient removes one, unless a dish still uses it.
//
// ⚠️ **Refused rather than cascaded.** A recipe line pointing at a deleted
// ingredient would cost nothing at all, which does not look like an error: the
// dish simply becomes cheaper to make, on every report, for as long as nobody
// notices. Saying which dishes hold it turns a refusal into an instruction.
func (h *Handler) AdminDeleteIngredient(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	cur, err := h.Store.Menu.Find(r.Context(), bson.M{"recipe.ingredientId": id},
		options.Find().SetLimit(5))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var used []models.MenuItem
	_ = cur.All(r.Context(), &used)
	if len(used) > 0 {
		names := make([]string, 0, len(used))
		for _, m := range used {
			names = append(names, m.Name)
		}
		httpx.Error(w, http.StatusConflict,
			"bu masalliq texkartada ishlatilmoqda: "+strings.Join(names, ", "))
		return
	}
	if _, err := h.Store.Ingredients.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, "ingredient.delete", "ingredient", id.Hex(), "", "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- The roll-up ----

// recipeCost is what one portion costs, from the card.
//
// ⚠️ **Rounded once, at the end.** A gram of anything is well under a som, so
// rounding each line would cost most dishes at zero — the arithmetic is carried
// as a rate and turned into money exactly once.
func recipeCost(lines []models.RecipeLine, prices map[primitive.ObjectID]float64) int {
	total := 0.0
	for _, l := range lines {
		rate, ok := prices[l.IngredientID]
		if !ok {
			// ⚠️ A missing ingredient contributes nothing and the caller is
			// told the card is incomplete — see recipeComplete. Silently
			// costing it at zero would make a dish look cheaper the moment
			// somebody deletes something it depends on.
			continue
		}
		total += rate * l.Qty
	}
	return int(math.Round(total))
}

// recipeComplete reports whether every line still points at a real ingredient.
func recipeComplete(lines []models.RecipeLine, prices map[primitive.ObjectID]float64) bool {
	for _, l := range lines {
		if _, ok := prices[l.IngredientID]; !ok {
			return false
		}
	}
	return len(lines) > 0
}

// ingredientRates reads every ingredient's cost per recipe unit.
func (h *Handler) ingredientRates(ctx context.Context) map[primitive.ObjectID]float64 {
	out := map[primitive.ObjectID]float64{}
	cur, err := h.Store.Ingredients.Find(ctx, bson.M{})
	if err != nil {
		return out
	}
	var rows []models.Ingredient
	if err := cur.All(ctx, &rows); err != nil {
		return out
	}
	for _, in := range rows {
		if rate := in.CostPerRecipeUnit(); rate > 0 {
			out[in.ID] = rate
		}
	}
	return out
}

// normalizeRecipe cleans a card arriving from the form.
//
// ⚠️ **A line with no quantity is dropped, not stored as zero.** A zero-gram
// ingredient costs nothing and looks like a considered decision on the card;
// it is always a half-finished row somebody left behind, and it makes the
// difference between "this card is complete" and "this card is nearly right"
// impossible to see.
func normalizeRecipe(lines []models.RecipeLine) []models.RecipeLine {
	out := make([]models.RecipeLine, 0, len(lines))
	seen := map[primitive.ObjectID]int{}
	for _, l := range lines {
		if l.IngredientID.IsZero() || l.Qty <= 0 {
			continue
		}
		// The same ingredient twice is one line: a card listing "beef 100 g"
		// and "beef 50 g" is a card nobody can check against a plate.
		if at, ok := seen[l.IngredientID]; ok {
			out[at].Qty += l.Qty
			continue
		}
		seen[l.IngredientID] = len(out)
		out = append(out, l)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
