"use client";

// The dishes a guest saved, on their profile.
//
// ⚠️ **Fetched from the server rather than read from the heart provider.** The provider
// holds ids, and a card needs a name, a price and a photograph — and those change: a dish
// renamed or repriced since the heart was tapped has to render as it is now, not as it
// was. It is also the one screen where a favourite that no longer exists should simply be
// absent, which the server already handles by returning only what it can find.
//
// Hidden entirely when there is nothing saved and nothing to explain — except the first
// time, where the empty state says how the hearts get there. An empty section with no
// sentence is a section that reads as broken.

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n/client";
import { useFavorites } from "@/lib/favorites";
import MenuItemCard from "@/components/menu/MenuItemCard";
import type { MenuItem } from "@/lib/types";

export default function FavoriteDishes({ currency }: { currency: string }) {
  const { t } = useI18n();
  // The id set is what changes when a heart is tapped elsewhere on the page, so it is
  // what this list re-reads on.
  const { ids } = useFavorites();
  const [items, setItems] = useState<MenuItem[] | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .favorites()
      .then((rows) => !cancelled && setItems(rows))
      .catch(() => !cancelled && setItems([]));
    return () => {
      cancelled = true;
    };
  }, [ids.size]);

  if (items === null) return null;

  return (
    <section className="mt-8 rounded-3xl border border-line bg-surface p-6 shadow-card">
      <h2 className="font-display text-lg font-bold">{t.contact.favorites}</h2>
      {items.length === 0 ? (
        <p className="mt-2 text-sm text-ink-muted">{t.contact.favoritesEmpty}</p>
      ) : (
        <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {items.map((item) => (
            <MenuItemCard key={item.id} item={item} currency={currency} />
          ))}
        </div>
      )}
    </section>
  );
}
