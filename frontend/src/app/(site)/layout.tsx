import { Suspense } from "react";
import { api } from "@/lib/api";
import { getSiteScope } from "@/lib/siteBrand.server";
import { CartProvider } from "@/lib/cart";
import { UserProvider } from "@/lib/user";
import { TableProvider } from "@/lib/table";
import TableBanner from "@/components/site/TableBanner";
import Header from "@/components/site/Header";
import Footer from "@/components/site/Footer";
import type { BrandsResponse, Restaurant } from "@/lib/types";

// Public site shell: cart state + header/footer around every public page.
export default async function SiteLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const scope = await getSiteScope();
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
            <div className="flex min-h-screen flex-col">
              <Header
                name={restaurant?.name ?? "Restoran"}
                logoUrl={restaurant?.logoUrl}
                brands={brands.brands}
                activeBrand={brandId}
              />
              <TableBanner />
              <div className="flex-1">{children}</div>
              <Footer restaurant={restaurant} />
            </div>
          </TableProvider>
        </Suspense>
      </CartProvider>
    </UserProvider>
  );
}
