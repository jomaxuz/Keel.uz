"use client";

// Counting the store, on the phone that is already in the counter's hand.
//
// ⚠️ **The count happened in the store and was typed in the office.** Somebody
// walked the shelves with a clipboard, carried the paper to a computer and
// entered forty numbers a second time. A number written twice is wrong the
// second time, and the error lands in the one figure the whole module exists to
// produce — the variance — where it cannot be told apart from a shortfall.
//
// ⚠️ **The expected figure is never shown before a number is typed.** That rule
// is the point of a count: a sheet that says "there should be 9.4 kg" beside an
// empty box is a sheet that gets 9.4 written into it. It appears afterwards, as
// feedback, which is when it is useful and no longer suggestive.

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import { ApiError, api, getStaffToken } from "@/lib/api";
import { useStaff } from "@/lib/staff";
import { useAdminT } from "@/lib/i18n/admin";
import { formatDate } from "@/lib/format";
import type { StocktakeSheetRow, Warehouse } from "@/lib/types";

export default function StaffStockPage() {
  const router = useRouter();
  const { staff, loading } = useStaff();
  const t = useAdminT();
  const [stores, setStores] = useState<Warehouse[]>([]);
  const [store, setStore] = useState("");
  const [rows, setRows] = useState<StocktakeSheetRow[]>([]);
  const [since, setSince] = useState<string | null>(null);
  const [counted, setCounted] = useState<Record<string, string>>({});
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);

  const token = getStaffToken();

  useEffect(() => {
    if (!loading && !staff) router.replace("/staff/login");
  }, [loading, staff, router]);

  useEffect(() => {
    if (!token) return;
    api
      .staffWarehouses(token)
      .then((d) => setStores(d.warehouses))
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      );
  }, [token, t.common.loadFailed]);

  const loadSheet = useCallback(() => {
    if (!token) return;
    api
      .staffStocktakeSheet(token, store)
      .then((d) => {
        setRows(d.rows);
        setSince(d.since);
        setCounted({});
        setSaved(false);
      })
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      );
  }, [token, store, t.common.loadFailed]);

  useEffect(loadSheet, [loadSheet]);

  // Only what was actually typed is sent. ⚠️ A blank box is "not reached yet",
  // never zero: saving untouched rows as zero reads the next morning as a
  // catastrophic shortfall, and it is the classic way a half-finished count
  // gets saved anyway.
  const filled = useMemo(
    () =>
      rows
        .filter((r) => counted[r.ingredientId]?.trim())
        .map((r) => ({
          ingredientId: r.ingredientId,
          counted: Number(counted[r.ingredientId]),
          expected: r.expected,
        }))
        .filter((l) => Number.isFinite(l.counted)),
    [rows, counted],
  );

  const off = filled.some((l) => Math.abs(l.counted - l.expected) > 0.0001);

  async function save() {
    if (!token) return;
    setBusy(true);
    setError("");
    try {
      await api.staffSaveStocktake(token, {
        warehouseId: store || undefined,
        lines: filled.map((l) => ({
          ingredientId: l.ingredientId,
          counted: l.counted,
        })),
        note: note.trim(),
      });
      setSaved(true);
      loadSheet();
      setNote("");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  if (loading || !staff) return null;

  return (
    <div className="mx-auto w-full max-w-lg space-y-4 p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-lg font-semibold">{t.staffStock.title}</h1>
          <p className="mt-0.5 text-xs text-ink-muted">
            {since
              ? t.staffStock.since(formatDate(since))
              : t.staffStock.neverCounted}
          </p>
        </div>
        <button
          type="button"
          className="btn-ghost shrink-0 px-3 py-1.5 text-xs"
          onClick={() => router.push("/staff")}
        >
          {t.common.back}
        </button>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}
      {saved && <p className="text-sm text-ink-soft">{t.staffStock.saved}</p>}

      {/* ⚠️ A count is one room. Handing somebody walking into the bar a list
          that also holds forty kitchen ingredients is how counts get abandoned
          halfway — and a half-counted list saved is a shortfall report. */}
      {stores.length > 0 && (
        <label className="block text-sm">
          <span className="text-xs text-ink-muted">{t.staffStock.store}</span>
          <select
            className="input mt-1 w-full"
            value={store}
            onChange={(e) => setStore(e.target.value)}
          >
            <option value="">{t.staffStock.mainStore}</option>
            {stores.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </select>
        </label>
      )}

      <ul className="space-y-1.5">
        {rows.map((r) => {
          const typed = counted[r.ingredientId] ?? "";
          const n = Number(typed);
          const show = typed.trim() !== "" && Number.isFinite(n);
          const diff = show ? Math.round((n - r.expected) * 1000) / 1000 : 0;
          return (
            <li
              key={r.ingredientId}
              className="flex items-center gap-3 rounded-xl border border-line px-3 py-2"
            >
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm">{r.name}</div>
                {/* Feedback, and only after a number exists — see the note at
                    the top of this file. */}
                {show && (
                  <div className="text-xs text-ink-muted">
                    {t.staffStock.expected}: {r.expected}
                    {diff !== 0 && (
                      <span
                        className={
                          diff < 0
                            ? " ml-1.5 text-danger"
                            : " ml-1.5 text-amber-700 dark:text-amber-300"
                        }
                      >
                        {diff > 0 ? "+" : ""}
                        {diff}
                      </span>
                    )}
                  </div>
                )}
              </div>
              <input
                type="number"
                step="any"
                inputMode="decimal"
                // Big enough to hit with a thumb, on a phone held in a cold
                // store by somebody whose other hand is full.
                className="input w-24 py-2 text-right text-base"
                value={typed}
                onChange={(e) =>
                  setCounted({ ...counted, [r.ingredientId]: e.target.value })
                }
              />
              <span className="w-8 shrink-0 text-xs text-ink-muted">
                {t.ingredients.units[r.unit] ?? r.unit}
              </span>
            </li>
          );
        })}
      </ul>

      {rows.length === 0 && (
        <p className="py-8 text-center text-sm text-ink-muted">
          {t.staffStock.empty}
        </p>
      )}

      {filled.length > 0 && (
        <div className="sticky bottom-0 space-y-2 border-t border-line bg-surface pt-3 pb-4">
          {/* ⚠️ A difference cannot be saved silently — the cash drawer's
              rule, for the same reason: a variance nobody explained is one
              nobody can use, and the explanation only exists on the day. */}
          {off && (
            <input
              className="input w-full"
              placeholder={t.staffStock.notePlaceholder}
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          )}
          <button
            className="btn-primary w-full py-3"
            disabled={busy || (off && !note.trim())}
            onClick={save}
          >
            {t.staffStock.save(filled.length)}
          </button>
        </div>
      )}
    </div>
  );
}
