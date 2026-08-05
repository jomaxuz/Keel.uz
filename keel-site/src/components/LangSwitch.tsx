"use client";

import { useEffect, useRef, useState } from "react";
import { LANGS } from "@/lib/i18n/dict";
import { setLang, useT } from "@/lib/i18n/client";

const NAMES: Record<string, string> = {
  uz: "O'zbekcha",
  ru: "Русский",
  en: "English",
};

/** A button that opens a small panel under the navbar.
 *
 *  Three inline buttons fit while there are three languages and stop fitting
 *  the moment there is a fourth — and on a phone they were already competing
 *  with the menu and the theme toggle for the same corner. */
export default function LangSwitch() {
  const { lang } = useT();
  const [open, setOpen] = useState(false);
  const box = useRef<HTMLDivElement>(null);

  // Closing on an outside click and on Escape, because a popup that can only
  // be dismissed by choosing something is a trap.
  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const current = LANGS.find((l) => l.id === lang) ?? LANGS[0];

  return (
    <div ref={box} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="menu"
        aria-expanded={open}
        className="flex h-9 items-center gap-1.5 rounded-xl border border-line px-2.5 text-xs font-semibold text-ink-soft transition hover:bg-raised hover:text-ink"
      >
        <svg viewBox="0 0 24 24" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth={1.8}>
          <circle cx="12" cy="12" r="9" />
          <path d="M3 12h18M12 3a15 15 0 0 1 0 18a15 15 0 0 1 0-18" />
        </svg>
        {current.label}
        <svg
          viewBox="0 0 24 24"
          className={`h-3 w-3 transition ${open ? "rotate-180" : ""}`}
          fill="none"
          stroke="currentColor"
          strokeWidth={2.5}
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="m6 9 6 6 6-6" />
        </svg>
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 top-[calc(100%+8px)] z-50 w-44 overflow-hidden rounded-2xl border border-line bg-surface p-1 shadow-xl shadow-hull-950/15"
        >
          {LANGS.map((l) => (
            <button
              key={l.id}
              role="menuitem"
              type="button"
              onClick={() => {
                setOpen(false);
                if (l.id !== lang) setLang(l.id);
              }}
              className={`flex w-full items-center justify-between rounded-xl px-3 py-2.5 text-sm transition ${
                l.id === lang
                  ? "bg-raised font-semibold text-ink"
                  : "text-ink-soft hover:bg-raised hover:text-ink"
              }`}
            >
              <span>{NAMES[l.id] ?? l.label}</span>
              {l.id === lang ? (
                <svg viewBox="0 0 24 24" className="h-4 w-4 text-signal-500" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
                  <path d="M20 6 9 17l-5-5" />
                </svg>
              ) : (
                <span className="text-xs text-ink-muted">{l.label}</span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
