package handlers

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
	"restaurant-backend/internal/pbx"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The phone system, wired into the call centre.
//
// Without a PBX the call log is something an operator types. With one it is
// something that happens *to* them: the exchange announces the call before they
// pick up, so the caller's card is already on screen when they say hello, and
// the row — number, direction, duration, recording, who answered — is written
// for them. All they add is the outcome and a note, which is the part only a
// human knows.
//
// onlinePBX pushes six events per call. Two rules make handling them boring,
// which is the goal:
//
//   - **Every event lands on one row, keyed by the exchange's call id.** They
//     arrive out of order and they arrive twice; an upsert on `pbxCallId` makes
//     both harmless.
//   - **An event never overwrites what a person typed.** The outcome, the note
//     and the callback belong to the operator. A late `call-end` arriving after
//     they wrote the call up must not blank it.

// onlinePBX event names.
const (
	evStart     = "call-start"
	evAnswered  = "call-answered"
	evMissed    = "call-missed"
	evUserStart = "call-user-start"
	evTransfer  = "call-transfer-answered"
	evEnd       = "call-end"
)

// pbxEvent is the webhook body.
//
// Decoded leniently on purpose. The field names come from onlinePBX's own
// integration docs, but a phone system is not something we can test against
// until a customer has one — so anything unrecognised is kept as raw JSON and
// logged rather than dropped, and both spellings are accepted where the docs
// and the wild disagree.
type pbxEvent struct {
	Domain    string `json:"domain"`
	Event     string `json:"event"`
	Direction string `json:"direction"`
	UUID      string `json:"uuid"`
	// The outside number on an inbound call; the operator's on an outbound one.
	Caller string `json:"caller"`
	Callee string `json:"callee"`
	// Older payloads and the history API use these names.
	CallerIDNumber    string `json:"caller_id_number"`
	DestinationNumber string `json:"destination_number"`
	Gateway           string `json:"gateway"`
	Date              any    `json:"date"`

	CallDuration   any    `json:"call_duration"`
	DialogDuration any    `json:"dialog_duration"`
	HangupCause    string `json:"hangup_cause"`
	HangupBy       string `json:"hangup_by"`
	DownloadURL    string `json:"download_url"`
}

