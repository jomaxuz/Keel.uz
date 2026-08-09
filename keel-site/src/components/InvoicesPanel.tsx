"use client";

// One customer's invoice ledger.
//
// The shape of this screen follows from how Keel actually gets paid right now:
// **somebody drives to the restaurant and is handed cash.** There is no MChJ
// yet, so no stamped invoice and no bank account. That means the important
// field is not the amount — it is *who took the money*, which is why it is a
// visible input with a sentence next to it rather than something recorded
// silently from the session.
//
// Everything else is a consequence of it being a ledger:
//   • an invoice is issued for the period that closed, never the running one;
//   • the amount is frozen server-side, so nothing here recomputes it;
//   • payments append, because money arrives in pieces.

import { useCallback, useEffect, useState } from "react";
import { useT } from "@/lib/i18n/client";
import {
  invoices as fetchInvoices,
  issueInvoice,
  payInvoice,
  voidInvoice,
  money,
  dayLabel,
  type Invoice,
  type PayMethod,
} from "@/lib/api";

export default function InvoicesPanel({ tenantId }: { tenantId: string }) {
  const { t } = useT();
  const [items, setItems] = useState<Invoice[] | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [payFor, setPayFor] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const out = await fetchInvoices({ tenantId });
      setItems(out.items);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [tenantId]);

  useEffect(() => {
    load();
  }, [load]);

  async function issue() {
    setBusy(true);
    setError("");
    try {
      await issueInvoice(tenantId);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function drop(inv: Invoice) {
    const reason = prompt(t.dash.invoiceVoidReason);
    // Required, exactly as for cancelling an order: an invoice that
    // disappeared without a sentence is one nobody can explain later.
    if (!reason?.trim()) return;
    try {
      await voidInvoice(inv.id, reason.trim());
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <section className="card">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm font-semibold text-ink">{t.dash.invoicesTitle}</p>
        <button
          type="button"
          onClick={issue}
          disabled={busy}
          className="rounded-xl bg-ink px-3 py-2 text-xs font-semibold text-surface disabled:opacity-50"
        >
          {busy ? t.dash.invoiceIssuing : t.dash.invoiceIssue}
        </button>
      </div>

      {error && (
        <p className="mt-3 text-sm text-rose-600 dark:text-rose-400">{error}</p>
      )}

      {items && items.length === 0 && (
        <p className="mt-3 text-sm text-ink-muted">{t.dash.invoiceNone}</p>
      )}

      {items && items.length > 0 && (
        <ul className="mt-4 divide-y divide-line">
          {items.map((inv) => (
            <li key={inv.id} className="py-3">
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <div className="min-w-0">
                  <span className="text-sm font-semibold text-ink">
                    {inv.number}
                  </span>
                  <span className="ml-2 text-xs text-ink-muted">
                    {dayLabel(inv.from)} — {dayLabel(inv.to)} ·{" "}
                    {t.dash.invoiceOrders(inv.orders)}
                    {/* Named on the row, because "why is this three million more?" is asked
                        while looking at the list rather than after opening something. */}
                    {!!inv.watermarkFee && (
                      <span className="block text-xs text-ink-muted">
                        {t.dash.invoiceWatermark(money(inv.watermarkFee))}
                      </span>
                    )}
                  </span>
                </div>
                <div className="flex items-center gap-3 text-sm">
                  <span className="tabular-nums font-semibold text-ink">
                    {money(inv.amount)}
                  </span>
                  <Badge status={inv.status} t={t} />
                </div>
              </div>

              {inv.status !== "void" && inv.collected > 0 && (
                <p className="mt-1 text-xs text-ink-soft">
                  {t.dash.invoiceCollected}: {money(inv.collected)}
                  {inv.outstanding > 0 && (
                    <span className="ml-2 text-amber-700 dark:text-amber-300">
                      {t.dash.invoiceOutstanding}: {money(inv.outstanding)}
                    </span>
                  )}
                </p>
              )}
              {inv.voidReason && (
                <p className="mt-1 text-xs text-ink-muted">{inv.voidReason}</p>
              )}

              {/* Who handed over what, and who took it. The whole reason this
                  is a list and not a paid/unpaid flag. */}
              {inv.paid.length > 0 && (
                <ul className="mt-1 space-y-0.5 text-xs text-ink-muted">
                  {inv.paid.map((p, i) => (
                    <li key={i}>
                      {money(p.amount)} ·{" "}
                      {p.method === "cash"
                        ? t.dash.invoiceCash
                        : t.dash.invoiceTransfer}{" "}
                      · {p.receivedBy}
                      {p.note ? ` · ${p.note}` : ""}
                    </li>
                  ))}
                </ul>
              )}

              {inv.status === "open" && (
                <div className="mt-2 flex flex-wrap gap-2">
                  <button
                    type="button"
                    className="text-xs font-semibold underline underline-offset-2"
                    onClick={() => setPayFor(payFor === inv.id ? null : inv.id)}
                  >
                    {t.dash.invoicePay}
                  </button>
                  <button
                    type="button"
                    className="text-xs text-ink-muted underline underline-offset-2"
                    onClick={() => drop(inv)}
                  >
                    {t.dash.invoiceVoid}
                  </button>
                </div>
              )}

              {payFor === inv.id && (
                <PayForm
                  invoice={inv}
                  onDone={async () => {
                    setPayFor(null);
                    await load();
                  }}
                  onError={setError}
                />
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function Badge({
  status,
  t,
}: {
  status: Invoice["status"];
  t: ReturnType<typeof useT>["t"];
}) {
  const map = {
    open: ["bg-amber-500/15 text-amber-700 dark:text-amber-300", t.dash.invoiceStatusOpen],
    paid: [
      "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
      t.dash.invoiceStatusPaid,
    ],
    void: ["bg-ink/10 text-ink-muted", t.dash.invoiceStatusVoid],
  } as const;
  const [cls, label] = map[status];
  return (
    <span className={`rounded-lg px-2 py-1 text-xs font-semibold ${cls}`}>
      {label}
    </span>
  );
}

function PayForm({
  invoice,
  onDone,
  onError,
}: {
  invoice: Invoice;
  onDone: () => void;
  onError: (s: string) => void;
}) {
  const { t } = useT();
  // Pre-filled with what is still owed: the common case is one handover that
  // settles the invoice, and retyping six digits invites a typo.
  const [amount, setAmount] = useState(String(invoice.outstanding));
  const [method, setMethod] = useState<PayMethod>("cash");
  const [receivedBy, setReceivedBy] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit() {
    const n = Number(amount);
    if (!Number.isFinite(n) || n <= 0) return;
    setBusy(true);
    try {
      await payInvoice(invoice.id, {
        amount: Math.round(n),
        method,
        receivedBy: receivedBy.trim(),
        note: note.trim(),
      });
      onDone();
    } catch (e) {
      onError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mt-3 rounded-xl border border-line bg-surface p-3">
      <div className="grid gap-2 sm:grid-cols-2">
        <label className="text-xs font-medium text-ink-soft">
          {t.dash.invoicePayAmount}
          <input
            className="mt-1 w-full rounded-lg border border-line px-2 py-1.5 text-sm"
            value={amount}
            inputMode="numeric"
            onChange={(e) => setAmount(e.target.value)}
          />
        </label>
        <label className="text-xs font-medium text-ink-soft">
          {t.dash.invoicePayMethod}
          <select
            className="mt-1 w-full rounded-lg border border-line px-2 py-1.5 text-sm"
            value={method}
            onChange={(e) => setMethod(e.target.value as PayMethod)}
          >
            <option value="cash">{t.dash.invoiceCash}</option>
            <option value="transfer">{t.dash.invoiceTransfer}</option>
          </select>
        </label>
        <label className="text-xs font-medium text-ink-soft">
          {t.dash.invoiceReceivedBy}
          <input
            className="mt-1 w-full rounded-lg border border-line px-2 py-1.5 text-sm"
            value={receivedBy}
            onChange={(e) => setReceivedBy(e.target.value)}
          />
        </label>
        <label className="text-xs font-medium text-ink-soft">
          {t.dash.invoiceNote}
          <input
            className="mt-1 w-full rounded-lg border border-line px-2 py-1.5 text-sm"
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
        </label>
      </div>
      {/* Said out loud, because "who took it" is the field an operator in a
          hurry will otherwise leave empty. */}
      <p className="mt-2 text-xs text-ink-muted">{t.dash.invoiceCashNote}</p>
      <button
        type="button"
        onClick={submit}
        disabled={busy}
        className="mt-2 rounded-xl bg-ink px-3 py-2 text-xs font-semibold text-surface disabled:opacity-50"
      >
        {busy ? t.dash.invoicePaying : t.dash.invoicePay}
      </button>
    </div>
  );
}
