"use client";

// Search over the knowledge base.
//
// ⚠️ **In the browser, over an index the server sent** — not a request per
// keystroke. Sixty-odd articles is a few kilobytes of titles and leads, and the
// alternative is a search endpoint that is slower than typing and useless the
// moment the connection is the reason somebody opened the help at all.
//
// ⚠️ The full body is *not* in the index, only the title, the lead and the
// search words. Shipping every article's text to every visitor to make search
// slightly better is the whole knowledge base downloaded by somebody who wanted
// one page.

import Link from "next/link";
import { useMemo, useState } from "react";
import type { Lang } from "@/lib/i18n/dict";
import { localePath } from "@/lib/i18n/url";

export type SearchItem = {
  slug: string;
  title: string;
  lead: string;
  section: string;
  /** Title + lead + the extra words somebody might search, lowercased once. */
  hay: string;
};

export default function SearchBox({
  items,
  lang,
  placeholder,
  empty,
  count,
}: {
  items: SearchItem[];
  lang: Lang;
  placeholder: string;
  /** `{q}` is replaced with what was typed. A template rather than a function:
   *  a function cannot be handed to a client component, and this one has to be
   *  translated. */
  empty: string;
  /** `{n}` is replaced with the number of hits. */
  count: string;
}) {
  const [q, setQ] = useState("");

  const hits = useMemo(() => {
    const terms = q
      .toLowerCase()
      .split(/\s+/)
      .filter((t) => t.length > 1);
    if (terms.length === 0) return null;
    // ⚠️ Every term has to match, and a title hit outranks a lead hit. Somebody
    // typing two words is narrowing; and the article actually *called*
    // "Printerni ulash" has to come above the four that mention a printer.
    return items
      .map((it) => {
        let score = 0;
        for (const t of terms) {
          if (it.title.toLowerCase().includes(t)) score += 10;
          else if (it.hay.includes(t)) score += 1;
          else return null;
        }
        return { it, score };
      })
      .filter((x) => x !== null)
      .sort((a, b) => b.score - a.score)
      .slice(0, 12)
      .map((x) => x.it);
  }, [q, items]);

  return (
    <div className="relative">
      <div className="relative">
        <svg
          viewBox="0 0 24 24"
          className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-muted"
          fill="none"
          stroke="currentColor"
          strokeWidth={2}
          strokeLinecap="round"
        >
          <circle cx="11" cy="11" r="7" />
          <path d="M20 20l-3.5-3.5" />
        </svg>
        <input
          className="input pl-11"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder={placeholder}
          // ⚠️ `type="search"` and not `type="text"`: on a phone it puts a
          // clear button in the field and the word "search" on the keyboard,
          // and this input is mostly used on a phone.
          type="search"
          autoComplete="off"
          aria-label={placeholder}
        />
      </div>

      {hits !== null && (
        <div className="mt-3 rounded-2xl border border-line bg-surface p-2">
          {hits.length === 0 ? (
            <p className="px-3 py-4 text-sm text-ink-muted">
              {empty.replace("{q}", q)}
            </p>
          ) : (
            <>
              <p className="px-3 pb-1 pt-2 text-xs uppercase tracking-wide text-ink-muted">
                {count.replace("{n}", String(hits.length))}
              </p>
              <ul>
                {hits.map((h) => (
                  <li key={h.slug}>
                    <Link
                      href={localePath(lang, `/help/${h.slug}`)}
                      className="block rounded-xl px-3 py-2.5 transition hover:bg-raised"
                    >
                      <span className="block text-sm font-medium text-ink">
                        {h.title}
                      </span>
                      <span className="mt-0.5 block text-xs text-ink-muted">
                        {h.lead}
                      </span>
                    </Link>
                  </li>
                ))}
              </ul>
            </>
          )}
        </div>
      )}
    </div>
  );
}
