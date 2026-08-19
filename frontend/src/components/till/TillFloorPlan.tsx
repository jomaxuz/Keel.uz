"use client";

// The room as it actually is: tables where the owner drew them, walls and
// zones around them.
//
// ⚠️ **Its own renderer, not the booking page's.** `components/booking/
// FloorPlanView` draws the same coordinates for a guest choosing a table, and
// it answers one question — free or taken. A till's table has to carry three
// more (what they owe, how long they have been sitting, whether the kitchen has
// been told) and a different colour language, and bending one component around
// both audiences is how the guest's page ends up showing money.
//
// ⚠️ **SVG in the plan's own units**, so the same drawing scales from a 1024px
// monoblock to a wall screen without re-laying anything out.

import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import type { Check, FloorShape, FloorTable } from "@/lib/types";

/** Minutes after which a table stops being ordinary. Matches the tile grid. */
const LATE_MIN = 45;

export default function TillFloorPlan({
  width,
  height,
  shapes,
  tables,
  byTable,
  currency,
  onPick,
}: {
  width: number;
  height: number;
  shapes: FloorShape[];
  tables: FloorTable[];
  /** The open check on each table, keyed by table id. */
  byTable: Map<string, Check>;
  currency: string;
  onPick: (table: FloorTable, check?: Check) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();

  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      className="h-full w-full"
      role="group"
      aria-label={t.till.planView}
    >
      {/* Walls and named areas — the entrance, the bar, the counter. They are
          what turns a grid of rectangles into a room somebody recognises from
          where they are standing. */}
      {shapes.map((s, i) => (
        <g key={`shape-${i}`}>
          <rect
            x={s.x}
            y={s.y}
            width={s.w}
            height={s.h}
            rx={s.kind === "area" ? 10 : 3}
            fill={s.kind === "area" ? "rgb(0 0 0 / 0.03)" : "rgb(0 0 0 / 0.07)"}
            stroke="rgb(0 0 0 / 0.12)"
            strokeWidth={2}
          />
          {s.label && (
            <text
              x={s.x + s.w / 2}
              y={s.y + s.h / 2}
              textAnchor="middle"
              dominantBaseline="middle"
              className="fill-[rgb(var(--till-dim))]"
              style={{ fontSize: 15, fontWeight: 600 }}
            >
              {s.label}
            </text>
          )}
        </g>
      ))}

      {tables.map((tb) => {
        const check = byTable.get(tb.id);
        const open = !!check;
        // The table has asked to pay — waiting for a person, not for food.
        const billed = open && !!check!.precheckAt;
        const late = open && !billed && check!.openMin >= LATE_MIN;
        const round = tb.shape === "circle";
        const cx = tb.x + tb.w / 2;
        const cy = tb.y + tb.h / 2;
        return (
          <g
            key={tb.id}
            onClick={() => onPick(tb, check)}
            style={{ cursor: "pointer" }}
            role="button"
            aria-label={`${tb.number} · ${
              billed ? t.till.billed : open ? t.till.busyLabel : t.till.free
            }`}
          >
            {round ? (
              <ellipse
                cx={cx}
                cy={cy}
                rx={tb.w / 2}
                ry={tb.h / 2}
                fill={fillOf(open, late, billed)}
                stroke={strokeOf(open, late, billed)}
                strokeWidth={2.5}
              />
            ) : (
              <rect
                x={tb.x}
                y={tb.y}
                width={tb.w}
                height={tb.h}
                rx={10}
                fill={fillOf(open, late, billed)}
                stroke={strokeOf(open, late, billed)}
                strokeWidth={2.5}
              />
            )}

            {/* The number is what somebody is looking for; everything else on
                the table is context, and only exists once it is occupied. */}
            <text
              x={cx}
              y={open ? cy - 14 : cy - 4}
              textAnchor="middle"
              dominantBaseline="middle"
              className="fill-ink"
              style={{ fontSize: 26, fontWeight: 700 }}
            >
              {tb.number}
            </text>
            {open ? (
              <>
                <text
                  x={cx}
                  y={cy + 10}
                  textAnchor="middle"
                  dominantBaseline="middle"
                  className="fill-ink"
                  style={{ fontSize: 15, fontWeight: 700 }}
                >
                  {formatPrice(check!.total, currency, lang)}
                </text>
                <text
                  x={cx}
                  y={cy + 28}
                  textAnchor="middle"
                  dominantBaseline="middle"
                  fill={
                    billed
                      ? "rgb(var(--till-info))"
                      : late
                        ? "rgb(var(--till-late))"
                        : "rgb(var(--till-mid))"
                  }
                  style={{ fontSize: 13, fontWeight: 600 }}
                >
                  {check!.openMin} {t.till.minShort}
                </text>
                {/* The kitchen has not been told about something on this table
                    — the one thing that goes quietly wrong. */}
                {check!.unfired > 0 && (
                  <circle
                    cx={tb.x + tb.w - 12}
                    cy={tb.y + 12}
                    r={6}
                    fill="rgb(var(--till-info))"
                  />
                )}
              </>
            ) : (
              <text
                x={cx}
                y={cy + 16}
                textAnchor="middle"
                dominantBaseline="middle"
                className="fill-[rgb(var(--till-dim))]"
                style={{ fontSize: 13, fontWeight: 600 }}
              >
                {tb.seats ? `${tb.seats} ${t.till.seatsShort}` : t.till.free}
              </text>
            )}
          </g>
        );
      })}
    </svg>
  );
}

function fillOf(open: boolean, late: boolean, billed: boolean): string {
  if (billed) return "#f2f7fd";
  if (late) return "#fdf3f3";
  if (open) return "#fffbf2";
  return "rgb(var(--surface))";
}

function strokeOf(open: boolean, late: boolean, billed: boolean): string {
  if (billed) return "rgb(var(--till-info) / 0.6)";
  if (late) return "rgb(var(--till-late) / 0.55)";
  if (open) return "rgb(var(--till-accent))";
  return "var(--line-strong)";
}
