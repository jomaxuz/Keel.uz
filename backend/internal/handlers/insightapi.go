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
	if !scope.BranchID.IsZero() {
		filter["branchId"] = scope.BranchID
	}
	facts := h.gatherFacts(ctx, filter)
	if len(facts) == 0 {
		// ⚠️ Nothing wrong and nothing notable is a real answer, and a young
		// restaurant with three weeks of orders will get it often. Stored like
		// any other, so the empty case is not recomputed all morning either.
		h.storeBriefing(ctx, key, day, lang, scope, nil)
		httpx.JSON(w, http.StatusOK, map[string]any{"cards": []insight.Card{}})
		return
	}

	cards := h.writeBriefing(ctx, facts, lang)
	h.storeBriefing(ctx, key, day, lang, scope, cards)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"cards": cardsJSON(cards), "madeAt": time.Now(),
	})
}

// writeBriefing asks the platform to turn facts into sentences.
//
// ⚠️ **Whatever comes back is filtered against the facts we sent.** That check
// is the reason this feature can be trusted at all, so it lives on the path
// every answer takes rather than at the call site — the lesson `downloadsJSON`
// was written for, applied before it is needed rather than after.
func (h *Handler) writeBriefing(
	ctx context.Context, facts []insight.Fact, lang string,
) []insight.Card {
	if h.Cfg.ControlURL == "" || h.Cfg.ControlToken == "" {
		return nil
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
		return nil
	}
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
	return insight.Keep(said, facts)
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
