package handlers

// ---- Telling a guest what happened to their dinner ----
//
// ⚠️ **The one thing a guest cannot find out by looking.** Everything else in
// the app is a screen they can open. Whether the kitchen has started, and
// whether somebody has left with it, happens in a building they are not in — and
// until this existed the only ways to learn it were to keep the app open or to
// ring the restaurant, which is the call this removes.
//
// ⚠️ **One kind of message, and the restraint is the feature.** A guest has one
// switch for this app's notifications, and everything we might ever want to send
// them — an offer, a reminder, a new menu — goes through the same switch. Spend
// it on anything but their own order and it gets turned off, taking the order
// updates with it.
//
// ⚠️ **Nothing is sent for a status the guest already caused.** Placing an order
// puts them on the tracking screen; a notification a second later tells somebody
// something they are looking at.

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/i18n"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/push"
)

// UserRegisterDevice remembers this phone for the signed-in guest.
func (h *Handler) UserRegisterDevice(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r.Context())
	userID, err := objectID(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Token    string `json:"token"`
		Lang     string `json:"lang"`
		Platform string `json:"platform"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	token := strings.TrimSpace(req.Token)
	// ⚠️ Refused here rather than at send time: an unusable token otherwise sits
	// in the collection forever, failing quietly once per order.
	if !push.IsPushToken(token) {
		httpx.Error(w, http.StatusBadRequest, "token noto'g'ri")
		return
	}
	now := time.Now()
	// ⚠️ **Upsert on the token, not insert.** The app re-registers on every
	// launch; a plain insert would deliver every update as many times as the app
	// has been opened, which reads as the restaurant spamming and is the fastest
	// way to have notifications switched off.
	//
	// ⚠️ **The user is part of the update, not part of the key.** A phone handed
	// to somebody else — sold, lent, shared in a family — must follow its new
	// owner, and a row keyed by both would leave the old one behind receiving
	// somebody else's dinner.
	_, err = h.Store.UserDevices.UpdateOne(r.Context(),
		bson.M{"token": token},
		bson.M{"$set": bson.M{
			"userId": userID, "lang": strings.TrimSpace(req.Lang),
			"platform": strings.TrimSpace(req.Platform), "updatedAt": now,
		}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// UserForgetDevice drops this phone.
//
// ⚠️ **Called before the token is cleared, on sign-out.** The other order sends
// the request unauthenticated, the row stays, and the next person's order
// updates go to a phone that has changed hands.
func (h *Handler) UserForgetDevice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	_ = httpx.DecodeOptional(r, &req)
	token := strings.TrimSpace(req.Token)
	if token == "" {
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	// ⚠️ Deleted by token alone, deliberately. A guest signing out is entitled to
	// stop this phone being told things whether or not the session is still
	// valid — and a delete narrowed by user id would fail exactly when the token
	// has expired, which is the commonest reason somebody signs out.
	_, _ = h.Store.UserDevices.DeleteMany(r.Context(), bson.M{"token": token})
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// pushOrderStatus tells the guest's phone what has happened to their order.
//
// ⚠️ **Beside the Telegram message, not instead of it** (`notifyOrderStatus` in
// notify.go). They reach different people: a guest who ordered through the bot
// reads the bot, and one who installed the restaurant's app reads the app. A
// guest who did both gets two, which is the honest outcome — they asked for both
// — and is far better than us guessing which one they meant.
//
// ⚠️ **"Which statuses are news" is asked of `notifiableStatus`**, the same
// function the Telegram path asks. Two lists would drift, and the drift is
// invisible: one channel simply stops mentioning a step, and nobody notices
// until a guest asks why the app went quiet.
func (h *Handler) pushOrderStatus(order *models.Order) {
	if order == nil || order.UserID.IsZero() || !notifiableStatus(order.Status) {
		return
	}
	title, body := orderStatusWords(order.Status)
	if title == "" {
		return
	}
	h.notifyUser(order.UserID, title, body, map[string]any{
		"type": "order", "number": order.Number, "status": string(order.Status),
	})
}

// orderStatusWords is the sentence for each step, in Uzbek — translated per
// device on the way out, like every other server-written message.
//
// ⚠️ **Its own wording rather than `orderStatusMessage`'s, and the shapes are
// genuinely different.** That one composes a single Telegram sentence carrying
// the restaurant's name, the number and the cancellation reason, in the
// language stored on the account. A notification is a title and a body, it is
// read on a lock screen where the app's own icon already says which restaurant,
// and it is translated per **device** — because the phone, not the account,
// is what has a language here.
func orderStatusWords(status models.OrderStatus) (title, body string) {
	switch status {
	case models.StatusConfirmed:
		return "Buyurtma qabul qilindi", "Restoran buyurtmangizni qabul qildi"
	case models.StatusPreparing:
		return "Buyurtma tayyorlanmoqda", "Oshxona buyurtmangizni tayyorlay boshladi"
	case models.StatusOnTheWay:
		return "Buyurtma yo'lda", "Kuryer buyurtmangiz bilan yo'lga chiqdi"
	case models.StatusDelivered:
		return "Buyurtma yetkazildi", "Yoqimli ishtaha!"
	case models.StatusCancelled:
		return "Buyurtma bekor qilindi", "Batafsil ma'lumot uchun restoranga murojaat qiling"
	}
	return "", ""
}

// notifyUser sends one message to every phone this guest has registered.
//
// ⚠️ **Failure is silent and must be.** A guest's notification is a courtesy on
// top of a screen that already shows the same thing; failing the status change
// that triggered it — which is a restaurant's actual work — because a phone is
// unreachable would be the tail wagging the dog.
func (h *Handler) notifyUser(
	userID primitive.ObjectID, title, body string, data map[string]any,
) {
	if userID.IsZero() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		cur, err := h.Store.UserDevices.Find(ctx, bson.M{"userId": userID})
		if err != nil {
			return
		}
		var devices []models.UserDevice
		if err := cur.All(ctx, &devices); err != nil || len(devices) == 0 {
			return
		}
		msgs := make([]push.Message, 0, len(devices))
		for _, d := range devices {
			msgs = append(msgs, push.Message{
				To: d.Token,
				// ⚠️ Translated per device out of the same catalogue the API's
				// errors use, patterns included.
				Title: i18n.Localize(d.Lang, title),
				Body:  i18n.Localize(d.Lang, body),
				Sound: "default",
				Data:  data,
				// ⚠️ The channel the phone actually created. A channel that was
				// never created arrives silent and unranked, which looks exactly
				// like a notification nobody sent.
				ChannelID: "orders",
				Priority:  "high",
			})
		}
		dead := push.Send(ctx, msgs, log.Printf)
		if len(dead) > 0 {
			// A reinstalled phone keeps a row otherwise, and every order after
			// it pays for a delivery nobody receives.
			_, _ = h.Store.UserDevices.DeleteMany(ctx,
				bson.M{"token": bson.M{"$in": dead}})
		}
	}()
}
