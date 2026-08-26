"use client";

// Who takes money off tables, each person beside their colleagues.
//
// ⚠️ **No number here is evidence of anything.** A void is normal, a discount
// is normal, and a cashier who does neither is probably not serving anybody.
// The only reading worth making is one person against the others on the same
// shifts: forty voids means nothing, forty when the rest did four is a
// question. So every column is a share, and the screen refuses to draw at all
// when there is nobody to compare against.
//
// ⚠️ **This is a screen to show the staff, not to keep from them.** Most of the
// value is spent before anybody reads it — a cashier who knows their void rate
// sits beside their colleagues' voids less. Built as a secret investigation
// tool it would forfeit the larger half of the benefit and turn the smaller
// half adversarial.

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import type { LossRow } from "@/lib/types";

export default function LossReport({
  range,
}: {
  range: { from?: string; to?: string };
}) {
  const t = useAdminT();
  const [rows, setRows] = useState<LossRow[] | null>(null);
  const [comparable, setComparable] = useState(true);

  useEffect(() => {
    let alive = true;
    api
      .adminLossReport(range)
      .then((r) => {
        if (!alive) return;
        setRows(r.rows ?? []);
        setComparable(r.comparable);
      })
      .catch(() => alive && setRows([]));
    return () => {
      alive = false;
    };
  }, [range]);

  if (!rows) return null;
  if (rows.length === 0) {
    return <p className="p-4 text-sm text-ink-muted">{t.loss.empty}</p>;
  }

  // ⚠️ The comparison line, computed here rather than sent: it is a property of
  // whoever happens to be on screen, and a server-side average would go stale
  // the moment the period changed.
  const totalChecks = rows.reduce((a, r) => a + r.checks, 0);
  const totalVoids = rows.reduce((a, r) => a + r.voids, 0);
  const houseVoidShare = totalChecks ? (totalVoids * 1000) / totalChecks : 0;

  return (
    <div className="space-y-3">
      <p className="text-sm text-ink-soft">{t.loss.intro}</p>

      {/* ⚠️ Said before the table, not after it. A person reads the numbers
          first and the caveat second, so the caveat has to be above them. */}
      {!comparable && (
        <p className="rounded-xl border border-amber-400/60 bg-amber-50 p-3 text-sm dark:bg-amber-950/30">
          {t.loss.notComparable}
        </p>
      )}

      <div className="card p-0">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="text-left text-xs text-ink-muted">
              <tr>
                <th className="px-3 py-2">{t.loss.person}</th>
                <th className="px-3 py-2 text-right">{t.loss.checks}</th>
                <th className="px-3 py-2 text-right">{t.loss.voidShare}</th>
                <th className="px-3 py-2 text-right">{t.loss.voidValue}</th>
                <th className="px-3 py-2 text-right">{t.loss.selfAuthed}</th>
                <th className="px-3 py-2 text-right">{t.loss.discountShare}</th>
                <th className="px-3 py-2 text-right">{t.loss.cash}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => {
                // ⚠️ Marked only against the house's own rate, never against a
                // fixed threshold. A steakhouse and a canteen void at
                // completely different rates, and a number we chose would be
                // wrong in one of them every day.
                const high =
                  comparable && houseVoidShare > 0 && r.voidShare > houseVoidShare * 2;
                return (
                  <tr key={r.id} className="border-t border-line">
                    <td className="px-3 py-2">{r.name || "—"}</td>
                    <td className="px-3 py-2 text-right tabular-nums">
                      {r.checks}
                    </td>
                    <td
                      className={`px-3 py-2 text-right tabular-nums ${
                        high ? "font-semibold text-amber-700 dark:text-amber-300" : ""
                      }`}
                    >
                      {(r.voidShare / 10).toFixed(1)}%
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums">
                      {formatPrice(r.voidValue)}
                    </td>
                    {/* Voids they did on their own authority — not wrong, and
                        the whole point of holding the permission. It is here
                        because it is the half nobody else saw happen. */}
                    <td className="px-3 py-2 text-right tabular-nums text-ink-muted">
                      {r.authedSelf}
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums">
                      {(r.discountShare / 10).toFixed(1)}%
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums text-ink-muted">
                      {formatPrice(r.cashSales)}
                    </td>
                  </tr>
                );
              })}
            </tbody>
            <tfoot>
              <tr className="border-t-2 border-line text-xs text-ink-muted">
                <td className="px-3 py-2">{t.loss.house}</td>
                <td className="px-3 py-2 text-right tabular-nums">
                  {totalChecks}
                </td>
                <td className="px-3 py-2 text-right tabular-nums">
                  {(houseVoidShare / 10).toFixed(1)}%
                </td>
                <td colSpan={4} />
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      <p className="text-xs text-ink-muted">{t.loss.footnote}</p>
    </div>
  );
}
