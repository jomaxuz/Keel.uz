"use client";

// Perechisleniye: the money a rail is still holding.
//
// ⚠️ **The gap between "the guest paid" and "we have it" is weeks, and nothing
// in the system knew it existed.** An aggregator — Yandex Eats, Uzum Tezkor —
// takes the guest's money when the order is placed and transfers it once a
// month, minus its commission. Card terminals and the online rails work the
// same way on a shorter clock. A February that sold twelve million through Uzum
// Tezkor is a February in which the restaurant *has* nothing yet, and a March
// statement reading nine and a half million is either right or short by two
// hundred thousand — and no screen could tell those apart.
//
// ⚠️ **The comparison is the page.** Recording that money arrived is
// bookkeeping; putting it beside what was sold through that rail is the
// sentence that catches an underpayment.
//
// ⚠️ **What arrives is not revenue.** The sale was counted the day the guest
// paid; this is the same money changing location, exactly like cash carried
// from the drawer to the safe. Only the commission is a cost, and only the
// commission reaches the financial report.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDate, formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll } from "@/components/admin/PagedList";
import { useAsk } from "@/components/ui/Ask";
import type { AggregatorAccount, Payout, PayoutBalance } from "@/lib/types";

const TODAY = () => new Date().toISOString().slice(0, 10);

