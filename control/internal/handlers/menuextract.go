package handlers

// ---- Reading a menu off a page ----
//
// The second job the assistant does that saves an owner a day rather than a
// minute. Typing a menu in is the longest task in setting a restaurant up — a
// hundred dishes, each with a name, a price, a description and a photograph —
// and it is the reason a signed customer takes three weeks to go live.
//
// ⚠️ **This is the fallback, not the path.** The tenant reads schema.org data
// off the page first, which is exact and free; it reaches here only when the
// page has none. Asking a model to read prices off a page that already states
// them would cost money to be less accurate.
//
// ⚠️ **Nothing here writes anything.** The answer is a proposal the owner
// reviews, edits and ticks before a single dish reaches their menu.

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"keel-control/internal/ai"
	"keel-control/internal/billing"
	"keel-control/internal/httpx"
)

// ⚠️ Identical for every restaurant, like the other two prompts: caching
// matches on a prefix and a name in here would make every call a miss.
const menuExtractSystem = `You read a restaurant's web page and return the dishes on it.

The input is one page, reduced to its text. Image addresses appear on their own
lines as "[img] https://…", immediately before or after the dish they belong to.

Rules, in order of importance:

1. NEVER invent a dish, a price or a description. If the page does not say what
   something costs, return price 0. A price you guessed becomes an argument at
   a till with a guest holding a menu.

2. Prices are Uzbek so'm, as a whole number. The page writes forty-five thousand
   as "45 000", "45,000", "45.000" or "45000" — all four mean 45000. Never
   return 45, and never return a fractional part.

3. Return only things a guest can order. Not the delivery fee, not the service
   charge, not "minimum order", not restaurant names, not headings.

4. Put each dish under the section heading it appeared beneath, in the page's
   own words. If the page has no sections, leave category empty rather than
   inventing one.

5. Copy names and descriptions as written. Do not translate, do not tidy, do not
   expand abbreviations. The owner will edit them; a name you improved is one
   they have to compare against the original to notice.

6. Attach an image only when it is clearly that dish's. A picture you paired by
   proximity to the wrong dish is worse than no picture, because it looks
   correct in a list and is discovered by a guest.

Return an empty list if the page is not a menu.`

var menuExtractSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"dishes": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":        map[string]any{"type": "string"},
					"description": map[string]any{"type": "string"},
					"price":       map[string]any{"type": "integer"},
					"imageUrl":    map[string]any{"type": "string"},
					"category":    map[string]any{"type": "string"},
				},
				"required":             []string{"name", "description", "price", "imageUrl", "category"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"dishes"},
	"additionalProperties": false,
}

// MenuExtract turns one page's text into a list of dishes.
func (h *Handler) MenuExtract(w http.ResponseWriter, r *http.Request) {
	t, err := h.tenantFromLink(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !h.aiConfigured() {
		httpx.JSON(w, http.StatusOK, map[string]any{"dishes": []any{}, "off": true})
		return
	}
	plan, addons, extra := h.grantOf(r.Context(), t)
	if !billing.AIEntitled(plan, addons) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"dishes": []any{}, "entitled": false, "monthly": billing.AIMonthly,
		})
		return
	}
	// ⚠️ **The same daily budget as the briefing and the campaign writer.**
	// Three separate counters would mean the cap on the plan is not the cap on
	// the spending, and the number an owner was told is the one that has to be
	// true. An import is one call, so a restaurant setting itself up spends a
	// handful of a day's allowance and keeps the rest.
	if n, err := h.briefingsToday(r.Context(), t.Slug); err != nil ||
		n >= int64(billing.AIDailyCapWith(plan, extra)) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"dishes": []any{}, "capped": true,
			"cap": billing.AIDailyCapWith(plan, extra),
		})
		return
	}

	var req struct {
		Text string `json:"text"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil ||
		req.Text == "" {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}

	dishes, usage, err := h.askForMenu(r.Context(), req.Text)
	if err != nil {
		log.Printf("menu extract: %v", err)
		httpx.Error(w, http.StatusBadGateway, ai.Explain(err, ""))
		return
	}
	h.recordBriefing(r.Context(), t.Slug, usage)
	httpx.JSON(w, http.StatusOK, map[string]any{"dishes": dishes})
}

func (h *Handler) askForMenu(
	ctx context.Context, text string,
) ([]map[string]any, ai.Usage, error) {
	// ⚠️ Low effort, unlike the campaign writer's. This is transcription, not
	// composition: everything the answer should contain is in front of it, and
	// thinking longer about a price that is written on the page does not make
	// it more correct — it makes it more expensive, on the one call that
	// processes a hundred dishes at once.
	out, usage, err := h.engine().JSON(ctx, menuExtractSystem, text, menuExtractSchema, "low")
	if err != nil {
		return nil, ai.Usage{}, err
	}
	var parsed struct {
		Dishes []map[string]any `json:"dishes"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return nil, usage, err
	}
	return parsed.Dishes, usage, nil
}
