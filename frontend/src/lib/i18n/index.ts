// Language plumbing shared by server and client code.
//
// The chosen language lives in a cookie so server components (home, menu,
// about) can render already-translated HTML, while client components read the
// same value through the LangProvider.
//
// ⚠️ A cookie alone is invisible to a search engine. A crawler carries no
// cookies, so with only the cookie every page on the site existed in exactly
// one language as far as Google and Yandex were concerned — the two Russian
// menus of a Tashkent restaurant were unreachable and unindexable, on the
// search most of its customers use. So the language also has a **URL**:
// `/ru/menu` and `/en/menu` alongside `/menu`. The prefix is stripped by
// middleware, which means every route file, link and API call stays as it was;
// what changed is that the same page now has three addresses and says so in
// `hreflang`.
//
// Uzbek is deliberately **unprefixed** rather than living at `/uz/`: it is the
// base language, its URLs are the ones already shared, printed on QR cards and
// indexed, and moving them would throw away whatever ranking they have.

import { dictionaries, LANGS, type Dict, type Lang } from "./dictionaries";

export { LANGS, LANG_LABEL, LANG_SHORT, dictionaries } from "./dictionaries";
export type { Dict, Lang } from "./dictionaries";

export const LANG_COOKIE = "lang";
export const DEFAULT_LANG: Lang = "uz";
// 1 year — the visitor's choice should stick.
export const LANG_COOKIE_MAX_AGE = 60 * 60 * 24 * 365;

export function isLang(value: unknown): value is Lang {
  return typeof value === "string" && (LANGS as readonly string[]).includes(value);
}

export function normalizeLang(value: unknown): Lang {
  return isLang(value) ? value : DEFAULT_LANG;
}

export function getDict(lang: Lang): Dict {
  return dictionaries[lang];
}

// ---- Language in the URL ----

/** The header middleware uses to tell the render which language the URL asked
 *  for. A header rather than a rewritten path segment: the page files stay
 *  unaware, and the value cannot be forged from outside — middleware sets it on
 *  every request, overwriting whatever a client sent. */
export const LANG_HEADER = "x-keel-lang";

/** The path with any language prefix removed, as middleware resolved it.
 *
 *  A layout cannot read the request path — `headers()` gives no route — and the
 *  canonical and hreflang tags need it. Passing it down from the one place that
 *  already parsed the URL beats re-deriving it in every page. */
export const PATH_HEADER = "x-keel-path";

/** Languages that appear as a URL prefix. Uzbek is the unprefixed base. */
export const PREFIXED_LANGS = LANGS.filter((l) => l !== DEFAULT_LANG);

/** The apps that live under this domain but are **not** the public site.
 *
 *  They are staff tools behind a login: nothing in them is crawled, indexed or
 *  shared, so a language in their URL would buy nothing and cost a second set
 *  of addresses for every screen. Their language stays cookie-only, which is
 *  also why `/ru/admin` must never be produced by the switcher. */
//
//  ⚠️ `/kassa` and `/zal` were missed when they were added, and the cost was
//  not a spare URL: switching to Russian pushed `/ru/kassa`, and every rule
//  keyed on the path stopped matching — starting with forcedLight() in
//  lib/theme.tsx, so the till went dark the first time anybody changed the
//  language on it.
const UNLOCALIZED = [
  "/admin",
  "/kuryer",
  "/staff",
  "/kiosk",
  "/kassa",
  "/zal",
];

/** Whether `path` is a public-site page, i.e. one that has language URLs. */
export function isLocalizedPath(path: string): boolean {
  const { path: bare } = splitLangPath(path);
  return !UNLOCALIZED.some((p) => bare === p || bare.startsWith(`${p}/`));
}

/** Splits "/ru/menu" into the language and the real path.
 *
 *  Returns the default language and the path unchanged when there is no
 *  prefix, so a caller never has to check first. */
export function splitLangPath(pathname: string): { lang: Lang; path: string } {
  const match = pathname.match(/^\/([a-z]{2})(\/.*)?$/);
  if (match && isLang(match[1]) && match[1] !== DEFAULT_LANG) {
    return { lang: match[1], path: match[2] || "/" };
  }
  return { lang: DEFAULT_LANG, path: pathname || "/" };
}

/** The address of `path` in `lang`.
 *
 *  `localePath("ru", "/menu")` → `/ru/menu`; the default language keeps the
 *  bare path. Accepts an already-prefixed path so callers can pass whatever
 *  they have (`usePathname()` returns the prefixed one). */
export function localePath(lang: Lang, path: string): string {
  const { path: bare } = splitLangPath(path.startsWith("/") ? path : `/${path}`);
  if (lang === DEFAULT_LANG) return bare;
  return bare === "/" ? `/${lang}` : `/${lang}${bare}`;
}
