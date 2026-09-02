package handlers

// ---- Pairing a television to a branch ----
//
// The screen shows a short code; somebody types it into the panel and picks the
// branch. That direction is not a preference: a television is driven with a
// remote control, and typing a password on a D-pad — one letter at a time,
// across an on-screen keyboard, in front of a dining room — is a setup nobody
// finishes. Reading six characters aloud is.
//
// ⚠️ **The kiosk's derived code does not work here.** A kiosk screen already
// holds a branch token, so its code can be an HMAC over (branch, time step) and
// nothing has to be stored. A television being paired has no identity at all —
// that is what pairing is for — so the code cannot be derived from anything and
// the server has to remember what it handed out.
//
// ⚠️ **The code is on a wall in a public room.** So it is short-lived, replaced
// every few seconds, single-use, rate-limited, and it is *not* what the
// television polls with: knowing the code must not be enough to collect the
// token. That is what PollSecret is for.

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/auth"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// RoleTV is what a paired television's token says it is. Its own role, so
	// every `owner, manager` group and every requireOwner in the router refuses
	// it without being edited — the same argument as RoleStock.
	RoleTV = "tv"

	// How long one pairing code lives. The app asks for a new one every ten
	// seconds; this is the window that covers a manager walking from the screen
	// to a laptop, and it is short enough that a photograph is worthless.
	tvCodeTTL = 90 * time.Second

	// ⚠️ **A year, and revoked by version rather than by expiry.** Nobody signs
	// a wall-mounted television back in every week — and the moment it would
	// ask is a Friday evening with a full room. Same shape as the kiosk token.
	tvTokenTTL = 365 * 24 * time.Hour

	// The alphabet a code is drawn from: no O/0, no I/1, no U/V confusion.
	// ⚠️ Read across a room and typed by somebody else — every character that
	// can be misread costs a support call, and the entropy lost is nothing
	// beside a 90-second life and a rate limit.
	tvCodeAlphabet = "ABCDEFGHJKLMNPQRSTWXYZ23456789"
	tvCodeLength   = 6
)

// newTVCode draws a fresh pairing code.
func newTVCode() (string, error) {
	out := make([]byte, tvCodeLength)
	max := big.NewInt(int64(len(tvCodeAlphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = tvCodeAlphabet[n.Int64()]
	}
	return string(out), nil
}

func newTVSecret() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// normalizeTVCode is how a typed code is read: case and spacing are the
// person's, not the code's.
//
// ⚠️ Dashes and spaces are dropped because the screen draws the code grouped
// ("K7P — 4RM") and somebody will type it the way they read it.
func normalizeTVCode(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	return strings.NewReplacer(" ", "", "-", "", "—", "").Replace(s)
}

type tvPairStartRequest struct {
	// The television's own id, generated on first launch and kept on the set.
	InstallID string `json:"installId"`
	// What the app is, for the panel's "this screen is running an old build".
	AppVersion string `json:"appVersion"`
}

// TVPairStart hands a television a code to show.
//
// ⚠️ **One live pairing per television**, replaced on every call: the app asks
// again every ten seconds, and leaving the old rows alive would mean a dozen
// working codes for one screen — including the one somebody photographed.
func (h *Handler) TVPairStart(w http.ResponseWriter, r *http.Request) {
	var req tvPairStartRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	install := clampText(req.InstallID, 64)
	if install == "" {
		httpx.Error(w, http.StatusBadRequest, "qurilma aniqlanmadi")
		return
	}

	code, err := newTVCode()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	secret, err := newTVSecret()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	now := time.Now()
	expires := now.Add(tvCodeTTL)
	// Replace this television's pairing rather than adding one. The unique
	// index on installId is what makes that the only possible outcome even if
	// two requests arrive together — a television with a flaky connection
	// retries, and two live codes for one screen is exactly the state this
	// endpoint must not produce.
	if _, err := h.Store.TVPairings.UpdateOne(r.Context(),
		bson.M{"installId": install},
		bson.M{"$set": bson.M{
			"code":       code,
			"pollSecret": secret,
			"expiresAt":  expires,
			"createdAt":  now,
			// ⚠️ Cleared, not left: a television that was claimed and then
			// asks for a new code is being paired again, and a stale token
			// sitting in this row would be handed to whoever asks next.
			"screenId": primitive.NilObjectID,
			"token":    "",
		}},
		options.Update().SetUpsert(true),
	); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"code":       code,
		"pollSecret": secret,
		"expiresAt":  expires,
		// So the screen can draw a countdown without trusting its own clock,
		// which on a cheap television is routinely months out.
		"expiresIn": int(tvCodeTTL.Seconds()),
	})
}

