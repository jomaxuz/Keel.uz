"use client";

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice, formatDateTime } from "@/lib/format";
import { runFiscalJob } from "@/lib/fiscal";
import type { Check } from "@/lib/types";
import { TillPager, usePaged } from "@/components/till/Pager";

/**
 * Sales that took money and have no tax receipt.
 *
 * ⚠️ **This is what keeps the whole feature honest.** Everything else records a
 * filing at the moment it happens, which works right up until it does not — and
 * a failed filing is invisible by nature: the guest has eaten and gone, the cash
 * is in the drawer, and nothing on any screen looks wrong. A restaurant would
 * find out at an inspection.
 *
 * ⚠️ **It renders nothing when there is nothing**, rather than showing an empty
 * "all filed" panel. A control that is always on screen saying everything is
 * fine is a control people stop reading, and this one has to be noticed on the
 * one day it is not empty.
 *
 * On the till rather than only in the panel because the person who can fix most
 * of these — start the register's day, switch its PC back on — is standing next
 * to it.
 */
export default function UnfiledPanel({
  currency,
  onError,
  onCount,
}: {
  currency: string;
  onError: (msg: string) => void;
  /** How many sales are still waiting on the register, so the navigation can
   *  carry a dot: this panel lives one tap away now, and a warning nobody is
   *  looking at is not a warning. */
  onCount?: (n: number) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [checks, setChecks] = useState<Check[]>([]);
  const [busy, setBusy] = useState("");
  const [open, setOpen] = useState(false);
  const paged = usePaged(checks, 8);

  const refresh = useCallback(async () => {
    try {
      const res = await api.tillUnfiledChecks();
      setChecks(res.checks);
      onCount?.(res.checks.length);
    } catch {
      // Silent. This is a background check, and a cashier who cannot be told
      // "some sales may be unfiled" is better served by the screen they came
      // here to use than by an error about a screen they did not ask for.
    }
  }, [onCount]);

  useEffect(() => {
    void refresh();
    // Slow on purpose: the answer changes when a filing fails, which is rare,
    // and this runs on a tablet that is doing something more urgent.
    const id = setInterval(() => void refresh(), 60000);
    return () => clearInterval(id);
  }, [refresh]);

  async function retry(check: Check) {
    setBusy(check.id);
    try {
      const res = await api.tillFileReceipt(check.id);
      if (res.job) {
        const reply = await runFiscalJob(res.job, t.fiscal);
        const next = await api.tillFileReceiptResult(check.id, reply);
        // The register's day was not open — the commonest reason a batch of
        // these piled up in the first place. Open it and file again.
        if (next.openShift) {
          await runFiscalJob(next.openShift, t.fiscal);
          const second = await api.tillFileReceipt(check.id);
          if (second.job) {
            await api.tillFileReceiptResult(
              check.id,
              await runFiscalJob(second.job, t.fiscal),
            );
          }
        }
      }
      // Whether it worked is the server's answer, not ours — so the list is
      // re-read rather than edited here. A row that removed itself optimistically
      // would hide a filing that failed again.
      await refresh();
    } catch (err) {
      onError(err instanceof ApiError ? err.message : t.till.retry);
    } finally {
      setBusy("");
    }
  }

  async function retryAll() {
    // Sequential, not parallel. These all hit one cash register on one PC, and
    // a burst of simultaneous filings against a machine that is already
    // struggling is how a recoverable backlog becomes a hung register.
    for (const c of checks) {
      await retry(c);
    }
  }

  if (checks.length === 0) return null;

  return (
    <div className="rounded-2xl border border-danger/40 bg-danger/5 p-3">
      <button
        className="flex w-full items-center justify-between gap-2 text-left"
        onClick={() => setOpen(!open)}
      >
        <span className="text-sm font-medium text-danger">
          {t.till.unfiledTitle} ({checks.length})
        </span>
        <span className="text-xs text-ink-muted">{open ? "▲" : "▼"}</span>
      </button>
      <p className="mt-1 text-xs text-ink-soft">{t.till.unfiledHint}</p>

      {open && (
        <>
          <ul className="mt-3 space-y-2">
            {paged.shown.map((c) => (
              <li
                key={c.id}
                className="flex items-center justify-between gap-2 rounded-xl bg-surface px-3 py-2"
              >
                <div className="min-w-0">
                  <div className="flex items-center gap-1.5">
                    <span className="truncate text-sm font-medium">
                      {c.number} · {formatPrice(c.total, currency, lang)}
                    </span>
                    {/* ⚠️ Said out loud, because the row is otherwise
                        indistinguishable from an unfiled sale and the two need
                        opposite things looked at: one is a sale the committee
                        never saw, the other a sale it still thinks stands. */}
                    {stuck(c) === "refund" && (
                      <span className="shrink-0 rounded-md bg-danger/10 px-1.5 py-0.5 text-[11px] font-medium text-danger">
                        {t.till.unfiledRefund}
                      </span>
                    )}
                  </div>
                  <div className="truncate text-xs text-ink-muted">
                    {c.closedAt && formatDateTime(c.closedAt)}
                    {/* The register's own words, read from whichever of the two
                        filings is the stuck one. They usually name something
                        fixable in seconds, and a summary would turn an
                        instruction into a category. */}
                    {errorOf(c) ? ` — ${errorOf(c)}` : ""}
                  </div>
                </div>
                <button
                  className="till-btn shrink-0"
                  disabled={busy !== ""}
                  onClick={() => retry(c)}
                >
                  {busy === c.id ? t.till.fiscalSending : t.till.fiscalRetry}
                </button>
              </li>
            ))}
          </ul>
          <TillPager page={paged.page} pages={paged.pages} onPage={paged.setPage} />
          <button
            className="till-btn-primary mt-2.5 w-full"
            disabled={busy !== ""}
            onClick={retryAll}
          >
            {t.till.unfiledRetryAll}
          </button>
        </>
      )}
    </div>
  );
}

/**
 * Which of the check's two tax documents this row is about.
 *
 * ⚠️ **The sale comes first even when both are outstanding**, mirroring the
 * server: a reversal names the sale it undoes by that sale's fiscal sign, so
 * until the sale is filed there is nothing for the reversal to point at.
 */
function stuck(c: Check): "sale" | "refund" {
  if (c.fiscal && c.fiscal.status !== "filed") return "sale";
  return c.fiscalRefund && c.fiscalRefund.status !== "filed" ? "refund" : "sale";
}

function errorOf(c: Check): string {
  return (stuck(c) === "refund" ? c.fiscalRefund?.error : c.fiscal?.error) ?? "";
}
