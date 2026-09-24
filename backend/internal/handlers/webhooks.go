package handlers

// ---- Outgoing webhooks: we call them when an order moves ----
//
// See docs/DECISIONS.md → "Ochiq API: kalitlar va webhook'lar".
//
//	status change ──▶ h.orderEvent ──▶ webhook_delivery (queue, unique per event)
//	                                          │
//	                     StartWebhookSender ◀─┘ signs, POSTs, retries ~2 days
//
// ⚠️ **Raised at every place an order's statusHistory grows, and a test holds
// the list** (TestEveryStatusChangeRaisesAWebhook). There are a dozen such
// places — site, till, kitchen, courier, Uzum, offline sync, merge — and one
// that forgot would be a receiver that silently never hears about dine-in
// checks. A change stream would catch them all by itself, but it needs a
// replica set and every install is a single mongod.
//
// ⚠️ **Never on the request's time.** Raising an event is one read of the
// endpoints (none registered → nothing else happens) and an insert per
// endpoint; the HTTP call is the sender's, later. A receiver that takes ten
// seconds to answer must not be a till that takes ten seconds to close a check.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/webhook"
)

const (
	webhookSecretPrefix = "whsec_"
	// How long a claimed delivery is hidden from the next poll. Longer than
	// one attempt can take, so two pollers never send the same one at once.
	webhookLease = 2 * time.Minute
	// Endpoints and keys a restaurant may hold. Nobody integrates twenty
	// programs; a list that long is a script that ran in a loop.
	maxWebhookEndpoints = 10
	maxAPIKeys          = 20
)

// webhookKick wakes the sender as soon as something is queued, so a receiver
// hears about an order in about a second rather than at the next tick.
var webhookKick = make(chan struct{}, 1)

func kickWebhooks() {
	select {
	case webhookKick <- struct{}{}:
	default:
	}
}

// webhookEnvelope is the body of every delivery. ⚠️ Published.
type webhookEnvelope struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	CreatedAt time.Time      `json:"createdAt"`
	Data      webhookEnvData `json:"data"`
}

type webhookEnvData struct {
	// The status this event is about, and the one before it ("" for a new
	// order). ⚠️ Not the same as `order.status` when events are read late: the
	// order is a snapshot taken when the event was queued, and a quick
	// confirm-then-cook can queue both events after the second change.
	Status         string    `json:"status"`
	PreviousStatus string    `json:"previousStatus"`
	Order          OpenOrder `json:"order"`
}

// orderEventID names the event for statusHistory entry i of an order. ⚠️ Its
// stability is the whole of deduplication — see models.WebhookDelivery.
func orderEventID(orderID primitive.ObjectID, i int) string {
	return fmt.Sprintf("evt_%s_%d", orderID.Hex(), i)
}

// orderEventsFor lists what an order's history says happened since `since`,
// as (event type, entry index) pairs. Entry 0 is the order being placed.
//
// ⚠️ **Every entry since the endpoint existed, not only the last.** Two changes
// a moment apart are read back by two callers who can both see the second one;
// taking "the last" from each would announce the second twice (deduplicated)
// and the first never.
func orderEventsFor(o *models.Order, since time.Time) []int {
	var out []int
	for i, ev := range o.StatusHistory {
		if ev.At.Before(since) {
			continue
		}
		// The panel lets an operator set the status an order already has, and
		// the history records the press. A receiver told "confirmed →
		// confirmed" has been told nothing, and has to be written to ignore it.
		if i > 0 && o.StatusHistory[i-1].Status == ev.Status {
			continue
		}
		out = append(out, i)
	}
	return out
}

func eventTypeOf(i int) string {
	if i == 0 {
		return models.EventOrderCreated
	}
	return models.EventOrderStatusChanged
}

