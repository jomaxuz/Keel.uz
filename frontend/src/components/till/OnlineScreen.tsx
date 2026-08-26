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

import { useCallback, useEffect, useState } from "react";
import {
  LuBanknote,
  LuBike,
  LuCheck,
  LuCircleAlert,
  LuPhone,
  LuShoppingBag,
} from "react-icons/lu";

import { api } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import type { OnlineOrder } from "@/lib/types";

/** What each answer looks like. ⚠️ "Nothing to do" is deliberately the quiet
 *  one — it is the commonest row and a cashier has to be able to skip it
 *  without reading it. The loud one is the order that was started online and
 *  never finished, because the kitchen may already be cooking. */
const LOOK: Record<
  string,
  { icon: React.ReactNode; tone: string }
> = {
  nothing: { icon: <LuCheck />, tone: "text-ink-muted" },
  from_courier: { icon: <LuBike />, tone: "text-amber-700 dark:text-amber-300" },
  at_counter: { icon: <LuBanknote />, tone: "text-emerald-700 dark:text-emerald-400" },
  unfinished: { icon: <LuCircleAlert />, tone: "text-danger" },
};

export default function OnlineScreen({
  onError,
}: {
  onError: (m: string) => void;
}) {
  const t = useAdminT();
  const [rows, setRows] = useState<OnlineOrder[] | null>(null);
  const [owed, setOwed] = useState(0);

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

      {rows.length === 0 ? (
        <p className="p-6 text-center text-sm text-ink-muted">{t.online.empty}</p>
      ) : (
        <ul className="flex-1 divide-y divide-line overflow-y-auto">
          {rows.map((o) => {
            const look = LOOK[o.settle] ?? LOOK.nothing;
            return (
              <li key={o.id} className="flex items-start gap-3 px-4 py-3">
                <span className={`mt-0.5 text-lg ${look.tone}`}>{look.icon}</span>
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-baseline gap-x-2">
                    <span className="font-semibold">#{o.number}</span>
                    <span className="text-xs text-ink-muted">
                      {o.type === "pickup" ? (
                        <LuShoppingBag className="inline" />
                      ) : (
                        <LuBike className="inline" />
                      )}{" "}
                      {t.online.types[o.type] ?? o.type}
                    </span>
                    <span className="ml-auto font-semibold tabular-nums">
                      {formatPrice(o.total)}
                    </span>
                  </div>
                  <div className={`text-sm ${look.tone}`}>
                    {t.online.settle[o.settle] ?? o.settle}
                  </div>
                  {(o.who || o.phone) && (
                    <div className="mt-0.5 flex flex-wrap items-center gap-x-2 text-xs text-ink-muted">
                      {o.who && <span>{o.who}</span>}
                      {/* ⚠️ A tel: link, not text. The cashier reading this is
                          holding a phone or standing at a screen with a headset
                          beside it, and the reason they are looking at this row
                          is usually that they need to call somebody. */}
                      {o.phone && (
                        <a href={`tel:${o.phone}`} className="inline-flex items-center gap-1">
                          <LuPhone /> {o.phone}
                        </a>
                      )}
                    </div>
                  )}
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
