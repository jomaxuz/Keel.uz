"use client";

// The till's history: every shift that was counted, and how it came out.
//
// ⚠️ **The rows are the shifts, not a summary**, and that is the whole design.
// A period total of zero can be a Tuesday 80 000 short and a Thursday 80 000
// over — two separate conversations with two different people, netted into a
// number that says nothing happened. The totals row is at the bottom because it
// is the least interesting thing here.
//
// ⚠️ **Every difference carries its sentence.** The server refuses to store one
// without an explanation, so the column is never empty when it matters — and
// showing the note beside the number is what turns this from an accusation into
// a record.

import { useCallback, useEffect, useState } from "react";
import { api, downloadReport } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import type { CashReportResponse, CashShift } from "@/lib/types";

type Range = { from?: string; to?: string };

export default function CashReport({ range }: { range: Range }) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [data, setData] = useState<CashReportResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const params = useCallback(() => ({ ...range }), [range]);

  useEffect(() => {
    setLoading(true);
    api
      .cashReport(params())
      .then(setData)
      .catch(() => setError(t.common.loadFailed))
      .finally(() => setLoading(false));
  }, [params, t]);

  async function download() {
    setBusy(true);
    setError("");
    try {
      await downloadReport("/admin/reports/cash", params());
    } catch {
      setError(t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  const money = (n: number) => formatPrice(n, "UZS", lang);
  const c = t.reports.cash;
  const shifts = data?.shifts ?? [];

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-lg font-semibold text-ink">{c.title}</h2>
        <button
          type="button"
          onClick={download}
          disabled={busy || shifts.length === 0}
          className="btn btn-primary disabled:opacity-60"
        >
          {busy ? t.common.loading : t.reports.excel}
        </button>
      </div>

      {error && <p className="text-sm text-brand">{error}</p>}
      {data?.note && <p className="text-sm text-ink-soft">{data.note}</p>}

      {loading ? (
        <p className="py-10 text-center text-ink-muted/70">
          {t.common.loading}
        </p>
      ) : shifts.length === 0 ? (
        <p className="py-10 text-center text-ink-muted/70">{c.empty}</p>
      ) : (
        <>
          {/* ⚠️ Shifts that did not balance, counted and named above the table.
              A list of thirty rows where three matter buries the three, and
              those three are the only reason anybody opens this screen. */}
          <Offenders shifts={shifts} money={money} />

          <div className="overflow-x-auto">
            <table className="w-full min-w-[46rem] text-sm">
              <thead className="sticky top-0 bg-surface text-left text-xs text-ink-muted">
                <tr className="border-b border-line">
                  <th className="py-2 pr-3 font-medium">{c.opened}</th>
                  <th className="py-2 pr-3 font-medium">{c.closed}</th>
                  <th className="py-2 pr-3 text-right font-medium">
                    {c.expected}
                  </th>
                  <th className="py-2 pr-3 text-right font-medium">
                    {c.counted}
                  </th>
                  <th className="py-2 pr-3 text-right font-medium">
                    {c.variance}
                  </th>
                  <th className="py-2 font-medium">{c.reason}</th>
                </tr>
              </thead>
              <tbody>
                {shifts.map((s) => (
                  <tr key={s.id} className="border-b border-line/60 align-top">
                    <td className="py-2 pr-3">
                      <div>{stamp(s.openedAt)}</div>
                      <div className="text-xs text-ink-muted">{s.openedBy}</div>
                    </td>
                    <td className="py-2 pr-3">
                      <div>{s.closedAt ? stamp(s.closedAt) : "—"}</div>
                      <div className="text-xs text-ink-muted">{s.closedBy}</div>
                    </td>
                    <td className="py-2 pr-3 text-right tabular-nums">
                      {money(s.expected)}
                    </td>
                    <td className="py-2 pr-3 text-right tabular-nums">
                      {money(s.counted)}
                    </td>
                    <td
                      className={`py-2 pr-3 text-right tabular-nums font-medium ${
                        s.variance === 0 ? "text-ink-muted" : "text-danger"
                      }`}
                    >
                      {s.variance > 0 ? "+" : ""}
                      {money(s.variance)}
                    </td>
                    <td className="py-2 text-xs text-ink-soft">
                      {s.varianceNote}
                      {/* ⚠️ The register's own figure, shown only where it
                          disagrees with ours. Two independent counts of the
                          same day are worth the space exactly when they differ;
                          printing them side by side every row would make the
                          rows that matter harder to find. */}
                      {s.fiscal && s.fiscal.saleTotal > 0 && (
                        <FiscalGap shift={s} money={money} />
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
              <tfoot>
                <tr className="border-t-2 border-line font-semibold">
                  <td className="py-2 pr-3" colSpan={2}>
                    {c.total}
                  </td>
                  <td className="py-2 pr-3 text-right tabular-nums">
                    {money(data?.totals.expected ?? 0)}
                  </td>
                  <td className="py-2 pr-3 text-right tabular-nums">
                    {money(data?.totals.counted ?? 0)}
                  </td>
                  <td
                    className={`py-2 pr-3 text-right tabular-nums ${
                      (data?.totals.variance ?? 0) === 0 ? "" : "text-danger"
                    }`}
                  >
                    {(data?.totals.variance ?? 0) > 0 ? "+" : ""}
                    {money(data?.totals.variance ?? 0)}
                  </td>
                  <td />
                </tr>
              </tfoot>
            </table>
          </div>
        </>
      )}
    </div>
  );
}

/** The shifts that did not balance, summarised before the table.
 *
 *  ⚠️ Shortfalls and surpluses are counted **separately and never netted**. They
 *  are different events: money missing is a question about a person, money over
 *  is usually a miscount or an unrecorded payout. A single "difference: 0"
 *  would report a calm month that contained neither. */
function Offenders({
  shifts,
  money,
}: {
  shifts: CashShift[];
  money: (n: number) => string;
}) {
  const t = useAdminT();
  const c = t.reports.cash;
  const short = shifts.filter((s) => s.variance < 0);
  const over = shifts.filter((s) => s.variance > 0);
  if (short.length === 0 && over.length === 0) return null;

  const sum = (rows: CashShift[]) =>
    rows.reduce((n, s) => n + Math.abs(s.variance), 0);

  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {short.length > 0 && (
        <div className="rounded-2xl border border-danger/40 bg-danger/5 p-3">
          <div className="text-sm font-medium text-danger">
            {c.shortCount(short.length)}
          </div>
          <div className="mt-1 font-display text-xl font-bold">
            − {money(sum(short))}
          </div>
        </div>
      )}
      {over.length > 0 && (
        <div className="rounded-2xl border border-line bg-ink/5 p-3">
          <div className="text-sm font-medium">{c.overCount(over.length)}</div>
          <div className="mt-1 font-display text-xl font-bold">
            + {money(sum(over))}
          </div>
        </div>
      )}
    </div>
  );
}

/** The gap between our record of cash sales and the register's.
 *
 *  ⚠️ **Counter cash against the register's cash sales — never the whole
 *  drawer.** `counted` is float plus courier handovers plus manual movements
 *  plus takings; the register knows only about sales it filed. Subtracting one
 *  from the other produces a large number every single time and would read as a
 *  discrepancy on every row, which is how a real one stops being visible.
 *
 *  When these two disagree, one of the two records of the same sales is wrong —
 *  which is exactly the question worth raising and cannot be raised at all
 *  without both numbers. */
function FiscalGap({
  shift,
  money,
}: {
  shift: CashShift;
  money: (n: number) => string;
}) {
  const t = useAdminT();
  const gap = shift.counterCash - (shift.fiscal?.saleCash ?? 0);
  if (gap === 0) return null;
  return (
    <div className="mt-1 text-ink-muted">
      {t.reports.cash.fiscalGap}: {gap > 0 ? "+" : ""}
      {money(gap)}
    </div>
  );
}

/** `DD.MM HH:MM` — the period is already named above the table, so the year is
 *  noise. Hand-formatted for the reason every other date in this app is: a
 *  locale-driven one prints "1:16 PM" on an English phone. */
function stamp(iso: string): string {
  const d = new Date(iso);
  const p = (n: number) => String(n).padStart(2, "0");
  return `${p(d.getDate())}.${p(d.getMonth() + 1)} ${p(d.getHours())}:${p(d.getMinutes())}`;
}
