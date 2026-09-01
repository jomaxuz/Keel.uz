package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/i18n"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/push"
)

// ---- Telling a courier something they cannot see by looking ----
//
// ⚠️ **A courier is the one person in this system who is not near a screen.**
// The waiter has a floor to walk, the kitchen has a monitor, the owner has the
// panel — a courier is on a bike with the phone in a pocket, and everything
// that happens to their orders happens while they are not looking. Until this
// existed the app answered by polling every twenty seconds, which is a battery
// cost paid all evening to learn something that happens twice.
//
// ⚠️ **Every event here is one the courier has to act on**, and that is the
// filter. "Assigned to you", "taken off you", "cancelled", "ready to collect",
// "the address moved", "your cash was accepted" — each one changes where the
// rider goes next or what they are carrying. Anything that does not is left to
// the list they can pull down.

type courierDeviceRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
	// ⚠️ The phone's language, sent with the token rather than read from the
	// account. Notifications are the only text in this app written by the
	// server, so the device cannot translate them itself — and the choice
	// belongs to whoever holds the phone.
	Lang string `json:"lang"`
}

// CourierRegisterDevice remembers this phone for the signed-in courier.
func (h *Handler) CourierRegisterDevice(w http.ResponseWriter, r *http.Request) {
	c, ok := h.courierFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var req courierDeviceRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	token := strings.TrimSpace(req.Token)
	// ⚠️ Refused here rather than at send time: an unusable token otherwise
	// sits in the collection forever, failing quietly once per event.
	if !push.IsExpoToken(token) {
		httpx.Error(w, http.StatusBadRequest, "token noto'g'ri")
		return
	}
	// ⚠️ **Keyed on the token, not on the courier.** A phone that changes hands
	// must move to its new rider rather than leave the previous one subscribed
	// — otherwise tomorrow's addresses go to somebody who has left. The unique
	// index is what makes this an upsert rather than a second row.
	_, err := h.Store.CourierDevices.UpdateOne(r.Context(),
		bson.M{"token": token},
		bson.M{"$set": bson.M{
			"courierId": c.ID,
			"platform":  req.Platform,
			"lang":      langOrUZ(req.Lang),
			"updatedAt": time.Now(),
		}},
		options.Update().SetUpsert(true))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// CourierForgetDevice drops this phone.
//
// ⚠️ **Called on sign-out, and it is not tidiness.** A courier's phone is the
// one most likely to be sold or handed on, and a token left behind delivers
// customer names, phone numbers and addresses to whoever holds it next — with
// no way for them to stop it from their side.
func (h *Handler) CourierForgetDevice(w http.ResponseWriter, r *http.Request) {
	c, ok := h.courierFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var req courierDeviceRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// ⚠️ Scoped to this courier: a token alone must not let one phone
	// unsubscribe another. The id in the filter is the permission, as
	// everywhere else here.
	_, _ = h.Store.CourierDevices.DeleteOne(r.Context(),
		bson.M{"token": strings.TrimSpace(req.Token), "courierId": c.ID})
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func langOrUZ(v string) string {
	switch v {
	case i18n.RU, i18n.EN:
		return v
	}
	return i18n.UZ
}

// notifyCourier sends one message to every phone a courier has.
//
// ⚠️ **Fire and forget, on purpose.** Every caller has already written down the
// thing being announced, and none of them may fail because a relay in another
// country did. It takes its own context for the same reason: the request that
// triggered it has already answered.
//
// ⚠️ **Written in Uzbek and translated per device.** This is the same catalogue
// the API's error messages use (`internal/i18n`), including its patterns — so
// "#12 buyurtma sizga berildi. Manzil: Chilonzor 5" is matched against
// "#%s buyurtma sizga berildi. Manzil: %s" and the values are dropped into the
// Russian sentence untranslated, which is what a street name wants.
func (h *Handler) notifyCourier(
	courierID primitive.ObjectID, title, body string, data map[string]any,
) {
	if courierID.IsZero() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		cur, err := h.Store.CourierDevices.Find(ctx, bson.M{"courierId": courierID})
		if err != nil {
			return
		}
		var devices []models.CourierDevice
		if err := cur.All(ctx, &devices); err != nil || len(devices) == 0 {
			return
		}
		msgs := make([]push.Message, 0, len(devices))
		for _, d := range devices {
			msgs = append(msgs, push.Message{
				To:    d.Token,
				Title: i18n.Localize(d.Lang, title),
				Body:  i18n.Localize(d.Lang, body),
				Sound: "default",
				Data:  data,
				// The two spellings of the channel have to agree; the app
				// creates it with this id at startup.
				ChannelID: push.DeliveryChannel,
				// ⚠️ High, and this is what it is for: the phone is in a pocket
				// on a moving bike, and a notification that waits for the next
				// unlock arrives after the food is cold.
				Priority: "high",
			})
		}
		dead := push.Send(ctx, msgs, log.Printf)
		if len(dead) > 0 {
			// A reinstalled phone keeps a row otherwise, and every event after
			// it pays for a delivery nobody receives.
			_, _ = h.Store.CourierDevices.DeleteMany(ctx,
				bson.M{"token": bson.M{"$in": dead}})
		}
	}()
}

// ---- The events themselves ----
//
// Each one is a small function rather than a call site with a string in it, so
// that the wording of an event lives in one place and the catalogue has exactly
// one key to hold. They are named for what the courier reads, not for the
// handler that happens to call them — several are called from two places.

// courierGotOrder: an order has been handed to this courier.
func (h *Handler) courierGotOrder(courierID primitive.ObjectID, o *models.Order) {
	where := o.Address.Text
	if where == "" {
		where = o.Customer.Name
	}
	h.notifyCourier(courierID, "Yangi buyurtma",
		fmt.Sprintf("#%s buyurtma sizga berildi. Manzil: %s", o.Number, where),
		map[string]any{"type": "assigned", "orderId": o.ID.Hex()})
}

// courierLostOrder: it was taken off them — reassigned, detached, or moved to
// another branch. ⚠️ Sent to the *previous* courier, and it matters: without it
// a rider keeps riding to an address that is no longer theirs, and finds out at
// the door.
func (h *Handler) courierLostOrder(courierID primitive.ObjectID, o *models.Order) {
	h.notifyCourier(courierID, "Buyurtma olindi",
		fmt.Sprintf("#%s buyurtma sizdan olindi", o.Number),
		map[string]any{"type": "unassigned", "orderId": o.ID.Hex()})
}

// courierOrderCancelled: the restaurant cancelled an order this courier holds.
func (h *Handler) courierOrderCancelled(o *models.Order) {
	reason := strings.TrimSpace(o.CancelReason)
	body := fmt.Sprintf("#%s buyurtma bekor qilindi", o.Number)
	if reason != "" {
		body = fmt.Sprintf("#%s buyurtma bekor qilindi: %s", o.Number, reason)
	}
	h.notifyCourier(o.CourierID, "Buyurtma bekor qilindi", body,
		map[string]any{"type": "cancelled", "orderId": o.ID.Hex()})
}

// courierOrderReady: the kitchen has finished it and it is waiting to be
// collected. ⚠️ The one event a courier used to learn by asking somebody.
func (h *Handler) courierOrderReady(o *models.Order) {
	h.notifyCourier(o.CourierID, "Buyurtma tayyor",
		fmt.Sprintf("#%s buyurtma tayyor — olib chiqing", o.Number),
		map[string]any{"type": "ready", "orderId": o.ID.Hex()})
}

// courierAddressMoved: the panel corrected the pin.
//
// ⚠️ **This one is not a courtesy.** The pin is what the arrival check measures
// against, so a courier standing at the old address has a "delivered" button
// that will not open and no way to know why — the correction happened on
// somebody else's screen.
func (h *Handler) courierAddressMoved(o *models.Order, address string) {
	h.notifyCourier(o.CourierID, "Manzil o'zgardi",
		fmt.Sprintf("#%s buyurtmaning manzili o'zgardi: %s", o.Number, address),
		map[string]any{"type": "address", "orderId": o.ID.Hex()})
}

// courierCashTaken: the restaurant recorded a handover.
//
// ⚠️ Sent because the courier's own screen shows what they still owe, and a
// number that changes with no explanation is a number they will ask about.
func (h *Handler) courierCashTaken(courierID primitive.ObjectID, amount int) {
	h.notifyCourier(courierID, "Naqd qabul qilindi",
		// ⚠️ The sum is a number inside the sentence rather than a
		// pre-formatted string: "12000 so'm" carries an Uzbek word into a
		// Russian notification, because a captured value is never translated.
		fmt.Sprintf("%d so'm naqd pul qabul qilindi", amount),
		map[string]any{"type": "settled"})
}

// courierAccountOff: the account was switched off while the app was signed in.
//
// ⚠️ Otherwise the first sign of it is a shift switch that refuses to move, in
// the middle of an evening, with nothing on the screen naming the cause.
func (h *Handler) courierAccountOff(courierID primitive.ObjectID) {
	h.notifyCourier(courierID, "Hisob o'chirildi",
		"Hisobingiz vaqtincha o'chirildi — restoran bilan bog'laning",
		map[string]any{"type": "account"})
}
