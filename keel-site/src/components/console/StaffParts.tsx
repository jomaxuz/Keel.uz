"use client";

// Pieces the staff list and a person's own page share.
//
// ⚠️ **Not exported from a `page.tsx`.** Next allows a page module only its
// default export and a few reserved names; anything else fails the build's type
// check, and only the build's — `next dev` renders it happily.

import { useT } from "@/lib/i18n/client";
import type { ConsoleLogRow } from "@/lib/api";

// ⚠️ Ids here, words from the dictionary at render — a const map is evaluated
// at import, before the language is known.
export const ROLES = ["owner", "admin", "manager", "agent", "support"] as const;

/** Toggle pills for a set of roles. ⚠️ The last one cannot be unticked: an
 *  account with no roles is refused by the server, and a pill that silently
 *  springs back teaches nothing. */
export function RolePicker({
  value,
  onChange,
  disabled,
}: {
  value: string[];
  onChange: (roles: string[]) => void;
  disabled?: boolean;
}) {
  const { t } = useT();
  return (
    <div className="flex flex-wrap gap-1">
      {ROLES.map((r) => {
        const on = value.includes(r);
        const last = on && value.length === 1;
        return (
          <button
            key={r}
            type="button"
            disabled={disabled || last}
            title={last ? t.console.staff.atLeastOne : undefined}
            onClick={(e) => {
              e.stopPropagation();
              onChange(on ? value.filter((x) => x !== r) : [...value, r]);
            }}
            className={`rounded-lg border px-2 py-1 text-[11px] font-semibold transition ${
              on
                ? "border-ink bg-ink text-surface"
                : "border-line text-ink-muted hover:text-ink"
            } ${last ? "cursor-default" : ""}`}
          >
            {t.console.staff.roles[r]}
          </button>
        );
      })}
    </div>
  );
}

export function LogBlock({ rows, title }: { rows: ConsoleLogRow[]; title?: string }) {
  const { t } = useT();
  return (
    <section className="card p-0">
      <div className="flex items-baseline justify-between gap-3 border-b border-line px-5 py-3">
        <h2 className="text-sm font-semibold text-ink">
          {title ?? t.console.staff.log}
        </h2>
        <span className="text-xs text-ink-muted">
          {t.console.staff.logCount(rows.length)}
        </span>
      </div>
      <ul className="max-h-96 divide-y divide-line overflow-y-auto px-5 text-xs">
        {rows.map((l) => (
          <li key={l.id} className="flex flex-wrap items-baseline gap-2 py-1.5">
            <span className="text-ink-muted">
              {new Date(l.at).toLocaleString()}
            </span>
            <span className="font-semibold text-ink">{l.actor}</span>
            <span className="text-ink-soft">{l.action}</span>
            {l.target && <span className="text-ink-muted">{l.target}</span>}
            {l.detail && <span className="text-ink-muted">· {l.detail}</span>}
          </li>
        ))}
        {rows.length === 0 && (
          <li className="py-2 text-ink-muted">{t.console.staff.logEmpty}</li>
        )}
      </ul>
    </section>
  );
}
