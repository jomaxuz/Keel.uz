"use client";

import { useEffect, useState } from "react";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useCart } from "@/lib/cart";
import { useUser } from "@/lib/user";
import { useI18n } from "@/lib/i18n/client";
import { formatUzPhone } from "@/lib/format";
import LangSwitch from "@/components/site/LangSwitch";
import ThemeToggle from "@/components/site/ThemeToggle";
import BrandMark from "@/components/site/BrandMark";
import BrandSwitch from "@/components/site/BrandSwitch";
import type { Brand } from "@/lib/types";

export default function Header({
  name,
  logoUrl,
  brands = [],
  activeBrand = "",
}: {
  name: string;
  logoUrl?: string;
  /** The company's brands. One (or none) renders no switcher at all. */
  brands?: Brand[];
  activeBrand?: string;
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
  const { user } = useUser();
  const { t } = useI18n();
  const pathname = usePathname();

  const nav = [
    { href: "/", label: t.nav.home },
    { href: "/menu", label: t.nav.menu },
    { href: "/bron", label: t.nav.booking },
    { href: "/about", label: t.nav.about },
  ];

  const isActive = (href: string) =>
    href === "/" ? pathname === "/" : pathname.startsWith(href);

  return (
    <header className="sticky top-0 z-40 border-b border-line bg-cream/85 backdrop-blur-md">
      <div className="container-page flex h-16 items-center justify-between gap-4 sm:h-20">
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
        <BrandSwitch brands={brands} active={activeBrand} className="hidden sm:flex" />

        {/* Desktop nav */}
        <nav className="hidden items-center gap-1 md:flex">
          {nav.map((n) => (
            <Link
              key={n.href}
              href={n.href}
              className={`rounded-full px-4 py-2 text-sm font-semibold transition-colors ${
                isActive(n.href)
                  ? "bg-brand-tint text-brand-dark"
                  : "text-ink-soft hover:bg-ink/5 hover:text-ink"
              }`}
            >
              {n.label}
            </Link>
          ))}
        </nav>

        <div className="flex shrink-0 items-center gap-2 sm:gap-3">
          {/* Language + theme live in the navbar itself. */}
          <LangSwitch />
          <ThemeToggle />

          {user ? (
            <Link
              href="/profile"
              className="hidden items-center gap-2 rounded-full border border-line bg-surface py-1 pl-1 pr-3 text-sm font-semibold text-ink-soft transition-colors hover:border-brand hover:text-brand sm:flex"
            >
              <span className="flex h-7 w-7 items-center justify-center rounded-full bg-brand-tint text-xs font-bold text-brand">
                {(user.firstName || user.phone || "?").charAt(0).toUpperCase()}
              </span>
              {user.firstName || formatUzPhone(user.phone)}
            </Link>
          ) : (
            <Link
              href="/login"
              className="hidden rounded-full px-3 py-2 text-sm font-semibold text-ink-soft transition-colors hover:text-brand sm:inline"
            >
              {t.nav.login}
            </Link>
          )}

          <Link
            href="/cart"
            className="btn-primary btn-icon relative sm:h-auto sm:w-auto sm:px-4 sm:py-2.5"
          >
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="h-4 w-4"
              aria-hidden
            >
              <circle cx="9" cy="20" r="1.5" />
              <circle cx="18" cy="20" r="1.5" />
              <path d="M2 3h2.2l2.3 12.2a2 2 0 0 0 2 1.6h8.3a2 2 0 0 0 2-1.55L21 7H5.2" />
            </svg>
            <span className="hidden sm:inline">{t.nav.cart}</span>
            {showCount && count > 0 && (
              <span className="absolute -right-1 -top-1 inline-flex min-w-5 items-center justify-center rounded-full bg-ink px-1.5 py-0.5 text-[11px] font-bold text-cream ring-2 ring-cream">
                {count}
              </span>
            )}
          </Link>
        </div>
      </div>

      {/* Mobile-only nav row (desktop nav sits in the bar above). */}
      <div className="border-t border-line md:hidden">
        <div className="container-page flex items-center gap-3 py-2">
          <nav className="no-scrollbar flex items-center gap-1 overflow-x-auto">
            {nav.map((n) => (
              <Link
                key={n.href}
                href={n.href}
                className={`whitespace-nowrap rounded-full px-3 py-1.5 text-sm font-semibold ${
                  isActive(n.href)
                    ? "bg-brand-tint text-brand-dark"
                    : "text-ink-soft"
                }`}
              >
                {n.label}
              </Link>
            ))}
            <Link
              href={user ? "/profile" : "/login"}
              className="whitespace-nowrap rounded-full px-3 py-1.5 text-sm font-semibold text-ink-soft"
            >
              {user ? t.nav.profile : t.nav.login}
            </Link>
          </nav>
        </div>
      </div>
    </header>
  );
}
