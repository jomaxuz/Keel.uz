"use client";

// Everyone on one page.
//
// The courier card and the employee calendar are the right screens for "how is
// Aziz doing". Neither can answer the question an owner asks at the end of a
// month — "how are they doing compared with each other" — which needs every row
// sorted, side by side, and downloadable.

import { useCallback, useEffect, useState } from "react";
import { api, downloadReport } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { ListScroll } from "@/components/admin/PagedList";
import type { CourierReportResponse, StaffReportResponse } from "@/lib/types";

type Range = { from?: string; to?: string };
type Which = "couriers" | "staff";

export default function TeamReport({ range }: { range: Range }) {
  const t = useAdminT();
  const [which, setWhich] = useState<Which>("couriers");
  const [couriers, setCouriers] = useState<CourierReportResponse | null>(null);
  const [staff, setStaff] = useState<StaffReportResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const params = useCallback(() => ({ ...range }), [range]);

  useEffect(() => {
    setLoading(true);
    setError("");
    const load = which === "couriers" ? api.adminCourierReport : api.adminStaffReport;
    load(params())
      .then((d) => (which === "couriers" ? setCouriers(d as CourierReportResponse) : setStaff(d as StaffReportResponse)))
      .catch(() => setError(t.common.loadFailed))
      .finally(() => setLoading(false));
  }, [params, which, t]);

  async function download() {
    setBusy(true);
    setError("");
    try {
      await downloadReport(`/admin/reports/${which}`, params());
    } catch {
      setError(t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  const tm = t.reports.team;
  const note = which === "couriers" ? couriers?.note : staff?.note;

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          {(["couriers", "staff"] as Which[]).map((w) => (
            <button
              key={w}
              type="button"
              onClick={() => setWhich(w)}
              className={
                which === w
                  ? "rounded-xl bg-ink px-3 py-2 text-xs font-semibold text-surface"
                  : "rounded-xl border border-line px-3 py-2 text-xs font-semibold text-ink-soft hover:bg-ink/5"
              }
            >
              {tm[w]}
            </button>
          ))}
        </div>
        <button
          type="button"
          onClick={download}
          disabled={busy}
          className="btn btn-primary disabled:opacity-60"
        >
          {busy ? t.common.loading : t.reports.excel}
        </button>
      </div>

      {error && <p className="text-sm text-brand">{error}</p>}
      {note && <p className="text-sm text-ink-soft">{note}</p>}

      {loading ? (
        <p className="py-10 text-center text-ink-muted/70">{t.common.loading}</p>
      ) : which === "couriers" ? (
        <Couriers data={couriers} />
      ) : (
        <Staff data={staff} />
      )}
    </div>
  );
}

function Couriers({ data }: { data: CourierReportResponse | null }) {
  const t = useAdminT();
  const tm = t.reports.team;
  const rows = data?.couriers ?? [];
  if (!rows.length) {
    return <p className="py-10 text-center text-ink-muted/70">{tm.noCouriers}</p>;
  }

  return (
    <ListScroll className="max-h-[60vh]">
      <table className="w-full min-w-[820px] text-sm">
        <thead className="sticky top-0 bg-raised text-left text-xs uppercase tracking-wider text-ink-muted">
          <tr>
            <th className="px-3 py-2">{tm.courier}</th>
            <th className="px-3 py-2 text-right">{tm.delivered}</th>
            <th className="px-3 py-2 text-right">{tm.avgMinutes}</th>
            <th className="px-3 py-2 text-right">{tm.earnings}</th>
            <th className="px-3 py-2 text-right">{tm.carried}</th>
            <th className="px-3 py-2 text-right">{tm.cash}</th>
            <th className="px-3 py-2 text-right">{tm.cashInHand}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-line">
          {rows.map((c) => (
            <tr key={c.id} className="hover:bg-raised/60">
              <td className="px-3 py-2 font-medium text-ink">{c.name}</td>
              <td className="px-3 py-2 text-right tabular-nums">{c.delivered}</td>
              <td className="px-3 py-2 text-right tabular-nums">
                {/* ⚠️ A dash, not "0 daq", when nothing was timed: a zero in an
                    average column reads as instant delivery. */}
                {c.timedOrders > 0 ? tm.minutes(c.avgMinutes) : tm.noTimings}
              </td>
              <td className="px-3 py-2 text-right tabular-nums font-medium">
                {formatPrice(c.earnings)}
              </td>
              <td className="px-3 py-2 text-right tabular-nums text-ink-muted">
                {formatPrice(c.carried)}
              </td>
              <td className="px-3 py-2 text-right tabular-nums">{formatPrice(c.cash)}</td>
              <td
                className={`px-3 py-2 text-right tabular-nums font-semibold ${
                  c.cashInHand > 0 ? "text-amber-700 dark:text-amber-300" : "text-ink-muted"
                }`}
              >
                {formatPrice(c.cashInHand)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </ListScroll>
  );
}

function Staff({ data }: { data: StaffReportResponse | null }) {
  const t = useAdminT();
  const tm = t.reports.team;
  const rows = data?.staff ?? [];
  if (!rows.length) {
    return <p className="py-10 text-center text-ink-muted/70">{tm.noStaff}</p>;
  }

  return (
    <ListScroll className="max-h-[60vh]">
      <table className="w-full min-w-[900px] text-sm">
        <thead className="sticky top-0 bg-raised text-left text-xs uppercase tracking-wider text-ink-muted">
          <tr>
            <th className="px-3 py-2">{tm.employee}</th>
            <th className="px-3 py-2">{tm.position}</th>
            <th className="px-3 py-2 text-right">{tm.days}</th>
            <th className="px-3 py-2 text-right">{tm.absent}</th>
            <th className="px-3 py-2 text-right">{tm.expected}</th>
            <th className="px-3 py-2 text-right">{tm.worked}</th>
            <th className="px-3 py-2 text-right">{tm.overtime}</th>
            <th className="px-3 py-2 text-right">{tm.pay}</th>
            <th className="px-3 py-2 text-right">{tm.paid}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-line">
          {rows.map((s) => (
            <tr key={s.id} className="hover:bg-raised/60">
              <td className="px-3 py-2 font-medium text-ink">{s.name}</td>
              <td className="px-3 py-2 text-ink-soft">{s.position}</td>
              <td className="px-3 py-2 text-right tabular-nums">{s.days}</td>
              <td className="px-3 py-2 text-right tabular-nums text-rose-700 dark:text-rose-300">
                {s.absent || ""}
              </td>
              <td className="px-3 py-2 text-right tabular-nums text-ink-muted">
                {tm.hours(s.expected)}
              </td>
              <td className="px-3 py-2 text-right tabular-nums">{tm.hours(s.worked)}</td>
              <td className="px-3 py-2 text-right tabular-nums text-ink-muted">
                {s.overtime ? tm.hours(s.overtime) : ""}
              </td>
              <td className="px-3 py-2 text-right tabular-nums font-medium">
                {formatPrice(s.pay)}
              </td>
              <td className="px-3 py-2 text-right tabular-nums">{formatPrice(s.paid)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </ListScroll>
  );
}
