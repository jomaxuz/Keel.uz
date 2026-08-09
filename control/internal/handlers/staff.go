package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"keel-control/internal/httpx"
	"keel-control/internal/middleware"
	"keel-control/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// Console accounts: who works here, and what each of them sees.
//
// ⚠️ **The whole point is the agent boundary**, and it is enforced in one place.
// `actor()` resolves the session into an account with a role, `tenantScope()` turns
// that role into the Mongo filter every customer query starts from, and nothing else
// decides who sees whom. A rule repeated per endpoint is a rule that will be written
// wrong on the endpoint added next month.

// actor is the account behind this request.
//
// Read from the database rather than from the token, deliberately: a role change or a
// deactivated account has to take effect on the next request, not when a token
// happens to expire. It is one indexed lookup per request against a collection with a
// handful of rows.
func (h *Handler) actor(r *http.Request) (*models.User, error) {
	c := middleware.From(r.Context())
	if c == nil {
		return nil, errUnauthorised
	}
	id, err := primitive.ObjectIDFromHex(c.UserID)
	if err != nil {
		return nil, errUnauthorised
	}
	var u models.User
	if err := h.Store.Users.FindOne(r.Context(), bson.M{"_id": id}).Decode(&u); err != nil {
		return nil, errUnauthorised
	}
	if !u.Active() {
		return nil, errUnauthorised
	}
	return &u, nil
}

var errUnauthorised = &httpError{code: http.StatusUnauthorized, msg: "kirish talab qilinadi"}

type httpError struct {
	code int
	msg  string
}

func (e *httpError) Error() string { return e.msg }

// fail writes the right status for an error from actor() or a permission check.
func fail(w http.ResponseWriter, err error) {
	if e, ok := err.(*httpError); ok {
		httpx.Error(w, e.code, e.msg)
		return
	}
	httpx.Error(w, http.StatusInternalServerError, err.Error())
}

var errForbidden = &httpError{code: http.StatusForbidden, msg: "bu amal uchun ruxsat yo'q"}

// tenantScope is the filter every customer query starts from.
//
// ⚠️ An agent sees the customers **they** brought in, and that is expressed as a
// filter rather than as a check after the read: a list that is fetched and then
// trimmed is a list that leaks the moment somebody adds a count, an aggregate or a
// CSV export next to it.
func tenantScope(u *models.User) bson.M {
	if models.CanSeeAllTenants(u.RoleOf()) {
		return bson.M{}
	}
	return bson.M{"createdById": u.ID}
}

// requireOwn refuses a customer that is not this account's to touch.
func (h *Handler) requireOwn(ctx context.Context, u *models.User, t *models.Tenant) error {
	if models.CanSeeAllTenants(u.RoleOf()) {
		return nil
	}
	if t.CreatedByID == u.ID {
		return nil
	}
	// ⚠️ 404, not 403: an agent guessing ids should not learn that a customer exists.
	return &httpError{code: http.StatusNotFound, msg: "topilmadi"}
}

// ---- Accounts (owner only) ----

