package handlers

// ---- The online orders the counter has to do something about ----
//
// ⚠️ **A delivery order was invisible from the till, and money was owed on it.**
// The counter's list is built from `check.openedAt` — a check opened at a
// table — so an order taken on the website or in the bot had no check, no row,
// and no cashier who knew it existed. The cash came back in a courier's pocket
// at the end of a shift and was reconciled against nothing.
//
// ⚠️ **A separate list, never mixed into the tables.** A table is something you
// serve; an online order is something you either collect money for or do not.
// Putting twelve deliveries among the tables means a cashier looking for table
// six scrolls past them, which is how both lists stop being read.
//
// ⚠️ **The whole value is the sentence, not the row.** A cashier does not need
// another list of orders — the kitchen screen has one. What they need is what
// *they* have to do about each one, and that depends on a combination nobody
// should have to hold in their head: how it was paid, and whether it is
// delivered or collected.

import (
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Settlement is what the counter has to do about one online order.
type Settlement string

const (
	// Already paid, online, before it ever reached the kitchen. Nothing to do,
	// and saying so is the point: an order with no action is the commonest kind
	// and a cashier must be able to skip it without opening it.
	SettleNothing Settlement = "nothing"
	// The courier carries it out and brings the money back.
	SettleFromCourier Settlement = "from_courier"
	// The guest is coming to the counter and will pay there.
	SettleAtCounter Settlement = "at_counter"
	// ⚠️ Started online and never finished. This is the one worth a cashier's
	// attention: the kitchen may already be cooking, and the money is neither
	// in a courier's pocket nor in the bank. It is the state the old invisible
	// list hid best.
	SettleUnfinished Settlement = "unfinished"
)

// settlementOf decides what the counter owes on this order.
//
// ⚠️ **Paid is checked before anything else.** A guest who paid with Payme is
// settled whether the order is delivered, collected or cancelled halfway, and
// asking any further question about it can only produce a wrong answer.
func settlementOf(o *models.Order) Settlement {
	if paymentStatusOf(o) == models.PayPaid {
		return SettleNothing
	}
	// ⚠️ Cash and card are the same answer here, and that surprised me until I
	// wrote it down: both are money that has not moved yet, and both are
	// collected by whoever is standing in front of the guest. On a delivery
	// that is the courier — with notes or with the terminal they carry — and at
	// a pickup it is the cashier.
	switch o.Type {
	case "pickup":
		return SettleAtCounter
	case "delivery":
		return SettleFromCourier
	}
	// A dine-in order has a check and never reaches this list.
	return SettleNothing
}

// onlineRow is one order as the counter sees it.
type onlineRow struct {
	ID     string     `json:"id"`
	Number string     `json:"number"`
	Type   string     `json:"type"`
	Status string     `json:"status"`
	Total  int        `json:"total"`
	Method string     `json:"paymentMethod"`
	Paid   string     `json:"paymentStatus"`
	At     time.Time  `json:"at"`
	Settle Settlement `json:"settle"`

	// Who to hand it to or ring, kept short: a courier's name when one is
	// assigned, otherwise the guest's.
	Who   string `json:"who,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// StaffOnlineOrders is today's online orders for this branch.
func (h *Handler) StaffOnlineOrders(w http.ResponseWriter, r *http.Request) {
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return
	}
	// ⚠️ **Today, not everything.** A counter's question is "what is still
	// owed on this shift", and an order from a fortnight ago is an accounting
	// question with a report behind it. The list has to be short enough to be
	// read at a glance between two guests.
	from := time.Now().In(time.Local)
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.Local)

	cur, err := h.Store.Orders.Find(r.Context(), bson.M{
		"branchId":  s.BranchID,
		"type":      bson.M{"$in": []string{"delivery", "pickup"}},
		"createdAt": bson.M{"$gte": from},
		// ⚠️ A cancelled order owes nothing and its row would be a cashier
		// asking a courier for money nobody took.
		"status": bson.M{"$ne": models.StatusCancelled},
	}, options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(200))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cur.Close(r.Context())

	// Never nil: this list is empty on every quiet morning, and `null.length`
	// is the JSON trap this codebase has been bitten by twice.
	rows := []onlineRow{}
	owed := 0
	for cur.Next(r.Context()) {
		var o models.Order
		if cur.Decode(&o) != nil {
			continue
		}
		row := onlineRow{
			ID: o.ID.Hex(), Number: o.Number, Type: o.Type,
			Status: string(o.Status), Total: o.Total,
			Method: o.PaymentMethod, Paid: paymentStatusOf(&o),
			At: o.CreatedAt, Settle: settlementOf(&o),
			Who: o.Customer.Name, Phone: o.Customer.Phone,
		}
		if o.CourierName != "" {
			row.Who = o.CourierName
		}
		// ⚠️ Only what is genuinely still owed. An order already paid online
		// adds nothing, and including it would give the counter a total that
		// disagrees with the drawer by the size of a good day's card takings.
		if row.Settle != SettleNothing {
			owed += o.Total
		}
		rows = append(rows, row)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"orders": rows,
		// The one figure worth putting at the top: what should come back to
		// this counter before the shift closes.
		"owed": owed,
	})
}
