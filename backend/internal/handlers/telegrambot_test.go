package handlers

import (
	"strings"
	"testing"
)

// The Start payload, which is a **different** parameter from the mini app's
// `startapp` — a table QR routed through the bot chat arrives as `/start t_<id>`.
// Sealed because the failure is invisible: an unparsed payload silently opens the
// menu with no table, so a guest sitting at table 7 orders delivery to nowhere.
func TestParseStartPayload(t *testing.T) {
	const table = "64b7f1a2c3d4e5f6a7b8c9d0"
	const branch = "1234567890abcdef12345678"

	cases := []struct {
		name, raw, table, branch string
	}{
		{"table only", "t_" + table, table, ""},
		{"table and branch", "t_" + table + "-b_" + branch, table, branch},
		{"order does not matter", "b_" + branch + "-t_" + table, table, branch},
		{"plain start has nothing", "", "", ""},
		// Anybody can type a link into a chat, and this value ends up in a URL we
		// hand back to a guest: a non-id must be dropped, not passed through.
		{"not an id", "t_../../etc/passwd", "", ""},
		{"too short", "t_64b7f1a2", "", ""},
		{"junk keys ignored", "x_" + table, "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotT, gotB := parseStartPayload(c.raw)
			if gotT != c.table || gotB != c.branch {
				t.Fatalf("%q: got (%q,%q), want (%q,%q)",
					c.raw, gotT, gotB, c.table, c.branch)
			}
		})
	}
}

// Every language has a greeting **and** a button label. A missing label is an
// empty button, which is worse than a missing one: it looks like the bot answered
// with something broken.
func TestBotWelcomeCoversEveryLanguage(t *testing.T) {
	for _, lang := range []string{"uz", "ru", "en", "de", ""} {
		text, button := botWelcome(lang, "Osh Markazi")
		if text == "" || button == "" {
			t.Fatalf("lang %q: text=%q button=%q", lang, text, button)
		}
		// The restaurant's own name, not "Restoran": the greeting is the first
		// thing a guest reads, and an anonymous one reads as a wrong number.
		if !strings.Contains(text, "Osh Markazi") {
			t.Fatalf("lang %q: name missing from %q", lang, text)
		}
	}
}
