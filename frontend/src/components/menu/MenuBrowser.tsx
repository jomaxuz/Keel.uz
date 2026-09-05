"use client";

// The menu, with a way through it.
//
// A printed menu is read top to bottom; a phone menu is not. A guest who came
// for lag'mon should not scroll past four sections to find it, and a guest who
// has 40 000 so'm should be able to say so. That is the whole brief: one search
// box and one funnel.
//
// ⚠️ **It filters what is already on the page, not on the server.** A restaurant
// menu is a hundred-odd dishes and they are all in this component already. A
// request per keystroke would cost a third of a second each on a Tashkent mobile
// connection, and would turn the one screen that must feel instant into the
// slowest one on the site. It also means search keeps working while the backend
// is having a bad minute.
//
// ⚠️ **The default state renders exactly what the server rendered**: the full
// grouped menu, every dish in the HTML. That is not only hydration safety — the
// dish pages and the menu are what the restaurant is found by (see the SEO notes
// in CLAUDE.md), and a menu that only appears after JavaScript runs is a menu
// Google reads as empty.

import { useMemo, useState } from "react";
import { imageUrl } from "@/lib/api";
import MenuItemCard from "@/components/menu/MenuItemCard";
import { useI18n } from "@/lib/i18n/client";
import { contentName } from "@/lib/i18n/content";
import {
  activeFilterCount,
  buildIndex,
  NO_FILTERS,
  runSearch,
  type MenuFilters,
  type MenuSort,
} from "@/lib/search";
import type { MenuGroup } from "@/lib/types";
import { oneCardPerModel } from "@/lib/types";

