import Link from "next/link";
import { api, imageUrl } from "@/lib/api";
import { getSiteScope } from "@/lib/siteBrand.server";
import { formatPrice, weekdayName } from "@/lib/format";
import MenuItemCard from "@/components/menu/MenuItemCard";
import { getTranslations } from "@/lib/i18n/server";
import { localized } from "@/lib/i18n/site-content";
import { contentName } from "@/lib/i18n/content";
import type { MenuGroup, RestaurantResponse } from "@/lib/types";

// Icons pair up with t.home.perks (same order).
const PERK_ICONS = [
  "M3 13h11V6H3v7Zm11 0 3-4h4v4M6.5 19a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3Zm11 0a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3Z",
  "M12 21c4.97 0 9-4.03 9-9-4.97 0-9 4.03-9 9Zm0 0c0-4.97-4.03-9-9-9 0 4.97 4.03 9 9 9Zm0 0V9",
  "M3 7h18v10H3V7Zm0 4h18M7 15h3",
];

export async function generateMetadata() {
  const { lang, t } = await getTranslations();
  try {
    const rest = (await api.getRestaurant(await getSiteScope())).restaurant;
    return {
      // The home page is the site root, so it keeps the plain restaurant name.
      title: { absolute: rest.name || t.common.restaurant },
      description:
        localized(rest.content?.tagline, lang, rest.description || "") ||
        t.home.heroFallback,
    };
  } catch {
    return { title: { absolute: t.common.restaurant } };
  }
}

export default async function HomePage() {
  const { lang, t } = await getTranslations();

  let data: RestaurantResponse | null = null;
  let menu: MenuGroup[] = [];
  try {
    const scope = await getSiteScope();
    [data, menu] = await Promise.all([
      api.getRestaurant(scope),
      api.getMenu(scope),
    ]);
  } catch {
    // backend unreachable — render an empty-state hero
  }

  const rest = data?.restaurant;
  const currency = rest?.currency ?? "UZS";
  const cover = imageUrl(rest?.coverUrl);

  // Popular items across all categories (fallback: first few items).
  const allItems = menu.flatMap((g) => g.items);
  const popular = allItems.filter((i) => i.isPopular);
  const featured = (popular.length > 0 ? popular : allItems).slice(0, 8);
  const categories = menu.filter((g) => g.items.length > 0);
  const minPrice = allItems.length ? Math.min(...allItems.map((i) => i.price)) : 0;

  // Order working hours Monday-first for display.
  const hours = [...(rest?.workingHours ?? [])].sort(
    (a, b) => ((a.day + 6) % 7) - ((b.day + 6) % 7),
  );
  const todayIdx = new Date().getDay();

  return (
    <main>
      {/* ---- Hero ---- */}
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

      {/* ---- Perks ---- */}
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

      {/* ---- Categories ---- */}
      {categories.length > 0 && (
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
              const img = imageUrl(g.category.imageUrl || g.items[0]?.imageUrl);
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
                    <p className="text-xs text-white/60">{t.common.dishes(g.items.length)}</p>
                  </div>
                </Link>
              );
            })}
          </div>
        </section>
      )}

      {/* ---- Popular items ---- */}
      {featured.length > 0 && (
        <section className="border-y border-line bg-surface">
          <div className="container-page py-16 sm:py-20">
            <p className="eyebrow">{t.home.popEyebrow}</p>
            <div className="mt-2 flex flex-wrap items-end justify-between gap-4">
              <h2 className="section-title">{t.home.popTitle}</h2>
              <Link
                href="/menu"
                className="text-sm font-semibold text-brand hover:underline"
              >
                {t.home.popAll}
              </Link>
            </div>
            <div className="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
              {featured.map((item) => (
                <MenuItemCard key={item.id} item={item} currency={currency} />
              ))}
            </div>
          </div>
        </section>
      )}

      {/* ---- Working hours + contact ---- */}
      <section className="container-page py-16 sm:py-20">
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          {hours.length > 0 && (
            <div className="card p-6 sm:p-8">
              <p className="eyebrow">{t.home.hoursEyebrow}</p>
              <h2 className="mt-2 font-display text-2xl font-bold">{t.home.hoursTitle}</h2>
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
                    <a href={`tel:${p}`} className="hover:text-white">
                      {p}
                    </a>
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
    </main>
  );
}
