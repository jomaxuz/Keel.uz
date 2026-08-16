"use client";

import { useEffect, useState } from "react";
// Locale-aware: hrefs stay unprefixed here and gain /ru or /en at render.
import Link from "@/components/site/LocaleLink";
import FavoriteButton from "@/components/menu/FavoriteButton";
import { imageUrl } from "@/lib/api";
import { useCart } from "@/lib/cart";
import { formatPrice } from "@/lib/format";
import { useI18n } from "@/lib/i18n/client";
import { contentDescription, contentName } from "@/lib/i18n/content";
import type { MenuItem } from "@/lib/types";

export default function MenuItemCard({
  item,
  currency,
}: {
  item: MenuItem;
  currency: string;
}) {
  const { add, lines, setQty } = useCart();
  const { lang, t } = useI18n();
  const img = imageUrl(item.imageUrl, 600);
  // Dishes with option groups cannot be added in one tap — the customer picks
  // the variant on the dish page. Without options the line id is just the id.
  const hasOptions = (item.options ?? []).some((g) => g.choices?.length);

  // The cart is read from localStorage, so the server has no idea what is in
  // it: it always renders the "add" button, while the browser may already know
  // this dish is in the basket and want the quantity stepper. Two different
  // trees for the same markup is a hydration mismatch.
  //
  // The flag is **this component's own**, not the cart provider's. The header
  // badge does the same thing for the same reason (see the hydration note in
  // CLAUDE.md): a provider's own `mounted` has already flipped by the time a
  // component inside <Suspense> hydrates, so leaning on it proves nothing.
  const [hydrated, setHydrated] = useState(false);
  useEffect(() => setHydrated(true), []);
  const stored = lines.find((l) => l.lineId === item.id)?.qty ?? 0;
  const qty = hydrated ? stored : 0;
  const name = contentName(item, lang);
  const description = contentDescription(item, lang);
  const hasDiscount = item.oldPrice != null && item.oldPrice > item.price;
  // Two different "no": the dish is off the menu, or the branch has run out of
  // it today. Both stop the button; only the wording differs.
  const orderable = item.isAvailable && !item.soldOut;
  // A combo is worth showing only when it actually saves something: a set
  // priced at or above its parts has no story to tell, and a "0% saving" badge
  // reads as a mistake.
  const saving =
    item.comboBasePrice && item.comboBasePrice > item.price
      ? item.comboBasePrice - item.price
      : 0;

  return (
    <div className="group card relative flex flex-col overflow-hidden transition-all duration-300 hover:-translate-y-1 hover:shadow-card-hover">
      {/* ⚠️ A direct child of the card, not of the image's own wrapper: the wrapper is
          inside the link, and anything inside it takes the navigation with it however
          many clicks are stopped. The card is the positioning context. */}
      <FavoriteButton id={item.id} />
      <Link href={`/menu/${item.id}`} className="block">
        <div className="relative aspect-[4/3] w-full overflow-hidden bg-ink/5">
          {img ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={img}
              alt={name}
              loading="lazy"
              className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-[1.07]"
            />
          ) : (
            <div className="flex h-full w-full items-center justify-center text-sm text-ink-muted">
              {t.common.noImage}
            </div>
          )}

          <div className="absolute left-3 top-3 flex flex-col items-start gap-1.5">
            {item.isPopular && <span className="badge-brand">★ {t.item.popular}</span>}
            {hasDiscount && (
              <span className="badge bg-ink text-cream">{t.item.discount}</span>
            )}
            {saving > 0 && (
              <span className="badge bg-emerald-600 text-white">
                {t.item.comboSaving(formatPrice(saving, currency, lang))}
              </span>
            )}
          </div>
          {!orderable && (
            <div className="absolute inset-0 flex items-center justify-center bg-surface/70 backdrop-blur-[2px]">
              <span className="badge bg-ink text-cream">
                {item.soldOut && item.isAvailable
                  ? t.item.soldOut
                  : t.item.unavailable}
              </span>
            </div>
          )}
        </div>
      </Link>

      <div className="flex flex-1 flex-col p-3 sm:p-5">
        <Link href={`/menu/${item.id}`}>
          <h3 className="font-display text-base font-bold leading-snug transition-colors group-hover:text-brand sm:text-lg">
            {name}
          </h3>
        </Link>
        {/* What is in the set. Without it "Oilaviy combo — 60 000" tells the
            guest nothing they can decide on. */}
        {item.comboContents?.length ? (
          <ul className="mt-1.5 space-y-0.5 text-xs text-ink-muted sm:text-sm">
            {item.comboContents.map((c, i) => (
              <li key={i} className="truncate">
                {c.qty > 1 ? `${c.name} × ${c.qty}` : c.name}
              </li>
            ))}
          </ul>
        ) : (
          description && (
            <p className="mt-1.5 line-clamp-2 text-xs leading-relaxed text-ink-muted sm:text-sm">
              {description}
            </p>
          )
        )}

        {/* ⚠️ Wraps only when it has to, and only on phones.
            This row used to be `flex` with no wrap and a `min-w-0` price, on the
            theory that "the price shrinks — it is the part that can". It cannot:
            nothing in it wraps and nothing clipped it, so the price simply
            painted outside its box and the button, later in the DOM, was drawn
            on top of it. Measured at 360px every card overflowed, and a
            discounted dish put its old price 61px underneath the button.
            The arithmetic is the real constraint: in two phone columns the row
            is 132px, the icon button is 40 and the gap 8, which leaves 84px for
            a price that needs 92 — and more for a six-figure one. So on phones
            the button drops to its own line when the price would not fit, which
            costs height on exactly the cards that need it and none of the
            others.

            ⚠️ **Nothing here may clip, and `overflow-hidden` is what made it.**
            A flex item whose overflow is not `visible` has an automatic minimum
            size of zero — so the clip added as a "last-resort guarantee" was
            itself permission to shrink, and the row stopped wrapping and started
            cutting instead. At exactly 640px, where the price steps up to 20px
            (and, back then, the button also grew a "Qo'shish" label),
            "145 000 so'm" came out as "145 000 so'". Four pixels, and it reads
            as a broken site.

            The add button is now a fixed 40px circle at every width, which only
            widens the margin this row had — but the geometry below is what
            guarantees it, not the button staying that size.

            The guarantee is geometry now, not a clip: the block cannot shrink
            below the current price, so when it does not fit the row wraps — and
            a wrapped row cannot overlap. The struck-out price is the one thing
            allowed to give way, inside the block, because it is the part the
            guest can lose. */}
        <div className="mt-3 flex flex-wrap items-end justify-between gap-2 pt-1 sm:mt-4 sm:gap-3">
          <div className="flex items-baseline gap-1.5 leading-tight sm:gap-2">
            {/* 14px on phones, not 16: at 16 the ordinary five-figure price
                needs 92px of the 84 the two-column card has, so every card
                wrapped and every card got taller. One step down buys 12px and
                leaves wrapping to the six-figure prices it was meant for. */}
            <span className="shrink-0 whitespace-nowrap font-display text-sm font-bold sm:text-xl">
              {formatPrice(item.price, currency, lang)}
            </span>
            {hasDiscount && (
              <span className="min-w-0 truncate text-xs text-ink-muted line-through sm:text-sm">
                {formatPrice(item.oldPrice!, currency, lang)}
              </span>
            )}
            {saving > 0 && (
              <span className="min-w-0 truncate text-xs text-ink-muted line-through sm:text-sm">
                {formatPrice(item.comboBasePrice!, currency, lang)}
              </span>
            )}
          </div>
          {/* Once in the cart the button turns into a −/qty/+ stepper. */}
          {hasOptions ? (
            <Link
              href={`/menu/${item.id}`}
              aria-disabled={!orderable}
              className={`btn-primary ml-auto shrink-0 px-3.5 py-2 ${
                orderable ? "" : "pointer-events-none opacity-50"
              }`}
            >
              {t.item.choose}
            </Link>
          ) : qty === 0 ? (
            <button
              type="button"
              onClick={() => add(item)}
              disabled={!orderable}
              aria-label={`${name} — ${t.item.addToCart}`}
              // ⚠️ A plus at every width, with no label beside it even where
              // there is room. The word is the same word on every card in a
              // grid of twenty, which makes it decoration rather than
              // information — the icon already says it, in every language, and
              // the dish name is what the guest is reading. The accessible name
              // still spells it out, and names the dish while it is at it.
              className="btn-primary btn-icon ml-auto"
            >
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.5"
                strokeLinecap="round"
                className="h-4 w-4"
                aria-hidden
              >
                <path d="M12 5v14M5 12h14" />
              </svg>
            </button>
          ) : (
            <div className="ml-auto flex shrink-0 items-center gap-1 rounded-full bg-brand p-1 text-white">
              <button
                type="button"
                onClick={() => setQty(item.id, qty - 1)}
                aria-label={t.item.decrease}
                className="flex h-7 w-7 items-center justify-center rounded-full transition-colors hover:bg-white/20"
              >
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2.5"
                  strokeLinecap="round"
                  className="h-4 w-4"
                  aria-hidden
                >
                  <path d="M5 12h14" />
                </svg>
              </button>
              <span className="min-w-6 text-center text-sm font-bold tabular-nums">
                {qty}
              </span>
              <button
                type="button"
                onClick={() => add(item)}
                aria-label={t.item.increase}
                className="flex h-7 w-7 items-center justify-center rounded-full transition-colors hover:bg-white/20"
              >
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2.5"
                  strokeLinecap="round"
                  className="h-4 w-4"
                  aria-hidden
                >
                  <path d="M12 5v14M5 12h14" />
                </svg>
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
