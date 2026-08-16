"use client";

// Page controls for the guest-facing lists (profile: favourites, order
// history, bookings).
//
// Separate from the panel's `Pager` for one reason only: the wording comes from
// the guest dictionary, not the admin one. The slicing is shared (`usePaged`).
//
// ⚠️ **Turning the page scrolls back to the list's own heading**, not to the
// top of the document and not nowhere at all. A profile has three of these
// stacked; pressing "Next" on the bottom one while looking at row 5 otherwise
// leaves the reader staring at rows that silently became different rows.

import { useI18n } from "@/lib/i18n/client";

export default function Pager({
  page,
  pageCount,
  from,
  to,
  total,
  onPage,
  anchorRef,
  className = "",
}: {
  page: number;
  pageCount: number;
  from: number;
  to: number;
  total: number;
  onPage: (page: number) => void;
  /** The section heading to bring back into view when the page changes. */
  anchorRef?: React.RefObject<HTMLElement | null>;
  className?: string;
}) {
  const { t } = useI18n();

  // A single page needs no controls, and the range line alone ("1–3 / 3") is
  // noise on a list the reader can already see in full.
  if (total === 0 || pageCount <= 1) return null;

  const go = (next: number) => {
    onPage(next);
    anchorRef?.current?.scrollIntoView({ behavior: "smooth", block: "start" });
  };

  return (
    <div
      className={`flex flex-wrap items-center justify-between gap-3 pt-1 text-xs text-ink-muted ${className}`}
    >
      <span>{t.common.pagerRange(from, to, total)}</span>
      <div className="flex items-center gap-2">
        <button
          type="button"
          disabled={page <= 1}
          onClick={() => go(page - 1)}
          className="rounded-full border border-line-strong px-3 py-1.5 font-semibold transition-colors enabled:hover:border-brand enabled:hover:text-brand disabled:opacity-40"
        >
          {t.common.prev}
        </button>
        <span className="font-medium text-ink tabular-nums">
          {t.common.pageOf(page, pageCount)}
        </span>
        <button
          type="button"
          disabled={page >= pageCount}
          onClick={() => go(page + 1)}
          className="rounded-full border border-line-strong px-3 py-1.5 font-semibold transition-colors enabled:hover:border-brand enabled:hover:text-brand disabled:opacity-40"
        >
          {t.common.next}
        </button>
      </div>
    </div>
  );
}
