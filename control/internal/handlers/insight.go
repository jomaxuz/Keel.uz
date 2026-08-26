package handlers

// ---- The morning briefing, written for one restaurant ----
//
// ⚠️ **The key lives here and nowhere else.** Every tenant runs in its own
// container with its own database, and a key copied into each of them would be
// one secret in N places — N chances to leak it and no way to rotate it in an
// afternoon. The tenant sends what it has computed; we hold the credential, add
// the instructions, and send back sentences.
//
// ⚠️ **And because we hold the key, we pay.** That is a deliberate product
// choice — a feature a restaurant has to open an Anthropic account to enable is
// a feature nobody in Tashkent enables — but it means spending is ours to bound.
// The cap below is per tenant per day, checked before the call rather than
// regretted after it.
//
// ⚠️ **What arrives here is already aggregate.** Counts and sums, never a name,
// a phone or an address. The control plane is not a safer place to put a
// restaurant's customer list than the restaurant is; it is simply not sent.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"keel-control/internal/ai"
	"keel-control/internal/billing"
	"keel-control/internal/httpx"
	"keel-control/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

// ⚠️ **This text is byte-identical for every restaurant and every morning, and
// that is load-bearing rather than tidy.** Prompt caching matches on a prefix:
// the moment a restaurant's name or today's date appears in it, every call
// misses the cache and the daily sweep costs several times what it should. The
// restaurant-specific part lives in the user message, after this, where it
// belongs.
const briefingSystem = `You advise the owner of a single restaurant in Uzbekistan.

Every morning you are given a short list of FACTS about their restaurant. Each
fact has a key and exact figures that were computed from the restaurant's own
database.

Your job is to choose the two to four facts that matter most this morning and
write one short card for each.

Rules, in order of importance:

1. NEVER state a number that is not in the facts you were given. Do not add,
   average, project, or estimate. The panel prints the exact figures beside your
   words, and a figure of yours that disagrees with one of ours destroys the
   owner's trust in every card, including the correct ones.

2. NEVER invent a fact. Answer only with keys from the list you were given. A
   card whose key was not in the list is discarded.

3. Write to a busy person about to open their restaurant. Title: at most six
   words, naming the thing. Body: one or two sentences — what is happening, and
   what to do about it. No greetings, no preamble, no "as an assistant".

4. Say what to do, concretely, in the body's last sentence. The owner has a
   button; your job is to make pressing it obvious.

5. Do not scold and do not congratulate. An owner reads this at 8am; the useful
   register is a good manager's, not a coach's.

6. If a fact is good news, say so plainly and briefly. A briefing that only ever
   reports problems is read as noise within a week.

7. Some facts name a member of staff. Write those with particular care, because
   the owner may act on them the same morning and the person named is not there
   to answer.

   - State the comparison, never a conclusion. "Aziz voided 34 lines where the
     restaurant averages 9" is a fact. "Aziz may be stealing" is not, and you
     were not given anything that could support it.
   - Name at least one ordinary explanation in the same breath — a new starter,
     a difficult section, equipment that failed, a shift nobody else works.
     There usually is one, and an owner who is reminded of that asks rather than
     accuses.
   - Never suggest dismissing, punishing, deducting from wages, or confronting
     anybody. The action is to look at the report and ask a question.
   - Do not soften the number to be kind. The figure is the reason the card
     exists; it is the interpretation that must stay open, not the arithmetic.

Write in the language named in the request. Uzbek means Latin-script Uzbek as
spoken in Tashkent, not Turkish and not Cyrillic.`

// briefingSchema is what we will accept back.
var briefingSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"cards": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key":   map[string]any{"type": "string"},
					"title": map[string]any{"type": "string"},
					"body":  map[string]any{"type": "string"},
				},
				"required":             []string{"key", "title", "body"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"cards"},
	"additionalProperties": false,
}

// planOf is what this tenant is actually entitled to, from the grant we wrote.
//
// ⚠️ **A read failure is no plan — the smallest cap and no entitlement.** Mongo
// being briefly unreachable has to fail towards spending nothing: the
// alternative is that a database blink hands every tenant on the platform an
// uncapped, unpaid assistant, and nothing on any screen would say so.
func (h *Handler) planOf(ctx context.Context, t *models.Tenant) (string, []string) {
	plan, addons, _ := h.grantOf(ctx, t)
	return plan, addons
}

// grantOf is the plan, its add-ons, and any AI blocks bought on top.
func (h *Handler) grantOf(ctx context.Context, t *models.Tenant) (string, []string, int) {
	var grant struct {
		Plan   string   `bson:"plan"`
		Addons []string `bson:"addons"`
		// ⚠️ Written by the console, read here. Blocks of ten daily requests,
		// added to whatever the plan already allows.
		AIExtra int `bson:"aiExtra"`
	}
	err := h.Store.TenantDB(t.DBName()).Collection("subscription").
		FindOne(ctx, bson.M{"_id": tillGrantID}).Decode(&grant)
	if err != nil {
		return "", nil, 0
	}
	return grant.Plan, grant.Addons, grant.AIExtra
}

