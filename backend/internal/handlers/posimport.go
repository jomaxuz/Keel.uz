package handlers

// ---- Moving in from another till system ----
//
// A restaurant running iiko, r_keeper, Clopos, Poster or Jowi does not stay
// because it prefers the software. It stays because the ingredient list, the
// tech cards and the stock are in there and moving them by hand is weeks of
// somebody's time. That objection — "hamma narsa boshqa POS'da" — is the real
// one, and it is the last thing between a signed customer and a live one.
//
// ⚠️ **A file, not an integration.** None of these systems documents an API for
// reading tech cards, and a restaurant on its way out has usually lost its API
// access anyway — the licence is the reseller's. Every one of them exports to
// Excel. Guessing a wire format here would be worse than useless: an import
// that puts the quantities in the wrong unit is a food cost that looks computed
// and is a thousand times wrong.
//
// ⚠️ **Two steps, like the menu importer, and for a sharper reason.** A wrong
// tech card is not visibly wrong: every dish still has a cost, the reports
// still add up, and the number is discovered at a stocktake weeks later.

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/menuimport"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/posimport"
)

const maxImportFile = 8 << 20

// AdminPosImportPreview reads an uploaded export and proposes what is in it.
func (h *Handler) AdminPosImportPreview(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	if err := r.ParseMultipartForm(maxImportFile); err != nil {
		httpx.Error(w, http.StatusBadRequest, "fayl juda katta")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "fayl yuborilmadi")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxImportFile))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	kind := r.FormValue("kind")
	sheet, err := posimport.Read(header.Filename, data)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	cols := posimport.Detect(sheet.Header)
	// ⚠️ The owner may have already corrected the mapping and pressed again;
	// their choice wins over the guess, per field, so fixing one column does
	// not throw away the four that were right.
	for _, f := range []string{
		posimport.FieldName, posimport.FieldUnit, posimport.FieldPrice,
		posimport.FieldQty, posimport.FieldCategory, posimport.FieldDish,
	} {
		if v := strings.TrimSpace(r.FormValue("col_" + f)); v != "" {
			if i, err := strconv.Atoi(v); err == nil && i >= 0 && i < len(sheet.Header) {
				cols[f] = i
			}
		}
	}

	lines := posimport.Build(kind, sheet, cols)
	h.markKnown(r, kind, lines)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"header":  sheet.Header,
		"columns": cols,
		"lines":   lines,
		"rows":    len(sheet.Rows),
	})
}

// markKnown flags the lines whose name is already in the catalogue.
//
// ⚠️ Names are compared normalised (menuimport.NormalName), because an export
// writes `Lagʻmon` where the panel has `Lag'mon` — the same apostrophe problem
// the menu importer already pays for, and here it would produce a second
// ingredient with the same name, half the recipes pointing at each.
func (h *Handler) markKnown(r *http.Request, kind string, lines []posimport.Line) {
	ctx := r.Context()
	_, _, brand, err := h.stockBranch(r)
	if err != nil {
		return
	}
	known := map[string]bool{}
	for _, ing := range h.scopedIngredients(ctx, brand) {
		known[menuimport.NormalName(ing.Name)] = true
	}
	for i := range lines {
		if kind == posimport.KindIngredients {
			lines[i].Exists = known[menuimport.NormalName(lines[i].Name)]
		}
	}
}

