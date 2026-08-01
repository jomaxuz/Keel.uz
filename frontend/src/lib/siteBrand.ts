// Which brand's site the guest is on.
//
// A company may sell under more than one name — the restaurant and the samsa
// chain in the decision record. Each brand is its own menu, its own look and its
// own cart; the customer account underneath is shared.
//
// The choice lives in a cookie, exactly like the language, because the menu and
// the home page are server-rendered: a value in localStorage would arrive too
// late and the first paint would show the wrong brand. `?brand=` in the URL —
// which is what a printed QR card carries — wins over the cookie and replaces
// it, so a guest who scans a samsa card is in the samsa brand from then on.
//
// A single-brand restaurant never sets any of this: the cookie stays empty and
// the server answers with its only brand.

export const BRAND_COOKIE = "brand";
export const BRANCH_COOKIE = "branch";
// A year: a regular of one brand should not be asked again every week.
export const BRAND_COOKIE_MAX_AGE = 60 * 60 * 24 * 365;

/** Cart storage key for a brand. Two brands never share a basket: they are two
 *  kitchens and two couriers, so one receipt cannot hold both. */
export function cartKeyFor(brand: string | null | undefined): string {
  return brand ? `cart_v2:${brand}` : "cart_v2";
}

/** Read a cookie in the browser. Client components use this where a server
 *  component would call getSiteScope(). */
function readCookie(name: string): string {
  if (typeof document === "undefined") return "";
  const hit = document.cookie
    .split("; ")
    .find((c) => c.startsWith(`${name}=`));
  return hit ? decodeURIComponent(hit.slice(name.length + 1)) : "";
}

export function readBrandCookie(): string {
  return readCookie(BRAND_COOKIE);
}

export function readBranchCookie(): string {
  return readCookie(BRANCH_COOKIE);
}

export function writeBrandCookie(brand: string): void {
  document.cookie = `${BRAND_COOKIE}=${encodeURIComponent(brand)}; path=/; max-age=${BRAND_COOKIE_MAX_AGE}; samesite=lax`;
}

/** The branch a table QR pinned the guest to — this visit only, like the table
 *  itself, so going home does not keep ordering from the branch they sat in. */
export function writeBranchCookie(branchId: string): void {
  document.cookie = `${BRANCH_COOKIE}=${encodeURIComponent(branchId)}; path=/; samesite=lax`;
}

export function clearBranchCookie(): void {
  document.cookie = `${BRANCH_COOKIE}=; path=/; max-age=0; samesite=lax`;
}
