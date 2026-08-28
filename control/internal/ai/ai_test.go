package ai

import (
	"context"
	"encoding/json"
	"errors"
	"os"
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

// ⚠️ **The usage field names were read off a live response, not the
// documentation, which does not list them.** Every obvious guess is wrong:
// `input_tokens`, `promptTokenCount`, `prompt_token_count` — it is
// `total_input_tokens`. Coding from the docs would have recorded zero for every
// request, silently, on the one screen that exists to say what this costs.
func TestUsageUsesTheNamesTheApiActuallySends(t *testing.T) {
	// Byte for byte the shape of the first live call.
	raw := `{"status":"completed","usage":{"total_tokens":274,` +
		`"total_input_tokens":11,"total_cached_tokens":0,` +
		`"total_output_tokens":1,"total_thought_tokens":262},` +
		`"steps":[{"type":"thought","signature":"..."},` +
		`{"type":"model_output","content":[{"text":"OK","type":"text"}]}]}`
	var r geminiResponse
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	u := r.usage()
	if u.Input != 11 {
		t.Fatalf("input tokens read as %d, want 11", u.Input)
	}
	// ⚠️ Thinking is billed and is not output. Dropping it would under-report
	// the cost by most of it — 262 of 274 tokens on that very first call.
	if u.Output != 263 {
		t.Fatalf("output tokens read as %d, want 263 (1 output + 262 thought)", u.Output)
	}
}

// ⚠️ **A thinking step is not an answer, even when it has text in it.** Taking
// one and printing it as a restaurant's briefing would put the model's working
// out on an owner's screen. Filtered by step type rather than by whichever text
// happens to be last.
func TestAThoughtIsNeverTheAnswer(t *testing.T) {
	raw := `{"steps":[{"type":"thought","content":[{"text":"let me think","type":"text"}]},` +
		`{"type":"model_output","content":[{"text":"{\"cards\":[]}","type":"text"}]}]}`
	var r geminiResponse
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if got := r.text(); got != `{"cards":[]}` {
		t.Fatalf("the model's thinking was returned as the answer: %q", got)
	}
}

// ⚠️ **`output_text` is documented and absent from every real response so far.**
// Kept and preferred because a field that appears later is free; walking the
// steps is what actually runs today.
func TestTheDocumentedFieldStillWinsWhenPresent(t *testing.T) {
	var r geminiResponse
	r.OutputText = `{"a":1}`
	if r.text() != `{"a":1}` {
		t.Fatal("the documented convenience field was ignored")
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

// ⚠️ **A 500 is temporary; a 429 quota and a 401 key are not.** Retrying those
// spends a second attempt learning what the first one already said — and on a
// quota, spends one of the requests the quota is counting.
func TestOnlyAnOverloadIsWorthRetrying(t *testing.T) {
	var over overloaded
	if !errors.As(overloaded{msg: "high demand"}, &over) {
		t.Fatal("an overload is no longer recognised as one")
	}
	if errors.As(errors.New("gemini: quota exceeded"), &over) {
		t.Fatal("a quota refusal would now be retried")
	}
}

// ⚠️ **The retry lives outside the attempt, and that is not a style choice.**
// Putting it inside makes the attempt call itself — unbounded recursion against
// a service that is already asking to be left alone. This runs unattended for
// every restaurant on the platform, so a retry storm would make an overload
// worse for everybody, including us.
func TestTheRetryIsNotInsideTheAttempt(t *testing.T) {
	src := readSourceFile(t, "gemini.go")
	i := strings.Index(src, "func (g Gemini) once(")
	if i < 0 {
		t.Fatal("the single attempt is gone")
	}
	if strings.Contains(src[i:], "g.once(") {
		t.Fatal("the attempt calls itself — a persistent 500 would recurse forever")
	}
	// And exactly one retry, not a loop.
	head := src[:i]
	if strings.Count(head, "g.once(ctx, raw)") != 2 {
		t.Fatalf("expected one attempt and one retry, found %d calls",
			strings.Count(head, "g.once(ctx, raw)"))
	}
}

func readSourceFile(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// ⚠️ **Every engine's failure, not the last one's.**
//
// Reporting only the last was actively misleading: with Gemini first and Claude
// second, a Gemini failure showed the owner Anthropic's "credit balance too
// low" — an accurate sentence about an engine that was never the problem,
// naming a bill they may not even have. They would go and top up an account
// that changes nothing.
func TestTheChainReportsEveryFailure(t *testing.T) {
	c := Chain{
		fake{name: "gemini:flash", err: errors.New("high demand")},
		fake{name: "claude:opus", err: errors.New("credit balance too low")},
	}
	_, _, err := c.JSON(context.Background(), "s", "u", nil, "low")
	if err == nil {
		t.Fatal("two failures reported success")
	}
	for _, want := range []string{"gemini", "high demand", "claude", "credit balance"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("%q missing from %q", want, err)
		}
	}
}

// An exhausted free-tier quota on one model is not an exhausted key.
//
// Google meters each model separately, so the answer to "gemini-3.7-flash has
// no allowance left today" is the next model, not the next morning. Before
// this the platform had one Gemini engine and a spent quota ended the Google
// half of the chain outright.
func TestEveryFreeGeminiModelBecomesItsOwnEngine(t *testing.T) {
	chain := GeminiChain("k", "")
	if len(chain) != len(FreeGeminiModels) {
		t.Fatalf("got %d engines, want %d", len(chain), len(FreeGeminiModels))
	}
	for i, m := range chain {
		want := "gemini:" + FreeGeminiModels[i]
		if m.Name() != want {
			t.Fatalf("engine %d is %q, want %q", i, m.Name(), want)
		}
	}
	if GeminiChain("", "") != nil {
		t.Fatal("no key must mean no engines")
	}
}

// A named model still means that model, and only that model.
func TestANamedGeminiModelPinsTheChain(t *testing.T) {
	got := GeminiModels(" gemini-2.5-flash , , gemini-2.5-flash ,gemini-2.0-flash ")
	want := []string{"gemini-2.5-flash", "gemini-2.0-flash"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if len(GeminiModels("  ,  ")) != len(FreeGeminiModels) {
		t.Fatal("a setting with no names left in it must mean the full list")
	}
}

// The reason a spent quota must survive the chain.
//
// The panel backs off for hours on `ai.Exhausted` and retries in minutes on
// anything else. Joining the failures into a plain string erased the type, so
// six spent free-tier quotas read as an ordinary error and the panel asked
// again ten minutes later — six more requests, all of them refused.
func TestASpentQuotaSurvivesTheChain(t *testing.T) {
	c := Chain{
		fake{name: "gemini:a", err: Exhausted{msg: "gemini: quota"}},
		fake{name: "gemini:b", err: errors.New("boom")},
	}
	_, _, err := c.JSON(context.Background(), "s", "u", nil, "low")
	var spent Exhausted
	if !errors.As(err, &spent) {
		t.Fatalf("errors.As lost the exhausted quota in %v", err)
	}
	for _, want := range []string{"gemini:a", "gemini:b", "quota", "boom"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not mention %q", err.Error(), want)
		}
	}
}

// Backing off for hours is for the day nothing can answer.
//
// The first free model being out of quota is the ordinary case now that the
// Gemini side is six of them — the answer to it is the next model. One engine
// that failed for any other reason is worth asking again in ten minutes.
func TestOnlyAWholeChainOfSpentQuotasIsExhausted(t *testing.T) {
	quota := func(name string) fake {
		return fake{name: name, err: Exhausted{msg: name + ": quota"}}
	}
	all := Chain{quota("gemini:a"), quota("gemini:b")}
	_, _, err := all.JSON(context.Background(), "s", "u", nil, "low")
	if !AllExhausted(err) {
		t.Fatalf("every engine was out of quota, got %v", err)
	}

	some := Chain{quota("gemini:a"), fake{name: "claude", err: errors.New("overloaded")}}
	_, _, err = some.JSON(context.Background(), "s", "u", nil, "low")
	if AllExhausted(err) {
		t.Fatalf("a transient failure must not buy an hours-long back-off: %v", err)
	}

	if AllExhausted(errors.New("boom")) {
		t.Fatal("an ordinary error is not an exhausted quota")
	}
	if !AllExhausted(Exhausted{msg: "gemini: quota"}) {
		t.Fatal("a bare spent quota still counts")
	}
}
