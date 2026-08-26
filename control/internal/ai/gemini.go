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
	"time"
)

// Gemini is Google's engine.
type Gemini struct {
	Key   string
	Model string
}

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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://generativelanguage.googleapis.com/v1beta/interactions",
		bytes.NewReader(raw))
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
	if out.Error.Message != "" {
		return "", Usage{}, fmt.Errorf("gemini: %s", out.Error.Message)
	}
	if res.StatusCode >= 500 {
		return "", Usage{}, overloaded{msg: "gemini: " + short(payload)}
	}
	// ⚠️ **A spent quota is told apart from every other refusal, because the
	// answer to it is completely different: wait, and do not ask again today.**
	//
	// The free tier is twenty requests a day. A rejected request still counts
	// against it, so a caller that keeps trying after "you exceeded your
	// quota" spends the rest of the day's allowance discovering the same
	// sentence — and pins the quota shut for the restaurants whose briefings
	// have not been built yet. Seven restaurants need seven requests; the
	// allowance is enough right up until something retries into it.
	if res.StatusCode == http.StatusTooManyRequests {
		return "", Usage{}, Exhausted{msg: "gemini: " + short(payload)}
	}
	if res.StatusCode >= 400 {
		// ⚠️ The body rather than the status: "429" tells an operator nothing
		// and "quota exceeded for this project" tells them everything.
		return "", Usage{}, fmt.Errorf("gemini: %s", short(payload))
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
