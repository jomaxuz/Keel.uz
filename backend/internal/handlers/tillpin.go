package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"restaurant-backend/internal/auth"
	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"github.com/go-chi/chi/v5"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// Who is standing at the till.
//
// # The problem this fixes
//
// ⚠️ A till is a **shared screen**. One account is signed in on the monoblock
// and stays signed in all evening, because nobody types a username and password
// between two guests. Which means every void, every discount and every closed
// check was recorded against whoever happened to unlock the screen at six —
// and those records exist for exactly one purpose, which is to answer "who did
// this". The mechanism was there and the login model was quietly defeating it.
//
// # The model: two facts, neither sufficient alone
//
//   - The **device** proves where it is. It holds a branch-scoped staff token,
//     obtained once with a real username and password when the monoblock was
//     set up.
//   - The **PIN** proves who is standing there. Four digits, tapped in seconds.
//
// ⚠️ **A PIN is not a password and is never treated as one.** It is guessable
// by hand; what makes it safe is that it is only accepted together with a token
// that already belongs to that branch's till. Possession plus knowledge — the
// same trade as a bank card.
//
// So the token issued here carries the role **"till"**, not "staff": it can run
// the floor and the till, and it cannot read that person's payroll, punch their
// clock or open the kitchen screen. Four digits must not reach any of those.

// tillSessionTTL is how long an unlocked till session lives.
//
// Long enough to outlast a shift so a cashier is not locked out mid-service by
// an expiry, short enough that a token copied off a machine is worthless the
// next day. The real protection against "one unlock covers the whole evening"
// is the screen's own idle lock, which is a client concern.
const tillSessionTTL = 14 * time.Hour

// ---- Rate limiting ----
//
// ⚠️ **Per branch, in memory, and deliberately not the shared rate limiter.**
// The IP gate in middleware guards the server; this guards a four-digit secret,
// and it has to count attempts against *the till* rather than against whoever
// is asking. A restaurant behind one NAT would otherwise lock its own cashiers
// out because a different branch was being probed.

const (
	pinMaxAttempts = 5
	pinLockout     = 60 * time.Second
)

type pinAttempts struct {
	mu    sync.Mutex
	count map[string]*pinCounter
}

type pinCounter struct {
	n      int
	locked time.Time
}

var pinGate = pinAttempts{count: map[string]*pinCounter{}}

// blocked reports whether this till is in its cooling-off period.
func (p *pinAttempts) blocked(key string) (bool, time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	c := p.count[key]
	if c == nil || c.locked.IsZero() {
		return false, 0
	}
	if left := time.Until(c.locked); left > 0 {
		return true, left
	}
	// The lockout expired. Reset rather than merely unblocking, so the next
	// wrong PIN starts a fresh count instead of locking again immediately.
	delete(p.count, key)
	return false, 0
}

func (p *pinAttempts) fail(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	c := p.count[key]
	if c == nil {
		c = &pinCounter{}
		p.count[key] = c
	}
	c.n++
	if c.n >= pinMaxAttempts {
		c.locked = time.Now().Add(pinLockout)
		c.n = 0
	}
}

func (p *pinAttempts) ok(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.count, key)
}

// ---- Unlocking ----

type tillUnlockRequest struct {
	PIN string `json:"pin"`
}

