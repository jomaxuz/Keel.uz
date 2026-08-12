package handlers

import (
	"context"
	"errors"
	"fmt"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Combos: a fixed set of dishes sold together for one price.
//
// The set references its dishes rather than copying them, so renaming or
// reshooting a dish updates every combo it appears in. Two things follow from
// that and are handled here rather than in each caller:
//
//   - The "you save X" figure is **computed**, never stored. A stored copy goes
//     stale the moment a member dish is repriced, and a saving that is no longer
//     true is worse than no saving shown at all.
//   - A combo is unavailable as soon as one of its dishes is. Half a set is not
//     a set, and the kitchen must never be handed one it cannot assemble.

// comboResolution is everything a caller needs to know about one combo.
type comboResolution struct {
	// What the dishes cost bought separately.
	BasePrice int
	// Names and quantities, in the order the admin arranged them.
	Contents []models.OrderComboLine
	// Set when the combo cannot be ordered right now, with the reason.
	Blocked string
}

// resolveCombo loads a combo's member dishes and reports what it contains, what
// it would cost separately, and whether it can be ordered at this branch.
//
// `soldOut` may be nil, which skips the check entirely (the caller does not
// know which kitchen yet). A lens rather than a branch because the public menu
// is not always drawn against one: with several branches and no address given,
// the honest question is "is this stopped everywhere" — see soldoutlens.go.
func (h *Handler) resolveCombo(
	ctx context.Context, combo *models.MenuItem, soldOut soldOutLens,
) (comboResolution, error) {
	out := comboResolution{Contents: make([]models.OrderComboLine, 0, len(combo.ComboItems))}

	ids := make([]primitive.ObjectID, 0, len(combo.ComboItems))
	for _, line := range combo.ComboItems {
		ids = append(ids, line.MenuItemID)
	}
	cur, err := h.Store.Menu.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return out, err
	}
	var members []models.MenuItem
	if err := cur.All(ctx, &members); err != nil {
		return out, err
	}
	byID := make(map[primitive.ObjectID]*models.MenuItem, len(members))
	for i := range members {
		byID[members[i].ID] = &members[i]
	}

	for _, line := range combo.ComboItems {
		qty := line.Qty
		if qty < 1 {
			qty = 1
		}
		dish, ok := byID[line.MenuItemID]
		if !ok {
			// A member was deleted from the menu. The combo is unsellable until
			// the admin fixes it — silently dropping the dish would sell a set
			// that is missing a course.
			out.Blocked = "to'plamdagi taomlardan biri menyudan olib tashlangan"
			continue
		}
		out.Contents = append(out.Contents, models.OrderComboLine{Name: dish.Name, Qty: qty})
		out.BasePrice += dish.Price * qty

		if !dish.IsAvailable {
			out.Blocked = dish.Name + " hozircha mavjud emas"
		}
		if soldOut != nil && soldOut(dish.ID) {
			out.Blocked = dish.Name + " bugun tugadi"
		}
	}
	return out, nil
}

// decorateCombos fills in the computed combo fields for a list of menu items,
// and marks a combo sold out when any of its dishes is. One pass over the menu,
// so a page of 48 dishes does not turn into 48 round trips.
func (h *Handler) decorateCombos(
	ctx context.Context, items []models.MenuItem, soldOut soldOutLens,
) {
	for i := range items {
		if !items[i].IsCombo() {
			continue
		}
		res, err := h.resolveCombo(ctx, &items[i], soldOut)
		if err != nil {
			continue
		}
		items[i].ComboBasePrice = res.BasePrice
		items[i].ComboContents = res.Contents
		if res.Blocked != "" {
			// Presented to the guest the same way a sold-out dish is: visible on
			// the menu, not orderable today.
			items[i].SoldOut = true
		}
	}
}

// validateCombo checks a combo the admin is trying to save.
//
// The rule worth stating: a dish with a **required** option group cannot go into
// a fixed set. "2 × Lag'mon" is unambiguous; "2 × Lag'mon (which size?)" is not,
// and a fixed set has nowhere to ask. Sets that need a choice are a different
// feature (pick-one-from-each-group), deliberately not built yet.
func (h *Handler) validateCombo(ctx context.Context, combo *models.MenuItem) error {
	if !combo.IsCombo() {
		return nil
	}
	if len(combo.Options) > 0 {
		return errors.New("to'plamga variantlar qo'shib bo'lmaydi")
	}
	seen := map[primitive.ObjectID]bool{}
	for _, line := range combo.ComboItems {
		if line.Qty < 1 {
			return errors.New("to'plamdagi har bir taom soni kamida 1 bo'lishi kerak")
		}
		if seen[line.MenuItemID] {
			return errors.New("bitta taom to'plamda ikki marta ko'rsatilgan — sonini oshiring")
		}
		seen[line.MenuItemID] = true

		var dish models.MenuItem
		if err := h.Store.Menu.FindOne(ctx, bson.M{"_id": line.MenuItemID}).Decode(&dish); err != nil {
			return errors.New("to'plamdagi taom menyuda topilmadi")
		}
		if dish.IsCombo() {
			return errors.New("to'plam ichiga boshqa to'plamni qo'shib bo'lmaydi")
		}
		if dish.BrandID != combo.BrandID {
			return errors.New("to'plamga faqat shu brend menyusidagi taomlar qo'shiladi")
		}
		for _, g := range dish.Options {
			if g.Required && len(g.Choices) > 0 {
				return fmt.Errorf(
					"%s da majburiy tanlov bor (%s) — bunday taomni belgilangan to'plamga qo'shib bo'lmaydi",
					dish.Name, g.Name)
			}
		}
	}
	if len(combo.ComboItems) < 2 {
		return errors.New("to'plamda kamida ikkita taom bo'lishi kerak")
	}
	return nil
}
