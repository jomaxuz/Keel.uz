"use client";

import { useEffect, useState } from "react";

// Locale-aware: hrefs stay unprefixed here and gain /ru or /en at render.
import Link from "@/components/site/LocaleLink";
import { usePathname } from "next/navigation";
import { useCart } from "@/lib/cart";
import { useUser } from "@/lib/user";
import { useI18n } from "@/lib/i18n/client";
import { splitLangPath } from "@/lib/i18n";
import { formatUzPhone } from "@/lib/format";
import LangSwitch from "@/components/site/LangSwitch";
import ThemeToggle from "@/components/site/ThemeToggle";
import BrandMark from "@/components/site/BrandMark";
import BrandSwitch from "@/components/site/BrandSwitch";
import { localized } from "@/lib/i18n/site-content";
import { hasTables } from "@/lib/types";
import { siteWords } from "@/lib/siteWords";
import {
  CHROME_TONE,
  CHROME_WIDTH,
  DEFAULT_CHROME,
  type SiteChrome,
} from "@/lib/siteChrome";
import type { Brand, NavLink } from "@/lib/types";

export default function Header({
  name,
  logoUrl,
  brands = [],
  activeBrand = "",
  branchCount = 0,
  navLinks = [],
  businessType,
  chrome,
}: {
  name: string;
  logoUrl?: string;
  /** The company's brands. One (or none) renders no switcher at all. */
  brands?: Brand[];
  activeBrand?: string;
  /** How many branches this brand has. See the note where the nav is built. */
  branchCount?: number;
  /** The bar drawn in the console's constructor, when this site has one.
   *
   *  ⚠️ **Empty means the built-in bar, never an empty bar.** Every brand on
   *  the platform has none of these, so reading absence as "no destinations"
   *  would take the navigation off every site at once. */
  navLinks?: NavLink[];
  /** What this brand sells — read only to decide whether a table-booking link
   *  belongs in the built-in bar. */
  businessType?: string;
  /** How the bar looks, read out of the drawn design's `navbar` band.
   *
   *  ⚠️ Absent is today's header, field by field — see DEFAULT_CHROME. */
  chrome?: SiteChrome;
}) {
  const { count } = useCart();
  // The badge waits for this component's *own* mount before it appears.
  //
  // The cart is hydrated from localStorage by CartProvider, which sits outside
  // the <Suspense> that defers this header — so by the time the header
  // hydrates, the count is already real while the server HTML has no badge at
  // all. A flag owned by the parent would be no help (that is what broke
  // ThemeToggle); a flag owned by *this* component is false during its own
  // hydration, matches the server, and flips immediately after.
  const [showCount, setShowCount] = useState(false);
  useEffect(() => setShowCount(true), []);
  // The mobile menu. Closed on every navigation: a panel still open over the
  // page it just navigated to reads as a link that did not work.
  const [menuOpen, setMenuOpen] = useState(false);
  const { user } = useUser();
  const { t, lang } = useI18n();
  const pathname = usePathname();
  const w = siteWords(t, businessType);
  const c = chrome ?? DEFAULT_CHROME;
  const transparent = c.variant === "transparent";
  const centered = c.variant === "centered";
  const minimal = c.variant === "minimal";
  // ⚠️ A transparent bar has no surface of its own, so it also gets no tone:
  // painting one would be the opposite of what the variant is for. It sits over
  // whatever band is underneath it, which is why it is not sticky either —
  // a transparent bar that follows the page down stops being over the hero and
  // starts being over the text, unreadable.
  const surface = transparent
    ? "border-b border-transparent"
    : (CHROME_TONE[c.tone] ?? CHROME_TONE[""]);
  const stick = c.sticky && !transparent ? "sticky top-0" : "absolute inset-x-0 top-0";
  // Lowercase with wide tracking is the fashion-site convention, and it is the
  // one thing that makes a bar copied from a lookbook read as one.
  const navCase = c.lowercase ? "lowercase tracking-wide" : "";

  /** The bar this site would have with nothing drawn for it. */
  const builtIn: NavItem[] = [
    { href: "/", label: t.nav.home },
    // ⚠️ "Katalog" in a shop, "Menyu" in a restaurant — see lib/siteWords.ts.
    // The route is the same one; only the word changes.
    { href: "/menu", label: w.nav },
    // ⚠️ Only with more than one. A single-branch restaurant already shows its address,
    // its hours and a map on the about page, so a nav item leading to a list of one is
    // a click that answers nothing — and this product's rule is that a one-branch
    // customer never sees the multi-branch machinery.
    ...(branchCount > 1 ? [{ href: "/filiallar", label: t.branches.title }] : []),
    // ⚠️ **Only where somebody can sit down.** A shop and an online store have
    // no tables, and a bar whose third item is "Bron" leading to a
    // table-booking form is the clearest possible statement that this site was
    // built for somebody else. The booking page itself is unchanged — the
    // brand's own `features.booking` still governs it; this is the bar.
    ...(hasTables({ businessType }) ? [{ href: "/bron", label: t.nav.booking }] : []),
    { href: "/about", label: t.nav.about },
  ];

  // ⚠️ **Drawn wins whole, rather than merging with the built-in bar.** Half of
  // somebody's design plus half of ours is a bar neither of them meant — and
  // the reason an online store gets this at all is that our half is wrong for
  // it. `hidden` links stay in the document so trying a bar and putting a link
  // back does not mean retyping it in three languages.
  const drawn = navLinks.filter((l) => !l.hidden && l.href);
  const nav: NavItem[] =
    drawn.length > 0
      ? drawn.map((l) => ({
          href: l.href,
          label: localized(l.label, lang),
          external: l.external,
        }))
      : builtIn;

  // ⚠️ Compared against the path with the language prefix removed. `usePathname`
  // returns what the browser shows — `/ru/menu`, not the rewritten `/menu` — so
  // a raw comparison marks nothing active for a Russian or English visitor, and
  // the whole navbar silently loses its highlight for two of three languages.
  const here = splitLangPath(pathname).path;
  useEffect(() => setMenuOpen(false), [pathname]);
  const isActive = (n: NavItem) => {
    // ⚠️ An outside address is never "here", and `startsWith` on one would be
    // comparing a path to `https://…` — always false, but by accident.
    if (n.external || !n.href.startsWith("/")) return false;
    // A drawn link may carry a query ("/menu?cat=ayollar"); the highlight is
    // about which page the guest is on, not which filter.
    const path = n.href.split(/[?#]/)[0] || "/";
    return path === "/" ? here === "/" : here.startsWith(path);
  };

  return (
    <header className={`${stick} z-40 ${surface}`} style={CHROME_WIDTH[c.width]}>
      <div
        className={`container-page gap-4 ${
          centered
            ? "flex flex-col items-center py-3"
            : "flex h-16 items-center justify-between sm:h-20"
        }`}
      >
        {/* Wordmark */}
        {/* A long restaurant name must give way to the cart and the language
            switch rather than push them off a phone screen. */}
        <Link href="/" className="flex min-w-0 items-center gap-2.5">
          <BrandMark name={name} logoUrl={logoUrl} />
          <span className="truncate font-display text-xl font-bold tracking-tight sm:text-2xl">
            {name}
          </span>
        </Link>

        {/* Which brand's shop this is — absent unless there is more than one. */}
        <BrandSwitch brands={brands} active={activeBrand} className="hidden lg:flex" />

        {/* Desktop nav.
            ⚠️ **`minimal` pushes it to the right** rather than hiding anything:
            the variant is about how little else is in the bar, not about
            removing the way to the catalogue. `centered` puts it on its own row
            under the wordmark, which is what the flex direction above decides. */}
        <nav
          className={`hidden items-center gap-1 lg:flex ${
            minimal ? "ml-auto" : ""
          } ${centered ? "order-3 mt-1" : ""}`}
        >
          {nav.map((n) => (
            <NavItemLink
              key={n.href + n.label}
              item={n}
              className={`rounded-full px-4 py-2 text-sm font-semibold transition-colors ${navCase} ${
                isActive(n)
                  ? transparent || c.tone
                    ? "underline underline-offset-8"
                    : "bg-brand-tint text-brand-dark"
                  : transparent || c.tone
                    ? "opacity-70 hover:opacity-100"
                    : "text-ink-soft hover:bg-ink/5 hover:text-ink"
              }`}
            />
          ))}
        </nav>

        <div
          className={`flex shrink-0 items-center gap-2 sm:gap-3 ${
            centered ? "order-2" : ""
          }`}
        >
          {/* The hamburger. Labelled by what it does rather than by its state:
              a label that flips between "open" and "close" has to survive
              hydration, and both icons are drawn so CSS alone decides which is
              visible — the same rule ThemeToggle follows. */}
          <button
            type="button"
            onClick={() => setMenuOpen((v) => !v)}
            aria-expanded={menuOpen}
            aria-label={t.nav.menuLabel}
            className={`flex h-9 w-9 items-center justify-center rounded-full transition-colors lg:hidden ${
              transparent || c.tone
                ? "border border-current/30 hover:opacity-80"
                : "border border-line bg-surface text-ink-soft hover:border-brand hover:text-brand"
            }`}
          >
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              className="h-5 w-5"
              aria-hidden
            >
              {menuOpen ? (
                <path d="M6 6l12 12M18 6 6 18" />
              ) : (
                <path d="M4 7h16M4 12h16M4 17h16" />
              )}
            </svg>
          </button>
          {/* ⚠️ On a phone the bar holds **only** the hamburger and the cart.
              Everything else — language, theme, profile — moved into the panel.
              Four small controls crowded into one corner all lose: the cart badge
              stops being noticed, and the cart is the one control that carries
              money. Language and theme are still one tap away, just behind a
              button that names itself. */}
          {/* ⚠️ **A magnifier that goes to the catalogue, not a second search
              box.** The catalogue page already carries the real one, with the
              same index the menu search uses; a box in the bar would be a
              second implementation of a feature where "lag'mon spelt six ways"
              has to work in both or in neither. Off unless a design asks for
              it — today's header has none, and a restaurant would be given a
              control it has no use for. */}
          {c.icons.search && (
            <Link
              href="/menu"
              aria-label={w.nav}
              className="hidden h-9 w-9 items-center justify-center rounded-full transition-opacity hover:opacity-70 lg:flex"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8"
                strokeLinecap="round" className="h-5 w-5" aria-hidden>
                <circle cx="11" cy="11" r="7" />
                <path d="M21 21l-4.3-4.3" />
              </svg>
            </Link>
          )}
          <div className="hidden items-center gap-2 lg:flex">
            {c.icons.lang && <LangSwitch />}
            {c.icons.theme && <ThemeToggle />}
          </div>

          {/* ⚠️ **Hidden, never replaced by a stub.** Switching the account
              control off takes it out of the bar; the profile and the login
              pages are untouched and still reachable, including from the phone
              panel below. A bar is a shortcut, not a permission. */}
          {c.icons.account &&
            (user ? (
              <Link
                href="/profile"
                className={`hidden items-center gap-2 rounded-full py-1 pl-1 pr-3 text-sm font-semibold transition-colors lg:flex ${
                  transparent || c.tone
                    ? "hover:opacity-70"
                    : "border border-line bg-surface text-ink-soft hover:border-brand hover:text-brand"
                }`}
              >
                <span className="flex h-7 w-7 items-center justify-center rounded-full bg-brand-tint text-xs font-bold text-brand">
                  {(user.firstName || user.phone || "?").charAt(0).toUpperCase()}
                </span>
                {user.firstName || formatUzPhone(user.phone)}
              </Link>
            ) : (
              <Link
                href="/login"
                className={`hidden rounded-full px-3 py-2 text-sm font-semibold transition-colors lg:inline ${
                  transparent || c.tone
                    ? "hover:opacity-70"
                    : "text-ink-soft hover:text-brand"
                }`}
              >
                {t.nav.login}
              </Link>
            ))}

          {/* ⚠️ **The cart is the one control that carries money**, so it keeps
              the badge in every form. On a plain bar it is a filled button —
              what every customer has today; on a bar with a tone of its own a
              filled button in a second colour is two accents fighting, so it
              becomes an icon like the ones beside it. That is a look, not a
              demotion: the badge, the label on wide screens and the target size
              are the same. */}
          {c.icons.cart && (
            <Link
              href="/cart"
              aria-label={t.nav.cart}
              className={
                transparent || c.tone
                  ? "relative flex h-9 w-9 items-center justify-center rounded-full transition-opacity hover:opacity-70"
                  : "btn-primary btn-icon relative sm:h-auto sm:w-auto sm:px-4 sm:py-2.5"
              }
            >
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className={transparent || c.tone ? "h-5 w-5" : "h-4 w-4"}
                aria-hidden
              >
                <circle cx="9" cy="20" r="1.5" />
                <circle cx="18" cy="20" r="1.5" />
                <path d="M2 3h2.2l2.3 12.2a2 2 0 0 0 2 1.6h8.3a2 2 0 0 0 2-1.55L21 7H5.2" />
              </svg>
              {!(transparent || c.tone) && (
                <span className="hidden sm:inline">{t.nav.cart}</span>
              )}
              {showCount && count > 0 && (
                <span className="absolute -right-1 -top-1 inline-flex min-w-5 items-center justify-center rounded-full bg-ink px-1.5 py-0.5 text-[11px] font-bold text-cream ring-2 ring-cream">
                  {count}
                </span>
              )}
            </Link>
          )}
        </div>
      </div>

      {/* ⚠️ A panel behind one button, not a scrolling row of chips.
          The row it replaces put four destinations plus the profile into a
          horizontally scrollable strip — which on a 360px phone showed two and a
          half of them and hid the rest behind a gesture nothing announced. The
          sections a guest is looking for were the ones off-screen.
          Rendered only when open: an always-mounted panel with `hidden` keeps its
          links in the tab order and in the accessibility tree, so a phone reader
          walks through a menu nobody opened.

          ⚠️ **Two shapes, one list.** `panel` drops under the bar — what every
          site has today. `drawer` comes in from the side over the page, which
          is what a catalogue with eight sections needs: a panel that long
          pushes the page down and opens somewhere the thumb is not. The links,
          the brand switcher and the settings row are the same in both; only
          the container differs, so there is no second menu to keep in step. */}
      {menuOpen && c.burger === "drawer" && (
        // The scrim. ⚠️ It closes the drawer, and it is a button rather than a
        // div with an onClick: a tap target that dismisses something has to be
        // reachable from a keyboard and announced to a screen reader.
        <button
          type="button"
          aria-label={t.common.close}
          onClick={() => setMenuOpen(false)}
          className="fixed inset-0 z-40 bg-ink/40 backdrop-blur-sm lg:hidden"
        />
      )}
      {menuOpen && (
        <div
          className={
            c.burger === "drawer"
              ? "fixed right-0 top-0 z-50 h-dvh w-[82%] max-w-sm overflow-y-auto border-l border-line bg-cream shadow-card lg:hidden"
              : "border-t border-line bg-cream lg:hidden"
          }
        >
          {c.burger === "drawer" && (
            <div className="flex justify-end p-3">
              <button
                type="button"
                onClick={() => setMenuOpen(false)}
                aria-label={t.common.close}
                className="flex h-9 w-9 items-center justify-center rounded-full border border-line text-ink-soft"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
                  strokeLinecap="round" className="h-5 w-5" aria-hidden>
                  <path d="M6 6l12 12M18 6 6 18" />
                </svg>
              </button>
            </div>
          )}
          <nav className="container-page flex flex-col py-2">
            {nav.map((n) => (
              <NavItemLink
                key={n.href + n.label}
                item={n}
                // Full-width rows, comfortably tall: this is a one-handed thumb
                // target, not a desktop pointer.
                className={`rounded-xl px-3 py-3 text-base font-semibold ${
                  isActive(n)
                    ? "bg-brand-tint text-brand-dark"
                    : "text-ink-soft"
                }`}
              />
            ))}
            <Link
              href={user ? "/profile" : "/login"}
              className={`rounded-xl px-3 py-3 text-base font-semibold ${
                isActive({ href: "/profile", label: "" })
                  ? "bg-brand-tint text-brand-dark"
                  : "text-ink-soft"
              }`}
            >
              {user ? t.nav.profile : t.nav.login}
            </Link>

            {/* The brand switcher lives here on a phone. In the bar it was
                `hidden sm:flex`, which meant a two-brand company had no way to
                switch brands at all on the screen most of their guests use. */}
            {brands.length > 1 && (
              <div className="mt-2 border-t border-line pt-3">
                <BrandSwitch brands={brands} active={activeBrand} />
              </div>
            )}

            {/* Language and theme, at the bottom of the panel rather than in the
                bar. Settings, not destinations — so they sit under the places a
                guest is actually going, separated by a rule. */}
            <div className="mt-2 flex items-center gap-2 border-t border-line pt-3">
              <LangSwitch />
              <ThemeToggle />
            </div>
          </nav>
        </div>
      )}
    </header>
  );
}

/** One destination in the bar, whichever of the two bars it came from. */
type NavItem = { href: string; label: string; external?: boolean };

/** ⚠️ **Two elements, because a drawn link may leave the site.** `LocaleLink`
 *  prefixes `/ru` or `/en` onto an href, which is right for a page of ours and
 *  nonsense on `https://t.me/...` — and Next's client router cannot navigate to
 *  another origin at all. `rel="noreferrer"` goes with `target="_blank"`: a new
 *  tab opened without it can reach back into this one through `window.opener`.
 */
function NavItemLink({ item, className }: { item: NavItem; className: string }) {
  if (item.external || !item.href.startsWith("/")) {
    return (
      <a
        href={item.href}
        target="_blank"
        rel="noreferrer noopener"
        className={className}
      >
        {item.label}
      </a>
    );
  }
  return (
    <Link href={item.href} className={className}>
      {item.label}
    </Link>
  );
}
