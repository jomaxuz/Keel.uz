package handlers

// ---- The order board ----
//
// The other half of what a television on a wall is for: in a fast-food room,
// which numbers are cooking and which are ready to collect.
//
// ⚠️ **Numbers, and nothing else.** No names, no items, no totals. This screen
// is read by everybody in the room including the people whose orders are not on
// it — a board that named customers would be a customer list on a wall, and one
// that listed dishes would tell forty strangers what the person at table six is
// eating. The number is what the receipt already shows, and it is enough.
//
// ⚠️ **"Ready" is a timestamp, not a status** (models.Order.ReadyAt) — the
// decision the kitchen screen made and the reason nothing here reads a status
// for it. What a status *is* used for is knowing an order has left: a collected
// or cancelled ticket must come off the wall.

import (
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// How long a ready number stays up after the kitchen finished it.
	//
	// ⚠️ **A cap, because nothing reliably takes it down.** A collected order
	// is marked so at the till, and on a busy evening it is not — so without a
	// window the board fills with numbers whose owners left an hour ago, and a
	// guest scanning it for their own gives up. Fifteen minutes is longer than
	// anybody waits at a counter and short enough that the wall stays readable.
	tvBoardReadyFor = 15 * time.Minute

	// What fits on a wall and can still be read across a room. Past this the
	// screen is a wall of digits nobody scans — and the numbers dropped are the
	// oldest, which are the ones already collected.
	tvBoardCookingMax = 24
	tvBoardReadyMax   = 12
)

// tvBoardTypes is whose orders appear on the wall.
//
// ⚠️ **Not delivery.** The board answers "is mine ready yet" for somebody
// standing in the room; a delivery customer is at home, and their number on a
// public screen is information the room has no use for. Their order is on the
// kitchen screen and in the courier's app, which is where it is acted on.
var tvBoardTypes = bson.A{"pickup", "dinein"}

// TVBoard is what a screen in board mode draws.
func (h *Handler) TVBoard(w http.ResponseWriter, r *http.Request) {
	screen, _, err := h.tvScreen(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	now := time.Now()

	// Cooking. ⚠️ The same filter as the kitchen pass, deliberately: a room
	// watching a number that the kitchen screen is not showing — or the other
	// way round — is two screens in one building disagreeing about one order,
	// and the guest is standing between them. `pending` is absent for the same
	// reason there; a pre-order's `queuedAt` is in the future and it appears by
	// itself when its lead time arrives.
	cooking, err := h.tvBoardNumbers(r, bson.M{
		"branchId": screen.BranchID,
		"type":     bson.M{"$in": tvBoardTypes},
		"status": bson.M{"$in": bson.A{
			models.StatusConfirmed, models.StatusPreparing}},
		"queuedAt": bson.M{"$ne": nil, "$lte": now},
		"readyAt":  nil,
	}, bson.D{{Key: "queuedAt", Value: 1}}, tvBoardCookingMax)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Ready, newest first: the number somebody is waiting for right now is the
	// one that just appeared, and it belongs where the eye lands.
	ready, err := h.tvBoardNumbers(r, bson.M{
		"branchId": screen.BranchID,
		"type":     bson.M{"$in": tvBoardTypes},
		"readyAt":  bson.M{"$gte": now.Add(-tvBoardReadyFor)},
		// ⚠️ Collected and cancelled come off immediately, without waiting for
		// the window: the till knowing an order has gone is better information
		// than a clock, and it is the case where the number on the wall would
		// otherwise send somebody to a counter for food they already have.
		"status": bson.M{"$nin": bson.A{
			models.StatusDelivered, models.StatusCancelled}},
	}, bson.D{{Key: "readyAt", Value: -1}}, tvBoardReadyMax)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"cooking": cooking,
		"ready":   ready,
		// ⚠️ Sent so the screen can tell fresh numbers from numbers it has been
		// holding since the wifi dropped — a board is the one thing on this
		// television that must go quiet rather than show something stale.
		"serverTime": now,
	})
}

// tvBoardNumbers reads one column of the board.
//
// ⚠️ A projection, not whole orders: everything else on an order document —
// the customer, the phone, the address, the total — is exactly what must not
// reach a device hanging in a public room. Narrowing it here means a later
// edit cannot widen it by accident.
func (h *Handler) tvBoardNumbers(r *http.Request, filter bson.M, sort bson.D, limit int64) ([]string, error) {
	cur, err := h.Store.Orders.Find(r.Context(), filter,
		options.Find().
			SetSort(sort).
			SetLimit(limit).
			SetProjection(bson.M{"number": 1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(r.Context())

	// ⚠️ An empty slice, never nil: Go marshals a nil slice as `null` and the
	// television maps over this. A quiet afternoon would blank the screen.
	out := []string{}
	for cur.Next(r.Context()) {
		var row struct {
			Number string `bson:"number"`
		}
		if err := cur.Decode(&row); err != nil || row.Number == "" {
			continue
		}
		out = append(out, row.Number)
	}
	return out, cur.Err()
}
