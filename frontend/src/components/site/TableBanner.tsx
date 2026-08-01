"use client";

// A quiet strip under the header: "you are at table 7".
//
// Without it, a guest who scanned a table QR would have no idea why checkout is
// offering to bring the food to a table — and a guest who scanned by accident
// would have no way out. Hence the "not at a table" escape next to it.

import { useTable } from "@/lib/table";
import { useI18n } from "@/lib/i18n/client";

export default function TableBanner() {
  const { table, clear } = useTable();
  const { t } = useI18n();
  if (!table) return null;

  return (
    <div className="border-b border-brand/30 bg-brand-tint/60">
      <div className="container-page flex flex-wrap items-center gap-x-3 gap-y-1 py-2 text-sm">
        <span className="font-semibold text-brand-dark">
          {t.table.atTable(table.number)}
        </span>
        <span className="text-ink-muted">{t.table.hint}</span>
        <button
          type="button"
          onClick={clear}
          className="ml-auto text-xs font-semibold text-ink-muted underline-offset-2 hover:text-brand hover:underline"
        >
          {t.table.leave}
        </button>
      </div>
    </div>
  );
}
