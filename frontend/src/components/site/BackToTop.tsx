"use client";

// "Back to the top", for the pages that got long.
//
// The menu is one scroll for every section a restaurant sells, and the profile
// and the about page are not far behind. Reaching the header — which is where
// the cart, the language and every other destination live — meant swiping back
// through all of it.
//
// ⚠️ **Appears only after the header is actually gone.** A button that is there
// from the first paint is a permanent obstruction over the hero of a page that
// never needed it; the threshold is a full viewport, so it shows up exactly
// when "go back up" stops being a small movement.
//
// ⚠️ **Brand-filled, not `btn-primary`.** The owner's button style can be an
// outline or a soft tint (see `lib/theme-css.ts`), and a transparent circle
// floating over photographs of food is invisible. This one always carries the
// restaurant's own accent as a solid fill — the same white-on-brand pairing
// every theme already uses for a hovered primary button.

import { useEffect, useState } from "react";
import { useI18n } from "@/lib/i18n/client";

export default function BackToTop() {
  const { t } = useI18n();
  const [show, setShow] = useState(false);

  useEffect(() => {
    let frame = 0;
    const read = () => {
      frame = 0;
      // One viewport: past this, the header is off-screen and getting back to
      // it is a real journey rather than a flick.
      setShow(window.scrollY > window.innerHeight);
    };
    // rAF-throttled: this fires on every scroll frame, on phones, on a page
    // that is already rendering a long list of dishes.
    const onScroll = () => {
      if (!frame) frame = window.requestAnimationFrame(read);
    };
    read();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      window.removeEventListener("scroll", onScroll);
      if (frame) window.cancelAnimationFrame(frame);
    };
  }, []);

  return (
    <button
      type="button"
      onClick={() =>
        window.scrollTo({
          top: 0,
          // Honoured rather than assumed: for a guest who asked their device
          // for less motion, a page flying past under their thumb is the exact
          // thing they turned off.
          behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches
            ? "auto"
            : "smooth",
        })
      }
      aria-label={t.common.toTop}
      title={t.common.toTop}
      // ⚠️ Hidden with opacity rather than unmounted, so it fades instead of
      // blinking into place mid-scroll. `pointer-events-none` while hidden
      // keeps the invisible circle from eating taps on whatever is under it.
      //
      // ⚠️ Below the cookie notice in the stack (z-40 against its z-50): that
      // bar is a one-time notice spanning the full width, and a button
      // punching through it would sit on top of its own text.
      className={`fixed bottom-[calc(1rem+env(safe-area-inset-bottom))] right-4 z-40 flex h-11 w-11 items-center justify-center rounded-full bg-brand text-white shadow-card-hover transition-all duration-200 hover:bg-brand-dark active:scale-95 ${
        show ? "opacity-100" : "pointer-events-none translate-y-2 opacity-0"
      }`}
      // Out of the tab order while invisible — a keyboard user should not land
      // on a control nobody can see.
      tabIndex={show ? 0 : -1}
      aria-hidden={!show}
    >
      <svg
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="h-5 w-5"
        aria-hidden
      >
        <path d="M12 19V5M5 12l7-7 7 7" />
      </svg>
    </button>
  );
}
