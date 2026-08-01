"use client";

// Draw the room, then put the tables in it.
//
// The owner works the way they would on paper: pick a tool, drag a rectangle
// where the thing is, then correct it. Walls and zones are decoration; tables
// are what guests book, so each one keeps a stable id (a reservation points at
// it and must survive renaming, moving and renumbering).
//
// Coordinates are stored in the plan's own units, never in pixels: the same
// drawing then renders correctly on a phone and on a desk monitor.

import { useRef, useState } from "react";
import FloorPlanView from "@/components/booking/FloorPlanView";
import { useAdminT } from "@/lib/i18n/admin";
import type { BookingSettings, FloorShape, FloorTable } from "@/lib/types";

type Tool = "select" | "table" | "circle" | "wall" | "area";

const MIN_SIZE = 24;

function newId(): string {
  // Stable, short, and no dependency: the id only has to be unique per plan.
  return `t${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`;
}

export default function FloorPlanEditor({
  value,
  onChange,
}: {
  value: BookingSettings;
  onChange: (next: BookingSettings) => void;
}) {
  const t = useAdminT();
  const svgWrapRef = useRef<HTMLDivElement>(null);
  const [tool, setTool] = useState<Tool>("table");
  const [selected, setSelected] = useState<string | null>(null);
  // The rectangle currently being dragged out, in plan units.
  const [draft, setDraft] = useState<{
    x: number;
    y: number;
    w: number;
    h: number;
  } | null>(null);
  const dragStart = useRef<{ x: number; y: number } | null>(null);
  // Dragging an existing table: its offset from the pointer, so it does not
  // jump to have its corner under the finger.
  const moving = useRef<{ id: string; dx: number; dy: number } | null>(null);

  const width = value.width || 1000;
  const height = value.height || 700;
  const tables = value.tables ?? [];
  const shapes = value.shapes ?? [];

  const patch = (p: Partial<BookingSettings>) => onChange({ ...value, ...p });

  /** Pointer position in plan units. */
  function toPlan(e: React.PointerEvent): { x: number; y: number } {
    const box = svgWrapRef.current?.getBoundingClientRect();
    if (!box) return { x: 0, y: 0 };
    return {
      x: ((e.clientX - box.left) / box.width) * width,
      y: ((e.clientY - box.top) / box.height) * height,
    };
  }

  function onPointerDown(e: React.PointerEvent) {
    const p = toPlan(e);
    if (tool === "select") {
      // Grab whichever table is under the pointer, topmost first.
      const hit = [...tables]
        .reverse()
        .find(
          (tb) => p.x >= tb.x && p.x <= tb.x + tb.w && p.y >= tb.y && p.y <= tb.y + tb.h,
        );
      if (hit) {
        setSelected(hit.id);
        moving.current = { id: hit.id, dx: p.x - hit.x, dy: p.y - hit.y };
        (e.target as Element).setPointerCapture?.(e.pointerId);
      } else {
        setSelected(null);
      }
      return;
    }
    dragStart.current = p;
    setDraft({ x: p.x, y: p.y, w: 0, h: 0 });
    (e.target as Element).setPointerCapture?.(e.pointerId);
  }

  function onPointerMove(e: React.PointerEvent) {
    const p = toPlan(e);
    if (moving.current) {
      const m = moving.current;
      patch({
        tables: tables.map((tb) =>
          tb.id === m.id
            ? {
                ...tb,
                x: Math.max(0, Math.min(width - tb.w, p.x - m.dx)),
                y: Math.max(0, Math.min(height - tb.h, p.y - m.dy)),
              }
            : tb,
        ),
      });
      return;
    }
    if (!dragStart.current) return;
    const s = dragStart.current;
    setDraft({
      x: Math.min(s.x, p.x),
      y: Math.min(s.y, p.y),
      w: Math.abs(p.x - s.x),
      h: Math.abs(p.y - s.y),
    });
  }

  function onPointerUp() {
    if (moving.current) {
      moving.current = null;
      return;
    }
    const d = draft;
    dragStart.current = null;
    setDraft(null);
    if (!d) return;

    // A tap rather than a drag still creates something usable.
    const w = Math.max(d.w, MIN_SIZE * (tool === "wall" ? 0.4 : 2));
    const h = Math.max(d.h, MIN_SIZE * (tool === "wall" ? 0.4 : 2));

    if (tool === "table" || tool === "circle") {
      const number = String(
        tables.reduce((max, tb) => Math.max(max, Number(tb.number) || 0), 0) + 1,
      );
      const table: FloorTable = {
        id: newId(),
        number,
        seats: 4,
        shape: tool === "circle" ? "circle" : "rect",
        x: d.x,
        y: d.y,
        w,
        h,
        isActive: true,
        note: "",
      };
      patch({ tables: [...tables, table] });
      setSelected(table.id);
      return;
    }

    const shape: FloorShape = {
      kind: tool === "area" ? "area" : "wall",
      label: "",
      x: d.x,
      y: d.y,
      w,
      h,
    };
    patch({ shapes: [...shapes, shape] });
  }

  const current = tables.find((tb) => tb.id === selected) ?? null;

  function updateTable(id: string, p: Partial<FloorTable>) {
    patch({ tables: tables.map((tb) => (tb.id === id ? { ...tb, ...p } : tb)) });
  }

  const tools: { key: Tool; label: string }[] = [
    { key: "table", label: t.booking.toolTable },
    { key: "circle", label: t.booking.toolCircle },
    { key: "wall", label: t.booking.toolWall },
    { key: "area", label: t.booking.toolArea },
    { key: "select", label: t.booking.toolMove },
  ];

  return (
    <div>
      <div className="flex flex-wrap items-center gap-2">
        {tools.map((op) => (
          <button
            key={op.key}
            type="button"
            onClick={() => setTool(op.key)}
            className={`rounded-full px-3 py-1.5 text-xs font-semibold transition-colors ${
              tool === op.key
                ? "bg-brand text-white"
                : "border border-line-strong text-ink-soft hover:border-brand"
            }`}
          >
            {op.label}
          </button>
        ))}
        {shapes.length > 0 && (
          <button
            type="button"
            onClick={() => patch({ shapes: [] })}
            className="ml-auto text-xs text-ink-muted hover:text-red-600"
          >
            {t.booking.clearShapes}
          </button>
        )}
      </div>

      <p className="mt-2 text-xs text-ink-muted">{t.booking.editorHint}</p>

      <div
        ref={svgWrapRef}
        className="mt-3 select-none"
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerLeave={onPointerUp}
      >
        <FloorPlanView
          width={width}
          height={height}
          shapes={shapes}
          tables={tables}
          selectedId={selected}
          overlay={
            draft ? (
              <rect
                x={draft.x}
                y={draft.y}
                width={draft.w}
                height={draft.h}
                className="fill-brand/20 stroke-brand"
                strokeDasharray="6 4"
                strokeWidth={2}
              />
            ) : null
          }
        />
      </div>

      {/* The selected table's details. Numbering is the owner's business: they
          may already have "12" painted on it. */}
      {current ? (
        <div className="mt-4 rounded-2xl border border-line bg-ink/[0.02] p-4">
          <div className="flex flex-wrap items-center gap-3">
            <label className="text-sm">
              <span className="font-medium">{t.booking.tableNumber}</span>
              <input
                className="mt-1 w-24 rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand"
                value={current.number}
                onChange={(e) => updateTable(current.id, { number: e.target.value })}
              />
            </label>
            <label className="text-sm">
              <span className="font-medium">{t.booking.seats}</span>
              <input
                type="number"
                min={1}
                className="mt-1 w-24 rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand"
                value={current.seats}
                onChange={(e) =>
                  updateTable(current.id, { seats: Number(e.target.value) || 0 })
                }
              />
            </label>
            <label className="flex items-center gap-2 pt-5 text-sm">
              <input
                type="checkbox"
                checked={current.isActive}
                onChange={(e) =>
                  updateTable(current.id, { isActive: e.target.checked })
                }
              />
              <span>{t.booking.tableActive}</span>
            </label>
            <button
              type="button"
              onClick={() =>
                updateTable(current.id, {
                  shape: current.shape === "circle" ? "rect" : "circle",
                })
              }
              className="mt-5 rounded-full border border-line-strong px-3 py-1.5 text-xs font-semibold hover:border-brand"
            >
              {t.booking.toggleShape}
            </button>
            <button
              type="button"
              onClick={() => {
                patch({ tables: tables.filter((tb) => tb.id !== current.id) });
                setSelected(null);
              }}
              className="ml-auto mt-5 text-xs text-ink-muted hover:text-red-600"
            >
              {t.booking.deleteTable}
            </button>
          </div>

          {/* Size, for when dragging is fiddly on a laptop trackpad. */}
          <div className="mt-3 flex flex-wrap gap-3 text-sm">
            <label>
              <span className="text-xs text-ink-muted">{t.booking.sizeW}</span>
              <input
                type="number"
                className="mt-1 w-24 rounded-xl border border-line-strong bg-surface px-3 py-1.5 text-sm outline-none focus:border-brand"
                value={Math.round(current.w)}
                onChange={(e) =>
                  updateTable(current.id, {
                    w: Math.max(MIN_SIZE, Number(e.target.value) || MIN_SIZE),
                  })
                }
              />
            </label>
            <label>
              <span className="text-xs text-ink-muted">{t.booking.sizeH}</span>
              <input
                type="number"
                className="mt-1 w-24 rounded-xl border border-line-strong bg-surface px-3 py-1.5 text-sm outline-none focus:border-brand"
                value={Math.round(current.h)}
                onChange={(e) =>
                  updateTable(current.id, {
                    h: Math.max(MIN_SIZE, Number(e.target.value) || MIN_SIZE),
                  })
                }
              />
            </label>
          </div>
        </div>
      ) : (
        <p className="mt-4 rounded-2xl border border-dashed border-line-strong p-4 text-center text-sm text-ink-muted/70">
          {tables.length === 0 ? t.booking.noTables : t.booking.selectHint}
        </p>
      )}
    </div>
  );
}
