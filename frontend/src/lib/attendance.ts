// Shared vocabulary for everything that reads a shift: the employee's app, the
// admin's staff card and the payroll screen. Kept in one place so a day that is
// amber on one screen is never green on another.

import type { StaffDayStatus, StaffPayMode, StaffPayPeriod } from "./types";

/** Minutes as "8 s 30 d". Labels come from the dictionary, so the same helper
 *  serves all three languages. */
export function formatDuration(
  minutes: number,
  hourLabel: string,
  minuteLabel: string,
): string {
  const sign = minutes < 0 ? "-" : "";
  const total = Math.abs(Math.round(minutes));
  const h = Math.floor(total / 60);
  const m = total % 60;
  if (h === 0) return `${sign}${m} ${minuteLabel}`;
  if (m === 0) return `${sign}${h} ${hourLabel}`;
  return `${sign}${h} ${hourLabel} ${m} ${minuteLabel}`;
}

/** Colour for a calendar square. One tone per meaning, not per feeling:
 *  green is "as rostered", and both directions away from it are coloured. */
export const STATUS_TONE: Record<StaffDayStatus, string> = {
  ok: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300 border-emerald-500/40",
  over: "bg-sky-500/15 text-sky-700 dark:text-sky-300 border-sky-500/40",
  under: "bg-amber-500/15 text-amber-700 dark:text-amber-300 border-amber-500/40",
  absent: "bg-red-500/15 text-red-700 dark:text-red-300 border-red-500/40",
  extra: "bg-violet-500/15 text-violet-700 dark:text-violet-300 border-violet-500/40",
  open: "bg-brand text-white border-brand",
  off: "bg-ink/[0.04] text-ink-muted border-line",
  upcoming: "bg-transparent text-ink-muted/70 border-dashed border-line-strong",
};

/** The same tone as a small dot, for legends and table cells. */
export const STATUS_DOT: Record<StaffDayStatus, string> = {
  ok: "bg-emerald-500",
  over: "bg-sky-500",
  under: "bg-amber-500",
  absent: "bg-red-500",
  extra: "bg-violet-500",
  open: "bg-brand",
  off: "bg-ink/20",
  upcoming: "bg-ink/10",
};

/** Order the legend reads in: what happened, then what did not. */
export const STATUS_ORDER: StaffDayStatus[] = [
  "open",
  "ok",
  "over",
  "under",
  "extra",
  "absent",
  "off",
  "upcoming",
];

/** "2026-08-01" → a local Date. Parsed by hand rather than with `new Date(s)`,
 *  which reads a bare date as UTC and shifts the whole calendar a day west. */
export function parseDayKey(key: string): Date {
  const [y, m, d] = key.split("-").map(Number);
  return new Date(y, (m ?? 1) - 1, d ?? 1);
}

export function toDayKey(date: Date): string {
  const p = (n: number) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${p(date.getMonth() + 1)}-${p(date.getDate())}`;
}

/** "2026-08-01" → "01.08".
 *
 *  Deliberately numeric rather than `toLocaleDateString`: the uz-UZ short month
 *  renders as "M08", so "M08 1 — M08 10" is what a pay period looked like. A
 *  numeric day.month reads the same in all three languages. */
export function shortDate(key: string): string {
  const d = parseDayKey(key);
  const p = (n: number) => String(n).padStart(2, "0");
  return `${p(d.getDate())}.${p(d.getMonth() + 1)}`;
}

/** The calendar week starts on Monday here — Uzbekistan reads it that way,
 *  while `Date.getDay()` counts from Sunday. */
export function mondayIndex(weekday: number): number {
  return (weekday + 6) % 7;
}

export const MONDAY_ORDER = [1, 2, 3, 4, 5, 6, 0];

interface PayLabels {
  payHourly: string;
  payShift: string;
  payMonthly: string;
  periodDaily: string;
  period10: string;
  period15: string;
  periodMonthly: string;
  perHour: string;
  perShift: string;
  perMonth: string;
}

export function payModeLabel(mode: StaffPayMode, t: PayLabels): string {
  if (mode === "shift") return t.payShift;
  if (mode === "monthly") return t.payMonthly;
  return t.payHourly;
}

/** "soatiga" / "smenasiga" / "oyiga" — what the rate is *per*. */
export function payUnitLabel(mode: StaffPayMode, t: PayLabels): string {
  if (mode === "shift") return t.perShift;
  if (mode === "monthly") return t.perMonth;
  return t.perHour;
}

export function payPeriodLabel(period: StaffPayPeriod, t: PayLabels): string {
  if (period === "daily") return t.periodDaily;
  if (period === "10days") return t.period10;
  if (period === "15days") return t.period15;
  return t.periodMonthly;
}