// StaffTillUnlock hands the till screen over to whoever tapped their PIN.
//
// ⚠️ The caller must already hold a till-capable token for the branch — this
// endpoint sits inside that middleware group. It does not authenticate a
// stranger; it names the person using an already-authenticated screen.
func (h *Handler) StaffTillUnlock(w http.ResponseWriter, r *http.Request) {
	branchID, ok := h.tillBranch(w, r)
	if !ok {
		return
	}
	var req tillUnlockRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	pin := strings.TrimSpace(req.PIN)
	if err := pinShape(pin); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	key := branchID.Hex()
	if locked, left := pinGate.blocked(key); locked {
		// ⚠️ 429 and the seconds remaining, rather than a bare refusal: a
		// cashier who cannot tell "wrong code" from "locked out" retypes the
		// same PIN and extends the lockout.
		httpx.Error(w, http.StatusTooManyRequests,
			"juda ko'p urinish — "+strconv.Itoa(int(left.Seconds()+1))+" soniyadan keyin qayta urining")
		return
	}

	person, found, err := h.staffByPIN(r.Context(), branchID, pin)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !found {
		pinGate.fail(key)
		// ⚠️ **Never says whose PIN it is or was close.** The screen is in a
		// public room and the message is read by whoever is standing there.
		httpx.Error(w, http.StatusUnauthorized, "PIN noto'g'ri")
		return
	}
	pinGate.ok(key)

	// ⚠️ **Not auth.Generate**, which issues seven days. Four digits must not
	// buy a week: the ordinary staff token is bought with a username and a
	// password, this one with a code tapped in front of the room.
	token, err := auth.GenerateLong(
		h.Cfg.JWTSecret, person.ID.Hex(), "till", 0, tillSessionTTL)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"token": token,
		"staff": tillPerson(person),
	})
}

// tillPerson is the narrow shape the lock screen shows.
//
// ⚠️ Written out by hand rather than returning models.Staff. That document
// carries a salary, a roster, a phone number and a password hash, and this
// response is rendered on a screen in a public room — the rule that has caught
// real leaks in this codebase (see publicReview).
type tillPersonView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Position   string `json:"position,omitempty"`
	CanWaiter  bool   `json:"canWaiter"`
	CanCashier bool   `json:"canCashier"`
}

func tillPerson(s models.Staff) tillPersonView {
	return tillPersonView{
		ID:         s.ID.Hex(),
		Name:       s.Name,
		Position:   s.Position,
		CanWaiter:  s.Can(models.PermWaiter),
		CanCashier: s.Can(models.PermCashier),
	}
}

// staffByPIN finds the employee whose PIN this is.
//
// ⚠️ **The candidate set is deliberately narrow**: this branch, still employed,
// allowed on the till, and has a PIN at all. That is a handful of people, which
// is what makes comparing them one by one affordable — bcrypt costs ~60ms a
// time on purpose, and scanning every employee in the company would turn a tap
// into a several-second wait.
//
// ⚠️ It also closes a real hole. Without the branch bound, a PIN that happens
// to match somebody in another branch would sign this restaurant's voids with
// a stranger's name.
func (h *Handler) staffByPIN(
	ctx context.Context, branchID primitive.ObjectID, pin string,
) (models.Staff, bool, error) {
	cur, err := h.Store.Staff.Find(ctx, bson.M{
		"branchId": branchID,
		"isActive": true,
		"pinHash":  bson.M{"$nin": bson.A{"", nil}},
		"$or": []bson.M{
			{"canWaiter": true},
			{"canCashier": true},
		},
	})
	if err != nil {
		return models.Staff{}, false, err
	}
	var rows []models.Staff
	if err := cur.All(ctx, &rows); err != nil {
		return models.Staff{}, false, err
	}
	for i := range rows {
		if bcrypt.CompareHashAndPassword([]byte(rows[i].PinHash), []byte(pin)) == nil {
			return rows[i], true, nil
		}
	}
	return models.Staff{}, false, nil
}

// ---- Setting a PIN ----

// pinShape rejects codes that cannot do the job.
//
// A pure function so the panel and the till cannot drift on what a PIN is, and
// so the rules can be tested without a database.
func pinShape(pin string) error {
	if pin == "" {
		return errors.New("PIN kiritilmagan")
	}
	if len(pin) != pinDigits {
		return errors.New("PIN 4 ta raqamdan iborat bo'lishi kerak")
	}
	for _, c := range pin {
		if c < '0' || c > '9' {
			return errors.New("PIN faqat raqamlardan iborat bo'lishi kerak")
		}
	}
	// ⚠️ **Obvious codes are allowed, deliberately.** An earlier version refused
	// 1234 and 0000, and that was the wrong trade for this screen: the PIN is
	// tapped dozens of times a shift by somebody holding plates, and a code
	// they cannot remember becomes a code written on a sticky note beside the
	// monoblock — which is worse than a guessable one, because it is readable
	// by anybody in the room and never changes.
	//
	// What actually protects this is not the code's cleverness: four digits are
	// only accepted from a device that already holds this branch's token, five
	// wrong tries cost a minute, and the PIN reaches the floor and the drawer
	// and nothing else. Difficulty here would buy nothing and cost adoption.
	return nil
}

