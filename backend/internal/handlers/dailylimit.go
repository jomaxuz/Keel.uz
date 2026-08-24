package handlers

// ---- "We cooked ten portions of osh today" ----
//
// A kitchen decides in the morning how much of something it is making, and the
// stop that follows is arithmetic nobody should have to do at eight in the
// evening. Until now the only way to act on it was for somebody to remember,
// count, and tap the stop list at the right moment — which is a job that gets
// done late, and being told a dish is off *after* ordering it is the complaint
// this whole area exists to prevent.
//
// ⚠️ **Counted from the orders, never from a counter.** How many were sold
// today is already written down. A number kept beside the limit would be a
// second copy of it, and it would drift the first time a check was cancelled,
// refunded, or moved to another branch — three ordinary events, none of which
// would think to decrement it. Recounting costs one aggregation at the moment a
// sale lands, which is not a hot path: the menu, the cart and the till read the
// stored list instead.
//
// ⚠️ **Applied after the sale, not before it.** The tenth portion is sold; it
// is the eleventh that is refused. Stopping at the tenth would mean a limit of
// ten sells nine, and the kitchen would learn to write eleven — which is how a
// number stops meaning what it says.

import (
	"context"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// applyDailyLimits recomputes which dishes today's limits have stopped.
//
// Called after a sale lands, from the two places a sale can land: an order the
// site or the operator created, and a check the till closed.
//
// ⚠️ **Never fails a sale.** The food is sold and the money is taken by the
// time this runs; a stop list that could not be updated is worth a log line,
// not a refused payment. The worst case is one dish selling past its limit
// until the next sale recomputes it.
func (h *Handler) applyDailyLimits(ctx context.Context, branchID primitive.ObjectID) {
	if branchID.IsZero() {
		return
	}
	var branch models.Branch
	if err := h.Store.Branches.FindOne(ctx, bson.M{"_id": branchID}).
		Decode(&branch); err != nil {
		return
	}
	// ⚠️ The common case, and it must cost nothing: almost no branch sets a
	// limit on anything, and this runs on every sale in the country.
	if len(branch.DailyLimits) == 0 {
		return
	}
	sold, err := h.soldToday(ctx, branchID)
	if err != nil {
		return
	}
	today := time.Now().Format("2006-01-02")
	stopped := []primitive.ObjectID{}
	for _, l := range branch.DailyLimits {
		if l.Limit <= 0 {
			continue
		}
		if sold[l.MenuItemID] >= l.Limit {
			stopped = append(stopped, l.MenuItemID)
		}
	}
	// ⚠️ Written whole, including when it is empty, and the date with it. This
	// list is derived — the sale that lifted a dish back over its limit (a
	// cancellation, an increased limit) has to be able to clear it, and an
	// `$addToSet` could only ever add.
	_, _ = h.Store.Branches.UpdateByID(ctx, branchID, bson.M{"$set": bson.M{
		"limitSoldOut": stopped,
		"limitDate":    today,
		"updatedAt":    time.Now(),
	}})
}

// soldToday counts portions sold at this branch since local midnight.
//
// ⚠️ **Local midnight, worked out in Go.** Mongo's date operators run in UTC
// unless told otherwise, so `$dateToString` here would start the day at seven
// in the evening Tashkent time — the whole dinner service would count against
// tomorrow, and the limit would lift in the middle of it. This codebase has
// been bitten by that twice; the boundary is computed here and passed as a
// plain timestamp.
//
// ⚠️ **Cancelled orders do not count**, and that is the only status filtered.
// A dish is sold when somebody orders it: waiting to be delivered or paid would
// have the limit lift and re-apply as the evening's orders moved through their
// statuses, stopping and unstopping a dish nobody had cooked more of.
func (h *Handler) soldToday(
	ctx context.Context, branchID primitive.ObjectID,
) (map[primitive.ObjectID]int, error) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	cur, err := h.Store.Orders.Aggregate(ctx, []bson.M{
		{"$match": bson.M{
			"branchId":  branchID,
			"createdAt": bson.M{"$gte": start},
			"status":    bson.M{"$ne": models.StatusCancelled},
		}},
		{"$unwind": "$items"},
		{"$group": bson.M{
			"_id":   "$items.menuItemId",
			"total": bson.M{"$sum": "$items.qty"},
		}},
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()

	out := map[primitive.ObjectID]int{}
	for cur.Next(ctx) {
		var row struct {
			ID    primitive.ObjectID `bson:"_id"`
			Total int                `bson:"total"`
		}
		if cur.Decode(&row) == nil && !row.ID.IsZero() {
			out[row.ID] = row.Total
		}
	}
	return out, nil
}

// ⚠️ **Combo sets are not expanded here, and that is deliberate.** A limit is a
// statement about a dish the kitchen cooks a batch of; a set is a bundle the
// site invented, and its members are counted by their own rows when they are
// sold on their own. Expanding a set would make "ten portions of osh" quietly
// include the osh inside every family combo — which is arguably right, and is
// certainly not what the person typing 10 was told. If a restaurant asks for
// it, it is `soldDishes` (handlers/costledger.go) rather than a second rule
// here.
