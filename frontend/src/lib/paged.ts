"use client";

// Client-side paging, shared by the panel and the public site.
//
// The hook is here rather than next to either pager because it holds no
// wording: the panel's pager reads the admin dictionary, the site's reads the
// guest one, and both slice the list exactly the same way. Two copies of the
// slicing would drift on the clamp, and the clamp is the part nobody tests.

import { useEffect, useMemo, useState } from "react";

/**
 * Slices `items` into pages. Resets to the first page whenever the list itself
 * changes (a new filter or search must not leave the reader on page 7 of a
 * three-page result), and clamps the page when rows disappear under it.
 */
export function usePaged<T>(items: T[], pageSize = 20) {
  const [page, setPage] = useState(1);
  const total = items.length;
  const pageCount = Math.max(1, Math.ceil(total / pageSize));

  useEffect(() => {
    setPage((p) => Math.min(p, Math.max(1, Math.ceil(items.length / pageSize))));
  }, [items.length, pageSize]);

  const pageItems = useMemo(
    () => items.slice((page - 1) * pageSize, page * pageSize),
    [items, page, pageSize],
  );

  return {
    page,
    setPage,
    pageItems,
    total,
    pageCount,
    from: total === 0 ? 0 : (page - 1) * pageSize + 1,
    to: Math.min(page * pageSize, total),
  };
}
