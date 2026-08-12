package handlers

import (
	"context"
	"strings"
	"time"

	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Guests' ratings and words, on the restaurant's own site.
//
// ⚠️ **Nothing here is published because a switch was flipped.** The rows this
// reads were written by guests answering "how was your order?" on their own
// tracking page — a private message to the restaurant, signed with their name.
// The setting opens the section; `feedback.isPublic` decides what goes in it,
// one review at a time, chosen by somebody looking at the actual words.
//
// The shape returned is deliberately narrow. A `models.Feedback` carries the
// phone number, the order number, the complaint handling and who dealt with it,
// and none of that belongs on a public page — so this is a separate struct
// rather than a filtered list of the real one. A field added to the model later
// cannot leak through a struct that has to be extended by hand.

// publicReview is one guest's words as a visitor sees them.
type publicReview struct {
	// First name only. What the guest typed is often their full name, and a
	// surname on a public page is more than anybody agreed to — the Uzbek
	// convention for this is a first name anyway.
	Name    string    `json:"name"`
	Rating  int       `json:"rating"`
	Comment string    `json:"comment"`
	At      time.Time `json:"at"`
}

type publicReviews struct {
	Items []publicReview `json:"items"`
	// The average and how many ratings it is from. Computed over **every**
	// rating, not only the published ones: an average of the reviews an owner
	// chose to show is not an average of anything, and printing it next to a
	// star row is a claim about the restaurant that is not true.
	Average float64 `json:"average"`
	Count   int     `json:"count"`
	// Whether the site should draw that average at all.
	ShowAverage bool `json:"showAverage"`
}

// firstName trims what the guest typed down to the part that is safe to print.
func firstName(full string) string {
	fields := strings.Fields(strings.TrimSpace(full))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// publicReviewsFor gathers what the site may show for one branch, or for the
// whole company when no branch is resolved.
//
// Returns nil when the feature is off, which is how the page knows to draw
// nothing at all rather than an empty section with a heading over it.
func (h *Handler) publicReviewsFor(
	ctx context.Context, rest *models.Restaurant, branchID any,
) *publicReviews {
	if !rest.Reviews.Enabled {
		return nil
	}
	filter := bson.M{}
	if branchID != nil {
		// ⚠️ **Rows with no branch count too.** Feedback written before
		// branches existed carries none, and a strict `branchId` match would
		// drop it — leaving an owner who has switched the section on, and
		// ticked reviews to publish, looking at a site that shows nothing and
		// a panel that says it should. Unassigned feedback belongs to the
		// company, so it belongs everywhere.
		filter["$or"] = []bson.M{{"branchId": branchID}, {"branchId": nil}}
	}

	out := &publicReviews{
		// Never nil: Go marshals a nil slice as `null`, and the section maps
		// over this. The same shape of bug that took out the console's customer
		// card, arriving on a public page this time.
		Items:       []publicReview{},
		ShowAverage: rest.Reviews.ShowAverage,
	}

	// The average, over every rating this branch has ever had.
	if rest.Reviews.ShowAverage {
		pipeline := []bson.M{
			{"$match": filter},
			{"$group": bson.M{
				"_id": nil,
				"sum": bson.M{"$sum": "$rating"},
				"n":   bson.M{"$sum": 1},
			}},
		}
		if cur, err := h.Store.Feedback.Aggregate(ctx, pipeline); err == nil {
			var rows []struct {
				Sum int `bson:"sum"`
				N   int `bson:"n"`
			}
			if err := cur.All(ctx, &rows); err == nil && len(rows) > 0 && rows[0].N > 0 {
				out.Count = rows[0].N
				// One decimal place, rounded here rather than in the browser:
				// a rating printed as "4.333333333" is a rating nobody wrote.
				out.Average = float64(int(float64(rows[0].Sum)/float64(rows[0].N)*10+0.5)) / 10
			}
		}
	}

	// The published ones, newest first. Capped because this rides along with
	// the profile on every page load — the same trade the banners make, and
	// the render tier is the platform's bottleneck.
	published := bson.M{}
	for k, v := range filter {
		published[k] = v
	}
	published["isPublic"] = true
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(12)
	cur, err := h.Store.Feedback.Find(ctx, published, opts)
	if err != nil {
		return out
	}
	var rows []models.Feedback
	if err := cur.All(ctx, &rows); err != nil {
		return out
	}
	for _, f := range rows {
		out.Items = append(out.Items, publicReview{
			Name:    firstName(f.Customer.Name),
			Rating:  f.Rating,
			Comment: f.Comment,
			At:      f.CreatedAt.In(time.Local),
		})
	}
	return out
}
