package ai

// ---- Gemini, over its interactions endpoint ----
//
// ⚠️ **Raw HTTP rather than a Go SDK**, matching how this codebase already
// talks to Telegram and to the fiscal register: one file, one shape, and
// nothing to keep in step with a dependency for four fields.
//
// ⚠️ **The request shape was read from Google's own documentation, not
// recalled.** It changed: the endpoint everybody knows — `generateContent` with
// `contents[].parts[].text` — is not what the current docs describe. This uses
// `/v1beta/interactions` with `input`, `system_instruction` and
// `response_format`, which is what they show today.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Gemini is Google's engine.
type Gemini struct {
	Key   string
	Model string
	// Endpoint is where the request goes; empty means Google.
	//
	// ⚠️ A seam for the tests, and only for them. Which refusal is retried and
	// which is not is a rule about status codes and response bodies, and the
	// only honest way to test it is to serve those bodies — the alternative is
	// asserting on the shape of the code, which passes for a version of that
	// code that does not work.
	Endpoint string
}

const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/interactions"

// DefaultGeminiModel is what a briefing is worth.
//
// ⚠️ **Flash, deliberately.** The work is choosing four things out of eleven and
// saying each in a sentence — not deep reasoning — and this runs once a morning
// for every restaurant on the platform. The effort parameter carries the cases
// that need more thought.
const DefaultGeminiModel = "gemini-3.7-flash"

func (g Gemini) Name() string {
	if g.Model != "" {
		return "gemini:" + g.Model
	}
	return "gemini:" + DefaultGeminiModel
}

