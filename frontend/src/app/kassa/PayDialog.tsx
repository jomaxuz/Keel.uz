"use client";

import { useEffect, useRef, useState, type ComponentType } from "react";
import {
  LuArrowLeft,
  LuBanknote,
  LuChevronDown,
  LuCreditCard,
  LuLandmark,
  LuNotebookPen,
  LuPercent,
  LuQrCode,
  LuScanLine,
  LuStore,
} from "react-icons/lu";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useTillWords } from "@/lib/tillWords";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice } from "@/lib/format";
import { runFiscalJob } from "@/lib/fiscal";
import { printReceipt } from "@/lib/print";
import { isNetworkError, newClientId, queueSale } from "@/lib/offline/sales";
import { isLocal, payLocal, type LocalCheck } from "@/lib/offline/checks";
import { optionLabel, railsOf, useTillPay } from "@/lib/tillPayOptions";
import FiscalPanel from "./FiscalPanel";
import OverrideDialog from "@/components/till/OverrideDialog";
import QrCode from "@/components/admin/QrCode";
import type {
  Check,
  FiscalReceipt,
  TillPaymentMethod,
  TillScanResult,
} from "@/lib/types";
import { TILL_ONLINE, TILL_SCAN } from "@/lib/types";

/**
 * Taking payment.
 *
 * ⚠️ **Two halves, and the left one is the money.** What the guest owes, what
 * they handed over and what goes back sit on one side in the largest type on
 * the screen; how they pay sits on the other. A cashier's eyes go to the number
 * first and the buttons second, and a layout that mixed them made the change —
 * the one figure said out loud to the guest — the smallest thing in the dialog.
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
 *
 * ⚠️ **A tile's accessible name is its label and nothing else** — icons are
 * `aria-hidden`. The till's tests, and anybody using a screen reader, find a
 * payment method by the word on it.
 */
