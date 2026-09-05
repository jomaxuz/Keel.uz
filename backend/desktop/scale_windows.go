//go:build windows

package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ---- Reading a weight off a counter scale ----
//
// ⚠️ **The safety property here is that a person confirms the number.** A scale
// label is decoded and charged without anybody reading it, which is why that
// path is guarded so heavily. This one is different by construction: the weight
// is shown on the screen and the cashier presses "add" — so a misread is caught
// by the person holding the goods, before it reaches a check.
//
// ⚠️ **No serial library, and that is a deliberate limit rather than a
// preference.** Windows exposes a COM port as a file, so a port that has already
// been configured — baud rate, parity, stop bits, in Device Manager or with the
// `mode` command — can be read with nothing but the standard library. Adding a
// dependency to set those from here would be the tidy version and would also
// mean this binary decides settings an installer has already made, on hardware
// nobody here can test against.
//
// ⚠️ **Untested against a physical scale.** Every scale sold here streams ASCII
// continuously, and the shapes below are the ones CAS, Digi and Mettler emit —
// but "the ones documented" is not "the one in that shop". The port and a live
// reading are both on the settings screen for exactly this reason: whoever
// installs it finds out in a minute rather than at a stocktake.

// weightRe finds a decimal number in whatever the scale said.
//
// ⚠️ **A number, not a protocol.** These devices wrap the same figure in
// different envelopes — "ST,GS,   1.234kg", "1.234 kg", "+0001234". Parsing one
// vendor's frame would work in one shop; taking the first decimal number out of
// the line works in all of them and fails the same way in all of them, which is
// the failure that can be diagnosed.
var weightRe = regexp.MustCompile(`-?\d+[.,]\d+`)

// Weigh returns the current reading, in kilograms.
func (a *App) Weigh(port string) (float64, error) {
	port = strings.TrimSpace(port)
	if port == "" {
		return 0, fmt.Errorf("tarozi porti ko'rsatilmagan")
	}
	// ⚠️ `\\.\COM10` and above: the bare name only works up to COM9, and a shop
	// with a hub reaches double digits routinely. The failure without this is
	// "the scale works on one machine and not the next".
	name := port
	if !strings.HasPrefix(name, `\\.\`) {
		name = `\\.\` + name
	}

	f, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		return 0, fmt.Errorf("tarozi portini ochib bo'lmadi (%s): %w", port, err)
	}
	defer f.Close()

	// ⚠️ **Bounded, because a scale that is switched off never answers.** An
	// unbounded read would hang the till on the one action a cashier repeats,
	// and the screen would look frozen rather than say what is wrong.
	_ = f.SetReadDeadline(time.Now().Add(2 * time.Second))

	// ⚠️ **Several lines, not one.** A continuous stream is mid-frame when the
	// port is opened, so the first line read is usually a fragment. Taking the
	// last complete reading is both correct and what the display shows.
	sc := bufio.NewScanner(f)
	var last string
	for i := 0; i < 8 && sc.Scan(); i++ {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			last = line
		}
	}
	if last == "" {
		return 0, fmt.Errorf("tarozi javob bermadi — yoqilganini va portni tekshiring")
	}

	m := weightRe.FindString(last)
	if m == "" {
		return 0, fmt.Errorf("tarozidan tushunarsiz javob: %s", last)
	}
	kg, err := strconv.ParseFloat(strings.Replace(m, ",", ".", 1), 64)
	if err != nil {
		return 0, fmt.Errorf("tarozidan tushunarsiz javob: %s", last)
	}
	// ⚠️ **A negative or zero reading is refused rather than added.** A scale
	// that has not been tared reads below zero, and a zero is an empty pan —
	// both are somebody about to charge for nothing, and both are visible on the
	// scale's own display, so saying so sends them to look at it.
	if kg <= 0 {
		return 0, fmt.Errorf("tarozida og'irlik yo'q — tovarni qo'ying yoki tarani nolang")
	}
	return kg, nil
}
