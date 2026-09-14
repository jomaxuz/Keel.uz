package handlers

import "testing"

// ⚠️ A container recreated by a deploy is gone for a few seconds, and one
// sample landing in them used to paint a whole day red. Down counts only when
// the previous sample saw the same tenant down.
func TestConfirmedDownNeedsTwoSamplesInARow(t *testing.T) {
	// First sample: a deploy is replacing "osh". Not an outage yet.
	n, prev := confirmedDown(nil, []string{"osh"})
	if n != 0 {
		t.Fatalf("a single down sample counted %d", n)
	}
	// Next minute it is back — the deploy finished. Still nothing counted.
	n, prev = confirmedDown(prev, nil)
	if n != 0 {
		t.Fatalf("a recovered tenant counted %d", n)
	}
	// A real outage: down, and down again a minute later.
	n, prev = confirmedDown(prev, []string{"lagmon"})
	if n != 0 {
		t.Fatalf("first minute of an outage counted %d", n)
	}
	n, prev = confirmedDown(prev, []string{"lagmon"})
	if n != 1 {
		t.Fatalf("second minute of an outage counted %d, want 1", n)
	}
	// Two different tenants each down once in a row do not add up to one outage.
	n, prev = confirmedDown(prev, []string{"somsa"})
	if n != 0 {
		t.Fatalf("a different tenant going down inherited the last one's minute: %d", n)
	}
	n, _ = confirmedDown(prev, []string{"somsa", "manti"})
	if n != 1 {
		t.Fatalf("somsa down twice and manti once counted %d, want 1", n)
	}
}
