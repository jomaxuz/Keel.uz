package handlers

import (
	"math"
	"net/http"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- What to buy ----
//
// ⚠️ **The warning existed and led nowhere.** `minQty` has been on the
// ingredient from the start and the balance screen turns a row amber when the
// shelf drops below it — and then stops. The owner reads "we are low on four
// things", opens a notebook and writes them down again. This is the second half
// the alert was always missing, and it is the same shape as the segments that
// had nowhere to lead until campaigns existed: a condition worth noticing has
// to end in the action it implies.
//
// ⚠️ **Grouped by supplier, because that is how shopping is actually done.** A
// flat list sorted by urgency means reading all forty rows to work out which
// four to mention to the butcher — and then reading them again for the next
// call. One group per phone number is one call per group.
//
// ⚠️ **It is an estimate and it says so**, exactly like the balance it is built
// from: last count, plus deliveries and moves in, less what the cards say the
// dishes used, less write-offs and moves out. The date it is measured from
// travels with it, because a suggestion to buy nine kilos is worth a different
// amount of trust depending on whether the shelf was counted last night or in
// March.

type shoppingRow struct {
	IngredientID string  `json:"ingredientId"`
	Name         string  `json:"name"`
	Unit         string  `json:"unit"`
	OnHand       float64 `json:"onHand"`
	MinQty       float64 `json:"minQty"`
	// How much short of the minimum the shelf is.
	//
	// ⚠️ **Suggested, not ordered.** It is the gap to the reorder point, which
	// is the smallest defensible number — a case size, a delivery rhythm and
	// next week's booking are all things only the owner knows. The field is
	// there so nobody has to do subtraction standing at a shelf; the quantity
	// actually bought is typed on the delivery.
	Suggested float64 `json:"suggested"`
	// What that much cost last time, and therefore roughly what this will.
	Price int `json:"price"`
	Cost  int `json:"cost"`
}

type shoppingGroup struct {
	SupplierID string        `json:"supplierId"`
	Name       string        `json:"name"`
	Phone      string        `json:"phone,omitempty"`
	Rows       []shoppingRow `json:"rows"`
	Cost       int           `json:"cost"`
}

// AdminShoppingList is what has fallen below its minimum, by supplier.
func (h *Handler) AdminShoppingList(w http.ResponseWriter, r *http.Request) {
	scope, branch, brand, err := h.stockBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	out, total, since, err := h.shoppingList(r, scope, branch, brand)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"groups": out, "cost": total, "since": since,
	})
}

