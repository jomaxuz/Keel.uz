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

// ⚠️ **The printer's own name has to survive parsing**, because it is what the
// spooler is asked for — and the spooler is what removes `net share` from the
// install. Before this the name was folded into `\\localhost\NAME` and thrown
// away, so the only way to reach a USB printer was to share it: a step that
// needs network discovery, sometimes a credential prompt, and occasionally a
// Windows policy the restaurant cannot change.
func TestALocalPrinterKeepsItsName(t *testing.T) {
	got, err := Parse("usb://XP-58")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "XP-58" {
		t.Fatalf("name=%q, want XP-58 — the spooler has nothing to ask for", got.Name)
	}
	// ⚠️ And the share path stays: it is the fallback on a machine with no
	// spooler, which is every machine except the monoblock.
	if got.Addr != `\\localhost\XP-58` {
		t.Fatalf("addr=%q", got.Addr)
	}
}

// ⚠️ A UNC path is a share on **another** PC. This machine's spooler has never
// heard of it, and asking it would turn one clear failure into two confusing
// ones.
func TestAShareOnAnotherPcIsNotAskedOfThisSpooler(t *testing.T) {
	for _, in := range []string{`\\SERVER\XP-58`, "usb://SERVER/XP-58"} {
		got, err := Parse(in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if got.Name != "" {
			t.Fatalf("%s: name=%q — a remote share is not a local printer", in, got.Name)
		}
		if got.Addr != `\\SERVER\XP-58` {
			t.Fatalf("%s: addr=%q", in, got.Addr)
		}
	}
}

// A device node and a COM port name nothing: there is no spooler in either
// story, and a name here would send every Linux install through a failing
// Windows call first.
func TestDeviceAndSerialNameNoPrinter(t *testing.T) {
	for _, in := range []string{"device:///dev/usb/lp0", "serial://COM3", "/dev/usb/lp0"} {
		got, err := Parse(in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if got.Name != "" {
			t.Fatalf("%s: name=%q", in, got.Name)
		}
	}
}

// Which printer a till prints to, sealed because it is decided on a machine
// none of us can run a test on.
func TestChoosePrefersTheConfiguredPrinter(t *testing.T) {
	// ⚠️ The one that matters: somebody installs an office printer on the till
	// and Windows makes it the default. A restaurant that chose its receipt
	// printer keeps it — otherwise every sale that evening prints on A4 in
	// another room, and nothing on the till says so.
	if got := Choose("usb://XP-58", "HP LaserJet"); got != "usb://XP-58" {
		t.Fatalf("configured printer lost to the Windows default: %q", got)
	}
	if got := Choose("", "HP LaserJet"); got != "usb://HP LaserJet" {
		t.Fatalf("Windows default not used: %q", got)
	}
	if got := Choose("", ""); got != "" {
		t.Fatalf("invented a printer out of nothing: %q", got)
	}
}

// A bare name must come back addressable, and a name with a dot in it is still
// a name.
func TestChooseQualifiesNamesButNotAddresses(t *testing.T) {
	cases := map[string]string{
		// The whole reason qualify exists: Parse reads a bare word as a host,
		// so an unwrapped name is dialled on TCP 9100 and times out.
		"XP-58": "usb://XP-58",
		// ⚠️ Real driver names carry dots and spaces. Reading either as a
		// hostname is the slow failure described in looksLikeIP.
		"XP-58.2":                 "usb://XP-58.2",
		"EPSON TM-T20III Receipt": "usb://EPSON TM-T20III Receipt",
		// Already addressed — left exactly as written.
		"192.168.1.50":      "192.168.1.50",
		"192.168.1.50:9100": "192.168.1.50:9100",
		"tcp://10.0.0.9":    "tcp://10.0.0.9",
		`\\PC\XP-58`:        `\\PC\XP-58`,
		"/dev/usb/lp0":      "/dev/usb/lp0",
	}
	for in, want := range cases {
		if got := Choose(in, ""); got != want {
			t.Errorf("Choose(%q) = %q, want %q", in, got, want)
		}
	}
}

// Every address Choose produces must be one Parse understands: the two are used
// one after the other, and a value that survives the first and fails the second
// is a printer that is configured and cannot print.
func TestChooseProducesParseableTargets(t *testing.T) {
	for _, in := range []string{
		"XP-58", "XP-58.2", "EPSON TM-T20III Receipt",
		"192.168.1.50", "192.168.1.50:9100", `\\PC\XP-58`, "/dev/usb/lp0",
	} {
		if _, err := Parse(Choose(in, "")); err != nil {
			t.Errorf("Choose(%q) produced an address Parse rejects: %v", in, err)
		}
	}
}
