"use client";

import { useState } from "react";

import { useAdminT } from "@/lib/i18n/admin";
import type { FloorTable } from "@/lib/types";

/**
 * Opening a table.
 *
 * ⚠️ **Tables that already have a check are shown, not hidden.** A waiter
 * looking for table 7 and not finding it assumes the tablet is broken and opens
 * a counter check instead — which is how one party ends up on two bills. Shown
 * and disabled, it says the true thing: somebody is already serving it.
 *
 * ⚠️ **"No table" is a first-class choice, not a fallback.** Half the places
 * that would buy this sell over a counter, and a till that insists on a table
 * number is a till they cannot use.
 */
export default function NewCheckDialog({
  tables,
  initialTable,
  busy,
  onCancel,
  onOpen,
}: {
  tables: FloorTable[];
  /** The table the floor tap chose, if the dialog was opened from there. */
  initialTable?: string;
  busy: string[];
  onCancel: () => void;
  onOpen: (tableId: string, guests: number) => void;
}) {
  const t = useAdminT();
  // ⚠️ Pre-filled when the floor was tapped: that tap already answered "which
  // table", and asking again is the screen forgetting what it was just told.
  const [tableId, setTableId] = useState(initialTable ?? "");
  const [guests, setGuests] = useState(2);

  const active = tables.filter((tb) => tb.isActive);

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center">
      <div className="till-dialog w-full max-w-md p-4">
        <h2 className="font-display text-xl font-bold">{t.till.newCheck}</h2>

        <p className="mt-4 text-sm text-ink-muted">{t.till.selectTable}</p>
        <div className="mt-2 grid max-h-56 grid-cols-4 gap-2 overflow-y-auto sm:grid-cols-6">
          <button
            onClick={() => setTableId("")}
            className={`col-span-2 rounded-xl border px-2 py-3 text-sm ${
              tableId === ""
                ? "border-[rgb(var(--till-accent))] bg-[rgb(var(--till-accent-tint))] font-semibold text-[rgb(var(--till-accent-ink))]"
                : "border-line"
            }`}
          >
            {t.till.noTable}
          </button>
          {active.map((tb) => {
            const taken = busy.includes(tb.id);
            return (
              <button
                key={tb.id}
                disabled={taken}
                onClick={() => setTableId(tb.id)}
                className={`rounded-xl border py-3 text-sm ${
                  tableId === tb.id
                    ? "border-[rgb(var(--till-accent))] bg-[rgb(var(--till-accent-tint))] font-bold text-[rgb(var(--till-accent-ink))]"
                    : "border-line"
                } disabled:opacity-40`}
                title={tb.note || undefined}
              >
                {tb.number}
              </button>
            );
          })}
        </div>

        <label className="mt-4 block text-sm">
          <span className="text-ink-muted">{t.till.guests}</span>
          <input
            className="till-input mt-1"
            inputMode="numeric"
            value={guests || ""}
            onChange={(e) =>
              setGuests(Math.max(0, Number(e.target.value.replace(/\D/g, ""))))
            }
          />
        </label>

        <div className="mt-5 flex gap-2">
          <button className="till-btn flex-1" onClick={onCancel}>
            {t.till.back}
          </button>
          <button
            className="till-btn-primary flex-1"
            onClick={() => onOpen(tableId, guests)}
          >
            {t.till.open}
          </button>
        </div>
      </div>
    </div>
  );
}
