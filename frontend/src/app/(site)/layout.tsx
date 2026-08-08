import { Suspense } from "react";
import { api, showWatermark } from "@/lib/api";
import { getSiteScope } from "@/lib/siteBrand.server";
import { CartProvider } from "@/lib/cart";
import { UserProvider } from "@/lib/user";
import { TableProvider } from "@/lib/table";
import TelegramApp from "@/components/site/TelegramApp";
import TableBanner from "@/components/site/TableBanner";
import Header from "@/components/site/Header";
import Footer from "@/components/site/Footer";
import StructuredData from "@/components/site/StructuredData";
import TrackVisit from "@/components/site/TrackVisit";
import { siteOrigin } from "@/lib/seo";
import type { BrandsResponse, Restaurant } from "@/lib/types";

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
  try {
    const data = await api.getRestaurant(scope);
    restaurant = data.restaurant;
    brandId = data.brand?.id ?? "";
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
      {/* The cart is per brand: a samsa and a plov are cooked in different
          kitchens and carried by different couriers, so they cannot share one
          basket. The id comes from the server, so the first render already
          reads the right cart. */}
      <CartProvider brandId={brandId}>
        {/* `?table=` is read from the URL, so the provider suspends like any
            other consumer of the search params. */}
        <Suspense fallback={null}>
          <TableProvider>
            {/* Inside Telegram this signs the guest in with no SMS at all, and
                teaches Telegram's own chrome (back button, closing
                confirmation, the honest viewport height) about this page. On the
                open web it renders nothing and loads nothing — see
                lib/telegram.tsx. */}
            <TelegramApp />
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
              />
              <TableBanner />
              <div className="flex-1">{children}</div>
              <Footer restaurant={restaurant} watermark={watermark} />
            </div>
          </TableProvider>
        </Suspense>
      </CartProvider>
    </UserProvider>
  );
}
