"use client";

// The courier's working screen: shift switch, live-location state and the
// orders assigned to them, each with a single button for the next step.

import { useCallback, useEffect, useState } from "react";
import RouteButtons from "@/components/map/RouteButtons";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import { useCourier } from "@/lib/courier";
import { formatPrice, formatTime, formatUzPhone } from "@/lib/format";
import { STATUS_LABEL } from "@/lib/orderStatus";
import { timeAgo } from "@/lib/orderFlow";
import { useAdminT } from "@/lib/i18n/admin";
import LangSwitch from "@/components/site/LangSwitch";
import ThemeToggle from "@/components/site/ThemeToggle";
import { formatDateTime } from "@/lib/orderFlow";
import type {
  CourierOrderRow,
  Restaurant,
  CourierStats,
  CourierStatus,
  Order,
} from "@/lib/types";



export default function CourierHomePage() {
  const router = useRouter();
  const {
    courier,
    loading,
    logout,
    setStatus,
    tracking,
    position,
    geoError,
    lastSentAt,
    pendingCount,
  } = useCourier();
  const [orders, setOrders] = useState<Order[]>([]);
  const [tab, setTab] = useState<"orders" | "earnings">("orders");
  // Which order's address the map picker is open for. A picker rather than one
  // hard-coded link: couriers have a navigator they already know, and that is
  // the one that gets them there fastest.
  const [mapFor, setMapFor] = useState<Order | null>(null);
  const [stats, setStats] = useState<CourierStats | null>(null);
  const [history, setHistory] = useState<CourierOrderRow[]>([]);
  // Needed for the arrival radius that gates the "delivered" button.
  const [restaurant, setRestaurant] = useState<Restaurant | null>(null);
  const t = useAdminT();
  const [busyId, setBusyId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const shifts: { value: CourierStatus; label: string; hint: string }[] = [
    { value: "off", label: t.courier.shiftOff, hint: t.courier.shiftOffHint },
    { value: "free", label: t.courier.shiftFree, hint: t.courier.shiftFreeHint },
    { value: "busy", label: t.courier.shiftBusy, hint: t.courier.shiftBusyHint },
  ];

  useEffect(() => {
    if (!loading && !courier) router.replace("/kuryer/login");
  }, [courier, loading, router]);

  useEffect(() => {
    api
      .getRestaurant()
      .then((r) => setRestaurant(r.restaurant))
      .catch(() => setRestaurant(null));
  }, []);

  const arrivalRadius = restaurant?.delivery.arrivalRadiusM ?? 0;

  // Straight-line distance in metres from the courier to an order address, or
  // null when either side is unknown. Mirrors the server-side check — the
  // server is still the authority, this only drives the button state.
  function metersTo(o: Order): number | null {
    if (!position || !o.address?.lat) return null;
    const R = 6371000;
    const dLat = ((o.address.lat - position.lat) * Math.PI) / 180;
    const dLng = ((o.address.lng - position.lng) * Math.PI) / 180;
    const a =
      Math.sin(dLat / 2) ** 2 +
      Math.cos((position.lat * Math.PI) / 180) *
        Math.cos((o.address.lat * Math.PI) / 180) *
        Math.sin(dLng / 2) ** 2;
    return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
  }

  // Why the courier may not close this order yet, or null when they may.
  function arrivalBlock(o: Order): string | null {
    if (o.status !== "on_the_way") return null;
    if (arrivalRadius <= 0 || o.type !== "delivery" || !o.address?.lat) {
      return null;
    }
    const m = metersTo(o);
    if (m === null) return t.courier.noFix;
    return m > arrivalRadius ? t.courier.tooFar(Math.round(m)) : null;
  }

  const load = useCallback(() => {
    if (!courier) return;
    api
      .courierOrders()
      .then(setOrders)
      .catch(() => setOrders([]));
  }, [courier]);

  useEffect(() => {
    load();
    const timer = setInterval(load, 20000);
    return () => clearInterval(timer);
  }, [load]);

  // Earnings are only fetched when the courier opens that tab.
  useEffect(() => {
    if (tab !== "earnings" || !courier) return;
    api.courierStats().then(setStats).catch(() => setStats(null));
    api.courierHistory().then(setHistory).catch(() => setHistory([]));
  }, [tab, courier]);

  async function advance(o: Order) {
    const next = o.status === "on_the_way" ? "delivered" : "on_the_way";
    setBusyId(o.id);
    setError(null);
    try {
      await api.courierAdvanceOrder(o.id, next);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.courier.actionFailed);
    } finally {
      setBusyId(null);
    }
  }

  if (loading || !courier) {
    return (
      <main className="flex min-h-dvh items-center justify-center">
        <p className="text-sm text-ink-muted">{t.common.loading}</p>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-lg px-4 pb-16 pt-6">
      <header className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <h1 className="truncate font-display text-xl font-bold">
            {courier.name}
          </h1>
          <p className="truncate text-xs text-ink-muted">@{courier.username}</p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <LangSwitch />
          <ThemeToggle />
          <button
            type="button"
            onClick={logout}
            className="btn-ghost shrink-0 gap-1.5 px-3 py-1.5 text-xs"
          >
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.8"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="h-3.5 w-3.5"
              aria-hidden
            >
              <path d="M15 17l5-5-5-5M20 12H9M12 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h6" />
            </svg>
            {t.courier.logout}
          </button>
        </div>
      </header>

      {/* Shift */}
      <section className="mt-5 rounded-3xl border border-line bg-surface p-4 shadow-card">
        <p className="text-sm font-semibold">{t.courier.statusTitle}</p>
        <div className="mt-3 grid grid-cols-3 gap-2">
          {shifts.map((s) => (
            <button
              key={s.value}
              type="button"
              onClick={() => setStatus(s.value).catch(() => {})}
              className={`rounded-2xl border px-2 py-3 text-center transition-colors ${
                courier.status === s.value
                  ? "border-brand bg-brand text-white"
                  : "border-line-strong hover:border-brand"
              }`}
            >
              <span className="block text-sm font-semibold">{s.label}</span>
              <span
                className={`mt-0.5 block text-[11px] ${
                  courier.status === s.value ? "text-white/80" : "text-ink-muted"
                }`}
              >
                {s.hint}
              </span>
            </button>
          ))}
        </div>

        {/* Location state */}
        <div className="mt-4 rounded-2xl bg-ink/[0.03] p-3 text-xs">
          {courier.status === "off" ? (
            <p className="text-ink-muted">{t.courier.startHint}</p>
          ) : geoError ? (
            <p className="text-red-600">
              {geoError === "denied"
                ? t.courier.geoDenied
                : geoError === "unsupported"
                  ? t.courier.geoUnsupported
                  : t.courier.geoFailed}
            </p>
          ) : (
            <p className="text-ink-muted">
              <span className="mr-1.5 inline-block h-2 w-2 rounded-full bg-emerald-500 align-middle" />
              {t.courier.sending}
              {tracking ? "" : t.courier.waitingSignal}
              {lastSentAt &&
                t.courier.lastSent(formatTime(lastSentAt))}
              {pendingCount > 0 && t.courier.queued(pendingCount)}
            </p>
          )}
        </div>

        {courier.status !== "off" && (
          <p className="mt-2 text-[11px] leading-relaxed text-ink-muted/80">
            {t.courier.keepOpen}
          </p>
        )}
      </section>

      {error && <p className="mt-4 text-sm text-red-600">{error}</p>}

      {/* Tabs */}
      <div className="mt-6 inline-flex rounded-full border border-line bg-ink/[0.03] p-1 text-sm font-semibold">
        {(
          [
            ["orders", t.courier.tabOrders],
            ["earnings", t.courier.tabEarnings],
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

      {/* Earnings */}
      {tab === "earnings" && (
        <section className="mt-4">
          <div className="grid grid-cols-2 gap-3">
            {(
              [
                [t.courier.today, stats?.today],
                [t.courier.week, stats?.week],
                [t.courier.month, stats?.month],
                [t.courier.allTime, stats?.all],
              ] as const
            ).map(([label, period]) => (
              <div
                key={label}
                className="rounded-3xl border border-line bg-surface p-4 shadow-card"
              >
                <p className="text-xs uppercase tracking-wider text-ink-muted">
                  {label}
                </p>
                <p className="mt-1 font-display text-xl font-bold text-brand">
                  {formatPrice(period?.earnings ?? 0)}
                </p>
                <p className="text-xs text-ink-muted">
                  {t.courier.deliveriesCount(period?.orders ?? 0)}
                </p>
              </div>
            ))}
          </div>

          {/* What is actually in the courier's pocket: everything collected
              less everything handed in. Showing the day's takings here was
              wrong — it stayed on screen after the money had been handed over,
              and it missed cash still owed from yesterday. */}
          {(stats?.cashInHand ?? 0) > 0 && (
            <p className="mt-3 rounded-2xl bg-amber-50 px-4 py-2.5 text-sm text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
              {t.courier.cashNote(formatPrice(stats!.cashInHand))}
            </p>
          )}
          {stats && stats.cashInHand === 0 && stats.cashSettled > 0 && (
            <p className="mt-3 rounded-2xl bg-emerald-50 px-4 py-2.5 text-sm text-emerald-800 dark:bg-emerald-500/10 dark:text-emerald-300">
              {t.courier.cashClear}
            </p>
          )}

          <h2 className="mt-6 text-sm font-semibold">
            {t.courier.historyTitle}
          </h2>
          {history.length === 0 ? (
            <p className="mt-3 rounded-3xl border border-dashed border-line-strong p-6 text-center text-sm text-ink-muted">
              {t.courier.historyEmpty}
            </p>
          ) : (
            <ul className="mt-3 space-y-2">
              {history.map((o) => (
                <li
                  key={o.id}
                  className="rounded-2xl border border-line bg-surface p-3 shadow-card"
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-semibold">#{o.number}</span>
                    <span className="font-bold tabular-nums text-brand">
                      +{formatPrice(o.earned)}
                    </span>
                  </div>
                  <p className="mt-1 text-xs text-ink-muted">
                    {formatDateTime(o.deliveredAt)}
                    {o.address?.text && ` · ${o.address.text}`}
                  </p>
                  <p className="text-xs text-ink-muted">
                    {t.receipt.total}: {formatPrice(o.total)} ·{" "}
                    {o.paymentMethod === "cash"
                      ? t.courier.cashToCollect(formatPrice(o.total))
                      : t.courier.paidOnline}
                  </p>
                </li>
              ))}
            </ul>
          )}
        </section>
      )}

      {/* Orders */}
      <section className={tab === "orders" ? "mt-4" : "hidden"}>
        <h2 className="text-sm font-semibold">
          {t.courier.myOrders(orders.length)}
        </h2>

        {orders.length === 0 ? (
          <p className="mt-3 rounded-3xl border border-dashed border-line-strong p-6 text-center text-sm text-ink-muted">
            {t.courier.noOrders}
          </p>
        ) : (
          <div className="mt-3 space-y-3">
            {orders.map((o) => (
              <article
                key={o.id}
                className="rounded-3xl border border-line bg-surface p-4 shadow-card"
              >
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-semibold">#{o.number}</span>
                  <span className="badge bg-ink/10 text-ink-muted">
                    {STATUS_LABEL[o.status]}
                  </span>
                  <span className="ml-auto font-bold tabular-nums">
                    {formatPrice(o.total)}
                  </span>
                </div>

                <p className="mt-2 text-sm">
                  {o.customer.name} ·{" "}
                  <a
                    href={`tel:+${o.customer.phone.replace(/\D/g, "")}`}
                    className="text-brand hover:underline"
                  >
                    {formatUzPhone(o.customer.phone)}
                  </a>
                </p>

                {o.type === "delivery" && o.address?.text && (
                  <p className="mt-1 text-sm text-ink-muted">
                    {o.address.text}
                    {o.address.comment && ` (${o.address.comment})`}
                  </p>
                )}

                <ul className="mt-2 space-y-0.5 text-xs text-ink-muted">
                  {o.items.map((it, i) => (
                    <li key={i}>
                      {it.name} × {it.qty}
                      {it.options?.length
                        ? ` — ${it.options.map((x) => x.choice).join(", ")}`
                        : ""}
                      {it.comment && (
                        <span className="block text-amber-700 dark:text-amber-300">
                          ✎ {it.comment}
                        </span>
                      )}
                    </li>
                  ))}
                </ul>

                <p className="mt-2 text-xs text-ink-muted">
                  {o.paymentMethod === "cash"
                    ? t.courier.cashToCollect(formatPrice(o.total))
                    : t.courier.paidOnline}{" "}
                  · {timeAgo(o.createdAt)}
                </p>

                {(() => {
                  const block = arrivalBlock(o);
                  const atDoor =
                    o.status === "on_the_way" &&
                    arrivalRadius > 0 &&
                    o.type === "delivery" &&
                    !!o.address?.lat &&
                    block === null;
                  return (
                    <>
                      {block && (
                        <p className="mt-2 rounded-xl bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
                          {block}
                        </p>
                      )}
                      {atDoor && (
                        <p className="mt-2 text-xs font-medium text-emerald-600">
                          ✓ {t.courier.atAddress}
                        </p>
                      )}
                    </>
                  );
                })()}

                <div className="mt-3 flex gap-2">
                  {o.address?.lat ? (
                    <button
                      type="button"
                      onClick={() => setMapFor(o)}
                      className="btn-ghost flex-1 py-2.5 text-center text-sm"
                    >
                      {t.courier.openMap}
                    </button>
                  ) : null}
                  {(o.status === "preparing" ||
                    o.status === "confirmed" ||
                    o.status === "on_the_way") && (
                    <button
                      type="button"
                      disabled={busyId === o.id || arrivalBlock(o) !== null}
                      onClick={() => advance(o)}
                      className="btn-primary flex-1 py-2.5 text-sm disabled:opacity-50"
                    >
                      {busyId === o.id
                        ? "..."
                        : o.status === "on_the_way"
                          ? t.courier.delivered
                          : t.courier.pickedUp}
                    </button>
                  )}
                </div>
              </article>
            ))}
          </div>
        )}
      </section>

      {/* Which navigator to open. A sheet from the bottom of the screen: this
          is a phone in someone's hand, on a doorstep, often one-handed. */}
      {mapFor?.address?.lat ? (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 sm:items-center sm:p-4"
          onClick={() => setMapFor(null)}
        >
          <div
            className="w-full max-w-md rounded-t-3xl border border-line bg-surface p-5 sm:rounded-3xl"
            onClick={(e) => e.stopPropagation()}
          >
            <p className="font-semibold">{t.courier.openMap}</p>
            <p className="mt-1 text-sm text-ink-muted">{mapFor.address.text}</p>
            <RouteButtons
              className="mt-4"
              title={t.courier.routeToCustomer}
              target={{
                lat: mapFor.address.lat,
                lng: mapFor.address.lng,
                label: mapFor.address.text,
              }}
            />
            <button
              type="button"
              onClick={() => setMapFor(null)}
              className="btn-ghost mt-4 w-full py-2.5 text-sm"
            >
              {t.common.cancel}
            </button>
          </div>
        </div>
      ) : null}
    </main>
  );
}
