package handlers

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
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/push"
)

// ---- Telling a waiter their food is ready ----
//
// ⚠️ **The one thing a waiter cannot find out by looking.** Everything else on
// their phone is a screen they can open: the room, the check, their hours. Food
// reaching the pass happens in another part of the building, and until now the
// only ways to learn it were a bell, a shout, or walking over — which is the
// walk this notification removes.
//
// ⚠️ **To one person, not to a branch.** A broadcast is how a restaurant
// teaches its staff to swipe notifications away without reading them, and the
// one that mattered goes with the rest.

type registerDeviceRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

// StaffRegisterDevice remembers this phone for the signed-in employee.
func (h *Handler) StaffRegisterDevice(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var req registerDeviceRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	token := strings.TrimSpace(req.Token)
	// ⚠️ Refused here rather than at send time: an unusable token otherwise
	// sits in the collection forever, failing quietly once per kitchen event.
	if !push.IsExpoToken(token) {
		httpx.Error(w, http.StatusBadRequest, "token noto'g'ri")
		return
	}
	now := time.Now()
	// ⚠️ **Keyed on the token, not on the employee.** A phone handed to somebody
	// else must move to them rather than leave the previous person subscribed —
	// otherwise the evening's tables are announced to whoever went home. The
	// unique index is what makes the upsert an upsert.
	_, err := h.Store.StaffDevices.UpdateOne(r.Context(),
		bson.M{"token": token},
		bson.M{"$set": bson.M{
			"staffId":   s.ID,
			"branchId":  s.BranchID,
			"platform":  req.Platform,
			"updatedAt": now,
		}},
		options.Update().SetUpsert(true))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// StaffForgetDevice drops this phone.
//
// ⚠️ **Called on sign-out, and that is not tidiness.** Phones are handed over at
// the end of a shift; a token left behind sends the next evening's tables to
// the person who went home, and they cannot turn it off from their side.
func (h *Handler) StaffForgetDevice(w http.ResponseWriter, r *http.Request) {
	s, ok := h.staffFromCtx(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var req registerDeviceRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// ⚠️ Scoped to this employee: a token alone must not let one phone
	// unsubscribe another. The id in the filter is the permission, as
	// everywhere else here.
	_, _ = h.Store.StaffDevices.DeleteOne(r.Context(),
		bson.M{"token": strings.TrimSpace(req.Token), "staffId": s.ID})
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// notifyStaff sends one message to every phone a person has.
//
// ⚠️ **Fire and forget, on purpose.** This is called after the thing it
// announces has been written down, and it must never be the reason that write
// is reported as having failed. It takes its own context for the same reason:
// the request that triggered it has already answered.
func (h *Handler) notifyStaff(
	staffID primitive.ObjectID, title, body string, data map[string]any,
) {
	if staffID.IsZero() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		cur, err := h.Store.StaffDevices.Find(ctx, bson.M{"staffId": staffID})
		if err != nil {
			return
		}
		var devices []models.StaffDevice
		if err := cur.All(ctx, &devices); err != nil || len(devices) == 0 {
			return
		}
		msgs := make([]push.Message, 0, len(devices))
		for _, d := range devices {
			msgs = append(msgs, push.Message{
				To:    d.Token,
				Title: title,
				Body:  body,
				Sound: "default",
				Data:  data,
				// The two spellings of the channel have to agree; the app
				// creates it with this id at startup.
				ChannelID: push.KitchenChannel,
				// ⚠️ High, and this is the case it is for: a dish at the pass
				// is going cold while the phone decides whether to wake up.
				Priority: "high",
			})
		}
		dead := push.Send(ctx, msgs, log.Printf)
		if len(dead) > 0 {
			// A reinstalled phone keeps a row otherwise, and every event after
			// it pays for a delivery nobody receives.
			_, _ = h.Store.StaffDevices.DeleteMany(ctx,
				bson.M{"token": bson.M{"$in": dead}})
		}
	}()
}
