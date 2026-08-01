package handlers

import (
	"context"
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Cashback points.
//
// One point is one so'm, so a balance needs no explaining. Two rules shape
// everything below:
//
//   - Points are **spent when the order is placed** but **earned when it is
//     delivered**. Spending has to be immediate or the same balance could be
//     promised to two orders; earning has to wait, or a cancelled order would
//     mint points out of nothing.
//   - Paying with points never earns points. Otherwise a balance feeds itself
//     and slowly turns into free food.
//
// Every movement is written to loyalty_txn. The balance on the user is a cache
// for speed; the ledger is the record, because "where did my 6 000 go?" must be
// answerable long after the order scrolled off the screen.

// loyaltySettings reads the company's rules, with the defaults that make an
// unconfigured install behave as if the feature were off.
func (h *Handler) loyaltySettings(ctx context.Context) models.LoyaltySettings {
	var rest models.Restaurant
	if err := h.Store.Restaurant.FindOne(ctx, bson.M{}).Decode(&rest); err != nil {
		return models.LoyaltySettings{}
	}
	l := rest.Loyalty
	if l.MaxRedeemPercent <= 0 || l.MaxRedeemPercent > 100 {
		// A missing ceiling reads as "half the order", never as "all of it".
		l.MaxRedeemPercent = 50
	}
	return l
}

// balanceOf returns a customer's current points.
func (h *Handler) balanceOf(ctx context.Context, userID primitive.ObjectID) int {
	if userID.IsZero() {
		return 0
	}
	var u models.User
	if err := h.Store.Users.FindOne(ctx, bson.M{"_id": userID}).Decode(&u); err != nil {
		return 0
	}
	return u.Points
}

// maxRedeemable is the most a customer may put towards this order: what they
// have, capped by the share of the bill points are allowed to cover.
func maxRedeemable(l models.LoyaltySettings, balance, payable int) int {
	if !l.Enabled || balance <= 0 || payable <= 0 {
		return 0
	}
	cap := payable * l.MaxRedeemPercent / 100
	if balance < cap {
		cap = balance
	}
	if cap < 0 {
		cap = 0
	}
	return cap
}

// moveBalance applies a change and writes the ledger entry that explains it.
//
// The balance is moved with $inc rather than by writing a computed total: two
// orders settling at the same moment would otherwise each write the total they
// worked out before the other one landed, and one of the movements would
// silently vanish.
func (h *Handler) moveBalance(
	ctx context.Context, userID primitive.ObjectID, points int,
	kind models.LoyaltyKind, order *models.Order, note string,
) {
	if userID.IsZero() || points == 0 {
		return
	}
	res := h.Store.Users.FindOneAndUpdate(ctx,
		bson.M{"_id": userID},
		bson.M{"$inc": bson.M{"points": points}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	var u models.User
	if err := res.Decode(&u); err != nil {
		return
	}
	// A balance can never be owed. If rounding or a stale read ever pushed it
	// under zero, straighten it here rather than letting it haunt the customer.
	if u.Points < 0 {
		_, _ = h.Store.Users.UpdateByID(ctx, userID, bson.M{"$set": bson.M{"points": 0}})
		u.Points = 0
	}

	txn := models.LoyaltyTxn{
		UserID:       userID,
		Kind:         kind,
		Points:       points,
		BalanceAfter: u.Points,
		Note:         note,
		At:           time.Now(),
	}
	if order != nil {
		txn.OrderID, txn.OrderNumber = order.ID, order.Number
	}
	_, _ = h.Store.LoyaltyTxns.InsertOne(ctx, txn)
}

// awardPoints gives the cashback for a completed order.
//
// Idempotent: a status flipped to delivered twice — or back and forth — must
// not pay twice. The ledger is the guard, not a flag on the order.
func (h *Handler) awardPoints(ctx context.Context, order *models.Order) {
	if order.UserID.IsZero() {
		return
	}
	l := h.loyaltySettings(ctx)
	if !l.Enabled || l.EarnPercent <= 0 {
		return
	}
	n, err := h.Store.LoyaltyTxns.CountDocuments(ctx, bson.M{
		"orderId": order.ID,
		"kind":    string(models.LoyaltyEarn),
	})
	if err != nil || n > 0 {
		return
	}

	// Cashback is on money, not on points: what the guest actually paid for the
	// food, with the delivery fee and anything settled from the balance left
	// out. Otherwise the balance would feed itself.
	base := order.Subtotal - order.DiscountTotal - order.PointsSpent
	if base < l.MinOrderToEarn || base <= 0 {
		return
	}
	points := base * l.EarnPercent / 100
	if points <= 0 {
		return
	}
	h.moveBalance(ctx, order.UserID, points, models.LoyaltyEarn, order, "")
	_, _ = h.Store.Orders.UpdateByID(ctx, order.ID,
		bson.M{"$set": bson.M{"pointsEarned": points}})
}

// revokePoints undoes an order's loyalty when it is cancelled: what was spent
// comes back, what was earned goes away. Also idempotent — cancelling an
// already-cancelled order must not hand out the refund twice.
func (h *Handler) revokePoints(ctx context.Context, order *models.Order) {
	if order.UserID.IsZero() {
		return
	}
	n, err := h.Store.LoyaltyTxns.CountDocuments(ctx, bson.M{
		"orderId": order.ID,
		"kind":    string(models.LoyaltyRevoke),
	})
	if err != nil || n > 0 {
		return
	}
	delta := 0
	if order.PointsSpent > 0 {
		delta += order.PointsSpent // the guest gets their points back
	}
	if order.PointsEarned > 0 {
		delta -= order.PointsEarned // cashback for an order that never happened
	}
	if delta == 0 {
		return
	}
	h.moveBalance(ctx, order.UserID, delta, models.LoyaltyRevoke, order,
		"buyurtma bekor qilindi")
	if order.PointsEarned > 0 {
		_, _ = h.Store.Orders.UpdateByID(ctx, order.ID,
			bson.M{"$set": bson.M{"pointsEarned": 0}})
	}
}

// ---- Customer-facing ----

// UserLoyalty is the customer's balance and its statement.
func (h *Handler) UserLoyalty(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	id, err := objectID(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	l := h.loyaltySettings(r.Context())
	opts := options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(50)
	cur, err := h.Store.LoyaltyTxns.Find(r.Context(), bson.M{"userId": id}, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	txns := []models.LoyaltyTxn{}
	_ = cur.All(r.Context(), &txns)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"enabled":          l.Enabled,
		"balance":          h.balanceOf(r.Context(), id),
		"earnPercent":      l.EarnPercent,
		"maxRedeemPercent": l.MaxRedeemPercent,
		"minOrderToEarn":   l.MinOrderToEarn,
		"transactions":     txns,
	})
}

// grantWelcomePoints hands a new customer their sign-up bonus, once.
func (h *Handler) grantWelcomePoints(ctx context.Context, user *models.User) {
	l := h.loyaltySettings(ctx)
	if !l.Enabled || l.WelcomePoints <= 0 {
		return
	}
	n, err := h.Store.LoyaltyTxns.CountDocuments(ctx, bson.M{
		"userId": user.ID,
		"kind":   string(models.LoyaltyAdjust),
		"note":   "welcome",
	})
	if err != nil || n > 0 {
		return
	}
	h.moveBalance(ctx, user.ID, l.WelcomePoints, models.LoyaltyAdjust, nil, "welcome")
}
