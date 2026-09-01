package handlers

// Selling a restaurant its counter.
//
// ⚠️ **Only the console switches this on, and only an operator who can
// provision.** The obvious alternative — a toggle in the restaurant's own
// settings — puts the product's price list in the hands of the person paying
// it, which is the same trap as `kioskSecret` and `soldOut` in the tenant app,
// except here the field *is* the business model. Agents and managers are out
// for a second reason: this is not a sales concession, it is a change to what
// the customer is billed, and `need("provision")` is already the line drawn
// around that everywhere else in this router.
//
// The shape is `export.go`'s, deliberately: a singleton document written
// straight into the **tenant's own database**, which the restaurant's server
// reads and enforces. One writer, one reader, nothing to keep in sync — and no
// new path from a tenant container back to the control plane. That path not
// existing is what makes one restaurant unable to reach another's data.
//
// ⚠️ **What crosses the boundary is entitlements, not the price list logic.**
// The console resolves the plan into a flat list — these modules, this many
// registers, paid until this date — so the restaurant's server holds no copy of
// the ladder. A second copy of pricing arithmetic is a second answer to "what
// does this customer get", and the two would part company on the first rung we
// renamed. The plan *prices* do go, because the panel draws an upgrade button
// with a number on it, and those numbers are on our public pricing page anyway.

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"keel-control/internal/billing"
	"keel-control/internal/httpx"
	"keel-control/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// tillGrantID is the singleton's _id in the tenant database.
const tillGrantID = "subscription"

// tillPlanView is one rung as the console and the restaurant's panel draw it.
type tillPlanView struct {
	ID         string   `bson:"id" json:"id"`
	Monthly    int      `bson:"monthly" json:"monthly"`
	Registers  int      `bson:"registers" json:"registers"`
	Modules    []string `bson:"modules" json:"modules"`
	Individual bool     `bson:"individual" json:"individual"`
}

// tillGrantDoc is what lands in the tenant's database.
//
// Every field here answers a question some screen in the restaurant asks. The
// ones that look redundant are not: `plan` names the rung for the panel's
// heading, while `modules` is what the server actually checks — a screen that
// re-derived the second from the first would be the second copy of the ladder
// this file exists to avoid.
//
// ⚠️ **No `_id` field, and its absence is the fix for a real failure.** It had
// one, nothing ever set it, and every save of a till plan died with "performing
// an update on the path '_id' would modify the immutable field '_id'" — Mongo
// refuses `_id` inside a `$set`, and an empty string is still a value. The
// identity is the filter's (`tillGrantID`) and belongs there alone: a document
// that names itself in its own body is a second copy of the same fact, and this
// is what the second copy cost.
type tillGrantDoc struct {
	Enabled bool   `bson:"enabled" json:"enabled"`
	Plan    string `bson:"plan" json:"plan"`
	// Resolved: the rung's modules plus anything bought on top.
	Modules []string `bson:"modules" json:"modules"`
	// 0 means no cap.
	Registers int `bson:"registers" json:"registers"`
	// ⚠️ **A date, not a flag.** The screens standing in the restaurant count
	// down to it; a stored "ok" boolean goes stale the moment the clock passes
	// it, and nobody is watching a monoblock at midnight.
	PaidUntil *time.Time `bson:"paidUntil,omitempty" json:"paidUntil,omitempty"`
	// ---- What it costs, mirrored ----
	//
	// ⚠️ **The resolved figure, not the rung's list price.** Branch discounts,
	// add-ons and a negotiated override all move it, so a panel that multiplied
	// a plan price by anything would quote a number the invoice does not agree
	// with — and the owner reading it is the person who signs the payment. It
	// comes from `TillMonthlyFor`, the same function the invoice uses, for
	// exactly that reason.
	//
	// ⚠️ 0 is meaningful and is **not** "free": it is what an Enterprise rung
	// returns, because that price is agreed per customer and inventing one here
	// would put a figure on a screen as though it had been. The panel shows the
	// plan without a price rather than a price of nothing.
	Monthly int `bson:"monthly" json:"monthly"`
	// What was bought on top of the rung, so the panel can name it. `Modules`
	// above already includes these — this is for the heading, not the check.
	Addons []string `bson:"addons" json:"addons"`
	// How many blocks of ten daily assistant requests were bought.
	AIExtra int `bson:"aiExtra" json:"aiExtra"`
	// How many branches the price was worked out over. Without it "1 250 000"
	// on a three-branch chain looks like a mistake.
	Branches int `bson:"branches" json:"branches"`
	// The ladder, so the panel can put a price on its upgrade button without
	// asking us a second question.
	Plans     []tillPlanView `bson:"plans" json:"plans"`
	UpdatedAt time.Time      `bson:"updatedAt" json:"updatedAt"`
}

