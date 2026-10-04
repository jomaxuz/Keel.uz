package handlers

// ---- One sign-in for the Keel app ----
//
// The Keel app is one install for everybody who works in a restaurant: the
// owner, a waiter, a cook, a courier. Each of them still has an account in the
// table they always had — `admin_user`, `staff`, `courier` — and each of those
// tables still has its own login (`/admin/login`, `/staff/login`,
// `/courier/login`), which the four older apps keep using unchanged.
//
// This endpoint asks all three tables the one question the Keel login screen
// can ask: whose username and password is this? It answers with every account
// the pair opens, each with its own token, and the app shows the person the
// parts of itself those accounts are for.
//
// ⚠️ **Asked of the server, not tried three times by the app.** Three login
// calls per attempt would spend the rate limit three times as fast, put a
// failed panel login in the audit log every time a cook signs in, and fall
// through `Login`'s storekeeper path — a side door that answers with a panel
// token to somebody who only wanted their shifts.
//
// ⚠️ **Its own device keys: `keel-owner`, `keel-staff`, `keel-courier`.** The
// older apps bind with `owner`, `waiter`, `team`, `courier`, and the Keel app
// must not collide with them: the same person moving to Keel on the same phone
// would otherwise be refused ("bound to another phone" — the other *app* on
// that very phone). One key per account kind rather than one for the app is
// what lets an owner who also waits tables hold both accounts on one handset,
// and still keeps each account to one phone.

import (
	"net/http"
	"time"

	"restaurant-backend/internal/auth"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/i18n"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type appLoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	deviceClaim
}

// appAccount is one account a username and password opened.
//
// ⚠️ **A refusal travels as an entry, not as the response.** A courier whose
// staff account is fine but whose courier account is bound to an old phone
// should get into the half that works and be told, in words, why the other half
// is missing — not be locked out of both by the one that failed.
type appAccount struct {
	// "admin", "staff" or "courier" — the table, and the token's audience.
	Kind  string `json:"kind"`
	Token string `json:"token,omitempty"`
	Name  string `json:"name"`
	// For "admin": "owner" or "manager".
	Role  string `json:"role,omitempty"`
	Error string `json:"error,omitempty"`

	Admin   *models.AdminUser `json:"admin,omitempty"`
	Staff   *models.Staff     `json:"staff,omitempty"`
	Courier *models.Courier   `json:"courier,omitempty"`
}

// keelAppKey is the device-binding key the Keel app uses for one account kind.
func keelAppKey(kind string) string {
	switch kind {
	case "admin":
		return "keel-owner"
	case "staff":
		return "keel-staff"
	case "courier":
		return "keel-courier"
	}
	return ""
}

const (
	errAppLogin   = "login yoki parol noto'g'ri"
	errStaffOff   = "hisob o'chirilgan — ma'muriyat bilan bog'laning"
	errCourierOff = "hisob o'chirilgan — restoran bilan bog'laning"
)

// AppLogin signs a person into the Keel app with every account their
// credentials open.
func (h *Handler) AppLogin(w http.ResponseWriter, r *http.Request) {
	var req appLoginRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	claim := deviceFrom(r, req.deviceClaim)
	ip := clientIP(r)
	ctx := r.Context()

	lang := httpx.LangOf(w)
	// ⚠️ The refusals inside the list are worded here, in the reader's
	// language: they never pass through httpx.Error, which is where every other
	// message in this server is translated.
	say := func(msg string) string { return i18n.Localize(lang, msg) }
	bind := func(kind string, id primitive.ObjectID) string {
		c := claim
		c.App = keelAppKey(kind)
		if err := h.bindDevice(ctx, kind, id, c, ip); err != nil {
			return say(err.Error())
		}
		return ""
	}

	out := []appAccount{}

	// ---- The panel's account: owner or manager ----
	var admin models.AdminUser
	if h.Store.Admins.FindOne(ctx, bson.M{"username": req.Username}).Decode(&admin) == nil &&
		bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)) == nil {
		a := appAccount{Kind: "admin", Name: firstNonEmpty(admin.Name, admin.Username), Role: admin.Role}
		if msg := bind("admin", admin.ID); msg != "" {
			a.Error = msg
		} else if token, err := auth.Generate(h.Cfg.JWTSecret, admin.ID.Hex(), admin.Role); err == nil {
			now := time.Now()
			_, _ = h.Store.Admins.UpdateByID(ctx, admin.ID, bson.M{"$set": bson.M{"lastLoginAt": now}})
			admin.LastLoginAt = &now
			h.logLogin(r, &admin)
			a.Token, a.Admin = token, &admin
		}
		out = append(out, a)
	}

	// ---- An employee: waiter, cook, cashier, storekeeper ----
	var staff models.Staff
	if h.Store.Staff.FindOne(ctx, bson.M{"username": normalizeUsername(req.Username)}).Decode(&staff) == nil &&
		bcrypt.CompareHashAndPassword([]byte(staff.PasswordHash), []byte(req.Password)) == nil {
		a := appAccount{Kind: "staff", Name: staff.Name}
		switch {
		case !staff.IsActive:
			a.Error = say(errStaffOff)
		default:
			if msg := bind("staff", staff.ID); msg != "" {
				a.Error = msg
			} else if token, err := auth.Generate(h.Cfg.JWTSecret, staff.ID.Hex(), "staff"); err == nil {
				h.withRole(ctx, &staff)
				a.Token, a.Staff = token, &staff
			}
		}
		out = append(out, a)
	}

	// ---- A courier ----
	var courier models.Courier
	if h.Store.Couriers.FindOne(ctx, bson.M{"username": req.Username}).Decode(&courier) == nil &&
		bcrypt.CompareHashAndPassword([]byte(courier.PasswordHash), []byte(req.Password)) == nil {
		a := appAccount{Kind: "courier", Name: courier.Name}
		switch {
		case !courier.IsActive:
			a.Error = say(errCourierOff)
		default:
			if msg := bind("courier", courier.ID); msg != "" {
				a.Error = msg
			} else if token, err := auth.Generate(h.Cfg.JWTSecret, courier.ID.Hex(), "courier"); err == nil {
				a.Token, a.Courier = token, &courier
			}
		}
		out = append(out, a)
	}

	if len(out) == 0 {
		httpx.Error(w, http.StatusUnauthorized, errAppLogin)
		return
	}
	// ⚠️ Every account refused: the first refusal is the answer, with the
	// status its own endpoint would have used, so the screen can say it in the
	// person's language.
	opened := 0
	for _, a := range out {
		if a.Token != "" {
			opened++
		}
	}
	if opened == 0 {
		// Already in the reader's language; httpx translates by exact match
		// and leaves a translated sentence as it is.
		httpx.Error(w, http.StatusConflict, out[0].Error)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"accounts": out})
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}
