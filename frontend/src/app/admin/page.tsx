"use client";

// The panel's front page: what happened in a period the operator picks.
//
// The numbers come from `GET /admin/stats`, counted in the database. The old
// version added up the last 200 orders in the browser, which stopped being true
// the day a restaurant passed 200 orders — and could not answer "how many
// customers do we have" at all.

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { ORDER_STATUSES, STATUS_BADGE } from "@/lib/orderStatus";
import { useAdminT } from "@/lib/i18n/admin";
import { ListScroll } from "@/components/admin/PagedList";
import { BreakdownChart, TrendChart } from "@/components/admin/Charts";
import type { AdminStats, Order } from "@/lib/types";

type Preset = "today" | "week" | "month" | "all" | "custom";

/** A date as the `yyyy-mm-dd` the API and <input type="date"> both take. */
function isoDate(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(
    d.getDate(),
  ).padStart(2, "0")}`;
}

function daysAgo(n: number): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return isoDate(d);
}

export default function AdminDashboard() {
  const t = useAdminT();
  // Opens on the week, not on today: the day-by-day chart needs at least two
  // points, so a dashboard that opens on "today" opens with its chart section
  // missing — which reads as a broken chart, not as an empty period.
  const [preset, setPreset] = useState<Preset>("week");
  // Only used by the "custom" preset; prefilled with the last week so the
  // inputs never open empty.
  const [from, setFrom] = useState(daysAgo(7));
  const [to, setTo] = useState(isoDate(new Date()));
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [recent, setRecent] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);

  const range = useMemo((): { from?: string; to?: string } => {
    const today = isoDate(new Date());
    switch (preset) {
      case "today":
        return { from: today, to: today };
      case "week":
        return { from: daysAgo(6), to: today };
      case "month":
        return { from: daysAgo(29), to: today };
      case "custom":
        return { from, to };
      case "all":
      default:
        return {};
    }
  }, [preset, from, to]);

  const load = useCallback(() => {
    setLoading(true);
    api
      .adminStats(range)
      .then(setStats)
      .catch(() => setStats(null))
      .finally(() => setLoading(false));
  }, [range]);

  useEffect(load, [load]);

  // The newest orders, independent of the period — this is the "is anything
  // waiting for me right now" list.
  useEffect(() => {
    api
      .adminOrders({ limit: 20 })
      .then(setRecent)
      .catch(() => setRecent([]));
  }, []);

  const p = stats?.period;
  const series = stats?.series ?? [];
  const channels = stats ? channelData(stats, t) : null;
  const show = (v: number | undefined) => (loading || v == null ? "…" : String(v));
  const money = (v: number | undefined) =>
    loading || v == null ? "…" : formatPrice(v);

  const presets: { key: Preset; label: string }[] = [
    { key: "today", label: t.dashboard.periodToday },
    { key: "week", label: t.dashboard.periodWeek },
    { key: "month", label: t.dashboard.periodMonth },
    { key: "all", label: t.dashboard.periodAll },
    { key: "custom", label: t.dashboard.periodCustom },
  ];

  return (
    <div>
      <h1 className="text-2xl font-bold">{t.dashboard.title}</h1>

      {/* ---- period ---- */}
      <div className="mt-4 flex flex-wrap items-center gap-2">
        {presets.map((op) => (
          <button
            key={op.key}
            type="button"
            onClick={() => setPreset(op.key)}
            className={`rounded-full px-3.5 py-1.5 text-sm font-semibold transition-colors ${
              preset === op.key
                ? "bg-brand text-white"
                : "border border-line bg-surface text-ink-soft hover:border-brand hover:text-brand"
            }`}
          >
            {op.label}
          </button>
        ))}

        {preset === "custom" && (
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <span className="text-ink-muted">{t.dashboard.periodFrom}</span>
            <input
              type="date"
              value={from}
              max={to}
              onChange={(e) => setFrom(e.target.value)}
              className="rounded-xl border border-line-strong bg-surface px-3 py-1.5 text-sm outline-none focus:border-brand"
            />
            <span className="text-ink-muted">{t.dashboard.periodTo}</span>
            <input
              type="date"
              value={to}
              min={from}
              onChange={(e) => setTo(e.target.value)}
              className="rounded-xl border border-line-strong bg-surface px-3 py-1.5 text-sm outline-none focus:border-brand"
            />
          </div>
        )}
      </div>

      {/* ---- orders ---- */}
      <Group title={t.dashboard.groupOrders}>
        <Tile label={t.dashboard.ordersTotal} value={show(p?.orders)} />
        <Tile label={t.dashboard.ordersDelivered} value={show(p?.delivered)} />
        <Tile label={t.dashboard.ordersDelivery} value={show(p?.delivery)} />
        <Tile label={t.dashboard.ordersPickup} value={show(p?.pickup)} />
        <Tile label={t.dashboard.ordersDineIn} value={show(p?.dineIn)} />
        <Tile label={t.dashboard.ordersCancelled} value={show(p?.cancelled)} />
      </Group>

      {/* ---- money ---- */}
      <Group title={t.dashboard.groupMoney}>
        <Tile label={t.dashboard.revenue} value={money(p?.revenue)} accent />
        {/* Beside the takings, never folded into them. An owner does want to
            know what today is still going to bring in — they just must not be
            told they already have it. */}
        <Tile label={t.dashboard.pending} value={money(p?.pending)} />
        <Tile label={t.dashboard.avgOrder} value={money(p?.avgOrder)} />
        <Tile label={t.dashboard.deliveryFees} value={money(p?.deliveryFee)} />
        <Tile label={t.dashboard.cashTotal} value={money(p?.cashTotal)} />
      </Group>
      {/* Says what "tushum" counts. Without it the number looks low to anybody
          who remembers the old one, and the honest explanation is short. */}
      <p className="mt-2 text-xs text-ink-muted/80">
        {t.dashboard.revenueNote}
      </p>
      <p className="mt-1 text-xs text-ink-muted/80">
        {t.dashboard.cancelledNote}
      </p>

      {/* ---- people ---- */}
      <Group title={t.dashboard.groupPeople}>
        <Tile label={t.dashboard.usersTotal} value={show(stats?.users.total)} />
        <Tile label={t.dashboard.usersNew} value={show(stats?.users.new)} />
        <Tile
          label={t.dashboard.usersActive}
          value={show(stats?.users.active)}
          hint={t.dashboard.usersActiveHint}
        />
        <Tile
          label={t.dashboard.couriersTotal}
          value={show(stats?.couriers.total)}
          hint={`${t.dashboard.couriersOnline}: ${show(stats?.couriers.online)}`}
        />
        <Tile
          label={t.dashboard.adminsTotal}
          value={show(stats?.admins.total)}
          hint={
            stats
              ? t.dashboard.adminsSplit(
                  stats.admins.owners,
                  stats.admins.managers,
                )
              : undefined
          }
        />
        <Tile
          label={t.dashboard.usersWithAddress}
          value={show(stats?.users.withAddress)}
        />
      </Group>

      {/* ---- menu ---- */}
      <Group title={t.dashboard.groupMenu}>
        <Tile label={t.dashboard.menuDishes} value={show(stats?.menu.dishes)} />
        <Tile
          label={t.dashboard.menuAvailable}
          value={show(stats?.menu.available)}
        />
        <Tile
          label={t.dashboard.menuCategories}
          value={show(stats?.menu.categories)}
        />
      </Group>

      {/* ---- the trend ----
           Full width and above the breakdowns: "is it going up" is the question
           a dashboard is opened for, and every figure above is one number from
           this line. Orders rather than money, because the count is the shape
           of the business and the revenue line only repeats it multiplied by
           the average cheque. */}
      {/* The section stays even when there is nothing to draw, and says why.
           Dropping it silently is how a period with no orders — or a single-day
           period, which cannot make a line — looks exactly like a chart that
           failed to load. */}
      <section className="mt-8 rounded-3xl border border-line bg-surface p-5 shadow-card">
        <h2 className="text-lg font-bold">{t.dashboard.trendTitle}</h2>
        <p className="mb-3 mt-1 text-xs text-ink-muted">
          {t.dashboard.trendNote}
        </p>
        {series.length > 1 ? (
          <TrendChart
            labels={series.map((d) => d.date.slice(5))}
            data={series.map((d) => d.orders)}
            label={t.dashboard.orders}
          />
        ) : (
          <p className="py-6 text-center text-sm text-ink-muted">
            {loading || !stats
              ? "…"
              : series.length === 0
                ? t.dashboard.trendEmpty
                : t.dashboard.trendOneDay}
          </p>
        )}
      </section>

      <div className="mt-8 grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* ---- how the orders arrive ----
             A chart rather than three numbers because the reader's question is
             a comparison ("is pickup worth the counter staff?"), and comparing
             is what a bar does and a list of figures does not. */}
        <section className="rounded-3xl border border-line bg-surface p-5 shadow-card">
          <h2 className="text-lg font-bold">{t.dashboard.channelsTitle}</h2>
          {channels && channels.values.some((v) => v > 0) ? (
            <BreakdownChart
              labels={channels.labels}
              data={channels.values}
              money={false}
            />
          ) : (
            <p className="py-6 text-center text-sm text-ink-muted">
              {loading || !stats ? "…" : t.dashboard.channelsEmpty}
            </p>
          )}
        </section>

        {/* ---- status breakdown ---- */}
        <section className="rounded-3xl border border-line bg-surface p-5 shadow-card">
          <h2 className="text-lg font-bold">{t.dashboard.statusBreakdown}</h2>
          <ul className="mt-3 space-y-1.5 text-sm">
            {ORDER_STATUSES.map((s) => {
              const n = p?.byStatus?.[s] ?? 0;
              const share = p?.orders ? Math.round((n / p.orders) * 100) : 0;
              return (
                <li key={s} className="flex items-center gap-3">
                  <span className={`badge ${STATUS_BADGE[s]} shrink-0`}>
                    {t.status[s]}
                  </span>
                  {/* The bar makes the shape of the day readable at a glance. */}
                  <span className="h-1.5 flex-1 overflow-hidden rounded-full bg-ink/5">
                    <span
                      className="block h-full rounded-full bg-brand"
                      style={{ width: `${share}%` }}
                    />
                  </span>
                  <span className="w-14 shrink-0 text-right tabular-nums">
                    {n}
                    <span className="ml-1 text-xs text-ink-muted">
                      {share}%
                    </span>
                  </span>
                </li>
              );
            })}
          </ul>
        </section>

        {/* ---- best sellers ---- */}
        <section className="rounded-3xl border border-line bg-surface p-5 shadow-card">
          <h2 className="text-lg font-bold">{t.dashboard.topDishes}</h2>
          {stats && stats.top.length === 0 ? (
            <p className="mt-3 text-sm text-ink-muted/70">
              {t.dashboard.topEmpty}
            </p>
          ) : (
            <ListScroll className="mt-3 pr-1" max="max-h-72">
              <ul className="space-y-1.5 text-sm">
                {(stats?.top ?? []).map((d, i) => (
                  <li
                    key={d.name}
                    className="flex items-center justify-between gap-3 rounded-xl px-2 py-1.5 odd:bg-ink/[0.02]"
                  >
                    <span className="min-w-0 flex-1 truncate">
                      <span className="mr-2 text-xs text-ink-muted">
                        {i + 1}.
                      </span>
                      {d.name}
                    </span>
                    <span className="shrink-0 tabular-nums text-ink-muted">
                      × {d.qty}
                    </span>
                    <span className="w-24 shrink-0 text-right font-semibold tabular-nums">
                      {formatPrice(d.total)}
                    </span>
                  </li>
                ))}
              </ul>
            </ListScroll>
          )}
        </section>
      </div>

      {/* ---- newest orders ---- */}
      <div className="mt-8 flex items-center justify-between">
        <h2 className="text-lg font-bold">{t.dashboard.recent}</h2>
        <Link href="/admin/orders" className="text-sm text-brand hover:underline">
          {t.orders.filterAll} →
        </Link>
      </div>

      <div className="mt-4 overflow-hidden rounded-3xl border border-line bg-surface shadow-card">
        <ListScroll className="overflow-x-auto" max="max-h-96">
          <table className="w-full text-sm">
            <thead className="sticky top-0 z-10 border-b border-line bg-surface text-left text-ink-muted">
              <tr>
                <th className="px-4 py-3 font-medium">{t.dashboard.number}</th>
                <th className="px-4 py-3 font-medium">{t.dashboard.customer}</th>
                <th className="px-4 py-3 font-medium">{t.dashboard.type}</th>
                <th className="px-4 py-3 font-medium">{t.dashboard.sum}</th>
                <th className="px-4 py-3 font-medium">{t.dashboard.status}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {recent.length === 0 ? (
                <tr>
                  <td
                    colSpan={5}
                    className="px-4 py-8 text-center text-ink-muted/70"
                  >
                    {t.orders.empty}
                  </td>
                </tr>
              ) : (
                recent.map((o) => (
                  <tr key={o.id}>
                    <td className="px-4 py-3 font-medium">#{o.number}</td>
                    <td className="px-4 py-3">{o.customer.name}</td>
                    <td className="px-4 py-3 text-ink-muted">
                      {o.type === "delivery"
                        ? t.dashboard.delivery
                        : o.type === "dinein"
                          ? t.receipt.tableLine(o.tableNumber ?? "—")
                          : t.dashboard.pickup}
                    </td>
                    <td className="px-4 py-3">{formatPrice(o.total)}</td>
                    <td className="px-4 py-3">
                      <span
                        className={`rounded-full px-2 py-0.5 text-xs font-semibold ${STATUS_BADGE[o.status]}`}
                      >
                        {t.status[o.status]}
                      </span>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </ListScroll>
      </div>
    </div>
  );
}

function Group({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="mt-6">
      <h2 className="text-xs font-semibold uppercase tracking-wider text-ink-muted">
        {title}
      </h2>
      <div className="mt-2 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        {children}
      </div>
    </section>
  );
}

function Tile({
  label,
  value,
  hint,
  accent = false,
}: {
  label: string;
  value: string;
  hint?: string;
  accent?: boolean;
}) {
  return (
    <div className="rounded-2xl border border-line bg-surface p-4 shadow-card">
      <p className="text-xs uppercase tracking-wider text-ink-muted">{label}</p>
      <p
        className={`mt-1 font-display text-xl font-bold ${accent ? "text-brand" : ""}`}
      >
        {value}
      </p>
      {hint && <p className="mt-0.5 text-xs text-ink-muted">{hint}</p>}
    </div>
  );
}

/** How the period's orders arrived. Kept beside the chart that draws it so the
 *  labels and the numbers cannot drift apart. */
function channelData(stats: AdminStats, t: ReturnType<typeof useAdminT>) {
  const p = stats.period;
  return {
    labels: [t.dashboard.delivery, t.dashboard.pickup, t.dashboard.dineIn],
    values: [p.delivery, p.pickup, p.dineIn],
  };
}
