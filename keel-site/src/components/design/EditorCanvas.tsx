"use client";

// The middle pane: the surface elements are actually moved on.
//
// The editor's first version placed elements by typing X/Y/W/H. That is not an
// editor, it is a form that happens to describe a page — and it makes the one
// thing a designer does constantly (nudge something until it looks right) the
// slowest thing available.
//
// ⚠️ **Why this is not the live site iframe.** Dragging inside the real page would
// mean shipping editor code into every tenant's production bundle and reaching
// across an origin boundary to do it. So the manipulation happens here, on a
// faithful-but-simple rendering of the same percent boxes, and the real site sits
// beside it as a preview. The two answer different questions: this one is "where
// is this element", the preview is "does it look like the picture the customer
// sent us". Neither can answer the other's.
//
// Everything below exists because of how people actually drag things:
//
//   • **Snapping to whole percent**, with alignment guides at 0/50/100 and to the
//     edges and centres of the other elements. Without them a "centred" headline
//     is centred to within a pixel of wherever the mouse was released, and the
//     page reads as sloppy for a reason nobody can point at.
//   • **Eight resize handles**, because resizing from the wrong corner moves the
//     element as well as sizing it, and then it has to be moved back.
//   • **Arrow keys nudge, shift-arrow nudges further.** The last 1% is always done
//     with the keyboard.
//   • **Pointer events, not mouse events.** The same code then works with a
//     trackpad, a touchscreen and a stylus, and a drag that leaves the canvas is
//     still tracked (`setPointerCapture`) rather than sticking to the cursor.

import { useCallback, useEffect, useRef, useState } from "react";
import type { DesignBox, DesignElement, DesignSection } from "@/lib/api";

/** Snap threshold, in percent of the canvas. Small enough to be ignorable, large
 *  enough that a deliberate alignment lands. */
const SNAP = 1.2;

const TONE_BG: Record<string, string> = {
  "": "transparent",
  surface: "var(--surface)",
  raised: "var(--raised)",
  charcoal: "#20201e",
  brand: "var(--signal-500, #e2483d)",
};

type Handle = "nw" | "n" | "ne" | "e" | "se" | "s" | "sw" | "w" | "move";

