"use client";

// What one guest still owes, and the button that closes it.
//
// ⚠️ **The card is where a debt has to be visible**, not only in a list of
// debts: the person asking "does he owe us anything?" is already looking at
// him — on the phone, or across the counter — and a figure that lives on
// another screen is a figure nobody checks in time.
//
// ⚠️ **Settling asks how the money arrived.** A debt paid in cash goes into the
// drawer somebody counts tonight; one paid by card does not, and a till that
// cannot tell them apart hands the cashier a shortage at closing.

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";

import { api, ApiError } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { formatDateTime } from "@/lib/orderFlow";
import { useAdminT } from "@/lib/i18n/admin";
import type { DebtRow } from "@/lib/types";

/** How a settled debt can arrive. Deliberately not "debt": paying a debt with
 *  a debt is a repayment that changes nothing. */
const METHODS = ["cash", "card", "transfer"] as const;

export default function CustomerDebts({
  userId,
  onSettled,
}: {
  userId: string;
  /** So the card's own total can be refreshed without reloading the page. */
  onSettled?: () => void;
}) {
  const t = useAdminT();
  const [rows, setRows] = useState<DebtRow[]>([]);
  const [total, setTotal] = useState(0);
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api
      .adminDebts(userId)
      .then((d) => {
        setRows(d.debts);
        setTotal(d.total);
      })
      .catch(() => setRows([]));
  }, [userId]);

  useEffect(load, [load]);

  async function settle(row: DebtRow, method: string) {
    setBusy(row.orderId);
    setError("");
    try {
      await api.adminPayDebt(row.orderId, method);
      load();
      onSettled?.();
    } catch (e) {
      // ⚠️ The refusal is worth showing: it means somebody else already took
      // this money, and taking it twice is the failure this guard exists for.
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setBusy("");
    }
  }

  // Nothing owed is not an empty state worth a box of its own: it is the
  // ordinary case, and a permanent "no debts" panel trains the eye past it.
  if (rows.length === 0) return null;

  return (
    <section className="mt-6 rounded-3xl border border-line bg-surface p-5 shadow-card">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 className="font-display text-lg font-bold">{t.debts.title}</h2>
        <span className="font-display text-lg font-bold text-danger tabular-nums">
          {formatPrice(total)}
        </span>
      </div>

      {error && <p className="mt-2 text-sm text-danger">{error}</p>}

      <div className="mt-3 space-y-2 text-sm">
        {rows.map((row) => (
          <div
            key={row.orderId}
            className="rounded-2xl bg-ink/[0.03] px-4 py-3"
          >
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div>
                <Link
                  href={`/admin/checks?number=${row.number}`}
                  className="font-medium hover:underline"
                >
                  {row.number}
                </Link>
                <span className="ml-2 text-ink-muted">
                  {formatDateTime(row.at)}
                </span>
                {row.table && (
                  <span className="ml-2 text-ink-muted">
                    {t.debts.table(row.table)}
                  </span>
                )}
              </div>
              <span className="font-display font-bold tabular-nums">
                {formatPrice(row.total)}
              </span>
            </div>
            {/* What was said at the counter, and who agreed to it. Both are
                the record: a debt nobody signed is one nobody can ask about. */}
            {(row.note || row.by) && (
              <p className="mt-1 text-xs text-ink-muted">
                {row.note}
                {row.note && row.by ? " · " : ""}
                {row.by && t.debts.by(row.by)}
              </p>
            )}
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <span className="text-xs text-ink-muted">{t.debts.settleAs}</span>
              {METHODS.map((m) => (
                <button
                  key={m}
                  className="btn-ghost px-3 py-1 text-xs"
                  disabled={busy === row.orderId}
                  onClick={() => settle(row, m)}
                >
                  {t.till[
                    m === "cash"
                      ? "methodCash"
                      : m === "card"
                        ? "methodCard"
                        : "methodTransfer"
                  ]}
                </button>
              ))}
            </div>
          </div>
        ))}
      </div>
      {/* ⚠️ Says which day the money lands on, because it is not the day the
          meal was eaten — and a cashier counting the drawer tonight needs to
          know this figure is in it. */}
      <p className="mt-3 text-xs text-ink-muted">{t.debts.datedToday}</p>
    </section>
  );
}
