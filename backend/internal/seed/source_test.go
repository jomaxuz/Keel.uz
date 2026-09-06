package seed

import (
	"os"
	"strings"
	"testing"
)

// Reading the seed's own source, for rules whose whole point is what does *not*
// happen — a menu that is not written cannot be observed from a test that only
// looks at an empty database, because an empty database is also what a failure
// looks like.

func source(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
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
