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
import { DEFAULT_LANG, LANGS, PATH_HEADER, localePath } from "@/lib/i18n";
import { getLang } from "@/lib/i18n/server";

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

/** The path being rendered, with any language prefix already removed.
 *
 *  Set by middleware, which is the only place that has the URL before routing.
 *  Falls back to "/" outside a request (a build-time render). */
export async function sitePath(): Promise<string> {
  try {
    const h = await headers();
    return h.get(PATH_HEADER) || "/";
  } catch {
    return "/";
  }
}

/** This page's canonical URL and its three language addresses.
 *
 *  Set once in the root layout, inherited by every page. Two separate jobs:
 *
 *  **canonical** says which URL is the real one for *this* render. It has to be
 *  per-page: the layout used to declare the site root for everything, which
 *  tells a search engine that `/menu` and every dish page are duplicates of the
 *  home page — and that is not reported as an error, the pages simply never
 *  appear.
 *
 *  **languages** is the `hreflang` set, and it is what makes three URLs one
 *  page instead of three thin near-duplicates competing with each other. Every
 *  variant lists all of them, including itself; a one-way declaration is
 *  ignored. `x-default` points at Uzbek — the base language and the one a
 *  visitor with no matching preference should land on. */
export async function localeAlternates(): Promise<{
  canonical: string;
  languages: Record<string, string>;
}> {
  const origin = await siteOrigin();
  const path = await sitePath();
  const abs = (lang: (typeof LANGS)[number]) => {
    const p = localePath(lang, path);
    return p === "/" ? origin : `${origin}${p}`;
  };
  const languages: Record<string, string> = {};
  for (const lang of LANGS) languages[lang] = abs(lang);
  languages["x-default"] = abs(DEFAULT_LANG);
  return { canonical: abs(await getLang()), languages };
}

/** The verification token out of whatever the owner pasted.
 *
 *  Both consoles show the owner a **whole meta tag** and say "copy this", so
 *  that is what lands in the field far more often than the bare token. Pasting
 *  the tag would otherwise write `<meta name="google-site-verification" …>`
 *  into a `content` attribute — a page that renders fine, verifies nothing, and
 *  gives the owner no reason to suspect the field rather than the console.
 *
 *  Accepting both is one regex; refusing the tag would be a support call. */
export function verificationToken(raw?: string): string | undefined {
  const value = (raw ?? "").trim();
  if (!value) return undefined;
  if (!value.startsWith("<")) return value;
  return value.match(/content\s*=\s*["']([^"']+)["']/i)?.[1]?.trim() || undefined;
}
