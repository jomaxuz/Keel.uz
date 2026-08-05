"use client";

import { createContext, useContext } from "react";
import { dicts, uz, type Dict, type Lang } from "./dict";

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
  // A full reload rather than a router refresh: the language is read on the
  // server, and every string on the page depends on it.
  window.location.reload();
}
