package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// The one place an order's money is decided.
//
// Four things can take money off a bill — a combo's own price, an automatic
// campaign, a typed promo code and (soon) loyalty points — and every one of them
// is a chance to reach zero by accident. So there is exactly one pipeline, it
// runs in a fixed order, and each step can only ever remove what is still left:
//
//	lines (combos already priced)      → subtotal
//	  → best automatic campaign
//	  → promo code
//	  → delivery fee (a free-delivery promo zeroes it)
//	= total, never below zero at any step
//
// Both the checkout preview and order creation call this. The preview is only a
// preview: CreateOrder runs the whole thing again and never trusts a number the
// browser sends back.

// Only one automatic campaign applies, the one worth most to the guest.
//
// Two overlapping campaigns the owner forgot about — "−20% on Tuesdays" and
// "−30% all July" — would otherwise stack to half price on a Tuesday in July.
// Picking the best single one keeps the promise generous and bounded.

type priceInput struct {
	Items    []models.OrderItem
	Subtotal int
	Type     string // delivery | pickup | dinein
	BrandID  primitive.ObjectID
	BranchID primitive.ObjectID
	UserID   primitive.ObjectID
	// The code the guest typed, if any.
	Code string
	// How many loyalty points the guest wants to put towards this order. Capped
	// here against their balance and the company's ceiling — the browser only
	// asks, it never decides.
	UsePoints int
	// Delivery fee as the zones/radius model priced it, before any promo.
	DeliveryFee int
	Now         time.Time
	// Which category each ordered dish belongs to, so a category-scoped
	// campaign can find its lines. Built per request by the caller — a map on
	// the handler would be shared state across concurrent orders.
	ItemCategory map[primitive.ObjectID]primitive.ObjectID
}

type priceResult struct {
	Subtotal      int                    `json:"subtotal"`
	Discounts     []models.OrderDiscount `json:"discounts"`
	DiscountTotal int                    `json:"discountTotal"`
	DeliveryFee   int                    `json:"deliveryFee"`
	Total         int                    `json:"total"`
	// Why the typed code was not applied. Empty when it was, or when none was
	// typed. Never fatal: an order with a mistyped code is still an order.
	CodeError string `json:"codeError,omitempty"`
	// True when the code the guest typed did apply — the field the checkout
	// turns green.
	CodeApplied bool `json:"codeApplied"`

	// ---- Loyalty ----
	// Points actually put towards this order, and the balance behind it.
	PointsSpent   int `json:"pointsSpent"`
	PointsBalance int `json:"pointsBalance"`
	// The most that could be applied here, so the checkout can offer it.
	PointsMax int `json:"pointsMax"`
	// What this order would earn once delivered. A promise, not a movement.
	PointsEarn int `json:"pointsEarn"`
}

// computePrice applies every discount to one order and returns the breakdown.
func (h *Handler) computePrice(ctx context.Context, in priceInput) (priceResult, error) {
	res := priceResult{
		Subtotal:    in.Subtotal,
		DeliveryFee: in.DeliveryFee,
		Discounts:   []models.OrderDiscount{},
	}
	if in.Now.IsZero() {
		in.Now = time.Now()
	}

	promos, err := h.livePromotions(ctx, in)
	if err != nil {
		return res, err
	}

	// What is still discountable. Each step eats into this, so three 50% rules
	// can never add up to more than the order is worth.
	remaining := in.Subtotal
	freeDelivery := false

	apply := func(p *models.Promotion, code string) int {
		base := h.discountBase(p, in.Items, remaining, in.ItemCategory)
		amount := 0
		switch p.Kind {
		case models.PromoFreeDelivery:
			if in.Type != "delivery" || res.DeliveryFee <= 0 || freeDelivery {
				return 0
			}
			amount = res.DeliveryFee
			freeDelivery = true
			res.DeliveryFee = 0
		case models.PromoPercent:
			amount = base * p.Value / 100
			if p.MaxDiscount > 0 && amount > p.MaxDiscount {
				amount = p.MaxDiscount
			}
		default: // fixed
			amount = p.Value
		}
		if p.Kind != models.PromoFreeDelivery {
			// Never more than is left to take.
			if amount > remaining {
				amount = remaining
			}
			if amount < 0 {
				amount = 0
			}
			remaining -= amount
		}
		if amount == 0 {
			return 0
		}
		res.Discounts = append(res.Discounts, models.OrderDiscount{
			PromotionID: p.ID,
			Name:        p.Name,
			Kind:        p.Kind,
			Trigger:     p.Trigger,
			Code:        code,
			Amount:      amount,
		})
		res.DiscountTotal += amount
		return amount
	}

	// 1. The best automatic campaign.
	if best := h.bestAuto(promos, in, remaining); best != nil {
		apply(best, "")
	}

	// 2. The typed code.
	if typed := strings.ToUpper(strings.TrimSpace(in.Code)); typed != "" {
		p, reason := h.findCode(ctx, promos, typed, in, remaining)
		switch {
		case p == nil:
			res.CodeError = reason
		default:
			if apply(p, typed) > 0 {
				res.CodeApplied = true
			} else {
				// Matched every rule but was worth nothing here — say so rather
				// than showing a green tick next to a zero.
				res.CodeError = "bu promokod shu buyurtmaga chegirma bermaydi"
			}
		}
	}

	// 3. Loyalty points, last of the three and only against what is left.
	//
	// Deliberately after the discounts: points are the guest's own money, and
	// spending them on an amount a campaign was about to take off anyway would
	// quietly burn a balance for nothing.
	l := h.loyaltySettings(ctx)
	res.PointsBalance = h.balanceOf(ctx, in.UserID)
	res.PointsMax = maxRedeemable(l, res.PointsBalance, remaining)
	if in.UsePoints > 0 && res.PointsMax > 0 {
		spend := in.UsePoints
		if spend > res.PointsMax {
			spend = res.PointsMax
		}
		if spend > 0 {
			res.PointsSpent = spend
			remaining -= spend
		}
	}

	// What the order will be worth in cashback once it is delivered. Money
	// only — paying with points must never earn points.
	if l.Enabled && l.EarnPercent > 0 {
		base := res.Subtotal - res.DiscountTotal - res.PointsSpent
		if base >= l.MinOrderToEarn && base > 0 {
			res.PointsEarn = base * l.EarnPercent / 100
		}
	}

	res.Total = remaining + res.DeliveryFee
	if res.Total < 0 {
		res.Total = 0
	}
	return res, nil
}

