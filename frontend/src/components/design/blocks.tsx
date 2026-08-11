// The page, as blocks.
//
// ⚠️ **The first five are the bands this site already renders, moved here
// unchanged.** That is the whole point: with no design drawn, `DesignRenderer`
// walks the default list and produces the page the restaurant already has. A
// block set built from the new ideas would have quietly dropped the perks strip
// from every existing site, and "existing customers see no change" is the
// precondition of the entire feature — not a nicety.
//
// So this file is deliberately a **move, not a rewrite**. The markup, the
// classes and the comments came across as they were; anything that looks like it
// could be tidier probably could be, and tidying it here would make the
// before/after impossible to verify.
//
// Blocks receive one `BlockData` object holding everything the page already
// computed, and each picks what it needs. That keeps the extraction mechanical
// and stops the renderer from having to know which block wants what.

import Link from "@/components/site/LocaleLink";
import { imageUrl } from "@/lib/api";
import HomeSearch from "@/components/menu/HomeSearch";
import { formatPrice, weekdayName } from "@/lib/format";
import MenuItemCard from "@/components/menu/MenuItemCard";
import { localized } from "@/lib/i18n/site-content";
import CallLink from "@/components/site/CallLink";
import { contentName } from "@/lib/i18n/content";
import { TONE } from "./tokens";
import type { Dict, Lang } from "@/lib/i18n/dictionaries";
import type {
  DesignSection,
  MenuGroup,
  MenuItem,
  RestaurantResponse,
} from "@/lib/types";

/** Everything the page loaded, in one bag. */
export interface BlockData {
  t: Dict;
  lang: Lang;
  data: RestaurantResponse | null;
  menu: MenuGroup[];
  currency: string;
}

// Icons pair up with t.home.perks (same order).
const PERK_ICONS = [
  "M3 13h11V6H3v7Zm11 0 3-4h4v4M6.5 19a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3Zm11 0a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3Z",
  "M12 21c4.97 0 9-4.03 9-9-4.97 0-9 4.03-9 9Zm0 0c0-4.97-4.03-9-9-9 0 4.97 4.03 9 9 9Zm0 0V9",
  "M3 7h18v10H3V7Zm0 4h18M7 15h3",
];

/** The dishes a `menu-grid` band shows.
 *
 *  The choice is of a *source*, never of words: popular dishes, or the categories
 *  somebody picked. Today's home page shows popular-with-fallback, which is what
 *  the default list asks for.
 *
 *  ⚠️ **Settings first, then the older `binding`.** The console draws this band's
 *  panel from the schema, which writes `settings`; the templates and every design
 *  drawn before the schema existed carry `binding`. Reading only `binding` — which
 *  is what this did — meant the panel's three controls changed a document and
 *  nothing else: a form. That is the failure SchemaBlocks was written to end, and
 *  `menu-grid` was the band left behind by it, because its renderer lives here
 *  rather than there.
 *
 *  The fallback is per field, not per band: a design saved by the newer console
 *  sets `categories` while its `popularOnly` still sits in `binding`, and an
 *  all-or-nothing read would quietly drop half of what somebody chose. */
function pickItems(d: BlockData, section: DesignSection): MenuItem[] {
  const s = (section.settings ?? {}) as Record<string, unknown>;
  const bind = section.binding ?? {};
  const chosen = Array.isArray(s.categories)
    ? (s.categories as unknown[]).filter((v): v is string => typeof v === "string")
    : bind.categories;
  const b = {
    categories: chosen,
    popularOnly:
      typeof s.popularOnly === "boolean" ? s.popularOnly : bind.popularOnly,
    limit: typeof s.limit === "number" && s.limit > 0 ? s.limit : bind.limit,
  };
  // ⚠️ A category chosen here and later deleted from the menu leaves an id that
  // matches nothing. Falling back to the whole menu would be wrong — the band
  // would silently start showing everything — but so is an empty band, which
  // reads as a broken site. Keeping only what still exists means deleting one of
  // three categories narrows the band instead of breaking it.
  const picked = b.categories?.length
    ? d.menu.filter((g) => b.categories!.includes(g.category.id))
    : [];
  const groups = b.categories?.length ? (picked.length > 0 ? picked : d.menu) : d.menu;
  const all = groups.flatMap((g) => g.items);
  if (b.popularOnly) {
    const popular = all.filter((i) => i.isPopular);
    // Fallback to everything when nothing is marked popular: an empty band on a
    // menu full of dishes reads as a broken site.
    return (popular.length > 0 ? popular : all).slice(0, b.limit || 8);
  }
  return all.slice(0, b.limit || 8);
}

