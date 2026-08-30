package handlers

// ---- The tech card as its own thing ----
//
// ⚠️ **The card was edited on the dish, and that is the wrong place for it.**
// A dish form is opened to change a price, a photo, a description — weekly, by
// whoever is looking after the menu. A card is written once, from a printed
// sheet, by whoever knows what goes in the pot, and it is the same work whether
// the thing being described is a plate that gets sold or a bucket of sauce that
// never does. Keeping the two together had two costs:
//
//  1. **Half the cards had nowhere to live.** A prep item — sushi rice, a
//     sauce, a dough — is not on the menu, so its card was hidden inside the
//     ingredient form, under a collapsed section, on a screen called "the
//     shopping list". Restaurants did not find it, and wrote the rice into
//     forty dishes by hand instead. That is the copy that stops agreeing.
//  2. **No screen could answer "which dishes are costed".** The answer was one
//     dish per modal, and the menu is two hundred modals long.
//
// So the card gets a screen, in the store, beside the deliveries and the count
// — and the dish form keeps only what a dish form is for. This is how iiko has
// always had it, and the reason is the same one.
//
// ⚠️ **This file adds one endpoint, not a second costing engine.** The screen
// reads dishes from `/admin/menu` and preps from `/admin/ingredients` — both
// already carry the card and what it works out to, priced by the one resolver
// in ingredients.go. A `/tech-cards` list that recomputed either would be the
// second implementation the cards exist to prevent.

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// AdminSaveDishCard writes one dish's tech card and nothing else.
//
// ⚠️ **A `$set` of one field, deliberately, where `UpdateMenuItem` replaces the
// whole document.** The card screen holds a dish it read minutes ago and knows
// nothing about its options, its combo contents or its fiscal codes; posting
// that back as a whole dish would let a card edit silently revert a price
// somebody changed on the menu screen in between. One field in, one field
// written — and the two screens can be open at once.
func (h *Handler) AdminSaveDishCard(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	// ⚠️ A pointer here too, for the reason the whole file exists: a body
	// without the field is not "clear the card".
	var in struct {
		Recipe *[]models.RecipeLine `json:"recipe"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Recipe == nil {
		httpx.Error(w, http.StatusBadRequest, "recipe kerak")
		return
	}
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// ⚠️ **The brand is in the filter, not in a check after the read.** A card
	// is the one document in this product that can be edited to make a theft
	// arithmetically invisible (see recipechange.go), so a manager reaching it
	// by id across a brand boundary is exactly the shape that must not work.
	// Out of scope is 404: a manager should not learn the dish exists.
	filter := scope.brandFilter(bson.M{"_id": id})

	var dish models.MenuItem
	if err := h.Store.Menu.FindOne(r.Context(), filter).Decode(&dish); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	// ⚠️ A combo has no card of its own: what it costs is what its members
	// cost, and a card here would be counted **beside** them — the set would
	// consume its own ingredients and its members' as well. The menu screen
	// already refuses to call a combo uncosted for the same reason.
	if dish.IsCombo() {
		httpx.Error(w, http.StatusBadRequest, "to'plamning o'z texkartasi yo'q")
		return
	}

	lines := normalizeRecipe(*in.Recipe)
	// ⚠️ Read before the write, exactly as UpdateMenuItem does it: afterwards
	// there is nothing left to compare against, and the moment a norm goes up
	// is the only moment it is visible at all.
	changes := h.recipeDiff(r.Context(), id, lines)
	set := bson.M{"recipe": lines, "updatedAt": time.Now()}
	if _, err := h.Store.Menu.UpdateOne(r.Context(), filter, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	dish.Recipe = lines
	h.logAction(r, ActMenuUpdate, "menu", id.Hex(), dish.Name, describeRecipeDiff(changes))
	h.alertOnRecipeIncrease(r, dish, changes)
	httpx.JSON(w, http.StatusOK,
		h.pricedCards(r.Context(), []menuItemIO{withCost(dish)})[0])
}
