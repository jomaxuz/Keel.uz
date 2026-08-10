"use client";

// Search on the home page.
//
// The home page is where a guest lands from a Telegram link, a QR card or a
// Google result, and a good half of them already know what they want. Today
// their path is: scroll past the hero, find the categories, guess which one
// holds lag'mon, open it, scroll. A search box on the front page is the
// shortcut, and it costs nothing to serve — **the home page already loads the
// whole menu** (it draws the categories and the popular dishes from it), so the
// suggestions below are matched against data that is on the page anyway.
//
// ⚠️ **It suggests, it does not become a results page.** Six dishes fit under an
// input without pushing the restaurant's own page out of the way; the seventh
// belongs on /menu, where the filters are. So the last row is always a way out
// to the full menu carrying the same query — the guest never types twice.

import { useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "@/components/site/LocaleLink";
import { imageUrl } from "@/lib/api";
import { localePath } from "@/lib/i18n";
import { formatPrice } from "@/lib/format";
import { useI18n } from "@/lib/i18n/client";
import { contentName } from "@/lib/i18n/content";
import { buildIndex, NO_FILTERS, runSearch } from "@/lib/search";
import type { MenuGroup } from "@/lib/types";

const MAX_SUGGESTIONS = 6;

export default function HomeSearch({
  menu,
  currency,
  big,
}: {
  menu: MenuGroup[];
  currency: string;
  /** The bigger variant: a headline above the box, for a band of its own. */
  big?: boolean;
}) {
  const { lang, t } = useI18n();
  const router = useRouter();
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const boxRef = useRef<HTMLDivElement>(null);

  const index = useMemo(() => buildIndex(menu, lang), [menu, lang]);
  const hits = useMemo(
    () => (query.trim() ? runSearch(index, query, NO_FILTERS) : []),
    [index, query],
  );
  const shown = hits.slice(0, MAX_SUGGESTIONS);
  const menuHref = `/menu?q=${encodeURIComponent(query.trim())}`;

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!query.trim()) return;
    // The whole point of the box: whatever was typed here arrives on /menu
    // already applied. Typing it a second time is how a guest concludes the
    // search does not work.
    router.push(localePath(lang, menuHref));
  }

  return (
    <div
      ref={boxRef}
      className="relative"
      // Blur closes the list, but only when focus left the box entirely —
      // otherwise clicking a suggestion would unmount it before the click lands.
      onBlur={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setOpen(false);
      }}
    >
      {big && (
        <p className="mb-3 text-center font-display text-xl font-bold sm:text-2xl">
          {t.search.homeTitle}
        </p>
      )}
      <form onSubmit={submit} className="relative">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-muted"
          aria-hidden
        >
          <circle cx="11" cy="11" r="7" />
          <path d="m20 20-3.5-3.5" />
        </svg>
        <input
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          // Not type="search": the browsers that draw their own clear cross
          // would put a second × next to ours (see MenuBrowser).
          type="text"
          inputMode="search"
          enterKeyHint="search"
          placeholder={t.search.placeholder}
          aria-label={t.search.placeholder}
          className={`input w-full pl-11 pr-11 ${big ? "h-14 text-base" : "h-12"}`}
        />
        {query && (
          <button
            type="button"
            onClick={() => setQuery("")}
            aria-label={t.search.clear}
            className="absolute right-3 top-1/2 flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-ink-muted transition-colors hover:bg-ink/5 hover:text-ink"
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
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
        )}
      </form>

      {open && query.trim() && (
        <div className="absolute left-0 right-0 top-full z-40 mt-2 overflow-hidden rounded-2xl border border-line bg-surface shadow-card-hover">
          {shown.length === 0 ? (
            <p className="px-4 py-5 text-center text-sm text-ink-muted">
              {t.search.nothing}
            </p>
          ) : (
            <ul className="divide-y divide-line">
              {shown.map((item) => {
                const img = imageUrl(item.imageUrl, 300);
                return (
                  <li key={item.id}>
                    <Link
                      href={`/menu/${item.id}`}
                      className="flex items-center gap-3 px-3 py-2.5 transition-colors hover:bg-ink/5"
                    >
                      <span className="h-10 w-10 shrink-0 overflow-hidden rounded-xl bg-ink/5">
                        {img && (
                          // eslint-disable-next-line @next/next/no-img-element
                          <img
                            src={img}
                            alt=""
                            loading="lazy"
                            className="h-full w-full object-cover"
                          />
                        )}
                      </span>
                      <span className="min-w-0 flex-1 truncate text-sm font-medium">
                        {contentName(item, lang)}
                        {/* Sold out is worth saying here rather than hiding the
                            row: a guest hunting for it deserves the answer. */}
                        {(item.soldOut || !item.isAvailable) && (
                          <span className="ml-2 text-xs text-ink-muted">
                            {item.soldOut && item.isAvailable
                              ? t.item.soldOut
                              : t.item.unavailable}
                          </span>
                        )}
                      </span>
                      <span className="shrink-0 text-sm font-semibold">
                        {formatPrice(item.price, currency, lang)}
                      </span>
                    </Link>
                  </li>
                );
              })}
            </ul>
          )}
          <Link
            href={menuHref}
            className="block border-t border-line px-4 py-3 text-center text-sm font-semibold text-brand hover:bg-brand/5"
          >
            {hits.length > shown.length
              ? t.search.seeAllCount(hits.length)
              : t.search.seeAll}
          </Link>
        </div>
      )}
    </div>
  );
}
