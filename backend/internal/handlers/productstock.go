package handlers

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
	"restaurant-backend/internal/repository"
)

// ---- A product that is its own stock row ----
//
// ⚠️ **The whole shop/restaurant difference lives in this file, and nothing else
// changes.** A kitchen turns inputs into outputs: what is sold is a `menu_item`,
// what is stocked is an `ingredient`, and a tech card joins them. A shop sells
// the object it bought, so the two are one thing — but the stock module is built
// on `ingredientId` from end to end (balances, purchases, write-offs, transfers,
// stocktakes, placements, the shopping list), and teaching all of it a second
// kind of key would be a rewrite of the one part of the product where a wrong
// number is silent.
//
// So the shape stays and the *bookkeeping* moves off the owner. A product that
// sells itself gets an ingredient created and kept in step by the server, and a
// one-line tech card saying "one unit of itself". Purchases go against that row,
// balances count it, a sale consumes it — every one of them running exactly the
// code it runs for a restaurant.
//
// ⚠️ **Two documents still exist, and that is the honest cost.** What changed is
// that nobody maintains the second one: it has no screen, no name of its own and
// no way to drift from the product it belongs to.

// syncProductStock creates or updates the stock row behind a product.
//
// ⚠️ **Called on every write to a menu item, and does nothing for a dish.** The
// alternative — calling it only when the flag is set — leaves a stale row behind
// a product whose flag was turned off, and stale stock is stock somebody orders
// against.
//
// ⚠️ **The work itself lives in the repository**, because the seed needs it too:
// a sample shop catalogue written straight into Mongo would be products that
// can be sold and cannot be counted, which is the one kind of wrong number this
// product never lets through. One implementation, two callers.
func (h *Handler) syncProductStock(ctx context.Context, m *models.MenuItem) error {
	return repository.SyncProductStock(ctx, h.Store, m)
}

// stockUnitOf is the store's own unit for a product — see repository.StockUnit.
func stockUnitOf(m *models.MenuItem) string { return repository.StockUnit(m.UnitCode) }

// keepStockID holds on to the stock row across a whole-document save.
//
// ⚠️ **The same shape as `keepRecipe`, and for the same reason.** The panel does
// not send this field — it has no screen — and `UpdateMenuItem` replaces the
// whole document, so without this a price change would drop the link and the
// next save would create a second stock row. The purchases, the write-offs and
// the counted balance would all stay on the first one, and the shelf would show
// zero for goods that are physically there.
func (h *Handler) keepStockID(
	ctx context.Context, id primitive.ObjectID, sent primitive.ObjectID,
) primitive.ObjectID {
	if !sent.IsZero() || id.IsZero() {
		return sent
	}
	var existing models.MenuItem
	if err := h.Store.Menu.FindOne(ctx, bson.M{"_id": id}).Decode(&existing); err != nil {
		return sent
	}
	return existing.StockID
}

// productStockNames maps stock rows back to the products that own them.
//
// ⚠️ Used by the stock screens so a shop's balance list reads in the owner's own
// words rather than showing a second set of names they never typed.
func (h *Handler) productStockNames(
	ctx context.Context, brand primitive.ObjectID,
) (map[primitive.ObjectID]string, error) {
	filter := bson.M{"sellsItself": true, "stockId": bson.M{"$ne": nil}}
	if !brand.IsZero() {
		filter["brandId"] = brand
	}
	cur, err := h.Store.Menu.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	var items []models.MenuItem
	if err := cur.All(ctx, &items); err != nil {
		return nil, err
	}
	out := make(map[primitive.ObjectID]string, len(items))
	for _, it := range items {
		out[it.StockID] = it.Name
	}
	return out, nil
}
