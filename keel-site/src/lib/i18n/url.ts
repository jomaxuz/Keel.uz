// Language in the URL, for the one site that has to be *found*.
//
// keel.uz has been trilingual from the start, but only through a cookie — and
// a crawler carries no cookies. Every page therefore existed in exactly one
// language as far as Google and Yandex were concerned, and that language was
// Uzbek. On the marketing site this is not a nicety: a restaurant owner in
// Tashkent looking for "сайт для ресторана с доставкой" searches in Russian,
// and the Russian version of this page had no address at all — it could not be
// linked, shared, or ranked.
//
// Same shape as the tenant app: middleware strips `/ru` or `/en` before
// routing, so every page file and `<Link>` stays as it was. Uzbek is the
// unprefixed base — its URLs are the ones already printed, shared and indexed,
// and moving them would throw away whatever ranking they have.

import type { Lang } from "./dict";

export const DEFAULT_LANG: Lang = "uz";
export const LANG_COOKIE = "lang";
export const LANG_COOKIE_MAX_AGE = 60 * 60 * 24 * 365;

/** The language this render is in, as middleware resolved it from the URL. */
export const LANG_HEADER = "x-keel-lang";
/** The path with the language prefix removed — a layout cannot read the route,
 *  and the canonical and hreflang tags need it. */
export const PATH_HEADER = "x-keel-path";

export const ALL_LANGS: Lang[] = ["uz", "ru", "en"];
/** Languages that appear as a URL prefix. Uzbek is the unprefixed base. */
export const PREFIXED_LANGS = ALL_LANGS.filter((l) => l !== DEFAULT_LANG);

export const ORIGIN = "https://keel.uz";

export function isLang(v: unknown): v is Lang {
  return v === "uz" || v === "ru" || v === "en";
}

/** Splits "/ru/status" into the language and the real path. Returns the default
 *  language and an unchanged path when there is no prefix. */
export function splitLangPath(pathname: string): { lang: Lang; path: string } {
  const m = pathname.match(/^\/([a-z]{2})(\/.*)?$/);
  if (m && isLang(m[1]) && m[1] !== DEFAULT_LANG) {
    return { lang: m[1], path: m[2] || "/" };
  }
  return { lang: DEFAULT_LANG, path: pathname || "/" };
}

/** The address of `path` in `lang`. The default language keeps the bare path. */
export function localePath(lang: Lang, path: string): string {
  const { path: bare } = splitLangPath(path.startsWith("/") ? path : `/${path}`);
  if (lang === DEFAULT_LANG) return bare;
  return bare === "/" ? `/${lang}` : `/${lang}${bare}`;
}

/** Absolute URL for a path in a language. */
export function localeUrl(lang: Lang, path: string): string {
  const p = localePath(lang, path);
  return p === "/" ? ORIGIN : `${ORIGIN}${p}`;
}

/** This page's canonical URL and its three language addresses.
 *
 *  `languages` is the `hreflang` set, and it is what makes three URLs one page
 *  rather than three thin near-duplicates competing with each other. Every
 *  variant lists all of them, including itself — a one-way declaration is
 *  ignored. `x-default` points at Uzbek, the base language. */
export function alternatesFor(lang: Lang, path: string) {
  const languages: Record<string, string> = {};
  for (const l of ALL_LANGS) languages[l] = localeUrl(l, path);
  languages["x-default"] = localeUrl(DEFAULT_LANG, path);
  return { canonical: localeUrl(lang, path), languages };
}
