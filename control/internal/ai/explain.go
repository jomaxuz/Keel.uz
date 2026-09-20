package ai

// ---- What a restaurant owner is told when the model cannot answer ----
//
// ⚠️ **A provider's error is not a sentence for a customer.** What the engines
// hand us reads like this:
//
//   gemini:gemini-3.7-flash: Rate limit exceeded (limit: 5 requests per minute
//   on Free Tier)… | claude:claude-opus-5: POST ".../v1/messages": 400 Bad
//   Request (Request-ID: req_011…) "Your credit balance is too low…"
//
// A restaurant owner cannot act on a word of it. It names models they have
// never heard of, an HTTP status, two request ids and a billing page that is
// not theirs — and the one thing they *can* read, "credit balance too low",
// sends them looking for a card of their own to top up.
//
// ⚠️ **We pay for the AI, so our bill is never their problem.** The owner pays
// Keel a monthly figure; whether our card worked this month is an operational
// fact about us. Telling a paying customer "the balance is too low" invites
// them to fix something they cannot fix and makes the platform look like it is
// running out of money — which is a far more expensive sentence than the outage
// it describes.
//
// ⚠️ **The real error is still written down**, on our side, by the caller. The
// sentence below is what a person reads; the log is what we debug from, and
// collapsing the two is how an outage becomes unexplainable.

import (
	"context"
	"errors"
	"strings"
)

// Trouble is the kind of failure, once the provider's words are thrown away.
type Trouble int

const (
	// TroubleBusy is a rate limit — minutes, not hours.
	TroubleBusy Trouble = iota
	// TroubleSpent is every quota exhausted; waiting is measured in hours.
	TroubleSpent
	// TroubleOff is the platform's own problem: no key, a rejected key, a bill
	// of ours. ⚠️ Never described as money to the restaurant.
	TroubleOff
	// TroubleSlow is a timeout or a dropped connection.
	TroubleSlow
	// TroubleUnknown is everything else.
	TroubleUnknown
)

// Classify reads a failure without believing its wording.
//
// ⚠️ **Matched on substrings because that is all a provider guarantees.** The
// status codes are inside prose ("400 Bad Request"), the messages are English
// sentences that change, and neither provider gives a stable machine code for
// "your card failed". So this is deliberately loose and its default is the
// safe one: an unrecognised failure is "could not answer", never "we have a
// billing problem".
func Classify(err error) Trouble {
	if err == nil {
		return TroubleUnknown
	}
	if AllExhausted(err) {
		return TroubleSpent
	}
	if errors.Is(err, ErrNoKey) {
		return TroubleOff
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) {
		return TroubleSlow
	}
	s := strings.ToLower(err.Error())
	switch {
	case has(s, "credit balance", "billing", "payment required", "402",
		"insufficient", "quota exceeded for your current plan",
		"invalid api key", "invalid x-api-key", "authentication",
		"unauthorized", "401", "permission denied", "403"):
		return TroubleOff
	case has(s, "rate limit", "429", "too many requests", "overloaded", "529"):
		return TroubleBusy
	case has(s, "timeout", "deadline", "connection reset", "eof",
		"no such host", "temporary failure"):
		return TroubleSlow
	}
	return TroubleUnknown
}

func has(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// Explain is the one sentence the restaurant sees.
//
// ⚠️ **It never names a model, a provider, a status code or a bill.** Those
// four are the whole content of a raw engine error and none of them is
// something an owner can do anything with. What is left is the only useful
// half: whether to try again in a minute, later today, or to expect us to fix
// something.
func Explain(err error, lang string) string {
	words, ok := troubleWords[strings.ToLower(strings.TrimSpace(lang))]
	if !ok {
		words = troubleWords["uz"]
	}
	return words[Classify(err)]
}

var troubleWords = map[string]map[Trouble]string{
	"uz": {
		TroubleBusy:  "AI hozir band — bir daqiqadan keyin qayta urinib ko'ring.",
		TroubleSpent: "AI bugungi chegarasiga yetdi — bir necha soatdan keyin qayta urinib ko'ring.",
		// ⚠️ "Keel tomonida" and nothing more: it is our problem, it is being
		// dealt with, and there is nothing the restaurant can do to help.
		TroubleOff:     "AI vaqtincha ishlamayapti — nosozlik Keel tomonida va biz xabardormiz. Birozdan keyin urinib ko'ring.",
		TroubleSlow:    "AI javob bermadi — qayta urinib ko'ring.",
		TroubleUnknown: "AI hozir javob bera olmadi — birozdan keyin qayta urinib ko'ring.",
	},
	"ru": {
		TroubleBusy:    "ИИ сейчас занят — попробуйте через минуту.",
		TroubleSpent:   "ИИ достиг сегодняшнего предела — попробуйте через несколько часов.",
		TroubleOff:     "ИИ временно не работает — неполадка на стороне Keel, мы уже знаем. Попробуйте чуть позже.",
		TroubleSlow:    "ИИ не ответил — попробуйте ещё раз.",
		TroubleUnknown: "ИИ сейчас не смог ответить — попробуйте чуть позже.",
	},
	"en": {
		TroubleBusy:    "The AI is busy right now — try again in a minute.",
		TroubleSpent:   "The AI has reached today's limit — try again in a few hours.",
		TroubleOff:     "The AI is temporarily unavailable — the fault is on Keel's side and we know about it. Try again shortly.",
		TroubleSlow:    "The AI did not answer — try again.",
		TroubleUnknown: "The AI could not answer just now — try again shortly.",
	},
}
