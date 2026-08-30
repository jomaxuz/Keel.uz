"use client";

import { useEffect, useRef } from "react";

/** One observer for the whole page rather than one per block: a landing page
 *  has a few dozen of these, and thirty observers watching thirty elements is
 *  thirty callbacks on every scroll frame. */
let observer: IntersectionObserver | null = null;

function watch(el: HTMLElement) {
  if (typeof IntersectionObserver === "undefined") {
    // No observer (an old browser, a test runner): show everything. The one
    // outcome that must never happen here is a page of invisible text.
    el.dataset.shown = "true";
    return () => {};
  }
  if (!observer) {
    observer = new IntersectionObserver(
      (entries) => {
        for (const e of entries) {
          if (!e.isIntersecting) continue;
          (e.target as HTMLElement).dataset.shown = "true";
          // Revealed once. A block that fades back out when it leaves the
          // viewport turns a scroll back up into a second show.
          observer?.unobserve(e.target);
        }
      },
      // Fires a little before the block reaches the edge, so the movement is
      // finished by the time it is properly in view.
      { rootMargin: "0px 0px -12% 0px", threshold: 0.05 },
    );
  }
  observer.observe(el);
  return () => observer?.unobserve(el);
}

/** Scroll-reveal.
 *
 *  ⚠️ **The hidden state lives in CSS behind `@media (scripting: enabled)`**
 *  (`globals.css`), not in React state. Hiding from JavaScript means the first
 *  paint shows the block and then snatches it back; hiding from plain CSS means
 *  a visitor whose script never ran — a bad connection on a phone, which is
 *  most of this page's audience — gets a blank marketing page. A browser that
 *  does not know the `scripting` feature evaluates it as false and simply shows
 *  everything, which is the right way for this to fail.
 *
 *  `prefers-reduced-motion` is handled in the same media query, matching the
 *  rule the partner strip already follows. */
export default function Reveal({
  children,
  delay = 0,
  className = "",
}: {
  children: React.ReactNode;
  delay?: number;
  className?: string;
}) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    return watch(el);
  }, []);

  return (
    <div
      ref={ref}
      className={`reveal ${className}`}
      style={delay ? { transitionDelay: `${delay}ms` } : undefined}
    >
      {children}
    </div>
  );
}
