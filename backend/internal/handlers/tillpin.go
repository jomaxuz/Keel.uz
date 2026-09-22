package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
// ⚠️ **In memory, and deliberately not the shared rate limiter.** The IP gate
// in middleware guards the server; this guards a four-digit secret, and it has
// to count attempts against the till rather than against whoever is asking. A
// restaurant behind one NAT would otherwise lock its own cashiers out because a
// different branch was being probed.
//
// ⚠️ **Two counters, and the reason is that a wrong PIN names nobody.** The
// unlock tries the digits against every employee on the branch; when none
// matches, the server does not know *whose* PIN was being attempted, so a
// lockout cannot be filed against a person. One counter for the whole till was
// the honest answer to that — and it meant one waiter fat-fingering their code
// five times stopped the counter for everybody, mid-service, which is a worse
// outcome than the attack it prevents.
//
// So the short lockout is keyed by **the digits that were typed**. Somebody
// mistyping 1111 five times blocks 1111; the cashier whose code is 2345 walks
// up and gets in. It is not a person, but it is the closest thing to one that a
// failed attempt actually carries.
//
// ⚠️ **The till-wide counter stays as the brute-force floor, and it has to be
// looser than the per-PIN one or it is the only one that ever fires.** Ten
// thousand codes at five tries each is not a wall; twenty wrong PINs from one
// till in a row is not a shift going badly, it is somebody working through the
// space.
const (
	pinMaxAttempts = 5
	pinLockout     = 5 * time.Minute

	tillMaxAttempts = 20
	tillLockout     = 15 * time.Minute
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

func (p *pinAttempts) fail(key string, max int, lockout time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	c := p.count[key]
	if c == nil {
		c = &pinCounter{}
		p.count[key] = c
	}
	c.n++
	if c.n >= max {
		c.locked = time.Now().Add(lockout)
		c.n = 0
	}
}

// pinKey is the till-and-digits key the short lockout counts against.
//
// ⚠️ **Hashed, and the branch is mixed in.** Holding typed PINs in a map in
// clear would put every code somebody fumbled into a heap dump, and without the
// branch a lockout in one restaurant would lock the same four digits in the one
// next door. SHA-256 rather than bcrypt: this runs on every attempt and is a
// map key, not a stored credential — the secret it protects is four digits
// long, so the work factor buys nothing here and costs a request.
func pinKey(branch, pin string) string {
	sum := sha256.Sum256([]byte(branch + ":" + pin))
	return "pin:" + hex.EncodeToString(sum[:8])
}

func (p *pinAttempts) ok(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.count, key)
}

