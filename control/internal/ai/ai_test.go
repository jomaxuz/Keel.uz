package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// A stand-in engine, so the chain can be tested without a network or a key.
type fake struct {
	name string
	out  string
	err  error
	// How many times it was asked, which is what the fallback rules are about.
	calls *int
}

func (f fake) Name() string { return f.name }

func (f fake) JSON(
	ctx context.Context, system, user string, schema map[string]any, effort string,
) (string, Usage, error) {
	if f.calls != nil {
		*f.calls++
	}
	if f.err != nil {
		return "", Usage{}, f.err
	}
	return f.out, Usage{Input: 10, Output: 5}, nil
}

// ⚠️ **A fallback, not a race.** Running both would double the cost of every
// briefing to save a few seconds nobody is waiting for — this runs at six in
// the morning and is read at nine. The second engine is asked only when the
// first could not answer at all.
func TestTheSecondEngineIsOnlyAskedWhenTheFirstFails(t *testing.T) {
	first, second := 0, 0
	c := Chain{
		fake{name: "a", out: `{"ok":1}`, calls: &first},
		fake{name: "b", out: `{"ok":2}`, calls: &second},
	}
	got, _, err := c.JSON(context.Background(), "s", "u", nil, "low")
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"ok":1}` {
		t.Fatalf("the wrong engine answered: %q", got)
	}
	if second != 0 {
		t.Fatal("the second engine was asked while the first was working")
	}
}

func TestAFailedEngineFallsThrough(t *testing.T) {
	second := 0
	c := Chain{
		fake{name: "a", err: errors.New("credit balance too low")},
		fake{name: "b", out: `{"ok":2}`, calls: &second},
	}
	got, _, err := c.JSON(context.Background(), "s", "u", nil, "low")
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"ok":2}` || second != 1 {
		t.Fatalf("the fallback did not run: %q, calls=%d", got, second)
	}
}

// ⚠️ **A cancelled request is the caller giving up, not the engine failing.**
// Trying the next one would ignore a timeout somebody set — and on a sweep
// across a hundred restaurants that turns one slow morning into two.
func TestACancelledRequestStopsTheChain(t *testing.T) {
	second := 0
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := Chain{
		fake{name: "a", err: errors.New("context canceled")},
		fake{name: "b", out: `{"ok":2}`, calls: &second},
	}
	if _, _, err := c.JSON(ctx, "s", "u", nil, "low"); err == nil {
		t.Fatal("a cancelled request was answered anyway")
	}
	if second != 0 {
		t.Fatal("the chain kept going after the caller gave up")
	}
}

// An empty chain is a platform with no keys, and it says so rather than
// answering with an empty string that would parse as no cards.
func TestNoEngineIsAnError(t *testing.T) {
	if _, _, err := (Chain{}).JSON(context.Background(), "s", "u", nil, "low"); err == nil {
		t.Fatal("a platform with no keys reported success")
	}
}

// ⚠️ **An unknown effort is the cheap end**, on both engines and for the same
// reason: a typo must not silently cost a platform money.
func TestAnUnknownEffortIsCheap(t *testing.T) {
	if thinkingFor("enormous") != "low" || thinkingFor("") != "low" {
		t.Fatal("a typo would buy the expensive setting")
	}
	if thinkingFor("high") != "high" || thinkingFor("medium") != "medium" {
		t.Fatal("a real effort level stopped working")
	}
}

// ⚠️ Google's docs did not name the usage fields, and their APIs are not
// consistent about it — so both spellings are read and whichever arrives is
// used. A missing count is zero, not a failed request: the answer is worth more
// than the accounting.
func TestUsageIsReadFromEitherSpelling(t *testing.T) {
	var a geminiResponse
	a.Usage.InputTokens, a.Usage.OutputTokens = 7, 3
	if u := a.usage(); u.Input != 7 || u.Output != 3 {
		t.Fatalf("%+v", u)
	}
	var b geminiResponse
	b.UsageMetadata.PromptTokenCount, b.UsageMetadata.CandidatesTokenCount = 11, 4
	if u := b.usage(); u.Input != 11 || u.Output != 4 {
		t.Fatalf("%+v", u)
	}
	// Neither present: zero, and no error anywhere.
	if u := (geminiResponse{}).usage(); u.Input != 0 || u.Output != 0 {
		t.Fatalf("%+v", u)
	}
}

// ⚠️ `output_text` is documented as a convenience for the last text block and
// `steps` as the full account. Preferring the convenience and walking the steps
// when it is absent means a response shaped either way still produces an
// answer — and the fallback is what runs if the convenience field is dropped.
func TestTheAnswerIsFoundEitherWay(t *testing.T) {
	var r geminiResponse
	r.OutputText = "{\"a\":1}"
	if r.text() != "{\"a\":1}" {
		t.Fatal("the convenience field was ignored")
	}

	var walk geminiResponse
	walk.Steps = []struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}{
		{Content: []struct {
			Text string `json:"text"`
		}{{Text: "thinking"}, {Text: "{\"b\":2}"}}},
	}
	if got := walk.text(); got != "{\"b\":2}" {
		t.Fatalf("the last text block was not taken: %q", got)
	}
	if (geminiResponse{}).text() != "" {
		t.Fatal("an empty response produced text")
	}
}

// The chain names both engines, so a log line says which answered.
func TestTheChainNamesItsEngines(t *testing.T) {
	c := Chain{Claude{}, Gemini{}}
	name := c.Name()
	if !strings.Contains(name, "claude") || !strings.Contains(name, "gemini") {
		t.Fatalf("a log line could not say which engine answered: %q", name)
	}
}
