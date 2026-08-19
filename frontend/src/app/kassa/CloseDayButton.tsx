"use client";

import { useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice } from "@/lib/format";
import { runFiscalJob } from "@/lib/fiscal";
import type { FiscalDay } from "@/lib/types";

/**
 * Ending the register's tax day — the Z-report.
 *
 * ⚠️ **Not the same thing as closing the cash shift**, and the screen says so.
 * The drawer count is ours and can happen twice a day at a staff handover; this
 * totals everything the register filed with the state since the day was opened
 * and cannot be undone. A restaurant that treats them as one button files a
 * Z-report at four in the afternoon.
 *
 * ⚠️ **Behind a confirmation**, which is rare in this app and earned here: it is
 * irreversible, it is the last thing done at night, and the button sits on a
 * screen used at speed with wet hands.
 */
export default function CloseDayButton({
  currency,
  onError,
}: {
  currency: string;
  onError: (msg: string) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [asking, setAsking] = useState(false);
  const [busy, setBusy] = useState(false);
  const [day, setDay] = useState<FiscalDay | null>(null);

  async function closeDay() {
    setBusy(true);
    try {
      const res = await api.tillCloseFiscalDay();
      if (res.queued) {
        // A relay will do it. ⚠️ We must not also make the call — two Z-reports
        // for one day is a tax document filed twice.
        setAsking(false);
        onError(t.till.closeDayQueued);
        return;
      }
      if (!res.job) {
        setAsking(false);
        return;
      }
      const reply = await runFiscalJob(res.job);
      setDay(await api.tillCloseFiscalDayResult(reply));
    } catch (err) {
      // ⚠️ The refusal that matters is 409: sales are still unfiled. Shown as
      // the register's own sentence rather than a generic failure, because it
      // names something the cashier can act on in the panel just above.
      onError(err instanceof ApiError ? err.message : t.till.retry);
      setAsking(false);
    } finally {
      setBusy(false);
    }
  }

  if (day) {
    return (
      <div className="rounded-2xl border border-line bg-surface p-3 text-sm">
        <p className="font-medium">
          {day.error ? t.till.closeDayFailed : t.till.closeDayDone}
        </p>
        {day.error ? (
          <p className="mt-1 text-xs text-ink-soft">{day.error}</p>
        ) : (
          <dl className="mt-2 space-y-1 text-xs text-ink-soft">
            {day.number && (
              <div className="flex justify-between">
                <dt>{t.till.zNumber}</dt>
                <dd>{day.number}</dd>
              </div>
            )}
            <div className="flex justify-between">
              <dt>{t.till.methodCash}</dt>
              <dd>{formatPrice(day.saleCash, currency, lang)}</dd>
            </div>
            <div className="flex justify-between">
              <dt>{t.till.methodCard}</dt>
              <dd>{formatPrice(day.saleCard, currency, lang)}</dd>
            </div>
            {/* ⚠️ Refunds on their own line, never netted into the total: a day
                of heavy refunds that happens to balance is a different story
                from a quiet one, and it is the story worth seeing. */}
            {day.refundTotal > 0 && (
              <div className="flex justify-between">
                <dt>{t.till.refunds}</dt>
                <dd>{formatPrice(day.refundTotal, currency, lang)}</dd>
              </div>
            )}
            <div className="flex justify-between font-medium text-ink">
              <dt>{t.till.total}</dt>
              <dd>{formatPrice(day.saleTotal, currency, lang)}</dd>
            </div>
          </dl>
        )}
        <button className="till-btn mt-3 w-full" onClick={() => setDay(null)}>
          {t.till.done}
        </button>
      </div>
    );
  }

  if (!asking) {
    return (
      <button className="till-btn w-full" onClick={() => setAsking(true)}>
        {t.till.closeDay}
      </button>
    );
  }

  return (
    <div className="rounded-2xl border border-line bg-surface p-3">
      <p className="text-sm font-medium">{t.till.closeDayConfirm}</p>
      <p className="mt-1 text-xs text-ink-soft">{t.till.closeDayHint}</p>
      <div className="mt-3 flex gap-2">
        <button
          className="till-btn flex-1"
          disabled={busy}
          onClick={() => setAsking(false)}
        >
          {t.till.back}
        </button>
        <button
          className="till-btn-primary flex-1"
          disabled={busy}
          onClick={closeDay}
        >
          {busy ? t.till.fiscalSending : t.till.closeDay}
        </button>
      </div>
    </div>
  );
}
