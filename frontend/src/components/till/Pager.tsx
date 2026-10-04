"use client";

import { useEffect, useMemo, useState } from "react";
import { LuChevronLeft, LuChevronRight } from "react-icons/lu";

import { useAdminT } from "@/lib/i18n/admin";

/**
 * Pages for the till's lists.
 *
 * ⚠️ **On the till, a long list is a list nobody reaches the end of.** Closed
 * shifts, the evening's checks, last week's shopping lists — each grew into a
 * column that had to be dragged with a finger on a monoblock, past rows the
 * cashier was not looking for, and the scroll bar on a touch screen is a few
 * pixels nobody can grab. Pages are tapped: two big buttons and a count.
 *
 * ⚠️ **Client-side, on what the server already sent.** Every one of these
 * lists is bounded on the server (today's checks, the last shifts), and asking
 * for the next page over the network would be a spinner between two pages of
 * data already on the machine.
 *
 * ⚠️ **The page follows the list.** When it shrinks — a debt paid, a search
 * typed — the page is pulled back to the last one that exists, rather than
 * showing an empty page with "3 / 2" under it.
 */
export function usePaged<T>(items: T[], size = TILL_PAGE) {
  const [page, setPage] = useState(0);
  const pages = Math.max(1, Math.ceil(items.length / size));
  useEffect(() => {
    if (page > pages - 1) setPage(pages - 1);
  }, [page, pages]);
  const shown = useMemo(
    () => items.slice(page * size, page * size + size),
    [items, page, size],
  );
  return { shown, page: Math.min(page, pages - 1), pages, setPage };
}

/** How many rows a till page holds — about what a 1080px screen shows. */
export const TILL_PAGE = 12;

/** The two buttons and the count. Draws nothing for a single page. */
export function TillPager({
  page,
  pages,
  onPage,
  className = "",
}: {
  page: number;
  pages: number;
  onPage: (page: number) => void;
  className?: string;
}) {
  const t = useAdminT();
  if (pages <= 1) return null;
  return (
    <div className={`flex items-center justify-center gap-3 py-2 ${className}`}>
      <button
        type="button"
        className="till-btn h-11 w-14 px-0"
        disabled={page <= 0}
        aria-label={t.till.pagePrev}
        onClick={() => onPage(page - 1)}
      >
        <LuChevronLeft className="mx-auto h-5 w-5" aria-hidden />
      </button>
      <span className="min-w-[4.5rem] text-center text-sm font-semibold tabular-nums text-ink-muted">
        {page + 1} / {pages}
      </span>
      <button
        type="button"
        className="till-btn h-11 w-14 px-0"
        disabled={page >= pages - 1}
        aria-label={t.till.pageNext}
        onClick={() => onPage(page + 1)}
      >
        <LuChevronRight className="mx-auto h-5 w-5" aria-hidden />
      </button>
    </div>
  );
}
