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
	var last error
	for _, m := range c {
		out, u, err := m.JSON(ctx, system, user, schema, effort)
		if err == nil {
			return out, u, nil
		}
		last = err
		// ⚠️ A cancelled request is the caller giving up, not the engine
		// failing — trying the next one would ignore a timeout somebody set.
		if ctx.Err() != nil {
			break
		}
	}
	if last == nil {
		last = errors.New("ai: no engine configured")
	}
	return "", Usage{}, last
}

// ErrNoKey is what an unconfigured engine returns.
//
// ⚠️ Its own error so a Chain can tell "not set up" from "tried and failed":
// the first is a platform half-configured and the second is worth an alert.
var ErrNoKey = errors.New("ai: no api key")
