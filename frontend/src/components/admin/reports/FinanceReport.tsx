"use client";

// Where the money came from and where it went.
//
// ⚠️ **This is not a profit and loss statement, and the screen says so before
// it shows a single number.** "In − out" is cash movement: what a plate costs
// the kitchen is only in here for the dishes somebody priced, and rent, tax and
// half the wages are not in the system at all. The report has always carried
// that sentence — it had no screen to carry it on, which is the gap this
// closes: a report reachable only as a spreadsheet is a report read without its
// note.
//
// ⚠️ **Three kinds of line, kept apart on purpose**: money in, money out, and
// information — figures an owner wants that are *not* movements (discounts
// given, the cost of food, the gross margin). Mixing them is exactly how a
// dashboard once came to call an uncooked order "revenue".

import { useCallback, useEffect, useState } from "react";

import { api, downloadReport } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import type { FinanceReportResponse } from "@/lib/types";

type Range = { from?: string; to?: string };

export default function FinanceReport({ range }: { range: Range }) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [data, setData] = useState<FinanceReportResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const params = useCallback(() => ({ ...range }), [range]);

  useEffect(() => {
    setLoading(true);
    api
      .financeReport(params())
      .then(setData)
      .catch(() => setError(t.common.loadFailed))
      .finally(() => setLoading(false));
  }, [params, t]);

  async function download() {
    setBusy(true);
    setError("");
    try {
      await downloadReport("/admin/reports/finance", params());
    } catch {
      setError(t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  const money = (n: number) => formatPrice(n, "UZS", lang);
  const f = t.reports.finance;
  const lines = data?.lines ?? [];
  const totals = data?.totals;

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-lg font-semibold text-ink">{f.title}</h2>
        <button
          type="button"
          onClick={download}
          disabled={busy || lines.length === 0}
          className="btn btn-primary disabled:opacity-60"
        >
          {busy ? t.common.loading : t.reports.excel}
        </button>
      </div>

      {error && <p className="text-sm text-brand">{error}</p>}
      {/* The note comes from the server, so the screen and the spreadsheet
          carry the same warning word for word. */}
      {data?.note && <p className="text-sm text-ink-soft">{data.note}</p>}

      {loading ? (
        <p className="py-10 text-center text-ink-muted/70">
          {t.common.loading}
        </p>
      ) : lines.length === 0 ? (
        <p className="py-10 text-center text-ink-muted/70">{f.empty}</p>
      ) : (
        <>
          {totals && (
            <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
              <Kpi label={f.in} value={money(totals.in)} />
              <Kpi label={f.out} value={money(totals.out)} />
              <Kpi label={f.net} value={money(totals.net)} strong />
              {/* ⚠️ Outside the totals, deliberately: it is real money and it
                  has not arrived. Adding it would be the old dashboard's
                  mistake with a different label. */}
              <Kpi label={f.pending} value={money(totals.pending)} muted />
            </div>
          )}

          <div className="card overflow-hidden p-0">
            <table className="w-full text-sm">
              <tbody className="divide-y divide-line">
                {lines.map((l, i) => (
                  <tr
                    key={i}
                    className={l.kind === "info" ? "text-ink-muted" : ""}
                  >
                    <td
                      className={`px-3 py-2 ${l.sub ? "pl-8 text-ink-soft" : ""}`}
                    >
                      {l.label}
                      {l.count > 0 && (
                        <span className="ml-1.5 text-xs text-ink-muted">
                          ×{l.count}
                        </span>
                      )}
                    </td>
                    <td
                      className={`px-3 py-2 text-right tabular-nums ${
                        l.kind === "out" ? "text-danger" : ""
                      }`}
                    >
                      {/* A minus sign on the way out: a column of unsigned
                          numbers is a column somebody adds up wrongly. */}
                      {l.kind === "out" ? "−" : ""}
                      {money(l.amount)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}

function Kpi({
  label,
  value,
  strong,
  muted,
}: {
  label: string;
  value: string;
  strong?: boolean;
  muted?: boolean;
}) {
  return (
    <div className="card p-3">
      <div className="text-xs text-ink-muted">{label}</div>
      <div
        className={`mt-0.5 font-display tabular-nums ${
          strong ? "text-2xl" : "text-lg"
        } ${muted ? "text-ink-muted" : ""}`}
      >
        {value}
      </div>
    </div>
  );
}
