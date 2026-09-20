package handlers

// ---- The campaign plan: what to advertise, from this kitchen's own week ----
//
// The half of the advertising section that works without Meta. Connecting an ad
// account needs an app Meta has reviewed; deciding *what* to advertise needs
// only what this server already knows, and it is the part a $500-a-month
// targetolog cannot do at all — they see clicks, this sees which dish carried
// the week and how far the van goes.
//
// ⚠️ **The numbers are computed here, the words come from the model.** Shares,
// totals and comparisons are arithmetic done in this file and sent as facts; the
// plan that comes back is sentences about them. The panel draws its bars from
// the same facts rather than from the plan, so what an owner reads as "41% of
// the week" is measured — and a model that got carried away cannot move a bar.
//
// ⚠️ **The plan proposes and the owner picks** — three dishes, a few areas,
// three budgets, three pieces of wording, each with its reason. This is what the
// section was sold as, and it is also the only honest shape: a machine that
// picked one would be spending somebody else's money on its own judgement.
//
// ⚠️ **One plan a day per lens.** The add-on is sold with thirty requests a day
// and a plan is the expensive one; a week's sales do not change between two
// presses of a button, so a second press inside the same day is answered from
// what was already paid for. The screen says which morning it is as of.

import (
	"context"
	"net/http"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The window the plan thinks in, and the one it compares against.
//
// ⚠️ **A week against the week before**, the same span the briefing uses. A
// month smooths away exactly what an advert is for — the dish that started
// moving — and a day is one delivery van breaking down.
const adsWeek = 7 * 24 * time.Hour

// How many dishes travel as candidates.
//
// ⚠️ **Five, when the plan returns three.** The model needs something to reject
// with a reason; handed exactly three it has no choice to make and the reasons
// become decoration.
const adsDishRows = 5

// adsPlanShape is bumped whenever the plan's structure changes.
//
// v2: every wording carries the dish it advertises, and each dish carries
// whether the restaurant has a photograph of it.
const adsPlanShape = "v2"

// storedAdsPlan is one day's plan for one lens.
type storedAdsPlan struct {
	ID    primitive.ObjectID `bson:"_id,omitempty"`
	Day   string             `bson:"day"`
	Scope string             `bson:"scope"`
	// The figures the plan was written from — kept because the panel draws its
	// bars from them, and because a plan whose numbers are gone cannot be
	// checked by the owner who is about to act on it.
	Facts  map[string]any `bson:"facts"`
	Plan   map[string]any `bson:"plan"`
	MadeAt time.Time      `bson:"madeAt"`
}

// AdminAdsPlan answers "what should I advertise?" for this restaurant.
func (h *Handler) AdminAdsPlan(w http.ResponseWriter, r *http.Request) {
	// ⚠️ Owner only, like the rest of the section: this plan ends in a campaign
	// that spends money, and it reads the whole menu's takings on the way.
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx := r.Context()
	// Local, like the advisor's day: a day that turns over at five in the
	// morning Tashkent time would change the plan during the night shift's close.
	day := time.Now().In(time.Local).Format("2006-01-02")
	// ⚠️ **The shape of the plan is part of the key.** A day's plan is cached
	// so a second press does not pay for the same week twice — which also
	// means a release that changes what the plan *contains* would go on
	// serving yesterday's shape until midnight, and the change would look like
	// it had not deployed. Bumping this is how a shape change reaches the
	// screen the moment it ships.
	key := scopeKey(scope) + "#" + adsPlanShape

	// ---- Already planned this morning ----
	var have storedAdsPlan
	if h.Store.AdsPlans.FindOne(ctx, bson.M{"day": day, "scope": key}).
		Decode(&have) == nil && len(have.Plan) > 0 {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"plan": have.Plan, "facts": have.Facts,
			"asOf": have.MadeAt, "cached": true,
		})
		return
	}

	facts, ok := h.adsFacts(ctx, r, scope)
	if !ok {
		// ⚠️ **Refused rather than planned badly.** A restaurant with no week of
		// sales behind it would get a plan written from nothing, and it would
		// read exactly like a plan written from something.
		httpx.Error(w, http.StatusBadRequest,
			"reklama rejasi uchun bir haftalik sotuv kerak")
		return
	}

	out, err := h.callControlPath(ctx, "/internal/ads-advice", map[string]any{
		"lang":  reportLang(r),
		"facts": facts,
	})
	if err != nil {
		// The platform's own sentence rather than a status code — "bugungi limit
		// tugadi" is something an owner can act on and `502` is not.
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	if off, _ := out["off"].(bool); off {
		httpx.JSON(w, http.StatusOK, map[string]any{"off": true})
		return
	}
	if ok, present := out["entitled"].(bool); present && !ok {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"entitled": false, "monthly": out["monthly"],
		})
		return
	}
	if capped, _ := out["capped"].(bool); capped {
		httpx.JSON(w, http.StatusOK, map[string]any{"capped": true, "cap": out["cap"]})
		return
	}
	if msg, _ := out["error"].(string); msg != "" {
		httpx.JSON(w, http.StatusOK, map[string]any{"error": msg})
		return
	}
	plan, _ := out["plan"].(map[string]any)
	if len(plan) == 0 {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"error": httpx.T(w, "reja kelmadi"),
		})
		return
	}

	made := time.Now()
	// ⚠️ Upsert on the same (day, scope) the index is unique on: two tabs open
	// in the same second both miss and both plan, and a second row would leave
	// `FindOne` choosing — which the owner meets as the plan changing while they
	// read it.
	_, _ = h.Store.AdsPlans.UpdateOne(ctx,
		bson.M{"day": day, "scope": key},
		bson.M{"$set": bson.M{
			"day": day, "scope": key,
			"facts": facts, "plan": plan, "madeAt": made,
		}}, options.Update().SetUpsert(true))

	httpx.JSON(w, http.StatusOK, map[string]any{
		"plan": plan, "facts": facts, "asOf": made, "cached": false,
	})
}

