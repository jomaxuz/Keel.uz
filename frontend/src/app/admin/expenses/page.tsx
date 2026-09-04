"use client";

// What the restaurant spends that nothing else records.
//
// ⚠️ **This is the line that made the financial report optimistic.** Deliveries
// were counted and wages were counted; rent, electricity, gas, tax, repairs and
// the couriers' own pay were not — so "in − out" read better than the month had
// been, by roughly what the building costs, every month. A figure wrong in the
// same direction every time is one a restaurant learns to trust.
//
// ⚠️ **Only what has no document of its own.** A delivery belongs on the
// deliveries screen and a wage on payroll; both already have their own line in
// the report, and entering either here as well would count it twice — a
// double-counted cost is indistinguishable from a real one. The page says so
// rather than relying on somebody knowing it.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDate, formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll } from "@/components/admin/PagedList";
import { useAsk } from "@/components/ui/Ask";
import type { Expense } from "@/lib/types";

const TODAY = () => new Date().toISOString().slice(0, 10);

export default function AdminExpensesPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const { ask } = useAsk();

  const [rows, setRows] = useState<Expense[]>([]);
  const [total, setTotal] = useState(0);
  const [at, setAt] = useState(TODAY);
  const [category, setCategory] = useState("");
  const [amount, setAmount] = useState("");
  const [note, setNote] = useState("");
  const [method, setMethod] = useState("cash");
  const [fromSafe, setFromSafe] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api
      .adminExpenses()
      .then((res) => {
        setRows(res.expenses);
        setTotal(res.total);
        setError("");
      })
      .catch((e: unknown) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      );
  }, [t.common.loadFailed]);

  useEffect(load, [load, scope.scopeKey]);

  async function save() {
    const sum = Number(amount);
    if (!(sum > 0) || !category.trim()) return;
    setBusy(true);
    try {
      await api.adminCreateExpense({
        at,
        category: category.trim(),
        amount: sum,
        note: note.trim(),
        method,
        // ⚠️ Only meaningful for cash — a transfer never touches the box, and
        // sending it would put a movement in a ledger nothing physically did.
        fromSafe: method === "cash" && fromSafe,
      });
      setAmount("");
      setNote("");
      setCategory("");
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  async function remove(row: Expense) {
    if (
      !(await ask({
        title: t.expenses.deleteAsk(formatPrice(row.amount)),
        body: t.expenses.deleteBody,
        danger: true,
      }))
    ) {
      return;
    }
    try {
      await api.adminDeleteExpense(row.id);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.expenses.title}</h1>
        <p className="mt-1 max-w-3xl text-sm text-ink-soft">
          {t.expenses.intro}
        </p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      <div className="card space-y-2 p-3">
        <div className="flex flex-wrap items-end gap-2">
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.expenses.date}</span>
            <input
              type="date"
              className="input mt-1 w-auto"
              value={at}
              onChange={(e) => setAt(e.target.value)}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.expenses.category}</span>
            {/* ⚠️ Free text with suggestions rather than a fixed list: every
                restaurant pays for something the next one does not, and a closed
                list sends all of it to "boshqa". */}
            <input
              className="input mt-1 w-52"
              list="expense-categories"
              placeholder={t.expenses.categoryPh}
              value={category}
              onChange={(e) => setCategory(e.target.value)}
            />
            <datalist id="expense-categories">
              {t.expenses.suggestions.map((c) => (
                <option key={c} value={c} />
              ))}
            </datalist>
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.expenses.amount}</span>
            <input
              className="input mt-1 w-32"
              inputMode="numeric"
              value={amount}
              onChange={(e) => setAmount(e.target.value.replace(/\D/g, ""))}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.expenses.method}</span>
            <select
              className="input mt-1 w-auto"
              value={method}
              onChange={(e) => setMethod(e.target.value)}
            >
              <option value="cash">{t.expenses.cash}</option>
              <option value="transfer">{t.expenses.transfer}</option>
              <option value="card">{t.expenses.card}</option>
            </select>
          </label>
          {/* ⚠️ Shown only for cash, because a transfer never touches the box —
              and a tickbox that does nothing teaches a room that the screen is
              decorative. */}
          {method === "cash" && (
            <label className="flex items-center gap-1.5 pb-2 text-sm text-ink-muted">
              <input
                type="checkbox"
                checked={fromSafe}
                onChange={(e) => setFromSafe(e.target.checked)}
              />
              {t.expenses.fromSafe}
            </label>
          )}
          <label className="block flex-1 text-sm">
            <span className="text-xs text-ink-muted">{t.expenses.note}</span>
            <input
              className="input mt-1 w-full"
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          </label>
          <button
            className="btn-primary px-4 py-2"
            disabled={busy || !(Number(amount) > 0) || !category.trim()}
            onClick={() => void save()}
          >
            {t.common.add}
          </button>
        </div>
      </div>

      <div className="card p-0">
        <div className="flex items-center justify-between border-b border-line px-3 py-2 text-sm">
          <span className="text-ink-muted">{t.expenses.listTitle}</span>
          <span className="font-semibold tabular-nums">
            {formatPrice(total)}
          </span>
        </div>
        <ListScroll>
          {rows.length === 0 ? (
            <p className="p-6 text-center text-sm text-ink-muted">
              {t.expenses.empty}
            </p>
          ) : (
            <ul className="divide-y divide-line text-sm">
              {rows.map((row) => (
                <li key={row.id} className="flex items-center gap-3 px-3 py-2.5">
                  <span className="min-w-0 flex-1">
                    <span className="block truncate font-medium">
                      {row.category}
                    </span>
                    <span className="text-xs text-ink-muted">
                      {formatDate(row.at)}
                      {row.method ? ` · ${t.expenses.methods[row.method] ?? row.method}` : ""}
                      {row.createdBy ? ` · ${row.createdBy}` : ""}
                      {row.note ? ` · ${row.note}` : ""}
                    </span>
                  </span>
                  <span className="shrink-0 font-semibold tabular-nums">
                    {formatPrice(row.amount)}
                  </span>
                  <button
                    className="btn-ghost px-2 py-1 text-xs"
                    onClick={() => void remove(row)}
                  >
                    {t.common.delete}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </ListScroll>
      </div>
    </div>
  );
}