// Briefing turns one tenant's facts into cards.
func (h *Handler) Briefing(w http.ResponseWriter, r *http.Request) {
	t, err := h.tenantFromLink(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	// ⚠️ **Either engine counts.** Checking only Anthropic's key would leave a
	// platform running on Gemini reporting itself as switched off — the setting
	// says one thing and the feature does another, which is the shape of bug
	// this codebase keeps paying for.
	if !h.aiConfigured() {
		// ⚠️ Not an error: a platform without a key configured simply has no
		// assistant, and the panel draws its own numbers regardless.
		httpx.JSON(w, http.StatusOK, map[string]any{"cards": []any{}, "off": true})
		return
	}
	var req struct {
		Lang  string            `json:"lang"`
		Facts []json.RawMessage `json:"facts"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil ||
		len(req.Facts) == 0 {
		httpx.Error(w, http.StatusBadRequest, "facts required")
		return
	}
	// ⚠️ **The plan is read from our own record of it, never from the request.**
	// A tenant container is the customer's side of the wire — the rule the
	// domain link is built on — and a handler that believed a posted
	// `plan: "enterprise"` would be selling upgrades to anybody who can edit a
	// JSON body.
	plan, addons, extra := h.grantOf(r.Context(), t)
	if !billing.AIEntitled(plan, addons) {
		// ⚠️ Not an error and not silence: the panel has to be able to tell the
		// owner this is something they can buy, and a feature that fails
		// invisibly is one nobody ever asks the price of.
		httpx.JSON(w, http.StatusOK, map[string]any{
			"cards": []any{}, "entitled": false, "monthly": billing.AIMonthly,
		})
		return
	}
	limit := billing.AIDailyCapWith(plan, extra)
	if n, err := h.briefingsToday(r.Context(), t.Slug); err != nil || n >= int64(limit) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"cards": []any{}, "capped": true, "cap": limit,
		})
		return
	}

	cards, usage, err := h.askForBriefing(r.Context(), req.Lang, req.Facts)
	if err != nil {
		// ⚠️ Reported as an empty briefing, never as a failure. The panel's
		// numbers do not depend on this call, and a red banner over a dashboard
		// because a third party was slow is a worse morning than no cards.
		// ⚠️ **A spent daily allowance is reported as such, so the tenant waits
		// hours rather than minutes.** Every rejected request still counts
		// against the free tier, so retrying every ten minutes after "you
		// exceeded your quota" spends the rest of the day discovering the same
		// sentence — and pins the allowance shut for the restaurants whose
		// briefings have not been built yet. Seven restaurants need seven
		// requests; twenty is enough right up until something retries into it.
		var spent ai.Exhausted
		httpx.JSON(w, http.StatusOK, map[string]any{
			"cards": []any{}, "error": err.Error(),
			"exhausted": errors.As(err, &spent),
		})
		return
	}
	h.recordBriefing(r.Context(), t.Slug, usage)
	httpx.JSON(w, http.StatusOK, map[string]any{"cards": cards})
}

func (h *Handler) askForBriefing(
	ctx context.Context, lang string, facts []json.RawMessage,
) ([]map[string]string, ai.Usage, error) {
	blob, _ := json.Marshal(map[string]any{
		"language": languageName(lang),
		"facts":    facts,
	})
	// ⚠️ **Low effort on purpose.** Choosing four things out of eleven and
	// saying each in a sentence is not deep reasoning; the cost of thinking
	// hard about it every morning for every restaurant is.
	text, usage, err := h.engine().JSON(ctx, briefingSystem, string(blob), briefingSchema, "low")
	if err != nil {
		return nil, ai.Usage{}, err
	}
	var parsed struct {
		Cards []map[string]string `json:"cards"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, usage, err
	}
	return parsed.Cards, usage, nil
}

// languageName spells the language out rather than passing a code.
//
// ⚠️ "uz" is a label a model has to guess the intent of, and the wrong guess
// here is Turkish or Cyrillic — both of which look like a working feature to
// anybody who does not read the result.
func languageName(code string) string {
	switch code {
	case "ru":
		return "Russian"
	case "en":
		return "English"
	default:
		return "Uzbek (Latin script)"
	}
}

// briefingsToday counts what this tenant has already bought from us today.
func (h *Handler) briefingsToday(ctx context.Context, slug string) (int64, error) {
	from := time.Now().Truncate(24 * time.Hour)
	return h.Store.BriefingLog.CountDocuments(ctx,
		bson.M{"slug": slug, "at": bson.M{"$gte": from}})
}

func (h *Handler) recordBriefing(ctx context.Context, slug string, u ai.Usage) {
	// ⚠️ Tokens are recorded, not a price. A rate that is right today is a
	// number that quietly stops being right, and the arithmetic is better done
	// where somebody can see the rate they used.
	_, _ = h.Store.BriefingLog.InsertOne(ctx, models.BriefingLog{
		Slug: slug, At: time.Now(),
		InputTokens:  u.Input,
		CachedTokens: u.Cached,
		OutputTokens: u.Output,
	})
}

// AIUsage is what the assistant has cost, per tenant, this month.
//
// ⚠️ **A screen because we pay.** A cost that only appears on an invoice is one
// nobody looks at until it is surprising, and the surprise arrives a month
// after the behaviour that caused it. Tokens rather than money, for the reason
// recorded on the log itself: a rate written into code is a number that quietly
// stops being right.
func (h *Handler) AIUsage(w http.ResponseWriter, r *http.Request) {
	from := time.Now().AddDate(0, 0, -30)
	cur, err := h.Store.BriefingLog.Aggregate(r.Context(), []bson.M{
		{"$match": bson.M{"at": bson.M{"$gte": from}}},
		{"$group": bson.M{
			"_id":    "$slug",
			"calls":  bson.M{"$sum": 1},
			"in":     bson.M{"$sum": "$inputTokens"},
			"cached": bson.M{"$sum": "$cachedTokens"},
			"out":    bson.M{"$sum": "$outputTokens"},
			"last":   bson.M{"$max": "$at"},
		}},
		{"$sort": bson.M{"out": -1}},
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cur.Close(r.Context())
	// ⚠️ Never nil. The JSON trap this codebase has been bitten by twice, and
	// this list is empty on every platform until somebody enables the feature —
	// which is exactly the first time anybody would open this screen.
	rows := []map[string]any{}
	for cur.Next(r.Context()) {
		var g struct {
			Slug   string    `bson:"_id"`
			Calls  int       `bson:"calls"`
			In     int64     `bson:"in"`
			Cached int64     `bson:"cached"`
			Out    int64     `bson:"out"`
			Last   time.Time `bson:"last"`
		}
		if cur.Decode(&g) != nil {
			continue
		}
		rows = append(rows, map[string]any{
			"slug": g.Slug, "calls": g.Calls,
			"inputTokens": g.In, "cachedTokens": g.Cached,
			"outputTokens": g.Out,
			// ⚠️ `.In(time.Local)` before it is shown: the driver decodes every
			// date as UTC whatever TZ says, and a "last used" five hours early
			// looks like the feature stopped working last night.
			"last": g.Last.In(time.Local),
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"rows": rows, "days": 30})
}

// engine is whichever model this platform is configured to ask.
//
// ⚠️ **A chain, not a choice, and the order is the configuration.** Both keys
// can be set and the first one that answers wins — which matters because one of
// them can be unreachable for reasons that have nothing to do with the code:
// Anthropic billing needs a card that works internationally, and the first
// thing this platform's own key ever answered was "credit balance too low". A
// restaurant's morning briefing should not depend on which payment rails were
// available to us that month.
//
// ⚠️ **Built per request rather than held on the handler.** These are two
// structs and a string; the cost is nothing, and a cached client would mean a
// key changed in the environment does nothing until something restarts.
func (h *Handler) engine() ai.Model {
	claude := ai.Claude{Key: h.Cfg.AnthropicKey, Model: h.Cfg.AIModel}
	gemini := ai.Gemini{Key: h.Cfg.GeminiKey, Model: h.Cfg.GeminiModel}
	// ⚠️ An unset key is not an engine. `Chain` would try it and get
	// `ErrNoKey`, which works — but then the error a platform with neither key
	// reports is whichever engine happened to be last, rather than the honest
	// "nothing is configured".
	var chain ai.Chain
	add := func(m ai.Model, key string) {
		if key != "" {
			chain = append(chain, m)
		}
	}
	if h.Cfg.AIProvider == "gemini" {
		add(gemini, h.Cfg.GeminiKey)
		add(claude, h.Cfg.AnthropicKey)
	} else {
		add(claude, h.Cfg.AnthropicKey)
		add(gemini, h.Cfg.GeminiKey)
	}
	return chain
}

// aiConfigured reports whether anything can answer at all.
func (h *Handler) aiConfigured() bool {
	return h.Cfg.AnthropicKey != "" || h.Cfg.GeminiKey != ""
}

// AIQuota is what one restaurant has used today and what it may use.
//
// ⚠️ **On the account page rather than beside the feature**, because it answers
// a question about the bill: how much is left, and what to do when it runs out.
// The briefing itself never mentions a quota — a card that spent a line saying
// "7 of 20 used" would be a line not spent on the restaurant.
func (h *Handler) AIQuota(w http.ResponseWriter, r *http.Request) {
	t, err := h.tenantFromLink(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	plan, addons, extra := h.grantOf(r.Context(), t)
	used, _ := h.briefingsToday(r.Context(), t.Slug)
	limit := billing.AIDailyCapWith(plan, extra)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"entitled": billing.AIEntitled(plan, addons),
		"on":       h.aiConfigured(),
		"used":     used,
		"limit":    limit,
		// What the plan gives before anything was bought, so the panel can say
		// "20 + 10 bought" rather than one number nobody can account for.
		"planLimit":   billing.AIDailyCap(plan),
		"extraBlocks": extra,
		"blockSize":   billing.AIExtraBlock,
		"blockPrice":  billing.AIExtraMonthly,
		// ⚠️ The handle to write to, from the server rather than the screen: it
		// is ours, and a panel that carried its own copy would be as many
		// copies as there are tenants on the day it changes.
		"contact": "@keeluz",
	})
}
