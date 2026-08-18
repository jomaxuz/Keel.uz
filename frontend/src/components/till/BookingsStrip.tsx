"use client";

// Today's bookings, on the screen the room is run from.
//
// ⚠️ **A booking is agreed on the phone and then has to survive until seven in
// the evening.** It is written in the panel by whoever answered the call, and
// the person who needs it is standing at the till two hours later. A booked
// table that gets walked in on is a party turned away at the door of a
// restaurant that had a table for them — and nobody finds out until the guests
// with the booking arrive.
//
// ⚠️ **A strip, not a screen.** It is context for the room, not a job of its
// own: the next few bookings, in time order, and nothing else. Managing them is
// still the panel's work.

import { useEffect, useState } from "react";

import { api } from "@/lib/api";
import { formatTime } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import type { TillReservation } from "@/lib/types";

export default function BookingsStrip({ active }: { active: boolean }) {
  const t = useAdminT();
  const [rows, setRows] = useState<TillReservation[] | null>(null);

  useEffect(() => {
    if (!active) return;
    let alive = true;
    const load = () =>
      api
        .tillReservations()
        .then((r) => alive && setRows(r.reservations))
        // Silent: a waiter who cannot be told about bookings is better served
        // by the room they came here to look at than by an error about a strip
        // they did not ask for.
        .catch(() => {});
    void load();
    // Slow on purpose — bookings are made hours ahead, and this is the one
    // thing on the screen that does not change while somebody is looking at it.
    const timer = setInterval(load, 120_000);
    return () => {
      alive = false;
      clearInterval(timer);
    };
  }, [active]);

  // ⚠️ Nothing at all until there is something to say. A restaurant that takes
  // no bookings must not be given an empty box on the busiest screen it owns.
  if (!rows || rows.length === 0) return null;

  return (
    <div className="flex shrink-0 items-center gap-2 border-b border-line bg-[rgb(var(--till-accent-tint))] px-3 py-1.5">
      <span className="till-label shrink-0 text-[rgb(var(--till-accent-ink))]">
        {t.till.bookings} {rows.length}
      </span>
      <div className="no-scrollbar flex min-w-0 flex-1 gap-1.5 overflow-x-auto">
        {rows.slice(0, 8).map((b) => (
          <span
            key={b.id}
            className="flex shrink-0 items-center gap-1.5 rounded-[8px] border border-[rgb(var(--till-accent-line))] bg-surface px-2 py-1 text-[12px]"
            title={b.comment || undefined}
          >
            <span className="till-num font-bold">{formatTime(b.at)}</span>
            {b.tableNumber && (
              <span className="font-semibold">{b.tableNumber}</span>
            )}
            <span className="max-w-[8rem] truncate text-ink-muted">
              {b.name}
            </span>
            <span className="till-num text-ink-muted">· {b.guests}</span>
          </span>
        ))}
      </div>
    </div>
  );
}
