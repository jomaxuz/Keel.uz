// Locale-aware: hrefs stay unprefixed here and gain /ru or /en at render.
import Link from "@/components/site/LocaleLink";
import { notFound } from "next/navigation";
import { api, imageUrl, ApiError } from "@/lib/api";
import { getSiteScope } from "@/lib/siteBrand.server";
import { formatPrice } from "@/lib/format";
import { hasId } from "@/lib/id";
import AddToCartControl from "@/components/menu/AddToCartControl";
import Recommendations from "@/components/menu/Recommendations";
import { getTranslations } from "@/lib/i18n/server";
import { contentDescription, contentName } from "@/lib/i18n/content";
import type { MenuGroup, MenuItem, RestaurantResponse } from "@/lib/types";

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const { lang } = await getTranslations();
  try {
    const item = await api.getMenuItem(id);
    const name = contentName(item, lang);
    const description = contentDescription(item, lang);
    const img = imageUrl(item.imageUrl, 1200);
    return {
      title: name,
      description: description || name,
      openGraph: {
        title: name,
        description: description || name,
        images: img ? [img] : undefined,
      },
    };
  } catch {
    return {};
  }
}

export default async function MenuItemPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const { lang, t } = await getTranslations();

  let item: MenuItem;
  let rest: RestaurantResponse | null = null;
  // ⚠️ **The whole catalogue, for the sizes.** A variant knows which model it
  // belongs to and nothing about its siblings, and the guest needs all of them
  // on one screen — choosing a size is the decision this page exists for in a
  // clothes shop. The menu is cached for a minute and this page is already
  // fetching two things; a third endpoint answering "the other sizes" would be
  // a second place the grouping rule lives.
  let catalogue: MenuGroup[] = [];
  try {
    const scope = await getSiteScope();
    [item, rest, catalogue] = await Promise.all([
      api.getMenuItem(id),
      api.getRestaurant(scope).catch(() => null),
      api.getMenu(scope).catch(() => [] as MenuGroup[]),
    ]);
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) notFound();
    throw err;
  }

  // The sizes and colours of this model, this one included. Empty for anything
  // that is not a variant, which is every dish ever written.
  //
  // ⚠️ **`hasId`, never a bare truthiness check.** A dish with no model arrives
  // with `variantOf: "000000000000000000000000"` — truthy in JavaScript — and
  // read naively every dish on the menu matched every other, so an ordinary
  // dish page listed the whole catalogue as its own sizes.
  const siblings = hasId(item.variantOf)
    ? catalogue
        .flatMap((g) => g.items)
        .filter((m) => m.variantOf === item.variantOf)
    : [];

  const currency = rest?.restaurant.currency ?? "UZS";
  const img = imageUrl(item.imageUrl, 1200);
  const name = contentName(item, lang);
  const description = contentDescription(item, lang);

  return (
    <main className="container-page py-10">
      <Link href="/menu" className="text-sm text-ink-muted hover:text-brand">
        ← {t.common.backToMenu}
      </Link>

      <div className="mt-6 grid grid-cols-1 gap-8 md:grid-cols-2">
        <div className="overflow-hidden rounded-2xl border border-line bg-ink/5">
          <div className="aspect-square w-full">
            {img ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={img}
                alt={name}
                className="h-full w-full object-cover"
              />
            ) : (
              <div className="flex h-full w-full items-center justify-center text-ink-muted/50">
                {t.common.noImage}
              </div>
            )}
          </div>
        </div>

        <div>
          <div className="flex items-center gap-2">
            {item.isPopular && (
<span className="badge-brand">★ {t.item.popular}</span>
            )}
          </div>
          <h1 className="mt-2 font-display text-3xl font-bold tracking-tight">{name}</h1>

          <div className="mt-3 flex items-end gap-3">
            <span className="font-display text-2xl font-bold">
              {formatPrice(item.price, currency, lang)}
            </span>
            {item.oldPrice != null && item.oldPrice > item.price && (
              <span className="text-lg text-ink-muted/70 line-through">
                {formatPrice(item.oldPrice, currency, lang)}
              </span>
            )}
          </div>

          {description && <p className="mt-4 text-ink-muted">{description}</p>}

          {/* A set has to list its courses: the price alone is not a decision. */}
          {item.comboContents?.length ? (
            <div className="mt-5 rounded-2xl border border-line bg-surface p-4">
              <p className="text-sm font-semibold">{t.item.comboContents}</p>
              <ul className="mt-2 space-y-1 text-sm text-ink-soft">
                {item.comboContents.map((c, i) => (
                  <li key={i} className="flex justify-between gap-3">
                    <span>{c.name}</span>
                    <span className="text-ink-muted">× {c.qty}</span>
                  </li>
                ))}
              </ul>
              {!!item.comboBasePrice && item.comboBasePrice > item.price && (
                <p className="mt-3 border-t border-line pt-3 text-sm font-semibold text-emerald-600 dark:text-emerald-400">
                  {t.item.comboSaving(
                    formatPrice(item.comboBasePrice - item.price, currency, lang),
                  )}
                </p>
              )}
            </div>
          ) : null}

          {item.tags && item.tags.length > 0 && (
            <div className="mt-4 flex flex-wrap gap-2">
              {item.tags.map((t) => (
                <span
                  key={t}
                  className="rounded-full bg-ink/5 px-3 py-1 text-xs text-ink-muted"
                >
                  {t}
                </span>
              ))}
            </div>
          )}

          {/* ⚠️ **Every size on one screen, and the current one marked.** A
              clothes shop's guest is not choosing a shirt, they are choosing
              M/black — and a page that shows one variant with no way to reach
              the others is a page that sells the wrong size or nothing at all.
              Links rather than a client-side picker: each variant is a real
              product with its own price, its own photo and its own stock, so
              its page is a real page. */}
          {siblings.length > 1 && (
            <div className="mt-6">
              <p className="text-sm font-semibold">{t.item.pickVariant}</p>
              <div className="mt-2 flex flex-wrap gap-2">
                {siblings.map((v) => {
                  const label = (v.variant ?? []).join(" / ");
                  const on = v.id === item.id;
                  return (
                    <Link
                      key={v.id}
                      href={`/menu/${v.id}`}
                      aria-current={on ? "true" : undefined}
                      className={`rounded-full border px-3.5 py-1.5 text-sm font-semibold transition-colors ${
                        on
                          ? "border-brand bg-brand text-white"
                          : "border-line hover:border-brand"
                      }`}
                    >
                      {label}
                    </Link>
                  );
                })}
              </div>
            </div>
          )}

          <div className="mt-8">
            <AddToCartControl item={item} currency={currency} />
          </div>
        </div>
      </div>

      {/* Below the fold, under the add button. Above it the suggestion would
          compete with the one action this page exists for. */}
      <Recommendations itemIds={[item.id]} currency={currency} />
    </main>
  );
}
