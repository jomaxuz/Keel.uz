"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useT } from "@/lib/i18n/client";
import { money, stats, type Stats } from "@/lib/api";

export default function OverviewPage() {
  const { t } = useT();
  const [data, setData] = useState<Stats | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    stats().then(setData).catch(() => setError(t.dash.loadFailed));
  }, [t]);

  if (error) return <p className="text-sm text-rose-600 dark:text-rose-400">{error}</p>;
  if (!data) return <p className="text-sm text-ink-muted">{t.dash.loading}</p>;

  const cards = [
    { label: t.dash.totalTenants, value: String(data.tenants.total) },
    { label: t.dash.active, value: String(data.tenants.active) },
    { label: t.dash.trial, value: String(data.tenants.trial) },
    { label: t.dash.suspended, value: String(data.tenants.suspended) },
    { label: t.dash.monthOrders, value: money(data.month.orders) },
    { label: t.dash.monthRevenue, value: money(data.month.revenue) },
    { label: t.dash.monthBillable, value: money(data.month.billable) },
    { label: t.dash.watermarkOff, value: String(data.tenants.watermarkRemoved) },
  ];

  // Who needs a phone call, each one a link straight into the filtered list.
  // Rendered only when there is somebody to call: a row of permanent zeroes is
  // read as decoration within a week, and then the day it says "3" nobody
  // notices.
  const calls = [
    { key: "trial_expired", label: t.dash.trialExpiredFilter, n: data.attention.trialExpired },
    { key: "trial_ending", label: t.dash.trialEndingFilter, n: data.attention.trialEnding },
    { key: "suspended", label: t.dash.unpaidFilter, n: data.attention.unpaid },
  ].filter((c) => c.n > 0);

  return (
    <div className="space-y-8">
      {calls.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 rounded-2xl border border-amber-500/40 bg-amber-500/10 p-4">
          <span className="text-sm font-semibold text-ink">{t.dash.needsAttention}</span>
          {calls.map((c) => (
            <Link
              key={c.key}
              href={`/console/tenants?attention=${c.key}`}
              className="rounded-xl bg-surface px-3 py-2 text-xs font-semibold text-ink-soft hover:text-ink"
            >
              {c.label}: <span className="tabular-nums">{c.n}</span>
            </Link>
          ))}
        </div>
      )}

      <div className="grid gap-px overflow-hidden rounded-2xl border border-line bg-line sm:grid-cols-2 lg:grid-cols-4">
        {cards.map((c) => (
          <div key={c.label} className="bg-surface p-5">
            <p className="text-xs uppercase tracking-wider text-ink-muted">{c.label}</p>
            <p className="h-display mt-2 text-2xl">{c.value}</p>
          </div>
        ))}
      </div>

      <section className="card">
        <p className="text-sm font-semibold text-ink">{t.dash.last30}</p>
        <Chart series={data.series} empty={t.dash.noData} />
      </section>

      <section className="card">
        <p className="text-sm font-semibold text-ink">{t.dash.topTenants}</p>
        {data.top.length === 0 ? (
          <p className="mt-3 text-sm text-ink-muted">{t.dash.noData}</p>
        ) : (
          <ul className="mt-4 divide-y divide-line">
            {data.top.map((row) => (
              <li key={row.id} className="flex items-center justify-between gap-4 py-3">
                <Link
                  href={`/console/tenants/${row.id}`}
                  className="truncate text-sm font-semibold text-ink hover:underline"
                >
                  {row.name}
                  <span className="ml-2 text-xs font-normal text-ink-muted">{row.slug}</span>
                </Link>
                <span className="shrink-0 text-sm text-ink-soft">
                  {money(row.orders)} · {money(row.billable)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

/** Bars in divs. A chart library for one series is a megabyte to say what
 *  twelve flex children already say. */
function Chart({ series, empty }: { series: { date: string; orders: number }[]; empty: string }) {
  const max = Math.max(1, ...series.map((p) => p.orders));
  if (series.every((p) => p.orders === 0)) {
    return <p className="mt-3 text-sm text-ink-muted">{empty}</p>;
  }
  return (
    <>
      <div className="mt-4 flex h-40 items-end gap-1">
        {series.map((p) => (
          <div key={p.date} className="group relative flex-1" title={`${p.date}: ${p.orders}`}>
            <div
              style={{ height: `${Math.max(2, (p.orders / max) * 100)}%` }}
              className="w-full rounded-t-md bg-signal-500/70 transition group-hover:bg-signal-500"
            />
          </div>
        ))}
      </div>
      <div className="mt-2 flex justify-between text-xs text-ink-muted">
        <span>{series[0]?.date}</span>
        <span>{series[series.length - 1]?.date}</span>
      </div>
    </>
  );
}
