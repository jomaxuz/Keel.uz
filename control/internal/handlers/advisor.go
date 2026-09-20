package handlers

// ---- The advisor: the owner's own question, answered from their own figures ----
//
// The briefing decides what to tell a restaurant every morning. This answers
// what they ask. Same seam, same rule, and the rule is the reason either can be
// trusted: **the figures are computed by the restaurant's own server and the
// model only writes sentences about them.** A model asked to derive a number
// gets it occasionally wrong, and an owner who catches one wrong number stops
// reading the other four — the feature is not switched off at that point, it is
// simply never opened again.
//
// ⚠️ **It answers about this restaurant and about Keel, and refuses everything
// else.** A general chat box in a restaurant panel is a support burden with no
// ceiling, and the first time it explains tax law to somebody who acts on it we
// own the consequence. The refusal is a first-class answer here, exactly as it
// is for the support assistant next door.
//
// ⚠️ **No customer ever reaches this service.** A lapsed regular arrives as
// `c3` and four numbers; the tenant's own server puts the name back on the way
// to the browser. So the advice can say "ring these four first" while the
// control plane holds no customer list at all — see the tenant's
// handlers/advisor.go.
//
// ⚠️ **It spends the same daily allowance as the briefing**, deliberately: an
// owner buys "the assistant", not "the briefing and, separately, the advisor".
// Two counters would also mean two numbers on the account page and a support
// conversation about which one ran out.

import (
	"encoding/json"
	"log"
	"net/http"

	"keel-control/internal/ai"
	"keel-control/internal/billing"
	"keel-control/internal/httpx"
)

// ⚠️ **Byte-identical for every restaurant and every question.** Prompt caching
// matches on a prefix: the moment a restaurant's name, today's date or its
// figures appear in here, every call misses the cache and each question costs
// several times what it should. Everything specific is in the user message.
const adviseSystem = `You are the business advisor inside Keel, a restaurant management
system used in Uzbekistan. The owner or manager of ONE restaurant is asking you a
question in their own panel.

You are given, as data:
  - FACTS: exact figures computed from this restaurant's own database this morning.
    Each has a key, an area and named numbers.
  - PROFILE: the shape of the business — how many branches, dishes, staff, suppliers,
    stores, the working hours, and a sample of lapsed regular guests.
  - The conversation so far, if there is one.

Rules, in order of importance:

1. NEVER STATE A NUMBER THAT IS NOT IN THE DATA. Do not add, average, project,
   extrapolate or estimate. If the answer needs a figure you were not given, say which
   figure is missing and where in the panel it is computed. An invented number is the
   one failure this feature cannot survive.

2. ANSWER ONLY ABOUT THIS RESTAURANT AND ABOUT KEEL ITSELF. Anything else — law, tax,
   medicine, politics, general knowledge, code, another company, writing that is not
   about this business — is refused. Set refused to true and say, in one sentence, that
   you only answer questions about this restaurant and this system. Do not apologise at
   length and do not explain the policy.

3. IF THE DATA DOES NOT ANSWER THE QUESTION, SAY SO. "Bu savolga javob beradigan raqam
   menda yo'q" and then where to look is a good answer. A plausible guess is not: the
   owner will act on it, and nobody finds out until something is broken.

4. GUESTS ARE ANONYMOUS AND STAY THAT WAY. Lapsed guests are identified as c1, c2, c3.
   Use those identifiers exactly as given when you point at one — the panel turns them
   back into names. Never invent an identifier that was not in the data, and never
   guess who somebody is.

5. BE SPECIFIC AND SHORT. Two to five sentences. Answer the question first, then the
   most likely reason drawn only from the figures — said as a possibility, not a
   verdict — then the one thing to do about it, naming the screen in the panel where it
   is done. The owner is reading this between two services.

6. WHEN A FACT NAMES A MEMBER OF STAFF, write it with care. State the comparison, never
   a conclusion. Name at least one ordinary explanation in the same breath — a new
   starter, a hard section, equipment that failed. Never suggest dismissing, deducting
   pay from, or confronting anybody: the action is to look and ask. Do not soften the
   figure to be kind; it is the interpretation that stays open, not the arithmetic.

7. Do not promise anything on the company's behalf — no refunds, no dates, no "we will
   fix it". You do not know.

8. Do not scold and do not congratulate. The useful register is a good manager's.

Write in the language named in the request. Uzbek means Latin-script Uzbek as spoken in
Tashkent, not Turkish and not Cyrillic.`

var adviseSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"answered": map[string]any{"type": "boolean"},
		"refused":  map[string]any{"type": "boolean"},
		"text":     map[string]any{"type": "string"},
	},
	"required":             []string{"answered", "refused", "text"},
	"additionalProperties": false,
}

type adviseRequest struct {
	Lang     string              `json:"lang"`
	Question string              `json:"question"`
	History  []map[string]string `json:"history"`
	// ⚠️ Passed through as raw JSON: these are the tenant's own fact and profile
	// shapes, and a struct here would be a second copy of them — one that drops
	// a field silently the first time the tenant adds one.
	Facts   json.RawMessage `json:"facts"`
	Profile json.RawMessage `json:"profile"`
}

// Advise answers one question about one restaurant.
func (h *Handler) Advise(w http.ResponseWriter, r *http.Request) {
	t, err := h.tenantFromLink(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !h.aiConfigured() {
		// A platform with no key has no assistant. Answered rather than failed,
		// so the panel can say so plainly instead of drawing an error.
		httpx.JSON(w, http.StatusOK, map[string]any{"off": true, "answer": ""})
		return
	}
	var req adviseRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	question := clampSupport(req.Question)
	if question == "" {
		httpx.Error(w, http.StatusBadRequest, "question required")
		return
	}

	// ⚠️ **The plan is read from our own record, never from the request.** A
	// tenant container is the customer's side of the wire, and a handler that
	// believed a posted plan would be selling upgrades to anybody who can edit
	// a JSON body.
	plan, addons, extra := h.grantOf(r.Context(), t)
	if !billing.AIEntitled(plan, addons) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"entitled": false, "monthly": billing.AIMonthly,
		})
		return
	}
	limit := billing.AIDailyCapWith(plan, extra)
	// ⚠️ **The briefing's own counter**, so one allowance covers the assistant
	// however it is reached. Checked before the call rather than regretted
	// after it: we hold the key, so we pay.
	if n, err := h.briefingsToday(r.Context(), t.Slug); err != nil || n >= int64(limit) {
		httpx.JSON(w, http.StatusOK, map[string]any{"capped": true, "cap": limit})
		return
	}

	blob, _ := json.Marshal(map[string]any{
		"language": languageName(req.Lang),
		"question": question,
		"history":  req.History,
		"facts":    req.Facts,
		"profile":  req.Profile,
	})
	// ⚠️ **Low effort, like the briefing.** Choosing which of fifteen figures
	// answers a question and saying it in four sentences is not deep reasoning;
	// paying to think hard about it on every question is.
	text, usage, err := h.engine().JSON(
		r.Context(), adviseSystem, string(blob), adviseSchema, "low")
	if err != nil {
		// ⚠️ **The engines' own words stay in our log and never reach the
		// restaurant.** What they produce names models, HTTP statuses, request
		// ids and — worst of it — our own billing. See ai/explain.go.
		log.Printf("advisor %s: %v", t.Slug, err)
		httpx.JSON(w, http.StatusOK, map[string]any{
			"error": ai.Explain(err, req.Lang),
		})
		return
	}
	h.recordBriefing(r.Context(), t.Slug, usage)

	var parsed struct {
		Answered bool   `json:"answered"`
		Refused  bool   `json:"refused"`
		Text     string `json:"text"`
	}
	if json.Unmarshal([]byte(text), &parsed) != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"error": "javob o'qilmadi"})
		return
	}
	answer := clampSupport(parsed.Text)
	if answer == "" {
		httpx.JSON(w, http.StatusOK, map[string]any{"error": "javob kelmadi"})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"answer": answer,
		// ⚠️ Passed through rather than inferred from the words: the panel does
		// not cache a refusal, and "is this a refusal" must not be a question
		// answered by reading the sentence.
		"refused": parsed.Refused || !parsed.Answered,
	})
}
