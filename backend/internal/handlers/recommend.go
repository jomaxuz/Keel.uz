package handlers

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// "People usually order this with it."
//
// Two sources, and both are needed:
//
//   - **What actually sells together**, counted from the order history. Nobody
//     in the restaurant knows this as well as the receipts do — the pairings
//     that matter are usually not the ones on the menu next to each other.
//   - **What the owner picked by hand** (`menu_item.recommendedIds`). ⚠️ Without
//     this the feature can never promote anything **new**: a dish added on
//     Monday has no history, so an automatic-only version would recommend
//     exactly the dishes that already sell and nothing else — the opposite of
//     what an owner would use it for.
//
// The hand-picked ones come first. They are a decision; the counts are an
// observation, and a decision that an observation can overrule is not a
// decision.
//
// ⚠️ **Suggestions are never a reason to show something that cannot be sold.**
// Everything below is filtered through availability and the branch's sold-out
// list before it reaches a page — a suggested dish the kitchen ran out of two
// hours ago turns a helpful line into "this restaurant does not know its own
// menu", which costs more than the upsell was worth.

const (
	// How far back the pairing counts look.
	//
	// ⚠️ Ninety days rather than all time. A menu changes, a season changes,
	// and a pairing from eighteen months ago is a fact about a dish that may
	// not exist — the count would be real and the recommendation useless.
	recommendWindowDays = 90
	// How long the computed index is kept before it is built again.
	//
	// The index reads the period's orders, which is the sort of query that must
	// not run on every dish page view. Half an hour is far shorter than the
	// timescale on which what-sells-with-what actually changes, and this is one
	// process per restaurant, so an in-memory cache is the whole story — the
	// same reasoning as the SMS sender cache and the rate limiter.
	recommendTTL = 30 * time.Minute
	// How many suggestions a page gets. Four fits a row on a phone, and a list
	// long enough to browse is a second menu — which is a reason to leave the
	// checkout, not to add to it.
	recommendLimit = 4
	// A pairing seen fewer times than this is a coincidence, not a habit. Two
	// people ordering the same two things once each says nothing at all, and
	// showing it makes every other suggestion less believable.
	recommendMinPairs = 3
)

// pairScore is one candidate and how often it came with the seed dish.
type pairScore struct {
	ID    primitive.ObjectID
	Count int
}

// pairIndex is "what came with what", per brand.
type pairIndex struct {
	// dish id → the dishes ordered alongside it, most frequent first.
	with map[primitive.ObjectID][]pairScore
	at   time.Time
}

var (
	pairCacheMu sync.Mutex
	// Keyed by brand: the menu is the brand's, so a pairing from one brand's
	// receipts means nothing on another's page.
	pairCache = map[primitive.ObjectID]*pairIndex{}
)

// pairsFor returns the co-occurrence index for a brand, building it if the
// cached one has expired.
func (h *Handler) pairsFor(ctx context.Context, brandID primitive.ObjectID) *pairIndex {
	pairCacheMu.Lock()
	defer pairCacheMu.Unlock()

	if idx, ok := pairCache[brandID]; ok && time.Since(idx.at) < recommendTTL {
		return idx
	}
	idx := h.buildPairs(ctx, brandID)
	pairCache[brandID] = idx
	return idx
}