// orderEvent queues webhooks for what just happened to an order. Called after
// the write, with the order's id; it reads the order back itself so no caller
// can hand it a stale copy.
//
// Never fails the caller: the order moved whether or not anybody is listening.
func (h *Handler) orderEvent(ctx context.Context, orderID primitive.ObjectID) {
	if orderID.IsZero() {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	endpoints, err := h.liveWebhookEndpoints(ctx)
	if err != nil || len(endpoints) == 0 {
		return
	}
	var o models.Order
	if err := h.Store.Orders.FindOne(ctx, bson.M{"_id": orderID}).Decode(&o); err != nil {
		return
	}
	snapshot := openOrderOf(&o)
	now := time.Now()
	queued := false
	for _, ep := range endpoints {
		// ⚠️ **Only history written after the endpoint was added.** A receiver
		// registered at noon must not be told about every step of an order
		// placed at eleven the first time that order moves again.
		for _, i := range orderEventsFor(&o, ep.CreatedAt) {
			kind := eventTypeOf(i)
			if !ep.Subscribes(kind) {
				continue
			}
			prev := ""
			if i > 0 {
				prev = string(o.StatusHistory[i-1].Status)
			}
			env := webhookEnvelope{
				ID: orderEventID(o.ID, i), Type: kind, CreatedAt: o.StatusHistory[i].At,
				Data: webhookEnvData{
					Status: string(o.StatusHistory[i].Status), PreviousStatus: prev, Order: snapshot,
				},
			}
			body, err := json.Marshal(env)
			if err != nil {
				continue
			}
			_, err = h.Store.WebhookDeliveries.InsertOne(ctx, models.WebhookDelivery{
				EndpointID: ep.ID, EventID: env.ID, Event: kind, Payload: string(body),
				OrderNumber: o.Number, Status: models.DeliveryPending,
				NextAttemptAt: now, CreatedAt: now,
			})
			switch {
			case err == nil:
				queued = true
			case mongo.IsDuplicateKeyError(err):
				// Already queued by an earlier call — the index doing its job.
			default:
				log.Printf("webhook queue %s → %s: %v", env.ID, ep.URL, err)
			}
		}
	}
	if queued {
		kickWebhooks()
	}
}

func (h *Handler) liveWebhookEndpoints(ctx context.Context) ([]models.WebhookEndpoint, error) {
	cur, err := h.Store.WebhookEndpoints.Find(ctx, bson.M{"enabled": true})
	if err != nil {
		return nil, err
	}
	var out []models.WebhookEndpoint
	err = cur.All(ctx, &out)
	return out, err
}

// ---- The sender ----

// StartWebhookSender delivers the queue until ctx ends.
func (h *Handler) StartWebhookSender(ctx context.Context) {
	go func() {
		tick := time.NewTicker(15 * time.Second)
		defer tick.Stop()
		for {
			h.sendDueWebhooks(ctx)
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			case <-webhookKick:
			}
		}
	}()
}

// sendDueWebhooks sends what is due, oldest first, one at a time.
//
// ⚠️ **One at a time, and so in order per restaurant** as far as a first
// attempt goes. A retry can overtake nothing it was queued before, but it can
// arrive after a newer event did — receivers are told to order by the event's
// `createdAt` (the docs say so), not by arrival.
func (h *Handler) sendDueWebhooks(ctx context.Context) {
	for i := 0; i < 200 && ctx.Err() == nil; i++ {
		d, ok := h.claimWebhook(ctx)
		if !ok {
			return
		}
		h.deliverWebhook(ctx, &d)
	}
}

// claimWebhook takes the next due delivery and hides it for webhookLease.
func (h *Handler) claimWebhook(ctx context.Context) (models.WebhookDelivery, bool) {
	now := time.Now()
	var d models.WebhookDelivery
	err := h.Store.WebhookDeliveries.FindOneAndUpdate(ctx,
		bson.M{"status": models.DeliveryPending, "nextAttemptAt": bson.M{"$lte": now}},
		bson.M{"$set": bson.M{"nextAttemptAt": now.Add(webhookLease)}, "$inc": bson.M{"attempts": 1}},
		options.FindOneAndUpdate().
			SetSort(bson.D{{Key: "nextAttemptAt", Value: 1}}).
			SetReturnDocument(options.After),
	).Decode(&d)
	return d, err == nil
}

func (h *Handler) deliverWebhook(ctx context.Context, d *models.WebhookDelivery) {
	var ep models.WebhookEndpoint
	err := h.Store.WebhookEndpoints.FindOne(ctx, bson.M{"_id": d.EndpointID}).Decode(&ep)
	if err != nil {
		// Deleted while this waited. The delete removes its queue too; this is
		// the narrow race between the two.
		_, _ = h.Store.WebhookDeliveries.DeleteOne(ctx, bson.M{"_id": d.ID})
		return
	}
	if !ep.Enabled {
		_, _ = h.Store.WebhookDeliveries.UpdateByID(ctx, d.ID, bson.M{"$set": bson.M{
			"status": models.DeliverySkipped,
		}})
		return
	}
	res := webhook.Send(ctx, webhook.Client, webhook.Request{
		URL: ep.URL, Secret: ep.Secret, Event: d.Event, EventID: d.EventID,
		DeliveryID: d.ID.Hex(), Body: []byte(d.Payload),
	}, time.Now())
	h.recordWebhookResult(ctx, d, res)
}

