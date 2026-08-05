package handlers

import (
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OrderQuote is the checkout's preview of the bill: the same pipeline the order
// itself will run, so what the guest agrees to is what they are charged.
//
// It re-prices every line from the menu rather than trusting the cart, for the
// same reason CreateOrder does — a preview that flatters the browser's numbers
// would only disagree with the receipt a second later.

type orderQuoteRequest struct {
	Items     []models.OrderItem  `json:"items"`
	Type      string              `json:"type"`
	Address   models.OrderAddress `json:"address"`
	BranchID  string              `json:"branchId"`
	BrandID   string              `json:"brandId"`
	PromoCode string              `json:"promoCode"`
	UsePoints int                 `json:"usePoints"`
}

func (h *Handler) OrderQuote(w http.ResponseWriter, r *http.Request) {
	var req orderQuoteRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// Whoever is signed in on the site, if anyone.
	userID, _ := h.optionalUserID(r)
	h.quote(w, r, req, userID)
}

// AdminOrderQuote is the same preview for an operator taking the order over the
// phone. It exists because the customer is not the one holding the browser: the
// per-customer half of the price — their points balance, "first order only"
// codes — hangs off an account the panel names rather than a token it carries.
//
// An operator who cannot read the total back to the caller has to guess it, and
// a guessed total is an argument at the door.
func (h *Handler) AdminOrderQuote(w http.ResponseWriter, r *http.Request) {
	var req adminQuoteRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var userID primitive.ObjectID
	if id, err := objectID(req.UserID); err == nil {
		userID = id
	}
	h.quote(w, r, req.orderQuoteRequest, userID)
}

type adminQuoteRequest struct {
	orderQuoteRequest
	// Which customer this basket is for. Empty is fine — a first-time caller
	// has no account yet, and prices the same as any guest.
	UserID string `json:"userId"`
}

// quote runs the pricing pipeline for a basket that has not been placed yet.
func (h *Handler) quote(
	w http.ResponseWriter, r *http.Request, req orderQuoteRequest, userID primitive.ObjectID,
) {
	if req.Type == "" {
		req.Type = "delivery"
	}
	brandID, err := h.brandIDFrom(r, req.BrandID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Prices come from the menu, never from the cart.
	subtotal := 0
	items := make([]models.OrderItem, 0, len(req.Items))
	for _, it := range req.Items {
		var dish models.MenuItem
		if err := h.Store.Menu.FindOne(r.Context(),
			bson.M{"_id": it.MenuItemID}).Decode(&dish); err != nil {
			continue // a dish that vanished; CreateOrder will say so properly
		}
		unit := dish.Price
		for _, o := range it.Options {
			// Deltas are re-resolved at order time; for a preview the client's
			// selection is close enough and never becomes the charged price.
			unit += o.PriceDelta
		}
		if unit < 0 {
			unit = 0
		}
		qty := it.Qty
		if qty < 1 {
			qty = 1
		}
		items = append(items, models.OrderItem{
			MenuItemID: dish.ID, Name: dish.Name, Price: unit, Qty: qty,
		})
		subtotal += unit * qty
	}

	// Which kitchen, and what it charges to carry the order.
	var branch *models.Branch
	deliveryFee, minOrder := 0, 0
	available := true
	if req.Type == "delivery" {
		b, quote, err := h.deliveryBranch(r, brandID, req.Address.Lat, req.Address.Lng, subtotal)
		if err != nil {
			available = false
			minOrder = h.minOrderOf(r, brandID)
		} else {
			branch, deliveryFee, minOrder = b, quote.Fee, quote.MinOrder
		}
	} else {
		var id primitive.ObjectID
		if req.BranchID != "" {
			id, _ = objectID(req.BranchID)
		}
		if !id.IsZero() {
			branch, _ = h.branchByID(r, id)
		} else {
			branch, _ = h.defaultBranch(r, brandID)
		}
	}

	var branchID primitive.ObjectID
	if branch != nil {
		branchID = branch.ID
	}
	price, err := h.computePrice(r.Context(), priceInput{
		Items:        items,
		Subtotal:     subtotal,
		Type:         req.Type,
		BrandID:      brandID,
		BranchID:     branchID,
		UserID:       userID,
		Code:         req.PromoCode,
		UsePoints:    req.UsePoints,
		DeliveryFee:  deliveryFee,
		ItemCategory: h.itemCategories(r.Context(), items),
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := map[string]any{
		"subtotal":      price.Subtotal,
		"discounts":     price.Discounts,
		"discountTotal": price.DiscountTotal,
		"deliveryFee":   price.DeliveryFee,
		"total":         price.Total,
		"codeError":     price.CodeError,
		"codeApplied":   price.CodeApplied,
		"pointsSpent":   price.PointsSpent,
		"pointsBalance": price.PointsBalance,
		"pointsMax":     price.PointsMax,
		"pointsEarn":    price.PointsEarn,
		"available":     available,
		"minOrder":      minOrder,
		// The minimum is measured against what the restaurant receives for the
		// food, so a code that drops the basket under it blocks the order — the
		// checkout needs both numbers to explain that.
		"payableSubtotal": price.Subtotal - price.DiscountTotal,
		"belowMinimum":    req.Type == "delivery" && price.Subtotal-price.DiscountTotal < minOrder,
	}
	if branch != nil {
		resp["branchId"] = branch.ID
		resp["branchName"] = branch.Name
		resp["prepMinutes"] = branch.PrepMinutes
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// ---- Public: what is on offer right now ----

// GetPromotions lists the automatic campaigns a guest can see, so the site can
// show "what's on" without anyone having to type anything. Codes are never
// listed: a code nobody was given is not a promotion, it is a leak.
func (h *Handler) GetPromotions(w http.ResponseWriter, r *http.Request) {
	scope, _, err := h.publicScope(r)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	filter := scope.brandFilter(bson.M{
		"isActive": true,
		"trigger":  string(models.TriggerAuto),
	})
	cur, err := h.Store.Promotions.Find(r.Context(), filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var all []models.Promotion
	if err := cur.All(r.Context(), &all); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	var branchID primitive.ObjectID
	if branch, err := h.bookingBranch(r, r.URL.Query().Get("branchId")); err == nil {
		branchID = branch.ID
	}
	out := make([]models.Promotion, 0, len(all))
	for _, p := range all {
		// Order type is left open here: the list answers "what is on today",
		// and the checkout decides what applies to this particular basket.
		if promotionLive(&p, branchID, "", time.Now()) {
			p.UsedCount = 0 // not the guest's business
			out = append(out, p)
		}
	}
	httpx.JSON(w, http.StatusOK, out)
}
