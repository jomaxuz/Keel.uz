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
import DashboardCustomiser from "@/components/admin/DashboardCustomiser";
import { TILE_GROUPS, tilesForGroup, type TileGroup } from "@/lib/dashboardTiles";
import type { AdminStats, DashboardPrefs, Order } from "@/lib/types";

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
  // This admin's arrangement of the tiles. Null until it arrives, and the
  // dashboard draws its default self in the meantime — a page that flickers
  // from empty to full is indistinguishable from one that failed.
  const [prefs, setPrefs] = useState<DashboardPrefs | null>(null);
  const [editing, setEditing] = useState(false);

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

  // ⚠️ A failed preference load leaves `prefs` null, which draws the default
  // dashboard. A stored layout must never be able to take the front page down,
  // and "everything, in the usual order" is the right thing to fall back to.
  useEffect(() => {
    api
      .adminDashboardPrefs()
      .then(setPrefs)
      .catch(() => setPrefs(null));
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
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold">{t.dashboard.title}</h1>
        {/* Only offered once the preferences have loaded: a button that opens
            an editor with nothing in it teaches people the feature is broken. */}
        {prefs && (
          <button
            type="button"
            onClick={() => setEditing((v) => !v)}
            className="btn btn-ghost text-sm"
          >
            {editing ? t.common.cancel : t.dashboard.customise.open}
          </button>
        )}
      </div>

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

      {editing && prefs && (
        <DashboardCustomiser
          prefs={prefs}
          onClose={() => setEditing(false)}
          onSaved={setPrefs}
        />
      )}

      {/* ---- the tiles ----
           Drawn from the registry in the order this admin arranged, and grouped
           by the headings the tiles carry with them. A tile keeps its group
           whatever the order says: "Pul" with an order count under it is a
           heading that lies, and the headings are the only thing making twenty
           figures readable at all.

           A group whose tiles are all switched off disappears entirely — an
           empty heading is worse than no heading. */}
      {TILE_GROUPS.map((group) => {
        const tiles = tilesForGroup(group, prefs?.visible ?? null);
        if (!tiles.length) return null;
        return (
          <Group key={group} title={groupTitle(group, t)}>
            {tiles.map((tile) => (
              <Tile
                key={tile.id}
                label={tile.label(t)}
                value={
                  !stats
                    ? "…"
                    : tile.money
                      ? money(tile.value(stats))
                      : show(tile.value(stats))
                }
                hint={stats ? tile.hint?.(stats, t) : undefined}
                accent={tile.accent}
              />
            ))}
            {/* The two sentences that keep the money group honest, kept with
                it rather than floating below every group: without them the
                takings figure looks low to anybody who remembers the old one,
                and the honest explanation is short. */}
            {group === "money" && (
              <div className="col-span-full">
                <p className="mt-1 text-xs text-ink-muted/80">{t.dashboard.revenueNote}</p>
                <p className="mt-1 text-xs text-ink-muted/80">{t.dashboard.cancelledNote}</p>
              </div>
            )}
          </Group>
        );
      })}

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

/** The heading for a tile group.
 *
 *  A lookup rather than a computed key so a missing translation is a compile
 *  error, which is the whole point of deriving `AdminDict` from the Uzbek
 *  dictionary. */
function groupTitle(group: TileGroup, t: ReturnType<typeof useAdminT>): string {
  switch (group) {
    case "orders":
      return t.dashboard.groupOrders;
    case "money":
      return t.dashboard.groupMoney;
    case "people":
      return t.dashboard.groupPeople;
    case "menu":
      return t.dashboard.groupMenu;
  }
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
