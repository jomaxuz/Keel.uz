"use client";

// The roster: from what time to what time this employee works, per weekday.
//
// This is the input the whole attendance feature is measured against. Without
// it every day looks identical and "worked more than they should have" has no
// meaning — so the form says so out loud when it is left empty.

import { useI18n } from "@/lib/i18n/client";
import { useAdminT } from "@/lib/i18n/admin";
import { weekdayName } from "@/lib/format";
import { MONDAY_ORDER } from "@/lib/attendance";
import type { StaffSchedule } from "@/lib/types";

interface Props {
  value: StaffSchedule[];
  onChange: (next: StaffSchedule[]) => void;
}

/** A weekday's row, defaulting to a day off when the admin never filled it in.
 *  Same rule as the server: an empty roster accuses nobody of absence. */
function rowFor(rows: StaffSchedule[], day: number): StaffSchedule {
  return (
    rows.find((r) => r.day === day) ?? {
      day,
      start: "",
      end: "",
      isOff: true,
    }
  );
}

export default function ScheduleEditor({ value, onChange }: Props) {
  const t = useAdminT();
  const { lang } = useI18n();

  function set(day: number, patch: Partial<StaffSchedule>) {
    const next = MONDAY_ORDER.map((d) => {
      const row = rowFor(value, d);
      return d === day ? { ...row, ...patch } : row;
    });
    onChange(next.sort((a, b) => a.day - b.day));
  }

  /** Most restaurants roster the same hours every open day; typing them seven
   *  times is the kind of chore that makes an owner skip the feature. */
  function copyToAll(from: StaffSchedule) {
    onChange(
      MONDAY_ORDER.map((d) => {
        const row = rowFor(value, d);
        return row.isOff
          ? row
          : { ...row, start: from.start, end: from.end, isOff: false };
      }).sort((a, b) => a.day - b.day),
    );
  }

  const anyWorking = MONDAY_ORDER.some((d) => {
    const row = rowFor(value, d);
    return !row.isOff && row.start && row.end;
  });

  return (
    <div>
      <p className="text-sm font-medium">{t.staff.scheduleTitle}</p>
      <p className="mt-0.5 text-xs text-ink-muted">{t.staff.scheduleHint}</p>

      <div className="mt-2 space-y-1.5">
        {MONDAY_ORDER.map((day) => {
          const row = rowFor(value, day);
          return (
            <div key={day} className="flex flex-wrap items-center gap-2">
              <span className="w-24 shrink-0 text-sm capitalize">
                {weekdayName(day, lang)}
              </span>

              <label className="flex items-center gap-1.5 text-xs text-ink-muted">
                <input
                  type="checkbox"
                  checked={row.isOff}
                  onChange={(e) =>
                    set(day, {
                      isOff: e.target.checked,
                      ...(e.target.checked ? {} : { start: row.start || "09:00", end: row.end || "18:00" }),
                    })
                  }
                />
                {t.staff.dayOff}
              </label>

              {!row.isOff && (
                <>
                  <input
                    type="time"
                    className="rounded-xl border border-line-strong bg-surface px-2 py-1 text-sm outline-none focus:border-brand"
                    value={row.start}
                    onChange={(e) => set(day, { start: e.target.value })}
                  />
                  <span className="text-ink-muted">—</span>
                  <input
                    type="time"
                    className="rounded-xl border border-line-strong bg-surface px-2 py-1 text-sm outline-none focus:border-brand"
                    value={row.end}
                    onChange={(e) => set(day, { end: e.target.value })}
                  />
                  {row.start && row.end && (
                    <button
                      type="button"
                      onClick={() => copyToAll(row)}
                      className="text-xs text-brand hover:underline"
                    >
                      {t.staff.copyToAll}
                    </button>
                  )}
                </>
              )}
            </div>
          );
        })}
      </div>

      {!anyWorking && (
        <p className="mt-2 rounded-xl bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
          {t.staff.scheduleEmpty}
        </p>
      )}
    </div>
  );
}