// pinDigits is how long a till code is.
//
// ⚠️ **Exactly four, not a range.** A variable length means the pad cannot show
// how many digits are expected — it either draws too many empty dots or cannot
// submit on its own — and both cost a tap on the busiest screen in the
// building. Every till converges on four for the same reason.
const pinDigits = 4

type setPinRequest struct {
	// Empty removes the PIN, which is how somebody is taken off the till
	// without touching their account.
	PIN string `json:"pin"`
}

// AdminSetStaffPin sets or clears an employee's till code.
//
// ⚠️ **Its own endpoint, not a field on the staff form** — the same rule as
// branch.soldOut and kioskSecret, and for the same reason: a form that does not
// show the PIN would send it empty on every save, and saving an unrelated
// change (a phone number, a rota) would quietly lock that person out of the
// till. Here an empty value means "remove it" because that is the only thing it
// can mean when it is the single field being sent.
func (h *Handler) AdminSetStaffPin(w http.ResponseWriter, r *http.Request) {
	id, err := objectID(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	person, err := h.staffByID(r, id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "ishchi topilmadi")
		return
	}
	if err := h.requireBranchAccess(r, person.BranchID); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	var req setPinRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	pin := strings.TrimSpace(req.PIN)
	if pin == "" {
		if _, err := h.Store.Staff.UpdateByID(r.Context(), id,
			bson.M{"$unset": bson.M{"pinHash": ""}}); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		h.logAction(r, ActStaffUpdate, "staff", id.Hex(), person.Name, "PIN o'chirildi")
		httpx.JSON(w, http.StatusOK, map[string]any{"hasPin": false})
		return
	}
	if err := pinShape(pin); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// ⚠️ **Unique within the branch, and this is not tidiness.** Two people
	// sharing a code means the till attributes every void and discount to
	// whichever row the scan reaches first — a journal that names the wrong
	// person is worse than one that names nobody, because it is believed.
	if other, found, err := h.staffByPIN(r.Context(), person.BranchID, pin); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	} else if found && other.ID != id {
		// ⚠️ The name is **not** returned. An admin who can discover whose PIN
		// a guess belongs to by typing guesses has been handed a lookup table.
		httpx.Error(w, http.StatusConflict,
			"bu PIN shu filialda allaqachon ishlatilyapti — boshqasini tanlang")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := h.Store.Staff.UpdateByID(r.Context(), id,
		bson.M{"$set": bson.M{"pinHash": string(hash), "updatedAt": time.Now()}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ⚠️ The PIN itself never reaches the log — it is a live credential, and an
	// audit trail that records secrets is a second place to steal them from.
	h.logAction(r, ActStaffUpdate, "staff", id.Hex(), person.Name, "PIN o'rnatildi")
	httpx.JSON(w, http.StatusOK, map[string]any{"hasPin": true})
}

// StaffTillSession says whether this screen has to be unlocked at all.
//
// ⚠️ **The answer is "does anybody here have a PIN", not a setting.** A
// restaurant that has not handed out any codes keeps working exactly as it did
// before this shipped — an upgrade must never lock a live till out mid-service
// over a checkbox nobody was told to tick. The moment the first PIN is set, the
// screen starts asking, which is also the moment somebody clearly meant it to.
//
// The same shape as the "zero value is today's behaviour" rule elsewhere,
// arrived at from the data rather than from a flag.
func (h *Handler) StaffTillSession(w http.ResponseWriter, r *http.Request) {
	branchID, ok := h.tillBranch(w, r)
	if !ok {
		return
	}
	n, err := h.Store.Staff.CountDocuments(r.Context(), bson.M{
		"branchId": branchID,
		"isActive": true,
		"pinHash":  bson.M{"$nin": bson.A{"", nil}},
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"pinsUsed": n > 0})
}

// ---- The device ----
//
// ⚠️ **A till screen is bound to a branch, not signed into by a person.**
// Nobody types a username and a password on a monoblock between two guests, and
// asking them to is how a restaurant ends up with one login shared by everybody
// and written on the wall. So the machine is set up once, holds a long-lived
// branch token, and from then on the only thing anybody enters is four digits.
//
// The same shape as the branch kiosk screen next door, including how it is
// revoked: a version counter on the branch, bumped when a monoblock walks out
// of the building.

// tillDeviceTTL is how long a monoblock stays bound.
//
// A year, like the kiosk. This is not a session — it is "this machine belongs
// to that restaurant", and a till that logged itself out every fortnight would
// be a support call every fortnight.
const tillDeviceTTL = 365 * 24 * time.Hour

var (
	errTillToken   = errors.New("qurilma tokeni yaroqsiz")
	errTillRevoked = errors.New("bu qurilmaning kaliti almashtirilgan — paneldan yangi havola oling")
)

// AdminTillToken binds a monoblock to a branch, or cuts an old one loose.
func (h *Handler) AdminTillToken(w http.ResponseWriter, r *http.Request) {
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
		// ⚠️ Rotating kills **every** till on this branch, not just the lost
		// one — the tokens carry no device identity, so there is nothing finer
		// to revoke. Said plainly in the panel, because the fix is walking to
		// each monoblock with a new link.
		branch.TillVersion++
		if _, err := h.Store.Branches.UpdateByID(r.Context(), id, bson.M{"$set": bson.M{
			"tillVersion": branch.TillVersion,
			"updatedAt":   time.Now(),
		}}); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		h.logAction(r, ActSettingsUpdate, "branch", id.Hex(), branch.Name,
			"kassa qurilma kaliti almashtirildi")
	}

	token, err := auth.GenerateLong(
		h.Cfg.JWTSecret, id.Hex(), "tilldevice", branch.TillVersion, tillDeviceTTL)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"token":      token,
		"branchId":   id.Hex(),
		"branchName": branch.Name,
		"version":    branch.TillVersion,
	})
}

