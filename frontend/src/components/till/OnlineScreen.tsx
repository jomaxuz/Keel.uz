"use client";

// Today's online orders, and what the counter has to do about each one.
//
// ⚠️ **A delivery order was invisible from the till, and money was owed on it.**
// The counter's list is built from checks opened at a table, so an order taken
// on the website or in the bot had no row and no cashier who knew it existed.
// The cash came back in a courier's pocket at the end of a shift and was
// reconciled against nothing.
//
// ⚠️ **The value is the sentence, not the row.** A cashier does not need another
// list of orders — the kitchen screen has one. What they need is what *they*
// have to do, and that depends on a combination nobody should hold in their
// head: how it was paid, and whether it is delivered or collected.
//
// ⚠️ **Live and finished are two lists, not one sorted list.** They are read for
// opposite reasons: the first is work — somebody is waiting, or money is still
// out with a courier — and the second is a record somebody goes looking through
// when a guest rings back. Mixed together, an evening's forty delivered orders
// bury the three that need doing, which is how the whole screen stops being
// read.

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  LuBanknote,
  LuBike,
  LuCheck,
  LuCircleAlert,
  LuPhone,
  LuSearch,
  LuShoppingBag,
  LuX,
} from "react-icons/lu";

import { api } from "@/lib/api";
import { formatPrice, formatTime } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAsk } from "@/components/ui/Ask";
import type { OnlineOrder, Order } from "@/lib/types";

/** What each answer looks like. ⚠️ "Nothing to do" is deliberately the quiet
 *  one — it is the commonest row and a cashier has to be able to skip it
 *  without reading it. The loud one is the order that was started online and
 *  never finished, because the kitchen may already be cooking. */
const LOOK: Record<string, { icon: React.ReactNode; tone: string }> = {
  nothing: { icon: <LuCheck />, tone: "text-ink-muted" },
  from_courier: {
    icon: <LuBike />,
    tone: "text-amber-700 dark:text-amber-300",
  },
  at_counter: {
    icon: <LuBanknote />,
    tone: "text-emerald-700 dark:text-emerald-400",
  },
  unfinished: { icon: <LuCircleAlert />, tone: "text-danger" },
};

/** Which orders are still somebody's job.
 *
 *  ⚠️ **Delivered but unpaid is still live**, and that is the whole reason this
 *  is a function rather than `status !== "delivered"`. The courier is back with
 *  the notes in their pocket; the food has arrived and the money has not, and
 *  that row is exactly what this screen was built for. */
function isLive(o: OnlineOrder): boolean {
  return o.status !== "delivered" || o.settle !== "nothing";
}

type Tab = "live" | "done";

