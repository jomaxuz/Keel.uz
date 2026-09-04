"use client";

// Where the restaurant's money is.
//
// ⚠️ **Three kinds of having, and they are never added together.** Cash in a
// box can be spent tonight. Money in the bank account can be spent this week.
// Money an aggregator is still holding can be spent when somebody else decides.
// A single "we have X" would be the most quotable and least true number the
// panel could show — and it is exactly the one an owner would take to a bank or
// a landlord.
//
// ⚠️ **Every figure says whether it was counted or added up.** They fail in
// opposite directions: a counted one (a drawer, a balance read off a bank app)
// goes stale, a summed one (the safe's ledger, a rail's unsettled sales) goes
// wrong when a document is missing. Which failure to look for is half the
// information.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDate, formatDateTime, formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { hasId } from "@/lib/id";
import type { Collection, MoneyPlace, MoneyPosition } from "@/lib/types";

export default function AdminMoneyPage() {
  const t = useAdminT();
  const scope = useAdminScope();

  const [pos, setPos] = useState<MoneyPosition | null>(null);
  const [rows, setRows] = useState<Collection[]>([]);
  const [due, setDue] = useState<Collection | null>(null);
  const [error, setError] = useState("");

  const [account, setAccount] = useState("");
  const [bankAmount, setBankAmount] = useState("");
  const [bankAt, setBankAt] = useState(() => new Date().toISOString().slice(0, 10));

  const [amount, setAmount] = useState("");
  const [takenBy, setTakenBy] = useState("");
  const [bag, setBag] = useState("");
  const [to, setTo] = useState<"bank" | "safe">("bank");
  const [fromSafe, setFromSafe] = useState(true);
  const [limit, setLimit] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    Promise.all([api.adminMoney(), api.adminCollections()])
      .then(([m, c]) => {
        setPos(m);
        setRows(c.collections);
        setDue(c.due);
        setLimit(m.cashLimit ? String(m.cashLimit) : "");
        setError("");
      })
      .catch((e: unknown) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      );
  }, [t.common.loadFailed]);

  useEffect(load, [load, scope.scopeKey]);

  async function saveBank() {
    if (!account.trim() || !bankAmount) return;
    setBusy(true);
    try {
      await api.adminSaveBankBalance({
        account: account.trim(),
        amount: Number(bankAmount) || 0,
        at: bankAt,
      });
      setBankAmount("");
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  async function collect() {
    if (!(Number(amount) > 0)) return;
    setBusy(true);
    try {
      await api.adminCreateCollection({
        amount: Number(amount),
        to,
        takenBy: takenBy.trim(),
        bag: bag.trim(),
        fromSafe,
      });
      setAmount("");
      setBag("");
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  // ⚠️ `hasId`, not truthiness: a zero ObjectID reaches the browser as
  // "000…0", which is truthy, and the save would go to a branch that is not
  // one. This trap has cost this codebase a whole afternoon before.
  const limitBranch = hasId(pos?.branchId) ? pos!.branchId! : "";

  async function saveLimit() {
    if (!limitBranch) return;
    setBusy(true);
    try {
      await api.adminSetCashLimit(limitBranch, Number(limit) || 0);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  const group = (title: string, hint: string, places: MoneyPlace[], total: number) => (
    <div className="card p-4">
      <div className="flex items-baseline justify-between gap-3">
        <p className="text-sm font-medium">{title}</p>
        <p className="font-display text-xl font-bold tabular-nums">
          {formatPrice(total)}
        </p>
      </div>
      <p className="mt-0.5 text-xs text-ink-muted">{hint}</p>
      <ul className="mt-3 space-y-1.5 text-sm">
        {places.length === 0 ? (
          <li className="text-ink-muted">{t.money.nothingHere}</li>
        ) : (
          places.map((p, i) => (
            <li key={`${p.kind}-${p.name}-${i}`} className="flex items-start gap-3">
              <span className="min-w-0 flex-1">
                <span className="block truncate">{p.name}</span>
                <span className="text-xs text-ink-muted">
                  {/* ⚠️ Counted or summed, said out loud — the two go wrong in
                      opposite directions. */}
                  {p.counted ? t.money.counted : t.money.summed}
                  {p.at ? ` · ${formatDate(p.at)}` : ""}
                  {p.note ? ` · ${p.note}` : ""}
                </span>
              </span>
              <span className="shrink-0 tabular-nums">{formatPrice(p.amount)}</span>
            </li>
          ))
        )}
      </ul>
    </div>
  );

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.money.title}</h1>
        <p className="mt-1 max-w-3xl text-sm text-ink-soft">{t.money.intro}</p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      {pos && (
        <div className="grid gap-3 lg:grid-cols-3">
          {group(t.money.cash, t.money.cashHint, pos.cash, pos.cashTotal)}
          {group(t.money.bank, t.money.bankHint, pos.bank, pos.bankTotal)}
          {group(t.money.rails, t.money.railsHint, pos.rails, pos.railsTotal)}
        </div>
      )}

      {/* ⚠️ **A legal ceiling, not a preference**, and the warning says which.
          Cash above the limit agreed with the bank must be handed over for
          crediting to the account; only wages may stay, for three working
          days. Shown only when a limit has been entered — inventing the bank's
          figure would be worse than silence. */}
      {pos?.overLimit && (
        <p className="rounded-2xl bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
          {t.money.overLimit(
            formatPrice(pos.cashTotal),
            formatPrice(pos.cashLimit ?? 0),
          )}
        </p>
      )}

      {/* ---- The bank balance, counted ---- */}
      <div className="card p-3">
        <p className="text-sm font-medium">{t.money.bankTitle}</p>
        <p className="mt-0.5 text-xs text-ink-muted">{t.money.bankFormHint}</p>
        <div className="mt-2 flex flex-wrap items-end gap-2">
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.money.account}</span>
            <input
              className="input mt-1 w-52"
              value={account}
              onChange={(e) => setAccount(e.target.value)}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.money.balance}</span>
            <input
              className="input mt-1 w-36"
              inputMode="numeric"
              value={bankAmount}
              onChange={(e) => setBankAmount(e.target.value.replace(/\D/g, ""))}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.money.asOf}</span>
            <input
              type="date"
              className="input mt-1 w-auto"
              value={bankAt}
              onChange={(e) => setBankAt(e.target.value)}
            />
          </label>
          <button
            className="btn-primary px-4 py-2"
            disabled={busy || !account.trim() || !bankAmount}
            onClick={() => void saveBank()}
          >
            {t.common.save}
          </button>
        </div>
      </div>

      {/* ---- Inkassatsiya ---- */}
      <div className="card p-3">
        <p className="text-sm font-medium">{t.money.collectTitle}</p>
        <p className="mt-0.5 text-xs text-ink-muted">{t.money.collectHint}</p>

        {/* What a handover made right now would be checked against. ⚠️ Shown
            before signing, and frozen onto the document afterwards — the same
            function produces both. */}
        {due && (
          <p className="mt-2 rounded-xl bg-ink/[0.04] px-3 py-2 text-sm">
            {t.money.dueLine(
              due.shifts,
              formatPrice(due.counted),
              formatPrice(due.safeBefore),
            )}
            {due.variance !== 0 && (
              <span className="ml-2 text-amber-600 dark:text-amber-400">
                {t.money.dueVariance(formatPrice(due.variance))}
              </span>
            )}
            {due.fromAt && (
              <span className="ml-2 text-ink-muted">
                {t.money.since(formatDateTime(due.fromAt))}
              </span>
            )}
          </p>
        )}

        <div className="mt-2 flex flex-wrap items-end gap-2">
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.money.amount}</span>
            <input
              className="input mt-1 w-36"
              inputMode="numeric"
              value={amount}
              onChange={(e) => setAmount(e.target.value.replace(/\D/g, ""))}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.money.to}</span>
            <select
              className="input mt-1 w-auto"
              value={to}
              onChange={(e) => setTo(e.target.value as "bank" | "safe")}
            >
              <option value="bank">{t.money.toBank}</option>
              <option value="safe">{t.money.toSafe}</option>
            </select>
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.money.takenBy}</span>
            <input
              className="input mt-1 w-44"
              value={takenBy}
              onChange={(e) => setTakenBy(e.target.value)}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.money.bag}</span>
            <input
              className="input mt-1 w-28"
              value={bag}
              onChange={(e) => setBag(e.target.value)}
            />
          </label>
          <label className="flex items-center gap-1.5 pb-2 text-sm text-ink-muted">
            <input
              type="checkbox"
              checked={fromSafe}
              onChange={(e) => setFromSafe(e.target.checked)}
            />
            {t.money.fromSafe}
          </label>
          <button
            className="btn-primary px-4 py-2"
            disabled={busy || !(Number(amount) > 0)}
            onClick={() => void collect()}
          >
            {t.money.collect}
          </button>
        </div>

        {rows.length > 0 && (
          <ul className="mt-3 divide-y divide-line border-t border-line text-sm">
            {rows.slice(0, 10).map((c) => (
              <li key={c.id} className="flex items-center gap-3 py-2">
                <span className="min-w-0 flex-1">
                  <span className="block truncate">
                    {c.to === "bank" ? t.money.toBank : t.money.toSafe}
                    {c.takenBy ? ` · ${c.takenBy}` : ""}
                    {c.bag ? ` · ${c.bag}` : ""}
                  </span>
                  <span className="text-xs text-ink-muted">
                    {formatDateTime(c.at)} · {t.money.coveredShifts(c.shifts)}
                    {c.diff !== 0
                      ? ` · ${t.money.diff(formatPrice(c.diff))}`
                      : ` · ${t.money.matched}`}
                  </span>
                </span>
                <span className="shrink-0 font-semibold tabular-nums">
                  {formatPrice(c.amount)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* ---- The bank's ceiling ---- */}
      {limitBranch && (
        <div className="card p-3">
          <p className="text-sm font-medium">{t.money.limitTitle}</p>
          <p className="mt-0.5 max-w-3xl text-xs text-ink-muted">
            {t.money.limitHint}
          </p>
          <div className="mt-2 flex flex-wrap items-end gap-2">
            <input
              className="input w-40"
              inputMode="numeric"
              value={limit}
              onChange={(e) => setLimit(e.target.value.replace(/\D/g, ""))}
            />
            <button
              className="btn px-4 py-2"
              disabled={busy}
              onClick={() => void saveLimit()}
            >
              {t.common.save}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
