"use client";

// "Order now, for later" — the guest's half.
//
// Two decisions shape this control:
//
//   • **Slots, not a datetime field.** `<input type="datetime-local">` would be
//     one line of code and would let a guest ask for 03:40 on a day the
//     restaurant is shut, only to be refused after they had filled in
//     everything else. The times offered here are ones the branch is open for
//     and far enough ahead to cook — so the only thing left to refuse is a race,
//     and the server still checks (this component is a convenience, never the
//     rule; see resolvePreorder).
//
//   • **"As soon as possible" is the default and stays one tap away.** Almost
//     every order is that one. A picker that opens on a date makes the ordinary
//     case pay for the rare one.

import { useEffect, useMemo } from "react";
import { formatTime, weekdayName } from "@/lib/format";
import type { Dict } from "@/lib/i18n";
import type { Lang } from "@/lib/i18n";
import type { PreorderSettings, WorkingHour } from "@/lib/types";

const MINUTE = 60_000;

/** A day the guest may pick, with the slots that day actually offers. */
interface DayOption {
  /** Local "YYYY-MM-DD" — the value carried in the day selector. */
  key: string;
  label: string;
  slots: Date[];
}

const dayKey = (d: Date) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(
    d.getDate(),
  ).padStart(2, "0")}`;

/** "13:30" → minutes since midnight. */
function hhmm(v: string): number | null {
  const m = /^(\d{1,2}):(\d{2})$/.exec(v.trim());
  if (!m) return null;
  return Number(m[1]) * 60 + Number(m[2]);
}

/**
 * The slots one calendar day offers.
 *
 * ⚠️ An overnight shift (18:00–02:00) is read the way the rest of the app reads
 * it: the entry belongs to the day it *opens* on, and its tail runs into the
 * next morning. Generating slots only up to `close` would silently drop every
 * late-evening restaurant's whole booking window.
 */
function slotsFor(
  day: Date,
  hours: WorkingHour[],
  slotMinutes: number,
  earliest: Date,
  latest: Date,
): Date[] {
  const rule = hours.find((h) => h.day === day.getDay());
  if (!rule || rule.isClosed) return [];
  const open = hhmm(rule.open);
  const close = hhmm(rule.close);
  if (open === null || close === null) return [];

  const midnight = new Date(day);
  midnight.setHours(0, 0, 0, 0);
  // An overnight close wraps past midnight; a same-day one does not.
  const end = close < open ? close + 24 * 60 : close;

  const out: Date[] = [];
  // Start on a clean boundary: a list reading 13:07, 13:37 looks like a bug
  // even when the arithmetic is right.
  const first = Math.ceil(open / slotMinutes) * slotMinutes;
  for (let m = first; m <= end; m += slotMinutes) {
    const at = new Date(midnight.getTime() + m * MINUTE);
    if (at < earliest || at > latest) continue;
    out.push(at);
  }
  return out;
}

export default function PreorderPicker({
  settings,
  workingHours,
  value,
  onChange,
  t,
  lang,
}: {
  settings: PreorderSettings;
  workingHours: WorkingHour[];
  /** The chosen instant, or null for "as soon as possible". */
  value: Date | null;
  onChange: (next: Date | null) => void;
  t: Dict;
  lang: Lang;
}) {
  const tp = t.checkout.preorder;

  // Recomputed from the clock rather than held in state: this form is often
  // open for several minutes, and a slot that has drifted into the past while
  // the guest typed their address would be offered and then refused.
  const days = useMemo<DayOption[]>(() => {
    const now = new Date();
    const earliest = new Date(now.getTime() + settings.minMinutes * MINUTE);
    const latest = new Date(now);
    latest.setHours(23, 59, 59, 999);
    latest.setDate(latest.getDate() + settings.maxDays);

    const out: DayOption[] = [];
    for (let i = 0; i <= settings.maxDays; i++) {
      const day = new Date(now);
      day.setDate(day.getDate() + i);
      const slots = slotsFor(
        day,
        workingHours,
        settings.slotMinutes,
        earliest,
        latest,
      );
      // A day with nothing on it is left out entirely. An empty option in the
      // list teaches that the control is decorative.
      if (slots.length === 0) continue;
      out.push({
        key: dayKey(day),
        label:
          i === 0
            ? tp.today
            : i === 1
              ? tp.tomorrow
              : `${weekdayName(day.getDay(), lang)}, ${day.getDate()}.${String(
                  day.getMonth() + 1,
                ).padStart(2, "0")}`,
        slots,
      });
    }
    return out;
  }, [settings, workingHours, tp, lang]);

  const selectedDay = value ? dayKey(value) : days[0]?.key;
  const daySlots = days.find((d) => d.key === selectedDay)?.slots ?? [];

  // The restaurant offers pre-orders but has no slot inside its own window —
  // shut all week, or a notice longer than the day is left. Nothing is drawn:
  // an empty picker is worse than no picker, because it looks broken.
  useEffect(() => {
    if (days.length === 0 && value) onChange(null);
  }, [days.length, value, onChange]);
  if (days.length === 0) return null;

  return (
    <div className="mt-5 border-t border-line pt-5">
      <h3 className="text-sm font-semibold">{tp.title}</h3>
      <div className="mt-3 flex gap-3">
        <label className="flex flex-1 cursor-pointer items-center gap-2 rounded-xl border border-line-strong px-4 py-3 text-sm has-[:checked]:border-brand has-[:checked]:bg-brand/5">
          <input
            type="radio"
            checked={value === null}
            onChange={() => onChange(null)}
          />
          <span className="font-medium">{tp.asap}</span>
        </label>
        <label className="flex flex-1 cursor-pointer items-center gap-2 rounded-xl border border-line-strong px-4 py-3 text-sm has-[:checked]:border-brand has-[:checked]:bg-brand/5">
          <input
            type="radio"
            checked={value !== null}
            // Opening straight onto the first free slot: a control that starts
            // empty asks the guest to make two choices to express one.
            onChange={() => onChange(days[0].slots[0] ?? null)}
          />
          <span className="font-medium">{tp.later}</span>
        </label>
      </div>

      {value !== null && (
        <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
          <label className="block text-sm">
            <span className="font-medium">{tp.day}</span>
            <select
              className="input mt-1"
              value={selectedDay}
              onChange={(e) => {
                const day = days.find((d) => d.key === e.target.value);
                // Moving to another day keeps a time only if that day has it;
                // otherwise the first slot, so the control is never in a state
                // the server would refuse.
                onChange(day?.slots[0] ?? null);
              }}
            >
              {days.map((d) => (
                <option key={d.key} value={d.key}>
                  {d.label}
                </option>
              ))}
            </select>
          </label>
          <label className="block text-sm">
            <span className="font-medium">{tp.time}</span>
            <select
              className="input mt-1"
              value={value ? String(value.getTime()) : ""}
              onChange={(e) => onChange(new Date(Number(e.target.value)))}
            >
              {daySlots.length === 0 && <option value="">{tp.noSlots}</option>}
              {daySlots.map((s) => (
                <option key={s.getTime()} value={s.getTime()}>
                  {formatTime(s)}
                </option>
              ))}
            </select>
          </label>
        </div>
      )}

      <p className="mt-2 text-xs text-ink-muted/70">
        {tp.hint(settings.minMinutes)}
      </p>
    </div>
  );
}