// ---- hero ----

export function HeroBlock({ d }: { d: BlockData; section: DesignSection }) {
  const { t, lang, data, currency } = d;
  const rest = data?.restaurant;
  const cover = imageUrl(rest?.coverUrl, 1200);
  const allItems = d.menu.flatMap((g) => g.items);
  const minPrice = allItems.length ? Math.min(...allItems.map((i) => i.price)) : 0;

  return (
    <section className="relative overflow-hidden bg-charcoal text-white">
      {cover && (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={cover}
          alt=""
          className="absolute inset-0 h-full w-full object-cover opacity-30"
        />
      )}
      <div className="absolute inset-0 bg-gradient-to-br from-charcoal via-charcoal/85 to-charcoal/40" />
      {/* warm glow */}
      <div className="pointer-events-none absolute -right-24 -top-24 h-96 w-96 rounded-full bg-brand/30 blur-3xl" />

      <div className="container-page relative py-20 sm:py-28">
        <div className="max-w-2xl animate-fade-up">
          {data && (
            <span
              className={`badge ${
                data.isOpenNow
                  ? "bg-emerald-500/15 text-emerald-300"
                  : "bg-white/10 text-white/70"
              }`}
            >
              <span
                className={`h-1.5 w-1.5 rounded-full ${
                  data.isOpenNow ? "bg-emerald-400" : "bg-white/50"
                }`}
              />
              {data.isOpenNow ? t.common.openNow : t.common.closedNow}
            </span>
          )}

          {/* 48px of display type is a lot of screen for a restaurant with a
              long name — a phone starts one size down. */}
          <h1 className="mt-5 break-words font-display text-4xl font-bold leading-[1.08] tracking-tight sm:text-5xl lg:text-6xl">
            {rest?.name ?? t.common.restaurant}
          </h1>
          <p className="mt-5 max-w-xl text-lg leading-relaxed text-white/70">
            {localized(
              rest?.content?.tagline,
              lang,
              rest?.description ?? t.home.heroFallback,
            )}
          </p>

          <div className="mt-9 flex flex-wrap gap-3">
            <Link href="/menu" className="btn-primary px-7 py-3.5 text-base">
              {t.home.ctaMenu}
            </Link>
            <Link
              href="/about"
              className="btn px-7 py-3.5 text-base text-white ring-1 ring-white/25 transition-colors hover:bg-white/10"
            >
              {t.home.ctaAbout}
            </Link>
          </div>

          {allItems.length > 0 && (
            <dl className="mt-12 flex flex-wrap gap-x-10 gap-y-5">
              <div>
                <dt className="text-xs uppercase tracking-widest text-white/40">
                  {t.home.statDishes}
                </dt>
                <dd className="mt-1 font-display text-2xl font-bold">
                  {allItems.length}+
                </dd>
              </div>
              {minPrice > 0 && (
                <div>
                  <dt className="text-xs uppercase tracking-widest text-white/40">
                    {t.home.statPrices}
                  </dt>
                  <dd className="mt-1 font-display text-2xl font-bold">
                    {t.common.priceFrom(formatPrice(minPrice, currency, lang))}
                  </dd>
                </div>
              )}
              {rest?.delivery?.enabled && rest.delivery.minOrder > 0 && (
                <div>
                  <dt className="text-xs uppercase tracking-widest text-white/40">
                    {t.home.statMinOrder}
                  </dt>
                  <dd className="mt-1 font-display text-2xl font-bold">
                    {formatPrice(rest.delivery.minOrder, currency, lang)}
                  </dd>
                </div>
              )}
            </dl>
          )}
        </div>
      </div>
    </section>
  );
}

