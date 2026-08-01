"use client";

// Building a combo: pick dishes from this brand's menu and say how many of each.
//
// The set references dishes rather than copying them, so a renamed or reshot
// dish updates every combo it is in. Two consequences shown here:
//
//   • The "sold separately" total is added up live from the current prices, so
//     the owner can see at a glance whether the set still saves anything. A
//     combo priced above its parts is a mistake worth catching before saving.
//   • Dishes with a **required** option group cannot go in. "2 × Lag'mon" is an
//     instruction; "2 × Lag'mon (which size?)" is not, and a fixed set has
//     nowhere to ask. The server refuses these too — this only explains why.

import { useMemo, useState } from "react";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import type { ComboLine, MenuItem } from "@/lib/types";

export default function ComboEditor({
  value,
  price,
  menu,
  onChange,
}: {
  value: ComboLine[];
  /** The set's own price, for the live saving figure. */
  price: number;
  /** Everything on this brand's menu, to pick from. */
  menu: MenuItem[];
  onChange: (next: ComboLine[]) => void;
}) {
  const t = useAdminT();
  const [picking, setPicking] = useState(false);
  const [query, setQuery] = useState("");

  const byId = useMemo(() => {
    const map = new Map<string, MenuItem>();
    for (const m of menu) map.set(m.id, m);
    return map;
  }, [menu]);

  // What can go in: real dishes only — no other combos (a set inside a set has
  // no sensible price) and nothing with a required choice.
  const selectable = useMemo(() => {
    const chosen = new Set(value.map((l) => l.menuItemId));
    const q = query.trim().toLowerCase();
    return menu.filter((m) => {
      if (m.comboItems?.length) return false;
      if (chosen.has(m.id)) return false;
      if ((m.options ?? []).some((g) => g.required && g.choices?.length)) {
        return false;
      }
      return !q || m.name.toLowerCase().includes(q);
    });
  }, [menu, value, query]);

  const basePrice = value.reduce((sum, line) => {
    const dish = byId.get(line.menuItemId);
    return sum + (dish ? dish.price * Math.max(1, line.qty) : 0);
  }, 0);
  const saving = basePrice - price;

  function setQty(id: string, qty: number) {
    onChange(
      value.map((l) => (l.menuItemId === id ? { ...l, qty: Math.max(1, qty) } : l)),
    );
  }

  return (
    <div>
      <p className="text-sm text-ink-muted">{t.menu.comboHint}</p>

      {value.length > 0 && (
        <ul className="mt-3 divide-y divide-line rounded-2xl border border-line">
          {value.map((line) => {
            const dish = byId.get(line.menuItemId);
            return (
              <li
                key={line.menuItemId}
                className="flex items-center gap-3 px-3 py-2 text-sm"
              >
                <span className="min-w-0 flex-1 truncate">
                  {/* A dish deleted from the menu still has to be visible here —
                      it is why the combo stopped working. */}
                  {dish?.name ?? t.menu.comboMissingDish}
                </span>
                <span className="tabular-nums text-ink-muted">
                  {dish ? formatPrice(dish.price) : "—"}
                </span>
                <div className="flex items-center rounded-lg border border-line-strong">
                  <button
                    type="button"
                    onClick={() => setQty(line.menuItemId, line.qty - 1)}
                    className="px-2 py-1 text-ink-muted hover:text-brand"
                    aria-label="−"
                  >
                    −
                  </button>
                  <span className="w-7 text-center tabular-nums">{line.qty}</span>
                  <button
                    type="button"
                    onClick={() => setQty(line.menuItemId, line.qty + 1)}
                    className="px-2 py-1 text-ink-muted hover:text-brand"
                    aria-label="+"
                  >
                    +
                  </button>
                </div>
                <button
                  type="button"
                  onClick={() =>
                    onChange(value.filter((l) => l.menuItemId !== line.menuItemId))
                  }
                  className="text-ink-muted/70 hover:text-brand"
                  aria-label={t.common.delete}
                >
                  ✕
                </button>
              </li>
            );
          })}
        </ul>
      )}

      {/* The whole point of a set, in one line. */}
      {value.length > 0 && (
        <div className="mt-3 flex flex-wrap items-center justify-between gap-2 rounded-2xl bg-ink/5 px-3 py-2 text-sm">
          <span className="text-ink-muted">
            {t.menu.comboSeparately}: {formatPrice(basePrice)}
          </span>
          {saving > 0 ? (
            <span className="font-semibold text-emerald-600 dark:text-emerald-400">
              {t.menu.comboSaves(formatPrice(saving))}
            </span>
          ) : (
            <span className="font-semibold text-amber-700 dark:text-amber-300">
              {t.menu.comboNoSaving}
            </span>
          )}
        </div>
      )}

      {picking ? (
        <div className="mt-3 rounded-2xl border border-line p-3">
          <input
            className="input w-full text-sm"
            placeholder={t.menu.comboSearch}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            autoFocus
          />
          <ul className="mt-2 max-h-56 divide-y divide-line overflow-y-auto">
            {selectable.map((m) => (
              <li key={m.id}>
                <button
                  type="button"
                  onClick={() => {
                    onChange([...value, { menuItemId: m.id, qty: 1 }]);
                    setQuery("");
                    setPicking(false);
                  }}
                  className="flex w-full items-center justify-between gap-3 px-2 py-2 text-left text-sm hover:bg-ink/5"
                >
                  <span className="min-w-0 truncate">{m.name}</span>
                  <span className="shrink-0 tabular-nums text-ink-muted">
                    {formatPrice(m.price)}
                  </span>
                </button>
              </li>
            ))}
            {selectable.length === 0 && (
              <li className="px-2 py-3 text-sm text-ink-muted/70">
                {t.menu.comboNothingToAdd}
              </li>
            )}
          </ul>
          <button
            type="button"
            onClick={() => {
              setPicking(false);
              setQuery("");
            }}
            className="mt-2 text-sm text-ink-muted hover:text-ink"
          >
            {t.common.cancel}
          </button>
        </div>
      ) : (
        <button
          type="button"
          onClick={() => setPicking(true)}
          className="btn-ghost mt-3 px-4 py-2 text-sm"
        >
          {t.menu.comboAddDish}
        </button>
      )}
    </div>
  );
}
