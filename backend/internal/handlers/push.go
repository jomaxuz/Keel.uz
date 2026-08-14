package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/webpush"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Browser push, from the server's side: the keys, the subscriptions, and one
// send.
//
// ⚠️ **Nothing here is configured by the restaurant.** The VAPID keypair is
// generated on first use and never shown to anybody. This is deliberate and it
// is what makes the channel usable at all: a feature that begins with "ask your
// owner to register with a push provider" is a feature no restaurant in this
// product would ever switch on, and the keys are just a keypair — there is
// nobody to register with.

// pushTTL is how long a push service should hold a message for a browser that
// is currently offline.
//
// ⚠️ Twelve hours rather than the maximum. These are restaurant promotions:
// "20% off until tonight" delivered on Thursday morning is worse than not
// delivered, because it makes the restaurant look careless rather than generous.
const pushTTL = 12 * time.Hour

// pushClient is the HTTP client every push goes through.
//
// Its own client with a short timeout rather than `http.DefaultClient`: a
// campaign walks hundreds of these in a row, and the default has **no timeout
// at all** — one push service holding a connection open would stall the whole
// send, in a goroutine nobody is watching, with the progress counter frozen
// where it stopped.
var pushClient = &http.Client{Timeout: 10 * time.Second}

// pushKeys returns the restaurant's VAPID identity, creating it on first use.
//
// ⚠️ The insert is guarded by a unique index on a constant field rather than by
// this mutex alone: the mutex covers one process, and the create-on-read
// pattern is exactly the shape that produces two documents when two guests load
// the site at the same moment. A second identity is not a harmless duplicate —
// half the subscriptions would be signed with a key the push service does not
// recognise, and only half the notifications would arrive.
var pushKeysOnce sync.Mutex

func (h *Handler) pushKeys(ctx context.Context) (webpush.Keys, error) {
	load := func() (webpush.Keys, bool) {
		var s models.PushSettings
		if err := h.Store.PushSettings.FindOne(ctx, bson.M{}).Decode(&s); err != nil {
			return webpush.Keys{}, false
		}
		return webpush.Keys{
			Public: s.PublicKey, Private: s.PrivateKey, Subject: s.Subject,
		}, s.PublicKey != "" && s.PrivateKey != ""
	}
	if keys, ok := load(); ok {
		return keys, nil
	}

	pushKeysOnce.Lock()
	defer pushKeysOnce.Unlock()
	// Read again inside the lock: the request that was waiting for it has
	// almost certainly had the keys created underneath it.
	if keys, ok := load(); ok {
		return keys, nil
	}

	// The subject is a contact address for the push services, not an identity
	// check. The public base URL is the honest answer and needs no setting.
	subject := strings.TrimSpace(h.Cfg.PublicBaseURL)
	if subject == "" {
		subject = "https://keel.uz"
	}
	keys, err := webpush.NewKeys(subject)
	if err != nil {
		return webpush.Keys{}, err
	}
	doc := models.PushSettings{
		PublicKey: keys.Public, PrivateKey: keys.Private,
		Subject: keys.Subject, CreatedAt: time.Now(),
	}
	if _, err := h.Store.PushSettings.InsertOne(ctx, doc); err != nil {
		// A duplicate here means another process won the race; its keys are as
		// good as ours and are already the ones being handed to browsers.
		if keys, ok := load(); ok {
			return keys, nil
		}
		return webpush.Keys{}, err
	}
	return keys, nil
}

// PushPublicKey hands the browser the key it needs to subscribe.
//
// Public, and correctly so: this key is delivered to every visitor by
// definition — a subscription cannot be created without it. Same category as
// the map key (§ "Xarita kaliti sir emas"), and the exact opposite of the
// private half sitting beside it in the same document.
func (h *Handler) PushPublicKey(w http.ResponseWriter, r *http.Request) {
	keys, err := h.pushKeys(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"publicKey": keys.Public})
}