// ---- perks ----

export function PerksBlock({ d }: { d: BlockData; section: DesignSection }) {
  const { t } = d;
  return (
    <section className="container-page relative z-10 -mt-10">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        {t.home.perks.map((p, i) => (
          <div key={p.title} className="card p-6">
            <span className="flex h-11 w-11 items-center justify-center rounded-2xl bg-brand-tint text-brand">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="h-5 w-5"
                aria-hidden
              >
                <path d={PERK_ICONS[i]} />
              </svg>
            </span>
            <h3 className="mt-4 font-display text-lg font-bold">{p.title}</h3>
            <p className="mt-1.5 text-sm leading-relaxed text-ink-muted">
              {p.text}
            </p>
          </div>
        ))}
      </div>
    </section>
  );
}

// ---- categories ----

/** A search box, on the page the guest lands on.
 *
 *  The band owns nothing but its padding: the box itself is the same module the
 *  menu page searches with, so a dish found here and a dish found there are
 *  found the same way. Draws nothing when the menu is empty — a search over
 *  nothing is a box that answers "not found" to everything.
 */
export function SearchBlock({ d, section }: { d: BlockData; section: DesignSection }) {
  const groups = d.menu.filter((g) => g.items.length > 0);
  if (groups.length === 0) return null;
  const big = section.variant === "big";
  return (
    <section className={`container-page ${big ? "py-12 sm:py-16" : "py-6"}`}>
      <div className={big ? "mx-auto max-w-2xl" : "mx-auto max-w-xl"}>
        <HomeSearch menu={groups} currency={d.currency} big={big} />
      </div>
    </section>
  );
}

export function CategoriesBlock({ d }: { d: BlockData; section: DesignSection }) {
  const { t, lang } = d;
  const categories = d.menu.filter((g) => g.items.length > 0);
  if (categories.length === 0) return null;

  return (
    <section className="container-page py-16 sm:py-20">
      <p className="eyebrow">{t.home.catsEyebrow}</p>
      <div className="mt-2 flex flex-wrap items-end justify-between gap-4">
        <h2 className="section-title">{t.home.catsTitle}</h2>
        <Link
          href="/menu"
          className="text-sm font-semibold text-brand hover:underline"
        >
          {t.home.catsAll}
        </Link>
      </div>

      <div className="mt-8 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
        {categories.map((g) => {
          const img = imageUrl(g.category.imageUrl || g.items[0]?.imageUrl, 600);
          return (
            <Link
              key={g.category.id}
              href={`/menu#cat-${g.category.slug || g.category.id}`}
              className="group relative overflow-hidden rounded-3xl bg-charcoal shadow-card transition-shadow hover:shadow-card-hover"
            >
              <div className="aspect-[4/3] w-full">
                {img && (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={img}
                    alt=""
                    loading="lazy"
                    className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105"
                  />
                )}
              </div>
              <div className="absolute inset-0 bg-gradient-to-t from-charcoal/85 via-charcoal/15 to-transparent" />
              <div className="absolute inset-x-0 bottom-0 p-4">
                <h3 className="font-display text-lg font-bold text-white">
                  {contentName(g.category, lang)}
                </h3>
                <p className="text-xs text-white/60">
                  {t.common.dishes(g.items.length)}
                </p>
              </div>
            </Link>
          );
        })}
      </div>
    </section>
  );
}

// ---- menu-grid ----

