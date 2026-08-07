"use client";

import { createContext, useCallback, useContext, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import {
  getDict,
  isLocalizedPath,
  localePath,
  LANG_COOKIE,
  LANG_COOKIE_MAX_AGE,
  type Dict,
  type Lang,
} from "./index";

type Ctx = {
  lang: Lang;
  t: Dict;
  setLang: (l: Lang) => void;
};

const LangContext = createContext<Ctx | null>(null);

// `initial` comes from the server layout (cookie), so the first client render
// matches the server HTML exactly — no hydration mismatch.
export function LangProvider({
  initial,
  children,
}: {
  initial: Lang;
  children: React.ReactNode;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const [lang, setLangState] = useState<Lang>(initial);

  const setLang = useCallback(
    (l: Lang) => {
      setLangState(l);
      document.cookie = `${LANG_COOKIE}=${l}; path=/; max-age=${LANG_COOKIE_MAX_AGE}; samesite=lax`;

      // ⚠️ **Go to the URL, do not just re-render.** The address bar is now
      // part of the answer: since `getLang()` reads the URL before the cookie,
      // switching to Russian while standing on `/menu` and only refreshing
      // would render Uzbek again — the cookie says ru, the URL says otherwise,
      // and the URL wins. The switch would appear to do nothing.
      //
      // It also makes the visible address match what is on screen, so a link
      // copied from a Russian page opens in Russian for whoever receives it.
      const here = pathname || "/";
      // The admin panel, the courier app and the staff app have no language
      // URL: they are behind a login and nothing in them is crawled, so the
      // cookie is the whole mechanism and `/ru/admin` must never be created.
      const next = isLocalizedPath(here) ? localePath(l, here) : here;
      if (next !== here) router.push(next);
      else router.refresh();
    },
    [router, pathname],
  );

  return (
    <LangContext.Provider value={{ lang, t: getDict(lang), setLang }}>
      {children}
    </LangContext.Provider>
  );
}

export function useI18n(): Ctx {
  const ctx = useContext(LangContext);
  if (!ctx) throw new Error("useI18n must be used inside LangProvider");
  return ctx;
}
