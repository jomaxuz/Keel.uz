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
//
// ⚠️ **The brand is chosen in a dialog, the branch in the dropdown.** They were
// both dropdowns — a column of names in a rail narrow enough to truncate most
// of them — and the two questions are not the same size. A branch is a room of
// one business somebody is already looking at; a brand is the whole business,
// and picking the wrong one puts an owner in somebody else's menu, orders and
// money with nothing on the page saying so. So it takes the middle of the
// screen, it carries each brand's own logo (which is how an owner recognises
// their businesses — they named them, they did not memorise a list order), and
// the choice is deliberate enough to be worth an extra press.
//
// ⚠️ **Switching brand reloads the page, on purpose.** Every screen in the
// panel reads through this lens, and not all of them refetch on `scopeKey`:
// the briefing, anything holding a draft, anything that loaded once in an
// effect with an empty dependency list. The result was a dashboard wearing the
// new brand's name over the old brand's figures — which is worse than a slow
// switch, because nothing about it looks wrong. A reload is the one way to be
// certain nothing survives the change, and it costs a second on an action
// somebody takes a handful of times a day.

import { useEffect, useRef, useState } from "react";
import { imageUrl } from "@/lib/api";
import { useAdminScope } from "@/lib/adminScope";
import { useAdminT } from "@/lib/i18n/admin";
import Modal from "@/components/admin/Modal";
import type { Brand } from "@/lib/types";

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
        <BrandPicker
          brands={brands}
          current={brand}
          onPick={(id) => {
            if (id === brand?.id) return;
            setBrand(id);
            // ⚠️ **The choice is written before the reload, not after.**
            // `setBrand` puts it in localStorage synchronously and the fresh
            // page reads it back on load; a reload scheduled first would race
            // the write and land on the brand the owner just left.
            window.location.reload();
          }}
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

/** The brands, as a dialog with faces in it.
 *
 *  ⚠️ **The logo, not only the name.** An owner running three businesses knows
 *  them by their signs; the panel knew them by a list order it had inherited
 *  from whichever was created first. A tile with the mark on it is recognised
 *  before it is read, which is the whole difference on a control that changes
 *  what every other screen is about.
 *
 *  ⚠️ **The trigger says which brand is current even while shut**, because the
 *  rail is the only place in the panel that answers "whose numbers am I looking
 *  at?" — and that question is asked most often by somebody who has just walked
 *  back to the laptop. */
function BrandPicker({
  brands,
  current,
  onPick,
}: {
  brands: Brand[];
  current: Brand | null;
  onPick: (id: string) => void;
}) {
  const t = useAdminT();
  const [open, setOpen] = useState(false);

  return (
    <div>
      <span className="mb-1 block text-[11px] font-semibold uppercase tracking-wide text-ink-muted">
        {t.scope.brand}
      </span>
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-haspopup="dialog"
        className="flex w-full items-center gap-2 rounded-xl border border-line-strong bg-surface px-2.5 py-2 text-left text-sm font-semibold text-ink-soft transition-colors hover:border-brand hover:text-brand"
      >
        <BrandLogo brand={current} className="h-7 w-7 text-[13px]" />
        <span className="min-w-0 flex-1 truncate">{current?.name ?? ""}</span>
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.2"
          strokeLinecap="round"
          className="h-3.5 w-3.5 shrink-0"
          aria-hidden
        >
          <path d="m6 9 6 6 6-6" />
        </svg>
      </button>

      {open && (
        <Modal onClose={() => setOpen(false)}>
          <h2 className="text-lg font-bold">{t.scope.brandPickTitle}</h2>
          <p className="mt-1 text-sm text-ink-muted">{t.scope.brandPickHint}</p>
          <div className="mt-4 grid gap-2 sm:grid-cols-2">
            {brands.map((b) => {
              const on = b.id === current?.id;
              return (
                <button
                  key={b.id}
                  type="button"
                  onClick={() => {
                    setOpen(false);
                    onPick(b.id);
                  }}
                  aria-current={on ? "true" : undefined}
                  className={`flex items-center gap-3 rounded-2xl border p-3 text-left transition-colors ${
                    on
                      ? "border-brand bg-brand-tint"
                      : "border-line bg-surface hover:border-brand"
                  }`}
                >
                  <BrandLogo brand={b} className="h-11 w-11 text-lg" />
                  <span className="min-w-0">
                    <span className="block truncate font-semibold">
                      {b.name}
                    </span>
                    {/* A brand switched off is still switchable *to* — its
                        orders and its reports did not stop existing — so it is
                        listed and labelled rather than hidden. */}
                    {!b.isActive && (
                      <span className="block text-xs text-ink-muted">
                        {t.scope.inactive}
                      </span>
                    )}
                  </span>
                </button>
              );
            })}
          </div>
        </Modal>
      )}
    </div>
  );
}

/** A brand's mark, or its initial when it has not uploaded one.
 *
 *  ⚠️ Never an empty box: a fresh brand has no logo for as long as it takes
 *  somebody to upload one, and that is exactly when this control is being
 *  learned. Same fallback as the site header's BrandMark, which is deliberately
 *  not imported — that one belongs to the guest's page and carries its sizing. */
function BrandLogo({
  brand,
  className,
}: {
  brand: Brand | null;
  className: string;
}) {
  const src = imageUrl(brand?.logoUrl ?? "", 300);
  if (src) {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={src}
        alt=""
        className={`shrink-0 rounded-xl object-cover ${className}`}
      />
    );
  }
  return (
    <span
      className={`flex shrink-0 items-center justify-center rounded-xl bg-brand font-bold text-white ${className}`}
      aria-hidden
    >
      {(brand?.name ?? "").trim().charAt(0).toUpperCase() || "?"}
    </span>
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
