//go:build windows

package main

import (
	"fmt"

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
