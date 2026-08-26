package handlers

// ---- The briefing, as the panel asks for it ----
//
// ⚠️ **Built once a day and then read from the database.** An owner opens the
// dashboard several times before lunch, and a card that rewrote itself on every
// visit would be a different set of priorities each time — which reads as the
// assistant changing its mind rather than as the restaurant changing. It is
// also, incidentally, what stops one open tab from spending the platform's
// money in a loop.

import (
	"context"
	"net/http"
	"time"

	"restaurant-backend/internal/httpx"
	"restaurant-backend/internal/insight"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// storedBriefing is one morning's cards for one lens onto the business.
type storedBriefing struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Day      string             `bson:"day"`
	Scope    string             `bson:"scope"`
	Lang     string             `bson:"lang"`
	Cards    []insight.Card     `bson:"cards"`
	MadeAt   time.Time          `bson:"madeAt"`
	Attempts int                `bson:"attempts"`
}

// AdminInsights is what this restaurant should look at today.
func (h *Handler) AdminInsights(w http.ResponseWriter, r *http.Request) {
	if _, err := h.adminUser(r); err != nil {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	scope, err := h.adminScope(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx := r.Context()
	// ⚠️ The same resolution order every report already uses: the panel says so
	// explicitly, the cookie is the fallback, Uzbek is the base. A second rule
	// here would eventually disagree with the reports on the same screen.
	lang := reportLang(r)
	// ⚠️ The day is local, not UTC. A briefing keyed on a UTC date turns over
	// at five in the morning in Tashkent — during the night shift's close, and
	// four hours before anybody reads it.
	day := time.Now().In(time.Local).Format("2006-01-02")
	key := bson.M{
		"day": day, "lang": lang,
		"scope": scope.BrandID.Hex() + "/" + scope.BranchID.Hex(),
	}

	var have storedBriefing
	if err := h.Store.Briefings.FindOne(ctx, key).Decode(&have); err == nil {
		httpx.JSON(w, http.StatusOK, briefingJSON(have))
		return
	}

	filter := bson.M{}
	if !scope.BrandID.IsZero() {
		filter["brandId"] = scope.BrandID
	}
	branch := scope.BranchID
	if branch.IsZero() {
		// ⚠️ **One branch means there was never a choice to make, and half the
		// briefing depended on somebody making it.**
		//
		// The stock facts need a single branch — a shortfall spread across
		// three fridges is a number nobody can act on, the rule the whole stock
		// module is built on. But an owner of a one-branch restaurant is pinned
		// to no branch, so their scope arrived as zero and those facts were
		// skipped in silence. The two most reliable things the briefing has to
		// say — an uncounted store and an unexplained shortfall — never
		// appeared, and the panel drew nothing.
		//
		// Resolved the way every stock screen already resolves it. A real chain
		// still gets nothing here, which is correct: they pick a branch.
		if only, err := h.onlyBranch(r, scope.BrandID); err == nil {
			branch = only
		}
	}
	if !branch.IsZero() {
		filter["branchId"] = branch
	}
	facts := h.gatherFacts(ctx, filter)
	if len(facts) == 0 {
		// ⚠️ **Not stored, and the first version storing it was a day-long
		// mistake.**
		//
		// The reasoning was that an empty answer should not be recomputed all
		// morning. But gathering facts is a handful of Mongo queries — the
		// expensive part is the model, which is not reached here at all — and
		// writing "nothing" into the day's slot meant a restaurant that turned
		// the feature on at nine saw an empty panel until the following
		// morning, with nothing anywhere saying why. Which is exactly what
		// happened on the day this shipped.
		httpx.JSON(w, http.StatusOK, map[string]any{"cards": []insight.Card{}})
		return
	}

	cards, state := h.writeBriefing(ctx, facts, lang)
	// ⚠️ **Only a real answer is kept for the day.** A restaurant whose plan
	// does not include the assistant, or one that hit today's cap, must not
	// have "nothing" written into the day's slot — an owner who buys the add-on
	// at eleven would then see an empty panel until tomorrow morning and
	// reasonably conclude they had paid for nothing.
	if state.answered {
		h.storeBriefing(ctx, key, day, lang, scope, cards)
	}
	out := map[string]any{"cards": cardsJSON(cards), "madeAt": time.Now()}
	if !state.entitled {
		out["entitled"] = false
		out["monthly"] = state.monthly
	}
	if state.capped {
		out["capped"] = true
	}
	if state.failed != "" {
		out["error"] = state.failed
	}
	httpx.JSON(w, http.StatusOK, out)
}

// writeBriefing asks the platform to turn facts into sentences.
//
// ⚠️ **Whatever comes back is filtered against the facts we sent.** That check
// is the reason this feature can be trusted at all, so it lives on the path
// every answer takes rather than at the call site — the lesson `downloadsJSON`
// was written for, applied before it is needed rather than after.
// briefingState is why a briefing is empty, when it is.
type briefingState struct {
	// answered means the platform actually wrote something — the only case
	// worth keeping for the rest of the day.
	answered bool
	entitled bool
	capped   bool
	monthly  int
	// Why the platform could not answer, when it said so. ⚠️ Kept rather than
	// dropped: an owner who enabled this and sees nothing needs the sentence,
	// and "quota exceeded, retry in 18s" is a completely different morning
	// from "no key configured".
	failed string
}

func (h *Handler) writeBriefing(
	ctx context.Context, facts []insight.Fact, lang string,
) ([]insight.Card, briefingState) {
	// Unconfigured platform: entitled is left true because there is nothing to
	// buy and nothing to explain — the panel simply draws no panel.
	state := briefingState{entitled: true}
	if h.Cfg.ControlURL == "" || h.Cfg.ControlToken == "" {
		return nil, state
	}
	payload := make([]any, 0, len(facts))
	for _, f := range facts {
		// ⚠️ Only the key and the figures travel. The action and its parameters
		// are ours to decide and are attached again on the way back.
		payload = append(payload, map[string]any{
			"key": f.Key, "area": f.Area, "numbers": f.Numbers,
		})
	}
	res, err := h.callControlPath(ctx, "/internal/insight", map[string]any{
		"lang": lang, "facts": payload,
	})
	if err != nil {
		return nil, state
	}
	if ok, present := res["entitled"].(bool); present && !ok {
		state.entitled = false
		if m, num := res["monthly"].(float64); num {
			state.monthly = int(m)
		}
		return nil, state
	}
	if capped, _ := res["capped"].(bool); capped {
		state.capped = true
		return nil, state
	}
	// ⚠️ **A reported error is not an answer, and treating it as one cost a
	// whole day.**
	//
	// The control plane answers 200 with an `error` field when the model could
	// not be reached — a quota, a dead key, a network — because the panel's
	// figures do not depend on it and a red banner over a working dashboard is
	// worse than no cards. But this side read any 200 as "the platform
	// answered", stored the empty result for the day, and the briefing then
	// stayed blank until tomorrow over a rate limit that cleared in eighteen
	// seconds.
	if msg, _ := res["error"].(string); msg != "" {
		state.failed = msg
		return nil, state
	}
	state.answered = true
	raw, _ := res["cards"].([]any)
	said := make([]insight.Card, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		title, _ := m["title"].(string)
		body, _ := m["body"].(string)
		key, _ := m["key"].(string)
		said = append(said, insight.Card{Key: key, Title: title, Body: body})
	}
	return insight.Keep(said, facts), state
}

func (h *Handler) storeBriefing(
	ctx context.Context, key bson.M, day, lang string, scope Scope, cards []insight.Card,
) {
	if cards == nil {
		cards = []insight.Card{}
	}
	_, _ = h.Store.Briefings.UpdateOne(ctx, key, bson.M{
		"$set": bson.M{
			"day": day, "lang": lang,
			"scope":  scope.BrandID.Hex() + "/" + scope.BranchID.Hex(),
			"cards":  cards,
			"madeAt": time.Now(),
		},
		"$inc": bson.M{"attempts": 1},
	}, options.Update().SetUpsert(true))
}

func briefingJSON(b storedBriefing) map[string]any {
	return map[string]any{"cards": cardsJSON(b.Cards), "madeAt": b.MadeAt}
}

// cardsJSON is the never-nil rule, in a function because this codebase has been
// bitten by a nil slice reaching a browser as `null` twice.
func cardsJSON(in []insight.Card) []insight.Card {
	if in == nil {
		return []insight.Card{}
	}
	return in
}

// AdminAIQuota is what this restaurant has used of its assistant today.
//
// ⚠️ **On the account page, not beside the briefing.** It answers a question
// about the bill — how much is left, and what to do when it runs out — and a
// card that spent one of its four lines saying "7 of 20 used" would be a line
// not spent on the restaurant.
func (h *Handler) AdminAIQuota(w http.ResponseWriter, r *http.Request) {
	if err := h.requireOwner(r); err != nil {
		httpx.Error(w, http.StatusForbidden, err.Error())
		return
	}
	if h.Cfg.ControlURL == "" || h.Cfg.ControlToken == "" {
		// ⚠️ A tenant that is not connected to the platform has no quota to
		// report and no way to buy one. Answered as "off" rather than as an
		// error: the panel then draws nothing, which is the truth.
		httpx.JSON(w, http.StatusOK, map[string]any{"on": false})
		return
	}
	res, err := h.callControlPath(r.Context(), "/internal/ai-quota", map[string]any{})
	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"on": false})
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}
