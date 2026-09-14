"use client";

// Orders board for the restaurant. Design goal: a normal order needs ONE tap
// per stage ("Tasdiqlash" → "Tayyorlashni boshlash" → …), not a dropdown, and
// the list refreshes itself so nobody has to press reload while cooking.

import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatDate, formatPrice, formatTime } from "@/lib/format";
import { hasId, realId } from "@/lib/id";
import { ORDER_STATUSES, STATUS_BADGE, STATUS_ROW } from "@/lib/orderStatus";
import { nextActionLabel, nextStatus, timeAgo } from "@/lib/orderFlow";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import CallDeliveryModal from "@/components/admin/CallDeliveryModal";
import CancelOrderModal from "@/components/admin/CancelOrderModal";
import type { BranchLoad, Courier } from "@/lib/types";
import ChannelBadge from "@/components/admin/ChannelBadge";
import PhoneOrderButton from "@/components/admin/PhoneOrderButton";
import OrderReceipt from "@/components/admin/OrderReceipt";
import { ORDERS_CHANGED_EVENT } from "@/components/admin/AlertBell";
import { ListScroll, Pager, usePaged } from "@/components/admin/PagedList";
import type { Order, OrderStatus } from "@/lib/types";
import { useAsk } from "@/components/ui/Ask";

const REFRESH_MS = 20000;

// Statuses that still need someone's attention (the "active" tab).
const ACTIVE: OrderStatus[] = [
  "pending",
  "confirmed",
  "preparing",
  "on_the_way",
];

// "preorders" is not a status and deliberately sits alongside them: an order
// placed for later has a perfectly ordinary status (usually `confirmed`) and
// what makes it its own tab is a different question — not "where is this in the
// kitchen" but "what is this branch due to cook, and when". Which is also why
// the server sorts that list by the time it is wanted rather than newest-first.
type Filter = OrderStatus | "all" | "active" | "preorders";

