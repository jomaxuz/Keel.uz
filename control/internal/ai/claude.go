package ai

// ---- Claude, over the Anthropic SDK ----
//
// ⚠️ **The SDK here and raw HTTP for Gemini, and that is not an inconsistency
// to tidy up.** Anthropic publishes a Go SDK that carries the request shapes,
// the beta flags and the prompt-cache controls; Google's current endpoint is
// four fields of JSON. Using the SDK where one exists and HTTP where the whole
// surface fits on a screen is the smaller amount of code to be wrong in.

import (
	"context"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Claude is Anthropic's engine.
type Claude struct {
	Key   string
	Model string
}

// DefaultClaudeModel is what the briefing and the campaign writer run on.
const DefaultClaudeModel = "claude-opus-5"

func (c Claude) Name() string {
	if c.Model != "" {
		return "claude:" + c.Model
	}
	return "claude:" + DefaultClaudeModel
}

func (c Claude) JSON(
	ctx context.Context, system, user string, schema map[string]any, effort string,
) (string, Usage, error) {
	if c.Key == "" {
		return "", Usage{}, ErrNoKey
	}
	model := c.Model
	if model == "" {
		model = DefaultClaudeModel
	}
	client := anthropic.NewClient(option.WithAPIKey(c.Key))
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	adaptive := anthropic.ThinkingConfigAdaptiveParam{}
	res, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 4000,
		Thinking:  anthropic.ThinkingConfigParamUnion{OfAdaptive: &adaptive},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: effortFor(effort),
			Format: anthropic.JSONOutputFormatParam{Schema: schema},
		},
		System: []anthropic.TextBlockParam{{
			Text: system,
			// ⚠️ An hour, not the default five minutes: the daily sweep runs
			// through every restaurant back to back, and a platform with a
			// hundred of them takes longer than five minutes to get round.
			CacheControl: anthropic.CacheControlEphemeralParam{
				TTL: anthropic.CacheControlEphemeralTTLTTL1h,
			},
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
		},
	})
	if err != nil {
		return "", Usage{}, err
	}
	for _, block := range res.Content {
		if b, ok := block.AsAny().(anthropic.TextBlock); ok && b.Text != "" {
			return b.Text, Usage{
				Input:  res.Usage.InputTokens,
				Cached: res.Usage.CacheReadInputTokens,
				Output: res.Usage.OutputTokens,
			}, nil
		}
	}
	return "", Usage{}, errEmpty
}

// effortFor maps our three words onto Anthropic's.
//
// ⚠️ An unknown value is the cheap end, the same rule the Gemini side follows:
// a typo must not silently cost a platform money.
func effortFor(effort string) anthropic.OutputConfigEffort {
	switch effort {
	case "medium":
		return anthropic.OutputConfigEffortMedium
	case "high":
		return anthropic.OutputConfigEffortHigh
	}
	return anthropic.OutputConfigEffortLow
}

type aiErr string

func (e aiErr) Error() string { return string(e) }

const errEmpty = aiErr("claude: bo'sh javob")
