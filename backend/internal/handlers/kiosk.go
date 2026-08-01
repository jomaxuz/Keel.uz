package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"restaurant-backend/internal/auth"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// The branch kiosk: a rotating clock-in code shown on a screen at the branch.
//
// Why it rotates. A printed QR code is a password written on a wall: photograph
// it once and you can clock in from your kitchen at home for the next year. So
// the code is derived from the branch secret and the current 30-second step —
// a photograph is worthless a minute later.
//
// Why it is not enough on its own. Somebody could still scan and forward the
// code to a colleague *within* that minute. That is why the geofence is not
// replaced by this but added to: the colleague would also have to be standing
// at the restaurant. Each guard covers the other's blind spot — the code
// answers "were you here?", the position answers "are you here now?".
//
// Nothing is stored per code: the server recomputes and compares. No table of
// live codes to expire, no cleanup, and a restart invalidates nothing.

const (
	// How long one code lives. Long enough to walk from the screen to the app,
	// short enough that a photograph is useless.
	kioskStepSeconds = 30
	// Steps either side of "now" that still verify: covers a slow scan and a
	// phone whose clock is a little off.
	kioskStepSlack = 1
	// The screen at the branch is signed in for a year — nobody wants to log a
	// wall-mounted tablet back in every week. Revocation is by version bump,
	// not by expiry.
	kioskTokenTTL = 365 * 24 * time.Hour
)

// kioskCodeFor derives the code for one branch at one time step.
func kioskCodeFor(secret string, branchID primitive.ObjectID, step int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%s.%d", branchID.Hex(), step)
	// Half the digest is 128 bits — far beyond guessable in a 60-second window,
	// and short enough to keep the QR sparse and easy to scan.
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:16])
	return fmt.Sprintf("%s.%d.%s", branchID.Hex(), step, sig)
}

func kioskStepNow() int64 { return time.Now().Unix() / kioskStepSeconds }

// codeFingerprint identifies a code without storing it. Used to make a code
// single-use per employee (see models.Shift.InCode).
func codeFingerprint(code string) string {
	if strings.TrimSpace(code) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(code)))
	return hex.EncodeToString(sum[:12])
}

// verifyKioskCode reports whether a code is a live one for this branch.
func verifyKioskCode(branch *models.Branch, code string) bool {
	if branch.KioskSecret == "" || strings.TrimSpace(code) == "" {
		return false
	}
	parts := strings.Split(strings.TrimSpace(code), ".")
	if len(parts) != 3 {
		return false
	}
	// The code names its own branch: a code from the Chilonzor screen must not
	// open a shift at Yunusobod.
	if !strings.EqualFold(parts[0], branch.ID.Hex()) {
		return false
	}
	step, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false
	}
	now := kioskStepNow()
	if step < now-kioskStepSlack || step > now+kioskStepSlack {
		return false
	}
	want := kioskCodeFor(branch.KioskSecret, branch.ID, step)
	// Constant time: a byte-by-byte comparison leaks how much of a guess was
	// right, and this runs on an endpoint anybody can call.
	return subtle.ConstantTimeCompare([]byte(want), []byte(strings.TrimSpace(code))) == 1
}

// ensureKioskSecret gives a branch a secret the first time one is needed.
func (h *Handler) ensureKioskSecret(r *http.Request, branch *models.Branch) (string, error) {
	if branch.KioskSecret != "" {
		return branch.KioskSecret, nil
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	secret := hex.EncodeToString(buf)
	if _, err := h.Store.Branches.UpdateByID(r.Context(), branch.ID,
		bson.M{"$set": bson.M{"kioskSecret": secret, "updatedAt": time.Now()}}); err != nil {
		return "", err
	}
	branch.KioskSecret = secret
	return secret, nil
}

// ---- Admin: handing the screen its token ----

// AdminKioskToken issues (or re-issues) the long-lived token for a branch's
// kiosk screen and returns the URL to open on it.
//
// `?rotate=1` revokes every token issued before: the version bump kills the old
// ones and a fresh secret kills every code they might still be showing. That is
// the answer to a tablet walking out of the building.
func (h *Handler) AdminKioskToken(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.requireBranchAccess(r, id); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	branch, err := h.branchByID(r, id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "filial topilmadi")
		return
	}

	if r.URL.Query().Get("rotate") == "1" {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		branch.KioskSecret = hex.EncodeToString(buf)
		branch.KioskVersion++
		if _, err := h.Store.Branches.UpdateByID(r.Context(), id, bson.M{"$set": bson.M{
			"kioskSecret":  branch.KioskSecret,
			"kioskVersion": branch.KioskVersion,
			"updatedAt":    time.Now(),
		}}); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		h.logAction(r, ActKioskRotate, "branch", id.Hex(), branch.Name, "kiosk kaliti almashtirildi")
	} else if _, err := h.ensureKioskSecret(r, branch); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	token, err := auth.GenerateLong(h.Cfg.JWTSecret, id.Hex(), "kiosk", branch.KioskVersion, kioskTokenTTL)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"token":      token,
		"branchId":   id.Hex(),
		"branchName": branch.Name,
		"version":    branch.KioskVersion,
	})
}

// ---- The screen ----

// kioskBranch loads the branch behind a kiosk token, refusing tokens issued
// before the last revocation.
func (h *Handler) kioskBranch(r *http.Request) (*models.Branch, error) {
	claims := middleware.ClaimsFrom(r.Context())
	if claims == nil {
		return nil, errKioskToken
	}
	id, err := objectID(claims.UserID)
	if err != nil {
		return nil, errKioskToken
	}
	branch, err := h.branchByID(r, id)
	if err != nil {
		return nil, errKioskToken
	}
	if claims.Ver != branch.KioskVersion {
		return nil, errKioskRevoked
	}
	return branch, nil
}

// KioskCode is what the screen polls: the code to show and how long it lasts.
func (h *Handler) KioskCode(w http.ResponseWriter, r *http.Request) {
	branch, err := h.kioskBranch(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	if branch.KioskSecret == "" {
		httpx.Error(w, http.StatusConflict, "kiosk sozlanmagan")
		return
	}
	step := kioskStepNow()
	// Seconds left on the code being handed out, so the screen can draw a
	// countdown and refresh exactly when it expires rather than guessing.
	expiresIn := kioskStepSeconds - (time.Now().Unix() % kioskStepSeconds)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"code":       kioskCodeFor(branch.KioskSecret, branch.ID, step),
		"expiresIn":  expiresIn,
		"stepSecs":   kioskStepSeconds,
		"branchName": branch.Name,
		"branchId":   branch.ID.Hex(),
	})
}
