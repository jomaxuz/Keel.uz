package handlers

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
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
func (h *Handler) syncProductStock(ctx context.Context, m *models.MenuItem) error {
	if !m.SellsItself {
		// ⚠️ The row is deliberately *not* deleted when the flag comes off: it
		// may carry purchases, write-offs and a counted balance, and deleting it
		// would take a real quantity of real goods out of the books to tidy up a
		// checkbox. It simply stops being maintained.
		return nil
	}

	unit := stockUnitOf(m)
	now := time.Now()

	if m.StockID.IsZero() {
		ing := models.Ingredient{
			BrandID: m.BrandID,
			Name:    m.Name,
			Unit:    unit,
			// ⚠️ **The purchase price, not the sale price**, and it starts at
			// zero rather than at the shelf price. `Ingredient.Price` is what
			// the goods cost us; seeding it from what we charge would make the
			// first stock valuation and every margin read from it wrong, and
			// wrong in the flattering direction.
			Price:     0,
			CreatedAt: now,
			UpdatedAt: now,
		}
		res, err := h.Store.Ingredients.InsertOne(ctx, ing)
		if err != nil {
			return err
		}
		m.StockID = oidOf(res.InsertedID)
	} else {
		// ⚠️ Only the fields a person can see on the product. The price, the
		// minimum and the placement belong to the stock screens and an owner may
		// have set them there; overwriting those from the catalogue would undo
		// somebody's work every time a name was corrected.
		_, err := h.Store.Ingredients.UpdateOne(ctx,
			bson.M{"_id": m.StockID},
			bson.M{"$set": bson.M{"name": m.Name, "unit": unit, "updatedAt": now}})
		if err != nil {
			return err
		}
	}

	// ⚠️ **The tech card is written here, not by the owner.** It is what makes a
	// sale consume the goods through exactly the code a restaurant uses — one
	// unit of itself. Written every time, because a product whose flag was just
	// turned on has no card yet and one whose stock row was recreated has the
	// wrong one.
	m.Recipe = []models.RecipeLine{{IngredientID: m.StockID, Qty: 1}}
	return nil
}

// stockUnitOf turns the product's fiscal measure code into the store's own unit.
//
// ⚠️ **Two vocabularies for one fact, and neither is going away.** `unitCode` is
// the state classifier's — it goes on the receipt and the tax filing, and its
// numbers are not ours to choose. The store speaks `kg`/`l`/`pcs`, because that
// is what a purchase is written in and what `PerUnit` divides by. This is the
// only place they meet, so a code the classifier adds later is one line here
// rather than a wrong balance somewhere.
//
// ⚠️ Anything unrecognised is a piece, which is what an unset code already means
// and what almost everything in a shop is.
func stockUnitOf(m *models.MenuItem) string {
	switch m.UnitCode {
	case 10, 11: // gram, kilogram
		return models.UnitKg
	case 41: // litre
		return models.UnitL
	default:
		return models.UnitPcs
	}
}

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
