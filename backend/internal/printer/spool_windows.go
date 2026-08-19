//go:build windows

package printer

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Printing through the Windows spooler, without cgo.
//
// ⚠️ **This is what removes `net share` from the install.** A USB receipt
// printer on a monoblock is an ordinary installed printer; reaching it as a
// file path requires it to be *shared*, which on Windows 10/11 means network
// discovery, sometimes a credential prompt, and occasionally a policy the
// restaurant cannot change. The spooler needs none of that — it is the same
// call every Windows program makes to print.
//
// ⚠️ **Datatype "RAW".** Anything else hands the bytes to a driver that will
// try to render them as a document; ESC/POS *is* the printer's language, and a
// driver rewriting it produces pages of garbage — or, worse, a receipt that
// looks right on one model and prints control codes on the next.
var (
	winspool           = syscall.NewLazyDLL("winspool.drv")
	procOpenPrinter    = winspool.NewProc("OpenPrinterW")
	procClosePrinter   = winspool.NewProc("ClosePrinter")
	procStartDocPrn    = winspool.NewProc("StartDocPrinterW")
	procEndDocPrinter  = winspool.NewProc("EndDocPrinter")
	procStartPagePrn   = winspool.NewProc("StartPagePrinter")
	procEndPagePrinter = winspool.NewProc("EndPagePrinter")
	procWritePrinter   = winspool.NewProc("WritePrinter")
)

type docInfo1 struct {
	DocName    *uint16
	OutputFile *uint16
	Datatype   *uint16
}

func spoolPrint(name string, payload []byte) error {
	if err := winspool.Load(); err != nil {
		return errNoSpooler
	}
	wName, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return fmt.Errorf("printer nomi noto'g'ri: %w", err)
	}
	var h syscall.Handle
	if r, _, e := procOpenPrinter.Call(
		uintptr(unsafe.Pointer(wName)), uintptr(unsafe.Pointer(&h)), 0,
	); r == 0 {
		return fmt.Errorf("printer ochilmadi (%s): %w", name, e)
	}
	defer procClosePrinter.Call(uintptr(h))

	doc, _ := syscall.UTF16PtrFromString("chek")
	raw, _ := syscall.UTF16PtrFromString("RAW")
	info := docInfo1{DocName: doc, Datatype: raw}
	if r, _, e := procStartDocPrn.Call(
		uintptr(h), 1, uintptr(unsafe.Pointer(&info)),
	); r == 0 {
		return fmt.Errorf("chop etish boshlanmadi (%s): %w", name, e)
	}
	defer procEndDocPrinter.Call(uintptr(h))

	if r, _, e := procStartPagePrn.Call(uintptr(h)); r == 0 {
		return fmt.Errorf("chop etish boshlanmadi (%s): %w", name, e)
	}
	defer procEndPagePrinter.Call(uintptr(h))

	// ⚠️ Written in one call and the count is checked. A short write on a
	// receipt is a bill cut off in the middle, which the person holding it
	// reads as the restaurant's arithmetic rather than a printer fault.
	var written uint32
	if r, _, e := procWritePrinter.Call(
		uintptr(h), uintptr(unsafe.Pointer(&payload[0])),
		uintptr(len(payload)), uintptr(unsafe.Pointer(&written)),
	); r == 0 {
		return fmt.Errorf("printerga yozib bo'lmadi (%s): %w", name, e)
	}
	if int(written) != len(payload) {
		return fmt.Errorf("printerga to'liq yozilmadi (%s): %d/%d",
			name, written, len(payload))
	}
	return nil
}
