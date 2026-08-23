"use client";

// What to buy.
//
// ⚠️ **The warning existed and led nowhere.** A minimum could be set on an
// ingredient and the balance screen turned the row amber when the shelf fell
// below it — and then stopped. The owner read "we are low on four things",
// opened a notebook and wrote them down again. This is the half that was
// missing, and it is the same shape as the customer segments that had nowhere
// to lead until campaigns existed: a condition worth noticing has to end in the
// action it implies.
//
// ⚠️ **Grouped by supplier**, because that is how shopping is actually done. A
// flat list sorted by urgency means reading forty rows to find the four to
// mention to the butcher, and then reading them again for the next call.

import { useCallback, useEffect, useState } from "react";

import { api } from "@/lib/api";
import { formatDate, formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import type { ShoppingGroup } from "@/lib/types";

export default function ShoppingPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [groups, setGroups] = useState<ShoppingGroup[]>([]);
  const [cost, setCost] = useState(0);
  const [since, setSince] = useState<string | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api
      .adminShoppingList()
      .then((d) => {
        setGroups(d.groups);
        setCost(d.cost);
        setSince(d.since);
      })
      .catch(() => setError(t.common.loadFailed));
  }, [t.common.loadFailed]);

  useEffect(load, [load, scope.scopeKey]);

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.shopping.title}</h1>
        <p className="mt-1 text-sm text-ink-soft">{t.shopping.intro}</p>
        {/* ⚠️ The same caveat the balance carries, for the same reason: a
            suggestion to buy nine kilos is worth a different amount of trust
            depending on whether the shelf was counted last night or in March. */}
        <p className="mt-1 text-xs text-ink-muted">
          {since ? t.shopping.since(formatDate(since)) : t.shopping.neverCounted}
        </p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      {groups.length === 0 ? (
        <div className="card p-6 text-center text-sm text-ink-muted">
          {/* ⚠️ Not "no data": an empty list here is good news, and phrasing it
              as an absence teaches people the screen is broken. */}
          {t.shopping.nothingNeeded}
        </div>
      ) : (
        <div className="space-y-3">
          {groups.map((g) => (
            <div key={g.supplierId} className="card p-0">
              <div className="flex flex-wrap items-baseline justify-between gap-2 border-b border-line px-3 py-2">
                <div className="text-sm font-medium">
                  {g.name || t.shopping.noSupplier}
                  {g.phone && (
                    // One tap to make the call the list exists to prompt.
                    <a
                      className="ml-2 text-xs text-ink-muted underline"
                      href={`tel:${g.phone}`}
                    >
                      {g.phone}
                    </a>
                  )}
                </div>
                <span className="text-sm tabular-nums text-ink-soft">
                  ≈ {formatPrice(g.cost)}
                </span>
              </div>
              <table className="w-full text-sm">
                <thead className="text-left text-xs text-ink-muted">
                  <tr>
                    <th className="px-3 py-1.5">{t.shopping.what}</th>
                    <th className="px-3 py-1.5 text-right">
                      {t.shopping.onHand}
                    </th>
                    <th className="px-3 py-1.5 text-right">
                      {t.shopping.minimum}
                    </th>
                    <th className="px-3 py-1.5 text-right">
                      {t.shopping.buy}
                    </th>
                    <th className="px-3 py-1.5 text-right">
                      {t.shopping.cost}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {g.rows.map((row) => {
                    const unit = t.ingredients.units[row.unit] ?? row.unit;
                    return (
                      <tr key={row.ingredientId} className="border-t border-line">
                        <td className="px-3 py-2">{row.name}</td>
                        <td className="px-3 py-2 text-right tabular-nums">
                          {/* Empty shelves read differently from low ones, and
                              the difference is what decides the order today. */}
                          <span
                            className={
                              row.onHand <= 0
                                ? "font-semibold text-danger"
                                : "text-amber-700 dark:text-amber-300"
                            }
                          >
                            {row.onHand} {unit}
                          </span>
                        </td>
                        <td className="px-3 py-2 text-right tabular-nums text-ink-muted">
                          {row.minQty} {unit}
                        </td>
                        <td className="px-3 py-2 text-right font-medium tabular-nums">
                          {row.suggested} {unit}
                        </td>
                        <td className="px-3 py-2 text-right tabular-nums text-ink-soft">
                          {formatPrice(row.cost)}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ))}
          <div className="card px-3 py-2 text-right text-sm">
            {/* ⚠️ "About", and it is not a commitment: the quantities are the
                gap to a reorder point and the prices are the last ones paid. */}
            {t.shopping.total}:{" "}
            <span className="font-medium tabular-nums">
              ≈ {formatPrice(cost)}
            </span>
          </div>
        </div>
      )}
    </div>
  );
}
