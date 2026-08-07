// Turns `/ru` into the landing page rendered in Russian.
//
// See `lib/i18n/url.ts` for why this exists at all. In short: the language
// lived only in a cookie, and a crawler carries none — so the Russian version
// of the page that sells this product had no address, and could not be found by
// the people most likely to buy it.
//
// Doing it here rather than with a `[lang]` route segment keeps every page
// file and `<Link>` exactly as it is: the rewrite happens before routing.

import { NextRequest, NextResponse } from "next/server";
import {
  DEFAULT_LANG,
  LANG_COOKIE,
  LANG_COOKIE_MAX_AGE,
  LANG_HEADER,
  PATH_HEADER,
  splitLangPath,
} from "@/lib/i18n/url";

export function middleware(req: NextRequest) {
  const { lang, path } = splitLangPath(req.nextUrl.pathname);

  const headers = new Headers(req.headers);
  // ⚠️ Cleared first, and set only when the URL really carries a prefix.
  //
  // Two reasons, both silent when wrong. A client may send any header it likes,
  // so the value has to come from us or not at all. And an unprefixed URL must
  // leave the **cookie** in charge — setting this unconditionally would pin
  // every unprefixed request to Uzbek, which includes every screen of the
  // console, whose language is cookie-only by design.
  headers.delete(LANG_HEADER);
  headers.set(PATH_HEADER, path);

  if (lang === DEFAULT_LANG) {
    return NextResponse.next({ request: { headers } });
  }
  headers.set(LANG_HEADER, lang);

  const url = req.nextUrl.clone();
  url.pathname = path;
  const res = NextResponse.rewrite(url, { request: { headers } });
  // Carry the choice into ordinary navigation, so a visitor who arrived on
  // `/ru` stays in Russian when they open the console or the status page.
  res.cookies.set(LANG_COOKIE, lang, {
    path: "/",
    maxAge: LANG_COOKIE_MAX_AGE,
    sameSite: "lax",
  });
  return res;
}

export const config = {
  // `sitemap.xml` and `robots.txt` are excluded on purpose: they describe the
  // whole site and have no language of their own, and a `/ru/sitemap.xml` would
  // be a second sitemap claiming to describe the same one.
  matcher: ["/((?!api|_next|favicon.ico|icon.svg|apple-icon.png|robots.txt|sitemap.xml).*)"],
};
