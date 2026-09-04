"use client";

// The safe: where the restaurant's cash physically is.
//
// ⚠️ **A place, not a profit and loss.** The financial report answers "did we
// make money"; this answers "where is it". Mixing them is how a report
// subtracts the same money twice — cash moved from the drawer into the safe is
// not an expense, and money handed to a buyer is not spent until it buys
// something. Nothing on this page reaches that report, deliberately.
//
// ⚠️ **The three places cash lives are three screens on purpose.** The drawer
// is counted at the end of a shift and belongs to the till; petty cash is a
// person's account and belongs beside the deliveries it pays for; this is the
// box in the office. One screen adding all three into a single number would be
// a figure nobody can count against anything.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDateTime, formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll } from "@/components/admin/PagedList";
import { useAsk } from "@/components/ui/Ask";
import type { SafeBalance, SafeEntry } from "@/lib/types";

export default function AdminSafePage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const { ask } = useAsk();

  const [balance, setBalance] = useState<SafeBalance | null>(null);
  const [entries, setEntries] = useState<SafeEntry[]>([]);
  const [kind, setKind] = useState<"in" | "out">("in");
  const [amount, setAmount] = useState("");
  const [category, setCategory] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api
      .adminSafe()
      .then((res) => {
        setBalance(res.balance);
        setEntries(res.entries);
        setError("");
      })
      // ⚠️ Said rather than swallowed: an empty ledger and a refused request
      // look identical on this screen, and only one of them means the safe is
      // empty.
      .catch((e: unknown) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      );
  }, [t.common.loadFailed]);

  useEffect(load, [load, scope.scopeKey]);

  async function save() {
    const sum = Number(amount);
    if (!(sum > 0)) return;
    setBusy(true);
    try {
      await api.adminCreateSafeEntry({
        kind,
        amount: sum,
        category: category.trim(),
        note: note.trim(),
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

  async function remove(row: SafeEntry) {
    if (!(await ask({ title: t.safe.deleteAsk(formatPrice(row.amount)), danger: true }))) {
      return;
    }
    try {
      await api.adminDeleteSafeEntry(row.id);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.safe.title}</h1>
        <p className="mt-1 max-w-3xl text-sm text-ink-soft">{t.safe.intro}</p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      {/* ---- What is in it ----
          ⚠️ Beside the date of the last movement, not alone. A balance with no
          date is a number somebody trusts; one with a date is a number they can
          check against the notes. */}
      <div className="card p-4">
        <p className="text-xs text-ink-muted">{t.safe.balance}</p>
        <p
          className={`text-3xl font-bold tabular-nums ${
            (balance?.balance ?? 0) < 0 ? "text-danger" : ""
          }`}
        >
          {formatPrice(balance?.balance ?? 0)}
        </p>
        {(balance?.balance ?? 0) < 0 && (
          <p className="mt-1 text-sm text-danger">{t.safe.negative}</p>
        )}
        <p className="mt-1 text-sm text-ink-muted">
          {t.safe.inOut(
            formatPrice(balance?.in ?? 0),
            formatPrice(balance?.out ?? 0),
          )}
          {balance?.lastAt ? ` · ${formatDateTime(balance.lastAt)}` : ""}
        </p>
      </div>

      {/* ---- Adding a movement ---- */}
      <div className="card space-y-3 p-3">
        <div className="flex flex-wrap items-end gap-2">
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.safe.kind}</span>
            <select
              className="input mt-1 w-auto"
              value={kind}
              onChange={(e) => setKind(e.target.value as "in" | "out")}
            >
              <option value="in">{t.safe.kindIn}</option>
              <option value="out">{t.safe.kindOut}</option>
            </select>
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.safe.amount}</span>
            <input
              className="input mt-1 w-32"
              inputMode="numeric"
              value={amount}
              onChange={(e) => setAmount(e.target.value.replace(/\D/g, ""))}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.safe.category}</span>
            {/* ⚠️ Free text, like a till entry's: every restaurant spends money
                on something the next one does not, and a fixed list sends all of
                it to "boshqa". */}
            <input
              className="input mt-1 w-44"
              placeholder={t.safe.categoryPh}
              value={category}
              onChange={(e) => setCategory(e.target.value)}
            />
          </label>
          <label className="block flex-1 text-sm">
            <span className="text-xs text-ink-muted">{t.safe.note}</span>
            <input
              className="input mt-1 w-full"
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          </label>
          <button
            className="btn-primary px-4 py-2"
            disabled={busy || !(Number(amount) > 0)}
            onClick={() => void save()}
          >
            {t.common.add}
          </button>
        </div>
      </div>

      {/* ---- The ledger ---- */}
      <div className="card p-0">
        <ListScroll>
          {entries.length === 0 ? (
            <p className="p-6 text-center text-sm text-ink-muted">
              {t.safe.empty}
            </p>
          ) : (
            <ul className="divide-y divide-line text-sm">
              {entries.map((row) => (
                <li
                  key={row.id}
                  className="flex items-center gap-3 px-3 py-2.5"
                >
                  <span className="min-w-0 flex-1">
                    <span className="block truncate font-medium">
                      {row.category || t.safe.noCategory}
                      {/* ⚠️ An automatic row says what wrote it. Without that,
                          somebody deleting "a duplicate" would be deleting the
                          safe's only record of a hand-over that really
                          happened. */}
                      {row.refKind && (
                        <span className="ml-2 rounded-full bg-ink/[0.06] px-1.5 text-xs text-ink-muted">
                          {t.safe.refKinds[row.refKind] ?? row.refKind}
                        </span>
                      )}
                    </span>
                    <span className="text-xs text-ink-muted">
                      {formatDateTime(row.at)}
                      {row.by ? ` · ${row.by}` : ""}
                      {row.note ? ` · ${row.note}` : ""}
                    </span>
                  </span>
                  <span
                    className={`shrink-0 font-semibold tabular-nums ${
                      row.kind === "out" ? "text-danger" : "text-success"
                    }`}
                  >
                    {row.kind === "out" ? "−" : "+"}
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