export default function EditorCanvas({
  band,
  device,
  selected,
  onSelect,
  onBox,
  editing,
  zoom,
}: {
  band: DesignSection;
  device: "desktop" | "phone";
  selected: number | null;
  onSelect: (i: number | null) => void;
  onBox: (i: number, box: Partial<DesignBox>) => void;
  /** Which layout the drag is editing — the desktop composition or the phone one. */
  editing: "desktop" | "mobile";
  zoom: number;
}) {
  const surface = useRef<HTMLDivElement>(null);
  const [guides, setGuides] = useState<{ x: number[]; y: number[] }>({ x: [], y: [] });
  const drag = useRef<{
    index: number;
    handle: Handle;
    startX: number;
    startY: number;
    box: DesignBox;
  } | null>(null);

  const elements = band.canvas?.elements ?? [];
  const boxOf = useCallback(
    (el: DesignElement): DesignBox =>
      editing === "mobile" ? (el.mobile ?? el.box) : el.box,
    [editing],
  );

  // The band's own height, in the same units the site uses.
  const heightVh =
    (device === "phone" ? band.canvas?.heightMobile || band.canvas?.height : band.canvas?.height) ??
    60;
  // A canvas the size of a phone or of a desktop viewport, scaled. The aspect
  // ratio matters more than the absolute size: it is what makes a composition
  // that fits here fit there.
  const width = device === "phone" ? 390 : 1280;
  const height = Math.round(((device === "phone" ? 780 : 800) * heightVh) / 100);

  /** Percent of the canvas from a pointer position. */
  function pct(e: PointerEvent | React.PointerEvent) {
    const rect = surface.current?.getBoundingClientRect();
    if (!rect) return { x: 0, y: 0 };
    return {
      x: ((e.clientX - rect.left) / rect.width) * 100,
      y: ((e.clientY - rect.top) / rect.height) * 100,
    };
  }

  /** Snap one edge against the canvas and the other elements. */
  function snap(value: number, axis: "x" | "y", skip: number): { value: number; guide?: number } {
    const targets: number[] = [0, 50, 100];
    elements.forEach((el, i) => {
      if (i === skip || el.hidden) return;
      const b = boxOf(el);
      if (axis === "x") targets.push(b.x, b.x + b.w, b.x + b.w / 2);
      else targets.push(b.y, b.y + b.h, b.y + b.h / 2);
    });
    for (const t of targets) {
      if (Math.abs(value - t) <= SNAP) return { value: t, guide: t };
    }
    return { value: Math.round(value) };
  }

  const onPointerDown = (index: number, handle: Handle) => (e: React.PointerEvent) => {
    e.stopPropagation();
    e.preventDefault();
    const el = elements[index];
    if (!el) return;
    onSelect(index);
    const p = pct(e);
    drag.current = { index, handle, startX: p.x, startY: p.y, box: { ...boxOf(el) } };
    // Captured so a drag that leaves the canvas — which is most of them, at the
    // edges — keeps being tracked instead of freezing mid-move.
    (e.target as HTMLElement).setPointerCapture?.(e.pointerId);
  };

  useEffect(() => {
    function move(e: PointerEvent) {
      const d = drag.current;
      if (!d) return;
      const p = pct(e);
      const dx = p.x - d.startX;
      const dy = p.y - d.startY;
      const b = { ...d.box };
      const g: { x: number[]; y: number[] } = { x: [], y: [] };

      if (d.handle === "move") {
        const sx = snap(d.box.x + dx, "x", d.index);
        const sy = snap(d.box.y + dy, "y", d.index);
        b.x = sx.value;
        b.y = sy.value;
        if (sx.guide != null) g.x.push(sx.guide);
        if (sy.guide != null) g.y.push(sy.guide);
        // The trailing edges snap too: aligning a block's right edge to another's
        // is as common as aligning its left, and only one of them is the value
        // being dragged.
        const rx = snap(d.box.x + dx + d.box.w, "x", d.index);
        if (rx.guide != null && Math.abs(rx.value - (d.box.x + dx + d.box.w)) < SNAP) {
          b.x = rx.value - d.box.w;
          g.x.push(rx.guide);
        }
      } else {
        if (d.handle.includes("w")) {
          const s = snap(d.box.x + dx, "x", d.index);
          b.w = d.box.w + (d.box.x - s.value);
          b.x = s.value;
          if (s.guide != null) g.x.push(s.guide);
        }
        if (d.handle.includes("e")) {
          const s = snap(d.box.x + d.box.w + dx, "x", d.index);
          b.w = s.value - d.box.x;
          if (s.guide != null) g.x.push(s.guide);
        }
        if (d.handle.includes("n")) {
          const s = snap(d.box.y + dy, "y", d.index);
          b.h = d.box.h + (d.box.y - s.value);
          b.y = s.value;
          if (s.guide != null) g.y.push(s.guide);
        }
        if (d.handle.includes("s")) {
          const s = snap(d.box.y + d.box.h + dy, "y", d.index);
          b.h = s.value - d.box.y;
          if (s.guide != null) g.y.push(s.guide);
        }
        // A box dragged past its own opposite edge would invert. Clamped rather
        // than allowed to flip: an element that turns inside out under the cursor
        // is impossible to recover by dragging back.
        b.w = Math.max(2, b.w);
        b.h = Math.max(2, b.h);
      }
      setGuides(g);
      onBox(d.index, b);
    }
    function up() {
      drag.current = null;
      setGuides({ x: [], y: [] });
    }
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
    window.addEventListener("pointercancel", up);
    return () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
      window.removeEventListener("pointercancel", up);
    };
    // `snap` and `onBox` are stable enough for a drag session; re-subscribing on
    // every element change would drop an in-flight drag.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [elements, editing, onBox]);

  // Keyboard nudging. ⚠️ Ignored while typing: the inspector's fields are one tab
  // away, and an arrow key that moves the element instead of the caret makes the
  // text fields feel broken.
  useEffect(() => {
    function key(e: KeyboardEvent) {
      if (selected == null) return;
      const tag = (e.target as HTMLElement)?.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
      const step = e.shiftKey ? 5 : 1;
      const el = elements[selected];
      if (!el) return;
      const b = boxOf(el);
      const map: Record<string, Partial<DesignBox>> = {
        ArrowLeft: { x: b.x - step },
        ArrowRight: { x: b.x + step },
        ArrowUp: { y: b.y - step },
        ArrowDown: { y: b.y + step },
      };
      const patch = map[e.key];
      if (!patch) return;
      e.preventDefault();
      onBox(selected, patch);
    }
    window.addEventListener("keydown", key);
    return () => window.removeEventListener("keydown", key);
  }, [selected, elements, boxOf, onBox]);

  return (
    <div className="flex justify-center">
      <div
        style={{
          width,
          height,
          transform: `scale(${zoom})`,
          transformOrigin: "top center",
        }}
        // A ring rather than a border: a border would take a pixel off the inside
        // and every percent below would be measured against the wrong box.
        className="relative shrink-0 overflow-hidden rounded-xl bg-surface ring-1 ring-line"
        ref={surface}
        onPointerDown={() => onSelect(null)}
      >
        {/* The band's own background, so a composition on a dark band is judged
            against the dark band it will actually sit on. */}
        <div
          className="absolute inset-0"
          style={{ background: TONE_BG[band.canvas?.background ?? ""] }}
        />

        {elements.map((el, i) => {
          if (el.hidden) return null;
          if (device === "phone" && el.hiddenMobile) return null;
          const b = boxOf(el);
          const isSel = selected === i;
          // ⚠️ On a phone with no drawn mobile layout the site stacks in flow. The
          // canvas says so rather than pretending the desktop box applies: a
          // dashed outline and no handles, because dragging here would silently
          // create a phone layout the operator did not ask for.
          const flows = device === "phone" && editing === "desktop" && !el.mobile;
          return (
            <div
              key={i}
              onPointerDown={flows ? undefined : onPointerDown(i, "move")}
              style={{
                left: `${b.x}%`,
                top: `${b.y}%`,
                width: `${b.w}%`,
                height: `${b.h}%`,
                zIndex: (b.z ?? 0) + (isSel ? 30 : 0),
                background:
                  el.type === "box" || el.type === "image"
                    ? TONE_BG[el.style?.tone ?? ""] || "rgba(0,0,0,.06)"
                    : "transparent",
                opacity: el.style?.opacity != null ? el.style.opacity / 100 : 1,
              }}
              className={`absolute flex items-center justify-center overflow-hidden text-center text-[11px] leading-tight ${
                flows ? "cursor-not-allowed border border-dashed border-line-strong" : "cursor-move"
              } ${isSel ? "ring-2 ring-signal-500" : "ring-1 ring-line/60"}`}
            >
              <span className="pointer-events-none px-1 text-ink-soft">
                {label(el)}
              </span>

              {/* ⚠️ The selection's own label, above the box.
                  A selected rectangle with handles says "something is selected";
                  it does not say **what**, and on a composition of overlapping
                  shapes that is the question. It sits outside the box (negative
                  top) so it never covers the content it names. */}
              {isSel && (
                <span className="pointer-events-none absolute -top-5 left-0 z-40 whitespace-nowrap rounded-md bg-signal-500 px-1.5 py-0.5 text-[10px] font-semibold text-white">
                  {label(el) === el.type ? el.type : kind(el)}
                </span>
              )}

              {isSel && !flows && (
                <>
                  {(["nw", "n", "ne", "e", "se", "s", "sw", "w"] as Handle[]).map((h) => (
                    <span
                      key={h}
                      onPointerDown={onPointerDown(i, h)}
                      style={handleStyle(h)}
                      className="absolute h-3 w-3 rounded-sm border border-surface bg-signal-500"
                    />
                  ))}
                </>
              )}
            </div>
          );
        })}

        {/* Alignment guides, only while dragging. */}
        {guides.x.map((x) => (
          <div
            key={`x${x}`}
            className="pointer-events-none absolute top-0 z-40 h-full w-px bg-signal-500/70"
            style={{ left: `${x}%` }}
          />
        ))}
        {guides.y.map((y) => (
          <div
            key={`y${y}`}
            className="pointer-events-none absolute left-0 z-40 h-px w-full bg-signal-500/70"
            style={{ top: `${y}%` }}
          />
        ))}
      </div>
    </div>
  );
}

/** What an element shows on the canvas: its own text where it has some, its kind
 *  where it does not. A canvas of identical grey rectangles is a canvas nobody can
 *  navigate. */
function label(el: DesignElement): string {
  if (el.text?.uz) return el.text.uz;
  const names: Record<string, string> = {
    text: "matn",
    image: "rasm",
    button: "tugma",
    box: "shakl",
    divider: "chiziq",
    "widget-menu": "MENYU",
    "widget-categories": "KATEGORIYALAR",
    "widget-hours": "ISH VAQTI",
    "widget-map": "XARITA",
    "widget-cart": "SAVAT",
  };
  return names[el.type] ?? el.type;
}

/** The element's kind, for the selection label — as opposed to its text. */
function kind(el: DesignElement): string {
  const names: Record<string, string> = {
    text: "Matn",
    image: "Rasm",
    button: "Tugma",
    box: "Shakl",
    divider: "Chiziq",
    "widget-menu": "Menyu",
    "widget-categories": "Kategoriyalar",
    "widget-hours": "Ish vaqti",
    "widget-map": "Xarita",
    "widget-cart": "Savat",
  };
  return names[el.type] ?? el.type;
}

function handleStyle(h: Handle): React.CSSProperties {
  const mid = "calc(50% - 6px)";
  const map: Record<string, React.CSSProperties> = {
    nw: { left: -6, top: -6, cursor: "nwse-resize" },
    n: { left: mid, top: -6, cursor: "ns-resize" },
    ne: { right: -6, top: -6, cursor: "nesw-resize" },
    e: { right: -6, top: mid, cursor: "ew-resize" },
    se: { right: -6, bottom: -6, cursor: "nwse-resize" },
    s: { left: mid, bottom: -6, cursor: "ns-resize" },
    sw: { left: -6, bottom: -6, cursor: "nesw-resize" },
    w: { left: -6, top: mid, cursor: "ew-resize" },
  };
  return map[h] ?? {};
}
