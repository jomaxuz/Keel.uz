"use client";

// The attendance calendar, shared by the employee's own app and the admin's
// staff card.
//
// A month grid rather than a list because the question people actually ask is
// shaped like a month: "which days did he not come in?" is answered by looking
// for the red squares, not by reading twenty rows.

import { useMemo, useState } from "react";
import { useI18n } from "@/lib/i18n/client";
import { useAdminT } from "@/lib/i18n/admin";
import { formatTime, weekdayName } from "@/lib/format";
import {
  MONDAY_ORDER,
  STATUS_DOT,
  STATUS_ORDER,
  STATUS_TONE,
  formatDuration,
  mondayIndex,
  parseDayKey,
} from "@/lib/attendance";
import type { StaffDay } from "@/lib/types";

interface Props {
  days: StaffDay[];
  /** Rendered under the selected day — the admin card hangs its edit buttons
   *  here, the employee's app shows nothing. */
  renderDayExtra?: (day: StaffDay) => React.ReactNode;
}

export default function AttendanceCalendar({ days, renderDayExtra }: Props) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [picked, setPicked] = useState<string | null>(null);

  const dur = (m: number) =>
    formatDuration(m, t.staff.hoursShort, t.staff.minutesShort);

  // Split into months, so a range that spans a boundary draws two grids rather
  // than one nonsensical one.
  const months = useMemo(() => {
    const out: { key: string; label: string; days: StaffDay[] }[] = [];
    for (const day of days) {
      const date = parseDayKey(day.date);
      const key = `${date.getFullYear()}-${date.getMonth()}`;
      const last = out[out.length - 1];
      if (last?.key === key) {
        last.days.push(day);
      } else {
        out.push({
          key,
          label: `${t.staff.months[date.getMonth()]} ${date.getFullYear()}`,
          days: [day],
        });
      }
    }
    return out;
  }, [days, t]);

  const selected = days.find((d) => d.date === picked) ?? null;

  if (days.length === 0) {
    return (
      <p className="rounded-3xl border border-dashed border-line-strong p-6 text-center text-sm text-ink-muted">
        {t.staff.noData}
      </p>
    );
  }

  return (
    // Capped rather than fluid: the squares are `aspect-square`, so a full-width
    // grid on a desktop panel draws seven 150px tiles and pushes everything else
    // below the fold.
    <div className="max-w-md space-y-6">
      {months.map((month) => (
        <div key={month.key}>
          <p className="mb-2 text-sm font-semibold capitalize">{month.label}</p>

          <div className="grid grid-cols-7 gap-1 sm:gap-1.5">
            {MONDAY_ORDER.map((wd) => (
              <div
                key={wd}
                className="pb-1 text-center text-[11px] font-medium uppercase tracking-wide text-ink-muted"
              >
                {weekdayName(wd, lang).slice(0, 2)}
              </div>
            ))}

            {/* Blanks so the 1st lands under the right weekday. */}
            {Array.from({ length: mondayIndex(month.days[0].weekday) }).map(
              (_, i) => (
                <div key={`pad-${i}`} />
              ),
            )}

            {month.days.map((day) => {
              const isPicked = day.date === picked;
              return (
                <button
                  key={day.date}
                  type="button"
                  onClick={() => setPicked(isPicked ? null : day.date)}
                  title={`${day.date} · ${t.staff.statuses[day.status]}`}
                  className={`flex aspect-square flex-col items-center justify-center rounded-xl border px-0.5 text-center transition-all ${
                    STATUS_TONE[day.status]
                  } ${isPicked ? "ring-2 ring-brand ring-offset-1 ring-offset-cream" : ""}`}
                >
                  <span className="text-xs font-bold tabular-nums leading-none">
                    {parseDayKey(day.date).getDate()}
                  </span>
                  {day.worked > 0 && (
                    <span className="mt-0.5 text-[10px] leading-none tabular-nums opacity-80">
                      {(day.worked / 60).toFixed(1)}
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        </div>
      ))}

      {/* What the colours mean. Without it the grid is decoration. */}
      <div className="flex flex-wrap gap-x-4 gap-y-1.5 rounded-2xl bg-ink/[0.03] px-4 py-3 text-xs text-ink-muted">
        <span className="font-medium text-ink">{t.staff.legend}:</span>
        {STATUS_ORDER.map((status) => (
          <span key={status} className="inline-flex items-center gap-1.5">
            <span className={`h-2 w-2 rounded-full ${STATUS_DOT[status]}`} />
            {t.staff.statuses[status]}
          </span>
        ))}
      </div>

      {selected && (
        <div className="rounded-3xl border border-line bg-surface p-4 shadow-card">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-semibold capitalize">
              {(() => {
                const d = parseDayKey(selected.date);
                return `${weekdayName(d.getDay(), lang)}, ${d.getDate()} ${t.staff.months[d.getMonth()]}`;
              })()}
            </span>
            <span
              className={`badge border ${STATUS_TONE[selected.status]}`}
            >
              {t.staff.statuses[selected.status]}
            </span>
          </div>

          <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-1.5 text-sm sm:grid-cols-3">
            <div>
              <dt className="text-xs text-ink-muted">{t.staff.plan}</dt>
              <dd className="tabular-nums">
                {selected.expected > 0
                  ? `${selected.planStart} — ${selected.planEnd} (${dur(selected.expected)})`
                  : t.staff.statuses.off}
              </dd>
            </div>
            <div>
              <dt className="text-xs text-ink-muted">{t.staff.fact}</dt>
              <dd className="tabular-nums">
                {selected.worked > 0
                  ? `${selected.first} — ${selected.last || t.staff.stillOpen} (${dur(selected.worked)})`
                  : "—"}
              </dd>
            </div>
            {(selected.expected > 0 || selected.worked > 0) && (
              <div>
                <dt className="text-xs text-ink-muted">
                  {selected.diff >= 0 ? t.staff.overtime : t.staff.shortage}
                </dt>
                <dd
                  className={`tabular-nums font-semibold ${
                    selected.diff >= 0 ? "text-sky-600" : "text-amber-600"
                  }`}
                >
                  {dur(Math.abs(selected.diff))}
                </dd>
              </div>
            )}
          </dl>

          {selected.sessions.length > 0 && (
            <ul className="mt-3 space-y-1 border-t border-line pt-3 text-xs text-ink-muted">
              {selected.sessions.map((s) => (
                <li key={s.id} className="tabular-nums">
                  {formatTime(s.in)}
                  {" → "}
                  {s.out ? formatTime(s.out) : t.staff.stillOpen}
                  {" · "}
                  {dur(s.minutes)}
                  {s.editedBy && (
                    <span className="ml-1.5 text-amber-600 dark:text-amber-400">
                      ({t.staff.byAdmin(s.editedBy)})
                    </span>
                  )}
                  {s.note && <span className="ml-1.5">✎ {s.note}</span>}
                </li>
              ))}
            </ul>
          )}

          {renderDayExtra?.(selected)}
        </div>
      )}
    </div>
  );
}
