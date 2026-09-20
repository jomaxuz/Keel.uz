"use client";

// The site's navigation bar, as a list.
//
// ⚠️ **Why this screen exists at all.** Every other part of a restaurant's page
// is drawn from the restaurant's own data — the bands carry no text, which is
// what makes a layout portable between customers. The navigation bar is the one
// exception, and an online store is why: every restaurant has the same five
// destinations because every restaurant is the same shape, a room with food in
// it. An online store's shape is whatever it sells. A boutique wants women /
// men / sale, a phone case shop wants brands, a book seller wants genres, and
// all three want the Telegram channel that brings them their customers. Left as
// a template, the first thing a guest sees on every one of those sites is a
// button marked "Bron" leading to a table-booking form.
//
// ⚠️ **Empty is not "a bar with no links" — it is "leave the header alone".**
// Every tenant on the platform has this unset, and reading absence as a
// decision would empty the header of every site at once. Same rule as
// `DEFAULT_SECTIONS`, an empty `mapProvider` and a zero `businessType`.
//
// ⚠️ **Drawn wins whole rather than merging with the built-in bar.** Half of
// somebody's design plus half of ours is a bar neither of them meant — and the
// reason an online store gets this at all is that our half is wrong for it.

import type { NavLink } from "@/lib/api";
import type { EditorDict } from "@/lib/i18n/editor";

/** What fits on a desktop bar beside a logo, a cart and a language switch.
 *  Mirrors `maxNavLinks` on the server, which enforces it. */
const MAX = 8;

const EMPTY: NavLink = {
  label: { uz: "", ru: "", en: "" },
  href: "",
};