type tillRequest struct {
	Enabled       bool       `json:"enabled"`
	Plan          string     `json:"plan"`
	Addons        []string   `json:"addons"`
	AIExtra       int        `json:"aiExtra"`
	Branches      int        `json:"branches"`
	PriceOverride int        `json:"priceOverride"`
	PaidUntil     *time.Time `json:"paidUntil"`
	Note          string     `json:"note"`
}

// tillView is what the console screen shows back.
type tillView struct {
	models.TenantTill
	// What this configuration costs per month, computed from the ladder — so
	// the operator sees the number before they save it rather than on the
	// invoice a month later.
	//
	// ⚠️ Zero for an Enterprise customer with no override, and the console says
	// so in words: a blank price and a free customer must not look alike.
	Monthly int            `json:"monthly"`
	Plans   []tillPlanView `json:"plans"`
}

// GetTenantTill reads the subscription as configured.
func (h *Handler) GetTenantTill(w http.ResponseWriter, r *http.Request) {
	t := h.tenantFromURL(r)
	if t == nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	httpx.JSON(w, http.StatusOK, tillViewOf(t.Till))
}

// PutTenantTill switches the counter on, moves a customer between rungs, or
// switches it off.
func (h *Handler) PutTenantTill(w http.ResponseWriter, r *http.Request) {
	t := h.tenantFromURL(r)
	if t == nil {
		httpx.Error(w, http.StatusNotFound, "topilmadi")
		return
	}
	var req tillRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now()
	next := t.Till
	next.Note = strings.TrimSpace(req.Note)
	next.UpdatedBy = currentOperator(r)
	next.UpdatedAt = &now

	if !req.Enabled {
		// ⚠️ **Switching off keeps the rung, the add-ons and the date.**
		// Wiping them would mean a customer paused for a month comes back on
		// the cheapest plan with their stock module gone — and the person who
		// switched them off would have no way to put it back exactly as it was.
		// Only the answer to "is the counter sold right now" changes.
		next.Enabled = false
		if err := h.saveTill(r, *t, next); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		httpx.JSON(w, http.StatusOK, tillViewOf(next))
		return
	}

	plan, ok := billing.PlanByID(req.Plan)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "tarif tanlanmagan")
		return
	}
	if req.Branches < 1 {
		// ⚠️ Refused rather than defaulted to 1. The branch count is the unit
		// the whole price is built on, and a silent 1 on a five-branch chain is
		// an invoice short by two thirds that nobody would query — it looks
		// like a number somebody chose.
		httpx.Error(w, http.StatusBadRequest, "filiallar soni kamida 1 bo'lishi kerak")
		return
	}
	if plan.Individual && req.PriceOverride <= 0 {
		// The ladder has no Enterprise number on purpose, so without one here
		// the customer would be invoiced zero every month — a failure that
		// looks like a working system until somebody reconciles a quarter.
		httpx.Error(w, http.StatusBadRequest,
			"Enterprise narxi kelishiladi — kelishilgan summani yozing")
		return
	}

	next.Enabled = true
	next.Plan = plan.ID
	next.Addons = cleanAddons(req.Addons)
	// ⚠️ **Blocks of ten daily AI requests, sold on top of whatever the plan
	// allows.** Kept on the same document as everything else the restaurant
	// bought, so there is one record of what they are paying for rather than
	// two that can disagree about the same month.
	next.AIExtra = cleanBlocks(req.AIExtra)
	next.Branches = req.Branches
	next.PriceOverride = req.PriceOverride
	next.PaidUntil = req.PaidUntil
	// ⚠️ Stamped only on the transition into "on", never on an edit. It is what
	// the day-counted fee is measured from, so rewriting it when somebody fixes
	// a typo in the note would silently forgive the days already owed.
	if !t.Till.Enabled || t.Till.Since == nil {
		next.Since = &now
	}

	if err := h.saveTill(r, *t, next); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, tillViewOf(next))
}

