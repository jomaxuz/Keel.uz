package handlers

// ---- The advisor: the owner's own questions, answered from their own figures ----
//
// The morning briefing decides for the owner what matters today. This is the
// other half of the same seam: the owner asks, and the answer is built from
// exactly the same kind of material — figures this system computed, never
// figures a model derived. `insight`'s rule is the rule here too, and it is
// worth restating because it is the whole reason this can be trusted: **the
// numbers are ours, only the words are the model's.**
//
// ⚠️ **It answers about this restaurant and about Keel, and refuses everything
// else.** Not out of primness — a general-purpose chat box in a restaurant
// panel is a support burden with no ceiling, and the first time it confidently
// explains Uzbek tax law to somebody who acts on it, we own the consequence.
// The refusal lives in the prompt (the control plane's advisor.go) and in what
// we send: a model given only this restaurant's aggregates has very little to
// be wrong about.
//
// ⚠️ **Nothing personal leaves this server, and that is a property of the
// payload rather than a promise.** A lapsed guest travels as `c3`, four
// numbers and nothing else; the model writes "c3" and *this* server puts the
// name back before the browser sees it. So the advice can be specific — "these
// four regulars are worth chasing first" — while the control plane holds no
// customer list at all. See `aliasOf` and `resolveAliases`.
//
// ⚠️ **The snapshot is built once a day per lens**, like the briefing and for
// one more reason besides cost: an owner who asks the same question twice
// before lunch and gets two different answers stops believing both. Advice is
// "as of this morning", and the screen says so.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/insight"
	"restaurant-backend/internal/middleware"
	"restaurant-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// How much of a conversation travels with the next question.
//
// ⚠️ **Four turns, not the whole thread.** Every previous turn is tokens paid
// for again on every question, and an advisor conversation is short by nature:
// a question, a follow-up, maybe a "and what about deliveries". Past that the
// owner has moved on to another subject, and carrying the first one is paying
// to confuse the answer.
const advisorHistoryTurns = 4

// advisorMaxQuestion is where a question stops being a question.
const advisorMaxQuestion = 500

// How many lapsed regulars travel as rows rather than as a count.
//
// ⚠️ **A sample, and the number is a cost decision.** The count already tells
// the owner the size of the problem; the rows exist so the answer can say which
// ones to chase first. Twenty is more than anybody rings in a morning.
const advisorGuestRows = 20

// storedSnapshot is one day's picture of the business, for one lens.
//
// ⚠️ **`Aliases` never leaves this server.** It is the half of the snapshot
// that makes the other half safe to send.
type storedSnapshot struct {
	ID    primitive.ObjectID `bson:"_id,omitempty"`
	Day   string             `bson:"day"`
	Scope string             `bson:"scope"`
	// What the control plane is allowed to see: keys, areas and figures.
	Facts   []insight.Fact `bson:"facts"`
	Profile map[string]any `bson:"profile"`
	// `c3` → the name this restaurant knows that person by.
	Aliases map[string]string `bson:"aliases"`
	MadeAt  time.Time         `bson:"madeAt"`
}

// storedAnswer is one question already paid for.
//
// ⚠️ **Keyed by the question, the day and the lens** — the three things that
// decide the answer. Two managers asking "nega tushum tushdi?" on the same
// morning are asking one question, and the second one should not cost anything.
type storedAnswer struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Key      string             `bson:"key"`
	Day      string             `bson:"day"`
	Question string             `bson:"question"`
	Answer   string             `bson:"answer"`
	MadeAt   time.Time          `bson:"madeAt"`
}

// advisorRole reports whether this session may ask.
//
// ⚠️ **Owner and manager, and the two limited roles never reach here anyway**
// (panelgate.go refuses a path off their list). Written out rather than left to
// the gate: this endpoint reads the whole business — takings, wages, waste —
// and a future edit to that list must not be the only thing standing between a
// call-centre operator and the payroll.
func advisorRole(r *http.Request) bool {
	claims := middleware.ClaimsFrom(r.Context())
	return claims != nil && (claims.Role == "owner" || claims.Role == "manager")
}

