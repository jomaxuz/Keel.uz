"use client";

// What came in against what the food used.
//
// ⚠️ **Not a stock balance, and the screen leads with that.** There is no
// opening count, no write-offs and no stocktake in this system, so a
// "remaining" figure would be a number nobody can check and everybody
// believes. What can be said honestly is a flow: forty kilos of beef came
// through the door this month and the dishes sold account for twenty-six.
//
// ⚠️ **The difference is the question, not the answer.** A heavy hand, a
// dropped tray, a portion given to a regular and a delivery still in the fridge
// on the last day all live in that column. Somebody who knows the kitchen reads
// it; the panel must not pretend to.

import { useCallback, useEffect, useState } from "react";

import { api, downloadReport } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { ListScroll } from "@/components/admin/PagedList";
import type { StockReportResponse } from "@/lib/types";

type Range = { from?: string; to?: string };

export default function StockReport({ range }: { range: Range }) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [data, setData] = useState<StockReportResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const params = useCallback(() => ({ ...range }), [range]);

  useEffect(() => {
    setLoading(true);
    api
      .stockReport(params())
      .then(setData)
      .catch(() => setError(t.common.loadFailed))
      .finally(() => setLoading(false));
  }, [params, t]);

  async function download() {
    setBusy(true);
    try {
      await downloadReport("/admin/reports/stock", params());
    } catch {
      setError(t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  const rows = data?.rows ?? [];
  const qty = (n: number) =>
    n.toLocaleString(lang === "ru" ? "ru-RU" : "en-US", {
      maximumFractionDigits: 3,
    });

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-lg font-semibold text-ink">
          {t.reports.stock.title}
        </h2>
        <button
          type="button"
          onClick={download}
          disabled={busy || rows.length === 0}
          className="btn btn-primary disabled:opacity-60"
        >
          {busy ? t.common.loading : t.reports.excel}
        </button>
      </div>

      {error && <p className="text-sm text-brand">{error}</p>}
      {/* The server's own sentence, so the screen and the spreadsheet cannot
          carry two different warnings. */}
      {data?.note && <p className="text-sm text-ink-soft">{data.note}</p>}

      {loading ? (
        <p className="py-10 text-center text-ink-muted/70">
          {t.common.loading}
        </p>
      ) : rows.length === 0 ? (
        <p className="py-10 text-center text-ink-muted/70">
          {t.reports.stock.empty}
        </p>
      ) : (
        <div className="card p-0">
          <ListScroll>
            <table className="w-full text-sm">
              <thead className="sticky top-0 bg-raised text-left text-xs uppercase tracking-wider text-ink-muted">
                <tr>
                  <th className="px-3 py-2">{t.reports.stock.ingredient}</th>
                  <th className="px-3 py-2 text-right">{t.reports.stock.in}</th>
                  <th className="px-3 py-2 text-right">
                    {t.reports.stock.used}
                  </th>
                  <th className="px-3 py-2 text-right">
                    {t.reports.stock.written}
                  </th>
                  <th className="px-3 py-2 text-right">
                    {t.reports.stock.diff}
                  </th>
                  <th className="px-3 py-2 text-right">
                    {t.reports.stock.spent}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line">
                {rows.map((r) => (
                  <tr key={r.name}>
                    <td className="px-3 py-2 font-medium text-ink">
                      {r.name}
                      <span className="ml-1 text-xs text-ink-muted">
                        {t.ingredients.units[r.unit] ?? r.unit}
                      </span>
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums">
                      {qty(r.in)}
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums text-ink-soft">
                      {qty(r.used)}
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums text-ink-soft">
                      {r.written ? qty(r.written) : "—"}
                    </td>
                    {/* ⚠️ Not coloured red. A difference is expected — food
                        bought on the last day has not been cooked yet — and
                        painting every row as an alarm is how a screen stops
                        being read. */}
                    <td className="px-3 py-2 text-right tabular-nums">
                      {qty(r.diff)}
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums">
                      {formatPrice(r.spent, "UZS", lang)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </ListScroll>
          <div className="border-t border-line px-3 py-2 text-right text-sm">
            {t.reports.stock.spent}:{" "}
            <span className="font-medium tabular-nums">
              {formatPrice(data?.spent ?? 0, "UZS", lang)}
            </span>
            {/* What was thrown away, in money. One number for the period,
                under the table rather than repeated on every row — a running
                total in a column invites somebody to add it up twice. */}
            {(data?.writtenValue ?? 0) > 0 && (
              <span className="ml-4 text-ink-muted">
                {t.reports.stock.written}:{" "}
                <span className="tabular-nums">
                  {formatPrice(data?.writtenValue ?? 0, "UZS", lang)}
                </span>
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