// adsFacts is everything the plan is allowed to be written from.
//
// ⚠️ **Aggregates and the branch's own address — no guest ever appears here.**
// An advertising plan needs to know what sells and how far the kitchen reaches;
// it has no use for who bought it, and a payload that carried customers would
// put a customer list on the platform for the sake of a sentence about osh.
func (h *Handler) adsFacts(
	ctx context.Context, r *http.Request, scope Scope,
) (map[string]any, bool) {
	filter := bson.M{}
	if !scope.BrandID.IsZero() {
		filter["brandId"] = scope.BrandID
	}
	branch := scope.BranchID
	if branch.IsZero() {
		// The same resolution the advisor makes: half of this needs one branch,
		// and the owner of a one-branch restaurant never made that choice
		// because there was none to make.
		if only, err := h.onlyBranch(r, scope.BrandID); err == nil {
			branch = only
		}
	}
	if !branch.IsZero() {
		filter["branchId"] = branch
	}

	now := time.Now()
	weekFrom := now.Add(-adsWeek)
	prevFrom := now.Add(-2 * adsWeek)

	top, err := h.dishSales(ctx, filter, weekFrom, now, adsDishRows)
	if err != nil || len(top) == 0 {
		return nil, false
	}
	total, err := h.revenueBetween(ctx, filter, weekFrom, now)
	if err != nil || total == 0 {
		return nil, false
	}
	prevTotal, _ := h.revenueBetween(ctx, filter, prevFrom, weekFrom)

	// Last week by dish, so a riser can be named as one.
	was := map[string]int64{}
	if prev, err := h.dishSales(ctx, filter, prevFrom, weekFrom, 50); err == nil {
		for _, d := range prev {
			was[d.ID] = d.Money
		}
	}

	// ⚠️ **Which dishes can actually be advertised.** A Meta advert cannot be
	// created without a picture and we never invent one — so a plan that
	// proposed a dish nobody has photographed sent the owner all the way to the
	// launch button before anything said so, and what it said there was "no
	// photograph for this dish", four screens away from the choice that caused
	// it. The planner is told, and the panel marks it at the moment of choosing.
	photos := h.dishPhotos(ctx, top)

	dishes := make([]map[string]any, 0, len(top))
	for _, d := range top {
		dishes = append(dishes, map[string]any{
			// ⚠️ **The id travels with the name.** The campaign used to find
			// the dish by its name, which is the one field a menu screen lets
			// somebody rename — and a rename between planning and launching
			// turned into "dish not found" at the button that spends money.
			// The model never sees this; it is passed through for the panel.
			"id":    d.ID,
			"name":  d.Name,
			"qty":   d.Qty,
			"money": d.Money,
			"photo": photos[d.ID],
			// ⚠️ Computed here, in integers, and sent as a fact. The share is
			// the number the whole plan argues from and the one the panel draws
			// a bar from — it is not a thing to ask a model for.
			"share":    d.Money * 100 / total,
			"lastWeek": was[d.ID],
		})
	}

	facts := map[string]any{
		"currency":      "UZS",
		"weekTotal":     total,
		"prevWeekTotal": prevTotal,
		"dishes":        dishes,
	}

	// ---- How far this kitchen reaches ----
	var b models.Branch
	if !branch.IsZero() &&
		h.Store.Branches.FindOne(ctx, bson.M{"_id": branch}).Decode(&b) == nil {
		reach := map[string]any{
			"address":    b.Address.Text,
			"deliveryOn": b.Delivery.Enabled,
		}
		// ⚠️ **Only when delivery is actually on.** `maxKm` on a branch that
		// does not deliver is a leftover setting, and an advert aimed at a
		// radius nobody drives is money spent on people who cannot order.
		if b.Delivery.Enabled && b.Delivery.MaxKm > 0 {
			reach["maxKm"] = b.Delivery.MaxKm
		}
		hours := make([]map[string]any, 0, len(b.WorkingHours))
		for _, wh := range b.WorkingHours {
			hours = append(hours, map[string]any{
				"day": wh.Day, "open": wh.Open, "close": wh.Close,
				"closed": wh.IsClosed,
			})
		}
		reach["workingHours"] = hours
		facts["reach"] = reach
	}

	if h.Store.Menu != nil {
		brandFilter := bson.M{}
		if !scope.BrandID.IsZero() {
			brandFilter["brandId"] = scope.BrandID
		}
		if n, err := h.Store.Menu.CountDocuments(ctx, brandFilter); err == nil {
			facts["menuItems"] = n
		}
	}
	return facts, true
}

// dishPhotos says, for each dish in the week, whether there is a photograph of
// it on this server.
//
// ⚠️ **One query for the lot.** This runs inside the plan, which an owner is
// already waiting on a model for; a lookup per dish would be five round trips
// to say one word each.
func (h *Handler) dishPhotos(
	ctx context.Context, rows []dishSale,
) map[string]bool {
	out := map[string]bool{}
	ids := make([]primitive.ObjectID, 0, len(rows))
	for _, d := range rows {
		if id, err := primitive.ObjectIDFromHex(d.ID); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return out
	}
	cur, err := h.Store.Menu.Find(ctx, bson.M{"_id": bson.M{"$in": ids}},
		options.Find().SetProjection(bson.M{"imageUrl": 1, "images": 1}))
	if err != nil {
		return out
	}
	var items []models.MenuItem
	if cur.All(ctx, &items) != nil {
		return out
	}
	for _, it := range items {
		out[it.ID.Hex()] = strings.TrimSpace(it.ImageURL) != "" ||
			len(it.Images) > 0
	}
	return out
}
