"use client";

// One customer's card: what is happening at that restaurant right now.
//
// Loaded separately from the tenant record on purpose. The record comes out of
// the control plane's own collection instantly; this dials the customer's
// database and can be slow, or hang, or fail — a stopped container, a tenant
// mid-migration. Folded together, any of those would take down the page that
// shows the container status, which is the page somebody opens *because*
// something is wrong. Split, the card renders immediately and the figures
// arrive after, or do not, with a sentence saying why.
//
// Form choices, in the order the skill's procedure asks them:
//
//   • **Today's numbers are stat tiles, not charts.** One number per question,
//     with yesterday beside it. A four-bar chart of "today vs yesterday" says
//     less than the two numbers do.
//   • **Two charts over time, never one with two axes.** Counts and so'm are
//     different scales; drawing them on one frame invents a crossing point that
//     means nothing.
//   • **Orders and cancellations share a frame** because they are the same
//     unit and the comparison is the point.
//   • **Couriers and staff are counters, not a pie.** "Three free, one busy"
//     is read faster as three words than as a wheel.
//
// Colours are the validated default categorical slots plus the reserved
// critical red for cancellations. Cancelled is a *status*, not a series, so it
// never takes a categorical slot — nothing else on the page may use that red.

import { useEffect, useState } from "react";
import { useT } from "@/lib/i18n/client";
import { tenantLive, money, dayLabel, type TenantDay, type TenantLive } from "@/lib/api";

// Validated against both Keel surfaces (#ffffff / #0A1A28) with the skill's
// six checks: adjacent CVD ΔE 9.2 light / 9.4 dark, normal-vision 27.6 / 26.5,
// contrast ≥ 3:1 everywhere except light-mode aqua — which is why the type
// split ships direct labels rather than relying on the swatch alone.
const C = {
  orders: "var(--viz-1)",
  cancelled: "var(--viz-bad)",
  delivery: "var(--viz-1)",
  pickup: "var(--viz-2)",
  dinein: "var(--viz-3)",
};