// pinWaitMessage says how long is left in words a cashier can act on.
//
// ⚠️ **Minutes once it is minutes.** The lockout used to be a minute, so
// "83 soniyadan keyin" was fine; at five minutes the same sentence reads
// "300 soniyadan keyin qayta urining", which nobody converts in their head
// while a queue watches.
func pinWaitMessage(left time.Duration) string {
	secs := int(left.Seconds()) + 1
	if secs < 90 {
		return "juda ko'p urinish — " + strconv.Itoa(secs) + " soniyadan keyin qayta urining"
	}
	return "juda ko'p urinish — " + strconv.Itoa((secs+59)/60) + " daqiqadan keyin qayta urining"
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

	tillKey := branchID.Hex()
	codeKey := pinKey(tillKey, pin)
	// ⚠️ Both gates are checked, and the message is the same either way. Telling
	// the room which one fired would say whether these particular digits are
	// the ones somebody has been trying.
	for _, key := range []string{codeKey, tillKey} {
		if locked, left := pinGate.blocked(key); locked {
			// ⚠️ 429 and the time remaining, rather than a bare refusal: a
			// cashier who cannot tell "wrong code" from "locked out" retypes
			// the same PIN and extends the lockout.
			httpx.Error(w, http.StatusTooManyRequests, pinWaitMessage(left))
			return
		}
	}

	person, found, err := h.staffByPIN(r.Context(), branchID, pin)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !found {
		pinGate.fail(codeKey, pinMaxAttempts, pinLockout)
		pinGate.fail(tillKey, tillMaxAttempts, tillLockout)
		// ⚠️ **Never says whose PIN it is or was close.** The screen is in a
		// public room and the message is read by whoever is standing there.
		httpx.Error(w, http.StatusUnauthorized, "PIN noto'g'ri")
		return
	}
	pinGate.ok(codeKey)
	pinGate.ok(tillKey)

	// ⚠️ **The role, and without this line the whole permission model is off on
	// this screen.**
	//
	// `staffByPIN` reads the employee straight out of Mongo, and a raw Staff
	// answers `Can()` from the three legacy booleans — which have no `void` at
	// all, so an unknown permission is refused. The result was not an error
	// anywhere: an "Ish boshqaruvchi" unlocked the till normally, was labelled
	// a cashier because `canCashier` fell through to the legacy flag, and the
	// exit button simply never appeared. Nothing failed, nothing was logged,
	// and the only symptom was a button that was missing for the exact people
	// it exists for.
	//
	// `staffFromCtx` already does this for every *authenticated* request, which
	// is why the server-side checks were right the whole time — it is only the
	// view handed back to the screen that was reading the old answer.
	h.withRole(r.Context(), &person)

	// ⚠️ **Refused here, on the screen the person is standing at.**
	//
	// A PIN used to be enough to unlock the till whatever the role granted — so
	// a technologist, whose seeded role grants nothing, opened the counter
	// normally and then met "ruxsat yo'q" on the main screen, having already
	// taken over somebody else's session. Two wrong things at once: the refusal
	// arrived one screen too late, and the till was left locked to a person who
	// cannot use it until somebody works out how to get back.
	//
	// ⚠️ **`waiter` is the floor, not `cashier`.** A waiter, a barman and a host
	// all have business on this screen; the drawer is a separate permission
	// asked for separately. Requiring `cashier` here would lock out most of the
	// people the till was built for.
	if !person.Can(models.PermWaiter) && !person.Can(models.PermKitchen) {
		// ⚠️ **Not counted as a failed PIN.** The code was right — this is a
		// person who may not work here, and locking the whole branch out for a
		// minute because a technologist tried their own PIN would punish the
		// queue for somebody else's curiosity.
		httpx.Error(w, http.StatusForbidden,
			person.Name+": bu ekran uchun ruxsat yo'q")
		return
	}

	// ⚠️ **The work shift, asked here, on the screen the person is standing
	// at.**
	//
	// A PIN says who is at the counter; attendance says whether they are at
	// work. Until this check existed those were different questions with no
	// connection: somebody could sell all evening without ever clocking in, and
	// the absence only surfaced at the end of the month as a payroll row with
	// no hours in it — long after the evening it belonged to, and against a
	// person who by then remembers it differently.
	//
	// ⚠️ **Refused rather than warned.** A banner on a working till is a banner
	// that gets worked past: the first guest is already standing there and
	// clocking in is a thing you will do in a minute. The same argument the
	// cash-shift gate is built on, one layer earlier.
	//
	// ⚠️ **Its own field, not just a sentence.** The screen draws a dialog that
	// names the person and says where a shift is opened; a refusal it cannot
	// recognise would come out as another red line under the pad, which is
	// where "PIN noto'g'ri" lives — and the answer to those two is not the same.
	//
	// ⚠️ **The monoblock is not where a shift is opened**, so the message does
	// not offer to take anybody there. Clocking in is done from the employee's
	// own phone or at the branch's kiosk code — both of which check *where the
	// person is*, which is the entire point of attendance. A till screen bolted
	// to the counter could only ever answer "yes, they are at the counter".
	//
	// ⚠️ **A branch setting, off by default.** A restaurant that has never used
	// attendance would meet this as every PIN being refused, with a queue at
	// the counter and nothing on the screen the cashier can act on — so the
	// rule is switched on by the person who runs the restaurant, in the branch
	// form, and not by a deploy. `branchByID` failing is read as "off" for the
	// same reason the shift lookup fails open: this check may not be the reason
	// a counter cannot sell.
	branch, berr := h.branchByID(r, branchID)
	if berr == nil && branch.RequireShift && !h.hasOpenShift(r.Context(), person.ID) {
		httpx.JSON(w, http.StatusConflict, map[string]any{
			"error": httpx.T(w, person.Name+
				": ish smenangiz ochilmagan — smenani o'z telefoningizdan yoki kiosk QR orqali boshlang"),
			"needsShift": true,
			"staffName":  person.Name,
		})
		return
	}

	// ⚠️ **The machine is carried forward into the person's session.**
	//
	// Unlocking swaps a device token for one naming the employee, and until the
	// device id came along with it the session knew who was acting but not
	// where — so a screen could not retire itself, and nothing could say which
	// counter a void was rung on. Carried rather than re-derived because after
	// the swap there is nothing left to derive it from: the device token is
	// gone from the request.
	//
	// Empty when the screen was signed in with a staff login rather than bound,
	// and empty for tills paired before the registry — both read as "no machine
	// to retire", which is exactly true.
	dev := ""
	if c := middleware.ClaimsFrom(r.Context()); c != nil {
		dev = c.Dev
	}

	// ⚠️ **Not auth.Generate**, which issues seven days. Four digits must not
	// buy a week: the ordinary staff token is bought with a username and a
	// password, this one with a code tapped in front of the room.
	token, err := auth.GenerateDevice(
		h.Cfg.JWTSecret, person.ID.Hex(), "till", dev, 0, tillSessionTTL)
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
	// ⚠️ **Whether this person runs the restaurant rather than the counter.**
	//
	// Sent so the screen can hide the exit button from a cashier or a waiter —
	// retiring a bound machine takes it out of service and getting it back
	// needs somebody with a panel login to fetch a fresh link, which is not a
	// thing to leave one mis-tap away during service.
	//
	// ⚠️ It is `void` and not a permission of its own, and that is a judgement
	// worth stating: the two acts are not alike — one takes money out, the
	// other stops a machine selling — but the *set of people* is exactly right,
	// and every seeded manager role holds it while Kassir, Ofitsiant, Barmen
	// and Xostes do not. A seventh permission would sit unticked in every
	// restaurant until each one discovered it, which is a button that exists
	// for nobody.
	CanExit bool `json:"canExit"`
	// Whether this person may write the shopping list somebody is sent to the
	// market with.
	//
	// ⚠️ **Sent as its own flag rather than inferred from the role name.** The
	// screen decides what to draw from answers, never from spelling — the trap
	// staffrole.go opens with — and a cashier standing at this monoblock is
	// exactly the person who first hears the kitchen say something has run out.
	CanBuyOrder bool `json:"canBuyOrder"`
	// What this person's job is called, from the role. ⚠️ **The role's own
	// name, not a label derived from the permissions.** The screen used to
	// print "Kassir" for anybody who could work a till, so a manager and a
	// cashier standing at the same monoblock read as the same person — and the
	// name in the corner is how the room knows who is unlocked.
	//
	// Empty for an account with no role at all, which the screen falls back on
	// exactly as it did before. Never *read* as a permission: `Position` is
	// free text and this is a role name, and treating either as a right is the
	// trap staffrole.go opens with.
	Role string `json:"role,omitempty"`
	// The same name in the other two languages, empty when the role has none.
	// ⚠️ Sent rather than resolved here: which word this screen shows is the
	// screen's own language, and one till can be unlocked by a manager reading
	// Russian right after a waiter reading Uzbek.
	RoleRu string `json:"roleRu,omitempty"`
	RoleEn string `json:"roleEn,omitempty"`
}

