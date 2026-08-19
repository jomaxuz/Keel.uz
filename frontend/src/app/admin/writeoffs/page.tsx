"use client";

// Food that left without being sold.
//
// ⚠️ **This is the half of the flow nobody records, and it is why the
// difference column could not be read.** Food leaves a restaurant three ways:
// sold, eaten by the staff, thrown away. The tech cards account for the first;
// until the other two are written down, the gap between what came in and what
// was sold could be waste, theft, or a delivery still in the fridge — and
// whoever reads it has to guess.
//
// ⚠️ **A reason is required**, as everywhere else here where something
// disappears: a void, a refund, a cancelled order. "Twelve kilos of beef,
// written off" with no sentence beside it is the line every argument starts
// from, and by then the person who could answer has gone home.

import { useCallback, useEffect, useMemo, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDate, formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll } from "@/components/admin/PagedList";
import type { Ingredient, WriteOff } from "@/lib/types";

function today() {
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

export default function WriteOffsPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [rows, setRows] = useState<WriteOff[]>([]);
  const [value, setValue] = useState(0);
  const [ingredients, setIngredients] = useState<Ingredient[]>([]);
  const [at, setAt] = useState(today);
  const [ingredientId, setIngredientId] = useState("");
  const [qty, setQty] = useState("");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api
      .adminWriteOffs()
      .then((d) => {
        setRows(d.writeOffs);
        setValue(d.value);
      })
      .catch(() => setError(t.common.loadFailed));
    api
      .adminIngredients()
      .then((d) => setIngredients(d.ingredients))
      .catch(() => setIngredients([]));
  }, [t.common.loadFailed]);

  useEffect(load, [load, scope.scopeKey]);

  const byId = useMemo(() => {
    const m = new Map<string, Ingredient>();
    for (const i of ingredients) m.set(i.id, i);
    return m;
  }, [ingredients]);

  async function save() {
    setBusy(true);
    setError("");
    try {
      await api.adminCreateWriteOff({
        at: new Date(`${at}T12:00:00`).toISOString(),
        ingredientId,
        qty: Number(qty) || 0,
        reason: reason.trim(),
      });
      setQty("");
      setReason("");
      setIngredientId("");
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  const picked = byId.get(ingredientId);
  // What it will be worth, before it is saved: the number that makes somebody
  // stop and check the quantity they just typed.
  const preview =
    picked && Number(qty) > 0
      ? Math.round(
          (picked.made
            ? (picked.rate ?? 0) * (picked.unit === "pcs" ? 1 : 1000)
            : picked.price) * Number(qty),
        )
      : 0;

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.writeoffs.title}</h1>
        <p className="mt-1 text-sm text-ink-soft">{t.writeoffs.intro}</p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      <div className="card p-3">
        <div className="flex flex-wrap items-end gap-2">
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.writeoffs.date}</span>
            <input
              type="date"
              className="input mt-1 w-auto"
              value={at}
              onChange={(e) => setAt(e.target.value)}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">
              {t.writeoffs.ingredient}
            </span>
            <select
              className="input mt-1 w-56"
              value={ingredientId}
              onChange={(e) => setIngredientId(e.target.value)}
            >
              <option value="">—</option>
              {ingredients.map((i) => (
                <option key={i.id} value={i.id}>
                  {i.name}
                </option>
              ))}
            </select>
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">
              {t.writeoffs.qty}
              {picked
                ? ` (${t.ingredients.units[picked.unit] ?? picked.unit})`
                : ""}
            </span>
            <input
              type="number"
              min={0}
              step="any"
              className="input mt-1 w-28"
              value={qty}
              onChange={(e) => setQty(e.target.value)}
            />
          </label>
          <label className="block flex-1 text-sm">
            <span className="text-xs text-ink-muted">{t.writeoffs.reason}</span>
            <input
              className="input mt-1 w-full"
              placeholder={t.writeoffs.reasonPlaceholder}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
          </label>
          {preview > 0 && (
            <span className="pb-2 text-sm text-ink-soft">
              ≈ {formatPrice(preview)}
            </span>
          )}
          <button
            className="btn-primary px-4 py-2"
            disabled={busy || !ingredientId || !reason.trim() || !Number(qty)}
            onClick={save}
          >
            {t.common.save}
          </button>
        </div>
      </div>

      <div className="card p-0">
        <ListScroll>
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-surface text-left text-xs text-ink-muted">
              <tr>
                <th className="px-3 py-2">{t.writeoffs.date}</th>
                <th className="px-3 py-2">{t.writeoffs.ingredient}</th>
                <th className="px-3 py-2 text-right">{t.writeoffs.qty}</th>
                <th className="px-3 py-2">{t.writeoffs.reason}</th>
                <th className="px-3 py-2 text-right">{t.writeoffs.value}</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {rows.map((x) => (
                <tr key={x.id} className="border-t border-line">
                  <td className="px-3 py-2 whitespace-nowrap">
                    {formatDate(x.at)}
                  </td>
                  <td className="px-3 py-2">
                    {byId.get(x.ingredientId)?.name ?? "—"}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">{x.qty}</td>
                  <td className="px-3 py-2 text-ink-soft">
                    {x.reason}
                    {/* Who wrote it off. The reason explains the food; the
                        name is what makes the record worth keeping. */}
                    {x.by && (
                      <span className="ml-1 text-xs text-ink-muted">
                        · {x.by}
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {formatPrice(x.value)}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <button
                      className="btn-ghost px-2 py-1 text-xs text-danger"
                      disabled={busy}
                      onClick={async () => {
                        setBusy(true);
                        try {
                          await api.adminDeleteWriteOff(x.id);
                          load();
                        } finally {
                          setBusy(false);
                        }
                      }}
                    >
                      {t.common.delete}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {rows.length === 0 && (
            <p className="p-6 text-center text-sm text-ink-muted">
              {t.writeoffs.empty}
            </p>
          )}
        </ListScroll>
        {rows.length > 0 && (
          <div className="border-t border-line px-3 py-2 text-right text-sm">
            {t.writeoffs.total}:{" "}
            <span className="font-medium tabular-nums">
              {formatPrice(value)}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
