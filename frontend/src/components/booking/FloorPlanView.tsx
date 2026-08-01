"use client";

// The restaurant's floor plan, drawn once and used everywhere: the guest picks
// a table on it, the panel shows who is sitting where. Everything is an SVG in
// the plan's own coordinate space, so the same drawing scales from a phone to a
// wall-mounted screen without any of it being re-laid out.

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
          <rect
            x={s.x}
            y={s.y}
            width={s.w}
            height={s.h}
            rx={s.kind === "area" ? 12 : 2}
            className={
              s.kind === "area"
                ? "fill-brand/5 stroke-brand/30"
                : "fill-ink/15 stroke-ink/25"
            }
            strokeWidth={2}
          />
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
            {tb.shape === "circle" ? (
              <ellipse
                cx={cx}
                cy={cy}
                rx={tb.w / 2}
                ry={tb.h / 2}
                className={`${fill} ${stroke}`}
                strokeWidth={2}
              />
            ) : (
              <rect
                x={tb.x}
                y={tb.y}
                width={tb.w}
                height={tb.h}
                rx={8}
                className={`${fill} ${stroke}`}
                strokeWidth={2}
              />
            )}
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
