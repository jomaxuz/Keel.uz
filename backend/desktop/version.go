package main

import (
	"strconv"
	"strings"
)

// ---- What this build is ----
//
// ⚠️ **A constant in the binary, read from nowhere else.** The version has to
// match the code that is answering, and every other source drifts from it: a
// file on disk survives a rollback, an environment variable is set by whoever
// edited the shortcut last, and a git tag is a fact about the repository rather
// than about the executable sitting on this counter. The same reasoning the
// control plane's version follows.
//
// ⚠️ **This is what the updater compares.** Forgetting to raise it before a
// release does not break anything loudly — it means every till decides it is
// already up to date and the fix reaches nobody, which is the quietest possible
// failure for a fix. It is the one line a release has to touch.
const Version = "0.1.0"

// newerVersion reports whether `have` should be replaced by `want`.
//
// ⚠️ **Numeric, part by part, not string comparison.** "1.10.0" sorts before
// "1.9.0" as text, so a string compare would stop offering updates at the tenth
// release of a minor — silently, and only for the tills that had taken 1.9.
func newerVersion(have, want string) bool {
	h, w := splitVersion(have), splitVersion(want)
	for i := 0; i < 3; i++ {
		if w[i] != h[i] {
			return w[i] > h[i]
		}
	}
	return false
}

func splitVersion(v string) [3]int {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(v), "v"), ".")
	for i := 0; i < 3 && i < len(parts); i++ {
		// Anything unparseable is zero: a version somebody typed by hand into a
		// manifest must not be able to make a till think it is a thousand
		// releases behind.
		n, _ := strconv.Atoi(strings.TrimSpace(parts[i]))
		if n > 0 {
			out[i] = n
		}
	}
	return out
}

// errNoStore is what the disk calls answer with when the local database could
// not be opened.
//
// ⚠️ **An error and not a silent false.** The screen's fallback is the
// browser's own storage, which is a real fallback and a weaker promise — and
// the difference has to be visible to the person who will be asked why an
// evening's sales are missing.
var errNoStore = errStr("lokal baza ochilmagan")

type errStr string

func (e errStr) Error() string { return string(e) }