// TVPairStatus is what the television polls: "has somebody claimed me yet?"
//
// ⚠️ **Answered by the poll secret, never by the code.** The code is readable
// by every guest in the room; if it were enough to ask this question, anybody
// with a phone could collect the token meant for the wall by polling faster
// than the app does.
func (h *Handler) TVPairStatus(w http.ResponseWriter, r *http.Request) {
	install := clampText(r.URL.Query().Get("installId"), 64)
	secret := clampText(r.URL.Query().Get("pollSecret"), 64)
	if install == "" || secret == "" {
		httpx.Error(w, http.StatusBadRequest, "qurilma aniqlanmadi")
		return
	}
	var p models.TVPairing
	if err := h.Store.TVPairings.FindOne(r.Context(),
		bson.M{"installId": install}).Decode(&p); err != nil {
		// No row: the code expired, or this television has never asked. Not an
		// error — the app's answer to both is the same, ask for a new code.
		httpx.JSON(w, http.StatusOK, map[string]any{"status": "expired"})
		return
	}
	if subtle.ConstantTimeCompare([]byte(p.PollSecret), []byte(secret)) != 1 {
		// ⚠️ Constant time, and the same answer as an expired code: this
		// endpoint is open to anybody, and a different reply for "wrong secret"
		// tells a guessing client it is on the right television.
		httpx.JSON(w, http.StatusOK, map[string]any{"status": "expired"})
		return
	}
	if p.Token == "" {
		httpx.JSON(w, http.StatusOK, map[string]any{"status": "pending"})
		return
	}

	// Claimed. The token is handed over **once**: the row goes now, so a second
	// caller — a retry that arrives late, or anybody who learned the secret —
	// gets "expired" rather than a working television token.
	_, _ = h.Store.TVPairings.DeleteOne(r.Context(), bson.M{"_id": p.ID})

	screen, branch, err := h.tvScreenAndBranch(r, p.ScreenID)
	if err != nil {
		// The screen was removed between the claim and this poll. Rare, and the
		// honest answer is the one that sends the television back to the code.
		httpx.JSON(w, http.StatusOK, map[string]any{"status": "expired"})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"status": "paired",
		"token":  p.Token,
		"screen": tvScreenView(screen, branch),
	})
}

// tvScreenAndBranch loads a screen with the branch it hangs in.
func (h *Handler) tvScreenAndBranch(r *http.Request, id primitive.ObjectID) (models.TVScreen, models.Branch, error) {
	var screen models.TVScreen
	if err := h.Store.TVScreens.FindOne(r.Context(),
		bson.M{"_id": id}).Decode(&screen); err != nil {
		return screen, models.Branch{}, err
	}
	var branch models.Branch
	if err := h.Store.Branches.FindOne(r.Context(),
		bson.M{"_id": screen.BranchID}).Decode(&branch); err != nil {
		return screen, models.Branch{}, err
	}
	return screen, branch, nil
}

// tvScreenView is what a television is told about itself.
//
// ⚠️ Deliberately narrow — a name, a branch and a mode. This is drawn on a
// screen in a public room and held by a device we do not control; the branch
// document behind it carries addresses, keys and delivery settings.
func tvScreenView(s models.TVScreen, b models.Branch) map[string]any {
	return map[string]any{
		"id":         s.ID.Hex(),
		"name":       s.Name,
		"mode":       s.Mode,
		"branchId":   s.BranchID.Hex(),
		"branchName": b.Name,
	}
}

