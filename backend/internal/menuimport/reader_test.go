package menuimport

import "testing"

// ⚠️ **The order is the design, not an implementation detail.** Every reader
// above the assistant reads numbers the site published — exact, free, no model.
// Asking a model to read a price out of a sentence when the same price is
// sitting in a JSON field two readers up is paying to be less accurate, and a
// reordering here would do exactly that without failing anything.
func TestTheReadersAreTriedInOrderOfCertainty(t *testing.T) {
	// ⚠️ `embedded` sits between the named blob and the API for the same
	// reason the list is ordered at all: a document that *says* it is the
	// page's data is better evidence than a value that merely parses, and both
	// beat a guessed URL — but all three beat a model reading prose.
	// ⚠️ `aggregator` is first and that does not break the rule above: it is
	// the site's own published JSON, the most exact source there is, and it
	// costs no request anywhere else — the host has to match before anything
	// is fetched.
	want := []string{
		ReaderAggregator, ReaderStructured, ReaderInline, ReaderEmbedded, ReaderAPI,
	}
	got := Readers()
	if len(got) != len(want) {
		t.Fatalf("got %d readers, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("reader %d is %q, want %q", i, got[i].ID, id)
		}
	}
	// ⚠️ The assistant is deliberately not in the list: it needs the handler's
	// link to the platform. Everything here is offline, and that is what makes
	// the ordering safe to state.
	for _, r := range got {
		if r.ID == ReaderText || r.Assisted {
			t.Fatalf("%q is in the offline reader list", r.ID)
		}
		if r.Label == "" {
			t.Fatalf("%q has no label — the panel would show an empty line", r.ID)
		}
		if r.Read == nil {
			t.Fatalf("%q has no reader function", r.ID)
		}
	}
}

// ⚠️ An id with no label would draw a blank line where the panel says how the
// page was read — which is exactly the sentence that tells an owner whether to
// trust the list.
func TestEveryReaderIdHasAName(t *testing.T) {
	for _, id := range []string{
		ReaderAggregator, ReaderStructured, ReaderInline, ReaderEmbedded,
		ReaderAPI, ReaderText,
	} {
		if ReaderLabel(id) == "" || ReaderLabel(id) == id {
			t.Fatalf("reader %q has no readable name", id)
		}
	}
}