// AdminPosImportApply writes what the owner ticked.
func (h *Handler) AdminPosImportApply(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req struct {
		Kind  string           `json:"kind"`
		Lines []posimport.Line `json:"lines"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Lines) == 0 {
		httpx.Error(w, http.StatusBadRequest, "hech nima tanlanmagan")
		return
	}

	switch req.Kind {
	case posimport.KindIngredients:
		h.applyIngredients(w, r, req.Lines)
	case posimport.KindRecipes:
		h.applyRecipes(w, r, req.Lines)
	case posimport.KindStock:
		h.applyStock(w, r, req.Lines)
	default:
		httpx.Error(w, http.StatusBadRequest, "import turi noma'lum")
	}
}

// ---- The ingredient list ----

func (h *Handler) applyIngredients(
	w http.ResponseWriter, r *http.Request, lines []posimport.Line,
) {
	ctx := r.Context()
	_, _, brand, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	taken := map[string]bool{}
	for _, ing := range h.scopedIngredients(ctx, brand) {
		taken[menuimport.NormalName(ing.Name)] = true
	}

	now := time.Now()
	created, skipped := 0, 0
	for _, l := range lines {
		name := strings.TrimSpace(l.Name)
		key := menuimport.NormalName(name)
		if !l.OK() || key == "" || taken[key] {
			skipped++
			continue
		}
		taken[key] = true

		price := l.Price
		// ⚠️ **The price beside a gram is a price per kilo.** No supplier in
		// the country invoices by the gram; a `гр` in the unit column means the
		// exporter wrote the recipe unit where the purchase unit belongs, and
		// taking the number at face value makes the ingredient a thousand times
		// too cheap — after which every dish containing it looks free.
		//
		// Nothing is multiplied here: the price is already the price of a kilo.
		// The unit is the thing that was wrong, and it has been corrected.
		ing := models.Ingredient{
			BrandID: brand,
			Name:    name,
			Unit:    l.Unit,
			Price:   price,
			Note:    strings.TrimSpace(l.Note),
		}
		if price > 0 {
			// The first price is history from the day it is entered, so a
			// report covering last month costs the dish at something rather
			// than at nothing (see Ingredient.PriceAt).
			ing.History = []models.PriceEntry{{At: now, Price: price}}
		}
		if _, err := h.Store.Ingredients.InsertOne(ctx, ing); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		created++
	}
	h.logAction(r, ActPosImport, "ingredient", "", "import",
		strconv.Itoa(created)+" ta masalliq boshqa POS'dan ko'chirildi")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"created": created, "skipped": skipped,
	})
}

// ---- Tech cards ----

func (h *Handler) applyRecipes(
	w http.ResponseWriter, r *http.Request, lines []posimport.Line,
) {
	ctx := r.Context()
	_, _, brand, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	ingByName := map[string]models.Ingredient{}
	for _, ing := range h.scopedIngredients(ctx, brand) {
		ingByName[menuimport.NormalName(ing.Name)] = ing
	}
	dishByName := map[string]models.MenuItem{}
	filter := bson.M{}
	if !brand.IsZero() {
		filter["brandId"] = brand
	}
	cur, err := h.Store.Menu.Find(ctx, filter)
	if err == nil {
		var items []models.MenuItem
		if cur.All(ctx, &items) == nil {
			for _, m := range items {
				dishByName[menuimport.NormalName(m.Name)] = m
			}
		}
	}

	// ⚠️ **Grouped by dish and written once per dish**, because a tech card is
	// the whole card. Appending line by line would mean a re-import doubles
	// every recipe, and a partially-applied import leaves a dish costed from
	// half its ingredients — which is a cost that is wrong and looks fine.
	byDish := map[string][]models.RecipeLine{}
	missingIng := map[string]bool{}
	missingDish := map[string]bool{}
	for _, l := range lines {
		if !l.OK() {
			continue
		}
		dish, ok := dishByName[menuimport.NormalName(l.Dish)]
		if !ok {
			missingDish[l.Dish] = true
			continue
		}
		ing, ok := ingByName[menuimport.NormalName(l.Name)]
		if !ok {
			// ⚠️ **Reported, never created.** An ingredient invented here would
			// have no unit and no price, so every dish using it would be costed
			// at zero for that line — a card that is quietly incomplete is
			// worse than one that is visibly missing.
			missingIng[l.Name] = true
			continue
		}
		qty := l.Qty
		if !l.SmallUnit && l.Unit != "" {
			// The export stated a purchase unit: 0,18 kg is 180 g.
			qty = l.Qty * float64(models.PerUnit(l.Unit))
		} else if l.Unit == "" {
			// No unit column: the number is in the ingredient's own recipe
			// unit, which is what every tech-card export without a unit means.
			qty = l.Qty
		}
		if qty <= 0 {
			continue
		}
		byDish[dish.ID.Hex()] = append(byDish[dish.ID.Hex()],
			models.RecipeLine{IngredientID: ing.ID, Qty: round3(qty)})
	}

	now := time.Now()
	updated := 0
	for id, recipe := range byDish {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		if _, err := h.Store.Menu.UpdateOne(ctx, bson.M{"_id": oid},
			bson.M{"$set": bson.M{"recipe": recipe, "updatedAt": now}}); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		updated++
	}

	h.logAction(r, ActPosImport, "menu", "", "import",
		strconv.Itoa(updated)+" ta texkarta boshqa POS'dan ko'chirildi")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"updated":        updated,
		"missingDish":    keysOf(missingDish),
		"missingProduct": keysOf(missingIng),
	})
}

// ---- Opening stock ----

// applyStock records what was on the shelf as a count.
//
// ⚠️ **A stocktake, not a delivery, and the choice matters.** Entering the
// opening balance as purchases would invent a supplier, a date and a cost that
// never happened, and every "what did we buy this month" report would carry
// them for ever. A count is the honest record: this is what was on the shelf
// the day we started.
//
// ⚠️ **The variance on that first count is the opening balance, and it is
// supposed to be large.** Our books expect nothing, because nothing has been
// delivered into them — so the whole shelf arrives as a surplus. That is a true
// statement about our records rather than a problem with theirs, and the note
// says so, on the count, where somebody reading the report will find it.
func (h *Handler) applyStock(
	w http.ResponseWriter, r *http.Request, lines []posimport.Line,
) {
	ctx := r.Context()
	scope, branch, brand, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ingByName := map[string]models.Ingredient{}
	for _, ing := range h.scopedIngredients(ctx, brand) {
		ingByName[menuimport.NormalName(ing.Name)] = ing
	}

	take := models.Stocktake{BranchID: branch}
	missing := map[string]bool{}
	for _, l := range lines {
		if !l.OK() {
			continue
		}
		ing, ok := ingByName[menuimport.NormalName(l.Name)]
		if !ok {
			missing[l.Name] = true
			continue
		}
		// The sheet counts in purchase units, which is how a shelf is counted
		// and what StocktakeLine.Counted holds. A `гр` column is the one that
		// is not, and it has to come back to kilos or the count is a thousand
		// times too high.
		counted := l.Qty
		if l.SmallUnit {
			counted = l.Qty / float64(models.PerUnit(ing.Unit))
		}
		take.Lines = append(take.Lines,
			models.StocktakeLine{IngredientID: ing.ID, Counted: round3(counted)})
	}
	if len(take.Lines) == 0 {
		httpx.Error(w, http.StatusBadRequest,
			"birorta ham masalliq topilmadi — avval masalliqlar ro'yxatini import qiling")
		return
	}
	take.Note = "Boshqa POS'dan ko'chirilgan boshlang'ich qoldiq"
	if len(missing) > 0 {
		// ⚠️ Named on the count itself. A month later "why was this count
		// short" is asked about a list nobody still has.
		take.Note += " · topilmagan: " + strings.Join(keysOf(missing), ", ")
	}
	take.Note = clampText(take.Note, 400)

	by := "import"
	if u, err := h.adminUser(r); err == nil {
		by = u.Name
	}
	h.saveStocktake(w, r, take, scope, brand, branch, by)
}

func keysOf(m map[string]bool) []string {
	// ⚠️ An empty map must go out as `[]`, not `null` — the panel maps over it.
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
