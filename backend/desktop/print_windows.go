//go:build windows

package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"restaurant-backend/internal/escpos"
	"restaurant-backend/internal/printer"
)

// PrintOptions is everything about paper that this machine decides.
//
// ⚠️ **These belong to a printer the server does not know about, and that is
// the whole reason this exists.** When the branch has a printer configured,
// the server assembles escpos.Options from that record and queues the finished
// bytes — the agent loop prints them and nothing here is involved. This path
// covers the other case: a monoblock with a USB printer that nobody has
// entered into the panel yet, which today falls through to the browser's print
// dialog (lib/print.ts). One printer, one set of options, one decider.
//
// ⚠️ The field names deliberately mirror escpos.Options rather than being
// simplified. A local vocabulary would have to be translated every time the two
// are compared, and they will be compared — by whoever is holding a receipt
// that came out right on one printer and wrong on the other.
type PrintOptions struct {
	// Target is what the owner typed: "usb://XP-58", "192.168.1.50:9100",
	// "COM3", a share path. See printer.Parse for the whole list.
	Target string `json:"target"`
	// Charset is "cyrillic" for a Russian menu, anything else for Latin —
	// the same two values, spelled the same way, as the panel's printer record.
	Charset string `json:"charset"`
	// FeedLines is blank lines before the cut, so the tear-off lands below the
	// last line rather than through it.
	FeedLines int `json:"feedLines"`
	// Cut the paper. ⚠️ Off for printers without a cutter — some of them print
	// the command instead of ignoring it.
	Cut bool `json:"cut"`
	// FullCut severs the roll; partial leaves a tab, which is what most
	// restaurants want.
	FullCut bool `json:"fullCut"`
	// OpenDrawer kicks the cash drawer. ⚠️ Only ever for the till's own copy:
	// a drawer that springs open on every kitchen ticket is a drawer somebody
	// props shut with a fork.
	OpenDrawer bool `json:"openDrawer"`
}

// PrintLines encodes already-laid-out receipt lines and writes them to a
// printer attached to this machine.
//
// ⚠️ **It encodes; it does not lay out.** The lines arrive from the server,
// character by character, from the same receipt.Render that draws the preview
// the owner approved. Re-wrapping or re-aligning them here would produce a
// second layout, and the difference would be found by a guest holding the
// paper — the warning lib/print.ts opens with.
func (a *App) PrintLines(lines []string, o PrintOptions) error {
	if len(lines) == 0 {
		return fmt.Errorf("chop etishga hech nima yo'q")
	}
	o = a.printDefaults(o)
	if o.Target == "" {
		// ⚠️ **Told, not swallowed.** The screen falls back to the browser
		// dialog on any error here, and it needs this one to be an error rather
		// than a silent success — a "printed" that produced no paper is the
		// failure this whole path exists to remove.
		return fmt.Errorf("bu kompyuterda printer topilmadi")
	}
	target, err := printer.Parse(o.Target)
	if err != nil {
		return fmt.Errorf("printer manzili noto'g'ri: %w", err)
	}
	payload := escpos.Encode(lines, escpos.Options{
		Charset:    charsetOf(o.Charset),
		FeedLines:  o.FeedLines,
		Cut:        o.Cut,
		FullCut:    o.FullCut,
		OpenDrawer: o.OpenDrawer,
	})
	return printer.Send(a.ctx, target, payload)
}

// charsetOf mirrors handlers.charsetOf: one spelling is recognised and
// everything else is Latin.
//
// ⚠️ Unknown values fall back rather than failing. A charset nobody recognises
// is a receipt printed in the wrong code page — readable enough to notice and
// fix; refusing to print is a till that has stopped selling.
func charsetOf(name string) escpos.Charset {
	if name == string(escpos.Cyrillic) {
		return escpos.Cyrillic
	}
	return escpos.Latin
}

