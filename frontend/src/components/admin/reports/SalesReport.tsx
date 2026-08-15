"use client";

// Sales over time.
//
// The dashboard already says what this period was. This says whether it was
// good, which is a different question and needs a second number next to every
// first one: a month's takings mean one thing after a better month and the
// opposite after a worse one.
//
// ⚠️ **Takings are money in hand, not orders placed** — the same rule the
// dashboard uses. Two screens that both say "tushum" and disagree are worse
// than one screen, and this panel has made that mistake once already.

import { useCallback, useEffect, useState } from "react";
import { api, downloadReport } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { ListScroll } from "@/components/admin/PagedList";
import { TrendChart, HoursChart } from "@/components/admin/Charts";
import type { SalesGroup, SalesHour, SalesReportResponse, SalesTotals } from "@/lib/types";

type Range = { from?: string; to?: string };

export default function SalesReport({ range }: { range: Range }) {
  const t = useAdminT();
  const [group, setGroup] = useState<SalesGroup>("day");
  const [data, setData] = useState<SalesReportResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const params = useCallback(() => ({ ...range, group }), [range, group]);

  useEffect(() => {
    setLoading(true);
    api
      .salesReport(params())
      .then(setData)
      .catch(() => setError(t.common.loadFailed))
      .finally(() => setLoading(false));
  }, [params, t]);

  async function download() {
    setBusy(true);
    setError("");
    try {
      // The same parameters the screen asked with, `group` included. A
      // spreadsheet cut by days when the screen showed months is the exact
      // disagreement this export exists to prevent.
      await downloadReport("/admin/reports/sales", params());
    } catch {
      setError(t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  const s = t.reports.sales;
  const buckets = data?.buckets ?? [];

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        {/* How finely to cut the period. Buttons, not a select: this decides
            what the whole screen means, and a question inside a dropdown is one
            nobody asks. */}
        <div className="flex flex-wrap items-center gap-2">
          {(["day", "week", "month"] as SalesGroup[]).map((g) => (
            <button
              key={g}
              type="button"
              onClick={() => setGroup(g)}
              className={
                group === g
                  ? "rounded-xl bg-ink px-3 py-2 text-xs font-semibold text-surface"
                  : "rounded-xl border border-line px-3 py-2 text-xs font-semibold text-ink-soft hover:bg-ink/5"
              }
            >
              {t.reports.group[g]}
            </button>
          ))}
        </div>
        <button
          type="button"
          onClick={download}
          disabled={busy || !buckets.length}
          className="btn btn-primary disabled:opacity-60"
        >
          {busy ? t.common.loading : t.reports.excel}
        </button>
      </div>

      {error && <p className="text-sm text-brand">{error}</p>}
      {data?.note && <p className="text-sm text-ink-soft">{data.note}</p>}

      {loading ? (
        <p className="py-10 text-center text-ink-muted/70">{t.common.loading}</p>
      ) : !buckets.length ? (
        <p className="py-10 text-center text-ink-muted/70">{s.empty}</p>
      ) : (
        <>
          <Headline data={data!} />

          {/* One axis, always. Orders and money are never drawn on one chart
              with two scales: the shape of such a chart is decided by where the
              two axes are zeroed, which is to say by nothing. */}
          <div className="card p-4">
            <h3 className="mb-3 text-sm font-semibold text-ink-soft">{s.revenue}</h3>
            <TrendChart
              labels={buckets.map((b) => b.label)}
              data={buckets.map((b) => b.revenue)}
              label={s.revenue}
              money
            />
          </div>

          <BusiestHours hours={data!.byHour ?? []} />

          <ListScroll className="max-h-[60vh]">
            <table className="w-full min-w-[900px] text-sm">
              <thead className="sticky top-0 bg-raised text-left text-xs uppercase tracking-wider text-ink-muted">
                <tr>
                  <th className="px-3 py-2">{s.period}</th>
                  <th className="px-3 py-2 text-right">{s.orders}</th>
                  <th className="px-3 py-2 text-right">{s.paid}</th>
                  <th className="px-3 py-2 text-right">{s.cancelled}</th>
                  <th className="px-3 py-2 text-right">{s.revenue}</th>
                  <th className="px-3 py-2 text-right">{s.avgCheck}</th>
                  <th className="px-3 py-2 text-right">{s.pending}</th>
                  <th className="px-3 py-2 text-right">{s.items}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line">
                {buckets.map((b) => (
                  <tr key={b.key} className="hover:bg-raised/60">
                    <td className="px-3 py-2 font-medium text-ink">{b.label}</td>
                    <td className="px-3 py-2 text-right tabular-nums">{b.orders}</td>
                    <td className="px-3 py-2 text-right tabular-nums">{b.paid}</td>
                    <td className="px-3 py-2 text-right tabular-nums text-ink-muted">
                      {b.cancelled || ""}
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums font-medium">
                      {formatPrice(b.revenue)}
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums">
                      {formatPrice(b.avgCheck)}
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums text-ink-muted">
                      {b.pending ? formatPrice(b.pending) : ""}
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums">{b.items}</td>
                  </tr>
                ))}
              </tbody>
              <tfoot className="border-t border-line-strong bg-raised/60 font-semibold">
                <tr>
                  <td className="px-3 py-2">{s.total}</td>
                  <td className="px-3 py-2 text-right tabular-nums">{data!.totals.orders}</td>
                  <td className="px-3 py-2 text-right tabular-nums">{data!.totals.paid}</td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {data!.totals.cancelled}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {formatPrice(data!.totals.revenue)}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {formatPrice(data!.totals.avgCheck)}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {formatPrice(data!.totals.pending)}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">{data!.totals.items}</td>
                </tr>
              </tfoot>
            </table>
          </ListScroll>
        </>
      )}
    </div>
  );
}

/** When the orders arrive, across the day.
 *
 *  A staffing question rather than a money one — "when do we need a second
 *  courier on" — which is why it is orders and not takings, and why it is
 *  summed over the whole period: one Friday says nothing, thirty Fridays say
 *  where the rush is.
 *
 *  ⚠️ **The range runs from the first hour of trade to the last, gaps
 *  included.** Listing only the hours that sold puts 12:00 next to 15:00 and
 *  draws a day with no quiet middle — the lull between lunch and dinner is
 *  exactly what somebody reads this to find. The closed hours at either end
 *  stay out: a quarter of the chart pinned at zero tells nobody anything they
 *  did not know. */
function BusiestHours({ hours }: { hours: SalesHour[] }) {
  const t = useAdminT();
  const s = t.reports.sales;

  const first = hours.findIndex((h) => h.orders > 0);
  if (first < 0) return null;
  let last = first;
  hours.forEach((h, i) => {
    if (h.orders > 0) last = i;
  });
  const open = hours.slice(first, last + 1);

  const total = hours.reduce((sum, h) => sum + h.orders, 0);
  const peak = open.reduce((best, h) => (h.orders > best.orders ? h : best), open[0]);
  const share = total ? Math.round((peak.orders / total) * 100) : 0;

  return (
    <div className="card p-4">
      <h3 className="text-sm font-semibold text-ink-soft">{s.busiest}</h3>
      {/* The sentence sits above the chart, not under it: a reader who has
          already guessed what the bars mean does not come back to check. */}
      <p className="mt-1 mb-3 text-xs text-ink-muted">{s.busiestHint}</p>
      <HoursChart
        labels={open.map((h) => s.hour(h.hour))}
        data={open.map((h) => h.orders)}
        label={s.orders}
      />
      {/* The one line off this chart anybody repeats out loud, written out
          rather than left to be read off the tallest bar. */}
      <p className="mt-3 text-sm font-medium text-ink">
        {s.peak(s.hour(peak.hour), peak.orders, share)}
      </p>
    </div>
  );
}

/** The figures with their direction — the part of this screen an owner reads
 *  and repeats. */
function Headline({ data }: { data: SalesReportResponse }) {
  const t = useAdminT();
  const s = t.reports.sales;
  const pct = data.compare?.percent ?? {};

  return (
    <div className="space-y-3">
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Tile label={s.revenue} value={formatPrice(data.totals.revenue)} change={pct.revenue} />
        <Tile label={s.orders} value={String(data.totals.orders)} change={pct.orders} />
        <Tile
          label={s.avgCheck}
          value={formatPrice(data.totals.avgCheck)}
          change={pct.avgCheck}
        />
        <Tile label={s.items} value={String(data.totals.items)} change={pct.items} />
      </div>

      <p className="text-xs text-ink-muted">
        {data.compare
          ? s.vsPrevious(data.compare.from, data.compare.to)
          : /* No baseline, so no arrows. Stated rather than left blank: an
               owner who sees no percentages should know it is because there is
               nothing behind this period, not because nothing changed. */
            s.noCompare}
        {data.best && ` · ${s.best}: ${data.best.label} — ${formatPrice(data.best.revenue)}`}
      </p>
    </div>
  );
}

function Tile({
  label,
  value,
  change,
}: {
  label: string;
  value: string;
  change?: number;
}) {
  return (
    <div className="card p-4">
      <p className="text-xs uppercase tracking-wider text-ink-muted">{label}</p>
      <p className="mt-1 text-2xl font-bold tabular-nums text-ink">{value}</p>
      {change !== undefined && (
        <p
          className={`mt-1 text-xs font-semibold tabular-nums ${
            change >= 0
              ? "text-emerald-700 dark:text-emerald-300"
              : "text-rose-700 dark:text-rose-300"
          }`}
        >
          {change >= 0 ? "↑" : "↓"} {Math.abs(change).toFixed(1)}%
        </p>
      )}
    </div>
  );
}

export type { SalesTotals };
