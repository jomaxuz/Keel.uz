package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// "Write to us" from the contact page: a rating, and words if they have any.
//
// Separate from the per-order feedback (`SubmitFeedback`) because it answers a different
// question. That one is "how was order 1745", asked on the tracking page of an order we
// can name; this one is "how are we", asked by somebody who may not have ordered today.
// Both land in the same collection, so the panel has one list to read — a second inbox
// is a second inbox somebody stops opening.
//
// ⚠️ **Signed in only.** Not to protect the table but to protect the restaurant: an open
// form on a public page is a form that fills with rubbish within a week, and the owner
// then stops reading the one place complaints arrive. A phone number that was proved by
// SMS is also what makes a complaint answerable — a review nobody can reply to is a
// review that cannot be fixed.

type siteFeedbackRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

// SubmitSiteFeedback stores a rating left from the contact page.
func (h *Handler) SubmitSiteFeedback(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	id, err := objectID(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "kirish talab qilinadi")
		return
	}
	var req siteFeedbackRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Rating < 1 || req.Rating > 5 {
		httpx.Error(w, http.StatusBadRequest, "bahoni tanlang")
		return
	}
	comment := strings.TrimSpace(req.Comment)
	if len([]rune(comment)) > 1000 {
		comment = string([]rune(comment)[:1000])
	}

	var user models.User
	if err := h.Store.Users.FindOne(r.Context(), bson.M{"_id": id}).Decode(&user); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "kirish talab qilinadi")
		return
	}

	// ⚠️ One per five minutes. Not a rate limit against an attacker — they are signed
	// in — but against the double tap: a slow connection, a second press, and the panel
	// shows one guest complaining twice, which is how a list of complaints stops being
	// countable.
	recent := bson.M{
		"userId": user.ID,
		// Only the page's own feedback: an order rated four minutes ago is a
		// different thing and must not block this one.
		"orderId":   bson.M{"$in": []any{nil, primitive.NilObjectID}},
		"createdAt": bson.M{"$gte": time.Now().Add(-5 * time.Minute)},
	}
	if n, err := h.Store.Feedback.CountDocuments(r.Context(), recent,
		options.Count().SetLimit(1)); err == nil && n > 0 {
		httpx.Error(w, http.StatusTooManyRequests,
			"fikringiz allaqachon yuborilgan — rahmat")
		return
	}

	fb := models.Feedback{
		UserID: user.ID,
		// ⚠️ Without a branch the row is filtered out of every panel view — see
		// backfillFeedbackBranch for the migration that had to repair exactly this.
		BranchID: h.feedbackBranch(r.Context(), &user),
		Customer: models.OrderCustomer{
			Name:  strings.TrimSpace(user.FirstName + " " + user.LastName),
			Phone: user.Phone,
		},
		Rating:    req.Rating,
		Comment:   comment,
		CreatedAt: time.Now(),
	}
	if _, err := h.Store.Feedback.InsertOne(r.Context(), fb); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ratedRecentlyWindow is how long one rating stands for.
//
// An hour, and the number is a judgement rather than a limit: a guest who ate once has
// one opinion of that visit, and somebody who genuinely wants to say more writes it in
// the comment. Long enough that a spammer gets nowhere, short enough that a guest who
// comes back for dinner can rate dinner.
const ratedRecentlyWindow = time.Hour

// ratedRecently reports whether this guest has already rated us without an order.
//
// ⚠️ **One rule, one function, both doors.** The website form and the bot's star buttons
// are the same act, and the two had different windows for a while — which means the same
// guest could rate twice by switching surface. A rule written in two places is a rule
// that will disagree with itself.
//
// Order feedback is deliberately excluded: rating order 1745 an hour ago says nothing
// about how they feel today, and blocking it would silence the more useful of the two.
func (h *Handler) ratedRecently(ctx context.Context, userID primitive.ObjectID) bool {
	if userID.IsZero() {
		return false
	}
	n, err := h.Store.Feedback.CountDocuments(ctx, bson.M{
		"userId":    userID,
		"orderId":   bson.M{"$in": []any{nil, primitive.NilObjectID}},
		"createdAt": bson.M{"$gte": time.Now().Add(-ratedRecentlyWindow)},
	}, options.Count().SetLimit(1))
	return err == nil && n > 0
}
