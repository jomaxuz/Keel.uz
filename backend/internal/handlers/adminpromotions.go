package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// errBadPromo is a plain rejection of a form the owner filled in; the panel
// shows the text as-is.
func errBadPromo(msg string) error { return errors.New(msg) }

// Managing campaigns and promo codes. One CRUD for both — they differ only in
// what sets them off (see models.Promotion).

func (h *Handler) AdminListPromotions(w http.ResponseWriter, r *http.Request) {
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter := scope.brandFilter(bson.M{})
	if t := strings.TrimSpace(r.URL.Query().Get("trigger")); t != "" {
		filter["trigger"] = t
	}
	opts := options.Find().SetSort(bson.D{
		{Key: "isActive", Value: -1}, {Key: "sortOrder", Value: 1}, {Key: "createdAt", Value: -1},
	})
	cur, err := h.Store.Promotions.Find(r.Context(), filter, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := []models.Promotion{}
	_ = cur.All(r.Context(), &out)
	// What each rule is actually doing right now, so an expired code does not
	// sit in the list wearing a green "active" badge.
	now := time.Now()
	for i := range out {
		out[i].Status = out[i].StatusAt(now)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// normalizePromotion applies the rules that must hold however the form was
// filled in, so a half-configured campaign cannot become a blank cheque.
func normalizePromotion(p *models.Promotion) error {
	p.Name = clampText(p.Name, 120)
	if p.Name == "" {
		return errBadPromo("nomini yozing")
	}
	if p.Trigger != models.TriggerCode && p.Trigger != models.TriggerAuto {
		p.Trigger = models.TriggerAuto
	}
	switch p.Kind {
	case models.PromoPercent, models.PromoFixed, models.PromoFreeDelivery:
	default:
		return errBadPromo("chegirma turini tanlang")
	}

	if p.Trigger == models.TriggerCode {
		// Codes are read aloud and retyped: letters and digits only, upper case.
		p.Code = strings.Map(func(r rune) rune {
			switch {
			case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
				return r
			case r >= 'a' && r <= 'z':
				return r - 32
			}
			return -1
		}, p.Code)
		if len(p.Code) < 3 {
			return errBadPromo("promokod kamida 3 belgi bo'lishi kerak")
		}
	} else {
		// An automatic campaign with a code would be a code nobody has to type.
		p.Code = ""
		p.UsageLimit, p.PerUserLimit, p.FirstOrderOnly = 0, 0, false
	}

	switch p.Kind {
	case models.PromoPercent:
		if p.Value < 1 || p.Value > 100 {
			return errBadPromo("foiz 1 dan 100 gacha bo'lishi kerak")
		}
	case models.PromoFixed:
		if p.Value < 1 {
			return errBadPromo("chegirma summasini yozing")
		}
	case models.PromoFreeDelivery:
		// The value carries no meaning here; the fee itself is what comes off.
		p.Value, p.MaxDiscount = 0, 0
	}

	switch p.Scope {
	case models.ScopeCategory, models.ScopeItems:
	default:
		p.Scope = models.ScopeOrder
		p.CategoryIDs, p.MenuItemIDs = nil, nil
	}
	if p.Scope == models.ScopeCategory && len(p.CategoryIDs) == 0 {
		return errBadPromo("kategoriyalarni tanlang")
	}
	if p.Scope == models.ScopeItems && len(p.MenuItemIDs) == 0 {
		return errBadPromo("taomlarni tanlang")
	}
	if p.StartsAt != nil && p.EndsAt != nil && p.EndsAt.Before(*p.StartsAt) {
		return errBadPromo("tugash sanasi boshlanishidan oldin bo'lishi mumkin emas")
	}
	if p.BranchIDs == nil {
		p.BranchIDs = []primitive.ObjectID{}
	}
	for _, v := range []*int{&p.Value, &p.MaxDiscount, &p.MinOrder, &p.UsageLimit, &p.PerUserLimit} {
		if *v < 0 {
			*v = 0
		}
	}
	return nil
}

func (h *Handler) AdminCreatePromotion(w http.ResponseWriter, r *http.Request) {
	var p models.Promotion
	if err := httpx.Decode(r, &p); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if p.BrandID.IsZero() {
		if scope, err := h.adminScope(r); err == nil {
			p.BrandID = h.scopeBrand(r, scope)
		}
	}
	if err := normalizePromotion(&p); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.codeFree(r, &p); err != nil {
		httpx.Error(w, http.StatusConflict, err.Error())
		return
	}
	now := time.Now()
	p.ID = primitiveNil
	p.UsedCount = 0
	p.CreatedAt, p.UpdatedAt = now, now
	res, err := h.Store.Promotions.InsertOne(r.Context(), p)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	p.ID = oidOf(res.InsertedID)
	h.logAction(r, ActPromotionCreate, "promotion", p.ID.Hex(), p.Name, string(p.Trigger))
	httpx.JSON(w, http.StatusCreated, p)
}

func (h *Handler) AdminUpdatePromotion(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var p models.Promotion
	if err := httpx.Decode(r, &p); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	p.ID = id
	if err := normalizePromotion(&p); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.codeFree(r, &p); err != nil {
		httpx.Error(w, http.StatusConflict, err.Error())
		return
	}

	// usedCount is the ledger of what has already been redeemed. The form never
	// carries it, and a whole-document write would reset it to zero — handing
	// out a spent code all over again.
	var existing models.Promotion
	if err := h.Store.Promotions.FindOne(r.Context(), bson.M{"_id": id}).Decode(&existing); err == nil {
		p.UsedCount = existing.UsedCount
		p.CreatedAt = existing.CreatedAt
		if p.BrandID.IsZero() {
			p.BrandID = existing.BrandID
		}
	}
	p.UpdatedAt = time.Now()

	if _, err := h.Store.Promotions.ReplaceOne(r.Context(), bson.M{"_id": id}, p); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActPromotionUpdate, "promotion", id.Hex(), p.Name, string(p.Trigger))
	httpx.JSON(w, http.StatusOK, p)
}

// AdminDeletePromotion removes a campaign. Orders keep their own copy of the
// discount (name and amount), so past receipts still explain themselves.
func (h *Handler) AdminDeletePromotion(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var removed models.Promotion
	_ = h.Store.Promotions.FindOne(r.Context(), bson.M{"_id": id}).Decode(&removed)
	if _, err := h.Store.Promotions.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActPromotionDelete, "promotion", id.Hex(), removed.Name, removed.Code)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// codeFree keeps one code meaning one thing inside a brand.
func (h *Handler) codeFree(r *http.Request, p *models.Promotion) error {
	if p.Code == "" {
		return nil
	}
	filter := bson.M{"code": p.Code, "brandId": p.BrandID}
	if !p.ID.IsZero() {
		filter["_id"] = bson.M{"$ne": p.ID}
	}
	n, err := h.Store.Promotions.CountDocuments(r.Context(), filter)
	if err != nil {
		return err
	}
	if n > 0 {
		return errBadPromo("bu promokod allaqachon ishlatilgan")
	}
	return nil
}

// ---- Who used a code, and how many of them ----

// AdminPromotionUsage answers the two questions an owner actually asks about a
// code: how many *people* used it (not just how many times), and who.
//
// Counted from the orders themselves rather than from usedCount: that counter
// is a fast guard for the redemption limit, while this is the record. Cancelled
// orders are left out — a code refunded with the order was not spent.
func (h *Handler) AdminPromotionUsage(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	filter := bson.M{
		"discounts.promotionId": id,
		"status":                bson.M{"$ne": string(models.StatusCancelled)},
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(200)
	cur, err := h.Store.Orders.Find(r.Context(), filter, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var orders []models.Order
	if err := cur.All(r.Context(), &orders); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	type usageRow struct {
		OrderID   string    `json:"orderId"`
		Number    string    `json:"number"`
		Customer  string    `json:"customer"`
		Phone     string    `json:"phone"`
		Amount    int       `json:"amount"`
		Total     int       `json:"total"`
		CreatedAt time.Time `json:"createdAt"`
	}
	rows := make([]usageRow, 0, len(orders))
	// "How many people" is distinct customers, not redemptions: one regular
	// using a code five times is one person, and the answer changes what the
	// owner concludes about the campaign.
	people := map[string]struct{}{}
	discounted := 0
	for _, o := range orders {
		amount := 0
		for _, d := range o.Discounts {
			if d.PromotionID == id {
				amount += d.Amount
			}
		}
		discounted += amount
		key := o.Customer.Phone
		if !o.UserID.IsZero() {
			key = o.UserID.Hex()
		}
		if key != "" {
			people[key] = struct{}{}
		}
		rows = append(rows, usageRow{
			OrderID:   o.ID.Hex(),
			Number:    o.Number,
			Customer:  o.Customer.Name,
			Phone:     o.Customer.Phone,
			Amount:    amount,
			Total:     o.Total,
			CreatedAt: o.CreatedAt,
		})
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"orders": rows,
		"stats": map[string]int{
			"redemptions": len(rows),
			"people":      len(people),
			// What the campaign has cost so far — the number that decides
			// whether it stays on.
			"discounted": discounted,
		},
	})
}
