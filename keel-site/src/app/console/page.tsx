"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { useT } from "@/lib/i18n/client";
import RolloutPanel from "@/components/RolloutPanel";
import CollectorStatus from "@/components/CollectorStatus";
import ServerHealth from "@/components/ServerHealth";
import { money, stats, type Stats } from "@/lib/api";
import { BreakdownChart, TrendChart } from "@/components/Charts";

export default function OverviewPage() {
  const { t } = useT();
  const [data, setData] = useState<Stats | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    stats()
      .then(setData)
      .catch(() => setError(t.dash.loadFailed));
  }, [t]);

  useEffect(() => {
    load();
  }, [load]);

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
    // First, and not by date order: a dark site is the only one of these where
    // a restaurant is losing orders while the row is being read.
    { key: "down", label: t.dash.downFilter, n: data.attention.down },
    { key: "trial_expired", label: t.dash.trialExpiredFilter, n: data.attention.trialExpired },
    { key: "trial_ending", label: t.dash.trialEndingFilter, n: data.attention.trialEnding },
    { key: "suspended", label: t.dash.unpaidFilter, n: data.attention.unpaid },
    // Last in the row because it is the least urgent, and present because it
    // is the only one nobody would otherwise find: a customer whose period
    // closed unbilled behaves exactly like a customer who is paid up.
    { key: "invoice_due", label: t.dash.invoiceDueFilter, n: data.attention.invoiceDue },
  ].filter((c) => c.n > 0);

  return (
    <div className="space-y-8">
      {/* Above the numbers on purpose: "twelve customers are running last
          week's code" outranks any figure on this page, and it is invisible
          everywhere else — a stale container reports itself healthy. */}
      <RolloutPanel />

      {/* Says why the numbers below are empty, when they are. Without it an
          untouched platform and a broken collector are the same picture. */}
      <CollectorStatus run={data.collector} onDone={load} />

      {/* The machine everything sits on. Above the business numbers because a
          full disk stops all of them being true. */}
      <ServerHealth />

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
        {data.series.every((p) => p.orders === 0) ? (
          <p className="mt-3 text-sm text-ink-muted">{t.dash.noData}</p>
        ) : (
          <div className="mt-4">
            <TrendChart
              labels={data.series.map((p) => p.date.slice(5))}
              data={data.series.map((p) => p.orders)}
              label={t.dash.monthOrders}
            />
          </div>
        )}
      </section>

      <section className="card">
        <p className="text-sm font-semibold text-ink">{t.dash.topTenants}</p>
        {/* The bars first, the list under them. The question here is "how much
            of the month rests on how few customers", and a ranked list answers
            it one row at a time while a chart answers it at a glance. */}
        {data.top.length > 0 && (
          <div className="mt-4">
            <BreakdownChart
              labels={data.top.map((r) => r.name)}
              data={data.top.map((r) => r.billable)}
            />
          </div>
        )}
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
