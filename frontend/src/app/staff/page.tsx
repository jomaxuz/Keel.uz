"use client";

// The employee's working screen: one big button, then the record it produces.
//
// The button is the whole app. Everything below it — today's times, the month
// grid, the wage — exists so the employee can check the restaurant's numbers
// against their own memory before pay day, which is the only reason a time
// clock is ever trusted.

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { ApiError, api } from "@/lib/api";
import { useStaff } from "@/lib/staff";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice, formatTime } from "@/lib/format";
import {
  formatDuration,
  payModeLabel,
  payPeriodLabel,
  payUnitLabel,
  shortDate,
  toDayKey,
} from "@/lib/attendance";
import AttendanceCalendar from "@/components/staff/AttendanceCalendar";
import GeoPermission from "@/components/GeoPermission";
import LangSwitch from "@/components/site/LangSwitch";
import ThemeToggle from "@/components/site/ThemeToggle";
import type { StaffDay, StaffReport, StaffTrend } from "@/lib/types";

type Tab = "today" | "calendar" | "pay";

export default function StaffHomePage() {
  const router = useRouter();
  const {
    staff,
    workplace,
    openShift,
    loading,
    logout,
    clock,
    position,
    geoError,
    distance,
    blocked,
    refresh,
    geoState,
    geoChecking,
  } = useStaff();
  const t = useAdminT();
  const { lang } = useI18n();

  const params = useSearchParams();
  // Code scanned from the branch screen. Held in state because clocking in
  // also needs a position, which may still be arriving when the page opens.
  const [code, setCode] = useState<string | null>(null);
  const [autoDone, setAutoDone] = useState(false);

  const [tab, setTab] = useState<Tab>("today");
  const [today, setToday] = useState<StaffDay | null>(null);
  const [report, setReport] = useState<StaffReport | null>(null);
  const [range, setRange] = useState<{ from: string; to: string } | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // Ticks once a minute so an open shift's duration keeps up with the clock.
  const [tick, setTick] = useState(0);

  const dur = useCallback(
    (m: number) => formatDuration(m, t.staff.hoursShort, t.staff.minutesShort),
    [t],
  );

  useEffect(() => {
    if (!loading && !staff) router.replace("/staff/login");
  }, [staff, loading, router]);

  useEffect(() => {
    const timer = setInterval(() => setTick((n) => n + 1), 60000);
    return () => clearInterval(timer);
  }, []);

  // Today is fetched on its own, so browsing the calendar to March cannot
  // empty the screen the employee actually clocks in on.
  const loadToday = useCallback(() => {
    if (!staff) return;
    const key = toDayKey(new Date());
    api
      .staffReport(key, key)
      .then((r) => setToday(r.days[0] ?? null))
      .catch(() => setToday(null));
  }, [staff]);

  const loadReport = useCallback(() => {
    if (!staff) return;
    api
      .staffReport(range?.from, range?.to)
      .then(setReport)
      .catch(() => setReport(null));
  }, [staff, range]);

  useEffect(loadToday, [loadToday, openShift]);
  useEffect(loadReport, [loadReport, openShift]);

  const punch = useCallback(
    async (action: "in" | "out", withCode?: string | null) => {
      setBusy(true);
      setError(null);
      try {
        await clock(action, withCode ?? undefined);
        // A code is good for one punch; keeping it around would let a second
        // tap reuse it inside its window.
        setCode(null);
        loadToday();
        loadReport();
      } catch (e) {
        setError(e instanceof ApiError ? e.message : t.staff.clockFailed);
      } finally {
        setBusy(false);
      }
    },
    [clock, loadToday, loadReport, t],
  );

  // How long the current shift has been running, live. `tick` is in the deps
  // on purpose: it is the only thing that changes as the shift runs.
  const openFor = useMemo(() => {
    if (!openShift) return 0;
    return Math.max(
      0,
      Math.round((Date.now() - new Date(openShift.in).getTime()) / 60000),
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [openShift, tick]);

  // The QR opens /staff?c=... in the phone's browser. Take the code out of the
  // address bar straight away: it must not survive into a bookmark, a shared
  // link or the back button.
  useEffect(() => {
    const c = params.get("c");
    if (!c) return;
    setCode(c);
    window.history.replaceState(null, "", "/staff");
  }, [params]);

  // Scanning is the whole gesture — the employee should not have to scan and
  // then also find a button. Fires once a position is in hand, and only once.
  useEffect(() => {
    if (!code || autoDone || !staff || busy) return;
    if (blocked !== null) return; // no fix yet, or too far away
    setAutoDone(true);
    void punch(openShift ? "out" : "in", code);
  }, [code, autoDone, staff, busy, blocked, openShift, punch]);

  if (loading || !staff) {
    return (
      <main className="flex min-h-dvh items-center justify-center">
        <p className="text-sm text-ink-muted">{t.common.loading}</p>
      </main>
    );
  }

  const clockedIn = !!openShift;
  const radius = workplace?.radiusM ?? 0;

  return (
    <main className="mx-auto max-w-lg px-4 pb-16 pt-6">
      <header className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <h1 className="truncate font-display text-xl font-bold">{staff.name}</h1>
          <p className="truncate text-xs text-ink-muted">
            {staff.position || `@${staff.username}`}
            {workplace && ` · ${workplace.name}`}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {/* The kitchen screen, reachable from the app the cook already has
              open. Not a separate login: the same staff account, and the branch
              comes from it either way. */}
          <button
            type="button"
            onClick={() => router.push("/staff/kitchen")}
            className="btn-ghost shrink-0 px-3 py-1.5 text-xs"
          >
            {t.kitchen.title}
          </button>
          <LangSwitch />
          <ThemeToggle />
          <button
            type="button"
            onClick={logout}
            className="btn-ghost shrink-0 px-3 py-1.5 text-xs"
          >
            {t.staff.logout}
          </button>
        </div>
      </header>

      {/* ---- The button ---- */}
      <section className="mt-5 rounded-3xl border border-line bg-surface p-4 shadow-card">
        <div className="flex items-center justify-between gap-2">
          <p className="text-sm font-semibold">{t.staff.todayTitle}</p>
          <span
            className={`badge ${
              clockedIn
                ? "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300"
                : "bg-ink/10 text-ink-muted"
            }`}
          >
            {clockedIn ? t.staff.onShiftFor(dur(openFor)) : t.staff.notClockedIn}
          </span>
        </div>

        {/* Where the employee is, relative to where they should be — or, when
            the browser has not been asked yet, the button that asks. A denied
            permission cannot be re-requested by the page, so GeoPermission
            shows the steps for this device instead of a dead end. */}
        <GeoPermission
          className="mt-3"
          state={geoState}
          checking={geoChecking}
          onRequest={async () => {
            await refresh();
            return null;
          }}
        >
          <div className="mt-3 rounded-2xl bg-ink/[0.03] p-3 text-xs">
            {!workplace ? (
              <p className="text-red-600">{t.staff.noWorkplace}</p>
            ) : geoError ? (
              <p className="text-red-600">
                {geoError === "unsupported"
                  ? t.staff.geoUnsupported
                  : t.staff.geoFailed}
              </p>
            ) : !position ? (
              <p className="text-ink-muted">{t.staff.locating}</p>
            ) : blocked === "too-far" ? (
              <p className="text-amber-700 dark:text-amber-300">
                {t.staff.tooFar(Math.round(distance ?? 0), radius)}
              </p>
            ) : (
              <p className="text-emerald-600">
                ✓ {t.staff.atWork}
                {distance !== null &&
                  ` · ${t.staff.punchDistance(Math.round(distance))}`}
              </p>
            )}
            {(geoError || blocked === "too-far") && (
              <button
                type="button"
                onClick={() => void refresh()}
                className="btn-ghost mt-2 w-full py-1.5 text-xs"
              >
                {t.staff.retryLocation}
              </button>
            )}
          </div>
        </GeoPermission>

        <button
          type="button"
          disabled={busy || blocked !== null}
          onClick={() => void punch(clockedIn ? "out" : "in", code)}
          className={`mt-3 w-full rounded-2xl py-5 text-lg font-bold text-white transition-opacity disabled:opacity-40 ${
            clockedIn ? "bg-ink" : "bg-brand"
          }`}
        >
          {busy ? "..." : clockedIn ? t.staff.clockOut : t.staff.clockIn}
        </button>
        <p className="mt-2 text-[11px] leading-relaxed text-ink-muted/80">
          {t.staff.geoRequired}
        </p>

        {code && !error && (
          <p className="mt-3 rounded-2xl bg-emerald-50 px-4 py-2 text-sm text-emerald-800 dark:bg-emerald-500/10 dark:text-emerald-300">
            {t.staff.codeScanned}
          </p>
        )}

        {error && <p className="mt-3 text-sm text-red-600">{error}</p>}
      </section>

      {/* Tabs */}
      <div className="mt-6 inline-flex rounded-full border border-line bg-ink/[0.03] p-1 text-sm font-semibold">
        {(
          [
            ["today", t.staff.tabToday],
            ["calendar", t.staff.tabCalendar],
            ["pay", t.staff.tabPay],
          ] as const
        ).map(([key, label]) => (
          <button
            key={key}
            type="button"
            onClick={() => setTab(key)}
            className={`rounded-full px-4 py-1.5 transition-colors ${
              tab === key ? "bg-brand text-white" : "text-ink-muted"
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      {/* ---- Today ---- */}
      {tab === "today" && (
        <section className="mt-4 space-y-3">
          <div className="rounded-3xl border border-line bg-surface p-4 shadow-card">
            <p className="text-xs uppercase tracking-wider text-ink-muted">
              {today && today.expected > 0
                ? t.staff.planToday(today.planStart, today.planEnd)
                : today
                  ? t.staff.dayOffToday
                  : t.staff.noPlanToday}
            </p>

            <div className="mt-3 grid grid-cols-2 gap-3">
              <div>
                <p className="text-xs text-ink-muted">{t.staff.shiftIn}</p>
                <p className="font-display text-2xl font-bold tabular-nums">
                  {today?.first || "—"}
                </p>
              </div>
              <div>
                <p className="text-xs text-ink-muted">{t.staff.shiftOut}</p>
                <p className="font-display text-2xl font-bold tabular-nums">
                  {today?.last || (clockedIn ? "—" : "—")}
                </p>
              </div>
            </div>

            <p className="mt-3 border-t border-line pt-3 text-sm">
              {t.staff.workedToday}:{" "}
              <span className="font-bold text-brand">
                {dur(today?.worked ?? 0)}
              </span>
              {today && today.expected > 0 && (
                <span className="text-ink-muted">
                  {" / "}
                  {dur(today.expected)}
                </span>
              )}
            </p>

            {today && today.sessions.length > 1 && (
              <ul className="mt-2 space-y-0.5 text-xs text-ink-muted">
                {today.sessions.map((s) => (
                  <li key={s.id} className="tabular-nums">
                    {formatTime(s.in)}
                    {" → "}
                    {s.out ? formatTime(s.out) : t.staff.stillOpen}
                  </li>
                ))}
              </ul>
            )}
          </div>

          {/* Trends. Each against its own previous window, so widening the
              calendar above cannot move them. */}
          <div className="rounded-3xl border border-line bg-surface p-4 shadow-card">
            <p className="text-sm font-semibold">{t.staff.trendsTitle}</p>
            <div className="mt-3 space-y-2">
              {(
                [
                  [t.staff.trendToday, report?.today],
                  [t.staff.trendWeek, report?.week],
                  [t.staff.trendMonth, report?.month],
                ] as const
              ).map(([label, trend]) => (
                <TrendRow
                  key={label}
                  label={label}
                  trend={trend}
                  dur={dur}
                  t={t}
                />
              ))}
            </div>
          </div>
        </section>
      )}

      {/* ---- Calendar ---- */}
      {tab === "calendar" && (
        <section className="mt-4 space-y-4">
          <RangePicker range={range} setRange={setRange} t={t} />

          {report && (
            <div className="grid grid-cols-2 gap-3">
              <Stat label={t.staff.totalWorked} value={dur(report.totals.worked)} />
              <Stat
                label={t.staff.totalExpected}
                value={dur(report.totals.expected)}
              />
              <Stat
                label={t.staff.daysWorked}
                value={String(report.totals.days)}
              />
              <Stat
                label={t.staff.daysAbsent}
                value={String(report.totals.absent)}
                tone={report.totals.absent > 0 ? "bad" : undefined}
              />
              <Stat
                label={t.staff.overtime}
                value={dur(report.totals.overtime)}
                tone="up"
              />
              <Stat
                label={t.staff.shortage}
                value={dur(report.totals.shortage)}
                tone={report.totals.shortage > 0 ? "warn" : undefined}
              />
            </div>
          )}

          <AttendanceCalendar days={report?.days ?? []} />
        </section>
      )}

      {/* ---- Pay ---- */}
      {tab === "pay" && report && (
        <section className="mt-4 space-y-3">
          <div className="rounded-3xl border border-line bg-surface p-4 shadow-card">
            <p className="text-xs uppercase tracking-wider text-ink-muted">
              {t.staff.payMode}
            </p>
            <p className="mt-1 font-semibold">
              {payModeLabel(report.payMode, t.staff)} ·{" "}
              <span className="text-brand">
                {formatPrice(report.rate, "UZS", lang)}
              </span>{" "}
              <span className="text-ink-muted">
                {payUnitLabel(report.payMode, t.staff)}
              </span>
            </p>
            <p className="mt-1 text-xs text-ink-muted">
              {t.staff.payPeriod}: {payPeriodLabel(report.payPeriod, t.staff)}
            </p>
          </div>

          <div className="rounded-3xl border border-line bg-surface p-4 shadow-card">
            <p className="text-xs uppercase tracking-wider text-ink-muted">
              {t.staff.currentPeriod} ·{" "}
              {t.staff.periodRange(
                shortDate(report.periodFrom),
                shortDate(report.periodTo),
              )}
            </p>
            <dl className="mt-3 space-y-2 text-sm">
              <Row
                label={t.staff.periodEarned}
                value={formatPrice(report.periodPay, "UZS", lang)}
              />
              <Row
                label={t.staff.periodPaid}
                value={formatPrice(report.periodPaid, "UZS", lang)}
              />
              <div className="flex items-center justify-between border-t border-line pt-2">
                <dt className="font-semibold">
                  {report.periodDue < 0 ? t.staff.overpaid : t.staff.periodDue}
                </dt>
                <dd className="font-display text-xl font-bold text-brand">
                  {formatPrice(Math.abs(report.periodDue), "UZS", lang)}
                </dd>
              </div>
            </dl>
            {report.periodDue === 0 && report.periodPaid > 0 && (
              <p className="mt-3 rounded-2xl bg-emerald-50 px-4 py-2 text-sm text-emerald-800 dark:bg-emerald-500/10 dark:text-emerald-300">
                {t.staff.settled}
              </p>
            )}
          </div>

          {/* What the browsed range is worth, so the employee can check any
              stretch of days rather than only the current period. */}
          <div className="rounded-3xl border border-line bg-surface p-4 shadow-card">
            <p className="text-xs uppercase tracking-wider text-ink-muted">
              {t.staff.periodRange(
                shortDate(report.from),
                shortDate(report.to),
              )}
            </p>
            <dl className="mt-3 space-y-2 text-sm">
              <Row label={t.staff.totalWorked} value={dur(report.totals.worked)} />
              <Row
                label={t.staff.daysWorked}
                value={String(report.totals.days)}
              />
              <div className="flex items-center justify-between border-t border-line pt-2">
                <dt className="font-semibold">{t.staff.periodEarned}</dt>
                <dd className="font-display text-xl font-bold text-brand">
                  {formatPrice(report.totals.pay, "UZS", lang)}
                </dd>
              </div>
            </dl>
          </div>
        </section>
      )}
    </main>
  );
}

// ---- small pieces ----

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between">
      <dt className="text-ink-muted">{label}</dt>
      <dd className="tabular-nums">{value}</dd>
    </div>
  );
}

function Stat({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: "up" | "warn" | "bad";
}) {
  const color =
    tone === "up"
      ? "text-sky-600"
      : tone === "warn"
        ? "text-amber-600"
        : tone === "bad"
          ? "text-red-600"
          : "text-ink";
  return (
    <div className="rounded-2xl border border-line bg-surface p-3 shadow-card">
      <p className="text-[11px] uppercase tracking-wider text-ink-muted">{label}</p>
      <p className={`mt-0.5 font-display text-lg font-bold tabular-nums ${color}`}>
        {value}
      </p>
    </div>
  );
}

function TrendRow({
  label,
  trend,
  dur,
  t,
}: {
  label: string;
  trend: StaffTrend | undefined;
  dur: (m: number) => string;
  t: ReturnType<typeof useAdminT>;
}) {
  const up = (trend?.percent ?? 0) > 0;
  return (
    <div className="flex items-baseline justify-between gap-2 text-sm">
      <span className="text-ink-muted">{label}</span>
      <span className="text-right">
        <span className="tabular-nums font-semibold">{dur(trend?.current ?? 0)}</span>
        <span
          className={`ml-2 text-xs ${
            !trend?.hasPrev
              ? "text-ink-muted/70"
              : trend.percent === 0
                ? "text-ink-muted"
                : up
                  ? "text-sky-600"
                  : "text-amber-600"
          }`}
        >
          {!trend?.hasPrev
            ? t.staff.trendNoPrev
            : trend.percent === 0
              ? t.staff.trendSame
              : up
                ? t.staff.trendUp(String(trend.percent))
                : t.staff.trendDown(String(Math.abs(trend.percent)))}
        </span>
      </span>
    </div>
  );
}

/** Presets plus two date inputs. `null` means "let the server decide", which is
 *  this calendar month — the client must not compute it, or the two disagree at
 *  midnight on the first. */
function RangePicker({
  range,
  setRange,
  t,
}: {
  range: { from: string; to: string } | null;
  setRange: (r: { from: string; to: string } | null) => void;
  t: ReturnType<typeof useAdminT>;
}) {
  function preset(kind: "month" | "last" | "d7" | "d30") {
    const now = new Date();
    if (kind === "month") return setRange(null);
    if (kind === "last") {
      const first = new Date(now.getFullYear(), now.getMonth() - 1, 1);
      const last = new Date(now.getFullYear(), now.getMonth(), 0);
      return setRange({ from: toDayKey(first), to: toDayKey(last) });
    }
    const days = kind === "d7" ? 6 : 29;
    const from = new Date(now);
    from.setDate(from.getDate() - days);
    setRange({ from: toDayKey(from), to: toDayKey(now) });
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-1.5">
        {(
          [
            ["month", t.staff.rangeThisMonth],
            ["last", t.staff.rangeLastMonth],
            ["d7", t.staff.range7],
            ["d30", t.staff.range30],
          ] as const
        ).map(([kind, label]) => (
          <button
            key={kind}
            type="button"
            onClick={() => preset(kind)}
            className="chip"
          >
            {label}
          </button>
        ))}
      </div>
      <div className="flex flex-wrap items-center gap-2 text-xs">
        <input
          type="date"
          className="input py-1.5 text-xs"
          value={range?.from ?? ""}
          onChange={(e) =>
            setRange({
              from: e.target.value,
              to: range?.to || e.target.value,
            })
          }
        />
        <span className="text-ink-muted">—</span>
        <input
          type="date"
          className="input py-1.5 text-xs"
          value={range?.to ?? ""}
          onChange={(e) =>
            setRange({
              from: range?.from || e.target.value,
              to: e.target.value,
            })
          }
        />
      </div>
    </div>
  );
}
