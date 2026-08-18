"use client";

import QrCode from "@/components/admin/QrCode";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice } from "@/lib/format";
import type { FiscalReceipt } from "@/lib/types";

/**
 * What happened to the tax receipt, on the screen the guest is standing at.
 *
 * ⚠️ **The change comes first, above everything about the filing.** The money
 * is already in the drawer and somebody is waiting for their notes back; a
 * screen that leads with a fiscal sign makes the cashier read past the only
 * number the next ten seconds depend on. The receipt matters to the restaurant,
 * the change matters to the person in front of them.
 *
 * ⚠️ **A failed filing does not look like a failed payment**, and the wording
 * carries most of that weight. The sale is complete and closed either way; what
 * is missing is a registration that can still be made. A cashier who reads this
 * as "the payment did not go through" will take the money twice, which is the
 * one mistake here that reaches the guest.
 */
export default function FiscalPanel({
  fiscal,
  busy,
  change,
  currency,
  onRetry,
  onPrint,
  onDone,
}: {
  fiscal: FiscalReceipt;
  busy: boolean;
  /** Cash to hand back, or 0. */
  change: number;
  currency: string;
  onRetry: () => void;
  /** Print the guest's copy again — or for the first time, where the branch
   *  has no printer and the browser is doing it. */
  onPrint: () => void;
  onDone: () => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();

  return (
    <div>
      <h2 className="font-display text-xl font-bold">{t.till.paidTitle}</h2>

      {change > 0 && (
        <div className="till-sunken mt-3 rounded-[12px] border border-line px-4 py-3">
          <div className="text-sm text-ink-muted">{t.till.change}</div>
          <div className="font-display text-2xl font-bold">
            {formatPrice(change, currency, lang)}
          </div>
        </div>
      )}

      {fiscal.status === "filed" && (
        <div className="mt-4 text-center">
          {fiscal.qrText ? (
            <>
              {/* The guest's own copy. Shown large enough to scan off the
                  screen, because that is the whole point of it — the paper
                  roll is on the register in the corner, not here. */}
              <QrCode value={fiscal.qrText} size={168} className="mx-auto" />
              <p className="mt-2 text-xs text-ink-muted">{t.till.fiscalScan}</p>
            </>
          ) : (
            <p className="text-sm text-ink-soft">{t.till.fiscalFiled}</p>
          )}
          {fiscal.fiscalSign && (
            <p className="mt-2 font-mono text-xs text-ink-muted">
              {t.till.fiscalSign}: {fiscal.fiscalSign}
            </p>
          )}
        </div>
      )}

      {fiscal.status === "failed" && (
        <div className="mt-4 rounded-2xl border border-line bg-ink/5 p-3">
          <p className="text-sm font-medium">{t.till.fiscalFailed}</p>
          {/* ⚠️ The register's own words, verbatim. They name the actual
              condition ("the shift is not open") and the cashier can act on
              most of them in seconds; a translated summary would turn a fixable
              instruction into a category. Never shown to a guest. */}
          {fiscal.error && (
            <p className="mt-1 text-xs text-ink-soft">{fiscal.error}</p>
          )}
          <p className="mt-2 text-xs text-ink-muted">
            {t.till.fiscalPaidAnyway}
          </p>
        </div>
      )}

      {fiscal.status === "pending" && (
        <p className="mt-4 text-sm text-ink-soft">{t.till.fiscalPending}</p>
      )}

      {/* ⚠️ **A copy, not the copy.** A paid sale prints itself the moment the
          register answers — this is for the branch with no printer, where the
          browser is doing the printing, and for the guest who asks for another
          one at the door. */}
      <button
        className="till-btn mt-4 w-full"
        onClick={onPrint}
        disabled={busy}
      >
        {t.till.printCustomer}
      </button>

      <div className="mt-2 flex gap-2">
        {fiscal.status !== "filed" && (
          <button className="till-btn flex-1" onClick={onRetry} disabled={busy}>
            {busy ? t.till.fiscalSending : t.till.fiscalRetry}
          </button>
        )}
        {/* ⚠️ Always available, even with the filing unfinished. A dialog that
            traps a cashier until a broken PC answers stops the queue over
            something they cannot fix from here — and the sale is recorded, so
            the filing can be retried from the check afterwards. */}
        <button className="till-btn-primary flex-1" onClick={onDone}>
          {t.till.done}
        </button>
      </div>
    </div>
  );
}
