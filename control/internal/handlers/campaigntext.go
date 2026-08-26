package handlers

// ---- Proposing a campaign message ----
//
// The second thing the assistant does, and the one an owner asked for first:
// what to send, to whom, this week.

import (
	"context"
	"encoding/json"
	"net/http"

	"keel-control/internal/ai"
	"keel-control/internal/billing"
	"keel-control/internal/httpx"
)

// ⚠️ Identical for every restaurant and every send, for the same reason the
// briefing's is: caching matches on a prefix, and the restaurant's name in here
// would make every call a miss. The name goes in the user message.
const campaignSystem = `You write short marketing messages for restaurants in Uzbekistan.

You are given a customer segment, how many people are in it, the channel, a
character budget, and sometimes an offer the owner has decided on.

Write exactly three variants. They must differ in approach, not in wording — a
set of three near-identical sentences wastes the owner's time choosing.

Rules, in order of importance:

1. NEVER invent an offer, a discount, a price, a dish or an event. If the owner
   gave you an offer, that is the only one that exists. If they gave you none,
   write a message that has none — an invitation, a reminder, news that the
   restaurant is there. A guest who arrives expecting a discount you made up
   argues with a cashier who has never heard of it.

2. Stay inside the character budget. On SMS this is money: one character over
   and every message in the send costs twice as much.

3. Write like a restaurant messaging its own guests, not like an advertisement.
   No exclamation marks stacked up, no capitals for emphasis, no emoji unless
   the channel is Telegram and even then at most one.

4. Say who it is from. A message with no restaurant name is a message people
   report as spam.

5. Do not promise a time, a table, or availability. Nobody sending this can
   guarantee them.

For each variant add a short note, in the requested language, saying what the
approach is — so the owner is choosing between ideas rather than between
sentences.

Write the message text in the language named in the request. Uzbek means
Latin-script Uzbek as spoken in Tashkent, not Turkish and not Cyrillic.`

var campaignSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"variants": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"text": map[string]any{"type": "string"},
					"note": map[string]any{"type": "string"},
				},
				"required":             []string{"text", "note"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"variants"},
	"additionalProperties": false,
}

// CampaignText writes three messages for one segment.
func (h *Handler) CampaignText(w http.ResponseWriter, r *http.Request) {
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
		httpx.JSON(w, http.StatusOK, map[string]any{"variants": []any{}, "off": true})
		return
	}
	// The same gate as the briefing, read from our own record of the plan.
	plan, addons, extra := h.grantOf(r.Context(), t)
	if !billing.AIEntitled(plan, addons) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"variants": []any{}, "entitled": false, "monthly": billing.AIMonthly,
		})
		return
	}
	// ⚠️ **One budget, shared with the briefing.** Two separate daily counters
	// would mean the cap on the plan is not the cap on the spending, and the
	// number an owner was told is the one that has to be true.
	if n, err := h.briefingsToday(r.Context(), t.Slug); err != nil ||
		n >= int64(billing.AIDailyCapWith(plan, extra)) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"variants": []any{}, "capped": true,
			"cap": billing.AIDailyCapWith(plan, extra),
		})
		return
	}

	var req map[string]any
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	variants, usage, err := h.askForCampaign(r.Context(), req)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	h.recordBriefing(r.Context(), t.Slug, usage)
	httpx.JSON(w, http.StatusOK, map[string]any{"variants": variants})
}

func (h *Handler) askForCampaign(
	ctx context.Context, req map[string]any,
) ([]map[string]string, ai.Usage, error) {
	if lang, ok := req["lang"].(string); ok {
		req["language"] = languageName(lang)
		delete(req, "lang")
	}
	blob, _ := json.Marshal(req)
	// ⚠️ Higher effort than the briefing's. The briefing picks from a list we
	// prepared; this writes something that goes out over the restaurant's name
	// to a thousand of its own guests, and a clumsy sentence there is not
	// recoverable by pressing again.
	text, usage, err := h.engine().JSON(ctx, campaignSystem, string(blob), campaignSchema, "medium")
	if err != nil {
		return nil, ai.Usage{}, err
	}
	var parsed struct {
		Variants []map[string]string `json:"variants"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, usage, err
	}
	return parsed.Variants, usage, nil
}