// ---- The paired television ----

var (
	errTVToken   = errors.New("ekran tokeni yaroqsiz")
	errTVRevoked = errors.New("bu ekran uzilgan")
)

// tvScreen loads the screen behind a television's token, refusing one that has
// been unpaired or whose branch has been rotated.
//
// ⚠️ **Checked on every request, not only when the token is issued.** A token
// lives for a year; "unpair this screen" has to mean the next request fails,
// not the next year. Two ways it can die, and both matter: the row is gone (one
// screen removed), or the branch's version has moved past it (every screen in
// the branch revoked at once).
func (h *Handler) tvScreen(r *http.Request) (models.TVScreen, models.Branch, error) {
	claims := middleware.ClaimsFrom(r.Context())
	if claims == nil || claims.Role != RoleTV {
		return models.TVScreen{}, models.Branch{}, errTVToken
	}
	id, err := objectID(claims.UserID)
	if err != nil {
		return models.TVScreen{}, models.Branch{}, errTVToken
	}
	screen, branch, err := h.tvScreenAndBranch(r, id)
	if err != nil {
		return models.TVScreen{}, models.Branch{}, errTVRevoked
	}
	if claims.Ver != screen.Version || screen.Version != branch.TVVersion {
		return models.TVScreen{}, models.Branch{}, errTVRevoked
	}
	return screen, branch, nil
}

// TVMe is the television's heartbeat: who am I, and am I still allowed here.
//
// ⚠️ **The panel's "is this screen alive" answer is built here**, from a fact
// with a time on it rather than a stored flag. A paired row and a television
// that was unplugged three weeks ago look identical otherwise — the lesson
// `provisionStatus` taught the console, applied before it can be repeated.
func (h *Handler) TVMe(w http.ResponseWriter, r *http.Request) {
	screen, branch, err := h.tvScreen(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	now := time.Now()
	set := bson.M{"lastSeenAt": now}
	if v := clampText(r.URL.Query().Get("appVersion"), 32); v != "" {
		set["appVersion"] = v
	}
	if _, err := h.Store.TVScreens.UpdateByID(r.Context(),
		screen.ID, bson.M{"$set": set}); err != nil {
		// ⚠️ Not fatal. A television that cannot record its heartbeat must
		// still be told it may keep playing; the alternative is a dining room
		// going dark because a write failed.
		_ = err
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"screen": tvScreenView(screen, branch),
		// The screen's own clock is not to be trusted — cheap sets ship with
		// the wrong year — and every schedule this app follows is the
		// restaurant's local time. So the server says what time it is.
		"serverTime": now,
	})
}

// issueTVToken mints the long-lived token for one screen.
func (h *Handler) issueTVToken(screen models.TVScreen) (string, error) {
	return auth.GenerateLong(h.Cfg.JWTSecret, screen.ID.Hex(), RoleTV,
		screen.Version, tvTokenTTL)
}

// tvPairingByCode finds a live pairing by the code somebody typed.
//
// ⚠️ The expiry is in the filter as well as on a TTL index: Mongo's TTL sweeper
// runs about once a minute, so a code can outlive its own row by a minute — and
// a code that works after it expired is the one thing this design promises it
// will not do.
func (h *Handler) tvPairingByCode(r *http.Request, code string) (models.TVPairing, error) {
	var p models.TVPairing
	err := h.Store.TVPairings.FindOne(r.Context(), bson.M{
		"code":      code,
		"expiresAt": bson.M{"$gt": time.Now()},
	}).Decode(&p)
	if err == mongo.ErrNoDocuments {
		return p, errTVCodeUnknown
	}
	return p, err
}

var errTVCodeUnknown = errors.New("kod topilmadi yoki eskirgan — ekrandagi yangi kodni kiriting")
