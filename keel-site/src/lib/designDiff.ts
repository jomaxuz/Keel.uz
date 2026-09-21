// What a change to the design costs: a word in the page, or the page again.
//
// ⚠️ **This is the whole of the "why does it reload on every click" fix.** The
// editor's preview is an iframe of the customer's real site, and re-pointing it
// is a full navigation — a white flash, a jump to the top of the page, and
// another render for their container. Most edits do not need one: a colour, a
// size, a heading, a box that moved are all things a rendered page can simply be
// *told* (see PreviewBridge, which writes them into the DOM using the same
// `canvasStyle` module the server rendered with).
//
// What still needs a reload is anything that changes what the page is made of —
// a band added, removed or reordered, an element added, a variant changed, a
// binding that decides which dishes are shown. Those are structure, and the
// honest answer for structure is to render the page again.
//
// ⚠️ **The default is a reload, never a patch.** Every branch that cannot prove
// a change is drawable falls through to "reload": a missed reload shows the
// operator a page that does not match the document, which is the one failure
// this whole mechanism must not have. A missed patch only costs a flash.

import type { DesignSection } from "./api";

/** Settings the renderer prints as words, and marks with `data-keel-set`.
 *
 *  ⚠️ A key outside this list is not "text we forgot"; it is a setting that
 *  changes what the band *does* — how many dishes, which categories, whether
 *  there is a map — and none of those can be drawn without the server. */
const TEXT_KEYS = new Set([
  "heading", "subheading", "eyebrow", "body", "buttonLabel",
  "primaryLabel", "secondaryLabel",
]);

export type Change =
  | { kind: "none" }
  | { kind: "reload" }
  | {
      kind: "patch";
      /** Elements to redraw, as (band, index) pairs. */
      elements: { band: number; index: number }[];
      /** Bands whose own look or wording changed. */
      bands: number[];
    };

const same = (a: unknown, b: unknown) => JSON.stringify(a ?? null) === JSON.stringify(b ?? null);

/** Is this settings value one the page prints, and is it safe to write in? */
function drawableText(key: string, value: unknown): boolean {
  if (!TEXT_KEYS.has(key)) return false;
  // ⚠️ **An emptied heading is a reload.** Most of these fall back to something
  // when they are blank — the restaurant's own name, the built-in wording — and
  // the page is the only side that knows what to. Writing "" would leave a gap
  // where the fallback should be.
  if (typeof value === "string") return value.trim() !== "";
  if (value && typeof value === "object") {
    const bag = value as Record<string, string>;
    return Object.values(bag).some((v) => (v ?? "").trim() !== "");
  }
  return false;
}

export function diffDesign(prev: DesignSection[], next: DesignSection[]): Change {
  if (prev === next) return { kind: "none" };
  // A band added, removed or reordered: the page is a different page.
  if (prev.length !== next.length) return { kind: "reload" };

  const elements: { band: number; index: number }[] = [];
  const bands: number[] = [];

  for (let i = 0; i < next.length; i++) {
    const a = prev[i];
    const b = next[i];
    if (a === b) continue;
    if (!a || !b) return { kind: "reload" };
    // Structure, in order of how badly it would show: a different kind of band,
    // a different inner layout, a band hidden, a different data binding, a
    // different list of repeatable items.
    if (a.type !== b.type || a.variant !== b.variant || a.span !== b.span) {
      return { kind: "reload" };
    }
    if (!!a.hidden !== !!b.hidden) return { kind: "reload" };
    if (!same(a.binding, b.binding)) return { kind: "reload" };
    if (!same(a.blocks, b.blocks)) return { kind: "reload" };

    if (!same(a.style, b.style)) bands.push(i);

    if (!same(a.settings, b.settings)) {
      const keys = new Set([
        ...Object.keys(a.settings ?? {}),
        ...Object.keys(b.settings ?? {}),
      ]);
      for (const key of keys) {
        const before = (a.settings ?? {})[key];
        const after = (b.settings ?? {})[key];
        if (same(before, after)) continue;
        if (!drawableText(key, after)) return { kind: "reload" };
      }
      bands.push(i);
    }

    const ca = a.canvas;
    const cb = b.canvas;
    if (!ca !== !cb) return { kind: "reload" };
    if (ca && cb) {
      const ea = ca.elements ?? [];
      const eb = cb.elements ?? [];
      // An element added or removed: the indices below would no longer mean the
      // same thing, and the page has a box in it that does not exist yet.
      if (ea.length !== eb.length) return { kind: "reload" };
      if (
        ca.height !== cb.height ||
        ca.heightMobile !== cb.heightMobile ||
        ca.background !== cb.background ||
        ca.image !== cb.image ||
        ca.backgroundOpacity !== cb.backgroundOpacity
      ) {
        bands.push(i);
      }
      for (let j = 0; j < eb.length; j++) {
        if (same(ea[j], eb[j])) continue;
        // ⚠️ A different element type is a different node: a photograph cannot
        // be turned into a button by rewriting its classes.
        if (ea[j]?.type !== eb[j]?.type) return { kind: "reload" };
        // Shown or hidden is the presence of the node itself.
        if (!!ea[j]?.hidden !== !!eb[j]?.hidden) return { kind: "reload" };
        if (!!ea[j]?.hiddenMobile !== !!eb[j]?.hiddenMobile) return { kind: "reload" };
        // The list of photographs inside a carousel is markup, not a class.
        if (!same(ea[j]?.images, eb[j]?.images)) return { kind: "reload" };
        // ⚠️ The icon is an SVG path chosen by name; swapping it is markup too.
        if (ea[j]?.icon !== eb[j]?.icon) return { kind: "reload" };
        elements.push({ band: i, index: j });
      }
    }
  }

  if (elements.length === 0 && bands.length === 0) return { kind: "none" };
  return { kind: "patch", elements, bands: [...new Set(bands)] };
}
