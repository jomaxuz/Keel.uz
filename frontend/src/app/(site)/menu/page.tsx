// Locale-aware: hrefs stay unprefixed here and gain /ru or /en at render.
import Link from "@/components/site/LocaleLink";
import { api, imageUrl } from "@/lib/api";
import { getSiteScope } from "@/lib/siteBrand.server";
import MenuItemCard from "@/components/menu/MenuItemCard";
import { getTranslations } from "@/lib/i18n/server";
import { contentName } from "@/lib/i18n/content";
import type { MenuGroup, RestaurantResponse } from "@/lib/types";

export async function generateMetadata() {
  const { lang, t } = await getTranslations();
  return { title: t.menu.title };
}

export default async function MenuPage() {
  const { lang, t } = await getTranslations();

  let menu: MenuGroup[] = [];
  let rest: RestaurantResponse | null = null;
  try {
    const scope = await getSiteScope();
    [menu, rest] = await Promise.all([
      api.getMenu(scope),
      api.getRestaurant(scope),
    ]);
  } catch {
    // backend unreachable
  }

  const currency = rest?.restaurant.currency ?? "UZS";
  const nonEmpty = menu.filter((g) => g.items.length > 0);
  const total = nonEmpty.reduce((n, g) => n + g.items.length, 0);

  if (nonEmpty.length === 0) {
    return (
      <main className="container-page py-28 text-center">
        <h1 className="section-title">{t.menu.title}</h1>
        <p className="mx-auto mt-4 max-w-md text-ink-muted">{t.menu.empty}</p>
        <Link href="/" className="btn-ghost mt-8 px-6 py-3">
          {t.menu.home}
        </Link>
      </main>
    );
  }

  return (
    <main>
      {/* Page header */}
      <section className="border-b border-line bg-surface">
        <div className="container-page py-12 sm:py-16">
          <p className="eyebrow">{t.menu.eyebrow}</p>
          <h1 className="mt-2 section-title">{t.menu.title}</h1>
          <p className="mt-3 max-w-xl text-ink-muted">
            {t.menu.subtitle(nonEmpty.length, total)}
          </p>
        </div>
      </section>

      {/* Sticky category rail, pinned directly under the header.
          ⚠️ The offsets here are the header's own height and nothing else. They
          used to be 108px and 124px — the bar **plus** the second row of nav
          chips it had on a phone. When that row moved into the hamburger the
          header got 44px shorter and these numbers did not, so the rail floated
          44px below it with a strip of page showing through: an empty gap between
          the navbar and the categories that looked like a layout bug because it
          was one.
          A magic number that means "the height of another component" goes stale
          the moment that component changes, so it is written as the same Tailwind
          steps the header uses (h-16 / sm:h-20) rather than as pixels. */}
      <div className="sticky top-16 z-30 border-b border-line bg-cream/90 backdrop-blur-md sm:top-20">
        <div className="container-page no-scrollbar flex gap-2 overflow-x-auto py-3">
          {nonEmpty.map((g) => (
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
      </div>

      <div className="container-page space-y-16 py-12">
        {nonEmpty.map((g) => {
          const catImg = imageUrl(g.category.imageUrl, 600);
          return (
            <section
              key={g.category.id}
              id={`cat-${g.category.slug || g.category.id}`}
              // ⚠️ Header + rail, for the same reason as the offset above: this is
              // how far a tapped category has to stop short so its heading is not
              // hidden under both of them. 180px was the two-row header's total;
              // it is now 64+52 on a phone and 80+52 from `sm` up.
              className="scroll-mt-[116px] sm:scroll-mt-[132px]"
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
                  <p className="text-sm text-ink-muted">{t.common.dishes(g.items.length)}</p>
                </div>
              </div>

              <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
                {g.items.map((item) => (
                  <MenuItemCard key={item.id} item={item} currency={currency} />
                ))}
              </div>
            </section>
          );
        })}
      </div>
    </main>
  );
}
