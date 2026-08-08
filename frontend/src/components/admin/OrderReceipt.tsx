"use client";

// Full order "receipt" shown in the admin panel: items with prices, totals,
// delivery address, payment method and the status timeline. This is what the
// operator opens when a customer calls back about an old order.

import { useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { hasId } from "@/lib/id";
import { STATUS_BADGE } from "@/lib/orderStatus";
import ChannelBadge from "@/components/admin/ChannelBadge";
import { useAdminT } from "@/lib/i18n/admin";
import { PAYMENT_LABEL } from "@/lib/payment";
import { formatDateTime } from "@/lib/orderFlow";
import OrderAddressMap from "@/components/admin/OrderAddressMap";
import type { Order } from "@/lib/types";

export default function OrderReceipt({
  order,
  showCustomer = true,
  editableAddress = false,
  onAddressSaved,
  courierPhone,
}: {
  order: Order;
  /** So the receipt can ring the courier. The order stores only the name, on
   *  purpose — the name has to survive the account being deleted. */
  courierPhone?: string;
  showCustomer?: boolean;
  // The orders section may correct the delivery pin; the customer card, which
  // reads as history, only shows it.
  editableAddress?: boolean;
  onAddressSaved?: () => void;
}) {
  const t = useAdminT();
  // The "accepted" row below already covers the initial pending state.
  const history = (order.statusHistory ?? []).filter(
    (h) => h.status !== "pending",
  );

  return (
    <div className="grid gap-6 text-sm sm:grid-cols-2">
      {/* ---- items ---- */}
      <div>
        <h3 className="font-semibold">{t.receipt.title}</h3>
        <p className="mt-0.5 text-xs text-ink-muted">
          ID: <span className="font-mono">{order.id}</span>
        </p>
        <ul className="mt-3 space-y-1">
          {order.items.map((it, i) => (
            <li key={i} className="flex justify-between gap-2">
              <span className="text-ink-muted">
                {it.name} × {it.qty}
                {it.options?.length ? (
                  <span className="block text-xs text-ink-muted/70">
                    {it.options
                      .map((o) => `${o.name}: ${o.choice}`)
                      .join(" · ")}
                  </span>
                ) : null}
                {/* A combo's courses. The kitchen cooks from the receipt, so
                    the set's name alone is not an instruction. */}
                {it.comboItems?.length ? (
                  <span className="block text-xs text-ink-muted/70">
                    {it.comboItems
                      .map((c) => (c.qty > 1 ? `${c.name} × ${c.qty}` : c.name))
                      .join(" · ")}
                  </span>
                ) : null}
                {/* The customer's note for this dish — the kitchen must not
                    miss it, so it is marked, not greyed out. */}
                {it.comment && (
                  <span className="mt-0.5 block rounded-lg bg-amber-50 px-2 py-1 text-xs font-medium text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
                    ✎ {it.comment}
                  </span>
                )}
              </span>
              <span className="tabular-nums">
                {formatPrice(it.price * it.qty)}
              </span>
            </li>
          ))}
        </ul>
        <div className="mt-3 space-y-0.5 border-t border-line pt-2 text-ink-muted">
          <div className="flex justify-between">
            <span>{t.receipt.items}</span>
            <span className="tabular-nums">{formatPrice(order.subtotal)}</span>
          </div>
          {/* Each discount by name and amount, exactly as it was applied. This
              is what answers "why is this order 27 000 less?" months later —
              the campaign itself may be long gone. */}
          {(order.discounts ?? []).map((d, i) => (
            <div
              key={i}
              className="flex justify-between text-emerald-700 dark:text-emerald-400"
            >
              <span className="min-w-0 truncate pr-2">
                {d.name}
                {d.code ? ` · ${d.code}` : ""}
              </span>
              <span className="shrink-0 tabular-nums">
                −{formatPrice(d.amount)}
              </span>
            </div>
          ))}
          {!!order.pointsSpent && (
            <div className="flex justify-between text-emerald-700 dark:text-emerald-400">
              <span>{t.receipt.pointsSpent}</span>
              <span className="shrink-0 tabular-nums">
                −{formatPrice(order.pointsSpent)}
              </span>
            </div>
          )}
          {order.type === "delivery" && (
            <div className="flex justify-between">
              <span>
                {t.receipt.delivery}
                {order.distanceKm ? ` (${order.distanceKm.toFixed(1)} km)` : ""}
              </span>
              <span className="tabular-nums">
                {formatPrice(order.deliveryFee)}
              </span>
            </div>
          )}
          <div className="flex justify-between pt-1 text-base font-bold text-ink">
            <span>{t.receipt.total}</span>
            <span className="tabular-nums">{formatPrice(order.total)}</span>
          </div>
          {!!order.pointsEarned && (
            <div className="flex justify-between pt-1 text-xs">
              <span>{t.receipt.pointsEarned}</span>
              <span className="tabular-nums">
                +{formatPrice(order.pointsEarned)}
              </span>
            </div>
          )}
        </div>
        <p className="mt-2 text-xs text-ink-muted">
          {t.receipt.paymentLine(
            PAYMENT_LABEL[order.paymentMethod],
            order.type === "delivery"
              ? t.receipt.delivery
              : order.type === "dinein"
                ? t.receipt.tableLine(order.tableNumber ?? "—")
                : t.dashboard.pickup,
          )}
        </p>
      </div>

      {/* ---- customer, address, timeline ---- */}
      <div className="space-y-4">
        {order.type === "dinein" && (
          <div className="rounded-2xl bg-brand-tint/50 px-3 py-2 text-sm font-semibold text-brand-dark">
            {t.receipt.tableLine(order.tableNumber ?? "—")}
          </div>
        )}
        {order.status === "cancelled" && (
          <div className="rounded-2xl bg-rose-50 px-3 py-2 text-sm text-brand dark:bg-rose-500/10">
            <span className="font-semibold">{t.orders.cancelReason}: </span>
            {order.cancelReason || t.common.none}
          </div>
        )}

        {showCustomer && (
          <div>
            <h3 className="font-semibold">{t.receipt.customer}</h3>
            <p className="mt-1">{order.customer.name}</p>
            <a
              href={`tel:${order.customer.phone}`}
              className="text-brand hover:underline"
            >
              {order.customer.phone}
            </a>
            {hasId(order.userId) && (
              <p className="mt-1">
                <Link
                  href={`/admin/users/${order.userId}`}
                  className="text-xs text-brand hover:underline"
                >
                  {t.receipt.profileLink}
                </Link>
              </p>
            )}
          </div>
        )}

        {order.type === "delivery" && (order.address?.text || editableAddress) && (
          <>
            <div>
              <h3 className="font-semibold">{t.receipt.address}</h3>
              <p className="mt-1 text-ink-muted">{order.address?.text}</p>
              {order.address?.comment && (
                <p className="text-xs text-ink-muted">
                  {t.receipt.comment}: {order.address.comment}
                </p>
              )}
              {!!order.address?.lat && (
                <a
                  href={`https://2gis.uz/geo/${order.address.lng},${order.address.lat}`}
                  target="_blank"
                  rel="noreferrer"
                  className="text-xs text-brand hover:underline"
                >
                  {t.receipt.openMapLink}
                </a>
              )}
            </div>

            {/* Where the customer actually pointed — and, in the orders
                section, the place to fix it when they pointed wrong. */}
            <OrderAddressMap
              order={order}
              editable={editableAddress}
              onSaved={onAddressSaved}
            />
          </>
        )}

        {/* Who took it outside — the answer to "so where is it?" weeks later. */}
        {order.externalDelivery?.providerName && (
          <div>
            <h3 className="font-semibold">{t.settings.providersTitle}</h3>
            <p className="mt-1 text-ink-muted">
              {t.settings.calledAtLine(
                order.externalDelivery.providerName,
                formatDateTime(order.externalDelivery.calledAt),
              )}
            </p>
            {order.externalDelivery.trackingId && (
              <p className="text-xs text-ink-muted">
                {t.settings.trackingId}:{" "}
                <span className="font-mono">
                  {order.externalDelivery.trackingId}
                </span>
              </p>
            )}
            {!!order.externalDelivery.cost && (
              <p className="text-xs text-ink-muted">
                {t.settings.callCost}:{" "}
                {formatPrice(order.externalDelivery.cost)}
              </p>
            )}
            {order.externalDelivery.status && (
              <p className="text-xs text-ink-muted">
                {t.settings.apiStatus}: {order.externalDelivery.status}
              </p>
            )}
            {order.externalDelivery.note && (
              <p className="text-xs text-ink-muted">
                {order.externalDelivery.note}
              </p>
            )}
          </div>
        )}

        {/* Did the kitchen actually get this? Only rendered once a POS has
            been connected — most installs have none and should see nothing. */}
        {order.pos && <POSBlock order={order} t={t} />}

        <div>
          <h3 className="font-semibold">{t.receipt.timeline}</h3>
          {/* Only phone orders carry a name here, which is what makes it worth
              showing: "who typed this in?" is the first question asked about a
              wrong address. */}
          {order.takenBy && (
            <p className="mt-1 text-xs text-ink-muted">
              {t.calls.takenBy(order.takenBy)}
            </p>
          )}
          {/* And which door it came in through. On the receipt rather than only
              in the list, because this is the screen somebody opens when a guest
              rings up about an order — and how to reach them back depends on it. */}
          <p className="mt-1">
            <ChannelBadge channel={order.channel} />
          </p>
          <ul className="mt-2 space-y-1 text-xs">
            <li className="flex justify-between gap-3">
              <span className="text-ink-muted">{t.receipt.accepted}</span>
              <span>{formatDateTime(order.createdAt)}</span>
            </li>
            {history.map((h, i) => (
              <li key={i} className="flex justify-between gap-3">
                <span className={`badge ${STATUS_BADGE[h.status]}`}>
                  {t.status[h.status]}
                </span>
                <span>{formatDateTime(h.at)}</span>
              </li>
            ))}
            {history.length === 0 && order.updatedAt !== order.createdAt && (
              <li className="flex justify-between gap-3">
                <span className="text-ink-muted">{t.receipt.lastChange}</span>
                <span>{formatDateTime(order.updatedAt)}</span>
              </li>
            )}
          </ul>
        </div>

        {/* Who is carrying it, as something you can ring. The receipt is the
            screen you open to deal with one order — the phone number belongs
            here, not only in the list. */}
        {order.courierName && (
          <div className="mt-3 flex flex-wrap items-center gap-2 border-t border-line pt-3">
            <span className="text-ink-muted">{t.receipt.courier}:</span>
            <span className="font-medium">{order.courierName}</span>
            {courierPhone && (
              <a
                href={`tel:+${courierPhone.replace(/\D/g, "")}`}
                className="ml-auto rounded-full bg-brand-tint px-3 py-1.5 text-xs font-bold text-brand-dark transition-colors hover:bg-brand hover:text-white"
              >
                📞 {t.receipt.call}
              </a>
            )}
          </div>
        )}
      </div>
    </div>
  );
}


/** The till's side of one order: whether it arrived, why not, and a way to try
 *  again. A failed push leaves the order untouched and visible — an order that
 *  exists here and not there is recoverable; one silently dropped is not. */
function POSBlock({
  order,
  t,
}: {
  order: Order;
  t: ReturnType<typeof useAdminT>;
}) {
  const [sending, setSending] = useState(false);
  const [state, setState] = useState(order.pos);
  const [error, setError] = useState("");

  async function send() {
    setSending(true);
    setError("");
    try {
      const res = await api.sendOrderToPOS(order.id);
      if (res.pos) setState(res.pos);
      if (!res.ok) setError(res.message ?? "");
    } catch (e) {
      setError(e instanceof Error ? e.message : "");
    } finally {
      setSending(false);
    }
  }

  const status = state?.status ?? "";
  const tone =
    status === "sent"
      ? "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
      : status === "failed"
        ? "bg-rose-500/10 text-rose-700 dark:text-rose-300"
        : "bg-ink/5 text-ink-muted";
  const label =
    status === "sent"
      ? t.pos.sent
      : status === "failed"
        ? t.pos.failed
        : status === "pending"
          ? t.pos.pendingState
          : t.pos.notSent;

  return (
    <div>
      <h3 className="font-semibold">{t.pos.orderTitle}</h3>
      <div className={`mt-2 rounded-xl p-3 text-sm ${tone}`}>
        <div className="font-semibold">{label}</div>
        {state?.note && <div className="text-xs">{state.note}</div>}
        {state?.error && <div className="mt-1 text-xs">{state.error}</div>}
        {error && <div className="mt-1 text-xs">{error}</div>}
        {(state?.attempts ?? 0) > 1 && (
          <div className="mt-1 text-xs opacity-70">
            {t.pos.attempts(state!.attempts)}
          </div>
        )}
        {status !== "sent" && (
          <button
            type="button"
            onClick={send}
            disabled={sending}
            className="btn btn-ghost mt-2 text-xs"
          >
            {sending ? t.pos.sending : status ? t.pos.resend : t.pos.send}
          </button>
        )}
      </div>
    </div>
  );
}
