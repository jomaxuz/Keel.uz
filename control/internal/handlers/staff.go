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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// Console accounts: who works here, and what each of them sees.
//
// ⚠️ **The whole point is the agent boundary**, and it is enforced in one place.
// `actor()` resolves the session into an account with its roles, `tenantScope()`
// turns them into the Mongo filter every customer query starts from, and nothing
// else decides who sees whom. A rule repeated per endpoint is a rule that will be
// written wrong on the endpoint added next month.

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
	if u.Can(models.CanSeeAllTenants) {
		return bson.M{}
	}
	return bson.M{"createdById": u.ID}
}

// requireOwn refuses a customer that is not this account's to touch.
func (h *Handler) requireOwn(ctx context.Context, u *models.User, t *models.Tenant) error {
	if u.Can(models.CanSeeAllTenants) {
		return nil
	}
	if t.CreatedByID == u.ID {
		return nil
	}
	// ⚠️ 404, not 403: an agent guessing ids should not learn that a customer exists.
	return &httpError{code: http.StatusNotFound, msg: "topilmadi"}
}

// ---- Accounts (owner only) ----

// staffView is an account as the console draws it. ⚠️ `roles` is always the
// resolved list — the browser never re-derives it from the legacy `role`, which
// is how two screens end up disagreeing about what somebody may do.
type staffView struct {
	models.User
	Roles []string `json:"roles"`
}

func viewOf(u models.User) staffView { return staffView{User: u, Roles: u.RolesOf()} }

func (h *Handler) ListStaff(w http.ResponseWriter, r *http.Request) {
	u, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	if !u.Can(models.CanManageStaff) {
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
		staffView
		Tenants int `json:"tenants"`
	}
	out := make([]row, 0, len(rows))
	for _, x := range rows {
		out = append(out, row{staffView: viewOf(x), Tenants: counts[x.ID.Hex()]})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": out, "roles": models.StaffRoles})
}

type staffRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	// ⚠️ `role` is still accepted, for a console tab that was open before an
	// account could hold several.
	Role   string   `json:"role"`
	Roles  []string `json:"roles"`
	Active *bool    `json:"isActive"`
}

// requestRoles is what a request asks for, or nil when it asks for nothing.
//
// ⚠️ **Nothing asked is not "owner".** A stored empty role reads as owner because
// of the seeded account; a request that forgot the field must not inherit that and
// create a second owner. Each name also goes through staffRole, so a typo is an
// agent.
func requestRoles(req staffRequest) []string {
	asked := []string{}
	for _, v := range append(append([]string{}, req.Roles...), req.Role) {
		if strings.TrimSpace(v) != "" {
			asked = append(asked, staffRole(v))
		}
	}
	if len(asked) == 0 {
		return nil
	}
	return models.NormalizeRoles(asked, "")
}

func (h *Handler) CreateStaff(w http.ResponseWriter, r *http.Request) {
	u, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	if !u.Can(models.CanManageStaff) {
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
	roles := requestRoles(req)
	if roles == nil {
		roles = []string{models.RoleAgent}
	}
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
		// ⚠️ Both written: `role` as the widest, for anything that still reads
		// one word.
		Role:      roles[0],
		Roles:     roles,
		IsActive:  &yes,
		CreatedBy: u.Username,
		CreatedAt: time.Now(),
	}
	res, err := h.Store.Users.InsertOne(r.Context(), acc)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	acc.ID, _ = res.InsertedID.(primitive.ObjectID)
	h.logConsole(r.Context(), u, "staff.create", name, strings.Join(roles, ","))
	httpx.JSON(w, http.StatusOK, viewOf(acc))
}

