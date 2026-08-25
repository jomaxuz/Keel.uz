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
import type { Ingredient, WriteOff, WriteOffReason } from "@/lib/types";
import { QtyInput } from "@/components/QtyInput";
import { qtyNumber } from "@/lib/qty";

function today() {
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

/** The first of this month, in the restaurant's own day rather than UTC. */
function monthStart() {
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-01`;
}

export default function WriteOffsPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [rows, setRows] = useState<WriteOff[]>([]);
  const [value, setValue] = useState(0);
  const [reasons, setReasons] = useState<WriteOffReason[]>([]);
  // ⚠️ A period, defaulting to this month. Without one the question "how much
  // are the staff meals costing us" has no answer — a list of the last two
  // hundred entries spans whatever length of time it happens to span.
  const [from, setFrom] = useState(monthStart);
  const [to, setTo] = useState(today);
  const [ingredients, setIngredients] = useState<Ingredient[]>([]);
  const [at, setAt] = useState(today);
  const [ingredientId, setIngredientId] = useState("");
  const [qty, setQty] = useState("");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api
      .adminWriteOffs({ from, to })
      .then((d) => {
        setRows(d.writeOffs);
        setValue(d.value);
        setReasons(d.reasons ?? []);
      })
      .catch(() => setError(t.common.loadFailed));
    api
      .adminIngredients()
      .then((d) => setIngredients(d.ingredients))
      .catch(() => setIngredients([]));
  }, [t.common.loadFailed, from, to]);

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
        qty: qtyNumber(qty),
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
    picked && qtyNumber(qty) > 0
      ? Math.round(
          (picked.made
            ? (picked.rate ?? 0) * (picked.unit === "pcs" ? 1 : 1000)
            : picked.price) * qtyNumber(qty),
        )
      : 0;

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.writeoffs.title}</h1>
        <p className="mt-1 text-sm text-ink-soft">{t.writeoffs.intro}</p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      {/* The window everything below is measured over. ⚠️ Defaults to this
          month rather than "the last two hundred entries", which spans
          whatever length of time it happens to span — and a cost with no
          period attached is not a cost anybody can act on. */}
      <div className="flex flex-wrap items-center gap-2 text-sm">
        <span className="text-xs text-ink-muted">{t.writeoffs.period}</span>
        <input
          type="date"
          className="input w-auto"
          value={from}
          onChange={(e) => setFrom(e.target.value)}
        />
        <span className="text-ink-muted">—</span>
        <input
          type="date"
          className="input w-auto"
          value={to}
          onChange={(e) => setTo(e.target.value)}
        />
      </div>

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
            <QtyInput
              className="input mt-1 w-28"
              value={qty}
              onValue={setQty}
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
            disabled={busy || !ingredientId || !reason.trim() || !qtyNumber(qty)}
            onClick={save}
          >
            {t.common.save}
          </button>
        </div>
      </div>

      {/* ---- What it is costing, by reason ----

          ⚠️ **The number that changes behaviour is the split, not the total.**
          "3 200 000 thrown away this month" tells an owner something is wrong;
          "1 900 000 of it is staff meals" tells them what to do — and those are
          two different conversations, one with the kitchen and one with the
          rota. Costliest first: twelve spilled coffees and one ruined tray of
          meat make the same length of list and not the same problem. */}
      {reasons.length > 0 && (
        <div className="card p-3">
          <div className="flex flex-wrap items-end justify-between gap-2">
            <span className="text-sm font-medium">{t.writeoffs.byReason}</span>
            <span className="text-sm text-ink-soft">
              {t.writeoffs.total}:{" "}
              <span className="font-medium tabular-nums">
                {formatPrice(value)}
              </span>
            </span>
          </div>
          <ul className="mt-2 space-y-1">
            {reasons.map((r) => (
              <li key={r.reason} className="text-sm">
                <div className="flex justify-between gap-3">
                  <span className="min-w-0 truncate">
                    {r.reason}
                    <span className="ml-1.5 text-xs text-ink-muted">
                      ×{r.count}
                    </span>
                  </span>
                  <span className="shrink-0 tabular-nums">
                    {formatPrice(r.value)}
                  </span>
                </div>
                {/* A bar rather than a percentage: the question is "which of
                    these is the big one", and a row of numbers makes that
                    something to work out rather than something to see. */}
                <div className="mt-0.5 h-1 rounded-full bg-ink/[0.06]">
                  <div
                    className="h-1 rounded-full bg-ink/30"
                    style={{
                      width: `${value > 0 ? Math.max(2, Math.round((r.value / value) * 100)) : 0}%`,
                    }}
                  />
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}

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
