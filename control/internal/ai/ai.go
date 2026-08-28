// Package ai is one interface and two engines behind it.
//
// ⚠️ **The prompts, the schemas and the validation stay shared; only the wire
// differs.** The briefing's rules about what may be said, the campaign writer's
// refusal to invent a discount, the check that a card names a real fact —
// none of that is a property of who generated the text, and a second copy of
// any of it would be a second set of rules that agree until the day they do
// not. This is the shape `lib/map` already uses for its three map providers,
// and it exists for the reason written there: four files with "change this one
// to switch provider" in each is not one file.
//
// ⚠️ **Two engines because one of them can be unreachable for reasons that have
// nothing to do with the code.** Anthropic billing needs a card that works
// internationally, and the first thing this platform's own key did was answer
// "credit balance too low". A restaurant's morning briefing should not depend
// on which payment rails were available to us that month.
package ai

import (
	"context"
	"errors"
	"strings"
)

// Usage is what one request cost, in tokens.
//
// ⚠️ **Tokens, never money**, the rule the billing log already follows: a rate
// written into code is a number that quietly stops being right, and the two
// providers do not price alike anyway.
type Usage struct {
	Input  int64
	Cached int64
	Output int64
}

// Model is one engine that can be asked for JSON.
//
// ⚠️ **JSON only, and the schema is a parameter.** Both providers can be told
// the exact shape they must answer in, and asking for free text and parsing it
// afterwards would give up the one guarantee that keeps the rest of the feature
// honest — a card is dropped unless it names a fact we sent, and that check
// needs a `key` field to exist.
type Model interface {
	// Name is what goes in the log and the usage screen.
	Name() string
	// JSON asks for one answer matching the schema.
	//
	// `effort` is "low" | "medium" | "high": the briefing chooses from a list
	// we prepared, the campaign writer writes something that goes out over a
	// restaurant's name to a thousand of its guests, and those are not the same
	// amount of thinking.
	JSON(ctx context.Context, system, user string, schema map[string]any, effort string) (string, Usage, error)
}

// Chain tries each engine in turn.
//
// ⚠️ **A fallback, not a race.** Both would double the cost of every briefing
// to save a few seconds nobody is waiting for — this runs at six in the morning
// and is read at nine. The second engine is asked only when the first could not
// answer at all.
//
// ⚠️ **And it does not fall back on a refusal.** A model declining to write
// something is an answer, and asking a different one until somebody agrees is
// how a feature that was careful about what it says stops being careful.
type Chain []Model

func (c Chain) Name() string {
	names := make([]string, 0, len(c))
	for _, m := range c {
		names = append(names, m.Name())
	}
	return strings.Join(names, "→")
}

func (c Chain) JSON(
	ctx context.Context, system, user string, schema map[string]any, effort string,
) (string, Usage, error) {
	// ⚠️ **Every engine's failure, not the last one's.**
	//
	// Reporting only the last was actively misleading: with Gemini first and
	// Claude second, a Gemini failure showed the owner Anthropic's "credit
	// balance too low" — an accurate sentence about an engine that was never
	// the problem, naming a bill they may not even have. They would go and top
	// up an account that changes nothing.
	//
	// Both, in order, so the sentence says what actually happened: the first
	// engine failed *and* the second could not cover for it.
	var failures []error
	var lines []string
	for _, m := range c {
		out, u, err := m.JSON(ctx, system, user, schema, effort)
		if err == nil {
			return out, u, nil
		}
		failures = append(failures, err)
		lines = append(lines, m.Name()+": "+err.Error())
		// ⚠️ A cancelled request is the caller giving up, not the engine
		// failing — trying the next one would ignore a timeout somebody set.
		if ctx.Err() != nil {
			break
		}
	}
	if len(failures) == 0 {
		return "", Usage{}, errors.New("ai: no engine configured")
	}
	return "", Usage{}, chainError{text: strings.Join(lines, " | "), causes: failures}
}

// chainError is every engine's failure, still inspectable.
//
// ⚠️ **The message is the engines in order; the type keeps what they were.**
// Flattening the failures into one string lost the one distinction the caller
// acts on — `ai.Exhausted` means "wait hours, not minutes", and an
// `errors.New` of the same words means nothing to `errors.As`. That mattered
// the moment the Gemini side became several models on one key: with the joined
// string, six spent free-tier quotas reported themselves as an ordinary error
// and the panel asked again ten minutes later, six more times.
type chainError struct {
	text   string
	causes []error
}

func (e chainError) Error() string { return e.text }

// Unwrap gives `errors.Is` and `errors.As` every cause, not just the last.
func (e chainError) Unwrap() []error { return e.causes }

// ErrNoKey is what an unconfigured engine returns.
//
// ⚠️ Its own error so a Chain can tell "not set up" from "tried and failed":
// the first is a platform half-configured and the second is worth an alert.
var ErrNoKey = errors.New("ai: no api key")

// AllExhausted reports that nothing is left to try until a quota resets.
//
// ⚠️ **Every engine, not any one of them.** `errors.As` finding a spent quota
// somewhere in the chain is the wrong question now that the Gemini side is six
// models: the first one being out is the ordinary case, and the answer to it
// is the second model, not a wait of several hours. Backing off that long
// belongs to the day nothing can answer — and a Claude 500 in the same chain
// is worth trying again in ten minutes, so one non-quota failure is enough to
// make this false.
func AllExhausted(err error) bool {
	var spent Exhausted
	if errors.As(err, &spent) && !isChain(err) {
		return true
	}
	var c chainError
	if !errors.As(err, &c) || len(c.causes) == 0 {
		return false
	}
	for _, cause := range c.causes {
		if !AllExhausted(cause) {
			return false
		}
	}
	return true
}

func isChain(err error) bool {
	var c chainError
	return errors.As(err, &c)
}
