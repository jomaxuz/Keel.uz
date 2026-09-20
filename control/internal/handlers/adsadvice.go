package handlers

// ---- The targetolog: what to advertise, decided from the kitchen's figures ----
//
// A restaurant that wants adverts pays somebody $500–1000 a month to read its
// clicks. What that person cannot see is the kitchen: which dish actually
// carries the week, how far the van goes, what the quiet day is. This service
// sees exactly that and nothing else, and it is the whole argument for the
// feature — the advice is better because of what it is allowed to know.
//
// ⚠️ **Same seam as the advisor: the figures are the restaurant's server's, the
// words are the model's.** Nothing here computes a share, a total or a budget
// from data; it is handed numbers and asked which of them to act on and how to
// say it. The panel draws its bars from the same numbers, so what the owner
// reads as "41% of the week" is measured rather than written.
//
// ⚠️ **It proposes, the owner picks.** Every part of the plan comes back as
// several options with a reason each — which dish, which area, how much a day,
// which wording. A single answer would be a machine spending somebody's money
// on their behalf, and the first bad week would be ours rather than a choice
// they made. This shape is also what the section was sold as.
//
// ⚠️ **Its own daily allowance, not the assistant's** (`billing.AdsDailyCap`).
// One pot would mean a restaurant that spent the morning on campaign wording
// finds tomorrow's briefing missing — two features failing as one, with neither
// screen able to explain why.

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"keel-control/internal/ai"
	"keel-control/internal/billing"
	"keel-control/internal/httpx"
	"keel-control/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

// ⚠️ **Byte-identical for every restaurant, like the advisor's.** Prompt
// caching matches on a prefix: a restaurant name or today's date in here and
// every plan pays full price. Everything specific travels in the user message.
const adsPlanSystem = `You are the advertising planner inside Keel, a restaurant
management system used in Uzbekistan. You are planning ONE Meta (Facebook and
Instagram) advertising campaign for ONE restaurant, and the owner will choose
from what you propose.

You are given, as data:
  - DISHES: what actually sold in the last seven days, by name, with the quantity,
    the money it brought and its share of the week's takings. The week before is
    given for comparison, and "photo" says whether the restaurant has a
    photograph of that dish.
  - REACH: the address the branch cooks at, the area it delivers to and how far,
    and the hours it is open.
  - SHAPE: how many dishes are on the menu, how many branches, whether delivery
    is switched on.

Rules, in order of importance:

1. NEVER STATE A NUMBER THAT IS NOT IN THE DATA. Do not add, average, project or
   estimate. Budgets are the one exception and they are labelled as proposals —
   everything else you say about this restaurant must be a figure you were handed.
   An invented sales number is the one failure this feature cannot survive: the
   owner is about to spend real money on the strength of it.

2. PROPOSE, DO NOT DECIDE. Give three dishes worth advertising, two or three areas,
   three daily budgets, and three pieces of ad wording FOR EACH of those dishes
   (nine in all). Each one carries a short reason drawn from the data. Never say which is
   best overall, and never write as though the campaign is already running.

3. THE REASON IS THE POINT. "Osh — 41% of the week, and it rose against last week"
   is a reason. "Osh is popular" is not. If a proposal has no reason in the data,
   leave it out rather than dress it up.

4. THE AD WORDING IS FOR THIS RESTAURANT AND THIS CITY. Name the dish. Do not
   invent a price, a discount, a delivery time or an opening hour that was not
   given. Do not promise free delivery. No emoji walls, no all-caps, no
   exclamation marks stacked — Meta rejects that and so do readers.

4a. WRITE EXACTLY THREE WORDINGS FOR EACH DISH YOU PROPOSED. Three dishes means
   nine wordings, and the three that belong to one dish must be genuinely
   different from each other — a different angle, not the same sentence
   rearranged. One leads with the dish itself, one with the area it is
   delivered to, one with when it is worth ordering. The owner chooses a dish
   and is then choosing between ways to advertise THAT dish; three near-identical
   lines is no choice at all.

   Put the dish's exact name in the "dish" field of every wording, spelled
   exactly as it appears in the dishes you proposed. A wording filed under one
   dish that names another is worse than no wording: it is an advert for food
   the owner did not choose to advertise.

4b. ONLY PROPOSE A DISH THAT HAS A PHOTOGRAPH unless there is no other choice.
   Each dish in the data carries "photo": true or "photo": false, and a Meta
   advert cannot be created without a picture — we do not generate one, because
   an invented picture of a dish is a lie about what arrives at the door. If you
   must propose one without a photograph, say so plainly in its reason.

5. AREAS ARE NAMED AS THE DATA NAMES THEM. Use the address and the delivery
   distance given. Never suggest advertising outside the distance the kitchen
   actually delivers to, and if delivery is off, plan for people who can walk or
   drive in.

6. BUDGETS ARE IN UZBEK SO'M PER DAY, as whole numbers, with the number of days
   beside them. Keep them proportionate to what the restaurant takes in a week:
   a daily budget larger than a day's takings is not a plan, it is a loss.

7. DO NOT PROMISE RESULTS. No predicted orders, no return on spend, no "this will
   double". You do not know, and an owner who was promised a number will measure
   you against it.

Write in the language named in the request. Uzbek means Latin-script Uzbek as
spoken in Tashkent, not Turkish and not Cyrillic.`

