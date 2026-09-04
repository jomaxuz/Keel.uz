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

// ingredientView is one ingredient with what it works out to.
//
// The embedded document plus the derived figures: the panel needs both, and
// they are kept apart so nothing derived can be posted back and stored.
type ingredientView struct {
	models.Ingredient
	// Cost per gram / millilitre / piece. Fractional on purpose.
	Rate float64 `json:"rate"`
	// Made in-house from a card rather than bought.
	Made      bool `json:"made,omitempty"`
	BatchCost int  `json:"batchCost,omitempty"`
	Unpriced  bool `json:"unpriced,omitempty"`
	// What should be on the shelf, and whether that is under the minimum.
	//
	// ⚠️ **An estimate, and the screen has to say so.** It is the last count
	// plus deliveries less what the cards and the write-offs account for — it
	// drifts exactly as far as the kitchen drifts from its cards, and the
	// further away the last count is, the further it drifts. A "remaining"
	// column presented without that sentence is a number people order against.
	Expected float64 `json:"expected"`
	Low      bool    `json:"low,omitempty"`
	// ⚠️ Where **this branch** keeps it, sent back under the name the form
	// posts. The model's own field is `json:"-"` now: it is legacy, read only
	// by the migration, and letting it out beside this one would give the
	// screen two answers.
	WarehouseID string `json:"warehouseId,omitempty"`
}

// withPlacement answers a save with the store this branch keeps it in.
//
// ⚠️ The form posted a warehouse and has to be told what was stored, or an
// edit made while the lens spanned branches — where the store is deliberately
// not part of the edit — would leave the field looking saved.
func (h *Handler) withPlacement(
	r *http.Request, in models.Ingredient,
) ingredientView {
	out := ingredientView{Ingredient: in, Made: in.MadeInHouse()}
	if _, branch, _, err := h.stockBranch(r); err == nil && !branch.IsZero() {
		if store := h.placementsIn(r.Context(), branch)[in.ID]; !store.IsZero() {
			out.WarehouseID = store.Hex()
		}
	}
	return out
}

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

	// What each one costs per gram / millilitre / piece, prep items included.
	// ⚠️ Sent rather than computed on the screen: a prep item's rate depends on
	// every other rate, and a browser recomputing that chain is a second
	// implementation of the resolver — which would disagree with the reports on
	// exactly the cards that are hardest to check by hand.
	rates := h.ingredientRates(r.Context())
	// What should be there now, measured from the last count. ⚠️ Read once for
	// the whole list rather than per row: it walks the period's deliveries,
	// sales and write-offs, and doing that per ingredient would be the same
	// work forty times.
	expected := map[primitive.ObjectID]float64{}
	var countedAt *time.Time
	// ⚠️ Best-effort, and silent when the lens spans branches: this column is a
	// convenience on the ingredient list, not the stock screen. Refusing the
	// whole list because a chain has not picked a building would be a much
	// worse trade than showing it without the estimate.
	placed := map[primitive.ObjectID]primitive.ObjectID{}
	if scope, branch, brand, err := h.stockBranch(r); err == nil {
		if got, since, err := h.expectedStock(r, scope, brand, branch, time.Now()); err == nil {
			expected, countedAt = got, since
		}
		placed = h.placementsIn(r.Context(), branch)
	}
	out := make([]ingredientView, 0, len(rows))
	for _, in := range rows {
		v := ingredientView{Ingredient: in, Rate: rates[in.ID]}
		if store := placed[in.ID]; !store.IsZero() {
			v.WarehouseID = store.Hex()
		}
		v.Expected = round3(expected[in.ID])
		// ⚠️ Only when a minimum was set: zero means "do not warn me", and a
		// list where every line eventually turns red is a list nobody reads.
		v.Low = in.MinQty > 0 && v.Expected < in.MinQty
		if in.Recipe == nil {
			// ⚠️ Nil slices arrive as `null`, and the card editor maps over it.
			v.Recipe = []models.RecipeLine{}
		}
		if in.MadeInHouse() {
			v.Made = true
			// What one batch costs, so the screen can show the number somebody
			// can check against a pot rather than only a rate per gram.
			v.BatchCost = recipeCost(in.Recipe, rates)
			// ⚠️ A prep item whose own inputs are unpriced has no rate, and
			// the screen has to say so: silently showing zero would make every
			// dish containing it look cheap.
			v.Unpriced = rates[in.ID] == 0
		}
		out = append(out, v)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"ingredients": out,
		// When the expected figures were last anchored to a count. The screen
		// needs it to say how much of an estimate they are — and "never" is
		// the most important answer it can carry.
		"countedAt": countedAt,
	})
}

