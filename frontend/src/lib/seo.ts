// Where this render is being served from.
//
// ⚠️ **Read from the request, never from the build.** One Next build serves
// every restaurant on the platform, so a base URL baked in at build time would
// be the same wrong address for all of them — the same trap `rewrites()` set
// with `CONTROL_ORIGIN`, except the symptom here is silent: canonical tags,
// Open Graph images and sitemap entries would all point at somebody else's
// domain, and Google would quietly merge or drop the pages.
//
// `PUBLIC_BASE_URL` stays as the fallback for the single-restaurant product,
// where there is exactly one address and it is known at deploy time.

import { headers } from "next/headers";

/** The site's own origin for this request, e.g. "https://osh.uz". */
export async function siteOrigin(): Promise<string> {
  try {
    const h = await headers();
    const host = h.get("x-forwarded-host") ?? h.get("host");
    if (host) {
      // Behind Caddy every real request is HTTPS; only a local dev run is not.
      const proto =
        h.get("x-forwarded-proto") ??
        (host.startsWith("localhost") || host.startsWith("127.") ? "http" : "https");
      return `${proto}://${host}`;
    }
  } catch {
    // Not inside a request — a build-time render.
  }
  return process.env.PUBLIC_BASE_URL ?? "http://localhost:3000";
}

/** Absolute URL for a path on this site. */
export async function siteUrl(path = "/"): Promise<string> {
  const base = await siteOrigin();
  return path === "/" ? base : `${base}${path.startsWith("/") ? path : `/${path}`}`;
}
