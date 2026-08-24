"use client";

import Link from "next/link";
import { useState } from "react";
import { Logo } from "./Logo";
import LangSwitch from "./LangSwitch";
import ThemeToggle from "./ThemeToggle";
import { useT } from "@/lib/i18n/client";
import { localePath } from "@/lib/i18n/url";
import { TELEGRAM } from "@/lib/links";

export default function Header() {
  const { t, lang } = useT();
  const [open, setOpen] = useState(false);

  // Rooted at "/", not bare fragments.
  //
  // Every one of these points at a section of the landing page, and the header
  // is shared with pages that have no such section — /status was the first.
  // A bare "#pricing" there scrolls nowhere and looks like a dead link; "/#pricing"
  // goes home and lands on it. From the landing page itself the behaviour is
  // unchanged: same route, so the browser just scrolls.
  const links = [
    // ⚠️ **The till is first**, because it is what we mainly sell and it is
    // the first section of the page. A nav whose order disagrees with the
    // page's is a nav that sends people backwards.
    { href: "/#till", label: t.nav.till },
    { href: "/#product", label: t.nav.product },
    { href: "/#who", label: t.nav.who },
    { href: "/#integrations", label: t.nav.integrations },
    { href: "/#pricing", label: t.nav.pricing },
    { href: "/#faq", label: t.nav.faq },
  ];

  return (
    <header className="sticky top-0 z-40 border-b border-line/70 bg-page/85 backdrop-blur">
      <div className="container-page flex h-16 items-center justify-between gap-4">
        {/* Keeps the language in the address. The most-clicked link on the
            page must not quietly drop a visitor back to the Uzbek URL. */}
        <Link href={localePath(lang, "/")} aria-label="Keel">
          <Logo />
        </Link>

        <nav className="hidden items-center gap-7 md:flex">
          {links.map((l) => (
            <a
              key={l.href}
              href={l.href}
              className="text-sm font-medium text-ink-soft transition hover:text-ink"
            >
              {l.label}
            </a>
          ))}
        </nav>

        <div className="flex items-center gap-2">
          <div className="hidden sm:block">
            <LangSwitch />
          </div>
          <ThemeToggle />
          <a href={TELEGRAM} className="btn-primary hidden sm:inline-flex">
            {t.nav.start}
          </a>
          <button
            type="button"
            onClick={() => setOpen((v) => !v)}
            aria-expanded={open}
            aria-label={t.nav.product}
            className="grid h-9 w-9 place-items-center rounded-xl border border-line md:hidden"
          >
            <svg viewBox="0 0 24 24" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round">
              {open ? <path d="M6 6l12 12M18 6L6 18" /> : <path d="M4 7h16M4 12h16M4 17h16" />}
            </svg>
          </button>
        </div>
      </div>

      {open && (
        <div className="border-t border-line bg-page md:hidden">
          <div className="container-page flex flex-col gap-1 py-3">
            {links.map((l) => (
              <a
                key={l.href}
                href={l.href}
                onClick={() => setOpen(false)}
                className="rounded-xl px-2 py-2.5 text-sm font-medium text-ink-soft hover:bg-raised hover:text-ink"
              >
                {l.label}
              </a>
            ))}
            <div className="flex items-center justify-between pt-2">
              <LangSwitch />
              <a href={TELEGRAM} onClick={() => setOpen(false)} className="btn-primary">
                {t.nav.start}
              </a>
            </div>
          </div>
        </div>
      )}
    </header>
  );
}
