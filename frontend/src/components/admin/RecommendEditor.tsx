"use client";

// "Suggest these alongside it" — the owner's own picks.
//
// ⚠️ **This exists because the automatic half cannot promote anything new.**
// What sells together is counted from the order history, and the history knows
// it far better than anybody in the restaurant does — but a dish added on
// Monday has no history at all, and the thing an owner most wants to push is
// exactly the thing nobody has ordered yet. An automatic-only feature would
// recommend the dishes that already sell, and only those.
//
// So: empty is the normal state and means "work it out from the orders". A
// filled list is a decision, and it comes first on the page.

import { useMemo, useState } from "react";
import { useAdminT } from "@/lib/i18n/admin";
import { usePanelWords } from "@/lib/panelWords";
import type { MenuItem } from "@/lib/types";

export default function RecommendEditor({
  value,
  menu,
  onChange,
}: {
  value: string[];
  /** The menu to pick from. The caller filters out the dish being edited — a
   *  dish that recommends itself is the easiest mistake here to make, and the
   *  server refuses it too. */
  menu: MenuItem[];
  onChange: (ids: string[]) => void;
}) {
  const t = useAdminT();
  const w = usePanelWords();
  const [q, setQ] = useState("");

  const byId = useMemo(() => new Map(menu.map((m) => [m.id, m])), [menu]);
  const picked = value.filter((id) => byId.has(id));

  const matches = useMemo(() => {
    const needle = q.trim().toLowerCase();
    if (!needle) return [];
    return menu
      .filter((m) => !value.includes(m.id) && m.name.toLowerCase().includes(needle))
      .slice(0, 8);
  }, [q, menu, value]);

  const r = t.menu.recommend;

  return (
    <div className="rounded-2xl border border-line p-3">
      <p className="text-sm font-medium">{w.recommendTitle}</p>
      <p className="mt-0.5 text-xs text-ink-muted">{r.hint}</p>

      {picked.length > 0 && (
        <ul className="mt-3 space-y-1">
          {picked.map((id, i) => (
            <li
              key={id}
              className="flex items-center gap-2 rounded-xl border border-line px-2 py-1.5 text-sm"
            >
              <span className="flex-1 truncate">{byId.get(id)?.name}</span>
              {/* The order is the owner's priority — the first one is what they
                  most want offered — so it has to be editable. */}
              <button
                type="button"
                onClick={() => onChange(swap(value, i, i - 1))}
                disabled={i === 0}
                aria-label={r.up}
                className="rounded-lg border border-line px-2 text-xs text-ink-soft disabled:opacity-30"
              >
                ↑
              </button>
              <button
                type="button"
                onClick={() => onChange(swap(value, i, i + 1))}
                disabled={i === picked.length - 1}
                aria-label={r.down}
                className="rounded-lg border border-line px-2 text-xs text-ink-soft disabled:opacity-30"
              >
                ↓
              </button>
              <button
                type="button"
                onClick={() => onChange(value.filter((x) => x !== id))}
                aria-label={r.remove}
                className="rounded-lg px-2 text-xs text-brand"
              >
                ✕
              </button>
            </li>
          ))}
        </ul>
      )}

      {/* A search box rather than a select. A restaurant menu is a hundred
          dishes; a dropdown of a hundred is one nobody scrolls to the bottom
          of, and the dish they want is usually one they can name. */}
      <input
        className="input mt-3"
        value={q}
        placeholder={w.recommendSearch}
        onChange={(e) => setQ(e.target.value)}
      />
      {matches.length > 0 && (
        <ul className="mt-1 divide-y divide-line rounded-xl border border-line">
          {matches.map((m) => (
            <li key={m.id}>
              <button
                type="button"
                onClick={() => {
                  onChange([...value, m.id]);
                  setQ("");
                }}
                className="w-full px-3 py-2 text-left text-sm hover:bg-ink/5"
              >
                {m.name}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function swap(list: string[], a: number, b: number): string[] {
  if (b < 0 || b >= list.length) return list;
  const next = [...list];
  [next[a], next[b]] = [next[b], next[a]];
  return next;
}
