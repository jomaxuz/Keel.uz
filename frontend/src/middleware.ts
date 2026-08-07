// Turns `/ru/menu` into `/menu` rendered in Russian.
//
// The site has always been trilingual, but only through a cookie — and a
// crawler carries no cookies. Every page therefore existed in exactly one
// language as far as Google and Yandex were concerned: the Russian menu of a
// Tashkent restaurant had no address, so it could not be linked, shared or
// ranked. Giving each language a URL is the whole fix, and `hreflang` (set in
// each page's metadata) is what tells search engines the three are the same
// page rather than three thin duplicates of each other.
//
// Doing it here rather than with a `[lang]` route segment keeps every page
// file, `<Link>` and API path exactly as it was: the rewrite happens before
// routing, so `/ru/menu/123` reaches `app/(site)/menu/[id]` unchanged. The
// alternative moves the entire app one directory deeper for a concern that is
// two lines of URL handling.

import { NextRequest, NextResponse } from "next/server";
import {
  DEFAULT_LANG,
  LANG_COOKIE,
  LANG_COOKIE_MAX_AGE,
  LANG_HEADER,
  PATH_HEADER,
  splitLangPath,
} from "@/lib/i18n";

export function middleware(req: NextRequest) {
  const { lang, path } = splitLangPath(req.nextUrl.pathname);

  const headers = new Headers(req.headers);

  // ⚠️ **Cleared first, and set only when the URL really carries a prefix.**
  //
  // Two separate reasons, and getting either wrong is silent. A client may send
  // any header it likes, so passing one through unexamined would let a visitor
  // read any page in any language — the value has to come from us or not at
  // all. And an unprefixed URL must leave the *cookie* in charge: setting the
  // header unconditionally pinned every unprefixed request to Uzbek, which
  // included every screen of the admin panel, the courier app and the staff
  // app. Those have no language URL by design, so their language would simply
  // have stopped working — with the switcher still moving and the cookie still
  // correct.
  headers.delete(LANG_HEADER);
  // The unprefixed path, so the root layout can build this page's canonical and
  // its three hreflang alternates. Every page inherits them from there, which
  // is also the fix for an older mistake: the layout declared the site root as
  // the canonical URL of *every* page, quietly asking search engines to drop
  // the menu and the dish pages — the only pages here worth ranking.
  headers.set(PATH_HEADER, path);

  if (lang === DEFAULT_LANG) {
    return NextResponse.next({ request: { headers } });
  }
  headers.set(LANG_HEADER, lang);

  const url = req.nextUrl.clone();
  url.pathname = path;
  const res = NextResponse.rewrite(url, { request: { headers } });

  // Carry the choice into ordinary navigation. Links inside the site are
  // unprefixed for everything the crawler does not read (cart, checkout,
  // profile), so without this a Russian visitor would land back in Uzbek the
  // moment they opened their basket. The URL wins over the cookie on the way
  // in, so this can never fight the address bar.
  res.cookies.set(LANG_COOKIE, lang, {
    path: "/",
    maxAge: LANG_COOKIE_MAX_AGE,
    sameSite: "lax",
  });
  return res;
}

export const config = {
  // Everything except the things that are not pages. `sitemap.xml` and
  // `robots.txt` are excluded on purpose: they are generated per host and have
  // no language of their own, and a `/ru/sitemap.xml` would be a second sitemap
  // claiming to describe the same site.
  matcher: ["/((?!api|uploads|_next|favicon.ico|icon.svg|robots.txt|sitemap.xml).*)"],
};
