package printer

import "testing"

// ⚠️ **Everything here is something a restaurant actually pastes into the box.**
// The address is copied off a printer's self-test page, out of Windows' printer
// list, or from an installer's notes — and a parser that insisted on one
// spelling would be a support call per install.
func TestParseAcceptsWhatPeopleActuallyType(t *testing.T) {
	cases := []struct {
		in   string
		kind Kind
		addr string
	}{
		// A network printer, with and without the port. 9100 is the raw port
		// every ESC/POS unit answers on, and nobody types it.
		{"tcp://192.168.1.50:9100", Network, "192.168.1.50:9100"},
		{"tcp://192.168.1.50", Network, "192.168.1.50:9100"},
		{"192.168.1.50", Network, "192.168.1.50:9100"},
		{"printer.local", Network, "printer.local:9100"},

		// USB, which on Windows means a share by name.
		{"usb://XP-58", Device, `\\localhost\XP-58`},
		{`\\KASSA-PC\XP-58`, Device, `\\KASSA-PC\XP-58`},

		// Linux USB, and a bare path.
		{"device:///dev/usb/lp0", Device, "/dev/usb/lp0"},
		{"/dev/usb/lp0", Device, "/dev/usb/lp0"},

		// ⚠️ COM10 and above need the \\.\ prefix on Windows — "it works on
		// COM3 but not COM10" is the classic serial install failure.
		{"serial://COM3", Serial, `\\.\COM3`},
		{"com://COM10", Serial, `\\.\COM10`},
		{"serial:///dev/ttyUSB0", Serial, "/dev/ttyUSB0"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := Parse(c.in)
			if err != nil {
				t.Fatalf("refused: %v", err)
			}
			if got.Kind != c.kind || got.Addr != c.addr {
				t.Fatalf("got %s %q, want %s %q", got.Kind, got.Addr, c.kind, c.addr)
			}
		})
	}
}

func TestParseRefusesNothingAndNonsense(t *testing.T) {
	if _, err := Parse("   "); err == nil {
		t.Fatal("an empty address was accepted")
	}
	// ⚠️ Named rather than guessed: a typo'd scheme silently treated as a
	// hostname would sit there timing out, and the restaurant would be told
	// "the printer is not answering" about a printer that is switched on.
	if _, err := Parse("bluetooth://XP-58"); err == nil {
		t.Fatal("an unsupported transport was accepted")
	}
}
