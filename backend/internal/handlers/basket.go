package handlers

import (
	"context"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// What a basket needs from a kitchen, and what a kitchen has run out of.
//
// Both questions used to be answered inline in CreateOrder, which was fine
// while the answer was only ever "refuse". It is not any more: the branch is
// now chosen with the basket in hand (see deliveryBranch), and the checkout
// says what is missing *before* the guest presses confirm rather than after —
// so three places need the same two answers and must not each work them out
// their own way.

// basketDishes lists every dish a basket actually requires: the lines
// themselves, plus the courses inside any combo.
//
// ⚠️ **The combo's contents count.** A set is only sellable where every dish in
// it is, so a branch missing one of its courses cannot fill the order however
// well it stocks the rest. Left out, the branch preference would happily route
// a basket to a kitchen that cannot cook half of it.
func (h *Handler) basketDishes(ctx context.Context, items []models.OrderItem) []primitive.ObjectID {
	out := make([]primitive.ObjectID, 0, len(items))
	seen := map[primitive.ObjectID]bool{}
	add := func(id primitive.ObjectID) {
		if id.IsZero() || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	for _, it := range items {
		add(it.MenuItemID)
		var dish models.MenuItem
		if err := h.Store.Menu.FindOne(ctx, bson.M{"_id": it.MenuItemID}).Decode(&dish); err != nil {
			continue
		}
		for _, member := range dish.ComboItems {
			add(member.MenuItemID)
		}
	}
	return out
}

// soldOutAt names what this branch has run out of, in words a guest reads.
//
// Names rather than ids because the answer is shown to somebody: "Lag'mon
// bugun tugadi" is actionable, a hex string is not. A dish reached only through
// a combo is named by itself, since that is the thing that ran out — but the
// set it came in is what the guest has in their basket, so both are worth
// having and the combo's own name comes first when it is the one stopped.
func (h *Handler) soldOutAt(
	ctx context.Context, branch *models.Branch, items []models.OrderItem,
) []string {
	if branch == nil {
		return nil
	}
	var out []string
	seen := map[primitive.ObjectID]bool{}
	name := func(id primitive.ObjectID, fallback string) {
		if seen[id] {
			return
		}
		seen[id] = true
		var dish models.MenuItem
		if err := h.Store.Menu.FindOne(ctx, bson.M{"_id": id}).Decode(&dish); err == nil && dish.Name != "" {
			out = append(out, dish.Name)
			return
		}
		if fallback == "" {
			fallback = "Taom"
		}
		out = append(out, fallback)
	}
	for _, it := range items {
		if branch.IsSoldOut(it.MenuItemID) {
			name(it.MenuItemID, it.Name)
			// The set itself is stopped; what is inside it is beside the point.
			continue
		}
		var dish models.MenuItem
		if err := h.Store.Menu.FindOne(ctx, bson.M{"_id": it.MenuItemID}).Decode(&dish); err != nil {
			continue
		}
		for _, member := range dish.ComboItems {
			if branch.IsSoldOut(member.MenuItemID) {
				name(member.MenuItemID, "")
			}
		}
	}
	return out
}