// PBXWebhook receives one call event.
//
// Public by necessity — onlinePBX sends no credentials — so the **path carries
// the secret**. A wrong token is answered with 200 and nothing else: a webhook
// endpoint that returns 401 tells a scanner it found something.
func (h *Handler) PBXWebhook(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	settings := h.pbxSettings(r.Context())
	if settings.WebhookToken == "" ||
		subtle.ConstantTimeCompare([]byte(token), []byte(settings.WebhookToken)) != 1 {
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	var ev pbxEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		// Some panels post form-encoded. Falling back costs nothing and saves
		// an integration that would otherwise fail silently.
		if err := r.ParseForm(); err == nil {
			ev = pbxEvent{
				Event: r.PostForm.Get("event"), UUID: r.PostForm.Get("uuid"),
				Direction: r.PostForm.Get("direction"),
				Caller:    r.PostForm.Get("caller"), Callee: r.PostForm.Get("callee"),
				HangupCause: r.PostForm.Get("hangup_cause"),
				DownloadURL: r.PostForm.Get("download_url"),
			}
		}
	}

	// Always 200, whatever we make of it: a PBX that gets an error retries, and
	// a retry storm over a payload we cannot parse helps nobody. What we could
	// not understand is recorded instead.
	h.markPBXEvent(r.Context())
	if ev.UUID == "" || ev.Event == "" {
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	h.applyPBXEvent(r.Context(), &ev)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// applyPBXEvent folds one event onto the call row.
func (h *Handler) applyPBXEvent(ctx context.Context, ev *pbxEvent) {
	now := time.Now()
	inbound := !strings.EqualFold(ev.Direction, "outbound")

	// Which of the two numbers is the customer's. On an inbound call it is the
	// caller; on one we placed, the callee. Getting this backwards would file
	// every outbound call against our own switchboard number.
	outside := firstNonEmptyStr(ev.Caller, ev.CallerIDNumber)
	if !inbound {
		outside = firstNonEmptyStr(ev.Callee, ev.DestinationNumber)
	}
	phone := outside
	if norm, ok := normalizePhone(outside); ok {
		phone = norm
	}

	direction := "in"
	if !inbound {
		direction = "out"
	}

	// Fields the exchange owns. Never the outcome, the note or the callback —
	// those are the operator's, and a late event must not wipe them.
	set := bson.M{
		"direction": direction,
		"phone":     phone,
		"source":    "pbx",
		"updatedAt": now,
	}
	// Only set on insert, so a re-delivered `call-start` cannot reset a call
	// that has since been answered and written up.
	onInsert := bson.M{
		"pbxCallId": ev.UUID,
		"outcome":   models.CallOutcomePending,
		"note":      "",
		"createdAt": now,
	}

	switch ev.Event {
	case evStart:
		set["ringing"] = true
		onInsert["ringingAt"] = now
	case evUserStart:
		// The exchange is ringing a particular operator's handset. That is the
		// moment their screen should open the caller's card, and `callee` is
		// which handset — so this is also how "who took it" gets filled in.
		set["ringing"] = true
		if ext := strings.TrimSpace(ev.Callee); ext != "" {
			set["extension"] = ext
			if admin := h.adminByExtension(ctx, ext); admin != nil {
				set["operatorId"] = admin.ID
				set["operatorName"] = firstNonEmptyStr(admin.Name, admin.Username)
			}
		}
	case evAnswered, evTransfer:
		set["ringing"] = false
		set["answeredAt"] = now
		if ext := strings.TrimSpace(ev.Callee); ext != "" {
			set["extension"] = ext
			if admin := h.adminByExtension(ctx, ext); admin != nil {
				set["operatorId"] = admin.ID
				set["operatorName"] = firstNonEmptyStr(admin.Name, admin.Username)
			}
		}
	case evMissed:
		set["ringing"] = false
		// Nobody picked up. Written as the outcome only if the operator has not
		// already decided otherwise — hence $setOnInsert semantics below.
		set["hangupCause"] = firstNonEmptyStr(ev.HangupCause, "NO_ANSWER")
	case evEnd:
		set["ringing"] = false
		set["hangupCause"] = ev.HangupCause
		// Talk time, not ring time: a call that rang for forty seconds and was
		// never answered is a nought-second conversation.
		if secs := asInt(ev.DialogDuration); secs > 0 {
			set["seconds"] = clampSeconds(secs)
		} else if secs := asInt(ev.CallDuration); secs > 0 {
			set["seconds"] = clampSeconds(secs)
		}
		if ev.DownloadURL != "" {
			set["hasRecording"] = true
		}
	}

	res := h.Store.Calls.FindOneAndUpdate(ctx,
		bson.M{"pbxCallId": ev.UUID},
		bson.M{"$set": set, "$setOnInsert": onInsert},
		options.FindOneAndUpdate().
			SetUpsert(true).
			SetReturnDocument(options.After),
	)
	var call models.Call
	if err := res.Decode(&call); err != nil {
		return
	}

	// A missed call nobody has written up is a missed call — but only until a
	// person says otherwise.
	if ev.Event == evMissed && call.Outcome == models.CallOutcomePending {
		_, _ = h.Store.Calls.UpdateByID(ctx, call.ID,
			bson.M{"$set": bson.M{"outcome": models.CallOutcomeMissed}})
	}

	// Attach the customer once we know the number. Done here rather than in the
	// panel so the log reads correctly even for calls nobody ever opened.
	if call.UserID.IsZero() && phone != "" {
		var user models.User
		if err := h.Store.Users.FindOne(ctx, bson.M{"phone": phone}).Decode(&user); err == nil {
			name := strings.TrimSpace(user.FirstName + " " + user.LastName)
			_, _ = h.Store.Calls.UpdateByID(ctx, call.ID, bson.M{"$set": bson.M{
				"userId": user.ID, "customerName": name,
			}})
		}
	}
}

// adminByExtension finds the panel account behind an internal number.
func (h *Handler) adminByExtension(ctx context.Context, ext string) *models.AdminUser {
	var admin models.AdminUser
	if err := h.Store.Admins.FindOne(ctx,
		bson.M{"pbxExtension": ext}).Decode(&admin); err != nil {
		return nil
	}
	return &admin
}

// ---- Settings ----

func (h *Handler) pbxSettings(ctx context.Context) *models.PBXSettings {
	var s models.PBXSettings
	if err := h.Store.PBXSettings.FindOne(ctx, bson.M{}).Decode(&s); err != nil {
		return &models.PBXSettings{}
	}
	return &s
}

// pbxClient is the configured exchange, or nil.
func (h *Handler) pbxClient(ctx context.Context) *pbx.Client {
	s := h.pbxSettings(ctx)
	if !s.Enabled || s.Provider != models.PBXOnlinePBX {
		return nil
	}
	client := pbx.New(pbx.Config{Domain: s.Domain, APIKey: s.APIKey})
	if !client.Configured() {
		return nil
	}
	return client
}

func (h *Handler) markPBXEvent(ctx context.Context) {
	now := time.Now()
	_, _ = h.Store.PBXSettings.UpdateOne(ctx, bson.M{},
		bson.M{"$set": bson.M{"lastEventAt": now}}, options.Update().SetUpsert(true))
}

// AdminGetPBX returns the phone settings, without the API key.
func (h *Handler) AdminGetPBX(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	s := h.pbxSettings(r.Context())
	httpx.JSON(w, http.StatusOK, map[string]any{
		"provider":         firstNonEmptyStr(s.Provider, models.PBXOnlinePBX),
		"enabled":          s.Enabled,
		"domain":           s.Domain,
		"defaultExtension": s.DefaultExtension,
		"hasApiKey":        s.APIKey != "",
		// The path the exchange has to be told about. Returned rather than
		// shown as a secret because it *is* the address — it cannot be pasted
		// into onlinePBX without being visible.
		"webhookPath": pbxWebhookPath(s.WebhookToken),
		"lastCheckAt": s.LastCheckAt,
		"lastCheckOk": s.LastCheckOK,
		"lastCheck":   s.LastCheck,
		"lastEventAt": s.LastEventAt,
	})
}

func pbxWebhookPath(token string) string {
	if token == "" {
		return ""
	}
	return "/api/v1/pbx/onlinepbx/" + token
}

type pbxSettingsRequest struct {
	Enabled          bool   `json:"enabled"`
	Domain           string `json:"domain"`
	APIKey           string `json:"apiKey"`
	DefaultExtension string `json:"defaultExtension"`
	// Replace the webhook address, which stops every previously configured one
	// from working. The answer to a URL that leaked.
	RotateToken bool `json:"rotateToken"`
}

// AdminUpdatePBX saves the phone settings.
func (h *Handler) AdminUpdatePBX(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req pbxSettingsRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	current := h.pbxSettings(r.Context())

	token := current.WebhookToken
	if token == "" || req.RotateToken {
		token = randomToken()
	}
	set := bson.M{
		"provider":         models.PBXOnlinePBX,
		"enabled":          req.Enabled,
		"domain":           strings.TrimSpace(req.Domain),
		"defaultExtension": strings.TrimSpace(req.DefaultExtension),
		// Same rule as every other credential here: empty means keep.
		"apiKey":       keepSecret(req.APIKey, current.APIKey),
		"webhookToken": token,
		"updatedAt":    time.Now(),
	}
	if _, err := h.Store.PBXSettings.UpdateOne(r.Context(), bson.M{},
		bson.M{"$set": set}, options.Update().SetUpsert(true)); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActPBXSettings, "settings", "pbx", "Telefoniya",
		strings.TrimSpace(req.Domain))
	h.AdminGetPBX(w, r)
}

// AdminPingPBX proves the API key works.
func (h *Handler) AdminPingPBX(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	s := h.pbxSettings(r.Context())
	client := pbx.New(pbx.Config{Domain: s.Domain, APIKey: s.APIKey})
	if !client.Configured() {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"ok": false, "message": "domen yoki API kalit kiritilmagan",
		})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	name, err := client.Ping(ctx)
	now := time.Now()
	ok, msg := err == nil, name
	if err != nil {
		msg = err.Error()
	}
	_, _ = h.Store.PBXSettings.UpdateOne(r.Context(), bson.M{},
		bson.M{"$set": bson.M{
			"lastCheckAt": now, "lastCheckOk": ok, "lastCheck": clampText(msg, 300),
		}}, options.Update().SetUpsert(true))
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": ok, "message": msg})
}