func tillPerson(s models.Staff) tillPersonView {
	return tillPersonView{
		ID:          s.ID.Hex(),
		Name:        s.Name,
		Position:    s.Position,
		CanWaiter:   s.Can(models.PermWaiter),
		CanCashier:  s.Can(models.PermCashier),
		CanExit:     s.Can(models.PermVoid),
		CanBuyOrder: s.Can(models.PermBuyOrder),
		Role:        s.RoleName,
		RoleRu:      s.RoleNameRu,
		RoleEn:      s.RoleNameEn,
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

	// ---- What the lock screen puts on itself ----
	//
	// ⚠️ **Folded into the call the screen already makes**, exactly as the site's
	// banners are folded into `GET /restaurant`. A second request would be paid
	// on every wake of every monoblock to fetch three strings and a list of
	// filenames, and it would arrive *after* the pad — so the lock screen would
	// visibly redraw itself in front of whoever is standing there.
	//
	// ⚠️ **Named from the branch's own brand**, not from the first active one.
	// A two-brand company has two sets of monoblocks and the wrong name on a
	// lock screen is the kind of error nobody reports and everybody notices.
	//
	// ⚠️ **And what this business sells**, which the till had no way to ask.
	// Every word on the screen is a kitchen's — "Taom qidirish", "Chek bo'sh —
	// menyudan taom tanlang" — and a grocery's cashier reads them all day. The
	// brand is already being loaded here for its name, so the type rides along
	// rather than costing a request of its own.
	brandName, branchName, businessType := "", "", ""
	banners := []models.Banner{}
	if br, err := h.branchByID(r, branchID); err == nil {
		branchName = br.Name
		banners = h.tillBanners(r.Context(), br.BrandID)
		var brand models.Brand
		if err := h.Store.Brands.FindOne(
			r.Context(), bson.M{"_id": br.BrandID},
		).Decode(&brand); err == nil {
			brandName = brand.Name
			businessType = string(brand.BusinessType)
		}
	}
	// ⚠️ Falls back to the company name, and only when the brand has none: a
	// single-brand restaurant that never opened the brand editor has the name in
	// `restaurant`, and a lock screen labelled with an empty string reads as a
	// machine that has not been set up.
	if brandName == "" {
		brandName = h.restaurantName(r.Context())
	}

	// ⚠️ Only the picture goes to the screen. The rest of a banner row is the
	// owner's editing state — the schedule, the sort key, the caption nobody
	// draws here — and this response is served to a screen with nobody signed
	// in to it.
	images := make([]string, 0, len(banners))
	for _, b := range banners {
		images = append(images, b.ImageURL)
	}

	// ⚠️ **Folded in here for the same reason the banners are.** A second
	// request would be paid on every wake of every monoblock to fetch three
	// fields, and it would arrive *after* the screen — so a warning about a
	// subscription running out would appear a beat late, in front of whoever is
	// standing there. Nil on the overwhelmingly common day, and the screens
	// draw nothing for nil.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"pinsUsed":     n > 0,
		"brandName":    brandName,
		"branchName":   branchName,
		// ⚠️ Empty for a restaurant and for every brand written before the
		// field existed, which the screens read as a kitchen — the same
		// fallback every predicate makes on both sides.
		"businessType": businessType,
		"banners":      images,
		"subscription": h.subscriptionNotice(r.Context()),
	})
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
		// ⚠️ Rotating still kills **every** till on this branch, and now that
		// is a choice rather than a limitation. A single machine can be
		// unbound one at a time (AdminRemoveTillDevice), which is the right
		// tool for a replaced tablet; this one is for a theft, where the
		// question is not "which machine" but "none of them, right now" — and
		// a manager who has just lost a monoblock should not have to work out
		// which row it was. Said plainly in the panel, because the fix is
		// walking to each of the others with a new link.
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

	// ⚠️ **The cap is checked here and nowhere else.** Issuing a link is the
	// act of adding a register — it is what somebody does when a new machine
	// goes on the counter — so this is the one moment where declining costs a
	// restaurant nothing that was already working. A check anywhere later would
	// mean a till that stops selling, and the evening it chose to do that would
	// be the busiest one. See tilldevices.go.
	//
	// ⚠️ After the rotation above, deliberately: rotating frees every slot, and
	// a manager replacing a stolen monoblock must not be refused by a limit
	// filled with the hardware they just reported gone.
	n, _ := h.countTillDevices(r, *branch)
	if full, body := h.tillCapReached(r, *branch); full {
		// 402 rather than 403: "your plan does not include this" and "you are
		// not allowed this" send the manager to two different people.
		//
		// Translated here rather than where the body is built: the builder has
		// no writer to ask, and the writer is what knows the language.
		if msg, ok := body["error"].(string); ok {
			body["error"] = httpx.T(w, msg)
		}
		httpx.JSON(w, http.StatusPaymentRequired, body)
		return
	}

	// ⚠️ **The row is written before the token exists**, so a machine can never
	// hold a working link that nothing counts. The reverse order fails towards
	// a restaurant quietly running more registers than it bought, which is the
	// failure this whole registry was added to close.
	dev := models.TillDevice{
		BranchID: id,
		Name:     deviceName(r.URL.Query().Get("name"), n),
		Version:  branch.TillVersion,
		// ⚠️ Whoever is asking, which is the till application itself when the
		// setup screen is doing the asking. Paired from the panel instead, this
		// is the manager's laptop — corrected by the first call the machine
		// makes for itself (touchTillDevice), which is within the minute.
		Host:      tillHost(r),
		IP:        clientIP(r),
		IssuedBy:  h.adminName(r),
		CreatedAt: time.Now(),
	}
	res, err := h.Store.TillDevices.InsertOne(r.Context(), dev)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	dev.ID = res.InsertedID.(primitive.ObjectID)

	token, err := auth.GenerateDevice(
		h.Cfg.JWTSecret, id.Hex(), "tilldevice", dev.ID.Hex(),
		branch.TillVersion, tillDeviceTTL)
	if err != nil {
		// The row would otherwise count against the cap for a link nobody ever
		// received.
		_, _ = h.Store.TillDevices.DeleteOne(r.Context(), bson.M{"_id": dev.ID})
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"token":      token,
		"branchId":   id.Hex(),
		"branchName": branch.Name,
		"version":    branch.TillVersion,
		"deviceId":   dev.ID.Hex(),
		"deviceName": dev.Name,
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
	// ⚠️ **An unnamed device is not refused.** Every till paired before the
	// registry existed carries a token with no `dev` claim, and treating those
	// as invalid would take every one of them offline on the deploy that adds
	// this — the same grandfathering EnsureKitchenAccess exists for, arrived at
	// from the token instead of from a column.
	if claims.Dev != "" {
		devID, err := objectID(claims.Dev)
		if err != nil {
			return primitive.NilObjectID, errTillToken
		}
		n, err := h.Store.TillDevices.CountDocuments(r.Context(), bson.M{
			"_id": devID, "branchId": id,
		})
		if err == nil && n == 0 {
			// Removed from the panel, or the screen retired itself. This is the
			// finer revocation the branch counter could not express.
			return primitive.NilObjectID, errTillRevoked
		}
		h.touchTillDevice(r, devID)
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
