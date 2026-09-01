package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"
)

// ---- Binding an account to the phone it signs in from ----
//
// ⚠️ **The password stopped being the identity the moment there were four
// apps.** A staff login is written on a card in the office, a courier hands
// theirs to a friend covering a shift, and a waiter who wants an evening off
// lends a colleague their username — and every one of those sign-ins is
// *correct*, so nothing on the server can object. What the server can see is
// that the phone changed.
//
// Two rules, both enforced here and both backed by a unique index:
//
//	one account, one install  — a second phone is refused
//	one install, one account  — a second account on that phone is refused
//
// ⚠️ **Per app.** The same person is a waiter in Keel Waiter and an employee in
// Keel Team, and an owner may carry Owner and Waiter on one handset; binding
// across apps would refuse the ordinary case rather than the abuse.
//
// ⚠️ **The browser is not bound, and that is not an oversight.** The panel is
// opened from a laptop at home, a machine at the restaurant and a phone
// browser, all legitimately — the lock is about the four apps, which say who
// they are by sending a device id. A request with no device id changes nothing
// here, which is exactly what every existing caller does.

// deviceClaim is what an app sends with its login.
type deviceClaim struct {
	DeviceID string `json:"deviceId"`
	App      string `json:"app"`
	Platform string `json:"platform"`
	Name     string `json:"name"`
}

// deviceFrom reads the claim off a request, from the body or the headers.
//
// ⚠️ **Headers as well as the body**, because the binding has to be refreshed
// on requests that carry no body of their own — `me`, the one call every app
// makes on every launch. One shape, read from two places, rather than a second
// concept.
func deviceFrom(r *http.Request, body deviceClaim) deviceClaim {
	out := body
	if out.DeviceID == "" {
		out.DeviceID = r.Header.Get("X-Keel-Device")
	}
	if out.App == "" {
		out.App = r.Header.Get("X-Keel-App")
	}
	if out.Platform == "" {
		out.Platform = r.Header.Get("X-Keel-Platform")
	}
	if out.Name == "" {
		out.Name = r.Header.Get("X-Keel-Device-Name")
	}
	out.DeviceID = clampText(strings.TrimSpace(out.DeviceID), 120)
	out.App = clampText(strings.TrimSpace(out.App), 20)
	out.Platform = clampText(strings.TrimSpace(out.Platform), 20)
	out.Name = clampText(strings.TrimSpace(out.Name), 60)
	return out
}

// errDeviceTaken and errAccountBound are the two refusals, kept apart because
// they send the reader to two different people: one to whoever else uses this
// phone, the other to the office that can release the binding.
var (
	errDeviceTaken  = errors.New("bu telefonda boshqa hisob ishlatilyapti — administratorga murojaat qiling")
	errAccountBound = errors.New("hisobingiz boshqa telefonga biriktirilgan — administratordan uni o'chirishni so'rang")
)

// bindDevice ties an account to this install, or explains why it cannot.
//
// ⚠️ **Called after the password is checked and before the token is issued.**
// Before the password it would tell an outsider which phones a restaurant uses;
// after the token it would hand out a session the next request refuses.
func (h *Handler) bindDevice(
	ctx context.Context, kind string, subject primitive.ObjectID, d deviceClaim, ip string,
) error {
	if d.DeviceID == "" || d.App == "" {
		// A browser, or a build older than this feature. Nothing is bound and
		// nothing is refused — see the header note.
		return nil
	}
	now := time.Now()

	// Is this install already somebody else's?
	var onDevice models.LoginDevice
	err := h.Store.LoginDevices.FindOne(ctx,
		bson.M{"app": d.App, "deviceId": d.DeviceID}).Decode(&onDevice)
	if err == nil && onDevice.SubjectID != subject {
		return errDeviceTaken
	}
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		// ⚠️ **A database that will not answer lets the sign-in through.** The
		// harm this prevents is two people sharing a login; the harm it can
		// cause is a restaurant that cannot open. Same direction as the till's
		// shift gate, and for the same reason.
		return nil
	}

	// Is this account already on another install?
	var onAccount models.LoginDevice
	err = h.Store.LoginDevices.FindOne(ctx,
		bson.M{"kind": kind, "subjectId": subject, "app": d.App}).Decode(&onAccount)
	if err == nil && onAccount.DeviceID != d.DeviceID {
		return errAccountBound
	}
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return nil
	}

	// ⚠️ Upsert on the account, not on the device: the same phone reinstalled
	// mints a new id, and the row that has to move is this person's.
	_, werr := h.Store.LoginDevices.UpdateOne(ctx,
		bson.M{"kind": kind, "subjectId": subject, "app": d.App},
		bson.M{
			"$set": bson.M{
				"deviceId": d.DeviceID, "platform": d.Platform, "name": d.Name,
				"ip": ip, "lastSeenAt": now,
			},
			"$setOnInsert": bson.M{"createdAt": now},
		},
		options.Update().SetUpsert(true))
	if werr != nil {
		// A duplicate key here is the race the indexes exist for: two sign-ins
		// seconds apart, and the loser is told the honest thing.
		if mongo.IsDuplicateKeyError(werr) {
			return errDeviceTaken
		}
		return nil
	}
	return nil
}