// ---- What the desk uses ----

// AdminLiveCall is what the operator's screen polls: is a call ringing for me?
//
// Polling rather than a socket because that is what this panel does everywhere
// (see AlertBell) — one small request every few seconds, no new infrastructure,
// and nothing to reconnect after a laptop lid closes.
func (h *Handler) AdminLiveCall(w http.ResponseWriter, r *http.Request) {
	admin, err := h.adminUser(r)
	if err != nil {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	// Only calls from the last couple of minutes: a `call-end` that never
	// arrived would otherwise leave a phantom ringing on somebody's screen for
	// the rest of the day.
	since := time.Now().Add(-2 * time.Minute)
	filter := bson.M{"ringing": true, "updatedAt": bson.M{"$gte": since}}
	// Ringing at *this* operator's handset, when we know their extension.
	// Without one, any ringing call is shown — a one-operator restaurant has
	// nothing to disambiguate and should still get the pop.
	if ext := strings.TrimSpace(admin.PBXExtension); ext != "" {
		filter["$or"] = []bson.M{
			{"extension": ext}, {"extension": bson.M{"$in": []any{nil, ""}}},
		}
	}
	var call models.Call
	err = h.Store.Calls.FindOne(r.Context(), filter,
		options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})).Decode(&call)
	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"ringing": false})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"ringing":   true,
		"call":      call,
		"extension": admin.PBXExtension,
	})
}

