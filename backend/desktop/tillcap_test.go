package main

import (
	"strings"
	"testing"
)

// ⚠️ **The number this refusal used to be.** A monoblock that met the plan's
// register limit answered "qurilma kaliti: server 402" — a status code, on the
// screen of somebody standing in front of a new machine with nothing they can
// act on. What the sentence has to carry is the count, so "we bought five and
// only run four" is visible, and the two things that fix it.
func TestCapMessageSaysTheCountAndWhatToDo(t *testing.T) {
	got := capMessage([]byte(`{"error":"tarifingizdagi kassalar soni to'lgan",` +
		`"registers":5,"limit":5,"plan":"pro"}`))
	if !strings.Contains(got, "limiti tugadi") {
		t.Fatalf("no plain refusal in %q", got)
	}
	if !strings.Contains(got, "5 / 5") {
		t.Fatalf("no count in %q", got)
	}
	if strings.Contains(got, "402") {
		t.Fatalf("still a status code: %q", got)
	}
}

// ⚠️ An older server answers 402 with a message and no figures, and a till
// installed today talks to whatever version the restaurant is running. The
// sentence still has to read as a sentence.
func TestCapMessageWithoutFiguresIsStillASentence(t *testing.T) {
	got := capMessage([]byte(`{"error":"tarifingizdagi kassalar soni to'lgan"}`))
	if strings.Contains(got, "(") || strings.Contains(got, "0 / 0") {
		t.Fatalf("empty figures leaked into %q", got)
	}
	if !strings.Contains(got, "limiti tugadi") {
		t.Fatalf("no refusal in %q", got)
	}
}

// A body that is not JSON at all — a proxy's error page, say — must not produce
// an empty string, which on the setup screen is a button that did nothing.
func TestCapMessageSurvivesRubbish(t *testing.T) {
	if got := capMessage([]byte("<html>402</html>")); !strings.Contains(got, "limiti tugadi") {
		t.Fatalf("nothing useful from a non-JSON body: %q", got)
	}
}
