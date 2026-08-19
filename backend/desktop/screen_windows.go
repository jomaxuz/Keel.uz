//go:build windows

package main

import "syscall"

// How big the screen actually is.
//
// ⚠️ **Because the till is drawn for a wider screen than most of them have.**
// The design assumes roughly 1280 logical pixels across; the hardware it is
// sold onto is commonly 1024×768. At 1:1 the layout does not fit, and it does
// not fail cleanly — labels overlap, the bottom row is pushed off to one side,
// and the browser spends every frame laying out and repainting a page larger
// than the window it is in. "Everything is slow" and "everything is too big"
// were one problem.
var (
	user32              = syscall.NewLazyDLL("user32.dll")
	procGetSystemMetric = user32.NewProc("GetSystemMetrics")
)

const smCXScreen = 0

func screenWidth() int {
	w, _, _ := procGetSystemMetric.Call(uintptr(smCXScreen))
	return int(w)
}

// designWidth is the width the till's layout is drawn for.
const designWidth = 1280.0

// autoZoom scales the screen up to the width the design needs.
//
// ⚠️ **Never above 1.** On a large display the answer is more room, not bigger
// text: magnifying a till that already fits wastes the space that lets a
// cashier see more of the check at once.
//
// ⚠️ **And never below 0.65.** Past that the design is legible only to somebody
// who already knows where everything is, and a till nobody can read across a
// counter has failed differently rather than less.
func autoZoom() float64 {
	w := float64(screenWidth())
	if w <= 0 {
		return 1
	}
	z := w / designWidth
	if z >= 1 {
		return 1
	}
	if z < 0.65 {
		return 0.65
	}
	return z
}
