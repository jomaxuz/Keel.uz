"use client";

import { createContext, useCallback, useContext, useState } from "react";
import { useRouter } from "next/navigation";
import {
  getDict,
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
  const [lang, setLangState] = useState<Lang>(initial);

  const setLang = useCallback(
    (l: Lang) => {
      setLangState(l);
      document.cookie = `${LANG_COOKIE}=${l}; path=/; max-age=${LANG_COOKIE_MAX_AGE}; samesite=lax`;
      // Re-render the server components (home/menu/about) in the new language.
      router.refresh();
    },
    [router],
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
