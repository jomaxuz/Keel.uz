package handlers

import (
	"context"
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The kitchen screen (KDS).
//
// The panel's order list is the **owner's** screen: filters, receipts, money,
// customer history. A cook standing at a pass with wet hands needs a different
// thing entirely — what to cook, in what order, how long it has been waiting,
// and one button. Handing them the owner's screen is how a restaurant ends up
// with the tablet permanently on the orders page, nobody touching it, and the
// statuses moved later from the office.
//
// Three decisions shape this:
//
//   - **It authenticates as staff, not as an admin.** The tablet by the pass is
//     shared and never logs out. An owner token on it would be the whole
//     business — settings, customers, payments — left unlocked on a shelf.
//     A staff token can see this screen and nothing else.
//   - **The branch comes from the employee, never from the request.** A cook in
//     Chilonzor cannot ask for Yunusobod's tickets by editing a URL, and no
//     screen anywhere needs them to.
//   - **Unpaid orders are not shown.** `queuedAt` is "when this became the
//     kitchen's problem" — the instant it was placed for cash, the instant the
//     bank confirmed for a card. Cooking to an order the bank never confirms is
//     the exact failure that timestamp exists to prevent.
type kitchenTicket struct {
	ID     string             `json:"id"`
	Number string             `json:"number"`
	Status models.OrderStatus `json:"status"`
	// delivery | pickup | dinein, and the table when it is dine-in: a cook
	// plates the same food differently for a tray and for a courier bag.
	Type        string `json:"type"`
	TableNumber string `json:"tableNumber,omitempty"`
	// What the guest asked for, with their line comments — the one field on
	// this screen that changes what is cooked.
	Items []models.OrderItem `json:"items"`
	// The guest's note on the address — "3rd floor", but also, often enough,
	// something the kitchen needs. Shown because a cook reading a ticket should
	// never have to wonder whether there was more of it somewhere else.
	Comment string `json:"comment,omitempty"`
	// When the kitchen became responsible, and how long ago that was. The age
	// is computed here rather than in the browser: a tablet with a wrong clock
	// is common, and the number this screen is judged by must not depend on it.
	QueuedAt   time.Time `json:"queuedAt"`
	WaitingMin int       `json:"waitingMin"`
}

// StaffKitchen lists what this branch has to cook right now.
func (h *Handler) StaffKitchen(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	ctx := r.Context()

	filter := bson.M{
		"branchId": s.BranchID,
		// Confirmed and preparing only. `pending` is deliberately absent: an
		// order nobody has accepted yet may still be refused, and a kitchen
		// that starts on it has already spent the food.
		"status":   bson.M{"$in": bson.A{models.StatusConfirmed, models.StatusPreparing}},
		"queuedAt": bson.M{"$ne": nil},
		"readyAt":  nil,
	}
	cur, err := h.Store.Orders.Find(ctx, filter,
		// Oldest first, always. A kitchen screen sorted any other way quietly
		// starves the order that has been waiting longest, which is the one
		// about to become a complaint.
		options.Find().SetSort(bson.D{{Key: "queuedAt", Value: 1}}).SetLimit(60))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cur.Close(ctx)

	now := time.Now()
	out := []kitchenTicket{}
	for cur.Next(ctx) {
		var o models.Order
		if err := cur.Decode(&o); err != nil {
			continue
		}
		queued := o.CreatedAt
		if o.QueuedAt != nil {
			queued = *o.QueuedAt
		}
		queued = queued.In(time.Local)
		// Never nil: Go marshals a nil slice as `null` and the screen maps over
		// this. One order written without lines would blank the whole pass — the
		// same shape of bug that took out the console's customer card.
		items := o.Items
		if items == nil {
			items = []models.OrderItem{}
		}
		out = append(out, kitchenTicket{
			ID:          o.ID.Hex(),
			Number:      o.Number,
			Status:      o.Status,
			Type:        o.Type,
			TableNumber: o.TableNumber,
			Items:       items,
			Comment:     o.Address.Comment,
			QueuedAt:    queued,
			WaitingMin:  int(now.Sub(queued).Minutes()),
		})
	}
	httpx.JSON(w, http.StatusOK, out)
}

type kitchenActionRequest struct {
	// start | ready. Two words rather than a status, because "ready" is not one
	// — see models.Order.ReadyAt.
	Action string `json:"action" validate:"required,oneof=start ready"`
}

// StaffKitchenAction is the one button on the ticket.
func (h *Handler) StaffKitchenAction(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var req kitchenActionRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now()
	set := bson.M{"status": models.StatusPreparing, "updatedAt": now}
	if req.Action == "ready" {
		set["readyAt"] = now
	}

	// The filter is the permission: this employee's branch, an order the
	// kitchen may still act on, and one that is actually queued. A cook cannot
	// reach another branch's ticket even with a valid id, because the id alone
	// never selects a document here.
	filter := bson.M{
		"_id":      id,
		"branchId": s.BranchID,
		"status":   bson.M{"$in": bson.A{models.StatusConfirmed, models.StatusPreparing}},
		"queuedAt": bson.M{"$ne": nil},
	}
	update := bson.M{"$set": set}
	if req.Action == "start" {
		// Narrowed to `confirmed`, so only the **first** press writes history.
		// A cook pressing the button twice is ordinary (a wet finger, a screen
		// that did not visibly react), and a timeline carrying four "preparing"
		// entries answers "when did this start" worse than one carrying a
		// single entry.
		filter["status"] = models.StatusConfirmed
		update["$push"] = bson.M{"statusHistory": bson.M{
			"$each": bson.A{models.StatusEvent{Status: models.StatusPreparing, At: now}},
			// Nothing is dropped; the slice bound is here to keep a stuck
			// screen from growing the document without limit.
			"$slice": -40,
		}}
	}

	res, err := h.Store.Orders.UpdateOne(r.Context(), filter, update)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.MatchedCount == 0 {
		// Two very different situations reach this point and only one is a
		// problem. A second press of "start" is not an error — the ticket is
		// already where the cook wants it — and telling them otherwise teaches
		// them the screen is unreliable.
		if req.Action == "start" && h.orderIsPreparing(r.Context(), id, s.BranchID) {
			httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		// The real case is the honest race: the office cancelled the order
		// while the cook was reaching for the button. Said plainly, because the
		// ticket is about to disappear from their screen either way.
		httpx.Error(w, http.StatusConflict, "bu buyurtma o'zgargan — ro'yxat yangilandi")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) orderIsPreparing(ctx context.Context, id, branchID primitive.ObjectID) bool {
	n, err := h.Store.Orders.CountDocuments(ctx, bson.M{
		"_id": id, "branchId": branchID, "status": models.StatusPreparing,
	})
	return err == nil && n > 0
}
