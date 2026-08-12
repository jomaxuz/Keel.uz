package handlers

import (
	"context"
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

// Feedback on a delivered order.
//
// The rating is asked for on the tracking page the guest is already looking at,
// once the order is done — not by e-mail a day later, which is when people stop
// answering. The value is almost entirely in the low ones: a complaint nobody
// sees is a customer lost without a word, so an unhandled complaint stays on
// the list until someone writes down what was done about it.

// A rating at or below this is a complaint, not a score.
const lowRating = 3

// How long a complaint keeps its customer flagged as unhappy.
const unhappyDays = 60

type feedbackRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

// SubmitFeedback records the guest's rating of an order.
//
// Reachable by order number, like the tracking page it is shown on. When the
// order belongs to an account, the rating must come from that account: the
// number alone is enough to *look* at an order, and that is deliberately a
// lower bar than speaking for the person who placed it.
func (h *Handler) SubmitFeedback(w http.ResponseWriter, r *http.Request) {
	number := strings.TrimPrefix(strings.TrimSpace(chi.URLParam(r, "number")), "#")
	var req feedbackRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Rating < 1 || req.Rating > 5 {
		httpx.Error(w, http.StatusBadRequest, "bahoni tanlang")
		return
	}

	var order models.Order
	if err := h.Store.Orders.FindOne(r.Context(),
		bson.M{"number": number}).Decode(&order); err != nil {
		httpx.Error(w, http.StatusNotFound, "buyurtma topilmadi")
		return
	}
	// Rating an order that has not arrived yet rates nothing.
	if order.Status != models.StatusDelivered {
		httpx.Error(w, http.StatusBadRequest, "buyurtma hali yetkazilmagan")
		return
	}
	if !order.UserID.IsZero() {
		uid, ok := h.optionalUserID(r)
		if !ok || uid != order.UserID {
			httpx.Error(w, http.StatusForbidden,
				"baho qoldirish uchun o'z hisobingizga kiring")
			return
		}
	}
	// One rating per order. A second opinion is a phone call, not a second row.
	n, err := h.Store.Feedback.CountDocuments(r.Context(), bson.M{"orderId": order.ID})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n > 0 {
		httpx.Error(w, http.StatusConflict, "bu buyurtmaga baho allaqachon qoldirilgan")
		return
	}

	fb := models.Feedback{
		OrderID:     order.ID,
		OrderNumber: order.Number,
		UserID:      order.UserID,
		BranchID:    order.BranchID,
		Customer:    order.Customer,
		Rating:      req.Rating,
		Comment:     clampText(req.Comment, 1000),
		CreatedAt:   time.Now(),
	}
	res, err := h.Store.Feedback.InsertOne(r.Context(), fb)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	fb.ID = oidOf(res.InsertedID)
	httpx.JSON(w, http.StatusCreated, fb)
}

// OrderFeedback tells the tracking page whether this order has been rated, so
// it can stop asking.
func (h *Handler) OrderFeedback(w http.ResponseWriter, r *http.Request) {
	number := strings.TrimPrefix(strings.TrimSpace(chi.URLParam(r, "number")), "#")
	var fb models.Feedback
	if err := h.Store.Feedback.FindOne(r.Context(),
		bson.M{"orderNumber": number}).Decode(&fb); err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"rated": false})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"rated":   true,
		"rating":  fb.Rating,
		"comment": fb.Comment,
	})
}

// ---- Admin ----

// AdminListFeedback returns ratings, newest first.
// ?filter=low|unhandled|all, ?q= over the number, name, phone and comment.
func (h *Handler) AdminListFeedback(w http.ResponseWriter, r *http.Request) {
	filter, _, err := h.orderScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	switch r.URL.Query().Get("filter") {
	case "low":
		filter["rating"] = bson.M{"$lte": lowRating}
	case "unhandled":
		// The list that must reach zero: complaints nobody has answered.
		filter["rating"] = bson.M{"$lte": lowRating}
		filter["handled"] = false
	}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		rx := textSearch(strings.TrimPrefix(q, "#"))
		filter["$or"] = []bson.M{
			{"orderNumber": rx}, {"customer.name": rx},
			{"customer.phone": rx}, {"comment": rx},
		}
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(300)
	cur, err := h.Store.Feedback.Find(r.Context(), filter, opts)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.Feedback{}
	_ = cur.All(r.Context(), &rows)

	// The summary the owner actually looks at: the average, and how many
	// complaints are still open.
	var sum, count, open int
	all, err := h.Store.Feedback.Find(r.Context(), func() bson.M {
		base, _, _ := h.orderScope(r)
		return base
	}())
	if err == nil {
		var every []models.Feedback
		if err := all.All(r.Context(), &every); err == nil {
			for _, f := range every {
				sum += f.Rating
				count++
				if f.Rating <= lowRating && !f.Handled {
					open++
				}
			}
		}
	}
	avg := 0.0
	if count > 0 {
		avg = float64(sum) / float64(count)
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"feedback": rows,
		"stats": map[string]any{
			"count":   count,
			"average": avg,
			"open":    open,
		},
	})
}