func (g Gemini) JSON(
	ctx context.Context, system, user string, schema map[string]any, effort string,
) (string, Usage, error) {
	if g.Key == "" {
		return "", Usage{}, ErrNoKey
	}
	model := g.Model
	if model == "" {
		model = DefaultGeminiModel
	}
	body := map[string]any{
		"model": model,
		"input": user,
		// ⚠️ Sent as its own field rather than glued to the front of the input.
		// The instructions are what stays identical across every restaurant and
		// every morning, and a provider that can be told which half is which is
		// one that can cache it.
		"system_instruction": system,
		"response_format": map[string]any{
			"type":      "text",
			"mime_type": "application/json",
			"schema":    schema,
		},
		"generation_config": map[string]any{
			"thinking_level": thinkingFor(effort),
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", Usage{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	// ⚠️ **One retry on a transient refusal, because the very first real call
	// hit one.** "gemini-3.7-flash is currently experiencing high demand,
	// spikes in demand are usually temporary. Please try again later" — a 500
	// that the same request answered on the next attempt. Without this a
	// briefing simply does not appear that morning, and a restaurant has no way
	// to tell an overloaded model from a broken key.
	//
	// ⚠️ **One, not a loop, and the retry lives here rather than inside
	// `once`.** Putting it in the attempt itself makes the attempt call itself
	// — which on a service that stays overloaded is unbounded recursion
	// against something already asking to be left alone. This runs unattended
	// for every restaurant on the platform, so a retry storm would make it
	// worse for everybody including us. Once, after a pause, and then the
	// chain's other engine has its turn.
	text, usage, err := g.once(ctx, raw)
	var over overloaded
	if errors.As(err, &over) && ctx.Err() == nil {
		select {
		case <-time.After(2 * time.Second):
			return g.once(ctx, raw)
		case <-ctx.Done():
			return "", Usage{}, ctx.Err()
		}
	}
	return text, usage, err
}

// Exhausted is a quota that will not clear soon.
//
// ⚠️ Its own type so a caller can back off for hours rather than minutes. A
// transient overload wants a second attempt; a spent daily allowance wants
// silence until it resets, and treating them alike turns twenty wasted requests
// into two hundred.
type Exhausted struct{ msg string }

func (e Exhausted) Error() string { return e.msg }

// overloaded is a refusal worth trying again, told apart from one that is not.
//
// ⚠️ A 500 is temporary; a 429 quota and a 401 key are not, and retrying those
// spends a second attempt learning what the first one already said.
type overloaded struct{ msg string }

func (e overloaded) Error() string { return e.msg }

// once is one attempt: send the bytes, read the answer.
//
// ⚠️ Split out so the retry above re-sends the *same* body rather than
// rebuilding it — a second marshal is a second chance for the two attempts to
// differ, which is the sort of difference nobody would look for.
func (g Gemini) once(ctx context.Context, raw []byte) (string, Usage, error) {
	url := g.Endpoint
	if url == "" {
		url = geminiEndpoint
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", Usage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	// ⚠️ The header, not `?key=` in the query string. A key in a URL reaches
	// every proxy log between here and Google.
	req.Header.Set("x-goog-api-key", g.Key)

	res, err := (&http.Client{Timeout: 95 * time.Second}).Do(req)
	if err != nil {
		return "", Usage{}, err
	}
	defer res.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return "", Usage{}, err
	}

	var out geminiResponse
	if err := json.Unmarshal(payload, &out); err != nil {
		return "", Usage{}, fmt.Errorf("gemini: %s", short(payload))
	}
	// ⚠️ **The status decides what kind of failure this is; the body only
	// decides how it reads.** These were the other way round — the
	// `error.message` branch came first and returned a plain error — and that
	// made the retry above unreachable for the one refusal it was written for.
	// Google sends "currently experiencing high demand … please try again
	// later" as a 500 *with* `error.message` set, so every real overload took
	// the message branch, lost its type, and was never tried a second time.
	// The comment above `once` quotes that exact sentence as the reason the
	// retry exists, and the retry had not fired once.
	//
	// ⚠️ The body rather than the status in the text: "429" tells an operator
	// nothing and "quota exceeded for this project" tells them everything.
	// Google's own sentence when there is one, the raw payload when there is
	// not — a 500 from a proxy in front of the API has no JSON in it at all.
	msg := "gemini: " + short(payload)
	if out.Error.Message != "" {
		msg = "gemini: " + out.Error.Message
	}
	switch {
	case res.StatusCode >= 500:
		return "", Usage{}, overloaded{msg: msg}
	// ⚠️ **A spent quota is told apart from every other refusal, because the
	// answer to it is completely different: wait, and do not ask again today.**
	//
	// The free tier is twenty requests a day per model. A rejected request
	// still counts against it, so a caller that keeps trying after "you
	// exceeded your quota" spends the rest of that model's allowance
	// discovering the same sentence. The chain's answer is the next model;
	// this type is what tells the panel when there is no next model left.
	case res.StatusCode == http.StatusTooManyRequests:
		return "", Usage{}, Exhausted{msg: msg}
	// ⚠️ A 200 carrying `error.message` is still a failure. It is not a
	// documented shape, but a body that names an error is not an answer, and
	// falling through would hand `text()` an empty response and report
	// "bo'sh javob" — the least informative sentence available.
	case res.StatusCode >= 400 || out.Error.Message != "":
		return "", Usage{}, errors.New(msg)
	}
	text := out.text()
	if text == "" {
		return "", Usage{}, fmt.Errorf("gemini: bo'sh javob")
	}
	return text, out.usage(), nil
}

type geminiResponse struct {
	// ⚠️ **Documented and absent from every real response so far.** The docs
	// describe `output_text` as a convenience for the last text block; a live
	// call returns only `steps`. Kept and preferred because a field that
	// appears later is free, and walking the steps is what actually runs.
	OutputText string `json:"output_text"`
	Status     string `json:"status"`
	Steps      []struct {
		// ⚠️ **"thought" or "model_output", and the difference matters.** A
		// thinking step carries no content today — and taking one that did and
		// printing it as a restaurant's briefing would put the model's working
		// out on an owner's screen. Filtered by type rather than by whichever
		// text happens to be last.
		Type    string `json:"type"`
		Content []struct {
			Text string `json:"text"`
			Type string `json:"type"`
		} `json:"content"`
	} `json:"steps"`
	// ⚠️ **The names were read off a live response, not the documentation**,
	// which does not list them. The obvious guesses — `input_tokens`,
	// `promptTokenCount` — are all wrong: it is `total_input_tokens`. Coding
	// from the docs here would have recorded zero for every request, silently,
	// on the one screen that exists to say what this costs.
	Usage struct {
		TotalInput  int64 `json:"total_input_tokens"`
		TotalOutput int64 `json:"total_output_tokens"`
		TotalCached int64 `json:"total_cached_tokens"`
		// Thinking is billed and is not output. Counted with the output
		// because that is where it is paid for, and dropping it would
		// under-report the cost by most of it: 262 of 274 tokens on the very
		// first live call.
		TotalThought int64 `json:"total_thought_tokens"`
	} `json:"usage"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// text is the answer, from whichever place this response carries it.
//
// ⚠️ `output_text` is documented as a convenience for the last text block and
// `steps` as the full account. Preferring the convenience and falling back to
// walking the steps means a response shaped either way still produces an
// answer — and the fallback is what runs if the convenience field is ever
// dropped.
func (g geminiResponse) text() string {
	if g.OutputText != "" {
		return g.OutputText
	}
	// ⚠️ Backwards, and only through model output. The answer is the last thing
	// the model said; a thinking step is not an answer even when it has text
	// in it.
	for i := len(g.Steps) - 1; i >= 0; i-- {
		if g.Steps[i].Type == "thought" {
			continue
		}
		for j := len(g.Steps[i].Content) - 1; j >= 0; j-- {
			c := g.Steps[i].Content[j]
			if c.Text != "" && (c.Type == "" || c.Type == "text") {
				return c.Text
			}
		}
	}
	return ""
}

func (g geminiResponse) usage() Usage {
	return Usage{
		Input:  g.Usage.TotalInput,
		Cached: g.Usage.TotalCached,
		Output: g.Usage.TotalOutput + g.Usage.TotalThought,
	}
}

// thinkingFor maps our three words onto Gemini's.
//
// ⚠️ The two providers name effort differently and the caller should not have
// to know that — the whole point of the seam. An unknown value is "low", the
// cheap end, because a typo must not silently cost a platform money.
func thinkingFor(effort string) string {
	switch effort {
	case "medium":
		return "medium"
	case "high":
		return "high"
	}
	return "low"
}

// short is enough of a body to diagnose with and not enough to fill a log.
func short(b []byte) string {
	const most = 400
	if len(b) > most {
		return string(b[:most]) + "…"
	}
	return string(b)
}

// FreeGeminiModels is every Gemini model with its own free daily allowance,
// strongest first.
//
// ⚠️ **A list rather than one name, because the free tier's quota is per
// model.** A spent allowance on `gemini-3.7-flash` says nothing about
// `gemini-2.5-flash`: they are separate counters on the same key, and the
// morning briefing does not need the best model in the world — it needs a
// model. When the first is out, the next one is a second full allowance for
// the price of one more entry in a chain.
//
// ⚠️ **Ordered by capability, not by what is cheapest to us**, because on the
// free tier none of them costs anything: the only thing the order buys is a
// better sentence for the restaurants whose briefing is built first in the day.
//
// ⚠️ **A name that Google retires is not a failure worth handling.** It comes
// back as a 404 with the model named in the body, the chain writes it down and
// asks the next one — the same path as an exhausted quota. So a stale entry
// here costs one wasted request per briefing, never a missing briefing, and
// `GEMINI_MODEL` can fix it without a release.
var FreeGeminiModels = []string{
	"gemini-3.7-flash",
	"gemini-3.7-flash-lite",
	"gemini-2.5-flash",
	"gemini-2.5-flash-lite",
	"gemini-2.0-flash",
	"gemini-2.0-flash-lite",
}

// GeminiModels turns the configured setting into the models to try, in order.
//
// ⚠️ **A comma-separated list, and an empty setting means all of them.** The
// setting used to name one model and that is still what most people will
// write; naming one now means "only this one", which is the honest reading of
// a field somebody filled in deliberately — and it stays the way to pin the
// platform to a single model when a paid key makes the fallback pointless.
func GeminiModels(setting string) []string {
	if strings.TrimSpace(setting) == "" {
		return append([]string(nil), FreeGeminiModels...)
	}
	var out []string
	seen := map[string]bool{}
	for _, part := range strings.Split(setting, ",") {
		name := strings.TrimSpace(part)
		// ⚠️ Duplicates dropped: the same model twice is the same spent quota
		// twice, one wasted request per briefing to learn nothing new.
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	if len(out) == 0 {
		return append([]string(nil), FreeGeminiModels...)
	}
	return out
}

// GeminiChain is one engine per model, all on the same key.
//
// ⚠️ **Built here rather than in the handler** so the campaign writer and the
// briefing cannot end up with different fallback lists — the same reason the
// prompts and schemas live on this side of the seam.
func GeminiChain(key, setting string) []Model {
	if key == "" {
		return nil
	}
	models := GeminiModels(setting)
	out := make([]Model, 0, len(models))
	for _, m := range models {
		out = append(out, Gemini{Key: key, Model: m})
	}
	return out
}
