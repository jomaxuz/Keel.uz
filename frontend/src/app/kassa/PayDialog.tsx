"use client";

import { useEffect, useRef, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice } from "@/lib/format";
import { runFiscalJob } from "@/lib/fiscal";
import { printReceipt } from "@/lib/print";
import { isNetworkError, newClientId, queueSale } from "@/lib/offline/sales";
import { isLocal, payLocal, type LocalCheck } from "@/lib/offline/checks";
import FiscalPanel from "./FiscalPanel";
import OverrideDialog from "@/components/till/OverrideDialog";
import QrCode from "@/components/admin/QrCode";
import type { Check, FiscalReceipt, TillPaymentMethod } from "@/lib/types";
import { TILL_ONLINE } from "@/lib/types";

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
  initialMethod = "cash",
  onCancel,
  onPaid,
  onError,
  onOffline,
  onSeen,
}: {
  check: Check;
  currency: string;
  /** What the panel already asked. ⚠️ The guest says "karta" while the check is
   *  still being read back, so the answer arrives before the dialog does — and
   *  a dialog that opens on cash every time asks it twice. */
  initialMethod?: TillPaymentMethod;
  onCancel: () => void;
  onPaid: () => void;
  onError: (msg: string) => void;
  /** Said once, in the ordinary colour: the sale is fine, we are not. */
  onOffline: (msg: string) => void;
  /** Whether the server answered — the till's own signal for the banner. */
  onSeen: (ok: boolean) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [method, setMethod] = useState<TillPaymentMethod>(initialMethod);
  // Who owes it, and what was said at the counter. ⚠️ The phone is how a guest
  // is found, because it is the one thing a cashier can ask for and a guest
  // will answer — a name is not unique and nobody knows their customer id.
  const [debtPhone, setDebtPhone] = useState("");
  const [debtUser, setDebtUser] = useState<{ id: string; name: string } | null>(
    null,
  );
  const [debtNote, setDebtNote] = useState("");
  const [debtSearching, setDebtSearching] = useState(false);
  // Why the search came back empty, when it was our fault rather than the
  // guest simply not having an account.
  const [debtError, setDebtError] = useState("");
  const [discount, setDiscount] = useState("");
  const [percent, setPercent] = useState("");
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
  // Which rails this restaurant has actually signed up for. ⚠️ Starts as the
  // three that need no configuring: a till whose network is down still has to
  // be able to take cash, and an empty list would leave the cashier with no
  // buttons at all.
  const [allowed, setAllowed] = useState<TillPaymentMethod[]>([
    "cash",
    "card",
    "transfer",
    "debt",
  ]);
  // The QR the guest is looking at, once one has been asked for.
  const [payLink, setPayLink] = useState<{ url: string; number: string } | null>(
    null,
  );
  const [waiting, setWaiting] = useState(false);

  useEffect(() => {
    // ⚠️ A local check has no server-side order, so no provider can be asked to
    // invoice it — the buttons stay as they are.
    if (isLocal(check)) return;
    let live = true;
    api
      .tillPaymentMethods()
      .then((d) => {
        if (live && d.methods.length > 0) setAllowed(d.methods);
      })
      .catch(() => {});
    return () => {
      live = false;
    };
  }, [check]);

  const off = Math.max(0, Math.min(Number(discount) || 0, check.subtotal));
  const due = Math.max(0, check.subtotal - off);
  const change = Math.max(0, (Number(taken) || 0) - due);
  const needsReason = off > 0 && !reason.trim();

  const LABELS: Record<TillPaymentMethod, string> = {
    cash: t.till.methodCash,
    // "Karta" is the terminal on the counter, and it stays outside the provider
    // rails on purpose: it has its own receipt and its own settlement, and
    // money that never passes through this system cannot be confirmed by it.
    card: t.till.methodCard,
    transfer: t.till.methodTransfer,
    payme: "Payme",
    click: "Click",
    uzum: "Uzum",
    // ⚠️ Last, and it is not a way of paying: it is the record that replaces
    // the notebook by the till. A check closed this way leaves as delivered
    // and unpaid, owed by a named guest.
    debt: t.till.methodDebt,
  };
  const methods = allowed.map((id) => ({ id, label: LABELS[id] }));
  const online = TILL_ONLINE.includes(method);

  /** Print the guest's copy from here.
   *
   *  ⚠️ Through the same endpoint everything else prints through, so a branch
   *  with a printer gets paper and one without gets the browser's dialog — the
   *  screen does not need to know which it is. */
  async function printCopy() {
    try {
      const res = await api.tillPrint(check.id, "customer");
      if (res.queued === 0) printReceipt(res.lines, res.widthMM, res.logoUrl);
    } catch (e) {
      onError(e instanceof ApiError ? e.message : t.till.retry);
    }
  }

  /** Ask the provider for a link and put it on screen as a QR.
   *
   *  ⚠️ **Nothing about the sale is settled here.** The check stays open and
   *  the money stays unbooked until the provider tells the server it arrived —
   *  see tillpay.go. What this does is give the guest something to scan. */
  async function startOnline() {
    setBusy(true);
    try {
      const res = await api.tillStartPayment(check.id, method);
      setPayLink({ url: res.url, number: res.number });
      setWaiting(true);
      onSeen(true);
    } catch (e) {
      onError(e instanceof ApiError ? e.message : t.till.retry);
    } finally {
      setBusy(false);
    }
  }

  // ⚠️ **Polled while the code is on screen, and only then.** The guest is
  // standing there with a phone; a socket for a wait measured in seconds would
  // be a second transport to keep alive across the outages this till is built
  // to survive. The panel answers every question this way.
  const closeRef = useRef<() => void>(() => {});
  closeRef.current = () => void submit();
  useEffect(() => {
    if (!waiting) return;
    let live = true;
    const id = window.setInterval(async () => {
      try {
        const res = await api.tillPaymentStatus(check.id);
        if (live && res.paid) {
          setWaiting(false);
          // The money is in. Closing goes through the ordinary path, so the
          // kitchen, the receipt and the drawer all behave exactly as they do
          // for cash — the only thing that differed was who confirmed it.
          closeRef.current();
        }
      } catch {
        // A poll that fails is not a payment that failed: the guest may be
        // mid-transaction and the till may be on a bad connection. Left to the
        // next tick rather than shown, which would put an error on screen in
        // front of somebody who has done nothing wrong.
      }
    }, 2000);
    return () => {
      live = false;
      window.clearInterval(id);
    };
  }, [waiting, check.id]);

  async function submit(pin = "") {
    if (needsReason) return;
    setBusy(true);
    setOverrideError("");

    // ⚠️ **A check this device owns is paid here and owed to the server.** The
    // server has never heard of it — there is nothing to close — so the sale is
    // finished locally and handed over whole when the connection returns, which
    // is what /staff/checks/sync exists for.
    if (isLocal(check)) {
      await payLocal(
        check as LocalCheck,
        method,
        off,
        off ? reason.trim() : "",
      );
      onOffline(t.till.offlineSaved);
      onPaid();
      return;
    }

    try {
      await api.tillClose(check.id, {
        paymentMethod: method,
        discount: off || undefined,
        discountReason: off ? reason.trim() : undefined,
        pin: pin || undefined,
        userId: method === "debt" ? (debtUser?.id ?? "") : undefined,
        debtNote: method === "debt" ? debtNote.trim() : undefined,
      });
      onSeen(true);
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
      // ⚠️ **A network failure here is not a refused payment.** The cash is in
      // the drawer and the guest is leaving; the only thing that went wrong is
      // that we could not tell the server. Refusing would send the cashier back
      // to a screen that still shows the table as open, with money that no
      // longer matches it — and the honest repair, taking payment again, is the
      // one mistake here that reaches the guest.
      if (isNetworkError(err)) {
        const saved = await queueSale({
          clientId: newClientId(),
          checkId: check.id,
          label: check.tableNumber || check.number,
          total: due,
          method,
          discount: off || undefined,
          discountReason: off ? reason.trim() : undefined,
          at: Date.now(),
          tries: 0,
        });
        onSeen(false);
        if (saved) {
          // The sale is done as far as this room is concerned: the drawer is
          // shut, the guest has gone, and the queue owes the server an answer.
          onOffline(t.till.offlineSaved);
          onPaid();
          return;
        }
        // ⚠️ Could not even write it down — a locked-down browser, a private
        // window, a full disk. Said plainly, because the only safe next step is
        // a person's: do not close the check until the connection is back.
        onError(t.till.offlineNoStore);
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
            onPrint={() => void printCopy()}
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
                  ? "border-[rgb(var(--till-accent))] bg-[rgb(var(--till-accent-tint))] font-semibold text-[rgb(var(--till-accent-ink))]"
                  : "border-line"
              }`}
            >
              {m.label}
            </button>
          ))}
        </div>

        {/* ---- Who owes it ----

            ⚠️ **A debt without a name is the notebook again**, and the
            notebook is what this replaces: nothing to chase, nothing on
            anybody's card, and a total that stops adding up at the end of the
            month. The server refuses without a customer, so the button does
            too — and the search is by phone because that is the one thing a
            cashier can ask for and a guest will answer. */}
        {method === "debt" && (
          <div className="mt-3 space-y-2">
            <div className="flex gap-2">
              <input
                className="till-input h-11 flex-1"
                inputMode="tel"
                placeholder={t.till.debtPhone}
                value={debtPhone}
                onChange={(e) => {
                  setDebtPhone(e.target.value);
                  setDebtUser(null);
                }}
              />
              <button
                className="till-btn"
                disabled={debtSearching || debtPhone.trim().length < 4}
                onClick={async () => {
                  setDebtSearching(true);
                  setDebtError("");
                  try {
                    // ⚠️ The till's own lookup, not the panel's. See
                    // api.tillCustomer: the admin one needs a token a monoblock
                    // does not have, so this failed silently on the desktop app
                    // and worked in a browser only because somebody had signed
                    // into the panel on that machine.
                    const res = await api.tillCustomer(debtPhone.trim());
                    setDebtUser(res.user);
                  } catch (e) {
                    // ⚠️ **Said, not swallowed.** The old code caught this and
                    // cleared the name, so a refused request and a guest with
                    // no account looked identical — and the cashier's next move
                    // was to ask for the number again, which never helps.
                    setDebtUser(null);
                    setDebtError(e instanceof ApiError ? e.message : t.till.retry);
                  } finally {
                    setDebtSearching(false);
                  }
                }}
              >
                {t.till.debtFind}
              </button>
            </div>
            {debtError ? (
              <p className="text-xs text-danger">{debtError}</p>
            ) : debtUser ? (
              <p className="text-sm font-medium">{debtUser.name}</p>
            ) : (
              <p className="text-xs text-ink-muted">{t.till.debtNotFound}</p>
            )}
            <input
              className="till-input h-11"
              placeholder={t.till.debtNote}
              value={debtNote}
              onChange={(e) => setDebtNote(e.target.value)}
            />
          </div>
        )}

        {/* ⚠️ **Per cent and so'm, side by side and always in step.** A
            restaurant agrees discounts in per cent ("ten off for the staff
            table") and the till has to charge a number; typing 10% by hand on a
            216 000 check is arithmetic done at a counter with a queue, which is
            where wrong discounts come from. Either field may be typed and the
            other follows — the amount is what is sent, because it is what the
            guest actually pays and what the report has to add up. */}
        <div className="mt-4 grid grid-cols-[5.5rem_1fr] gap-2">
          <label className="block text-sm">
            <span className="text-ink-muted">{t.till.discountPercent}</span>
            <input
              className="till-input mt-1"
              inputMode="numeric"
              value={percent}
              onChange={(e) => {
                const digits = e.target.value.replace(/\D/g, "").slice(0, 3);
                setPercent(digits);
                const pc = Math.min(100, Number(digits) || 0);
                setDiscount(
                  pc ? String(Math.round((check.subtotal * pc) / 100)) : "",
                );
              }}
            />
          </label>
          <label className="block text-sm">
            <span className="text-ink-muted">{t.till.discountAmount}</span>
            <input
              className="till-input mt-1"
              inputMode="numeric"
              value={discount}
              onChange={(e) => {
                const digits = e.target.value.replace(/\D/g, "");
                setDiscount(digits);
                // ⚠️ The per cent box is cleared rather than recomputed to a
                // rounded figure: "12%" shown against 30 000 off a 216 000
                // check is a number that does not quite mean what it says, and
                // a cashier reading it back to a guest would be wrong.
                setPercent("");
              }}
            />
          </label>
        </div>
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

        {/* ---- The guest's phone ----

            ⚠️ **The code, the amount and the check number, in that order.**
            The first is what the guest points a camera at; the second is what
            they are about to confirm — a QR with no sum beside it is a request
            to approve an unknown number; and the third is the only way anybody
            finds this payment in the provider's cabinet when the automatic
            path does not finish. */}
        {payLink && (
          <div className="mt-4 flex flex-col items-center gap-2 rounded-2xl border border-line p-3">
            <QrCode value={payLink.url} size={168} />
            <p className="font-display text-xl font-bold">
              {formatPrice(due, currency, lang)}
            </p>
            <p className="text-xs text-ink-muted">
              {t.till.payOnlineNumber(payLink.number)}
            </p>
            {waiting && (
              <p className="text-sm font-medium">{t.till.payOnlineWaiting}</p>
            )}
            {/* ⚠️ Said on screen rather than assumed: a cashier who does not
                know the till is watching will start pressing things, and the
                thing they press is "pay". */}
            <p className="text-center text-xs text-ink-muted">
              {t.till.payOnlineHint}
            </p>
          </div>
        )}

        <div className="mt-5 flex gap-2">
          <button className="till-btn flex-1" onClick={onCancel}>
            {t.till.back}
          </button>
          {online && !waiting ? (
            <button
              className="till-btn-primary flex-1"
              disabled={busy || needsReason}
              onClick={() => void startOnline()}
            >
              {payLink ? t.till.payOnlineAgain : t.till.payOnlineShow}
            </button>
          ) : (
            <button
              className="till-btn-primary flex-1"
              // ⚠️ A debt with nobody attached is refused by the server, so the
              // button refuses first: a cashier who presses "pay" and gets an
              // error while the guest is standing there presses it again.
              //
              // ⚠️ And while a provider payment is outstanding the button is
              // not offered at all: the server refuses an unconfirmed one, and
              // a button whose only outcome is an error teaches the cashier
              // that the screen is broken.
              disabled={
                busy ||
                needsReason ||
                waiting ||
                (method === "debt" && !debtUser)
              }
              onClick={() => void submit()}
            >
              {t.till.confirmPay}
            </button>
          )}
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
