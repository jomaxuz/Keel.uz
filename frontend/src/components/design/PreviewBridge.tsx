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
//
// ⚠️ **It also applies a patch, and that is what stopped the editor reloading on
// every keystroke.** Until now the only way the console could show a changed word
// was to save the draft and re-point the iframe — a full navigation of somebody's
// real site, with the white flash and the jump back to the top that go with it,
// after every letter typed and every box dragged. So a `keel:patch` carries the
// boxes and the words that changed and this file writes them straight into the
// DOM: the position and the text are exactly the two things a page can be told
// without re-rendering it.
//
// ⚠️ **A patch is a preview of a change, never the change itself.** The console
// still saves the draft, and the saved document is what the page shows the next
// time it loads. If the two ever disagree the reload wins, which is the right way
// round: the document is the truth and this is a picture of it.



/** What the console receives. Percentages are not computed here — the console
 *  needs the pixel rects anyway to place handles, and one side doing all the
 *  arithmetic keeps the two from disagreeing about rounding. */
interface Report {
  type: "keel:geometry";
  scrollY: number;
  bands: { band: number; x: number; y: number; w: number; h: number }[];
  elements: { band: number; index: number; x: number; y: number; w: number; h: number }[];
}

/** Starts reporting. Returns the teardown.
 *
 *  ⚠️ A plain function rather than a component, so the gate below can `import()`
 *  it — which is what keeps this file out of every guest's bundle. As a statically
 *  imported component its code shipped to everybody and merely never ran, which is
 *  a weaker claim than the one worth making: the same reasoning as the Telegram SDK,
 *  loaded only inside Telegram. */
export function startBridge(): () => void {
  {
    // Not in an iframe: the preview link opened in a tab, which is a perfectly
    // normal thing to do with it. Nothing to report to.
    if (window.parent === window) return () => {};

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

    // ⚠️ **Edit mode turns the page into a selection surface**, which is what a
    // theme editor feels like: you click the thing you want to change, wherever it
    // is, and its settings open. It also has to *stop the click*, because the page
    // is a working site — a link followed mid-edit takes the operator to the menu
    // and loses the pane they were in.
    let editing = false;

    function onClick(e: MouseEvent) {
      if (!editing) return;
      const el = (e.target as HTMLElement | null)?.closest<HTMLElement>("[data-keel-el]");
      const bandNode = (e.target as HTMLElement | null)?.closest<HTMLElement>("[data-keel-band]");
      if (!bandNode) return;
      // Captured and cancelled: on a real page every second element is a link.
      e.preventDefault();
      e.stopPropagation();
      window.parent.postMessage(
        {
          type: "keel:select",
          band: Number(bandNode.dataset.keelBand),
          index: el ? Number(el.dataset.keelEl) : null,
        },
        "*",
      );
    }

    /** Every rendering of one element: the desktop surface, the phone surface,
     *  and the stacked fallback. ⚠️ All of them, because only one is laid out at
     *  a time and the console does not know which — patching just the first
     *  would leave the phone preview showing the previous word. */
    function nodesOf(band: number, index: number): HTMLElement[] {
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

    interface Patch {
      band: number;
      index: number;
      box?: { x: number; y: number; w: number; h: number; z?: number };
      text?: string;
    }

    function applyPatch(items: Patch[]) {
      for (const item of items) {
        if (typeof item?.band !== "number" || typeof item?.index !== "number") continue;
        for (const node of nodesOf(item.band, item.index)) {
          if (item.box) {
            // ⚠️ Only where the site placed the element itself. On a phone with
            // no drawn layout the band is flow, and writing `left: 40%` onto a
            // element in a column moves nothing and confuses everything.
            if (getComputedStyle(node).position === "absolute") {
              node.style.left = `${item.box.x}%`;
              node.style.top = `${item.box.y}%`;
              node.style.width = `${item.box.w}%`;
              node.style.height = `${item.box.h}%`;
              if (item.box.z != null) node.style.zIndex = String(item.box.z);
            }
          }
          if (typeof item.text === "string") writeText(node, item.text);
        }
      }
      // The boxes moved, so the reported geometry is stale — and stale geometry
      // is handles sitting where the element is not.
      measure();
    }

    function onMessage(e: MessageEvent) {
      // ⚠️ **Only the window that framed this page.** Nothing here can write a
      // design — the worst a forged message could do is move a box or a word in
      // the operator's own preview — but a patch now changes what is on screen,
      // and "it cannot do damage today" is not a property that survives edits.
      // The console is the parent frame; anything else is not talking to us.
      if (e.source !== window.parent) return;
      const data = e.data as {
        type?: string;
        edit?: boolean;
        y?: number;
        items?: Patch[];
      };
      if (data?.type === "keel:measure") measure();
      if (data?.type === "keel:mode") editing = !!data.edit;
      if (data?.type === "keel:patch" && Array.isArray(data.items)) {
        applyPatch(data.items);
      }
      // ⚠️ Put back where the operator was. A reload is sometimes unavoidable —
      // a new band, a changed variant — and the part of it that actually hurts
      // is not the wait, it is landing back at the top of the page having lost
      // the band being worked on.
      if (data?.type === "keel:scrollto" && typeof data.y === "number") {
        window.scrollTo(0, data.y);
        measure();
      }
    }

    window.addEventListener("resize", measure);
    window.addEventListener("scroll", measure, { passive: true });
    window.addEventListener("message", onMessage);
    // Capture phase, so the click is intercepted before the site's own handlers
    // and before a link does its default.
    document.addEventListener("click", onClick, true);
    return () => {
      window.clearTimeout(first);
      window.clearTimeout(second);
      window.removeEventListener("resize", measure);
      window.removeEventListener("scroll", measure);
      window.removeEventListener("message", onMessage);
      document.removeEventListener("click", onClick, true);
    };
  }
}
