"use client";

import { useT } from "@/lib/i18n/client";

/** Draws both icons and lets CSS choose.
 *
 *  Deliberately not driven by React state: the theme is applied by an inline
 *  script before first paint, so any component that renders from state would
 *  disagree with the DOM on the first pass and hydrate wrong. The label names
 *  the action, not the state, for the same reason. */
export default function ThemeToggle() {
  const { t } = useT();
  return (
    <button
      type="button"
      aria-label={t.theme.toggle}
      title={t.theme.toggle}
      onClick={() => {
        const root = document.documentElement;
        const dark = root.classList.toggle("dark");
        localStorage.setItem("keel-theme", dark ? "dark" : "light");
      }}
      className="grid h-9 w-9 place-items-center rounded-xl border border-line text-ink-soft transition hover:bg-raised hover:text-ink"
    >
      <svg viewBox="0 0 24 24" className="h-4 w-4 dark:hidden" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round">
        <circle cx="12" cy="12" r="4" />
        <path d="M12 2v2M12 20v2M2 12h2M20 12h2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M19.1 4.9l-1.4 1.4M6.3 17.7l-1.4 1.4" />
      </svg>
      <svg viewBox="0 0 24 24" className="hidden h-4 w-4 dark:block" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
        <path d="M20 14.5A8.5 8.5 0 1 1 9.5 4a7 7 0 0 0 10.5 10.5Z" />
      </svg>
    </button>
  );
}
