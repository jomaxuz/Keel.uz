//go:build windows

package printer

import (
	"sort"
	"syscall"
	"unsafe"
)

// Asking Windows what printers this machine has.
//
// ⚠️ **Because nobody can type the name.** The spooler is addressed by the
// printer's name exactly as Windows spells it, and that spelling is not what is
// written on the box: a driver installs "XP-58", the second one on the counter
// becomes "XP-58 (Copy 1)", and a Russian-language driver names itself in
// Cyrillic. Every one of those is a receipt that never prints, with no error
// anybody sees — the job goes to a printer that does not exist and the spooler
// discards it. A list to choose from removes the whole class of failure.
//
// ⚠️ **Level 4, not level 2.** Level 2 fills in the port, the driver, the
// comment and the queue status, and to do it the spooler contacts every printer
// — including the network ones. On a machine with an offline printer installed
// that call blocks for its timeout, which on a till is the setup screen hanging
// for half a minute. Level 4 returns names only, from the local registry, and
// names are the entire question here.
var (
	procEnumPrinters      = winspool.NewProc("EnumPrintersW")
	procGetDefaultPrinter = winspool.NewProc("GetDefaultPrinterW")
)

// Flags for EnumPrinters: printers installed on this machine, plus the
// connections this user has to printers shared by other machines.
//
// ⚠️ **Both, because a kitchen printer is often the second kind.** The pass
// printer is regularly installed on the manager's PC and shared; the till
// reaches it as a connection, and enumerating only local printers would leave
// the one printer the restaurant most needs out of the list.
const (
	printerEnumLocal       = 0x00000002
	printerEnumConnections = 0x00000004
)

// printerInfo4 is PRINTER_INFO_4W.
//
// ⚠️ **No hand-written padding.** The tail padding after Attributes is 4 bytes
// on 64-bit and none at all on 32-bit, and Go's own layout rules produce
// exactly the C one on both — a field added to "make it match" is right on the
// machine it was tested on and silently misreads every second name on the
// other, because the stride would no longer be the struct's size.
type printerInfo4 struct {
	PrinterName *uint16
	ServerName  *uint16
	Attributes  uint32
}

// List returns the printers this machine can reach, the default one first.
//
// ⚠️ **An empty list is not an error.** A machine with no printer installed is
// the ordinary state of a laptop running the till for a demo, and the caller's
// correct response is to offer the browser dialog — not to show a failure the
// person cannot act on.
func List() []Installed {
	if err := winspool.Load(); err != nil {
		return nil
	}
	var needed, count uint32
	// First call sizes the buffer: r == 0 with everything nil is expected.
	procEnumPrinters.Call(
		uintptr(printerEnumLocal|printerEnumConnections), 0, 4,
		0, 0, uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&count)),
	)
	if needed == 0 {
		return nil
	}
	buf := make([]byte, needed)
	if r, _, _ := procEnumPrinters.Call(
		uintptr(printerEnumLocal|printerEnumConnections), 0, 4,
		uintptr(unsafe.Pointer(&buf[0])), uintptr(needed),
		uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&count)),
	); r == 0 {
		return nil
	}
	def := defaultPrinter()
	out := make([]Installed, 0, count)
	// ⚠️ The names are pointers *into* buf, not copies. `UTF16PtrToString`
	// reads them out; keeping the pointers would hand the caller strings that
	// stay valid only until this slice is collected.
	items := unsafe.Slice((*printerInfo4)(unsafe.Pointer(&buf[0])), count)
	for _, it := range items {
		name := utf16Str(it.PrinterName)
		if name == "" {
			continue
		}
		out = append(out, Installed{Name: name, Default: name == def})
	}
	// The default first, then alphabetical — the order a person scans.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Default != out[j].Default {
			return out[i].Default
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// defaultPrinter is what Windows prints to when a program does not choose.
//
// ⚠️ **This is the whole zero-configuration case.** On a monoblock sold with a
// receipt printer, that printer is the default and usually the only one — so a
// till that prints to the default prints correctly on the day it is installed,
// with nobody opening a settings screen at all.
func defaultPrinter() string {
	if err := winspool.Load(); err != nil {
		return ""
	}
	var n uint32
	procGetDefaultPrinter.Call(0, uintptr(unsafe.Pointer(&n)))
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n)
	if r, _, _ := procGetDefaultPrinter.Call(
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)),
	); r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func utf16Str(p *uint16) string {
	if p == nil {
		return ""
	}
	// Walk to the terminator rather than assuming a length: the strings sit
	// inside the buffer EnumPrinters filled and carry no count of their own.
	var n int
	for ptr := unsafe.Pointer(p); *(*uint16)(ptr) != 0; ptr = unsafe.Add(ptr, 2) {
		n++
	}
	return syscall.UTF16ToString(unsafe.Slice(p, n))
}
