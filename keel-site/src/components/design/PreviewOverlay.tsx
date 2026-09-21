"use client";

// Editing on the live site: handles drawn over the iframe, arithmetic done here.
//
// ⚠️ **Why an overlay rather than editing inside the page.** Dragging inside the
// iframe would mean shipping the editor into every tenant's production bundle and
// letting a foreign document run code that mutates a design. Instead the site
// reports only geometry (see PreviewBridge — four numbers per box, no identifiers,
// no token) and everything that can change a design stays in the console, behind
// the console's own session.
//
// So the split is: the site says *where* the boxes are, this file decides *what
// the drag means*, and the editor's existing state is the single writer.
//
// Three things make it usable rather than a demo:
//
//   • **Percent, from the band's own rect.** A drag of 40 pixels means different
//     things in a 1280px band and a 390px one, and the stored value is percent —
//     so the conversion happens against the band the element is actually in.
//   • **The handles move optimistically, and the iframe catches up.** A full page
//     render per pixel would make the editor unusable and the tenant's container
//     busy; a reload on drag end, debounced, is the honest compromise. The box the
//     operator is dragging is the console's own div, so it never lags.
//   • **Only what the site reported is editable.** A band the site did not render
//     — hidden, or below a `lg` breakpoint — has no handles, rather than handles
//     over empty space that move something invisible.

import { useCallback, useEffect, useRef, useState } from "react";
import type { DesignBox } from "@/lib/api";

/** What is being dragged in from the palette, if anything. ⚠️ The console owns
 *  this rather than the browser's own drag data, because the drop has to be
 *  decided *while* the pointer is over the page — the highlight under the
 *  cursor is the whole point — and `dataTransfer` cannot be read during
 *  `dragover`. */
export interface Incoming {
  kind: "band" | "element";
  type: string;
}

interface Rect {
  x: number;
  y: number;
  w: number;
  h: number;
}
interface Geometry {
  bands: (Rect & { band: number })[];
  elements: (Rect & { band: number; index: number })[];
}

/** ⚠️ **Eight, not four.** Corners change width and height together, which is
 *  the one thing you almost never want on a band of text or a button: making a
 *  button wider should not make it taller. Every direct-manipulation editor
 *  people already know — Figma, Canva, Shopify's — puts a handle on each edge,
 *  and its absence reads as "this editor cannot do that" rather than as a
 *  missing convenience. */
type Handle = "move" | "n" | "s" | "e" | "w" | "ne" | "nw" | "se" | "sw";

const EDGE: Handle[] = ["n", "s", "e", "w"];
const CORNER: Handle[] = ["nw", "ne", "sw", "se"];

const CURSOR: Record<string, string> = {
  n: "ns-resize", s: "ns-resize", e: "ew-resize", w: "ew-resize",
  nw: "nwse-resize", se: "nwse-resize", ne: "nesw-resize", sw: "nesw-resize",
};

