"use client";

// One piece of site copy in three languages. RU/EN are optional — the Uzbek
// text is used wherever they are blank, which is why it stands in as their
// placeholder: an owner who fills only the first box can see what the other two
// will show rather than having to remember the rule.
//
// Lifted out of the settings page when the perks editor needed the same field.
// Two copies would have drifted the day one of them grew a character counter.

import type { LocalizedText } from "@/lib/types";

export default function LocalizedField({
  label,
  value,
  onChange,
  multiline = false,
  /** Bare inputs, for callers that already own the surrounding card. */
  plain = false,
  rows = 4,
}: {
  label: string;
  value: LocalizedText;
  onChange: (next: LocalizedText) => void;
  multiline?: boolean;
  plain?: boolean;
  rows?: number;
}) {
  const cls =
    "mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand";
  const body = (
    <>
      <span className={plain ? "text-sm font-medium" : "text-sm font-semibold"}>
        {label}
      </span>
      <div className="mt-2 grid gap-3 sm:grid-cols-3">
        {(["uz", "ru", "en"] as const).map((code) => (
          <label key={code} className="block text-sm">
            <span className="text-xs font-medium uppercase text-ink-muted">
              {code}
            </span>
            {multiline ? (
              <textarea
                className={cls}
                rows={rows}
                value={value[code]}
                placeholder={code === "uz" ? "" : value.uz}
                onChange={(e) => onChange({ ...value, [code]: e.target.value })}
              />
            ) : (
              <input
                className={cls}
                value={value[code]}
                placeholder={code === "uz" ? "" : value.uz}
                onChange={(e) => onChange({ ...value, [code]: e.target.value })}
              />
            )}
          </label>
        ))}
      </div>
    </>
  );
  if (plain) return <div>{body}</div>;
  return (
    <div className="rounded-2xl border border-line bg-ink/[0.02] p-4">{body}</div>
  );
}
