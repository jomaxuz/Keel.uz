import {
  createContext,
  createElement,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import { DICTS, type Dict, type Lang } from "./i18n";
import { THEMES, useSystemScheme, type Scheme, type Theme } from "./theme";
import { readSaved, saveValue } from "./tokens";

// Language and appearance: what somebody chose, remembered.
//
// ⚠️ **Not the web's mechanism.** On the site the language lives in the URL and
// a cookie, because a page has to be linkable and indexable in each language.
// A phone has neither problem — so the choice is simply a value on the device,
// and copying the web's machinery would have brought a router along with it.
//
// ⚠️ **The language is the phone's, not the restaurant's.** Couriers move
// between jobs and a Russian-speaking one keeps reading Russian at the next
// restaurant; storing this against the account would reset it every time
// somebody changed employer, which is the group this setting exists for.

const LANG_KEY = "keel_lang";
const THEME_KEY = "keel_theme";

/** "system" is a real answer and the default one: most people set their phone
 *  once and expect everything to follow it. */
export type ThemeChoice = Scheme | "system";



interface Prefs {
  t: Dict;
  lang: Lang;
  setLang: (l: Lang) => void;
  theme: Theme;
  choice: ThemeChoice;
  setChoice: (c: ThemeChoice) => void;
}

const Ctx = createContext<Prefs | null>(null);

export function PrefsProvider({ children }: { children: ReactNode }) {
  // ⚠️ Read from what was already hydrated at startup rather than fetched here:
  // the first render must be in the right language, and a screen that painted
  // Uzbek and then switched to Russian is a flash somebody reads as a fault.
  const [lang, setLangState] = useState<Lang>(() => {
    const saved = readSaved(LANG_KEY);
    return saved === "ru" || saved === "en" ? saved : "uz";
  });
  const [choice, setChoiceState] = useState<ThemeChoice>(() => {
    const saved = readSaved(THEME_KEY);
    return saved === "light" || saved === "dark" ? saved : "system";
  });

  const system = useSystemScheme();
  const scheme: Scheme = choice === "system" ? system : choice;

  const setLang = useCallback((l: Lang) => {
    setLangState(l);
    saveValue(LANG_KEY, l);
  }, []);

  const setChoice = useCallback((c: ThemeChoice) => {
    setChoiceState(c);
    saveValue(THEME_KEY, c);
  }, []);

  const value = useMemo<Prefs>(
    () => ({
      t: DICTS[lang],
      lang,
      setLang,
      theme: THEMES[scheme],
      choice,
      setChoice,
    }),
    [lang, setLang, scheme, choice, setChoice],
  );

  return createElement(Ctx.Provider, { value }, children);
}

export function usePrefs(): Prefs {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("PrefsProvider yo'q");
  return ctx;
}
