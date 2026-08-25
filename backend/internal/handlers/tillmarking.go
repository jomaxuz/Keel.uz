package handlers

import (
	"context"

	"restaurant-backend/internal/marking"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Marking codes, where a check meets them.
//
// ⚠️ **Two gates, and they answer different questions.** Adding a marked dish
// asks for its code *there* — the cashier is holding the bottle and the scanner
// is in their other hand, which is the only moment scanning is free. Closing
// the check asks again, because a line can predate the flag being turned on and
// because the receipt is what the law is about: a bottle sold without its code
// filed is not withdrawn from circulation, and nothing later can put that
// right.

// markedItems says which of these menu items carry a marking code.
//
// ⚠️ Read from the menu rather than from the line, for the reason the ИКПУ is:
// "is this product marked" is a fact about the product, and the owner who ticks
// the box has to affect the bottle being rung up this minute.
func (h *Handler) markedItems(
	ctx context.Context, ids []primitive.ObjectID,
) (map[primitive.ObjectID]bool, error) {
	out := make(map[primitive.ObjectID]bool, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	cur, err := h.Store.Menu.Find(ctx,
		bson.M{"_id": bson.M{"$in": ids}, "marked": true})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var m models.MenuItem
		if err := cur.Decode(&m); err != nil {
			return nil, err
		}
		out[m.ID] = true
	}
	return out, cur.Err()
}

// markingRefusal checks a whole check's lines together.
//
// ⚠️ **Together is the point.** A per-line check cannot see the failure that
// actually happens: the pistol beeps, nobody is sure it took, the same bottle
// is scanned again — and two lines then withdraw one bottle while the second
// stays in circulation. The tax register accepts that receipt, so this is the
// only place it can be caught.
//
// Returns "" when the check may proceed.
func (h *Handler) markingRefusal(
	ctx context.Context, items []models.OrderItem,
) (string, error) {
	ids := make([]primitive.ObjectID, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.MenuItemID)
	}
	marked, err := h.markedItems(ctx, ids)
	if err != nil {
		return "", err
	}
	lines := make([]marking.Line, 0, len(items))
	names := make([]string, 0, len(items))
	for _, it := range items {
		// ⚠️ A voided line is not sold and is not filed, so it withdraws
		// nothing — asking it for a code would hold a check shut over a bottle
		// that went back in the fridge.
		if it.Void != nil {
			continue
		}
		lines = append(lines, marking.Line{
			Marked: marked[it.MenuItemID],
			Code:   it.MarkCode,
			Qty:    it.Qty,
		})
		names = append(names, it.Name)
	}
	if err := marking.CheckAll(lines); err != nil {
		// The dish's name is put in front of the reason: a cashier reading
		// "the code is already on this receipt" needs to know which bottle to
		// look at, and a check can hold a dozen lines.
		for i, l := range lines {
			if !l.Marked {
				continue
			}
			if e := marking.Check(l.Code); e != nil {
				return names[i] + ": " + e.Error(), nil
			}
		}
		return err.Error(), nil
	}
	return "", nil
}
