"use client";

import { useTheme } from "@/lib/theme";
import { useI18n } from "@/lib/i18n/client";

// Sun / moon switch.
//
// Nothing rendered here is derived from React state, on purpose. The obvious
// version — "read the theme, render the matching icon" — cannot survive
// hydration in this tree: the header sits inside a <Suspense> boundary, so it
// hydrates *after* ThemeProvider's effect has already read localStorage and
// flipped its state. By then the client renders the dark icon against server
// HTML that says light, and React throws the whole subtree away. Gating on a
// `mounted` flag does not help for exactly the same reason — by the time this
// boundary hydrates, `mounted` is already true.
//
// So both icons are always rendered and **CSS** picks one, keyed off the `dark`
// class that the inline script in <head> puts on <html> before the first paint.
// The markup is then identical on the server and on the client, the right icon
// is visible immediately, and there is no flash.
export default function ThemeToggle({ className = "" }: { className?: string }) {
  const { toggle } = useTheme();
  const { t } = useI18n();

  return (
    <button
      type="button"
      onClick={toggle}
      // Stable for the same reason: a label that names the action rather than
      // the current state cannot disagree with what the server rendered.
      aria-label={t.common.themeToggle}
      title={t.common.themeToggle}
      className={`inline-flex h-9 w-9 items-center justify-center rounded-full border border-line bg-surface text-ink-soft transition-colors hover:border-brand hover:text-brand ${className}`}
    >
      {/* Moon — shown in light mode, click to go dark. */}
      <svg
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="h-4 w-4 dark:hidden"
        aria-hidden
      >
        <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8Z" />
      </svg>
      {/* Sun — shown in dark mode, click to go light. */}
      <svg
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="hidden h-4 w-4 dark:block"
        aria-hidden
      >
        <circle cx="12" cy="12" r="4" />
        <path d="M12 2v2m0 16v2M4.9 4.9l1.4 1.4m11.4 11.4 1.4 1.4M2 12h2m16 0h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
      </svg>
    </button>
  );
}
