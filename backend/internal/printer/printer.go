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
// class driver and a serial port all expose. The alternative is winspool
// through cgo, which would make the agent unbuildable from Linux and add a
// second thing to sign.
package printer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

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
	u, err := url.Parse(s)
	if err != nil {
		return Target{}, fmt.Errorf("printer manzili tushunarsiz: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "tcp", "net", "socket", "lan":
		return Target{Kind: Network, Addr: withPort(u.Host)}, nil
	case "usb", "share", "printer":
		// usb://XP-58 — the name lands in Host, and on Windows a local share is
		// reached as \\localhost\<name>.
		name := u.Host + u.Path
		if !strings.HasPrefix(name, `\\`) {
			name = `\\localhost\` + strings.Trim(name, `\/`)
		}
		return Target{Kind: Device, Addr: name}, nil
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
