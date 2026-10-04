"use client";

/**
 * A table, drawn as a table.
 *
 * ⚠️ **The grid used to be a grid of cards, and a room is not a list.** A
 * waiter crossing the floor is looking for an object they can see from where
 * they are standing — four seats, two seats, the long one by the window — and a
 * rectangle with a number in it makes them read every tile in order until they
 * find the right number. Chairs and a shape turn scanning back into
 * recognising, which is the only reason a floor screen beats a printed list.
 *
 * ⚠️ **The seat count is drawn, not written.** "4 o'rin" is a fact you read;
 * four chairs is a fact you see, and this tile is looked at from two metres
 * away across a room with people in it. The number stays in the accessible
 * name, where a screen reader needs the words.
 *
 * ⚠️ **The badge is the number and nothing else.** Everything a table can tell
 * you — the money, the minutes, whether the kitchen has it — is arranged
 * around a circle that never moves, so "where is T-15" is answered by the same
 * glance whatever state T-15 is in. A layout that reflows per state makes the
 * busiest room the hardest to read.
 */

import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice } from "@/lib/format";
import {
  stateColor,
  stateLine,
  stateTint,
  tableState,
  type TableState,
} from "@/lib/tableState";
import type { Check } from "@/lib/types";

export default function TableObject({
  label,
  seats,
  check,
  currency,
  compact,
  onClick,
}: {
  label: string;
  /** 0 on a counter slot, which has no chairs to draw. */
  seats?: number;
  check?: Check;
  currency: string;
  /** A counter slot rather than a table in a room.
   *
   *  ⚠️ **Drops the chair rows entirely, rather than drawing none.** A table
   *  keeps its empty chair rows so a four-top and a two-top line up on the same
   *  baseline; a counter has no chairs at all, and thirty numbers each carrying
   *  two rows of blank space is a wall that scrolls for no reason. Same object,
   *  same badge, same colours — only the furniture goes. */
  compact?: boolean;
  onClick: () => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const state = tableState(check);
  const open = !!check;

  // The time chip's colours, shared by both places it can be drawn.
  const chip = {
    background:
      state === "late"
        ? "rgb(var(--till-late) / 0.12)"
        : state === "billed"
          ? "rgb(var(--till-info) / 0.12)"
          : "rgb(var(--till-accent) / 0.18)",
    color:
      state === "late"
        ? "rgb(var(--till-late))"
        : state === "billed"
          ? "rgb(var(--till-info))"
          : "rgb(var(--till-accent-ink))",
  };

  const word =
    state === "billed"
      ? t.till.billed
      : open
        ? t.till.busyLabel
        : t.till.free;

  return (
    <button
      onClick={onClick}
      // Named by the table and its state: the tile's own text runs the number,
      // the chairs and the money together into one unreadable string.
      aria-label={`${label} · ${word}${
        seats ? ` · ${seats} ${t.till.seatsShort}` : ""
      }${open ? ` · ${formatPrice(check!.total, currency, lang)}` : ""}`}
      // ⚠️ `transition-transform` rather than `transition`: the bare class eases
      // every animatable property, so a room of forty tables re-animates its
      // colours, borders and shadows on every hover and every re-render. On a
      // monoblock that is the difference between a floor plan that answers and
      // one that swims.
      className="group flex flex-col items-center gap-1.5 rounded-[14px] p-1.5 transition-transform active:scale-[0.97]"
    >
      <span className="relative flex w-full flex-col items-center">
        {!compact && <Chairs count={topSeats(seats)} state={state} />}

        {/* ---- The table itself ---- */}
        <span
          className={`relative flex w-full items-center justify-center rounded-[14px] border-2 transition-colors group-hover:border-[rgb(var(--till-accent))] ${
            compact ? "h-[5.4rem] flex-col gap-1" : "h-[5.6rem]"
          }`}
          style={{ background: stateTint(state), borderColor: stateLine(state) }}
        >
          {/* ⚠️ **Filled when free, outlined when taken** — and that is the one
              inversion on this screen worth having. A free table is what
              somebody seating a party is hunting for, so it is the shape that
              carries colour at full strength; a taken one has three other
              things to say and needs its middle left legible. */}
          <span
            className={`flex items-center justify-center rounded-full border-2 font-bold leading-none tracking-tight ${
              compact && open
                ? "h-[2.1rem] w-[2.1rem] text-[13px]"
                : "h-[2.9rem] w-[2.9rem] text-[15px]"
            }`}
            style={
              open
                ? {
                    background: "rgb(var(--surface))",
                    borderColor: stateColor(state),
                    color: "rgb(var(--fg))",
                  }
                : {
                    background: stateColor(state),
                    borderColor: stateColor(state),
                    color: "#fff",
                  }
            }
          >
            {label}
          </span>

          {/* ⚠️ **Inside the slot, on a takeaway counter.** The time and the
              sum used to hang below every tile, as on a dining table — but a
              counter slot is drawn small and packed tight, so the line under
              one tile sat closer to the next tile than to its own, and a row of
              numbers floated between the slots belonging to none of them.
              Inside the box there is no doubt whose they are. */}
          {compact && open && (
            <span className="flex max-w-full items-center gap-1 px-1">
              <span
                className="till-num shrink-0 rounded-full px-1.5 py-[0.1rem] text-[10px] font-bold"
                style={chip}
              >
                {check!.openMin} {t.till.minShort}
              </span>
              <span className="till-num truncate text-[12px] font-bold text-ink">
                {formatPrice(check!.total, currency, lang)}
              </span>
            </span>
          )}

          {/* ⚠️ The one thing that goes quietly wrong: a line nobody has sent
              to the kitchen. It sits on the table's own corner rather than in
              the chip row, because it is true *underneath* whatever the chip
              is saying — a table can be billed and still have an unsent line. */}
          {open && check!.unfired > 0 && (
            <span
              className="absolute right-2 top-2 h-2.5 w-2.5 rounded-full"
              style={{ background: "rgb(var(--till-info))" }}
              title={t.till.pendingLabel}
            />
          )}
          {/* ⚠️ **Food standing at the pass, on the table it belongs to.** The
              dot above is what the waiter has not sent; this is what the
              kitchen has finished and nobody has collected — the half that goes
              cold, and the half a room could previously only learn by walking
              over. Green, filled and counted: at a glance across a dining room
              a colour is read and a number is trusted. */}
          {open && (check!.readyWaiting ?? 0) > 0 && (
            <span
              className="absolute left-2 top-2 flex h-5 min-w-[1.25rem] items-center justify-center rounded-full bg-emerald-600 px-1 text-[11px] font-bold text-white tabular-nums"
              title={t.till.waitingCount(check!.readyWaiting ?? 0)}
            >
              {check!.readyWaiting}
            </span>
          )}
          {/* ⚠️ **Somebody is on this table right now**, and the room says so
              before anybody taps it. The server refuses the edit either way,
              but a table that opens and then refuses every button reads as a
              broken till; a table wearing a colleague's name reads as a
              colleague. It clears itself when the hold goes stale. */}
          {open && check!.heldBy && (
            // On a counter slot the bottom of the box is the sum's, so the
            // name sits on the border instead of over the figure.
            <span
              className={`till-chip till-chip-info absolute max-w-[85%] truncate ${
                compact ? "-bottom-2.5 left-1/2 -translate-x-1/2" : "bottom-1.5 left-1.5"
              }`}
            >
              {check!.heldBy}
            </span>
          )}
          {/* Marked, because nothing else in the building knows about it: not
              the kitchen screen, not the panel, not the till next to it. */}
          {open && check!.id.startsWith("local:") && (
            <span className="till-chip till-chip-warn absolute left-1.5 top-1.5">
              {t.till.offlineCheck}
            </span>
          )}
        </span>

        {!compact && <Chairs count={bottomSeats(seats)} state={state} />}
      </span>

      {/* ---- What it is doing ----

          ⚠️ **A chip, and only when there is something to say.** A free table
          carries no row at all: a room where every tile has a badge on it is a
          room with no badges in it, and "free" under a green circle is the
          same fact twice. */}
      {!compact && (
      <span className="flex h-[1.35rem] items-center gap-1.5">
        {open ? (
          <>
            <span
              className="till-num flex items-center gap-1 rounded-full px-2 py-[0.15rem] text-[11px] font-bold"
              style={chip}
            >
              {check!.openMin} {t.till.minShort}
            </span>
            <span className="till-num text-[13px] font-bold text-ink">
              {formatPrice(check!.total, currency, lang)}
            </span>
          </>
        ) : null}
      </span>
      )}
    </button>
  );
}

/** ⚠️ Capped at four a side and never zero-padded. Chairs are here to say "this
 *  is a two-top, that is a big table" at a glance; drawing eleven of them makes
 *  a stripe, and drawing none for a counter slot is right — it has no chairs. */
function topSeats(seats?: number): number {
  if (!seats) return 0;
  return Math.min(4, Math.ceil(seats / 2));
}
function bottomSeats(seats?: number): number {
  if (!seats) return 0;
  return Math.min(4, Math.floor(seats / 2));
}

function Chairs({ count, state }: { count: number; state: TableState }) {
  // The row keeps its height at zero chairs so a counter slot lines up with the
  // tables beside it: a grid where the tiles sit at different heights reads as
  // a rendering fault, not as a difference in furniture.
  return (
    <span className="flex h-[0.7rem] w-full items-center justify-center gap-1.5">
      {Array.from({ length: count }, (_, i) => (
        <span
          key={i}
          className="h-[0.42rem] w-[1.6rem] rounded-full"
          style={{
            background:
              state === "free" ? "var(--line-strong)" : stateLine(state),
          }}
        />
      ))}
    </span>
  );
}