export default function AdminOrdersPage() {
  const search = useSearchParams();
  const { tell } = useAsk();
  const [orders, setOrders] = useState<Order[]>([]);
  // Arriving with a search — from the feedback screen, or a pasted receipt
  // number — means looking for one particular order, and that order is usually
  // finished. Opening on "active" would hide exactly what was asked for.
  const [filter, setFilter] = useState<Filter>(
    // ?tab=preorders is where the "pre-order is due" banner sends the panel:
    // the order it is about was placed hours ago and is nowhere near the top
    // of the ordinary, newest-first list.
    search.get("tab") === "preorders"
      ? "preorders"
      : search.get("q")
        ? "all"
        : "active",
  );
  // Seeded from the URL so a link like /admin/orders?q=AB12-3456 — the one the
  // feedback screen hands out — actually lands on that order instead of an
  // empty list.
  const [q, setQ] = useState(search.get("q") ?? "");
  const [loading, setLoading] = useState(true);
  const [openId, setOpenId] = useState<string | null>(null);
  // 12 receipts fill the scroll block without the pager ever going quiet.
  const paged = usePaged(orders, 12);
  // Delivery orders need someone to carry them — the list is small, so it is
  // fetched once and reused in every row.
  const [couriers, setCouriers] = useState<Courier[]>([]);
  const t = useAdminT();
  const scope = useAdminScope();
  // What each kitchen is holding. Only fetched — and only drawn — when the
  // company has more than one, because on a single-branch install the answer
  // is "all of it" and the strip would be a row of numbers with no decision
  // attached to them.
  const multiBranch = scope.brandBranches.length > 1;
  const [branchLoad, setBranchLoad] = useState<BranchLoad[]>([]);
  const [moving, setMoving] = useState<string | null>(null);
  // Order handed to an outside delivery service from the modal.
  const [calling, setCalling] = useState<Order | null>(null);
  // Cancelling always goes through the modal — the reason is mandatory.
  const [cancelling, setCancelling] = useState<Order | null>(null);
  const [saving, setSaving] = useState<string | null>(null);
  const [newCount, setNewCount] = useState(0);
  const [lastSync, setLastSync] = useState<Date | null>(null);
  const knownIds = useRef<Set<string> | null>(null);

  const load = useCallback(
    async (opts?: { silent?: boolean }) => {
      if (!opts?.silent) setLoading(true);
      try {
        const preorders = filter === "preorders";
        const status =
          filter === "all" || filter === "active" || preorders
            ? undefined
            : filter;
        const rows = await api.adminOrders({
          status,
          q: q.trim() || undefined,
          // The server does this one: the sort is part of the answer, and a
          // page that re-sorted the 200 rows it was given would disagree with
          // it the moment there were more than 200.
          scheduled: preorders || undefined,
        });
        const visible =
          filter === "active"
            ? rows.filter((o) => ACTIVE.includes(o.status))
            : rows;

        // Count orders that appeared since the previous poll.
        if (knownIds.current) {
          const fresh = visible.filter((o) => !knownIds.current!.has(o.id));
          if (fresh.length > 0) setNewCount((n) => n + fresh.length);
        }
        knownIds.current = new Set(visible.map((o) => o.id));
        setOrders(visible);
        setLastSync(new Date());
      } catch {
        setOrders([]);
      } finally {
        setLoading(false);
      }
    },
    [filter, q],
  );

  // Reload on filter/search change (debounced while typing).
  useEffect(() => {
    knownIds.current = null;
    setNewCount(0);
    const id = setTimeout(() => load(), q ? 300 : 0);
    return () => clearTimeout(id);
  }, [filter, q, load]);

  // Courier list for the assignment dropdown.
  useEffect(() => {
    api
      .adminCouriers()
      .then(setCouriers)
      .catch(() => setCouriers([]));
  }, []);

  // Background polling — new orders appear on their own.
  useEffect(() => {
    const id = setInterval(() => load({ silent: true }), REFRESH_MS);
    return () => clearInterval(id);
  }, [load]);

  // Kitchen load, on the same beat as the order list.
  useEffect(() => {
    if (!multiBranch) return;
    const read = () =>
      api
        .adminBranchLoad()
        .then(setBranchLoad)
        .catch(() => setBranchLoad([]));
    read();
    const id = setInterval(read, REFRESH_MS);
    return () => clearInterval(id);
  }, [multiBranch, scope.scopeKey]);

  // Hand an order to another kitchen. The server refuses a branch that has run
  // out of something in it, and says which dish — so the reply is worth
  // showing rather than swallowing.
  async function moveBranch(order: Order, branchId: string) {
    setMoving(order.id);
    try {
      const next = await api.moveOrderBranch(order.id, branchId);
      setOrders((prev) =>
        prev.map((o) =>
          o.id === order.id
            ? {
                ...o,
                branchId: next.branchId,
                courierId: undefined,
                courierName: undefined,
              }
            : o,
        ),
      );
    } catch (e) {
      void tell({
        title: e instanceof Error ? e.message : t.common.saveFailed,
      });
    } finally {
      setMoving(null);
    }
  }

  async function changeStatus(
    order: Order,
    status: OrderStatus,
    reason?: string,
  ) {
    // Never cancel without a reason: the request is routed through the modal.
    if (status === "cancelled" && !reason) {
      setCancelling(order);
      return;
    }
    setSaving(order.id);
    try {
      await api.updateOrderStatus(order.id, status, reason);
      // The alarm is still ringing until the server says the order is no
      // longer waiting, and it asks every few seconds. Telling it now is the
      // difference between "accepted" and "accepted, and it kept blaring".
      window.dispatchEvent(new Event(ORDERS_CHANGED_EVENT));
      setOrders((prev) =>
        filter === "active" && !ACTIVE.includes(status)
          ? prev.filter((o) => o.id !== order.id)
          : prev.map((o) =>
              o.id === order.id
                ? {
                    ...o,
                    status,
                    cancelReason: status === "cancelled" ? reason : undefined,
                  }
                : o,
            ),
      );
    } catch {
      void tell({ title: t.orders.statusFailed });
    } finally {
      setSaving(null);
    }
  }

  async function assignCourier(order: Order, courierId: string) {
    setSaving(order.id);
    try {
      await api.assignCourier(order.id, courierId);
      const c = couriers.find((x) => x.id === courierId);
      setOrders((prev) =>
        prev.map((o) =>
          o.id === order.id
            ? { ...o, courierId: courierId || undefined, courierName: c?.name }
            : o,
        ),
      );
    } catch {
      void tell({ title: t.orders.assignFailed });
    } finally {
      setSaving(null);
    }
  }

  const activeCount = orders.filter((o) => ACTIVE.includes(o.status)).length;

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="font-display text-2xl font-bold">
          {t.orders.title}
          {filter === "active" && activeCount > 0 && (
            <span className="badge-brand ml-2 align-middle">{activeCount}</span>
          )}
        </h1>
        <div className="flex flex-wrap items-center gap-3">
          {/* An operator with a customer on the line is usually watching this
              screen, not the call log — so the phone order starts here too, with
              the same pipeline behind it. */}
          <PhoneOrderButton onCreated={() => load({ silent: true })} />
          {newCount > 0 && (
            <button
              type="button"
              onClick={() => setNewCount(0)}
              className="badge bg-emerald-100 text-emerald-700"
            >
              +{t.orders.newArrived(newCount)}
            </button>
          )}
          <span className="text-xs text-ink-muted">
            {lastSync ? t.orders.updatedAt(formatTime(lastSync)) : ""}
          </span>
          <button
            type="button"
            onClick={() => load()}
            className="btn-ghost px-3 py-1.5 text-xs"
          >
            {t.orders.refresh}
          </button>
        </div>
      </div>

      <div className="mt-5 flex flex-wrap items-center gap-2">
        <FilterChip
          active={filter === "active"}
          onClick={() => setFilter("active")}
        >
          {t.orders.filterActive}
        </FilterChip>
        <FilterChip
          active={filter === "preorders"}
          onClick={() => setFilter("preorders")}
        >
          {t.orders.filterPreorder}
        </FilterChip>
        <FilterChip active={filter === "all"} onClick={() => setFilter("all")}>
          {t.orders.filterAll}
        </FilterChip>
        {ORDER_STATUSES.map((s) => (
          <FilterChip
            key={s}
            active={filter === s}
            onClick={() => setFilter(s)}
          >
            {t.status[s]}
          </FilterChip>
        ))}
        <input
          className="input ml-auto max-w-xs"
          placeholder={t.orders.searchPh}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>

      {/* Which kitchen is behind, right now.
          ⚠️ Information, not an instruction: nothing here moves an order by
          itself. Routing away from a busy branch puts the food further from the
          guest, and whether that trade is worth making at seven on a Friday
          depends on how many couriers are out — which no ticket count knows. */}
      {multiBranch && branchLoad.length > 0 && (
        <div className="mt-5 flex flex-wrap gap-2">
          {branchLoad.map((b) => {
            const busy = b.oldestMin >= 30 || b.cooking >= 10;
            const warm = !busy && (b.oldestMin >= 15 || b.cooking >= 5);
            return (
              <div
                key={b.branchId}
                className={`rounded-2xl border px-4 py-2 text-xs ${
                  busy
                    ? "border-rose-400 bg-rose-50 dark:bg-rose-500/10"
                    : warm
                      ? "border-amber-400 bg-amber-50 dark:bg-amber-500/10"
                      : "border-line bg-surface"
                }`}
              >
                <p className="font-semibold">{b.name}</p>
                <p className="mt-0.5 text-ink-muted">
                  {t.orders.loadCooking(b.cooking)}
                  {b.pending > 0 && ` · ${t.orders.loadPending(b.pending)}`}
                  {/* The number that actually means "behind". */}
                  {b.oldestMin > 0 && ` · ${t.orders.loadOldest(b.oldestMin)}`}
                </p>
              </div>
            );
          })}
        </div>
      )}

      {/* The list is bounded: a busy day would otherwise make this page metres
          long, with the filters scrolled far out of reach. */}
      <div className="mt-5 rounded-3xl border border-line bg-surface shadow-card">
        <ListScroll className="space-y-3 p-3" max="max-h-[72vh]">
          {loading ? (
            <p className="py-10 text-center text-ink-muted/70">
              {t.common.loading}
            </p>
          ) : orders.length === 0 ? (
            <p className="py-10 text-center text-ink-muted/70">
              {t.orders.empty}
            </p>
          ) : (
            paged.pageItems.map((o) => {
              const open = openId === o.id;
              const next = nextStatus(o);
              const label = nextActionLabel(o, t.nextAction);
              // Money that has not arrived yet. The one-tap button is disabled
              // for these: the whole reason an online order waits is so nobody
              // cooks it before it is paid for. The status dropdown beside it
              // still works — the same deliberate escape hatch the courier's
              // "delivered" check has, for when a payment lands but the
              // callback did not.
              const awaitingPayment = o.paymentStatus === "pending";
              return (
                // The row wears its status: green settled, amber in the
                // kitchen, red a problem, brand new. Light enough to read
                // through — see STATUS_ROW.
                <div
                  key={o.id}
                  className={`rounded-3xl border shadow-card transition-colors ${STATUS_ROW[o.status]}`}
                >
                  <div className="flex flex-wrap items-center gap-3 p-4">
                    {/* A brand-new order gets its own accept button, first in the
                      row — the one action the kitchen needs at a glance. */}
                    {o.status === "pending" && (
                      <button
                        type="button"
                        disabled={saving === o.id}
                        onClick={() => changeStatus(o, "confirmed")}
                        className="btn-primary shrink-0 px-4 py-2.5 text-sm"
                      >
                        {saving === o.id ? "..." : t.orders.accept}
                      </button>
                    )}

                    <button
                      type="button"
                      onClick={() => setOpenId(open ? null : o.id)}
                      className="min-w-[220px] flex-1 text-left"
                    >
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-semibold">#{o.number}</span>
                        <ChannelBadge channel={o.channel} />
                        {o.status === "pending" && !awaitingPayment && (
                          <span className="badge bg-brand text-white">
                            {t.orders.isNew}
                          </span>
                        )}
                        {awaitingPayment && (
                          <span className="badge bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300">
                            {t.orders.awaitingPayment}
                          </span>
                        )}
                        {o.paymentStatus === "paid" && (
                          <span className="badge bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300">
                            {t.orders.paid}
                          </span>
                        )}
                        {o.paymentStatus === "refunded" && (
                          <span className="badge bg-ink/5 text-ink-muted">
                            {t.orders.refunded}
                          </span>
                        )}
                        <span className={`badge ${STATUS_BADGE[o.status]}`}>
                          {t.status[o.status]}
                        </span>
                        {/* The kitchen's own mark. Shown beside the status rather
                          than inside it because it is not one: the cook is done,
                          and what happens next (a courier, a counter, a table)
                          is somebody else's step. */}
                        {o.readyAt &&
                          (o.status === "confirmed" ||
                            o.status === "preparing") && (
                            <span className="badge bg-emerald-500/15 text-emerald-700 dark:text-emerald-300">
                              {t.orders.kitchenReady}
                            </span>
                          )}
                        {/* Placed for a later time. Shown on every tab, not
                          just the pre-order one: a scheduled order in the
                          middle of the active list looks like an ordinary
                          one nobody has started, and somebody eventually
                          "fixes" it by cooking it early. */}
                        {o.scheduledAt && (
                          <span className="badge bg-brand-tint text-brand-dark">
                            🕒{" "}
                            {t.orders.preorderFor(
                              `${formatDate(o.scheduledAt)} ${formatTime(o.scheduledAt)}`,
                            )}
                          </span>
                        )}
                        {/* Its lead time has arrived — this is the kitchen's now.
                          The one thing the pre-order badge above cannot say,
                          and the reason the bell rang. */}
                        {o.scheduledAt &&
                          o.queuedAt &&
                          new Date(o.queuedAt) <= new Date() &&
                          o.status !== "delivered" &&
                          o.status !== "cancelled" && (
                            <span className="badge bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300">
                              {t.orders.preorderDue}
                            </span>
                          )}
                        {/* ⚠️ **The order never reached the till.** Usually one
                          dish with no mapping: pos.CheckMapped refuses the
                          whole order by design, but sending happens in the
                          background on confirm — so the row turned green, the
                          operator moved on, and the kitchen has no ticket and
                          no reason to suspect it. This row was the one place
                          somebody was already looking, and it said nothing. */}
                        {o.pos?.status === "failed" && (
                          <span className="badge bg-rose-500/15 text-rose-700 dark:text-rose-300">
                            {t.orders.posFailed}
                          </span>
                        )}
                        {/* Reached the till, and nobody there has accepted it. A
                          lesser problem than the above — the ticket is at
                          least on their screen — so it is amber, not red. */}
                        {o.pos?.till?.state === "waiting" && (
                          <span className="badge bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300">
                            {t.orders.posWaiting}
                          </span>
                        )}
                        <span className="text-xs text-ink-muted">
                          {timeAgo(o.createdAt, t.common.timeAgo)}
                        </span>
                      </div>
                      <p className="mt-1 text-sm text-ink-muted">
                        {o.customer.name} · {o.customer.phone} ·{" "}
                        {o.type === "uzum_tezkor"
                          ? t.dashboard.uzumTezkor
                          : o.type === "delivery"
                          ? t.dashboard.delivery
                          : o.type === "dinein"
                            ? t.receipt.tableLine(o.tableNumber ?? "—")
                            : t.dashboard.pickup}{" "}
                        · {t.orders.dishes(o.items.length)}
                        {o.externalDelivery?.providerName &&
                          ` · ${t.settings.calledBy(o.externalDelivery.providerName)}`}
                      </p>
                    </button>

                    {/* The courier, as something you can actually ring. It used
                      to be plain text inside the row button: a name and no way
                      to reach the person carrying the order. */}
                    {o.courierName && (
                      <CourierCall
                        name={o.courierName}
                        phone={
                          couriers.find((c) => c.id === realId(o.courierId))
                            ?.phone
                        }
                        t={t}
                      />
                    )}

                    <span className="font-bold tabular-nums">
                      {formatPrice(o.total)}
                    </span>

                    {/* One tap moves the order to the next stage. */}
                    {next && label && o.status !== "pending" && (
                      <button
                        type="button"
                        disabled={saving === o.id || awaitingPayment}
                        title={
                          awaitingPayment
                            ? t.orders.awaitingPaymentHint
                            : undefined
                        }
                        onClick={() => changeStatus(o, next)}
                        className="btn-primary px-4 py-2 text-xs disabled:opacity-40"
                      >
                        {saving === o.id ? "..." : label}
                      </button>
                    )}

                    {/* Who is carrying it — only delivery orders need a courier. */}
                    {o.type === "delivery" &&
                      o.status !== "delivered" &&
                      o.status !== "cancelled" && (
                        <select
                          value={realId(o.courierId)}
                          disabled={saving === o.id}
                          onChange={(e) => assignCourier(o, e.target.value)}
                          className={`rounded-xl border bg-surface px-2 py-1.5 text-xs outline-none focus:border-brand ${
                            hasId(o.courierId)
                              ? "border-line-strong"
                              : "border-brand/50 text-brand"
                          }`}
                          title={t.orders.pickCourier}
                        >
                          <option value="">{t.orders.pickCourier}</option>
                          {couriers
                            .filter((c) => c.isActive)
                            .map((c) => (
                              <option key={c.id} value={c.id}>
                                {c.name}
                                {c.status === "off" ? t.orders.courierOff : ""}
                                {c.status === "busy"
                                  ? t.orders.courierBusy
                                  : ""}
                              </option>
                            ))}
                        </select>
                      )}

                    {/* The dropdown stays for corrections (e.g. stepping back). */}
                    <select
                      value={o.status}
                      disabled={saving === o.id}
                      onChange={(e) =>
                        changeStatus(o, e.target.value as OrderStatus)
                      }
                      className="rounded-xl border border-line-strong bg-surface px-2 py-1.5 text-xs outline-none focus:border-brand"
                      title={t.orders.manualStatus}
                    >
                      {ORDER_STATUSES.map((s) => (
                        <option key={s} value={s}>
                          {t.status[s]}
                        </option>
                      ))}
                    </select>
                  </div>

                  {open && (
                    <div className="border-t border-line p-4">
                      <OrderReceipt
                        order={o}
                        courierPhone={
                          couriers.find((c) => c.id === realId(o.courierId))
                            ?.phone
                        }
                        editableAddress
                        onAddressSaved={() => load({ silent: true })}
                      />
                      <div className="mt-4 flex flex-wrap gap-3 text-xs">
                        <a
                          href={`tel:${o.customer.phone}`}
                          className="btn-ghost px-3 py-1.5"
                        >
                          {t.orders.call}
                        </a>
                        {o.type === "delivery" &&
                          o.status !== "delivered" &&
                          o.status !== "cancelled" && (
                            <button
                              type="button"
                              onClick={() => setCalling(o)}
                              className="btn-ghost px-3 py-1.5"
                            >
                              {t.settings.callDelivery}
                            </button>
                          )}
                        <Link
                          href={`/order/${o.number}`}
                          target="_blank"
                          className="btn-ghost px-3 py-1.5"
                        >
                          {t.orders.customerView}
                        </Link>
                        {/* Hand it to another kitchen. ⚠️ The money does not
                          change — it was agreed with the guest — and the
                          number keeps its old prefix, because that is the
                          tracking link they were given. */}
                        {multiBranch &&
                          o.status !== "delivered" &&
                          o.status !== "cancelled" && (
                            <label className="flex items-center gap-2">
                              <span className="text-ink-muted">
                                {t.orders.moveBranch}
                              </span>
                              <select
                                value={realId(o.branchId)}
                                disabled={moving === o.id}
                                onChange={(e) => moveBranch(o, e.target.value)}
                                className="rounded-xl border border-line-strong bg-surface px-2 py-1.5 text-xs outline-none focus:border-brand"
                              >
                                {scope.brandBranches.map((b) => (
                                  <option key={b.id} value={b.id}>
                                    {b.name}
                                  </option>
                                ))}
                              </select>
                            </label>
                          )}
                        {o.status !== "cancelled" &&
                          o.status !== "delivered" && (
                            <button
                              type="button"
                              onClick={() => setCancelling(o)}
                              className="btn-ghost px-3 py-1.5 text-brand"
                            >
                              {t.orders.cancelOrder}
                            </button>
                          )}
                      </div>
                    </div>
                  )}
                </div>
              );
            })
          )}
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

      <p className="mt-6 text-xs text-ink-muted">
        {filter === "preorders"
          ? t.orders.preorderNote
          : t.orders.autoRefresh(REFRESH_MS / 1000)}
      </p>

      {cancelling && (
        <CancelOrderModal
          order={cancelling}
          onClose={() => setCancelling(null)}
          onConfirm={(reason) => changeStatus(cancelling, "cancelled", reason)}
        />
      )}

      {calling && (
        <CallDeliveryModal
          order={calling}
          onClose={() => setCalling(null)}
          onDone={() => load({ silent: true })}
        />
      )}
    </div>
  );
}

