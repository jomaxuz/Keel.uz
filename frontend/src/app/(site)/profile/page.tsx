"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { usePaged } from "@/lib/paged";
import { useUser } from "@/lib/user";
import { formatDateTime, formatPrice, formatUzPhone } from "@/lib/format";
import { STATUS_BADGE } from "@/lib/orderStatus";
import { useI18n } from "@/lib/i18n/client";
import PhoneLogin from "@/components/auth/PhoneLogin";
import FavoriteDishes from "@/components/site/FavoriteDishes";
import ProfileDetails from "@/components/auth/ProfileDetails";
import PushToggle from "@/components/site/PushToggle";
import Pager from "@/components/site/Pager";
import type {
  LoyaltyInfo,
  Order,
  Reservation,
  ReservationStatus,
} from "@/lib/types";

/**
 * Rows per page on the profile. Five, not the panel's twenty: this is a phone
 * screen with three stacked lists, and the point of the pager is that the next
 * section stays reachable without a long scroll.
 */
const PAGE_SIZE = 5;

/**
 * Rows per page in the points ledger. Eight, because these are single compact
 * lines rather than cards — and because eight is exactly what the section used
 * to show, so the card keeps the height it already had.
 */
const POINTS_PAGE = 8;

/** Stable identity while the ledger is loading, so the slice memo survives. */
const NO_POINTS: LoyaltyInfo["transactions"] = [];

