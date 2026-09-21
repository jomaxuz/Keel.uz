import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import CanvasBlock from "./CanvasBlock";
import { LangProvider } from "@/lib/i18n/client";
import type { DesignElement } from "@/lib/types";

// ⚠️ **What this protects is a silent failure in the console, not a style rule.**
//
// The layout editor no longer reloads the tenant's page when a word changes: it
// posts the new word and the bridge writes it into the node marked
// `data-keel-text` (see PreviewBridge). So an element that renders words without
// that marker is an element whose text **cannot be typed in place** — the caret
// appears, the letters arrive, and nothing on the page changes until something
// else forces a reload. Nothing errors, no test fails, and the editor simply
// feels broken for that one element type.
//
// The list below is the console's `TEXTUAL` set (page.tsx in keel-site), which
// is the set of elements a double-click opens a typing box over. The two cannot
// be bound by the compiler — they are different applications — so they are
// bound here.
const TEXTUAL: { type: string; el: DesignElement }[] = [
  {
    type: "text",
    el: { type: "text", box: { x: 0, y: 0, w: 50, h: 10 }, text: { uz: "Salom", ru: "", en: "" } },
  },
  {
    type: "text with a link",
    el: {
      type: "text",
      box: { x: 0, y: 0, w: 50, h: 10 },
      text: { uz: "Ayollar", ru: "", en: "" },
      link: "/menu?cat=ayollar",
    },
  },
  {
    type: "button",
    el: {
      type: "button",
      box: { x: 0, y: 0, w: 20, h: 8 },
      text: { uz: "Buyurtma", ru: "", en: "" },
      link: "/menu",
    },
  },
  {
    type: "badge",
    el: { type: "badge", box: { x: 0, y: 0, w: 10, h: 5 }, text: { uz: "Yangi", ru: "", en: "" } },
  },
  {
    type: "quote",
    el: {
      type: "quote",
      box: { x: 0, y: 0, w: 40, h: 20 },
      text: { uz: "Ajoyib", ru: "", en: "" },
      subtext: { uz: "Mijoz", ru: "", en: "" },
    },
  },
  {
    type: "stat",
    el: {
      type: "stat",
      box: { x: 0, y: 0, w: 20, h: 15 },
      text: { uz: "12", ru: "", en: "" },
      subtext: { uz: "yil", ru: "", en: "" },
    },
  },
  {
    type: "list",
    el: {
      type: "list",
      box: { x: 0, y: 0, w: 30, h: 25 },
      text: { uz: "Bir\nIkki", ru: "", en: "" },
    },
  },
];

describe("a freely placed element the console can type into", () => {
  for (const { type, el } of TEXTUAL) {
    it(`marks where the words live: ${type}`, () => {
      const { container } = render(
        // ⚠️ The provider, because a linked line renders `LocaleLink` — which is
        // the element the shop templates are built from and the one whose marker
        // was missing before this test existed.
        <LangProvider initial="uz">
          <CanvasBlock canvas={{ height: 60, elements: [el] }} bandIndex={0} lang="uz" />
        </LangProvider>,
      );
      const node = container.querySelector('[data-keel-el="0"]');
      expect(node, "the element publishes its index for the editor").not.toBeNull();
      const holder = node!.matches("[data-keel-text]")
        ? node!
        : node!.querySelector("[data-keel-text]");
      expect(holder, `${type} renders words with nowhere to write them`).not.toBeNull();
      expect(["plain", "quote", "lines"]).toContain(
        holder!.getAttribute("data-keel-text"),
      );
    });
  }

  // ⚠️ The quote is the one element whose words are not its whole text content:
  // the typographic quotes belong to the design, not to what was typed. The
  // bridge puts them back, which it can only do because the marker says so.
  it("says which shape the words are in", () => {
    const { container } = render(
      <CanvasBlock
        canvas={{
          height: 60,
          elements: [
            { type: "quote", box: { x: 0, y: 0, w: 40, h: 20 }, text: { uz: "Ajoyib", ru: "", en: "" } },
            { type: "list", box: { x: 0, y: 0, w: 30, h: 20 }, text: { uz: "Bir\nIkki", ru: "", en: "" } },
          ],
        }}
        bandIndex={0}
        lang="uz"
      />,
    );
    expect(
      container.querySelector('[data-keel-el="0"] [data-keel-text]')?.getAttribute("data-keel-text"),
    ).toBe("quote");
    expect(
      container.querySelector('[data-keel-el="1"]')?.getAttribute("data-keel-text"),
    ).toBe("lines");
  });

  // The stacked phone surface is a second rendering of the same elements, and a
  // word changed in the console has to change there too. It is marked with
  // `data-keel-flow` rather than `data-keel-band` on purpose: it is flow, so it
  // must never be reported as a draggable band.
  it("marks the stacked phone surface without claiming it is draggable", () => {
    const { container } = render(
      <CanvasBlock
        canvas={{
          height: 60,
          elements: [{ type: "text", box: { x: 0, y: 0, w: 50, h: 10 }, text: { uz: "Salom", ru: "", en: "" } }],
        }}
        bandIndex={3}
        lang="uz"
      />,
    );
    expect(container.querySelectorAll('[data-keel-band="3"]').length).toBe(1);
    expect(container.querySelector('[data-keel-flow="3"]')).not.toBeNull();
  });
});
