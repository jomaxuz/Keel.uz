package handlers

import "testing"

// The advisor's two pure decisions, tested here because neither is visible from
// a screen when it goes wrong.

// ⚠️ **The cache key decides whether a question costs money**, and both
// failures are silent: too strict and every question is paid for twice, too
// loose and an owner gets yesterday's sentence to today's question.
func TestAdvisorKeyNormalisesTheQuestion(t *testing.T) {
	base := advisorKey("2026-09-16", "b/br", "uz", "Nega tushum tushdi?")
	same := []string{
		"nega tushum tushdi",
		"  Nega   tushum tushdi? ",
		"NEGA TUSHUM TUSHDI!",
	}
	for _, q := range same {
		if got := advisorKey("2026-09-16", "b/br", "uz", q); got != base {
			t.Fatalf("%q should be the same question as the original", q)
		}
	}
	// ⚠️ The day, the lens and the language are part of it. A key without the
	// lens answers a chain's second brand with the first one's figures — the
	// exact defect the briefing had to be fixed for.
	if advisorKey("2026-09-17", "b/br", "uz", "Nega tushum tushdi?") == base {
		t.Fatal("a new day is a new answer")
	}
	if advisorKey("2026-09-16", "b/other", "uz", "Nega tushum tushdi?") == base {
		t.Fatal("another branch is another answer")
	}
	if advisorKey("2026-09-16", "b/br", "ru", "Nega tushum tushdi?") == base {
		t.Fatal("another language is another answer")
	}
}

// ⚠️ **`c1` is a prefix of `c12`.** Replacing shortest-first turns "c12" into
// "Dilnoza2" — a name that belongs to nobody, inside a sentence that otherwise
// reads perfectly. Nothing downstream could catch it.
func TestResolveAliasesLongestFirst(t *testing.T) {
	aliases := map[string]string{"c1": "Dilnoza", "c12": "Aziz"}
	got := resolveAliases("c12 va c1 olti haftadan beri kelmadi", aliases)
	if got != "Aziz va Dilnoza olti haftadan beri kelmadi" {
		t.Fatalf("aliases resolved wrongly: %q", got)
	}
}

// A guest with no name keeps their alias rather than becoming an empty string —
// otherwise the sentence loses the thing it was pointing at.
func TestResolveAliasesKeepsUnnamedGuests(t *testing.T) {
	got := resolveAliases("c3 uch oydan beri yo'q", map[string]string{"c3": "c3"})
	if got != "c3 uch oydan beri yo'q" {
		t.Fatalf("an unnamed guest should keep the alias: %q", got)
	}
}

func TestAliasOfCountsFromOne(t *testing.T) {
	if aliasOf(0) != "c1" || aliasOf(11) != "c12" {
		t.Fatalf("aliases are c1, c2, …: %q %q", aliasOf(0), aliasOf(11))
	}
}
