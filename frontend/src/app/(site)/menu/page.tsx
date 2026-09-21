// Locale-aware: hrefs stay unprefixed here and gain /ru or /en at render.
import Link from "@/components/site/LocaleLink";
import { api } from "@/lib/api";
import { getSiteScope } from "@/lib/siteBrand.server";
import MenuBrowser from "@/components/menu/MenuBrowser";
import { getTranslations } from "@/lib/i18n/server";
import { siteWords } from "@/lib/siteWords";
import { guardCatalogRoute } from "@/lib/catalogRoute";
import type { MenuGroup, RestaurantResponse } from "@/lib/types";

export async function generateMetadata() {
  const { lang, t } = await getTranslations();
  return { title: t.menu.title };
}

export default async function MenuPage({
  searchParams,
}: {
  // `?q=` is how the home page's search box hands its query over. Read on the
  // server so the results are in the first paint rather than appearing after
  // hydration — the guest already typed, they should not watch it happen twice.
  //
  // ⚠️ It does not create a page: canonical is built from the path alone (see
  // lib/seo.ts and the middleware header), so every `/menu?q=…` still declares
  // `/menu` as the real URL and no search term becomes an indexable duplicate.
  // ⚠️ Typed loosely because this page is also where a table QR and a branch
  // link arrive, and the redirect below has to carry all of it across.
  searchParams?: Promise<Record<string, string | string[] | undefined>>;
}) {
  const asked = await searchParams;
  // ⚠️ A shop's catalogue lives at `/catalog`; this address stays for every
  // link that was ever written down and sends the guest on — with the query it
  // arrived with. See lib/catalogRoute.
  await guardCatalogRoute(asked);
  const { lang, t } = await getTranslations();
  const initialQuery = String(asked?.q ?? "").slice(0, 100);

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
  // ⚠️ "Katalog" in a shop, "Menyu" in a restaurant — see lib/siteWords.ts.
  const w = siteWords(t, rest?.brand?.businessType);
  const nonEmpty = menu.filter((g) => g.items.length > 0);
  const total = nonEmpty.reduce((n, g) => n + g.items.length, 0);

  if (nonEmpty.length === 0) {
    return (
      <main className="container-page py-28 text-center">
        <h1 className="section-title">{w.title}</h1>
        <p className="mx-auto mt-4 max-w-md text-ink-muted">{w.empty}</p>
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
          <p className="eyebrow">{w.eyebrow}</p>
          <h1 className="mt-2 section-title">{w.title}</h1>
          <p className="mt-3 max-w-xl text-ink-muted">
            {w.subtitle(nonEmpty.length, total)}
          </p>
        </div>
      </section>

      {/* Search, filters and the menu itself. A client component, but one that
          renders the whole grouped menu on the server in its default state —
          the dish list is what this page is found by. */}
      <MenuBrowser
        groups={nonEmpty}
        currency={currency}
        initialQuery={initialQuery}
        businessType={rest?.brand?.businessType}
      />
    </main>
  );
}
