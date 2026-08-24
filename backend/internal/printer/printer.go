// Package printer sends bytes to a receipt printer, however it is plugged in.
//
// ⚠️ **Four ways, because restaurants have all four.** The unit sold with a
// monoblock in Tashkent is USB; the one at the pass is usually on the network
// because the kitchen is thirty metres from the till; older installs are on a
// COM port; and a printer somebody already set up in Windows is reachable by
// its share. A till that only speaks one of these is a till that cannot be
// installed in half the restaurants that would buy it.
//
// ⚠️ **No driver, no dependency.** Every transport here is an ordinary write to
// a socket or a file handle — which is what the Windows spooler, the Linux USB
// class driver and a serial port all expose.
//
// ⚠️ **Except on Windows, where the share was an install step that fails.**
// Reaching a USB printer as \\localhost\NAME requires the printer to be
// *shared*, and sharing on a Windows 10/11 machine in a restaurant means
// network discovery, sometimes a password prompt and occasionally a policy
// nobody in the building can change. So the printer's own name is tried
// through the spooler first (`winspool.drv`, called through the lazy DLL
// loader — no cgo, so this still cross-compiles from Linux), and the share
// path stays as the fallback. Nobody has to retype a setting: `usb://XP-58`
// already carries the name.
package printer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

// errNoSpooler means this machine has no Windows print spooler — the normal
// case everywhere except a monoblock, and never worth logging.
var errNoSpooler = errors.New("spooler yo'q")

// Kind is how a printer is attached.
type Kind string

const (
	// Over the network, on the port every ESC/POS printer answers.
	Network Kind = "net"
	// A Windows printer share, or a device path on Linux. USB printers arrive
	// here: Windows shares them by name, Linux exposes /dev/usb/lp0.
	Device Kind = "device"
	// A serial port. ⚠️ Windows needs the port's speed set once with `mode`;
	// writing to \\.\COM3 does not configure it.
	Serial Kind = "serial"
)

// Target is one printer, as the panel stores it.
//
// ⚠️ **A single string, not a form of five fields.** The person setting this up
// is reading a sticker on the printer or a line in its self-test page, and one
// box they can paste into is the difference between a restaurant that installs
// this and one that phones us.
//
//	tcp://192.168.1.50:9100     — network (the port is optional, 9100 is default)
//	usb://XP-58                 — a Windows printer share on this PC
//	\\\\SERVER\\XP-58            — the same thing written the way Windows does
//	device:///dev/usb/lp0       — Linux USB
//	serial://COM3               — a COM port
type Target struct {
	Kind Kind
	// Where to write: "host:9100", a share or device path, or a port name.
	Addr string
	// The printer's name as Windows knows it, when the address named one.
	//
	// ⚠️ Kept beside Addr rather than replacing it: the spooler is tried first
	// and the share is what answers when this machine is not Windows, when the
	// name is a share on another PC, or when the spooler refuses.
	Name string
}

// Parse reads what the owner typed.
//
// ⚠️ **Bare host names and Windows paths are accepted**, because both are what
// people actually paste. "192.168.1.50" is a printer on the network and
// "\\\\PC\\XP-58" is a share, and refusing them to insist on a scheme would be
// a support call for every install.
func Parse(raw string) (Target, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return Target{}, errors.New("printer manzili yozilmagan")
	}
	// A Windows UNC share, pasted exactly as Windows shows it.
	if strings.HasPrefix(s, `\\`) {
		return Target{Kind: Device, Addr: s}, nil
	}
	if !strings.Contains(s, "://") {
		// A bare path is a device, anything else is a host on the network.
		if strings.HasPrefix(s, "/") {
			return Target{Kind: Device, Addr: s}, nil
		}
		return Target{Kind: Network, Addr: withPort(s)}, nil
	}
	// ⚠️ **A printer name is read before any URL parsing, and never through
	// it.** Windows names printers the way people name things — "EPSON
	// TM-T20III Receipt", "XP-58 (Copy 1)", a Cyrillic name from a Russian
	// driver — and `url.Parse` rejects a space in a host outright. So every
	// name with a space in it, which is most of them, failed here with
	// "invalid character in host name": a printer chosen from a list the
	// machine itself reported, refused as a malformed address. The scheme has
	// no query, no port and no escaping to interpret; the rest of the string
	// *is* the name.
	if rest, ok := cutScheme(s, "usb", "share", "printer"); ok {
		name := strings.Trim(rest, `\/`)
		if name == "" {
			return Target{}, errors.New("printer nomi yozilmagan")
		}
		// ⚠️ Two shapes arrive here and they mean different machines.
		// `usb://XP-58` is a printer installed on *this* PC — the spooler can
		// be asked for it by name, which is what removes `net share` from the
		// install. `usb://SERVER/XP-58` names somebody else's, and this
		// machine's spooler has never heard of it: asking would turn one clear
		// failure into two confusing ones.
		if host, share, ok := strings.Cut(name, "/"); ok {
			return Target{Kind: Device, Addr: `\\` + host + `\` + share}, nil
		}
		return Target{Kind: Device, Addr: `\\localhost\` + name, Name: name}, nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return Target{}, fmt.Errorf("printer manzili tushunarsiz: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "tcp", "net", "socket", "lan":
		return Target{Kind: Network, Addr: withPort(u.Host)}, nil
	case "device", "file":
		// device:///dev/usb/lp0 — the path is what matters.
		return Target{Kind: Device, Addr: u.Path}, nil
	case "serial", "com":
		// ⚠️ `serial://COM3` puts the name in Host, `serial:///dev/ttyUSB0` puts
		// it in Path — and the leading slash of a device path is part of the
		// path, not punctuation to strip.
		port := u.Host
		if port == "" {
			port = u.Path
		}
		return Target{Kind: Serial, Addr: serialPath(port)}, nil
	}
	return Target{}, fmt.Errorf("printer turi noma'lum: %s", u.Scheme)
}

// withPort adds the port every ESC/POS printer listens on.
//
// ⚠️ 9100 is the raw/JetDirect port, and it is the one thing about network
// printers that is genuinely universal.
func withPort(host string) string {
	if host == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}
	return net.JoinHostPort(host, "9100")
}

