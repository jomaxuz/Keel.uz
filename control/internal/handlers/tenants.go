package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"keel-control/internal/aggregate"
	"keel-control/internal/httpx"
	"keel-control/internal/models"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ListTenants returns every customer, newest first, with their numbers folded
// in — the dashboard's main screen is "who is on the platform and how are they
// doing", and two round trips to build one table is one too many.
//
// The numbers are each tenant's **own billing period**, not the calendar month:
// that is the figure an operator reads out when a restaurant asks what it owes,
// and a table that shows a different one invites exactly that mistake.
func (h *Handler) ListTenants(w http.ResponseWriter, r *http.Request) {
	now := time.Now()

	// Collected as a list and ANDed rather than written into one map: the
	// search and the warning filter both want `$or`, and assigning that key
	// twice would let the second silently discard the first — a search box that
	// stopped searching while still looking like it worked.
	// ⚠️ Every customer query starts from the role's own filter. An agent's list is
	// narrowed in the query rather than after it: a list fetched and then trimmed
	// leaks the moment somebody adds a count or an export beside it.
	actor, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	var and []bson.M
	if scope := tenantScope(actor); len(scope) > 0 {
		and = append(and, scope)
	}
	if s := strings.TrimSpace(r.URL.Query().Get("status")); s != "" {
		and = append(and, bson.M{"status": s})
	} else {
		// Closed customers are kept, not shown. They are still reachable by
		// asking for them (`?status=deleted`) — a record you cannot find again
		// is a record nobody trusts, and somebody will eventually need to know
		// what happened to a restaurant that left.
		and = append(and, bson.M{"status": bson.M{"$ne": models.StatusDeleted}})
	}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		rx := primitive.Regex{Pattern: regexpQuote(q), Options: "i"}
		and = append(and, bson.M{"$or": []bson.M{
			{"name": rx}, {"slug": rx}, {"ownerPhone": rx}, {"domains": rx},
		}})
	}
	// ⚠️ Two of the warnings **cannot be database clauses** and are applied after
	// the rows are read, below.
	//
	// Neither is a property of the tenant document. "Invoice due" depends on
	// whether the *ledger* reaches the day the customer's period closed, which
	// lives in another collection; "down" depends on what *Docker* says right
	// now, which lives nowhere at all. Reading them out and filtering in Go
	// keeps one code path deciding what each one means — the alternative is a
	// Mongo expression and a Go function that must agree forever, and the day
	// they stop agreeing the badge says one thing and the filter shows another.
	attentionKind := strings.TrimSpace(r.URL.Query().Get("attention"))
	computed := attentionKind == AttentionInvoiceDue || attentionKind == AttentionDown
	postFilter := computed || attentionKind == "any"
	if attentionKind != "" && !computed {
		clause, ok := attentionFilter(attentionKind, now)
		if !ok {
			httpx.Error(w, http.StatusBadRequest, "noma'lum filtr")
			return
		}
		// "Any" must not narrow to the document-shaped warnings, or the button
		// would hide the customers it was most important to show.
		if attentionKind != "any" {
			and = append(and, clause)
		}
	}
	filter := bson.M{}
	if len(and) > 0 {
		filter["$and"] = and
	}

	cur, err := h.Store.Tenants.Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	tenants := []models.Tenant{}
	if err := cur.All(r.Context(), &tenants); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	period, periods, err := h.periodTotals(r.Context(), tenants, now)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	lifetime, err := h.lifetimeTotals(r.Context(), tenants)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	billed, err := h.billedThrough(r.Context(), tenants)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Live, in one Docker call for the whole page. The stored `provisionStatus`
	// records what happened once and is wrong the moment a container dies —
	// which is exactly how a customer sat at "ready" with no container while
	// their domain answered 502.
	states := h.containerStates(r.Context())
	out := make([]map[string]any, 0, len(tenants))
	for _, t := range tenants {
		// Says "a password is stored" without ever carrying the password: the
		// field itself is json:"-", and this is what the form renders from.
		t.HasAdminPassword = t.AdminPassword != ""
		id := t.ID.Hex()
		m := period[id]
		container := stateOf(states, t.Slug)
		att := tenantAttention(t, now, billed[id], container)
		if postFilter {
			if attentionKind == "any" && att.Kind == "" {
				continue
			}
			if attentionKind != "any" && att.Kind != attentionKind {
				continue
			}
		}
		out = append(out, map[string]any{
			"tenant":    t,
			"period":    periods[id],
			"attention": att,
			"orders":    m.Orders,
			"revenue":   m.Revenue,
			"billable":  m.Billable,
			// Our fee against what the restaurant took. The number that says
			// whether this customer is about to start negotiating.
			"share":    m.Share,
			"lifetime": lifetime[id],
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": out})
}

func regexpQuote(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(`\.+*?()|[]{}^$`, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// CreateTenant opens a customer.
//
// The database, the container and the domain all follow from the slug, so it
// is validated hard here and never changed afterwards: renaming it later would
// orphan a database that still holds a real restaurant's orders.
func (h *Handler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Slug          string `json:"slug"`
		Name          string `json:"name"`
		Kind          string `json:"kind"`
		Domain        string `json:"domain"`
		OwnerName     string `json:"ownerName"`
		OwnerPhone    string `json:"ownerPhone"`
		PricePerOrder int    `json:"pricePerOrder"`
		Note          string `json:"note"`
		AdminUsername string `json:"adminUsername"`
		AdminPassword string `json:"adminPassword"`
		// Whether this customer gets an evaluation period, and how long.
		//
		// A pointer so that "field absent" and "false" are different things: a
		// caller that does not mention a trial gets one, which is what every
		// tenant created before this field existed got. Turning it off has to
		// be typed.
		Trial     *bool `json:"trial"`
		TrialDays int   `json:"trialDays"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if !slugRe.MatchString(slug) {
		httpx.Error(w, http.StatusBadRequest,
			"slug faqat kichik lotin harflari, raqam va tire bo'lishi mumkin (3–32 belgi)")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.Error(w, http.StatusBadRequest, "nom kerak")
		return
	}

	// Every tenant gets a working address immediately, whether or not the
	// customer owns a domain yet.
	domains := []string{slug + "." + h.Cfg.BaseDomain}
	if d := normalizeDomain(req.Domain); d != "" && d != domains[0] {
		domains = append(domains, d)
	}

	// The panel account has to be decided now: the tenant server seeds its
	// first owner on first boot and never asks again.
	adminUser := strings.ToLower(strings.TrimSpace(req.AdminUsername))
	if adminUser == "" {
		adminUser = "admin"
	}
	if len(req.AdminPassword) < 8 {
		httpx.Error(w, http.StatusBadRequest, "admin paroli kamida 8 belgi bo'lishi kerak")
		return
	}

	// Who signed them up, by id as well as by name: the agent list filters on the
	// id, and a name is editable while an id is not.
	actor, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	if !models.CanProvision(actor.RoleOf()) && actor.RoleOf() != models.RoleAgent {
		fail(w, errForbidden)
		return
	}

	// ⚠️ **An agent brings customers in; the commercial terms are not theirs.**
	//
	// Price and the trial decide what this customer pays, so a sales account that
	// could set them could sell at any price it liked — and the request is a JSON
	// body, so hiding the fields in the form would prove nothing. Cleared here, on
	// the server, where the role is known.
	if actor.RoleOf() == models.RoleAgent {
		req.PricePerOrder = 0
		req.Trial = nil
		req.TrialDays = 0
	}

	price := req.PricePerOrder
	if price <= 0 {
		price = h.Cfg.DefaultPricePerOrder
	}
	// A demo, or a customer who pays from day one.
	//
	// The two are not a cosmetic difference: a trial carries a deadline and is
	// swept when it passes, while a paying customer carries a billing anchor
	// and is never switched off by a timer. Deciding it here, once, is what
	// keeps a restaurant that agreed to pay from being cut off two weeks later
	// by a trial nobody meant to give it.
	status, trialEndsAt, subscribedAt := models.StatusActive, (*time.Time)(nil), (*time.Time)(nil)
	if req.Trial == nil || *req.Trial {
		days := req.TrialDays
		if days <= 0 {
			days = h.Cfg.TrialDays
		}
		// Bounded because the field is typed by hand: a demo of 1400 days is a
		// typo, and it would sit in the list looking like a paying customer
		// until somebody counted the months.
		if days > 365 {
			httpx.Error(w, http.StatusBadRequest, "demo muddati 1–365 kun bo'lishi kerak")
			return
		}
		// From the start of today rather than from this instant, so a demo
		// created at 23:50 is not a day shorter than one created at 09:00.
		ends := startOfToday().AddDate(0, 0, days)
		status, trialEndsAt = models.StatusTrial, &ends
	} else {
		// Billing runs from today — see internal/billing for why the anchor is
		// the customer's own date rather than the calendar's.
		from := startOfToday()
		subscribedAt = &from
	}

	t := models.Tenant{
		Slug:          slug,
		Name:          strings.TrimSpace(req.Name),
		CreatedByID:   actor.ID,
		CreatedBy:     actor.Username,
		CreatedByRole: actor.RoleOf(),
		Kind:          strings.TrimSpace(req.Kind),
		Domains:       domains,
		Status:        status,
		TrialEndsAt:   trialEndsAt,
		SubscribedAt:  subscribedAt,
		PricePerOrder: price,
		OwnerName:     strings.TrimSpace(req.OwnerName),
		OwnerPhone:    strings.TrimSpace(req.OwnerPhone),
		AdminUsername: adminUser,
		AdminPassword: req.AdminPassword,
		JWTSecret:     newSecret(),
		Note:          strings.TrimSpace(req.Note),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	res, err := h.Store.Tenants.InsertOne(r.Context(), t)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			// The closed-customer case is spelled out because it is invisible:
			// deleted tenants are hidden from the list, so an operator sees a
			// free slug and is told it is taken. It is taken — by a database
			// that still holds somebody's year of orders.
			httpx.Error(w, http.StatusConflict,
				"bu slug yoki domen allaqachon band (o'chirilgan mijozda bo'lishi mumkin — "+
					"holat bo'yicha «O'chirilgan» ni tanlab ko'ring)")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	t.ID = res.InsertedID.(primitive.ObjectID)

	// Started here rather than by a later button: a customer created and not
	// running is a customer somebody has to remember about.
	h.apply(r.Context(), &t, false)
	_ = h.Store.Tenants.FindOne(r.Context(), bson.M{"_id": t.ID}).Decode(&t)
	t.HasAdminPassword = t.AdminPassword != ""
	t.ContainerStatus = h.containerStatus(r.Context(), t.Slug)
	httpx.JSON(w, http.StatusCreated, t)
}

// GetTenant returns one customer with its daily curve.
func (h *Handler) GetTenant(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	actor, err := h.actor(r)
	if err != nil {
		fail(w, err)
		return
	}
	var t models.Tenant
	if err := h.Store.Tenants.FindOne(r.Context(), bson.M{"_id": id}).Decode(&t); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	if err := h.requireOwn(r.Context(), actor, &t); err != nil {
		fail(w, err)
		return
	}
	// Newest first in the query, oldest first in the answer. Sorting ascending
	// and then limiting returns a long-lived tenant's *first* 120 days — a
	// chart that stopped moving months ago and looks like a quiet customer
	// rather than a broken screen.
	cur, err := h.Store.Days.Find(r.Context(),
		bson.M{"tenantId": id},
		options.Find().SetSort(bson.D{{Key: "date", Value: -1}}).SetLimit(120))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	days := []models.TenantDay{}
	_ = cur.All(r.Context(), &days)
	for i, j := 0, len(days)-1; i < j; i, j = i+1, j-1 {
		days[i], days[j] = days[j], days[i]
	}

	// Summed in the database over the whole period, not in the browser over the
	// rows above: the two agree only until a tenant outlives the limit, and the
	// day they stop agreeing is the day an invoice is quoted short.
	now := time.Now()
	totals, periods, err := h.periodTotals(r.Context(), []models.Tenant{t}, now)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	lifetime, err := h.lifetimeTotals(r.Context(), []models.Tenant{t})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	billed, err := h.billedThrough(r.Context(), []models.Tenant{t})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	t.HasAdminPassword = t.AdminPassword != ""
	t.ContainerStatus = h.containerStatus(r.Context(), t.Slug)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenant": t,
		"days":   days,
		"period": periods[t.ID.Hex()],
		// The same live container state the badge on the list is built from —
		// this card must not disagree with the row that led here.
		"attention": tenantAttention(t, now, billed[t.ID.Hex()], t.ContainerStatus),
		"totals":    totals[t.ID.Hex()],
		"lifetime":  lifetime[t.ID.Hex()],
	})
}

// UpdateTenant changes what the dashboard is allowed to change.
//
// Deliberately not the slug: it names a database that holds a live business.
func (h *Handler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var req struct {
		Name          *string `json:"name"`
		Kind          *string `json:"kind"`
		Status        *string `json:"status"`
		SubscribedAt  *string `json:"subscribedAt"`
		PricePerOrder *int    `json:"pricePerOrder"`
		HideWatermark *bool   `json:"hideWatermark"`
		Showcase      *bool   `json:"showcase"`
		// Free terms, and the middle ground between free and paying.
		Free            *bool    `json:"free"`
		FreeReason      *string  `json:"freeReason"`
		FreeUntil       *string  `json:"freeUntil"`
		DiscountPercent *int     `json:"discountPercent"`
		MinMonthly      *int     `json:"minMonthly"`
		OwnerName       *string  `json:"ownerName"`
		OwnerPhone      *string  `json:"ownerPhone"`
		Note            *string  `json:"note"`
		Domains         []string `json:"domains"`
		AdminUsername   *string  `json:"adminUsername"`
		AdminPassword   *string  `json:"adminPassword"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Read the current document before changing it: the subscription anchor is
	// decided by where the tenant is coming *from*, not only by what the form
	// sends.
	var before models.Tenant
	if err := h.Store.Tenants.FindOne(r.Context(), bson.M{"_id": id}).Decode(&before); err != nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}

	set := bson.M{"updatedAt": time.Now()}
	unset := bson.M{}
	if req.Name != nil {
		set["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Kind != nil {
		set["kind"] = strings.TrimSpace(*req.Kind)
	}
	if req.Status != nil {
		switch *req.Status {
		case models.StatusActive, models.StatusTrial, models.StatusSuspended, models.StatusDeleted:
			set["status"] = *req.Status
			// A tenant that is no longer suspended cannot still carry "the
			// sweep switched this off". Left behind, the note would go on to
			// describe the next switch-off, which a human did on purpose.
			if *req.Status != models.StatusSuspended {
				unset["autoSuspendedAt"] = ""
			}
		default:
			httpx.Error(w, http.StatusBadRequest, "noma'lum holat")
			return
		}
		// Take the last numbers before the customer leaves the list.
		//
		// The aggregator skips deleted tenants, so without this the final part
		// of a day would never be collected — and those rows are what the last
		// invoice is built from. Collected before the status changes, while the
		// tenant is still something the aggregator will look at.
		if *req.Status == models.StatusDeleted && before.Status != models.StatusDeleted {
			if err := aggregate.One(r.Context(), h.Store, before, 2); err != nil {
				// Not fatal: a customer we cannot reach must still be closable,
				// and the alternative is an operator stuck with a row they
				// cannot remove.
				log.Printf("delete %s: oxirgi hisobni yig'ib bo'lmadi: %v", before.Slug, err)
			}
		}
	}

	// The subscription anchor.
	//
	// An explicit date is an operator correcting a record and is honoured; but
	// moving it re-dates every invoice that follows, so it is only ever set
	// from a value somebody typed on purpose — never as a side effect of saving
	// a phone number, and never overwritten by the automatic rule below.
	if req.SubscribedAt != nil {
		d, err := parseDay(*req.SubscribedAt)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "obuna sanasi noto'g'ri (YYYY-MM-DD)")
			return
		}
		set["subscribedAt"] = d
	} else if req.Status != nil && *req.Status == models.StatusActive &&
		before.Status != models.StatusActive && before.SubscribedAt == nil {
		// A trial that just became a paying customer. The anchor is today,
		// recorded at the one moment we know it: nobody comes back later to
		// remember which day the money arrived, and without it the first
		// invoice would be counted from the day the tenant was opened — demo
		// weeks included.
		set["subscribedAt"] = startOfToday()
	}
	if req.PricePerOrder != nil && *req.PricePerOrder >= 0 {
		set["pricePerOrder"] = *req.PricePerOrder
	}
	// The floor this customer's periods are charged at. 0 is a meaningful value
	// — "fall back to the platform's" — so it is written, not skipped: an
	// operator clearing the field is undoing a negotiated floor, and silently
	// keeping the old one would be the same trap as an empty API key that
	// disables a payment provider.
	if req.MinMonthly != nil && *req.MinMonthly >= 0 {
		set["minMonthly"] = *req.MinMonthly
	}
	if req.HideWatermark != nil {
		set["hideWatermark"] = *req.HideWatermark
		// ⚠️ The date is written when it turns on and cleared when it turns off, and only on
		// a real change: re-saving the tenant card with the switch already on must not move
		// the date forward, or a customer who has paid for three months starts again from
		// today every time somebody edits their phone number.
		if *req.HideWatermark && !before.HideWatermark {
			now := time.Now()
			set["hideWatermarkSince"] = now
		}
		if !*req.HideWatermark {
			set["hideWatermarkSince"] = nil
		}
	}
	if req.Showcase != nil {
		set["showcase"] = *req.Showcase
	}

	// Free terms.
	//
	// **The reason is required**, and refused rather than defaulted: an account
	// that pays nothing for a reason nobody wrote down becomes an argument the
	// day somebody asks, and by then the person who agreed it has left. This is
	// the same rule as cancelling an order or voiding an invoice — the entry
	// that removes money must explain itself.
	if req.Free != nil {
		reason := strings.TrimSpace(deref(req.FreeReason, before.FreeReason))
		if *req.Free && reason == "" {
			httpx.Error(w, http.StatusBadRequest,
				"bepul xizmat uchun sabab yozilishi kerak")
			return
		}
		set["free"] = *req.Free
		set["freeReason"] = reason
		if !*req.Free {
			// Turning it off clears the end date too: a stale date left behind
			// would silently switch the customer back to free the next time
			// somebody ticked the box.
			unset["freeUntil"] = ""
		}
	} else if req.FreeReason != nil {
		set["freeReason"] = strings.TrimSpace(*req.FreeReason)
	}
	if req.FreeUntil != nil {
		// Empty means **forever**, which is a real answer here rather than an
		// oversight: an anchor customer may well have been promised exactly
		// that, and the form has to be able to express it.
		if strings.TrimSpace(*req.FreeUntil) == "" {
			unset["freeUntil"] = ""
		} else {
			d, err := parseDay(*req.FreeUntil)
			if err != nil {
				httpx.Error(w, http.StatusBadRequest, "bepul muddati noto'g'ri (YYYY-MM-DD)")
				return
			}
			set["freeUntil"] = d
		}
	}
	if req.DiscountPercent != nil {
		d := *req.DiscountPercent
		if d < 0 || d > 100 {
			httpx.Error(w, http.StatusBadRequest, "chegirma 0 dan 100 gacha bo'lishi kerak")
			return
		}
		set["discountPercent"] = d
	}
	if req.OwnerName != nil {
		set["ownerName"] = strings.TrimSpace(*req.OwnerName)
	}
	if req.OwnerPhone != nil {
		set["ownerPhone"] = strings.TrimSpace(*req.OwnerPhone)
	}
	if req.Note != nil {
		set["note"] = strings.TrimSpace(*req.Note)
	}
	if req.AdminUsername != nil {
		if u := strings.ToLower(strings.TrimSpace(*req.AdminUsername)); u != "" {
			set["adminUsername"] = u
		}
	}
	// Empty means keep the stored one. The form cannot show a password it never
	// received, so a blank field must never be read as "erase it" — the same
	// rule the payment keys and the POS credentials follow.
	if req.AdminPassword != nil && *req.AdminPassword != "" {
		if len(*req.AdminPassword) < 8 {
			httpx.Error(w, http.StatusBadRequest, "admin paroli kamida 8 belgi bo'lishi kerak")
			return
		}
		set["adminPassword"] = *req.AdminPassword
	}
	if req.Domains != nil {
		clean := []string{}
		seen := map[string]bool{}
		for _, d := range req.Domains {
			if n := normalizeDomain(d); n != "" && !seen[n] {
				seen[n] = true
				clean = append(clean, n)
			}
		}
		if len(clean) == 0 {
			httpx.Error(w, http.StatusBadRequest, "kamida bitta domen qolishi kerak")
			return
		}
		set["domains"] = clean
	}

	update := bson.M{"$set": set}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	if _, err := h.Store.Tenants.UpdateByID(r.Context(), id, update); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			httpx.Error(w, http.StatusConflict, "bu domen boshqa mijozda band")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// A status change starts or stops the container; a domain change teaches
	// the edge about it. Anything else — a phone number, a note — is not worth
	// reloading the thing that stands in front of every customer.
	if req.Status != nil || req.Domains != nil {
		var t models.Tenant
		if err := h.Store.Tenants.FindOne(r.Context(), bson.M{"_id": id}).Decode(&t); err == nil {
			// A domain change is baked into the container's environment, so the
			// container has to be replaced; a status change only starts or stops
			// it, and replacing it there would be an outage for no reason.
			h.apply(r.Context(), &t, req.Domains != nil)
		}
	}
	h.GetTenant(w, r)
}

// normalizeDomain strips what people paste: a scheme, a path, a port, a case.
// deref reads an optional field, falling back to what is already stored.
func deref(v *string, fallback string) string {
	if v == nil {
		return fallback
	}
	return *v
}

func normalizeDomain(d string) string {
	d = strings.ToLower(strings.TrimSpace(d))
	d = strings.TrimPrefix(strings.TrimPrefix(d, "https://"), "http://")
	if i := strings.IndexAny(d, "/:"); i >= 0 {
		d = d[:i]
	}
	if !strings.Contains(d, ".") {
		return ""
	}
	return d
}
