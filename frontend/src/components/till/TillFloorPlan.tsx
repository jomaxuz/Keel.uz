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
import {
  stateColor,
  stateLine,
  stateTint,
  tableState,
  type TableState,
} from "@/lib/tableState";
import type { Check, FloorShape, FloorTable } from "@/lib/types";

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
        const state = tableState(check);
        const round = tb.shape === "circle";
        const cx = tb.x + tb.w / 2;
        const cy = tb.y + tb.h / 2;
        // ⚠️ The badge scales with the table but is floored: a two-seat table
        // drawn small in the editor still has to carry a number readable from
        // across the room, and a proportional circle on it is a dot. It also
        // shrinks once the table is occupied, because two more lines have to
        // fit under it — and they have to fit **inside** the table: the first
        // version put the minutes over the bottom chairs on any table the owner
        // had drawn short, which is most of the two-tops.
        const r = Math.max(open ? 15 : 18, Math.min(tb.w, tb.h) * (open ? 0.2 : 0.26));
        // Everything an occupied table says, spaced off its own height rather
        // than off the badge, so a short table stays inside its own edges.
        const badgeY = open ? cy - tb.h * 0.17 : cy;
        const moneyY = cy + tb.h * 0.14;
        const timeY = cy + tb.h * 0.33;
        return (
          <g
            key={tb.id}
            onClick={() => onPick(tb, check)}
            style={{ cursor: "pointer" }}
            role="button"
            aria-label={`${tb.number} · ${
              state === "billed"
                ? t.till.billed
                : open
                  ? t.till.busyLabel
                  : t.till.free
            }`}
          >
            {/* ⚠️ **Chairs here too, and the grid draws the same ones.** The two
                views are the same room; a table that is an object in one and a
                rectangle in the other is two things to learn. Only rectangles
                get them: a round table's chairs would need placing around an
                ellipse, and at plan scale that is four smudges. */}
            {!round && <Chairs table={tb} state={state} />}

            {round ? (
              <ellipse
                cx={cx}
                cy={cy}
                rx={tb.w / 2}
                ry={tb.h / 2}
                fill={stateTint(state)}
                stroke={stateLine(state)}
                strokeWidth={2.5}
              />
            ) : (
              <rect
                x={tb.x}
                y={tb.y}
                width={tb.w}
                height={tb.h}
                rx={12}
                fill={stateTint(state)}
                stroke={stateLine(state)}
                strokeWidth={2.5}
              />
            )}

            {/* ⚠️ **Filled when free, outlined when taken** — the grid's
                inversion, for the grid's reason: a free table is what somebody
                seating a party is hunting for, so it carries colour at full
                strength, while a taken one needs its middle left legible under
                the three things it has to say. */}
            <circle
              cx={cx}
              cy={badgeY}
              r={r}
              fill={open ? "rgb(var(--surface))" : stateColor(state)}
              stroke={stateColor(state)}
              strokeWidth={2.5}
            />
            <text
              x={cx}
              y={badgeY}
              textAnchor="middle"
              dominantBaseline="central"
              fill={open ? "rgb(var(--fg))" : "#fff"}
              style={{ fontSize: r * 0.86, fontWeight: 700 }}
            >
              {tb.number}
            </text>

            {open ? (
              <>
                {/* The two numbers worth knowing without walking over: what
                    they owe, and how long nobody has looked at them. */}
                <Money
                  x={cx}
                  y={moneyY}
                  width={tb.w}
                  text={formatPrice(check!.total, currency, lang)}
                />
                <text
                  x={cx}
                  y={timeY}
                  textAnchor="middle"
                  dominantBaseline="central"
                  fill={
                    state === "open"
                      ? "rgb(var(--till-mid))"
                      : stateColor(state)
                  }
                  style={{ fontSize: 13, fontWeight: 700 }}
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
                {/* Cooked and not carried out yet — see TableObject for why
                    this one is counted rather than a dot. */}
                {(check!.readyWaiting ?? 0) > 0 && (
                  <>
                    <circle cx={tb.x + 14} cy={tb.y + 12} r={9} fill="#0f8a5f" />
                    <text
                      x={tb.x + 14}
                      y={tb.y + 12}
                      textAnchor="middle"
                      dominantBaseline="central"
                      fill="#fff"
                      style={{ fontSize: 11, fontWeight: 700 }}
                    >
                      {check!.readyWaiting}
                    </text>
                  </>
                )}
              </>
            ) : null}
          </g>
        );
      })}
    </svg>
  );
}

/** Chair stubs along the long edges — see components/till/TableObject.
 *
 *  ⚠️ Capped at four a side. Chairs say "two-top" or "big table" at a glance;
 *  eleven of them say "stripe", and the editor lets an owner type any number. */
function Chairs({ table: tb, state }: { table: FloorTable; state: TableState }) {
  const top = Math.min(4, Math.ceil((tb.seats || 0) / 2));
  const bottom = Math.min(4, Math.floor((tb.seats || 0) / 2));
  const fill = state === "free" ? "rgb(0 0 0 / 0.14)" : stateLine(state);
  const w = Math.min(28, tb.w / 5);
  const h = 7;
  const row = (n: number, y: number) =>
    Array.from({ length: n }, (_, i) => (
      <rect
        key={`${y}-${i}`}
        x={tb.x + (tb.w * (i + 1)) / (n + 1) - w / 2}
        y={y}
        width={w}
        height={h}
        rx={h / 2}
        fill={fill}
      />
    ));
  return (
    <>
      {row(top, tb.y - h - 4)}
      {row(bottom, tb.y + tb.h + 4)}
    </>
  );
}

/** What a table owes, written so that it stays on the table.
 *
 *  ⚠️ **SVG text neither wraps nor clips.** At a fixed size a long total simply
 *  runs out past the table and across its neighbours — and "long" here is not
 *  exotic: 1 250 000 so'm is a normal evening for four people, and the number
 *  grows by a digit exactly when the room is busiest and the plan is most
 *  crowded.
 *
 *  So the size is chosen from the space there is. It steps down to a floor
 *  rather than shrinking without limit — past that the number is unreadable
 *  from standing height, which is the only distance this screen is read from —
 *  and below the floor the glyphs are squeezed instead (`textLength`), because
 *  a compressed number that is still on its table beats a comfortable one that
 *  is on somebody else's. */
function Money({
  x,
  y,
  width,
  text,
}: {
  x: number;
  y: number;
  /** The table's own width, in plan units. */
  width: number;
  text: string;
}) {
  // The plan's rounded corners and stroke take the edges; leave them alone.
  const inner = Math.max(24, width - 14);
  // ⚠️ A digit's advance is about 0.58 of the font size in this face, and the
  // spaces in "1 250 000" are narrower — measuring properly would mean a DOM
  // read per table per render, on the screen that has to stay smooth during
  // service.
  const natural = text.length * 0.58 * 15;
  const size = natural <= inner ? 15 : Math.max(10, (inner / (text.length * 0.58)));
  const squeezed = text.length * 0.58 * size > inner;
  return (
    <text
      x={x}
      y={y}
      textAnchor="middle"
      dominantBaseline="central"
      className="fill-ink"
      style={{ fontSize: size, fontWeight: 700 }}
      textLength={squeezed ? inner : undefined}
      lengthAdjust={squeezed ? "spacingAndGlyphs" : undefined}
    >
      {text}
    </text>
  );
}
