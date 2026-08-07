import { cookies, headers } from "next/headers";
import {
  getDict,
  normalizeLang,
  LANG_COOKIE,
  LANG_HEADER,
  isLang,
  type Dict,
  type Lang,
} from "./index";

// Reads the language for this render. Using cookies()/headers() makes the page
// dynamic, which is what we want: the same route renders in three languages.
//
// ⚠️ **The URL outranks the cookie**, and the order is the whole point. A
// Russian visitor who has been on the site has `lang=ru` in their cookie; when
// they open a link to `/en/menu`, the page they get must be the page the link
// promised. With the cookie first, every shared link would silently render in
// the recipient's own language — and the sender would never see it, because on
// their screen it looked right.
//
// The header is set by middleware on every request (see `middleware.ts`), so
// its absence means "not a page request" rather than "trust what came in".
export async function getLang(): Promise<Lang> {
  const h = await headers();
  const fromUrl = h.get(LANG_HEADER);
  if (isLang(fromUrl)) return fromUrl;
  const store = await cookies();
  return normalizeLang(store.get(LANG_COOKIE)?.value);
}

export async function getTranslations(): Promise<{ lang: Lang; t: Dict }> {
  const lang = await getLang();
  return { lang, t: getDict(lang) };
}
