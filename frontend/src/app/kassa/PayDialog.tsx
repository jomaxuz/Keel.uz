"use client";

import { useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice } from "@/lib/format";
import { runFiscalJob } from "@/lib/fiscal";
import FiscalPanel from "./FiscalPanel";
import OverrideDialog from "@/components/till/OverrideDialog";
import type { Check, FiscalReceipt, TillPaymentMethod } from "@/lib/types";

/**
 * Taking payment.
 *
 * ⚠️ **The change calculator is the reason this is a dialog and not one tap.**
 * A cashier doing arithmetic in their head with a queue behind them is where
 * short drawers come from, and the shift's variance report cannot tell an
 * honest mistake from anything else. It is shown only for cash, because it is
 * the only method where money physically comes back.
 *
 * ⚠️ **A discount needs a reason.** The same rule as a void, for the same
 * reason: an untraceable discount and an untraceable void take money out of a
 * restaurant by exactly the same route. The server refuses one without the
 * other, so this is not the only guard.
 */
export default function PayDialog({
  check,
  currency,
  onCancel,
  onPaid,
  onError,
}: {
  check: Check;
  currency: string;
  onCancel: () => void;
  onPaid: () => void;
  onError: (msg: string) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [method, setMethod] = useState<TillPaymentMethod>("cash");
  const [discount, setDiscount] = useState("");
  const [reason, setReason] = useState("");
  const [taken, setTaken] = useState("");
  const [busy, setBusy] = useState(false);
  // The filing, once the money is in. Null means we are still on the form —
  // which is also why it is not a boolean: "no receipt yet" and "a receipt that
  // failed" need to look nothing alike.
  const [fiscal, setFiscal] = useState<FiscalReceipt | null>(null);
  const [filing, setFiling] = useState(false);
  // The discount the server asked a manager to authorise. Held so the retry
  // sends the same amount and reason — retyping either is how a busy cashier
  // ends up giving a different discount than the one that was approved.
  const [override, setOverride] = useState<string | null>(null);
  const [overrideError, setOverrideError] = useState("");

  const off = Math.max(0, Math.min(Number(discount) || 0, check.subtotal));
  const due = Math.max(0, check.subtotal - off);
  const change = Math.max(0, (Number(taken) || 0) - due);
  const needsReason = off > 0 && !reason.trim();

  const methods: { id: TillPaymentMethod; label: string }[] = [
    { id: "cash", label: t.till.methodCash },
    { id: "card", label: t.till.methodCard },
    { id: "transfer", label: t.till.methodTransfer },
  ];

  async function submit(pin = "") {
    if (needsReason) return;
    setBusy(true);
    setOverrideError("");
    try {
      await api.tillClose(check.id, {
        paymentMethod: method,
        discount: off || undefined,
        discountReason: off ? reason.trim() : undefined,
        pin: pin || undefined,
      });
      setOverride(null);
    } catch (err) {
      // ⚠️ A discount this person may not give is a **request for a manager**,
      // not a refusal — see OverrideDialog for why refusing is the answer that
      // destroys the attribution.
      const perm = err instanceof ApiError ? err.needsOverride : null;
      if (perm) {
        if (pin) setOverrideError(t.till.overrideWrong);
        setOverride(err instanceof ApiError ? err.permissionName : "");
        setBusy(false);
        return;
      }
      onError(err instanceof ApiError ? err.message : t.till.retry);
      setBusy(false);
      return;
    }
    // ⚠️ **The money is taken from here on, and nothing below may undo that.**
    // The filing is a second, separate step precisely because it can fail on
    // its own: a register that does not answer must not turn a completed sale
    // into an error, and the guest must not be asked to pay twice because a PC
    // in the corner was rebooting.
    await file();
  }

  /** File the receipt, in the two halves the transport forces.
   *
   *  The server builds the document and hands it over as an opaque job; this
   *  screen is on the restaurant's network and is the only thing that can
   *  deliver it. See lib/fiscal.ts. */
  async function file(afterShiftOpen = false) {
    setFiling(true);
    try {
      const res = await api.tillFileReceipt(check.id);
      if (res.skip) {
        // No register connected. The ordinary state for a restaurant that has
        // not signed a contract yet, and not a failure to report — the sale
        // stands, we simply do not claim it was registered.
        onPaid();
        return;
      }
      if (res.filed && res.fiscal) {
        setFiscal(res.fiscal);
        return;
      }
      if (res.queued) {
        // A relay on the register's PC has it. ⚠️ We must **not** also make the
        // call: two callers filing one sale is one receipt registered twice,
        // and a duplicate fiscal document is undone by paperwork rather than by
        // us. So this waits for the result the relay posts back.
        setFiscal({ status: "pending", provider: res.fiscal?.provider ?? "" });
        await waitForFiling();
        return;
      }
      if (!res.job) {
        onPaid();
        return;
      }
      const reply = await runFiscalJob(res.job);
      const next = await api.tillFileReceiptResult(check.id, reply);

      // ⚠️ The morning case: the register refused because its day has not been
      // started. Handled here rather than shown as an error — it happens on the
      // first sale of every single day and the fix is entirely mechanical, so
      // making the cashier learn it would be making them learn our plumbing
      // while a guest waits.
      //
      // The sale is still pending server-side, so filing again after opening
      // the day is safe and cannot produce a second receipt.
      // ⚠️ Once only. If opening the day did not take, the register is refusing
      // for a reason we have not understood, and retrying forever would spin
      // silently while a guest stands at the counter — the failure shown below
      // at least names what the register said.
      if (next.openShift && !afterShiftOpen) {
        await runFiscalJob(next.openShift);
        await file(true);
        return;
      }
      setFiscal(next.fiscal ?? null);
    } catch (err) {
      // ⚠️ Shown here rather than thrown out to the page, because the dialog
      // must stay open: it is the only place that offers the retry, and closing
      // it would leave a paid, unfiled sale with no way back to it except
      // knowing to look.
      setFiscal({
        status: "failed",
        provider: "",
        error: err instanceof ApiError ? err.message : t.till.retry,
      });
    } finally {
      setFiling(false);
    }
  }

  /** Wait for the relay to file, then show whatever it got.
   *
   *  ⚠️ **Bounded, and giving up shows the guest's screen anyway.** The relay
   *  runs on a PC in the corner that can be mid-Windows-update; a cashier held
   *  at a spinner over that stops the queue for something nobody at the counter
   *  can fix. Timing out here leaves the sale pending — which is exactly what
   *  it is — and the relay will still file it the moment it comes back. */
  async function waitForFiling() {
    const deadline = Date.now() + 20000;
    while (Date.now() < deadline) {
      await new Promise((r) => setTimeout(r, 1500));
      try {
        const next = await api.tillCheck(check.id);
        if (next.fiscal && next.fiscal.status !== "pending") {
          setFiscal(next.fiscal);
          return;
        }
      } catch {
        // A dropped poll is not a failed filing. Keep waiting: the answer lives
        // on the server, not in this loop.
      }
    }
  }

  // The receipt step. Replaces the form rather than sitting under it: the money
  // is already taken, so every control that could change the amount is now a
  // lie about what can still happen.
  if (override !== null) {
    return (
      <OverrideDialog
        permissionName={override}
        busy={busy}
        error={overrideError}
        onCancel={() => {
          setOverride(null);
          setOverrideError("");
        }}
        onSubmit={(pin) => void submit(pin)}
      />
    );
  }

  if (fiscal) {
    return (
      <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center">
        <div className="till-dialog w-full max-w-sm p-4">
          <FiscalPanel
            fiscal={fiscal}
            busy={filing}
            change={method === "cash" ? change : 0}
            currency={currency}
            onRetry={file}
            onDone={onPaid}
          />
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center">
      <div className="till-dialog w-full max-w-sm p-4">
        <h2 className="font-display text-xl font-bold">{t.till.payTitle}</h2>

        <div className="mt-4 grid grid-cols-3 gap-2">
          {methods.map((m) => (
            <button
              key={m.id}
              onClick={() => setMethod(m.id)}
              className={`rounded-xl border px-2 py-3 text-sm ${
                method === m.id
                  ? "border-transparent bg-[rgb(var(--till-action))] font-semibold text-white"
                  : "border-line"
              }`}
            >
              {m.label}
            </button>
          ))}
        </div>

        <label className="mt-4 block text-sm">
          <span className="text-ink-muted">{t.till.discountAmount}</span>
          <input
            className="till-input mt-1"
            inputMode="numeric"
            value={discount}
            onChange={(e) => setDiscount(e.target.value.replace(/\D/g, ""))}
          />
        </label>
        {off > 0 && (
          <label className="mt-2 block text-sm">
            <span className="text-ink-muted">{t.till.discountReason}</span>
            <input
              className="till-input mt-1"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
          </label>
        )}

        <div className="mt-4 flex justify-between font-display text-2xl font-bold">
          <span>{t.till.total}</span>
          <span>{formatPrice(due, currency, lang)}</span>
        </div>

        {method === "cash" && (
          <>
            <label className="mt-3 block text-sm">
              <span className="text-ink-muted">{t.till.cashTaken}</span>
              <input
                className="till-input mt-1"
                inputMode="numeric"
                value={taken}
                onChange={(e) => setTaken(e.target.value.replace(/\D/g, ""))}
              />
            </label>
            {/* Shown from the first digit, not on a button: the number is
                needed while the notes are still in the cashier's hand. */}
            <div className="mt-2 flex justify-between text-lg font-semibold">
              <span className="text-ink-muted">{t.till.change}</span>
              <span>{formatPrice(change, currency, lang)}</span>
            </div>
          </>
        )}

        <div className="mt-5 flex gap-2">
          <button className="till-btn flex-1" onClick={onCancel}>
            {t.till.back}
          </button>
          <button
            className="till-btn-primary flex-1"
            disabled={busy || needsReason}
            onClick={() => void submit()}
          >
            {t.till.confirmPay}
          </button>
        </div>
        {needsReason && (
          <p className="mt-2 text-center text-xs text-ink-muted">
            {t.till.reasonRequired}
          </p>
        )}
      </div>
    </div>
  );
}
