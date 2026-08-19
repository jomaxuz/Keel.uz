"use client";

// The shopping list: what the kitchen buys and what it pays.
//
// ⚠️ **One price, in the unit it is bought in.** An invoice says "one kilo,
// 90 000" — nobody has a price per gram written anywhere, and a form asking for
// one is a form that gets the wrong number. The conversion to grams happens in
// the tech card, where the recipe is written.
//
// ⚠️ **This is costing, not stock.** There is no quantity on hand here and
// there deliberately is not: a restaurant that believes a stock figure and
// finds it wrong stops believing the panel entirely. What this list answers is
// "what does a portion cost", and the answer updates itself the day a price
// changes — which is the whole reason it exists.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll } from "@/components/admin/PagedList";
import type { Ingredient } from "@/lib/types";

const UNITS = ["kg", "l", "pcs"] as const;

type Draft = Partial<Ingredient> & {
  name: string;
  unit: string;
  price: number;
};

const EMPTY: Draft = { name: "", unit: "kg", price: 0 };

export default function IngredientsPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [rows, setRows] = useState<Ingredient[]>([]);
  const [draft, setDraft] = useState<Draft>(EMPTY);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    setLoading(true);
    api
      .adminIngredients()
      .then(setRows)
      .catch(() => setError(t.common.loadFailed))
      .finally(() => setLoading(false));
  }, [t.common.loadFailed]);

  useEffect(load, [load, scope.scopeKey]);

  async function save() {
    if (!draft.name.trim()) return;
    setBusy(true);
    setError("");
    try {
      await api.adminSaveIngredient({
        id: draft.id,
        name: draft.name.trim(),
        unit: draft.unit,
        price: Math.max(0, Math.round(draft.price) || 0),
        note: draft.note ?? "",
      });
      setDraft(EMPTY);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  async function remove(row: Ingredient) {
    setBusy(true);
    setError("");
    try {
      await api.adminDeleteIngredient(row.id);
      load();
    } catch (e) {
      // ⚠️ The refusal names the dishes that still hold it. A recipe pointing
      // at a deleted ingredient would cost nothing at all, which does not look
      // like an error — the dish simply becomes cheaper to make, on every
      // report, until somebody notices.
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.ingredients.title}</h1>
        <p className="mt-1 text-sm text-ink-soft">{t.ingredients.intro}</p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      <div className="card space-y-2 p-3">
        <div className="flex flex-wrap items-end gap-2">
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.ingredients.name}</span>
            <input
              className="input mt-1 w-56"
              value={draft.name}
              onChange={(e) => setDraft({ ...draft, name: e.target.value })}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.ingredients.unit}</span>
            <select
              className="input mt-1 w-28"
              value={draft.unit}
              onChange={(e) => setDraft({ ...draft, unit: e.target.value })}
            >
              {UNITS.map((u) => (
                <option key={u} value={u}>
                  {t.ingredients.units[u]}
                </option>
              ))}
            </select>
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">
              {t.ingredients.pricePer(t.ingredients.units[draft.unit as "kg"])}
            </span>
            <input
              type="number"
              min={0}
              className="input mt-1 w-40"
              value={draft.price || ""}
              onChange={(e) =>
                setDraft({ ...draft, price: Number(e.target.value) || 0 })
              }
            />
          </label>
          <label className="block flex-1 text-sm">
            <span className="text-xs text-ink-muted">{t.ingredients.note}</span>
            <input
              className="input mt-1 w-full"
              placeholder={t.ingredients.notePlaceholder}
              value={draft.note ?? ""}
              onChange={(e) => setDraft({ ...draft, note: e.target.value })}
            />
          </label>
          <button
            className="btn-primary px-4 py-2"
            disabled={busy || !draft.name.trim()}
            onClick={save}
          >
            {draft.id ? t.common.save : t.common.add}
          </button>
          {draft.id && (
            <button
              className="btn-ghost px-3 py-2"
              onClick={() => setDraft(EMPTY)}
            >
              {t.common.cancel}
            </button>
          )}
        </div>
      </div>

      <div className="card p-0">
        <ListScroll>
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-surface text-left text-xs text-ink-muted">
              <tr>
                <th className="px-3 py-2">{t.ingredients.name}</th>
                <th className="px-3 py-2">{t.ingredients.unit}</th>
                <th className="px-3 py-2 text-right">{t.ingredients.price}</th>
                <th className="px-3 py-2">{t.ingredients.note}</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.id} className="border-t border-line">
                  <td className="px-3 py-2 font-medium">{row.name}</td>
                  <td className="px-3 py-2 text-ink-soft">
                    {t.ingredients.units[row.unit as "kg"] ?? row.unit}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {formatPrice(row.price)}
                  </td>
                  <td className="px-3 py-2 text-ink-muted">{row.note}</td>
                  <td className="px-3 py-2 text-right whitespace-nowrap">
                    <button
                      className="btn-ghost px-2 py-1 text-xs"
                      onClick={() => setDraft(row)}
                    >
                      {t.common.edit}
                    </button>
                    <button
                      className="btn-ghost px-2 py-1 text-xs text-danger"
                      disabled={busy}
                      onClick={() => remove(row)}
                    >
                      {t.common.delete}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {!loading && rows.length === 0 && (
            <p className="p-6 text-center text-sm text-ink-muted">
              {t.ingredients.empty}
            </p>
          )}
        </ListScroll>
      </div>
    </div>
  );
}