export default function ProfilePage() {
  const { user, loading, logout } = useUser();
  const { lang, t } = useI18n();
  const [orders, setOrders] = useState<Order[]>([]);
  const [ordersLoading, setOrdersLoading] = useState(true);
  const [bookings, setBookings] = useState<Reservation[]>([]);
  const [bookingsLoading, setBookingsLoading] = useState(true);
  const [loyalty, setLoyalty] = useState<LoyaltyInfo | null>(null);

  // Both histories only grow, and a guest who orders weekly has hundreds of
  // rows within a year — rendered in full they push the bookings section (and
  // everything else) off the bottom of the page. Paged, the profile stays the
  // same height on the first visit and the thousandth.
  const ordersRef = useRef<HTMLHeadingElement>(null);
  const bookingsRef = useRef<HTMLHeadingElement>(null);
  const pagedOrders = usePaged(orders, PAGE_SIZE);
  const pagedBookings = usePaged(bookings, PAGE_SIZE);

  // The points ledger, same problem. It used to be cut at the eight most recent
  // rows — which kept the page short, but it also meant the rest of the guest's
  // history simply did not exist on any screen: the one question a cashback
  // ledger answers is "where did my points go", and it is usually asked about a
  // row older than the last eight. Paged, every entry is reachable and the
  // section is no taller than it was.
  const pointsRef = useRef<HTMLElement>(null);
  const pagedPoints = usePaged(loyalty?.transactions ?? NO_POINTS, POINTS_PAGE);

  useEffect(() => {
    if (!user) return;
    api
      .userOrders()
      .then(setOrders)
      .catch(() => setOrders([]))
      .finally(() => setOrdersLoading(false));
    api
      .userReservations()
      .then(setBookings)
      .catch(() => setBookings([]))
      .finally(() => setBookingsLoading(false));
    api
      .userLoyalty()
      .then(setLoyalty)
      .catch(() => setLoyalty(null));
  }, [user]);

  if (loading) {
    return (
      <main className="container-page py-24 text-center text-ink-muted/70">
        {t.common.loading}
      </main>
    );
  }

  if (!user) {
    return (
      <main className="container-page flex min-h-[60vh] items-center justify-center py-12">
        <div className="w-full max-w-sm rounded-3xl border border-line bg-surface shadow-card p-8 text-center">
          <h1 className="font-display text-xl font-bold">{t.login.requiredTitle}</h1>
          <p className="mt-2 text-sm text-ink-muted">{t.login.requiredText}</p>
          <div className="mt-6">
            <PhoneLogin />
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="container-page max-w-3xl py-10">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <span className="flex h-14 w-14 items-center justify-center rounded-full bg-brand-tint font-display text-xl font-bold text-brand">
            {(user.firstName || user.phone || "?").charAt(0).toUpperCase()}
          </span>
          <div>
            <h1 className="font-display text-2xl font-bold">
              {[user.firstName, user.lastName].filter(Boolean).join(" ") ||
                formatUzPhone(user.phone)}
            </h1>
            {user.phone ? (
              <p className="text-sm text-ink-muted">{formatUzPhone(user.phone)}</p>
            ) : null}
          </div>
        </div>
        <button
          type="button"
          onClick={logout}
          className="btn-ghost shrink-0 gap-2 px-4 py-2 text-sm"
        >
          <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.8"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="h-4 w-4"
              aria-hidden
            >
              <path d="M15 17l5-5-5-5M20 12H9M12 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h6" />
            </svg>
          {t.nav.logout}
        </button>
      </div>

      <ProfileDetails />

      {/* Browser notifications. Under the details rather than at the top: it is
          an offer, not something the guest came here to fix. Renders nothing at
          all on a browser that cannot do it. */}
      <div className="mt-6">
        <PushToggle />
      </div>

      {/* The hearts, as cards. Above the order history because a favourite is what the
          guest wants next, and the history is what they already did. */}
      <FavoriteDishes currency="UZS" />

      {/* Cashback. Hidden entirely when the restaurant does not run one — an
          empty balance card is worse than no card. */}
      {loyalty?.enabled && (
        <section
          ref={pointsRef}
          className="mt-8 scroll-mt-24 rounded-3xl border border-line bg-surface p-6 shadow-card"
        >
          <div className="flex flex-wrap items-end justify-between gap-3">
            <div>
              <p className="text-sm text-ink-muted">{t.profile.pointsTitle}</p>
              <p className="font-display text-3xl font-bold text-brand">
                {formatPrice(loyalty.balance, undefined, lang)}
              </p>
            </div>
            <p className="text-sm text-ink-muted">
              {t.profile.pointsRule(
                loyalty.earnPercent,
                loyalty.maxRedeemPercent,
              )}
            </p>
          </div>

          {loyalty.transactions.length > 0 && (
            <ul className="mt-5 divide-y divide-line border-t border-line">
              {pagedPoints.pageItems.map((x) => (
                <li key={x.id} className="flex items-center gap-3 py-2 text-sm">
                  <div className="min-w-0 flex-1">
                    <p className="truncate">
                      {t.profile.pointsKind[x.kind]}
                      {x.orderNumber ? ` · #${x.orderNumber}` : ""}
                    </p>
                    <p className="text-xs text-ink-muted">
                      {new Date(x.at).toLocaleDateString(t.locale)}
                    </p>
                  </div>
                  <span
                    className={`shrink-0 tabular-nums font-semibold ${
                      x.points > 0
                        ? "text-emerald-600 dark:text-emerald-400"
                        : "text-ink-muted"
                    }`}
                  >
                    {x.points > 0 ? "+" : "−"}
                    {formatPrice(Math.abs(x.points), undefined, lang)}
                  </span>
                </li>
              ))}
            </ul>
          )}
          <Pager
            className="mt-3"
            page={pagedPoints.page}
            pageCount={pagedPoints.pageCount}
            from={pagedPoints.from}
            to={pagedPoints.to}
            total={pagedPoints.total}
            onPage={pagedPoints.setPage}
            anchorRef={pointsRef}
          />
        </section>
      )}

      <h2
        ref={ordersRef}
        className="mt-10 scroll-mt-24 font-display text-lg font-bold"
      >
        {t.profile.ordersTitle}
      </h2>
      <div className="mt-4 space-y-3">
        {ordersLoading ? (
          <p className="py-8 text-center text-ink-muted/70">{t.common.loading}</p>
        ) : orders.length === 0 ? (
          <div className="rounded-3xl border border-line bg-surface shadow-card py-10 text-center">
            <p className="text-ink-muted">{t.profile.empty}</p>
            <Link href="/menu" className="btn-primary mt-4 px-5 py-2.5">
              {t.common.goToMenu}
            </Link>
          </div>
        ) : (
          pagedOrders.pageItems.map((o) => (
            <Link
              key={o.id}
              href={`/order/${o.number}`}
              className="flex items-center justify-between rounded-3xl border border-line bg-surface shadow-card p-4 transition-colors hover:border-brand"
            >
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-semibold">#{o.number}</span>
                  <span
                    className={`rounded-full px-2 py-0.5 text-xs font-semibold ${STATUS_BADGE[o.status]}`}
                  >
                    {t.order.steps[o.status]}
                  </span>
                </div>
                <p className="mt-1 text-sm text-ink-muted">
                  {new Date(o.createdAt).toLocaleDateString(t.locale)} ·{" "}
                  {o.type === "delivery" ? t.profile.delivery : t.profile.pickup}
                </p>
                {o.status === "cancelled" && o.cancelReason && (
                  <p className="mt-1 text-xs text-brand">
                    {t.order.cancelReasonLabel}: {o.cancelReason}
                  </p>
                )}
              </div>
              <span className="font-bold">{formatPrice(o.total, undefined, lang)}</span>
            </Link>
          ))
        )}
        <Pager
          page={pagedOrders.page}
          pageCount={pagedOrders.pageCount}
          from={pagedOrders.from}
          to={pagedOrders.to}
          total={pagedOrders.total}
          onPage={pagedOrders.setPage}
          anchorRef={ordersRef}
        />
      </div>

      {/* ---- table bookings ---- */}
      <h2
        ref={bookingsRef}
        className="mt-10 scroll-mt-24 font-display text-lg font-bold"
      >
        {t.booking.myBookings}
      </h2>
      <div className="mt-4 space-y-3">
        {bookingsLoading ? (
          <p className="py-8 text-center text-ink-muted/70">{t.common.loading}</p>
        ) : bookings.length === 0 ? (
          <div className="rounded-3xl border border-line bg-surface p-8 text-center shadow-card">
            <p className="text-ink-muted">{t.profile.noBookings}</p>
            <Link href="/bron" className="btn-primary mt-4 px-5 py-2.5">
              {t.booking.title}
            </Link>
          </div>
        ) : (
          pagedBookings.pageItems.map((r) => (
            <div
              key={r.id}
              className="rounded-3xl border border-line bg-surface p-4 shadow-card"
            >
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-semibold">#{r.number}</span>
                <span
                  className={`rounded-full px-2 py-0.5 text-xs font-semibold ${BOOKING_BADGE[r.status]}`}
                >
                  {bookingStatusLabel(r.status, t)}
                </span>
                <span className="ml-auto text-sm font-semibold">
                  {t.booking.tableLabel(r.tableNumber)}
                </span>
              </div>
              <p className="mt-1 text-sm text-ink-muted">
                {formatDateTime(r.at)}{" "}
                · {t.booking.guestsCount(r.guests)}
              </p>
              {r.comment && (
                <p className="mt-1 text-xs italic text-ink-muted/80">
                  “{r.comment}”
                </p>
              )}
              {/* The restaurant's own words, exactly as on a cancelled order. */}
              {r.status === "cancelled" && r.cancelReason && (
                <p className="mt-1 text-xs text-brand">
                  {t.booking.cancelReason}: {r.cancelReason}
                </p>
              )}
            </div>
          ))
        )}
        <Pager
          page={pagedBookings.page}
          pageCount={pagedBookings.pageCount}
          from={pagedBookings.from}
          to={pagedBookings.to}
          total={pagedBookings.total}
          onPage={pagedBookings.setPage}
          anchorRef={bookingsRef}
        />
      </div>
    </main>
  );
}

const BOOKING_BADGE: Record<ReservationStatus, string> = {
  pending: "bg-amber-100 text-amber-800 dark:bg-amber-500/15 dark:text-amber-300",
  confirmed: "bg-sky-100 text-sky-800 dark:bg-sky-500/15 dark:text-sky-300",
  seated:
    "bg-emerald-100 text-emerald-800 dark:bg-emerald-500/15 dark:text-emerald-300",
  done: "bg-ink/10 text-ink-soft",
  cancelled: "bg-rose-100 text-brand dark:bg-rose-500/15 dark:text-rose-300",
};

function bookingStatusLabel(
  status: ReservationStatus,
  t: ReturnType<typeof useI18n>["t"],
): string {
  switch (status) {
    case "confirmed":
      return t.booking.statusConfirmed;
    case "seated":
      return t.booking.statusSeated;
    case "done":
      return t.booking.statusDone;
    case "cancelled":
      return t.booking.statusCancelled;
    default:
      return t.booking.statusPending;
  }
}