// AdminSaveIngredient creates or updates one.
func (h *Handler) AdminSaveIngredient(w http.ResponseWriter, r *http.Request) {
	// ⚠️ The store arrives beside the ingredient rather than on it, because it
	// is not a fact about the ingredient: the catalogue belongs to the brand
	// and the rooms belong to the branch. The form still shows one field, and
	// what it saves is "this branch keeps it here" — see models/placement.go.
	//
	// ⚠️ A pointer, so "not sent" and "the undivided store" stay different
	// answers: an older tab, or a script, posting a dish without the field must
	// not silently move it out of the store somebody filed it in.
	var body struct {
		models.Ingredient
		WarehouseID *string `json:"warehouseId"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	in := body.Ingredient
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
	if in.MinQty < 0 {
		in.MinQty = 0
	}
	in.Note = clampText(in.Note, 120)
	// ⚠️ **Both halves or neither.** A packaging name with no size converts
	// nothing, and a size with no name would multiply a buyer's quantity by a
	// factor nobody can see on screen — which is the failure the field exists to
	// prevent, arriving through the form that configures it.
	in.PackName = clampText(strings.TrimSpace(in.PackName), 20)
	if in.PackQty < 0 {
		in.PackQty = 0
	}
	if in.PackName == "" || in.PackQty <= 0 {
		in.PackName, in.PackQty = "", 0
	}
	in.Recipe = normalizeRecipe(in.Recipe)
	if in.Output < 0 {
		in.Output = 0
	}
	// ⚠️ A prep item's price is not typed — it is what its batch costs. Keeping
	// an old typed figure beside a card would leave two answers on one
	// document, and the stale one would be the one that looks authoritative.
	if len(in.Recipe) > 0 && in.Output > 0 {
		in.Price = 0
	}
	in.UpdatedAt = time.Now()
	// ⚠️ **An owner saving this form is what finishes a half-record**, and it is
	// the only thing that does. A row invented at a market has a name, a price
	// and nothing else — no unit anybody chose, no minimum, no store, no card —
	// and it is flagged so the catalogue says so out loud. Clearing the flag on
	// a timer, or on any read, would quietly promote it to a finished record on
	// a date nobody picked: the stale-flag bug this codebase has already paid
	// for twice.
	in.NeedsCare = false

	if id, err := objectID(chi.URLParam(r, "id")); err == nil && !id.IsZero() {
		in.ID = id
		// ⚠️ **Kept, because this is a whole-document replace and the form does
		// not send it.** Without this an edit as small as fixing a typo in a
		// name moved the ingredient out of its brand — and out of every screen
		// that reads through the brand lens. The catalogue row stayed in the
		// database, the dish cards kept pointing at it by id, and it simply
		// stopped appearing on the list somebody was about to count.
		in.BrandID = h.keepBrandID(r, h.Store.Ingredients, id, in.BrandID)
		in.History = h.priceHistoryFor(r.Context(), id, in)
		if _, err := h.Store.Ingredients.ReplaceOne(r.Context(),
			bson.M{"_id": id}, in); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		h.placeIngredient(r, id, body.WarehouseID)
		h.logAction(r, "ingredient.update", "ingredient", id.Hex(), in.Name, "")
		httpx.JSON(w, http.StatusOK, h.withPlacement(r, in))
		return
	}

	if scope, err := h.adminScope(r); err == nil && in.BrandID.IsZero() {
		in.BrandID = h.scopeBrand(r, scope)
	}
	// ⚠️ The first price is history from the day it is entered, not from the
	// beginning of time — but it is also the only price we know, so PriceAt
	// reaches back with it. Both facts have to be true at once: the report for
	// last month has to cost this ingredient at *something*, and the something
	// must not pretend to be a measurement.
	if in.Price > 0 {
		in.History = []models.PriceEntry{{Price: in.Price, At: in.UpdatedAt}}
	}
	res, err := h.Store.Ingredients.InsertOne(r.Context(), in)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	in.ID = oidOf(res.InsertedID)
	h.placeIngredient(r, in.ID, body.WarehouseID)
	h.logAction(r, "ingredient.create", "ingredient", in.ID.Hex(), in.Name, "")
	httpx.JSON(w, http.StatusCreated, h.withPlacement(r, in))
}

// placeIngredient files one ingredient into one of this branch's stores.
//
// ⚠️ **Silent when the lens spans branches.** Saving an ingredient is a
// catalogue edit and must not be refused because a chain has not picked a
// building; the store simply is not part of that edit. The stock screens, which
// are the ones the answer matters to, refuse instead.
func (h *Handler) placeIngredient(
	r *http.Request, id primitive.ObjectID, warehouse *string,
) {
	if warehouse == nil {
		return
	}
	_, branch, _, err := h.stockBranch(r)
	if err != nil || branch.IsZero() {
		return
	}
	store, err := optionalObjectID(*warehouse)
	if err != nil {
		return
	}
	// Not this branch's room: filing it there would put the shelf where nobody
	// can walk to it, and every screen afterwards would look consistent.
	if !store.IsZero() {
		if err := h.Store.Warehouses.FindOne(r.Context(),
			bson.M{"_id": store, "branchId": branch}).Err(); err != nil {
			return
		}
	}
	_, _ = h.Store.Placements.UpdateOne(r.Context(),
		bson.M{"branchId": branch, "ingredientId": id},
		bson.M{"$set": bson.M{"warehouseId": store, "updatedAt": time.Now()},
			"$setOnInsert": bson.M{"branchId": branch, "ingredientId": id}},
		options.Update().SetUpsert(true),
	)
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

// scopedIngredients reads the ingredients of the brand in view.
//
// ⚠️ **The list endpoint was scoped and every computation was not.** Costing,
// balances, counts, the flow report and the stop list all read `bson.M{}`, so a
// manager pinned to one brand saw the other brand's ingredients and its buying
// prices on the stock screen. The panel's own rule states it: the boundary is a
// **filter**, not a check — a list that is trimmed after it is read leaks the
// day somebody adds a count, an aggregate or an export beside it.
//
// ⚠️ A zero brand means "no lens", which is what a single-brand install always
// produces: byte for byte the query it ran before, and that customer sees
// nothing new.
//
// ⚠️ Deliberately **not** applied to `ingredientRates`: that map is keyed by id
// and only ever read by id, so a foreign entry in it is unreachable rather than
// disclosed — and narrowing it would make a prep card stop resolving the moment
// its ingredient sat in another brand.
func (h *Handler) scopedIngredients(
	ctx context.Context, brand primitive.ObjectID,
) []models.Ingredient {
	filter := bson.M{}
	if !brand.IsZero() {
		filter["brandId"] = brand
	}
	var rows []models.Ingredient
	if cur, err := h.Store.Ingredients.Find(ctx, filter); err == nil {
		_ = cur.All(ctx, &rows)
	}
	return rows
}

// ingredientRates reads every ingredient's cost per recipe unit.
//
// ⚠️ **Bought things first, then whatever can be worked out from them.** A prep
// item — a sauce, a stock, a dough — has no typed price: its rate is what one
// batch costs divided by what the batch yields, and the batch may itself
// contain another prep item. So the map is filled in passes, each one costing
// every card whose inputs are now all known.
//
// ⚠️ **Bounded, because two cards can name each other.** Nothing stops somebody
// putting the sauce in the dough and the dough in the sauce; a resolver that
// recursed would hang the panel, and one that "handled" it by costing the
// missing side at zero would quietly underprice both. After the passes stop
// making progress, whatever is left is simply absent — an unpriced ingredient,
// which every screen downstream already knows how to say something honest
// about (an incomplete card does not cost its dish).
func (h *Handler) ingredientRates(ctx context.Context) map[primitive.ObjectID]float64 {
	var rows []models.Ingredient
	cur, err := h.Store.Ingredients.Find(ctx, bson.M{})
	if err == nil {
		_ = cur.All(ctx, &rows)
	}
	return ratesAt(rows, time.Now())
}

// ratesAt is the same resolution as of a given day — what everything cost then.
//
// ⚠️ Split out from the read so a report can ask it once per day of the period
// without going back to the database, and so the resolution itself is testable
// against a hand-built list.
func ratesAt(rows []models.Ingredient, at time.Time) map[primitive.ObjectID]float64 {
	out := map[primitive.ObjectID]float64{}
	var made []models.Ingredient
	for _, in := range rows {
		if in.MadeInHouse() {
			made = append(made, in)
			continue
		}
		// ⚠️ The price **as of that day**, not today's: raising a price must
		// not rewrite a month somebody has already read.
		priced := in
		priced.Price = in.PriceAt(at)
		if rate := priced.CostPerRecipeUnit(); rate > 0 {
			out[in.ID] = rate
		}
	}
	// Each pass resolves at least one prep item or nothing further can be
	// resolved at all, so this cannot run longer than the number of them.
	for range made {
		progress := false
		for _, in := range made {
			if _, done := out[in.ID]; done {
				continue
			}
			if !recipeComplete(in.Recipe, out) {
				continue
			}
			batch := 0.0
			for _, l := range in.Recipe {
				batch += out[l.IngredientID] * l.Qty
			}
			if batch > 0 && in.Output > 0 {
				out[in.ID] = batch / in.Output
				progress = true
			}
		}
		if !progress {
			break
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

// priceHistoryFor decides what the stored history becomes after an edit.
//
// ⚠️ **An edit is always "from today", never a correction of the past.** The
// form cannot tell the two apart — "we typed it wrong" and "beef went up" look
// identical — and treating every edit as retroactive is what silently rewrites
// a month somebody has already read. Whoever really needs to fix a past figure
// needs a screen that says so, and it does not exist yet.
func (h *Handler) priceHistoryFor(
	ctx context.Context, id primitive.ObjectID, next models.Ingredient,
) []models.PriceEntry {
	var stored models.Ingredient
	if err := h.Store.Ingredients.FindOne(ctx, bson.M{"_id": id}).Decode(&stored); err != nil {
		if next.Price > 0 {
			return []models.PriceEntry{{Price: next.Price, At: next.UpdatedAt}}
		}
		return nil
	}
	hist := stored.History
	// A prep item has no price of its own; its history stops where it stopped.
	if next.MadeInHouse() {
		return hist
	}
	if len(hist) == 0 {
		if next.Price > 0 {
			return []models.PriceEntry{{Price: next.Price, At: next.UpdatedAt}}
		}
		return nil
	}
	if hist[len(hist)-1].Price == next.Price {
		// Nothing about the money changed — renaming an ingredient must not
		// leave a price "change" in the history for somebody to explain.
		return hist
	}
	return append(hist, models.PriceEntry{Price: next.Price, At: next.UpdatedAt})
}
