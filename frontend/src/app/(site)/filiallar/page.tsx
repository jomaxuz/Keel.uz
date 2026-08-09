// Every branch, in a list beside one map.
//
// ⚠️ **The list and the map together, not one or the other.** They answer different
// halves of the same question: the list says which branch is which (and gives the phone
// number somebody is about to ring), the map says which one is near. A page with only a
// list makes a guest read six addresses to find the closest; a page with only pins makes
// them tap each one to find a phone number.
//
// ⚠️ Rendered on the server, like every other public page — the branches come from the
// same call the header already makes, so this page costs one request and is indexable.
// A restaurant's branch list is exactly the kind of page somebody finds through a search
// engine.

import { api } from "@/lib/api";
import { getTranslations } from "@/lib/i18n/server";
import { getSiteScope } from "@/lib/siteBrand.server";
import BranchesView from "@/components/site/BranchesView";
import type { Branch } from "@/lib/types";

export async function generateMetadata() {
  const { t } = await getTranslations();
  return { title: t.branches.title };
}

export default async function BranchesPage() {
  const { t, lang } = await getTranslations();

  let branches: Branch[] = [];
  try {
    const scope = await getSiteScope();
    const [brands, rest] = await Promise.all([
      api.getBrands(),
      api.getRestaurant(scope),
    ]);
    // Only the branches of the brand the guest is looking at: in a two-brand company the
    // other brand's addresses are a different restaurant as far as this page is
    // concerned.
    branches = (brands.branches ?? []).filter(
      (b) => !scope.brand || b.brandId === scope.brand || b.brandId === rest.brand?.id,
    );
  } catch {
    // Backend unreachable: the page still renders its heading and says there is nothing
    // to show, rather than failing the whole route.
  }

  return (
    <main>
      <section className="border-b border-line bg-surface">
        <div className="container-page py-12 sm:py-16">
          <p className="eyebrow">{t.branches.eyebrow}</p>
          <h1 className="mt-2 section-title">{t.branches.title}</h1>
          <p className="mt-3 max-w-xl text-ink-muted">
            {t.branches.subtitle(branches.length)}
          </p>
        </div>
      </section>

      <BranchesView branches={branches} lang={lang} />
    </main>
  );
}
