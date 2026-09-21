import { Suspense } from "react";
import CrashReporter from "@/components/CrashReporter";
import { api, showWatermark } from "@/lib/api";
import { getSiteScope } from "@/lib/siteBrand.server";
import { BusinessProvider } from "@/lib/business";
import { CartProvider } from "@/lib/cart";
import CookieNotice from "@/components/site/CookieNotice";
import { FavoritesProvider } from "@/lib/favorites";
import { UserProvider } from "@/lib/user";
import { TableProvider } from "@/lib/table";
import TelegramApp from "@/components/site/TelegramApp";
import TableBanner from "@/components/site/TableBanner";
import Header from "@/components/site/Header";
import Footer from "@/components/site/Footer";
import StructuredData from "@/components/site/StructuredData";
import TrackVisit from "@/components/site/TrackVisit";
import BackToTop from "@/components/site/BackToTop";
import { siteOrigin } from "@/lib/seo";
import { DEFAULT_CHROME, siteChrome } from "@/lib/siteChrome";
import type { BrandsResponse, NavLink, Restaurant } from "@/lib/types";

// Public site shell: cart state + header/footer around every public page.
export default async function SiteLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const scope = await getSiteScope();
  // Resolved per request from the control plane: the flag is flipped in the
  // Keel console the moment a customer pays, and anything baked into the
  // container would stay wrong until the tenant was re-provisioned.
  const watermark = await showWatermark();
  const origin = await siteOrigin();
  let restaurant: Restaurant | null = null;
  let brands: BrandsResponse = { brands: [], branches: [] };
  let brandId = "";
  // The navigation bar and what this brand sells. ⚠️ Both come from the call
  // the shell already makes: a second request for the header would be one per
  // page view on every site on the platform, to decide five links.
  let navLinks: NavLink[] = [];
  let businessType = "";
  // How the bar itself looks, read out of the design's `navbar` band.
  //
  // ⚠️ **That band draws nothing in the page flow, and this is why.** The header
  // is rendered here, above `<main>`, on every page — the design only describes
  // the home page. Rendered inline it would put a second bar under the real one
  // on the home page and leave every other page with the built-in bar. It used
  // to draw nothing *silently*: all five built-in templates open with a navbar
  // band, and the renderer skipped every one of them with nothing saying why.
  let chrome = DEFAULT_CHROME;
  try {
    const data = await api.getRestaurant(scope);
    restaurant = data.restaurant;
    brandId = data.brand?.id ?? "";
    navLinks = data.design?.nav ?? [];
    businessType = data.brand?.businessType ?? "";
    chrome = siteChrome(data.design);
  } catch {
    restaurant = null;
  }
  try {
    brands = await api.getBrands();
  } catch {
    /* the site still works with no brand list — it just shows no switcher */
  }

  return (
    <UserProvider>
      <CrashReporter app="site" />
      {/* The cart is per brand: a samsa and a plov are cooked in different
          kitchens and carried by different couriers, so they cannot share one
          basket. The id comes from the server, so the first render already
          reads the right cart. */}
      {/* Inside the cart provider because it needs the same session, and above the
          pages because a dish's heart appears on four different screens — see
          lib/favorites.tsx for why the state cannot live per card. */}
      {/* What this site sells, for the client pages that are furthest from the
          server — the basket and the checkout, which are exactly where a shop's
          guest was being asked not to make it too spicy. */}
      <BusinessProvider type={businessType}>
      <CartProvider brandId={brandId}>
        <FavoritesProvider>
          {/* `?table=` is read from the URL, so the provider suspends like any
            other consumer of the search params. */}
          <Suspense fallback={null}>
            <TableProvider>
              {/* Inside Telegram this signs the guest in with no SMS at all, and
                teaches Telegram's own chrome (back button, closing
                confirmation, the honest viewport height) about this page. On the
                open web it renders nothing and loads nothing — see
                lib/telegram.tsx.

                ⚠️ **Wraps the pages, and must.** It was written self-closing
                here, which put every page *outside* its provider — so
                `useTelegram()` on the checkout, the phone link and the cookie
                notice all read the default context, where `inTelegram` is
                false. Nothing looked broken: the bridge itself worked (it is
                inside its own provider), the guest signed in, the back button
                worked. Only the pages were blind, and the visible cost was
                that **every mini app order was recorded as channel "web"** —
                so a restaurant paying for a bot was told nobody used it. */}
              <TelegramApp>
                <div className="flex min-h-screen flex-col">
                  {/* Read by Google and Yandex, invisible to a visitor: it is the
                  difference between a blue link and a card with the opening
                  hours, the phone number and a map pin. */}
                  <StructuredData restaurant={restaurant} origin={origin} />
                  {/* How many people came, as opposed to how many ordered — the
                  difference between "nobody wants this" and "nobody can find
                  it". Inside the existing Suspense: it reads the pathname. */}
                  <TrackVisit />
                  <Header
                    name={restaurant?.name ?? "Restoran"}
                    logoUrl={restaurant?.logoUrl}
                    brands={brands.brands}
                    activeBrand={brandId}
                    // ⚠️ The branch count, so the header can decide whether "Filiallar" is
                    // a destination or noise. A one-branch restaurant's address, hours and
                    // map are already on the about page, and a nav item that leads to a
                    // list of one is the kind of complexity this product deliberately
                    // hides from single-branch customers.
                    branchCount={(brands.branches ?? []).length}
                    // ⚠️ **The bar an online store's sections are typed into**,
                    // drawn in the console's constructor. Empty on every
                    // restaurant, which is what keeps today's header exactly as
                    // it is — see models/design.go, NavLink.
                    navLinks={navLinks}
                    businessType={businessType}
                    chrome={chrome}
                  />
                  <TableBanner />
                  {/* ⚠️ Below the fold and never blocking: it is a notice, not a gate. Hidden
                  inside Telegram, where a banner costs the first screen of a small viewport
                  to ask about storage the guest cannot see. */}
                  <CookieNotice />
                  <div className="flex-1">{children}</div>
                  <Footer
                    restaurant={restaurant}
                    watermark={watermark}
                    businessType={businessType}
                  />
                  {/* In the shell rather than on the long pages: which page is long
                  depends on how many dishes this restaurant sells, and a
                  per-page decision would be wrong for somebody. It costs
                  nothing on a short page — it never appears there. */}
                  <BackToTop />
                </div>
              </TelegramApp>
            </TableProvider>
          </Suspense>
        </FavoritesProvider>
      </CartProvider>
      </BusinessProvider>
    </UserProvider>
  );
}