// AdminAdvisorAsk answers one question about this restaurant.
func (h *Handler) AdminAdvisorAsk(w http.ResponseWriter, r *http.Request) {
	if _, err := h.adminUser(r); err != nil {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	if !advisorRole(r) {
		httpx.Error(w, http.StatusForbidden, "bu bo'lim faqat egasi va menejer uchun")
		return
	}
	var req struct {
		Question string `json:"question"`
		// The turns before this one, as the screen has them. ⚠️ Read from the
		// browser rather than stored here: the conversation is one person's own
		// screen, it is worth nothing tomorrow, and a server-side thread would
		// be a second place for the same words to live.
		History []struct {
			Question string `json:"question"`
			Answer   string `json:"answer"`
		} `json:"history"`
	}
	if httpx.Decode(r, &req) != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	question := clampText(strings.TrimSpace(req.Question), advisorMaxQuestion)
	if question == "" {
		httpx.Error(w, http.StatusBadRequest, "savol yozing")
		return
	}
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx := r.Context()
	lang := reportLang(r)
	// ⚠️ Local, like the briefing's: a day that turns over at five in the
	// morning Tashkent time changes the answer during the night shift's close.
	day := time.Now().In(time.Local).Format("2006-01-02")
	key := scopeKey(scope)

	snap, err := h.advisorSnapshot(ctx, r, scope, day, key)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// ---- Already answered this morning ----
	//
	// ⚠️ **Only when there is no history.** A follow-up ("va yetkazib berishda
	// nima bo'ldi?") repeats none of its own context, so two follow-ups with the
	// same words after different questions are different questions — and a cache
	// that could not tell them apart would answer the second with the first.
	cacheKey := advisorKey(day, key, lang, question)
	if len(req.History) == 0 {
		var hit storedAnswer
		if h.Store.AdvisorAnswers.FindOne(ctx, bson.M{"key": cacheKey}).
			Decode(&hit) == nil && hit.Answer != "" {
			httpx.JSON(w, http.StatusOK, map[string]any{
				"answer": resolveAliases(hit.Answer, snap.Aliases),
				"asOf":   snap.MadeAt,
				"cached": true,
			})
			return
		}
	}

	history := req.History
	if len(history) > advisorHistoryTurns {
		history = history[len(history)-advisorHistoryTurns:]
	}
	turns := make([]map[string]string, 0, len(history))
	for _, t := range history {
		turns = append(turns, map[string]string{
			"question": clampText(t.Question, advisorMaxQuestion),
			"answer":   clampText(t.Answer, 1500),
		})
	}

	out, err := h.callControlPath(ctx, "/internal/advise", map[string]any{
		"lang":     lang,
		"question": question,
		"history":  turns,
		"facts":    snap.Facts,
		"profile":  snap.Profile,
	})
	if err != nil {
		// ⚠️ The platform's own sentence, not a status code: "bugungi limit
		// tugadi" is something an owner can act on, and `502` is not.
		httpx.Error(w, http.StatusBadGateway, err.Error())
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
	answer, _ := out["answer"].(string)
	refused, _ := out["refused"].(bool)
	if answer == "" {
		httpx.JSON(w, http.StatusOK, map[string]any{"error": httpx.T(w, "javob kelmadi")})
		return
	}
	// ⚠️ **A refusal is never cached.** "Men bunga javob bera olmayman" is about
	// the question being outside this panel, which does not change during the
	// day — but neither does it cost anything to repeat, and a cached refusal
	// would survive the fix if the refusal turns out to be wrong.
	if !refused && len(req.History) == 0 {
		h.rememberAnswer(ctx, cacheKey, day, question, answer)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		// ⚠️ The names go back in **here**, on the restaurant's own server. The
		// control plane wrote `c3` because `c3` is all it was given.
		"answer":  resolveAliases(answer, snap.Aliases),
		"refused": refused,
		"asOf":    snap.MadeAt,
	})
}

// scopeKey is the lens, spelled the way the briefing spells it.
func scopeKey(s Scope) string {
	return s.BrandID.Hex() + "/" + s.BranchID.Hex()
}

// advisorKey is the cache key: the day, the lens, the language and the question.
//
// ⚠️ **The question is normalised first**, or "Nega tushum tushdi?" and "nega
// tushum tushdi" are two questions and the cache halves in value for nothing.
func advisorKey(day, scope, lang, question string) string {
	norm := strings.Join(strings.Fields(strings.ToLower(question)), " ")
	norm = strings.Trim(norm, "?!. ")
	sum := sha256.Sum256([]byte(day + "|" + scope + "|" + lang + "|" + norm))
	return hex.EncodeToString(sum[:])
}

func (h *Handler) rememberAnswer(ctx context.Context, key, day, question, answer string) {
	_, _ = h.Store.AdvisorAnswers.UpdateOne(ctx, bson.M{"key": key}, bson.M{
		"$set": bson.M{
			"key": key, "day": day,
			"question": clampText(question, advisorMaxQuestion),
			"answer":   clampText(answer, 4000),
			"madeAt":   time.Now(),
		},
	}, options.Update().SetUpsert(true))
}

// ---- The snapshot ----

// advisorSnapshot is today's picture of the business, built once.
func (h *Handler) advisorSnapshot(
	ctx context.Context, r *http.Request, scope Scope, day, key string,
) (storedSnapshot, error) {
	var have storedSnapshot
	if err := h.Store.AdvisorSnapshots.FindOne(ctx,
		bson.M{"day": day, "scope": key}).Decode(&have); err == nil {
		return have, nil
	}

	filter := bson.M{}
	if !scope.BrandID.IsZero() {
		filter["brandId"] = scope.BrandID
	}
	branch := scope.BranchID
	if branch.IsZero() {
		// ⚠️ The same resolution the briefing makes, for the same reason: half
		// the facts need one branch, and an owner of a one-branch restaurant
		// never made that choice because there was none to make.
		if only, err := h.onlyBranch(r, scope.BrandID); err == nil {
			branch = only
		}
	}
	if !branch.IsZero() {
		filter["branchId"] = branch
	}

	snap := storedSnapshot{
		Day: day, Scope: key,
		Facts:   h.gatherFacts(ctx, filter),
		Profile: h.advisorProfile(ctx, scope, branch),
		Aliases: map[string]string{},
		MadeAt:  time.Now(),
	}
	guests, aliases := h.lapsedSample(ctx, filter)
	if len(guests) > 0 {
		snap.Profile["lapsedGuests"] = guests
		snap.Aliases = aliases
	}
	if snap.Facts == nil {
		snap.Facts = []insight.Fact{}
	}
	// ⚠️ Upsert on the same (day, scope) the index is unique on: two managers
	// opening the panel in the same second both miss and both build, and the
	// second insert would otherwise leave `FindOne` picking between two
	// snapshots — which an owner meets as the advice changing between questions.
	_, err := h.Store.AdvisorSnapshots.UpdateOne(ctx,
		bson.M{"day": day, "scope": key},
		bson.M{"$set": bson.M{
			"day": day, "scope": key, "facts": snap.Facts,
			"profile": snap.Profile, "aliases": snap.Aliases, "madeAt": snap.MadeAt,
		}}, options.Update().SetUpsert(true))
	return snap, err
}

// advisorProfile is the shape of the business: how big, how many, what hours.
//
// ⚠️ **Counts, not lists.** "Forty-one dishes, nine of them without a recipe"
// is what changes advice; the forty-one names would be tokens spent on
// something the owner can already read off their own menu screen.
func (h *Handler) advisorProfile(
	ctx context.Context, scope Scope, branch primitive.ObjectID,
) map[string]any {
	out := map[string]any{}
	count := func(name string, c *mongo.Collection, filter bson.M) {
		if c == nil {
			return
		}
		n, err := c.CountDocuments(ctx, filter)
		if err == nil {
			out[name] = n
		}
	}
	brandFilter := bson.M{}
	if !scope.BrandID.IsZero() {
		brandFilter["brandId"] = scope.BrandID
	}
	branchFilter := bson.M{}
	if !branch.IsZero() {
		branchFilter["branchId"] = branch
	}

	count("menuItems", h.Store.Menu, brandFilter)
	count("categories", h.Store.Categories, brandFilter)
	count("ingredients", h.Store.Ingredients, brandFilter)
	count("suppliers", h.Store.Suppliers, brandFilter)
	count("warehouses", h.Store.Warehouses, branchFilter)
	count("branches", h.Store.Branches, brandFilter)
	count("staff", h.Store.Staff, mergeFilter(branchFilter, bson.M{"isActive": true}))
	count("couriers", h.Store.Couriers, branchFilter)

	// ⚠️ **Dishes with no recipe, because it is the number that decides whether
	// any cost advice is worth giving at all.** A restaurant whose menu is half
	// uncosted cannot be told anything true about margin, and an assistant that
	// answered anyway would be inventing the half it cannot see.
	if h.Store.Menu != nil {
		n, err := h.Store.Menu.CountDocuments(ctx,
			mergeFilter(brandFilter, bson.M{"recipe": bson.M{"$in": bson.A{nil, []any{}}}}))
		if err == nil {
			out["dishesWithoutRecipe"] = n
		}
	}

	// The hours, so "the quiet hour" can be read against when the doors are
	// actually open.
	var b models.Branch
	if !branch.IsZero() &&
		h.Store.Branches.FindOne(ctx, bson.M{"_id": branch}).Decode(&b) == nil {
		hours := make([]map[string]any, 0, len(b.WorkingHours))
		for _, wh := range b.WorkingHours {
			hours = append(hours, map[string]any{
				"day": wh.Day, "open": wh.Open, "close": wh.Close,
				"closed": wh.IsClosed,
			})
		}
		out["workingHours"] = hours
		out["deliveryOn"] = b.Delivery.Enabled
		out["prepMinutes"] = b.PrepMinutes
	}
	return out
}

func mergeFilter(a, b bson.M) bson.M {
	out := bson.M{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// lapsedSample is the regulars who stopped coming, as rows the model may point
// at — and as aliases only this server can read back.
//
// ⚠️ **`c3`, never a name and never a phone.** The count of lapsed regulars is
// already a fact; what the owner actually wants from an assistant is "ring
// these first", and that needs rows. Sending the rows pseudonymously is what
// lets the answer be specific without the platform holding a customer list:
// the substitution happens in `resolveAliases`, on this server, on the way
// back.
func (h *Handler) lapsedSample(
	ctx context.Context, scope bson.M,
) ([]map[string]any, map[string]string) {
	match := bson.M{
		"status": models.StatusDelivered,
		"userId": bson.M{"$ne": primitive.NilObjectID},
	}
	for k, v := range scope {
		match[k] = v
	}
	cutoff := time.Now().Add(-sleepingDays * 24 * time.Hour)
	cur, err := h.Store.Orders.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$userId",
			"n":     bson.M{"$sum": 1},
			"spend": bson.M{"$sum": "$total"},
			"last":  bson.M{"$max": "$createdAt"},
		}}},
		{{Key: "$match", Value: bson.M{
			"n":    bson.M{"$gte": regularOrders},
			"last": bson.M{"$lt": cutoff},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "spend", Value: -1}}}},
		{{Key: "$limit", Value: advisorGuestRows}},
	})
	if err != nil {
		return nil, nil
	}
	defer cur.Close(ctx)

	type row struct {
		ID    primitive.ObjectID `bson:"_id"`
		N     int                `bson:"n"`
		Spend int64              `bson:"spend"`
		Last  time.Time          `bson:"last"`
	}
	var rows []row
	if cur.All(ctx, &rows) != nil || len(rows) == 0 {
		return nil, nil
	}

	ids := make([]primitive.ObjectID, 0, len(rows))
	for _, x := range rows {
		ids = append(ids, x.ID)
	}
	names := map[primitive.ObjectID]string{}
	if c, err := h.Store.Users.Find(ctx, bson.M{"_id": bson.M{"$in": ids}}); err == nil {
		defer c.Close(ctx)
		var users []models.User
		if c.All(ctx, &users) == nil {
			for _, u := range users {
				names[u.ID] = strings.TrimSpace(u.FirstName + " " + u.LastName)
			}
		}
	}

	out := make([]map[string]any, 0, len(rows))
	aliases := map[string]string{}
	for i, x := range rows {
		alias := aliasOf(i)
		name := names[x.ID]
		if name == "" {
			// ⚠️ A guest who never gave a name still gets an alias: the row is
			// what makes the advice specific, and the panel can open the card
			// by id even when there is nothing to call them.
			name = alias
		}
		aliases[alias] = name
		out = append(out, map[string]any{
			"id":     alias,
			"orders": x.N,
			"spend":  x.Spend,
			// Days rather than a date: a date would be one more thing that has
			// to be read in the right timezone, and "84 kun" is the sentence
			// anyway.
			"quietDays": int(time.Since(x.Last).Hours() / 24),
		})
	}
	return out, aliases
}

