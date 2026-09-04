// Package insight is what the restaurant should look at today.
//
// ⚠️ **The numbers are computed here; the words come from a model.** Everything
// an owner asks an assistant — who stopped coming, what is not selling, which
// store has not been counted — is already exact arithmetic over data this
// system owns. Asking a language model to derive those figures would make them
// occasionally wrong, and the cost of that is asymmetric in a way worth
// spelling out: an owner who catches one wrong number stops believing the
// other four. The feature is not switched off at that point — it is simply
// never read again.
//
// So the split is absolute. This package produces Facts: keyed, exact, with
// their own arithmetic and their own tests. The model receives them and does
// the one job it is better at than any rule we could write — deciding which of
// eleven true things matters this morning, and saying it in one sentence in the
// owner's language.
//
// ⚠️ **A card that names no Fact is dropped.** That is the seam that makes the
// rule enforceable rather than aspirational: the model answers with fact keys,
// never with figures, and a key we did not send cannot be rendered. An invented
// statistic has nowhere to live.
package insight

import "sort"

// Area is the part of the business a fact belongs to.
//
// It exists so the briefing cannot become five cards about the same thing: a
// quiet week shows up in sales, in guests and in the menu at once, and three
// sentences about one problem read as three problems.
type Area string

const (
	Guests Area = "guests"
	Menu   Area = "menu"
	Stock  Area = "stock"
	Team   Area = "team"
	Money  Area = "money"
)

// Action is what the panel can offer to do about a fact.
//
// ⚠️ **An insight with no button is a paragraph, and paragraphs are not read.**
// Every action here is a screen that already exists — the point is that the
// owner reaches it from the sentence that explained why they should.
type Action string

const (
	NoAction    Action = ""
	NewCampaign Action = "campaign"  // CRM → segment → send
	Shopping    Action = "shopping"  // the buying list
	Purchases   Action = "purchases" // book the delivery notes in
	Stocktake   Action = "stocktake" // count a store
	OpenMenu    Action = "menu"      // the dish's card
	TechCards   Action = "techcards" // write the cards the store is missing
	OpenTeam    Action = "team"      // the team report
	OpenReports Action = "reports"
)

// Fact is one true, quantified thing about this restaurant.
//
// ⚠️ **`Numbers` is exact and travels with the card to the screen.** The
// sentence an owner reads is the model's; the figure printed beside it is
// this. Somebody who wants to check can, in the same glance — which is the
// only reason to trust the sentence at all.
type Fact struct {
	// Key is stable across days and releases: it is what the model answers
	// with, what the panel routes on, and what a test asserts.
	Key  string `json:"key"`
	Area Area   `json:"area"`

	// Weight is how much this matters, before the model sees it — money at
	// risk, roughly, in so'm. It orders the list and decides what is dropped
	// when there are more facts than a morning has attention.
	Weight int64 `json:"-"`

	// Numbers are the figures themselves, named. They are sent to the model as
	// data it may quote but never recompute.
	Numbers map[string]any `json:"numbers"`

	Action Action            `json:"action,omitempty"`
	Params map[string]string `json:"params,omitempty"`
}

// Card is one line of the briefing: the model's words, our numbers.
type Card struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Body  string `json:"body"`

	// Filled in from the Fact after the model has answered — never from the
	// model itself.
	Area   Area              `json:"area"`
	Action Action            `json:"action,omitempty"`
	Params map[string]string `json:"params,omitempty"`
}

// MaxCards is how many things a person actually acts on before service starts.
//
// ⚠️ A briefing of eleven items is a report, and a report is read once. The
// limit is the feature: it forces the model to choose, which is the judgement
// we are paying for.
const MaxCards = 4

// Rank orders facts by what they are worth and drops the duplicates.
//
// ⚠️ **One card per area.** A slow week is simultaneously a sales fact, a guest
// fact and a menu fact — all three true, all three the same problem — and an
// owner reading three sentences about it concludes the assistant is padding.
func Rank(facts []Fact) []Fact {
	sorted := make([]Fact, len(facts))
	copy(sorted, facts)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Weight != sorted[j].Weight {
			return sorted[i].Weight > sorted[j].Weight
		}
		// ⚠️ Ties broken by key, not left to map order: a briefing that
		// reshuffles between two identical mornings looks like something
		// changed when nothing did.
		return sorted[i].Key < sorted[j].Key
	})
	seen := map[Area]bool{}
	out := make([]Fact, 0, len(sorted))
	for _, f := range sorted {
		if seen[f.Area] {
			continue
		}
		seen[f.Area] = true
		out = append(out, f)
	}
	return out
}

// Keep drops every card that does not name a fact we sent.
//
// ⚠️ **This is the anti-invention seam, and it is deliberately dumb.** No
// parsing of the sentence, no checking figures against the text — just: did
// this card come back attached to something true? A model that answers with a
// key it was not given has invented the subject, and a sentence about an
// invented subject cannot be salvaged by editing it.
//
// It also fixes the ordering: the model chooses *which* facts, we keep the
// order they were ranked in, so the most expensive problem stays at the top
// however the model listed them.
func Keep(cards []Card, facts []Fact) []Card {
	byKey := make(map[string]Fact, len(facts))
	for _, f := range facts {
		byKey[f.Key] = f
	}
	said := make(map[string]Card, len(cards))
	for _, c := range cards {
		if c.Title == "" && c.Body == "" {
			continue
		}
		if _, ok := byKey[c.Key]; !ok {
			continue
		}
		// First mention wins: a model that lists one fact twice gets one card.
		if _, dup := said[c.Key]; !dup {
			said[c.Key] = c
		}
	}
	out := make([]Card, 0, len(said))
	for _, f := range facts {
		c, ok := said[f.Key]
		if !ok {
			continue
		}
		// ⚠️ Area and action are taken from the fact, never from the answer.
		// The model decides what to say; it does not decide what a button does.
		c.Area, c.Action, c.Params = f.Area, f.Action, f.Params
		out = append(out, c)
		if len(out) == MaxCards {
			break
		}
	}
	return out
}
