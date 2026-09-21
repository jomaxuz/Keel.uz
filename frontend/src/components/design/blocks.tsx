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
import { siteWords } from "@/lib/siteWords";
import CallLink from "@/components/site/CallLink";
import { contentName } from "@/lib/i18n/content";
import { TONE } from "./tokens";
import {
  DEFAULT_PERK_ICONS,
  PERK_ICON_NAMES,
  PerkIcon,
} from "./perkIcons";
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
  const { t, lang } = d;
  const content = d.data?.restaurant?.content;
  // Switched off from the panel. A promise the restaurant cannot keep — "30–45
  // minutes" on a place that does not deliver — is worse than no strip at all.
  if (content?.hidePerks) return null;
  // The restaurant's own cards when it wrote any, otherwise the built-in copy.
  // Empty means "never opened this section", not "wanted it blank": that is why
  // there is a separate hide flag rather than deleting the last card.
  const own = (content?.perks ?? []).filter(
    (p) => localized(p.title, lang).trim() || localized(p.text, lang).trim(),
  );
  const cards = own.length
    ? own.map((p, i) => ({
        title: localized(p.title, lang),
        text: localized(p.text, lang),
        icon: p.icon || PERK_ICON_NAMES[i % PERK_ICON_NAMES.length],
      }))
    : t.home.perks.map((p, i) => ({
        title: p.title,
        text: p.text,
        // The dictionary's three pair up with the first three names, in order.
        icon: DEFAULT_PERK_ICONS[i] ?? "star",
      }));
  return (
    // ⚠️ **No pull-up over the hero.** The strip used to sit on the hero's dark
    // edge with `-mt-10`, which was true when it came directly after it. It no
    // longer does — search and banners were added above — and worse, whether it
    // does is a *runtime* question: both of those bands draw nothing when there
    // is no menu or no banner. Spacing that depends on what its neighbours
    // decided to render is spacing that breaks without anyone changing it, and
    // what it broke into was the cards sitting on top of the search box.
    <section className="container-page py-10 sm:py-12">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        {cards.map((p, i) => (
          <div key={`${p.title}-${i}`} className="card p-6">
            <span className="flex h-11 w-11 items-center justify-center rounded-2xl bg-brand-tint text-brand">
              <PerkIcon name={p.icon} className="h-5 w-5" />
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

// ---- reviews ----

/** What guests said, on the restaurant's own page.
 *
 *  ⚠️ **Draws nothing unless there is something real to draw.** The server sends
 *  this key only when the setting is on, and only with the reviews an owner
 *  published one at a time — so an install that has never opened the section,
 *  and one that switched it on but has not chosen any words yet, both render
 *  exactly the page they rendered before. A heading over an empty strip is a
 *  restaurant advertising that nobody has said anything nice about it.
 *
 *  The average is drawn from **every** rating, including the ones not shown.
 *  Averaging only the published reviews would be a number the restaurant
 *  assembled about itself, printed in the place a visitor trusts most.
 */
export function ReviewsBlock({ d }: { d: BlockData; section: DesignSection }) {
  const { t } = d;
  const reviews = d.data?.reviews;
  if (!reviews) return null;
  const items = reviews.items ?? [];
  const showAverage = reviews.showAverage && reviews.count > 0;
  if (items.length === 0 && !showAverage) return null;

  return (
    <section className="container-page py-10 sm:py-14">
      <div className="flex flex-wrap items-baseline justify-between gap-3">
        <h2 className="section-title">{t.home.reviewsTitle}</h2>
        {showAverage && (
          <p className="flex items-baseline gap-2 text-sm text-ink-muted">
            <ReviewStars rating={Math.round(reviews.average)} />
            <span className="font-display text-xl font-bold text-ink">
              {reviews.average.toFixed(1)}
            </span>
            <span>{t.home.reviewsCount(reviews.count)}</span>
          </p>
        )}
      </div>
      {items.length > 0 && (
        <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {items.map((rv, i) => (
            <figure key={i} className="card flex h-full flex-col p-5">
              <ReviewStars rating={rv.rating} />
              <blockquote className="mt-3 flex-1 text-sm leading-relaxed text-ink-soft">
                {rv.comment}
              </blockquote>
              {/* First name only, and no date beyond the day: the guest wrote
                  this to the restaurant, and everything printed beside it is
                  something they did not choose to publish. */}
              <figcaption className="mt-4 text-sm font-semibold">
                {rv.name || t.home.reviewsAnon}
              </figcaption>
            </figure>
          ))}
        </div>
      )}
    </section>
  );
}

/** Five stars with `rating` of them filled.
 *
 *  All five are always drawn: three filled stars alone reads as a three-star
 *  scale, which flatters the restaurant by accident. */
function ReviewStars({ rating }: { rating: number }) {
  return (
    <span className="text-base leading-none text-amber-500" aria-hidden>
      {"★".repeat(Math.max(0, Math.min(5, rating)))}
      <span className="text-ink-muted/30">
        {"★".repeat(Math.max(0, 5 - Math.max(0, Math.min(5, rating))))}
      </span>
    </span>
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

  // ⚠️ **"Mijozlar tanlovi" over the whole catalogue is a lie the band told
  // itself.** The eyebrow was hardcoded to the popular-dishes wording, so a
  // grid bound to every product — which is what a shop's catalogue band is —
  // announced a selection nobody made. Explicit wins; otherwise it follows the
  // binding, and a business that sells goods says "our products" rather than
  // "our dishes" (lib/siteWords.ts).
  const popular = section.binding?.popularOnly !== false;
  const eyebrow =
    (typeof s.eyebrow === "string"
      ? s.eyebrow
      : s.eyebrow
        ? localized(s.eyebrow as never, lang)
        : "") ||
    (popular
      ? t.home.popEyebrow
      : siteWords(t, d.data?.brand?.businessType).eyebrow);

  return (
    <section className={`border-y border-line ${tone ?? "bg-surface"}`}>
      <div className="container-page py-16 sm:py-20">
        <p className="eyebrow" data-keel-set="eyebrow">{eyebrow}</p>
        <div className="mt-2 flex flex-wrap items-end justify-between gap-4">
          <h2 className="section-title" data-keel-set="heading">{heading}</h2>
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

export function HoursAddressBlock({ d, section }: { d: BlockData; section: DesignSection }) {
  const { t, lang, data } = d;
  // ⚠️ **The band's own heading, which it ignored.** The console declares a
  // `heading` setting for this band and draws a field for it; nothing read it,
  // so an operator typed a title, saved, published and the page kept saying
  // "Ish vaqti". Empty still means the built-in wording — every site on the
  // platform has this unset.
  // ⚠️ A shop is not hungry. See lib/siteWords.ts — the same decision as
  // «Menyu» / «Katalog», one band further down the page.
  const words = siteWords(t, data?.brand?.businessType);
  const titled = (() => {
    const raw = (section.settings ?? {})["heading"];
    if (!raw) return "";
    if (typeof raw === "string") return raw;
    return localized(raw as never, lang);
  })();
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
            <h2 className="mt-2 font-display text-2xl font-bold" data-keel-set="heading">
              {titled || t.home.hoursTitle}
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
              {words.orderTitle}
            </h2>
            <p className="mt-3 text-sm leading-relaxed text-white/60">
              {words.orderText}
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
  const ctaWords = siteWords(t, data?.brand?.businessType);

  return (
    <section className="container-page py-12 sm:py-16">
      <div
        className={`flex flex-col gap-5 rounded-3xl p-8 sm:p-10 ${
          banner ? "bg-charcoal text-white" : "border border-line bg-surface"
        } sm:flex-row sm:items-center sm:justify-between`}
      >
        <div>
          <h2 className="font-display text-2xl font-bold">{ctaWords.orderTitle}</h2>
          <p
            className={`mt-2 text-sm leading-relaxed ${
              banner ? "text-white/60" : "text-ink-muted"
            }`}
          >
            {ctaWords.orderText}
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
