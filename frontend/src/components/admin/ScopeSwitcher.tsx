"use client";

// Which brand and branch the whole panel is read through.
//
// Renders nothing at all when the company has one brand and one branch: the
// single-restaurant customer must not be shown machinery they will never use.
//
// Built as a button that opens a list rather than a native <select>, to match
// the language and theme controls it sits under. A bare select in a sidebar
// reads as a label: it has no affordance next to the pill-shaped buttons above
// it, and an owner looking for "the switcher" walks straight past it.

import { useEffect, useRef, useState } from "react";
import { useAdminScope } from "@/lib/adminScope";
import { useAdminT } from "@/lib/i18n/admin";

export default function ScopeSwitcher({
  className = "",
}: {
  className?: string;
}) {
  const t = useAdminT();
  const {
    brands,
    brandBranches,
    brand,
    branch,
    multi,
    pinned,
    setBrand,
    setBranch,
  } = useAdminScope();

  if (!multi) return null;

  // A manager tied to one branch cannot move the lens — the server would clamp
  // them back. Show which branch they are in instead of a control that lies.
  if (pinned) {
    return (
      <div className={`text-xs text-ink-muted ${className}`}>
        <span className="block rounded-xl border border-line bg-surface px-3 py-2 font-semibold text-ink-soft">
          {branch?.name ?? brand?.name}
        </span>
      </div>
    );
  }

  return (
    <div className={`flex flex-col gap-2 ${className}`}>
      {brands.length > 1 && (
        <Picker
          label={t.scope.brand}
          value={brand?.name ?? ""}
          options={brands.map((b) => ({ id: b.id, label: b.name }))}
          selected={brand?.id ?? ""}
          onPick={setBrand}
        />
      )}
      {brandBranches.length > 1 && (
        <Picker
          label={t.scope.branch}
          value={branch?.name ?? t.scope.allBranches}
          // Highlighted while on the overview: that is the state in which
          // several screens have nothing branch-specific to show, so it is the
          // one worth noticing.
          highlight={!branch}
          options={[
            { id: "", label: t.scope.allBranches },
            ...brandBranches.map((b) => ({
              id: b.id,
              label: b.name + (b.isActive ? "" : ` · ${t.scope.inactive}`),
            })),
          ]}
          selected={branch?.id ?? ""}
          onPick={setBranch}
        />
      )}
    </div>
  );
}

function Picker({
  label,
  value,
  options,
  selected,
  onPick,
  highlight = false,
}: {
  label: string;
  value: string;
  options: { id: string; label: string }[];
  selected: string;
  onPick: (id: string) => void;
  highlight?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  // Close on an outside click or Escape, the same way the language popup does.
  useEffect(() => {
    if (!open) return;
    function onClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <div ref={ref} className="relative">
      <span className="mb-1 block text-[11px] font-semibold uppercase tracking-wide text-ink-muted">
        {label}
      </span>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="listbox"
        aria-expanded={open}
        className={`flex w-full items-center justify-between gap-2 rounded-xl border px-3 py-2 text-left text-sm font-semibold transition-colors ${
          highlight
            ? "border-brand/50 bg-brand-tint text-brand-dark hover:border-brand"
            : "border-line-strong bg-surface text-ink-soft hover:border-brand hover:text-brand"
        }`}
      >
        <span className="truncate">{value}</span>
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.2"
          strokeLinecap="round"
          className={`h-3.5 w-3.5 shrink-0 transition-transform ${open ? "rotate-180" : ""}`}
          aria-hidden
        >
          <path d="m6 9 6 6 6-6" />
        </svg>
      </button>

      {open && (
        <div
          role="listbox"
          className="absolute left-0 right-0 z-50 mt-1 overflow-hidden rounded-2xl border border-line bg-surface p-1 shadow-card-hover"
        >
          {options.map((o) => (
            <button
              key={o.id}
              type="button"
              role="option"
              aria-selected={o.id === selected}
              onClick={() => {
                onPick(o.id);
                setOpen(false);
              }}
              className={`block w-full truncate rounded-xl px-3 py-2 text-left text-sm font-semibold transition-colors ${
                o.id === selected
                  ? "bg-brand-tint text-brand-dark"
                  : "text-ink-soft hover:bg-ink/5"
              }`}
            >
              {o.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