// serialPath is how a COM port is opened on each system.
func serialPath(port string) string {
	if strings.HasPrefix(port, "/") {
		return port // /dev/ttyUSB0
	}
	if strings.HasPrefix(strings.ToUpper(port), "COM") {
		// ⚠️ The \\.\ prefix is required above COM9 on Windows and harmless
		// below it — and "COM10 does not work" is the classic install failure.
		return `\\.\` + strings.ToUpper(port)
	}
	return port
}

// Send writes one job to one printer.
//
// ⚠️ **Timeouts on everything.** A printer that is switched off does not refuse
// a connection, it fails to answer — and without a deadline the agent stops
// dead in a restaurant with no way to see why.
func Send(ctx context.Context, t Target, payload []byte) error {
	switch t.Kind {
	case Network:
		return sendNet(ctx, t.Addr, payload)
	case Device, Serial:
		// ⚠️ The spooler first when a name is known, the path second. On any
		// system without a spooler this returns errNoSpooler immediately and
		// costs nothing; on Windows it removes `net share` from the install.
		if t.Name != "" {
			if err := spoolPrint(t.Name, payload); err == nil {
				return nil
			} else if !errors.Is(err, errNoSpooler) {
				// The spooler exists and refused — usually a name that does
				// not match any installed printer. The share is still worth
				// trying, and the reason is worth having in the log rather
				// than replaced by whatever the second attempt says.
				log.Printf("printer: spooler %q: %v (share bilan urinilmoqda)", t.Name, err)
			}
		}
		return sendFile(t.Addr, payload)
	}
	return fmt.Errorf("printer turi qo'llab-quvvatlanmaydi: %s", t.Kind)
}

func sendNet(ctx context.Context, addr string, payload []byte) error {
	d := net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("printerga ulanib bo'lmadi (%s): %w", addr, err)
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
	if _, err := conn.Write(payload); err != nil {
		return fmt.Errorf("printerga yozib bo'lmadi (%s): %w", addr, err)
	}
	return nil
}

// sendFile writes to a share, a device node or a COM port.
//
// ⚠️ **Opened write-only and appending.** A printer is not a file: opening it
// for reading fails on a Windows share, and truncating is meaningless for a
// device — both are mistakes that only show up on the machine in the restaurant.
func sendFile(path string, payload []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		// Windows shares cannot be opened with O_APPEND on every build; a plain
		// write-only open is the fallback rather than a failure.
		f, err = os.OpenFile(path, os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("printer topilmadi (%s): %w", path, err)
		}
	}
	defer f.Close()
	if _, err := f.Write(payload); err != nil {
		return fmt.Errorf("printerga yozib bo'lmadi (%s): %w", path, err)
	}
	return nil
}

// Installed is one printer this machine can reach, as a person should see it.
//
// ⚠️ **The name is the address.** `usb://<name>` is what Parse already
// understands, so a chosen row needs no translation and no second field — the
// same string is stored in the panel, tried through the spooler, and falls back
// to the share path if the spooler refuses.
type Installed struct {
	Name string `json:"name"`
	// Whether Windows prints here when a program does not choose.
	Default bool `json:"default"`
	// The port Windows has it on: "USB001", "192.168.1.50", "IP_192.168.1.50",
	// "COM3", a share. Shown to a person so a row with two similar names can be
	// told apart, and read by `Target` below.
	Port string `json:"port"`
	// Where to send bytes, worked out from the port — so nobody types an
	// address at all. Empty when the port says nothing useful, in which case the
	// spooler by name is still correct and is what the screen falls back to.
	Target string `json:"target"`
}

// TargetFromPort turns what Windows calls a printer's port into an address.
//
// ⚠️ **This is what removes the form.** Windows already knows how every printer
// is attached and where: a receipt printer on the counter is on "USB001", the
// one at the pass is on "192.168.1.50" or "IP_192.168.1.50", an old one is on
// "COM3". Asking somebody to choose "USB or LAN" and then to read an IP address
// off the back of a machine bolted under a shelf is asking them to retype
// something the operating system was willing to say.
//
// ⚠️ **A network port becomes a socket, everything else becomes the spooler.**
// For a network printer, going direct is better than going through Windows —
// the driver is skipped entirely, which is what makes ESC/POS come out as
// ESC/POS. For USB and serial the spooler *is* the way in (it is what removes
// `net share` from the install), so the address stays the printer's name.
func TargetFromPort(name, port string) string {
	p := strings.TrimSpace(port)
	// ⚠️ Windows' own naming for a TCP/IP port, and the prefix is not part of
	// the address. "IP_192.168.1.50" dialled as a host is a DNS lookup that
	// fails, which reads as the printer being offline.
	host := strings.TrimPrefix(p, "IP_")
	if h, _, ok := strings.Cut(host, ":"); ok {
		host = h
	}
	if net.ParseIP(host) != nil {
		return "tcp://" + net.JoinHostPort(host, "9100")
	}
	if name = strings.TrimSpace(name); name != "" {
		return "usb://" + name
	}
	return ""
}

// Default is the printer Windows uses when nothing is configured, or "".
func Default() string { return defaultPrinter() }

// Choose settles which printer a machine prints to.
//
// ⚠️ **Here, and not in the Windows-only file it is called from, so it can be
// tested at all.** Every other line of that path needs a spooler; this rule
// needs nothing, and it is the part that decides whether a restaurant's
// receipts come out of its receipt printer or the office laser down the hall.
// A rule that can only run on the machine it is wrong on is a rule nobody
// checks — the same reason exposeDemoCode and downloadsJSON are functions.
//
// `configured` is what somebody chose on this machine, `fallback` the printer
// Windows uses when a program does not choose.
//
// ⚠️ **A chosen printer always wins**, including over a Windows default that
// changed afterwards — which happens the first time somebody installs an office
// printer on the till, and would otherwise send every receipt to A4 in another
// room with nothing on the till saying so.
//
// ⚠️ **A bare name is wrapped as `usb://`.** Parse reads a bare word with no
// scheme as a *network host*, so an unwrapped "XP-58" would be dialled as TCP
// 9100 on a machine of that name and time out — a printer that fails slowly
// rather than not at all, which is worse at a counter.
func Choose(configured, fallback string) string {
	if s := strings.TrimSpace(configured); s != "" {
		return qualify(s)
	}
	if s := strings.TrimSpace(fallback); s != "" {
		return qualify(s)
	}
	return ""
}

// qualify adds the scheme a bare Windows printer name needs, and leaves every
// address that already carries one — or is plainly not a name — alone.
func qualify(s string) string {
	if strings.Contains(s, "://") || strings.HasPrefix(s, `\\`) ||
		strings.HasPrefix(s, "/") || strings.Contains(s, ":") {
		return s
	}
	// ⚠️ A bare IP address means the printer on the network, not a printer
	// named "192.168.1.50" — and Parse already reads it that way.
	if looksLikeIP(s) {
		return s
	}
	return "usb://" + s
}

// looksLikeIP reports whether a bare word is a network address.
//
// ⚠️ **An IP address, and nothing looser.** The tempting rule — "it has a dot,
// so it is a host" — is wrong on real hardware: printers are called
// "EPSON TM-T20III Receipt" and, in one driver shipped in Tashkent, "XP-58.2".
// Reading either as a hostname sends the receipt to TCP 9100 on a machine that
// does not exist, and it fails by *timing out*, which at a counter is worse
// than not printing at all — the cashier waits, then prints again.
//
// Anything a person types that genuinely is a host reaches Parse through the
// panel's printer field, where it is written with a port or a scheme.
func looksLikeIP(s string) bool { return net.ParseIP(s) != nil }

// cutScheme returns what follows "<name>://" when the address carries one of
// the given schemes, matched case-insensitively.
//
// ⚠️ Deliberately not `url.Parse`: the point is to get the remainder *without*
// interpreting it. See the note in Parse — the remainder is a printer's name,
// and names contain spaces, brackets and Cyrillic.
func cutScheme(s string, names ...string) (string, bool) {
	i := strings.Index(s, "://")
	if i < 0 {
		return "", false
	}
	scheme := strings.ToLower(s[:i])
	for _, n := range names {
		if scheme == n {
			return s[i+3:], true
		}
	}
	return "", false
}