export default function TenantInsights({
  tenantId,
  days,
}: {
  tenantId: string;
  /** The nightly rows the control plane already holds — history, as opposed to
   *  the live snapshot below. */
  days: TenantDay[];
}) {
  const { t } = useT();
  const [live, setLive] = useState<TenantLive | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    tenantLive(tenantId)
      .then(setLive)
      .catch(() => setFailed(true));
  }, [tenantId]);

  const recent = days.slice(-30);
  // Summed over the window the charts already show, so the sentence below and
  // the bars above are talking about the same thirty days.
  const reversed = recent.reduce((n, d) => n + (d.reversed ?? 0), 0);
  const cancelledCooked = recent.reduce((n, d) => n + (d.cancelledCooked ?? 0), 0);

  return (
    <div className="space-y-6">
      <VizTokens />

      {/* ---- Today ---- */}
      <section className="card">
        <div className="flex flex-wrap items-baseline justify-between gap-2">
          <p className="text-sm font-semibold text-ink">{t.dash.liveTitle}</p>
          {live && (
            <span className="text-xs text-ink-muted">
              {t.dash.liveAt(new Date(live.collectedAt).toLocaleTimeString("ru-RU", {
                hour: "2-digit",
                minute: "2-digit",
              }))}
            </span>
          )}
        </div>

        {/* A stopped or unreadable tenant renders zeroes, and zeroes look like
            a quiet Tuesday. Say which it is. */}
        {failed && (
          <p className="mt-3 rounded-xl bg-amber-500/10 px-3 py-2 text-sm text-ink-soft">
            {t.dash.liveFailed}
          </p>
        )}
        {live?.error && (
          <p className="mt-3 rounded-xl bg-amber-500/10 px-3 py-2 text-sm text-ink-soft">
            {live.error}
          </p>
        )}
        {!live && !failed && (
          <p className="mt-3 text-sm text-ink-muted">{t.dash.liveLoading}</p>
        )}

        {live && (
          <>
            <div className="mt-4 grid gap-px overflow-hidden rounded-2xl border border-line bg-line sm:grid-cols-2 lg:grid-cols-4">
              <Tile
                label={t.dash.liveOrders}
                value={String(live.today.orders)}
                delta={delta(live.today.orders, live.yesterday.orders)}
                note={t.dash.liveVsYesterday}
              />
              <Tile
                label={t.dash.liveCancelled}
                value={String(live.today.cancelled)}
                /* No delta arrow: fewer cancellations is better, and an arrow
                   whose good direction flips per tile teaches nothing. */
                tone={live.today.cancelled > 0 ? "bad" : undefined}
              />
              <Tile
                label={t.dash.liveRevenue}
                value={money(live.today.revenue)}
                delta={delta(live.today.revenue, live.yesterday.revenue)}
                note={t.dash.liveVsYesterday}
              />
              {/* Beside the takings, never folded into them: the kitchen's
                  workload is not money in hand. Same split as the restaurant's
                  own dashboard, so the two screens agree about one day. */}
              <Tile label={t.dash.livePendingMoney} value={money(live.today.pending)} />
              <Tile label={t.dash.liveAvgOrder} value={money(live.today.avgOrder)} />
            </div>
            <p className="mt-2 text-xs text-ink-muted">{t.dash.liveRevenueNote}</p>

            {/* ---- The kitchen queue ---- */}
            <div className="mt-5">
              <p className="text-xs font-semibold uppercase tracking-wider text-ink-muted">
                {t.dash.liveActive}
              </p>
              {live.active.total === 0 ? (
                <p className="mt-2 text-sm text-ink-muted">{t.dash.liveActiveNone}</p>
              ) : (
                <div className="mt-2 flex flex-wrap gap-2">
                  <Pill label={t.dash.livePending} n={live.active.pending} />
                  <Pill label={t.dash.liveConfirmed} n={live.active.confirmed} />
                  <Pill label={t.dash.livePreparing} n={live.active.preparing} />
                  <Pill label={t.dash.liveOnTheWay} n={live.active.onTheWay} />
                </div>
              )}
            </div>

            {/* ---- Order types: three parts of one whole ---- */}
            {live.today.orders > 0 && (
              <div className="mt-5">
                <p className="text-xs font-semibold uppercase tracking-wider text-ink-muted">
                  {t.dash.liveTypeTitle}
                </p>
                <TypeSplit
                  parts={[
                    { label: t.dash.liveDelivery, n: live.today.delivery, color: C.delivery },
                    { label: t.dash.livePickup, n: live.today.pickup, color: C.pickup },
                    { label: t.dash.liveDinein, n: live.today.dinein, color: C.dinein },
                  ]}
                />
              </div>
            )}
          </>
        )}
      </section>

      {/* ---- Two charts, two scales, never one frame ---- */}
      <section className="card">
        <p className="text-sm font-semibold text-ink">{t.dash.liveChartOrders}</p>
        <Legend
          items={[
            { label: t.dash.liveOrders, color: C.orders },
            { label: t.dash.liveCancelled, color: C.cancelled },
          ]}
        />
        <Columns
          rows={recent}
          series={[
            { key: (d) => d.orders, color: C.orders, label: t.dash.liveOrders },
            {
              key: (d) => d.cancelled ?? 0,
              color: C.cancelled,
              label: t.dash.liveCancelled,
            },
          ]}
          empty={t.dash.noData}
          format={(n) => String(n)}
        />

        {/* ⚠️ Only when it happened, and only as a sentence.
            A permanent row of zeroes here would be read as decoration within a
            week — the same rule the overview's attention strip follows — and
            this is a line somebody has to actually read the first time it is
            not zero. It is deliberately not a chart: the question is not "how
            many", it is "is this restaurant doing something we should ask
            about", and one sentence answers that better than a bar. */}
        {(reversed > 0 || cancelledCooked > 0) && (
          <p className="mt-4 rounded-xl border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-xs text-ink-soft">
            {reversed > 0 && <span className="font-semibold">{t.dash.reversedNote(reversed)} </span>}
            {cancelledCooked > 0 && t.dash.cancelledCookedNote(cancelledCooked)}
          </p>
        )}
      </section>

      <section className="card">
        <p className="text-sm font-semibold text-ink">{t.dash.chartVisitors}</p>
        {/* One series, so the title names it and no legend box is needed. */}
        <Columns
          rows={recent}
          series={[
            { key: (d) => d.visitors ?? 0, color: C.dinein, label: t.dash.chartVisitors },
          ]}
          empty={t.dash.noData}
          format={(n) => String(n)}
        />
      </section>

      <section className="card">
        <p className="text-sm font-semibold text-ink">{t.dash.liveChartRevenue}</p>
        {/* One series: the title says what it is, so no legend box. */}
        <Columns
          rows={recent}
          series={[{ key: (d) => d.revenue, color: C.orders, label: t.dash.liveRevenue }]}
          empty={t.dash.noData}
          format={money}
        />
      </section>

      {/* ---- Who and what the restaurant runs on ---- */}
      {live && (
        <section className="grid gap-4 sm:grid-cols-2">
          <Group
            title={t.dash.livePeople}
            rows={[
              [t.dash.livePeopleTotal, live.people.customers],
              [t.dash.livePeopleNew, live.people.new30d],
              [t.dash.livePeopleActive, live.people.active30d],
            ]}
          />
          <Group
            title={t.dash.liveStaff}
            rows={[
              [t.dash.liveStaffTotal, live.staff.total],
              [t.dash.liveStaffActive, live.staff.active],
              [t.dash.liveStaffOnShift, live.staff.onShift],
            ]}
          />
          <Group
            title={t.dash.liveCouriers}
            rows={[
              [t.dash.livePeopleTotal, live.couriers.total],
              [t.dash.liveCourierFree, live.couriers.free],
              [t.dash.liveCourierBusy, live.couriers.busy],
              [t.dash.liveCourierOff, live.couriers.off],
            ]}
          />
          <Group
            title={t.dash.liveMenu}
            rows={[
              [t.dash.liveMenuItems, live.menu.items],
              [t.dash.liveMenuAvailable, live.menu.available],
              [t.dash.liveMenuCategories, live.menu.categories],
              [t.dash.liveBranches, live.menu.branches],
              [t.dash.liveBrands, live.menu.brands],
            ]}
          />
          <Group
            title={t.dash.liveTraffic}
            rows={[
              [t.dash.liveVisitorsToday, live.traffic.todayVisitors],
              [t.dash.liveViewsToday, live.traffic.todayViews],
              [t.dash.liveVisitors30d, live.traffic.visitors30d],
            ]}
            note={t.dash.liveTrafficNote}
          />
          <Group
            title={t.dash.liveReservations}
            rows={[
              [t.dash.liveResToday, live.reservations.today],
              [t.dash.liveResUpcoming, live.reservations.upcoming],
            ]}
          />
        </section>
      )}

      {/* ---- What they actually sell ---- */}
      {live && live.topItems.length > 0 && (
        <section className="card">
          <p className="text-sm font-semibold text-ink">{t.dash.liveTopItems}</p>
          <TopItems items={live.topItems} />
        </section>
      )}
    </div>
  );
}

