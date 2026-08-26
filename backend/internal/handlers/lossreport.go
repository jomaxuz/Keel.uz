package handlers

// ---- Who takes money off tables, and how that compares ----
//
// Every fact this reads was already being recorded. A void carries who did it,
// who allowed it, why, and whether the food was thrown away. A close carries
// who closed it. Discounts now carry both names too. What was missing was
// anybody looking across them.
//
// ⚠️ **Nothing here is evidence of anything.** A void is normal, a discount is
// normal, and a cashier who does neither is probably not serving anybody. The
// single number worth reading is a person against their colleagues on the same
// shifts: forty voids is meaningless, forty when the others did four is a
// question. So every row is a share, and the report refuses to draw at all
// until there is somebody to compare against.
//
// ⚠️ **Deterrence, not detection.** This is a screen a restaurant shows its
// staff, and most of its value is spent before anybody reads it: a cashier who
// knows their void rate sits beside their colleagues' voids less. Building it
// as a secret investigation tool would forfeit the larger half of the benefit
// and make the smaller half adversarial.

import (
	"net/http"
	"sort"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// personTally is one member of staff across the period.
type personTally struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// What they handled, which is what everything else is divided by.
	Checks int   `json:"checks"`
	Sales  int64 `json:"sales"`

	// Lines taken off a check after the kitchen had made them.
	Voids      int   `json:"voids"`
	VoidValue  int64 `json:"voidValue"`
	VoidShare  int   `json:"voidShare"`  // per thousand of their own checks
	AuthedSelf int   `json:"authedSelf"` // voids they did on their own authority

	// Money taken off a bill by a person.
	Discounts     int   `json:"discounts"`
	DiscountValue int64 `json:"discountValue"`
	DiscountShare int   `json:"discountShare"` // per thousand of their own sales

	// ⚠️ Cash is separated because it is the only tender that can leave in a
	// pocket. A card refund goes back to a card.
	CashChecks int   `json:"cashChecks"`
	CashSales  int64 `json:"cashSales"`
}

// AdminLossReport is the counter, person by person.
func (h *Handler) AdminLossReport(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	scope, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	from, to := reportRange(r)
	filter := bson.M{"check.closedAt": bson.M{"$gte": from, "$lt": to}}
	for k, v := range scope {
		filter[k] = v
	}
	cur, err := h.Store.Orders.Find(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cur.Close(r.Context())

	tally := map[string]*personTally{}
	get := func(id primitive.ObjectID, name string) *personTally {
		key := id.Hex()
		if id.IsZero() {
			// ⚠️ Somebody with no id still gets a row, under their name. These
			// are the oldest checks, from before the till recorded ids — and
			// dropping them would quietly shrink the denominator every row is
			// divided by.
			key = "name:" + name
		}
		if t, ok := tally[key]; ok {
			return t
		}
		t := &personTally{ID: key, Name: name}
		tally[key] = t
		return t
	}

	for cur.Next(r.Context()) {
		var o models.Order
		if cur.Decode(&o) != nil || o.Check == nil {
			continue
		}
		closer := get(o.Check.ClosedByID, o.Check.ClosedBy)
		closer.Checks++
		closer.Sales += int64(o.Total)
		if o.PaymentMethod == "cash" {
			closer.CashChecks++
			closer.CashSales += int64(o.Total)
		}
		for _, it := range o.Items {
			if it.Void == nil {
				continue
			}
			// ⚠️ Counted against whoever *did* it, not whoever allowed it. The
			// override exists so a waiter can act without learning a manager's
			// PIN; blaming the manager for every void in the building is
			// exactly the outcome that design refused, and it would be a
			// strange thing for this report to reintroduce.
			v := get(it.Void.ByID, it.Void.By)
			v.Voids++
			v.VoidValue += int64(it.Price * it.Qty)
			if it.Void.AuthByID.IsZero() {
				v.AuthedSelf++
			}
		}
		for _, d := range o.Discounts {
			// ⚠️ A discount with no name on it was decided by a rule — a
			// promotion that matched, points spent. Nobody chose it, so nobody
			// is measured by it.
			if d.ByID.IsZero() && d.By == "" {
				continue
			}
			p := get(d.ByID, d.By)
			p.Discounts++
			p.DiscountValue += int64(d.Amount)
		}
	}

	rows := make([]personTally, 0, len(tally))
	for _, t := range tally {
		// ⚠️ Per thousand rather than per cent: a cashier who voids two lines
		// in four hundred checks is 0% at one decimal and 5‰ here, and the
		// difference between 5‰ and 40‰ is the entire content of this report.
		if t.Checks > 0 {
			t.VoidShare = int(int64(t.Voids) * 1000 / int64(t.Checks))
		}
		if t.Sales > 0 {
			t.DiscountShare = int(t.DiscountValue * 1000 / t.Sales)
		}
		rows = append(rows, *t)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].VoidShare != rows[j].VoidShare {
			return rows[i].VoidShare > rows[j].VoidShare
		}
		return rows[i].Name < rows[j].Name
	})

	httpx.JSON(w, http.StatusOK, map[string]any{
		"rows": rows,
		"from": from, "to": to,
		// ⚠️ **The report says when it cannot be read.** One person on the list
		// has no colleagues, so every share is 100% of itself and means
		// nothing — and a table of confident-looking numbers about one named
		// employee is worse than no table. The screen says so instead of
		// drawing.
		"comparable": len(rows) > 1,
	})
}

// reportRange is the window, defaulting to the last thirty days.
//
// ⚠️ **Thirty days, not a shift.** A single evening's voids are noise: a broken
// plate, a table that changed its mind, a kitchen that ran out. Only a month
// separates somebody's habit from somebody's Tuesday.
func reportRange(r *http.Request) (time.Time, time.Time) {
	to := time.Now()
	from := to.AddDate(0, 0, -30)
	if v := r.URL.Query().Get("from"); v != "" {
		if d, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
			from = d
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if d, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
			to = d.AddDate(0, 0, 1)
		}
	}
	return from, to
}