// recordWebhookResult writes the outcome to the delivery and the endpoint.
func (h *Handler) recordWebhookResult(ctx context.Context, d *models.WebhookDelivery, res webhook.Result) {
	now := time.Now()
	if res.OK() {
		_, _ = h.Store.WebhookDeliveries.UpdateByID(ctx, d.ID, bson.M{
			"$set":   bson.M{"status": models.DeliveryDelivered, "deliveredAt": now, "lastStatus": res.StatusCode},
			"$unset": bson.M{"lastError": ""},
		})
		_, _ = h.Store.WebhookEndpoints.UpdateByID(ctx, d.EndpointID, bson.M{"$set": bson.M{
			"lastSuccessAt": now, "consecutiveFailures": 0,
		}})
		return
	}
	set := bson.M{"lastError": res.Err, "lastStatus": res.StatusCode}
	if wait, more := webhook.Backoff(d.Attempts); more {
		set["nextAttemptAt"] = now.Add(wait)
	} else {
		set["status"] = models.DeliveryFailed
	}
	_, _ = h.Store.WebhookDeliveries.UpdateByID(ctx, d.ID, bson.M{"$set": set})
	_, _ = h.Store.WebhookEndpoints.UpdateByID(ctx, d.EndpointID, bson.M{
		"$set": bson.M{"lastFailureAt": now, "lastError": res.Err},
		"$inc": bson.M{"consecutiveFailures": 1},
	})
}

// ---- The panel: API keys ----

// AdminAPIKeys lists the keys, newest first, revoked ones included.
func (h *Handler) AdminAPIKeys(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	cur, err := h.Store.APIKeys.Find(r.Context(), bson.M{},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	keys := []models.APIKey{}
	if err := cur.All(r.Context(), &keys); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	for i := range keys {
		if keys[i].Scopes == nil {
			keys[i].Scopes = []string{}
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"keys": keys, "scopes": models.APIScopes, "basePath": OpenAPIBasePath,
	})
}

// cleanChoices keeps the members of allowed that were asked for, in allowed's
// order, once each.
func cleanChoices(asked, allowed []string) []string {
	want := map[string]bool{}
	for _, s := range asked {
		want[strings.TrimSpace(s)] = true
	}
	out := []string{}
	for _, s := range allowed {
		if want[s] {
			out = append(out, s)
		}
	}
	return out
}

// AdminCreateAPIKey mints a key.
//
// ⚠️ **The key is in this answer and nowhere else, ever** — see models.APIKey.
func (h *Handler) AdminCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	name := clampText(strings.TrimSpace(req.Name), 80)
	if name == "" {
		httpx.Error(w, http.StatusBadRequest, "kalitga nom bering — qaysi dastur uchun ekanini keyin shu aytadi")
		return
	}
	scopes := cleanChoices(req.Scopes, models.APIScopes)
	if len(scopes) == 0 {
		httpx.Error(w, http.StatusBadRequest, "kamida bitta ruxsatni tanlang")
		return
	}
	if n, _ := h.Store.APIKeys.CountDocuments(r.Context(), bson.M{"revokedAt": nil}); n >= maxAPIKeys {
		httpx.Error(w, http.StatusBadRequest, "faol kalitlar juda ko'p — keraksizlarini bekor qiling")
		return
	}
	secret, err := newAPIKey()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	key := models.APIKey{
		Name: name, Prefix: secret[:apiKeyShown], Hash: apiKeyHash(secret),
		Scopes: scopes, CreatedAt: time.Now(),
	}
	// Who let this program in, on the row itself: the journal says it too, but
	// the key list is where the question is asked.
	if c := middleware.ClaimsFrom(r.Context()); c != nil {
		if id, err := objectID(c.UserID); err == nil {
			key.CreatedByID = id
			var admin models.AdminUser
			if h.Store.Admins.FindOne(r.Context(), bson.M{"_id": id}).Decode(&admin) == nil {
				key.CreatedBy = admin.Username
			}
		}
	}
	res, err := h.Store.APIKeys.InsertOne(r.Context(), key)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	key.ID = res.InsertedID.(primitive.ObjectID)
	h.logAction(r, ActAPIKeyCreate, "api_key", key.ID.Hex(), key.Name, strings.Join(scopes, ", "))
	httpx.JSON(w, http.StatusCreated, map[string]any{"key": key, "secret": secret})
}

