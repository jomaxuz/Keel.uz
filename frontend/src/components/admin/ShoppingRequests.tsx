"use client";

// ---- The morning, as the office sees it ----
//
// ⚠️ **The one screen that shows the whole errand.** The barman sees what he
// asked for, the buyer sees his half, the storekeeper sees theirs — and until
// this existed nobody could see the sentence the three of them make together:
// asked at six, half of it picked off a shelf at ten past, the fruit bought at
// nine, and one line of it never signed for. That last state is the reason to
// open a panel at all, and it was invisible from every other screen here.
//
// ⚠️ **Beside the computed list rather than replacing it.** They answer
// different questions — one is arithmetic about shelves, the other is a record
// of what people did — and a screen that merged them would let a suggestion and
// a request argue about "do we need beef", which is the argument that happens
// *after* the money is spent.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import type { ShoppingOrder, ShoppingRequestGroup } from "@/lib/types";

export default function ShoppingRequests() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [groups, setGroups] = useState<ShoppingRequestGroup[]>([]);
  const [openOnly, setOpenOnly] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api
      .adminShoppingRequests(openOnly)
      .then((d) => {
        setGroups(d.groups);
        setError("");
      })
      // ⚠️ **Said, not swallowed.** An empty list here means "nobody asked for
      // anything", which is a real and ordinary answer — so a failed load that
      // rendered the same emptiness would be indistinguishable from a quiet
      // fortnight, and the one of the two that needs action is the silent one.
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      );
  }, [openOnly, t.common.loadFailed]);

  useEffect(load, [load, scope.scopeKey]);

  return (
    <section className="space-y-3">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <div>
          <h2 className="text-lg font-semibold">{t.shopping.requestsTitle}</h2>
          <p className="mt-1 text-sm text-ink-soft">
            {t.shopping.requestsIntro}
          </p>
        </div>
        <label className="flex items-center gap-2 text-sm text-ink-soft">
          <input
            type="checkbox"
            checked={openOnly}
            onChange={(e) => setOpenOnly(e.target.checked)}
          />
          {t.shopping.requestsOpenOnly}
        </label>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      {groups.length === 0 ? (
        <div className="card p-6 text-center text-sm text-ink-muted">
          {t.shopping.requestsEmpty}
        </div>
      ) : (
        <div className="space-y-3">
          {groups.map((g) => (
            <div key={g.groupId} className="card p-0">
              <div className="flex flex-wrap items-baseline justify-between gap-2 border-b border-line px-3 py-2">
                <div className="text-sm font-medium">
                  {formatDate(g.forDate)}
                  {g.createdBy && (
                    <span className="ml-2 text-xs font-normal text-ink-muted">
                      {t.shopping.askedBy(g.createdBy)}
                    </span>
                  )}
                </div>
                {g.branch && (
                  <span className="text-xs text-ink-muted">{g.branch}</span>
                )}
              </div>
              {g.orders.map((o) => (
                <Half key={o.id} order={o} supply={g.supply} />
              ))}
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

/** One half of one request: where it went, who answered it, and whether anybody
 *  ever signed for it. */
function Half({ order, supply }: { order: ShoppingOrder; supply?: string }) {
  const t = useAdminT();
  const store = order.source === "store";
  // ⚠️ **Three states, and only the middle one is worth a colour.** "Waiting"
  // is what every request looks like for its first hour and "signed for" is the
  // end; the one an owner has to act on is goods that left somebody's hands and
  // reached nobody's — the state this whole feature was built to make visible.
  const status =
    order.status === "done"
      ? { text: t.shopping.reqDone, cls: "text-ink-muted" }
      : order.status === "shipped"
        ? {
            text: t.shopping.reqShipped,
            cls: "font-semibold text-amber-700 dark:text-amber-300",
          }
        : { text: t.shopping.reqSent, cls: "text-ink-soft" };

  return (
    <div className="border-b border-line px-3 py-2 last:border-b-0">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <span className="text-sm font-medium">
          {store ? t.shopping.reqFromStore : t.shopping.reqFromMarket}
          {store && supply && (
            <span className="ml-2 text-xs font-normal text-ink-muted">
              {t.shopping.fromBranch(supply)}
            </span>
          )}
        </span>
        <span className={`text-xs ${status.cls}`}>{status.text}</span>
      </div>
      <p className="text-xs text-ink-muted">
        {[
          order.shippedBy && t.shopping.shippedBy(order.shippedBy),
          order.acceptedBy && t.shopping.acceptedBy(order.acceptedBy),
        ]
          .filter(Boolean)
          .join(" · ")}
      </p>
      <ul className="mt-1 space-y-0.5 text-sm">
        {order.lines.map((l) => {
          const unit = l.unit ?? "";
          // ⚠️ **All three figures, side by side.** "Asked for ten, sent six,
          // counted five" is the sentence this document exists to make
          // possible; showing only the last one would put back the silence the
          // buying had before any of this existed.
          const parts = [t.shopping.lineAsked(`${l.qty} ${unit}`)];
          if (l.missing) parts.push(t.shopping.lineMissing);
          else if (l.gotAt) parts.push(t.shopping.lineGot(`${l.gotQty} ${unit}`));
          // ⚠️ `!= null`, never truthiness: a counted zero is a real answer —
          // the bag arrived empty — and it is exactly the row somebody has to
          // be asked about.
          if (l.tookQty != null) {
            parts.push(t.shopping.lineTook(`${l.tookQty} ${unit}`));
          }
          return (
            <li key={l.id} className="flex flex-wrap gap-x-2">
              <span className="font-medium">{l.name}</span>
              <span
                className={
                  l.missing || (l.tookQty != null && l.tookQty < l.gotQty!)
                    ? "text-danger"
                    : "text-ink-muted"
                }
              >
                {parts.join(" · ")}
              </span>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