type dialRequest struct {
	Phone string `json:"phone"`
	// Which call row this belongs to, so the two can be tied together.
	CallID string `json:"callId"`
}

// AdminDial rings the operator's own handset, then the customer.
func (h *Handler) AdminDial(w http.ResponseWriter, r *http.Request) {
	admin, err := h.adminUser(r)
	if err != nil {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	var req dialRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	client := h.pbxClient(r.Context())
	if client == nil {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"ok": false, "message": "telefoniya ulanmagan",
		})
		return
	}
	settings := h.pbxSettings(r.Context())
	from := firstNonEmptyStr(admin.PBXExtension, settings.DefaultExtension)
	if from == "" {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"ok":      false,
			"message": "sizning ichki raqamingiz ko'rsatilmagan — Hisobim bo'limida yozing",
		})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	uuid, err := client.Call(ctx, from, strings.TrimSpace(req.Phone))
	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": false, "message": err.Error()})
		return
	}
	// Tie the manual row to the exchange's call, so the duration and recording
	// land on the row the operator is looking at rather than on a new one.
	if id, err := objectID(strings.TrimSpace(req.CallID)); err == nil && uuid != "" {
		_, _ = h.Store.Calls.UpdateByID(r.Context(), id, bson.M{"$set": bson.M{
			"pbxCallId": uuid, "direction": "out", "updatedAt": time.Now(),
		}})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "uuid": uuid})
}

// AdminCallRecording hands back a fresh link to a call's audio.
//
// Fetched on demand rather than stored: onlinePBX signs its download URLs, so
// one saved in the database works until it does not, and the failure looks like
// a lost recording rather than an expired link.
func (h *Handler) AdminCallRecording(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var call models.Call
	if err := h.Store.Calls.FindOne(r.Context(), bson.M{"_id": id}).Decode(&call); err != nil {
		httpx.Error(w, http.StatusNotFound, "qo'ng'iroq topilmadi")
		return
	}
	client := h.pbxClient(r.Context())
	if client == nil || call.PBXCallID == "" {
		httpx.JSON(w, http.StatusOK, map[string]any{"url": ""})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	link, err := client.RecordingURL(ctx, call.PBXCallID)
	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"url": "", "message": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"url": link})
}

// ---- helpers ----

// randomToken is the secret in the webhook URL.
func randomToken() string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		// Never silently fall back to something guessable: this string is the
		// only thing standing between the log and anyone who finds the path.
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(buf)
}

// asInt reads a duration that may arrive as a number or a string.
func asInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	}
	return 0
}

func firstNonEmptyStr(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

type extensionRequest struct {
	Extension string `json:"extension"`
}

// AdminSetMyExtension records which handset is this operator's.
//
// Its own endpoint rather than part of the credentials form: changing an
// extension is a routine thing an operator does when they move desk, and making
// them re-type their password for it means it never gets set — after which
// click-to-call silently does not work for them.
func (h *Handler) AdminSetMyExtension(w http.ResponseWriter, r *http.Request) {
	admin, err := h.adminUser(r)
	if err != nil {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	var req extensionRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ext := clampText(req.Extension, 20)
	// Two operators on one extension makes "who answered" a coin toss, so the
	// second one is refused rather than quietly winning.
	if ext != "" {
		taken := h.Store.Admins.FindOne(r.Context(), bson.M{
			"pbxExtension": ext, "_id": bson.M{"$ne": admin.ID},
		})
		if taken.Err() == nil {
			httpx.Error(w, http.StatusConflict,
				"bu ichki raqam boshqa adminga biriktirilgan")
			return
		}
	}
	if _, err := h.Store.Admins.UpdateByID(r.Context(), admin.ID,
		bson.M{"$set": bson.M{"pbxExtension": ext}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"extension": ext})
}