// saveTill writes both halves: what we bill from, and what the restaurant
// enforces.
//
// ⚠️ **Two independent writes, reported separately if they fail.** The
// alternative — chaining them so the mirror only happens when the tenant row
// saved — is the `else if` in `apply()` that once stopped every customer's
// domain from being written because one container would not start. Here the
// failure would be quieter still: the console would show a customer on Pro
// whose till had never been told, and the only symptom would be a restaurant
// whose second register is refused.
func (h *Handler) saveTill(r *http.Request, t models.Tenant, next models.TenantTill) error {
	if _, err := h.Store.Tenants.UpdateByID(r.Context(), t.ID, bson.M{
		"$set": bson.M{"till": next, "updatedAt": time.Now()},
	}); err != nil {
		return err
	}
	return h.mirrorTill(r.Context(), t, next)
}

// mirrorTill pushes the resolved entitlements into the tenant's own database.
func (h *Handler) mirrorTill(ctx context.Context, t models.Tenant, till models.TenantTill) error {
	doc := tillGrantDoc{
		Enabled:   till.Enabled,
		Plan:      till.Plan,
		Modules:   []string{},
		Addons:    []string{},
		Branches:  till.Branches,
		PaidUntil: till.PaidUntil,
		Plans:     planViews(),
		UpdatedAt: time.Now(),
	}
	if plan, ok := billing.PlanByID(till.Plan); ok && till.Enabled {
		doc.Registers = plan.Registers
		doc.Modules = resolveModules(plan, till.Addons)
		if till.Addons != nil {
			doc.Addons = till.Addons
		}
		// ⚠️ Through the shared helper, never recomputed here: the restaurant's
		// own screen and the invoice we send have to name the same number, and
		// the day they do not is the day an owner stops trusting both.
		doc.Monthly = tillMonthly(till)
	}
	_, err := h.Store.TenantDB(t.DBName()).Collection("subscription").
		UpdateOne(ctx, bson.M{"_id": tillGrantID},
			bson.M{"$set": doc}, options.Update().SetUpsert(true))
	return err
}

