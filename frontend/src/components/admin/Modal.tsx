"use client";

// The panel's dialog: a card over the page, with the three ways out a dialog is
// expected to have.
//
// ⚠️ **The visible × is the one that matters, and it was missing.** Clicking
// the backdrop closed this from the day it was written, and nothing on screen
// said so — so a dialog somebody opened by accident, or one they had finished
// reading, had no button to press, and the page underneath stayed covered. A
// close affordance nobody can see is not an affordance; it is a thing the
// author knows.
//
// ⚠️ **Escape as well**, because a form is where a keyboard already is. It is
// listened for on the document rather than on the card: the key press lands
// wherever the focus happens to be — often an input, sometimes nothing at all.

import { useEffect } from "react";

import { useAdminT } from "@/lib/i18n/admin";

export default function Modal({
  children,
  onClose,
  wide,
}: {
  children: React.ReactNode;
  onClose: () => void;
  wide?: boolean;
}) {
  const t = useAdminT();

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
      role="presentation"
    >
      <div
        role="dialog"
        aria-modal="true"
        className={`relative max-h-[90vh] w-full overflow-y-auto rounded-2xl bg-surface p-6 ${
          wide ? "max-w-2xl" : "max-w-md"
        }`}
        onClick={(e) => e.stopPropagation()}
      >
        {/* ⚠️ Sticky rather than absolute: these cards scroll, and a button
            pinned to the top of the *content* leaves the screen on a long form
            — which is exactly the dialog somebody most wants to escape from.
            Zero height so it never pushes the content down. */}
        <div className="sticky top-0 z-10 h-0 text-right">
          <button
            type="button"
            onClick={onClose}
            aria-label={t.common.close}
            title={t.common.close}
            className="-mr-2 -mt-2 rounded-full bg-surface/90 p-1.5 text-ink-muted backdrop-blur transition-colors hover:bg-ink/5 hover:text-ink"
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
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}