export function MenuGridBlock({
  d,
  section,
}: {
  d: BlockData;
  section: DesignSection;
}) {
  const { t, currency, lang } = d;
  const s = (section.settings ?? {}) as Record<string, unknown>;
  const items = pickItems(d, section);
  if (items.length === 0) return null;

  // The schema offers a heading and a background for this band, so both are read
  // here — an unread control is the same lie as an unread binding, just smaller.
  // Empty falls back to the dictionary's own wording, which is what every
  // existing site shows and what a band added and left alone should show.
  const heading =
    (typeof s.heading === "string"
      ? s.heading
      : s.heading
        ? localized(s.heading as never, lang)
        : "") || t.home.popTitle;
  const tone = typeof s.tone === "string" ? TONE[s.tone] : undefined;

  return (
    <section className={`border-y border-line ${tone ?? "bg-surface"}`}>
      <div className="container-page py-16 sm:py-20">
        <p className="eyebrow">{t.home.popEyebrow}</p>
        <div className="mt-2 flex flex-wrap items-end justify-between gap-4">
          <h2 className="section-title">{heading}</h2>
          <Link
            href="/menu"
            className="text-sm font-semibold text-brand hover:underline"
          >
            {t.home.popAll}
          </Link>
        </div>
        <div className="mt-8 grid grid-cols-2 gap-3 sm:gap-5 sm:grid-cols-2 lg:grid-cols-4">
          {items.map((item) => (
            <MenuItemCard key={item.id} item={item} currency={currency} />
          ))}
        </div>
      </div>
    </section>
  );
}

// ---- hours-address ----

