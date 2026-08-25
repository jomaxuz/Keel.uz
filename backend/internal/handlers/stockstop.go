package handlers

// Dishes stopped because the store is empty.
//
// ⚠️ **The gap this closes.** A dish whose ingredients had run out was still
// rung up: the till took the money, the ticket printed, and the kitchen found
// out at the pass. Nothing on any screen had said so, because nothing connected
// what the store knows to what the till sells.
//
// Three decisions shape everything below.
//
//   - **A third list** (branch.stockSoldOut), never merged into the counter's
//     tap or the POS mirror. One is written by a person, one by a poller, and
//     this one by an arithmetic — merged, each would undo the others, and every
//     undo reads as the feature being broken rather than busy. The same
//     judgement as handlers/posstop.go, arrived at for the third time.
//   - **Off unless the owner turns it on** (branch.stockStop). Refusing a sale
//     is the most expensive thing this system can do, and the balance behind it
//     is an estimate: last count plus deliveries less what the cards say was
//     used. A restaurant that has not entered Tuesday's invoice would have its
//     till refuse food that is sitting on the shelf, during service, with the
//     guest already at the counter.
//   - **An uncounted store stops nothing.** Zero has two meanings — "we have
//     none" and "nobody has ever told us" — and only the first is a reason to
//     refuse a sale. A store with no stocktake behind it has no baseline, so
//     its figures are not measurements yet and are not allowed to close a
//     kitchen. This is the guard that keeps the feature from emptying a menu on
//     the day it is switched on.

