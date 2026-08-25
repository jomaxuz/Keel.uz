package httpx

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// ⚠️ **`if r.ContentLength > 0` is the idiom this replaces, and it is wrong in
// a way that only shows up in production.** A chunked request reports a length
// of **-1**, so the test skips a body that is genuinely there — and the till's
// requests arrive chunked once they have been through the Windows app's proxy.
// A field then arrives empty and every caller reads that as "not asked for":
// the bill printed nothing, and a failed print was recorded as a success.

type body struct {
	Kind  string `json:"kind"`
	Error string `json:"error"`
}

func TestABodySentWithoutALengthIsStillRead(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"kind":"precheck"}`))
	// What a chunked request looks like to a handler.
	r.ContentLength = -1

	var got body
	if err := DecodeOptional(r, &got); err != nil {
		t.Fatalf("DecodeOptional: %v", err)
	}
	if got.Kind != "precheck" {
		t.Fatalf("kind = %q — the body was skipped", got.Kind)
	}
}

func TestNoBodyIsNotAnError(t *testing.T) {
	// "Optional" has to mean optional: firing a whole check sends nothing.
	r := httptest.NewRequest("POST", "/", strings.NewReader(""))
	var got body
	if err := DecodeOptional(r, &got); err != nil {
		t.Fatalf("an absent body was refused: %v", err)
	}
	if got.Kind != "" {
		t.Fatalf("kind = %q, want the zero value", got.Kind)
	}
}

func TestBrokenJSONIsStillRefused(t *testing.T) {
	// ⚠️ Optional is not "ignore whatever arrives". A malformed body means the
	// caller believes it said something, and reading it as silence is how a
	// failed print gets recorded as a success.
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"kind":`))
	var got body
	if err := DecodeOptional(r, &got); err == nil {
		t.Fatal("malformed JSON was accepted as an empty body")
	}
}