// buildPairs counts what was ordered together over the window.
func (h *Handler) buildPairs(ctx context.Context, brandID primitive.ObjectID) *pairIndex {
	idx := &pairIndex{with: map[primitive.ObjectID][]pairScore{}, at: time.Now()}

	filter := bson.M{
		"createdAt": bson.M{"$gte": time.Now().AddDate(0, 0, -recommendWindowDays)},
		// ⚠️ Cancelled orders are excluded. A cancellation is the one signal
		// that those dishes did *not* go together — often literally, because
		// the kitchen could not do both.
		"status": bson.M{"$ne": models.StatusCancelled},
	}
	if !brandID.IsZero() {
		filter["brandId"] = brandID
	}

	// Only the item ids are read back. The rest of an order is large and this
	// query walks a quarter of a year of them.
	cur, err := h.Store.Orders.Find(ctx, filter,
		options.Find().SetProjection(bson.M{"items.menuItemId": 1}))
	if err != nil {
		return idx
	}
	defer cur.Close(ctx)

	counts := map[primitive.ObjectID]map[primitive.ObjectID]int{}
	for cur.Next(ctx) {
		var o struct {
			Items []struct {
				MenuItemID primitive.ObjectID `bson:"menuItemId"`
			} `bson:"items"`
		}
		if err := cur.Decode(&o); err != nil {
			continue
		}
		// Distinct dishes on the receipt: two portions of the same thing is one
		// dish, and pairing a dish with itself is not a suggestion.
		seen := map[primitive.ObjectID]bool{}
		var ids []primitive.ObjectID
		for _, it := range o.Items {
			if it.MenuItemID.IsZero() || seen[it.MenuItemID] {
				continue
			}
			seen[it.MenuItemID] = true
			ids = append(ids, it.MenuItemID)
		}
		for i, a := range ids {
			for j, b := range ids {
				if i == j {
					continue
				}
				if counts[a] == nil {
					counts[a] = map[primitive.ObjectID]int{}
				}
				counts[a][b]++
			}
		}
	}

	for dish, others := range counts {
		list := make([]pairScore, 0, len(others))
		for id, n := range others {
			if n < recommendMinPairs {
				continue
			}
			list = append(list, pairScore{ID: id, Count: n})
		}
		sort.Slice(list, func(i, j int) bool {
			if list[i].Count != list[j].Count {
				return list[i].Count > list[j].Count
			}
			// A stable tiebreak, so two dishes with the same count do not swap
			// places between page loads — which reads as a broken widget.
			return list[i].ID.Hex() < list[j].ID.Hex()
		})
		idx.with[dish] = list
	}
	return idx
}