import (
	"context"
	"log"
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// How often the shelves are worked out.
//
// ⚠️ Slower than the POS mirror's three minutes, and deliberately: that one
// reflects a person tapping a button and should catch up while the guest is
// still browsing. This one moves when an invoice is entered or a count is
// saved — events that happen a few times a day — and every run reads every
// order in the window for every branch that has it switched on.
const stockStopSyncEvery = 5 * time.Minute

// How deep a prep card may nest before we stop following it.
//
// ⚠️ A guard, not a limit anybody should reach: a sauce made from a stock made
// from a broth is three, and two cards that reference each other is a loop that
// would otherwise hang the sync — see ingredientRates, which resolves the same
// shape in passes for the same reason.
const prepDepthLimit = 6

// StartStockStopSync keeps every opted-in branch's stock stop list current.
//
// Started from cmd/server and stopped with the process. Failures are logged and
// nothing else: a branch whose figures cannot be worked out must keep selling,
// which means the previous list stands rather than being cleared.
func (h *Handler) StartStockStopSync(ctx context.Context) {
	go func() {
		// A short delay, like the POS sync's: the first request of the morning
		// should not queue behind an aggregation over every order in the month.
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		ticker := time.NewTicker(stockStopSyncEvery)
		defer ticker.Stop()
		for {
			h.syncAllStockStopLists(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (h *Handler) syncAllStockStopLists(ctx context.Context) {
	cur, err := h.Store.Branches.Find(ctx, bson.M{"stockStop": true, "isActive": true})
	if err != nil {
		log.Printf("stock stop list: %v", err)
		return
	}
	var branches []models.Branch
	if err := cur.All(ctx, &branches); err != nil {
		log.Printf("stock stop list: %v", err)
		return
	}
	for _, b := range branches {
		if err := h.syncStockStopList(ctx, b.ID, b.BrandID); err != nil {
			log.Printf("stock stop list %s: %v", b.Name, err)
		}
	}
}

// syncStockStopList works out one branch's empty shelves and writes the list.
func (h *Handler) syncStockStopList(
	ctx context.Context, branchID, brandID primitive.ObjectID,
) error {
	// ⚠️ The stock helpers were written for handlers and take a request for its
	// context alone — checked, not assumed. Rather than duplicate four
	// aggregations here, the context is handed to them in the shape they
	// expect; a second copy of that arithmetic is how the report and the stop
	// list would start disagreeing about what is on the shelf.
	r := (&http.Request{}).WithContext(ctx)
	scope := bson.M{"branchId": branchID}

	byWarehouse, since, err := h.expectedStockByWarehouse(r, scope, brandID, branchID, time.Now())
	if err != nil {
		return err
	}

	ingredients := h.scopedIngredients(ctx, brandID)
	placed := h.placementsIn(ctx, branchID)

	shelf := countedShelves(byWarehouse, since, ingredients, placed)
	empty := emptyIngredients(byWarehouse, since, ingredients, placed)
	// ⚠️ No longer short-circuits on "nothing is empty": a shelf can hold two
	// hundred grams against a five-hundred-gram card, which is a stopped dish
	// with nothing at zero.
	if len(shelf) == 0 {
		return h.writeStockStopList(ctx, branchID, nil)
	}

	// ⚠️ The brand's menu, not the company's: a branch belongs to one brand,
	// and stopping another brand's dishes would take food off a menu this
	// kitchen does not cook from — invisibly, since the two never share a screen.
	menuFilter := bson.M{}
	if !brandID.IsZero() {
		menuFilter["brandId"] = brandID
	}
	var menu []models.MenuItem
	if cur, err := h.Store.Menu.Find(ctx, menuFilter); err == nil {
		_ = cur.All(ctx, &menu)
	}
	stopped := dishesShortOf(menu, ingredients, empty, shelf)
	return h.writeStockStopList(ctx, branchID, stopped)
}

// emptyIngredients is which shelves are bare, and which are merely unknown.
//
// ⚠️ **Only a counted store speaks.** An ingredient in a store nobody has ever
// counted has an expected figure of zero because there is no baseline, not
// because there is none of it — and stopping dishes on that would close the
// menu of every restaurant on the day it switched this on. This is the single
// guard the whole feature rests on.
//
// ⚠️ **Prep items are skipped.** A sauce is not on a shelf as itself; what it
// was made from is, and those are checked directly. Treating a pot of sauce as
// an ingredient that can run out would stop dishes twice for one shortage — and
// the second reason would name something the kitchen cannot go and buy.
func emptyIngredients(
	byWarehouse map[primitive.ObjectID]map[primitive.ObjectID]float64,
	since map[primitive.ObjectID]*time.Time,
	ingredients []models.Ingredient,
	placed map[primitive.ObjectID]primitive.ObjectID,
) map[primitive.ObjectID]bool {
	out := map[primitive.ObjectID]bool{}
	for id, q := range countedShelves(byWarehouse, since, ingredients, placed) {
		if q <= 0 {
			out[id] = true
		}
	}
	return out
}

// countedShelves is how much of each bought ingredient is on a counted shelf.
//
// ⚠️ **Absent means "nobody has told us", and absent is not zero.** An
// ingredient in a store with no stocktake behind it is left out of the map
// entirely rather than entered as nothing, and every reader below treats a
// missing key as "no opinion". This is the single guard the whole feature rests
// on: without it, switching the stop list on would empty the menu of every
// restaurant that has never counted.
func countedShelves(
	byWarehouse map[primitive.ObjectID]map[primitive.ObjectID]float64,
	since map[primitive.ObjectID]*time.Time,
	ingredients []models.Ingredient,
	placed map[primitive.ObjectID]primitive.ObjectID,
) map[primitive.ObjectID]float64 {
	out := map[primitive.ObjectID]float64{}
	for _, in := range ingredients {
		// A prep item is not on a shelf as itself; what it was made from is,
		// and that is counted directly.
		store := placed[in.ID]
		if in.DerivedOnly() || since[store] == nil {
			continue
		}
		out[in.ID] = byWarehouse[store][in.ID]
	}
	return out
}

// portionNeeds is what one portion takes off each shelf, in purchase units.
//
// ⚠️ **Resolved through the prep cards**, so a dish made with a sauce made with
// tomatoes needs tomatoes: `rawInputs` already does that resolution for the
// consumption report, and using the same one is what keeps "we cannot make
// this" and "this is what making it used" from ever disagreeing.
func portionNeeds(
	lines []models.RecipeLine,
	raw map[primitive.ObjectID]map[primitive.ObjectID]float64,
	ingredients []models.Ingredient,
) map[primitive.ObjectID]float64 {
	need := map[primitive.ObjectID]float64{}
	for _, l := range lines {
		for id, per := range raw[l.IngredientID] {
			// Recipe units (g, ml, pieces) → purchase units (kg, l, pieces).
			div := float64(models.PerUnit(byUnit(ingredients, id)))
			need[id] += per * l.Qty / div
		}
	}
	return need
}

// dishesShortOf is every dish that cannot be made from what is left.
//
// ⚠️ **Followed through prep cards.** A dish made with a sauce made with
// tomatoes is short of tomatoes, and a check that only looked at the dish's own
// lines would happily sell it — which is the shape of bug that makes people
// stop trusting a stop list, because it is wrong only for the dishes with the
// most work in them.
func dishesShortOf(
	menu []models.MenuItem,
	ingredients []models.Ingredient,
	empty map[primitive.ObjectID]bool,
	shelf map[primitive.ObjectID]float64,
) []primitive.ObjectID {
	raw := rawInputs(ingredients)
	cards := map[primitive.ObjectID][]models.RecipeLine{}
	for _, in := range ingredients {
		if in.DerivedOnly() {
			cards[in.ID] = in.Recipe
		}
	}

	var short func(lines []models.RecipeLine, depth int) bool
	short = func(lines []models.RecipeLine, depth int) bool {
		if depth > prepDepthLimit {
			return false
		}
		for _, l := range lines {
			if empty[l.IngredientID] {
				return true
			}
			if sub, ok := cards[l.IngredientID]; ok && short(sub, depth+1) {
				return true
			}
		}
		return false
	}
	// ⚠️ **Not enough for one portion is the same statement as empty**, only
	// measured against the dish instead of against zero. Two hundred grams of
	// beef left and a card that calls for five hundred is a dish that cannot be
	// cooked, and selling it is the failure this whole feature exists to
	// prevent — arriving through the one door the zero test leaves open.
	//
	// ⚠️ **One portion, and deliberately not two.** The balance is an estimate,
	// and anything beyond "we literally cannot make this one" would be a
	// forecast: how many are about to be ordered is not something an arithmetic
	// over last month's invoices can know. Refusing a sale is the most
	// expensive thing this system does, and the threshold is the smallest one
	// that is still a fact.
	//
	// ⚠️ An ingredient with **no counted shelf behind it is not short**, only
	// unknown — the same guard `emptyIngredients` rests on, applied one level
	// up: a missing key here must never read as nothing.
	cannotMakeOne := func(lines []models.RecipeLine) bool {
		if len(shelf) == 0 {
			return false
		}
		for id, want := range portionNeeds(lines, raw, ingredients) {
			have, counted := shelf[id]
			if counted && want > have {
				return true
			}
		}
		return false
	}

	// ⚠️ **A set is short when any dish in it is.** A combo has no card of its
	// own — the tomatoes are in its members — so the two checks above look at
	// an empty recipe and pass it. The kitchen then gets an order for a family
	// set it cannot assemble, having been told by the same screen that every
	// dish in that set is off. Resolved against the menu already in hand rather
	// than with a second query, and only one level deep: `validateCombo`
	// refuses a set inside a set.
	byID := make(map[primitive.ObjectID]models.MenuItem, len(menu))
	for _, m := range menu {
		byID[m.ID] = m
	}
	dishShort := func(m models.MenuItem) bool {
		if len(m.Recipe) == 0 && !anyChoiceCosted(m.Options) {
			return false
		}
		if short(m.Recipe, 0) || cannotMakeOne(m.Recipe) {
			return true
		}
		return requiredChoicesAllShort(m.Options, func(lines []models.RecipeLine, d int) bool {
			return short(lines, d) || cannotMakeOne(lines)
		})
	}

	out := []primitive.ObjectID{}
	for _, m := range menu {
		if m.IsCombo() {
			for _, c := range m.ComboItems {
				member, ok := byID[c.MenuItemID]
				// ⚠️ A member that is no longer on the menu is not a shortage.
				// The set is already unsellable for a different reason
				// (`resolveCombo` blocks it), and reporting it here would
				// name an empty shelf the kitchen can do nothing about.
				if ok && dishShort(member) {
					out = append(out, m.ID)
					break
				}
			}
			continue
		}
		// ⚠️ A dish with no card **and no priced-out choices** is never
		// stopped. An empty recipe is "nobody has written this one down", not
		// "this needs nothing" — and stopping it would take out every dish in a
		// restaurant that has costed half its menu.
		if dishShort(m) {
			out = append(out, m.ID)
		}
	}
	return out
}

// anyChoiceCosted reports whether a card lives on the choices rather than the
// dish — a bar's vodka, whose only recipe is behind "Hajm".
func anyChoiceCosted(groups []models.MenuOption) bool {
	for _, g := range groups {
		for _, c := range g.Choices {
			if len(c.Recipe) > 0 {
				return true
			}
		}
	}
	return false
}

// requiredChoicesAllShort is the pour rule.
//
// ⚠️ **A required group stops the dish only when every one of its answers is
// short.** A vodka offered at 40, 50 and 100 ml is unorderable when the vodka
// is gone — all three answers are the same empty bottle — but a dish is still
// perfectly orderable when one size of it has run out and another has not, and
// stopping it then would take a sellable dish off the menu.
//
// ⚠️ **An optional group never stops anything.** "Extra olive" being out is a
// reason to refuse the olive, not the drink; refusing the drink would be the
// stop list making a decision the guest was never asked about.
//
// ⚠️ A required group with no costed choice at all is skipped rather than read
// as "every answer is short" — an empty card is nobody having written it down,
// which is the same rule the dish itself follows two lines above.
func requiredChoicesAllShort(
	groups []models.MenuOption, short func([]models.RecipeLine, int) bool,
) bool {
	for _, g := range groups {
		if !g.Required || len(g.Choices) == 0 {
			continue
		}
		costed, allShort := false, true
		for _, c := range g.Choices {
			if len(c.Recipe) == 0 {
				// An answer that takes nothing measurable out of the store can
				// always be given, so the group is answerable.
				allShort = false
				continue
			}
			costed = true
			if !short(c.Recipe, 0) {
				allShort = false
			}
		}
		if costed && allShort {
			return true
		}
	}
	return false
}

// writeStockStopList stores what is stopped and when it was worked out.
//
// ⚠️ The timestamp is written even when nothing is stopped, because the panel's
// useful line is "last worked out at", not a flag. A stored flag goes stale the
// moment the clock passes it, and a stop list that stopped updating at lunchtime
// looks exactly like one with nothing stopped — the same rule as the POS
// mirror's, and as lastEventAt on the phone system.
func (h *Handler) writeStockStopList(
	ctx context.Context, branchID primitive.ObjectID, stopped []primitive.ObjectID,
) error {
	if stopped == nil {
		stopped = []primitive.ObjectID{}
	}
	now := time.Now()
	_, err := h.Store.Branches.UpdateByID(ctx, branchID, bson.M{"$set": bson.M{
		"stockSoldOut":   stopped,
		"stockSoldOutAt": now,
	}})
	return err
}

// AdminSyncStockStopList is the "work it out now" button.
//
// The background loop already runs every few minutes, so this exists for the
// two moments it does not cover: right after an invoice or a count has been
// entered — when the owner wants to see the menu come back — and while somebody
// is standing in the panel wondering whether the feature does anything at all.
func (h *Handler) AdminSyncStockStopList(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	branchID := h.scopeBranch(r, scope)
	if branchID.IsZero() {
		httpx.Error(w, http.StatusBadRequest, "filial tanlanmagan")
		return
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	branch, err := h.branchByID(r, branchID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "filial topilmadi")
		return
	}
	// ⚠️ Refused rather than quietly computed: running it for a branch that has
	// the feature off would write a stop list nothing else maintains, and it
	// would then sit there going stale with no loop to correct it.
	if !branch.StockStop {
		httpx.Error(w, http.StatusBadRequest,
			"bu filialda ombor bo'yicha to'xtatish yoqilmagan")
		return
	}
	if err := h.syncStockStopList(r.Context(), branchID, branch.BrandID); err != nil {
		// 200 with ok:false, like the POS button: the request was handled, the
		// arithmetic is what failed, and the panel says so in its own words.
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": false, "message": err.Error()})
		return
	}
	var saved models.Branch
	_ = h.Store.Branches.FindOne(r.Context(), bson.M{"_id": branchID}).Decode(&saved)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"stopped":  len(saved.StockSoldOut),
		"syncedAt": saved.StockSoldOutAt,
	})
}

// AdminSetStockStop turns stopping-by-stock on or off for one branch.
//
// ⚠️ **Its own endpoint, not a field on the settings form.** The branch form is
// a page that may have been open for an hour, and saving it writes every field
// it holds — which is exactly how the sold-out list and the kiosk key have each
// been zeroed by somebody editing an address. This switch is pressed from the
// stop-list screen, where its consequences are on display, and it writes one
// field.
//
// ⚠️ Switching it **off** clears the list rather than leaving it. A stop that
// nothing maintains any more would sit there going stale, with no loop to lift
// it and no button that admits to owning it — dishes off sale for a reason the
// panel would no longer be able to explain.
func (h *Handler) AdminSetStockStop(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	branchID := h.scopeBranch(r, scope)
	if branchID.IsZero() {
		httpx.Error(w, http.StatusBadRequest, "filial tanlanmagan")
		return
	}
	if err := h.requireBranchAccess(r, branchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := bson.M{"stockStop": req.Enabled, "updatedAt": time.Now()}
	if !req.Enabled {
		set["stockSoldOut"] = []primitive.ObjectID{}
	}
	if _, err := h.Store.Branches.UpdateByID(r.Context(), branchID,
		bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActSettingsUpdate, "branch", branchID.Hex(), "",
		"ombor bo'yicha to'xtatish: "+onOff(req.Enabled))
	// Worked out immediately on the way on, so the owner sees the answer rather
	// than an empty list for the next five minutes — which reads as "nothing
	// happened" and gets the switch turned off again.
	var saved models.Branch
	_ = h.Store.Branches.FindOne(r.Context(), bson.M{"_id": branchID}).Decode(&saved)
	if req.Enabled {
		if err := h.syncStockStopList(r.Context(), branchID, saved.BrandID); err != nil {
			log.Printf("stock stop list: %v", err)
		}
		_ = h.Store.Branches.FindOne(r.Context(), bson.M{"_id": branchID}).Decode(&saved)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"enabled":  saved.StockStop,
		"stopped":  len(saved.StockSoldOut),
		"syncedAt": saved.StockSoldOutAt,
	})
}

func onOff(v bool) string {
	if v {
		return "yoqildi"
	}
	return "o'chirildi"
}
