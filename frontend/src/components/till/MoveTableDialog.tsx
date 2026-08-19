"use client";

import { useAdminT } from "@/lib/i18n/admin";
import type { FloorTable } from "@/lib/types";

/**
 * Moving a party to another table.
 *
 * ⚠️ **The one thing this must get right is not letting two checks land on one
 * table.** The server refuses it, but a screen that offers an occupied table and
 * then reports a conflict has made the waiter do the work twice — so occupied
 * tables are shown and disabled, carrying the number of the check already on
 * them.
 *
 * ⚠️ **Shown, not hidden.** A waiter who cannot find table 7 concludes the
 * screen is wrong rather than that the table is taken, and the next thing they
 * try is opening a second check for the same party. Present and unusable says
 * the true thing.
 *
 * ⚠️ **"No table" stays available.** A party that moved to the counter, or a
 * check opened against the wrong table in the first place, both need it — and a
 * dialog that can only move sideways cannot undo the mistake that opened it.
 */
export default function MoveTableDialog({
  tables,
  busyTables,
  current,
  onCancel,
  onMove,
}: {
  tables: FloorTable[];
  busyTables: string[];
  /** Where the check is now, so it is not offered as a destination. */
  current: string;
  onCancel: () => void;
  onMove: (tableId: string) => void | Promise<void>;
}) {
  const t = useAdminT();
  const active = tables.filter((tb) => tb.isActive);

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center">
      <div className="till-dialog w-full max-w-md p-4">
        <h2 className="font-display text-xl font-bold">{t.till.moveTable}</h2>
        <p className="mt-1 text-sm text-ink-muted">{t.till.moveTableHint}</p>

        <div className="mt-4 grid max-h-72 grid-cols-4 gap-2 overflow-y-auto sm:grid-cols-5">
          {current !== "" && (
            <button
              onClick={() => void onMove("")}
              className="col-span-2 min-h-14 rounded-xl border border-line text-sm font-medium"
            >
              {t.till.noTable}
            </button>
          )}
          {active.map((tb) => {
            const taken = busyTables.includes(tb.id) && tb.id !== current;
            const here = tb.id === current;
            return (
              <button
                key={tb.id}
                disabled={taken || here}
                onClick={() => void onMove(tb.id)}
                className={`min-h-14 rounded-xl border font-display text-lg font-bold disabled:opacity-40 ${
                  here
                    ? "border-[rgb(var(--till-accent))] bg-[rgb(var(--till-accent-tint))] font-bold text-[rgb(var(--till-accent-ink))]"
                    : "border-line"
                }`}
              >
                {tb.number}
              </button>
            );
          })}
        </div>

        <button className="till-btn mt-5 w-full" onClick={onCancel}>
          {t.common.cancel}
        </button>
      </div>
    </div>
  );
}