type pushSubscribeRequest struct {
	Endpoint string `json:"endpoint" validate:"required"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
	Device string `json:"device"`
}

// PushSubscribe stores a browser's subscription against the signed-in customer.
//
// ⚠️ **Requires a signed-in account.** An anonymous subscription could be
// notified but never *chosen* — every audience in this system is built from
// order history, and a row with no customer behind it can only ever be
// messaged by "everybody", which is not an audience this product offers. It
// would also be un-revocable by the person it belongs to.
func (h *Handler) PushSubscribe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	userID, err := objectID(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req pushSubscribeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Keys.P256dh == "" || req.Keys.Auth == "" {
		httpx.Error(w, http.StatusBadRequest, "obuna kalitlari yetishmayapti")
		return
	}

	var user models.User
	_ = h.Store.Users.FindOne(r.Context(), bson.M{"_id": userID}).Decode(&user)

	now := time.Now()
	// ⚠️ Upsert on the endpoint, not insert. Browsers re-register silently
	// after an update, and a plain insert would give the guest a second copy of
	// every campaign — then a third. The endpoint is the browser's own identity
	// for this subscription, which is what makes it the right key.
	_, err = h.Store.PushSubscriptions.UpdateOne(r.Context(),
		bson.M{"endpoint": req.Endpoint},
		bson.M{
			"$set": bson.M{
				"userId": userID,
				"p256dh": req.Keys.P256dh,
				"auth":   req.Keys.Auth,
				"lang":   notifyLang(&user),
				"device": clampText(req.Device, 120),
				"seenAt": now,
			},
			"$setOnInsert": bson.M{"createdAt": now},
		},
		options.Update().SetUpsert(true))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type pushUnsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

// PushUnsubscribe forgets one browser.
//
// ⚠️ Narrowed to the caller's own account as well as the endpoint. The endpoint
// is unguessable, which makes it a secret in practice — but "in practice" is
// not a permission check, and a filter costs nothing.
func (h *Handler) PushUnsubscribe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	userID, err := objectID(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req pushUnsubscribeRequest
	_ = httpx.Decode(r, &req)

	filter := bson.M{"userId": userID}
	if strings.TrimSpace(req.Endpoint) != "" {
		filter["endpoint"] = req.Endpoint
	}
	// An empty endpoint removes every device: "stop notifying me" is what the
	// guest means when they switch it off in their profile on a phone they may
	// not even be holding.
	if _, err := h.Store.PushSubscriptions.DeleteMany(r.Context(), filter); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// subscriptionsOf returns every browser one customer has registered.
func (h *Handler) subscriptionsOf(ctx context.Context, userID any) ([]models.PushSubscription, error) {
	cur, err := h.Store.PushSubscriptions.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return nil, err
	}
	var out []models.PushSubscription
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// subscribedUserSet is every customer with at least one browser subscribed.
//
// A distinct-ids query rather than a lookup per candidate: the audience loop
// walks the whole customer base, and doing this inside it would make the
// preview button quadratic in the number of customers — on the one screen an
// owner presses before spending money, and therefore watches.
func (h *Handler) subscribedUserSet(ctx context.Context) (map[string]bool, error) {
	ids, err := h.Store.PushSubscriptions.Distinct(ctx, "userId", bson.M{})
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		if hex, ok := id.(interface{ Hex() string }); ok {
			out[hex.Hex()] = true
		}
	}
	return out, nil
}

// PushMessage is what a notification says.
type PushMessage struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	// Where clicking it goes. Relative to the site, always: an absolute URL
	// here would be a stored open redirect delivered by the restaurant's own
	// notification.
	URL string `json:"url,omitempty"`
	// A dish photo or the restaurant's logo.
	Icon string `json:"icon,omitempty"`
}

// sendPush delivers one message to every browser a customer has registered.
//
// Returns how many arrived. ⚠️ **A customer with three devices counts once**:
// the caller is reporting "how many people did we reach", and a campaign that
// claimed 240 sends to 90 people would be worse than no number at all.
func (h *Handler) sendPush(ctx context.Context, keys webpush.Keys, subs []models.PushSubscription, msg PushMessage) (int, error) {
	if len(subs) == 0 {
		return 0, nil
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}

	delivered := 0
	var firstErr error
	for _, sub := range subs {
		err := webpush.Send(pushClient, keys, webpush.Subscription{
			Endpoint: sub.Endpoint, P256dh: sub.P256dh, Auth: sub.Auth,
		}, payload, pushTTL)

		switch {
		case err == nil:
			delivered++
		case errors.Is(err, webpush.ErrGone):
			// ⚠️ Deleted, not retried. The guest cleared their site data or
			// revoked the permission; this subscription will never work again,
			// and keeping it means paying for the attempt on every campaign for
			// ever while the delivery count quietly overstates the reach.
			_, _ = h.Store.PushSubscriptions.DeleteOne(ctx, bson.M{"endpoint": sub.Endpoint})
		default:
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	if delivered > 0 {
		// One device answering is the person reached. Reporting the error
		// instead would mark a delivered message as failed because their old
		// laptop is switched off.
		return 1, nil
	}
	return 0, firstErr
}

// pushCampaign sends one campaign message to one customer's browsers.
//
// ⚠️ **The title is the restaurant, the body is the message.** A notification
// arrives on a lock screen next to a dozen others, and the guest decides in one
// glance whether to read it. A title that said "Yangi xabar" would be indistinguishable
// from every other app; the restaurant's name is the only part that earns the
// glance — and it comes from the brand, not the company (§ "Restoran nomi
// brenddan olinadi"), which is the trap that has already bitten three times.
func (h *Handler) pushCampaign(ctx context.Context, keys webpush.Keys,
	m audienceMember, text, image string) error {
	subs, err := h.subscriptionsOf(ctx, m.UserID)
	if err != nil {
		return err
	}
	if len(subs) == 0 {
		// Subscribed when the audience was built, gone by the time we got here:
		// a browser that revoked in between. Not an error — nothing failed, the
		// recipient simply stopped existing.
		return nil
	}
	sent, err := h.sendPush(ctx, keys, subs, PushMessage{
		Title: h.restaurantName(ctx),
		Body:  text,
		// Straight to the menu. A notification that opens the front page makes
		// the guest find the offer again, which is where most of them stop.
		URL:  "/menu",
		Icon: image,
	})
	if err != nil {
		return err
	}
	if sent == 0 {
		return errors.New("hech bir qurilmaga yetkazilmadi")
	}
	return nil
}
