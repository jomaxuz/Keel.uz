"use client";

// Moving between a company's brands.
//
// A company may sell under two names — the restaurant and the samsa chain. Each
// is its own menu, its own look and its own basket, so this is not a filter: it
// is a different shop the guest walks into, sharing only their account.
//
// **With one brand nothing is rendered at all.** That is the rule the whole
// brand feature rests on: the small restaurant this template is mostly sold to
// must never see a control for something it does not have.
//
// ⚠️ **One button in the bar, the brands in a dialog.** The brands used to sit
// in the header side by side, and every brand added took its width from the
// nav: with three of them the links were squeezed until they wrapped. The bar
// now carries only the current brand, whatever the count, and the choice opens
// over the page with each brand's logo — a guest knows a shop by its sign.
//
// ⚠️ **Switching is a full page load, not `router.refresh()`.** The theme, the
// menu and the cart key all follow the brand; a soft refresh re-renders the
// server tree but keeps client state that was built for the brand just left.

import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { writeBrandCookie } from "@/lib/siteBrand";
import { localePath } from "@/lib/i18n";
import { useI18n } from "@/lib/i18n/client";
import BrandMark from "@/components/site/BrandMark";
import type { Brand } from "@/lib/types";

export default function BrandSwitch({
  brands,
  active,
  className = "",
}: {
  brands: Brand[];
  active: string;
  className?: string;
}) {
  const { t, lang } = useI18n();
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open]);

  if (brands.length < 2) return null;
  const current = brands.find((b) => b.id === active) ?? brands[0];

  function pick(brand: Brand) {
    setOpen(false);
    if (brand.id === current.id) return;
    // ⚠️ Written before the navigation: the next page is rendered from it.
    writeBrandCookie(brand.slug || brand.id);
    // The menu rather than the current page — a dish page belongs to the brand
    // being left and would be a 404 in the new one.
    window.location.assign(localePath(lang, "/menu"));
  }

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-haspopup="dialog"
        title={t.brandSwitch.title}
        className={`flex min-w-0 items-center gap-2 rounded-full border border-line bg-surface py-1 pl-1 pr-3 text-sm font-semibold text-ink-soft transition-colors hover:border-brand hover:text-ink ${className}`}
      >
        <BrandMark
          name={current.name}
          logoUrl={current.logoUrl}
          className="h-7 w-7 text-sm"
        />
        <span className="min-w-0 max-w-[9rem] truncate">{current.name}</span>
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

      {/* ⚠️ Portalled to <body>: the sticky header has `backdrop-blur`, and a
          backdrop filter makes it the containing block of every `fixed`
          child — the dialog would be drawn inside the 80px bar. */}
      {open &&
        createPortal(
          <div
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
            onClick={() => setOpen(false)}
            role="presentation"
          >
            <div
              role="dialog"
              aria-modal="true"
              aria-labelledby="brand-switch-title"
              className="relative max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-3xl bg-surface p-6 text-ink shadow-xl"
              onClick={(e) => e.stopPropagation()}
            >
              <button
                type="button"
                onClick={() => setOpen(false)}
                aria-label={t.common.close}
                title={t.common.close}
                className="absolute right-3 top-3 rounded-full p-1.5 text-ink-muted transition-colors hover:bg-ink/5 hover:text-ink"
              >
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  className="h-5 w-5"
                  aria-hidden
                >
                  <path d="M18 6 6 18M6 6l12 12" />
                </svg>
              </button>
              <h2
                id="brand-switch-title"
                className="pr-8 font-display text-xl font-bold"
              >
                {t.brandSwitch.title}
              </h2>
              <p className="mt-1 text-sm text-ink-muted">
                {t.brandSwitch.hint}
              </p>
              <div className="mt-5 grid gap-2 sm:grid-cols-2">
                {brands.map((b) => {
                  const on = b.id === current.id;
                  return (
                    <button
                      key={b.id}
                      type="button"
                      onClick={() => pick(b)}
                      aria-current={on ? "true" : undefined}
                      className={`flex items-center gap-3 rounded-2xl border p-3 text-left transition-colors ${
                        on
                          ? "border-brand bg-brand-tint"
                          : "border-line bg-surface hover:border-brand"
                      }`}
                    >
                      <BrandMark
                        name={b.name}
                        logoUrl={b.logoUrl}
                        className="h-12 w-12"
                      />
                      <span className="min-w-0 flex-1 truncate font-semibold">
                        {b.name}
                      </span>
                    </button>
                  );
                })}
              </div>
            </div>
          </div>,
          document.body,
        )}
    </>
  );
}