// recommend picks what to suggest alongside a set of dishes.
//
// `seed` is the dish being looked at, or everything in the cart. `exclude` is
// what must not come back — always the seed itself, because suggesting
// something already in the basket is the clearest possible sign that nothing is
// actually being computed.
func (h *Handler) recommend(
	ctx context.Context, brandID primitive.ObjectID,
	seed []primitive.ObjectID, limit int,
) ([]models.MenuItem, error) {
	if len(seed) == 0 {
		return nil, nil
	}
	exclude := map[primitive.ObjectID]bool{}
	for _, id := range seed {
		exclude[id] = true
	}

	// ---- 1. The owner's own picks, in the order they wrote them ----
	var ordered []primitive.ObjectID
	picked := map[primitive.ObjectID]bool{}
	add := func(id primitive.ObjectID) {
		if id.IsZero() || exclude[id] || picked[id] {
			return
		}
		picked[id] = true
		ordered = append(ordered, id)
	}

	seeds, err := h.menuByIDs(ctx, seed)
	if err != nil {
		return nil, err
	}
	for _, id := range seed {
		if item, ok := seeds[id]; ok {
			for _, rec := range item.RecommendedIDs {
				add(rec)
			}
		}
	}

	// ---- 2. What the receipts say, strongest pairing first ----
	//
	// Scores are summed across the seed rather than taken per dish: for a cart
	// of three things, the useful suggestion is what goes with the *cart*, and
	// the top pairing of each item separately would be three unrelated answers.
	idx := h.pairsFor(ctx, brandID)
	totals := map[primitive.ObjectID]int{}
	for _, id := range seed {
		for _, p := range idx.with[id] {
			if exclude[p.ID] || picked[p.ID] {
				continue
			}
			totals[p.ID] += p.Count
		}
	}
	auto := make([]pairScore, 0, len(totals))
	for id, n := range totals {
		auto = append(auto, pairScore{ID: id, Count: n})
	}
	sort.Slice(auto, func(i, j int) bool {
		if auto[i].Count != auto[j].Count {
			return auto[i].Count > auto[j].Count
		}
		return auto[i].ID.Hex() < auto[j].ID.Hex()
	})
	for _, p := range auto {
		ordered = append(ordered, p.ID)
	}

	if len(ordered) == 0 {
		return nil, nil
	}

	// ---- 3. Only what can actually be sold ----
	items, err := h.menuByIDs(ctx, ordered)
	if err != nil {
		return nil, err
	}
	out := make([]models.MenuItem, 0, limit)
	for _, id := range ordered {
		item, ok := items[id]
		if !ok || !item.IsAvailable {
			continue
		}
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// menuByIDs loads menu items by id, keyed for lookup.
func (h *Handler) menuByIDs(ctx context.Context, ids []primitive.ObjectID) (map[primitive.ObjectID]models.MenuItem, error) {
	out := map[primitive.ObjectID]models.MenuItem{}
	if len(ids) == 0 {
		return out, nil
	}
	cur, err := h.Store.Menu.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	var items []models.MenuItem
	if err := cur.All(ctx, &items); err != nil {
		return nil, err
	}
	for _, item := range items {
		out[item.ID] = item
	}
	return out, nil
}

// normalizeRecommended cleans the hand-picked list before it is stored.
//
// ⚠️ **A dish must not recommend itself.** It is the easiest thing in the world
// to pick from a list that includes the dish being edited, and the result is a
// card offering the guest what they are already looking at — which reads as a
// bug in the suggestions rather than a mistake in the settings, and makes every
// other suggestion on the page less believable.
//
// Duplicates and unparsable ids go the same way. Ids of dishes that no longer
// exist are **left alone**: the read path already skips anything it cannot load,
// and quietly rewriting the owner's list because a dish was temporarily removed
// would lose a decision they would have to make again.
func normalizeRecommended(self primitive.ObjectID, ids []primitive.ObjectID) []primitive.ObjectID {
	// Capped: this is stored on the dish and read on every page that shows it,
	// and nothing sensible needs more than a handful.
	const max = 12
	out := make([]primitive.ObjectID, 0, len(ids))
	seen := map[primitive.ObjectID]bool{}
	for _, id := range ids {
		if id.IsZero() || id == self || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
		if len(out) >= max {
			break
		}
	}
	return out
}

// ---- Endpoints ----

type recommendRequest struct {
	// The dishes to suggest alongside: one for a dish page, the whole basket
	// for the cart and checkout.
	ItemIDs []string `json:"itemIds"`
	// Which branch is cooking, when it is known. Decides the sold-out list.
	BranchID string `json:"branchId"`
}

// Recommendations suggests dishes to go with a basket or a dish.
//
// A POST rather than a GET with a long query string: the basket is the input,
// and a URL carrying eight dish ids is one that gets truncated, logged and
// cached by something along the way.
func (h *Handler) Recommendations(w http.ResponseWriter, r *http.Request) {
	var req recommendRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	seed := make([]primitive.ObjectID, 0, len(req.ItemIDs))
	for _, raw := range req.ItemIDs {
		if id, err := objectID(raw); err == nil {
			seed = append(seed, id)
		}
	}
	if len(seed) == 0 {
		// ⚠️ An empty array, not null. Go marshals a nil slice as `null` and the
		// page maps over this — the trap that has taken a screen down twice in
		// this codebase.
		httpx.JSON(w, http.StatusOK, []models.MenuItem{})
		return
	}

	// The brand comes from the dishes themselves rather than the request: a
	// menu belongs to a brand, and a browser naming a different one would be
	// asking for suggestions from a menu it cannot order from.
	seeds, err := h.menuByIDs(r.Context(), seed)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var brandID primitive.ObjectID
	for _, id := range seed {
		if item, ok := seeds[id]; ok && !item.BrandID.IsZero() {
			brandID = item.BrandID
			break
		}
	}

	items, err := h.recommend(r.Context(), brandID, seed, recommendLimit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []models.MenuItem{}
	}
	// ⚠️ The same sold-out lens the menu itself is drawn through. A suggestion
	// that contradicts the card it sits under — offered here, greyed out one
	// screen away — reads as a restaurant that does not know its own menu, which
	// costs more than the upsell was worth.
	if lens, _ := h.publicSoldOut(r, brandID); lens != nil {
		for i := range items {
			items[i].SoldOut = lens(items[i].ID)
		}
	}

	httpx.JSON(w, http.StatusOK, items)
}