// printDefaults fills in what the screen did not say.
//
// ⚠️ **The machine's own settings first, the Windows default last.** Three
// sources answer "which printer", and the order is the order of how deliberate
// each one is: what the till was told on this machine (till.json), then what
// Windows prints to when nothing is chosen. A restaurant that never opens a
// settings screen still prints, and one that chose a printer keeps it even
// after somebody makes the office laser the Windows default — which happens,
// and would otherwise send every receipt to A4 in another room.
func (a *App) printDefaults(o PrintOptions) PrintOptions {
	p := a.cfg.Print
	if o.Target == "" {
		// ⚠️ The rule itself lives in internal/printer, where it can be tested:
		// nothing about "which printer wins" needs a spooler, and a rule that
		// only runs on Windows is a rule nobody checks. This file supplies the
		// two answers; it does not decide between them.
		o.Target = printer.Choose(p.Target, printer.Default())
	}
	if o.Charset == "" {
		o.Charset = p.Charset
	}
	if o.FeedLines == 0 {
		o.FeedLines = p.FeedLines
	}
	// ⚠️ **The booleans come from the settings whenever the caller did not ask
	// for them**, and "did not ask" is false — which is why the caller sends
	// the drawer flag explicitly and never relies on this. Cut is the opposite:
	// almost every till printer has a cutter, so a stored `cut: false` is a
	// deliberate answer about a printer that prints the command instead of
	// obeying it, and it must win over a caller that simply left it out.
	if !o.Cut {
		o.Cut = p.Cut
	}
	if !o.FullCut {
		o.FullCut = p.FullCut
	}
	return o
}

// Printers is what this machine can print to, for the settings screen.
func (a *App) Printers() []printer.Installed { return printer.List() }

// PrintConfig is the printer this till uses, as the settings screen sees it.
type PrintConfig struct {
	// What was chosen, or "" for "whatever Windows prints to".
	Target string `json:"target"`
	// The name that choice resolves to right now — including when nothing was
	// chosen, so the screen can show what will actually happen rather than an
	// empty box that looks unconfigured.
	Effective string             `json:"effective"`
	Charset   string             `json:"charset"`
	FeedLines int                `json:"feedLines"`
	Cut       bool               `json:"cut"`
	FullCut   bool               `json:"fullCut"`
	Drawer    bool               `json:"drawer"`
	Printers  []printer.Installed `json:"printers"`
}

// PrintConfig hands the screen the settings and the list in one call.
//
// ⚠️ One call rather than two, because they are read together and a screen
// that renders a chosen printer before the list arrives shows an empty select
// with a value — which reads as the setting having been lost.
func (a *App) PrintConfig() PrintConfig {
	p := a.cfg.Print
	eff := a.printDefaults(PrintOptions{}).Target
	return PrintConfig{
		Target:    p.Target,
		Effective: strings.TrimPrefix(eff, "usb://"),
		Charset:   p.Charset,
		FeedLines: p.FeedLines,
		Cut:       p.Cut,
		FullCut:   p.FullCut,
		Drawer:    p.Drawer,
		Printers:  printer.List(),
	}
}

// SavePrintConfig writes the choice to till.json.
//
// ⚠️ **Saved on this machine, not in the panel.** A printer plugged into this
// monoblock is a fact about this monoblock: the same branch runs a second till
// with its own printer, and a branch-wide setting would have them fight over
// one name. The panel's printer records are the other case — the shared kitchen
// printer every till queues to — and the two do not overlap.
func (a *App) SavePrintConfig(c PrintConfig) error {
	a.cfg.Print = printSettings{
		Target:    strings.TrimSpace(c.Target),
		Charset:   c.Charset,
		FeedLines: c.FeedLines,
		Cut:       c.Cut,
		FullCut:   c.FullCut,
		Drawer:    c.Drawer,
	}
	if err := saveSettings(a.cfg); err != nil {
		return fmt.Errorf("saqlanmadi: %w", err)
	}
	log.Printf("printer tanlandi: %q", a.cfg.Print.Target)
	return nil
}

// TestPrint puts a sheet through the chosen printer.
//
// ⚠️ **The point is the paper, not the return value.** Every part of this can
// be right and still produce nothing — the printer is off, out of paper, or the
// name belongs to a driver for a printer that was unplugged last month — and
// none of that reaches us as an error, because the spooler accepts the job.
// So the screen says "look at the printer", and this only reports the failures
// that happen before the queue.
func (a *App) TestPrint() error {
	return a.PrintLines([]string{
		"Keel — sinov cheki",
		"",
		"Agar buni o'qiyotgan bo'lsangiz,",
		"kassa printeri to'g'ri sozlangan.",
		"",
		time.Now().Format("02.01.2006 15:04"),
	}, PrintOptions{Cut: true})
}
