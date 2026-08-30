package handlers

import (
	"strings"
	"testing"
	"time"
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