// The shape the plan comes back in. ⚠️ `additionalProperties: false` throughout:
// a model given room to add a field uses it, and the panel would draw a plan
// with a key nobody wrote a line for.
var adsPlanSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"dishes": listOfVariants(map[string]any{
			"name": map[string]any{"type": "string"},
		}, "name"),
		"areas": listOfVariants(map[string]any{
			"label":    map[string]any{"type": "string"},
			"radiusKm": map[string]any{"type": "number"},
		}, "label", "radiusKm"),
		"budgets": listOfVariants(map[string]any{
			"daily": map[string]any{"type": "integer"},
			"days":  map[string]any{"type": "integer"},
		}, "daily", "days"),
		// ⚠️ **Every wording says which dish it is for.** The first version
		// returned one wording per dish as a flat list of three, so an owner
		// who chose the osh was offered wordings for the soup and the somsa —
		// two of the three choices were about food they had just decided not
		// to advertise, and the screen had no way to say so.
		"texts": listOfVariants(map[string]any{
			"dish":     map[string]any{"type": "string"},
			"headline": map[string]any{"type": "string"},
			"body":     map[string]any{"type": "string"},
		}, "dish", "headline", "body"),
	},
	"required":             []string{"dishes", "areas", "budgets", "texts"},
	"additionalProperties": false,
}

// listOfVariants is one row of the plan: some fields of its own, plus the reason
// every proposal has to carry.
//
// ⚠️ **`why` is required on all four**, which is rule 3 expressed where it is
// enforced rather than only where it is asked for. A proposal without a reason
// is the one the owner cannot weigh, and it is exactly what a model produces
// when it has nothing in the data to say.
func listOfVariants(props map[string]any, required ...string) map[string]any {
	fields := map[string]any{"why": map[string]any{"type": "string"}}
	for k, v := range props {
		fields[k] = v
	}
	return map[string]any{
		"type": "array",
		"items": map[string]any{
			"type":                 "object",
			"properties":           fields,
			"required":             append(append([]string{}, required...), "why"),
			"additionalProperties": false,
		},
	}
}

type adsAdviceRequest struct {
	Lang string `json:"lang"`
	// ⚠️ Raw, like the advisor's: this is the tenant's own fact shape, and a
	// struct here would be a second copy that drops a field silently the first
	// time the tenant adds one.
	Facts json.RawMessage `json:"facts"`
}

// AdsAdvice plans one campaign for one restaurant.
func (h *Handler) AdsAdvice(w http.ResponseWriter, r *http.Request) {
	t, err := h.tenantFromLink(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !h.aiConfigured() {
		// A platform with no key plans nothing. Answered rather than failed, so
		// the panel says so plainly instead of drawing an error.
		httpx.JSON(w, http.StatusOK, map[string]any{"off": true})
		return
	}
	var req adsAdviceRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	if len(req.Facts) == 0 {
		httpx.Error(w, http.StatusBadRequest, "facts required")
		return
	}

	// ⚠️ **The grant is read from our own record, never from the request.** A
	// tenant container is the customer's side of the wire; a handler that
	// believed a posted add-on list would be giving away a $100 section to
	// anybody who can edit a JSON body.
	_, addons, _ := h.grantOf(r.Context(), t)
	if !billing.AdsEntitled(addons) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"entitled": false, "monthly": billing.AdsMonthly,
		})
		return
	}
	// Checked before the call rather than regretted after it: we hold the key,
	// so we pay for the request that goes over the line.
	if n, err := h.adsToday(r.Context(), t.Slug); err != nil || n >= int64(billing.AdsDailyCap) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"capped": true, "cap": billing.AdsDailyCap,
		})
		return
	}

	blob, _ := json.Marshal(map[string]any{
		"language": languageName(req.Lang),
		"facts":    req.Facts,
	})
	// ⚠️ **Low effort, like the briefing and the advisor.** Choosing three of
	// five dishes and writing a sentence about each is not deep reasoning, and
	// paying to think hard about it on every plan is how a $100 add-on stops
	// being worth selling.
	text, usage, err := h.engine().JSON(
		r.Context(), adsPlanSystem, string(blob), adsPlanSchema, "low")
	if err != nil {
		// ⚠️ The engines' own words are for our log; the owner gets one
		// sentence they can act on. See ai/explain.go.
		log.Printf("ads plan %s: %v", t.Slug, err)
		httpx.JSON(w, http.StatusOK, map[string]any{
			"error": ai.Explain(err, req.Lang),
		})
		return
	}
	h.recordAds(r.Context(), t.Slug, usage)

	plan := map[string]any{}
	if json.Unmarshal([]byte(text), &plan) != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"error": "reja o'qilmadi"})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"plan": plan})
}

// adsToday counts the campaign plans this tenant has already bought from us
// today.
//
// ⚠️ **A separate ledger from the briefing's**, which is the point: the
// advertising add-on is sold with its own daily number on it, and a counter
// shared with the assistant would make that number a lie in both directions.
func (h *Handler) adsToday(ctx context.Context, slug string) (int64, error) {
	from := time.Now().Truncate(24 * time.Hour)
	return h.Store.AdsLog.CountDocuments(ctx,
		bson.M{"slug": slug, "at": bson.M{"$gte": from}})
}

func (h *Handler) recordAds(ctx context.Context, slug string, u ai.Usage) {
	// Tokens rather than a price, for the reason written on the briefing log:
	// a rate hard-coded today is a number that quietly stops being right.
	_, _ = h.Store.AdsLog.InsertOne(ctx, models.AdsLog{
		Slug: slug, At: time.Now(),
		InputTokens:  u.Input,
		CachedTokens: u.Cached,
		OutputTokens: u.Output,
	})
}