// resolveModules flattens a rung and its add-ons into the list the restaurant's
// server checks. Sorted-by-construction rather than by a sort call: the order is
// the ladder's own, which is stable and readable in a database shell.
func resolveModules(p billing.Plan, addons []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, m := range p.Modules {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	for _, m := range addons {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

// cleanAddons drops anything that is not a module somebody can buy.
//
// ⚠️ An unpriced module id would be stored, mirrored, and enforced as a
// permission — bought for nothing. The allowlist is `AddonPrice` itself, so a
// module becomes sellable at exactly the moment it has a price.
func cleanAddons(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, m := range in {
		m = strings.ToLower(strings.TrimSpace(m))
		if m == "" || seen[m] || billing.AddonPrice(m) <= 0 {
			continue
		}
		seen[m] = true
		out = append(out, m)
	}
	return out
}

func planViews() []tillPlanView {
	out := []tillPlanView{}
	for _, p := range billing.Plans() {
		mods := p.Modules
		if mods == nil {
			// ⚠️ An empty slice, not nil: Go marshals a nil slice as `null`,
			// and the panel reads `.length` off this to draw the plan card.
			// The trap this codebase has now been bitten by twice.
			mods = []string{}
		}
		out = append(out, tillPlanView{
			ID: p.ID, Monthly: p.Monthly, Registers: p.Registers,
			Modules: mods, Individual: p.Individual,
		})
	}
	return out
}

func tillViewOf(t models.TenantTill) tillView {
	v := tillView{TenantTill: t, Plans: planViews()}
	if t.Addons == nil {
		v.Addons = []string{}
	}
	if t.PriceOverride > 0 {
		v.Monthly = t.PriceOverride
	} else if p, ok := billing.PlanByID(t.Plan); ok {
		v.Monthly = billing.TillMonthly(p, t.Branches, t.Addons, t.AIExtra)
	}
	return v
}

// TillMonthlyFor is what this customer's counter costs per month, for the
// invoice. Shared with the console view so a bill can never disagree with the
// screen the operator agreed it on.
func TillMonthlyFor(t models.Tenant) int { return tillMonthly(t.Till) }

// tillMonthly is the same answer from the subscription alone.
//
// ⚠️ Split out because `mirrorTill` is handed the **new** subscription while
// the tenant document still holds the old one — pricing `t.Till` there would
// mirror the figure the customer was on before the operator pressed save, and
// the restaurant's own screen would quote last month's price until something
// else happened to rewrite it.
func tillMonthly(till models.TenantTill) int {
	if !till.Enabled {
		return 0
	}
	if till.PriceOverride > 0 {
		return till.PriceOverride
	}
	p, ok := billing.PlanByID(till.Plan)
	if !ok {
		return 0
	}
	return billing.TillMonthly(p, till.Branches, till.Addons, till.AIExtra)
}

// cleanBlocks bounds what the console may sell.
//
// ⚠️ **A ceiling, because this is a number typed into a box and every block is
// real money we spend.** Somebody meaning ten and holding a key means a hundred
// blocks — a thousand requests a day for one restaurant, paid by us until an
// invoice says otherwise. Fifty is far past any real restaurant and far short
// of an accident.
func cleanBlocks(n int) int {
	switch {
	case n < 0:
		// A negative is a typo, not a way to take a plan's own allowance away.
		return 0
	case n > 50:
		return 50
	}
	return n
}

// SyncTillGrants rewrites every restaurant's copy of what it has bought.
//
// ⚠️ **A mirror that is only written when somebody presses save is a mirror
// that goes stale, and this one did.** The extra assistant blocks were added to
// the monthly price in code; every existing customer went on being shown — and
// invoiced from — the figure that had been mirrored months earlier, and the
// only way to correct one was for an operator to open that tenant and press
// save with nothing changed. Nobody was ever going to do that fifty times.
//
// So it is rewritten from its one source on a schedule, the same argument
// `SyncEdge` is built on: a document regenerated on a tick cannot drift away
// from what generates it. When nothing has moved the write is byte-identical
// and costs a no-op update per tenant, once an hour.
func (h *Handler) SyncTillGrants(ctx context.Context) {
	cur, err := h.Store.Tenants.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("till grants: %v", err)
		return
	}
	var tenants []models.Tenant
	if err := cur.All(ctx, &tenants); err != nil {
		log.Printf("till grants: %v", err)
		return
	}
	for _, t := range tenants {
		// ⚠️ Deleted tenants are skipped and suspended ones are not: a
		// suspended restaurant is one we switched off and may switch back on,
		// and its panel should be truthful about what it is paying for when it
		// returns.
		if t.Status == models.StatusDeleted {
			continue
		}
		if err := h.mirrorTill(ctx, t, t.Till); err != nil {
			// One unreachable database must not stop the other forty-nine.
			log.Printf("till grants %s: %v", t.Slug, err)
		}
	}
}