/** The chart palette as CSS custom properties, with a selected dark set.
 *
 *  Dark is not an automatic flip of light: each step was chosen for the dark
 *  surface and validated against it. Declared under both the media query and
 *  the `dark` class so the console's own toggle wins either way. */
function VizTokens() {
  return (
    <style>{`
      .viz { --viz-1:#2a78d6; --viz-2:#eb6834; --viz-3:#1baf7a; --viz-bad:#d03b3b;
             --viz-grid: rgb(var(--line)); --viz-surface: rgb(var(--surface)); }
      .dark .viz { --viz-1:#3987e5; --viz-2:#d95926; --viz-3:#199e70; --viz-bad:#d03b3b; }
    `}</style>
  );
}

function delta(now: number, before: number): number | null {
  if (!before) return null;
  return Math.round(((now - before) / before) * 100);
}

function Tile({
  label,
  value,
  delta,
  note,
  tone,
}: {
  label: string;
  value: string;
  delta?: number | null;
  note?: string;
  tone?: "bad";
}) {
  return (
    <div className="bg-surface p-4">
      <p className="text-xs uppercase tracking-wider text-ink-muted">{label}</p>
      <p
        className={`h-display mt-1 text-2xl tabular-nums ${
          tone === "bad" ? "text-rose-600 dark:text-rose-400" : ""
        }`}
      >
        {value}
      </p>
      {delta != null && (
        <p className="mt-1 text-xs text-ink-muted">
          <span className={delta >= 0 ? "text-emerald-700 dark:text-emerald-300" : "text-rose-600 dark:text-rose-400"}>
            {delta >= 0 ? "▲" : "▼"} {Math.abs(delta)}%
          </span>{" "}
          {note}
        </p>
      )}
    </div>
  );
}

