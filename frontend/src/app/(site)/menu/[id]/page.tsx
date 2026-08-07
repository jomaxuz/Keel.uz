// Locale-aware: hrefs stay unprefixed here and gain /ru or /en at render.
import Link from "@/components/site/LocaleLink";
import { notFound } from "next/navigation";
import { api, imageUrl, ApiError } from "@/lib/api";
import { getSiteScope } from "@/lib/siteBrand.server";
import { formatPrice } from "@/lib/format";
import AddToCartControl from "@/components/menu/AddToCartControl";
import { getTranslations } from "@/lib/i18n/server";
import { contentDescription, contentName } from "@/lib/i18n/content";
import type { MenuItem, RestaurantResponse } from "@/lib/types";

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
    const img = imageUrl(item.imageUrl);
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
  try {
    [item, rest] = await Promise.all([
      api.getMenuItem(id),
      api.getRestaurant(await getSiteScope()).catch(() => null),
    ]);
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) notFound();
    throw err;
  }

  const currency = rest?.restaurant.currency ?? "UZS";
  const img = imageUrl(item.imageUrl);
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

          <div className="mt-8">
            <AddToCartControl item={item} currency={currency} />
          </div>
        </div>
      </div>
    </main>
  );
}