// AdminRevokeAPIKey ends a key on the program's next request.
func (h *Handler) AdminRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var key models.APIKey
	err = h.Store.APIKeys.FindOneAndUpdate(r.Context(),
		bson.M{"_id": id, "revokedAt": nil},
		bson.M{"$set": bson.M{"revokedAt": time.Now()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&key)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "kalit topilmadi")
		return
	}
	h.logAction(r, ActAPIKeyRevoke, "api_key", key.ID.Hex(), key.Name, "")
	httpx.JSON(w, http.StatusOK, key)
}

// ---- The panel: webhook endpoints ----

// webhookView is an endpoint as the panel sees it — never with its secret.
type webhookView struct {
	models.WebhookEndpoint
	// Queued and not yet delivered: the one number that says "your receiver is
	// behind" before the failure count does.
	Pending int64 `json:"pending"`
}

// AdminWebhooks lists the endpoints.
func (h *Handler) AdminWebhooks(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	cur, err := h.Store.WebhookEndpoints.Find(r.Context(), bson.M{},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var eps []models.WebhookEndpoint
	if err := cur.All(r.Context(), &eps); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]webhookView, 0, len(eps))
	for _, ep := range eps {
		if ep.Events == nil {
			ep.Events = []string{}
		}
		n, _ := h.Store.WebhookDeliveries.CountDocuments(r.Context(), bson.M{
			"endpointId": ep.ID, "status": models.DeliveryPending,
		})
		out = append(out, webhookView{WebhookEndpoint: ep, Pending: n})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"endpoints": out, "events": models.WebhookEvents})
}

type webhookReq struct {
	URL     string   `json:"url"`
	Events  []string `json:"events"`
	Enabled *bool    `json:"enabled"`
}

func (req *webhookReq) validate() (string, []string, error) {
	u, err := webhook.CheckURL(req.URL)
	if err != nil {
		return "", nil, err
	}
	events := cleanChoices(req.Events, models.WebhookEvents)
	if len(events) == 0 {
		return "", nil, errors.New("kamida bitta hodisani tanlang")
	}
	return u.String(), events, nil
}

// newWebhookSecret is hex for the reason an API key is — see newAPIKey.
func newWebhookSecret() (string, error) {
	return randomHex(webhookSecretPrefix, 32)
}

// AdminCreateWebhook registers an address. ⚠️ The signing secret is in this
// answer and in AdminRotateWebhookSecret's, and nowhere else.
func (h *Handler) AdminCreateWebhook(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req webhookReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	u, events, err := req.validate()
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if n, _ := h.Store.WebhookEndpoints.CountDocuments(r.Context(), bson.M{}); n >= maxWebhookEndpoints {
		httpx.Error(w, http.StatusBadRequest, "manzillar juda ko'p — keraksizlarini o'chiring")
		return
	}
	secret, err := newWebhookSecret()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	now := time.Now()
	ep := models.WebhookEndpoint{
		URL: u, Events: events, Secret: secret, Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	res, err := h.Store.WebhookEndpoints.InsertOne(r.Context(), ep)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	ep.ID = res.InsertedID.(primitive.ObjectID)
	h.logAction(r, ActWebhookCreate, "webhook", ep.ID.Hex(), ep.URL, strings.Join(events, ", "))
	httpx.JSON(w, http.StatusCreated, map[string]any{"endpoint": webhookView{WebhookEndpoint: ep}, "secret": secret})
}

// AdminUpdateWebhook changes the address, the events or the switch.
//
// ⚠️ **Changing the URL keeps the secret.** The receiver moving house is not a
// reason to make its developer re-deploy a new secret; a leak is, and that is
// what rotate is for.
func (h *Handler) AdminUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var req webhookReq
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	u, events, err := req.validate()
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := bson.M{"url": u, "events": events, "updatedAt": time.Now()}
	if req.Enabled != nil {
		set["enabled"] = *req.Enabled
		if *req.Enabled {
			// A receiver switched back on starts from a clean count: the
			// failures were about the address as it was.
			set["consecutiveFailures"] = 0
		}
	}
	var ep models.WebhookEndpoint
	err = h.Store.WebhookEndpoints.FindOneAndUpdate(r.Context(), bson.M{"_id": id},
		bson.M{"$set": set}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&ep)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "manzil topilmadi")
		return
	}
	h.logAction(r, ActWebhookUpdate, "webhook", ep.ID.Hex(), ep.URL, strings.Join(ep.Events, ", "))
	httpx.JSON(w, http.StatusOK, webhookView{WebhookEndpoint: ep})
}

