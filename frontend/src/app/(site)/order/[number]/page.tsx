"use client";

import { use, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { api, ApiError } from "@/lib/api";
import { formatDate, formatPrice, formatTime } from "@/lib/format";
import DeliveredCelebration from "@/components/order/DeliveredCelebration";
import RateOrder from "@/components/order/RateOrder";
import LiveMap, { type MapPoint } from "@/components/map/LiveMap";
import RouteButtons from "@/components/map/RouteButtons";
import CallLink from "@/components/site/CallLink";
import { useI18n } from "@/lib/i18n/client";
import type { OrderStatus, OrderTrack, Restaurant } from "@/lib/types";

// Delivery status pipeline (cancelled is handled separately); labels come from
// the active dictionary.
const STEPS: OrderStatus[] = [
  "pending",
  "confirmed",
  "preparing",
  "on_the_way",
  "delivered",
];

export default function OrderTrackPage({
  params,
}: {
  params: Promise<{ number: string }>;
}) {
  const { number } = use(params);
  const { lang, t } = useI18n();
  const [order, setOrder] = useState<OrderTrack | null>(null);
  // Only needed for pickup orders — where to drive to.
  const [restaurant, setRestaurant] = useState<Restaurant | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    try {
      const data = await api.trackOrder(number);
      setOrder(data);
      setError(null);
    } catch (err) {
      setError(
        err instanceof ApiError && err.status === 404
          ? t.order.notFound
          : t.order.loadFailed,
      );
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [number]);

  useEffect(() => {
    api
      .getRestaurant()
      .then((r) => setRestaurant(r.restaurant))
      .catch(() => setRestaurant(null));
  }, []);

  useEffect(() => {
    refresh();
    // No realtime courier tracking (CLAUDE.md §7) — poll every 20s.
    const id = setInterval(refresh, 20000);
    return () => clearInterval(id);
  }, [refresh]);

  if (loading) {
    return (
      <main className="container-page py-24 text-center text-ink-muted">
        {t.common.loading}
      </main>
    );
  }

  if (error || !order) {
    return (
      <main className="container-page py-24 text-center">
        <h1 className="font-display text-2xl font-bold">{t.order.title(number)}</h1>
        <p className="mt-3 text-ink-muted">{error}</p>
        <Link href="/menu" className="btn-primary mt-6 px-6 py-3">
          {t.common.goToMenu}
        </Link>
      </main>
    );
  }

  const cancelled = order.status === "cancelled";
  const delivered = order.status === "delivered";
  const activeIdx = STEPS.indexOf(order.status);

  return (
    <main className="container-page max-w-2xl py-12">
      {delivered && <DeliveredCelebration />}

      <div className="rounded-3xl border border-line bg-surface shadow-card p-6 sm:p-8">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p className="text-sm text-ink-muted">{t.order.numberLabel}</p>
            <h1 className="font-display text-2xl font-bold">#{order.number}</h1>
          </div>
          <span
            className={`rounded-full px-3 py-1 text-sm font-semibold ${
              cancelled
                ? "bg-rose-100 text-brand dark:bg-rose-500/15 dark:text-rose-300"
                : "bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300"
            }`}
          >
            {t.order.steps[order.status]}
          </span>
        </div>

        {/* A pre-order, and the time it was placed for.
            First thing under the number, and above the payment note, because it
            is the answer to the question this page is being opened with: an
            order that stays "qabul qilindi" for six hours looks abandoned, and
            the guest's next move is a phone call the restaurant does not need. */}
        {order.scheduledAt && !cancelled && (
          <p className="mt-6 rounded-xl bg-brand-tint/60 px-4 py-3 text-sm font-semibold text-brand-dark">
            {t.order.scheduledFor(
              `${formatDate(order.scheduledAt)} ${formatTime(order.scheduledAt)}`,
            )}
          </p>
        )}

        {/* Where the money stands.
            Shown above the cooking steps on purpose: an order waiting on a
            half-finished card payment looks, from the kitchen's silence,
            exactly like an order that was ignored — and the guest's next move
            is a phone call unless the page says so plainly. */}
        {order.paymentStatus === "pending" && !cancelled && (
          <div className="mt-6 rounded-xl border border-amber-500/40 bg-amber-500/10 p-4">
            <p className="text-sm font-semibold text-amber-700 dark:text-amber-300">
              {t.order.payPending}
            </p>
            <p className="mt-1 text-sm text-ink-muted">{t.order.payPendingHint}</p>
            {order.payUrl && (
              <a href={order.payUrl} className="btn btn-primary mt-3 inline-flex">
                {t.order.payNow}
              </a>
            )}
          </div>
        )}
        {order.paymentStatus === "paid" && (
          <p className="mt-6 rounded-xl bg-emerald-500/10 px-4 py-3 text-sm font-semibold text-emerald-700 dark:text-emerald-300">
            {t.order.paid}
          </p>
        )}
        {order.paymentStatus === "refunded" && (
          <p className="mt-6 rounded-xl bg-ink/5 px-4 py-3 text-sm text-ink-muted">
            {t.order.refunded}
          </p>
        )}

        {cancelled ? (
          <div className="mt-8 rounded-lg bg-rose-50 px-4 py-3 text-sm text-brand dark:bg-rose-500/10">
            <p>{t.order.cancelledText}</p>
            {/* The restaurant's own words — why this order was cancelled. */}
            {order.cancelReason && (
              <p className="mt-2">
                <span className="font-semibold">
                  {t.order.cancelReasonLabel}:{" "}
                </span>
                {order.cancelReason}
              </p>
            )}
          </div>
        ) : (
          <ol className="mt-8 space-y-4">
            {STEPS.map((step, i) => {
              const done = i <= activeIdx;
              const current = i === activeIdx;
              return (
                <li key={step} className="flex items-center gap-3">
                  <span
                    className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold ${
                      done
                        ? "bg-brand text-white"
                        : "bg-ink/5 text-ink-muted/70"
                    }`}
                  >
                    {done ? "✓" : i + 1}
                  </span>
                  <span
                    className={
                      current
                        ? "font-semibold"
                        : done
                          ? "text-ink-soft"
                          : "text-ink-muted/70"
                    }
                  >
                    {t.order.steps[step]}
                  </span>
                </li>
              );
            })}
          </ol>
        )}

        {/* Where the courier is, while the order is on its way. */}
        {/* The rating, right where the guest already is once the food has
            arrived. Asked once; the component remembers. */}
        {delivered && <RateOrder number={order.number} />}

        {order.status === "on_the_way" && order.courier?.location && (
          <div className="mt-8 border-t border-line pt-6">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="text-sm font-semibold">
                {t.order.courierOnTheWay}
                {order.courier.name && ` — ${order.courier.name}`}
              </p>
              {/* A real button, not an underlined word. The guest reaching for
                  this is usually standing at the door wondering where the food
                  is — it has to be the obvious thing to press, and big enough
                  to hit on a phone. */}
              {order.courier.phone && (
                <CallLink
                  phone={order.courier.phone}
                  className="btn-primary inline-flex items-center gap-2 px-4 py-2.5 text-sm"
                >
                  <svg
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    className="h-4 w-4"
                    aria-hidden
                  >
                    <path d="M22 16.9v3a2 2 0 0 1-2.2 2 19.8 19.8 0 0 1-8.6-3.1 19.5 19.5 0 0 1-6-6A19.8 19.8 0 0 1 2.1 4.2 2 2 0 0 1 4.1 2h3a2 2 0 0 1 2 1.7c.1 1 .4 1.9.7 2.8a2 2 0 0 1-.5 2.1L8.1 9.9a16 16 0 0 0 6 6l1.3-1.2a2 2 0 0 1 2.1-.5c.9.3 1.8.6 2.8.7a2 2 0 0 1 1.7 2Z" />
                  </svg>
                  {t.order.callCourier}
                </CallLink>
              )}
            </div>
            <div className="mt-3">
              <LiveMap
                points={
                  [
                    {
                      id: "courier",
                      lat: order.courier.location.lat,
                      lng: order.courier.location.lng,
                      label: order.courier.name || t.order.courier,
                      kind: "courier",
                    },
                    order.address?.lat
                      ? {
                          id: "destination",
                          lat: order.address.lat,
                          lng: order.address.lng,
                          label: t.order.yourAddress,
                          kind: "destination",
                        }
                      : null,
                  ].filter(Boolean) as MapPoint[]
                }
                fallbackCenter={{
                  lat: order.courier.location.lat,
                  lng: order.courier.location.lng,
                }}
                autoFit="always"
                className="h-64 w-full"
              />
            </div>
            <p className="mt-2 text-xs text-ink-muted">{t.order.courierHint}</p>
          </div>
        )}

        {/* Pickup orders: build the route in the customer's own map app. */}
        {order.type === "pickup" && !cancelled && restaurant?.address?.lat && (
          <div className="mt-8 border-t border-line pt-6">
            <p className="text-sm font-medium">{restaurant.address.text}</p>
            <RouteButtons
              className="mt-3"
              target={{
                lat: restaurant.address.lat,
                lng: restaurant.address.lng,
                label: restaurant.name,
              }}
            />
          </div>
        )}

        <div className="mt-8 flex items-center justify-between border-t border-line pt-6 text-sm">
          <span className="text-ink-muted">
            {order.type === "delivery"
              ? t.order.delivery
              : order.type === "dinein"
                ? t.table.receiptLine(order.tableNumber ?? "")
                : t.order.pickup}
          </span>
          <span className="font-display text-lg font-bold">{formatPrice(order.total, undefined, lang)}</span>
        </div>
      </div>

      <div className="mt-4 flex items-center justify-between text-sm">
        <button
          type="button"
          onClick={refresh}
          className="text-brand hover:underline"
        >
          {t.common.refresh}
        </button>
        <Link href="/menu" className="text-ink-muted hover:text-brand">
          {t.common.backToMenu}
        </Link>
      </div>
    </main>
  );
}
