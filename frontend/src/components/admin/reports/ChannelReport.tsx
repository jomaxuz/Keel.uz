"use client";

// Which door the orders came in through.
//
// The restaurant pays for these doors separately — a bot token, printed table
// QR codes, an operator's shift — and until now the panel could not say whether
// any of them worked.
//
// ⚠️ **Two tables, never one.** "Kanal" is how the order was placed and "Turi"
// is how it was fulfilled; a guest can order pickup through the bot, so the two
// overlap. One combined list would have rows that silently double-count and a
// column that adds up to more than the business did.

import { useCallback, useEffect, useState } from "react";
import { api, downloadReport } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import type { ChannelReportResponse, ChannelRow } from "@/lib/types";

type Range = { from?: string; to?: string };

export default function ChannelReport({ range }: { range: Range }) {
  const t = useAdminT();
  const [data, setData] = useState<ChannelReportResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const params = useCallback(() => ({ ...range }), [range]);

  useEffect(() => {
    setLoading(true);
    api
      .channelReport(params())
      .then(setData)
      .catch(() => setError(t.common.loadFailed))
      .finally(() => setLoading(false));
  }, [params, t]);

  async function download() {
    setBusy(true);
    setError("");
    try {
      await downloadReport("/admin/reports/channels", params());
    } catch {
      setError(t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  const c = t.reports.channels;
  const empty = !data?.channels.length && !data?.types.length;

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-lg font-semibold text-ink">{c.title}</h2>
        <button
          type="button"
          onClick={download}
          disabled={busy || empty}
          className="btn btn-primary disabled:opacity-60"
        >
          {busy ? t.common.loading : t.reports.excel}
        </button>
      </div>

      {error && <p className="text-sm text-brand">{error}</p>}
      {data?.note && <p className="text-sm text-ink-soft">{data.note}</p>}

      {loading ? (
        <p className="py-10 text-center text-ink-muted/70">{t.common.loading}</p>
      ) : empty ? (
        <p className="py-10 text-center text-ink-muted/70">{c.empty}</p>
      ) : (
        <div className="space-y-6">
          <Cut title={c.channelCut} rows={data!.channels} />
          <Cut title={c.typeCut} rows={data!.types} />
        </div>
      )}
    </div>
  );
}

function Cut({ title, rows }: { title: string; rows: ChannelRow[] }) {
  const t = useAdminT();
  const c = t.reports.channels;

  return (
    <div className="space-y-2">
      <h3 className="text-sm font-semibold uppercase tracking-wider text-ink-muted">
        {title}
      </h3>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[760px] text-sm">
          <thead className="text-left text-xs uppercase tracking-wider text-ink-muted">
            <tr>
              <th className="px-3 py-2">{c.name}</th>
              <th className="px-3 py-2 text-right">{c.orders}</th>
              <th className="px-3 py-2 text-right">{c.share}</th>
              <th className="px-3 py-2 text-right">{c.revenue}</th>
              <th className="px-3 py-2 text-right">{c.avgCheck}</th>
              <th className="px-3 py-2 text-right">{c.customers}</th>
              <th className="px-3 py-2 text-right">{c.newCustomers}</th>
              <th className="px-3 py-2 text-right">{c.cancelled}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line">
            {rows.map((r) => (
              <tr key={r.key} className="hover:bg-raised/60">
                <td className="px-3 py-2 font-medium text-ink">
                  {c.labels[r.key as keyof typeof c.labels] ?? r.key}
                </td>
                <td className="px-3 py-2 text-right tabular-nums">{r.orders}</td>
                <td className="px-3 py-2">
                  {/* The share as a bar as well as a number: this table is read
                      to find the one row that dominates, and a proportion is
                      seen faster than it is compared. */}
                  <div className="flex items-center justify-end gap-2">
                    <div className="h-1.5 w-16 overflow-hidden rounded-full bg-ink/10">
                      <div
                        className="h-full rounded-full bg-brand"
                        style={{ width: `${Math.min(100, r.share)}%` }}
                      />
                    </div>
                    <span className="tabular-nums text-ink-soft">{r.share.toFixed(1)}%</span>
                  </div>
                </td>
                <td className="px-3 py-2 text-right tabular-nums font-medium">
                  {formatPrice(r.revenue)}
                </td>
                <td className="px-3 py-2 text-right tabular-nums">
                  {formatPrice(r.avgCheck)}
                </td>
                <td className="px-3 py-2 text-right tabular-nums">{r.customers}</td>
                <td className="px-3 py-2 text-right tabular-nums text-emerald-700 dark:text-emerald-300">
                  {r.newCustomers || ""}
                </td>
                <td className="px-3 py-2 text-right tabular-nums text-ink-muted">
                  {r.cancelled || ""}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
