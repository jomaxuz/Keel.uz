import { cookies } from "next/headers";
import { getDict, normalizeLang, LANG_COOKIE, type Dict, type Lang } from "./index";

// Reads the language cookie in a server component. Using cookies() makes the
// page dynamic, which is what we want: the same URL renders in three languages.
export async function getLang(): Promise<Lang> {
  const store = await cookies();
  return normalizeLang(store.get(LANG_COOKIE)?.value);
}

export async function getTranslations(): Promise<{ lang: Lang; t: Dict }> {
  const lang = await getLang();
  return { lang, t: getDict(lang) };
}