// discountBase is the amount a percentage is taken from: the whole remaining
// order, or only the lines the campaign covers.
func (h *Handler) discountBase(
	p *models.Promotion, items []models.OrderItem, remaining int,
	itemCategory map[primitive.ObjectID]primitive.ObjectID,
) int {
	if p.Scope != models.ScopeCategory && p.Scope != models.ScopeItems {
		return remaining
	}
	ids := map[primitive.ObjectID]bool{}
	for _, id := range p.MenuItemIDs {
		ids[id] = true
	}
	base := 0
	for _, it := range items {
		if p.Scope == models.ScopeItems && ids[it.MenuItemID] {
			base += it.Price * it.Qty
		}
		if p.Scope == models.ScopeCategory &&
			inCategories(itemCategory[it.MenuItemID], p.CategoryIDs) {
			base += it.Price * it.Qty
		}
	}
	// Earlier discounts already took money off; a scoped rule cannot hand back
	// more than the order still has.
	if base > remaining {
		base = remaining
	}
	return base
}

func inCategories(cat primitive.ObjectID, categories []primitive.ObjectID) bool {
	if cat.IsZero() {
		return false
	}
	for _, c := range categories {
		if c == cat {
			return true
		}
	}
	return false
}

// itemCategories maps each ordered dish to its category, for category-scoped
// campaigns. One query per order rather than one per line.
func (h *Handler) itemCategories(
	ctx context.Context, items []models.OrderItem,
) map[primitive.ObjectID]primitive.ObjectID {
	ids := make([]primitive.ObjectID, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.MenuItemID)
	}
	out := map[primitive.ObjectID]primitive.ObjectID{}
	cur, err := h.Store.Menu.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return out
	}
	var dishes []models.MenuItem
	if err := cur.All(ctx, &dishes); err != nil {
		return out
	}
	for _, d := range dishes {
		out[d.ID] = d.CategoryID
	}
	return out
}

// livePromotions loads the campaigns and codes that could apply to this order:
// right brand, right branch, active, inside their window.
func (h *Handler) livePromotions(ctx context.Context, in priceInput) ([]models.Promotion, error) {
	filter := bson.M{"isActive": true}
	if !in.BrandID.IsZero() {
		filter["brandId"] = in.BrandID
	}
	cur, err := h.Store.Promotions.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	var all []models.Promotion
	if err := cur.All(ctx, &all); err != nil {
		return nil, err
	}

	out := make([]models.Promotion, 0, len(all))
	for _, p := range all {
		if promotionLive(&p, in.BranchID, in.Type, in.Now) {
			out = append(out, p)
		}
	}
	return out, nil
}

