// Printing a receipt from the till.
//
// ⚠️ **Three paths, and the screen picks without being told which.** A branch
// with printers configured in the panel gets paper from the queue and never
// reaches this file (`queued > 0`). Everything else lands here, and the order
// is: the Windows till's own printer through the spooler, then the browser's
// print dialog.
//
// ⚠️ **The dialog is the fallback, not the design.** On a monoblock it is a
// modal the cashier taps through for every single sale — with no keyboard, on a
// touch screen, at the moment a guest is waiting — which is the difference
// between a till and a web page that can print. It stays because the browser
// till is real (a tablet on the floor, a laptop during a demo) and because a
// printer that has just been unplugged must not stop the bill being handed
// over.
//
// ⚠️ **The lines are laid out by the server**, character by character, using the
// same code that draws the preview in the settings. Nothing here re-wraps or
// re-aligns: a second layout would drift from the one the owner approved, and
// the difference would be found by a guest holding the paper.

import { canPrintLocally, printLines } from "@/lib/tillBridge";

/** How wide the paper is, in millimetres, when the branch has not said. */
const DEFAULT_MM = 80;

/**
 * Send already-rendered receipt lines to the printer.
 *
 * ⚠️ **Fire-and-forget on purpose.** Every call site prints as the last step of
 * something that has already happened — the sale is closed, the shift is
 * counted — so awaiting paper would make the screen wait on a device, and a
 * printer that is off would leave the till looking wedged after a sale that
 * actually succeeded.
 */
export function printReceipt(
  lines: string[],
  widthMM = DEFAULT_MM,
  logoUrl = "",
  opts: { drawer?: boolean } = {},
): void {
  if (typeof document === "undefined") return;
  if (canPrintLocally()) {
    void printLines(lines, {
      // ⚠️ Left empty rather than guessed: the Go side resolves the machine's
      // own setting and then the Windows default, and a target invented here
      // would override a choice somebody made in front of the printer.
      target: "",
      // ⚠️ **The drawer flag is passed, never defaulted.** It opens for the
      // till's own copy of a sale and nothing else — the same rule the queue
      // applies (printqueue.go) — and a drawer that springs open on a bill or
      // a kitchen ticket is one somebody props shut with a fork.
      openDrawer: opts.drawer === true,
    }).then((printed) => {
      if (!printed) browserPrint(lines, widthMM, logoUrl);
    });
    return;
  }
  browserPrint(lines, widthMM, logoUrl);
}

/** The browser's own print dialog: the fallback, and the whole browser till.
 *
 * ⚠️ **A detached iframe, not a new window.** A popup is blocked by default on
 * a machine nobody has configured, and the block is silent — the cashier
 * presses "print" and watches nothing happen. The iframe is removed once the
 * dialog is done with it. */
function browserPrint(lines: string[], widthMM: number, logoUrl: string): void {
  const frame = document.createElement("iframe");
  // Off-screen rather than hidden: `display: none` is not printed by every
  // engine, and a receipt that prints blank looks exactly like a paper jam.
  frame.setAttribute("aria-hidden", "true");
  frame.style.position = "fixed";
  frame.style.right = "100%";
  frame.style.bottom = "100%";
  frame.style.width = "0";
  frame.style.height = "0";
  frame.style.border = "0";
  document.body.appendChild(frame);

  const doc = frame.contentDocument;
  if (!doc) {
    frame.remove();
    return;
  }
  const mm = widthMM === 58 ? 58 : DEFAULT_MM;
  doc.open();
  doc.write(`<!doctype html><html><head><meta charset="utf-8">
<title>chek</title>
<style>
  /* ⚠️ The page **is** the paper: a receipt printer feeds continuous roll, and
     an A4 page box would put the receipt in the top-left corner of a sheet the
     printer does not have. */
  @page { size: ${mm}mm auto; margin: 0; }
  html, body { margin: 0; padding: 0; }
  pre {
    margin: 0;
    padding: 2mm 1.5mm;
    /* ⚠️ Monospace, and the server counted characters in it. Any proportional
       face turns a column of totals into a ragged edge. */
    font-family: "Courier New", ui-monospace, monospace;
    font-size: ${mm === 58 ? "10.5px" : "12px"};
    line-height: 1.25;
    white-space: pre-wrap;
    word-break: break-word;
  }
  /* ⚠️ **A fixed share of the paper, not "nearly all of it".** At 80% the mark
     is the biggest thing on the receipt and the guest is handed a poster with
     their total underneath — and it is the one element nobody reads closely.
     The share matches what the thermal path prints (escpos.LogoDots80 of a
     576-dot head), so a restaurant that prints through the browser and one
     that prints through the agent get the same receipt. */
  img { display: block; margin: 2mm auto 1mm; width: ${mm === 58 ? 50 : 44}%; height: auto; }
</style></head><body></body></html>`);
  doc.close();
  // ⚠️ **Built as nodes, never as an HTML string.** A dish called "<b>" is a
  // dish, not markup — and the receipt is assembled from the restaurant's own
  // menu, which is exactly the text nobody sanitises.
  if (logoUrl) {
    const img = doc.createElement("img");
    img.src = logoUrl;
    img.alt = "";
    doc.body.appendChild(img);
  }
  const pre = doc.createElement("pre");
  pre.textContent = lines.join("\n");
  doc.body.appendChild(pre);

  const win = frame.contentWindow;
  if (!win) {
    frame.remove();
    return;
  }
  win.focus();
  win.print();
  // ⚠️ Removed on a timer rather than after `print()` returns: in several
  // engines the call is asynchronous, and tearing the frame down underneath an
  // open dialog cancels the job that was about to be sent.
  window.setTimeout(() => frame.remove(), 60_000);
}
