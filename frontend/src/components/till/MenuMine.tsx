"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import { api } from "@/lib/api";
import type { MenuGroup, MenuItem } from "@/lib/types";

/**
 * The person's own corner of the menu: what they pinned, and what they ring
 * most. Shared by the till and the floor screen.
 *
 * ⚠️ **Two chips before the categories, not a third screen.** They behave
 * exactly like a category — tap, and the grid shows those dishes — because a
 * cashier already knows how the category bar works and would not look for a
 * new control.
 *
 * ⚠️ **Kept on the server, per person** (handlers/tillfavorites.go): a waiter
 * moves between tablets during one evening.
 */
export const FAV_CAT = "__fav";
export const TOP_CAT = "__top";

export interface MenuMine {
  favorites: string[];
  top: string[];
}

export function useMenuMine(personKey: string) {
  const [mine, setMine] = useState<MenuMine>({ favorites: [], top: [] });
  // The last list the server accepted, to fall back to if a save fails.
  const saved = useRef<string[]>([]);

  useEffect(() => {
    setMine({ favorites: [], top: [] });
    saved.current = [];
    if (!personKey) return;
    let alive = true;
    api
      .tillMenuMine()
      .then((res) => {
        if (!alive) return;
        saved.current = res.favorites ?? [];
        setMine({ favorites: res.favorites ?? [], top: res.top ?? [] });
      })
      // Silent: the menu works without the shortlist, and a till offline
      // must not show an error for a convenience.
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, [personKey]);

  /** Pin or unpin — on screen at once, saved behind it. */
  const toggle = useCallback((id: string) => {
    setMine((cur) => {
      const has = cur.favorites.includes(id);
      const next = has
        ? cur.favorites.filter((x) => x !== id)
        : [...cur.favorites, id];
      api
        .tillSaveFavorites(next)
        .then((res) => {
          saved.current = res.favorites;
        })
        // ⚠️ Put back what the server still has: a star that stays lit after
        // a failed save is a favourite that is gone the next morning.
        .catch(() =>
          setMine((m) => ({ ...m, favorites: saved.current })),
        );
      return { ...cur, favorites: next };
    });
  }, []);

  return { mine, toggle };
}

/** The dishes behind one of the two chips, in the list's own order — or null
 *  when the category is an ordinary one. */
export function minePool(
  menu: MenuGroup[],
  catID: string | null,
  mine: MenuMine,
): MenuItem[] | null {
  if (catID !== FAV_CAT && catID !== TOP_CAT) return null;
  const byId = new Map<string, MenuItem>();
  for (const g of menu) for (const it of g.items) byId.set(it.id, it);
  const ids = catID === FAV_CAT ? mine.favorites : mine.top;
  return ids.map((id) => byId.get(id)).filter((x): x is MenuItem => !!x);
}
