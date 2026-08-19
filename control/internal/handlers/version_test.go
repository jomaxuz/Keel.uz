package handlers

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ⚠️ **One product, one number.** The landing, the console, a restaurant's dashboard
// and the Windows till are four artefacts of one thing to the person paying for it, and
// "which version are you on?" has to have a single answer — otherwise a support call
// begins by working out which of four numbers is being discussed, and the one that is
// wrong is whichever nobody thought to check.
//
// Each of them keeps its **own** constant, because a version read at runtime can
// disagree with the code that is running (see version.go). This test is what keeps four
// constants from becoming four versions: the root VERSION file is where a human changes
// it, and anything that did not follow fails here rather than in a restaurant.
func TestEveryPartOfKeelDeclaresTheSameVersion(t *testing.T) {
	root := repoRoot(t)

	want := strings.TrimSpace(read(t, filepath.Join(root, "VERSION")))
	if want == "" {
		t.Fatal("VERSION is empty — it is the source every declaration is checked against")
	}
	if Version != want {
		t.Errorf("handlers.Version = %q, VERSION file says %q", Version, want)
	}

	// file → the pattern that finds its declaration.
	decls := map[string]*regexp.Regexp{
		"backend/internal/version/version.go": regexp.MustCompile(`const Version = "([^"]+)"`),
		"keel-site/src/lib/version.ts":        regexp.MustCompile(`export const VERSION = "([^"]+)"`),
		"frontend/src/lib/version.ts":         regexp.MustCompile(`export const VERSION = "([^"]+)"`),
	}
	for rel, re := range decls {
		m := re.FindStringSubmatch(read(t, filepath.Join(root, rel)))
		if m == nil {
			t.Errorf("%s: no version declaration found — did it move? this test is the only thing holding it", rel)
			continue
		}
		if m[1] != want {
			t.Errorf("%s declares %q, VERSION file says %q", rel, m[1], want)
		}
	}
}

// repoRoot walks up until it finds the VERSION file, so the test does not depend on how
// deep this package happens to sit.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for range 8 {
		if _, err := os.Stat(filepath.Join(dir, "VERSION")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("VERSION file not found above the test's directory")
	return ""
}

func read(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return string(raw)
}
