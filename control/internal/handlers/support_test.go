package handlers

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ⚠️ **The title is the first line, and an operator reads a hundred of them.**
// Chopped at a fixed width it breaks mid-word, and a queue of half-words is a
// queue that gets read twice.
func TestTheSubjectIsTheQuestion(t *testing.T) {
	cases := map[string]string{
		"Kassa ochilmayapti. Ertalabdan beri urinamiz, PIN qabul qilmayapti.": "Kassa ochilmayapti.",
		"Chek nega ruscha chiqmayapti? Sozlamada rus tili turibdi.":           "Chek nega ruscha chiqmayapti?",
		"Salom": "Salom",
	}
	for in, want := range cases {
		if got := summarise(in); got != want {
			t.Errorf("summarise(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestALongUnpunctuatedQuestionIsCutOnAWord(t *testing.T) {
	long := "Buyurtma kelganda ovoz chiqmayapti va panelda ham hech qanday " +
		"belgi yo'q shuning uchun kuryerlar kutib qolishyapti"
	got := summarise(long)
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("a long line was not marked as cut: %q", got)
	}
	if strings.HasSuffix(strings.TrimSuffix(got, "…"), " ") {
		t.Fatalf("cut left a trailing space: %q", got)
	}
	// The cut must land on a word boundary, not inside one.
	trimmed := strings.TrimSuffix(got, "…")
	if !strings.HasPrefix(long, trimmed) {
		t.Fatalf("the cut is not a prefix of the message: %q", got)
	}
	rest := long[len(trimmed):]
	if rest != "" && !strings.HasPrefix(rest, " ") {
		t.Fatalf("the cut fell inside a word: %q | %q", trimmed, rest)
	}
}

// ⚠️ An unbounded text field arriving from a browser on a customer's domain is
// a way to fill our disk with somebody else's problem.
func TestOneMessageCannotBeUnbounded(t *testing.T) {
	huge := strings.Repeat("a", 50_000)
	if len(clampSupport(huge)) > 8000 {
		t.Fatal("a message was stored beyond the cap")
	}
	if clampSupport("   ") != "" {
		t.Fatal("whitespace was accepted as a message")
	}
}

// ⚠️ **Closing the channel rather than sending on it.** A send needs a reader
// still there, and the reader that matters is the one whose request was
// cancelled a moment ago. This asserts the wake reaches every waiter and that a
// waiter who left does not block the next reply.
func TestAWakeReachesEveryWaiterAndNoneBlockIt(t *testing.T) {
	h := supportHub{waiting: map[string][]chan struct{}{}}
	a, stopA := h.listen("b5somsa")
	b, _ := h.listen("b5somsa")
	other, _ := h.listen("another")

	// One waiter goes away before the reply lands.
	stopA()

	done := make(chan struct{})
	go func() { h.wake("b5somsa"); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("wake blocked on a waiter that had gone")
	}

	select {
	case <-b:
	case <-time.After(time.Second):
		t.Fatal("a waiting connection was not woken")
	}
	select {
	case <-a:
		t.Fatal("a cancelled waiter was still woken")
	default:
	}
	select {
	case <-other:
		t.Fatal("one restaurant's reply woke another restaurant")
	default:
	}
}

// The hub must not leak: a slug with nobody listening is a key that has to go.
func TestTheHubForgetsEmptySlugs(t *testing.T) {
	h := supportHub{waiting: map[string][]chan struct{}{}}
	_, stop := h.listen("b5somsa")
	stop()
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, still := h.waiting["b5somsa"]; still {
		t.Fatal("the hub kept an empty slug")
	}
}

// ⚠️ **The search box has to find "printerdan" when an operator types
// "printer".** Uzbek is agglutinative and Mongo's text index tokenises against
// a stemmer that has no Uzbek in it, so the first version of this returned
// nothing for a thread sitting two rows below in the same list.
func TestSearchFindsASuffixedWord(t *testing.T) {
	or := supportSearch("printer")
	if len(or) == 0 {
		t.Fatal("no search clauses")
	}
	var pattern string
	for _, clause := range or {
		if rx, ok := clause["lastText"].(primitive.Regex); ok {
			pattern = rx.Pattern
		}
	}
	if pattern == "" {
		t.Fatal("the message text is not searched")
	}
	re := regexp.MustCompile("(?i)" + pattern)
	for _, hit := range []string{
		"ikkinchi printerdan savol belgilari chiqyapti",
		"Printer ishlamayapti",
		"chek printerga bormadi",
	} {
		if !re.MatchString(hit) {
			t.Errorf("searching \"printer\" missed %q", hit)
		}
	}
	// And it must not match a word that merely contains the term.
	if re.MatchString("kompyuterprinter") {
		t.Error("the search matched inside a word")
	}
}

// patternsOf pulls every regex out of the clause list, whichever field it is
// filed under — the first version of this test indexed one clause by the wrong
// key, got a zero-value Regex whose empty pattern matches everything, and
// failed for a reason that had nothing to do with the code.
func patternsOf(or []bson.M) []string {
	var out []string
	for _, clause := range or {
		for _, v := range clause {
			if rx, ok := v.(primitive.Regex); ok {
				out = append(out, rx.Pattern)
			}
		}
	}
	return out
}

// ⚠️ An operator pastes a customer's error message. Unquoted, `(` is an invalid
// pattern and `.*` is a search that matches everything.
func TestSearchQuotesWhatTheOperatorTyped(t *testing.T) {
	for _, term := range []string{"chek (58mm)", "narx * 2", "a[b"} {
		for _, rx := range patternsOf(supportSearch(term)) {
			if _, err := regexp.Compile(rx); err != nil {
				t.Errorf("searching %q built an invalid pattern: %v", term, err)
			}
		}
	}
	for _, rx := range patternsOf(supportSearch(".*")) {
		if regexp.MustCompile(rx).MatchString("hech qanday nuqta yo'q") {
			t.Error("a wildcard was taken literally and matched everything")
		}
	}
}

// ⚠️ **The rule that makes this feature safe enough to ship is in the prompt,
// so it is asserted here.** A model that answers about the product from its own
// training is worse than no assistant: a restaurant that follows one wrong
// instruction has a real problem caused by us, and nothing this widget says
// afterwards is believed.
func TestTheAssistantIsToldItMayOnlyUseTheArticles(t *testing.T) {
	for _, must := range []string{
		"ANSWER ONLY FROM THE ARTICLES",
		"IF THE ARTICLES DO NOT ANSWER THE QUESTION, SAY SO",
		"Never invent a screen",
		"Do not promise anything on the company's behalf",
	} {
		if !strings.Contains(assistSystem, must) {
			t.Errorf("the assistant prompt no longer says %q", must)
		}
	}
}

// ⚠️ The refusal has to be a first-class answer, not an empty string the caller
// guesses at — otherwise "I cannot help" and "the model failed" look the same
// and only one of them should reach the operator differently.
func TestTheAssistantCanRefuse(t *testing.T) {
	props, _ := assistSchema["properties"].(map[string]any)
	if _, ok := props["answered"]; !ok {
		t.Fatal("the schema has no way to say the articles did not cover it")
	}
	req, _ := assistSchema["required"].([]string)
	var has bool
	for _, r := range req {
		if r == "answered" {
			has = true
		}
	}
	if !has {
		t.Fatal("a model may omit `answered`, and an omitted refusal reads as an answer")
	}
}

// The same prompt-caching rule the briefing follows: nothing per restaurant.
func TestTheAssistantPromptCarriesNothingPerRestaurant(t *testing.T) {
	for _, banned := range []string{"%s", "%d", "{{", "${"} {
		if strings.Contains(assistSystem, banned) {
			t.Errorf("the assistant prompt interpolates %q", banned)
		}
	}
}
