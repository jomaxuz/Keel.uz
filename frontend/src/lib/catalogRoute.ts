import { redirect } from "next/navigation";

import { getSiteScope } from "@/lib/siteBrand.server";
import { api } from "@/lib/api";
import { localePath } from "@/lib/i18n";
import { getTranslations } from "@/lib/i18n/server";
import { sellsGoods } from "@/lib/types";
import { sitePath } from "@/lib/seo";

// Where this site's catalogue lives, and how the other address gets there.
//
// ⚠️ **Two routes, one page, and whichever is wrong for this business
// redirects.** A restaurant's is `/menu`; a shop's is `/catalog`, because
// `shop.uz/menu` over a rail of trainers says "this site was built for somebody
// else" in the one place a guest is most likely to copy and paste.
//
// ⚠️ **Nothing that already exists stops working**, and that is the whole
// reason this is a redirect rather than a rename. Every saved link, QR code,
// Telegram button, sitemap entry and printed card points at `/menu`; a shop
// that simply moved would lose all of them. A permanent redirect keeps them and
// tells a crawler which address is the real one.
//
// ⚠️ The language prefix is carried across by hand: the middleware rewrites
// `/ru/catalog` to `/catalog` before a page ever runs, so a redirect written as
// `/menu` would drop a Russian guest into Uzbek.
//
// ⚠️ **Which address was asked for is read, not passed in.** Both routes render
// the same page module, so a hard-coded answer would make one of them send the
// guest to itself — a redirect loop, on the page a shop is found by.

/** Everything after the path, rebuilt. ⚠️ `sitePath()` is the path alone, and
 *  the query is what carries the table a QR code encodes, the branch a guest
 *  arrived through and the words they typed into the search box. A redirect
 *  that drops it turns "search for lag'mon" into "here is the catalogue" and a
 *  table QR into an ordinary visit — silently, because the page it lands on is
 *  a perfectly good page. */
function queryOf(params?: Record<string, string | string[] | undefined>): string {
  if (!params) return "";
  const qs = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value == null) continue;
    for (const one of Array.isArray(value) ? value : [value]) qs.append(key, one);
  }
  const out = qs.toString();
  return out ? `?${out}` : "";
}

/** Sends the guest to the other address when this one is not theirs.
 *
 *  ⚠️ **Temporary (307), not permanent (308), and that is deliberate.** A
 *  permanent redirect is what a search engine would prefer — but browsers cache
 *  it for as long as they like, and `businessType` is a setting the owner can
 *  change back. A restaurant that was briefly a shop would leave guests with
 *  `/menu → /catalog` cached forever, meeting a fresh `/catalog → /menu` on the
 *  server: a loop, in the visitor's own browser, that nothing on our side can
 *  clear.
 *
 *  Nothing is lost by it. The address that answers declares itself canonical
 *  (lib/seo.ts builds it from the path being rendered), the sitemap lists that
 *  same address and not this one, and the other address never renders a page —
 *  so there is no second copy for a crawler to weigh against it. */
export async function guardCatalogRoute(
  searchParams?: Record<string, string | string[] | undefined>,
) {
  const { lang } = await getTranslations();
  const here = await sitePath();
  const asked = here.startsWith("/catalog") ? "/catalog" : "/menu";
  let goods = false;
  try {
    const scope = await getSiteScope();
    const rest = await api.getRestaurant(scope);
    goods = sellsGoods({ businessType: rest?.brand?.businessType });
  } catch {
    // ⚠️ The backend is unreachable. Render what we have rather than bouncing
    // the guest between two addresses: a redirect loop is a worse page than a
    // page with the wrong word on it.
    return;
  }
  const right = goods ? "/catalog" : "/menu";
  if (right === asked) return;
  // The rest of the address travels with it: `/menu/abc` becomes `/catalog/abc`.
  const tail = here.slice(asked.length);
  redirect(localePath(lang, right + tail) + queryOf(searchParams));
}
