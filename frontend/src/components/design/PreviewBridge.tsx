"use client";

// The only piece of the editor that lives on the restaurant's own site.
//
// ⚠️ **It reports geometry and nothing else.** The console draws the drag handles,
// does the arithmetic and saves the result; this component's whole job is to
// answer "where, in this rendered page, is band 2's element 3?".
//
// That split is deliberate, and it is the reason editing in the live preview is
// safe to ship:
//
//   • **No editor code reaches a guest.** This component is mounted only when the
//     page was rendered with a valid preview token — which only the console can
//     mint, and which lasts two hours. A visitor's page never contains it.
//   • **It cannot change anything.** It has no writes, no API calls and it ignores
//     every message it receives except a request to measure again. A bug here
//     cannot corrupt a design; at worst the handles sit in the wrong place.
//   • **The console never has to know our markup.** It matches boxes by the
//     `data-keel-band` / `data-keel-el` attributes, so the site's classes, its
//     layout and its blocks can all change without breaking the editor.
//
// Geometry is posted on load, on resize, on scroll and on request. Scroll matters
// most: the operator scrolls the preview to reach a band, and handles that stay
// where the band used to be are worse than no handles.

import { useEffect } from "react";

/** What the console receives. Percentages are not computed here — the console
 *  needs the pixel rects anyway to place handles, and one side doing all the
 *  arithmetic keeps the two from disagreeing about rounding. */
interface Report {
  type: "keel:geometry";
  scrollY: number;
  bands: { band: number; x: number; y: number; w: number; h: number }[];
  elements: { band: number; index: number; x: number; y: number; w: number; h: number }[];
}

export default function PreviewBridge() {
  useEffect(() => {
    // Not in an iframe: the preview link opened in a tab, which is a perfectly
    // normal thing to do with it. Nothing to report to.
    if (window.parent === window) return;

    function measure() {
      const bands: Report["bands"] = [];
      const elements: Report["elements"] = [];

      document.querySelectorAll<HTMLElement>("[data-keel-band]").forEach((node) => {
        // Both the desktop and the phone surface carry the same band index; only
        // the one actually laid out has a size, so the hidden one is skipped.
        const rect = node.getBoundingClientRect();
        if (rect.width === 0 || rect.height === 0) return;
        const band = Number(node.dataset.keelBand);
        bands.push({ band, x: rect.left, y: rect.top, w: rect.width, h: rect.height });

        node.querySelectorAll<HTMLElement>("[data-keel-el]").forEach((child) => {
          const r = child.getBoundingClientRect();
          elements.push({
            band,
            index: Number(child.dataset.keelEl),
            x: r.left,
            y: r.top,
            w: r.width,
            h: r.height,
          });
        });
      });

      const report: Report = {
        type: "keel:geometry",
        scrollY: window.scrollY,
        bands,
        elements,
      };
      // ⚠️ `"*"` as the target, and it is safe **because of what is in the
      // message**: four numbers per box and no identifier, no token and nothing
      // about the restaurant. The direction that matters is the other one — the
      // console verifies the origin of what it receives before acting on it.
      window.parent.postMessage(report, "*");
    }

    // Measured after paint, and again shortly after: web fonts and images change
    // the height of a band, and a measurement taken before them is wrong in
    // exactly the way that makes handles look broken.
    const first = window.setTimeout(measure, 60);
    const second = window.setTimeout(measure, 700);

    function onMessage(e: MessageEvent) {
      if ((e.data as { type?: string })?.type === "keel:measure") measure();
    }

    window.addEventListener("resize", measure);
    window.addEventListener("scroll", measure, { passive: true });
    window.addEventListener("message", onMessage);
    return () => {
      window.clearTimeout(first);
      window.clearTimeout(second);
      window.removeEventListener("resize", measure);
      window.removeEventListener("scroll", measure);
      window.removeEventListener("message", onMessage);
    };
  }, []);

  return null;
}
