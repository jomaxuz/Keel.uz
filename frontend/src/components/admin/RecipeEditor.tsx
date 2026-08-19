"use client";

// The tech card: what goes into one portion, and what that costs today.
//
// ⚠️ **This replaces a number that was correct on the day it was typed.** A
// hand-entered cost is right in March and silently wrong from the next delivery
// onwards, and nothing on any screen says so. A card moves the fact to where it
// actually changes — meat goes up once, and every dish containing meat is
// dearer the same afternoon.
//
// ⚠️ **Quantities are brutto**: what leaves the store to make the dish, not
// what ends up on the plate. A kilo of potatoes costs a kilo whether or not a
// third of it is peel, and costing the peeled weight understates every dish the
// card describes.

import { useMemo, useState } from "react";

import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import type { Ingredient, RecipeLine } from "@/lib/types";

/** Grams for a kilo, millilitres for a litre, pieces for pieces — the whole of
 *  the unit handling, in one place, exactly as the server has it. */
function recipeUnit(unit: string): string {
  if (unit === "kg") return "g";
  if (unit === "l") return "ml";
  return "pcs";
}

function ratePerUnit(ing: Ingredient): number {
  const per = ing.unit === "kg" || ing.unit === "l" ? 1000 : 1;
  return ing.price / per;
}

export default function RecipeEditor({
  lines,
  ingredients,
  price,
  onChange,
}: {
  lines: RecipeLine[];
  ingredients: Ingredient[];
  /** What the dish sells for, so the margin can be shown while it is typed. */
  price: number;
  onChange: (next: RecipeLine[]) => void;
}) {
  const t = useAdminT();
  const [picked, setPicked] = useState("");

  const byId = useMemo(() => {
    const m = new Map<string, Ingredient>();
    for (const i of ingredients) m.set(i.id, i);
    return m;
  }, [ingredients]);

  // ⚠️ Carried as a rate and rounded once, exactly as the server does it. A
  // panel that rounded each line would show a different cost from every report.
  const cost = Math.round(
    lines.reduce((sum, l) => {
      const ing = byId.get(l.ingredientId);
      return ing ? sum + ratePerUnit(ing) * l.qty : sum;
    }, 0),
  );
  const missing = lines.some((l) => !byId.has(l.ingredientId));

  function set(i: number, qty: number) {
    const next = lines.slice();
    next[i] = { ...next[i], qty };
    onChange(next);
  }

  return (
    <div className="space-y-2">
      {lines.length > 0 && (
        <ul className="space-y-1">
          {lines.map((l, i) => {
            const ing = byId.get(l.ingredientId);
            return (
              <li key={l.ingredientId} className="flex items-center gap-2">
                <span className="min-w-0 flex-1 truncate text-sm">
                  {/* ⚠️ A line whose ingredient is gone is named as broken
                      rather than dropped: the card is not complete, and the
                      dish is not being costed while it looks like it is. */}
                  {ing ? ing.name : t.recipe.missingIngredient}
                </span>
                <input
                  type="number"
                  min={0}
                  step="any"
                  className="input w-24 py-1 text-right"
                  value={l.qty}
                  onChange={(e) => set(i, Number(e.target.value) || 0)}
                />
                <span className="w-8 text-xs text-ink-muted">
                  {ing ? recipeUnit(ing.unit) : ""}
                </span>
                <span className="w-24 text-right text-xs text-ink-muted tabular-nums">
                  {ing
                    ? formatPrice(Math.round(ratePerUnit(ing) * l.qty))
                    : "—"}
                </span>
                <button
                  type="button"
                  className="btn-ghost px-2 py-1 text-xs"
                  onClick={() => onChange(lines.filter((_, j) => j !== i))}
                >
                  ✕
                </button>
              </li>
            );
          })}
        </ul>
      )}

      <div className="flex flex-wrap items-center gap-2">
        <select
          className="input w-auto py-1"
          value={picked}
          onChange={(e) => setPicked(e.target.value)}
        >
          <option value="">{t.recipe.addLine}</option>
          {ingredients
            .filter((i) => !lines.some((l) => l.ingredientId === i.id))
            .map((i) => (
              <option key={i.id} value={i.id}>
                {i.name} — {formatPrice(i.price)}/{i.unit}
              </option>
            ))}
        </select>
        <button
          type="button"
          className="btn px-3 py-1 text-sm"
          disabled={!picked}
          onClick={() => {
            onChange([...lines, { ingredientId: picked, qty: 0 }]);
            setPicked("");
          }}
        >
          {t.common.add}
        </button>
        {ingredients.length === 0 && (
          <span className="text-xs text-ink-muted">
            {t.recipe.noIngredients}
          </span>
        )}
      </div>

      {lines.length > 0 && (
        <div className="rounded-xl bg-ink/[0.03] px-3 py-2 text-sm">
          <div className="flex justify-between">
            <span>{t.recipe.cost}</span>
            <span className="font-medium tabular-nums">
              {formatPrice(cost)}
            </span>
          </div>
          {/* The margin while the card is being typed: this is the moment the
              price is being decided, and the number it turns on is right here. */}
          {price > 0 && cost > 0 && !missing && (
            <div className="mt-0.5 flex justify-between text-xs text-ink-muted">
              <span>{t.recipe.margin}</span>
              <span
                className={`tabular-nums ${price <= cost ? "text-danger" : ""}`}
              >
                {price > cost
                  ? `${formatPrice(price - cost)} · ${Math.round(((price - cost) / price) * 100)}%`
                  : t.menu.belowCost}
              </span>
            </div>
          )}
          {missing && (
            <p className="mt-1 text-xs text-danger">{t.recipe.incomplete}</p>
          )}
        </div>
      )}
    </div>
  );
}