// promotionLive answers "is this rule running right now, here, for this kind of
// order?" — everything except the money and the per-customer limits.
func promotionLive(p *models.Promotion, branchID primitive.ObjectID, orderType string, now time.Time) bool {
	if !p.IsActive {
		return false
	}
	if len(p.BranchIDs) > 0 && !branchID.IsZero() {
		found := false
		for _, b := range p.BranchIDs {
			if b == branchID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if p.StartsAt != nil && now.Before(*p.StartsAt) {
		return false
	}
	if p.EndsAt != nil && now.After(*p.EndsAt) {
		return false
	}
	if !models.PromotionInWindow(p, now) {
		return false
	}
	if len(p.OrderTypes) > 0 && orderType != "" {
		found := false
		for _, t := range p.OrderTypes {
			if t == orderType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// bestAuto picks the single automatic campaign worth most to this guest.
func (h *Handler) bestAuto(promos []models.Promotion, in priceInput, remaining int) *models.Promotion {
	var best *models.Promotion
	bestWorth := 0
	for i := range promos {
		p := &promos[i]
		if p.Trigger != models.TriggerAuto {
			continue
		}
		if in.Subtotal < p.MinOrder {
			continue
		}
		worth := h.worthOf(p, in, remaining)
		if worth > bestWorth {
			best, bestWorth = p, worth
		}
	}
	return best
}

// worthOf is what a campaign would take off, used only to compare candidates.
func (h *Handler) worthOf(p *models.Promotion, in priceInput, remaining int) int {
	switch p.Kind {
	case models.PromoFreeDelivery:
		if in.Type != "delivery" {
			return 0
		}
		return in.DeliveryFee
	case models.PromoPercent:
		base := h.discountBase(p, in.Items, remaining, in.ItemCategory)
		amount := base * p.Value / 100
		if p.MaxDiscount > 0 && amount > p.MaxDiscount {
			amount = p.MaxDiscount
		}
		return amount
	default:
		if p.Value > remaining {
			return remaining
		}
		return p.Value
	}
}

// findCode resolves a typed code to a usable promotion, or explains why not.
//
// The messages are deliberately specific. "Promokod ishlamadi" sends the guest
// to the phone; "50 000 so'mdan yuqori buyurtmalarga" lets them fix it
// themselves by adding one more dish.
func (h *Handler) findCode(
	ctx context.Context, live []models.Promotion, code string, in priceInput, remaining int,
) (*models.Promotion, string) {
	// Look it up regardless of the live filter, so an expired or wrong-branch
	// code gets a real reason instead of "not found".
	filter := bson.M{"code": code}
	if !in.BrandID.IsZero() {
		filter["brandId"] = in.BrandID
	}
	var p models.Promotion
	if err := h.Store.Promotions.FindOne(ctx, filter).Decode(&p); err != nil {
		return nil, "bunday promokod yo'q"
	}
	if !p.IsActive {
		return nil, "bu promokod endi ishlamaydi"
	}
	if p.Trigger != models.TriggerCode {
		return nil, "bunday promokod yo'q"
	}
	if p.EndsAt != nil && in.Now.After(*p.EndsAt) {
		return nil, "promokod muddati tugagan"
	}
	if p.StartsAt != nil && in.Now.Before(*p.StartsAt) {
		return nil, "promokod hali ishga tushmagan"
	}
	if in.Subtotal < p.MinOrder {
		return nil, fmt.Sprintf("promokod %d so'mdan yuqori buyurtmalar uchun", p.MinOrder)
	}
	if p.UsageLimit > 0 && p.UsedCount >= p.UsageLimit {
		return nil, "bu promokod ishlatib bo'lingan"
	}
	// The remaining live checks (day, time, branch, order type) are the ones the
	// filtered list already answered.
	found := false
	for i := range live {
		if live[i].ID == p.ID {
			found = true
			break
		}
	}
	if !found {
		return nil, "promokod hozir yoki bu turdagi buyurtmaga amal qilmaydi"
	}

	if !in.UserID.IsZero() {
		if p.FirstOrderOnly {
			n, err := h.Store.Orders.CountDocuments(ctx, bson.M{"userId": in.UserID})
			if err == nil && n > 0 {
				return nil, "bu promokod faqat birinchi buyurtma uchun"
			}
		}
		if p.PerUserLimit > 0 {
			n, err := h.Store.Orders.CountDocuments(ctx, bson.M{
				"userId":                in.UserID,
				"discounts.promotionId": p.ID,
				"status":                bson.M{"$ne": models.StatusCancelled},
			})
			if err == nil && int(n) >= p.PerUserLimit {
				return nil, "siz bu promokodni ishlatib bo'lgansiz"
			}
		}
	} else if p.FirstOrderOnly || p.PerUserLimit > 0 {
		// A limit nobody can be held to is not a limit.
		return nil, "bu promokod uchun tizimga kirish kerak"
	}
	return &p, ""
}

// redeem records that a code was used. Guarded by the usage limit in the filter
// so two guests typing the last redemption at once cannot both get it.
func (h *Handler) redeem(ctx context.Context, discounts []models.OrderDiscount) {
	for _, d := range discounts {
		if d.PromotionID.IsZero() {
			continue
		}
		_, _ = h.Store.Promotions.UpdateOne(ctx, bson.M{
			"_id": d.PromotionID,
			"$or": []bson.M{
				{"usageLimit": 0},
				{"$expr": bson.M{"$lt": []string{"$usedCount", "$usageLimit"}}},
			},
		}, bson.M{"$inc": bson.M{"usedCount": 1}})
	}
}
