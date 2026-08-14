"use client";

// Who is on the line, in reading order.
//
// The order of the blocks below is the order an operator needs them, and it is
// not the order the data comes in:
//
//   1. an unanswered complaint — never greet someone cheerfully while the
//      restaurant still owes them an apology;
//   2. anything of theirs in the kitchen right now — this is what most calls
//      are actually about;
//   3. who they are and what they usually order — enough to say "the usual?";
//   4. history, for the questions that come after.
//
// A first-time caller is the common case, not an error state: the card still
// renders, it just says so and offers to take their order.

import Link from "next/link";
import { formatDateTime, formatPrice, formatUzPhone } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { STATUS_BADGE, STATUS_ROW } from "@/lib/orderStatus";
import { SegmentBadge } from "@/components/admin/Segments";
import type { CallerLookup, CallerOrder } from "@/lib/types";

export default function CallerCard({
  data,
  currency,
  onRepeat,
}: {
  data: CallerLookup;
  currency: string;
  /** "Repeat" hands the dishes of a past order to the order composer. */
  onRepeat: (order: CallerOrder) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const user = data.user;
  const name = user
    ? [user.firstName, user.lastName].filter(Boolean).join(" ").trim()
    : "";

  return (
    <div className="space-y-4">
      {/* ---- Identity ---- */}
      <div className="card p-4">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h3 className="font-display text-lg font-bold">
              {name || t.calls.unknownCaller}
            </h3>
            <p className="text-sm text-ink-muted">{formatUzPhone(data.phone)}</p>
          </div>
          {user && (
            <Link
              href={`/admin/users/${user.id}`}
              className="shrink-0 text-xs font-semibold text-brand hover:underline"
            >
              {t.calls.openCustomer}
            </Link>
          )}
        </div>

        {!user && (
          <p className="mt-2 rounded-lg bg-ink/5 p-2 text-xs text-ink-muted">
            {t.calls.unknownHint}
          </p>
        )}

        {data.segments.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-1">
            {data.segments.map((s) => (
              <SegmentBadge key={s} segment={s} />
            ))}
          </div>
        )}

        {/* The note the restaurant keeps: an allergy, a habit, a warning. It
            is the one thing here that has to be read *before* speaking. */}
        {user?.note && (
          <div className="mt-3 rounded-lg border border-amber-500/40 bg-amber-500/10 p-2">
            <div className="text-xs font-semibold text-amber-700 dark:text-amber-300">
              {t.calls.note}
            </div>
            <p className="text-sm">{user.note}</p>
          </div>
        )}

        {user && (
          <div className="mt-3 grid grid-cols-2 gap-2 text-sm sm:grid-cols-4">
            <Stat label={t.calls.ordersCount} value={String(data.ordersCount)} />
            <Stat
              label={t.calls.ordersTotal}
              value={formatPrice(data.ordersTotal, currency, lang)}
            />
            <Stat
              label={t.calls.avgOrder}
              value={formatPrice(data.avgOrder, currency, lang)}
            />
            <Stat
              label={t.calls.points}
              value={formatPrice(user.points ?? 0, currency, lang)}
            />
          </div>
        )}
      </div>

      {/* ---- Owed an apology ---- */}
      {data.openComplaints.length > 0 && (
        <div className="card border-rose-500/40 bg-rose-500/5 p-4">
          <div className="text-sm font-bold text-rose-700 dark:text-rose-300">
            {t.calls.complaints}
          </div>
          <p className="text-xs text-ink-muted">{t.calls.complaintsHint}</p>
          <ul className="mt-2 space-y-2">
            {data.openComplaints.map((f) => (
              <li key={f.id} className="text-sm">
                <Link
                  href={`/admin/feedback`}
                  className="font-semibold hover:underline"
                >
                  #{f.orderNumber}
                </Link>{" "}
                <span className="text-ink-muted">
                  {"★".repeat(f.rating)} · {formatDateTime(f.createdAt)}
                </span>
                {f.comment && <p className="text-ink-muted">{f.comment}</p>}
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* ---- In the kitchen right now ---- */}
      {data.activeOrders.length > 0 && (
        <div className="card p-4">
          <div className="text-sm font-bold">{t.calls.activeOrders}</div>
          <ul className="mt-2 space-y-2">
            {data.activeOrders.map((o) => (
              <OrderRow key={o.id} order={o} currency={currency} lang={lang} />
            ))}
          </ul>
        </div>
      )}

      {/* ---- Bookings ahead ---- */}
      {data.reservations.length > 0 && (
        <div className="card p-4">
          <div className="text-sm font-bold">{t.calls.reservations}</div>
          <ul className="mt-2 space-y-1 text-sm">
            {data.reservations.map((r) => (
              <li key={r.id} className="flex items-center justify-between gap-2">
                <span>
                  #{r.number} · {r.tableNumber} · {r.guests}
                </span>
                <span className="text-ink-muted">{formatDateTime(r.at)}</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* ---- "The usual?" ---- */}
      {data.favourites.length > 0 && (
        <div className="card p-4">
          <div className="text-sm font-bold">{t.calls.favourites}</div>
          <div className="mt-2 flex flex-wrap gap-1">
            {data.favourites.map((f) => (
              <span key={f.menuItemId} className="chip">
                {f.name}
                <span className="ml-1 text-xs text-ink-muted">
                  {t.calls.favouriteTimes(f.times)}
                </span>
              </span>
            ))}
          </div>
        </div>
      )}

      {/* ---- What to offer ----
           ⚠️ Right under "the usual", because that is what it is built from,
           and because the operator reads this card top to bottom while the
           customer is already talking. The phone is where upselling actually
           works and the only channel with no screen to put a card on — a guest
           on the site can be shown a suggestion, a guest on the telephone can
           only be told, by somebody with three seconds to think of something.
           The price is here so they can say it out loud without opening the
           menu. */}
      {(data.suggest?.length ?? 0) > 0 && (
        <div className="card p-4">
          <div className="text-sm font-bold">{t.calls.suggest}</div>
          <div className="mt-2 flex flex-wrap gap-1">
            {data.suggest.map((sgt) => (
              <span key={sgt.menuItemId} className="chip">
                {sgt.name}
                <span className="ml-1 text-xs text-ink-muted">
                  {formatPrice(sgt.price)}
                </span>
              </span>
            ))}
          </div>
        </div>
      )}

      {/* ---- Where they live ---- */}
      {(user?.addresses?.length ?? 0) > 0 && (
        <div className="card p-4">
          <div className="text-sm font-bold">{t.calls.addresses}</div>
          <ul className="mt-2 space-y-1 text-sm text-ink-muted">
            {user!.addresses!.map((a, i) => (
              <li key={i}>
                {a.label && <span className="font-semibold">{a.label}: </span>}
                {a.text}
                {a.comment && <span> ({a.comment})</span>}
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* ---- History ---- */}
      {data.recentOrders.length > 0 && (
        <div className="card p-4">
          <div className="text-sm font-bold">{t.calls.recentOrders}</div>
          <ul className="mt-2 space-y-2">
            {data.recentOrders.map((o) => (
              <OrderRow
                key={o.id}
                order={o}
                currency={currency}
                lang={lang}
                onRepeat={() => onRepeat(o)}
                repeatLabel={t.calls.repeat}
              />
            ))}
          </ul>
        </div>
      )}

      {/* ---- What was said last time ---- */}
      <div className="card p-4">
        <div className="text-sm font-bold">{t.calls.recentCalls}</div>
        {data.recentCalls.length === 0 ? (
          <p className="mt-1 text-sm text-ink-muted">{t.calls.noRecentCalls}</p>
        ) : (
          <ul className="mt-2 space-y-2 text-sm">
            {data.recentCalls.map((c) => (
              <li key={c.id}>
                <div className="flex items-center justify-between gap-2">
                  <span className="font-medium">
                    {t.calls.outcomes[c.outcome]}
                  </span>
                  <span className="text-xs text-ink-muted">
                    {formatDateTime(c.createdAt)}
                  </span>
                </div>
                {c.note && <p className="text-ink-muted">{c.note}</p>}
                <p className="text-xs text-ink-muted">
                  {t.calls.operator}: {c.operatorName || t.common.none}
                </p>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs text-ink-muted">{label}</div>
      <div className="font-semibold">{value}</div>
    </div>
  );
}

function OrderRow({
  order,
  currency,
  lang,
  onRepeat,
  repeatLabel,
}: {
  order: CallerOrder;
  currency: string;
  lang: "uz" | "ru" | "en";
  onRepeat?: () => void;
  repeatLabel?: string;
}) {
  const t = useAdminT();
  return (
    // Same status tint as the orders board, so an operator reading a caller's
    // history sees the same colours they see everywhere else.
    <li className={`rounded-lg border p-2 text-sm ${STATUS_ROW[order.status]}`}>
      <div className="flex flex-wrap items-center gap-2">
        <Link
          href={`/admin/orders?q=${order.number}`}
          className="font-semibold hover:underline"
        >
          #{order.number}
        </Link>
        <span className={`badge ${STATUS_BADGE[order.status]}`}>
          {t.status[order.status]}
        </span>
        <span className="ml-auto font-semibold">
          {formatPrice(order.total, currency, lang)}
        </span>
      </div>
      <p className="text-xs text-ink-muted">
        {formatDateTime(order.createdAt)}
        {order.address?.text ? ` · ${order.address.text}` : ""}
      </p>
      <p className="text-xs text-ink-muted">
        {order.items.map((i) => `${i.name} ×${i.qty}`).join(", ")}
      </p>
      {onRepeat && (
        <button
          type="button"
          onClick={onRepeat}
          className="mt-1 text-xs font-semibold text-brand hover:underline"
        >
          {repeatLabel}
        </button>
      )}
    </li>
  );
}
