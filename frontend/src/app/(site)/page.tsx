import { api } from "@/lib/api";
import { getSiteScope } from "@/lib/siteBrand.server";
import { getTranslations } from "@/lib/i18n/server";
import { localized } from "@/lib/i18n/site-content";
import DesignRenderer from "@/components/design/DesignRenderer";
import type { MenuGroup, RestaurantResponse } from "@/lib/types";

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
    // backend unreachable — the renderer still draws the default bands, which
    // degrade to an empty-state hero on their own
  }

  // The layout is data now (see components/design). With no design drawn the
  // renderer walks DEFAULT_SECTIONS, which is the page this file used to
  // contain — same components, same order.
  return (
    <DesignRenderer
      design={data?.design}
      data={{
        t,
        lang,
        data,
        menu,
        currency: data?.restaurant?.currency ?? "UZS",
      }}
    />
  );
}
