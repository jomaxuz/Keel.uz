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
	OutputText string `json:"output_text"`
	Steps      []struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"steps"`
	// ⚠️ **Two spellings, because the documentation did not name the usage
	// fields and Google's APIs are not consistent about it.** Both are read and
	// whichever arrives is used; a missing count records zero rather than
	// failing the request, since the answer is worth more than the accounting.
	Usage struct {
		InputTokens   int64 `json:"input_tokens"`
		OutputTokens  int64 `json:"output_tokens"`
		CachedTokens  int64 `json:"cached_tokens"`
		PromptTokens  int64 `json:"prompt_token_count"`
		OutTokenCount int64 `json:"candidates_token_count"`
	} `json:"usage"`
	UsageMetadata struct {
		PromptTokenCount     int64 `json:"promptTokenCount"`
		CandidatesTokenCount int64 `json:"candidatesTokenCount"`
		CachedContentTokens  int64 `json:"cachedContentTokenCount"`
	} `json:"usageMetadata"`
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
	for i := len(g.Steps) - 1; i >= 0; i-- {
		for j := len(g.Steps[i].Content) - 1; j >= 0; j-- {
			if t := g.Steps[i].Content[j].Text; t != "" {
				return t
			}
		}
	}
	return ""
}

func (g geminiResponse) usage() Usage {
	u := Usage{
		Input:  first(g.Usage.InputTokens, g.Usage.PromptTokens, g.UsageMetadata.PromptTokenCount),
		Output: first(g.Usage.OutputTokens, g.Usage.OutTokenCount, g.UsageMetadata.CandidatesTokenCount),
		Cached: first(g.Usage.CachedTokens, g.UsageMetadata.CachedContentTokens),
	}
	return u
}

func first(vals ...int64) int64 {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
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
