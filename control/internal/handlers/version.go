package handlers

import (
	"fmt"
	"strconv"
	"strings"
)

// What the platform calls itself, in one place per binary.
//
// ⚠️ **A constant in the binary, not a value read from anywhere.** The version has to
// agree with the code that is running, and every other source can disagree with it: a
// file on disk survives a rollback, an environment variable is set by whoever last
// edited the compose file, and a git tag is a fact about a repository rather than about
// the container answering the request.
//
// ⚠️ **And every part of Keel says the same number.** The landing, the console, a
// restaurant's dashboard and the Windows till are one product to the person paying for
// it — "which version?" has to have one answer, or a support call starts by
// establishing which of four numbers is being discussed.
//
// **Change it with `scripts/set-version.sh vX.Y.Z`**, which writes all five places at
// once. version_test.go then fails the build if one of them drifts anyway — the script
// is the convenient path and the test is the one that has to be true.
//
// The console's **Server holati** panel has buttons that do the same thing from a
// browser: they ask GitHub to run `release.yml`, which runs that script, pushes, and
// deploys. ⚠️ The script stays the only thing that knows *how* a version is written —
// the console knows a number and nothing else. See handlers/release.go.
//
// `Stage` is separate from the number and says what the number *means*. Empty means the
// number stands on its own.
const (
	Version = "v0.2.0"
	Stage   = ""
)

// ---- Raising it ----
//
// The console offers three buttons rather than a text field, because the only
// versions anybody should be able to reach from here are the next three. A
// field invites "v0.3" and "0.2.1 " and the release that skips to v1.4.0 by a
// typo — and this number is written into seven files and tagged onto a
// deployment, so a typo is not caught by anything until somebody sorts a list.
//
// ⚠️ **Computed from the running constant, never from a stored value.** The
// number offered has to follow the code that is answering the request: taking
// it from the VERSION file on disk, or from the last release row in Mongo,
// would offer v0.2.1 again on a container that had already been rolled back to
// v0.1.9 — and the resulting release would move the platform *backwards* while
// reading as a step forward everywhere on the screen.

// releaseParts is the order the buttons appear in — smallest first, because
// that is the one that should be easy and the others should take a moment.
var releaseParts = []string{"patch", "minor", "major"}

// NextVersion is what `part` would raise `cur` to.
//
// ⚠️ A minor zeroes the patch and a major zeroes both. Without that, v0.2.7 →
// "minor" would give v0.3.7, which sorts correctly, reads as a version, and is
// wrong in the way nobody checks: it claims seven patches that were never made.
func NextVersion(cur, part string) (string, bool) {
	n, ok := parseVersion(cur)
	if !ok {
		return "", false
	}
	switch part {
	case "patch":
		n[2]++
	case "minor":
		n[1]++
		n[2] = 0
	case "major":
		n[0]++
		n[1], n[2] = 0, 0
	default:
		return "", false
	}
	return fmt.Sprintf("v%d.%d.%d", n[0], n[1], n[2]), true
}

// NextVersions is every offer the panel shows, keyed by the button that makes it.
func NextVersions(cur string) map[string]string {
	out := map[string]string{}
	for _, p := range releaseParts {
		if v, ok := NextVersion(cur, p); ok {
			out[p] = v
		}
	}
	return out
}

// parseVersion reads "v1.2.3" strictly.
//
// ⚠️ **Strict, unlike the till's own parser.** That one is lenient because it
// reads a manifest somebody typed by hand and must never conclude a till is a
// thousand releases behind. This one reads the constant compiled into this
// binary: anything unexpected there is a mistake in our own source, and
// answering "no buttons" is how it becomes visible instead of producing an
// offer built on a misreading.
func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(v), "v"), ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
