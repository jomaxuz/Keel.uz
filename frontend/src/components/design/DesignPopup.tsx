"use client";

// A drawn popup, shown over the page.
//
// It is composed exactly like a canvas band — that is how somebody designs one —
// but it must not render in the flow, or it is a rectangle in the middle of the
// page instead of a dialog over it.
//
// ⚠️ **Once per visit, and dismissible.** A popup that returns on every page is
// the thing people install ad blockers for, and one that cannot be closed makes
// the menu unreachable on a phone — where the close button is also the smallest
// target on screen. So: `sessionStorage`, not `localStorage` (this is one visit,
// not a promise about next month), a close button with a real hit area, the
// backdrop closes it, and Escape closes it.
//
// It also waits a moment before appearing. A popup that lands during the first
// paint is dismissed reflexively, before it has been read — which is the same as
// not showing it, except it also cost the guest a tap.

import { useEffect, useState } from "react";
import type { Dict, Lang } from "@/lib/i18n/dictionaries";
import type { DesignSection } from "@/lib/types";
import CanvasBlock from "./CanvasBlock";

const SEEN_KEY = "design_popup_seen";

export default function DesignPopup({
  section,
  lang,
  t,
}: {
  section: DesignSection;
  lang: Lang;
  t: Dict;
}) {
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (typeof window === "undefined") return;
    try {
      if (window.sessionStorage.getItem(SEEN_KEY) === "1") return;
    } catch {
      // Private mode with storage refused: showing it once per page is a worse
      // answer than not showing it at all.
      return;
    }
    const timer = window.setTimeout(() => setOpen(true), 1200);
    return () => window.clearTimeout(timer);
  }, []);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && close();
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open]);

  function close() {
    setOpen(false);
    try {
      window.sessionStorage.setItem(SEEN_KEY, "1");
    } catch {
      /* nothing to remember it with; it simply shows again next page */
    }
  }

  if (!open) return null;

  const bottom = section.variant === "bottom";

  return (
    <div
      className={`fixed inset-0 z-[90] flex px-4 ${
        bottom ? "items-end pb-4" : "items-center"
      } justify-center`}
      role="dialog"
      aria-modal="true"
    >
      {/* The backdrop is a button, so a tap anywhere outside closes it — the
          gesture people already expect, and the one that saves the close icon
          from being the only way out. */}
      <button
        type="button"
        aria-label={t.common.close}
        onClick={close}
        className="absolute inset-0 bg-ink/50 backdrop-blur-sm"
      />
      <div
        // ⚠️ `--tg-viewport` as the ceiling, not `100vh`: inside Telegram a
        // 100vh-tall dialog puts its close button under Telegram's own chrome.
        style={{ maxHeight: "calc(var(--tg-viewport, 100vh) - 2rem)" }}
        className="relative w-full max-w-lg overflow-auto rounded-3xl bg-surface shadow-card-hover"
      >
        <button
          type="button"
          onClick={close}
          aria-label={t.common.close}
          // A real hit area. On a phone this is the smallest target on screen and
          // the only one that matters.
          className="absolute right-3 top-3 z-10 flex h-10 w-10 items-center justify-center rounded-full bg-surface/90 text-ink-soft shadow-card"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            className="h-5 w-5"
            aria-hidden
          >
            <path d="M6 6l12 12M18 6 6 18" />
          </svg>
        </button>
        <CanvasBlock canvas={section.canvas} lang={lang} t={t} popup />
      </div>
    </div>
  );
}
