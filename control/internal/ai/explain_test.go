package ai

import (
	"errors"
	"strings"
	"testing"
)

// The real thing, as both engines produced it on 2026-09-20. ⚠️ Kept verbatim:
// the classifier matches on substrings a provider never promised to keep, and
// a test written from a paraphrase would pass over the sentence that actually
// reached a restaurant.
const seen = `gemini:gemini-3.7-flash: gemini: Rate limit exceeded for model ` +
	`gemini-3.7-flash (limit: 5 requests per minute on Free Tier). Please ` +
	`retry in 17s or upgrade your tier at https://ai.dev/rate-limit. | ` +
	`claude:claude-opus-5: POST "https://api.anthropic.com/v1/messages": 400 ` +
	`Bad Request (Request-ID: req_011CfEUoSmdsnMqThZjjF9Bc) ` +
	`{"type":"error","error":{"type":"invalid_request_error","message":"Your ` +
	`credit balance is too low to access the Anthropic API."}}`

// ⚠️ **Our bill is never the restaurant's problem.** They pay Keel a monthly
// figure; whether our card worked is an operational fact about us. "Credit
// balance is too low" sends a paying customer looking for an account of their
// own to top up, and makes the platform read as one that is running out of
// money.
func TestTheOwnerIsNeverToldAboutOurBill(t *testing.T) {
	for _, lang := range []string{"uz", "ru", "en", ""} {
		got := strings.ToLower(Explain(errors.New(seen), lang))
		for _, leak := range []string{
			"credit", "balance", "баланс", "hisob", "billing", "anthropic",
			"gemini", "claude", "api", "400", "429", "http", "req_",
			"free tier", "upgrade",
		} {
			if strings.Contains(got, leak) {
				t.Fatalf("%q reached the owner in %q: %q", leak, lang, got)
			}
		}
		if got == "" {
			t.Fatalf("no sentence at all for %q", lang)
		}
	}
}

// ⚠️ **What the owner needs is when to try again**, and the three answers are
// different: a minute, a few hours, or "we are fixing it".
func TestWaitingAdviceMatchesTheFailure(t *testing.T) {
	cases := []struct {
		err  error
		want Trouble
	}{
		{errors.New("Rate limit exceeded, retry in 17s"), TroubleBusy},
		{errors.New("429 Too Many Requests"), TroubleBusy},
		{errors.New("Your credit balance is too low"), TroubleOff},
		{errors.New("invalid x-api-key"), TroubleOff},
		{ErrNoKey, TroubleOff},
		{errors.New("context deadline exceeded"), TroubleSlow},
		{errors.New("something nobody has seen"), TroubleUnknown},
	}
	for _, c := range cases {
		if got := Classify(c.err); got != c.want {
			t.Fatalf("%v classified as %v, want %v", c.err, got, c.want)
		}
	}
}

// ⚠️ **An unrecognised failure is never read as a billing problem.** The
// default has to be the harmless one: "could not answer" is always safe to
// say, and "the fault is on our side" said wrongly is an apology for an
// outage that did not happen.
func TestAnUnknownFailureIsNotBlamedOnUs(t *testing.T) {
	uz := Explain(errors.New("boom"), "uz")
	if uz == troubleWords["uz"][TroubleOff] {
		t.Fatal("an unrecognised failure is being reported as our own outage")
	}
}
