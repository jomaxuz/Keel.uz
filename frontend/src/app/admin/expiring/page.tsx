"use client";

// What goes out of date, and when.
//
// ⚠️ **A pharmacy is inspected on this and a grocery loses money to it**, and
// until this screen existed the only place a date was written down was the box
// itself — which is to say nowhere anybody looks until an inspector does.
//
// ⚠️ **Built from deliveries, and it says so on the screen.** Consumption keys
// on the ingredient rather than on the box, so this cannot claim "three of
// these four are still here". It reports what came in and until when it is
// good, which is exactly what the invoice and the package say. Inventing the
// subtraction would give a figure that is right most of the time — and a
// pharmacist who finds one stock number wrong stops believing all of them.

import { useCallback, useEffect, useState } from "react";

import { api } from "@/lib/api";
import type { ExpiringRow } from "@/lib/types";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";

/** How far ahead to look. ⚠️ Sixty days first, because that is roughly while a
 *  supplier will still take a box back; past that the choices are a discount or
 *  a write-off. */
const WINDOWS = [30, 60, 90, 180];

export default function ExpiringPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [rows, setRows] = useState<ExpiringRow[]>([]);
  const [days, setDays] = useState(60);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    setLoading(true);
    api
      .adminExpiring(days)
      .then((d) => {
        setRows(d.rows);
        setError("");
      })
      .catch(() => setError(t.common.loadFailed))
      .finally(() => setLoading(false));
  }, [days, t.common.loadFailed]);

  useEffect(load, [load, scope.scopeKey]);

  return (
    <div>
      <h1 className="text-2xl font-bold">{t.expiring.title}</h1>
      <p className="mt-2 max-w-3xl text-sm text-ink-muted">{t.expiring.intro}</p>

      <div className="mt-4 flex flex-wrap gap-2">
        {WINDOWS.map((d) => (
          <button
            key={d}
            type="button"
            onClick={() => setDays(d)}
            className={`rounded-lg border px-3 py-1.5 text-sm font-semibold ${
              d === days ? "border-brand bg-brand/10" : "border-line"
            }`}
          >
            {t.expiring.window(d)}
          </button>
        ))}
      </div>

      {error && <p className="mt-4 text-sm text-danger">{error}</p>}

      {loading ? (
        <p className="mt-6 text-sm text-ink-muted">{t.common.loading}</p>
      ) : rows.length === 0 ? (
        <p className="mt-6 text-sm text-ink-muted">{t.expiring.empty}</p>
      ) : (
        <div className="mt-6 overflow-x-auto rounded-3xl border border-line bg-surface shadow-card">
          <table className="w-full text-sm">
            <thead className="text-left text-xs uppercase text-ink-muted/70">
              <tr className="border-b border-line">
                <th className="px-4 py-3">{t.expiring.name}</th>
                <th className="px-4 py-3">{t.expiring.series}</th>
                <th className="px-4 py-3">{t.expiring.date}</th>
                <th className="px-4 py-3">{t.expiring.left}</th>
                <th className="px-4 py-3 text-right">{t.expiring.qty}</th>
                <th className="px-4 py-3">{t.expiring.supplier}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {rows.map((r, i) => (
                <tr key={`${r.purchaseId}-${r.ingredientId}-${i}`}>
                  <td className="px-4 py-3 font-medium">{r.name || "—"}</td>
                  <td className="px-4 py-3 text-ink-muted">{r.series || "—"}</td>
                  <td className="px-4 py-3 tabular-nums">
                    {r.expiresAt.slice(0, 10)}
                  </td>
                  {/* ⚠️ **Already past comes first and reads differently.** A
                      list sorted only by date buries what is a problem today
                      under what will be a problem in November, and today's is
                      the only row somebody has to walk to a shelf about. */}
                  <td
                    className={`px-4 py-3 tabular-nums ${
                      r.daysLeft < 0
                        ? "font-semibold text-danger"
                        : r.daysLeft <= 14
                          ? "text-warn"
                          : "text-ink-muted"
                    }`}
                  >
                    {r.daysLeft < 0
                      ? t.expiring.passed(-r.daysLeft)
                      : t.expiring.daysLeft(r.daysLeft)}
                  </td>
                  <td className="px-4 py-3 text-right tabular-nums">
                    {r.qty} {r.unit}
                  </td>
                  <td className="px-4 py-3 text-ink-muted">{r.supplier || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <p className="mt-4 max-w-3xl text-xs text-ink-muted">{t.expiring.note}</p>
    </div>
  );
}
