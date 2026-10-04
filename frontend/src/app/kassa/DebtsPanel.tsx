"use client";

import { useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAsk } from "@/components/ui/Ask";
import { TillPager, usePaged } from "@/components/till/Pager";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice, formatDateTime } from "@/lib/format";
import type { TillDebt, TillPaymentMethod } from "@/lib/types";

/**
 * A guest paying back what they owe, at the counter.
 *
 * ⚠️ **On the till because the money arrives at the till.** A regular walks in
 * on Friday with cash for Tuesday's dinner; settling that from the panel would
 * mean the cashier going to find somebody with a manager's login while the
 * guest stands there — which is exactly how a manager's password ends up
 * written on a note by the register.
 *
 * ⚠️ **Searched by phone, never listed.** A screen in a dining room showing
 * every debtor in the restaurant is the customer base on display to whoever is
 * standing at the counter. The phone is also the one thing a cashier can ask
 * for and a guest will answer.
 *
 * ⚠️ **The repayment is dated today**, so it lands in tonight's drawer — the
 * one somebody actually counts. The sale keeps Tuesday: its covers and its
 * dish counts do not move, because none of that happened on Friday.
 */
const METHODS: TillPaymentMethod[] = ["cash", "card", "transfer"];

export default function DebtsPanel({
  currency,
  onError,
}: {
  currency: string;
  onError: (msg: string) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [phone, setPhone] = useState("");
  const [found, setFound] = useState<{
    name: string;
    debts: TillDebt[];
    total: number;
  } | null>(null);
  const [busy, setBusy] = useState("");
  const [searched, setSearched] = useState(false);
  const { ask } = useAsk();
  const paged = usePaged(found?.debts ?? [], 6);

  async function search() {
    setBusy("search");
    try {
      const res = await api.tillDebts(phone.trim());
      setFound({ name: res.name ?? "", debts: res.debts, total: res.total });
      setSearched(true);
    } catch (e) {
      onError(e instanceof ApiError ? e.message : t.till.retry);
    } finally {
      setBusy("");
    }
  }

  function methodName(m: TillPaymentMethod) {
    return m === "cash"
      ? t.till.methodCash
      : m === "card"
        ? t.till.methodCard
        : t.till.methodTransfer;
  }

  async function settle(debt: TillDebt, method: TillPaymentMethod) {
    // ⚠️ **Asked before anything is written.** A tap on the method used to
    // close the debt there and then — and the method buttons sit right under
    // the debt a cashier has just found, so a thumb looking for the guest's
    // name closed a debt nobody had paid, and the drawer was short by exactly
    // that much at the end of the night. The question names the sum and the
    // method, because those are the two things being sworn to.
    const yes = await ask({
      title: t.till.debtConfirmTitle,
      body: t.till.debtConfirmBody(
        debt.number,
        formatPrice(debt.total, currency, lang),
        methodName(method),
      ),
      confirmLabel: t.till.debtConfirmYes,
      cancelLabel: t.till.debtConfirmNo,
    });
    if (!yes) return;
    setBusy(debt.orderId);
    try {
      await api.tillPayDebt(debt.orderId, method);
      // Re-read rather than edited here: a row that removed itself would hide
      // a repayment somebody else took a second earlier on another screen.
      await search();
    } catch (e) {
      onError(e instanceof ApiError ? e.message : t.till.retry);
    } finally {
      setBusy("");
    }
  }

  return (
    <div className="rounded-2xl border border-line bg-surface p-3">
      <h2 className="text-sm font-medium">{t.till.debtsTitle}</h2>
      <div className="mt-2 flex gap-2">
        <input
          className="till-input h-12 flex-1"
          inputMode="tel"
          placeholder={t.till.debtPhone}
          value={phone}
          onChange={(e) => {
            setPhone(e.target.value);
            setSearched(false);
          }}
        />
        <button
          className="till-btn min-w-24"
          disabled={busy !== "" || phone.trim().length < 4}
          onClick={() => void search()}
        >
          {t.till.debtFind}
        </button>
      </div>

      {/* ⚠️ "Nothing owed" is said out loud. A blank panel after a search looks
          identical to a search that never ran, and the cashier's next move —
          asking the guest to repeat the number — is the wrong one. */}
      {searched && found && found.debts.length === 0 && (
        <p className="mt-3 text-sm text-ink-muted">{t.till.debtsNone}</p>
      )}

      {found && found.debts.length > 0 && (
        <div className="mt-3 space-y-2">
          <div className="flex items-baseline justify-between">
            <span className="font-medium">{found.name}</span>
            <span className="font-display text-lg font-bold text-danger tabular-nums">
              {formatPrice(found.total, currency, lang)}
            </span>
          </div>
          {paged.shown.map((d) => (
            <div key={d.orderId} className="rounded-xl bg-ink/[0.03] p-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-sm">
                  {d.number}
                  <span className="ml-2 text-ink-muted">
                    {formatDateTime(d.at)}
                  </span>
                </span>
                <span className="font-display font-bold tabular-nums">
                  {formatPrice(d.total, currency, lang)}
                </span>
              </div>
              {d.note && (
                <p className="mt-1 text-xs text-ink-muted">{d.note}</p>
              )}
              {/* ⚠️ How the money arrived, asked rather than assumed: cash goes
                  into the drawer counted tonight and a card does not, and a
                  till that cannot tell them apart hands the cashier a shortage
                  at closing. */}
              <div className="mt-2 flex flex-wrap gap-2">
                {METHODS.map((m) => (
                  <button
                    key={m}
                    className="till-btn px-3 py-1.5 text-sm"
                    disabled={busy !== ""}
                    onClick={() => void settle(d, m)}
                  >
                    {methodName(m)}
                  </button>
                ))}
              </div>
            </div>
          ))}
          <TillPager page={paged.page} pages={paged.pages} onPage={paged.setPage} />
        </div>
      )}
    </div>
  );
}