func (h *Handler) ListStaff(w http.ResponseWriter, r *http.Request) {
	u, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	if !models.CanManageStaff(u.RoleOf()) {
		fail(w, errForbidden)
		return
	}
	cur, err := h.Store.Users.Find(r.Context(), bson.M{},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.User{}
	_ = cur.All(r.Context(), &rows)

	// How many customers each of them brought in. One aggregation rather than a
	// query per person: this is the number the page exists to show.
	counts := map[string]int{}
	agg, err := h.Store.Tenants.Aggregate(r.Context(), []bson.M{
		{"$match": bson.M{"createdById": bson.M{"$exists": true}}},
		{"$group": bson.M{"_id": "$createdById", "n": bson.M{"$sum": 1}}},
	})
	if err == nil {
		var got []struct {
			ID primitive.ObjectID `bson:"_id"`
			N  int                `bson:"n"`
		}
		_ = agg.All(r.Context(), &got)
		for _, g := range got {
			counts[g.ID.Hex()] = g.N
		}
	}

	type row struct {
		models.User
		Tenants int `json:"tenants"`
	}
	out := make([]row, 0, len(rows))
	for _, x := range rows {
		out = append(out, row{User: x, Tenants: counts[x.ID.Hex()]})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": out, "roles": models.StaffRoles})
}

type staffRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
	Active   *bool  `json:"isActive"`
}

func (h *Handler) CreateStaff(w http.ResponseWriter, r *http.Request) {
	u, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	if !models.CanManageStaff(u.RoleOf()) {
		fail(w, errForbidden)
		return
	}
	var req staffRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	name := strings.ToLower(strings.TrimSpace(req.Username))
	if len(name) < 3 {
		httpx.Error(w, http.StatusBadRequest, "login kamida 3 belgi")
		return
	}
	if len(req.Password) < 8 {
		httpx.Error(w, http.StatusBadRequest, "parol kamida 8 belgi")
		return
	}
	role := staffRole(req.Role)
	if n, _ := h.Store.Users.CountDocuments(r.Context(), bson.M{"username": name}); n > 0 {
		httpx.Error(w, http.StatusConflict, "bu login band")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	yes := true
	acc := models.User{
		Username:     name,
		PasswordHash: string(hash),
		Name:         strings.TrimSpace(req.Name),
		Phone:        strings.TrimSpace(req.Phone),
		Role:         role,
		IsActive:     &yes,
		CreatedBy:    u.Username,
		CreatedAt:    time.Now(),
	}
	res, err := h.Store.Users.InsertOne(r.Context(), acc)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	acc.ID, _ = res.InsertedID.(primitive.ObjectID)
	h.logConsole(r.Context(), u, "staff.create", name, role)
	httpx.JSON(w, http.StatusOK, acc)
}

func (h *Handler) UpdateStaff(w http.ResponseWriter, r *http.Request) {
	u, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	if !models.CanManageStaff(u.RoleOf()) {
		fail(w, errForbidden)
		return
	}
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	var req staffRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	set := bson.M{}
	if req.Name != "" {
		set["name"] = strings.TrimSpace(req.Name)
	}
	if req.Phone != "" {
		set["phone"] = strings.TrimSpace(req.Phone)
	}
	if req.Role != "" {
		set["role"] = staffRole(req.Role)
	}
	if req.Active != nil {
		set["isActive"] = *req.Active
	}
	// ⚠️ An empty password means "leave it", never "clear it" — the same trap the
	// payment keys and the kiosk secret both shipped with once.
	if req.Password != "" {
		if len(req.Password) < 8 {
			httpx.Error(w, http.StatusBadRequest, "parol kamida 8 belgi")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		set["passwordHash"] = string(hash)
	}
	// ⚠️ The last owner cannot be demoted or switched off. An account with nobody
	// able to create accounts is a console nobody can ever fix from inside.
	if (set["role"] != nil && set["role"] != models.RoleOwner) ||
		(req.Active != nil && !*req.Active) {
		if last, err := h.lastOwner(r.Context(), id); err == nil && last {
			httpx.Error(w, http.StatusBadRequest, "oxirgi owner hisobini o'zgartirib bo'lmaydi")
			return
		}
	}
	if len(set) == 0 {
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	if _, err := h.Store.Users.UpdateByID(r.Context(), id, bson.M{"$set": set}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logConsole(r.Context(), u, "staff.update", id.Hex(), "")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) lastOwner(ctx context.Context, id primitive.ObjectID) (bool, error) {
	var target models.User
	if err := h.Store.Users.FindOne(ctx, bson.M{"_id": id}).Decode(&target); err != nil {
		return false, err
	}
	if target.RoleOf() != models.RoleOwner {
		return false, nil
	}
	n, err := h.Store.Users.CountDocuments(ctx, bson.M{
		"_id": bson.M{"$ne": id},
		"$or": []bson.M{
			{"role": models.RoleOwner},
			// The seeded account predates the field, and it is an owner.
			{"role": bson.M{"$in": []any{"", nil}}},
			{"role": bson.M{"$exists": false}},
		},
	})
	return n == 0, err
}

func staffRole(v string) string {
	switch strings.TrimSpace(v) {
	case models.RoleOwner, models.RoleAdmin, models.RoleManager:
		return strings.TrimSpace(v)
	default:
		// ⚠️ Unknown becomes **agent**, the least reach. A typo in a role name must
		// not hand somebody the console.
		return models.RoleAgent
	}
}

// logConsole records one action. Best effort: a failed log must not fail the work,
// and a missing row is visible as a gap rather than as a refused request.
func (h *Handler) logConsole(ctx context.Context, u *models.User, action, target, detail string) {
	if u == nil {
		return
	}
	_, _ = h.Store.ConsoleLogs.InsertOne(ctx, models.ConsoleLog{
		ActorID: u.ID, Actor: u.Username, ActorRole: u.RoleOf(),
		Action: action, Target: target, Detail: detail, At: time.Now(),
	})
}

// ListConsoleLog is the owner's answer to "who did this".
func (h *Handler) ListConsoleLog(w http.ResponseWriter, r *http.Request) {
	u, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	if !models.CanSeeLog(u.RoleOf()) {
		fail(w, errForbidden)
		return
	}
	filter := bson.M{}
	if a := strings.TrimSpace(r.URL.Query().Get("actor")); a != "" {
		filter["actor"] = a
	}
	cur, err := h.Store.ConsoleLogs.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(300))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.ConsoleLog{}
	_ = cur.All(r.Context(), &rows)
	httpx.JSON(w, http.StatusOK, map[string]any{"items": rows})
}

// ---- The gate ----
//
// ⚠️ **Permissions belong on the router, not inside each handler.**
//
// Gating handler by handler is how this shipped wrong the first time: I checked the
// customer list, the card and the live figures, and left everything that *changes*
// something open — so any account could suspend a customer, delete one, issue an
// invoice or restart the fleet. The dangerous endpoints are exactly the ones easiest
// to forget, because they are the ones nobody opens while testing a sales account.
//
// So the rule is declared beside the route, once, and a new route with no gate is
// visible as a missing wrapper rather than as a handler that quietly allows anybody.

// need wraps a handler so only a role holding `perm` reaches it.
func (h *Handler) need(perm string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := h.actor(r)
		if err != nil {
			fail(w, err)
			return
		}
		role := u.RoleOf()
		allowed := false
		switch perm {
		case "provision":
			allowed = models.CanProvision(role)
		case "billing":
			allowed = models.CanBill(role)
		case "stats":
			allowed = models.CanSeeStats(role)
		case "staff":
			allowed = models.CanManageStaff(role)
		case "log":
			allowed = models.CanSeeLog(role)
		default:
			// ⚠️ An unknown permission name refuses everybody rather than allowing
			// them. A typo in a route's gate must fail closed: "nobody can reach the
			// invoices" is a bug report, "everybody can" is a loss.
			allowed = false
		}
		if !allowed {
			fail(w, errForbidden)
			return
		}
		next(w, r)
	}
}
