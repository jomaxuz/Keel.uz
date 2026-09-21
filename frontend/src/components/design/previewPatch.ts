// Writing a change into a page that is already rendered.
//
// ⚠️ **This is the half of the editor that stops it reloading somebody's shop.**
// The console cannot re-render the tenant's site — only the tenant's server can
// — so an edit is either drawn in here or paid for with a full navigation of a
// live site: a white flash, a jump to the top, another render for their
// container. `lib/designDiff.ts` in the console decides which; this file is what
// "drawn in" means.
//
// Three rules hold it together:
//
//   • **The classes come from `canvasStyle`**, the same module the server
//     rendered this page with. A second copy of those maps here would be the
//     copy that drifts, and drift shows up as a preview that lies.
//   • **Nodes are found by mark, never by their words.** `data-keel-el`,
//     `data-keel-band`, `data-keel-set`, `data-keel-text` — the same sentence
//     can appear twice on a page, and the one that is not the heading is the one
//     a text match would rewrite.
//   • **A patch is a picture of the document, never the document.** The console
//     saves the draft either way, and the next load renders from what was saved.
//     If the two disagree, the reload wins.
//
// It takes a `root` rather than reaching for `document`, which is what lets it
// be tested against a rendered band instead of against a real site.

import { imageUrl } from "@/lib/api";
import type { DesignElement, DesignSection, LocalizedText } from "@/lib/types";
import {
  TONE_CLASS,
  TYPE_CLASS,
  bandClass,
  bandWidthVars,
  elementClass,
  elementOpacity,
  imageFit,
} from "./canvasStyle";

/** Every class name any tone can contribute, one at a time. */
const TONE_TOKENS = new Set(
  Object.values(TONE_CLASS)
    .flatMap((v) => v.split(" "))
    .filter(Boolean),
);

/** What the console may ask this page to redraw in place.
 *
 *  ⚠️ **The element, whole, rather than the fields that changed.** The console
 *  does not know which of a dozen style keys produced which class name — only
 *  `canvasStyle` knows that, and it is the same module the server rendered with.
 *  Handing over the element and recomputing here is what keeps the patched page
 *  and the reloaded page the same page. */
export interface ElementPatch {
  band: number;
  index: number;
  el: DesignElement;
}

/** A band's own look: its wrapper, and — when it is a drawn band — its height,
 *  its background and the photograph behind it. */
export interface BandPatch {
  band: number;
  section: DesignSection;
}

/** Every rendering of one element: the desktop surface, the phone surface,
 *  and the stacked fallback. ⚠️ All of them, because only one is laid out at
 *  a time and the console does not know which — patching just the first
 *  would leave the phone preview showing the previous word. */
function nodesOf(root: ParentNode, band: number, index: number): HTMLElement[] {
  const out: HTMLElement[] = [];
  document
    .querySelectorAll<HTMLElement>(
      `[data-keel-band="${band}"], [data-keel-flow="${band}"]`,
    )
    .forEach((wrap) => {
      wrap
        .querySelectorAll<HTMLElement>(`[data-keel-el="${index}"]`)
        .forEach((n) => out.push(n));
    });
  return out;
}

/** Writes the words in.
 *
 *  ⚠️ **Into the marked node, never into the element.** An element is a box
 *  with markup inside it — a button holds a link, a quote holds typographic
 *  quotes, a list holds one row per line — and `textContent = value` on the
 *  outer box would delete that markup and replace a working button with a
 *  bare word. `data-keel-text` says which node holds the words and in what
 *  shape (CanvasBlock puts it there), so the shapes stay where they are. */