function FilterChip({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-full px-3.5 py-1.5 text-sm font-semibold transition-colors ${
        active
          ? "bg-brand text-white"
          : "border border-line bg-surface text-ink-soft hover:border-brand hover:text-brand"
      }`}
    >
      {children}
    </button>
  );
}

// CourierCall shows who is carrying the order and dials them in one tap.
// A plain `tel:` anchor styled as a button: on the phone the dispatcher is
// holding, that is the whole interaction.
function CourierCall({
  name,
  phone,
  t,
}: {
  name: string;
  phone?: string;
  t: ReturnType<typeof useAdminT>;
}) {
  if (!phone) {
    return (
      <span className="rounded-full border border-line-strong px-3 py-1.5 text-xs font-semibold text-ink-muted">
        🛵 {name}
      </span>
    );
  }
  return (
    <a
      href={`tel:+${phone.replace(/\D/g, "")}`}
      onClick={(e) => e.stopPropagation()}
      title={t.orders.callCourier(name)}
      className="flex shrink-0 items-center gap-1.5 rounded-full bg-brand-tint px-3 py-1.5 text-xs font-bold text-brand-dark transition-colors hover:bg-brand hover:text-white"
    >
      <svg
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="h-3.5 w-3.5"
        aria-hidden
      >
        <path d="M22 16.9v3a2 2 0 0 1-2.2 2 19.8 19.8 0 0 1-8.6-3.1 19.5 19.5 0 0 1-6-6A19.8 19.8 0 0 1 2.1 4.2 2 2 0 0 1 4.1 2h3a2 2 0 0 1 2 1.7c.1 1 .4 1.9.7 2.8a2 2 0 0 1-.5 2.1L8.1 9.9a16 16 0 0 0 6 6l1.3-1.2a2 2 0 0 1 2.1-.5c.9.3 1.8.6 2.8.7a2 2 0 0 1 1.7 2Z" />
      </svg>
      <span className="max-w-[8rem] truncate">{name}</span>
    </a>
  );
}
