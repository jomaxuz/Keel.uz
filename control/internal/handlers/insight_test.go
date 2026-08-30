package handlers

import (
	"strings"
	"testing"
)

// ⚠️ **The titles were the failure, and the prompt was the cause.** "Title: at
// most six words, naming the thing" is a rule that produces "Weekly revenue
// decline" — a category, not a finding — and an owner who reads two of those
// concludes the assistant is a relabelled report. The instruction is now that
// the title says what happened, with the example that made it clear.
func TestThePromptAsksForFindingsNotCategories(t *testing.T) {
	for _, must := range []string{
		"THE TITLE IS THE FINDING",
		"names the category",
		"the most likely reason",
	} {
		if !strings.Contains(briefingSystem, must) {
			t.Errorf("the briefing prompt no longer says %q", must)
		}
	}
}

// ⚠️ **Byte-identical for every restaurant and every morning.** Prompt caching
// matches on a prefix; the moment a name or a date appears here every call
// misses the cache and the daily sweep costs several times what it should.
func TestThePromptCarriesNothingPerRestaurant(t *testing.T) {
	for _, banned := range []string{"%s", "%d", "{{", "${"} {
		if strings.Contains(briefingSystem, banned) {
			t.Errorf("the briefing prompt interpolates %q, which breaks prompt caching", banned)
		}
	}
}