func (h *Handler) UpdateStaff(w http.ResponseWriter, r *http.Request) {
	u, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	if !u.Can(models.CanManageStaff) {
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
	// ⚠️ An explicitly empty list is refused rather than read as "no change": an
	// account with no roles would fall back to its legacy `role`, which for the
	// seeded account is owner — unticking every box would *widen* it.
	if req.Roles != nil && len(requestRoles(staffRequest{Roles: req.Roles})) == 0 {
		httpx.Error(w, http.StatusBadRequest, "kamida bitta rol tanlang")
		return
	}
	roles := requestRoles(req)
	if roles != nil {
		set["roles"] = roles
		set["role"] = roles[0]
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
	losesOwner := roles != nil && roles[0] != models.RoleOwner
	if losesOwner || (req.Active != nil && !*req.Active) {
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
	detail := ""
	if roles != nil {
		detail = strings.Join(roles, ",")
	}
	h.logConsole(r.Context(), u, "staff.update", id.Hex(), detail)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// DeleteStaff removes an account for good.
//
// ⚠️ **Beside switching off, not instead of it.** Switching off is the everyday
// answer — somebody on leave, a contract paused — and it keeps the login, so the
// person comes back as who they were. Deleting frees the login and cannot be
// undone, so it is its own button with its own confirmation.
//
// ⚠️ **What they did stays.** Customers keep the name they were signed up under
// (`createdBy`), visits keep `agentName`, the log keeps `actor` — every record
// that names this account copied the name when it was written, so nothing reads
// blank afterwards. An agent's customers simply stop belonging to anybody an
// agent filter matches; the roles that see every customer still see them.
func (h *Handler) DeleteStaff(w http.ResponseWriter, r *http.Request) {
	u, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	if !u.Can(models.CanManageStaff) {
		fail(w, errForbidden)
		return
	}
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	// ⚠️ Not yourself: the next request would be a 401 on a half-finished page,
	// and "I deleted my own login" is only ever an accident.
	if id == u.ID {
		httpx.Error(w, http.StatusBadRequest, "o'z hisobingizni o'chirib bo'lmaydi")
		return
	}
	var target models.User
	if err := h.Store.Users.FindOne(r.Context(), bson.M{"_id": id}).Decode(&target); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	if last, err := h.lastOwner(r.Context(), id); err == nil && last {
		httpx.Error(w, http.StatusBadRequest, "oxirgi owner hisobini o'zgartirib bo'lmaydi")
		return
	}
	if _, err := h.Store.Users.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logConsole(r.Context(), u, "staff.delete", target.Username, strings.Join(target.RolesOf(), ","))
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// GetStaff is one account and everything it has done.
//
// ⚠️ **Counted from the records the work already writes**, never from a
// per-person counter: customers by `createdById`, visits by `agentId`, support
// threads by `operatorId`, invoices by `issuedBy`, the log by `actorId`. A counter
// kept beside the work is a second copy, and it is the copy that is wrong after
// the first failed write.
func (h *Handler) GetStaff(w http.ResponseWriter, r *http.Request) {
	u, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	if !u.Can(models.CanManageStaff) {
		fail(w, errForbidden)
		return
	}
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	ctx := r.Context()
	var target models.User
	if err := h.Store.Users.FindOne(ctx, bson.M{"_id": id}).Decode(&target); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

	// ---- Customers they signed up ----
	byTenant := bson.M{"createdById": id}
	tenantsTotal, _ := h.Store.Tenants.CountDocuments(ctx, byTenant)
	tenantsMonth, _ := h.Store.Tenants.CountDocuments(ctx,
		bson.M{"createdById": id, "createdAt": bson.M{"$gte": monthStart}})
	byStatus := h.countBy(ctx, h.Store.Tenants, byTenant, "$status")
	type tenantRow struct {
		ID        primitive.ObjectID `bson:"_id" json:"id"`
		Name      string             `bson:"name" json:"name"`
		Slug      string             `bson:"slug" json:"slug"`
		Status    string             `bson:"status" json:"status"`
		CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	}
	recentTenants := []tenantRow{}
	if cur, err := h.Store.Tenants.Find(ctx, byTenant, options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(20).
		SetProjection(bson.M{"name": 1, "slug": 1, "status": 1, "createdAt": 1})); err == nil {
		_ = cur.All(ctx, &recentTenants)
	}

	// ---- Visits ----
	byAgent := bson.M{"agentId": id}
	visitStatus := h.countBy(ctx, h.Store.Visits, byAgent, "$status")
	visitOutcome := h.countBy(ctx, h.Store.Visits,
		bson.M{"agentId": id, "status": models.VisitDone}, "$outcome")
	recentVisits := []models.Visit{}
	if cur, err := h.Store.Visits.Find(ctx, byAgent, options.Find().
		SetSort(bson.D{{Key: "updatedAt", Value: -1}}).SetLimit(20)); err == nil {
		_ = cur.All(ctx, &recentVisits)
	}

	// ---- Support threads they picked up ----
	supportStatus := h.countBy(ctx, h.Store.SupportThreads, bson.M{"operatorId": id}, "$status")
	supportTotal := 0
	for _, n := range supportStatus {
		supportTotal += n
	}

	// ---- Crash reports they marked fixed ----
	//
	// ⚠️ **By name, because that is what the report stores** (`resolvedBy`). Both
	// the display name and the login are matched, and an empty one is never
	// matched: "resolvedBy is empty" is every report nobody named.
	names := []string{}
	for _, n := range []string{target.Name, target.Username} {
		if strings.TrimSpace(n) != "" {
			names = append(names, n)
		}
	}
	var resolved int64
	if len(names) > 0 {
		resolved, _ = h.Store.Reports.CountDocuments(ctx,
			bson.M{"resolved": true, "resolvedBy": bson.M{"$in": names}})
	}

	// ---- Invoices they issued ----
	invoiceCount, invoiceSum := 0, 0
	if agg, err := h.Store.Invoices.Aggregate(ctx, []bson.M{
		// ⚠️ Voided ones are not money anybody billed.
		{"$match": bson.M{"issuedBy": target.Username, "status": bson.M{"$ne": models.InvoiceVoid}}},
		{"$group": bson.M{"_id": nil, "n": bson.M{"$sum": 1}, "sum": bson.M{"$sum": "$amount"}}},
	}); err == nil {
		var got []struct {
			N   int `bson:"n"`
			Sum int `bson:"sum"`
		}
		if agg.All(ctx, &got) == nil && len(got) > 0 {
			invoiceCount, invoiceSum = got[0].N, got[0].Sum
		}
	}

	// ---- What they did in the console ----
	byActor := bson.M{"actorId": id}
	logTotal, _ := h.Store.ConsoleLogs.CountDocuments(ctx, byActor)
	logMonth, _ := h.Store.ConsoleLogs.CountDocuments(ctx,
		bson.M{"actorId": id, "at": bson.M{"$gte": monthStart}})
	recentLog := []models.ConsoleLog{}
	if cur, err := h.Store.ConsoleLogs.Find(ctx, byActor, options.Find().
		SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(100)); err == nil {
		_ = cur.All(ctx, &recentLog)
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"account": viewOf(target),
		"tenants": map[string]any{
			"total": tenantsTotal, "thisMonth": tenantsMonth,
			"byStatus": byStatus, "recent": recentTenants,
		},
		"visits": map[string]any{
			"planned":  visitStatus[models.VisitPlanned],
			"done":     visitStatus[models.VisitDone],
			"positive": visitOutcome[models.OutcomePositive],
			"negative": visitOutcome[models.OutcomeNegative],
			"callback": visitOutcome[models.OutcomeCallback],
			"recent":   recentVisits,
		},
		"support": map[string]any{
			"total":   supportTotal,
			"waiting": supportStatus[string(models.SupportWaiting)],
			"open":    supportStatus[string(models.SupportOpen)],
			"closed":  supportStatus[string(models.SupportClosed)],
		},
		"reports":  map[string]any{"resolved": resolved},
		"invoices": map[string]any{"issued": invoiceCount, "amount": invoiceSum},
		"log":      map[string]any{"total": logTotal, "thisMonth": logMonth, "recent": recentLog},
	})
}

// countBy groups a collection's matching documents by one field. ⚠️ Never nil —
// an account that has done nothing yet is the common case on this page.
func (h *Handler) countBy(
	ctx context.Context,
	coll *mongo.Collection, match bson.M, field string,
) map[string]int {
	out := map[string]int{}
	cur, err := coll.Aggregate(ctx, []bson.M{
		{"$match": match},
		{"$group": bson.M{"_id": field, "n": bson.M{"$sum": 1}}},
	})
	if err != nil {
		return out
	}
	var got []struct {
		ID any `bson:"_id"`
		N  int `bson:"n"`
	}
	if cur.All(ctx, &got) != nil {
		return out
	}
	for _, g := range got {
		if key, ok := g.ID.(string); ok {
			out[key] += g.N
		}
	}
	return out
}

func (h *Handler) lastOwner(ctx context.Context, id primitive.ObjectID) (bool, error) {
	var target models.User
	if err := h.Store.Users.FindOne(ctx, bson.M{"_id": id}).Decode(&target); err != nil {
		return false, err
	}
	if !target.Has(models.RoleOwner) {
		return false, nil
	}
	n, err := h.Store.Users.CountDocuments(ctx, bson.M{
		"_id": bson.M{"$ne": id},
		// ⚠️ Switched-off owners do not count: an owner nobody can sign in as
		// cannot fix the console either.
		"isActive": bson.M{"$ne": false},
		"$or":      ownerFilter(),
	})
	return n == 0, err
}

// ownerFilter matches every account that holds the owner role, in both shapes.
func ownerFilter() []bson.M {
	noList := bson.M{"$or": []bson.M{
		{"roles": bson.M{"$exists": false}},
		{"roles": bson.M{"$size": 0}},
	}}
	return []bson.M{
		{"roles": models.RoleOwner},
		// Accounts from before several roles: the single `role` decides.
		{"$and": []bson.M{noList, {"role": models.RoleOwner}}},
		// The seeded account predates both fields, and it is an owner.
		{"$and": []bson.M{noList, {"$or": []bson.M{
			{"role": bson.M{"$in": []any{"", nil}}},
			{"role": bson.M{"$exists": false}},
		}}}},
	}
}

func staffRole(v string) string {
	switch strings.TrimSpace(v) {
	case models.RoleOwner, models.RoleAdmin, models.RoleManager, models.RoleSupport:
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
	if !u.Can(models.CanSeeLog) {
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

// need wraps a handler so only an account holding `perm` — through any of its
// roles — reaches it.
func (h *Handler) need(perm string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := h.actor(r)
		if err != nil {
			fail(w, err)
			return
		}
		// ⚠️ An unknown permission name refuses everybody rather than allowing
		// them. A typo in a route's gate must fail closed: "nobody can reach the
		// invoices" is a bug report, "everybody can" is a loss.
		check, ok := models.Permissions[perm]
		if !ok || !u.Can(check) {
			fail(w, errForbidden)
			return
		}
		next(w, r)
	}
}
