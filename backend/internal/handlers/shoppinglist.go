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

	// ---- Why this quantity, and until when ----
	//
	// ⚠️ **Shown rather than trusted.** A number an owner cannot take apart is
	// a number they either follow blindly or ignore entirely, and both are
	// worse than the notebook this screen replaced. Each of these is a
	// measurement from documents the restaurant already writes — see
	// orderplan.go.

	// Which rule decided the quantity: "forecast" — what it will take before
	// the next delivery; "min" — the gap back up to the reorder point.
	//
	// ⚠️ **Named, because the two mean different things.** "Min" is a line
	// somebody drew once and may never have revisited; "forecast" is this
	// month's cooking. An owner reading a quantity deserves to know which of
	// those they are looking at.
	Basis string `json:"basis"`
	// How many days ahead this order is meant to last.
	Cover int `json:"cover,omitempty"`
	// Forecast consumption per day over that horizon, in purchase units.
	Daily float64 `json:"daily,omitempty"`
	// Measured days between deliveries, and how many were measured. Zero when
	// there have not been two.
	Every      float64 `json:"every,omitempty"`
	Deliveries int     `json:"deliveries,omitempty"`
	// Measured shelf life in days, where the dates are entered — it is what
	// caps the horizon, so a screen showing a short cover can say why.
	ShelfLife float64 `json:"shelfLife,omitempty"`
	// Already asked for on a list somebody is out with, and therefore already
	// subtracted from the suggestion.
	Requested float64 `json:"requested,omitempty"`
	// Days the branch traded and this line moved nothing, on a line that
	// otherwise moves nearly every day.
	//
	// ⚠️ **An empty shelf is not a quiet one, and the data cannot tell them
	// apart on its own.** These days are left out of the rate the forecast is
	// built from — otherwise an item that ran out is ordered less and runs out
	// again — and reported so the screen can say the forecast was measured
	// without them rather than quietly inventing demand.
	StockOuts int `json:"stockOuts,omitempty"`
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

	// ---- What the shelf actually consumes, and how it is refilled ----
	//
	// ⚠️ **Three measurements rather than a setting**, all read from documents
	// the restaurant already writes — see orderplan.go for why each one is
	// measured instead of asked for.
	now := time.Now()
	profile := h.demandProfile(r.Context(), scope,
		now.AddDate(0, 0, -7*demandWeeks), now)
	refill := h.deliveryRhythm(r.Context(), scope, now.AddDate(0, 0, -rhythmDays))
	requested := h.requestedQty(r.Context(), branch)

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
		// ⚠️ **A prep item never.** A sauce is cooked rather than bought, and
		// putting one on a shopping list is how a list stops being read.
		if in.DerivedOnly() {
			continue
		}
		onHand := byWarehouse[placed[in.ID]][in.ID]
		// ⚠️ A shelf that has gone negative is treated as empty and no worse:
		// the negative half is a measurement error (an unentered delivery, a
		// card that overstates), and ordering against it would buy twice.
		have := math.Max(onHand, 0) + requested[in.ID]

		// ---- The two rules, and the one that asks for more wins ----
		//
		// ⚠️ **Both, rather than the forecast replacing the minimum.** A
		// minimum is a line an owner drew deliberately — often for something
		// whose absence stops service rather than something that sells fast —
		// and a forecast that quietly overrode it would take away a control
		// people already rely on. Whichever asks for more is the honest answer,
		// and `basis` says which one it was.
		//
		// ⚠️ **Only where a minimum was set**: nobody tracks a minimum for
		// cinnamon, and a list where every ingredient eventually appears is a
		// list nobody reads.
		short := 0.0
		if in.MinQty > 0 && have < in.MinQty {
			short = in.MinQty - have
		}
		basis := "min"

		// ⚠️ **The forecast needs a history before it is allowed to speak.**
		// Three separate selling days is a thing that sells; anything less is a
		// rate invented from one event — a dress that left the rail once, and
		// an order for three more of it. See forecastMinDays.
		want := profile[in.ID]
		fill := refill[in.ID]
		cover := coverDays(fill)
		// ⚠️ **And a delivery history, which is the guard that keeps this
		// screen from filling up.** "How much until the next delivery" has no
		// meaning where nobody records deliveries: the horizon would be a
		// made-up week, and every fast-moving ingredient in the catalogue would
		// appear every morning — which is how a list stops being read. A
		// restaurant that does not enter its purchases keeps exactly the screen
		// it had: minimums only.
		forecastable := want.days >= forecastMinDays && fill.deliveries >= 2
		need := 0.0
		if forecastable {
			need = want.forecast(now, cover) - have
			if need > short {
				short = need
				basis = "forecast"
			}
		}
		if short <= 0 {
			continue
		}
		// ⚠️ **Rounded up, and the cost follows the rounded figure.** A price
		// worked out from 71.93 kilos beside an order for 72 is two numbers
		// that do not belong to each other, and the one somebody checks against
		// the invoice is the second.
		short = orderQty(short, in.Unit)

		price := in.PriceAt(now)
		row := shoppingRow{
			IngredientID: in.ID.Hex(),
			Name:         in.Name,
			Unit:         in.Unit,
			OnHand:       round3(onHand),
			MinQty:       in.MinQty,
			Suggested:    short,
			Price:        price,
			Cost:         int(math.Round(float64(price) * short)),
			Basis:        basis,
			Requested:    round3(requested[in.ID]),
			Every:        round3(fill.every),
			Deliveries:   fill.deliveries,
			ShelfLife:    round3(fill.shelfLife),
			StockOuts:    want.stockouts,
		}
		// The horizon and the rate only travel with a row the forecast had a
		// say in: printed beside a minimum-driven quantity they would look like
		// its reasoning, which they are not.
		if forecastable {
			row.Cover = cover
			row.Daily = round3(want.forecast(now, cover) / float64(cover))
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
