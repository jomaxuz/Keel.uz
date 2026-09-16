package handlers

import (
	"os"
	"strings"
	"testing"
)

// ⚠️ **The advertising allowance is not the assistant's.** One shared counter
// would mean a restaurant that spent the morning on campaign wording loses
// tomorrow's briefing — two features failing as one, with neither screen able to
// say why. The rule is one collection name in one function, which is exactly the
// kind of thing a later "tidy-up" merges back together.
func TestAdvertisingCountsAgainstItsOwnLedger(t *testing.T) {
	src, err := os.ReadFile("adsadvice.go")
	if err != nil {
		t.Fatal(err)
	}
	fn := string(src)
	start := strings.Index(fn, "func (h *Handler) adsToday")
	if start < 0 {
		t.Fatal("adsToday is gone; the daily cap is now somebody else's counter")
	}
	body := fn[start:]
	if end := strings.Index(body, "\n}\n"); end > 0 {
		body = body[:end]
	}
	if !strings.Contains(body, "h.Store.AdsLog") {
		t.Fatal("the advertising cap must count its own log")
	}
	if strings.Contains(body, "BriefingLog") {
		t.Fatal("the advertising cap is counting briefings: one restaurant's " +
			"campaign questions would eat its morning briefings")
	}
}

// ⚠️ **The system prompt is byte-identical for every restaurant and every
// question**, which is what makes prompt caching work. The moment a name, a date
// or a figure is formatted into it, every call misses the cache and the add-on
// costs several times what it was priced at — with nothing failing.
func TestThePlannerPromptCarriesNothingSpecific(t *testing.T) {
	for _, bad := range []string{"%s", "%d", "%v"} {
		if strings.Contains(adsPlanSystem, bad) {
			t.Fatalf("the prompt interpolates %q; caching is lost", bad)
		}
	}
	// The two rules the money rests on, checked as text because they are the
	// reason an owner can act on what comes back.
	if !strings.Contains(adsPlanSystem, "NEVER STATE A NUMBER THAT IS NOT IN THE DATA") {
		t.Fatal("the planner may invent sales figures somebody will budget against")
	}
	if !strings.Contains(adsPlanSystem, "PROPOSE, DO NOT DECIDE") {
		t.Fatal("the planner is deciding rather than offering the owner a choice")
	}
}

// ⚠️ **Every proposal carries its reason, enforced in the schema.** A dish
// suggested without a "why" is one the owner cannot weigh — they either take it
// on trust or ignore it, and both are worse than not proposing it. This is the
// one rule that cannot be left to the prompt: a model with nothing to say in the
// data is exactly the model that drops the field.
func TestEveryProposalMustCarryItsReason(t *testing.T) {
	props, _ := adsPlanSchema["properties"].(map[string]any)
	if len(props) != 4 {
		t.Fatalf("expected four groups of proposals, got %d", len(props))
	}
	for name, group := range props {
		g, _ := group.(map[string]any)
		items, _ := g["items"].(map[string]any)
		req, _ := items["required"].([]string)
		found := false
		for _, r := range req {
			if r == "why" {
				found = true
			}
		}
		if !found {
			t.Fatalf("%q may be proposed without a reason", name)
		}
		if items["additionalProperties"] != false {
			t.Fatalf("%q lets the model add fields the panel never draws", name)
		}
	}
}
