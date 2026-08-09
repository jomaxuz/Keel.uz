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

type Handle = "move" | "se" | "sw" | "ne" | "nw";

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
}) {
  const [geo, setGeo] = useState<Geometry>({ bands: [], elements: [] });
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
    }
    function up() {
      if (!drag.current) return;
      drag.current = null;
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
        return (
          <div
            key={`${el.band}-${el.index}`}
            onPointerDown={start(el.index, "move")}
            style={{
              position: "fixed",
              left: frameRect.left + el.x * zoom,
              top: frameRect.top + el.y * zoom,
              width: el.w * zoom,
              height: el.h * zoom,
              zIndex: 60,
            }}
            className={`cursor-move ${
              isSel ? "ring-2 ring-signal-500" : "ring-1 ring-signal-500/30 hover:ring-signal-500/70"
            }`}
          >
            {isSel && (
              <>
                {(["nw", "ne", "sw", "se"] as Handle[]).map((h) => (
                  <span
                    key={h}
                    onPointerDown={start(el.index, h)}
                    style={{
                      position: "absolute",
                      [h.includes("n") ? "top" : "bottom"]: -5,
                      [h.includes("w") ? "left" : "right"]: -5,
                      cursor: h === "nw" || h === "se" ? "nwse-resize" : "nesw-resize",
                    }}
                    className="h-2.5 w-2.5 rounded-sm border border-white bg-signal-500"
                  />
                ))}
              </>
            )}
          </div>
        );
      })}

      {/* ⚠️ Said out loud when there is nothing to edit. An overlay that is simply
          absent looks identical to an overlay that is broken, and the commonest
          reason is real: the selected band is not a freely drawn one, or the site
          has not reported its geometry yet. */}
      {!band && (
        <div
          style={{
            position: "fixed",
            left: frameRect.left + 12,
            top: frameRect.top + 12,
            zIndex: 60,
          }}
          className="rounded-lg bg-ink/85 px-2.5 py-1.5 text-[11px] font-semibold text-surface"
        >
          …
        </div>
      )}
    </>
  );
}
