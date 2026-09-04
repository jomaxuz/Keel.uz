"use client";

// One courier, everything about them: payout rule, what they earned in each
// period, how much cash they are holding, and every delivery with its full
// receipt — so payday and "where is my order" both have an answer here.

import { use, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatPrice, formatUzPhone } from "@/lib/format";
import { formatDateTime, timeAgo } from "@/lib/orderFlow";
import { ListScroll, Pager, usePaged } from "@/components/admin/PagedList";
import { useAdminT } from "@/lib/i18n/admin";
import DeviceList from "@/components/admin/DeviceList";
import Modal from "@/components/admin/Modal";
import OrderReceipt from "@/components/admin/OrderReceipt";
import type { AdminCourierDetail, CourierPeriod } from "@/lib/types";
import { useAsk } from "@/components/ui/Ask";

const EMPTY_ORDERS: never[] = [];

export default function AdminCourierPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const [data, setData] = useState<AdminCourierDetail | null>(null);
  const { tell } = useAsk();
  // Recording a cash handover.
  const [settleOpen, setSettleOpen] = useState(false);
  const [settleAmount, setSettleAmount] = useState("");
  const [settleNote, setSettleNote] = useState("");
  const [settling, setSettling] = useState(false);
  // Paying the courier their wage.
  //
  // ⚠️ **Deliberately not the same dialog as the settlement**, which is its
  // opposite: one records the courier handing our cash back, the other records
  // us handing them their pay. A single form with a direction toggle is exactly
  // where somebody credits a courier for money they returned.
  const [payOpen, setPayOpen] = useState(false);
  const [payAmount, setPayAmount] = useState("");
  const [payNote, setPayNote] = useState("");
  const [payFromSafe, setPayFromSafe] = useState(false);
  const [paying, setPaying] = useState(false);
  const [loading, setLoading] = useState(true);
  const [openOrder, setOpenOrder] = useState<string | null>(null);
  // The receipts arrive with the profile, so paging waits for it.
  const paged = usePaged(data?.orders ?? EMPTY_ORDERS, 10);
  const t = useAdminT();

  const load = useCallback(() => {
    api
      .adminCourier(id)
      .then(setData)
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, [id]);

  useEffect(load, [load]);

  if (loading) {
    return (
      <p className="py-10 text-center text-ink-muted/70">{t.common.loading}</p>
    );
  }
  if (!data) {
    return (
      <div className="py-10 text-center">
        <p className="text-ink-muted">{t.couriers.notFound}</p>
        <Link href="/admin/couriers" className="btn-ghost mt-4 px-4 py-2">
          {t.couriers.backToList}
        </Link>
      </div>
    );
  }

  const { courier, stats, orders } = data;
  const settlements = data.settlements ?? [];
  const payments = data.payments ?? [];
  const paidTotal = data.paid ?? 0;
  // ⚠️ Can read negative — a courier paid in advance — and it is shown as it
  // falls rather than clamped: "we are ahead by 300 000" is the reason the
  // next payment is smaller, and hiding it makes that look like a mistake.
  const due = stats.all.earnings - paidTotal;
  const payoutRule =
    courier.payoutMode === "perOrder"
      ? `${t.couriers.payoutPerOrder} — ${formatPrice(courier.payoutPerOrder ?? 0)}`
      : courier.payoutMode === "percent"
        ? `${t.couriers.payoutPercent} — ${courier.payoutPercent ?? 0}%`
        : t.couriers.payoutDeliveryFee;

  return (
    <div>
      <Link
        href="/admin/couriers"
        className="text-sm text-ink-muted hover:text-brand"
      >
        {t.couriers.backToList}
      </Link>

      {/* ---- header ---- */}
      <div className="mt-4 flex flex-wrap items-center gap-4">
        <span className="flex h-14 w-14 items-center justify-center rounded-full bg-brand-tint font-display text-xl font-bold text-brand">
          {courier.name.charAt(0).toUpperCase()}
        </span>
        <div className="flex-1">
          <h1 className="font-display text-2xl font-bold">{courier.name}</h1>
          <p className="text-sm text-ink-muted">
            @{courier.username}
            {courier.phone && (
              <>
                {" · "}
                <a
                  href={`tel:+${courier.phone.replace(/\D/g, "")}`}
                  className="hover:text-brand"
                >
                  {formatUzPhone(courier.phone)}
                </a>
              </>
            )}
          </p>
          <p className="mt-0.5 text-xs text-ink-muted">
            {t.couriers.payoutNote(payoutRule)}
          </p>
        </div>
        <span className="badge bg-ink/10 text-ink-muted">
          {t.couriers.status[courier.status]}
        </span>
      </div>

      {/* ---- earnings ---- */}
      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <PeriodCard label={t.couriers.periodToday} period={stats.today} t={t} />
        <PeriodCard label={t.couriers.periodWeek} period={stats.week} t={t} />
        <PeriodCard label={t.couriers.periodMonth} period={stats.month} t={t} />
        <PeriodCard label={t.couriers.periodAll} period={stats.all} t={t} />
      </div>

      {/* The money question the restaurant actually has: how much is still in
          this courier's pocket right now. Collected minus handed in — the old
          figure only ever grew and stopped meaning anything by the second day. */}
      <div className="mt-4 rounded-3xl border border-line bg-surface p-5 shadow-card">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div>
            <p className="text-xs uppercase tracking-wider text-ink-muted">
              {t.couriers.cashOnHand}
            </p>
            <p
              className={`mt-1 font-display text-3xl font-bold ${
                stats.cashInHand > 0
                  ? "text-amber-600 dark:text-amber-400"
                  : "text-emerald-600 dark:text-emerald-400"
              }`}
            >
              {formatPrice(stats.cashInHand)}
            </p>
            <p className="mt-1 text-xs text-ink-muted">
              {t.couriers.cashExplain(
                formatPrice(stats.all.cash),
                formatPrice(stats.cashSettled),
              )}
            </p>
          </div>
          {stats.cashInHand > 0 && (
            <button
              type="button"
              onClick={() => {
                setSettleAmount(String(stats.cashInHand));
                setSettleOpen(true);
              }}
              className="btn-primary px-5 py-2.5 text-sm"
            >
              {t.couriers.settle}
            </button>
          )}
        </div>

        {settlements.length > 0 && (
          <ul className="mt-4 divide-y divide-line border-t border-line text-sm">
            {settlements.slice(0, 5).map((sx) => (
              <li key={sx.id} className="flex items-center gap-3 py-2">
                <span className="flex-1 text-ink-muted">
                  {formatDateTime(sx.at)} · {sx.takenBy}
                  {sx.note ? ` · ${sx.note}` : ""}
                </span>
                <span className="shrink-0 tabular-nums font-semibold">
                  {formatPrice(sx.amount)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* ---- What the courier has earned, and what they have been given ----

          ⚠️ **Two figures side by side, neither derived from the other.**
          Earnings come from the deliveries and the payout rule — a fact about
          work. Pay is a document — a fact about money. Deriving either direction
          gives a number that moves when a rule is edited, months after the notes
          were counted out. */}
      <div className="mt-4 rounded-3xl border border-line bg-surface p-5 shadow-card">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div className="flex flex-wrap gap-6">
            <div>
              <p className="text-xs uppercase tracking-wider text-ink-muted">
                {t.couriers.payEarned}
              </p>
              <p className="mt-1 font-display text-2xl font-bold tabular-nums">
                {formatPrice(stats.all.earnings)}
              </p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-wider text-ink-muted">
                {t.couriers.payPaid}
              </p>
              <p className="mt-1 font-display text-2xl font-bold tabular-nums">
                {formatPrice(paidTotal)}
              </p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-wider text-ink-muted">
                {t.couriers.payDue}
              </p>
              <p
                className={`mt-1 font-display text-2xl font-bold tabular-nums ${
                  due > 0 ? "text-amber-600 dark:text-amber-400" : ""
                }`}
              >
                {formatPrice(due)}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={() => {
              setPayAmount(due > 0 ? String(due) : "");
              setPayOpen(true);
            }}
            className="btn-primary px-5 py-2.5 text-sm"
          >
            {t.couriers.pay}
          </button>
        </div>

        {payments.length > 0 && (
          <ul className="mt-4 divide-y divide-line border-t border-line text-sm">
            {payments.slice(0, 5).map((px) => (
              <li key={px.id} className="flex items-center gap-3 py-2">
                <span className="flex-1 text-ink-muted">
                  {formatDateTime(px.at)}
                  {px.paidBy ? ` · ${px.paidBy}` : ""}
                  {px.note ? ` · ${px.note}` : ""}
                </span>
                <span className="shrink-0 tabular-nums font-semibold">
                  {formatPrice(px.amount)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>

      {stats.active > 0 && (
        <p className="mt-3 rounded-2xl bg-amber-50 px-4 py-2.5 text-sm text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
          {t.couriers.activeNow}: {stats.active}
        </p>
      )}

      {/* ---- delivery history ---- */}
      <section className="mt-8">
        <h2 className="font-display text-lg font-bold">
          {t.couriers.history} ({orders.length})
        </h2>
        <div className="mt-3 rounded-3xl border border-line bg-surface shadow-card">
          <ListScroll className="space-y-3 p-3" max="max-h-[70vh]">
            {orders.length === 0 && (
              <p className="p-6 text-center text-sm text-ink-muted/70">
                {t.couriers.historyEmpty}
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
                      <span className="font-semibold">#{o.number}</span>
                      <p className="mt-1 text-xs text-ink-muted">
                        {t.couriers.deliveredAt}:{" "}
                        {formatDateTime(o.deliveredAt)} ·{" "}
                        {timeAgo(o.deliveredAt, t.common.timeAgo)}
                        {o.address?.text && ` · ${o.address.text}`}
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="text-xs text-ink-muted">
                        {t.receipt.total}: {formatPrice(o.total)}
                      </p>
                      <p className="font-bold tabular-nums text-brand">
                        +{formatPrice(o.earned)}
                      </p>
                    </div>
                    <span className="text-xs text-ink-muted">
                      {open ? "▲" : "▼"}
                    </span>
                  </button>

                  {open && (
                    <div className="border-t border-line p-4">
                      <OrderReceipt order={o} />
                      <div className="mt-4 flex flex-wrap gap-3 text-xs">
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
      {/* ⚠️ **On the courier's own page, where the person is.** A list of every
          device in the restaurant would be a screen somebody has to search;
          this is read while looking at the rider it is about — usually because
          they are standing there saying the app will not let them in. */}
      <DeviceList kind="courier" subjectId={id} className="mt-8" />

      {payOpen && (
        <Modal onClose={() => setPayOpen(false)}>
          <h2 className="text-lg font-bold">{t.couriers.payTitle}</h2>
          <p className="mt-1 text-sm text-ink-muted">{t.couriers.payHint}</p>
          <label className="mt-4 block text-sm">
            <span className="font-medium">{t.couriers.payAmount}</span>
            <input
              type="number"
              className="input mt-1 w-full"
              value={payAmount}
              onChange={(e) => setPayAmount(e.target.value)}
              autoFocus
            />
          </label>
          <label className="mt-3 block text-sm">
            <span className="font-medium">{t.couriers.payNote}</span>
            <input
              className="input mt-1 w-full"
              maxLength={200}
              value={payNote}
              onChange={(e) => setPayNote(e.target.value)}
            />
          </label>
          {/* ⚠️ Asked, not assumed: a courier is as likely to be paid out of
              the drawer they just handed cash into as out of the office box. */}
          <label className="mt-3 flex items-center gap-2 text-sm text-ink-muted">
            <input
              type="checkbox"
              checked={payFromSafe}
              onChange={(e) => setPayFromSafe(e.target.checked)}
            />
            {t.couriers.payFromSafe}
          </label>
          <div className="mt-6 flex justify-end gap-3">
            <button
              type="button"
              onClick={() => setPayOpen(false)}
              className="px-4 py-2 text-sm text-ink-muted hover:text-ink"
            >
              {t.common.cancel}
            </button>
            <button
              type="button"
              disabled={paying || !Number(payAmount)}
              onClick={async () => {
                setPaying(true);
                try {
                  await api.adminPayCourier(courier.id, {
                    amount: Number(payAmount),
                    note: payNote,
                    fromSafe: payFromSafe,
                  });
                  setPayOpen(false);
                  setPayNote("");
                  setPayFromSafe(false);
                  load();
                } catch {
                  void tell({ title: t.common.saveFailed });
                } finally {
                  setPaying(false);
                }
              }}
              className="btn-primary px-4 py-2 disabled:opacity-60"
            >
              {paying ? t.common.saving : t.couriers.payConfirm}
            </button>
          </div>
        </Modal>
      )}

      {settleOpen && (
        <Modal onClose={() => setSettleOpen(false)}>
          <h2 className="text-lg font-bold">{t.couriers.settleTitle}</h2>
          <p className="mt-1 text-sm text-ink-muted">{t.couriers.settleHint}</p>
          <label className="mt-4 block text-sm">
            <span className="font-medium">{t.couriers.settleAmount}</span>
            <input
              type="number"
              className="input mt-1 w-full"
              value={settleAmount}
              onChange={(e) => setSettleAmount(e.target.value)}
              autoFocus
            />
          </label>
          <label className="mt-3 block text-sm">
            <span className="font-medium">{t.couriers.settleNote}</span>
            <input
              className="input mt-1 w-full"
              maxLength={200}
              value={settleNote}
              onChange={(e) => setSettleNote(e.target.value)}
            />
          </label>
          <div className="mt-6 flex justify-end gap-3">
            <button
              type="button"
              onClick={() => setSettleOpen(false)}
              className="px-4 py-2 text-sm text-ink-muted hover:text-ink"
            >
              {t.common.cancel}
            </button>
            <button
              type="button"
              disabled={settling || !Number(settleAmount)}
              onClick={async () => {
                setSettling(true);
                try {
                  await api.settleCourierCash(
                    courier.id,
                    Number(settleAmount),
                    settleNote,
                  );
                  setSettleOpen(false);
                  setSettleNote("");
                  load();
                } catch {
                  void tell({ title: t.common.saveFailed });
                } finally {
                  setSettling(false);
                }
              }}
              className="btn-primary px-4 py-2 disabled:opacity-60"
            >
              {settling ? t.common.saving : t.couriers.settleConfirm}
            </button>
          </div>
        </Modal>
      )}
    </div>
  );
}

function PeriodCard({
  label,
  period,
  t,
}: {
  label: string;
  period: CourierPeriod;
  t: ReturnType<typeof useAdminT>;
}) {
  return (
    <div className="rounded-3xl border border-line bg-surface p-4 shadow-card">
      <p className="text-xs uppercase tracking-wider text-ink-muted">{label}</p>
      <p className="mt-1 font-display text-xl font-bold text-brand">
        {formatPrice(period.earnings)}
      </p>
      <p className="mt-1 text-xs text-ink-muted">
        {t.couriers.deliveries}: {period.orders}
      </p>
      {period.cash > 0 && (
        <p className="text-xs text-ink-muted">
          {t.couriers.cashCollected}: {formatPrice(period.cash)}
        </p>
      )}
    </div>
  );
}
