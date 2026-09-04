"use client";

// Hisob-kitob — the money screen for staff.
//
// Each row is totalled over that employee's *own* pay period, so a cook settled
// every ten days and a manager settled monthly are both answered correctly on
// one page. That is the whole reason the period lives on the employee rather
// than on the restaurant.

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { ApiError, api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice } from "@/lib/format";
import Modal from "@/components/admin/Modal";
import { ListScroll, Pager, usePaged } from "@/components/admin/PagedList";
import {
  formatDuration,
  payModeLabel,
  payPeriodLabel,
  payUnitLabel,
  shortDate,
} from "@/lib/attendance";
import type { PayrollResponse, PayrollRow } from "@/lib/types";

const inputCls =
  "mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand";

export default function AdminPayrollPage() {
  const t = useAdminT();
  const { lang } = useI18n();
  const [data, setData] = useState<PayrollResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [paying, setPaying] = useState<PayrollRow | null>(null);
  const [error, setError] = useState<string | null>(null);

  const dur = useCallback(
    (m: number) => formatDuration(m, t.staff.hoursShort, t.staff.minutesShort),
    [t],
  );

  const load = useCallback(() => {
    api
      .adminPayroll()
      .then(setData)
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, []);

  useEffect(load, [load]);

  const rows = data?.rows ?? [];
  const paged = usePaged(rows, 15);

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="font-display text-2xl font-bold">
            {t.staff.payrollTitle}
          </h1>
          <p className="mt-0.5 max-w-2xl text-sm text-ink-muted">
            {t.staff.payrollHint}
          </p>
        </div>
        <Link href="/admin/staff" className="btn-ghost px-4 py-2 text-sm">
          {t.staff.title}
        </Link>
      </div>

      <div className="grid gap-3 sm:grid-cols-3">
        <Total label={t.staff.totalEarned} value={formatPrice(data?.earned ?? 0, "UZS", lang)} />
        <Total label={t.staff.totalPaid} value={formatPrice(data?.paid ?? 0, "UZS", lang)} />
        <Total
          label={t.staff.totalDue}
          value={formatPrice(data?.due ?? 0, "UZS", lang)}
          accent
        />
      </div>

      {error && <p className="text-sm text-red-600">{error}</p>}

      <div className="rounded-2xl border border-line bg-surface shadow-card">
        {loading ? (
          <p className="p-6 text-sm text-ink-muted">{t.common.loading}</p>
        ) : rows.length === 0 ? (
          <p className="p-6 text-sm text-ink-muted">{t.staff.payrollEmpty}</p>
        ) : (
          <>
            <ListScroll>
              <table className="w-full text-sm">
                <thead className="sticky top-0 bg-surface text-left text-xs uppercase tracking-wide text-ink-muted">
                  <tr className="border-b border-line">
                    <th className="px-4 py-3">{t.staff.colStaff}</th>
                    <th className="px-4 py-3">{t.staff.currentPeriod}</th>
                    <th className="px-4 py-3">{t.staff.colWorked}</th>
                    <th className="px-4 py-3 text-right">{t.staff.colEarned}</th>
                    <th className="px-4 py-3 text-right">{t.staff.colPaid}</th>
                    <th className="px-4 py-3 text-right">{t.staff.colDue}</th>
                    <th className="px-4 py-3" />
                  </tr>
                </thead>
                <tbody>
                  {paged.pageItems.map((row) => (
                    <tr key={row.staffId} className="border-b border-line last:border-0">
                      <td className="px-4 py-3">
                        <Link
                          href={`/admin/staff/${row.staffId}`}
                          className="font-medium hover:text-brand"
                        >
                          {row.name}
                        </Link>
                        {!row.isActive && (
                          <span className="ml-1.5 text-xs text-ink-muted">
                            {t.staff.disabled}
                          </span>
                        )}
                        <p className="text-xs text-ink-muted">
                          {row.position}
                          {row.branchName && ` · ${row.branchName}`}
                        </p>
                      </td>

                      <td className="px-4 py-3 text-xs text-ink-muted">
                        {shortDate(row.from)} — {shortDate(row.to)}
                        <p>{payPeriodLabel(row.payPeriod, t.staff)}</p>
                      </td>

                      <td className="px-4 py-3">
                        <span className="tabular-nums">{dur(row.worked)}</span>
                        <p className="text-xs text-ink-muted">
                          {t.staff.daysWorked}: {row.days} ·{" "}
                          {payModeLabel(row.payMode, t.staff)}{" "}
                          {formatPrice(row.rate, "UZS", lang)}{" "}
                          {payUnitLabel(row.payMode, t.staff)}
                        </p>
                      </td>

                      <td className="px-4 py-3 text-right tabular-nums">
                        {formatPrice(row.earned, "UZS", lang)}
                      </td>
                      <td className="px-4 py-3 text-right tabular-nums text-ink-muted">
                        {formatPrice(row.paid, "UZS", lang)}
                      </td>
                      <td
                        className={`px-4 py-3 text-right font-semibold tabular-nums ${
                          row.due > 0 ? "text-brand" : "text-emerald-600"
                        }`}
                      >
                        {row.due !== 0
                          ? formatPrice(Math.abs(row.due), "UZS", lang)
                          : row.earned > 0
                            ? t.staff.settled
                            : "—"}
                        {row.due < 0 && (
                          <p className="text-[11px] font-normal text-amber-600">
                            {t.staff.overpaid}
                          </p>
                        )}
                      </td>

                      <td className="px-4 py-3 text-right">
                        <button
                          type="button"
                          onClick={() => setPaying(row)}
                          className="text-xs text-brand hover:underline"
                        >
                          {t.staff.payButton}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </ListScroll>
            <Pager {...paged} onPage={paged.setPage} />
          </>
        )}
      </div>

      {paying && (
        <PayModal
          row={paying}
          onClose={() => setPaying(null)}
          onDone={() => {
            setPaying(null);
            load();
          }}
          onError={setError}
        />
      )}
    </div>
  );
}

function Total({
  label,
  value,
  accent,
}: {
  label: string;
  value: string;
  accent?: boolean;
}) {
  return (
    <div className="rounded-2xl border border-line bg-surface p-4 shadow-card">
      <p className="text-xs uppercase tracking-wider text-ink-muted">{label}</p>
      <p
        className={`mt-1 font-display text-2xl font-bold tabular-nums ${
          accent ? "text-brand" : ""
        }`}
      >
        {value}
      </p>
    </div>
  );
}

function PayModal({
  row,
  onClose,
  onDone,
  onError,
}: {
  row: PayrollRow;
  onClose: () => void;
  onDone: () => void;
  onError: (msg: string) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [amount, setAmount] = useState(String(Math.max(0, row.due)));
  const [note, setNote] = useState("");
  const [fromSafe, setFromSafe] = useState(false);
  const [busy, setBusy] = useState(false);

  async function submit() {
    const value = Number(amount) || 0;
    if (value <= 0) return;
    setBusy(true);
    try {
      await api.payStaff(row.staffId, {
        amount: value,
        from: row.from,
        to: row.to,
        note,
        fromSafe,
      });
      onDone();
    } catch (e) {
      onError(e instanceof ApiError ? e.message : t.common.saveFailed);
      onClose();
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal onClose={onClose}>
      <h2 className="font-display text-lg font-bold">
        {t.staff.payModalTitle(row.name)}
      </h2>
      <p className="mt-1 text-sm text-ink-muted">
        {t.staff.periodRange(shortDate(row.from), shortDate(row.to))}
      </p>

      <label className="mt-4 block text-sm">
        <span className="font-medium">{t.staff.payAmount}</span>
        <input
          className={inputCls}
          inputMode="numeric"
          value={amount}
          onChange={(e) => setAmount(e.target.value.replace(/\D/g, ""))}
        />
      </label>
      <label className="mt-3 block text-sm">
        <span className="font-medium">{t.staff.payNote}</span>
        <input
          className={inputCls}
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
      </label>

      {/* ⚠️ Asked rather than assumed. A wage paid in notes empties the office
          box; the same wage transferred to a card does not — and a safe balance
          that guessed would be a confident figure about money still sitting
          there. */}
      <label className="mt-3 flex items-center gap-2 text-sm text-ink-muted">
        <input
          type="checkbox"
          checked={fromSafe}
          onChange={(e) => setFromSafe(e.target.checked)}
        />
        {t.staff.payFromSafe}
      </label>
      <p className="mt-3 text-xs text-ink-muted">{t.staff.payHint}</p>

      <div className="mt-5 flex justify-end gap-2">
        <button type="button" onClick={onClose} className="btn-ghost px-4 py-2 text-sm">
          {t.common.cancel}
        </button>
        <button
          type="button"
          disabled={busy || !Number(amount)}
          onClick={submit}
          className="btn-primary px-4 py-2 text-sm disabled:opacity-50"
        >
          {busy ? t.common.saving : t.staff.payConfirm}
        </button>
      </div>
    </Modal>
  );
}
