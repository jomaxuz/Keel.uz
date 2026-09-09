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
import { setApiLang } from "@/lib/api";

// Language and appearance: what somebody chose, remembered.
//
// ⚠️ **Not the web's mechanism.** On the site the language lives in the URL and
// a cookie, because a page has to be linkable and indexable in each language.
// A phone has neither problem — so the choice is simply a value on the device,
// and copying the web's machinery would have brought a router along with it.

const LANG_KEY = "keel_lang";
const THEME_KEY = "keel_theme";
const VIEW_KEY = "keel_menu_view";

/** "system" is a real answer and the default one: most people set their phone
 *  once and expect everything to follow it. */
export type ThemeChoice = Scheme | "system";

/** How the menu is drawn.
 *
 *  ⚠️ **Three, because one answer does not fit two restaurants.** A café with
 *  forty drinks and no photographs wants a list it can scan; a restaurant that
 *  has photographed its menu wants the picture, because that is how a waiter
 *  answers "what does it look like" without walking to the kitchen. And a
 *  waiter who knows the menu by heart wants neither — the tighter the rows, the
 *  fewer scrolls between the guest speaking and the dish being on the check.
 *
 *  ⚠️ Remembered per phone, not per restaurant: it is a preference of the
 *  person holding it. */
export type MenuView = "list" | "cards" | "photos";

interface Prefs {
  t: Dict;
  lang: Lang;
  setLang: (l: Lang) => void;
  theme: Theme;
  choice: ThemeChoice;
  setChoice: (c: ThemeChoice) => void;
  view: MenuView;
  setView: (v: MenuView) => void;
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

  const [view, setViewState] = useState<MenuView>(() => {
    const saved = readSaved(VIEW_KEY);
    return saved === "cards" || saved === "photos" ? saved : "list";
  });

  // ⚠️ **The shared API client is told too, and it is not decoration.** Half of
  // what these screens show is written by the server — report headings, error
  // messages, the owner's morning briefing — and a phone has no cookie for the
  // server to read the choice from. Without this the app is Russian and its
  // sentences are Uzbek, which nothing reports as a fault.
  setApiLang(lang);

  const system = useSystemScheme();
  const scheme: Scheme = choice === "system" ? system : choice;

  const setLang = useCallback((l: Lang) => {
    setLangState(l);
    saveValue(LANG_KEY, l);
    setApiLang(l);
  }, []);

  const setChoice = useCallback((c: ThemeChoice) => {
    setChoiceState(c);
    saveValue(THEME_KEY, c);
  }, []);

  const setView = useCallback((v: MenuView) => {
    setViewState(v);
    saveValue(VIEW_KEY, v);
  }, []);

  const value = useMemo<Prefs>(
    () => ({
      t: DICTS[lang],
      lang,
      setLang,
      theme: THEMES[scheme],
      choice,
      setChoice,
      view,
      setView,
    }),
    [lang, setLang, scheme, choice, setChoice, view, setView],
  );

  return createElement(Ctx.Provider, { value }, children);
}

export function usePrefs(): Prefs {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("PrefsProvider yo'q");
  return ctx;
}
