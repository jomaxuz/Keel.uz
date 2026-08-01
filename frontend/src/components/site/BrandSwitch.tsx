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

import { useRouter } from "next/navigation";
import { writeBrandCookie } from "@/lib/siteBrand";
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
  const router = useRouter();
  if (brands.length < 2) return null;

  function pick(brand: Brand) {
    if (brand.id === active) return;
    writeBrandCookie(brand.slug || brand.id);
    // The menu, the home page and the theme are server-rendered from the
    // cookie, so the whole shell has to be re-fetched, not just re-styled.
    router.refresh();
    router.push("/menu");
  }

  return (
    <div
      className={`flex items-center gap-1 rounded-full border border-line bg-surface p-1 ${className}`}
      role="tablist"
    >
      {brands.map((b) => (
        <button
          key={b.id}
          type="button"
          role="tab"
          aria-selected={b.id === active}
          onClick={() => pick(b)}
          className={`rounded-full px-3 py-1.5 text-xs font-bold transition-colors ${
            b.id === active
              ? "bg-brand-tint text-brand-dark"
              : "text-ink-soft hover:text-ink"
          }`}
        >
          {b.name}
        </button>
      ))}
    </div>
  );
}
