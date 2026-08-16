"use client";

// Long admin lists, bounded.
//
// Every list in the panel grows without limit — orders, customers, couriers,
// the audit log — and a page that simply renders all of them gets taller by the
// day until the operator is scrolling past hundreds of rows to reach the filter
// they wanted. So each list lives in its own scroll block of a fixed height,
// with a pager underneath when there is more than one page of rows.

import { type ReactNode } from "react";
import { useAdminT } from "@/lib/i18n/admin";

// The slicing itself lives in `lib/paged` — the guest profile pages its lists
// the same way, and only the wording differs.
export { usePaged } from "@/lib/paged";

/**
 * The scroll block a list sits in. `max-h` keeps the page from growing with the
 * data; `overscroll-contain` stops a finished inner scroll from carrying on
 * into the page behind it.
 */
export function ListScroll({
  children,
  className = "",
  max = "max-h-[65vh]",
}: {
  children: ReactNode;
  className?: string;
  max?: string;
}) {
  return (
    <div className={`${max} overflow-y-auto overscroll-contain ${className}`}>
      {children}
    </div>
  );
}

/** Page controls: how many rows are shown, and the way to the rest. */
export function Pager({
  page,
  pageCount,
  from,
  to,
  total,
  onPage,
  className = "",
}: {
  page: number;
  pageCount: number;
  from: number;
  to: number;
  total: number;
  onPage: (page: number) => void;
  className?: string;
}) {
  const t = useAdminT();
  if (total === 0) return null;

  return (
    <div
      className={`flex flex-wrap items-center justify-between gap-3 border-t border-line px-4 py-3 text-xs text-ink-muted ${className}`}
    >
      <span>{t.common.pagerRange(from, to, total)}</span>
      {pageCount > 1 && (
        <div className="flex items-center gap-2">
          <button
            type="button"
            disabled={page <= 1}
            onClick={() => onPage(page - 1)}
            className="rounded-full border border-line-strong px-3 py-1 font-semibold transition-colors enabled:hover:border-brand enabled:hover:text-brand disabled:opacity-40"
          >
            {t.common.prev}
          </button>
          <span className="font-medium text-ink">
            {t.common.pageOf(page, pageCount)}
          </span>
          <button
            type="button"
            disabled={page >= pageCount}
            onClick={() => onPage(page + 1)}
            className="rounded-full border border-line-strong px-3 py-1 font-semibold transition-colors enabled:hover:border-brand enabled:hover:text-brand disabled:opacity-40"
          >
            {t.common.next}
          </button>
        </div>
      )}
    </div>
  );
}