function writeText(node: HTMLElement, value: string) {
  const holder = node.matches("[data-keel-text]")
    ? node
    : node.querySelector<HTMLElement>("[data-keel-text]");
  if (!holder) return;
  const kind = holder.dataset.keelText || "plain";
  if (kind === "quote") {
    holder.textContent = `\u201c${value}\u201d`;
    return;
  }
  if (kind === "lines") {
    const lines = value.split("\n").map((l) => l.trim()).filter(Boolean);
    const template = holder.querySelector("li");
    const rows = lines.map((line) => {
      const li = template
        ? (template.cloneNode(true) as HTMLElement)
        : document.createElement("li");
      const spans = li.querySelectorAll("span");
      // The bullet is the first span and the words are the last; a row
      // cloned and then filled whole would lose the dot.
      if (spans.length > 1) spans[spans.length - 1].textContent = line;
      else li.textContent = line;
      return li;
    });
    holder.replaceChildren(...rows);
    return;
  }
  holder.textContent = value;
}

/** The words in the language this page was rendered in.
 *
 *  ⚠️ Read off the document rather than assumed: the preview opens on the
 *  site's own default, and an operator may be looking at the Russian page —
 *  writing the Uzbek string into it would "fix" the wrong sentence. */
function localOf(v?: LocalizedText | null): string {
  if (!v) return "";
  const lang = (document.documentElement.lang || "uz") as keyof LocalizedText;
  return (v[lang] || v.uz || "") as string;
}

/** The second line some elements have: a quote's attribution, a stat's
 *  label. Marked in the renderer because the dash in front of an
 *  attribution belongs to the design, not to what was typed. */
function writeSub(node: HTMLElement, value: string) {
  const holder = node.querySelector<HTMLElement>("[data-keel-sub]");
  if (!holder) return;
  holder.textContent =
    holder.dataset.keelSub === "dash" ? `\u2014 ${value}` : value;
}

/** Writes one element back into the page: where it sits, what it looks
 *  like, and what it says.
 *
 *  ⚠️ **Everything visual goes through `canvasStyle`**, the module the
 *  server rendered this page with. What cannot be redrawn here is stated
 *  rather than half-done: an element's *type* never changes under a patch
 *  (the console reloads for that), and the working widgets — the menu, the
 *  cart — keep whatever they rendered, because their insides are the site's
 *  own code and not a design. */
export function applyElement(root: ParentNode, item: ElementPatch) {
  const el = item.el;
  if (!el || typeof el !== "object") return;
  for (const node of nodesOf(root, item.band, item.index)) {
    const absolute = getComputedStyle(node).position === "absolute";
    // ⚠️ **Which layout this node is.** The same element is rendered twice —
    // once on the desktop surface and once on the phone one — and the phone
    // surface uses the element's own phone box when it has one. Writing the
    // desktop numbers into both is how a drag on one moves the wrong thing
    // on the other.
    const phone = !!node.closest("[data-keel-band]")?.classList.contains("lg:hidden");
    const box = absolute ? (phone && el.mobile ? el.mobile : el.box) : null;
    // ⚠️ Only where the site placed the element itself. On a phone with no
    // drawn layout the band is flow, and writing `left: 40%` onto an
    // element in a column moves nothing and confuses everything.
    if (box) {
      node.style.left = `${box.x}%`;
      node.style.top = `${box.y}%`;
      node.style.width = `${box.w}%`;
      node.style.height = `${box.h}%`;
      node.style.zIndex = String(box.z ?? 0);
    }
    const opacity = elementOpacity(el);
    node.style.opacity = opacity == null ? "" : String(opacity);
    // ⚠️ The data attributes are set from the DOM, not from the patch: this
    // node is already the right element, and rebuilding its marks from a
    // message would be a way to lose them.
    node.className = `${elementClass(el)} ${TYPE_CLASS[el.type] ?? ""}`.trim();

    if (el.text) writeText(node, localOf(el.text));
    if (el.subtext) writeSub(node, localOf(el.subtext));

    const img = node.querySelector("img");
    if (img) {
      if (el.image) {
        const next = imageUrl(el.image, 1200) ?? "";
        if (next && img.getAttribute("src") !== next) img.setAttribute("src", next);
      }
      // ⚠️ Fills or fits — the class lives on the picture, not on the box the
      // element draws, so the patch has to reach inside.
      img.classList.remove("object-cover", "object-contain");
      img.classList.add(imageFit(el));
    }
    const link = node.matches("a") ? node : node.querySelector("a");
    if (link && el.link) link.setAttribute("href", el.link);

    if (el.type === "rating") {
      const filled = Math.max(0, Math.min(5, el.value ?? 5));
      node.querySelectorAll("svg").forEach((star, i) => {
        star.setAttribute("fill", i < filled ? "currentColor" : "none");
      });
    }
  }
}

