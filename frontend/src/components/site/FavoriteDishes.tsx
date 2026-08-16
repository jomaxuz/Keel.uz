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

import { useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n/client";
import { usePaged } from "@/lib/paged";
import { useFavorites } from "@/lib/favorites";
import MenuItemCard from "@/components/menu/MenuItemCard";
import Pager from "@/components/site/Pager";
import type { MenuItem } from "@/lib/types";

/**
 * Cards per page. Six fills two rows on a phone and two on the desktop grid,
 * so the section keeps a stable height rather than growing a row with every
 * heart the guest taps.
 */
const PAGE_SIZE = 6;

/** Stable identity while loading, so the slice memo is not thrown away every render. */
const NONE: MenuItem[] = [];

export default function FavoriteDishes({ currency }: { currency: string }) {
  const { t } = useI18n();
  // The id set is what changes when a heart is tapped elsewhere on the page, so it is
  // what this list re-reads on.
  const { ids } = useFavorites();
  const [items, setItems] = useState<MenuItem[] | null>(null);
  const headingRef = useRef<HTMLHeadingElement>(null);
  // `items` is null while loading; the hook must still run every render.
  const paged = usePaged(items ?? NONE, PAGE_SIZE);

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
      <h2 ref={headingRef} className="scroll-mt-24 font-display text-lg font-bold">
        {t.contact.favorites}
      </h2>
      {items.length === 0 ? (
        <p className="mt-2 text-sm text-ink-muted">{t.contact.favoritesEmpty}</p>
      ) : (
        <>
          <div className="mt-4 grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-3">
            {paged.pageItems.map((item) => (
              <MenuItemCard key={item.id} item={item} currency={currency} />
            ))}
          </div>
          <Pager
            className="mt-4"
            page={paged.page}
            pageCount={paged.pageCount}
            from={paged.from}
            to={paged.to}
            total={paged.total}
            onPage={paged.setPage}
            anchorRef={headingRef}
          />
        </>
      )}
    </section>
  );
}
