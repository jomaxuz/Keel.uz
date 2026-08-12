"use client";

// One customer, everything about them: when they registered, how to reach
// them, their saved addresses, and every order with its full receipt — so a
// complaint ("I ordered a few days ago…") can be traced in seconds.

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatDate, formatPrice, formatUzPhone } from "@/lib/format";
import { STATUS_BADGE } from "@/lib/orderStatus";
import { ListScroll, Pager, usePaged } from "@/components/admin/PagedList";
import { useAdminT } from "@/lib/i18n/admin";
import CustomerNotes from "@/components/admin/CustomerNotes";
import { formatDateTime, timeAgo } from "@/lib/orderFlow";
import OrderReceipt from "@/components/admin/OrderReceipt";
import type { AdminUserDetail } from "@/lib/types";

const EMPTY_ORDERS: never[] = [];

export default function AdminUserPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const [data, setData] = useState<AdminUserDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [openOrder, setOpenOrder] = useState<string | null>(null);
  const [knownTags, setKnownTags] = useState<string[]>([]);
  // The receipts arrive with the profile, so paging waits for it.
  const paged = usePaged(data?.orders ?? EMPTY_ORDERS, 10);
  const t = useAdminT();

  useEffect(() => {
    api
      .adminUser(id)
      .then(setData)
      .catch(() => setData(null))
      .finally(() => setLoading(false));
    api.adminTags().then(setKnownTags).catch(() => setKnownTags([]));
  }, [id]);

  if (loading) {
    return <p className="py-10 text-center text-ink-muted/70">{t.common.loading}</p>;
  }
  if (!data) {
    return (
      <div className="py-10 text-center">
        <p className="text-ink-muted">{t.users.notFound}</p>
        <Link href="/admin/users" className="btn-ghost mt-4 px-4 py-2">
          {t.users.backToListBtn}
        </Link>
      </div>
    );
  }

  const { user, orders, stats } = data;
  const name =
    [user.firstName, user.lastName].filter(Boolean).join(" ") ||
    (user.phone ? formatUzPhone(user.phone) : "—");

  return (
    <div>
      <Link
        href="/admin/users"
        className="text-sm text-ink-muted hover:text-brand"
      >
        {t.users.backToList}
      </Link>

      {/* ---- header ---- */}
      <div className="mt-4 flex flex-wrap items-center gap-4">
        <span className="flex h-14 w-14 items-center justify-center rounded-full bg-brand-tint font-display text-xl font-bold text-brand">
          {name.charAt(0).toUpperCase()}
        </span>
        <div className="flex-1">
          <h1 className="font-display text-2xl font-bold">{name}</h1>
          <p className="text-sm text-ink-muted">
            {user.phone ? (
              <a href={`tel:+${user.phone}`} className="hover:text-brand">
                {formatUzPhone(user.phone)}
              </a>
            ) : (
              t.users.noPhone
            )}
          </p>
        </div>
      </div>

      {/* What the restaurant knows about them — read before picking up the
          phone, so it sits above the order history. */}
      <CustomerNotes
        user={user}
        knownTags={knownTags}
        onSaved={(next) => setData((d) => (d ? { ...d, user: next } : d))}
      />

      {/* ---- stats ---- */}
      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Stat
          label={t.users.registered}
          value={formatDate(user.createdAt)}
          hint={timeAgo(user.createdAt, t.common.timeAgo)}
        />
        <Stat
          label={t.users.ordersCount}
          value={String(stats.ordersCount)}
          hint={t.users.deliveredCancelled(stats.delivered, stats.cancelled)}
        />
        <Stat label={t.users.totalSpent} value={formatPrice(stats.ordersTotal)} />
        <Stat
          label={t.users.colLastOrder}
          value={
            stats.lastOrderAt
              ? formatDate(stats.lastOrderAt)
              : "—"
          }
          hint={stats.lastOrderAt ? timeAgo(stats.lastOrderAt, t.common.timeAgo) : undefined}
        />
      </div>

      {/* ---- saved addresses ---- */}
      <section className="mt-6 rounded-3xl border border-line bg-surface p-5 shadow-card">
        <h2 className="font-display text-lg font-bold">{t.users.savedAddresses}</h2>
        {(user.addresses ?? []).length === 0 ? (
          <p className="mt-2 text-sm text-ink-muted/70">
            {t.users.noSavedAddresses}
          </p>
        ) : (
          <ListScroll className="mt-3 space-y-2 pr-1 text-sm" max="max-h-64">
            {user.addresses!.map((a, i) => (
              <div
                key={i}
                className="flex flex-wrap items-center justify-between gap-2 rounded-2xl bg-ink/[0.03] px-4 py-2.5"
              >
                <span>
                  {a.label && <span className="font-semibold">{a.label}: </span>}
                  {a.text}
                  {a.comment && (
                    <span className="text-ink-muted"> · {a.comment}</span>
                  )}
                </span>
                {a.lat !== 0 && (
                  <a
                    href={`https://2gis.uz/geo/${a.lng},${a.lat}`}
                    target="_blank"
                    rel="noreferrer"
                    className="text-xs text-brand hover:underline"
                  >
                    {t.users.onMap}
                  </a>
                )}
              </div>
            ))}
          </ListScroll>
        )}
      </section>

      {/* ---- orders with receipts ---- */}
      <section className="mt-6">
        <h2 className="font-display text-lg font-bold">
          {t.users.historyTitle(orders.length)}
        </h2>
        <div className="mt-3 rounded-3xl border border-line bg-surface shadow-card">
          <ListScroll className="space-y-3 p-3" max="max-h-[70vh]">
          {orders.length === 0 && (
            <p className="p-6 text-center text-sm text-ink-muted/70">
              {t.users.neverOrdered}
            </p>
          )}
          {paged.pageItems.map((o) => {
            const open = openOrder === o.id;
            return (
              <div
                key={o.id}
                className="rounded-3xl border border-line bg-surface shadow-card"
              >
                <button
                  type="button"
                  onClick={() => setOpenOrder(open ? null : o.id)}
                  className="flex w-full flex-wrap items-center gap-3 p-4 text-left"
                >
                  <div className="min-w-[200px] flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-semibold">#{o.number}</span>
                      <span className={`badge ${STATUS_BADGE[o.status]}`}>
                        {t.status[o.status]}
                      </span>
                    </div>
                    <p className="mt-1 text-xs text-ink-muted">
                      {formatDateTime(o.createdAt)} ·{" "}
                      {o.type === "delivery"
                        ? t.dashboard.delivery
                        : o.type === "dinein"
                          ? t.receipt.tableLine(o.tableNumber ?? "—")
                          : t.dashboard.pickup}{" "}
                      · {t.orders.dishes(o.items.length)}
                    </p>
                  </div>
                  <span className="font-bold tabular-nums">
                    {formatPrice(o.total)}
                  </span>
                  <span className="text-xs text-ink-muted">
                    {open ? "▲" : "▼"}
                  </span>
                </button>

                {open && (
                  <div className="border-t border-line p-4">
                    <OrderReceipt order={o} showCustomer={false} />
                    <div className="mt-4 flex flex-wrap gap-3 text-xs">
                      <Link
                        href={`/order/${o.number}`}
                        target="_blank"
                        className="btn-ghost px-3 py-1.5"
                      >
                        {t.orders.customerView}
                      </Link>
                      <Link
                        href={`/admin/orders?q=${o.number}`}
                        className="btn-ghost px-3 py-1.5"
                      >
                        {t.users.openInOrders}
                      </Link>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
          </ListScroll>
          <Pager
            page={paged.page}
            pageCount={paged.pageCount}
            from={paged.from}
            to={paged.to}
            total={paged.total}
            onPage={paged.setPage}
          />
        </div>
      </section>
    </div>
  );
}

function Stat({
  label,
  value,
  hint,
}: {
  label: string;
  value: string;
  hint?: string;
}) {
  return (
    <div className="rounded-3xl border border-line bg-surface p-4 shadow-card">
      <p className="text-xs uppercase tracking-wider text-ink-muted">{label}</p>
      <p className="mt-1 font-display text-xl font-bold">{value}</p>
      {hint && <p className="text-xs text-ink-muted">{hint}</p>}
    </div>
  );
}
