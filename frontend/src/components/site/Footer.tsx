"use client";

// Locale-aware: hrefs stay unprefixed here and gain /ru or /en at render.
import Link from "@/components/site/LocaleLink";
import { useI18n } from "@/lib/i18n/client";
import CallLink from "@/components/site/CallLink";
import BrandMark from "@/components/site/BrandMark";
import { localized } from "@/lib/i18n/site-content";
import type { Restaurant } from "@/lib/types";

export default function Footer({
  restaurant,
  /** Whether to print the "Powered by Keel" line. Passed in from the server
   *  layout rather than read here: it belongs to the control plane, and a
   *  client component has no way to ask. */
  watermark = false,
}: {
  restaurant: Restaurant | null;
  watermark?: boolean;
}) {
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
                <CallLink phone={p} className="transition-colors hover:text-brand">
                  {p}
                </CallLink>
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
        {/* Quiet on purpose. It has to be visible enough that removing it is
            worth paying for, and modest enough that a restaurant is not
            embarrassed to leave it — a loud badge gets removed with CSS by
            the first owner who knows how. */}
        {watermark && (
          <>
            {" · "}
            <a
              href="https://keel.uz"
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1.5 align-middle transition-colors hover:text-brand"
            >
              <KeelMark />
              {t.footer.poweredBy}
            </a>
          </>
        )}
      </div>
    </footer>
  );
}

/** Keel's mark, in the customer's own ink.
 *
 *  **Drawn in `currentColor`, never in Keel's amber.** This badge sits at the
 *  bottom of somebody else's restaurant, under a palette they chose — a second
 *  brand colour appearing there is the thing that makes an owner want it gone,
 *  and the badge only works if it is easy to leave alone. So it takes the
 *  footer's muted ink, and the restaurant's own accent on hover.
 *
 *  Copied rather than imported: keel.uz and this app are separate builds that
 *  share no code, and a shared package for two `<path>` elements would be a
 *  dependency to maintain forever. If the mark ever changes, it changes in both
 *  — `keel-site/src/components/Logo.tsx` is the other one.
 */
function KeelMark() {
  return (
    <svg
      viewBox="0 0 32 32"
      fill="none"
      stroke="currentColor"
      strokeWidth={2.8}
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-3.5 w-3.5 shrink-0"
      aria-hidden
    >
      {/* The fin is the mark. Kept full length even at this size — shortened,
          it reads as a smudge and the logo becomes an anonymous curve. */}
      <path d="M5 6c0 9.5 4.4 14.5 11 14.5S27 15.5 27 6" />
      <path d="M16 20.5V29" />
    </svg>
  );
}