type handleFeedbackRequest struct {
	Resolution string `json:"resolution"`
}

// AdminHandleFeedback closes a complaint with a note on what was done.
//
// The note is required. "Handled" with nothing written down is a checkbox that
// makes the list look better without making the restaurant any wiser.
func (h *Handler) AdminHandleFeedback(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req handleFeedbackRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resolution := clampText(req.Resolution, 500)
	if resolution == "" {
		httpx.Error(w, http.StatusBadRequest, "nima qilinganini yozing")
		return
	}
	admin, _ := h.adminUser(r)
	name := ""
	if admin != nil {
		name = admin.Username
	}
	now := time.Now()
	if _, err := h.Store.Feedback.UpdateByID(r.Context(), id, bson.M{"$set": bson.M{
		"handled":    true,
		"handledBy":  name,
		"handledAt":  now,
		"resolution": resolution,
	}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var fb models.Feedback
	_ = h.Store.Feedback.FindOne(r.Context(), bson.M{"_id": id}).Decode(&fb)
	h.logAction(r, ActFeedbackHandled, "feedback", id.Hex(), "#"+fb.OrderNumber, resolution)
	httpx.JSON(w, http.StatusOK, fb)
}

type publishFeedbackRequest struct {
	Public bool `json:"public"`
}

// AdminPublishFeedback puts one review on the public site, or takes it back off.
//
// ⚠️ **One at a time, and never in bulk.** The switch in settings opens the
// section; this is what fills it. Everything in this collection was written to
// the restaurant rather than to the internet — by a guest answering "how was
// your order?", with their name attached — so publishing has to be an act
// somebody performs while looking at the actual words, not a side effect of a
// checkbox somewhere else.
//
// Taking one down is the same call with `public: false`, and it is deliberately
// as easy as putting it up: a guest asking for their words to come off the site
// must not be waiting on us.
func (h *Handler) AdminPublishFeedback(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req publishFeedbackRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := h.Store.Feedback.UpdateByID(r.Context(), id,
		bson.M{"$set": bson.M{"isPublic": req.Public}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var fb models.Feedback
	_ = h.Store.Feedback.FindOne(r.Context(), bson.M{"_id": id}).Decode(&fb)
	action := ActFeedbackUnpublished
	if req.Public {
		action = ActFeedbackPublished
	}
	// Logged either way. "Whose words went onto our website, and who put them
	// there" is a question that gets asked exactly once, and only after it has
	// become a problem.
	h.logAction(r, action, "feedback", id.Hex(), "#"+fb.OrderNumber, fb.Comment)
	httpx.JSON(w, http.StatusOK, fb)
}

// unhappyUsers lists customers with a recent complaint nobody has answered.
// Used to flag them in the customer list — the one segment that is about the
// restaurant's own behaviour rather than the customer's.
func (h *Handler) unhappyUsers(r *http.Request) map[string]bool {
	return h.unhappyUserSet(r.Context())
}

// unhappyUserSet is the same question without a request, for the campaign
// audience — which is built in a background send where there is no request left.
func (h *Handler) unhappyUserSet(ctx context.Context) map[string]bool {
	out := map[string]bool{}
	since := time.Now().AddDate(0, 0, -unhappyDays)
	cur, err := h.Store.Feedback.Find(ctx, bson.M{
		"rating":    bson.M{"$lte": lowRating},
		"handled":   false,
		"createdAt": bson.M{"$gte": since},
	})
	if err != nil {
		return out
	}
	var rows []models.Feedback
	if err := cur.All(ctx, &rows); err != nil {
		return out
	}
	for _, f := range rows {
		if f.UserID != primitive.NilObjectID {
			out[f.UserID.Hex()] = true
		}
	}
	return out
}
