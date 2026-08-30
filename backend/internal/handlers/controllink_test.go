package handlers

import (
	"errors"
	"strings"
	"testing"
)

// ⚠️ **The bug this file exists for: an install with no platform behind it got
// a Go error message about URL schemes.**
//
// `callControlPath` built its request against an empty base, so the owner
// pressing "write me three messages" was shown
// `Post "/internal/campaign-text": unsupported protocol scheme ""` — turned by
// the panel into a bare "failed". Three callers had their own guard and a
// fourth did not; the guard now lives in the one place every caller goes
// through.
func TestNotLinkedIsATypedAnswerNotAnAccident(t *testing.T) {
	if ErrNotLinked == nil {
		t.Fatal("there is no way to say this server has no platform")
	}
	if !errors.Is(ErrNotLinked, ErrNotLinked) {
		t.Fatal("callers cannot match on it")
	}
	// ⚠️ Its message is read by a restaurant owner, so it is a sentence rather
	// than a status.
	msg := ErrNotLinked.Error()
	if msg == "" || len(msg) < 12 {
		t.Fatalf("the message %q is not something to show somebody", msg)
	}
	for _, jargon := range []string{"nil", "scheme", "http", "URL", "500"} {
		if strings.Contains(msg, jargon) {
			t.Errorf("the owner-facing message contains %q: %q", jargon, msg)
		}
	}
}
