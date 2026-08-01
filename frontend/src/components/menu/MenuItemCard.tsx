"use client";

import Link from "next/link";
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
  const img = imageUrl(item.imageUrl);
  // Dishes with option groups cannot be added in one tap — the customer picks
  // the variant on the dish page. Without options the line id is just the id.
  const hasOptions = (item.options ?? []).some((g) => g.choices?.length);
  const qty = lines.find((l) => l.lineId === item.id)?.qty ?? 0;
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
    <div className="group card flex flex-col overflow-hidden transition-all duration-300 hover:-translate-y-1 hover:shadow-card-hover">
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

      <div className="flex flex-1 flex-col p-4 sm:p-5">
        <Link href={`/menu/${item.id}`}>
          <h3 className="font-display text-lg font-bold leading-snug transition-colors group-hover:text-brand">
            {name}
          </h3>
        </Link>
        {/* What is in the set. Without it "Oilaviy combo — 60 000" tells the
            guest nothing they can decide on. */}
        {item.comboContents?.length ? (
          <ul className="mt-1.5 space-y-0.5 text-sm text-ink-muted">
            {item.comboContents.map((c, i) => (
              <li key={i} className="truncate">
                {c.qty > 1 ? `${c.name} × ${c.qty}` : c.name}
              </li>
            ))}
          </ul>
        ) : (
          description && (
            <p className="mt-1.5 line-clamp-2 text-sm leading-relaxed text-ink-muted">
              {description}
            </p>
          )
        )}

        <div className="mt-4 flex items-end justify-between gap-3 pt-1">
          <div className="leading-tight">
            <span className="whitespace-nowrap font-display text-lg font-bold sm:text-xl">
              {formatPrice(item.price, currency, lang)}
            </span>
            {hasDiscount && (
              <span className="ml-2 text-sm text-ink-muted line-through">
                {formatPrice(item.oldPrice!, currency, lang)}
              </span>
            )}
            {saving > 0 && (
              <span className="ml-2 text-sm text-ink-muted line-through">
                {formatPrice(item.comboBasePrice!, currency, lang)}
              </span>
            )}
          </div>
          {/* Once in the cart the button turns into a −/qty/+ stepper. */}
          {hasOptions ? (
            <Link
              href={`/menu/${item.id}`}
              aria-disabled={!orderable}
              className={`btn-primary shrink-0 px-3.5 py-2 ${
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
              className="btn-primary btn-icon sm:h-auto sm:w-auto sm:px-3.5 sm:py-2"
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
              <span className="hidden sm:inline">{t.item.add}</span>
            </button>
          ) : (
            <div className="flex shrink-0 items-center gap-1 rounded-full bg-brand p-1 text-white">
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
