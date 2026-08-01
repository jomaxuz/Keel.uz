import { cookies } from "next/headers";
import { BRAND_COOKIE, BRANCH_COOKIE } from "./siteBrand";
import type { SiteScope } from "./api";

// The site's lens, read in a server component. Both values are optional: a
// single-brand, single-branch install has neither cookie set and every API call
// below goes out exactly as it did before brands existed.
export async function getSiteScope(): Promise<SiteScope> {
  const store = await cookies();
  const brand = store.get(BRAND_COOKIE)?.value;
  const branchId = store.get(BRANCH_COOKIE)?.value;
  const scope: SiteScope = {};
  if (brand) scope.brand = brand;
  if (branchId) scope.branchId = branchId;
  return scope;
}
