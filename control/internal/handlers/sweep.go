package handlers

import (
	"context"
	"log"
	"time"

	"keel-control/internal/billing"
	"keel-control/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

// SweepTrials switches off demos whose deadline has passed.
//
// A demo is a deadline, not a promise. Left running past its end it quietly
// becomes a free product, and the customer who was going to decide never has to
// — which is worse for them too, because nobody ever calls.
//
// **Only trials are touched.** A paying customer who is late is never switched
// off by a timer: cutting a restaurant's ordering off in the middle of a lunch
// service over an unpaid invoice is the fastest way to lose it. That one is
// suspended by hand, after a phone call, and the console's "awaiting payment"
// filter is what puts the operator in front of it.
func (h *Handler) SweepTrials(ctx context.Context) {
	now := time.Now()

	// The very same clause the console's "trial over" button uses. Sharing it
	// is the point: if the sweep and the badge could disagree, the list would
	// show customers as expired that nothing acts on, or switch off customers
	// the list never warned about.
	clause, ok := attentionFilter(AttentionTrialExpired, now)
	if !ok {
		return
	}
	cur, err := h.Store.Tenants.Find(ctx, clause)
	if err != nil {
		log.Printf("sweep: %v", err)
		return
	}
	var tenants []models.Tenant
	if err := cur.All(ctx, &tenants); err != nil {
		log.Printf("sweep: %v", err)
		return
	}

	for _, t := range tenants {
		// A customer on free terms is never switched off by the clock.
		//
		// This is the case the flag exists for: an anchor customer brought in
		// on a promise of free service still has a trial end date sitting on
		// their row from the day the account was opened, and without this the
		// nightly sweep would take a twelve-restaurant chain offline over a
		// deadline nobody meant to apply to them. There is no worse first
		// impression available.
		if t.FreeAt(now) {
			continue
		}
		// The condition is carried in the filter, not checked beforehand.
		//
		// Between reading this list and writing the row, an operator may have
		// taken the customer's money — and a tick that switched off a paying
		// customer a second after they paid is the worst possible first day.
		// The guard also makes the sweep idempotent: a row already moved on is
		// simply not modified, so running every hour changes nothing twice.
		res, err := h.Store.Tenants.UpdateOne(ctx,
			bson.M{"_id": t.ID, "status": models.StatusTrial},
			bson.M{"$set": bson.M{
				"status":          models.StatusSuspended,
				"autoSuspendedAt": now,
				"updatedAt":       now,
			}})
		if err != nil {
			log.Printf("sweep %s: %v", t.Slug, err)
			continue
		}
		if res.ModifiedCount == 0 {
			continue
		}

		ends := ""
		if t.TrialEndsAt != nil {
			ends = billing.Day(local(*t.TrialEndsAt))
		}
		// Logged as well as stored: the stored date answers "why is this dark",
		// the log answers "what did the server do last night".
		log.Printf("sweep: %s — demo tugadi (%s), to'xtatildi", t.Slug, ends)

		t.Status = models.StatusSuspended
		// Stops the container and teaches the edge. Failure is recorded on the
		// tenant rather than retried here: the status has already changed, and
		// the retry button on the card is the same one an operator would reach
		// for anyway.
		h.apply(ctx, &t, false)
	}
}
