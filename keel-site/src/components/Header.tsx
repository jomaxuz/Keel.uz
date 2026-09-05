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
    // ⚠️ **The till is first and it is a page, not an anchor.** It is what we
    // mainly sell, so it keeps the first slot — but it is no longer a section
    // of the home page, and an anchor to a section that moved scrolls to the
    // top and reads as a broken link. The integration list went with it, so
    // its entry is gone from here: seven items where a visitor was already
    // telling us the page had too much on it is the same mistake one level up.
    { href: "/kassa", label: t.nav.till, page: true },
    { href: "/#compare", label: t.nav.why },
    { href: "/#product", label: t.nav.product },
    { href: "/#pricing", label: t.nav.pricing },
    // Straight after the price, because that is where the calculator is on the
    // page and it is the answer to the question the price section raises: the
    // counter is billed monthly and orders are billed per order, and this is
    // the only block that adds the two together.
    { href: "/#calc", label: t.nav.calc },
    { href: "/#faq", label: t.nav.faq },
    // ⚠️ **A real page, not an anchor**, and the only entry in this list that
    // is one. It is last because it is the link people come back for rather
    // than the one that sells them — but it has to be in the header at all,
    // because the person looking for it is usually already a customer with a
    // problem, and «where are the instructions» is a support message we would
    // otherwise answer by hand.
    { href: "/help", label: t.nav.help, page: true },
    // ⚠️ **After the manual, and a real page like it.** The blog sells nobody
    // on its own; it is what a visitor reads while deciding, and what a search
    // engine finds months before anybody comes looking for us by name.
    { href: "/blog", label: t.blog.title, page: true },
  ];

  return (
    <header className="sticky top-0 z-40 border-b border-line/70 bg-page/85 backdrop-blur">
      <div className="container-page flex h-16 items-center justify-between gap-4">
        {/* Keeps the language in the address. The most-clicked link on the
            page must not quietly drop a visitor back to the Uzbek URL. */}
        <Link href={localePath(lang, "/")} aria-label="Keel">
          <Logo />
        </Link>

        {/* ⚠️ `lg`, not `md`. The row held seven links comfortably; the eighth
            ("Qo'llanma") made it wrap at tablet widths — and a nav that wraps
            reads as a broken header rather than as a full one. Between md and
            lg the burger takes over, which is what it is for. */}
        <nav className="hidden items-center gap-6 lg:flex">
          {links.map((l) =>
            // ⚠️ A page link goes through `localePath`, an anchor does not: an
            // anchor is on the landing page, which the language prefix already
            // decided, while `/help` dropped a Russian visitor back onto the
            // Uzbek article set.
            l.page ? (
              <Link
                key={l.href}
                href={localePath(lang, l.href)}
                className="text-sm font-medium text-ink-soft transition hover:text-ink"
              >
                {l.label}
              </Link>
            ) : (
              <a
                key={l.href}
                href={l.href}
                className="text-sm font-medium text-ink-soft transition hover:text-ink"
              >
                {l.label}
              </a>
            ),
          )}
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
            className="grid h-9 w-9 place-items-center rounded-xl border border-line lg:hidden"
          >
            <svg viewBox="0 0 24 24" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round">
              {open ? <path d="M6 6l12 12M18 6L6 18" /> : <path d="M4 7h16M4 12h16M4 17h16" />}
            </svg>
          </button>
        </div>
      </div>

      {open && (
        <div className="border-t border-line bg-page lg:hidden">
          <div className="container-page flex flex-col gap-1 py-3">
            {links.map((l) => (
              <a
                key={l.href}
                // Same rule as the desktop row: a page keeps the language, an
                // anchor is already on the page it points into.
                href={l.page ? localePath(lang, l.href) : l.href}
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