export default function PayDialog({
  check,
  currency,
  initialChoice = "cash",
  onCancel,
  onPaid,
  onError,
  onOffline,
  onSeen,
}: {
  check: Check;
  currency: string;
  /** The button the panel already had pressed. ⚠️ The guest says "karta" while
   *  the check is still being read back, so the answer arrives before the
   *  dialog does — and a dialog that opens on cash every time asks it twice. */
  initialChoice?: string;
  onCancel: () => void;
  onPaid: () => void;
  onError: (msg: string) => void;
  /** Said once, in the ordinary colour: the sale is fine, we are not. */
  onOffline: (msg: string) => void;
  /** Whether the server answered — the till's own signal for the banner. */
  onSeen: (ok: boolean) => void;
}) {
  const t = useAdminT();
  const w = useTillWords();
  const { lang } = useI18n();
  // Which tile is pressed: one of the owner's buttons by id, or a rail by name.
  const [choice, setChoice] = useState<string>(initialChoice);
  // ⚠️ A local check has no server-side order, so no provider can be asked to
  // invoice it — the defaults stay, and cash always works.
  const pay = useTillPay(!isLocal(check));
  const option = pay.options.find((o) => o.id === choice);
  // How the money is booked. The owner's button says which kind it is; a rail
  // is its own kind.
  const method: TillPaymentMethod = option ? option.kind : (choice as TillPaymentMethod);

  // Who owes it, and what was said at the counter. ⚠️ The phone is how a guest
  // is found, because it is the one thing a cashier can ask for and a guest
  // will answer — a name is not unique and nobody knows their customer id.
  const [debtPhone, setDebtPhone] = useState("");
  const [debtUser, setDebtUser] = useState<{
    id: string;
    name: string;
    // Whether the owner has allowed this guest to owe. ⚠️ Carried from the
    // lookup so the screen can say why *before* the note is typed and the
    // button pressed; the server refuses on close either way.
    creditAllowed?: boolean;
  } | null>(null);
  const [debtNote, setDebtNote] = useState("");
  const [debtSearching, setDebtSearching] = useState(false);
  // Why the search came back empty, when it was our fault rather than the
  // guest simply not having an account.
  const [debtError, setDebtError] = useState("");
  const [discount, setDiscount] = useState("");
  const [percent, setPercent] = useState("");
  const [reason, setReason] = useState("");
  // Folded away by default: most bills have no discount, and the fields for
  // one are the last thing a cashier with a queue should have to look past.
  const [discountOpen, setDiscountOpen] = useState(false);
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
  // The QR the guest is looking at, once one has been asked for.
  const [payLink, setPayLink] = useState<{ url: string; number: string } | null>(
    null,
  );
  const [waiting, setWaiting] = useState(false);
  // ---- Scanning the guest's code ----
  //
  // ⚠️ **The scanner is a keyboard**, exactly as it is for a marked bottle
  // (see ScanDialog): pistol scanners run in HID mode and the code arrives as
  // typed text ending in Enter. So this is a focused input and nothing more —
  // no camera permission, no device API, nothing else to be wrong.
  const [scanCode, setScanCode] = useState("");
  const [scanOpen, setScanOpen] = useState(false);
  const [scanResult, setScanResult] = useState<TillScanResult | null>(null);
  const scanField = useRef<HTMLInputElement>(null);

  // ⚠️ **A pressed button that is not on offer is moved to the first that is.**
  // The panel may have had "Humo" pressed while the owner switched it off, and
  // a dialog with no tile lit would close the check on a button the server
  // has never heard of.
  useEffect(() => {
    const ids = [...pay.options.map((o) => o.id), ...pay.methods];
    if (!ids.includes(choice)) setChoice(pay.options[0]?.id ?? "cash");
  }, [pay, choice]);

  const off = Math.max(0, Math.min(Number(discount) || 0, check.subtotal));
  const due = Math.max(0, check.subtotal - off);
  const change = Math.max(0, (Number(taken) || 0) - due);
  const needsReason = off > 0 && !reason.trim();

  const RAIL_LABELS: Partial<Record<TillPaymentMethod, string>> = {
    payme: "Payme",
    click: "Click",
    uzum: "Uzum",
    // ⚠️ **The counter rails, named as the guest's app names them.** "Click"
    // and "Click Pass" are two different buttons on this screen and a cashier
    // has to be able to tell them apart at arm's length: one puts a QR on our
    // screen, the other asks the guest to open one on theirs.
    click_pass: "Click Pass",
    uzum_fastpay: "Uzum FastPay",
    yandex_eats: "Yandex Eats",
  };
  const tiles: { id: string; label: string; Icon: ComponentType<{ className?: string; "aria-hidden"?: boolean }> }[] = [
    ...pay.options.map((o) => ({
      id: o.id,
      label: optionLabel(o, t.till),
      Icon: o.kind === "cash" ? LuBanknote : o.kind === "card" ? LuCreditCard : LuLandmark,
    })),
    ...railsOf(pay.methods).map((id) => ({
      id,
      label: RAIL_LABELS[id] ?? id,
      Icon: TILL_SCAN.includes(id) ? LuScanLine : TILL_ONLINE.includes(id) ? LuQrCode : LuStore,
    })),
    // ⚠️ Last, and it is not a way of paying: it is the record that replaces
    // the notebook by the till. A check closed this way leaves as delivered
    // and unpaid, owed by a named guest.
    ...(pay.methods.includes("debt")
      ? [{ id: "debt", label: t.till.methodDebt, Icon: LuNotebookPen }]
      : []),
  ];
  const online = TILL_ONLINE.includes(method);
  const scan = TILL_SCAN.includes(method);
  // ⚠️ **"We do not know" is its own state and it is not "failed".** A request
  // that never came back may have charged the guest, so the screen must offer
  // *asking the bank* and must not offer *scanning again* — which is how one
  // guest is charged twice.
  const unknown = scanResult?.status === "pending";

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

  // ⚠️ A scan's outcome belongs to the rail it was made on. Left standing when
  // the cashier switches to cash, "to'landi · 8600 12** **** 8331" sits under a
  // cash payment and reads as though the card already went through.
  useEffect(() => {
    setScanResult(null);
    setScanOpen(false);
    setScanCode("");
  }, [choice]);

  /** Charge the card behind the scanned code.
   *
   *  ⚠️ **Nothing about the check changes here on a refusal**, which is the
   *  whole reason this is a separate step from closing: a declined card leaves
   *  the table open, the lines intact and the cashier free to take cash — where
   *  a combined "pay and close" would have to undo a close it had already
   *  performed, in front of a guest.
   *
   *  ⚠️ On success the check closes through the **ordinary** path, so the
   *  kitchen, the receipt and the drawer behave exactly as they do for cash.
   *  The only thing that differed was who confirmed the money. */
  async function scanPay(code: string) {
    const value = code.trim();
    if (!value) return;
    setBusy(true);
    setScanResult(null);
    try {
      const res = await api.tillScanPay(check.id, method, value);
      setScanResult(res);
      onSeen(true);
      setScanCode("");
      if (res.paid) {
        setScanOpen(false);
        await submit();
        return;
      }
      // Refused. The field is cleared and refocused rather than the dialog
      // closing: the next move is almost always another scan, and a guest's
      // second code is a second later.
      scanField.current?.focus();
    } catch (e) {
      // ⚠️ **A transport failure is not a refusal**, and treating it as one is
      // the mistake this whole flow is shaped around: the request may have
      // reached the bank. Left in the unknown state so the screen offers the
      // status check and hides the scan button.
      setScanResult({
        status: "pending",
        paid: false,
        error: e instanceof ApiError ? e.message : t.till.retry,
      });
    } finally {
      setBusy(false);
    }
  }

  /** Ask the bank about an attempt whose answer never arrived. */
  async function scanCheck() {
    setBusy(true);
    try {
      const res = await api.tillScanStatus(check.id);
      setScanResult(res);
      onSeen(true);
      if (res.paid) {
        setScanOpen(false);
        await submit();
      }
    } catch (e) {
      onError(e instanceof ApiError ? e.message : t.till.retry);
    } finally {
      setBusy(false);
    }
  }

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
        option?.id,
      );
      onOffline(t.till.offlineSaved);
      onPaid();
      return;
    }

    try {
      await api.tillClose(check.id, {
        paymentMethod: method,
        // ⚠️ The button, beside its kind. The server books the money by the
        // button's kind as it stands in the settings; the kind sent here is
        // only the fallback for a button that has since been removed.
        methodId: option?.id,
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
          methodId: option?.id,
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
      const reply = await runFiscalJob(res.job, t.fiscal);
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
        await runFiscalJob(next.openShift, t.fiscal);
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

  const quick = method === "cash" ? quickCash(due) : [];

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-3 sm:items-center sm:p-4">
      <div className="till-dialog flex max-h-[calc(100dvh-1.5rem)] w-full max-w-3xl flex-col overflow-hidden p-0">
        <div className="grid min-h-0 flex-1 overflow-y-auto md:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
          {/* ---- The money ---- */}
          <section className="till-sunken flex flex-col gap-4 p-5">
            <div className="flex items-baseline justify-between gap-3">
              <h2 className="font-display text-xl font-bold">{t.till.payTitle}</h2>
              <span className="truncate text-sm font-semibold text-[rgb(var(--till-mid))]">
                #{check.number}
              </span>
            </div>

            <div className="rounded-2xl border border-line bg-surface p-4">
              <p className="till-label">{t.till.dueNow}</p>
              <p className="mt-1 font-display text-4xl font-bold leading-tight tabular-nums">
                {formatPrice(due, currency, lang)}
              </p>
              <dl className="mt-3 space-y-1.5 border-t border-line pt-3 text-sm">
                <div className="flex justify-between gap-3">
                  <dt className="text-[rgb(var(--till-mid))]">{w.tillTotalLabel}</dt>
                  <dd className="font-semibold tabular-nums">
                    {formatPrice(check.subtotal, currency, lang)}
                  </dd>
                </div>
                {off > 0 && (
                  <div className="flex justify-between gap-3">
                    <dt className="text-[rgb(var(--till-mid))]">{t.till.discountToggle}</dt>
                    <dd className="font-semibold tabular-nums text-[rgb(var(--till-accent-ink))]">
                      −{formatPrice(off, currency, lang)}
                    </dd>
                  </div>
                )}
              </dl>
            </div>

            {method === "cash" && (
              <div className="space-y-3">
                <label className="block">
                  <span className="till-label">{t.till.cashTaken}</span>
                  <input
                    className="till-input mt-1.5 h-12 text-lg font-semibold tabular-nums"
                    inputMode="numeric"
                    value={taken}
                    onChange={(e) => setTaken(e.target.value.replace(/\D/g, ""))}
                  />
                </label>
                {/* ⚠️ **The notes a guest actually hands over**, one tap each.
                    Typing 100000 on a touch keypad is six presses and one of
                    them is the wrong zero; the round-ups are what a guest pays
                    with, and "exact" is what most of them do. */}
                {quick.length > 0 && (
                  <div className="flex flex-wrap gap-2">
                    {quick.map((q) => (
                      <button
                        key={q}
                        type="button"
                        onClick={() => setTaken(String(q))}
                        className={`till-btn-quiet min-h-10 px-3 tabular-nums ${
                          Number(taken) === q
                            ? "border-[rgb(var(--till-accent))] text-[rgb(var(--till-accent-ink))]"
                            : ""
                        }`}
                      >
                        {q === due ? t.till.exactAmount : formatPrice(q, currency, lang)}
                      </button>
                    ))}
                  </div>
                )}
                {/* Shown from the first digit, not on a button: the number is
                    needed while the notes are still in the cashier's hand. */}
                <div className="flex items-baseline justify-between gap-3 rounded-2xl border border-[rgb(var(--till-ok)/0.35)] bg-[rgb(var(--till-ok)/0.08)] px-4 py-3">
                  <span className="text-sm font-semibold text-[rgb(var(--till-ok))]">
                    {t.till.change}
                  </span>
                  <span className="font-display text-2xl font-bold tabular-nums text-[rgb(var(--till-ok))]">
                    {formatPrice(change, currency, lang)}
                  </span>
                </div>
              </div>
            )}

            {/* ⚠️ **Per cent and so'm, side by side and always in step.** A
                restaurant agrees discounts in per cent ("ten off for the staff
                table") and the till has to charge a number; typing 10% by hand on
                a 216 000 check is arithmetic done at a counter with a queue,
                which is where wrong discounts come from. Either field may be
                typed and the other follows — the amount is what is sent, because
                it is what the guest actually pays and what the report has to add
                up. */}
            <div className="mt-auto">
              <button
                type="button"
                className="till-btn-ghost -ml-2 px-2"
                aria-expanded={discountOpen || off > 0}
                onClick={() => setDiscountOpen((o) => !o)}
              >
                <LuPercent className="h-4 w-4" aria-hidden />
                {t.till.discountToggle}
                <LuChevronDown
                  className={`h-4 w-4 transition-transform ${
                    discountOpen || off > 0 ? "rotate-180" : ""
                  }`}
                  aria-hidden
                />
              </button>
              {(discountOpen || off > 0) && (
                <div className="mt-2 space-y-2">
                  <div className="grid grid-cols-[5.5rem_1fr] gap-2">
                    <label className="block text-sm">
                      <span className="text-[rgb(var(--till-mid))]">{t.till.discountPercent}</span>
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
                      <span className="text-[rgb(var(--till-mid))]">{t.till.discountAmount}</span>
                      <input
                        className="till-input mt-1"
                        inputMode="numeric"
                        value={discount}
                        onChange={(e) => {
                          const digits = e.target.value.replace(/\D/g, "");
                          setDiscount(digits);
                          // ⚠️ The per cent box is cleared rather than recomputed
                          // to a rounded figure: "12%" shown against 30 000 off a
                          // 216 000 check is a number that does not quite mean
                          // what it says, and a cashier reading it back to a guest
                          // would be wrong.
                          setPercent("");
                        }}
                      />
                    </label>
                  </div>
                  {off > 0 && (
                    <label className="block text-sm">
                      <span className="text-[rgb(var(--till-mid))]">{t.till.discountReason}</span>
                      <input
                        className="till-input mt-1"
                        value={reason}
                        onChange={(e) => setReason(e.target.value)}
                      />
                    </label>
                  )}
                </div>
              )}
            </div>
          </section>

          {/* ---- How they pay ---- */}
          <section className="flex flex-col gap-4 p-5">
            <div>
              <p className="till-label">{t.till.methodsTitle}</p>
              <div className="mt-2 grid grid-cols-2 gap-2 sm:grid-cols-3">
                {tiles.map((m) => {
                  const on = choice === m.id;
                  return (
                    <button
                      key={m.id}
                      type="button"
                      aria-pressed={on}
                      onClick={() => setChoice(m.id)}
                      className={`flex min-h-[4.75rem] flex-col items-start justify-between gap-2 rounded-2xl border p-3 text-left transition-colors active:scale-[0.98] ${
                        on
                          ? "border-[rgb(var(--till-accent))] bg-[rgb(var(--till-accent-tint))] text-[rgb(var(--till-accent-ink))] shadow-[0_0_0_1px_rgb(var(--till-accent))]"
                          : "border-line bg-surface text-ink-soft hover:border-line-strong"
                      }`}
                    >
                      <m.Icon className="h-5 w-5" aria-hidden />
                      <span className="w-full truncate text-sm font-semibold">{m.label}</span>
                    </button>
                  );
                })}
              </div>
            </div>

            {/* ---- Who owes it ----

                ⚠️ **A debt without a name is the notebook again**, and the
                notebook is what this replaces: nothing to chase, nothing on
                anybody's card, and a total that stops adding up at the end of
                the month. The server refuses without a customer, so the button
                does too — and the search is by phone because that is the one
                thing a cashier can ask for and a guest will answer. */}
            {method === "debt" && (
              <div className="till-panel space-y-2 p-3">
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
                        // api.tillCustomer: the admin one needs a token a
                        // monoblock does not have, so this failed silently on the
                        // desktop app and worked in a browser only because
                        // somebody had signed into the panel on that machine.
                        const res = await api.tillCustomer(debtPhone.trim());
                        setDebtUser(res.user);
                      } catch (e) {
                        // ⚠️ **Said, not swallowed.** The old code caught this
                        // and cleared the name, so a refused request and a guest
                        // with no account looked identical — and the cashier's
                        // next move was to ask for the number again, which never
                        // helps.
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
                  <>
                    <p className="text-sm font-medium">{debtUser.name}</p>
                    {/* ⚠️ **Said here, in front of the guest, and not as an error
                        after the press.** The refusal is the owner's rule, not a
                        fault: a cashier who learns it from a red box on "yopish"
                        reads it as a broken till and tries again. */}
                    {!debtUser.creditAllowed && (
                      <p className="text-xs text-danger">{t.till.debtNotAllowed}</p>
                    )}
                  </>
                ) : (
                  <p className="text-xs text-[rgb(var(--till-mid))]">{t.till.debtNotFound}</p>
                )}
                <input
                  className="till-input h-11"
                  placeholder={t.till.debtNote}
                  value={debtNote}
                  onChange={(e) => setDebtNote(e.target.value)}
                />
              </div>
            )}

            {/* ---- The guest's phone ----

                ⚠️ **The code, the amount and the check number, in that order.**
                The first is what the guest points a camera at; the second is what
                they are about to confirm — a QR with no sum beside it is a request
                to approve an unknown number; and the third is the only way anybody
                finds this payment in the provider's cabinet when the automatic
                path does not finish. */}
            {payLink && (
              <div className="till-panel flex flex-col items-center gap-2 p-4">
                <QrCode value={payLink.url} size={168} />
                <p className="font-display text-xl font-bold tabular-nums">
                  {formatPrice(due, currency, lang)}
                </p>
                <p className="text-xs text-[rgb(var(--till-mid))]">
                  {t.till.payOnlineNumber(payLink.number)}
                </p>
                {waiting && (
                  <p className="text-sm font-medium">{t.till.payOnlineWaiting}</p>
                )}
                {/* ⚠️ Said on screen rather than assumed: a cashier who does not
                    know the till is watching will start pressing things, and the
                    thing they press is "pay". */}
                <p className="text-center text-xs text-[rgb(var(--till-mid))]">
                  {t.till.payOnlineHint}
                </p>
              </div>
            )}

            {/* ---- The guest's own code ----

                ⚠️ **The opposite of the QR above, and the screen has to say so.**
                There the guest points a camera at us and we wait; here they open a
                code and the cashier reads it, and the card is charged before the
                phone is back in a pocket. A cashier who mixes the two stands
                waiting for a callback that is never coming. */}
            {scan && (scanOpen || scanResult) && (
              <div className="till-panel space-y-2 p-3">
                {scanOpen && !unknown && (
                  <input
                    ref={scanField}
                    className="till-input h-11 w-full font-mono text-sm"
                    // ⚠️ The till's own keypad must not open over this: the code
                    // comes from a pistol scanner, and a cashier typing forty
                    // characters by hand at a counter is not the flow this is for.
                    inputMode="none"
                    autoComplete="off"
                    autoFocus
                    placeholder={t.till.scanPayPlaceholder}
                    value={scanCode}
                    onChange={(e) => setScanCode(e.target.value)}
                    onKeyDown={(e) => {
                      // The Enter the scanner sends at the end of the code.
                      if (e.key === "Enter") {
                        e.preventDefault();
                        void scanPay(scanCode);
                      }
                    }}
                  />
                )}
                {busy && !unknown && (
                  <p className="text-sm font-medium">{t.till.scanPayWaiting}</p>
                )}
                {unknown ? (
                  // ⚠️ Not painted as an error. The guest may well have been
                  // charged, and a red "xatolik" sends the cashier to do the one
                  // thing that must not happen next — scan again.
                  <p className="text-sm text-[rgb(var(--till-mid))]">
                    {scanResult?.error ?? t.till.scanPayUnknown}
                  </p>
                ) : scanResult?.paid ? (
                  <p className="text-sm font-semibold text-[rgb(var(--till-ok))]">
                    {t.till.scanPayPaid(scanResult.cardMask ?? "")}
                  </p>
                ) : scanResult?.error ? (
                  <p className="text-sm text-danger">{scanResult.error}</p>
                ) : (
                  <p className="text-xs text-[rgb(var(--till-mid))]">{t.till.scanPayHint}</p>
                )}
              </div>
            )}
          </section>
        </div>

        <footer className="border-t border-line bg-surface p-4">
          <div className="flex gap-2">
            <button className="till-btn min-h-12 flex-1" onClick={onCancel}>
              <LuArrowLeft className="h-4 w-4" aria-hidden />
              {t.till.back}
            </button>
            {scan ? (
              // ⚠️ **After a lost answer there is no "scan again" button**, only
              // "check the payment". The button that is missing is the feature:
              // a retry of a charge that may have succeeded is how a guest pays
              // twice, and a cashier under pressure presses whatever is offered.
              <button
                className="till-btn-primary min-h-12 flex-[2] text-base"
                disabled={busy || needsReason}
                onClick={() => {
                  if (unknown) {
                    void scanCheck();
                    return;
                  }
                  if (!scanOpen) {
                    setScanOpen(true);
                    return;
                  }
                  void scanPay(scanCode);
                }}
              >
                {unknown
                  ? t.till.scanPayCheck
                  : scanResult
                    ? t.till.scanPayAgain
                    : t.till.scanPayShow}
              </button>
            ) : online && !waiting ? (
              <button
                className="till-btn-primary min-h-12 flex-[2] text-base"
                disabled={busy || needsReason}
                onClick={() => void startOnline()}
              >
                {payLink ? t.till.payOnlineAgain : t.till.payOnlineShow}
              </button>
            ) : (
              <button
                className="till-btn-primary min-h-12 flex-[2] text-base"
                // ⚠️ A debt with nobody attached — or with somebody the owner has
                // not allowed one — is refused by the server, so the button
                // refuses first: a cashier who presses "pay" and gets an error
                // while the guest is standing there presses it again.
                //
                // ⚠️ And while a provider payment is outstanding the button is
                // not offered at all: the server refuses an unconfirmed one, and
                // a button whose only outcome is an error teaches the cashier
                // that the screen is broken.
                disabled={
                  busy ||
                  needsReason ||
                  waiting ||
                  (method === "debt" && !debtUser?.creditAllowed)
                }
                onClick={() => void submit()}
              >
                {t.till.confirmPay}
              </button>
            )}
          </div>
          {needsReason && (
            <p className="mt-2 text-center text-xs text-[rgb(var(--till-mid))]">
              {t.till.reasonRequired}
            </p>
          )}
        </footer>
      </div>
    </div>
  );
}

/** The amounts a guest is likely to hand over: the exact sum, then the round
 *  notes above it. Four at most — a row of eight is a row nobody reads. */
function quickCash(due: number): number[] {
  if (due <= 0) return [];
  const out = [due];
  for (const step of [5000, 10000, 50000, 100000, 200000]) {
    const up = Math.ceil(due / step) * step;
    if (up > due && !out.includes(up)) out.push(up);
    if (out.length >= 4) break;
  }
  return out;
}
