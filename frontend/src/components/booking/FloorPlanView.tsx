"use client";

// The restaurant's floor plan, drawn once and used everywhere: the guest picks
// a table on it, the panel shows who is sitting where. Everything is an SVG in
// the plan's own coordinate space, so the same drawing scales from a phone to a
// wall-mounted screen without any of it being re-laid out.

import { floorPaint } from "@/lib/floorColors";
import type { FloorShape, FloorTable } from "@/lib/types";

export interface FloorPlanViewProps {
  width: number;
  height: number;
  shapes: FloorShape[];
  tables: FloorTable[];
  /** Tables held by someone else at the moment being viewed. */
  busyIds?: Set<string>;
  selectedId?: string | null;
  onSelect?: (table: FloorTable) => void;
  className?: string;
  /** Extra layer drawn on top (editor handles, labels). */
  overlay?: React.ReactNode;
}

export default function FloorPlanView({
  width,
  height,
  shapes,
  tables,
  busyIds,
  selectedId,
  onSelect,
  className = "",
  overlay,
}: FloorPlanViewProps) {
  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      className={`h-auto w-full touch-manipulation rounded-2xl border border-line bg-ink/[0.02] ${className}`}
      role="img"
    >
      {/* Room outline and any walls or zones the owner drew. */}
      {shapes.map((s, i) => (
        <g key={`shape-${i}`}>
          {/* ⚠️ **The owner's colour, and the default when they have not
              picked one.** Every shape drawn before colours existed has none,
              and reading that as a value would repaint every floor plan in the
              country on the deploy that shipped this. */}
          {(() => {
            const paint = floorPaint(s.color, {
              fill: s.kind === "area" ? "rgb(var(--brand) / 0.05)" : "rgb(0 0 0 / 0.10)",
              line: s.kind === "area" ? "rgb(var(--brand) / 0.30)" : "rgb(0 0 0 / 0.18)",
              swatch: "",
            });
            return (
              <rect
                x={s.x}
                y={s.y}
                width={s.w}
                height={s.h}
                rx={s.kind === "area" ? 12 : 2}
                fill={paint.fill}
                stroke={paint.line}
                strokeWidth={2}
              />
            );
          })()}
          {s.label && (
            <text
              x={s.x + s.w / 2}
              y={s.y + s.h / 2}
              textAnchor="middle"
              dominantBaseline="middle"
              className="fill-ink-muted text-[13px]"
            >
              {s.label}
            </text>
          )}
        </g>
      ))}

      {tables.map((tb) => {
        const busy = busyIds?.has(tb.id) ?? false;
        const selected = selectedId === tb.id;
        const disabled = !tb.isActive;
        const cx = tb.x + tb.w / 2;
        const cy = tb.y + tb.h / 2;
        // Free tables invite a tap; taken ones stay visible but read as taken —
        // a guest should see *where* the room is full, not just be blocked.
        const fill = disabled
          ? "fill-ink/10"
          : busy
            ? "fill-rose-500/25"
            : selected
              ? "fill-brand"
              : "fill-emerald-500/20";
        const stroke = disabled
          ? "stroke-ink/20"
          : busy
            ? "stroke-rose-500/70"
            : selected
              ? "stroke-brand"
              : "stroke-emerald-600/60";

        return (
          <g
            key={tb.id}
            onClick={() => onSelect?.(tb)}
            className={onSelect && !disabled ? "cursor-pointer" : undefined}
          >
            {/* ⚠️ **A table's own colour only shows when its state has nothing
                to say.** Free, taken, unavailable and selected are what a guest
                is reading this plan for, and a decorative colour that overrode
                any of them would make the room lie about which tables are left.
                So the paint applies to the ordinary free table and nowhere
                else — which is exactly where "this is the terrace" is useful. */}
            {(() => {
              const own =
                tb.color && !disabled && !busy && !selected
                  ? floorPaint(tb.color, { fill: "", line: "", swatch: "" })
                  : null;
              const shared = {
                strokeWidth: 2,
                ...(own
                  ? { fill: own.fill, stroke: own.line }
                  : { className: `${fill} ${stroke}` }),
              };
              return tb.shape === "circle" ? (
                <ellipse cx={cx} cy={cy} rx={tb.w / 2} ry={tb.h / 2} {...shared} />
              ) : (
                <rect x={tb.x} y={tb.y} width={tb.w} height={tb.h} rx={8} {...shared} />
              );
            })()}
            <text
              x={cx}
              y={cy - 2}
              textAnchor="middle"
              dominantBaseline="middle"
              className={`text-[16px] font-bold ${
                selected ? "fill-white" : "fill-ink"
              }`}
            >
              {tb.number}
            </text>
            {tb.seats > 0 && (
              <text
                x={cx}
                y={cy + 15}
                textAnchor="middle"
                dominantBaseline="middle"
                className={`text-[11px] ${
                  selected ? "fill-white/80" : "fill-ink-muted"
                }`}
              >
                {tb.seats} ✦
              </text>
            )}
          </g>
        );
      })}

      {overlay}
    </svg>
  );
}
