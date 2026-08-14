"use client";

// "People usually order this with it."
//
// One component for three places — the dish page, the cart and the checkout —
// because the question is the same in all of them and only the seed changes.
// Three copies would have drifted into three different limits, three different
// empty states and, eventually, three different answers.
//
// ⚠️ **It renders nothing when there is nothing to say.** A restaurant whose
// history is too thin to produce a pairing, or whose suggestions are all sold
// out, gets no heading and no empty box — an empty section under a basket reads
// as a widget that failed, and it pushes the button the guest came for further
// down the screen.

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n/client";
import MenuItemCard from "@/components/menu/MenuItemCard";
import type { MenuItem } from "@/lib/types";

export default function Recommendations({
  itemIds,
  branchId,
  currency,
  title,
}: {
  /** The dish being looked at, or everything in the basket. */
  itemIds: string[];
  branchId?: string;
  currency: string;
  /** Overridden on the cart, where "goes with this dish" is the wrong words. */
  title?: string;
}) {
  const { t } = useI18n();
  const [items, setItems] = useState<MenuItem[]>([]);

  // ⚠️ Keyed on the ids as a string, not on the array. A new array with the same
  // contents is a new object every render, which would re-fetch on every
  // keystroke in the checkout form — on the one page where a guest is one tap
  // from paying.
  const key = itemIds.join(",");

  useEffect(() => {
    if (!key) {
      setItems([]);
      return;
    }
    let live = true;
    api
      .recommendations({ itemIds: key.split(","), branchId })
      .then((rows) => live && setItems(rows))
      // A failure here is silence, not an error message. This is a suggestion:
      // nothing the guest asked for is missing, and an error box under the
      // basket would look like something went wrong with their order.
      .catch(() => live && setItems([]))
      ;
    return () => {
      live = false;
    };
  }, [key, branchId]);

  // Sold-out dishes are dropped rather than greyed out. On the menu a greyed
  // card answers "do they have it?" — a question the guest asked. Here nobody
  // asked, so an unbuyable suggestion is pure noise.
  const shown = items.filter((x) => !x.soldOut);
  if (shown.length === 0) return null;

  return (
    <section className="mt-8">
      <h2 className="section-title">{title ?? t.recommend.title}</h2>
      <div className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        {shown.map((item) => (
          <MenuItemCard key={item.id} item={item} currency={currency} />
        ))}
      </div>
    </section>
  );
}
