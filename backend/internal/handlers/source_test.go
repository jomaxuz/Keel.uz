package handlers

import (
	"os"
	"strings"
	"testing"
)

// Reading our own source in a test is unusual, so: these assertions are about
// rules that have no runtime surface to check. "This list must never include
// till checks" is a property of one line in one query, and the only ways to
// test it are a live database or the line itself.
func readSource(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

func between(t *testing.T, src, start, end string) string {
	t.Helper()
	i := strings.Index(src, start)
	if i < 0 {
		t.Fatalf("%q not found — was it renamed?", start)
	}
	rest := src[i:]
	j := strings.Index(rest, end)
	if j < 0 {
		t.Fatalf("end of %q not found", start)
	}
	return rest[:j]
}
