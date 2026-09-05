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
	"restaurant-backend/internal/i18n"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/push"
)

// ---- Telling the owner, on the phone they already carry ----
//
// ⚠️ **The loss alerts went to Telegram and nowhere else.** A discount after
// the bill was printed, a till short at the close, a dish written off — the
// whole point of them is that somebody sees them the same evening, and a
// restaurant that never linked a chat was told none of it. Nothing was broken
// and nothing said so: the records piled up in a list nobody opens.
//
// ⚠️ **So push is sent *as well as* Telegram, never instead.** A group chat
// keeps a searchable history, survives the owner changing their number and can
// hold an accountant; a phone buzzes tonight. They answer different halves of
// the same need, and the codebase has already learned once what chaining two
// independent deliveries with an `else if` costs (see the Caddy note in
// CLAUDE.md).

type adminDeviceRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
	Lang     string `json:"lang"`
}

// AdminRegisterDevice remembers this phone for the signed-in panel account.
func (h *Handler) AdminRegisterDevice(w http.ResponseWriter, r *http.Request) {
	admin, err := h.adminUser(r)
	if err != nil || admin == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req adminDeviceRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	token := strings.TrimSpace(req.Token)
	if !push.IsPushToken(token) {
		httpx.Error(w, http.StatusBadRequest, "token noto'g'ri")
		return
	}
	// ⚠️ **A native token is refused while Firebase is unconfigured, and that
	// refusal is the feature.** Stored, it would look exactly like a working
	// registration: the row is there, the settings screen says "on", and not one
	// notification ever arrives. Said here, the phone learns it at sign-in —
	// which is the only moment anybody is in a position to fix it.
	if push.IsFCMToken(token) && !push.FCMReady() {
		httpx.Error(w, http.StatusServiceUnavailable,
			"bildirishnomalar serverda sozlanmagan")
		return
	}
	// ⚠️ **Keyed on the token.** A phone handed over — a manager promoted, a
	// device replaced — must move to whoever signs in on it rather than leave
	// the previous account subscribed to a restaurant's takings.
	//
	// ⚠️ The role and the branch are copied on every registration, so an
	// account that changes either stops or starts receiving accordingly
	// without anybody touching the phone.
	_, err = h.Store.AdminDevices.UpdateOne(r.Context(),
		bson.M{"token": token},
		bson.M{"$set": bson.M{
			"adminId":   admin.ID,
			"branchId":  admin.BranchID,
			"role":      admin.Role,
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

// AdminForgetDevice drops this phone.
//
// ⚠️ Called on sign-out. What this channel carries is the restaurant's money,
// its discounts and who took them; a token left behind delivers all of it to
// whoever holds the handset next.
func (h *Handler) AdminForgetDevice(w http.ResponseWriter, r *http.Request) {
	admin, err := h.adminUser(r)
	if err != nil || admin == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req adminDeviceRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// Scoped to this account: a token alone must not let one phone unsubscribe
	// another.
	_, _ = h.Store.AdminDevices.DeleteOne(r.Context(),
		bson.M{"token": strings.TrimSpace(req.Token), "adminId": admin.ID})
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// adminText is the ordinary case: one sentence, translated per device out of
// the catalogue.
func adminText(title, body string) func(string) (string, string) {
	return func(lang string) (string, string) {
		return i18n.Localize(lang, title), i18n.Localize(lang, body)
	}
}

// notifyAdmins sends to the phones watching one branch.
//
// ⚠️ **`ownersOnly` is an access rule, not a preference.** A manager is one of
// the people the loss alerts are *about*: "Dilnoza took 400 000 off table 6"
// delivered to Dilnoza is not a leak of anything she does not know — it is a
// warning that she has been noticed, which is the opposite of what the channel
// is for. The Telegram path has drawn that line since it existed; this one
// draws the same one.
//
// ⚠️ **The text is built per language rather than translated per string.** The
// alerts are composed at runtime out of a words table (`notifyWordsFor`), so
// there is no catalogue key to look up — the caller is handed the language and
// answers with the finished sentence. Devices are grouped, so a restaurant with
// three owners in two languages builds two texts, not six.
func (h *Handler) notifyAdmins(
	branchID primitive.ObjectID,
	ownersOnly bool,
	text func(lang string) (title, body string),
	data map[string]any,
) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		filter := bson.M{}
		if ownersOnly {
			filter["role"] = "owner"
		}
		// ⚠️ An account with no branch is the company's — an owner — and has to
		// hear about every branch. A manager hears about theirs and no other:
		// the panel already refuses them another branch's orders, and a
		// notification is the one path that could have leaked one.
		if !branchID.IsZero() {
			filter["$or"] = []bson.M{
				{"branchId": bson.M{"$exists": false}},
				{"branchId": primitive.NilObjectID},
				{"branchId": branchID},
			}
		}
		cur, err := h.Store.AdminDevices.Find(ctx, filter)
		if err != nil {
			return
		}
		var devices []models.AdminDevice
		if err := cur.All(ctx, &devices); err != nil || len(devices) == 0 {
			return
		}

		built := map[string][2]string{}
		msgs := make([]push.Message, 0, len(devices))
		for _, d := range devices {
			lang := langOrUZ(d.Lang)
			if _, ok := built[lang]; !ok {
				title, body := text(lang)
				built[lang] = [2]string{title, body}
			}
			msgs = append(msgs, push.Message{
				To:        d.Token,
				Title:     built[lang][0],
				Body:      built[lang][1],
				Sound:     "default",
				Data:      data,
				ChannelID: push.OwnerChannel,
				// ⚠️ High, and this is the one channel where it is least
				// arguable: the events here are money leaving the building
				// while the person who can stop it is somewhere else.
				Priority: "high",
			})
		}
		dead := push.Send(ctx, msgs, log.Printf)
		if len(dead) > 0 {
			_, _ = h.Store.AdminDevices.DeleteMany(ctx,
				bson.M{"token": bson.M{"$in": dead}})
		}
	}()
}