export function HoursAddressBlock({ d }: { d: BlockData; section: DesignSection }) {
  const { t, lang, data } = d;
  const rest = data?.restaurant;
  // Order working hours Monday-first for display.
  const hours = [...(rest?.workingHours ?? [])].sort(
    (a, b) => ((a.day + 6) % 7) - ((b.day + 6) % 7),
  );
  const todayIdx = new Date().getDay();

  return (
    <section className="container-page py-16 sm:py-20">
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {hours.length > 0 && (
          <div className="card p-6 sm:p-8">
            <p className="eyebrow">{t.home.hoursEyebrow}</p>
            <h2 className="mt-2 font-display text-2xl font-bold">
              {t.home.hoursTitle}
            </h2>
            <div className="mt-5 divide-y divide-line">
              {hours.map((h) => (
                <div
                  key={h.day}
                  className={`flex items-center justify-between py-2.5 text-sm ${
                    h.day === todayIdx ? "font-bold text-brand" : ""
                  }`}
                >
                  <span className="flex items-center gap-2">
                    {weekdayName(h.day, lang)}
                    {h.day === todayIdx && (
                      <span className="badge-soft">{t.common.today}</span>
                    )}
                  </span>
                  <span className={h.day === todayIdx ? "" : "text-ink-muted"}>
                    {h.isClosed ? t.common.closed : `${h.open} – ${h.close}`}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="flex flex-col justify-between gap-6 rounded-3xl bg-charcoal p-6 text-white sm:p-8">
          <div>
            <p className="text-xs font-bold uppercase tracking-[0.18em] text-brand-light">
              {t.home.orderEyebrow}
            </p>
            <h2 className="mt-2 font-display text-2xl font-bold">
              {t.home.orderTitle}
            </h2>
            <p className="mt-3 text-sm leading-relaxed text-white/60">
              {t.home.orderText}
            </p>

            <ul className="mt-6 space-y-2 text-sm text-white/70">
              {rest?.phones?.map((p) => (
                <li key={p}>
                  <CallLink phone={p} className="hover:text-white">
                    {p}
                  </CallLink>
                </li>
              ))}
              {rest?.address?.text && (
                <li className="text-white/50">{rest.address.text}</li>
              )}
            </ul>
          </div>

          <div className="flex flex-wrap gap-3">
            <Link href="/menu" className="btn-primary px-6 py-3">
              {t.home.orderBtn}
            </Link>
            <Link
              href="/about"
              className="btn px-6 py-3 text-white ring-1 ring-white/25 hover:bg-white/10"
            >
              {t.home.addressBtn}
            </Link>
          </div>
        </div>
      </div>
    </section>
  );
}

// ---- about ----
//
// New in the constructor, and built from copy the settings page already edits
// (`content.aboutTitle` / `aboutText`, each in three languages). Nothing here is
// typed into the design.

export function AboutBlock({
  d,
  section,
}: {
  d: BlockData;
  section: DesignSection;
}) {
  const { t, lang, data } = d;
  const rest = data?.restaurant;
  // The same fallback chain the /about page uses: the edited title, then the
  // restaurant's name. Nothing is typed into the design.
  const title =
    localized(rest?.content?.aboutTitle, lang) ||
    rest?.name ||
    t.common.restaurant;
  const text = localized(rest?.content?.aboutText, lang, rest?.description ?? "");
  if (!text) return null;

  const image = imageUrl(rest?.coverUrl, 600);
  const withImage = section.variant === "text-image" && image;

  return (
    <section className="container-page py-16 sm:py-20">
      <div
        className={
          withImage ? "grid grid-cols-1 gap-8 lg:grid-cols-2 lg:items-center" : ""
        }
      >
        <div className={withImage ? "" : "max-w-2xl"}>
          <p className="eyebrow">{t.about.eyebrow}</p>
          <h2 className="section-title mt-2">{title}</h2>
          <p className="mt-5 whitespace-pre-line leading-relaxed text-ink-soft">
            {text}
          </p>
        </div>
        {withImage && (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={image}
            alt=""
            loading="lazy"
            className="aspect-[4/3] w-full rounded-3xl object-cover shadow-card"
          />
        )}
      </div>
    </section>
  );
}

// ---- gallery ----

export function GalleryBlock({
  d,
  section,
}: {
  d: BlockData;
  section: DesignSection;
}) {
  // Built from the photographs the restaurant already uploaded — dish images,
  // which is the only pool of pictures every tenant has. A gallery that needed
  // its own uploads would be empty on every site until somebody filled it.
  const items = d.menu
    .flatMap((g) => g.items)
    .filter((i) => i.imageUrl)
    .slice(0, section.binding?.limit || 8);
  if (items.length === 0) return null;

  const strip = section.variant === "strip";
  return (
    <section className="container-page py-16 sm:py-20">
      <p className="eyebrow">{d.t.home.gallery.eyebrow}</p>
      <h2 className="section-title mt-2">{d.t.home.gallery.title}</h2>
      <div
        className={
          strip
            ? "mt-8 flex snap-x gap-4 overflow-x-auto pb-2"
            : "mt-8 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4"
        }
      >
        {items.map((item) => (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            key={item.id}
            src={imageUrl(item.imageUrl, 600) ?? ""}
            alt=""
            loading="lazy"
            className={`aspect-square rounded-2xl object-cover ${
              strip ? "w-40 shrink-0 snap-start sm:w-56" : "w-full"
            }`}
          />
        ))}
      </div>
    </section>
  );
}

// ---- cta ----

export function CtaBlock({
  d,
  section,
}: {
  d: BlockData;
  section: DesignSection;
}) {
  const { t, data } = d;
  const booking = data?.restaurant?.booking?.enabled;
  const banner = section.variant !== "buttons";

  return (
    <section className="container-page py-12 sm:py-16">
      <div
        className={`flex flex-col gap-5 rounded-3xl p-8 sm:p-10 ${
          banner ? "bg-charcoal text-white" : "border border-line bg-surface"
        } sm:flex-row sm:items-center sm:justify-between`}
      >
        <div>
          <h2 className="font-display text-2xl font-bold">{t.home.orderTitle}</h2>
          <p
            className={`mt-2 text-sm leading-relaxed ${
              banner ? "text-white/60" : "text-ink-muted"
            }`}
          >
            {t.home.orderText}
          </p>
        </div>
        <div className="flex flex-wrap gap-3">
          <Link href="/menu" className="btn-primary px-6 py-3">
            {t.home.orderBtn}
          </Link>
          {booking && (
            <Link
              href="/bron"
              className={`btn px-6 py-3 ${
                banner
                  ? "text-white ring-1 ring-white/25 hover:bg-white/10"
                  : "btn-ghost"
              }`}
            >
              {t.nav.booking}
            </Link>
          )}
        </div>
      </div>
    </section>
  );
}