// shoppingList is the list itself, for the panel and for the buyer's phone.
//
// ⚠️ **One function because the two screens must agree about what is short.**
// The buyer standing at a market and the owner reading the panel are looking at
// the same shelves; a second implementation would eventually have them
// disagree, and the argument would happen with money already spent. Same
// reasoning as `soldOutHeldBy` and `saveStocktake`.
func (h *Handler) shoppingList(
	r *http.Request, scope bson.M, branch, brand primitive.ObjectID,
) ([]shoppingGroup, int, *time.Time, error) {
	byWarehouse, since, err := h.expectedStockByWarehouse(r, scope, brand, branch, time.Now())
	if err != nil {
		return nil, 0, nil, err
	}
	ingredients := h.scopedIngredients(r.Context(), brand)
	placed := h.placementsIn(r.Context(), branch)
	lastSupplier := h.lastSupplierOf(r, scope)

	named := map[primitive.ObjectID]models.Supplier{}
	if cur, err := h.Store.Suppliers.Find(r.Context(), Scope{BrandID: brand}.brandFilter(bson.M{})); err == nil {
		var rows []models.Supplier
		_ = cur.All(r.Context(), &rows)
		for _, s := range rows {
			named[s.ID] = s
		}
	}

	groups := map[primitive.ObjectID]*shoppingGroup{}
	total := 0
	for _, in := range ingredients {
		// ⚠️ **Only where a minimum was set**, and a prep item never: nobody
		// tracks a minimum for cinnamon, and a sauce is cooked rather than
		// bought — putting either on a shopping list is how a list stops being
		// read.
		if in.MinQty <= 0 || in.DerivedOnly() {
			continue
		}
		onHand := byWarehouse[placed[in.ID]][in.ID]
		if onHand >= in.MinQty {
			continue
		}
		// ⚠️ A shelf that has gone negative is still only short by its minimum:
		// the negative half is a measurement error (an unentered delivery, a
		// card that overstates), and ordering against it would buy twice.
		short := in.MinQty - math.Max(onHand, 0)
		price := in.PriceAt(time.Now())
		row := shoppingRow{
			IngredientID: in.ID.Hex(),
			Name:         in.Name,
			Unit:         in.Unit,
			OnHand:       round3(onHand),
			MinQty:       in.MinQty,
			Suggested:    round3(short),
			Price:        price,
			Cost:         int(math.Round(float64(price) * short)),
		}
		sup := lastSupplier[in.ID]
		g, ok := groups[sup]
		if !ok {
			g = &shoppingGroup{
				SupplierID: sup.Hex(),
				Name:       named[sup].Name,
				Phone:      named[sup].Phone,
			}
			groups[sup] = g
		}
		g.Rows = append(g.Rows, row)
		g.Cost += row.Cost
		total += row.Cost
	}

	out := make([]shoppingGroup, 0, len(groups))
	for _, g := range groups {
		// Shortest-stocked first inside a group: the call is made once, and the
		// thing that runs out tomorrow must not be below the thing that runs
		// out next week.
		sort.SliceStable(g.Rows, func(i, j int) bool {
			return g.Rows[i].OnHand < g.Rows[j].OnHand
		})
		out = append(out, *g)
	}
	// ⚠️ Named suppliers first, then by spend. The unnamed group is real
	// shopping — the market run — but it is the one nobody can be rung about,
	// so it belongs at the bottom rather than in the middle.
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].Name == "") != (out[j].Name == "") {
			return out[j].Name == ""
		}
		return out[i].Cost > out[j].Cost
	})

	// The oldest count behind any of this — the honest caveat, like the balance
	// screen's: the weakest half of the answer, not the strongest.
	var oldest *time.Time
	for _, t := range since {
		if t == nil {
			oldest = nil
			break
		}
		if oldest == nil || t.Before(*oldest) {
			oldest = t
		}
	}

	return out, total, oldest, nil
}

// lastSupplierOf is who each ingredient came from most recently.
//
// ⚠️ **Most recent rather than most frequent.** A restaurant that changed
// butcher last month should be told to ring the new one; the count over a year
// would keep naming the old one until the new one overtook him, which is
// exactly the period the answer is most wrong in.
//
// ⚠️ Read from the deliveries rather than stored on the ingredient: a field
// would need writing on every purchase and would drift the first time one was
// deleted.
func (h *Handler) lastSupplierOf(
	r *http.Request, scope bson.M,
) map[primitive.ObjectID]primitive.ObjectID {
	out := map[primitive.ObjectID]primitive.ObjectID{}
	filter := bson.M{"supplierId": bson.M{"$exists": true}}
	for k, v := range scope {
		filter[k] = v
	}
	// Oldest first, so a later delivery simply overwrites an earlier one and
	// the last write wins — cheaper than tracking a date per ingredient.
	cur, err := h.Store.Purchases.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "at", Value: 1}}).SetLimit(500))
	if err != nil {
		return out
	}
	var rows []models.Purchase
	if err := cur.All(r.Context(), &rows); err != nil {
		return out
	}
	for _, p := range rows {
		if p.SupplierID.IsZero() {
			continue
		}
		for _, l := range p.Lines {
			out[l.IngredientID] = p.SupplierID
		}
	}
	return out
}
