// Turning what somebody typed into the address of a Keel server.
//
// ⚠️ **A second implementation, and that is written down rather than hidden.**
// The rule lives in Go too (`backend/desktop/config_windows.go` → `apiBase`),
// because the Windows till resolves it before any JavaScript is running. There
// is nothing to import across that boundary, so the two are kept honest the way
// `serviceOn` is: one test, the same examples, both sides.
//
// ⚠️ **The whole point is that a restaurant types the short thing.** Whoever is
// setting up a phone knows their restaurant as "osh", not as
// "https://osh.keel.uz/api/v1" — and a form that demanded the long form would
// be answered with a guess. So a bare word becomes a Keel subdomain, and
// anything with a dot in it is taken as the host it obviously is (a restaurant
// on its own domain).

/** The API base for what was typed, or "" when nothing usable was. */
export function apiBaseFor(address: string): string {
  const host = hostFor(address);
  return host ? `https://${host}/api/v1` : "";
}

/** Where this restaurant's pictures are served from. Same host, and derived
 *  rather than asked: two fields would be two chances to mistype one answer. */
export function uploadsBaseFor(address: string): string {
  const host = hostFor(address);
  return host ? `https://${host}/uploads` : "";
}

function hostFor(address: string): string {
  let a = address.trim().toLowerCase();
  // ⚠️ Pasted addresses arrive with all of these. Refusing them would be
  // technically defensible and would fail the one person most likely to paste:
  // somebody who copied the link out of the panel.
  a = a.replace(/^https?:\/\//, "");
  a = a.replace(/\/+$/, "");
  a = a.replace(/\/api\/v1$/, "");
  a = a.replace(/\/+$/, "");
  if (!a) return "";
  // ⚠️ Whitespace and a slash inside mean this is not a host at all — a
  // sentence, or a deep link somebody copied. Better to refuse than to build an
  // address that resolves to nothing and fails as "no internet".
  if (/[\s/]/.test(a)) return "";
  // A bare word is a Keel subdomain; anything with a dot is its own host.
  return a.includes(".") ? a : `${a}.keel.uz`;
}