export default function AdminPayoutsPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const { ask } = useAsk();

  const [rows, setRows] = useState<Payout[]>([]);
  const [balances, setBalances] = useState<PayoutBalance[]>([]);
  const [rails, setRails] = useState<AggregatorAccount[]>([]);
  const [error, setError] = useState("");

  const [provider, setProvider] = useState("");
  const [periodFrom, setPeriodFrom] = useState("");
  const [periodTo, setPeriodTo] = useState("");
  const [gross, setGross] = useState("");
  const [commission, setCommission] = useState("");
  const [net, setNet] = useState("");
  const [receivedAt, setReceivedAt] = useState(TODAY);
  const [account, setAccount] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    api
      .adminPayouts()
      .then((res) => {
        setRows(res.payouts);
        setBalances(res.balances);
        setRails(res.rails);
        setProvider((p) => p || res.rails[0]?.id || "");
        setError("");
      })
      .catch((e: unknown) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      );
  }, [t.common.loadFailed]);

  useEffect(load, [load, scope.scopeKey]);

  async function save() {
    if (!provider || !(Number(net) > 0 || Number(gross) > 0)) return;
    setBusy(true);
    try {
      await api.adminCreatePayout({
        provider,
        periodFrom,
        periodTo,
        gross: Number(gross) || 0,
        commission: Number(commission) || 0,
        net: Number(net) || 0,
        receivedAt,
        account: account.trim(),
      });
      setGross("");
      setCommission("");
      setNet("");
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  async function remove(row: Payout) {
    if (
      !(await ask({
        title: t.payouts.deleteAsk(formatPrice(row.net)),
        danger: true,
      }))
    ) {
      return;
    }
    try {
      await api.adminDeletePayout(row.id);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    }
  }

  const num = (v: string, set: (s: string) => void, w = "w-32") => (
    <input
      className={`input mt-1 ${w}`}
      inputMode="numeric"
      value={v}
      onChange={(e) => set(e.target.value.replace(/\D/g, ""))}
    />
  );

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.payouts.title}</h1>
        <p className="mt-1 max-w-3xl text-sm text-ink-soft">{t.payouts.intro}</p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      {/* ---- Who is holding how much ----
          ⚠️ First on the page, because it is the question. The ledger below is
          the evidence for it. */}
      {balances.length > 0 && (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {balances.map((b) => (
            <div key={b.provider} className="card p-4">
              <p className="text-sm font-medium">{b.name}</p>
              <p className="mt-2 text-xs text-ink-muted">
                {b.settledThrough
                  ? t.payouts.owedSince(formatDate(b.settledThrough))
                  : t.payouts.soldEver}
              </p>
              <p
                className={`font-display text-2xl font-bold tabular-nums ${
                  b.settledThrough ? "text-amber-600 dark:text-amber-400" : ""
                }`}
              >
                {formatPrice(b.sold)}
              </p>
              <p className="mt-2 text-xs text-ink-muted">
                {t.payouts.receivedTotal(
                  formatPrice(b.received),
                  formatPrice(b.commission),
                )}
                {b.lastAt ? ` · ${formatDate(b.lastAt)}` : ""}
              </p>
            </div>
          ))}
        </div>
      )}

      {/* ---- Recording a statement ---- */}
      <div className="card p-3">
        <div className="flex flex-wrap items-end gap-2">
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.payouts.provider}</span>
            <select
              className="input mt-1 w-auto"
              value={provider}
              onChange={(e) => setProvider(e.target.value)}
            >
              {rails.map((rl) => (
                <option key={rl.id} value={rl.id}>
                  {rl.name}
                </option>
              ))}
            </select>
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">
              {t.payouts.periodFrom}
            </span>
            <input
              type="date"
              className="input mt-1 w-auto"
              value={periodFrom}
              onChange={(e) => setPeriodFrom(e.target.value)}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.payouts.periodTo}</span>
            <input
              type="date"
              className="input mt-1 w-auto"
              value={periodTo}
              onChange={(e) => setPeriodTo(e.target.value)}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.payouts.gross}</span>
            {num(gross, setGross)}
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">
              {t.payouts.commission}
            </span>
            {num(commission, setCommission, "w-28")}
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.payouts.net}</span>
            {num(net, setNet)}
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">
              {t.payouts.receivedAt}
            </span>
            <input
              type="date"
              className="input mt-1 w-auto"
              value={receivedAt}
              onChange={(e) => setReceivedAt(e.target.value)}
            />
          </label>
          <label className="block flex-1 text-sm">
            <span className="text-xs text-ink-muted">{t.payouts.account}</span>
            <input
              className="input mt-1 w-full"
              value={account}
              onChange={(e) => setAccount(e.target.value)}
            />
          </label>
          <button
            className="btn-primary px-4 py-2"
            disabled={
              busy || !provider || !(Number(net) > 0 || Number(gross) > 0)
            }
            onClick={() => void save()}
          >
            {t.common.add}
          </button>
        </div>
        <p className="mt-2 text-xs text-ink-muted">{t.payouts.formHint}</p>
      </div>

      {/* ---- What has arrived ---- */}
      <div className="card p-0">
        <ListScroll>
          {rows.length === 0 ? (
            <p className="p-6 text-center text-sm text-ink-muted">
              {t.payouts.empty}
            </p>
          ) : (
            <ul className="divide-y divide-line text-sm">
              {rows.map((row) => {
                // ⚠️ Shown, never corrected: a non-zero difference is a real
                // event — a refund clawed back, a penalty, an adjustment — and
                // hiding it turns a question the owner should ask into a number
                // nobody can explain three months later.
                const diff = row.gross - row.commission - row.net;
                return (
                  <li
                    key={row.id}
                    className="flex items-center gap-3 px-3 py-2.5"
                  >
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-medium">
                        {row.providerName || row.provider}
                      </span>
                      <span className="text-xs text-ink-muted">
                        {formatDate(row.receivedAt)}
                        {row.periodFrom
                          ? ` · ${t.payouts.forPeriod(
                              formatDate(row.periodFrom),
                              formatDate(row.periodTo),
                            )}`
                          : ""}
                        {row.account ? ` · ${row.account}` : ""}
                        {row.createdBy ? ` · ${row.createdBy}` : ""}
                      </span>
                      {diff !== 0 && (
                        <span className="mt-0.5 block text-xs text-amber-600 dark:text-amber-400">
                          {t.payouts.mismatch(formatPrice(diff))}
                        </span>
                      )}
                    </span>
                    <span className="shrink-0 text-right">
                      <span className="block font-semibold tabular-nums">
                        {formatPrice(row.net)}
                      </span>
                      {row.commission > 0 && (
                        <span className="block text-xs tabular-nums text-ink-muted">
                          −{formatPrice(row.commission)}
                        </span>
                      )}
                    </span>
                    <button
                      className="btn-ghost px-2 py-1 text-xs"
                      onClick={() => void remove(row)}
                    >
                      {t.common.delete}
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </ListScroll>
      </div>
    </div>
  );
}
