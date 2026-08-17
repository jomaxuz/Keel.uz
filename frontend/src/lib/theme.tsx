"use client";

// Light/dark theme. The actual colours live in CSS variables (globals.css);
// this only toggles the `dark` class on <html> and remembers the choice.
//
// The very first paint is handled by the inline script in app/layout.tsx, so
// there is no flash of the wrong theme.

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";

export type Theme = "light" | "dark";
export const THEME_KEY = "theme";

/** Screens that are light whatever the stored choice says.
 *
 *  ⚠️ **The till and the waiter screen are equipment, not a website.** They run
 *  all day on a monoblock under restaurant lighting, next to a printer and a
 *  cash drawer, and the dish photographs on them are the same ones shot and
 *  retouched against white. A cashier who taps the wrong thing at eleven at
 *  night and flips the screen to dark has changed the tool the whole shift
 *  works on — and nobody else on the floor knows how to change it back.
 *
 *  ⚠️ **The rule lives here and in the inline script in app/layout.tsx, and the
 *  two must say the same thing**: one runs before the first paint and the other
 *  after, so any disagreement is a visible flash of the wrong theme. */
export function forcedLight(pathname: string): boolean {
  return (
    pathname === "/kassa" ||
    pathname.startsWith("/kassa/") ||
    pathname === "/zal" ||
    pathname.startsWith("/zal/")
  );
}

type Ctx = {
  theme: Theme;
  setTheme: (t: Theme) => void;
  toggle: () => void;
  mounted: boolean;
};

const ThemeContext = createContext<Ctx | null>(null);

function apply(theme: Theme) {
  document.documentElement.classList.toggle("dark", theme === "dark");
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  // SSR renders the light default; the real value is read on mount (the inline
  // script has already applied it to <html>, so nothing visibly changes).
  const [theme, setThemeState] = useState<Theme>("light");
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    // ⚠️ **Light unless the visitor chose otherwise — the device preference is
    // deliberately not consulted.** A restaurant site is a shop window: the
    // owner picks the accent, approves the photographs and shows the result to
    // people, and a guest whose phone is in dark mode was seeing a different
    // restaurant from the one that was signed off. Menu photographs are shot
    // and retouched against white, too.
    //
    // The toggle is untouched and the choice still persists; what changed is
    // only the answer for somebody who has never expressed one.
    const stored = window.localStorage.getItem(THEME_KEY) as Theme | null;
    const initial: Theme = stored === "dark" || stored === "light" ? stored : "light";
    setThemeState(initial);
    apply(forcedLight(window.location.pathname) ? "light" : initial);
    setMounted(true);
  }, []);

  const setTheme = useCallback((t: Theme) => {
    setThemeState(t);
    // ⚠️ The stored choice is still written: the till does not get to decide
    // what the owner sees in the panel. Only the class on <html> is withheld.
    if (!forcedLight(window.location.pathname)) apply(t);
    try {
      window.localStorage.setItem(THEME_KEY, t);
    } catch {
      /* private mode — theme just won't persist */
    }
  }, []);

  const toggle = useCallback(
    () => setTheme(theme === "dark" ? "light" : "dark"),
    [theme, setTheme],
  );

  return (
    <ThemeContext.Provider value={{ theme, setTheme, toggle, mounted }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme(): Ctx {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error("useTheme must be used inside ThemeProvider");
  return ctx;
}