// touchDevice records that this account was seen from this install, without
// deciding anything.
//
// ⚠️ **Never refuses.** It runs on `me`, which every app calls on every launch;
// making it a second gate would mean a phone that lost its binding mid-shift
// stops working with no message anybody can read. The gate is the sign-in.
func (h *Handler) touchDevice(
	ctx context.Context, kind string, subject primitive.ObjectID, d deviceClaim, ip string,
) {
	if d.DeviceID == "" || d.App == "" {
		return
	}
	_, _ = h.Store.LoginDevices.UpdateOne(ctx,
		bson.M{"kind": kind, "subjectId": subject, "app": d.App, "deviceId": d.DeviceID},
		bson.M{"$set": bson.M{"ip": ip, "lastSeenAt": time.Now()}})
}

// devicesOf lists what an account is bound to, for the panel.
func (h *Handler) devicesOf(
	ctx context.Context, kind string, subject primitive.ObjectID,
) []models.LoginDevice {
	out := []models.LoginDevice{}
	cur, err := h.Store.LoginDevices.Find(ctx,
		bson.M{"kind": kind, "subjectId": subject})
	if err != nil {
		return out
	}
	_ = cur.All(ctx, &out)
	return out
}

// AdminListDevices answers "which phones is this account signed in on".
//
// ⚠️ Scoped by the same rules as everything else in the panel: a manager may
// look at their own branch's people and no others, which `subjectInScope`
// below decides per kind.
func (h *Handler) AdminListDevices(w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "kind")
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if msg := h.subjectInScope(r, kind, id); msg != "" {
		httpx.Error(w, http.StatusNotFound, msg)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"devices": h.devicesOf(r.Context(), kind, id),
	})
}

// AdminDeleteDevice releases a binding.
//
// ⚠️ **The button this whole feature depends on.** A reinstall mints a new id,
// a lost phone never comes back and a screen breaks on a Friday night — so the
// lock has to be liftable by somebody who is already in the building. Logged,
// because releasing one is also how somebody would quietly hand a login to a
// second person.
func (h *Handler) AdminDeleteDevice(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "deviceId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var d models.LoginDevice
	if err := h.Store.LoginDevices.FindOne(r.Context(), bson.M{"_id": id}).Decode(&d); err != nil {
		httpx.Error(w, http.StatusNotFound, "qurilma topilmadi")
		return
	}
	if msg := h.subjectInScope(r, d.Kind, d.SubjectID); msg != "" {
		httpx.Error(w, http.StatusNotFound, msg)
		return
	}
	if _, err := h.Store.LoginDevices.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAction(r, ActStaffUpdate, "device", id.Hex(), d.Name,
		d.App+" · "+d.DeviceID)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// subjectInScope answers whether this admin may see that person's devices, and
// returns the refusal when they may not.
//
// ⚠️ **404 rather than 403**, the rule the rest of the panel follows: a manager
// must not learn that another branch's courier exists by asking about their
// phones.
func (h *Handler) subjectInScope(r *http.Request, kind string, id primitive.ObjectID) string {
	switch kind {
	case "courier":
		if _, err := h.courierInScope(r, id); err != nil {
			return "kuryer topilmadi"
		}
	case "staff":
		s, err := h.staffByID(r, id)
		if err != nil {
			return "ishchi topilmadi"
		}
		if err := h.requireBranchAccess(r, s.BranchID); err != nil {
			return "ishchi topilmadi"
		}
	case "admin":
		// ⚠️ Panel accounts belong to the company rather than to a branch, so
		// the boundary is the role: only an owner may look at, or release,
		// another panel account's phone.
		if err := h.requireOwner(r); err != nil {
			return "bu amal faqat egasi uchun"
		}
	default:
		return "noma'lum turdagi hisob"
	}
	return ""
}