// AdminRotateWebhookSecret replaces the signing secret. Deliveries still in the
// queue are signed with the new one when they go — the old one stops at once.
func (h *Handler) AdminRotateWebhookSecret(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	secret, err := newWebhookSecret()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var ep models.WebhookEndpoint
	err = h.Store.WebhookEndpoints.FindOneAndUpdate(r.Context(), bson.M{"_id": id},
		bson.M{"$set": bson.M{"secret": secret, "updatedAt": time.Now()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&ep)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "manzil topilmadi")
		return
	}
	h.logAction(r, ActWebhookUpdate, "webhook", ep.ID.Hex(), ep.URL, "secret")
	httpx.JSON(w, http.StatusOK, map[string]any{"secret": secret})
}

// AdminDeleteWebhook removes an address and everything queued for it.
func (h *Handler) AdminDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var ep models.WebhookEndpoint
	if err := h.Store.WebhookEndpoints.FindOneAndDelete(r.Context(), bson.M{"_id": id}).Decode(&ep); err != nil {
		httpx.Error(w, http.StatusNotFound, "manzil topilmadi")
		return
	}
	_, _ = h.Store.WebhookDeliveries.DeleteMany(r.Context(), bson.M{"endpointId": id})
	h.logAction(r, ActWebhookDelete, "webhook", ep.ID.Hex(), ep.URL, "")
	w.WriteHeader(http.StatusNoContent)
}

// AdminTestWebhook sends a `ping` now, outside the queue, and says what came
// back — the "is my receiver right?" button. Not retried and not recorded as a
// delivery: it is a question, not an event.
func (h *Handler) AdminTestWebhook(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var ep models.WebhookEndpoint
	if err := h.Store.WebhookEndpoints.FindOne(r.Context(), bson.M{"_id": id}).Decode(&ep); err != nil {
		httpx.Error(w, http.StatusNotFound, "manzil topilmadi")
		return
	}
	now := time.Now()
	eventID := "evt_ping_" + primitive.NewObjectID().Hex()
	body, _ := json.Marshal(map[string]any{
		"id": eventID, "type": "ping", "createdAt": now,
		"data": map[string]any{"restaurant": h.restaurantName(r.Context())},
	})
	res := webhook.Send(r.Context(), webhook.Client, webhook.Request{
		URL: ep.URL, Secret: ep.Secret, Event: "ping", EventID: eventID,
		DeliveryID: eventID, Body: body,
	}, now)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok": res.OK(), "status": res.StatusCode, "error": res.Err, "ms": res.Took.Milliseconds(),
	})
}

// AdminWebhookDeliveries is the last fifty deliveries to one address.
func (h *Handler) AdminWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	cur, err := h.Store.WebhookDeliveries.Find(r.Context(), bson.M{"endpointId": id},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(50))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.WebhookDelivery{}
	if err := cur.All(r.Context(), &rows); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deliveries": rows})
}

// AdminRetryWebhookDelivery puts a failed (or skipped) delivery back at the
// front of the queue — for the morning after a receiver was fixed.
func (h *Handler) AdminRetryWebhookDelivery(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	res, err := h.Store.WebhookDeliveries.UpdateOne(r.Context(), bson.M{
		"_id": id, "status": bson.M{"$in": bson.A{models.DeliveryFailed, models.DeliverySkipped}},
	}, bson.M{"$set": bson.M{
		"status": models.DeliveryPending, "nextAttemptAt": time.Now(), "attempts": 0,
	}})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.MatchedCount == 0 {
		httpx.Error(w, http.StatusNotFound, "qayta yuboriladigan yetkazma topilmadi")
		return
	}
	kickWebhooks()
	w.WriteHeader(http.StatusNoContent)
}