export default function PreviewOverlay({
  frame,
  zoom,
  activeBand,
  selected,
  onSelect,
  onPick,
  onBox,
  onCommit,
  boxOf,
  textOf,
  onText,
  incoming,
  onDropElement,
  onDropBand,
  canDrawIn,
}: {
  frame: React.RefObject<HTMLIFrameElement | null>;
  /** The iframe's CSS scale, so a pixel on screen maps back to a page pixel. */
  zoom: number;
  /** Only this band's elements are editable — the same band the left column and
   *  the inspector are pointed at, so the three cannot disagree. */
  activeBand: number;
  selected: number | null;
  onSelect: (index: number | null) => void;
  /** A click inside the page: which band, and which element in it (or none). */
  onPick: (band: number, index: number | null) => void;
  onBox: (index: number, patch: Partial<DesignBox>) => void;
  /** Called when a drag finishes: the moment to save and let the page re-render. */
  onCommit: () => void;
  /** The element's stored box, needed because a drag is applied to the value in
   *  the document rather than to what happens to be on screen. */
  boxOf: (index: number) => DesignBox | null;
  /** What this element says, or null when it is not a thing with words in it.
   *  ⚠️ The source language: the preview renders the site in Uzbek, so the box
   *  that opens over a word has to hold the word that is on screen. */
  textOf: (index: number) => string | null;
  onText: (index: number, value: string) => void;
  /** A card being dragged off the palette, or null. */
  incoming?: Incoming | null;
  /** Dropped an element: which band, and where inside it, in percent. */
  onDropElement?: (band: number, x: number, y: number) => void;
  /** Dropped a section: the index it should be inserted at. */
  onDropBand?: (index: number) => void;
  /** Whether a band can hold freely placed elements. ⚠️ Asked rather than
   *  assumed: dropping a headline into a menu grid has nowhere to go, and an
   *  editor that accepts the drop and then does nothing is worse than one that
   *  says no while the card is still in the air. */
  canDrawIn?: (band: number) => boolean;
}) {
  const [geo, setGeo] = useState<Geometry>({ bands: [], elements: [] });
  // Which element is being typed into, and what has been typed so far.
  //
  // ⚠️ **Held here rather than written through on every keystroke.** Each
  // change to the design marks the draft dirty and schedules a save, and a save
  // per character is a page render per character on somebody's live site.
  // Committed on Enter, on blur, and never on Escape.
  const [editing, setEditing] = useState<{ index: number; value: string } | null>(null);
  // The size readout during a drag: a number, because "a bit wider" is not a
  // thing you can repeat on the next band.
  const [readout, setReadout] = useState<string>("");
  // Where the card in the air would land: the band under the cursor, and for an
  // element the point inside it. Drawn, because "it lands where you drop it" is
  // a promise that has to be visible before the mouse button comes up.
  const [hover, setHover] = useState<{
    band: number;
    x: number;
    y: number;
    ok: boolean;
    after: boolean;
  } | null>(null);
  const drag = useRef<{
    index: number;
    handle: Handle;
    startX: number;
    startY: number;
    box: DesignBox;
    band: Rect;
  } | null>(null);

  // Geometry arrives from the site. ⚠️ The origin is checked before anything is
  // believed: this window is open to messages from anywhere, and a page in another
  // tab could otherwise move the handles around. Nothing here writes a design, so
  // the worst a forged message could do is misplace them — checked anyway, because
  // "it cannot do damage today" is not a property that survives edits.
  useEffect(() => {
    function onMessage(e: MessageEvent) {
      const src = frame.current?.src ?? "";
      if (!src) return;
      let expected = "";
      try {
        expected = new URL(src).origin;
      } catch {
        return;
      }
      if (e.origin !== expected) return;
      const data = e.data as { type?: string; band?: number; index?: number | null } & Geometry;
      if (data?.type === "keel:geometry") {
        setGeo({ bands: data.bands ?? [], elements: data.elements ?? [] });
        return;
      }
      // ⚠️ A click on the real page. This is the gesture that makes an editor feel
      // like one — you click the thing you want to change, wherever it is on the
      // page, and its settings open. Any band, not only the selected one, so the
      // page itself becomes the navigation.
      if (data?.type === "keel:select" && typeof data.band === "number") {
        onPick(data.band, data.index ?? null);
      }
    }
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, [frame, onPick]);

  /** Asks the site to measure again — after a reload, or when the pane resizes.
   *  The mode goes with it: the bridge only intercepts clicks while editing, so a
   *  reloaded page has to be told again. */
  const remeasure = useCallback(() => {
    const win = frame.current?.contentWindow;
    win?.postMessage({ type: "keel:measure" }, "*");
    win?.postMessage({ type: "keel:mode", edit: true }, "*");
  }, [frame]);

  useEffect(() => {
    const timer = window.setInterval(remeasure, 1500);
    window.addEventListener("resize", remeasure);
    return () => {
      window.clearInterval(timer);
      window.removeEventListener("resize", remeasure);
    };
  }, [remeasure]);

  useEffect(() => {
    function move(e: PointerEvent) {
      const d = drag.current;
      if (!d) return;
      // Screen pixels back into page pixels, then into percent of the band. The
      // zoom division is the step that is easy to forget and impossible to miss
      // once wrong: the element moves at the wrong speed under the cursor.
      const dx = ((e.clientX - d.startX) / zoom / d.band.w) * 100;
      const dy = ((e.clientY - d.startY) / zoom / d.band.h) * 100;
      const b = { ...d.box };
      const round = (v: number) => Math.round(v);

      if (d.handle === "move") {
        b.x = round(d.box.x + dx);
        b.y = round(d.box.y + dy);
      } else {
        if (d.handle.includes("e")) b.w = Math.max(2, round(d.box.w + dx));
        if (d.handle.includes("w")) {
          b.w = Math.max(2, round(d.box.w - dx));
          b.x = round(d.box.x + dx);
        }
        if (d.handle.includes("s")) b.h = Math.max(2, round(d.box.h + dy));
        if (d.handle.includes("n")) {
          b.h = Math.max(2, round(d.box.h - dy));
          b.y = round(d.box.y + dy);
        }
      }
      onBox(d.index, b);
      setReadout(
        d.handle === "move" ? `${b.x} · ${b.y} %` : `${b.w} × ${b.h} %`,
      );
    }
    function up() {
      if (!drag.current) return;
      drag.current = null;
      setReadout("");
      // Save and let the real page redraw. ⚠️ On drag **end** only: a save per
      // pointer move would be a page render per pixel, on somebody's live site.
      onCommit();
    }
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
    window.addEventListener("pointercancel", up);
    return () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
      window.removeEventListener("pointercancel", up);
    };
  }, [zoom, onBox, onCommit]);

  const frameRect = frame.current?.getBoundingClientRect();
  if (!frameRect) return null;

  const band = geo.bands.find((b) => b.band === activeBand);
  const mine = geo.elements.filter((el) => el.band === activeBand);

  /** The rect the site reported for a band, in page pixels. */
  function hoverRect(index: number) {
    return geo.bands.find((b) => b.band === index) ?? null;
  }

  /** Bands are reported in the framed page's own viewport coordinates, and the
   *  overlay is positioned from the frame's top-left — so the two agree only
   *  once the frame's origin is subtracted. It is (0,0) today and named anyway:
   *  the day the preview gains a gutter, a silently misplaced highlight is the
   *  hardest kind of bug to see. */
  const bandOrigin = { x: 0, y: 0 };

  /** Which band a screen point is over, and where inside it. */
  function locate(clientX: number, clientY: number) {
    if (!frameRect) return null;
    const px = (clientX - frameRect.left) / zoom + bandOrigin.x;
    const py = (clientY - frameRect.top) / zoom + bandOrigin.y;
    // ⚠️ Last match wins: bands do not overlap, but a `canvas` band with a
    // bleeding element can extend over the one under it, and the operator means
    // the one they can see.
    let found: (Rect & { band: number }) | null = null;
    for (const b of geo.bands) {
      if (px >= b.x && px <= b.x + b.w && py >= b.y && py <= b.y + b.h) found = b;
    }
    if (!found) return null;
    return {
      band: found.band,
      x: Math.round(((px - found.x) / found.w) * 100),
      y: Math.round(((py - found.y) / found.h) * 100),
      ok: canDrawIn ? canDrawIn(found.band) : true,
      after: py > found.y + found.h / 2,
    };
  }

  function start(index: number, handle: Handle) {
    return (e: React.PointerEvent) => {
      e.preventDefault();
      e.stopPropagation();
      const box = boxOf(index);
      if (!box || !band) return;
      onSelect(index);
      drag.current = {
        index,
        handle,
        startX: e.clientX,
        startY: e.clientY,
        box: { ...box },
        band,
      };
    };
  }

  return (
    <>
      {/* Positioned in the viewport, over the iframe. `fixed` rather than absolute
          because the reported rects are viewport-relative — the page inside the
          iframe has already accounted for its own scrolling. */}
      {mine.map((el) => {
        const isSel = selected === el.index;
        const words = textOf(el.index);
        const typing = editing?.index === el.index;
        return (
          <div
            key={`${el.band}-${el.index}`}
            onPointerDown={typing ? undefined : start(el.index, "move")}
            // ⚠️ **Double-click to type, the gesture everybody already knows.**
            // Single click selects — it has to, because selecting is how you
            // reach an element's settings and most elements have no words at
            // all. Double-click is what every canvas editor uses for "edit the
            // words in place", and it is the one gesture that makes this feel
            // like an editor rather than a form with a preview beside it.
            onDoubleClick={(e) => {
              if (words == null) return;
              e.preventDefault();
              e.stopPropagation();
              onSelect(el.index);
              setEditing({ index: el.index, value: words });
            }}
            style={{
              position: "fixed",
              left: frameRect.left + el.x * zoom,
              top: frameRect.top + el.y * zoom,
              width: el.w * zoom,
              height: el.h * zoom,
              zIndex: typing ? 70 : 60,
            }}
            className={
              typing
                ? ""
                : `${words != null ? "cursor-text" : "cursor-move"} ${
                    isSel
                      ? "ring-2 ring-signal-500"
                      : "ring-1 ring-signal-500/30 hover:ring-signal-500/70"
                  }`
            }
          >
            {typing ? (
              // ⚠️ **A box exactly over the words, not a field in the sidebar.**
              // The iframe is another origin, so its fonts cannot be read and
              // this cannot literally be the page's own text node — but it sits
              // where the words sit, at the size they occupy, and that is what
              // "edit it where it is" has to mean from outside the frame. The
              // real typography comes back the moment it is committed and the
              // page redraws.
              <textarea
                autoFocus
                value={editing.value}
                onChange={(e) => setEditing({ index: el.index, value: e.target.value })}
                onBlur={() => {
                  onText(el.index, editing.value);
                  setEditing(null);
                  onCommit();
                }}
                onKeyDown={(e) => {
                  // Escape abandons. ⚠️ It must not also commit: the one thing
                  // an operator expects from Escape is that the words go back
                  // to what they were.
                  if (e.key === "Escape") {
                    e.preventDefault();
                    setEditing(null);
                    return;
                  }
                  // Enter commits, Shift+Enter is a line break — a headline
                  // typed on two lines is a real thing designers do, and the
                  // renderer already keeps the break (`whitespace-pre-line`).
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault();
                    (e.target as HTMLTextAreaElement).blur();
                  }
                }}
                className="h-full w-full resize-none rounded-sm bg-surface/95 p-0.5 text-ink shadow-[0_0_0_2px_rgb(var(--signal-500,59_130_246))] outline-none ring-2 ring-signal-500"
                style={{ fontSize: Math.max(11, Math.min(28, el.h * zoom * 0.45)) }}
              />
            ) : (
              isSel && (
                <>
                  {CORNER.map((h) => (
                    <span
                      key={h}
                      onPointerDown={start(el.index, h)}
                      style={{
                        position: "absolute",
                        [h.includes("n") ? "top" : "bottom"]: -5,
                        [h.includes("w") ? "left" : "right"]: -5,
                        cursor: CURSOR[h],
                      }}
                      className="h-2.5 w-2.5 rounded-sm border border-white bg-signal-500"
                    />
                  ))}
                  {/* The edges: width without height, height without width. */}
                  {EDGE.map((h) => {
                    const vertical = h === "n" || h === "s";
                    return (
                      <span
                        key={h}
                        onPointerDown={start(el.index, h)}
                        style={{
                          position: "absolute",
                          cursor: CURSOR[h],
                          ...(vertical
                            ? {
                                left: "50%",
                                marginLeft: -5,
                                [h === "n" ? "top" : "bottom"]: -5,
                              }
                            : {
                                top: "50%",
                                marginTop: -5,
                                [h === "w" ? "left" : "right"]: -5,
                              }),
                        }}
                        className="h-2.5 w-2.5 rounded-sm border border-white bg-signal-500"
                      />
                    );
                  })}
                </>
              )
            )}
          </div>
        );
      })}

      {/* ⚠️ **The drop layer, and it has to be a layer.** Drag events do not
          cross into an iframe: the document inside owns them, and it is another
          origin, so nothing dropped over the preview would ever be heard. A
          transparent sheet over the frame — mounted only while something is
          actually being dragged, so it never swallows an ordinary click —
          receives the drag instead, and the geometry the page already reports
          says which band is under the cursor. */}
      {incoming && (
        <div
          style={{
            position: "fixed",
            left: frameRect.left,
            top: frameRect.top,
            width: frameRect.width,
            height: frameRect.height,
            zIndex: 90,
          }}
          onDragOver={(e) => {
            e.preventDefault();
            e.dataTransfer.dropEffect = "copy";
            setHover(locate(e.clientX, e.clientY));
          }}
          onDragLeave={() => setHover(null)}
          onDrop={(e) => {
            e.preventDefault();
            const at = locate(e.clientX, e.clientY);
            setHover(null);
            if (!at) return;
            if (incoming.kind === "band") {
              onDropBand?.(at.after ? at.band + 1 : at.band);
              return;
            }
            if (!at.ok) return;
            onDropElement?.(at.band, at.x, at.y);
          }}
        >
          {hover && hoverRect(hover.band) && (
            <div
              style={{
                position: "absolute",
                left: (hoverRect(hover.band)!.x - bandOrigin.x) * zoom,
                top: (hoverRect(hover.band)!.y - bandOrigin.y) * zoom,
                width: hoverRect(hover.band)!.w * zoom,
                height: hoverRect(hover.band)!.h * zoom,
              }}
              className={`pointer-events-none flex items-start justify-center border-2 border-dashed ${
                incoming.kind === "band"
                  ? "border-transparent"
                  : hover.ok
                    ? "border-signal-500 bg-signal-500/10"
                    : "border-hot-600 bg-hot-600/10"
              }`}
            >
              {/* A section drops **between** bands, so the mark is a line where
                  it will land rather than a box around what it is beside. */}
              {incoming.kind === "band" && (
                <span
                  className="absolute left-0 right-0 h-1 rounded-full bg-signal-500"
                  style={{ [hover.after ? "bottom" : "top"]: -2 }}
                />
              )}
            </div>
          )}
        </div>
      )}

      {/* What the drag is actually doing, in the numbers the document stores.
          ⚠️ Percent, because that is what is saved — a readout in pixels would
          be a number the operator cannot find again anywhere in the editor. */}
      {readout && (
        <div
          style={{
            position: "fixed",
            left: frameRect.left + 12,
            top: frameRect.top + 12,
            zIndex: 80,
          }}
          className="rounded-lg bg-ink/85 px-2.5 py-1.5 text-[11px] font-semibold tabular-nums text-surface"
        >
          {readout}
        </div>
      )}

      {/* ⚠️ **Nothing is drawn when the selected band is a fixed one**, and that
          is correct rather than broken: a hero or a menu grid has no freely
          placed elements to drag. The page is still live — clicking anything on
          it selects that band, because the listener above is mounted whether or
          not this band happens to be drawable. An "…" badge used to sit here
          announcing the absence, which said nothing anybody could act on. */}
    </>
  );
}
