package handlers

// ---- Telling Meta what actually got ordered ----
//
// ⚠️ **This is the only part of the section a targetolog cannot copy.** Anybody
// with access to an ads account can read clicks. What this server has is the
// order: it was cooked, it was worth this much, and it was not cancelled
// afterwards. Sending that back is what turns "three hundred clicks" into
// "forty-seven orders", and it is the sentence the whole add-on is sold on.
//
// ⚠️ **Reported when the order is finished, not when it is placed.** A pending
// order is a hope; one that was cancelled at the door and already reported as a
// purchase teaches Meta to find more people who cancel at the door — and it
// spends the restaurant's budget doing it.
//
// ⚠️ **A sweeper rather than a call inside the status handler.** The status
// change is a request an owner is waiting on, and a Meta round trip inside it
// would make a slow evening at Meta look like a slow panel. It also gives the
// one thing an inline call cannot: a retry after an outage, without anybody
// noticing there was one.

import (
	"context"
	"log"
	"strings"
	"time"

	"restaurant-backend/internal/meta"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	adsEventEvery = 5 * time.Minute
	// How far back the sweeper will look for something it never sent.
	//
	// ⚠️ **Two days, not forever.** Meta refuses an event older than seven days
	// outright, and a restaurant that connects its pixel today has no business
	// flooding it with a month of history the adverts could not have caused.
	adsEventBack = 48 * time.Hour
	// How many go in one request.
	adsEventBatch = 50
)

// StartAdsEvents reports finished orders to Meta.
func (h *Handler) StartAdsEvents(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(adsEventEvery)
		defer ticker.Stop()
		for {
			h.sendAdsEventsOnce(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (h *Handler) sendAdsEventsOnce(ctx context.Context) {
	client, s, err := h.adsClient(ctx)
	if err != nil || strings.TrimSpace(s.PixelID) == "" {
		// No pixel means no place to report to, which is a setting rather than
		// a fault — half the point of the connect checklist saying so.
		return
	}
	cur, err := h.Store.Orders.Find(ctx, bson.M{
		"status":     models.StatusDelivered,
		"adsEventAt": nil,
		"createdAt":  bson.M{"$gte": time.Now().Add(-adsEventBack)},
	}, options.Find().SetLimit(adsEventBatch).
		SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		log.Printf("ads events: %v", err)
		return
	}
	var orders []models.Order
	if err := cur.All(ctx, &orders); err != nil || len(orders) == 0 {
		return
	}

	events := make([]meta.Event, 0, len(orders))
	ids := make([]any, 0, len(orders))
	for _, o := range orders {
		events = append(events, h.adsEventOf(o))
		ids = append(ids, o.ID)
	}
	if _, err := client.SendEvents(ctx, s.PixelID, events, ""); err != nil {
		// ⚠️ **Nothing is stamped when the batch failed**, so the next tick
		// sends it again. The opposite mistake — stamping first — loses the
		// orders the report exists to count, and loses them silently.
		log.Printf("ads events: %v", err)
		h.noteMetaError(ctx, err)
		return
	}
	now := time.Now()
	_, _ = h.Store.Orders.UpdateMany(ctx, bson.M{"_id": bson.M{"$in": ids}},
		bson.M{"$set": bson.M{"adsEventAt": now}})
}

// adsEventOf turns one finished order into one event.
func (h *Handler) adsEventOf(o models.Order) meta.Event {
	when := o.CreatedAt
	if o.PaidAt != nil {
		when = *o.PaidAt
	}
	ids := make([]string, 0, len(o.Items))
	for _, it := range o.Items {
		if !it.MenuItemID.IsZero() {
			ids = append(ids, it.MenuItemID.Hex())
		}
	}
	return meta.Event{
		EventName: "Purchase",
		EventTime: when.Unix(),
		// ⚠️ **The order number, which is also what the browser pixel sends.**
		// Meta collapses the pair only when both carry the same id; without it
		// every online order is counted twice, and a doubled number is worse
		// than a missing one because an owner will believe it.
		EventID: o.Number,
		// The order was completed by the restaurant, not by somebody's browser.
		Source: "website",
		// ⚠️ Hashed inside `Person` — nothing identifying a guest leaves this
		// server in the clear, and the normalisation is Meta's own so a match
		// that should happen does.
		UserData: meta.Person(o.Customer.Name, o.Customer.Phone, ""),
		Custom: meta.CustomData{
			Value: float64(o.Total),
			// ⚠️ **So'm, because that is what was actually charged.** Meta
			// converts for its own return-on-spend figure; inventing a rate here
			// would put a made-up number into the one report about money.
			Currency:    "UZS",
			OrderID:     o.Number,
			ContentIDs:  ids,
			ContentType: "product",
		},
	}
}