export default function OnlineScreen({
  onError,
}: {
  onError: (m: string) => void;
}) {
  const t = useAdminT();
  const { ask } = useAsk();
  const [rows, setRows] = useState<OnlineOrder[] | null>(null);
  const [owed, setOwed] = useState(0);
  const [busy, setBusy] = useState("");
  const [tab, setTab] = useState<Tab>("live");
  const [q, setQ] = useState("");
  const [open, setOpen] = useState<OnlineOrder | null>(null);

  const load = useCallback(() => {
    api
      .tillOnline()
      .then((r) => {
        setRows(r.orders ?? []);
        setOwed(r.owed ?? 0);
      })
      .catch((e) => onError(e instanceof Error ? e.message : ""));
  }, [onError]);

  useEffect(() => {
    load();
    // ⚠️ Polled, because an order arrives without anybody touching this screen
    // — which is the entire difference between this list and the tables. Half a
    // minute is slower than the kitchen screen on purpose: nothing here has to
    // be cooked, only collected.
    const id = setInterval(load, 30_000);
    return () => clearInterval(id);
  }, [load]);

  // ⚠️ **Searched here, not on the server.** Everything this screen can show is
  // already in the browser — today's orders, two hundred at most — so typing
  // filters at the speed of the keystroke instead of at the speed of the
  // restaurant's wifi. A cashier searching while a guest reads out their phone
  // number cannot wait for a round trip per character.
  const shown = useMemo(() => {
    const needle = q.trim().toLowerCase().replace(/^#/, "");
    const pool = (rows ?? []).filter((o) =>
      tab === "live" ? isLive(o) : !isLive(o),
    );
    if (!needle) return pool;
    return pool.filter((o) =>
      [o.number, o.who ?? "", o.phone ?? ""]
        .join(" ")
        .toLowerCase()
        .includes(needle),
    );
  }, [rows, tab, q]);

  const liveCount = (rows ?? []).filter(isLive).length;
  const doneCount = (rows ?? []).length - liveCount;

  async function take(o: OnlineOrder) {
    if (
      !(await ask({
        title: t.online.tookConfirm(formatPrice(o.total)),
        danger: true,
      }))
    )
      return;
    setBusy(o.id);
    try {
      await api.tillOnlinePaid(o.id);
      // Re-read rather than patched here: whether this settles a courier is the
      // server's answer, and a row that marked itself would hide a refusal.
      load();
      setOpen(null);
    } catch (e) {
      onError(e instanceof Error ? e.message : t.online.tookFailed);
    } finally {
      setBusy("");
    }
  }

  if (!rows) return null;

  return (
    <div className="flex h-full flex-col">
      {/* ⚠️ The one figure worth the top of the screen: what should come back
          to this counter before the shift closes. A cashier counting the drawer
          needs it before they start counting, not after. */}
      <div className="flex items-baseline justify-between gap-3 border-b border-line px-4 py-3">
        <span className="text-sm text-ink-muted">{t.online.owed}</span>
        <span className="text-xl font-semibold tabular-nums">
          {formatPrice(owed)}
        </span>
      </div>

      <div className="flex flex-wrap items-center gap-2 border-b border-line px-4 py-2">
        {/* Two lists, named and counted: the count is what tells somebody
            whether the other tab is worth opening. */}
        {(["live", "done"] as Tab[]).map((k) => (
          <button
            key={k}
            type="button"
            onClick={() => setTab(k)}
            className={`till-btn px-3 py-1.5 text-sm ${
              tab === k ? "till-btn-accent" : ""
            }`}
          >
            {k === "live" ? t.online.tabLive : t.online.tabDone}
            <span className="ml-1.5 tabular-nums opacity-70">
              {k === "live" ? liveCount : doneCount}
            </span>
          </button>
        ))}

        <label className="relative ml-auto flex min-w-[10rem] flex-1 items-center sm:max-w-xs">
          <LuSearch className="pointer-events-none absolute left-3 text-ink-muted" />
          <input
            className="till-input w-full pl-9 pr-8"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder={t.online.search}
            inputMode="search"
          />
          {q !== "" && (
            <button
              type="button"
              onClick={() => setQ("")}
              aria-label={t.common.close}
              className="absolute right-2 p-1 text-ink-muted"
            >
              <LuX />
            </button>
          )}
        </label>
      </div>

      {shown.length === 0 ? (
        <p className="p-6 text-center text-sm text-ink-muted">
          {q.trim() !== ""
            ? t.online.nothingFound
            : tab === "live"
              ? t.online.empty
              : t.online.emptyDone}
        </p>
      ) : (
        <ul className="flex-1 divide-y divide-line overflow-y-auto">
          {shown.map((o) => {
            const look = LOOK[o.settle] ?? LOOK.nothing;
            return (
              <li key={o.id}>
                {/* ⚠️ The whole row opens the order. A cashier holding a phone
                    to their ear is not aiming at a chevron. */}
                <button
                  type="button"
                  onClick={() => setOpen(o)}
                  className="flex w-full items-start gap-3 px-4 py-3 text-left active:bg-ink/5"
                >
                  <span className={`mt-0.5 text-lg ${look.tone}`}>
                    {look.icon}
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="flex flex-wrap items-baseline gap-x-2">
                      <span className="font-semibold">#{o.number}</span>
                      <span className="text-xs text-ink-muted">
                        {o.type === "pickup" ? (
                          <LuShoppingBag className="inline" />
                        ) : (
                          <LuBike className="inline" />
                        )}{" "}
                        {t.online.types[o.type] ?? o.type}
                        {" · "}
                        {t.status[o.status as keyof typeof t.status] ??
                          o.status}
                      </span>
                      <span className="ml-auto font-semibold tabular-nums">
                        {formatPrice(o.total)}
                      </span>
                    </span>
                    <span className={`block text-sm ${look.tone}`}>
                      {t.online.settle[o.settle] ?? o.settle}
                    </span>
                    {(o.who || o.phone) && (
                      <span className="mt-0.5 block truncate text-xs text-ink-muted">
                        {[o.who, o.phone, formatTime(o.at)]
                          .filter(Boolean)
                          .join(" · ")}
                      </span>
                    )}
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      )}

      {open && (
        <OrderCard
          row={open}
          busy={busy === open.id}
          onClose={() => setOpen(null)}
          onTake={() => void take(open)}
          onError={onError}
        />
      )}
    </div>
  );
}

/** One order, opened.
 *
 *  ⚠️ **Fetched when it is opened rather than carried in the list.** The list is
 *  polled every half minute and read at a glance; dragging every dish of two
 *  hundred orders through that poll is a menu's worth of JSON nobody looks at.
 */
function OrderCard({
  row,
  busy,
  onClose,
  onTake,
  onError,
}: {
  row: OnlineOrder;
  busy: boolean;
  onClose: () => void;
  onTake: () => void;
  onError: (m: string) => void;
}) {
  const t = useAdminT();
  const [order, setOrder] = useState<Order | null>(null);

  useEffect(() => {
    let alive = true;
    api
      .tillOnlineOrder(row.id)
      .then((r) => alive && setOrder(r.order))
      .catch((e) => onError(e instanceof Error ? e.message : ""));
    return () => {
      alive = false;
    };
  }, [row.id, onError]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center"
      onClick={onClose}
      role="presentation"
    >
      <div
        className="till-dialog flex max-h-[86dvh] w-full max-w-md flex-col p-4"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-3">
          <div>
            <h2 className="font-display text-lg font-bold">#{row.number}</h2>
            <p className="text-sm text-ink-muted">
              {t.online.types[row.type] ?? row.type} ·{" "}
              {t.status[row.status as keyof typeof t.status] ?? row.status} ·{" "}
              {formatTime(row.at)}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label={t.common.close}
            className="rounded-full p-1.5 text-ink-muted hover:bg-ink/5"
          >
            <LuX className="h-5 w-5" />
          </button>
        </div>

        {/* ⚠️ The phone as a link, not as text: the reason a cashier opens one
            of these is usually that they have to ring somebody. */}
        {(row.who || row.phone) && (
          <p className="mt-3 text-sm">
            {row.who}
            {row.phone && (
              <>
                {row.who ? " · " : ""}
                <a
                  href={`tel:${row.phone}`}
                  className="inline-flex items-center gap-1 font-semibold"
                >
                  <LuPhone /> {row.phone}
                </a>
              </>
            )}
          </p>
        )}

        {order?.address?.text && (
          <p className="mt-1 text-sm text-ink-soft">{order.address.text}</p>
        )}
        {order?.address?.comment && (
          <p className="text-xs text-ink-muted">{order.address.comment}</p>
        )}

        <div className="mt-3 min-h-0 flex-1 overflow-y-auto">
          {order ? (
            <ul className="space-y-1 text-sm">
              {order.items.map((it, i) => (
                <li key={i} className="flex items-baseline gap-2">
                  <span className="tabular-nums text-ink-muted">{it.qty}×</span>
                  <span className="min-w-0 flex-1">
                    {it.name}
                    {it.comment ? (
                      <span className="block text-xs text-ink-muted">
                        {it.comment}
                      </span>
                    ) : null}
                  </span>
                  <span className="tabular-nums">
                    {formatPrice(it.price * it.qty)}
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-sm text-ink-muted">{t.common.loading}</p>
          )}
        </div>

        <div className="mt-3 space-y-1 border-t border-line pt-3 text-sm">
          {(order?.deliveryFee ?? 0) > 0 && (
            <p className="flex justify-between">
              <span className="text-ink-muted">{t.online.deliveryFee}</span>
              <span className="tabular-nums">
                {formatPrice(order?.deliveryFee ?? 0)}
              </span>
            </p>
          )}
          <p className="flex justify-between font-semibold">
            <span>{t.online.total}</span>
            <span className="tabular-nums">{formatPrice(row.total)}</span>
          </p>
          <p className="flex justify-between text-ink-muted">
            <span>{t.online.payment}</span>
            <span>
              {t.till.methodName[row.paymentMethod] ?? row.paymentMethod}
              {" · "}
              {row.paymentStatus === "paid"
                ? t.online.paid
                : t.online.notPaidYet}
            </span>
          </p>
        </div>

        {/* ⚠️ Only where money is actually owed. A paid order has nothing to
            take, and a button on it would be a way to record a handover that
            never happened. */}
        {row.settle !== "nothing" && (
          <button
            type="button"
            disabled={busy}
            onClick={onTake}
            className="till-btn-primary mt-4 w-full py-3 text-base disabled:opacity-50"
          >
            {t.online.took[row.settle] ?? t.online.took.at_counter}
          </button>
        )}
      </div>
    </div>
  );
}