// aliasOf is the name a guest travels under.
func aliasOf(i int) string { return "c" + strconv.Itoa(i+1) }

// resolveAliases puts the real names back into an answer.
//
// ⚠️ **Longest alias first.** `c1` is a prefix of `c12`, and a shortest-first
// walk turns "c12" into "Dilnoza2" — a name that is not anybody's, in a
// sentence that otherwise reads correctly.
func resolveAliases(text string, aliases map[string]string) string {
	if len(aliases) == 0 {
		return text
	}
	keys := make([]string, 0, len(aliases))
	for k := range aliases {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, k := range keys {
		if aliases[k] != "" && aliases[k] != k {
			text = strings.ReplaceAll(text, k, aliases[k])
		}
	}
	return text
}

// AdminAdvisorState is what the screen needs before anybody has asked anything.
//
// ⚠️ **So the tab can say what it knows about, and when.** An empty chat box in
// a restaurant panel is a box nobody types in: it gives no clue what it can
// answer, and the first question is therefore usually one it has to refuse.
// This hands the panel the fact keys the snapshot holds and the moment it was
// taken, and the panel turns them into the two or three suggestions under the
// box.
func (h *Handler) AdminAdvisorState(w http.ResponseWriter, r *http.Request) {
	if _, err := h.adminUser(r); err != nil {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	if !advisorRole(r) {
		httpx.Error(w, http.StatusForbidden, "bu bo'lim faqat egasi va menejer uchun")
		return
	}
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if h.Cfg.ControlURL == "" || h.Cfg.ControlToken == "" {
		// A self-hosted restaurant with no platform behind it has no assistant,
		// and the tab draws nothing rather than a box that always fails.
		httpx.JSON(w, http.StatusOK, map[string]any{"on": false})
		return
	}
	ctx := r.Context()
	day := time.Now().In(time.Local).Format("2006-01-02")
	snap, err := h.advisorSnapshot(ctx, r, scope, day, scopeKey(scope))
	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"on": true, "areas": []string{}})
		return
	}
	areas := []string{}
	seen := map[string]bool{}
	for _, f := range snap.Facts {
		if a := string(f.Area); a != "" && !seen[a] {
			seen[a] = true
			areas = append(areas, a)
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"on":    true,
		"areas": areas,
		"asOf":  snap.MadeAt,
		// ⚠️ Never nil — the trap this codebase has been bitten by twice, and
		// this list is empty on a restaurant that opened last week.
		"facts": factKeys(snap.Facts),
	})
}

func factKeys(facts []insight.Fact) []string {
	out := []string{}
	for _, f := range facts {
		out = append(out, f.Key)
	}
	return out
}
