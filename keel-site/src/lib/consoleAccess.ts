import type { Me } from "./api";

// Which console page needs which permission.
//
// ⚠️ **Hiding a tab is not guarding a page.** The navigation used to follow the
// role and nothing else did: an agent who typed `/console/seo` into the address
// bar got the page — its headings, its buttons, its explanation of how the
// platform presents itself — with every request underneath refused by the
// server. Nothing leaked from the database, and it still looked exactly like a
// hole, because a page is information too. The layout now asks this before it
// renders anything.
//
// ⚠️ **Unknown means closed.** A console page that is not listed here cannot be
// opened by anybody, and `consoleAccess.test.ts` fails on any page directory
// missing from the list — so a new page is a failing test the day it is added,
// not an open door discovered by a salesperson.

export type ConsolePermission = keyof Me["can"];

// Longest prefixes are not needed: every section is one path segment.
const SECTIONS: Record<string, ConsolePermission> = {
  // Selling: the customers, the visits, and the two tools a salesperson carries
  // into a restaurant.
  tenants: "tenants",
  visits: "tenants",
  outreach: "tenants",
  leaflet: "tenants",
  // The queue and what broke, read from both ends.
  support: "support",
  reports: "support",
  referrers: "partners",
  seo: "seo",
  blog: "blog",
  staff: "staff",
};

/** The console sections, for the test that every page is listed. */
export const CONSOLE_SECTIONS = Object.keys(SECTIONS);

/** The permission a console path needs, or null when the path is not a known
 *  console page. `/console` itself is the owner's overview. */
export function permissionFor(path: string): ConsolePermission | null {
  const clean = path.split(/[?#]/)[0].replace(/\/+$/, "");
  if (clean === "/console") return "overview";
  const m = /^\/console\/([^/]+)(\/|$)/.exec(clean);
  if (!m) return null;
  return SECTIONS[m[1]] ?? null;
}

/** Whether this account may open this console page. */
export function canOpen(path: string, can: Me["can"]): boolean {
  const perm = permissionFor(path);
  return perm !== null && can[perm] === true;
}