// tillDeviceBranch resolves the branch behind a device token.
//
// ⚠️ Refuses tokens issued before the last rotation. Without the version check
// the counter would be decoration and a stolen monoblock would keep selling.
func (h *Handler) tillDeviceBranch(r *http.Request) (primitive.ObjectID, error) {
	claims := middleware.ClaimsFrom(r.Context())
	if claims == nil || claims.Role != "tilldevice" {
		return primitive.NilObjectID, errTillToken
	}
	id, err := objectID(claims.UserID)
	if err != nil {
		return primitive.NilObjectID, errTillToken
	}
	branch, err := h.branchByID(r, id)
	if err != nil {
		return primitive.NilObjectID, errTillToken
	}
	if claims.Ver != branch.TillVersion {
		return primitive.NilObjectID, errTillRevoked
	}
	return id, nil
}

// tillBranch answers "which branch is this screen" for either kind of caller.
//
// ⚠️ **Both are supported on purpose, and the device one is the future.** A
// monoblock bound to its branch is how this is meant to work; the staff-token
// path is what every till installed before this shipped is still using, and
// breaking those on deploy would take restaurants offline mid-service to fix a
// setup step nobody had been told about.
func (h *Handler) tillBranch(w http.ResponseWriter, r *http.Request) (primitive.ObjectID, bool) {
	claims := middleware.ClaimsFrom(r.Context())
	if claims != nil && claims.Role == "tilldevice" {
		id, err := h.tillDeviceBranch(r)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, err.Error())
			return primitive.NilObjectID, false
		}
		return id, true
	}
	s, ok := h.tillStaff(w, r, models.PermWaiter)
	if !ok {
		return primitive.NilObjectID, false
	}
	return s.BranchID, true
}