function Pill({ label, n }: { label: string; n: number }) {
  if (n === 0) return null;
  return (
    <span className="rounded-xl bg-raised px-3 py-1.5 text-xs text-ink-soft">
      {label}: <span className="font-semibold tabular-nums text-ink">{n}</span>
    </span>
  );
}

/** Three parts of one whole, as one stacked bar.
 *
 *  Direct-labelled underneath rather than inside: the segments are often too
 *  narrow for text, and a label clipped by its own segment is worse than none.
 *  The labels are also the relief for light-mode aqua, which sits under 3:1
 *  against a white surface. */
function TypeSplit({
  parts,
}: {
  parts: { label: string; n: number; color: string }[];
}) {
  const total = parts.reduce((s, p) => s + p.n, 0) || 1;
  const shown = parts.filter((p) => p.n > 0);
  return (
    <div className="viz mt-2">
      {/* The 2px gaps are the surface doing the separating — no strokes. */}
      <div className="flex h-3 gap-[2px] overflow-hidden rounded">
        {shown.map((p) => (
          <div
            key={p.label}
            style={{ width: `${(p.n / total) * 100}%`, background: p.color }}
            className="first:rounded-l last:rounded-r"
          />
        ))}
      </div>
      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs">
        {shown.map((p) => (
          <span key={p.label} className="flex items-center gap-1.5 text-ink-soft">
            <i
              aria-hidden
              className="inline-block h-2 w-2 shrink-0 rounded-full"
              style={{ background: p.color }}
            />
            {p.label}
            <span className="font-semibold tabular-nums text-ink">{p.n}</span>
          </span>
        ))}
      </div>
    </div>
  );
}

function Legend({ items }: { items: { label: string; color: string }[] }) {
  return (
    <div className="viz mt-1 flex flex-wrap gap-x-4 gap-y-1 text-xs text-ink-soft">
      {items.map((i) => (
        <span key={i.label} className="flex items-center gap-1.5">
          <i
            aria-hidden
            className="inline-block h-2 w-2 rounded-full"
            style={{ background: i.color }}
          />
          {i.label}
        </span>
      ))}
    </div>
  );
}

/** Grouped columns over time.
 *
 *  Divs rather than a chart library, following the precedent already set on the
 *  overview: a megabyte of dependency to say what flex children already say.
 *  Hover is native `title` — the hit target is the whole day column, including
 *  its empty space above the bars, so a quiet day is still hoverable. */
