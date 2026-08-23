package main

import "testing"

// ⚠️ **Numeric, part by part, and this is the test that earns it.** A string
// comparison puts "1.10.0" before "1.9.0", so updates would silently stop being
// offered at the tenth release of a minor — and only to the tills that happened
// to be on 1.9. Nothing would go wrong anywhere visible: every one of those
// machines would simply decide it was already current.
func TestNewerVersionComparesNumbersNotText(t *testing.T) {
	cases := []struct {
		have, want string
		newer      bool
	}{
		{"1.0.0", "1.0.1", true},
		{"1.0.0", "1.1.0", true},
		{"1.0.0", "2.0.0", true},
		// The one a string compare gets wrong.
		{"1.9.0", "1.10.0", true},
		{"1.10.0", "1.9.0", false},
		// Same build: nothing to do, and asking again every six hours is the
		// ordinary state of every till in the country.
		{"1.4.2", "1.4.2", false},
		// ⚠️ Never downgrade. A manifest rolled back by mistake would otherwise
		// walk every counter backwards onto a build we had just replaced.
		{"1.4.2", "1.4.1", false},
		// A "v" prefix is what a git tag looks like, and it will be pasted in.
		{"1.0.0", "v1.2.0", true},
	}
	for _, c := range cases {
		if got := newerVersion(c.have, c.want); got != c.newer {
			t.Errorf("newerVersion(%q, %q) = %v, want %v", c.have, c.want, got, c.newer)
		}
	}
}

// ⚠️ Anything unparseable is zero rather than an error. A version somebody typed
// by hand into a manifest must not be able to convince a hundred tills that
// they are a thousand releases behind — the worst it can do is nothing.
func TestAMistypedVersionCannotForceAnUpdate(t *testing.T) {
	for _, junk := range []string{"", "next", "1.x.0", "..", "latest"} {
		if newerVersion("1.0.0", junk) {
			t.Errorf("%q was treated as newer than 1.0.0", junk)
		}
	}
	// …and a till on a junk version still accepts a real one, so one bad
	// release cannot strand a machine forever.
	if !newerVersion("junk", "1.0.1") {
		t.Error("a till on an unreadable version can never update again")
	}
}
