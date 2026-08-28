package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// The overload this retry was written for was the one it never saw.
//
// Google answers "currently experiencing high demand, spikes in demand are
// usually temporary. Please try again later" as a **500 with `error.message`
// set**. The message branch used to run before the status check, so that reply
// came back as a plain error: not `overloaded`, never retried, and — once the
// Gemini side became several models — indistinguishable from a dead model.
func TestAHighDemand500IsRetried(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"message":"gemini-3.7-flash is currently experiencing high demand, spikes in demand are usually temporary. Please try again later."}}`))
			return
		}
		_, _ = w.Write([]byte(`{"output_text":"{\"ok\":1}"}`))
	}))
	defer srv.Close()

	g := Gemini{Key: "k", Model: "gemini-3.7-flash", Endpoint: srv.URL}
	out, _, err := g.JSON(context.Background(), "s", "u", nil, "low")
	if err != nil {
		t.Fatalf("the second attempt should have answered: %v", err)
	}
	if out != `{"ok":1}` {
		t.Fatalf("got %q", out)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("asked %d times, want 2 (one retry)", got)
	}
}

// A 429 still means the quota, and its own type — the panel reads it.
func TestAQuota429KeepsItsType(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"You exceeded your current quota"}}`))
	}))
	defer srv.Close()

	g := Gemini{Key: "k", Endpoint: srv.URL}
	_, _, err := g.JSON(context.Background(), "s", "u", nil, "low")
	var spent Exhausted
	if !errors.As(err, &spent) {
		t.Fatalf("a 429 must be Exhausted, got %v", err)
	}
	if !strings.Contains(err.Error(), "exceeded your current quota") {
		t.Fatalf("the operator needs Google's sentence, got %q", err)
	}
	// ⚠️ Not retried: every rejected request still counts against the quota.
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("asked %d times, want 1", got)
	}
}

// A retired model name is a 404, and the chain's answer is the next model.
func TestARetiredModelIsAnOrdinaryFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"models/gemini-2.0-flash is not found"}}`))
	}))
	defer srv.Close()

	g := Gemini{Key: "k", Model: "gemini-2.0-flash", Endpoint: srv.URL}
	_, _, err := g.JSON(context.Background(), "s", "u", nil, "low")
	var spent Exhausted
	var over overloaded
	if errors.As(err, &spent) || errors.As(err, &over) {
		t.Fatalf("a 404 is neither a quota nor an overload: %v", err)
	}
	if !strings.Contains(err.Error(), "is not found") {
		t.Fatalf("got %q", err)
	}
}
