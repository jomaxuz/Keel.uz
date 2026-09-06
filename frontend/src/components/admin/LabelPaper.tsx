"use client";

// One label, drawn the size it prints.
//
// ⚠️ **This is the whole point of the chooser.** A shop picking between six
// designs is asking one question — what comes out of the printer — and the only
// honest way to answer it on a screen is to draw the paper: the same character
// grid, the same two type sizes, the same bars, and the cut where the feed puts
// it. A list of six names answers nothing.
//
// ⚠️ **The text is not reformatted here.** The lines arrive already laid out by
// internal/receipt, which is the code that drives the printer; this component
// only decides how each one *looks*. Wrapping or padding anything here would be
// a second layout engine, and the drift would be found by a shop holding a
// sticker that does not match the card they chose.

import { ean13Modules, EAN13_GUARDS } from "@/lib/ean13";

/** The emphasis markers escpos reads, taken off the front of a line.
 *
 *  ⚠️ **Shown as size and weight rather than dropped.** They are control
 *  characters — a preview that printed them raw would show nothing at all, and
 *  "which line is the big one" is exactly what is being chosen here. */
function marked(line: string): { text: string; bold: boolean; big: boolean } {
  const head = line.charCodeAt(0);
  if (head === 1) return { text: line.slice(1), bold: true, big: false };
  if (head === 2) return { text: line.slice(1), bold: false, big: true };
  if (head === 3) return { text: line.slice(1), bold: true, big: true };
  return { text: line, bold: false, big: false };
}

export default function LabelPaper({
  lines,
  bars,
  code,
  widthMm,
  feedLines,
}: {
  lines: string[];
  /** Whether the printer draws a barcode under the text. */
  bars: boolean;
  code: string;
  widthMm: number;
  feedLines: number;
}) {
  // The paper is a character grid: 32 columns on 58 mm, 48 on 80 mm. Everything
  // below is measured in `ch` of the same monospace face, so a line that will
  // run off the roll runs off the card.
  const cols = widthMm === 58 ? 32 : 48;
  const modules = bars ? ean13Modules(code) : null;

  return (
    <div
      className="mx-auto inline-block bg-white font-mono text-black shadow-[0_1px_6px_rgba(0,0,0,0.18)]"
      style={{ fontSize: "11px", lineHeight: 1.25 }}
    >
      {/* ⚠️ **The width is on the text, not on the paper.** Put on the outer box
          it is the padding that gets the columns, and a line using the whole
          roll — which the renderer is allowed to produce, and the price line
          routinely does — hangs off the edge of the card while printing
          perfectly. The card would then be showing a fault the printer does not
          have. */}
      <div
        className="pb-1 pt-2"
        style={{ width: `${cols}ch`, paddingLeft: "4px", paddingRight: "4px", boxSizing: "content-box" }}
      >
        {lines.map((line, i) => {
          const m = marked(line);
          return (
            <div
              key={i}
              className={`${m.bold ? "font-bold" : ""} whitespace-pre`}
              // ⚠️ **Two em, because `GS !` is double width *and* height.** The
              // grid still lines up: every character of a big line takes two
              // columns of the same paper, which is exactly what makes a long
              // price run off a 58 mm roll — and why the renderer prints one at
              // ordinary size instead of cutting it.
              style={m.big ? { fontSize: "2em", lineHeight: 1.1 } : undefined}
            >
              {m.text === "" ? " " : m.text}
            </div>
          );
        })}

        {bars && (
          <div className="mt-1">
            {modules ? (
              // ⚠️ Drawn from the code's own modules rather than as a stripe:
              // the width of a barcode is the thing a shop is judging against
              // the width of its labels, and a decorative one is the wrong
              // width by definition.
              <svg
                viewBox="0 0 95 34"
                preserveAspectRatio="none"
                className="block h-8 w-full"
                aria-hidden
              >
                {[...modules].map((bit, i) =>
                  bit === "1" ? (
                    <rect
                      key={i}
                      x={i}
                      y={0}
                      width={1}
                      // The guards run below the rest, as the standard says —
                      // it is what makes a printed EAN look like one.
                      height={
                        EAN13_GUARDS.some(([a, b]) => i >= a && i < b) ? 34 : 28
                      }
                      fill="#000"
                    />
                  ) : null,
                )}
              </svg>
            ) : (
              // A code that is not an EAN-13 goes out as CODE128, whose bars
              // depend on the text. ⚠️ Not drawn rather than faked: a picture of
              // the wrong symbology is worse than none on the one screen that
              // is about what prints.
              <div className="h-8 w-full bg-[repeating-linear-gradient(90deg,#000_0_1px,transparent_1px_3px,#000_3px_5px,transparent_5px_8px)]" />
            )}
            {/* ⚠️ The digits under the bars are not decoration: a scanner that
                will not read a smudged label leaves a cashier with a queue, and
                the number is what lets them type it in. The printer prints
                them, so the card shows them. */}
            <div className="text-center text-[9px] tracking-[0.2em]">
              {code}
            </div>
          </div>
        )}

        {/* The blank paper fed before the cutter. ⚠️ Drawn because the setting
            is invisible otherwise, and getting it wrong means a tear-off
            through the barcode — which scans as nothing. */}
        {Array.from({ length: Math.min(feedLines, 10) }, (_, i) => (
          <div key={`feed-${i}`}>&nbsp;</div>
        ))}
      </div>

      {/* Where the paper is torn. */}
      <div
        className="border-t border-dashed border-black/40 text-center text-[8px] text-black/40"
        aria-hidden
      >
        ✂
      </div>
    </div>
  );
}
