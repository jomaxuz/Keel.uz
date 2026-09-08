package handlers

// ---- The morning, as the office sees it ----
//
// ⚠️ **The one screen that shows the whole errand.** The barman sees what he
// asked for, the buyer sees his half, the storekeeper sees theirs — and until
// this existed nobody could see the sentence those three make together: asked at
// six, half of it picked off a shelf at ten past, the fruit bought at nine, and
// one line of it never signed for. That last state is the one worth opening a
// panel over, and it was invisible from every other screen in the product.
//
// ⚠️ **Grouped by the request, not by the document.** Splitting a list in two is
// the server's idea, not the barman's: a panel that listed both halves as
// separate rows would answer "what did the bar ask for this morning" with two
// answers, and an owner comparing them would be comparing a request with itself.

import (
	"net/http"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// How far back the screen looks when nobody says. A fortnight, like the dispatch
// list: long enough to find the morning somebody is arguing about, short enough
// that it opens on this week.
const buyOrderDays = 14

// AdminBuyOrders is every request in view, newest first, with its halves
// together.
func (h *Handler) AdminBuyOrders(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := bson.M{}
	for k, v := range scope {
		filter[k] = v
	}
	// ⚠️ **Either end of it.** A store request answered by the central branch
	// belongs to two branches, and a manager clamped to the central one has to
	// see the work standing on their own shelf — otherwise the queue their phone
	// draws has no counterpart anywhere in the panel.
	if cond, ok := branchCond(scope); ok {
		delete(filter, "branchId")
		filter["$or"] = []bson.M{{"branchId": cond}, {"supplyBranchId": cond}}
	}
	from, to, err := parseRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if from == nil {
		f := time.Now().AddDate(0, 0, -buyOrderDays)
		from = &f
	}
	if rng := timeRange(from, to); len(rng) > 0 {
		filter["createdAt"] = rng
	}
	// ⚠️ **"Open" is everything that is not signed for**, which includes what is
	// on its way. A list somebody shopped and nobody counted is the most open
	// thing here — it is the exact moment goods are in a bag in a corridor.
	if r.URL.Query().Get("open") == "1" {
		filter["status"] = bson.M{"$ne": models.ShoppingDone}
	}

	cur, err := h.Store.BuyOrders.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(300))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.ShoppingOrder{}
	_ = cur.All(r.Context(), &rows)

	names := h.branchNames(r)

	// ⚠️ **Grouped in Go rather than in an aggregation.** The grouping key is
	// missing on every list written before this feature — those are whole
	// requests of one half — and a `$group` on an absent field would fold all of
	// them into one enormous row dated whenever the newest of them was written.
	type group struct {
		GroupID   string                 `json:"groupId"`
		ForDate   string                 `json:"forDate"`
		CreatedAt time.Time              `json:"createdAt"`
		CreatedBy string                 `json:"createdBy"`
		Branch    string                 `json:"branch,omitempty"`
		Supply    string                 `json:"supply,omitempty"`
		Orders    []models.ShoppingOrder `json:"orders"`
	}
	index := map[string]int{}
	out := []group{}
	for _, o := range rows {
		key := o.GroupID.Hex()
		if o.GroupID.IsZero() {
			// A request from before the split existed: its own group, named by
			// its own id, so it reads as the whole request it actually is.
			key = "o" + o.ID.Hex()
		}
		i, ok := index[key]
		if !ok {
			out = append(out, group{
				GroupID: key, ForDate: o.ForDate, CreatedAt: o.CreatedAt,
				CreatedBy: o.CreatedBy, Branch: names[o.BranchID],
			})
			i = len(out) - 1
			index[key] = i
		}
		if o.FromStore() && o.SupplyBranchID != o.BranchID {
			out[i].Supply = names[o.SupplyBranchID]
		}
		out[i].Orders = append(out[i].Orders, o)
	}
	// The halves in a fixed order, so a request does not shuffle between reads.
	for i := range out {
		sort.SliceStable(out[i].Orders, func(a, b int) bool {
			return out[i].Orders[a].Source < out[i].Orders[b].Source
		})
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"groups": out})
}
