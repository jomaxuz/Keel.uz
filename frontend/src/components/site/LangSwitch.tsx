"use client";

// Language picker in the navbar: shows the active language and opens a small
// popup with the full list in front of the header.

import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { useI18n } from "@/lib/i18n/client";
import { LANGS, LANG_LABEL, LANG_SHORT } from "@/lib/i18n";

// Popup width (w-44) — needed to decide which side it can open towards.
const POPUP_W = 176;

export default function LangSwitch({ className = "" }: { className?: string }) {
  const { lang, setLang, t } = useI18n();
  const [open, setOpen] = useState(false);
  // The switch sits on the right of the site header but on the *left* of the
  // admin sidebar, so a fixed alignment always clips somewhere. Pick the side
  // that actually has room.
  const [align, setAlign] = useState<"left" | "right">("right");
  const ref = useRef<HTMLDivElement>(null);

  // Close on outside click / Escape.
  useEffect(() => {
    if (!open) return;
    function onClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  // Decide the side before the browser paints, so the popup never flashes in
  // the wrong place.
  useLayoutEffect(() => {
    if (!open || !ref.current) return;
    const box = ref.current.getBoundingClientRect();
    const roomLeft = box.right >= POPUP_W + 8;
    const roomRight = window.innerWidth - box.left >= POPUP_W + 8;
    // Prefer growing leftwards (the header case); fall back when it would clip.
    setAlign(roomLeft || !roomRight ? "right" : "left");
  }, [open]);

  return (
    <div ref={ref} className={`relative ${className}`}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={`${t.common.language}: ${LANG_LABEL[lang]}`}
        className="flex h-9 items-center gap-1.5 rounded-full border border-line bg-surface px-3 text-xs font-bold text-ink-soft transition-colors hover:border-brand hover:text-brand"
      >
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          className="h-4 w-4"
          aria-hidden
        >
          <circle cx="12" cy="12" r="9" />
          <path d="M3 12h18M12 3c2.5 2.7 2.5 15.3 0 18M12 3c-2.5 2.7-2.5 15.3 0 18" />
        </svg>
        {LANG_SHORT[lang]}
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.2"
          strokeLinecap="round"
          className={`h-3 w-3 transition-transform ${open ? "rotate-180" : ""}`}
          aria-hidden
        >
          <path d="m6 9 6 6 6-6" />
        </svg>
      </button>

      {open && (
        <div
          role="listbox"
          className={`absolute z-50 mt-2 w-44 max-w-[calc(100vw-1.5rem)] overflow-hidden rounded-2xl border border-line bg-surface p-1 shadow-card-hover ${
            align === "right" ? "right-0" : "left-0"
          }`}
        >
          {LANGS.map((l) => (
            <button
              key={l}
              type="button"
              role="option"
              aria-selected={lang === l}
              onClick={() => {
                setLang(l);
                setOpen(false);
              }}
              className={`flex w-full items-center justify-between gap-2 rounded-xl px-3 py-2 text-sm font-semibold transition-colors ${
                lang === l
                  ? "bg-brand-tint text-brand-dark"
                  : "text-ink-soft hover:bg-ink/5"
              }`}
            >
              <span>{LANG_LABEL[l]}</span>
              <span className="text-xs text-ink-muted">{LANG_SHORT[l]}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