function Columns({
  rows,
  series,
  empty,
  format,
}: {
  rows: TenantDay[];
  series: {
    key: (d: TenantDay) => number;
    color: string;
    label: string;
  }[];
  empty: string;
  format: (n: number) => string;
}) {
  const { t } = useT();
  const [table, setTable] = useState(false);
  const max = Math.max(1, ...rows.flatMap((d) => series.map((s) => s.key(d))));
  const anyData = rows.some((d) => series.some((s) => s.key(d) > 0));

  if (rows.length === 0 || !anyData) {
    return <p className="mt-3 text-sm text-ink-muted">{empty}</p>;
  }

  return (
    <div className="viz">
      {/* The scale, stated once, above the plot. It sat between the two date
          labels at first and read as a third date — a number in an axis row is
          taken for an axis value wherever it lands. */}
      <p className="mt-3 text-xs text-ink-muted">
        max <span className="tabular-nums text-ink-soft">{format(max)}</span>
      </p>
      <div className="mt-2 flex h-40 items-end gap-[3px]">
        {rows.map((d) => (
          <div
            key={d.date}
            className="group flex h-full flex-1 items-end justify-center gap-[2px]"
            title={`${dayLabel(d.date)}\n${series
              .map((s) => `${s.label}: ${format(s.key(d))}`)
              .join("\n")}`}
          >
            {series.map((s) => {
              const v = s.key(d);
              return (
                <div
                  key={s.label}
                  className="w-full max-w-[10px] rounded-t transition-opacity group-hover:opacity-80"
                  style={{
                    // Zero keeps a hairline so the day is still a hit target
                    // and the baseline reads as a row of days, not a gap.
                    height: `${Math.max(1.5, (v / max) * 100)}%`,
                    background: s.color,
                  }}
                />
              );
            })}
          </div>
        ))}
      </div>
      <div className="mt-2 flex justify-between text-xs text-ink-muted">
        <span>{dayLabel(rows[0]?.date ?? "")}</span>
        <span>{dayLabel(rows[rows.length - 1]?.date ?? "")}</span>
      </div>

      {/* The table view. Required relief wherever a mark is under 3:1, and the
          only way anybody reads an exact value off a 30-day strip. */}
      <button
        type="button"
        className="mt-2 text-xs text-ink-muted underline underline-offset-2"
        onClick={() => setTable((v) => !v)}
      >
        {table ? "▾" : "▸"} {t.dash.liveTable}
      </button>
      {table && (
        <div className="mt-2 max-h-64 overflow-y-auto">
          <table className="w-full text-xs">
            <thead className="sticky top-0 bg-surface text-left text-ink-muted">
              <tr>
                <th className="py-1 font-medium">{t.dash.liveDate}</th>
                {series.map((s) => (
                  <th key={s.label} className="py-1 text-right font-medium">
                    {s.label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {[...rows].reverse().map((d) => (
                <tr key={d.date}>
                  <td className="py-1 text-ink-muted">{dayLabel(d.date)}</td>
                  {series.map((s) => (
                    <td key={s.label} className="py-1 text-right tabular-nums text-ink-soft">
                      {format(s.key(d))}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

/** Best sellers — one series, so one hue and no legend. Horizontal, because
 *  dish names are long and a rotated label is a label nobody reads. */
function TopItems({ items }: { items: { name: string; qty: number }[] }) {
  const max = Math.max(1, ...items.map((i) => i.qty));
  return (
    <ul className="viz mt-4 space-y-2">
      {items.map((i) => (
        <li key={i.name} className="flex items-center gap-3">
          <span className="w-40 shrink-0 truncate text-xs text-ink-soft" title={i.name}>
            {i.name}
          </span>
          <span className="h-2.5 flex-1 overflow-hidden rounded-r">
            <span
              className="block h-full rounded-r"
              style={{ width: `${(i.qty / max) * 100}%`, background: C.orders }}
            />
          </span>
          <span className="w-10 shrink-0 text-right text-xs font-semibold tabular-nums text-ink">
            {i.qty}
          </span>
        </li>
      ))}
    </ul>
  );
}

function Group({
  title,
  rows,
  note,
}: {
  title: string;
  rows: [string, number][];
  /** Said under the numbers when they mean something narrower than they
   *  look — "visitor-days, not people" is the difference between a useful
   *  figure and one somebody quotes wrongly to a customer. */
  note?: string;
}) {
  return (
    <div className="card">
      <p className="text-sm font-semibold text-ink">{title}</p>
      <dl className="mt-3 space-y-1.5 text-sm">
        {rows.map(([label, n]) => (
          <div key={label} className="flex items-baseline justify-between gap-3">
            <dt className="text-ink-muted">{label}</dt>
            <dd className="tabular-nums font-semibold text-ink">{n}</dd>
          </div>
        ))}
      </dl>
      {note && <p className="mt-2 text-xs text-ink-muted">{note}</p>}
    </div>
  );
}
