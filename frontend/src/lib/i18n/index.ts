// Language plumbing shared by server and client code.
//
// The chosen language lives in a cookie so server components (home, menu,
// about) can render already-translated HTML, while client components read the
// same value through the LangProvider.

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
