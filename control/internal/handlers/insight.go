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
	"net/http"
	"time"

	"keel-control/internal/httpx"
	"keel-control/internal/models"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
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

// dailyBriefingCap is how many briefings one tenant may buy from us in a day.
//
// ⚠️ **Above one because a retry is normal, far below infinity because a
// looping tenant is not.** A restaurant with a stuck panel tab refreshing every
// thirty seconds would otherwise bill us for a thousand calls before anybody
// noticed — and the first sign would be an invoice, not an alert.
const dailyBriefingCap = 6

// Briefing turns one tenant's facts into cards.
func (h *Handler) Briefing(w http.ResponseWriter, r *http.Request) {
	t, err := h.tenantFromLink(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if h.Cfg.AnthropicKey == "" {
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
	if n, err := h.briefingsToday(r.Context(), t.Slug); err != nil || n >= dailyBriefingCap {
		httpx.JSON(w, http.StatusOK, map[string]any{"cards": []any{}, "capped": true})
		return
	}

	cards, usage, err := h.askForBriefing(r.Context(), req.Lang, req.Facts)
	if err != nil {
		// ⚠️ Reported as an empty briefing, never as a failure. The panel's
		// numbers do not depend on this call, and a red banner over a dashboard
		// because a third party was slow is a worse morning than no cards.
		httpx.JSON(w, http.StatusOK, map[string]any{"cards": []any{}, "error": err.Error()})
		return
	}
	h.recordBriefing(r.Context(), t.Slug, usage)
	httpx.JSON(w, http.StatusOK, map[string]any{"cards": cards})
}

func (h *Handler) askForBriefing(
	ctx context.Context, lang string, facts []json.RawMessage,
) ([]map[string]string, anthropic.Usage, error) {
	blob, _ := json.Marshal(map[string]any{
		"language": languageName(lang),
		"facts":    facts,
	})
	client := anthropic.NewClient(option.WithAPIKey(h.Cfg.AnthropicKey))
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	adaptive := anthropic.ThinkingConfigAdaptiveParam{}
	res, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     "claude-opus-5",
		MaxTokens: 4000,
		Thinking:  anthropic.ThinkingConfigParamUnion{OfAdaptive: &adaptive},
		OutputConfig: anthropic.OutputConfigParam{
			// Choosing four things out of eleven and saying each in a sentence
			// is not deep reasoning; the cost of thinking hard about it every
			// morning for every restaurant is.
			Effort: anthropic.OutputConfigEffortLow,
			Format: anthropic.JSONOutputFormatParam{Schema: briefingSchema},
		},
		System: []anthropic.TextBlockParam{{
			Text: briefingSystem,
			// ⚠️ An hour, not the default five minutes: the sweep runs through
			// every restaurant back to back, and a platform with a hundred of
			// them takes longer than five minutes to get round them all.
			CacheControl: anthropic.CacheControlEphemeralParam{
				TTL: anthropic.CacheControlEphemeralTTLTTL1h,
			},
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(string(blob))),
		},
	})
	if err != nil {
		return nil, anthropic.Usage{}, err
	}
	var parsed struct {
		Cards []map[string]string `json:"cards"`
	}
	for _, block := range res.Content {
		if b, ok := block.AsAny().(anthropic.TextBlock); ok {
			if json.Unmarshal([]byte(b.Text), &parsed) == nil && len(parsed.Cards) > 0 {
				break
			}
		}
	}
	return parsed.Cards, res.Usage, nil
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

func (h *Handler) recordBriefing(ctx context.Context, slug string, u anthropic.Usage) {
	// ⚠️ Tokens are recorded, not a price. A rate that is right today is a
	// number that quietly stops being right, and the arithmetic is better done
	// where somebody can see the rate they used.
	_, _ = h.Store.BriefingLog.InsertOne(ctx, models.BriefingLog{
		Slug: slug, At: time.Now(),
		InputTokens:  u.InputTokens,
		CachedTokens: u.CacheReadInputTokens,
		OutputTokens: u.OutputTokens,
	})
}
