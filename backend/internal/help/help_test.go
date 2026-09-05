package help

import "testing"

// ⚠️ **The failure this seals is silence.** An embed that did not resolve, or a
// JSON shape that drifted, leaves an empty slice — and an empty help screen
// reads as "there are no articles about this", which is the answer that sends
// somebody to the support chat for a question one paragraph would have closed.

func TestEveryLanguageAnswersWithSomething(t *testing.T) {
	for _, lang := range []string{"uz", "ru", "en", "", "de"} {
		if got := For(lang); len(got) == 0 {
			t.Fatalf("help.For(%q) is empty", lang)
		}
	}
}

func TestAnArticleIsWorthSearching(t *testing.T) {
	// Title and body are what the ranking weighs; an entry missing either is a
	// row that can never be found and never be read.
	for _, lang := range Langs() {
		for _, a := range For(lang) {
			if a.ID == "" || a.Title == "" || a.Body == "" {
				t.Fatalf("%s: article %q is missing a field", lang, a.ID)
			}
		}
	}
}

func TestAMissingTranslationFallsBackRatherThanEmptying(t *testing.T) {
	// ⚠️ Uzbek is the base and the complete one. A language with fewer articles
	// must not answer with fewer — it answers with the base, because a short
	// help screen is indistinguishable from a broken one.
	if len(For("de")) != len(For("uz")) {
		t.Fatal("an unknown language did not fall back to the base")
	}
}
