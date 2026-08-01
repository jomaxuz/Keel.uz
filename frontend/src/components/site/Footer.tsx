"use client";

import Link from "next/link";
import { useI18n } from "@/lib/i18n/client";
import BrandMark from "@/components/site/BrandMark";
import { localized } from "@/lib/i18n/site-content";
import type { Restaurant } from "@/lib/types";

export default function Footer({ restaurant }: { restaurant: Restaurant | null }) {
  const { lang, t } = useI18n();
  const year = new Date().getFullYear();
  const pages = [
    { href: "/menu", label: t.nav.menu },
    { href: "/about", label: t.nav.about },
    { href: "/cart", label: t.nav.cart },
    { href: "/profile", label: t.footer.myOrders },
  ];
  const socials = [
    { href: restaurant?.socials?.instagram, label: "Instagram" },
    { href: restaurant?.socials?.telegram, label: "Telegram" },
    { href: restaurant?.socials?.facebook, label: "Facebook" },
  ].filter((s) => !!s.href);

  return (
    <footer className="mt-20 border-t border-line bg-surface text-ink">
      <div className="container-page grid grid-cols-1 gap-10 py-14 sm:grid-cols-2 lg:grid-cols-4">
        <div className="lg:col-span-2">
          <div className="flex items-center gap-2.5">
            <BrandMark
              name={restaurant?.name ?? "R"}
              logoUrl={restaurant?.logoUrl}
            />
            <h3 className="font-display text-2xl font-bold">
              {restaurant?.name ?? t.common.restaurant}
            </h3>
          </div>
          <p className="mt-4 max-w-sm text-sm leading-relaxed text-ink-muted">
            {localized(
              restaurant?.content?.footerNote,
              lang,
              restaurant?.description ?? "",
            )}
          </p>
          {socials.length > 0 && (
            <div className="mt-5 flex flex-wrap gap-2">
              {socials.map((s) => (
                <a
                  key={s.label}
                  href={s.href!}
                  target="_blank"
                  rel="noreferrer"
                  className="rounded-full border border-line-strong px-4 py-1.5 text-xs font-semibold text-ink-soft transition-colors hover:border-brand hover:text-brand"
                >
                  {s.label}
                </a>
              ))}
            </div>
          )}
        </div>

        <div>
          <h4 className="text-xs font-bold uppercase tracking-[0.18em] text-brand">
            {t.footer.contact}
          </h4>
          <ul className="mt-4 space-y-2 text-sm text-ink-muted">
            {restaurant?.phones?.map((p) => (
              <li key={p}>
                <a href={`tel:${p}`} className="transition-colors hover:text-brand">
                  {p}
                </a>
              </li>
            ))}
            {restaurant?.address?.text && (
              <li className="text-ink-muted/70">{restaurant.address.text}</li>
            )}
          </ul>
        </div>

        <div>
          <h4 className="text-xs font-bold uppercase tracking-[0.18em] text-brand">
            {t.footer.pages}
          </h4>
          <ul className="mt-4 space-y-2 text-sm text-ink-muted">
            {pages.map((p) => (
              <li key={p.href}>
                <Link href={p.href} className="transition-colors hover:text-brand">
                  {p.label}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      </div>

      <div className="border-t border-line py-5 text-center text-xs text-ink-muted/70">
        {t.footer.rights(year, restaurant?.name ?? t.common.restaurant)}
      </div>
    </footer>
  );
}
