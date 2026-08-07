"use client";

import { createContext, useContext } from "react";
import { dicts, uz, type Dict, type Lang } from "./dict";
import { localePath } from "./url";

const Ctx = createContext<{ t: Dict; lang: Lang }>({ t: uz, lang: "uz" });

export function I18nProvider({
  lang,
  children,
}: {
  lang: Lang;
  children: React.ReactNode;
}) {
  return <Ctx.Provider value={{ t: dicts[lang], lang }}>{children}</Ctx.Provider>;
}

/** For client components. The dictionary is handed down from the server render,
 *  so there is one source of language and no second fetch. */
export function useT() {
  return useContext(Ctx);
}

export function setLang(lang: Lang) {
  document.cookie = `lang=${lang}; path=/; max-age=${60 * 60 * 24 * 365}; samesite=lax`;
  // ⚠️ **Go to the URL, do not reload.**
  //
  // Since the language is read from the address before the cookie, switching to
  // Russian while standing on `/` and merely reloading would render Uzbek
  // again — the cookie says ru, the URL says otherwise, and the URL wins. The
  // switch would appear to do nothing at all.
  //
  // It also makes the address bar match what is on screen, so a link copied
  // from the Russian page opens in Russian for whoever receives it — which is
  // the entire point of giving each language a URL.
  //
  // Still a full navigation rather than a router push: every string on the page
  // is rendered on the server.
  window.location.href = localePath(lang, window.location.pathname) + window.location.hash;
}