export default function NavEditor({
  nav,
  setNav,
  d,
}: {
  nav: NavLink[];
  setNav: (next: NavLink[]) => void;
  d: EditorDict;
}) {
  const patch = (i: number, p: Partial<NavLink>) =>
    setNav(nav.map((l, k) => (k === i ? { ...l, ...p } : l)));

  // ⚠️ **Never `l.label.uz`, and this took the whole editor down once.**
  //
  // A link's three-language label is an object in a document written by
  // something else — an older console, a template, a restored backup, a hand
  // edit — and a link without one is not hypothetical. Read directly it threw
  // `Cannot read properties of undefined (reading 'uz')` inside the list's
  // `map`, which is not a broken row: React unmounts the whole tree, so the
  // constructor turned into the platform's error page mid-edit, with the
  // operator's unsaved work in it.
  //
  // Exactly the nil-slice lesson from CLAUDE.md, one field along: what the
  // server sends is never assumed to be shaped the way this component wants.
  const labelOf = (l: NavLink) => l.label ?? { uz: "", ru: "", en: "" };

  const label = (i: number, lang: "uz" | "ru" | "en", v: string) =>
    patch(i, { label: { ...labelOf(nav[i]), [lang]: v } });

  /** ⚠️ Swap rather than splice-and-insert: the bar's order is the only thing
   *  a move is meant to change, and a splice on the last row silently drops it
   *  when the index runs past the end. */
  const move = (i: number, by: number) => {
    const j = i + by;
    if (j < 0 || j >= nav.length) return;
    const next = [...nav];
    [next[i], next[j]] = [next[j], next[i]];
    setNav(next);
  };

  return (
    <div className="rounded-2xl border border-line p-3">
      <p className="text-xs font-bold text-ink">{d.navTitle}</p>
      <p className="mt-1 text-[11px] leading-relaxed text-ink-muted">
        {d.navHint}
      </p>

      {nav.length === 0 && (
        <p className="mt-3 rounded-xl border border-dashed border-line p-3 text-center text-[11px] text-ink-muted">
          {d.navEmpty}
        </p>
      )}

      <div className="mt-3 space-y-3">
        {nav.map((l, i) => (
          <div key={i} className="rounded-xl border border-line bg-surface p-2">
            <div className="flex items-center gap-1">
              <span className="w-5 text-center text-[11px] font-bold text-ink-muted">
                {i + 1}
              </span>
              {/* ⚠️ Three boxes, not one. The bar is typed per language and a
                  bar that is right in Uzbek and blank in Russian is invisible
                  to whoever typed it — the guest is the one who finds out. */}
              <input
                value={labelOf(l).uz}
                onChange={(e) => label(i, "uz", e.target.value)}
                placeholder="UZ"
                className="min-w-0 flex-1 rounded-lg border border-line bg-raised px-2 py-1 text-[11px] text-ink"
                aria-label={`${d.navLabel} UZ`}
              />
              <input
                value={labelOf(l).ru}
                onChange={(e) => label(i, "ru", e.target.value)}
                placeholder="RU"
                className="min-w-0 flex-1 rounded-lg border border-line bg-raised px-2 py-1 text-[11px] text-ink"
                aria-label={`${d.navLabel} RU`}
              />
              <input
                value={labelOf(l).en}
                onChange={(e) => label(i, "en", e.target.value)}
                placeholder="EN"
                className="min-w-0 flex-1 rounded-lg border border-line bg-raised px-2 py-1 text-[11px] text-ink"
                aria-label={`${d.navLabel} EN`}
              />
            </div>

            <input
              value={l.href ?? ""}
              onChange={(e) => patch(i, { href: e.target.value })}
              placeholder="/menu?cat=ayollar"
              spellCheck={false}
              className="mt-2 w-full rounded-lg border border-line bg-raised px-2 py-1 font-mono text-[11px] text-ink"
              aria-label={d.navHref}
            />

            <div className="mt-2 flex flex-wrap items-center gap-3 text-[11px] text-ink-muted">
              {/* ⚠️ Offered only on an address that actually leaves the site:
                  a new tab on a path of ours lands the guest in a second copy
                  of the shop with an empty basket. The server clears it too —
                  this is so nobody ticks it and wonders where it went. */}
              {(l.href ?? "").startsWith("http") && (
                <label className="flex items-center gap-1">
                  <input
                    type="checkbox"
                    checked={!!l.external}
                    onChange={(e) => patch(i, { external: e.target.checked })}
                  />
                  {d.navExternal}
                </label>
              )}
              {/* Kept in the document, not drawn: trying a bar and putting a
                  link back must not mean retyping it in three languages. */}
              <label className="flex items-center gap-1">
                <input
                  type="checkbox"
                  checked={!!l.hidden}
                  onChange={(e) => patch(i, { hidden: e.target.checked })}
                />
                {d.navHidden}
              </label>
              <span className="ml-auto flex gap-1">
                <button
                  type="button"
                  title={d.navUp}
                  onClick={() => move(i, -1)}
                  className="rounded-lg border border-line px-2 py-0.5 hover:text-ink"
                >
                  ↑
                </button>
                <button
                  type="button"
                  title={d.navDown}
                  onClick={() => move(i, 1)}
                  className="rounded-lg border border-line px-2 py-0.5 hover:text-ink"
                >
                  ↓
                </button>
                <button
                  type="button"
                  title={d.navRemove}
                  onClick={() => setNav(nav.filter((_, k) => k !== i))}
                  className="rounded-lg border border-line px-2 py-0.5 hover:text-ink"
                >
                  ×
                </button>
              </span>
            </div>
          </div>
        ))}
      </div>

      {nav.length < MAX && (
        <button
          type="button"
          onClick={() => setNav([...nav, { ...EMPTY, label: { ...EMPTY.label } }])}
          className="mt-3 w-full rounded-xl border border-dashed border-line py-2 text-[11px] font-semibold text-ink-muted hover:text-ink"
        >
          + {d.navAdd}
        </button>
      )}

      <p className="mt-2 text-[11px] leading-relaxed text-ink-muted">
        {d.navHrefHint}
      </p>
    </div>
  );
}
