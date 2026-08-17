package handlers

import (
	"context"
	"errors"
	"net/http"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Turning "the guest asked for this dish" into a receipt line.
//
// ⚠️ **Extracted rather than copied, for the reason the call centre was.**
// Three screens now build a line — the website, the operator's order form and
// the till — and every one of them must arrive at the same price for the same
// dish. A copy drifts silently and the symptom is the worst kind: the same
// basket costs one thing on the phone and another at the table, both look
// correct on their own screen, and the guest is the one who notices.
//
// Everything here is re-resolved against the live menu. The client says *which*
// dish and *which* options; it never says what they cost.
func (h *Handler) menuLine(
	ctx context.Context,
	req models.OrderItem,
	brand *primitive.ObjectID,
) (models.OrderItem, int, error) {
	var dbItem models.MenuItem
	if err := h.Store.Menu.FindOne(ctx, bson.M{"_id": req.MenuItemID}).Decode(&dbItem); err != nil {
		// The dish was removed or renamed since the cart was filled.
		name := req.Name
		if name == "" {
			name = "Taom"
		}
		return models.OrderItem{}, http.StatusBadRequest,
			errors.New(name + " menyuda topilmadi — savatni yangilang")
	}
	if !dbItem.IsAvailable {
		return models.OrderItem{}, http.StatusBadRequest,
			errors.New(dbItem.Name + " hozircha mavjud emas")
	}
	// One receipt, one brand. Taken from the dishes themselves rather than a
	// field the client sends: the cart is per-brand by construction, so the
	// menu is the honest source. Two kitchens cannot fill one receipt.
	if brand != nil {
		if brand.IsZero() {
			*brand = dbItem.BrandID
		} else if dbItem.BrandID != *brand {
			return models.OrderItem{}, http.StatusBadRequest,
				errors.New("savatda ikki xil brend taomi bor — alohida buyurtma bering")
		}
	}
	// Options are re-resolved against the menu: the client only says which
	// choice it picked, the price delta always comes from the DB.
	opts, err := resolveOptions(&dbItem, req.Options)
	if err != nil {
		return models.OrderItem{}, http.StatusBadRequest, err
	}
	unit := dbItem.Price
	for _, o := range opts {
		unit += o.PriceDelta
	}
	if unit < 0 {
		unit = 0
	}
	line := models.OrderItem{
		MenuItemID: dbItem.ID,
		Name:       dbItem.Name,
		Price:      unit,
		Qty:        req.Qty,
		Options:    opts,
		// Kept as typed, only trimmed and capped — a note the kitchen reads.
		Comment: clampText(req.Comment, 200),
	}
	// A combo carries its contents onto the receipt: "Oilaviy combo" alone is
	// not something a kitchen can cook from. Resolved here, against the live
	// menu, so a set whose dish was deleted or pulled cannot be sold.
	if dbItem.IsCombo() {
		res, err := h.resolveCombo(ctx, &dbItem, nil)
		if err != nil {
			return models.OrderItem{}, http.StatusInternalServerError, err
		}
		if res.Blocked != "" {
			return models.OrderItem{}, http.StatusBadRequest,
				errors.New(dbItem.Name + ": " + res.Blocked)
		}
		line.ComboItems = res.Contents
	}
	return line, http.StatusOK, nil
}

// menuLines resolves a whole cart and returns its subtotal and brand.
func (h *Handler) menuLines(
	ctx context.Context,
	reqs []models.OrderItem,
) (items []models.OrderItem, subtotal int, brandID primitive.ObjectID, status int, err error) {
	items = make([]models.OrderItem, 0, len(reqs))
	for _, req := range reqs {
		line, st, err := h.menuLine(ctx, req, &brandID)
		if err != nil {
			return nil, 0, primitive.NilObjectID, st, err
		}
		subtotal += line.Price * line.Qty
		items = append(items, line)
	}
	return items, subtotal, brandID, http.StatusOK, nil
}