export default function MenuBrowser({
  groups: allItems,
  currency,
  initialQuery = "",
}: {
  groups: MenuGroup[];
  currency: string;
  /** What the home page's box was asked for, carried in `?q=`. */
  initialQuery?: string;
}) {
  // ⚠️ **One card per model, before anything else reads the list.** A guest
  // browsing a clothes shop should see shirts, not sizes: twelve cards of one
  // photograph is a catalogue nobody scrolls, and the choice between them is
  // made on the page anyway, where every size has its own price and its own
  // stock. Grouped here rather than in each grid so the search index, the
  // facets and the category counts all agree about what a card is.
  const groups = useMemo(
    () =>
      allItems.map((g) => ({ ...g, items: oneCardPerModel(g.items) })),
    [allItems],
  );
  const { lang, t } = useI18n();
  const [query, setQuery] = useState(initialQuery);
  const [filters, setFilters] = useState<MenuFilters>(NO_FILTERS);
  const [panelOpen, setPanelOpen] = useState(false);

  // Folded once per menu, not once per keystroke.
  const index = useMemo(() => buildIndex(groups, lang), [groups, lang]);

  const filterCount = activeFilterCount(filters);
  const searching = query.trim().length > 0 || filterCount > 0;
  const results = useMemo(
    () => (searching ? runSearch(index, query, filters) : []),
    [searching, index, query, filters],
  );

  // What the filter panel can offer is decided by the menu, not by us: a
  // restaurant with no combos has no reason to see a "sets" switch, and an empty
  // control teaches the guest that the panel is decoration.
  const facets = useMemo(() => {
    const items = groups.flatMap((g) => g.items);
    const tags = new Set<string>();
    let hasCombo = false;
    let hasDiscount = false;
    let min = Number.POSITIVE_INFINITY;
    let max = 0;
    for (const it of items) {
      for (const tag of it.tags ?? []) if (tag.trim()) tags.add(tag.trim());
      if (it.comboItems?.length || it.comboContents?.length) hasCombo = true;
      if (it.oldPrice != null && it.oldPrice > it.price) hasDiscount = true;
      min = Math.min(min, it.price);
      max = Math.max(max, it.price);
    }
    return {
      tags: [...tags].sort(),
      hasCombo,
      hasDiscount,
      min: Number.isFinite(min) ? min : 0,
      max,
    };
  }, [groups]);

  // ⚠️ replaceState, not a router push: the query belongs in the address bar so
  // a found dish can be sent to somebody, but re-rendering the route on every
  // keystroke would throw away the instant filtering this whole component
  // exists for — and would stack a history entry per letter, so Back became
  // "delete one character".
  function updateQuery(next: string) {
    setQuery(next);
    if (typeof window === "undefined") return;
    const url = new URL(window.location.href);
    if (next.trim()) url.searchParams.set("q", next.trim());
    else url.searchParams.delete("q");
    window.history.replaceState(null, "", url);
  }

  function patch(next: Partial<MenuFilters>) {
    setFilters((f) => ({ ...f, ...next }));
  }
  function reset() {
    setFilters(NO_FILTERS);
    updateQuery("");
  }
  function toggleIn(list: string[], value: string): string[] {
    return list.includes(value) ? list.filter((x) => x !== value) : [...list, value];
  }

  return (
    <>
      {/* Sticky bar: search, funnel, and the category rail beneath.
          ⚠️ The `top` offsets are the header's own height (h-16 / sm:h-20) and
          nothing else — written as the same Tailwind steps rather than as pixels,
          because a number that means "the height of another component" goes stale
          the moment that component changes. That has already happened once here. */}
      <div className="sticky top-16 z-30 border-b border-line bg-cream/90 backdrop-blur-md sm:top-20">
        <div className="container-page flex items-center gap-2 py-3">
          <div className="relative flex-1">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-muted"
              aria-hidden
            >
              <circle cx="11" cy="11" r="7" />
              <path d="m20 20-3.5-3.5" />
            </svg>
            <input
              value={query}
              onChange={(e) => updateQuery(e.target.value)}
              // ⚠️ Not type="search": WebKit and Blink draw their own clear
              // cross inside it, so the guest would see two ×'s side by side —
              // and Firefox draws none, which is the other half of why the
              // button below is ours.
              type="text"
              inputMode="search"
              enterKeyHint="search"
              placeholder={t.search.placeholder}
              aria-label={t.search.placeholder}
              className="input h-10 w-full pl-9 pr-9"
            />
            {query && (
              <button
                type="button"
                onClick={() => updateQuery("")}
                aria-label={t.search.clear}
                className="absolute right-2 top-1/2 flex h-6 w-6 -translate-y-1/2 items-center justify-center rounded-full text-ink-muted transition-colors hover:bg-ink/5 hover:text-ink"
              >
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2.5"
                  strokeLinecap="round"
                  className="h-3.5 w-3.5"
                  aria-hidden
                >
                  <path d="M18 6 6 18M6 6l12 12" />
                </svg>
              </button>
            )}
          </div>

          {/* The funnel. Icon plus a count, because "3" is the only thing that
              explains a short result list to somebody who set the filters two
              scrolls ago and forgot.
              ⚠️ **The icon changes with the state, not just the colour.** A tinted
              funnel says "filters exist"; it does not say whether this tap opens
              the panel or closes it — and on a phone the panel pushes the dishes
              off screen, so that is the question being asked. A cross answers it
              without reading anything. The label follows the same rule: it names
              the action the tap performs, never the current state. */}
          <button
            type="button"
            onClick={() => setPanelOpen((v) => !v)}
            aria-expanded={panelOpen}
            aria-label={panelOpen ? t.search.filtersClose : t.search.filtersOpen}
            className={`relative flex h-10 shrink-0 items-center gap-2 rounded-full border px-3.5 text-sm font-semibold transition-colors ${
              panelOpen || filterCount > 0
                ? "border-brand bg-brand/10 text-brand"
                : "border-line-strong text-ink-muted hover:border-brand hover:text-brand"
            }`}
          >
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="h-4 w-4"
              aria-hidden
            >
              {panelOpen ? (
                <path d="M18 6 6 18M6 6l12 12" />
              ) : (
                <path d="M3 5h18l-7 8v6l-4 2v-8L3 5Z" />
              )}
            </svg>
            <span className="hidden sm:inline">
              {panelOpen ? t.search.filtersClose : t.search.filters}
            </span>
            {filterCount > 0 && (
              <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-brand px-1 text-xs font-bold text-white">
                {filterCount}
              </span>
            )}
          </button>
        </div>

        {panelOpen && (
          <div className="container-page border-t border-line py-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label={t.search.sort}>
                <div className="flex flex-wrap gap-1.5">
                  {(
                    [
                      ["relevance", t.search.sortRelevance],
                      ["cheap", t.search.sortCheap],
                      ["expensive", t.search.sortExpensive],
                      ["popular", t.search.sortPopular],
                    ] as [MenuSort, string][]
                  ).map(([value, label]) => (
                    <Toggle
                      key={value}
                      on={filters.sort === value}
                      onClick={() => patch({ sort: value })}
                    >
                      {label}
                    </Toggle>
                  ))}
                </div>
              </Field>

              <Field label={t.search.price}>
                <div className="flex items-center gap-2">
                  <input
                    type="number"
                    inputMode="numeric"
                    min={0}
                    value={filters.minPrice ?? ""}
                    onChange={(e) =>
                      patch({ minPrice: e.target.value ? Number(e.target.value) : null })
                    }
                    placeholder={`${t.search.priceFrom} ${facets.min}`}
                    aria-label={t.search.priceFrom}
                    className="input h-9 w-full"
                  />
                  <span className="text-ink-muted">—</span>
                  <input
                    type="number"
                    inputMode="numeric"
                    min={0}
                    value={filters.maxPrice ?? ""}
                    onChange={(e) =>
                      patch({ maxPrice: e.target.value ? Number(e.target.value) : null })
                    }
                    placeholder={`${t.search.priceTo} ${facets.max}`}
                    aria-label={t.search.priceTo}
                    className="input h-9 w-full"
                  />
                </div>
              </Field>

              {groups.length > 1 && (
                <Field label={t.search.category} wide>
                  <div className="flex flex-wrap gap-1.5">
                    {groups.map((g) => (
                      <Toggle
                        key={g.category.id}
                        on={filters.categoryIds.includes(g.category.id)}
                        onClick={() =>
                          patch({
                            categoryIds: toggleIn(filters.categoryIds, g.category.id),
                          })
                        }
                      >
                        {contentName(g.category, lang)}
                      </Toggle>
                    ))}
                  </div>
                </Field>
              )}

              {facets.tags.length > 0 && (
                <Field label={t.search.tags} wide>
                  <div className="flex flex-wrap gap-1.5">
                    {facets.tags.map((tag) => (
                      <Toggle
                        key={tag}
                        on={filters.tags.includes(tag)}
                        onClick={() => patch({ tags: toggleIn(filters.tags, tag) })}
                      >
                        {tag}
                      </Toggle>
                    ))}
                  </div>
                </Field>
              )}

              <Field wide>
                <div className="flex flex-wrap gap-1.5">
                  <Toggle
                    on={filters.availableOnly}
                    onClick={() => patch({ availableOnly: !filters.availableOnly })}
                  >
                    {t.search.availableOnly}
                  </Toggle>
                  <Toggle
                    on={filters.popularOnly}
                    onClick={() => patch({ popularOnly: !filters.popularOnly })}
                  >
                    ★ {t.search.popularOnly}
                  </Toggle>
                  {facets.hasDiscount && (
                    <Toggle
                      on={filters.discountOnly}
                      onClick={() => patch({ discountOnly: !filters.discountOnly })}
                    >
                      {t.search.discountOnly}
                    </Toggle>
                  )}
                  {facets.hasCombo && (
                    <Toggle
                      on={filters.comboOnly}
                      onClick={() => patch({ comboOnly: !filters.comboOnly })}
                    >
                      {t.search.comboOnly}
                    </Toggle>
                  )}
                </div>
              </Field>
            </div>

            {searching && (
              <div className="mt-4 flex items-center justify-between gap-3 border-t border-line pt-3 text-sm">
                <span className="text-ink-muted">{t.search.found(results.length)}</span>
                <button
                  type="button"
                  onClick={reset}
                  className="font-semibold text-brand hover:underline"
                >
                  {t.search.reset}
                </button>
              </div>
            )}
          </div>
        )}

        {/* The category rail is a browsing tool: its anchors point at sections
            that are not on screen while results are. */}
        {!searching && (
          <div className="container-page no-scrollbar flex gap-2 overflow-x-auto py-3">
            {groups.map((g) => (
              <a
                key={g.category.id}
                href={`#cat-${g.category.slug || g.category.id}`}
                className="chip"
              >
                {contentName(g.category, lang)}
                <span className="text-xs font-normal text-ink-muted">
                  {g.items.length}
                </span>
              </a>
            ))}
          </div>
        )}
      </div>

      {searching ? (
        <div className="container-page py-10">
          <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-ink-muted">{t.search.found(results.length)}</p>
            <button
              type="button"
              onClick={reset}
              className="text-sm font-semibold text-brand hover:underline"
            >
              {t.search.reset}
            </button>
          </div>
          {results.length === 0 ? (
            <div className="py-16 text-center">
              <p className="font-display text-xl font-bold">{t.search.nothing}</p>
              <p className="mx-auto mt-2 max-w-sm text-ink-muted">
                {t.search.nothingHint}
              </p>
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-3 sm:gap-5 lg:grid-cols-3">
              {results.map((item) => (
                <MenuItemCard key={item.id} item={item} currency={currency} />
              ))}
            </div>
          )}
        </div>
      ) : (
        <div className="container-page space-y-16 py-12">
          {groups.map((g) => {
            const catImg = imageUrl(g.category.imageUrl, 600);
            return (
              <section
                key={g.category.id}
                id={`cat-${g.category.slug || g.category.id}`}
                // ⚠️ How far a tapped category stops short so its heading clears
                // everything pinned above it: the header (h-16 / sm:h-20) plus
                // this bar's two rows, the search line and the category rail.
                // Measured against the rendered page rather than added up from
                // padding classes — the last time these were arithmetic they
                // drifted 44px behind the component they describe and left a
                // strip of page showing through.
                className="scroll-mt-[192px] sm:scroll-mt-[208px]"
              >
                <div className="mb-6 flex items-center gap-4">
                  {catImg && (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={catImg}
                      alt=""
                      loading="lazy"
                      className="h-14 w-14 rounded-2xl object-cover shadow-card sm:h-16 sm:w-16"
                    />
                  )}
                  <div>
                    <h2 className="font-display text-2xl font-bold tracking-tight sm:text-3xl">
                      {contentName(g.category, lang)}
                    </h2>
                    <p className="text-sm text-ink-muted">
                      {t.common.dishes(g.items.length)}
                    </p>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3 sm:gap-5 lg:grid-cols-3">
                  {g.items.map((item) => (
                    <MenuItemCard key={item.id} item={item} currency={currency} />
                  ))}
                </div>
              </section>
            );
          })}
        </div>
      )}
    </>
  );
}

function Field({
  label,
  wide,
  children,
}: {
  label?: string;
  wide?: boolean;
  children: React.ReactNode;
}) {
  return (
    <div className={wide ? "sm:col-span-2" : ""}>
      {label && (
        <p className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-ink-muted">
          {label}
        </p>
      )}
      {children}
    </div>
  );
}

function Toggle({
  on,
  onClick,
  children,
}: {
  on: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={on}
      className={`rounded-full border px-3 py-1.5 text-sm font-medium transition-colors ${
        on
          ? "border-brand bg-brand/10 text-brand"
          : "border-line-strong text-ink-muted hover:border-brand hover:text-brand"
      }`}
    >
      {children}
    </button>
  );
}