/** A band's own look. The wrapper classes for any band; the height, the
 *  tone and the photograph for a drawn one. */
export function applyBand(root: ParentNode, item: BandPatch) {
  const section = item.section;
  if (!section || typeof section !== "object") return;
  document
    .querySelectorAll<HTMLElement>(`[data-keel-band="${item.band}"]`)
    .forEach((node) => {
      const canvas = section.canvas;
      if (!canvas) {
        // A fixed band: its wrapper carries the whole of its look.
        node.className = bandClass(section);
        const vars = bandWidthVars(section.style?.width) as
          | Record<string, string>
          | undefined;
        node.style.removeProperty("--band-max");
        node.style.removeProperty("--band-pad");
        for (const [k, v] of Object.entries(vars ?? {})) node.style.setProperty(k, v);
        return;
      }
      // A drawn band. ⚠️ The desktop surface and the phone surface are two
      // nodes with the same index and different heights, and only the one
      // laid out has a size — so the height is written on whichever is
      // being looked at rather than on "the band".
      const phone = node.classList.contains("lg:hidden");
      const vh = phone ? canvas.heightMobile || canvas.height : canvas.height;
      if (vh) node.style.height = `${vh}vh`;
      // ⚠️ Token by token, because a tone is more than one class:
      // `charcoal` is "bg-charcoal text-white", and filtering by the whole
      // string would leave every old tone in place and stack them.
      const tone = TONE_CLASS[canvas.background ?? ""] ?? "";
      node.className = `${node.className
        .split(" ")
        .filter((c) => c !== "" && !TONE_TOKENS.has(c))
        .join(" ")} ${tone}`.trim();
      const back = node.querySelector<HTMLElement>(":scope > div > img")?.parentElement;
      if (back) {
        const o = canvas.backgroundOpacity;
        back.style.opacity = o != null && o < 100 ? String(o / 100) : "";
        const img = back.querySelector("img");
        const next = canvas.image ? (imageUrl(canvas.image, 1200) ?? "") : "";
        if (img && next && img.getAttribute("src") !== next) {
          img.setAttribute("src", next);
        }
      }
    });
}

/** A settings value the band prints as words. ⚠️ Found by the key that
 *  produced it (`data-keel-set`), never by matching the old text: the same
 *  sentence can appear twice on a page, and the one that is not the heading
 *  is the one that would be rewritten. */
export function applySettings(root: ParentNode, item: BandPatch) {
  const bag = (item.section?.settings ?? {}) as Record<string, unknown>;
  document
    .querySelectorAll<HTMLElement>(`[data-keel-band="${item.band}"]`)
    .forEach((band) => {
      band.querySelectorAll<HTMLElement>("[data-keel-set]").forEach((node) => {
        const key = node.dataset.keelSet;
        if (!key || !(key in bag)) return;
        const value = bag[key];
        const word =
          typeof value === "string" ? value : localOf(value as LocalizedText);
        // ⚠️ An emptied setting is not patched away. Most of these fall back
        // to the restaurant's own name or to the built-in wording when they
        // are blank, and this side does not know what that fallback is — the
        // console reloads instead, which does.
        if (!word) return;
        node.textContent = word;
      });
    });
}
