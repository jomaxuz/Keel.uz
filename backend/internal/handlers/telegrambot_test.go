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

// The greeting is Uzbek and offers all three languages, each labelled in its own
// script. Sealed because the whole point of the two-step flow is that the first
// message never picks a language for the guest: a regression here would greet a
// Russian speaker in a language they cannot read while holding the only button.
func TestBotGreetingOffersEveryLanguage(t *testing.T) {
	text, buttons := botGreeting("Osh Markazi")
	if !strings.Contains(text, "Osh Markazi") {
		t.Fatalf("name missing from greeting: %q", text)
	}
	if len(buttons) != 3 {
		t.Fatalf("got %d language buttons, want 3", len(buttons))
	}
	want := map[string]string{"lang:uz": "O'zbekcha", "lang:ru": "Русский", "lang:en": "English"}
	for _, b := range buttons {
		if want[b.Data] != b.Label {
			t.Fatalf("button %q labelled %q, want %q", b.Data, b.Label, want[b.Data])
		}
		// ⚠️ Telegram rejects callback_data over 64 bytes, and it rejects the
		// **whole message** with it — leaving the bot silent for a reason nothing
		// in the panel would show. The table id is appended to these later.
		if got := len(appendPayload(b.Data, "64b7f1a2c3d4e5f6a7b8c9d0", "")); got > 64 {
			t.Fatalf("callback_data with a table is %d bytes, Telegram allows 64", got)
		}
	}
}

// Every language has a greeting **and** a button label. A missing label is an
// empty button, which is worse than a missing one: it looks like the bot answered
// with something broken.
func TestBotMenuPromptCoversEveryLanguage(t *testing.T) {
	for _, lang := range []string{"uz", "ru", "en", "de", ""} {
		text, button := botMenuPrompt(lang, "Osh Markazi")
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
