package repository

// ---- A product that is its own stock row ----
//
// ⚠️ **The whole shop/restaurant difference lives in this file, and nothing else
// changes.** A kitchen turns inputs into outputs: what is sold is a `menu_item`,
// what is stocked is an `ingredient`, and a tech card joins them. A shop sells
// the object it bought, so the two are one thing — but the stock module is built
// on `ingredientId` from end to end, and teaching all of it a second kind of key
// would be a rewrite of the one part of the product where a wrong number is
// silent. So the shape stays and the bookkeeping moves off the owner.
//
// ⚠️ **Here rather than in handlers because the seed needs it too.** A sample
// shop catalogue written straight into Mongo would be products that can be sold
// and cannot be counted — sellable, uncountable, and wrong in silence. One
// implementation, two callers.

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"restaurant-backend/internal/models"
)

// SyncProductStock creates or updates the stock row behind a product.
//
// ⚠️ **Called on every write to a menu item, and does nothing for a dish.** The
// alternative — calling it only when the flag is set — leaves a stale row behind
// a product whose flag was turned off, and stale stock is stock somebody orders
// against.
func SyncProductStock(ctx context.Context, s *Store, m *models.MenuItem) error {
	if !m.SellsItself {
		// ⚠️ The row is deliberately *not* deleted when the flag comes off: it
		// may carry purchases, write-offs and a counted balance, and deleting it
		// would take a real quantity of real goods out of the books to tidy up a
		// checkbox. It simply stops being maintained.
		return nil
	}

	unit := StockUnit(m.UnitCode)
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
		res, err := s.Ingredients.InsertOne(ctx, ing)
		if err != nil {
			return err
		}
		m.StockID, _ = res.InsertedID.(primitive.ObjectID)
	} else {
		// ⚠️ Only the fields a person can see on the product. The price, the
		// minimum and the placement belong to the stock screens and an owner may
		// have set them there; overwriting those from the catalogue would undo
		// somebody's work every time a name was corrected.
		_, err := s.Ingredients.UpdateOne(ctx,
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

// StockUnit turns the product's fiscal measure code into the store's own unit.
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
func StockUnit(code int) string {
	switch code {
	case 10, 11: // gram, kilogram
		return models.UnitKg
	case 41: // litre
		return models.UnitL
	default:
		return models.UnitPcs
	}
}
